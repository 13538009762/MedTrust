package controller_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"medtrust-backend/config"
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
	"medtrust-backend/router"
	"medtrust-backend/service"
)

// initHighVulnTestEnv 初始化高危漏洞回归测试环境（不调用 EnsureBaselineLedgerAnchored 覆写既有记录）
func initHighVulnTestEnv(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	db, err := repository.InitDB()
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	repository.SeedInitialData(db)
	repository.InitRedis()

	tempDir := t.TempDir()
	blockchain.InitBlockchainService(filepath.Join(tempDir, "fabric_ledger"), blockchain.FabricGatewayConfig{
		Mode:    "mock",
		Enabled: false,
	})
	service.InitMedicalService("mock", cfg.IPFS.APIURL, filepath.Join(tempDir, "ipfs_storage"))
	service.InitAccessEngine(cfg.Risk.LowThreshold, cfg.Risk.HighThreshold)

	if service.DefaultAccessEngine != nil && service.DefaultAccessEngine.GetRiskEngine() != nil {
		service.DefaultAccessEngine.GetRiskEngine().SetClock(func() time.Time {
			now := time.Now()
			return time.Date(now.Year(), now.Month(), now.Day(), 14, 0, 0, 0, now.Location())
		})
		t.Cleanup(func() {
			service.DefaultAccessEngine.GetRiskEngine().SetClock(nil)
		})
	}

	return router.SetupRouter()
}

// -------------------------------------------------------------------
// V-01: 终诊归档接口水平/跨院越权篡改 (IDOR) 与重复归档覆写防护
// -------------------------------------------------------------------
func TestHighVuln_V01_CompleteEncounterFinal_IDORAndRepeatBlocked(t *testing.T) {
	r := initHighVulnTestEnv(t)
	docLi, docWang, _, patZhang, _, _ := loadSecuritySeedUsers(t)

	// 创建同院另一位医生（医院1，非本单经治医生）
	ts := time.Now().UnixNano()
	docColleague := model.User{
		UserNo:       fmt.Sprintf("DOC_V01_%d", ts),
		Username:     fmt.Sprintf("doc_v01_%d", ts),
		RealName:     "同院非经治医生",
		Role:         "doctor",
		HospitalID:   docLi.HospitalID,
		DepartmentID: docLi.DepartmentID,
		Status:       "NORMAL",
	}
	if err := repository.DB.Create(&docColleague).Error; err != nil {
		t.Fatalf("创建测试医生失败: %v", err)
	}
	t.Cleanup(func() {
		repository.DB.Unscoped().Delete(&docColleague)
	})

	// doc_li 为 pat_zhang 创建初诊记录（待检查流转态）
	rec, _, err := service.DefaultMedicalService.CreateEncounterInitial(service.CreateEncounterInitialParams{
		DoctorID:         docLi.ID,
		PatientID:        patZhang.ID,
		EncounterType:    "OUTPATIENT",
		DepartmentName:   "心内科",
		ChiefComplaint:   "胸前区压迫感2天",
		InitialDiagnosis: "冠心病待查",
		NeedExam:         true,
		ExamItems:        []string{"心电图"},
	})
	if err != nil || rec == nil {
		t.Fatalf("CreateEncounterInitial failed: %v", err)
	}
	t.Cleanup(func() {
		repository.DB.Where("record_id = ?", rec.ID).Delete(&model.MedicalExamOrder{})
		repository.DB.Where("record_id = ?", rec.ID).Delete(&model.MedicalFile{})
		repository.DB.Where("id = ?", rec.ID).Delete(&model.MedicalRecord{})
	})

	tamperBody := `{"diagnosis":"恶意越权篡改确诊","etiology":"伪造病因","treatment_plan":"危险处方"}`

	// 1. 外院医生 doc_wang 越权提交终诊 -> 必须返回 403
	reqCross := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/encounters/%d/complete", rec.ID), strings.NewReader(tamperBody))
	reqCross.Header.Set("Authorization", authHeader(t, docWang))
	reqCross.Header.Set("Content-Type", "application/json")
	wCross := httptest.NewRecorder()
	r.ServeHTTP(wCross, reqCross)
	if wCross.Code != http.StatusForbidden {
		t.Fatalf("【V-01 漏洞】外院医生越权完成他人初诊未被 403 拦截: status=%d body=%s", wCross.Code, wCross.Body.String())
	}

	// 2. 同院非经治医生 docColleague 横向越权提交终诊 -> 必须返回 403
	reqSameHosp := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/encounters/%d/complete", rec.ID), strings.NewReader(tamperBody))
	reqSameHosp.Header.Set("Authorization", authHeader(t, &docColleague))
	reqSameHosp.Header.Set("Content-Type", "application/json")
	wSameHosp := httptest.NewRecorder()
	r.ServeHTTP(wSameHosp, reqSameHosp)
	if wSameHosp.Code != http.StatusForbidden {
		t.Fatalf("【V-01 漏洞】同院非经治医生横向覆盖他人初诊未被 403 拦截: status=%d body=%s", wSameHosp.Code, wSameHosp.Body.String())
	}

	// 确认数据库中记录未被篡改
	afterAttack, _ := repository.DefaultMedicalRepository.FindRecordByID(rec.ID)
	if afterAttack.Status == "COMPLETED" || strings.Contains(afterAttack.Diagnosis, "恶意越权") {
		t.Fatalf("【V-01 漏洞】越权请求已实质污染数据库记录: status=%s diag=%s", afterAttack.Status, afterAttack.Diagnosis)
	}

	// 3. 经治医生本人 doc_li 正常签署终诊 -> 200 OK 且响应中身份证/手机号已脱敏
	legitBody := `{"diagnosis":"稳定性劳力型心绞痛","etiology":"冠脉供血不足","treatment_plan":"阿司匹林肠溶片 100mg qd"}`
	reqOwner := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/encounters/%d/complete", rec.ID), strings.NewReader(legitBody))
	reqOwner.Header.Set("Authorization", authHeader(t, docLi))
	reqOwner.Header.Set("Content-Type", "application/json")
	wOwner := httptest.NewRecorder()
	r.ServeHTTP(wOwner, reqOwner)
	if wOwner.Code != http.StatusOK {
		t.Fatalf("经治医生本人正常终诊失败: status=%d body=%s", wOwner.Code, wOwner.Body.String())
	}
	var ownerResp struct {
		Data model.MedicalRecord `json:"data"`
	}
	_ = json.Unmarshal(wOwner.Body.Bytes(), &ownerResp)
	if ownerResp.Data.PatientIDCard != "" && !strings.Contains(ownerResp.Data.PatientIDCard, "*") {
		t.Fatalf("【V-01 PII 泄露】终诊响应返回了未脱敏身份证号: %s", ownerResp.Data.PatientIDCard)
	}
	if ownerResp.Data.PatientPhone != "" && !strings.Contains(ownerResp.Data.PatientPhone, "*") {
		t.Fatalf("【V-01 PII 泄露】终诊响应返回了未脱敏手机号: %s", ownerResp.Data.PatientPhone)
	}

	// 4. 对已归档上链的病历再次发起 HTTP 终诊 -> 必须返回 409 Conflict 防二次覆写
	reqRepeat := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/encounters/%d/complete", rec.ID), strings.NewReader(tamperBody))
	reqRepeat.Header.Set("Authorization", authHeader(t, docLi))
	reqRepeat.Header.Set("Content-Type", "application/json")
	wRepeat := httptest.NewRecorder()
	r.ServeHTTP(wRepeat, reqRepeat)
	if wRepeat.Code != http.StatusConflict {
		t.Fatalf("【V-01 漏洞】已完结病历允许通过 HTTP 重复覆写确诊: status=%d body=%s", wRepeat.Code, wRepeat.Body.String())
	}
}

// -------------------------------------------------------------------
// V-02: 列表接口 admin 隔离 + 同院主任/同科室医生严禁绕过患者 PRIVATE 隐私锁
// -------------------------------------------------------------------
func TestHighVuln_V02_MedicalRecordList_AdminBlockedAndPrivateLockRespected(t *testing.T) {
	r := initHighVulnTestEnv(t)
	docLi, _, _, patZhang, _, admin := loadSecuritySeedUsers(t)

	// 1. admin 访问 GET /api/v1/medical-records 必须返回 403
	reqAdmin := httptest.NewRequest(http.MethodGet, "/api/v1/medical-records", nil)
	reqAdmin.Header.Set("Authorization", authHeader(t, admin))
	wAdmin := httptest.NewRecorder()
	r.ServeHTTP(wAdmin, reqAdmin)
	if wAdmin.Code != http.StatusForbidden {
		t.Fatalf("【V-02 漏洞】系统管理员访问病历列表未被 403 拦截: status=%d body=%s", wAdmin.Code, wAdmin.Body.String())
	}

	// 2. 创建同院同科室主任医师（非经治医生）
	ts := time.Now().UnixNano()
	docSeniorSameDept := model.User{
		UserNo:       fmt.Sprintf("DOC_SEN_%d", ts),
		Username:     fmt.Sprintf("doc_sen_%d", ts),
		RealName:     "同院同科主任医师",
		Role:         "doctor",
		HospitalID:   docLi.HospitalID,
		DepartmentID: docLi.DepartmentID,
		Title:        "主任医师",
		Status:       "NORMAL",
	}
	if err := repository.DB.Create(&docSeniorSameDept).Error; err != nil {
		t.Fatalf("创建同院主任医师失败: %v", err)
	}
	t.Cleanup(func() {
		repository.DB.Unscoped().Delete(&docSeniorSameDept)
	})

	// doc_li 上传一份敏感病历，随后患者将其锁定为 PRIVATE
	secretDiag := fmt.Sprintf("高度私密精神心理诊断_PRIVATE_%d", ts)
	rec, err := service.DefaultMedicalService.UploadRecord(service.UploadRecordParams{
		DoctorID:       docLi.ID,
		PatientID:      patZhang.ID,
		DataType:       "EMR",
		DepartmentName: "心内科",
		Diagnosis:      secretDiag,
		Symptoms:       "重度焦虑失眠隐私主诉",
		TreatmentPlan:  "私密心理干预方案",
	})
	if err != nil || rec == nil {
		t.Fatalf("UploadRecord failed: %v", err)
	}
	t.Cleanup(func() {
		repository.DB.Where("record_id = ?", rec.ID).Delete(&model.Authorization{})
		repository.DB.Where("record_id = ?", rec.ID).Delete(&model.MedicalFile{})
		repository.DB.Where("id = ?", rec.ID).Delete(&model.MedicalRecord{})
	})

	// 患者通过接口设置该单病历为 PRIVATE 隐私锁
	reqLock := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/medical-records/%d/access-policy", rec.ID),
		strings.NewReader(`{"policy_type":"PRIVATE"}`))
	reqLock.Header.Set("Authorization", authHeader(t, patZhang))
	reqLock.Header.Set("Content-Type", "application/json")
	wLock := httptest.NewRecorder()
	r.ServeHTTP(wLock, reqLock)
	if wLock.Code != http.StatusOK {
		t.Fatalf("患者设置 PRIVATE 锁失败: status=%d body=%s", wLock.Code, wLock.Body.String())
	}

	// 同院同科主任医师调用列表接口 GET /api/v1/medical-records?patient_id=...
	reqList := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/medical-records?patient_id=%d", patZhang.ID), nil)
	reqList.Header.Set("Authorization", authHeader(t, &docSeniorSameDept))
	wList := httptest.NewRecorder()
	r.ServeHTTP(wList, reqList)
	if wList.Code != http.StatusOK {
		t.Fatalf("列表查询异常: status=%d body=%s", wList.Code, wList.Body.String())
	}
	if strings.Contains(wList.Body.String(), secretDiag) {
		t.Fatalf("【V-02 漏洞】同院主任医师通过列表接口绕过了患者 PRIVATE 隐私锁，泄露了私密诊断: %s", wList.Body.String())
	}

	// 同院同科主任医师调用 AI 辅助诊断 POST /api/v1/ai/diagnostic-assist 也严禁引用 PRIVATE 病历
	reqAI := httptest.NewRequest(http.MethodPost, "/api/v1/ai/diagnostic-assist",
		strings.NewReader(fmt.Sprintf(`{"patient_id":%d,"current_symptoms":"胸闷"}`, patZhang.ID)))
	reqAI.Header.Set("Authorization", authHeader(t, &docSeniorSameDept))
	reqAI.Header.Set("Content-Type", "application/json")
	wAI := httptest.NewRecorder()
	r.ServeHTTP(wAI, reqAI)
	if strings.Contains(wAI.Body.String(), secretDiag) {
		t.Fatalf("【V-02 漏洞】AI 辅助诊断绕过了患者 PRIVATE 隐私锁，泄露了私密诊断")
	}
}

// -------------------------------------------------------------------
// V-03: 掩码前 SQL LIKE 关键字盲注探测 (Search Oracle) 防护
// -------------------------------------------------------------------
func TestHighVuln_V03_SearchKeywordOracleBlocked(t *testing.T) {
	r := initHighVulnTestEnv(t)
	docLi, docWang, _, _, _, _ := loadSecuritySeedUsers(t)

	ts := time.Now().UnixNano()
	uniquePhone := fmt.Sprintf("139%08d", ts%100000000)
	uniqueIDCard := fmt.Sprintf("44010119930615%04d", ts%10000)
	patSecret := model.User{
		UserNo:   fmt.Sprintf("PAT_V03_%d", ts),
		Username: fmt.Sprintf("pat_v03_%d", ts),
		RealName: "林隐秘",
		Role:     "patient",
		Phone:    uniquePhone,
		IDCard:   uniqueIDCard,
		Status:   "NORMAL",
	}
	if err := repository.DB.Create(&patSecret).Error; err != nil {
		t.Fatalf("创建测试患者失败: %v", err)
	}
	t.Cleanup(func() {
		repository.DB.Unscoped().Delete(&patSecret)
	})

	oracleToken := fmt.Sprintf("ORACLE_DIAG_%d", ts)
	rec, err := service.DefaultMedicalService.UploadRecord(service.UploadRecordParams{
		DoctorID:  docLi.ID,
		PatientID: patSecret.ID,
		DataType:  "EMR",
		Diagnosis: "特殊隐私病种_" + oracleToken,
		Symptoms:  "隐私症状_" + oracleToken,
	})
	if err != nil || rec == nil {
		t.Fatalf("UploadRecord failed: %v", err)
	}
	t.Cleanup(func() {
		repository.DB.Where("record_id = ?", rec.ID).Delete(&model.MedicalFile{})
		repository.DB.Where("id = ?", rec.ID).Delete(&model.MedicalRecord{})
	})

	// 1. 未授权外院医生 doc_wang 通过 keyword 盲猜患者诊断 -> 结果必须为空数组
	reqOracleDiag := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/v1/medical-records?patient_id=%d&keyword=%s", patSecret.ID, oracleToken), nil)
	reqOracleDiag.Header.Set("Authorization", authHeader(t, docWang))
	wOracleDiag := httptest.NewRecorder()
	r.ServeHTTP(wOracleDiag, reqOracleDiag)
	var diagResp struct {
		Data []model.MedicalRecord `json:"data"`
	}
	_ = json.Unmarshal(wOracleDiag.Body.Bytes(), &diagResp)
	if len(diagResp.Data) > 0 {
		t.Fatalf("【V-03 漏洞】未授权外院医生可通过 keyword 命中未授权诊断（布尔盲注预言机成立），命中 %d 条", len(diagResp.Data))
	}

	// 2. 通过身份证中段子串模糊检索 /api/v1/patients -> 必须不命中（防逐位还原掩码段）
	midIDSubstr := uniqueIDCard[6:14] // 被掩码隐藏的出生日期段 19930615
	reqSubID := httptest.NewRequest(http.MethodGet, "/api/v1/patients?keyword="+midIDSubstr, nil)
	reqSubID.Header.Set("Authorization", authHeader(t, docWang))
	wSubID := httptest.NewRecorder()
	r.ServeHTTP(wSubID, reqSubID)
	if strings.Contains(wSubID.Body.String(), patSecret.UserNo) {
		t.Fatalf("【V-03 漏洞】通过被掩码的身份证子串 %s 模糊探测命中了患者档案", midIDSubstr)
	}

	// 3. 对照组：使用完整手机号精确检索 /api/v1/patients 仍可正常定位患者（且返回已脱敏手机号）
	reqExactPhone := httptest.NewRequest(http.MethodGet, "/api/v1/patients?keyword="+uniquePhone, nil)
	reqExactPhone.Header.Set("Authorization", authHeader(t, docWang))
	wExactPhone := httptest.NewRecorder()
	r.ServeHTTP(wExactPhone, reqExactPhone)
	if wExactPhone.Code != http.StatusOK || !strings.Contains(wExactPhone.Body.String(), patSecret.UserNo) {
		t.Fatalf("完整手机号精确检索应正常命中患者: code=%d body=%s", wExactPhone.Code, wExactPhone.Body.String())
	}
}

// -------------------------------------------------------------------
// V-04: 高危传染病接口：监管员禁止查看临床病原诊断 + 局部授权严禁穿透外院私密病种与单号
// -------------------------------------------------------------------
func TestHighVuln_V04_InfectionRisks_SupervisorBlockedAndPrivateDiseaseRedacted(t *testing.T) {
	r := initHighVulnTestEnv(t)
	docLi, docWang, _, _, supZhao, _ := loadSecuritySeedUsers(t)

	ts := time.Now().UnixNano()
	patTest := model.User{
		UserNo:   fmt.Sprintf("PAT_V04_%d", ts),
		Username: fmt.Sprintf("pat_v04_%d", ts),
		RealName: "周隐私测试",
		Role:     "patient",
		Status:   "NORMAL",
	}
	if err := repository.DB.Create(&patTest).Error; err != nil {
		t.Fatalf("创建测试患者失败: %v", err)
	}
	t.Cleanup(func() {
		repository.DB.Unscoped().Delete(&patTest)
	})

	// 医院 1 (doc_li) 存在一份高度私密的 HIV 病历，且被患者锁定为 PRIVATE
	recHIV, err := service.DefaultMedicalService.UploadRecord(service.UploadRecordParams{
		DoctorID:  docLi.ID,
		PatientID: patTest.ID,
		DataType:  "EMR",
		Diagnosis: "获得性免疫缺陷综合征 HIV抗体阳性",
		Symptoms:  "持续低热消瘦",
	})
	if err != nil || recHIV == nil {
		t.Fatalf("上传 HIV 病历失败: %v", err)
	}

	// 医院 2 (doc_wang) 仅接诊过一次普通感冒
	recCold, err := service.DefaultMedicalService.UploadRecord(service.UploadRecordParams{
		DoctorID:  docWang.ID,
		PatientID: patTest.ID,
		DataType:  "EMR",
		Diagnosis: "急性上呼吸道感染（普通感冒）",
		Symptoms:  "流涕鼻塞1天",
	})
	if err != nil || recCold == nil {
		t.Fatalf("上传感冒病历失败: %v", err)
	}
	t.Cleanup(func() {
		repository.DB.Where("patient_id = ?", patTest.ID).Delete(&model.Authorization{})
		repository.DB.Where("record_id IN ?", []uint64{recHIV.ID, recCold.ID}).Delete(&model.MedicalFile{})
		repository.DB.Where("id IN ?", []uint64{recHIV.ID, recCold.ID}).Delete(&model.MedicalRecord{})
	})

	// 患者锁定医院 1 的 HIV 病历为 PRIVATE
	_, _ = service.DefaultAccessPolicyService.SetRecordAccessPolicy(nil, patTest.ID, recHIV, service.SetRecordAccessPolicyInput{
		RecordID:   recHIV.ID,
		PolicyType: "PRIVATE",
	})

	// 1. 监管员 sup_zhao 调用 /patients/:id/infection-risks 与 /medical-records/:id/infection-risks 必须被 403 拦截
	reqSupPat := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/patients/%d/infection-risks", patTest.ID), nil)
	reqSupPat.Header.Set("Authorization", authHeader(t, supZhao))
	wSupPat := httptest.NewRecorder()
	r.ServeHTTP(wSupPat, reqSupPat)
	if wSupPat.Code != http.StatusForbidden {
		t.Fatalf("【V-04 漏洞】监管员调用患者传染病明细未被 403 拦截，泄露临床隐私: status=%d body=%s", wSupPat.Code, wSupPat.Body.String())
	}

	reqSupRec := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/medical-records/%d/infection-risks", recHIV.ID), nil)
	reqSupRec.Header.Set("Authorization", authHeader(t, supZhao))
	wSupRec := httptest.NewRecorder()
	r.ServeHTTP(wSupRec, reqSupRec)
	if wSupRec.Code != http.StatusForbidden {
		t.Fatalf("【V-04 漏洞】监管员调用单病历传染病明细未被 403 拦截: status=%d body=%s", wSupRec.Code, wSupRec.Body.String())
	}

	// 监管员调用 GET /api/v1/medical-records/:id 详情时，也不得通过内嵌 infection_alert 泄露 HIV 病名
	reqSupDetail := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/medical-records/%d", recHIV.ID), nil)
	reqSupDetail.Header.Set("Authorization", authHeader(t, supZhao))
	wSupDetail := httptest.NewRecorder()
	r.ServeHTTP(wSupDetail, reqSupDetail)
	if strings.Contains(strings.ToUpper(wSupDetail.Body.String()), "HIV") || strings.Contains(wSupDetail.Body.String(), "艾滋") {
		t.Fatalf("【V-04 漏洞】监管员查阅脱敏病历详情时仍通过内嵌字段泄露了 HIV 诊断: %s", wSupDetail.Body.String())
	}

	// 2. 外院仅治疗普通感冒的 doc_wang 调用 /patients/:id/infection-risks：
	//    应当保留脱敏职业暴露三级防护提醒 (has_risk=true)，但严禁暴露外院 HIV 病名与外院单号 recHIV.RecordNo！
	reqDocWang := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/patients/%d/infection-risks", patTest.ID), nil)
	reqDocWang.Header.Set("Authorization", authHeader(t, docWang))
	wDocWang := httptest.NewRecorder()
	r.ServeHTTP(wDocWang, reqDocWang)
	if wDocWang.Code != http.StatusOK {
		t.Fatalf("医生查询职业暴露风险状态异常: code=%d body=%s", wDocWang.Code, wDocWang.Body.String())
	}
	bodyUpper := strings.ToUpper(wDocWang.Body.String())
	if strings.Contains(bodyUpper, "HIV") || strings.Contains(wDocWang.Body.String(), "艾滋") || strings.Contains(wDocWang.Body.String(), recHIV.RecordNo) {
		t.Fatalf("【V-04 漏洞】仅具感冒病历权限的外院医生穿透获知了患者 PRIVATE 锁定的 HIV 诊断或单号: %s", wDocWang.Body.String())
	}
}

// -------------------------------------------------------------------
// V-05: 生产环境跳过弱口令种子重置 + MustChangePwd 服务端强拦截
// -------------------------------------------------------------------
func TestHighVuln_V05_ProductionSkipsWeakSeedAndEnforcesMustChangePwd(t *testing.T) {
	r := initHighVulnTestEnv(t)
	docLi, _, _, _, _, _ := loadSecuritySeedUsers(t)

	origEnv := config.AppConfig.AppEnv
	t.Cleanup(func() {
		config.AppConfig.AppEnv = origEnv
	})

	// 1. 生产模式下调用 SeedInitialData 不得清空已有账号的失败锁定计数或注入默认弱口令
	config.AppConfig.AppEnv = "production"
	lockTime := time.Now().Add(10 * time.Minute)
	repository.DB.Model(&model.User{}).Where("id = ?", docLi.ID).Updates(map[string]interface{}{
		"failed_login_count": 5,
		"locked_until":       lockTime,
	})
	t.Cleanup(func() {
		repository.DB.Model(&model.User{}).Where("id = ?", docLi.ID).Updates(map[string]interface{}{
			"failed_login_count": 0,
			"locked_until":       nil,
		})
	})

	repository.SeedInitialData(repository.DB)

	var checkDoc model.User
	repository.DB.First(&checkDoc, docLi.ID)
	if checkDoc.FailedLoginCount == 0 {
		t.Fatalf("【V-05 漏洞】生产模式下 SeedInitialData 仍在执行并清空了防爆破锁定状态")
	}

	// 2. MustChangePwd=true 的临时密码账号在生产模式下未改密前访问业务接口必须被 403 拦截，仅允许改密与登出
	ts := time.Now().UnixNano()
	tempPwd := "TempPass2026!"
	pwdHash, _ := bcrypt.GenerateFromPassword([]byte(tempPwd), bcrypt.DefaultCost)
	tempUser := model.User{
		UserNo:        fmt.Sprintf("DOC_TMP_%d", ts),
		Username:      fmt.Sprintf("doc_tmp_%d", ts),
		PasswordHash:  string(pwdHash),
		RealName:      "待改密新医生",
		Role:          "doctor",
		HospitalID:    1,
		DepartmentID:  1,
		MustChangePwd: true,
		Status:        "NORMAL",
	}
	if err := repository.DB.Create(&tempUser).Error; err != nil {
		t.Fatalf("创建临时密码用户失败: %v", err)
	}
	t.Cleanup(func() {
		repository.DB.Unscoped().Delete(&tempUser)
	})

	tempToken := authHeader(t, &tempUser)

	// 未改密前尝试访问业务接口 GET /api/v1/system/hospitals -> 必须返回 403
	reqBiz := httptest.NewRequest(http.MethodGet, "/api/v1/system/hospitals", nil)
	reqBiz.Header.Set("Authorization", tempToken)
	wBiz := httptest.NewRecorder()
	r.ServeHTTP(wBiz, reqBiz)
	if wBiz.Code != http.StatusForbidden {
		t.Fatalf("【V-05 漏洞】MustChangePwd=true 账号未改密即可直接调用业务 API: status=%d body=%s", wBiz.Code, wBiz.Body.String())
	}

	// 允许调用 PUT /api/v1/auth/password 完成首次改密
	changeBody, _ := json.Marshal(map[string]string{
		"old_password": tempPwd,
		"new_password": "NewStrongPass2026#",
	})
	reqChange := httptest.NewRequest(http.MethodPut, "/api/v1/auth/password", bytes.NewReader(changeBody))
	reqChange.Header.Set("Authorization", tempToken)
	reqChange.Header.Set("Content-Type", "application/json")
	wChange := httptest.NewRecorder()
	r.ServeHTTP(wChange, reqChange)
	if wChange.Code != http.StatusOK {
		t.Fatalf("MustChangePwd=true 账号调用改密接口被意外阻断: status=%d body=%s", wChange.Code, wChange.Body.String())
	}

	// 改密完成后重新获取令牌，访问业务接口应正常 200 放行
	var refreshed model.User
	repository.DB.First(&refreshed, tempUser.ID)
	newToken := authHeader(t, &refreshed)
	reqBizAfter := httptest.NewRequest(http.MethodGet, "/api/v1/system/hospitals", nil)
	reqBizAfter.Header.Set("Authorization", newToken)
	wBizAfter := httptest.NewRecorder()
	r.ServeHTTP(wBizAfter, reqBizAfter)
	if wBizAfter.Code != http.StatusOK {
		t.Fatalf("改密完成后业务接口应正常放行 200，实际: status=%d body=%s", wBizAfter.Code, wBizAfter.Body.String())
	}
}
