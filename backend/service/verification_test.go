package service_test

import (
	"fmt"
	"testing"
	"time"

	"medtrust-backend/config"
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/pkg/crypto"
	"medtrust-backend/repository"
	"medtrust-backend/service"
)

func TestVerification_AuthenticRecordPasses(t *testing.T) {
	cfg, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("Load config failed: %v", err)
	}
	_, err = repository.InitDB()
	if err != nil {
		t.Fatalf("Init DB failed: %v", err)
	}

	blockchain.InitBlockchainService("../ledger_data", blockchain.FabricGatewayConfig{Mode: "local"})
	service.InitMedicalService(cfg.IPFS.APIURL, "../ipfs_storage")

	// 创建可信病历并上链
	recordNo := fmt.Sprintf("REC_VERIFY_%d", time.Now().UnixNano())
	fileData := []byte("可信病历存证数据")
	fileHash := crypto.CalculateSHA256(fileData)

	rec := model.MedicalRecord{
		RecordNo:       recordNo,
		PatientID:      1001,
		DoctorID:       2001,
		HospitalID:     1,
		DataType:       "门诊",
		Symptoms:       "咳嗽发热",
		Diagnosis:      "急性支气管炎",
		TreatmentPlan:  "对症止咳化痰",
		SyncStatus:     service.SyncStatusCompleted,
		CreatedAt:      time.Now(),
	}
	repository.DB.Create(&rec)
	defer repository.DB.Delete(&rec)

	f := model.MedicalFile{
		RecordID:  rec.ID,
		FileName:  "verify_cert.pdf",
		FileType:  "pdf",
		FileSize:  uint64(len(fileData)),
		IPFSCID:   "QmVerifyTestCID123456",
		FileHash:  fileHash,
		CreatedAt: time.Now(),
	}
	repository.DB.Create(&f)
	defer repository.DB.Delete(&f)
	rec.Files = []model.MedicalFile{f}

	clinicalHash := service.ComputeRecordHash(&rec)

	// 提交区块链上链存证
	txID, height, err := blockchain.DefaultService.CommitAsset("MEDICAL_RECORD", recordNo, map[string]interface{}{
		"record_no":     recordNo,
		"patient_id":    rec.PatientID,
		"doctor_id":     rec.DoctorID,
		"hospital_id":   rec.HospitalID,
		"cid":           f.IPFSCID,
		"file_hash":     fileHash,
		"clinical_hash": clinicalHash,
		"data_type":     rec.DataType,
		"create_time":   rec.CreatedAt.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("CommitAsset failed: %v", err)
	}

	repository.DB.Model(&rec).Updates(map[string]interface{}{
		"fabric_tx_id": txID,
		"block_height": height,
	})

	// 1. 测试初始真实病历：校验必须通过
	res, err := service.DefaultVerificationService.Verify(rec.ID)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !res.Verified {
		t.Fatalf("真实未经篡改的病历预期 Verified=true，但为 false (Message: %s)", res.Message)
	}

	// 2. 测试黑客模拟篡改演练 (修改诊断与医嘱)：校验必须拦截报警
	tamperRes, err := service.DefaultVerificationService.SimulateTamper(rec.ID)
	if err != nil {
		t.Fatalf("SimulateTamper failed: %v", err)
	}
	if tamperRes.Verified {
		t.Fatalf("数据被篡改后预期 Verified=false 报警，但仍返回 true")
	}

	// 3. 测试一键恢复：恢复后校验必须再次通过
	restoreRes, err := service.DefaultVerificationService.RestoreTamperedRecord(rec.ID)
	if err != nil {
		t.Fatalf("RestoreTamperedRecord failed: %v", err)
	}
	if !restoreRes.Verified {
		t.Fatalf("数据恢复后预期 Verified=true，但仍为 false")
	}
}

func TestVerification_MissingLedgerAsset(t *testing.T) {
	cfg, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("Load config failed: %v", err)
	}
	_, err = repository.InitDB()
	if err != nil {
		t.Fatalf("Init DB failed: %v", err)
	}

	blockchain.InitBlockchainService("../ledger_data", blockchain.FabricGatewayConfig{Mode: "local"})
	service.InitMedicalService(cfg.IPFS.APIURL, "../ipfs_storage")

	// 创建未上链的孤立病历
	orphanRec := model.MedicalRecord{
		RecordNo:   fmt.Sprintf("REC_ORPHAN_%d", time.Now().UnixNano()),
		PatientID:  1002,
		DoctorID:   2002,
		HospitalID: 1,
		Diagnosis:  "未上链病历",
		CreatedAt:  time.Now(),
	}
	repository.DB.Create(&orphanRec)
	defer repository.DB.Delete(&orphanRec)

	res, err := service.DefaultVerificationService.Verify(orphanRec.ID)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	// 链上存证缺失必须被判定为不可信/篡改
	if res.Verified {
		t.Fatalf("链上存证缺失的病历必须判定为 Verified=false，但返回了 true")
	}
}
