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

func TestConsistency_StateMachineAndReconciliation(t *testing.T) {
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
	service.InitAccessEngine(cfg.Risk.LowThreshold, cfg.Risk.HighThreshold)

	// 创建测试医生与患者
	doc := model.User{
		UserNo:       "DOC_CONSISTENCY_01",
		Username:     "DOC_CONSISTENCY_01",
		RealName:     "一致性测试医生",
		Role:         "doctor",
		HospitalID:   1,
		Status:       "NORMAL",
	}
	pat := model.User{
		UserNo:   "PAT_CONSISTENCY_01",
		Username: "PAT_CONSISTENCY_01",
		RealName: "一致性测试患者",
		Role:     "patient",
		Status:   "NORMAL",
	}
	repository.DB.Create(&doc)
	repository.DB.Create(&pat)
	defer repository.DB.Delete(&doc)
	defer repository.DB.Delete(&pat)

	consSvc := service.DefaultConsistencyService

	// 1. 正常创建流转状态机测试 (PENDING -> IPFS_SUCCESS -> FABRIC_SUCCESS -> DB_SUCCESS -> COMPLETED)
	idempotencyKey := fmt.Sprintf("IDEMP_%d", time.Now().UnixNano())
	params := service.UploadRecordParams{
		DoctorID:      doc.ID,
		PatientID:     pat.ID,
		DataType:      "门诊",
		Symptoms:      "持续发热伴头痛",
		Diagnosis:     "上呼吸道感染",
		TreatmentPlan: "口服阿莫西林胶囊，多饮水休息",
		FileData:      []byte("一致性状态机测试附件内容"),
	}

	rec, err := consSvc.CreateRecordWithConsistency(params, idempotencyKey)
	if err != nil {
		t.Fatalf("CreateRecordWithConsistency failed: %v", err)
	}
	defer repository.DB.Delete(rec)

	if rec.SyncStatus != service.SyncStatusCompleted {
		t.Fatalf("预期状态机终态为 COMPLETED，实际为: %s", rec.SyncStatus)
	}
	if rec.FabricTxID == "" {
		t.Fatalf("Fabric 交易 ID 缺失")
	}
	if len(rec.Files) == 0 || rec.Files[0].IPFSCID == "" {
		t.Fatalf("IPFS CID 缺失")
	}

	// 2. 幂等性测试 (重放相同 IdempotencyKey 应返回已存在记录，不新增)
	recDup, err := consSvc.CreateRecordWithConsistency(params, idempotencyKey)
	if err != nil {
		t.Fatalf("Idempotent replay failed: %v", err)
	}
	if recDup.ID != rec.ID {
		t.Fatalf("幂等拦截失效，产生了新记录 ID=%d (原 ID=%d)", recDup.ID, rec.ID)
	}

	// 3. 跨系统补偿对账测试 (模拟数据库存在但链上丢失时的自动补偿)
	recLoss := model.MedicalRecord{
		RecordNo:       fmt.Sprintf("MR-LOSS-%d", time.Now().UnixNano()),
		PatientID:      pat.ID,
		DoctorID:       doc.ID,
		HospitalID:     doc.HospitalID,
		DataType:       "门诊",
		Symptoms:       "急性胃肠炎",
		Diagnosis:      "急性胃肠炎",
		SyncStatus:     service.SyncStatusNeedsReconciliation,
		CreatedAt:      time.Now(),
	}
	repository.DB.Create(&recLoss)
	defer repository.DB.Delete(&recLoss)

	fileLoss := model.MedicalFile{
		RecordID:  recLoss.ID,
		FileName:  "loss.dat",
		FileType:  "dat",
		FileSize:  10,
		IPFSCID:   "QmFallbackLossCID",
		FileHash:  crypto.CalculateSHA256([]byte("loss-data")),
		CreatedAt: time.Now(),
	}
	repository.DB.Create(&fileLoss)
	defer repository.DB.Delete(&fileLoss)

	// 执行单条对账补偿
	reconciledRec, err := consSvc.ReconcileRecord(recLoss.ID)
	if err != nil {
		t.Fatalf("ReconcileRecord 补偿失败: %v", err)
	}
	if reconciledRec.SyncStatus != service.SyncStatusCompleted {
		t.Fatalf("补偿后状态应纠正为 COMPLETED，实际为: %s", reconciledRec.SyncStatus)
	}
	if reconciledRec.FabricTxID == "" {
		t.Fatalf("补偿后 FabricTxID 未被固化")
	}

	// 4. 单方篡改一致性检测 (本地临床数据被篡改，对账应检出冲突并置为 NEEDS_RECONCILIATION)
	repository.DB.Model(&recLoss).Update("diagnosis", "被恶意篡改的假诊断")
	_, err = consSvc.ReconcileRecord(recLoss.ID)
	if err == nil {
		t.Fatalf("数据被篡改后对账应检出冲突报错，但未报错")
	}

	var recTampered model.MedicalRecord
	repository.DB.First(&recTampered, recLoss.ID)
	if recTampered.SyncStatus != service.SyncStatusNeedsReconciliation {
		t.Fatalf("篡改后状态应为 NEEDS_RECONCILIATION，实际为: %s", recTampered.SyncStatus)
	}
}
