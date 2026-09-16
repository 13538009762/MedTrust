package service_test

import (
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
