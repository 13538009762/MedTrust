package service

import (
	"fmt"
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

type VerificationService struct{}

var DefaultVerificationService = &VerificationService{}

func (s *VerificationService) Verify(recordID uint64) (*VerificationResult, error) {
	var record model.MedicalRecord
	if err := repository.DB.Preload("Files").First(&record, recordID).Error; err != nil {
		return nil, fmt.Errorf("病历不存在: %w", err)
	}

	DefaultMedicalService.VerifyRecord(&record)

	cid := ""
	if chainData, exists := blockchain.DefaultLedger.QueryAsset(record.RecordNo); exists && chainData != nil {
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
