package controller

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
	"medtrust-backend/service"
	"net/http"
	"strconv"
	"time"
)

func revokeActiveAuthorizationsWithChainSync(patientID uint64, scopeType string, recordID uint64) {
	var oldAuths []model.Authorization
	q := repository.DB.Where("patient_id = ? AND scope_type = ? AND status = 'ACTIVE'", patientID, scopeType)
	if scopeType == "SINGLE" {
		q = q.Where("record_id = ?", recordID)
	}
	if err := q.Find(&oldAuths).Error; err == nil {
		for _, oa := range oldAuths {
			if oa.AuthNo != "" && blockchain.DefaultService != nil {
				_, _, _ = blockchain.DefaultService.CommitAsset("REVOKE_AUTH", oa.AuthNo, map[string]interface{}{
					"auth_no":    oa.AuthNo,
					"patient_id": patientID,
					"status":     "REVOKED",
					"timestamp":  time.Now().Format(time.RFC3339),
				})
			}
			repository.DB.Model(&model.Authorization{}).Where("id = ?", oa.ID).Update("status", "REVOKED")
		}
	}
}

func (ctrl *AccessController) RevokeAuthorization(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	// 原子条件更新抢占撤销执行权：状态必须为 ACTIVE
	// 避免高并发场景下多个请求同时读取到 ACTIVE 并重复提交 Fabric 区块链交易
	res := repository.DB.Model(&model.Authorization{}).
		Where("id = ? AND patient_id = ? AND status = 'ACTIVE'", id, patientID).
		Updates(map[string]interface{}{
			"status":       "REVOKE_PENDING",
			"revoke_error": "",
			"operator_id":  patientID,
			"updated_at":   time.Now(),
		})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: fmt.Sprintf("数据库状态锁定更新失败: %v", res.Error)})
		return
	}

	if res.RowsAffected == 0 {
		// 未能成功抢占 ACTIVE 状态：查询当前真实状态并做出幂等或冲突响应
		var currentAuth model.Authorization
		if err := repository.DB.Where("id = ? AND patient_id = ?", id, patientID).First(&currentAuth).Error; err != nil {
			c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "未找到该授权条目"})
			return
		}

		// 幂等性保障：若已经处于撤销终态，直接返回 200 成功
		if currentAuth.Status == "REVOKED" {
			c.JSON(http.StatusOK, model.Response{Code: 200, Message: "授权先前已完成撤回与链上存证，无需重复撤销", Data: currentAuth})
			return
		}
		if currentAuth.Status == "REVOKE_PENDING" {
			c.JSON(http.StatusConflict, model.Response{Code: 409, Message: "该授权正在撤销处理中，请勿重复发起并发请求", Data: currentAuth})
			return
		}
		if currentAuth.Status == "NEEDS_RECONCILIATION" {
			c.JSON(http.StatusConflict, model.Response{Code: 409, Message: "该授权处于待对账修复状态（链上已成功，本地待同步），请通过重试补偿接口处理", Data: currentAuth})
			return
		}
		if currentAuth.Status == "REVOKE_FAILED" {
			c.JSON(http.StatusConflict, model.Response{Code: 409, Message: "该授权先前撤回失败，请使用重试补偿接口（/retry）发起重试", Data: currentAuth})
			return
		}
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: fmt.Sprintf("授权状态不支持撤回: %s", currentAuth.Status), Data: currentAuth})
		return
	}

	// 成功获得锁定的协程加载完整记录信息
	var auth model.Authorization
	if err := repository.DB.Where("id = ? AND patient_id = ?", id, patientID).First(&auth).Error; err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "读取授权记录失败"})
		return
	}

	// 状态机流转 2: 提交 Fabric 区块链撤销存证 (Fabric 为 nil 必须明确失败，严禁假成功)
	if blockchain.DefaultService == nil {
		auth.Status = "REVOKE_FAILED"
		auth.RevokeError = "区块链存证服务未初始化，禁止假成功撤回"
		auth.UpdatedAt = time.Now()
		_ = repository.DB.Model(&model.Authorization{}).Where("id = ?", auth.ID).Updates(map[string]interface{}{
			"status":       "REVOKE_FAILED",
			"revoke_error": auth.RevokeError,
			"updated_at":   auth.UpdatedAt,
		})
		service.DefaultAuditService.Log(patientID, "REVOKE", "AUTH", auth.AuthNo, 0, "FAILED", "HIGH", c.ClientIP())
		c.JSON(http.StatusServiceUnavailable, model.Response{
			Code:    503,
			Message: "区块链存证服务不可用，授权撤回中断（状态已安全保留为 REVOKE_FAILED，支持后续重试补偿）",
			Data:    auth,
		})
		return
	}

	txID, _, err := blockchain.DefaultService.CommitAsset("REVOKE_AUTH", auth.AuthNo, map[string]interface{}{
		"auth_no":    auth.AuthNo,
		"patient_id": patientID,
		"status":     "REVOKED",
		"timestamp":  time.Now().Format(time.RFC3339),
	})
	if err != nil {
		auth.Status = "REVOKE_FAILED"
		auth.RevokeError = fmt.Sprintf("区块链撤销交易提交失败: %v", err)
		auth.UpdatedAt = time.Now()
		_ = repository.DB.Model(&model.Authorization{}).Where("id = ?", auth.ID).Updates(map[string]interface{}{
			"status":       "REVOKE_FAILED",
			"revoke_error": auth.RevokeError,
			"updated_at":   auth.UpdatedAt,
		})
		service.DefaultAuditService.Log(patientID, "REVOKE", "AUTH", auth.AuthNo, 0, "FAILED", "HIGH", c.ClientIP())
		c.JSON(http.StatusInternalServerError, model.Response{
			Code:    500,
			Message: fmt.Sprintf("链上撤销存证失败: %v（已置为 REVOKE_FAILED，支持后续重试补偿）", err),
			Data:    auth,
		})
		return
	}

	// 状态机流转 3: 链上确权成功，持久化 DB 终态
	// 重点防御：若 Fabric 成功但 MySQL 更新失败，严禁冒充成功！转入 NEEDS_RECONCILIATION 并保留已生成的 RevokeTxID
	dbErr := repository.DB.Model(&model.Authorization{}).Where("id = ?", auth.ID).Updates(map[string]interface{}{
		"status":       "REVOKED",
		"revoke_tx_id": txID,
		"revoke_error": "",
		"operator_id":  patientID,
		"updated_at":   time.Now(),
	}).Error
	if dbErr != nil {
		errMsg := fmt.Sprintf("链上撤回已成功(TxID: %s)，但本地数据库状态持久化失败: %v", txID, dbErr)
		_ = repository.DB.Model(&model.Authorization{}).Where("id = ?", auth.ID).Updates(map[string]interface{}{
			"status":       "NEEDS_RECONCILIATION",
			"revoke_tx_id": txID,
			"revoke_error": errMsg,
			"operator_id":  patientID,
			"updated_at":   time.Now(),
		})
		service.DefaultAuditService.Log(patientID, "REVOKE_DESYNC", "AUTH", auth.AuthNo, 0, "FAILED", "HIGH", c.ClientIP())
		c.JSON(http.StatusInternalServerError, model.Response{
			Code:    500,
			Message: errMsg,
			Data: gin.H{
				"id":           auth.ID,
				"auth_no":      auth.AuthNo,
				"status":       "NEEDS_RECONCILIATION",
				"revoke_tx_id": txID,
				"error":        errMsg,
			},
		})
		return
	}

	auth.Status = "REVOKED"
	auth.RevokeTxID = txID
	auth.RevokeError = ""
	auth.UpdatedAt = time.Now()

	service.DefaultAuditService.Log(patientID, "REVOKE", "AUTH", auth.AuthNo, 0, "SUCCESS", "LOW", c.ClientIP())
	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "授权已成功撤回并完成联盟链确权存证", Data: auth})
}

type RevokeItemDetail struct {
	AuthID     uint64 `json:"auth_id"`
	AuthNo     string `json:"auth_no"`
	Status     string `json:"status"` // COMPLETED, REVOKE_FAILED, NEEDS_RECONCILIATION
	FabricTxID string `json:"fabric_tx_id,omitempty"`
	Error      string `json:"error,omitempty"`
}

type RevokeAllResultDTO struct {
	Total        int                `json:"total"`
	RevokedCount int                `json:"revoked_count"`
	FailedCount  int                `json:"failed_count"`
	Details      []RevokeItemDetail `json:"details"`
}

// RevokeAllAuthorizations 患者一键撤销名下所有有效授权 (支持区块链逐笔确权与链下强一致性状态机)
func (ctrl *AccessController) RevokeAllAuthorizations(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()
	source := c.GetHeader("X-Source")
	if source == "" {
		source = "WEB"
	}

	var activeAuths []model.Authorization
	if err := repository.DB.Where("patient_id = ? AND status = 'ACTIVE'", patientID).Find(&activeAuths).Error; err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: fmt.Sprintf("查询有效授权列表失败: %v", err)})
		return
	}

	total := len(activeAuths)
	if total == 0 {
		c.JSON(http.StatusOK, model.Response{Code: 200, Message: "当前名下暂无生效中的授权策略", Data: RevokeAllResultDTO{Total: 0}})
		return
	}

	revokedCount := 0
	failedCount := 0
	details := make([]RevokeItemDetail, 0, total)

	for _, a := range activeAuths {
		// 阶段 1: 原子条件更新抢占该条记录撤回权，防止并发一键撤销或并发单笔撤销产生重复 Fabric 上链
		claimRes := repository.DB.Model(&model.Authorization{}).
			Where("id = ? AND patient_id = ? AND status = 'ACTIVE'", a.ID, patientID).
			Updates(map[string]interface{}{
				"status":      "REVOKE_PENDING",
				"operator_id": patientID,
				"updated_at":  time.Now(),
			})
		if claimRes.Error != nil || claimRes.RowsAffected == 0 {
			// 该记录已被并发处理或已处于非 ACTIVE 状态，安全跳过
			continue
		}

		// 阶段 2: 提交 Fabric 区块链撤销存证 (严格判断 service 是否存在，禁止假成功)
		var txID string
		var err error
		if blockchain.DefaultService == nil {
			err = errors.New("区块链存证服务未就绪，禁止假成功撤回")
		} else {
			txID, _, err = blockchain.DefaultService.CommitAsset("REVOKE_AUTH", a.AuthNo, map[string]interface{}{
				"auth_no":    a.AuthNo,
				"patient_id": patientID,
				"status":     "REVOKED",
				"timestamp":  time.Now().Format(time.RFC3339),
			})
		}

		if err != nil {
			failedCount++
			errMsg := fmt.Sprintf("区块链撤销存证失败: %v", err)
			_ = repository.DB.Model(&model.Authorization{}).Where("id = ?", a.ID).Updates(map[string]interface{}{
				"status":       "REVOKE_FAILED",
				"revoke_error": errMsg,
				"operator_id":  patientID,
				"updated_at":   time.Now(),
			})
			details = append(details, RevokeItemDetail{
				AuthID: a.ID,
				AuthNo: a.AuthNo,
				Status: "REVOKE_FAILED",
				Error:  errMsg,
			})
		} else {
			// 阶段 3: 链上存证成功，完成终态持久化
			dbErr := repository.DB.Model(&model.Authorization{}).Where("id = ?", a.ID).Updates(map[string]interface{}{
				"status":       "REVOKED",
				"revoke_tx_id": txID,
				"revoke_error": "",
				"operator_id":  patientID,
				"updated_at":   time.Now(),
			}).Error
			if dbErr != nil {
				failedCount++
				errMsg := fmt.Sprintf("链上撤回已确权(TxID: %s)，但本地状态持久化失败: %v", txID, dbErr)
				_ = repository.DB.Model(&model.Authorization{}).Where("id = ?", a.ID).Updates(map[string]interface{}{
					"status":       "NEEDS_RECONCILIATION",
					"revoke_tx_id": txID,
					"revoke_error": errMsg,
					"operator_id":  patientID,
					"updated_at":   time.Now(),
				})
				details = append(details, RevokeItemDetail{
					AuthID:     a.ID,
					AuthNo:     a.AuthNo,
					Status:     "NEEDS_RECONCILIATION",
					FabricTxID: txID,
					Error:      errMsg,
				})
			} else {
				revokedCount++
				details = append(details, RevokeItemDetail{
					AuthID:     a.ID,
					AuthNo:     a.AuthNo,
					Status:     "COMPLETED",
					FabricTxID: txID,
				})
			}
		}
	}

	resResult := "SUCCESS"
	if failedCount > 0 && revokedCount == 0 {
		resResult = "FAILED"
	} else if failedCount > 0 {
		resResult = "PARTIAL"
	}

	service.DefaultAuditService.LogDetailed(service.AuditEntry{
		UserID:        patientID,
		OperationType: "REVOKE_ALL_AUTH",
		TargetType:    "AUTHORIZATION",
		TargetID:      fmt.Sprintf("SUCCESS_%d_FAIL_%d", revokedCount, failedCount),
		HospitalID:    0,
		Result:        resResult,
		RiskLevel:     "LOW",
		Source:        source,
		Reason:        fmt.Sprintf("患者一键撤销名下授权: 成功 %d / 失败 %d", revokedCount, failedCount),
		IPAddress:     clientIP,
		UserAgent:     userAgent,
	})

	resultDTO := RevokeAllResultDTO{
		Total:        total,
		RevokedCount: revokedCount,
		FailedCount:  failedCount,
		Details:      details,
	}

	msg := fmt.Sprintf("一键撤销处理完毕：成功撤销 %d 份，失败 %d 份（详情已上链存证）", revokedCount, failedCount)
	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: msg,
		Data:    resultDTO,
	})
}

// RetryRevokeAuthorization 单笔重试处于异常撤销状态（REVOKE_FAILED / NEEDS_RECONCILIATION / 超时的 REVOKE_PENDING）的授权
func (ctrl *AccessController) RetryRevokeAuthorization(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var auth model.Authorization
	if err := repository.DB.Where("id = ? AND patient_id = ?", id, patientID).First(&auth).Error; err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "未找到该授权条目"})
		return
	}

	if auth.Status == "REVOKED" {
		c.JSON(http.StatusOK, model.Response{Code: 200, Message: "该授权先前已成功完成链上撤销，无需重试", Data: auth})
		return
	}

	// 场景 A: NEEDS_RECONCILIATION 对账补偿
	// 链上先前已提交成功并留存 RevokeTxID，严禁重复提交 Fabric，直接补偿同步本地数据库为 REVOKED 终态
	if auth.Status == "NEEDS_RECONCILIATION" && auth.RevokeTxID != "" {
		if err := repository.DB.Model(&model.Authorization{}).Where("id = ?", auth.ID).Updates(map[string]interface{}{
			"status":       "REVOKED",
			"revoke_error": "",
			"updated_at":   time.Now(),
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: fmt.Sprintf("本地状态对账同步失败: %v", err)})
			return
		}
		auth.Status = "REVOKED"
		auth.RevokeError = ""
		service.DefaultAuditService.Log(patientID, "RECONCILE_REVOKE", "AUTH", auth.AuthNo, 0, "SUCCESS", "LOW", c.ClientIP())
		c.JSON(http.StatusOK, model.Response{Code: 200, Message: "已成功对账同步链上撤销终态", Data: auth})
		return
	}

	// 场景 B: REVOKE_PENDING 状态检查
	// 若处于 REVOKE_PENDING 且更新时间在 2 分钟以内，说明可能有在途处理，防止并发重试干扰
	if auth.Status == "REVOKE_PENDING" && time.Since(auth.UpdatedAt) < 2*time.Minute {
		c.JSON(http.StatusConflict, model.Response{Code: 409, Message: "该授权正在撤销处理中（耗时未超过 2 分钟），请勿频繁重试", Data: auth})
		return
	}

	// 状态机流转 1: 标记为 REVOKE_PENDING
	if err := repository.DB.Model(&model.Authorization{}).Where("id = ?", auth.ID).Updates(map[string]interface{}{
		"status":      "REVOKE_PENDING",
		"operator_id": patientID,
		"updated_at":  time.Now(),
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: fmt.Sprintf("更新重试状态失败: %v", err)})
		return
	}

	if blockchain.DefaultService == nil {
		auth.Status = "REVOKE_FAILED"
		auth.RevokeError = "区块链存证服务未就绪，重试中断"
		auth.UpdatedAt = time.Now()
		_ = repository.DB.Model(&model.Authorization{}).Where("id = ?", auth.ID).Updates(map[string]interface{}{
			"status":       "REVOKE_FAILED",
			"revoke_error": auth.RevokeError,
			"updated_at":   auth.UpdatedAt,
		})
		c.JSON(http.StatusServiceUnavailable, model.Response{Code: 503, Message: "区块链存证服务未连接，重试失败", Data: auth})
		return
	}

	txID, _, err := blockchain.DefaultService.CommitAsset("REVOKE_AUTH", auth.AuthNo, map[string]interface{}{
		"auth_no":    auth.AuthNo,
		"patient_id": patientID,
		"status":     "REVOKED",
		"timestamp":  time.Now().Format(time.RFC3339),
	})
	if err != nil {
		auth.Status = "REVOKE_FAILED"
		auth.RevokeError = fmt.Sprintf("重试提交区块链撤销失败: %v", err)
		auth.UpdatedAt = time.Now()
		_ = repository.DB.Model(&model.Authorization{}).Where("id = ?", auth.ID).Updates(map[string]interface{}{
			"status":       "REVOKE_FAILED",
			"revoke_error": auth.RevokeError,
			"updated_at":   auth.UpdatedAt,
		})
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: auth.RevokeError, Data: auth})
		return
	}

	dbErr := repository.DB.Model(&model.Authorization{}).Where("id = ?", auth.ID).Updates(map[string]interface{}{
		"status":       "REVOKED",
		"revoke_tx_id": txID,
		"revoke_error": "",
		"operator_id":  patientID,
		"updated_at":   time.Now(),
	}).Error
	if dbErr != nil {
		errMsg := fmt.Sprintf("重试链上撤回已成功(TxID: %s)，但本地数据库状态更新失败: %v", txID, dbErr)
		_ = repository.DB.Model(&model.Authorization{}).Where("id = ?", auth.ID).Updates(map[string]interface{}{
			"status":       "NEEDS_RECONCILIATION",
			"revoke_tx_id": txID,
			"revoke_error": errMsg,
			"operator_id":  patientID,
			"updated_at":   time.Now(),
		})
		c.JSON(http.StatusInternalServerError, model.Response{
			Code:    500,
			Message: errMsg,
			Data: gin.H{
				"id":           auth.ID,
				"status":       "NEEDS_RECONCILIATION",
				"revoke_tx_id": txID,
			},
		})
		return
	}

	auth.Status = "REVOKED"
	auth.RevokeTxID = txID
	auth.RevokeError = ""
	auth.UpdatedAt = time.Now()

	service.DefaultAuditService.Log(patientID, "RETRY_REVOKE", "AUTH", auth.AuthNo, 0, "SUCCESS", "LOW", c.ClientIP())
	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "授权条目已成功重试并完成联盟链确权撤销", Data: auth})
}

// RetryFailedRevocations 批量重试名下所有处于 REVOKE_FAILED / NEEDS_RECONCILIATION 或超时 REVOKE_PENDING 的授权条目
func (ctrl *AccessController) RetryFailedRevocations(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()

	var failedAuths []model.Authorization
	if err := repository.DB.Where("patient_id = ? AND status IN ('REVOKE_FAILED', 'REVOKE_PENDING', 'NEEDS_RECONCILIATION')", patientID).Find(&failedAuths).Error; err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: fmt.Sprintf("查询失败授权列表失败: %v", err)})
		return
	}

	total := len(failedAuths)
	if total == 0 {
		c.JSON(http.StatusOK, model.Response{Code: 200, Message: "当前名下没有需要重试的失败撤回条目", Data: RevokeAllResultDTO{Total: 0}})
		return
	}

	revokedCount := 0
	failedCount := 0
	details := make([]RevokeItemDetail, 0, total)

	for _, a := range failedAuths {
		// 场景 A: 处于 NEEDS_RECONCILIATION 且链上已生成 RevokeTxID
		if a.Status == "NEEDS_RECONCILIATION" && a.RevokeTxID != "" {
			err := repository.DB.Model(&model.Authorization{}).Where("id = ?", a.ID).Updates(map[string]interface{}{
				"status":       "REVOKED",
				"revoke_error": "",
				"updated_at":   time.Now(),
			}).Error
			if err == nil {
				revokedCount++
				details = append(details, RevokeItemDetail{
					AuthID:     a.ID,
					AuthNo:     a.AuthNo,
					Status:     "COMPLETED",
					FabricTxID: a.RevokeTxID,
				})
			} else {
				failedCount++
				details = append(details, RevokeItemDetail{
					AuthID: a.ID,
					AuthNo: a.AuthNo,
					Status: "NEEDS_RECONCILIATION",
					Error:  err.Error(),
				})
			}
			continue
		}

		// 场景 B: 处于 REVOKE_PENDING 且未超时（2分钟内），跳过以防干扰在途处理
		if a.Status == "REVOKE_PENDING" && time.Since(a.UpdatedAt) < 2*time.Minute {
			continue
		}

		claimRes := repository.DB.Model(&model.Authorization{}).
			Where("id = ? AND patient_id = ? AND status IN ('REVOKE_FAILED', 'REVOKE_PENDING')", a.ID, patientID).
			Updates(map[string]interface{}{
				"status":      "REVOKE_PENDING",
				"operator_id": patientID,
				"updated_at":  time.Now(),
			})
		if claimRes.Error != nil || claimRes.RowsAffected == 0 {
			continue
		}

		var txID string
		var err error
		if blockchain.DefaultService == nil {
			err = errors.New("区块链存证服务未就绪，重试中断")
		} else {
			txID, _, err = blockchain.DefaultService.CommitAsset("REVOKE_AUTH", a.AuthNo, map[string]interface{}{
				"auth_no":    a.AuthNo,
				"patient_id": patientID,
				"status":     "REVOKED",
				"timestamp":  time.Now().Format(time.RFC3339),
			})
		}

		if err != nil {
			failedCount++
			errMsg := fmt.Sprintf("重试区块链撤销存证失败: %v", err)
			_ = repository.DB.Model(&model.Authorization{}).Where("id = ?", a.ID).Updates(map[string]interface{}{
				"status":       "REVOKE_FAILED",
				"revoke_error": errMsg,
				"operator_id":  patientID,
				"updated_at":   time.Now(),
			})
			details = append(details, RevokeItemDetail{
				AuthID: a.ID,
				AuthNo: a.AuthNo,
				Status: "REVOKE_FAILED",
				Error:  errMsg,
			})
		} else {
			dbErr := repository.DB.Model(&model.Authorization{}).Where("id = ?", a.ID).Updates(map[string]interface{}{
				"status":       "REVOKED",
				"revoke_tx_id": txID,
				"revoke_error": "",
				"operator_id":  patientID,
				"updated_at":   time.Now(),
			}).Error
			if dbErr != nil {
				failedCount++
				errMsg := fmt.Sprintf("链上撤回已确权(TxID: %s)，但本地状态持久化失败: %v", txID, dbErr)
				_ = repository.DB.Model(&model.Authorization{}).Where("id = ?", a.ID).Updates(map[string]interface{}{
					"status":       "NEEDS_RECONCILIATION",
					"revoke_tx_id": txID,
					"revoke_error": errMsg,
					"operator_id":  patientID,
					"updated_at":   time.Now(),
				})
				details = append(details, RevokeItemDetail{
					AuthID:     a.ID,
					AuthNo:     a.AuthNo,
					Status:     "NEEDS_RECONCILIATION",
					FabricTxID: txID,
					Error:      errMsg,
				})
			} else {
				revokedCount++
				details = append(details, RevokeItemDetail{
					AuthID:     a.ID,
					AuthNo:     a.AuthNo,
					Status:     "COMPLETED",
					FabricTxID: txID,
				})
			}
		}
	}

	service.DefaultAuditService.LogDetailed(service.AuditEntry{
		UserID:        patientID,
		OperationType: "RETRY_ALL_FAILED_AUTH",
		TargetType:    "AUTHORIZATION",
		TargetID:      fmt.Sprintf("SUCCESS_%d_FAIL_%d", revokedCount, failedCount),
		HospitalID:    0,
		Result:        "COMPLETED",
		RiskLevel:     "LOW",
		Source:        "WEB",
		Reason:        fmt.Sprintf("重试补偿撤销名下失败授权: 成功 %d / 仍失败 %d", revokedCount, failedCount),
		IPAddress:     clientIP,
		UserAgent:     userAgent,
	})

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: fmt.Sprintf("重试对账完成：成功确权撤回 %d 份，仍失败 %d 份", revokedCount, failedCount),
		Data: RevokeAllResultDTO{
			Total:        total,
			RevokedCount: revokedCount,
			FailedCount:  failedCount,
			Details:      details,
		},
	})
}
