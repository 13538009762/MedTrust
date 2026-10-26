package controller

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/pkg/sse"
	"medtrust-backend/repository"
	"medtrust-backend/service"
	"net/http"
	"strconv"
	"time"
)

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

	now := time.Now()
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
		Status:           "PENDING",
		ScopeType:        "SINGLE",
		Days:             7,
		ExpireAt:         now.AddDate(0, 0, 7),
		CreatedAt:        now,
	}
	repository.DB.Create(&accessRec)

	service.DefaultAuditService.Log(doctorID, "ACCESS", "RECORD", record.RecordNo, doctor.HospitalID, func() string {
		if decision.Allowed {
			return "SUCCESS"
		}
		return "INTERCEPTED"
	}(), decision.RiskEvaluation.Level, c.ClientIP())

	if !decision.Allowed {
		c.JSON(http.StatusForbidden, model.Response{
			Code:    403,
			Message: decision.Reason,
			Data:    decision,
		})
		return
	}

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
		ExpireAt:         time.Now().AddDate(0, 0, req.Days),
		CreatedAt:        time.Now(),
	}
	repository.DB.Create(&accessRec)

	service.DefaultAuditService.Log(doctorID, "CONSENT_APPLY", "REQUEST", reqNo, doctor.HospitalID, "PENDING", "LOW", c.ClientIP())

	// 通过 SSE 实时推送通知患者
	docName := doctor.RealName
	if docName == "" {
		docName = doctor.Username
	}
	var hospName = "外院医疗机构"
	var hosp model.Hospital
	if err := repository.DB.First(&hosp, doctor.HospitalID).Error; err == nil && hosp.Name != "" {
		hospName = hosp.Name
	}
	sse.Default.Publish(targetPatientID, sse.NotifyEvent{
		Type:    "access_request",
		Title:   "收到新的跨院调阅申请",
		Message: fmt.Sprintf("%s（%s）向您申请跨院调阅医疗病历，请及时在线知情核准", docName, hospName),
		Payload: map[string]interface{}{
			"request_id":  accessRec.ID,
			"request_no":  reqNo,
			"doctor_name": docName,
			"hospital":    hospName,
			"scope_type":  req.ScopeType,
		},
	})

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

	var records []model.MedicalRecord
	if err := repository.DB.Where("id IN ?", req.RecordIDs).Find(&records).Error; err != nil || len(records) != len(req.RecordIDs) {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "申请病历列表中存在不存在的病历记录"})
		return
	}
	for _, rec := range records {
		if rec.PatientID != req.PatientID {
			c.JSON(http.StatusForbidden, model.Response{
				Code:    403,
				Message: fmt.Sprintf("越权拦截：病历记录(ID:%d)不属于患者(ID:%d)，禁止跨患者批量申请", rec.ID, req.PatientID),
			})
			return
		}
	}

	var createdRequests []model.AccessRequest
	now := time.Now()
	expireAt := now.AddDate(0, 0, req.Days)

	for _, rec := range records {
		randBytes := make([]byte, 3)
		_, _ = rand.Read(randBytes)
		reqNo := fmt.Sprintf("REQ-CONSENT-%s-%s", now.Format("20060102"), hex.EncodeToString(randBytes))

		accessRec := model.AccessRequest{
			RequestNo:        reqNo,
			DoctorID:         doctorID,
			PatientID:        req.PatientID,
			RecordID:         rec.ID,
			SourceHospitalID: doctor.HospitalID,
			TargetHospitalID: rec.HospitalID,
			Purpose:          req.Purpose,
			RiskScore:        15,
			RiskLevel:        "LOW",
			Decision:         "PENDING_CONFIRM",
			Status:           "PENDING",
			ScopeType:        "SINGLE",
			Days:             req.Days,
			ExpireAt:         expireAt,
			CreatedAt:        now,
		}
		repository.DB.Create(&accessRec)
		createdRequests = append(createdRequests, accessRec)
	}

	service.DefaultAuditService.Log(doctorID, "BATCH_CONSENT_APPLY", "REQUEST_BATCH", fmt.Sprintf("PATIENT_%d", req.PatientID), doctor.HospitalID, "PENDING", "LOW", c.ClientIP())

	if len(createdRequests) > 0 {
		docName := doctor.RealName
		if docName == "" {
			docName = doctor.Username
		}
		var hospName = "外院医疗机构"
		var hosp model.Hospital
		if err := repository.DB.First(&hosp, doctor.HospitalID).Error; err == nil && hosp.Name != "" {
			hospName = hosp.Name
		}
		sse.Default.Publish(req.PatientID, sse.NotifyEvent{
			Type:    "access_request",
			Title:   "收到批量病历调阅申请",
			Message: fmt.Sprintf("%s（%s）向您批量申请调阅 %d 份就诊档案，请及时处理", docName, hospName, len(createdRequests)),
			Payload: map[string]interface{}{
				"count":       len(createdRequests),
				"doctor_name": docName,
				"hospital":    hospName,
			},
		})
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: fmt.Sprintf("已成功提交 %d 份病历的知情同意申请，已即时推送患者在线审批", len(createdRequests)),
		Data:    createdRequests,
	})
}

func (ctrl *AccessController) ListPendingRequests(c *gin.Context) {
	patientID := c.GetUint64("user_id")

	var list []model.AccessRequest
	repository.DB.Where("patient_id = ? AND status = 'PENDING'", patientID).Order("id desc").Find(&list)

	if len(list) == 0 {
		c.JSON(http.StatusOK, model.Response{Code: 200, Message: "查询成功", Data: list})
		return
	}

	docIDs := make([]uint64, 0, len(list))
	recIDs := make([]uint64, 0, len(list))
	for _, item := range list {
		docIDs = append(docIDs, item.DoctorID)
		recIDs = append(recIDs, item.RecordID)
	}

	docMap := loadUsersBatch(docIDs)
	recMap := loadRecordsBatch(recIDs)

	hospIDs := make([]uint64, 0, len(docMap)+len(recMap))
	for _, d := range docMap {
		if d.HospitalID > 0 {
			hospIDs = append(hospIDs, d.HospitalID)
		}
	}
	for _, r := range recMap {
		if r.HospitalID > 0 {
			hospIDs = append(hospIDs, r.HospitalID)
		}
	}
	hospMap := loadHospitalsBatch(hospIDs)

	for i := range list {
		if doc, ok := docMap[list[i].DoctorID]; ok {
			list[i].DoctorName = doc.RealName
			list[i].DoctorTitle = doc.Title
			if hosp, ok := hospMap[doc.HospitalID]; ok {
				list[i].DoctorHospitalName = hosp.Name
			}
		}
		if list[i].RecordID > 0 {
			if rec, ok := recMap[list[i].RecordID]; ok {
				list[i].RecordNo = rec.RecordNo
				list[i].RecordDiagnosis = rec.Diagnosis
				list[i].RecordDepartment = rec.DepartmentName
				list[i].RecordEncounterType = rec.EncounterType
				list[i].RecordCreatedAt = rec.CreatedAt.Format("2006-01-02 15:04")
				if rhosp, ok := hospMap[rec.HospitalID]; ok {
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

	service.DefaultAuditService.Log(patientID, "APPROVE_CONSENT", "AUTH", authNo, 0, "SUCCESS", "LOW", c.ClientIP())

	// 通过 SSE 实时通知申请医生
	sse.Default.Publish(req.DoctorID, sse.NotifyEvent{
		Type:    "request_approved",
		Title:   "跨院调阅申请已获患者批准",
		Message: fmt.Sprintf("单号为 %s 的跨院调阅申请已获患者核准同意，权限已激活并存证上链", req.RequestNo),
		Payload: map[string]interface{}{
			"request_id": req.ID,
			"request_no": req.RequestNo,
			"auth_no":    authNo,
			"record_id":  req.RecordID,
		},
	})

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

	service.DefaultAuditService.Log(patientID, "REJECT_CONSENT", "REQUEST", req.RequestNo, 0, "SUCCESS", "LOW", c.ClientIP())

	// 通过 SSE 实时通知申请医生
	sse.Default.Publish(req.DoctorID, sse.NotifyEvent{
		Type:    "request_rejected",
		Title:   "跨院调阅申请已被患者驳回",
		Message: fmt.Sprintf("单号为 %s 的跨院调阅申请已被患者驳回，未能获取调阅权限", req.RequestNo),
		Payload: map[string]interface{}{
			"request_id": req.ID,
			"request_no": req.RequestNo,
		},
	})

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "已驳回该调阅申请"})
}
