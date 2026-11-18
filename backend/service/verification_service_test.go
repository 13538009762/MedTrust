package service_test

import (
	"fmt"
	"path/filepath"
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

	tempLedger := t.TempDir()
	tempIPFS := t.TempDir()
	blockchain.InitBlockchainService(tempLedger, blockchain.FabricGatewayConfig{Mode: "local"})
	service.InitMedicalService(cfg.IPFS.APIURL, tempIPFS)

	// 创建可信病历并执行真实 AES-256-GCM 加密与 IPFS 密文存储
	recordNo := fmt.Sprintf("REC_VERIFY_%d", time.Now().UnixNano())
	fileData := []byte("可信病历存证数据")
	masterKey := cfg.GetAESMasterKey()
	cipherData, encDEK, ivNonce, fileHash, err := crypto.EncryptMedicalFile(fileData, 1001, recordNo, masterKey)
	if err != nil {
		t.Fatalf("EncryptMedicalFile failed: %v", err)
	}
	cid, err := service.DefaultMedicalService.GetIPFSService().Add(cipherData)
	if err != nil {
		t.Fatalf("IPFS Add failed: %v", err)
	}

	rec := model.MedicalRecord{
		RecordNo:      recordNo,
		PatientID:     1001,
		DoctorID:      2001,
		HospitalID:    1,
		DataType:      "门诊",
		Symptoms:      "咳嗽发热",
		Diagnosis:     "急性支气管炎",
		TreatmentPlan: "对症止咳化痰",
		SyncStatus:    service.SyncStatusCompleted,
		CreatedAt:     time.Now(),
	}
	repository.DB.Create(&rec)
	defer repository.DB.Delete(&rec)

	f := model.MedicalFile{
		RecordID:     rec.ID,
		FileName:     "verify_cert.pdf",
		FileType:     "pdf",
		FileSize:     uint64(len(fileData)),
		IPFSCID:      cid,
		EncryptedDEK: encDEK,
		IVNonce:      ivNonce,
		FileHash:     fileHash,
		CreatedAt:    time.Now(),
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

	// 4. 测试 IPFS 密文或 DEK 被篡改：第一阶段 AES-256-GCM 验签必须拦截报警
	repository.DB.Model(&f).Update("encrypted_dek", "dGFtcGVyZWRfZGVrX2Jhc2U2NF9pbnZhbGlkX3RhZw==")
	ipfsTamperRes, err := service.DefaultVerificationService.Verify(rec.ID)
	if err != nil {
		t.Fatalf("Verify after IPFS/DEK tamper failed: %v", err)
	}
	if ipfsTamperRes.Verified || (len(ipfsTamperRes.Stages) > 0 && ipfsTamperRes.Stages[0].Passed) {
		t.Fatalf("IPFS 密文/DEK 被篡改后预期 Verified=false 且 Stage1.Passed=false，但返回 Verified=%v", ipfsTamperRes.Verified)
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

	tempLedger := t.TempDir()
	tempIPFS := t.TempDir()
	blockchain.InitBlockchainService(tempLedger, blockchain.FabricGatewayConfig{Mode: "local"})
	service.InitMedicalService(cfg.IPFS.APIURL, tempIPFS)

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

func TestVerification_LegacyDefaultSecretDecryptCompatibility(t *testing.T) {
	cfg, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("Load config failed: %v", err)
	}
	tempIPFS := t.TempDir()
	service.InitMedicalService(cfg.IPFS.APIURL, tempIPFS)

	plain := []byte("%PDF-1.4 legacy medical record attachment")
	patientID := uint64(4)
	recordNo := fmt.Sprintf("ENC_LEGACY_%d", time.Now().UnixNano())

	// 1. 模拟升级 WP-09 前（未配置独立环境变量时使用 DefaultMasterSecret）生成的旧派生密文
	t.Setenv("MEDTRUST_CRYPTO_MASTER_KEY", "")
	t.Setenv("AES_MASTER_KEY", "")
	legacyPack, legacyHash, err := crypto.EncryptRecordFile(plain, patientID, recordNo)
	if err != nil {
		t.Fatalf("EncryptRecordFile failed: %v", err)
	}
	cid, err := service.DefaultMedicalService.GetIPFSService().Add(legacyPack)
	if err != nil {
		t.Fatalf("IPFS Add failed: %v", err)
	}

	rec := &model.MedicalRecord{RecordNo: recordNo, PatientID: patientID}
	file := &model.MedicalFile{FileName: "临床就诊规范归档凭据_" + recordNo + ".pdf", IPFSCID: cid, FileHash: legacyHash}

	// 2. 启用独立 AES_MASTER_KEY 后，非生产模式应能向下兼容解密历史旧密文并精准校验哈希
	t.Setenv("AES_MASTER_KEY", "k7R9vX2mL4pQ8sY1wZ3bN5cE7tH9uI0j")
	origEnv := config.AppConfig.AppEnv
	config.AppConfig.AppEnv = "demo"
	defer func() { config.AppConfig.AppEnv = origEnv }()

	decBytes, decHash, decErr := service.DefaultMedicalService.DecryptFileAndComputeHash(rec, file)
	if decErr != nil {
		t.Fatalf("expected legacy compatibility decryption to succeed in demo mode, got: %v", decErr)
	}
	if string(decBytes) != string(plain) || decHash != legacyHash {
		t.Fatalf("decrypted content/hash mismatch: got hash=%s want=%s", decHash, legacyHash)
	}

	// 3. 生产模式下禁止回退内置默认密钥
	config.AppConfig.AppEnv = "production"
	if _, _, prodErr := service.DefaultMedicalService.DecryptFileAndComputeHash(rec, file); prodErr == nil {
		t.Fatalf("production mode must NOT fall back to DefaultMasterSecret")
	}
}

func TestVerification_MultiFileChainHashCompatibility(t *testing.T) {
	cfg, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("Load config failed: %v", err)
	}
	if _, err := repository.InitDB(); err != nil {
		t.Fatalf("Init DB failed: %v", err)
	}

	tempLedger := t.TempDir()
	tempIPFS := t.TempDir()
	blockchain.InitBlockchainService(tempLedger, blockchain.FabricGatewayConfig{Mode: "local"})
	service.InitMedicalService(cfg.IPFS.APIURL, tempIPFS)

	recordNo := fmt.Sprintf("REC_MULTI_%d", time.Now().UnixNano())
	rec := model.MedicalRecord{
		RecordNo:      recordNo,
		PatientID:     1009,
		DoctorID:      2009,
		HospitalID:    1,
		DataType:      "OUTPATIENT",
		EncounterType: "OUTPATIENT",
		Status:        "COMPLETED",
		Diagnosis:     "多附件归档核验回归测试",
		TreatmentPlan: "常规随访复查",
		CreatedAt:     time.Now(),
	}
	if err := repository.DB.Create(&rec).Error; err != nil {
		t.Fatalf("Create record failed: %v", err)
	}
	defer repository.DB.Unscoped().Delete(&model.MedicalRecord{}, rec.ID)

	imgPlain := []byte("ORIGINAL_LAB_IMAGE_BYTES_01")
	imgHash := crypto.ComputeSHA256(imgPlain)
	archivePlain := []byte("%PDF-1.4 ARCHIVE_RECEIPT_BYTES_02")
	masterKey := config.AppConfig.GetAESMasterKey()
	cipherData, encDEK, ivNonce, archiveHash, err := crypto.EncryptMedicalFile(archivePlain, rec.PatientID, rec.RecordNo, masterKey)
	if err != nil {
		t.Fatalf("EncryptMedicalFile failed: %v", err)
	}
	archiveCID, err := service.DefaultMedicalService.GetIPFSService().Add(cipherData)
	if err != nil {
		t.Fatalf("IPFS Add failed: %v", err)
	}

	fImg := model.MedicalFile{RecordID: rec.ID, FileName: "检验影像_01.png", FileHash: imgHash, IPFSCID: "QmMockImgOnlyHashMatch"}
	fArc := model.MedicalFile{RecordID: rec.ID, FileName: "临床就诊规范归档凭据_" + recordNo + ".pdf", FileHash: archiveHash, IPFSCID: archiveCID, EncryptedDEK: encDEK, IVNonce: ivNonce}
	repository.DB.Create(&fImg)
	repository.DB.Create(&fArc)
	defer func() {
		repository.DB.Unscoped().Where("record_id = ?", rec.ID).Delete(&model.MedicalFile{})
	}()

	// 链上 file_hash 锚定首个上传的检验影像哈希，主归档凭据位于末尾且完整加密
	clinicalHash := service.ComputeRecordHash(&rec)
	txID, height, commitErr := blockchain.DefaultService.CommitAsset("MEDICAL_RECORD", rec.RecordNo, map[string]interface{}{
		"record_no":     rec.RecordNo,
		"patient_id":    rec.PatientID,
		"doctor_id":     rec.DoctorID,
		"hospital_id":   rec.HospitalID,
		"cid":           archiveCID,
		"file_hash":     imgHash,
		"clinical_hash": clinicalHash,
		"data_type":     rec.DataType,
		"create_time":   rec.CreatedAt.Format(time.RFC3339),
	})
	if commitErr != nil {
		t.Fatalf("CommitAsset failed: %v", commitErr)
	}
	repository.DB.Model(&rec).Updates(map[string]interface{}{
		"fabric_tx_id": txID,
		"block_height": height,
	})

	// 1. 多附件记录在主归档凭据与链上首个附件均真实一致时，Verify 与 VerifyRecord 均应通过
	res, err := service.DefaultVerificationService.Verify(rec.ID)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !res.Verified {
		t.Fatalf("expected multi-file record Verify=true, got false: %s", res.Message)
	}

	repository.DB.Preload("Files").First(&rec, rec.ID)
	service.DefaultMedicalService.VerifyRecord(&rec)
	if !rec.Verified || rec.IsTampered {
		t.Fatalf("expected multi-file VerifyRecord Verified=true IsTampered=false, got reason=%s", rec.TamperReason)
	}

	// 2. 若将主归档凭据对应 IPFS 内容替换为另一个可解密密文（与 primaryFile.FileHash 不符），仍必须判定篡改
	badPlain := []byte("TAMPERED_PDF_PAYLOAD")
	badCipher, badDEK, badIV, _, _ := crypto.EncryptMedicalFile(badPlain, rec.PatientID, rec.RecordNo, masterKey)
	badCID, _ := service.DefaultMedicalService.GetIPFSService().Add(badCipher)
	repository.DB.Model(&fArc).Updates(map[string]interface{}{"ipfs_cid": badCID, "encrypted_dek": badDEK, "iv_nonce": badIV})

	tamperedRes, err := service.DefaultVerificationService.Verify(rec.ID)
	if err != nil {
		t.Fatalf("Verify after tamper failed: %v", err)
	}
	if tamperedRes.Verified || (len(tamperedRes.Stages) > 0 && tamperedRes.Stages[0].Passed) {
		t.Fatalf("expected tampered primary attachment to fail verification even with sibling file on chain")
	}
}

// TestVerification_ExpandedFieldsAndNoBootOverwrite 验证：
// 1. 现病史(PresentIllness)、生命体征(VitalSigns)、检查结果(ExamResult) 等扩充临床字段遭篡改时可被 ComputeRecordHash / VerifyRecord 捕获；
// 2. EnsureBaselineLedgerAnchored 绝不会在服务重启时覆写已上链非种子病历的被篡改哈希。
func TestVerification_ExpandedFieldsAndNoBootOverwrite(t *testing.T) {
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

	ts := time.Now().UnixNano()
	recNo := fmt.Sprintf("REC_EXP_%d", ts)
	masterKey := config.AppConfig.GetAESMasterKey()
	cipherData, encDEK, ivNonce, fileHash, err := crypto.EncryptMedicalFile([]byte("EXPANDED_FIELDS_ARCHIVE"), 1, recNo, masterKey)
	if err != nil {
		t.Fatalf("EncryptMedicalFile failed: %v", err)
	}
	cid, err := service.DefaultMedicalService.GetIPFSService().Add(cipherData)
	if err != nil {
		t.Fatalf("IPFS Add failed: %v", err)
	}

	rec := model.MedicalRecord{
		RecordNo:         recNo,
		PatientID:        1,
		DoctorID:         1,
		HospitalID:       1,
		DataType:         "EMR",
		EncounterType:    "OUTPATIENT",
		DepartmentName:   "心内科",
		ChiefComplaint:   "胸闷气促1天",
		PresentIllness:   "昨日起无明显诱因胸闷",
		VitalSigns:       "BP 120/80mmHg, HR 72bpm",
		InitialDiagnosis: "待查胸闷",
		DiagnosticBasis:  "结合心电图与肌钙蛋白排査",
		ExamItems:        "心电图",
		ExamResult:       "窦性心律，未见ST段抬高",
		Diagnosis:        "稳定型心绞痛待排",
		Symptoms:         "胸闷",
		TreatmentPlan:    "休息并复查",
		CreatedAt:        time.Now(),
	}
	repository.DB.Create(&rec)
	fRec := model.MedicalFile{RecordID: rec.ID, FileName: "归档_" + recNo + ".pdf", FileHash: fileHash, IPFSCID: cid, EncryptedDEK: encDEK, IVNonce: ivNonce}
	repository.DB.Create(&fRec)
	defer func() {
		repository.DB.Unscoped().Where("record_id = ?", rec.ID).Delete(&model.MedicalFile{})
		repository.DB.Unscoped().Delete(&model.MedicalRecord{}, rec.ID)
	}()

	repository.DB.Preload("Files").First(&rec, rec.ID)
	origClinicalHash := service.ComputeRecordHash(&rec)
	txID, height, err := blockchain.DefaultService.CommitAsset("MEDICAL_RECORD", recNo, map[string]interface{}{
		"record_no":     recNo,
		"patient_id":    rec.PatientID,
		"doctor_id":     rec.DoctorID,
		"hospital_id":   rec.HospitalID,
		"cid":           cid,
		"file_hash":     fileHash,
		"clinical_hash": origClinicalHash,
		"data_type":     rec.DataType,
		"create_time":   rec.CreatedAt.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("CommitAsset failed: %v", err)
	}
	repository.DB.Model(&rec).Updates(map[string]interface{}{"fabric_tx_id": txID, "block_height": height})

	// 恶意篡改 VitalSigns 与 ExamResult (历史 9 字段哈希无法发现的字段)
	repository.DB.Model(&rec).Updates(map[string]interface{}{
		"vital_signs": "BP 210/130mmHg (TAMPERED)",
		"exam_result": "急性前壁ST段弓背抬高 (TAMPERED)",
	})

	// 重启锚定检查绝不可将篡改后哈希覆盖回链上账本
	service.EnsureBaselineLedgerAnchored()
	chainAfterBoot, exists := blockchain.DefaultService.QueryAsset(recNo)
	if !exists || chainAfterBoot == nil || fmt.Sprintf("%v", chainAfterBoot["clinical_hash"]) != origClinicalHash {
		t.Fatalf("EnsureBaselineLedgerAnchored 不应覆盖已有链上 clinical_hash! got=%v want=%s", chainAfterBoot, origClinicalHash)
	}

	// 验真必须捕获到 VitalSigns / ExamResult 篡改
	res, err := service.DefaultVerificationService.Verify(rec.ID)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if res.Verified {
		t.Fatalf("篡改 VitalSigns / ExamResult 后 Verify 必须失败")
	}
}
