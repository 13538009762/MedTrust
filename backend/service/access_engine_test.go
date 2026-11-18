package service_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"medtrust-backend/config"
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
	"medtrust-backend/service"
)

func TestAccessEngine_AuthorizationAndRiskFlow(t *testing.T) {
	cfg, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("Load config failed: %v", err)
	}
	_, err = repository.InitDB()
	if err != nil {
		t.Fatalf("Init DB failed: %v", err)
	}
	service.InitAccessEngine(cfg.Risk.LowThreshold, cfg.Risk.HighThreshold)

	engine := service.DefaultAccessEngine
	if engine == nil {
		t.Fatalf("DefaultAccessEngine not initialized")
	}
	// 注入工作时段时钟(14:00)，避免因非工作时段(RULE-T1: <8或>=18)触发加分导致中风险拦截
	mockNow := time.Date(2026, 10, 26, 14, 0, 0, 0, time.Local)
	engine.SetClock(func() time.Time { return mockNow })
	defer engine.SetClock(nil)

	// 准备测试数据: 医院、科室、医生、患者（赋予唯一 HospitalNo 防止唯一索引冲突）
	ts := time.Now().UnixNano()
	hospA := model.Hospital{HospitalNo: fmt.Sprintf("HOSP_TA_%d", ts), Name: "测试医院A", Level: "三甲", Status: 1}
	hospB := model.Hospital{HospitalNo: fmt.Sprintf("HOSP_TB_%d", ts), Name: "测试医院B", Level: "三甲", Status: 1}
	repository.DB.Create(&hospA)
	repository.DB.Create(&hospB)
	defer func() {
		if hospA.ID > 0 {
			repository.DB.Delete(&hospA)
		}
		if hospB.ID > 0 {
			repository.DB.Delete(&hospB)
		}
	}()

	deptA := model.Department{HospitalID: hospA.ID, Name: "心血管内科", DeptNo: "CARDIO_A"}
	repository.DB.Create(&deptA)
	defer repository.DB.Delete(&deptA)

	docA := model.User{
		UserNo:       "DOC_A_TEST",
		Username:     "DOC_A_TEST",
		RealName:     "张主任",
		Role:         "doctor",
		HospitalID:   hospA.ID,
		DepartmentID: deptA.ID,
		Title:        "主任医师",
		Status:       "NORMAL",
	}
	docB := model.User{
		UserNo:       "DOC_B_TEST",
		Username:     "DOC_B_TEST",
		RealName:     "李医生(外院)",
		Role:         "doctor",
		HospitalID:   hospB.ID,
		DepartmentID: 0,
		Title:        "主治医师",
		Status:       "NORMAL",
	}
	docRestricted := model.User{
		UserNo:     "DOC_RESTRICTED_TEST",
		Username:   "DOC_RESTRICTED_TEST",
		RealName:   "违规受限医生",
		Role:       "doctor",
		HospitalID: hospB.ID,
		Title:      "住院医师",
		Status:     "RESTRICTED",
	}
	patient := model.User{
		UserNo:   "PAT_TEST_001",
		Username: "PAT_TEST_001",
		RealName: "测试患者王五",
		Role:     "patient",
		Status:   "NORMAL",
	}
	repository.DB.Create(&docA)
	repository.DB.Create(&docB)
	repository.DB.Create(&docRestricted)
	repository.DB.Create(&patient)
	defer repository.DB.Delete(&docA)
	defer repository.DB.Delete(&docB)
	defer repository.DB.Delete(&docRestricted)
	defer repository.DB.Delete(&patient)

	record := model.MedicalRecord{
		RecordNo:       "REC_TEST_001",
		PatientID:      patient.ID,
		DoctorID:       docA.ID,
		HospitalID:     hospA.ID,
		DepartmentName: deptA.Name,
		Diagnosis:      "冠状动脉粥样硬化性心脏病",
		DataType:       "住院",
		CreatedAt:      time.Now(),
	}
	repository.DB.Create(&record)
	defer repository.DB.Delete(&record)

	// 场景 1: 本人开具的病历直接放行
	dec1, err := engine.EvaluateAccess(docA.ID, patient.ID, record.ID, false)
	if err != nil || !dec1.Allowed || dec1.Decision != "ALLOWED" {
		t.Fatalf("场景1 本人开具病历放行失败: allowed=%v, decision=%s, err=%v", dec1.Allowed, dec1.Decision, err)
	}

	// 场景 2: 受限医生尝试使用 Break-Glass 紧急抢救权限被拦截
	dec2, _ := engine.EvaluateAccess(docRestricted.ID, patient.ID, record.ID, true)
	if dec2.Allowed || dec2.Decision != "REJECTED" {
		t.Fatalf("场景2 受限医生应被阻断紧急访问: allowed=%v, decision=%s", dec2.Allowed, dec2.Decision)
	}

	// 场景 3: 外院医生无授权调阅（非紧急），提示 NEED_BREAK_GLASS
	dec3, _ := engine.EvaluateAccess(docB.ID, patient.ID, record.ID, false)
	if dec3.Allowed || dec3.Decision != "NEED_BREAK_GLASS" {
		t.Fatalf("场景3 外院无授权非紧急预期 NEED_BREAK_GLASS: allowed=%v, decision=%s", dec3.Allowed, dec3.Decision)
	}

	// 场景 4: 外院医生触发 Break-Glass 紧急访问通道放行
	dec4, _ := engine.EvaluateAccess(docB.ID, patient.ID, record.ID, true)
	if !dec4.Allowed || dec4.Decision != "BREAK_GLASS_ALLOWED" || !dec4.IsEmergency {
		t.Fatalf("场景4 外院 Break-Glass 预期放行: allowed=%v, decision=%s", dec4.Allowed, dec4.Decision)
	}

	// 场景 5: 患者在线显式颁发授权凭证后，外院常规调阅放行
	auth := model.Authorization{
		PatientID:      patient.ID,
		AuthTargetType: "DOCTOR",
		AuthTargetID:   docB.ID,
		ScopeType:      "SINGLE",
		RecordID:       record.ID,
		StartTime:      mockNow.Add(-24 * time.Hour),
		EndTime:        mockNow.Add(48 * time.Hour),
		Status:         "ACTIVE",
		CreatedAt:      time.Now(),
	}
	repository.DB.Create(&auth)
	defer repository.DB.Delete(&auth)

	dec5, _ := engine.EvaluateAccess(docB.ID, patient.ID, record.ID, false)
	if !dec5.Allowed || dec5.Decision != "ALLOWED" {
		t.Fatalf("场景5 患者显式授权后预期放行: allowed=%v, decision=%s, reason=%s", dec5.Allowed, dec5.Decision, dec5.Reason)
	}
}

// TestAccessEngine_DetailedSecurityRules 深度验证访问控制引擎多层边界防护规则：
// 1. Action 动作白名单校验与未知动作拦截
// 2. 伪造身份/角色篡改防御 (DB 真实角色 vs 请求声明角色)
// 3. 水平越权 (IDOR) 防御 (病历所属 patient_id 与请求参数 patient_id 冲突)
// 4. 管理员职能隔离 (禁止调阅/下载任何临床病历)
// 5. 监管人员权限边界 (禁止下载临床病历附件原文)
// 6. 患者自主隔离 (严禁查阅他人病历)
func TestAccessEngine_DetailedSecurityRules(t *testing.T) {
	_, _ = config.LoadConfig("../config/config.yaml")
	_, _ = repository.InitDB()
	engine := service.DefaultAccessEngine

	// 准备基准用户和数据
	patA := model.User{
		UserNo:   "PAT_SEC_A",
		Username: "PAT_SEC_A",
		RealName: "患者甲",
		Role:     "patient",
		Status:   "NORMAL",
	}
	patB := model.User{
		UserNo:   "PAT_SEC_B",
		Username: "PAT_SEC_B",
		RealName: "患者乙",
		Role:     "patient",
		Status:   "NORMAL",
	}
	adminUser := model.User{
		UserNo:   "ADM_SEC_01",
		Username: "ADM_SEC_01",
		RealName: "系统管理员",
		Role:     "admin",
		Status:   "NORMAL",
	}
	supervisorUser := model.User{
		UserNo:   "SUP_SEC_01",
		Username: "SUP_SEC_01",
		RealName: "监管员李四",
		Role:     "supervisor",
		Status:   "NORMAL",
	}
	docUser := model.User{
		UserNo:     "DOC_SEC_01",
		Username:   "DOC_SEC_01",
		RealName:   "主治医生赵六",
		Role:       "doctor",
		HospitalID: 1,
		Status:     "NORMAL",
	}
	repository.DB.Create(&patA)
	repository.DB.Create(&patB)
	repository.DB.Create(&adminUser)
	repository.DB.Create(&supervisorUser)
	repository.DB.Create(&docUser)
	defer repository.DB.Delete(&patA)
	defer repository.DB.Delete(&patB)
	defer repository.DB.Delete(&adminUser)
	defer repository.DB.Delete(&supervisorUser)
	defer repository.DB.Delete(&docUser)

	recA := model.MedicalRecord{
		RecordNo:  "REC_SEC_A",
		PatientID: patA.ID,
		DoctorID:  docUser.ID,
		Diagnosis: "急性胃炎",
	}
	repository.DB.Create(&recA)
	defer repository.DB.Delete(&recA)

	// 1. Action 白名单校验：非法动作注入应直接被阻断
	decUnknownAction, err := engine.Evaluate(service.AccessRequest{
		UserID:    docUser.ID,
		Role:      "doctor",
		PatientID: patA.ID,
		RecordID:  recA.ID,
		Action:    "DROP_DATABASE",
	})
	if err != nil || decUnknownAction.Allowed || decUnknownAction.Decision != "REJECTED" || !strings.Contains(decUnknownAction.Reason, "不在安全白名单内") {
		t.Fatalf("Action 动作白名单防御失效: allowed=%v, reason=%s", decUnknownAction.Allowed, decUnknownAction.Reason)
	}

	// 2. 伪造角色身份攻击：DB 中是 patient，请求中冒充 doctor
	decRoleSpoof, err := engine.Evaluate(service.AccessRequest{
		UserID:    patA.ID,
		Role:      "doctor", // 伪造角色
		PatientID: patA.ID,
		RecordID:  recA.ID,
		Action:    "VIEW",
	})
	if err != nil || decRoleSpoof.Allowed || decRoleSpoof.Decision != "REJECTED" || !strings.Contains(decRoleSpoof.Reason, "角色伪造拦截") {
		t.Fatalf("角色篡改伪造防御失效: allowed=%v, reason=%s", decRoleSpoof.Allowed, decRoleSpoof.Reason)
	}

	// 3. 水平越权 (IDOR) 攻击：病历属于 patA，请求声称是 patB 的病历
	decIDOR, err := engine.Evaluate(service.AccessRequest{
		UserID:    docUser.ID,
		Role:      "doctor",
		PatientID: patB.ID, // 参数篡改
		RecordID:  recA.ID, // 实际上属于 patA
		Action:    "VIEW",
	})
	if err != nil || decIDOR.Allowed || decIDOR.Decision != "REJECTED" || !strings.Contains(decIDOR.Reason, "IDOR") {
		t.Fatalf("IDOR 越权调阅防御失效: allowed=%v, reason=%s", decIDOR.Allowed, decIDOR.Reason)
	}

	// 4. 管理员职能隔离：禁止任何系统管理员越权调阅临床病历
	decAdminView, err := engine.Evaluate(service.AccessRequest{
		UserID:    adminUser.ID,
		Role:      "admin",
		PatientID: patA.ID,
		RecordID:  recA.ID,
		Action:    "VIEW",
	})
	if err != nil || decAdminView.Allowed || decAdminView.Decision != "REJECTED" || !strings.Contains(decAdminView.Reason, "禁止系统管理员直接调阅或下载患者临床隐私明文") {
		t.Fatalf("管理员越权调阅临床病历防御失效: allowed=%v, reason=%s", decAdminView.Allowed, decAdminView.Reason)
	}

	// 5. 监管人员权限边界：允许合规监管核验，但严格禁止下载患者临床附件原文
	decSupervisorDownload, err := engine.Evaluate(service.AccessRequest{
		UserID:    supervisorUser.ID,
		Role:      "supervisor",
		PatientID: patA.ID,
		RecordID:  recA.ID,
		Action:    "DOWNLOAD",
	})
	if err != nil || decSupervisorDownload.Allowed || decSupervisorDownload.Decision != "REJECTED" || !strings.Contains(decSupervisorDownload.Reason, "监管人员禁止下载临床原始诊疗附件") {
		t.Fatalf("监管员下载附件限制失效: allowed=%v, reason=%s", decSupervisorDownload.Allowed, decSupervisorDownload.Reason)
	}

	// 6. 患者横向隔离：患者 B 试图直接调阅患者 A 的病历
	decPatientCross, err := engine.Evaluate(service.AccessRequest{
		UserID:    patB.ID,
		Role:      "patient",
		PatientID: patA.ID,
		RecordID:  recA.ID,
		Action:    "VIEW",
	})
	if err != nil || decPatientCross.Allowed || decPatientCross.Decision != "REJECTED" || !strings.Contains(decPatientCross.Reason, "水平越权拦截(IDOR)") {
		t.Fatalf("患者跨账号越权调阅防御失效: allowed=%v, reason=%s", decPatientCross.Allowed, decPatientCross.Reason)
	}
}

// TestAccessEngine_ExpiredAuthorizationDenied 验证授权时间窗边界：
// 已过期的 ACTIVE 授权不得放行（防止 ||/&& 优先级 bug 导致过期凭证永续可用），
// 处于有效期内的授权方可正常放行。
func TestAccessEngine_ExpiredAuthorizationDenied(t *testing.T) {
	cfg, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("Load config failed: %v", err)
	}
	if _, err = repository.InitDB(); err != nil {
		t.Fatalf("Init DB failed: %v", err)
	}
	service.InitAccessEngine(cfg.Risk.LowThreshold, cfg.Risk.HighThreshold)
	engine := service.DefaultAccessEngine

	mockNow := time.Date(2026, 10, 26, 14, 0, 0, 0, time.Local)
	engine.SetClock(func() time.Time { return mockNow })
	defer engine.SetClock(nil)

	ts := time.Now().UnixNano()
	hospA := model.Hospital{HospitalNo: fmt.Sprintf("HOSP_EXP_A_%d", ts), Name: "授权测试医院A", Level: "三甲", Status: 1}
	hospB := model.Hospital{HospitalNo: fmt.Sprintf("HOSP_EXP_B_%d", ts), Name: "授权测试医院B", Level: "三甲", Status: 1}
	repository.DB.Create(&hospA)
	repository.DB.Create(&hospB)
	defer repository.DB.Delete(&hospA)
	defer repository.DB.Delete(&hospB)

	patient := model.User{UserNo: fmt.Sprintf("PAT_EXP_%d", ts), Username: fmt.Sprintf("PAT_EXP_%d", ts), RealName: "授权测试患者", Role: "patient", Status: "NORMAL"}
	docOwner := model.User{UserNo: fmt.Sprintf("DOC_OWN_%d", ts), Username: fmt.Sprintf("DOC_OWN_%d", ts), RealName: "开单医生", Role: "doctor", HospitalID: hospA.ID, Status: "NORMAL"}
	docOut := model.User{UserNo: fmt.Sprintf("DOC_OUT_%d", ts), Username: fmt.Sprintf("DOC_OUT_%d", ts), RealName: "外院医生", Role: "doctor", HospitalID: hospB.ID, Title: "主治医师", Status: "NORMAL"}
	repository.DB.Create(&patient)
	repository.DB.Create(&docOwner)
	repository.DB.Create(&docOut)
	defer repository.DB.Delete(&patient)
	defer repository.DB.Delete(&docOwner)
	defer repository.DB.Delete(&docOut)

	record := model.MedicalRecord{
		RecordNo:       fmt.Sprintf("REC_EXP_%d", ts),
		PatientID:      patient.ID,
		DoctorID:       docOwner.ID,
		HospitalID:     hospA.ID,
		DepartmentName: "消化内科",
		Diagnosis:      "慢性胃炎",
		CreatedAt:      time.Now(),
	}
	repository.DB.Create(&record)
	defer repository.DB.Delete(&record)

	// 过期授权：StartTime 在过去，但 EndTime 也已过期
	expiredAuth := model.Authorization{
		PatientID:      patient.ID,
		AuthTargetType: "DOCTOR",
		AuthTargetID:   docOut.ID,
		ScopeType:      "SINGLE",
		RecordID:       record.ID,
		StartTime:      mockNow.Add(-72 * time.Hour),
		EndTime:        mockNow.Add(-24 * time.Hour), // 已于 24 小时前过期
		Status:         "ACTIVE",
		CreatedAt:      time.Now(),
	}
	repository.DB.Create(&expiredAuth)
	defer repository.DB.Delete(&expiredAuth)

	decExpired, err := engine.EvaluateAccess(docOut.ID, patient.ID, record.ID, false)
	if err != nil {
		t.Fatalf("过期授权场景评估异常: %v", err)
	}
	if decExpired.Allowed || decExpired.Decision != "NEED_BREAK_GLASS" {
		t.Fatalf("过期授权不应放行: allowed=%v, decision=%s, reason=%s", decExpired.Allowed, decExpired.Decision, decExpired.Reason)
	}

	// 将同一授权续期至有效窗口后，应恢复放行
	repository.DB.Model(&model.Authorization{}).Where("id = ?", expiredAuth.ID).Updates(map[string]interface{}{
		"start_time": mockNow.Add(-1 * time.Hour),
		"end_time":   mockNow.Add(48 * time.Hour),
	})
	decValid, err := engine.EvaluateAccess(docOut.ID, patient.ID, record.ID, false)
	if err != nil || !decValid.Allowed {
		t.Fatalf("有效期内授权应放行: allowed=%v, decision=%s, err=%v", decValid.Allowed, decValid.Decision, err)
	}
}

// TestAccessEngine_PrivatePolicyUnblockedByConsentAndKeyUnlock 验证：
// 1. 病历设为 PRIVATE 后，即使存在早期全局 ALL_DOCTORS 授权，未获单独同意的外院医生仍被阻断；
// 2. 患者审批单次申请 (ApproveRequest) 或现场密钥解锁 (UnlockByKey / UnlockPatientByKey) 后，
//    该医生立即可通过 AccessEngine.Evaluate、GetRecords 与 AI Scope 核验调阅，其他医生仍受 PRIVATE 保护；
// 3. 随后若患者重新对该病历执行 SetRecordAccessPolicy("PRIVATE")，先前 ALL 密钥解锁亦被新 PRIVATE 锁定阻断。
func TestAccessEngine_PrivatePolicyUnblockedByConsentAndKeyUnlock(t *testing.T) {
	cfg, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("Load config failed: %v", err)
	}
	if _, err = repository.InitDB(); err != nil {
		t.Fatalf("Init DB failed: %v", err)
	}
	tempDir := t.TempDir()
	blockchain.InitBlockchainService(filepath.Join(tempDir, "ledger"), blockchain.FabricGatewayConfig{Mode: "mock"})
	service.InitMedicalService("mock", cfg.IPFS.APIURL, filepath.Join(tempDir, "ipfs"))
	service.InitAccessEngine(cfg.Risk.LowThreshold, cfg.Risk.HighThreshold)

	engine := service.DefaultAccessEngine
	// 使用明天 14:00 作为模拟时钟：既落在工作时段 (08:00-18:00) 内，又晚于 ApproveRequest / UnlockByKey 写入的 StartTime(time.Now())
	nowReal := time.Now()
	mockNow := time.Date(nowReal.Year(), nowReal.Month(), nowReal.Day(), 14, 0, 0, 0, nowReal.Location()).AddDate(0, 0, 1)
	engine.SetClock(func() time.Time { return mockNow })
	defer engine.SetClock(nil)

	ts := time.Now().UnixNano()
	hospA := model.Hospital{HospitalNo: fmt.Sprintf("HOSP_PRV_A_%d", ts), Name: "私密测试医院A", Level: "三甲", Status: 1}
	hospB := model.Hospital{HospitalNo: fmt.Sprintf("HOSP_PRV_B_%d", ts), Name: "私密测试医院B", Level: "三甲", Status: 1}
	hospC := model.Hospital{HospitalNo: fmt.Sprintf("HOSP_PRV_C_%d", ts), Name: "私密测试医院C", Level: "三甲", Status: 1}
	repository.DB.Create(&hospA)
	repository.DB.Create(&hospB)
	repository.DB.Create(&hospC)
	defer func() {
		repository.DB.Delete(&hospA)
		repository.DB.Delete(&hospB)
		repository.DB.Delete(&hospC)
	}()

	patUserNo := fmt.Sprintf("PAT_PRV_%d", ts)
	pat := model.User{
		UserNo:         patUserNo,
		Username:       patUserNo,
		RealName:       "私密测试患者",
		Role:           "patient",
		Status:         "NORMAL",
		MedicalKeyHash: service.HashMedicalKey("654321", patUserNo),
	}
	docOwner := model.User{UserNo: fmt.Sprintf("DOC_PRV_OWN_%d", ts), Username: fmt.Sprintf("DOC_PRV_OWN_%d", ts), RealName: "主诊医生", Role: "doctor", HospitalID: hospA.ID, Status: "NORMAL"}
	docApproved := model.User{UserNo: fmt.Sprintf("DOC_PRV_APP_%d", ts), Username: fmt.Sprintf("DOC_PRV_APP_%d", ts), RealName: "获批外院医生", Role: "doctor", HospitalID: hospB.ID, Status: "NORMAL"}
	docOther := model.User{UserNo: fmt.Sprintf("DOC_PRV_OTH_%d", ts), Username: fmt.Sprintf("DOC_PRV_OTH_%d", ts), RealName: "未获批外院医生", Role: "doctor", HospitalID: hospC.ID, Status: "NORMAL"}
	repository.DB.Create(&pat)
	repository.DB.Create(&docOwner)
	repository.DB.Create(&docApproved)
	repository.DB.Create(&docOther)
	defer func() {
		repository.DB.Where("patient_id = ?", pat.ID).Delete(&model.Authorization{})
		repository.DB.Where("patient_id = ?", pat.ID).Delete(&model.AccessRequest{})
		repository.DB.Delete(&pat)
		repository.DB.Delete(&docOwner)
		repository.DB.Delete(&docApproved)
		repository.DB.Delete(&docOther)
	}()

	rec := model.MedicalRecord{
		RecordNo:       fmt.Sprintf("REC_PRV_%d", ts),
		PatientID:      pat.ID,
		DoctorID:       docOwner.ID,
		HospitalID:     hospA.ID,
		DepartmentName: "内分泌科",
		Diagnosis:      "2型糖尿病",
		CreatedAt:      mockNow,
	}
	repository.DB.Create(&rec)
	defer repository.DB.Delete(&rec)

	// 1. 先设全局 ALL_DOCTORS，再将单份病历锁定为 PRIVATE
	if _, err := service.DefaultAccessPolicyService.BatchSetAccessPolicy(nil, pat.ID, service.BatchSetAccessPolicyInput{PolicyType: "ALL_DOCTORS", Days: 30}); err != nil {
		t.Fatalf("BatchSetAccessPolicy ALL_DOCTORS failed: %v", err)
	}
	if _, err := service.DefaultAccessPolicyService.SetRecordAccessPolicy(nil, pat.ID, &rec, service.SetRecordAccessPolicyInput{RecordID: rec.ID, PolicyType: "PRIVATE"}); err != nil {
		t.Fatalf("SetRecordAccessPolicy PRIVATE failed: %v", err)
	}

	// 验证 PRIVATE 生效：两外院医生均被阻断
	if dec, _ := engine.EvaluateAccess(docApproved.ID, pat.ID, rec.ID, false); dec.Allowed {
		t.Fatalf("PRIVATE 病历不应允许 docApproved 在审批前调阅")
	}

	// 2. 患者审批 docApproved 的单份调阅申请 (ApproveRequest)
	reqRec := model.AccessRequest{
		RequestNo: fmt.Sprintf("REQ-PRV-%d", ts),
		DoctorID:  docApproved.ID,
		PatientID: pat.ID,
		RecordID:  rec.ID,
		ScopeType: "SINGLE",
		Status:    "PENDING",
		Days:      7,
		ExpireAt:  time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}
	repository.DB.Create(&reqRec)
	if _, err := service.DefaultAccessRequestService.ApproveRequest(nil, pat.ID, &reqRec); err != nil {
		t.Fatalf("ApproveRequest failed: %v", err)
	}

	// 2a. AccessEngine 应放行 docApproved，继续拦截 docOther
	if dec, _ := engine.EvaluateAccess(docApproved.ID, pat.ID, rec.ID, false); !dec.Allowed {
		t.Fatalf("ApproveRequest 后 docApproved 仍被 PRIVATE 死锁拦截: decision=%s reason=%s", dec.Decision, dec.Reason)
	}
	if dec, _ := engine.EvaluateAccess(docOther.ID, pat.ID, rec.ID, false); dec.Allowed {
		t.Fatalf("未获批的 docOther 不应受全局 ALL_DOCTORS 漏出而获得 PRIVATE 病历访问权")
	}

	// 2b. GetRecords 列表视图与 AI Scope 校验同样放行 docApproved、拦截 docOther
	listApp, err := service.DefaultMedicalService.GetRecords(pat.ID, 0, 0, 0, docApproved.ID, false, "", "", 1, 20)
	if err != nil || len(listApp) != 1 || !listApp[0].HasAccess {
		t.Fatalf("GetRecords 中 docApproved 预期 HasAccess=true，实际: err=%v list=%+v", err, listApp)
	}
	listOth, err := service.DefaultMedicalService.GetRecords(pat.ID, 0, 0, 0, docOther.ID, false, "", "", 1, 20)
	if err != nil || len(listOth) != 1 || listOth[0].HasAccess {
		t.Fatalf("GetRecords 中 docOther 预期 HasAccess=false，实际: err=%v list=%+v", err, listOth)
	}
	if cnt := repository.DefaultAuthPolicyRepository.CountActiveConsentAuthorizations(pat.ID, docApproved.ID, docApproved.HospitalID, rec.ID, time.Now()); cnt == 0 {
		t.Fatalf("CountActiveConsentAuthorizations 预期对 docApproved 返回有效授权")
	}

	// 3. 重新设为 PRIVATE 后，测试 UnlockPatientByKey (ScopeType=ALL) 也可解锁，且再次设为 PRIVATE 可重新上锁
	if _, err := service.DefaultAccessPolicyService.SetRecordAccessPolicy(nil, pat.ID, &rec, service.SetRecordAccessPolicyInput{RecordID: rec.ID, PolicyType: "PRIVATE"}); err != nil {
		t.Fatalf("re-lock PRIVATE failed: %v", err)
	}
	if dec, _ := engine.EvaluateAccess(docApproved.ID, pat.ID, rec.ID, false); dec.Allowed {
		t.Fatalf("重新设为 PRIVATE 后，先前的审批授权必须被撤销拦截")
	}
	if _, _, err := service.DefaultKeyUnlockService.UnlockPatientByKey(nil, docOther.ID, service.UnlockPatientByKeyInput{
		PatientID:  pat.ID,
		MedicalKey: "654321",
		Days:       7,
	}); err != nil {
		t.Fatalf("UnlockPatientByKey failed: %v", err)
	}
	if dec, _ := engine.EvaluateAccess(docOther.ID, pat.ID, rec.ID, false); !dec.Allowed {
		t.Fatalf("UnlockPatientByKey 解锁后 docOther 仍被 PRIVATE 死锁拦截: decision=%s reason=%s", dec.Decision, dec.Reason)
	}
	listOth2, _ := service.DefaultMedicalService.GetRecords(pat.ID, 0, 0, 0, docOther.ID, false, "", "", 1, 20)
	if len(listOth2) != 1 || !listOth2[0].HasAccess {
		t.Fatalf("UnlockPatientByKey 后 GetRecords 预期 HasAccess=true")
	}

	// 4. 再次将该病历设为 PRIVATE 时，先前 UnlockPatientByKey 的 ALL 授权对该单份病历亦应受控锁定；
	//    随后使用 UnlockByKey (单份密钥解锁) 应再次解锁 docApproved 而不放行 docOther
	if _, err := service.DefaultAccessPolicyService.SetRecordAccessPolicy(nil, pat.ID, &rec, service.SetRecordAccessPolicyInput{RecordID: rec.ID, PolicyType: "PRIVATE"}); err != nil {
		t.Fatalf("final PRIVATE lock failed: %v", err)
	}
	if dec, _ := engine.EvaluateAccess(docOther.ID, pat.ID, rec.ID, false); dec.Allowed {
		t.Fatalf("再次设为 PRIVATE 后，应当优先锁定该单份病历")
	}
	if _, err := service.DefaultKeyUnlockService.UnlockByKey(nil, docApproved.ID, service.UnlockByKeyInput{
		RecordID:   rec.ID,
		MedicalKey: "654321",
		Days:       7,
	}); err != nil {
		t.Fatalf("UnlockByKey failed: %v", err)
	}
	if dec, _ := engine.EvaluateAccess(docApproved.ID, pat.ID, rec.ID, false); !dec.Allowed {
		t.Fatalf("UnlockByKey 解锁单份 PRIVATE 病历后 docApproved 仍被拦截: decision=%s reason=%s", dec.Decision, dec.Reason)
	}
	if dec, _ := engine.EvaluateAccess(docOther.ID, pat.ID, rec.ID, false); dec.Allowed {
		t.Fatalf("UnlockByKey 解锁后其他医生 docOther 仍应被 PRIVATE 拦截")
	}
}

// TestAccessEngine_IntraHospitalRiskEvaluationAndBreakGlassTelemetry 验证：
// 1. 同院同科室/高年资医生不得绕过 RiskEngine (如 RESTRICTED + 批量访问仍受动态风控拦截)；
// 2. Break-Glass 紧急放行时 RiskScore 对外保持 0/EMERGENCY，但写入 risk_records 的遥测评分与因子反映真实风险。
func TestAccessEngine_IntraHospitalRiskEvaluationAndBreakGlassTelemetry(t *testing.T) {
	cfg, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("Load config failed: %v", err)
	}
	if _, err = repository.InitDB(); err != nil {
		t.Fatalf("Init DB failed: %v", err)
	}
	tempDir := t.TempDir()
	blockchain.InitBlockchainService(filepath.Join(tempDir, "ledger"), blockchain.FabricGatewayConfig{Mode: "mock"})
	service.InitAccessEngine(cfg.Risk.LowThreshold, cfg.Risk.HighThreshold)

	engine := service.DefaultAccessEngine
	nightClock := time.Date(2026, 5, 10, 2, 30, 0, 0, time.Local)
	engine.SetClock(func() time.Time { return nightClock })
	defer engine.SetClock(nil)

	ts := time.Now().UnixNano()
	hosp := model.Hospital{HospitalNo: fmt.Sprintf("HOSP_IH_%d", ts), Name: "同院风控测试医院", Level: "三甲", Status: 1}
	repository.DB.Create(&hosp)
	dept := model.Department{HospitalID: hosp.ID, DeptNo: fmt.Sprintf("D_%d", ts), Name: "心内科"}
	repository.DB.Create(&dept)
	defer func() {
		repository.DB.Delete(&dept)
		repository.DB.Delete(&hosp)
	}()

	pat := model.User{UserNo: fmt.Sprintf("PAT_IH_%d", ts), Username: fmt.Sprintf("PAT_IH_%d", ts), RealName: "同院患者", Role: "patient", Status: "NORMAL"}
	docCreator := model.User{UserNo: fmt.Sprintf("DOC_IH_C_%d", ts), Username: fmt.Sprintf("DOC_IH_C_%d", ts), RealName: "开单医生", Role: "doctor", HospitalID: hosp.ID, DepartmentID: dept.ID, Status: "NORMAL"}
	docColleagueRestricted := model.User{UserNo: fmt.Sprintf("DOC_IH_R_%d", ts), Username: fmt.Sprintf("DOC_IH_R_%d", ts), RealName: "受限同科同事", Role: "doctor", HospitalID: hosp.ID, DepartmentID: dept.ID, Title: "主任医师", Status: "RESTRICTED"}
	repository.DB.Create(&pat)
	repository.DB.Create(&docCreator)
	repository.DB.Create(&docColleagueRestricted)
	defer func() {
		repository.DB.Delete(&pat)
		repository.DB.Delete(&docCreator)
		repository.DB.Delete(&docColleagueRestricted)
	}()

	rec := model.MedicalRecord{
		RecordNo:       fmt.Sprintf("REC_IH_%d", ts),
		PatientID:      pat.ID,
		DoctorID:       docCreator.ID,
		HospitalID:     hosp.ID,
		DepartmentName: "心内科",
		Diagnosis:      "房颤",
		CreatedAt:      time.Now(),
	}
	repository.DB.Create(&rec)
	defer func() {
		repository.DB.Where("target_id = ?", rec.RecordNo).Delete(&model.AuditLog{})
		repository.DB.Where("analysis_result LIKE ?", fmt.Sprintf("[REC#%d]%%", rec.ID)).Delete(&model.RiskRecord{})
		repository.DB.Delete(&rec)
	}()

	// 同院同科室受限医生在凌晨(02:30, RULE-T1:+20 + RULE-S1:+30 = 50 -> MEDIUM) 调阅他人病历不得再被硬编码 5 分直接放行
	dec, err := engine.EvaluateAccess(docColleagueRestricted.ID, pat.ID, rec.ID, false)
	if err != nil {
		t.Fatalf("EvaluateAccess failed: %v", err)
	}
	if dec.Allowed || dec.RiskScore < 50 {
		t.Fatalf("同院受限医生凌晨调阅不应绕过 RiskEngine: allowed=%v decision=%s score=%d", dec.Allowed, dec.Decision, dec.RiskScore)
	}

	// 外院医生凌晨走 Break-Glass：对外决策保持 EMERGENCY/0，risk_records 需沉淀真实基线分与夜间因子
	hospOther := model.Hospital{HospitalNo: fmt.Sprintf("HOSP_BG_%d", ts), Name: "外院测试", Level: "三甲", Status: 1}
	repository.DB.Create(&hospOther)
	docExt := model.User{UserNo: fmt.Sprintf("DOC_BG_%d", ts), Username: fmt.Sprintf("DOC_BG_%d", ts), RealName: "外院急救", Role: "doctor", HospitalID: hospOther.ID, Status: "NORMAL"}
	repository.DB.Create(&docExt)
	defer func() {
		repository.DB.Delete(&docExt)
		repository.DB.Delete(&hospOther)
	}()

	bgDec, err := engine.EvaluateAccess(docExt.ID, pat.ID, rec.ID, true)
	if err != nil {
		t.Fatalf("Break-Glass EvaluateAccess failed: %v", err)
	}
	if !bgDec.Allowed || bgDec.RiskLevel != "EMERGENCY" || bgDec.RiskScore != 0 {
		t.Fatalf("Break-Glass 对外决策不符合规范: %+v", bgDec)
	}

	var latestRisk model.RiskRecord
	if err := repository.DB.Where("analysis_result LIKE ?", fmt.Sprintf("[REC#%d] Break-Glass%%", rec.ID)).Order("id DESC").First(&latestRisk).Error; err != nil {
		t.Fatalf("未查询到 Break-Glass 写入的 risk_records 遥测记录: %v", err)
	}
	if latestRisk.AccessRequestID != 0 {
		t.Fatalf("RiskRecord.AccessRequestID 不应错误写入 MedicalRecord.ID=%d", latestRisk.AccessRequestID)
	}
	if latestRisk.TotalScore == 0 || latestRisk.TimeRisk == 0 {
		t.Fatalf("Break-Glass risk_records 应沉淀真实基线风险分与夜间时间因子, got=%+v", latestRisk)
	}

	// 3. 验证 AuditRepository 按 record_no JOIN 与 ListRecentActiveEmgs 过滤已裁决违规/患者异议事件
	tsRepo := time.Now().UnixNano()
	patRepo := model.User{UserNo: fmt.Sprintf("PAT_R_%d", tsRepo), Username: fmt.Sprintf("PAT_R_%d", tsRepo), RealName: "审计联表测试患者", Role: "patient", Status: "NORMAL"}
	docRepo := model.User{UserNo: fmt.Sprintf("DOC_R_%d", tsRepo), Username: fmt.Sprintf("DOC_R_%d", tsRepo), RealName: "审计联表测试医生", Role: "doctor", HospitalID: hosp.ID, Status: "NORMAL"}
	repository.DB.Create(&patRepo)
	repository.DB.Create(&docRepo)
	defer func() {
		repository.DB.Delete(&patRepo)
		repository.DB.Delete(&docRepo)
	}()

	recRepo := model.MedicalRecord{
		RecordNo:   fmt.Sprintf("REC_AUD_%d", tsRepo),
		PatientID:  patRepo.ID,
		DoctorID:   docCreator.ID,
		HospitalID: hosp.ID,
		Diagnosis:  "审计JOIN测试",
		CreatedAt:  time.Now(),
	}
	repository.DB.Create(&recRepo)
	auditRow := model.AuditLog{
		LogID:         fmt.Sprintf("LOG_AUD_%d", tsRepo),
		UserID:        docRepo.ID,
		OperationType: "ACCESS_EVALUATE",
		TargetType:    "RECORD",
		TargetID:      recRepo.RecordNo,
		HospitalID:    hosp.ID,
		Result:        "ALLOWED",
		RiskLevel:     "LOW",
		CreatedAt:     time.Now(),
	}
	repository.DB.Create(&auditRow)
	defer func() {
		repository.DB.Delete(&auditRow)
		repository.DB.Delete(&recRepo)
	}()

	if cnt := repository.DefaultAuditRepository.RecentAuditAccessCount(docRepo.ID, patRepo.ID, 10*time.Minute); cnt != 1 {
		t.Fatalf("RecentAuditAccessCount 应经 record_no 联表统计 ACCESS_EVALUATE/ALLOWED=1，实际=%d", cnt)
	}
	if dCnt := repository.DefaultAuditRepository.DistinctPatientsAccessedByDoctor(docRepo.ID, 10*time.Minute); dCnt != 1 {
		t.Fatalf("DistinctPatientsAccessedByDoctor 应统计=1，实际=%d", dCnt)
	}

	emgViolation := model.EmergencyAccessEvent{
		EventNo:         fmt.Sprintf("EMG_VIO_%d", tsRepo),
		DoctorID:        docRepo.ID,
		PatientID:       patRepo.ID,
		RecordID:        recRepo.ID,
		AuditStatus:     "CLOSED_VIOLATION",
		PatientFeedback: "OBJECTED",
		CreatedAt:       time.Now(),
	}
	emgValid := model.EmergencyAccessEvent{
		EventNo:         fmt.Sprintf("EMG_OK_%d", tsRepo),
		DoctorID:        docRepo.ID,
		PatientID:       patRepo.ID,
		RecordID:        recRepo.ID,
		AuditStatus:     "PENDING_AUDIT",
		PatientFeedback: "PENDING",
		CreatedAt:       time.Now(),
	}
	repository.DB.Create(&emgViolation)
	repository.DB.Create(&emgValid)
	defer func() {
		repository.DB.Delete(&emgViolation)
		repository.DB.Delete(&emgValid)
	}()

	activeEmgs, emgListErr := repository.DefaultEmergencyRepository.ListRecentActiveEmgs(docRepo.ID, time.Now().Add(-1*time.Hour))
	if emgListErr != nil {
		t.Fatalf("ListRecentActiveEmgs failed: %v", emgListErr)
	}
	if len(activeEmgs) != 1 || activeEmgs[0].EventNo != emgValid.EventNo {
		t.Fatalf("ListRecentActiveEmgs 应过滤 CLOSED_VIOLATION 与 OBJECTED，实际=%+v", activeEmgs)
	}
}
