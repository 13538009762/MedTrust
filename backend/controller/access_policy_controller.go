package controller

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
	"medtrust-backend/service"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type SetRecordAccessPolicyDTO struct {
	PolicyType   string `json:"policy_type" binding:"required"` // ALL_DOCTORS, HOSPITAL, DOCTOR, PRIVATE
	AuthTargetID uint64 `json:"auth_target_id"`                 // 医院ID 或 医生用户ID
	Days         int    `json:"days"`                           // 默认 365 天
}

// SetRecordAccessPolicy 患者快捷设置单份病历的访问权限（如：让所有医生都可见 / 指定医院全体医生 / 指定医生 / 恢复私密）
func (ctrl *AccessController) SetRecordAccessPolicy(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	recordID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	recPtr, err := repository.DefaultMedicalRepository.FindPatientRecord(recordID, patientID)
	if err != nil || recPtr == nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "未找到该就诊病历记录"})
		return
	}
	rec := *recPtr

	var req SetRecordAccessPolicyDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误：缺少访问策略类型"})
		return
	}

	now := time.Now()
	days := req.Days
	if days <= 0 {
		days = 365
	}
	startTime := now
	endTime := startTime.Add(time.Duration(days) * 24 * time.Hour)

	if req.PolicyType == "ALL_DOCTORS" {
		// 先撤销旧的单份活跃授权
		revokeActiveAuthorizationsWithChainSync(patientID, "SINGLE", recordID)

		randBytes := make([]byte, 3)
		_, _ = rand.Read(randBytes)
		authNo := fmt.Sprintf("AUTH%s%s", now.Format("20060102"), hex.EncodeToString(randBytes))

		txID, _, _ := blockchain.DefaultService.CommitAsset("AUTHORIZATION", authNo, map[string]interface{}{
			"auth_no":     authNo,
			"patient_id":  patientID,
			"target_type": "ALL_DOCTORS",
			"target_id":   0,
			"scope":       "SINGLE",
			"record_id":   recordID,
			"status":      "ACTIVE",
			"timestamp":   now.Format(time.RFC3339),
		})

		auth := model.Authorization{
			AuthNo:         authNo,
			PatientID:      patientID,
			AuthTargetType: "ALL_DOCTORS",
			AuthTargetID:   0,
			ScopeType:      "SINGLE",
			RecordID:       recordID,
			StartTime:      startTime,
			EndTime:        endTime,
			Status:         "ACTIVE",
			FabricTxID:     txID,
			CreatedAt:      now,
		}
		_ = repository.DefaultAuthPolicyRepository.CreateAuthorization(&auth)

		service.DefaultAuditService.Log(patientID, "SET_RECORD_VISIBILITY", "RECORD", rec.RecordNo, 0, "SUCCESS", "LOW", c.ClientIP())
		c.JSON(http.StatusOK, model.Response{
			Code:    200,
			Message: "已成功设置该病历对「全体医生可见」，所有医院执业医生可直接免审调阅，策略已固化上链",
			Data:    auth,
		})
	} else if req.PolicyType == "HOSPITAL" {
		if req.AuthTargetID == 0 {
			c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "请指定授权开放的目标医疗机构"})
			return
		}
		hospPtr, err := repository.DefaultHospitalRepository.FindByID(req.AuthTargetID)
		if err != nil || hospPtr == nil {
			c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "指定的医疗机构不存在"})
			return
		}
		hosp := *hospPtr

		// 先撤销旧的单份活跃授权
		revokeActiveAuthorizationsWithChainSync(patientID, "SINGLE", recordID)

		randBytes := make([]byte, 3)
		_, _ = rand.Read(randBytes)
		authNo := fmt.Sprintf("AUTH%s%s", now.Format("20060102"), hex.EncodeToString(randBytes))

		txID, _, _ := blockchain.DefaultService.CommitAsset("AUTHORIZATION", authNo, map[string]interface{}{
			"auth_no":     authNo,
			"patient_id":  patientID,
			"target_type": "HOSPITAL",
			"target_id":   req.AuthTargetID,
			"scope":       "SINGLE",
			"record_id":   recordID,
			"status":      "ACTIVE",
			"timestamp":   now.Format(time.RFC3339),
		})

		auth := model.Authorization{
			AuthNo:         authNo,
			PatientID:      patientID,
			AuthTargetType: "HOSPITAL",
			AuthTargetID:   req.AuthTargetID,
			ScopeType:      "SINGLE",
			RecordID:       recordID,
			StartTime:      startTime,
			EndTime:        endTime,
			Status:         "ACTIVE",
			FabricTxID:     txID,
			CreatedAt:      now,
		}
		_ = repository.DefaultAuthPolicyRepository.CreateAuthorization(&auth)

		service.DefaultAuditService.Log(patientID, "SET_RECORD_VISIBILITY", "RECORD", rec.RecordNo, 0, "SUCCESS", "LOW", c.ClientIP())
		c.JSON(http.StatusOK, model.Response{
			Code:    200,
			Message: fmt.Sprintf("已成功设置该病历对「%s」全体医生可见，策略已固化上链", hosp.Name),
			Data:    auth,
		})
	} else if req.PolicyType == "DOCTOR" {
		if req.AuthTargetID == 0 {
			c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "请指定授权开放的目标医生"})
			return
		}
		docPtr, err := repository.DefaultUserRepository.FindDoctorByID(req.AuthTargetID)
		if err != nil || docPtr == nil {
			c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "指定的医生不存在或非执业医生"})
			return
		}
		doc := *docPtr

		// 先撤销旧的单份活跃授权
		revokeActiveAuthorizationsWithChainSync(patientID, "SINGLE", recordID)

		randBytes := make([]byte, 3)
		_, _ = rand.Read(randBytes)
		authNo := fmt.Sprintf("AUTH%s%s", now.Format("20060102"), hex.EncodeToString(randBytes))

		txID, _, _ := blockchain.DefaultService.CommitAsset("AUTHORIZATION", authNo, map[string]interface{}{
			"auth_no":     authNo,
			"patient_id":  patientID,
			"target_type": "DOCTOR",
			"target_id":   req.AuthTargetID,
			"scope":       "SINGLE",
			"record_id":   recordID,
			"status":      "ACTIVE",
			"timestamp":   now.Format(time.RFC3339),
		})

		auth := model.Authorization{
			AuthNo:         authNo,
			PatientID:      patientID,
			AuthTargetType: "DOCTOR",
			AuthTargetID:   req.AuthTargetID,
			ScopeType:      "SINGLE",
			RecordID:       recordID,
			StartTime:      startTime,
			EndTime:        endTime,
			Status:         "ACTIVE",
			FabricTxID:     txID,
			CreatedAt:      now,
		}
		_ = repository.DefaultAuthPolicyRepository.CreateAuthorization(&auth)

		docName := doc.RealName
		if !strings.HasSuffix(docName, "医生") && !strings.HasSuffix(docName, "医师") {
			docName += " 医生"
		}
		service.DefaultAuditService.Log(patientID, "SET_RECORD_VISIBILITY", "RECORD", rec.RecordNo, 0, "SUCCESS", "LOW", c.ClientIP())
		c.JSON(http.StatusOK, model.Response{
			Code:    200,
			Message: fmt.Sprintf("已成功设置该病历对「%s」专属授权调阅，策略已固化上链", docName),
			Data:    auth,
		})
	} else if req.PolicyType == "PRIVATE" {
		// 撤销对该病历的所有单份活跃授权
		revokeActiveAuthorizationsWithChainSync(patientID, "SINGLE", recordID)

		randBytes := make([]byte, 3)
		_, _ = rand.Read(randBytes)
		authNo := fmt.Sprintf("AUTH%s%s", now.Format("20060102"), hex.EncodeToString(randBytes))

		txID, _, _ := blockchain.DefaultService.CommitAsset("AUTHORIZATION", authNo, map[string]interface{}{
			"auth_no":     authNo,
			"patient_id":  patientID,
			"target_type": "PRIVATE",
			"target_id":   0,
			"scope":       "SINGLE",
			"record_id":   recordID,
			"status":      "ACTIVE",
			"timestamp":   now.Format(time.RFC3339),
		})

		auth := model.Authorization{
			AuthNo:         authNo,
			PatientID:      patientID,
			AuthTargetType: "PRIVATE",
			AuthTargetID:   0,
			ScopeType:      "SINGLE",
			RecordID:       recordID,
			StartTime:      now,
			EndTime:        now.Add(3650 * 24 * time.Hour),
			Status:         "ACTIVE",
			FabricTxID:     txID,
			CreatedAt:      now,
		}
		_ = repository.DefaultAuthPolicyRepository.CreateAuthorization(&auth)

		service.DefaultAuditService.Log(patientID, "REVOKE_RECORD_VISIBILITY", "RECORD", rec.RecordNo, 0, "SUCCESS", "LOW", c.ClientIP())
		c.JSON(http.StatusOK, model.Response{
			Code:    200,
			Message: "已将该病历恢复为「私密受控」，其他医生调阅需通过系统知情同意审批，存证已固化上链",
			Data:    auth,
		})
	} else {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "不支持的策略类型，支持 ALL_DOCTORS、HOSPITAL、DOCTOR 或 PRIVATE"})
	}
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

	now := time.Now()
	days := req.Days
	if days <= 0 {
		days = 365
	}
	startTime := now
	endTime := startTime.Add(time.Duration(days) * 24 * time.Hour)

	if req.PolicyType == "ALL_DOCTORS" {
		// 撤销旧的全局策略
		revokeActiveAuthorizationsWithChainSync(patientID, "ALL", 0)

		randBytes := make([]byte, 3)
		_, _ = rand.Read(randBytes)
		authNo := fmt.Sprintf("AUTH%s%s", now.Format("20060102"), hex.EncodeToString(randBytes))

		txID, _, _ := blockchain.DefaultService.CommitAsset("AUTHORIZATION", authNo, map[string]interface{}{
			"auth_no":     authNo,
			"patient_id":  patientID,
			"target_type": "ALL_DOCTORS",
			"target_id":   0,
			"scope":       "ALL",
			"record_id":   0,
			"status":      "ACTIVE",
			"timestamp":   now.Format(time.RFC3339),
		})

		auth := model.Authorization{
			AuthNo:         authNo,
			PatientID:      patientID,
			AuthTargetType: "ALL_DOCTORS",
			AuthTargetID:   0,
			ScopeType:      "ALL",
			RecordID:       0,
			StartTime:      startTime,
			EndTime:        endTime,
			Status:         "ACTIVE",
			FabricTxID:     txID,
			CreatedAt:      now,
		}
		_ = repository.DefaultAuthPolicyRepository.CreateAuthorization(&auth)

		service.DefaultAuditService.Log(patientID, "SET_ALL_VISIBILITY", "ALL_RECORDS", "ALL", 0, "SUCCESS", "LOW", c.ClientIP())
		c.JSON(http.StatusOK, model.Response{
			Code:    200,
			Message: "已成功开启「全部健康档案对所有医生可见」，全联盟医生均可直接查阅您的历史健康档案",
			Data:    auth,
		})
	} else if req.PolicyType == "HOSPITAL" {
		if req.AuthTargetID == 0 {
			c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "请指定授权开放的目标医疗机构"})
			return
		}
		hospPtr, err := repository.DefaultHospitalRepository.FindByID(req.AuthTargetID)
		if err != nil || hospPtr == nil {
			c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "指定的医疗机构不存在"})
			return
		}
		hosp := *hospPtr

		revokeActiveAuthorizationsWithChainSync(patientID, "ALL", 0)

		randBytes := make([]byte, 3)
		_, _ = rand.Read(randBytes)
		authNo := fmt.Sprintf("AUTH%s%s", now.Format("20060102"), hex.EncodeToString(randBytes))

		txID, _, _ := blockchain.DefaultService.CommitAsset("AUTHORIZATION", authNo, map[string]interface{}{
			"auth_no":     authNo,
			"patient_id":  patientID,
			"target_type": "HOSPITAL",
			"target_id":   req.AuthTargetID,
			"scope":       "ALL",
			"record_id":   0,
			"status":      "ACTIVE",
			"timestamp":   now.Format(time.RFC3339),
		})

		auth := model.Authorization{
			AuthNo:         authNo,
			PatientID:      patientID,
			AuthTargetType: "HOSPITAL",
			AuthTargetID:   req.AuthTargetID,
			ScopeType:      "ALL",
			RecordID:       0,
			StartTime:      startTime,
			EndTime:        endTime,
			Status:         "ACTIVE",
			FabricTxID:     txID,
			CreatedAt:      now,
		}
		_ = repository.DefaultAuthPolicyRepository.CreateAuthorization(&auth)

		service.DefaultAuditService.Log(patientID, "SET_ALL_VISIBILITY", "ALL_RECORDS", "HOSPITAL", 0, "SUCCESS", "LOW", c.ClientIP())
		c.JSON(http.StatusOK, model.Response{
			Code:    200,
			Message: fmt.Sprintf("已成功开启「全部健康档案对 %s 全体医生可见」，策略已固化上链", hosp.Name),
			Data:    auth,
		})
	} else if req.PolicyType == "DOCTOR" {
		if req.AuthTargetID == 0 {
			c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "请指定授权开放的目标医生"})
			return
		}
		docPtr, err := repository.DefaultUserRepository.FindDoctorByID(req.AuthTargetID)
		if err != nil || docPtr == nil {
			c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "指定的医生不存在或非执业医生"})
			return
		}
		doc := *docPtr

		revokeActiveAuthorizationsWithChainSync(patientID, "ALL", 0)

		randBytes := make([]byte, 3)
		_, _ = rand.Read(randBytes)
		authNo := fmt.Sprintf("AUTH%s%s", now.Format("20060102"), hex.EncodeToString(randBytes))

		txID, _, _ := blockchain.DefaultService.CommitAsset("AUTHORIZATION", authNo, map[string]interface{}{
			"auth_no":     authNo,
			"patient_id":  patientID,
			"target_type": "DOCTOR",
			"target_id":   req.AuthTargetID,
			"scope":       "ALL",
			"record_id":   0,
			"status":      "ACTIVE",
			"timestamp":   now.Format(time.RFC3339),
		})

		auth := model.Authorization{
			AuthNo:         authNo,
			PatientID:      patientID,
			AuthTargetType: "DOCTOR",
			AuthTargetID:   req.AuthTargetID,
			ScopeType:      "ALL",
			RecordID:       0,
			StartTime:      startTime,
			EndTime:        endTime,
			Status:         "ACTIVE",
			FabricTxID:     txID,
			CreatedAt:      now,
		}
		_ = repository.DefaultAuthPolicyRepository.CreateAuthorization(&auth)

		docName := doc.RealName
		if !strings.HasSuffix(docName, "医生") && !strings.HasSuffix(docName, "医师") {
			docName += " 医生"
		}
		service.DefaultAuditService.Log(patientID, "SET_ALL_VISIBILITY", "ALL_RECORDS", "DOCTOR", 0, "SUCCESS", "LOW", c.ClientIP())
		c.JSON(http.StatusOK, model.Response{
			Code:    200,
			Message: fmt.Sprintf("已成功开启「全部健康档案对 %s 可见」，策略已固化上链", docName),
			Data:    auth,
		})
	} else if req.PolicyType == "PRIVATE" {
		revokeActiveAuthorizationsWithChainSync(patientID, "ALL", 0)

		service.DefaultAuditService.Log(patientID, "REVOKE_ALL_VISIBILITY", "ALL_RECORDS", "ALL", 0, "SUCCESS", "LOW", c.ClientIP())
		c.JSON(http.StatusOK, model.Response{
			Code:    200,
			Message: "已关闭全局公开，恢复为私密受控模式",
		})
	} else {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "不支持的策略类型"})
	}
}
