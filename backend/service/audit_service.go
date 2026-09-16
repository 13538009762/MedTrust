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

type AuditService struct{}

var DefaultAuditService = &AuditService{}

func (s *AuditService) Log(userID uint64, opType, targetType, targetID string, hospitalID uint64, result, riskLevel, ip string) {
	go func() {
		uuidBytes := make([]byte, 8)
		_, _ = rand.Read(uuidBytes)
		logID := fmt.Sprintf("LOG-%s-%d", hex.EncodeToString(uuidBytes), time.Now().Unix())

		// 上链存证
		var txID string
		if blockchain.DefaultService != nil {
			txID, _, _ = blockchain.DefaultService.CommitAsset("AUDIT", logID, map[string]interface{}{
				"user_id":     userID,
				"op_type":     opType,
				"target_type": targetType,
				"target_id":   targetID,
				"hospital_id": hospitalID,
				"result":      result,
				"risk_level":  riskLevel,
				"timestamp":   time.Now().Format(time.RFC3339),
			})
		}

		log := model.AuditLog{
			LogID:         logID,
			UserID:        userID,
			OperationType: opType,
			TargetType:    targetType,
			TargetID:      targetID,
			HospitalID:    hospitalID,
			Result:        result,
			RiskLevel:     riskLevel,
			IPAddress:     ip,
			FabricTxID:    txID,
			CreatedAt:     time.Now(),
		}
		repository.DB.Create(&log)
	}()
}
