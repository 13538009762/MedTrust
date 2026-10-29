package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/pkg/sse"
	"medtrust-backend/repository"
	"medtrust-backend/service"
)

type NotificationController struct{}

var DefaultNotificationController = &NotificationController{}

// Stream 建立客户端 SSE 长连接
// GET /api/v1/notifications/stream?token=...
func (ctrl *NotificationController) Stream(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.Status(http.StatusUnauthorized)
		return
	}
	userID := userIDVal.(uint64)

	// 设置 SSE 响应标头
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	// 订阅用户通知通道
	ch := sse.Default.Subscribe(userID)
	defer sse.Default.Unsubscribe(userID, ch)

	// 发送初次握手确认
	_, _ = fmt.Fprintf(c.Writer, "event: connected\ndata: {\"online\":true,\"user_id\":%d}\n\n", userID)
	c.Writer.Flush()

	// 启动心跳定时器（每25秒一次，保持 TCP 连接存活）
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	notifyClose := c.Request.Context().Done()

	for {
		select {
		case <-notifyClose:
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(event)
			if err == nil {
				_, _ = fmt.Fprintf(c.Writer, "event: notify\ndata: %s\n\n", string(data))
				c.Writer.Flush()
			}
		case <-ticker.C:
			_, _ = fmt.Fprintf(c.Writer, "event: heartbeat\ndata: {\"time\":%d}\n\n", time.Now().Unix())
			c.Writer.Flush()
		}
	}
}

// ConfirmAccessRequest 患者在线确认加急访问申请 (POST /api/v1/access-requests/:req_no/confirm)
func (ctrl *NotificationController) ConfirmAccessRequest(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	reqNo := c.Param("req_no")
	if reqNo == "" {
		reqNo = c.Param("id")
	}

	req, err := repository.DefaultAuthPolicyRepository.FindPatientAccessRequest(patientID, reqNo)
	if err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "未找到待确认的加急访问申请记录"})
		return
	}

	if req.Status == "APPROVED" {
		c.JSON(http.StatusOK, model.Response{Code: 200, Message: "该申请已确认同意授权", Data: req})
		return
	}

	// 自动颁发生效授权策略
	now := time.Now()
	startTime := now
	endTime := now.Add(24 * time.Hour)
	authNo := fmt.Sprintf("AUTH%s%04d", now.Format("20060102150405"), req.ID%10000)

	txID, _, _ := blockchain.DefaultService.CommitAsset("AUTHORIZATION", authNo, map[string]interface{}{
		"auth_no":     authNo,
		"patient_id":  patientID,
		"target_type": "DOCTOR",
		"target_id":   req.DoctorID,
		"scope":       req.ScopeType,
		"record_id":   req.RecordID,
		"status":      "ACTIVE",
		"timestamp":   now.Format(time.RFC3339),
	})

	auth := model.Authorization{
		AuthNo:         authNo,
		PatientID:      patientID,
		AuthTargetType: "DOCTOR",
		AuthTargetID:   req.DoctorID,
		ScopeType:      req.ScopeType,
		RecordID:       req.RecordID,
		StartTime:      startTime,
		EndTime:        endTime,
		Status:         "ACTIVE",
		FabricTxID:     txID,
		CreatedAt:      now,
	}
	_ = repository.DefaultAuthPolicyRepository.CreateAuthorization(&auth)

	req.Status = "APPROVED"
	req.Decision = "ALLOWED"
	_ = repository.DefaultAuthPolicyRepository.SaveAccessRequest(req)

	if service.DefaultAuditService != nil {
		service.DefaultAuditService.Log(patientID, "APPROVE_CONSENT", "AUTH", authNo, 0, "SUCCESS", "LOW", c.ClientIP())
	}

	// SSE 实时推送到医生工作台
	sse.Default.Publish(req.DoctorID, sse.NotifyEvent{
		Type:    "request_approved",
		Title:   "加急访问申请已获患者确认同意",
		Message: fmt.Sprintf("申请单 %s 已获患者在线核准同意，临时调阅凭据已生效并在区块链存证", req.RequestNo),
		Payload: map[string]interface{}{
			"request_id": req.ID,
			"request_no": req.RequestNo,
			"auth_no":    authNo,
			"record_id":  req.RecordID,
		},
	})

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "患者已在线确认同意加急访问，授权凭证已在区块链存证激活",
		Data:    auth,
	})
}

// EmergencyFeedback 患者对 Break-Glass 紧急访问的知情确认或投诉异议反馈接口
func (ctrl *NotificationController) EmergencyFeedback(c *gin.Context) {
	DefaultAccessController.PatientFeedback(c)
}

// ListNotifications 获取当前用户的实时通知/待办事项列表 (GET /api/v1/notifications)
func (ctrl *NotificationController) ListNotifications(c *gin.Context) {
	userID := c.GetUint64("user_id")
	userRole := c.GetString("role")

	var list []map[string]interface{}

	if userRole == "patient" {
		reqs, _ := repository.DefaultAuthPolicyRepository.FindPendingRequestsByPatient(userID, 10)
		for _, r := range reqs {
			list = append(list, map[string]interface{}{
				"id":         r.ID,
				"type":       "access_request",
				"title":      "待确认的加急访问申请",
				"message":    fmt.Sprintf("医生发起了关于就诊记录 #%d 的跨院调阅申请，等待您的确认", r.RecordID),
				"created_at": r.CreatedAt,
				"payload":    r,
			})
		}

		emgs, _ := repository.DefaultEmergencyRepository.FindPendingFeedbackEventsByPatient(userID, 10)
		for _, e := range emgs {
			list = append(list, map[string]interface{}{
				"id":         e.ID,
				"type":       "break_glass_alert",
				"title":      "紧急救治调阅事后知情通知",
				"message":    fmt.Sprintf("您的就诊记录发生紧急破窗调阅（原因：%s），请核实并提供反馈", e.EmergencyReason),
				"created_at": e.CreatedAt,
				"payload":    e,
			})
		}
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "获取通知列表成功",
		Data:    list,
	})
}
