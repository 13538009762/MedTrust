package service

import (
	"fmt"
	"strings"
	"sync"

	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
)

type VerificationResult struct {
	RecordID       uint64 `json:"record_id"`
	RecordNo       string `json:"record_no"`
	CalculatedHash string `json:"calculated_hash"`
	ChainHash      string `json:"chain_hash"`
	Verified       bool   `json:"verified"`
	CID            string `json:"cid"`
	FabricTxID     string `json:"fabric_tx_id"`
	Message        string `json:"message"`
}

type RecordBackup struct {
	Diagnosis     string
	TreatmentPlan string
	Symptoms      string
}

type VerificationService struct {
	backups map[uint64]RecordBackup
	mu      sync.Mutex
}

var DefaultVerificationService = &VerificationService{
	backups: make(map[uint64]RecordBackup),
}

// Verify 执行动态闭环验真：从 IPFS 拉取密文解密，重算临床与明文哈希，自动比对 Fabric 账本原始指纹
func (s *VerificationService) Verify(recordID uint64) (*VerificationResult, error) {
	var record model.MedicalRecord
	if err := repository.DB.Preload("Files").First(&record, recordID).Error; err != nil {
		return nil, fmt.Errorf("病历不存在: %w", err)
	}

	DefaultMedicalService.VerifyRecord(&record)

	cid := ""
	if chainData, exists := blockchain.DefaultService.QueryAsset(record.RecordNo); exists && chainData != nil {
		if c, ok := chainData["cid"].(string); ok && c != "" {
			cid = c
		}
	}
	if cid == "" && len(record.Files) > 0 {
		cid = record.Files[len(record.Files)-1].IPFSCID
	}

	msg := "动态核验成功：数据库临床内容与 Fabric 链上固化凭证完全一致，数据真实完整，未遭篡改"
	if record.IsTampered {
		msg = "【高危安全警报】检测到数据完整性哈希不匹配！数据库中的病历数据已被非法篡改！"
	}

	DefaultAuditService.Log(0, "VERIFY", "RECORD", record.RecordNo, record.HospitalID, func() string {
		if !record.IsTampered {
			return "SUCCESS"
		}
		return "FAILED"
	}(), func() string {
		if !record.IsTampered {
			return "LOW"
		}
		return "HIGH"
	}(), "127.0.0.1")

	return &VerificationResult{
		RecordID:       record.ID,
		RecordNo:       record.RecordNo,
		CalculatedHash: record.CurrentHash,
		ChainHash:      record.ChainHash,
		Verified:       !record.IsTampered,
		CID:            cid,
		FabricTxID:     record.FabricTxID,
		Message:        msg,
	}, nil
}

// SimulateTamper 模拟真实数据库恶意篡改演练 (答辩核心演示亮点)
func (s *VerificationService) SimulateTamper(recordID uint64) (*VerificationResult, error) {
	var record model.MedicalRecord
	if err := repository.DB.First(&record, recordID).Error; err != nil {
		return nil, fmt.Errorf("病历不存在: %w", err)
	}

	s.mu.Lock()
	if _, exists := s.backups[recordID]; !exists {
		s.backups[recordID] = RecordBackup{
			Diagnosis:     record.Diagnosis,
			TreatmentPlan: record.TreatmentPlan,
			Symptoms:      record.Symptoms,
		}
	}
	s.mu.Unlock()

	// 模拟黑客绕过应用层直接篡改底层数据库内容
	record.Diagnosis = record.Diagnosis + " [恶意篡改：伪造头孢曲松钠严重过敏史与青霉素休克体征]"
	record.TreatmentPlan = "【非法篡改医嘱】立即停用原方案，改用大剂量高糖输注 (篡改指纹)"
	repository.DB.Save(&record)

	DefaultAuditService.Log(0, "TAMPER_ATTACK", "RECORD", record.RecordNo, record.HospitalID, "INTERCEPTED", "HIGH", "127.0.0.1")

	// 触发即时比对核验，立即产生红标高危警报
	return s.Verify(recordID)
}

// RestoreTamperedRecord 一键恢复病历真实数据并重新核验通过
func (s *VerificationService) RestoreTamperedRecord(recordID uint64) (*VerificationResult, error) {
	var record model.MedicalRecord
	if err := repository.DB.First(&record, recordID).Error; err != nil {
		return nil, fmt.Errorf("病历不存在: %w", err)
	}

	s.mu.Lock()
	backup, exists := s.backups[recordID]
	if exists {
		record.Diagnosis = backup.Diagnosis
		record.TreatmentPlan = backup.TreatmentPlan
		record.Symptoms = backup.Symptoms
		delete(s.backups, recordID)
	} else {
		record.Diagnosis = strings.ReplaceAll(record.Diagnosis, " [恶意篡改：伪造头孢曲松钠严重过敏史与青霉素休克体征]", "")
		record.TreatmentPlan = strings.ReplaceAll(record.TreatmentPlan, "【非法篡改医嘱】立即停用原方案，改用大剂量高糖输注 (篡改指纹)", "")
	}
	s.mu.Unlock()

	repository.DB.Save(&record)
	DefaultAuditService.Log(0, "TAMPER_RESTORE", "RECORD", record.RecordNo, record.HospitalID, "SUCCESS", "LOW", "127.0.0.1")

	return s.Verify(recordID)
}
