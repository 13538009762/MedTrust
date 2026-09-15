package controller

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
	"medtrust-backend/service"
)

type AccessController struct{}

var DefaultAccessController = &AccessController{}

type AccessRequestDTO struct {
	PatientID uint64 `json:"patient_id" binding:"required"`
	RecordID  uint64 `json:"record_id" binding:"required"`
	Purpose   string `json:"purpose"`
}

// RequestAccess 医生发起常规访问申请
func (ctrl *AccessController) RequestAccess(c *gin.Context) {
	doctorID := c.GetUint64("user_id")
	var req AccessRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误"})
		return
	}

	decision, err := service.DefaultAccessEngine.EvaluateAccess(doctorID, req.PatientID, req.RecordID, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: err.Error()})
		return
	}

	// 记录访问申请与系统决策
	var doctor model.User
	_ = repository.DB.First(&doctor, doctorID)
	var record model.MedicalRecord
	_ = repository.DB.First(&record, req.RecordID)

	randBytes := make([]byte, 3)
	_, _ = rand.Read(randBytes)
	reqNo := fmt.Sprintf("REQ%s%s", time.Now().Format("20060102"), hex.EncodeToString(randBytes))

	accessRec := model.AccessRequest{
		RequestNo:        reqNo,
		DoctorID:         doctorID,
		PatientID:        req.PatientID,
		RecordID:         req.RecordID,
		SourceHospitalID: doctor.HospitalID,
		TargetHospitalID: record.HospitalID,
		Purpose:          req.Purpose,
		RiskScore:        decision.RiskEvaluation.TotalScore,
		RiskLevel:        decision.RiskEvaluation.Level,
		Decision:         decision.Decision,
		CreatedAt:        time.Now(),
	}
	repository.DB.Create(&accessRec)

	service.DefaultAuditService.Log(doctorID, "ACCESS", "RECORD", record.RecordNo, doctor.HospitalID, func() string {
		if decision.Allowed {
			return "SUCCESS"
		}
		return "INTERCEPTED"
	}(), decision.RiskEvaluation.Level, "127.0.0.1")

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "权限评估完成",
		Data:    decision,
	})
}

type ApplyConsentDTO struct {
	RecordID  uint64 `json:"record_id"`
	PatientID uint64 `json:"patient_id"`
	Purpose   string `json:"purpose"`
	ScopeType string `json:"scope_type"` // SINGLE, ALL
	Days      int    `json:"days"`
}

// ApplyConsent 医生在常规诊疗场景下发起知情授权申请（通知患者在线核准）
func (ctrl *AccessController) ApplyConsent(c *gin.Context) {
	doctorID := c.GetUint64("user_id")
	var req ApplyConsentDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误：缺少必要参数"})
		return
	}

	var targetPatientID uint64
	var targetHospitalID uint64
	if req.RecordID > 0 {
		var rec model.MedicalRecord
		if err := repository.DB.First(&rec, req.RecordID).Error; err != nil {
			c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "病历记录不存在"})
			return
		}
		targetPatientID = rec.PatientID
		targetHospitalID = rec.HospitalID
	} else if req.PatientID > 0 {
		var pat model.User
		if err := repository.DB.First(&pat, req.PatientID).Error; err != nil {
			c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "患者不存在"})
			return
		}
		targetPatientID = pat.ID
		req.ScopeType = "ALL"
	} else {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "请指定病历ID或目标患者ID"})
		return
	}

	var doctor model.User
	_ = repository.DB.First(&doctor, doctorID)

	if req.Purpose == "" {
		req.Purpose = "门诊专科联合随访评估与既往慢病复核，需调阅外院历史健康档案"
	}
	if req.ScopeType == "" {
		req.ScopeType = "SINGLE"
	}
	if req.Days <= 0 {
		req.Days = 7
	}

	randBytes := make([]byte, 3)
	_, _ = rand.Read(randBytes)
	reqNo := fmt.Sprintf("REQ-CONSENT-%s-%s", time.Now().Format("20060102"), hex.EncodeToString(randBytes))

	accessRec := model.AccessRequest{
		RequestNo:        reqNo,
		DoctorID:         doctorID,
		PatientID:        targetPatientID,
		RecordID:         req.RecordID,
		SourceHospitalID: doctor.HospitalID,
		TargetHospitalID: targetHospitalID,
		Purpose:          req.Purpose,
		RiskScore:        15,
		RiskLevel:        "LOW",
		Decision:         "PENDING_CONFIRM",
		Status:           "PENDING",
		ScopeType:        req.ScopeType,
		Days:             req.Days,
		CreatedAt:        time.Now(),
	}
	repository.DB.Create(&accessRec)

	service.DefaultAuditService.Log(doctorID, "CONSENT_APPLY", "REQUEST", reqNo, doctor.HospitalID, "PENDING", "LOW", "127.0.0.1")

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "知情同意申请已成功提交并通知患者，等待患者在线核准",
		Data:    accessRec,
	})
}

type BatchApplyConsentDTO struct {
	PatientID uint64   `json:"patient_id" binding:"required"`
	RecordIDs []uint64 `json:"record_ids" binding:"required"`
	Purpose   string   `json:"purpose"`
	Days      int      `json:"days"`
}

// BatchApplyConsent 医生分批/勾选申请多份病历的调阅知情同意
func (ctrl *AccessController) BatchApplyConsent(c *gin.Context) {
	doctorID := c.GetUint64("user_id")
	var req BatchApplyConsentDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误：缺少患者ID或待申请病历列表"})
		return
	}

	if len(req.RecordIDs) == 0 {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "请至少勾选一份病历申请权限"})
		return
	}

	var doctor model.User
	_ = repository.DB.First(&doctor, doctorID)

	if req.Purpose == "" {
		req.Purpose = "急救临床协同诊疗，批量申请调阅患者既往受控病历档案"
	}
	if req.Days <= 0 {
		req.Days = 7
	}

	var createdRequests []model.AccessRequest
	now := time.Now()

	for _, recID := range req.RecordIDs {
		var rec model.MedicalRecord
		if err := repository.DB.First(&rec, recID).Error; err != nil {
			continue
		}

		randBytes := make([]byte, 3)
		_, _ = rand.Read(randBytes)
		reqNo := fmt.Sprintf("REQ-CONSENT-%s-%s", now.Format("20060102"), hex.EncodeToString(randBytes))

		accessRec := model.AccessRequest{
			RequestNo:        reqNo,
			DoctorID:         doctorID,
			PatientID:        req.PatientID,
			RecordID:         recID,
			SourceHospitalID: doctor.HospitalID,
			TargetHospitalID: rec.HospitalID,
			Purpose:          req.Purpose,
			RiskScore:        15,
			RiskLevel:        "LOW",
			Decision:         "PENDING_CONFIRM",
			Status:           "PENDING",
			ScopeType:        "SINGLE",
			Days:             req.Days,
			CreatedAt:        now,
		}
		repository.DB.Create(&accessRec)
		createdRequests = append(createdRequests, accessRec)
	}

	service.DefaultAuditService.Log(doctorID, "BATCH_CONSENT_APPLY", "REQUEST_BATCH", fmt.Sprintf("PATIENT_%d", req.PatientID), doctor.HospitalID, "PENDING", "LOW", "127.0.0.1")

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: fmt.Sprintf("已成功提交 %d 份病历的知情同意申请，已即时推送患者在线审批", len(createdRequests)),
		Data:    createdRequests,
	})
}

type UnlockPatientByKeyDTO struct {
	PatientID  uint64 `json:"patient_id" binding:"required"`
	MedicalKey string `json:"medical_key" binding:"required"`
	Purpose    string `json:"purpose"`
	Days       int    `json:"days"`
}

// UnlockPatientByKey 医生在现场输入患者专属密钥，核验通过后即时解密放行该患者全套病历并固化上链
func (ctrl *AccessController) UnlockPatientByKey(c *gin.Context) {
	doctorID := c.GetUint64("user_id")
	var req UnlockPatientByKeyDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误：请输入患者ID与专属授权密钥"})
		return
	}

	var patient model.User
	if err := repository.DB.First(&patient, req.PatientID).Error; err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "目标患者档案不存在"})
		return
	}

	var doctor model.User
	_ = repository.DB.First(&doctor, doctorID)

	expectedKey := patient.MedicalKey
	if expectedKey == "" {
		expectedKey = "123456"
	}

	inputKey := strings.TrimSpace(req.MedicalKey)
	if inputKey != expectedKey {
		service.DefaultAuditService.Log(doctorID, "KEY_UNLOCK_FAILED", "PATIENT", patient.UserNo, doctor.HospitalID, "INTERCEPTED", "HIGH", "127.0.0.1")
		c.JSON(http.StatusForbidden, model.Response{
			Code:    403,
			Message: "患者授权密钥校验失败，密码不正确！请由患者在个人中心核实或重新设置调阅密钥（初始默认: 123456）",
		})
		return
	}

	// 密钥核验通过：生成对该患者全部档案全局生效的授权凭证
	days := req.Days
	if days <= 0 {
		days = 7
	}
	startTime := time.Now()
	endTime := startTime.Add(time.Duration(days) * 24 * time.Hour)

	randBytes := make([]byte, 3)
	_, _ = rand.Read(randBytes)
	authNo := fmt.Sprintf("AUTH-KEY-ALL-%s-%s", startTime.Format("20060102"), hex.EncodeToString(randBytes))

	txID, _, _ := blockchain.DefaultService.CommitAsset("AUTHORIZATION_KEY", authNo, map[string]interface{}{
		"auth_no":     authNo,
		"patient_id":  patient.ID,
		"target_type": "DOCTOR",
		"target_id":   doctorID,
		"scope":       "ALL",
		"record_id":   0,
		"status":      "ACTIVE",
		"timestamp":   startTime.Format(time.RFC3339),
	})

	auth := model.Authorization{
		AuthNo:         authNo,
		PatientID:      patient.ID,
		AuthTargetType: "DOCTOR",
		AuthTargetID:   doctorID,
		ScopeType:      "ALL",
		RecordID:       0,
		StartTime:      startTime,
		EndTime:        endTime,
		Status:         "ACTIVE",
		FabricTxID:     txID,
		CreatedAt:      startTime,
	}
	repository.DB.Create(&auth)

	service.DefaultAuditService.Log(doctorID, "KEY_UNLOCK_SUCCESS", "PATIENT", patient.UserNo, doctor.HospitalID, "SUCCESS", "LOW", "127.0.0.1")

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: fmt.Sprintf("患者现场密钥核验通过！已即时解密放行「%s」名下全部历史就诊与检验影像档案，授权策略已固化上链", patient.RealName),
		Data: gin.H{
			"auth": auth,
		},
	})
}

type EmergencyBatchAccessDTO struct {
	PatientID       uint64   `json:"patient_id" binding:"required"`
	RecordIDs       []uint64 `json:"record_ids"` // 可选，若为空则针对该患者所有病历
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
		)
		if err == nil && ev != nil {
			events = append(events, *ev)
		}
	}

	service.DefaultAuditService.Log(doctorID, "EMERGENCY_BATCH_ACCESS", "PATIENT", patient.UserNo, doctor.HospitalID, "SUCCESS", "HIGH", "127.0.0.1")

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
func (ctrl *AccessController) ListPendingRequests(c *gin.Context) {
	patientID := c.GetUint64("user_id")

	var list []model.AccessRequest
	repository.DB.Where("patient_id = ? AND status = 'PENDING'", patientID).Order("id desc").Find(&list)

	for i := range list {
		var doc model.User
		if err := repository.DB.First(&doc, list[i].DoctorID).Error; err == nil {
			list[i].DoctorName = doc.RealName
			list[i].DoctorTitle = doc.Title
			var hosp model.Hospital
			if err := repository.DB.First(&hosp, doc.HospitalID).Error; err == nil {
				list[i].DoctorHospitalName = hosp.Name
			}
		}
		if list[i].RecordID > 0 {
			var rec model.MedicalRecord
			if err := repository.DB.First(&rec, list[i].RecordID).Error; err == nil {
				list[i].RecordNo = rec.RecordNo
				list[i].RecordDiagnosis = rec.Diagnosis
				list[i].RecordDepartment = rec.DepartmentName
				list[i].RecordEncounterType = rec.EncounterType
				list[i].RecordCreatedAt = rec.CreatedAt.Format("2006-01-02 15:04")
				var rhosp model.Hospital
				if err := repository.DB.First(&rhosp, rec.HospitalID).Error; err == nil {
					list[i].RecordHospitalName = rhosp.Name
				}
			}
		}
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "查询成功", Data: list})
}

// ApproveRequest 患者核准同意调阅申请，释放访问权限并自动建立上链授权
func (ctrl *AccessController) ApproveRequest(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var req model.AccessRequest
	if err := repository.DB.Where("id = ? AND patient_id = ?", id, patientID).First(&req).Error; err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "未找到该申请记录"})
		return
	}

	if req.Status != "PENDING" {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "该申请已处理过"})
		return
	}

	days := req.Days
	if days <= 0 {
		days = 7
	}
	startTime := time.Now()
	endTime := startTime.Add(time.Duration(days) * 24 * time.Hour)

	randBytes := make([]byte, 3)
	_, _ = rand.Read(randBytes)
	authNo := fmt.Sprintf("AUTH%s%s", time.Now().Format("20060102"), hex.EncodeToString(randBytes))

	txID, _, _ := blockchain.DefaultService.CommitAsset("AUTHORIZATION", authNo, map[string]interface{}{
		"auth_no":     authNo,
		"patient_id":  patientID,
		"target_type": "DOCTOR",
		"target_id":   req.DoctorID,
		"scope":       req.ScopeType,
		"record_id":   req.RecordID,
		"status":      "ACTIVE",
		"timestamp":   time.Now().Format(time.RFC3339),
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
		CreatedAt:      time.Now(),
	}
	repository.DB.Create(&auth)

	// 更新申请单状态
	req.Status = "APPROVED"
	req.Decision = "ALLOWED"
	repository.DB.Save(&req)

	service.DefaultAuditService.Log(patientID, "APPROVE_CONSENT", "AUTH", authNo, 0, "SUCCESS", "LOW", "127.0.0.1")

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "已核准同意授权！权限已向经治医生释放并完成区块链固化存证",
		Data:    auth,
	})
}

// RejectRequest 患者驳回调阅申请
func (ctrl *AccessController) RejectRequest(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var req model.AccessRequest
	if err := repository.DB.Where("id = ? AND patient_id = ?", id, patientID).First(&req).Error; err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "未找到该申请记录"})
		return
	}

	req.Status = "REJECTED"
	req.Decision = "REJECTED"
	repository.DB.Save(&req)

	service.DefaultAuditService.Log(patientID, "REJECT_CONSENT", "REQUEST", req.RequestNo, 0, "SUCCESS", "LOW", "127.0.0.1")

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "已驳回该调阅申请"})
}

type BreakGlassRequest struct {
	RecordID        uint64 `json:"record_id" binding:"required"`
	EmergencyReason string `json:"emergency_reason" binding:"required"` // RESCUE, COMA, CRITICAL, OTHER
	Description     string `json:"description" binding:"required"`
	DoctorConfirmed bool   `json:"doctor_confirmed" binding:"required"`
}

// BreakGlass 医生发起紧急抢救调阅
func (ctrl *AccessController) BreakGlass(c *gin.Context) {
	doctorID := c.GetUint64("user_id")
	var req BreakGlassRequest
	if err := c.ShouldBindJSON(&req); err != nil || !req.DoctorConfirmed {
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

	// 生成事件单并上链
	event, err := service.DefaultEmergencyService.SubmitEmergencyAccess(doctorID, req.RecordID, req.EmergencyReason, req.Description, req.DoctorConfirmed)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: err.Error()})
		return
	}

	// 解密放行病历数据
	fullRecord, _ := service.DefaultMedicalService.GetRecordByID(req.RecordID)

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "紧急访问申请已核准放行，生成待审核事件单并全量存证",
		Data: gin.H{
			"event":  event,
			"record": fullRecord,
		},
	})
}

type PatientFeedbackRequest struct {
	EventNo  string `json:"event_no" binding:"required"`
	Feedback string `json:"feedback" binding:"required"` // CONFIRMED, OBJECTED
	Comment  string `json:"comment"`
}

func (ctrl *AccessController) PatientFeedback(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	var req PatientFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误"})
		return
	}

	if err := service.DefaultEmergencyService.SubmitPatientFeedback(patientID, req.EventNo, req.Feedback, req.Comment); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "反馈已成功提交并归档"})
}

type CreateAuthRequest struct {
	AuthTargetType string    `json:"auth_target_type" binding:"required"` // DOCTOR, HOSPITAL, ALL_DOCTORS
	AuthTargetID   uint64    `json:"auth_target_id"`                      // 当 ALL_DOCTORS 时可为 0
	ScopeType      string    `json:"scope_type" binding:"required"`       // ALL, SINGLE
	RecordID       uint64    `json:"record_id"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	Days           int       `json:"days"`
}

func (ctrl *AccessController) CreateAuthorization(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	var req CreateAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误"})
		return
	}

	if req.AuthTargetType != "ALL_DOCTORS" && req.AuthTargetID == 0 {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "请选择具体的授权目标医生或医疗机构"})
		return
	}
	if req.ScopeType == "SINGLE" && req.RecordID == 0 {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "指定单份病历授权时必须选择具体病历"})
		return
	}

	if req.Days > 0 {
		req.StartTime = time.Now()
		req.EndTime = req.StartTime.Add(time.Duration(req.Days) * 24 * time.Hour)
	} else {
		if req.StartTime.IsZero() {
			req.StartTime = time.Now()
		}
		if req.EndTime.IsZero() {
			req.EndTime = time.Now().Add(365 * 24 * time.Hour)
		}
	}

	randBytes := make([]byte, 3)
	_, _ = rand.Read(randBytes)
	authNo := fmt.Sprintf("AUTH%s%s", time.Now().Format("20060102"), hex.EncodeToString(randBytes))

	txID, _, _ := blockchain.DefaultService.CommitAsset("AUTHORIZATION", authNo, map[string]interface{}{
		"auth_no":     authNo,
		"patient_id":  patientID,
		"target_type": req.AuthTargetType,
		"target_id":   req.AuthTargetID,
		"scope":       req.ScopeType,
		"record_id":   req.RecordID,
		"status":      "ACTIVE",
		"timestamp":   time.Now().Format(time.RFC3339),
	})

	auth := model.Authorization{
		AuthNo:         authNo,
		PatientID:      patientID,
		AuthTargetType: req.AuthTargetType,
		AuthTargetID:   req.AuthTargetID,
		ScopeType:      req.ScopeType,
		RecordID:       req.RecordID,
		StartTime:      req.StartTime,
		EndTime:        req.EndTime,
		Status:         "ACTIVE",
		FabricTxID:     txID,
		CreatedAt:      time.Now(),
	}
	repository.DB.Create(&auth)

	service.DefaultAuditService.Log(patientID, "AUTHORIZE", "AUTH", authNo, 0, "SUCCESS", "LOW", "127.0.0.1")
	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "授权策略建立并上链成功", Data: auth})
}

func (ctrl *AccessController) ListAuthorizations(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	role := c.GetString("role")

	var list []model.Authorization
	query := repository.DB.Model(&model.Authorization{})
	if role == "patient" {
		query = query.Where("patient_id = ?", patientID)
	}
	query.Order("id desc").Find(&list)

	for i := range list {
		if list[i].AuthTargetType == "ALL_DOCTORS" {
			list[i].TargetName = "全体执业医生 (全联盟公开)"
		} else if list[i].AuthTargetType == "DOCTOR" {
			var doc model.User
			if err := repository.DB.First(&doc, list[i].AuthTargetID).Error; err == nil {
				hospSuffix := ""
				if doc.HospitalID > 0 {
					var hosp model.Hospital
					if err := repository.DB.First(&hosp, doc.HospitalID).Error; err == nil {
						hospSuffix = " (" + hosp.Name + ")"
					}
				}
				list[i].TargetName = doc.RealName + " 医生" + hospSuffix
			}
		} else if list[i].AuthTargetType == "HOSPITAL" {
			var hosp model.Hospital
			if err := repository.DB.First(&hosp, list[i].AuthTargetID).Error; err == nil {
				list[i].TargetName = hosp.Name
			}
		}

		if list[i].RecordID > 0 {
			var rec model.MedicalRecord
			if err := repository.DB.First(&rec, list[i].RecordID).Error; err == nil {
				list[i].RecordNo = rec.RecordNo
				list[i].RecordDiagnosis = rec.Diagnosis
				list[i].RecordDepartment = rec.DepartmentName
				var hosp model.Hospital
				if err := repository.DB.First(&hosp, rec.HospitalID).Error; err == nil {
					list[i].RecordHospitalName = hosp.Name
				}
			}
		}
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "查询成功", Data: list})
}

func (ctrl *AccessController) RevokeAuthorization(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var auth model.Authorization
	if err := repository.DB.Where("id = ? AND patient_id = ?", id, patientID).First(&auth).Error; err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "未找到该授权条目"})
		return
	}

	auth.Status = "REVOKED"
	repository.DB.Save(&auth)

	_, _, _ = blockchain.DefaultService.CommitAsset("REVOKE_AUTH", auth.AuthNo, map[string]interface{}{
		"auth_no":   auth.AuthNo,
		"status":    "REVOKED",
		"timestamp": time.Now().Format(time.RFC3339),
	})

	service.DefaultAuditService.Log(patientID, "REVOKE", "AUTH", auth.AuthNo, 0, "SUCCESS", "LOW", "127.0.0.1")
	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "授权已撤回上链"})
}

type UnlockByKeyDTO struct {
	RecordID   uint64 `json:"record_id" binding:"required"`
	MedicalKey string `json:"medical_key" binding:"required"`
	Purpose    string `json:"purpose"`
	Days       int    `json:"days"`
}

// UnlockByKey 医生在就诊现场输入患者病历专属密钥，核验通过后即时解密放行并存证上链
func (ctrl *AccessController) UnlockByKey(c *gin.Context) {
	doctorID := c.GetUint64("user_id")
	var req UnlockByKeyDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误：请输入病历ID与患者密钥"})
		return
	}

	var rec model.MedicalRecord
	if err := repository.DB.First(&rec, req.RecordID).Error; err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "目标病历档案不存在"})
		return
	}

	var patient model.User
	if err := repository.DB.First(&patient, rec.PatientID).Error; err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "患者用户档案不存在"})
		return
	}

	var doctor model.User
	_ = repository.DB.First(&doctor, doctorID)

	expectedKey := patient.MedicalKey
	if expectedKey == "" {
		expectedKey = "123456"
	}

	inputKey := strings.TrimSpace(req.MedicalKey)
	if inputKey != expectedKey {
		service.DefaultAuditService.Log(doctorID, "KEY_UNLOCK_FAILED", "RECORD", rec.RecordNo, doctor.HospitalID, "INTERCEPTED", "HIGH", "127.0.0.1")
		c.JSON(http.StatusForbidden, model.Response{
			Code:    403,
			Message: "患者授权密钥校验失败，密码不正确！请由患者在个人中心核实或重新设置调阅密钥（初始默认: 123456）",
		})
		return
	}

	// 密钥核验通过：生成即时生效的授权凭证
	days := req.Days
	if days <= 0 {
		days = 7
	}
	startTime := time.Now()
	endTime := startTime.Add(time.Duration(days) * 24 * time.Hour)

	randBytes := make([]byte, 3)
	_, _ = rand.Read(randBytes)
	authNo := fmt.Sprintf("AUTH-KEY-%s-%s", time.Now().Format("20060102"), hex.EncodeToString(randBytes))

	txID, _, _ := blockchain.DefaultService.CommitAsset("AUTHORIZATION_KEY", authNo, map[string]interface{}{
		"auth_no":     authNo,
		"patient_id":  patient.ID,
		"target_type": "DOCTOR",
		"target_id":   doctorID,
		"scope":       "SINGLE",
		"record_id":   req.RecordID,
		"status":      "ACTIVE",
		"auth_method": "PATIENT_KEY_VERIFIED",
		"timestamp":   time.Now().Format(time.RFC3339),
	})

	auth := model.Authorization{
		AuthNo:         authNo,
		PatientID:      patient.ID,
		AuthTargetType: "DOCTOR",
		AuthTargetID:   doctorID,
		ScopeType:      "SINGLE",
		RecordID:       req.RecordID,
		StartTime:      startTime,
		EndTime:        endTime,
		Status:         "ACTIVE",
		FabricTxID:     txID,
		CreatedAt:      time.Now(),
	}
	repository.DB.Create(&auth)

	// 若有关于此病历与医生的待处理申请单，顺便标记为已核准
	repository.DB.Model(&model.AccessRequest{}).
		Where("doctor_id = ? AND record_id = ? AND status = 'PENDING'", doctorID, req.RecordID).
		Updates(map[string]interface{}{
			"status":   "APPROVED",
			"decision": "ALLOWED",
		})

	service.DefaultAuditService.Log(doctorID, "KEY_UNLOCK_SUCCESS", "RECORD", rec.RecordNo, doctor.HospitalID, "SUCCESS", "LOW", "127.0.0.1")

	// 加载完整解密病历
	fullRecord, err := service.DefaultMedicalService.GetRecordByID(req.RecordID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "解密病历失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "患者现场密钥核验通过！已即时解密放行该份跨院病历并完成区块链存证",
		Data: gin.H{
			"record": fullRecord,
			"auth":   auth,
		},
	})
}

type SetRecordAccessPolicyDTO struct {
	PolicyType   string `json:"policy_type" binding:"required"` // ALL_DOCTORS, HOSPITAL, DOCTOR, PRIVATE
	AuthTargetID uint64 `json:"auth_target_id"`                // 医院ID 或 医生用户ID
	Days         int    `json:"days"`                          // 默认 365 天
}

// SetRecordAccessPolicy 患者快捷设置单份病历的访问权限（如：让所有医生都可见 / 指定医院全体医生 / 指定医生 / 恢复私密）
func (ctrl *AccessController) SetRecordAccessPolicy(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	recordID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var rec model.MedicalRecord
	if err := repository.DB.Where("id = ? AND patient_id = ?", recordID, patientID).First(&rec).Error; err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "未找到该就诊病历记录"})
		return
	}

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
		repository.DB.Model(&model.Authorization{}).
			Where("patient_id = ? AND scope_type = 'SINGLE' AND record_id = ? AND status = 'ACTIVE'", patientID, recordID).
			Update("status", "REVOKED")

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
		repository.DB.Create(&auth)

		service.DefaultAuditService.Log(patientID, "SET_RECORD_VISIBILITY", "RECORD", rec.RecordNo, 0, "SUCCESS", "LOW", "127.0.0.1")
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
		var hosp model.Hospital
		if err := repository.DB.First(&hosp, req.AuthTargetID).Error; err != nil {
			c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "指定的医疗机构不存在"})
			return
		}

		// 先撤销旧的单份活跃授权
		repository.DB.Model(&model.Authorization{}).
			Where("patient_id = ? AND scope_type = 'SINGLE' AND record_id = ? AND status = 'ACTIVE'", patientID, recordID).
			Update("status", "REVOKED")

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
		repository.DB.Create(&auth)

		service.DefaultAuditService.Log(patientID, "SET_RECORD_VISIBILITY", "RECORD", rec.RecordNo, 0, "SUCCESS", "LOW", "127.0.0.1")
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
		var doc model.User
		if err := repository.DB.Where("id = ? AND role = 'doctor'", req.AuthTargetID).First(&doc).Error; err != nil {
			c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "指定的医生不存在或非执业医生"})
			return
		}

		// 先撤销旧的单份活跃授权
		repository.DB.Model(&model.Authorization{}).
			Where("patient_id = ? AND scope_type = 'SINGLE' AND record_id = ? AND status = 'ACTIVE'", patientID, recordID).
			Update("status", "REVOKED")

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
		repository.DB.Create(&auth)

		docName := doc.RealName
		if !strings.HasSuffix(docName, "医生") && !strings.HasSuffix(docName, "医师") {
			docName += " 医生"
		}
		service.DefaultAuditService.Log(patientID, "SET_RECORD_VISIBILITY", "RECORD", rec.RecordNo, 0, "SUCCESS", "LOW", "127.0.0.1")
		c.JSON(http.StatusOK, model.Response{
			Code:    200,
			Message: fmt.Sprintf("已成功设置该病历对「%s」专属授权调阅，策略已固化上链", docName),
			Data:    auth,
		})
	} else if req.PolicyType == "PRIVATE" {
		// 撤销对该病历的所有单份活跃授权
		repository.DB.Model(&model.Authorization{}).
			Where("patient_id = ? AND scope_type = 'SINGLE' AND record_id = ? AND status = 'ACTIVE'", patientID, recordID).
			Update("status", "REVOKED")

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
		repository.DB.Create(&auth)

		service.DefaultAuditService.Log(patientID, "REVOKE_RECORD_VISIBILITY", "RECORD", rec.RecordNo, 0, "SUCCESS", "LOW", "127.0.0.1")
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
		repository.DB.Model(&model.Authorization{}).
			Where("patient_id = ? AND scope_type = 'ALL' AND status = 'ACTIVE'", patientID).
			Update("status", "REVOKED")

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
		repository.DB.Create(&auth)

		service.DefaultAuditService.Log(patientID, "SET_ALL_VISIBILITY", "ALL_RECORDS", "ALL", 0, "SUCCESS", "LOW", "127.0.0.1")
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
		var hosp model.Hospital
		if err := repository.DB.First(&hosp, req.AuthTargetID).Error; err != nil {
			c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "指定的医疗机构不存在"})
			return
		}

		repository.DB.Model(&model.Authorization{}).
			Where("patient_id = ? AND scope_type = 'ALL' AND status = 'ACTIVE'", patientID).
			Update("status", "REVOKED")

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
		repository.DB.Create(&auth)

		service.DefaultAuditService.Log(patientID, "SET_ALL_VISIBILITY", "ALL_RECORDS", "HOSPITAL", 0, "SUCCESS", "LOW", "127.0.0.1")
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
		var doc model.User
		if err := repository.DB.Where("id = ? AND role = 'doctor'", req.AuthTargetID).First(&doc).Error; err != nil {
			c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "指定的医生不存在或非执业医生"})
			return
		}

		repository.DB.Model(&model.Authorization{}).
			Where("patient_id = ? AND scope_type = 'ALL' AND status = 'ACTIVE'", patientID).
			Update("status", "REVOKED")

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
		repository.DB.Create(&auth)

		docName := doc.RealName
		if !strings.HasSuffix(docName, "医生") && !strings.HasSuffix(docName, "医师") {
			docName += " 医生"
		}
		service.DefaultAuditService.Log(patientID, "SET_ALL_VISIBILITY", "ALL_RECORDS", "DOCTOR", 0, "SUCCESS", "LOW", "127.0.0.1")
		c.JSON(http.StatusOK, model.Response{
			Code:    200,
			Message: fmt.Sprintf("已成功开启「全部健康档案对 %s 可见」，策略已固化上链", docName),
			Data:    auth,
		})
	} else if req.PolicyType == "PRIVATE" {
		repository.DB.Model(&model.Authorization{}).
			Where("patient_id = ? AND scope_type = 'ALL' AND status = 'ACTIVE'", patientID).
			Update("status", "REVOKED")

		service.DefaultAuditService.Log(patientID, "REVOKE_ALL_VISIBILITY", "ALL_RECORDS", "ALL", 0, "SUCCESS", "LOW", "127.0.0.1")
		c.JSON(http.StatusOK, model.Response{
			Code:    200,
			Message: "已关闭全局公开，恢复为私密受控模式",
		})
	} else {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "不支持的策略类型"})
	}
}

