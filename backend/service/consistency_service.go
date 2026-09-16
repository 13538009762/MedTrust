package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/pkg/crypto"
	"medtrust-backend/pkg/pdf"
	"medtrust-backend/repository"
)

// 数据一致性状态机流转状态定义 (学术规范与工程鲁棒性对齐)
const (
	SyncStatusPending             = "PENDING"              // 初始接收，准备入库
	SyncStatusIPFSSuccess         = "IPFS_SUCCESS"         // 阶段一：密文文件已安全上传至分布式 IPFS 网络
	SyncStatusFabricSuccess       = "FABRIC_SUCCESS"       // 阶段二：资产综合哈希存证已上链背书
	SyncStatusDBSuccess           = "DB_SUCCESS"           // 阶段三：关系型数据库事务写入成功
	SyncStatusCompleted           = "COMPLETED"            // 状态机流转终态：三方状态严格一致
	SyncStatusFailed              = "FAILED"               // 失败终态：执行阻断
	SyncStatusRetrying            = "RETRYING"             // 瞬态错误，等待后台补偿对账重试
	SyncStatusNeedsReconciliation = "NEEDS_RECONCILIATION" // 数据不一致/链上缺失，需人工或对账任务深度补偿
)

type ConsistencyService struct {
	mu sync.Mutex
}

var DefaultConsistencyService = &ConsistencyService{}

// CreateRecordWithConsistency 基于状态机流水线创建病历，支持幂等去重与跨分布式节点事务补偿
func (s *ConsistencyService) CreateRecordWithConsistency(p UploadRecordParams, idempotencyKey string) (*model.MedicalRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. 幂等性校验 (Idempotency Key Check)
	if idempotencyKey != "" && repository.DB != nil {
		var existing model.MedicalRecord
		if err := repository.DB.Where("idempotency_key = ?", idempotencyKey).First(&existing).Error; err == nil {
			log.Printf("[ConsistencyService] ⚠️ 触发幂等重放拦截: IdempotencyKey=%s, 已存在记录 ID=%d, RecordNo=%s", idempotencyKey, existing.ID, existing.RecordNo)
			var files []model.MedicalFile
			_ = repository.DB.Where("record_id = ?", existing.ID).Find(&files)
			existing.Files = files
			return &existing, nil
		}
	}

	var doc model.User
	if repository.DB != nil {
		if err := repository.DB.First(&doc, p.DoctorID).Error; err != nil {
			return nil, errors.New("医生信息不存在")
		}
	} else {
		doc = model.User{ID: p.DoctorID, HospitalID: 1}
	}

	// 生成业务编号
	randBytes := make([]byte, 4)
	_, _ = rand.Read(randBytes)
	recordNo := fmt.Sprintf("ENC%s%s", time.Now().Format("20060102"), hex.EncodeToString(randBytes))

	if p.Symptoms == "" && p.ChiefComplaint != "" {
		p.Symptoms = p.ChiefComplaint
	}
	if p.ChiefComplaint == "" && p.Symptoms != "" {
		p.ChiefComplaint = p.Symptoms
	}
	if p.DataType == "" {
		p.DataType = "门诊"
	}
	if p.EncounterType == "" {
		p.EncounterType = "普通门诊"
	}
	if p.Status == "" {
		p.Status = "已接诊"
	}

	if len(p.FileData) == 0 {
		pdfBytes, _ := pdf.GenerateAttestationPDF(pdf.AttestationData{
			RecordNo:         recordNo,
			PatientName:      fmt.Sprintf("患者#%d", p.PatientID),
			DoctorName:       doc.RealName,
			HospitalName:     fmt.Sprintf("医疗机构#%d", doc.HospitalID),
			EncounterType:    p.EncounterType,
			DepartmentName:   p.DepartmentName,
			ChiefComplaint:   p.ChiefComplaint,
			PresentIllness:   p.PresentIllness,
			DiagnosticBasis:  p.DiagnosticBasis,
			Diagnosis:        p.Diagnosis,
			Etiology:         p.Etiology,
			TreatmentPlan:    p.TreatmentPlan,
		})
		p.FileData = pdfBytes
		p.FileName = fmt.Sprintf("临床就诊规范归档凭据_%s.pdf", recordNo)
		p.FileType = "pdf"
	}

	// 初始草稿记录（处于 PENDING 状态）
	record := model.MedicalRecord{
		RecordNo:         recordNo,
		PatientID:        p.PatientID,
		DoctorID:         p.DoctorID,
		HospitalID:       doc.HospitalID,
		DataType:         p.DataType,
		OnsetTime:        p.OnsetTime,
		Duration:         p.Duration,
		Symptoms:         p.Symptoms,
		Etiology:         p.Etiology,
		TreatmentPlan:    p.TreatmentPlan,
		VitalSigns:       p.VitalSigns,
		Diagnosis:        p.Diagnosis,
		EncounterType:    p.EncounterType,
		DepartmentName:   p.DepartmentName,
		Status:           p.Status,
		ChiefComplaint:   p.ChiefComplaint,
		PresentIllness:   p.PresentIllness,
		InitialDiagnosis: p.InitialDiagnosis,
		DiagnosticBasis:  p.DiagnosticBasis,
		NeedExam:         p.NeedExam,
		ExamItems:        p.ExamItems,
		ExamReason:       p.ExamReason,
		ExamResult:       p.ExamResult,
		ExamDoctor:       p.ExamDoctor,
		ExamTime:         p.ExamTime,
		IdempotencyKey:   idempotencyKey,
		SyncStatus:       SyncStatusPending,
		RetryCount:       0,
		CreatedAt:        time.Now(),
	}

	// 阶段 1: 密文上链下存储 (IPFS)
	pack, fileHash, err := crypto.EncryptRecordFile(p.FileData, p.PatientID, recordNo)
	if err != nil {
		return nil, fmt.Errorf("AES-256-GCM 加密失败: %w", err)
	}

	ipfsSvc := DefaultMedicalService.GetIPFSService()
	cid, err := ipfsSvc.PutData(pack)
	if err != nil {
		record.SyncStatus = SyncStatusFailed
		record.SyncError = fmt.Sprintf("IPFS 上传失败: %v", err)
		if repository.DB != nil {
			_ = repository.DB.Create(&record)
		}
		return nil, fmt.Errorf("阶段一 IPFS 上传失败: %w", err)
	}
	record.SyncStatus = SyncStatusIPFSSuccess

	fileRecord := model.MedicalFile{
		FileName:  p.FileName,
		FileType:  p.FileType,
		FileSize:  uint64(len(p.FileData)),
		IPFSCID:   cid,
		FileHash:  fileHash,
		CreatedAt: time.Now(),
	}
	record.Files = []model.MedicalFile{fileRecord}

	// 计算包含临床结构化字段与文件指纹的全局哈希
	clinicalHash := ComputeRecordHash(&record)

	// 阶段 2: 资产综合哈希上链存证 (Fabric)
	txID, height, err := blockchain.DefaultService.CommitAsset("MEDICAL_RECORD", recordNo, map[string]interface{}{
		"record_no":      recordNo,
		"patient_id":     p.PatientID,
		"doctor_id":      p.DoctorID,
		"hospital_id":    doc.HospitalID,
		"cid":            cid,
		"file_hash":      fileHash,
		"clinical_hash":  clinicalHash,
		"data_type":      p.DataType,
		"encounter_type": p.EncounterType,
		"create_time":    record.CreatedAt.Format(time.RFC3339),
	})
	if err != nil {
		record.SyncStatus = SyncStatusRetrying
		record.SyncError = fmt.Sprintf("Fabric 上链背书失败: %v", err)
		if repository.DB != nil {
			_ = repository.DB.Create(&record)
			fileRecord.RecordID = record.ID
			_ = repository.DB.Create(&fileRecord)
		}
		return nil, fmt.Errorf("阶段二 Fabric 上链失败(已置为 RETRYING 待补偿): %w", err)
	}

	record.FabricTxID = txID
	record.BlockHeight = height
	record.SyncStatus = SyncStatusFabricSuccess

	// 阶段 3: 本地 MySQL 数据库持久化
	if repository.DB != nil {
		tx := repository.DB.Begin()
		record.SyncStatus = SyncStatusDBSuccess
		if err := tx.Create(&record).Error; err != nil {
			tx.Rollback()
			record.SyncStatus = SyncStatusFailed
			record.SyncError = fmt.Sprintf("DB 写入事务失败: %v", err)
			return nil, fmt.Errorf("阶段三 DB 写入失败: %w", err)
		}

		fileRecord.RecordID = record.ID
		if err := tx.Create(&fileRecord).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("保存病历附件失败: %w", err)
		}
		tx.Commit()
	}

	// 阶段 4: 终态确认 (COMPLETED)
	record.SyncStatus = SyncStatusCompleted
	record.SyncError = ""
	if repository.DB != nil {
		repository.DB.Model(&model.MedicalRecord{}).Where("id = ?", record.ID).Updates(map[string]interface{}{
			"sync_status": SyncStatusCompleted,
			"sync_error":  "",
		})
		uploadIP := p.ClientIP
		if uploadIP == "" {
			uploadIP = "127.0.0.1"
		}
		DefaultAuditService.Log(p.DoctorID, "UPLOAD", "RECORD", recordNo, doc.HospitalID, "SUCCESS", "LOW", uploadIP)
	}

	return &record, nil
}

// ReconcileRecord 对指定病历执行幂等补偿与一致性修复
func (s *ConsistencyService) ReconcileRecord(recordID uint64) (*model.MedicalRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if repository.DB == nil {
		return nil, errors.New("数据库未就绪")
	}

	var rec model.MedicalRecord
	if err := repository.DB.Preload("Files").First(&rec, recordID).Error; err != nil {
		return nil, fmt.Errorf("病历不存在: %w", err)
	}

	fileHash := ""
	cid := ""
	if len(rec.Files) > 0 {
		fileHash = rec.Files[0].FileHash
		cid = rec.Files[0].IPFSCID
	}
	clinicalHash := ComputeRecordHash(&rec)

	chainData, exists := blockchain.DefaultService.QueryAsset(rec.RecordNo)

	// 情况 1: 链上完全缺失，重新提交资产存证
	if !exists || chainData == nil {
		txID, height, err := blockchain.DefaultService.CommitAsset("MEDICAL_RECORD", rec.RecordNo, map[string]interface{}{
			"record_no":      rec.RecordNo,
			"patient_id":     rec.PatientID,
			"doctor_id":      rec.DoctorID,
			"hospital_id":    rec.HospitalID,
			"cid":            cid,
			"file_hash":      fileHash,
			"clinical_hash":  clinicalHash,
			"data_type":      rec.DataType,
			"encounter_type": rec.EncounterType,
			"create_time":    rec.CreatedAt.Format(time.RFC3339),
		})
		if err != nil {
			rec.RetryCount++
			rec.SyncStatus = SyncStatusNeedsReconciliation
			rec.SyncError = fmt.Sprintf("对账补偿上链重试失败: %v", err)
			repository.DB.Model(&rec).Updates(map[string]interface{}{
				"retry_count": rec.RetryCount,
				"sync_status": rec.SyncStatus,
				"sync_error":  rec.SyncError,
			})
			return &rec, fmt.Errorf("补偿上链失败: %w", err)
		}

		rec.FabricTxID = txID
		rec.BlockHeight = height
		rec.SyncStatus = SyncStatusCompleted
		rec.SyncError = ""
		repository.DB.Model(&rec).Updates(map[string]interface{}{
			"fabric_tx_id": txID,
			"block_height": height,
			"sync_status":  SyncStatusCompleted,
			"sync_error":   "",
		})
		log.Printf("[ConsistencyService] ✅ 成功对账并补齐病历 %s 链上存证: TxID=%s, Height=%d", rec.RecordNo, txID, height)
		return &rec, nil
	}

	// 情况 2: 链上存在，核查临床哈希一致性
	currChainHash := fmt.Sprintf("%v", chainData["clinical_hash"])
	if currChainHash == "" {
		currChainHash = fmt.Sprintf("%v", chainData["file_hash"])
	}

	if currChainHash != clinicalHash {
		// 存在不一致，标记为 NEEDS_RECONCILIATION
		rec.SyncStatus = SyncStatusNeedsReconciliation
		rec.SyncError = fmt.Sprintf("链上哈希(%s)与本地临床哈希(%s)不一致，疑似单方篡改", currChainHash, clinicalHash)
		repository.DB.Model(&rec).Updates(map[string]interface{}{
			"sync_status": rec.SyncStatus,
			"sync_error":  rec.SyncError,
		})
		return &rec, fmt.Errorf("数据存在一致性冲突: %s", rec.SyncError)
	}

	// 状态一致，纠正为 COMPLETED
	if rec.SyncStatus != SyncStatusCompleted {
		rec.SyncStatus = SyncStatusCompleted
		rec.SyncError = ""
		repository.DB.Model(&rec).Updates(map[string]interface{}{
			"sync_status": SyncStatusCompleted,
			"sync_error":  "",
		})
	}

	return &rec, nil
}

// ReconcileAllPendingOrFailed 批量对账补偿任务，处理所有非 COMPLETED 状态的异常病历
func (s *ConsistencyService) ReconcileAllPendingOrFailed() (scanned int, reconciled int, err error) {
	if repository.DB == nil {
		return 0, 0, nil
	}

	var records []model.MedicalRecord
	err = repository.DB.Where("sync_status IN (?)", []string{
		SyncStatusPending,
		SyncStatusIPFSSuccess,
		SyncStatusFabricSuccess,
		SyncStatusRetrying,
		SyncStatusFailed,
		SyncStatusNeedsReconciliation,
	}).Find(&records).Error
	if err != nil {
		return 0, 0, err
	}

	scanned = len(records)
	for _, r := range records {
		_, recErr := s.ReconcileRecord(r.ID)
		if recErr == nil {
			reconciled++
		}
	}
	return scanned, reconciled, nil
}
