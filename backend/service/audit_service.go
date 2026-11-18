package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	stdlog "log"
	"sync"
	"time"

	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
)

type auditChainJob struct {
	logRecordID uint64
	logID       string
	payload     map[string]interface{}
}

var (
	auditChainQueue = make(chan auditChainJob, 1024)
	auditWorkerOnce sync.Once
)

func ensureAuditChainWorkers() {
	auditWorkerOnce.Do(func() {
		const workers = 4
		for i := 0; i < workers; i++ {
			go func() {
				for job := range auditChainQueue {
					if blockchain.DefaultService == nil {
						continue
					}
					txID, _, cErr := CommitAssetWithCompensation("AUDIT", job.logID, job.payload)
					if txID != "" {
						_ = repository.DefaultAuditRepository.UpdateFabricTxID(job.logRecordID, txID)
					} else if cErr != nil {
						stdlog.Printf("[AuditChain] ⚠️ 审计存证上链失败，已入补偿队列 log_id=%s err=%v", job.logID, cErr)
					}
				}
			}()
		}
	})
}

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

	// 区块链存证通过有界工作池异步推进并回填 fabric_tx_id（杜绝高并发下 goroutine 无限膨胀）
	if blockchain.DefaultService != nil {
		ensureAuditChainWorkers()
		job := auditChainJob{
			logRecordID: log.ID,
			logID:       logID,
			payload: map[string]interface{}{
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
			},
		}
		select {
		case auditChainQueue <- job:
		default:
			stdlog.Printf("[AuditChain] ⚠️ 异步上链队列已满 (1024)，审计流水 %s 已保留于数据库待后续补偿对账", logID)
		}
	}
}
