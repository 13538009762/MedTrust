package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"medtrust-backend/model"
	"medtrust-backend/pkg/sse"
	"medtrust-backend/repository"
	"medtrust-backend/service"
	"net/http"
	"strings"
	"time"
)

type EmergencyBatchAccessDTO struct {
	PatientID       uint64   `json:"patient_id" binding:"required"`
	RecordIDs       []uint64 `json:"record_ids"`                          // 可选，若为空则针对该患者所有病历
	EmergencyReason string   `json:"emergency_reason" binding:"required"` // RESCUE, COMA, CRITICAL, OTHER
	Description     string   `json:"description" binding:"required"`
	DoctorConfirmed bool     `json:"doctor_confirmed" binding:"required"`
}

// EmergencyBatchAccess 急救抢救场景：医生一键开启绿色通道/紧急破窗调阅该患者全部病历
func (ctrl *AccessController) EmergencyBatchAccess(c *gin.Context) {
	doctorID := c.GetUint64("user_id")
	var req EmergencyBatchAccessDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误：请勾选医生法定免责声明并提供抢救原由"})
		return
	}

	if !req.DoctorConfirmed {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "开启急诊破窗绿色通道必须经执业医生本人签署知情法律声明"})
		return
	}

	var patient model.User
	if err := repository.DB.First(&patient, req.PatientID).Error; err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "患者档案不存在"})
		return
	}

	var doctor model.User
	_ = repository.DB.First(&doctor, doctorID)

	var records []model.MedicalRecord
	if len(req.RecordIDs) > 0 {
		repository.DB.Where("patient_id = ? AND id IN ?", req.PatientID, req.RecordIDs).Find(&records)
	} else {
		repository.DB.Where("patient_id = ?", req.PatientID).Find(&records)
	}

	if len(records) == 0 {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "该患者名下暂无可解锁的就诊档案"})
		return
	}

	var events []model.EmergencyAccessEvent
	for _, rec := range records {
		ev, err := service.DefaultEmergencyService.SubmitEmergencyAccess(
			doctorID,
			rec.ID,
			req.EmergencyReason,
			req.Description,
			req.DoctorConfirmed,
			c.ClientIP(),
		)
		if err == nil && ev != nil {
			events = append(events, *ev)
		}
	}

	service.DefaultAuditService.Log(doctorID, "EMERGENCY_BATCH_ACCESS", "PATIENT", patient.UserNo, doctor.HospitalID, "SUCCESS", "HIGH", c.ClientIP())

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: fmt.Sprintf("【急诊绿色通道已放行】已成功对患者「%s」的 %d 份病历执行紧急破窗调阅，事件与区块链存证已实时固化至 Fabric 分布式账本！", patient.RealName, len(events)),
		Data: gin.H{
			"unlocked_count": len(events),
			"events":         events,
		},
	})
}

// ListPendingRequests 患者查询向自己发起的待处理调阅申请

type BreakGlassRequest struct {
	RecordID        uint64      `json:"record_id"`
	EmergencyReason string      `json:"emergency_reason"` // RESCUE, COMA, CRITICAL, OTHER
	Description     string      `json:"description"`
	DoctorConfirmed interface{} `json:"doctor_confirmed"`
}

func formatEmergencyEventResponse(ev *model.EmergencyAccessEvent, rec *model.MedicalRecord) gin.H {
	return gin.H{
		"id":                 ev.ID,
		"event_no":           ev.EventNo,
		"doctor_id":          ev.DoctorID,
		"patient_id":         ev.PatientID,
		"record_id":          ev.RecordID,
		"source_hospital_id": ev.SourceHospitalID,
		"target_hospital_id": ev.TargetHospitalID,
		"emergency_reason":   ev.EmergencyReason,
		"description":        ev.Description,
		"doctor_confirmed":   ev.DoctorConfirmed,
		"patient_feedback":   ev.PatientFeedback,
		"patient_comment":    ev.PatientComment,
		"audit_status":       ev.AuditStatus,
		"supervisor_id":      ev.SupervisorID,
		"audit_comment":      ev.AuditComment,
		"fabric_tx_id":       ev.FabricTxID,
		"created_at":         ev.CreatedAt,
		"updated_at":         ev.UpdatedAt,
		"event":              *ev,
		"record":             rec,
	}
}

// BreakGlass 医生发起紧急抢救调阅
func (ctrl *AccessController) BreakGlass(c *gin.Context) {
	doctorID := c.GetUint64("user_id")
	var req BreakGlassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "必须如实填写抢救说明并勾选临床法律责任免责声明"})
		return
	}

	isConfirmed := false
	switch v := req.DoctorConfirmed.(type) {
	case bool:
		isConfirmed = v
	case float64:
		isConfirmed = (v == 1)
	case int:
		isConfirmed = (v == 1)
	case string:
		isConfirmed = (v == "1" || strings.EqualFold(v, "true"))
	}

	if !isConfirmed || req.RecordID == 0 || req.EmergencyReason == "" || len([]rune(req.Description)) < 10 {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "必须如实填写抢救说明并勾选临床法律责任免责声明"})
		return
	}

	var rec model.MedicalRecord
	if err := repository.DB.First(&rec, req.RecordID).Error; err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "病历不存在"})
		return
	}

	// 统一权限网关的前置核验 (医生身份、是否被处罚 RESTRICTED)
	decision, _ := service.DefaultAccessEngine.EvaluateAccess(doctorID, rec.PatientID, req.RecordID, true)
	if !decision.Allowed {
		c.JSON(http.StatusForbidden, model.Response{Code: 403, Message: decision.Reason, Data: decision})
		return
	}

	// 检查同一医生针对同一患者及病历在最近 1 小时内是否已经存在状态为有效/待审的紧急访问记录
	var existingEvent model.EmergencyAccessEvent
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	if err := repository.DB.Where("doctor_id = ? AND patient_id = ? AND record_id = ? AND created_at >= ? AND audit_status != 'CLOSED_VIOLATION'",
		doctorID, rec.PatientID, req.RecordID, oneHourAgo).Order("id desc").First(&existingEvent).Error; err == nil && existingEvent.ID > 0 {
		// 已存在有效破窗凭证，复用并返回病历数据
		fullRecord, _ := service.DefaultMedicalService.GetRecordByID(req.RecordID)
		c.JSON(http.StatusOK, model.Response{
			Code:    200,
			Message: "该病历在 1 小时内已处于有效急救破窗授权状态，直接复用既有紧急调阅凭证放行",
			Data:    formatEmergencyEventResponse(&existingEvent, fullRecord),
		})
		return
	}

	// 生成事件单并上链
	event, err := service.DefaultEmergencyService.SubmitEmergencyAccess(doctorID, req.RecordID, req.EmergencyReason, req.Description, isConfirmed, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: err.Error()})
		return
	}

	// 解密放行病历数据
	fullRecord, _ := service.DefaultMedicalService.GetRecordByID(req.RecordID)

	// 通过 SSE 实时推送急救破窗知情通知给患者
	sse.Default.Publish(rec.PatientID, sse.NotifyEvent{
		Type:    "break_glass",
		Title:   "紧急救治破窗访问通知",
		Message: fmt.Sprintf("急诊医生对您的病历（%s）执行了紧急抢救破窗调阅，事件已固化上链，请及时在知情面板核查", rec.RecordNo),
		Payload: map[string]interface{}{
			"record_id": rec.ID,
			"record_no": rec.RecordNo,
			"reason":    req.EmergencyReason,
		},
	})

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "紧急访问申请已核准放行，生成待审核事件单并全量存证",
		Data:    formatEmergencyEventResponse(event, fullRecord),
	})
}

type PatientFeedbackRequest struct {
	EventNo  string `json:"event_no"`
	Feedback string `json:"feedback"` // CONFIRMED, OBJECTED
	Comment  string `json:"comment"`
}

func (ctrl *AccessController) PatientFeedback(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	var req PatientFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误: " + err.Error()})
		return
	}

	if req.EventNo == "" {
		req.EventNo = c.Param("event_no")
	}
	if req.EventNo == "" || req.Feedback == "" {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误：缺少事件单号或反馈意见"})
		return
	}

	if err := service.DefaultEmergencyService.SubmitPatientFeedback(patientID, req.EventNo, req.Feedback, req.Comment, c.ClientIP()); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "反馈已成功提交并归档"})
}
