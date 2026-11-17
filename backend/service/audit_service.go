package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	stdlog "log"
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

// AuditChainAnchoringPending 链上存证失败但已登记补偿队列时落一条显式审计。
// 修复 5：禁止把"存证确认中"静默伪装为"已上链"——撤销/破窗/授权等关键动作的
// 链上凭证缺失必须可被审计追溯，而非只在日志里一闪而过。
func AuditChainAnchoringPending(userID uint64, opType, targetType, targetID string, hospitalID uint64, commitErr error) {
	DefaultAuditService.LogDetailed(AuditEntry{
		UserID:        userID,
		OperationType: opType,
		TargetType:    targetType,
		TargetID:      targetID,
		HospitalID:    hospitalID,
		Result:        "PENDING_CHAIN_ANCHOR",
		RiskLevel:     "MEDIUM",
		Source:        "SYSTEM",
		Reason:        fmt.Sprintf("链上存证暂未固化为账本交易，已登记补偿队列待后台重放: %v", commitErr),
	})
}

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
	uuidBytes := make([]byte, 8)
	_, _ = rand.Read(uuidBytes)
	logID := fmt.Sprintf("LOG-%s-%d", hex.EncodeToString(uuidBytes), time.Now().UnixNano())

	if entry.Source == "" {
		entry.Source = "WEB"
	}
	if entry.RiskLevel == "" {
		entry.RiskLevel = "LOW"
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
		CreatedAt:     time.Now(),
	}
	_ = repository.DefaultAuditRepository.Create(&log)

	// 区块链存证异步推进并回填 fabric_tx_id
	go func(logRecordID uint64, lID string) {
		if blockchain.DefaultService != nil {
			txID, _, cErr := CommitAssetWithCompensation("AUDIT", lID, map[string]interface{}{
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
			if txID != "" {
				_ = repository.DefaultAuditRepository.UpdateFabricTxID(logRecordID, txID)
			} else if cErr != nil {
				// 审计日志本体已落库（DB 为准），补偿队列已接手链上重放；此处仅做结构化告警，
				// 不递归写审计（会再次触发上链，形成风暴）。
				stdlog.Printf("[AuditChain] ⚠️ 审计存证上链失败，已入补偿队列 log_id=%s err=%v", lID, cErr)
			}
		}
	}(log.ID, logID)
}
