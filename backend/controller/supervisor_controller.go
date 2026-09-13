package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
	"medtrust-backend/service"
)

type SupervisorController struct{}

var DefaultSupervisorController = &SupervisorController{}

func (ctrl *SupervisorController) Overview(c *gin.Context) {
	var totalRecords int64
	repository.DB.Model(&model.MedicalRecord{}).Count(&totalRecords)

	var totalEmergencies int64
	repository.DB.Model(&model.EmergencyAccessEvent{}).Count(&totalEmergencies)

	var pendingAudits int64
	repository.DB.Model(&model.EmergencyAccessEvent{}).Where("audit_status = 'PENDING_AUDIT'").Count(&pendingAudits)

	var totalLogs int64
	repository.DB.Model(&model.AuditLog{}).Count(&totalLogs)

	totalTx, blockHeight := blockchain.DefaultLedger.GetStats()

	// 统计不同机构病历占比
	type HospStat struct {
		Name  string `json:"name"`
		Count int64  `json:"count"`
	}
	var hospStats []HospStat
	repository.DB.Raw(`
		SELECT h.name, COUNT(r.id) as count 
		FROM hospitals h 
		LEFT JOIN medical_records r ON h.id = r.hospital_id 
		GROUP BY h.id, h.name
	`).Scan(&hospStats)

	// 统计风险等级分布
	type RiskStat struct {
		RiskLevel string `json:"risk_level"`
		Count     int64  `json:"count"`
	}
	var riskStats []RiskStat
	repository.DB.Raw(`SELECT risk_level, COUNT(id) as count FROM audit_logs GROUP BY risk_level`).Scan(&riskStats)

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "查询成功",
		Data: gin.H{
			"total_records":     totalRecords,
			"total_emergencies": totalEmergencies,
			"pending_audits":    pendingAudits,
			"total_logs":        totalLogs,
			"total_tx":          totalTx,
			"block_height":      blockHeight,
			"hosp_stats":        hospStats,
			"risk_stats":        riskStats,
		},
	})
}

func (ctrl *SupervisorController) ListEmergencyEvents(c *gin.Context) {
	status := c.Query("status")
	query := repository.DB.Model(&model.EmergencyAccessEvent{})
	if status != "" {
		query = query.Where("audit_status = ?", status)
	}

	var list []model.EmergencyAccessEvent
	query.Order("id desc").Find(&list)

	for i := range list {
		var doc model.User
		if err := repository.DB.First(&doc, list[i].DoctorID).Error; err == nil {
			list[i].DoctorName = doc.RealName
		}
		var pat model.User
		if err := repository.DB.First(&pat, list[i].PatientID).Error; err == nil {
			list[i].PatientName = pat.RealName
		}
		var rec model.MedicalRecord
		if err := repository.DB.First(&rec, list[i].RecordID).Error; err == nil {
			list[i].RecordNo = rec.RecordNo
		}
		var h1, h2 model.Hospital
		if err := repository.DB.First(&h1, list[i].SourceHospitalID).Error; err == nil {
			list[i].SourceHospitalName = h1.Name
		}
		if err := repository.DB.First(&h2, list[i].TargetHospitalID).Error; err == nil {
			list[i].TargetHospitalName = h2.Name
		}
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "查询成功", Data: list})
}

type AuditEventRequest struct {
	AuditStatus  string `json:"audit_status" binding:"required"` // CLOSED_APPROVED, CLOSED_VIOLATION, UNDER_INVESTIGATION
	AuditComment string `json:"audit_comment"`
	Punishment   string `json:"punishment"` // NORMAL, RESTRICTED, DISABLED
}

func (ctrl *SupervisorController) AuditEmergencyEvent(c *gin.Context) {
	supervisorID := c.GetUint64("user_id")
	eventNo := c.Param("event_no")

	var req AuditEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误"})
		return
	}

	if err := service.DefaultEmergencyService.AuditEvent(supervisorID, eventNo, req.AuditStatus, req.AuditComment, req.Punishment); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "审核裁决与处置已固化生效"})
}

func (ctrl *SupervisorController) ListAuditLogs(c *gin.Context) {
	opType := c.Query("operation_type")
	riskLevel := c.Query("risk_level")

	query := repository.DB.Model(&model.AuditLog{})
	if opType != "" {
		query = query.Where("operation_type = ?", opType)
	}
	if riskLevel != "" {
		query = query.Where("risk_level = ?", riskLevel)
	}

	var list []model.AuditLog
	query.Order("id desc").Limit(100).Find(&list)

	for i := range list {
		var u model.User
		if err := repository.DB.First(&u, list[i].UserID).Error; err == nil {
			list[i].UserName = u.RealName
		}
		var h model.Hospital
		if err := repository.DB.First(&h, list[i].HospitalID).Error; err == nil {
			list[i].HospitalName = h.Name
		}
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "查询成功", Data: list})
}

func (ctrl *SupervisorController) VerifyRecord(c *gin.Context) {
	recordID, _ := strconv.ParseUint(c.Param("record_id"), 10, 64)
	res, err := service.DefaultVerificationService.Verify(recordID)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "防篡改核验完成", Data: res})
}
