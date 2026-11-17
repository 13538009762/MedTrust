package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"medtrust-backend/model"
	"medtrust-backend/repository"
	"medtrust-backend/service"
)

type SetRecordAccessPolicyDTO struct {
	PolicyType   string `json:"policy_type" binding:"required"` // ALL_DOCTORS, HOSPITAL, DOCTOR, PRIVATE
	AuthTargetID uint64 `json:"auth_target_id"`                 // 医院ID 或 医生用户ID
	Days         int    `json:"days"`                           // 默认 365 天
}

// SetRecordAccessPolicy 患者快速设置单份病历的访问权限
func (ctrl *AccessController) SetRecordAccessPolicy(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	recordID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	recPtr, err := repository.DefaultMedicalRepository.FindPatientRecord(recordID, patientID)
	if err != nil || recPtr == nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "未找到该就诊记录"})
		return
	}
	rec := *recPtr

	var req SetRecordAccessPolicyDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误：缺少访问策略类型"})
		return
	}

	result, err := service.DefaultAccessPolicyService.SetRecordAccessPolicy(serviceCtx(c), patientID, &rec, service.SetRecordAccessPolicyInput{
		RecordID:     recordID,
		PolicyType:   req.PolicyType,
		AuthTargetID: req.AuthTargetID,
		Days:         req.Days,
	})
	if err != nil {
		respondAccessPolicyError(c, err)
		return
	}

	auditTarget := "ALL_RECORDS"
	auditID := "ALL"
	if rec.RecordNo != "" {
		auditTarget = "RECORD"
		auditID = rec.RecordNo
	}
	service.DefaultAuditService.Log(patientID, result.AuditOp, auditTarget, auditID, 0, "SUCCESS", "LOW", c.ClientIP())

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: result.Message, Data: result.Auth})
}

type BatchSetAccessPolicyDTO struct {
	PolicyType   string `json:"policy_type" binding:"required"` // ALL_DOCTORS, HOSPITAL, DOCTOR, PRIVATE
	AuthTargetID uint64 `json:"auth_target_id"`
	Days         int    `json:"days"`
}

// BatchSetAccessPolicy 患者批量设置全部健康档案的访问共享权限
func (ctrl *AccessController) BatchSetAccessPolicy(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	var req BatchSetAccessPolicyDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误"})
		return
	}

	result, err := service.DefaultAccessPolicyService.BatchSetAccessPolicy(serviceCtx(c), patientID, service.BatchSetAccessPolicyInput{
		PolicyType:   req.PolicyType,
		AuthTargetID: req.AuthTargetID,
		Days:         req.Days,
	})
	if err != nil {
		respondAccessPolicyError(c, err)
		return
	}

	auditID := "ALL"
	if req.PolicyType == "HOSPITAL" {
		auditID = "HOSPITAL"
	} else if req.PolicyType == "DOCTOR" {
		auditID = "DOCTOR"
	}
	service.DefaultAuditService.Log(patientID, result.AuditOp, "ALL_RECORDS", auditID, 0, "SUCCESS", "LOW", c.ClientIP())

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: result.Message, Data: result.Auth})
}

// respondAccessPolicyError 依据领域错误类型映射 HTTP 状态码
func respondAccessPolicyError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrMissingTarget):
		// 按策略语义返回更具体的提示
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "请指定授权开放的目标机构或医生"})
	case errors.Is(err, service.ErrHospitalNotFound):
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
	case errors.Is(err, service.ErrDoctorNotFound):
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
	case errors.Is(err, service.ErrUnsupportedPolicyType):
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "不支持的策略类型，支持 ALL_DOCTORS、HOSPITAL、DOCTOR 或 PRIVATE"})
	case errors.Is(err, service.ErrAuthDaysExceeded):
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: err.Error()})
	}
}
