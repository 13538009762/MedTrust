package service_test

import (
	"path/filepath"
	"strings"
	"testing"

	"medtrust-backend/config"
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
	"medtrust-backend/service"
)

func TestCompleteExamOrder_TransitionsToExamCompleted(t *testing.T) {
	cfg, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	db, err := repository.InitDB()
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	repository.SeedInitialData(db)

	tempDir := t.TempDir()
	blockchain.InitBlockchainService(filepath.Join(tempDir, "ledger"), blockchain.FabricGatewayConfig{Mode: "mock"})
	service.InitMedicalService("mock", cfg.IPFS.APIURL, filepath.Join(tempDir, "ipfs"))

	docLi, err := repository.DefaultUserRepository.FindByUsername("doc_li")
	if err != nil || docLi == nil {
		t.Fatalf("missing doc_li: %v", err)
	}
	patZhang, err := repository.DefaultUserRepository.FindByUsername("pat_zhang")
	if err != nil || patZhang == nil {
		t.Fatalf("missing pat_zhang: %v", err)
	}

	// 1. 创建包含 2 项检查的初诊记录
	rec, orders, err := service.DefaultMedicalService.CreateEncounterInitial(service.CreateEncounterInitialParams{
		DoctorID:         docLi.ID,
		PatientID:        patZhang.ID,
		EncounterType:    "OUTPATIENT",
		DepartmentName:   "心内科",
		ChiefComplaint:   "胸闷气促 2 天",
		InitialDiagnosis: "待排冠心病",
		NeedExam:         true,
		ExamItems:        []string{"心电图", "心肌酶谱"},
		ExamReason:       "排查急性心肌缺血",
	})
	if err != nil {
		t.Fatalf("CreateEncounterInitial failed: %v", err)
	}
	t.Cleanup(func() {
		repository.DB.Where("record_id = ?", rec.ID).Delete(&model.MedicalExamOrder{})
		repository.DB.Where("record_id = ?", rec.ID).Delete(&model.MedicalFile{})
		repository.DB.Where("id = ?", rec.ID).Delete(&model.MedicalRecord{})
	})

	if rec.Status != "WAITING_EXAM" || len(orders) != 2 {
		t.Fatalf("expected WAITING_EXAM with 2 orders, got status=%s len=%d", rec.Status, len(orders))
	}

	// 2. 完成第 1 项检查 -> 应流转为 PROCESSING_EXAM，并包含第 1 项结果
	_, err = service.DefaultMedicalService.CompleteExamOrder(service.CompleteExamOrderParams{
		OrderID:        orders[0].ID,
		TechnicianID:   docLi.ID,
		TechnicianName: docLi.RealName,
		ExamResult:     "窦性心律，ST段轻度压低",
		ExamConclusion: "心肌缺血待排",
	})
	if err != nil {
		t.Fatalf("CompleteExamOrder #1 failed: %v", err)
	}
	recAfter1, err := repository.DefaultMedicalRepository.FindRecordByID(rec.ID)
	if err != nil {
		t.Fatalf("FindRecordByID after #1 failed: %v", err)
	}
	if recAfter1.Status != "PROCESSING_EXAM" {
		t.Fatalf("完成第 1 项检查后预期状态为 PROCESSING_EXAM，实际: %s", recAfter1.Status)
	}
	if !strings.Contains(recAfter1.ExamResult, "窦性心律，ST段轻度压低") {
		t.Fatalf("完成第 1 项检查后 ExamResult 未回写当前检查结果: %q", recAfter1.ExamResult)
	}

	// 3. 完成第 2 项检查（含加密附件）-> 必须流转为 EXAM_COMPLETED，且同时包含两项结果
	_, err = service.DefaultMedicalService.CompleteExamOrder(service.CompleteExamOrderParams{
		OrderID:        orders[1].ID,
		TechnicianID:   docLi.ID,
		TechnicianName: docLi.RealName,
		ExamResult:     "肌钙蛋白 I 0.02 ng/mL",
		ExamConclusion: "未见明显心肌坏死标志物升高",
		FileName:       "lab_report.txt",
		FileType:       "txt",
		FileData:       []byte("Troponin I: 0.02 ng/mL"),
	})
	if err != nil {
		t.Fatalf("CompleteExamOrder #2 failed: %v", err)
	}
	recAfter2, err := repository.DefaultMedicalRepository.FindRecordByID(rec.ID)
	if err != nil {
		t.Fatalf("FindRecordByID after #2 failed: %v", err)
	}
	if recAfter2.Status != "EXAM_COMPLETED" {
		t.Fatalf("全部检查完成后预期状态流转为 EXAM_COMPLETED，实际仍卡在: %s", recAfter2.Status)
	}
	if !strings.Contains(recAfter2.ExamResult, "窦性心律，ST段轻度压低") || !strings.Contains(recAfter2.ExamResult, "肌钙蛋白 I 0.02 ng/mL") {
		t.Fatalf("全部检查完成后 ExamResult 缺失结果汇总: %q", recAfter2.ExamResult)
	}
}
