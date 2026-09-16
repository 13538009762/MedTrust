package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
)

type AuditEntry struct {
	UserID        uint64
	OperationType string
	TargetType    string
	TargetID      string
	HospitalID    uint64
	Result        string
	RiskLevel     string
	RiskScore     int
	Source        string // WEB, API, AI_AGENT, SYSTEM, BREAK_GLASS
	Reason        string
	IPAddress     string
	UserAgent     string
}

type AuditService struct{}

var DefaultAuditService = &AuditService{}

func (s *AuditService) Log(userID uint64, opType, targetType, targetID string, hospitalID uint64, result, riskLevel, ip string) {
	s.LogDetailed(AuditEntry{
		UserID:        userID,
		OperationType: opType,
		TargetType:    targetType,
		TargetID:      targetID,
		HospitalID:    hospitalID,
		Result:        result,
		RiskLevel:     riskLevel,
		IPAddress:     ip,
		Source:        "WEB",
	})
}

func (s *AuditService) LogDetailed(entry AuditEntry) {
	go func() {
		uuidBytes := make([]byte, 8)
		_, _ = rand.Read(uuidBytes)
		logID := fmt.Sprintf("LOG-%s-%d", hex.EncodeToString(uuidBytes), time.Now().Unix())

		if entry.Source == "" {
			entry.Source = "WEB"
		}
		if entry.RiskLevel == "" {
			entry.RiskLevel = "LOW"
		}

		// 上链存证
		var txID string
		if blockchain.DefaultService != nil {
			txID, _, _ = blockchain.DefaultService.CommitAsset("AUDIT", logID, map[string]interface{}{
				"user_id":     entry.UserID,
				"op_type":     entry.OperationType,
				"target_type": entry.TargetType,
				"target_id":   entry.TargetID,
				"hospital_id": entry.HospitalID,
				"result":      entry.Result,
				"risk_level":  entry.RiskLevel,
				"risk_score":  entry.RiskScore,
				"source":      entry.Source,
				"reason":      entry.Reason,
				"timestamp":   time.Now().Format(time.RFC3339),
			})
		}

		log := model.AuditLog{
			LogID:         logID,
			UserID:        entry.UserID,
			OperationType: entry.OperationType,
			TargetType:    entry.TargetType,
			TargetID:      entry.TargetID,
			HospitalID:    entry.HospitalID,
			Result:        entry.Result,
			RiskLevel:     entry.RiskLevel,
			RiskScore:     entry.RiskScore,
			Source:        entry.Source,
			Reason:        entry.Reason,
			IPAddress:     entry.IPAddress,
			UserAgent:     entry.UserAgent,
			FabricTxID:    txID,
			CreatedAt:     time.Now(),
		}
		if repository.DB != nil {
			repository.DB.Create(&log)
		}
	}()
}
