package service

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
)

// MaxChainCompensationRetries 补偿任务最大重试次数；超过后标记 FAILED 转人工介入，
// 避免持续失败的存证无限占用重放队列。
const MaxChainCompensationRetries = 10

// CommitAssetWithCompensation (WP-10) 统一链上存证提交入口：
// 检查 CommitAsset 返回的 err；失败时记录结构化错误上下文、将关联病历置为 RETRYING 状态，
// 并插入 ConsistencyTask 补偿任务，由 ConsistencyService 后台自动对账补齐缺失的链上交易。
// 返回 (txID, height, err) 与底层 CommitAsset 一致；err 非空时调用方应中断后续本地写入或按业务降级。
func CommitAssetWithCompensation(businessType, businessID string, payload map[string]interface{}) (string, uint64, error) {
	txID, height, err := blockchain.DefaultService.CommitAsset(businessType, businessID, payload)
	if err == nil {
		return txID, height, nil
	}

	log.Printf("[ChainCompensation] ❌ 链上存证失败 business_type=%s business_id=%s err=%v", businessType, businessID, err)

	// WP-10: 序列化原始载荷，供补偿重放时对全业务类型自动上链
	payloadJSON, _ := json.Marshal(payload)

	// 1. 关联病历置为 RETRYING，纳入 ReconcileAllPendingOrFailed 扫描范围
	if repository.DB != nil {
		if businessType == "MEDICAL_RECORD" {
			if dbErr := repository.DB.Model(&model.MedicalRecord{}).
				Where("record_no = ?", businessID).
				Updates(map[string]interface{}{
					"sync_status": SyncStatusRetrying,
					"sync_error":  fmt.Sprintf("链上存证失败待补偿: %v", err),
				}).Error; dbErr != nil {
				log.Printf("[ChainCompensation] ⚠️ 更新病历补偿状态失败 record_no=%s err=%v", businessID, dbErr)
			}
		}

		// 2. 幂等插入补偿任务（同业务单号已有 PENDING_RETRY 任务时不重复入队）
		var existing model.ConsistencyTask
		if err2 := repository.DB.Where("business_type = ? AND business_id = ? AND status = ?",
			businessType, businessID, "PENDING_RETRY").First(&existing).Error; err2 != nil {
			task := model.ConsistencyTask{
				BusinessType: businessType,
				BusinessID:   businessID,
				PayloadJSON:  string(payloadJSON),
				Status:       "PENDING_RETRY",
				RetryCount:   0,
				LastError:    err.Error(),
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			if err3 := repository.DB.Create(&task).Error; err3 != nil {
				log.Printf("[ChainCompensation] ⚠️ 写入补偿任务失败 business_id=%s err=%v", businessID, err3)
			} else {
				log.Printf("[ChainCompensation] 📌 已登记链上补偿任务 task_id=%d business_type=%s business_id=%s", task.ID, businessType, businessID)
			}
		}
	}

	return "", 0, err
}

// ProcessPendingConsistencyTasks 后台处理补偿任务队列（幂等：仅重试仍处于 PENDING_RETRY 的任务）
func ProcessPendingConsistencyTasks() (processed int, recovered int) {
	if repository.DB == nil {
		return 0, 0
	}
	var tasks []model.ConsistencyTask
	if err := repository.DB.Where("status = ?", "PENDING_RETRY").Order("id asc").Limit(50).Find(&tasks).Error; err != nil {
		return 0, 0
	}
	for _, t := range tasks {
		processed++
		var recoveredOne bool
		switch t.BusinessType {
		case "MEDICAL_RECORD":
			var rec model.MedicalRecord
			if err := repository.DB.Where("record_no = ?", t.BusinessID).First(&rec).Error; err == nil {
				if _, rErr := DefaultConsistencyService.ReconcileRecord(rec.ID); rErr == nil {
					recoveredOne = true
				} else {
					t.LastError = rErr.Error()
				}
			}
		default:
			// 针对 REVOKE_AUTH 先查本地授权表：若用户已通过人工 retry-revoke 接口完成链上确权撤销，则直接收口任务，避免重复上链
			if t.BusinessType == "REVOKE_AUTH" || t.BusinessType == "AUTH_REVOKE" {
				if existingAuth, aErr := repository.DefaultAuthPolicyRepository.FindAuthorizationByNo(t.BusinessID); aErr == nil && existingAuth != nil {
					if existingAuth.Status == "REVOKED" && existingAuth.RevokeTxID != "" {
						recoveredOne = true
						break
					}
				}
			}

			// WP-10: 其余业务类型基于 PayloadJSON 自动重放上链，并按业务类型精确回写对应字段
			payload := make(map[string]interface{})
			if t.PayloadJSON != "" {
				if uErr := json.Unmarshal([]byte(t.PayloadJSON), &payload); uErr != nil {
					log.Printf("[ChainCompensation] ⚠️ 补偿任务载荷解析失败 task_id=%d err=%v", t.ID, uErr)
				}
			}
			if replayTxID, _, cErr := blockchain.DefaultService.CommitAsset(t.BusinessType, t.BusinessID, payload); cErr != nil {
				// 不在此处 continue，确保落入下方重试计数递进逻辑并在耗尽后置为 FAILED
				t.LastError = cErr.Error()
			} else {
				recoveredOne = true
				switch t.BusinessType {
				case "AUTHORIZATION", "AUTHORIZATION_KEY", "AUTH_RECORD":
					if uErr := repository.DB.Model(&model.Authorization{}).Where("auth_no = ?", t.BusinessID).
						Updates(map[string]interface{}{"fabric_tx_id": replayTxID, "updated_at": time.Now()}).Error; uErr != nil {
						log.Printf("[ChainCompensation] ⚠️ 回写授权 fabric_tx_id 失败 auth_no=%s txID=%s err=%v", t.BusinessID, replayTxID, uErr)
					}
				case "REVOKE_AUTH", "AUTH_REVOKE":
					if uErr := repository.DB.Model(&model.Authorization{}).Where("auth_no = ?", t.BusinessID).
						Updates(map[string]interface{}{
							"status":       "REVOKED",
							"revoke_tx_id": replayTxID,
							"revoke_error": "",
							"updated_at":   time.Now(),
						}).Error; uErr != nil {
						log.Printf("[ChainCompensation] ⚠️ 回写授权撤销 revoke_tx_id 失败 auth_no=%s txID=%s err=%v", t.BusinessID, replayTxID, uErr)
					}
				case "EMERGENCY_AUDIT", "EMERGENCY_ACCESS", "EMERGENCY", "EMERGENCY_STATUS":
					if uErr := repository.DB.Model(&model.EmergencyAccessEvent{}).Where("event_no = ?", t.BusinessID).
						Update("fabric_tx_id", replayTxID).Error; uErr != nil {
						log.Printf("[ChainCompensation] ⚠️ 回写紧急事件 fabric_tx_id 失败 event_no=%s txID=%s err=%v", t.BusinessID, replayTxID, uErr)
					}
				case "AUDIT", "AUDIT_LOG", "AUDIT_RECORD":
					if uErr := repository.DB.Model(&model.AuditLog{}).Where("log_id = ?", t.BusinessID).
						Update("fabric_tx_id", replayTxID).Error; uErr != nil {
						log.Printf("[ChainCompensation] ⚠️ 回写审计日志 fabric_tx_id 失败 log_id=%s txID=%s err=%v", t.BusinessID, replayTxID, uErr)
					}
				}
			}
		}
		if recoveredOne {
			recovered++
			if dErr := repository.DB.Model(&model.ConsistencyTask{}).Where("id = ?", t.ID).
				Updates(map[string]interface{}{"status": "DONE", "updated_at": time.Now()}).Error; dErr != nil {
				// mark-DONE 失败必须显式记录：任务保留 PENDING_RETRY，下轮扫描幂等重放（链上 CommitAsset 幂等），
				// 绝不能静默丢弃——否则同一笔存证可能被无限重提交或彻底丢失追踪。
				log.Printf("[ChainCompensation] ⚠️ 补偿任务标记 DONE 失败（已重放成功，任务保留待下轮幂等扫描）task_id=%d business_type=%s business_id=%s err=%v", t.ID, t.BusinessType, t.BusinessID, dErr)
			}
		} else {
			// 重试计数达到上限后转为 FAILED，避免永久失败的存证被无限重提交占用补偿队列
			nextRetry := t.RetryCount + 1
			nextStatus := "PENDING_RETRY"
			if nextRetry >= MaxChainCompensationRetries {
				nextStatus = "FAILED"
				log.Printf("[ChainCompensation] ❌ 补偿任务重试次数耗尽，标记 FAILED 待人工介入 task_id=%d business_type=%s business_id=%s last_error=%s",
					t.ID, t.BusinessType, t.BusinessID, t.LastError)
			}
			if rErr := repository.DB.Model(&model.ConsistencyTask{}).Where("id = ?", t.ID).
				Updates(map[string]interface{}{"status": nextStatus, "retry_count": nextRetry, "last_error": t.LastError, "updated_at": time.Now()}).Error; rErr != nil {
				log.Printf("[ChainCompensation] ⚠️ 补偿任务回写重试计数失败 task_id=%d err=%v", t.ID, rErr)
			}
		}
	}
	return processed, recovered
}