package service_test

import (
	"strings"
	"testing"
	"time"

	"medtrust-backend/config"
	"medtrust-backend/model"
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

	// 准备测试数据: 医院、科室、医生、患者
	hospA := model.Hospital{Name: "测试医院A", Level: "三甲"}
	hospB := model.Hospital{Name: "测试医院B", Level: "三甲"}
	repository.DB.Create(&hospA)
	repository.DB.Create(&hospB)
	defer repository.DB.Delete(&hospA)
	defer repository.DB.Delete(&hospB)

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
		UserNo:       "DOC_RESTRICTED_TEST",
		Username:     "DOC_RESTRICTED_TEST",
		RealName:     "违规受限医生",
		Role:         "doctor",
		HospitalID:   hospB.ID,
		Title:        "住院医师",
		Status:       "RESTRICTED",
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
		StartTime:      time.Now().Add(-1 * time.Hour),
		EndTime:        time.Now().Add(24 * time.Hour),
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

