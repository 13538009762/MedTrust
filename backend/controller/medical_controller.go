package controller

import (
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"medtrust-backend/model"
	"medtrust-backend/pkg/pdf"
	"medtrust-backend/pkg/security"
	"medtrust-backend/repository"
	"medtrust-backend/service"
)

type MedicalController struct{}

var DefaultMedicalController = &MedicalController{}

func (ctrl *MedicalController) Upload(c *gin.Context) {
	doctorID := c.GetUint64("user_id")
	patientIDStr := c.PostForm("patient_id")
	patientID, _ := strconv.ParseUint(patientIDStr, 10, 64)
	if patientID == 0 {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "缺少必要参数：患者ID"})
		return
	}

	dataType := c.DefaultPostForm("data_type", "EMR")
	encounterType := c.DefaultPostForm("encounter_type", "OUTPATIENT")
	departmentName := c.DefaultPostForm("department_name", "综合门诊")
	status := c.DefaultPostForm("status", "COMPLETED")
	onsetTime := c.PostForm("onset_time")
	duration := c.PostForm("duration")
	symptoms := c.PostForm("symptoms")
	chiefComplaint := c.PostForm("chief_complaint")
	presentIllness := c.PostForm("present_illness")
	initialDiagnosis := c.PostForm("initial_diagnosis")
	diagnosticBasis := c.PostForm("diagnostic_basis")
	needExam := c.PostForm("need_exam") == "true" || c.PostForm("need_exam") == "1"
	examItems := c.PostForm("exam_items")
	examReason := c.PostForm("exam_reason")
	examResult := c.PostForm("exam_result")
	examDoctor := c.PostForm("exam_doctor")
	examTime := c.PostForm("exam_time")
	etiology := c.PostForm("etiology")
	treatmentPlan := c.PostForm("treatment_plan")
	vitalSigns := c.PostForm("vital_signs")
	diagnosis := c.PostForm("diagnosis")
	idempotencyKey := c.DefaultPostForm("idempotency_key", c.GetHeader("X-Idempotency-Key"))

	var fileBytes []byte
	fileName := ""
	fileType := "pdf"

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 50<<20)
	file, header, err := c.Request.FormFile("file")
	if err == nil && file != nil {
		defer file.Close()
		fb, readErr := io.ReadAll(file)
		if readErr != nil {
			c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "读取上传附件失败: " + readErr.Error()})
			return
		}
		if len(fb) > 0 {
			if valErr := security.ValidateMedicalFile(header.Filename, fb); valErr != nil {
				c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "文件安全校验未通过: " + valErr.Error()})
				return
			}
			fileBytes = fb
			fileName = header.Filename
			if ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), "."); ext != "" {
				fileType = ext
			}
		}
	}

	rec, err := service.DefaultMedicalService.UploadRecord(service.UploadRecordParams{
		DoctorID:         doctorID,
		PatientID:        patientID,
		DataType:         dataType,
		OnsetTime:        onsetTime,
		Duration:         duration,
		Symptoms:         symptoms,
		Etiology:         etiology,
		TreatmentPlan:    treatmentPlan,
		VitalSigns:       vitalSigns,
		Diagnosis:        diagnosis,
		FileName:         fileName,
		FileType:         fileType,
		FileData:         fileBytes,
		EncounterType:    encounterType,
		DepartmentName:   departmentName,
		Status:           status,
		ChiefComplaint:   chiefComplaint,
		PresentIllness:   presentIllness,
		InitialDiagnosis: initialDiagnosis,
		DiagnosticBasis:  diagnosticBasis,
		NeedExam:         needExam,
		ExamItems:        examItems,
		ExamReason:       examReason,
		ExamResult:       examResult,
		ExamDoctor:       examDoctor,
		ExamTime:         examTime,
		ClientIP:         c.ClientIP(),
		IdempotencyKey:   idempotencyKey,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "就诊档案已完成加密存储、规则病历整理并成功锚定至联盟链存证",
		Data:    rec,
	})
}

func (ctrl *MedicalController) List(c *gin.Context) {
	patientID, _ := strconv.ParseUint(c.Query("patient_id"), 10, 64)
	doctorID, _ := strconv.ParseUint(c.Query("doctor_id"), 10, 64)
	hospitalID, _ := strconv.ParseUint(c.Query("hospital_id"), 10, 64)
	excludeHospitalID, _ := strconv.ParseUint(c.Query("exclude_hospital_id"), 10, 64)
	onlyCross := c.Query("only_cross") == "true"
	onlyCrossAccessed := c.Query("only_cross_accessed") == "true"
	currentUserID := c.GetUint64("user_id")

	if onlyCross && excludeHospitalID == 0 {
		if doctor, err := repository.DefaultUserRepository.FindByID(currentUserID); err == nil {
			excludeHospitalID = doctor.HospitalID
		}
	}
	keyword := c.Query("keyword")
	dataType := c.Query("data_type")
	// 分页：默认第 1 页、每页 MaxRecordPageSize 条（与上限一致，现有前端不传参时行为不变）
	page := 1
	if v, err := strconv.Atoi(c.Query("page")); err == nil && v > 0 {
		page = v
	}
	pageSize := service.MaxRecordPageSize
	if v, err := strconv.Atoi(c.Query("page_size")); err == nil && v > 0 {
		pageSize = v
	}

	// 管理员严禁访问患者病历列表（管理权限与临床数据隔离）；患者仅限查询本人
	role := c.GetString("role")
	if role == "admin" {
		c.JSON(http.StatusForbidden, model.Response{
			Code:    403,
			Message: "权限拦截 (403 Forbidden): 系统管理权限与临床数据分离，管理员严禁查阅患者病历列表",
		})
		return
	}
	if role == "patient" {
		patientID = currentUserID
	}

	list, err := service.DefaultMedicalService.GetRecords(patientID, doctorID, hospitalID, excludeHospitalID, currentUserID, onlyCrossAccessed, keyword, dataType, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: err.Error()})
		return
	}

	if currentUserID > 0 {
		targetStr := "ALL"
		if patientID > 0 {
			targetStr = fmt.Sprintf("PAT_%d", patientID)
		}
		service.DefaultAuditService.LogDetailed(service.AuditEntry{
			UserID:        currentUserID,
			OperationType: "LIST_RECORDS",
			TargetType:    "RECORD",
			TargetID:      targetStr,
			HospitalID:    c.GetUint64("hospital_id"),
			Result:        "ALLOWED",
			RiskLevel:     "LOW",
			Source:        "WEB",
			IPAddress:     c.ClientIP(),
			UserAgent:     c.Request.UserAgent(),
			Reason:        fmt.Sprintf("分页查询病历列表(返回 %d 条)", len(list)),
		})
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "查询成功", Data: list})
}

func (ctrl *MedicalController) GetByID(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	rec, err := service.DefaultMedicalService.GetRecordByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "记录不存在"})
		return
	}

	currentUserID := c.GetUint64("user_id")
	role := c.GetString("role")

	// 1. 若当前用户角色为 admin：直接返回 403 Forbidden（管理员严禁查阅患者病历详情）
	if role == "admin" {
		c.JSON(http.StatusForbidden, model.Response{
			Code:    403,
			Message: "权限拦截 (403 Forbidden): 系统管理权限与临床数据分离，管理员严禁查阅患者病历详情",
		})
		return
	}

	// 2. 若当前用户角色为 patient：必须满足 record.PatientID == currentUserID，否则返回 403 Forbidden
	if role == "patient" && rec.PatientID != currentUserID {
		c.JSON(http.StatusForbidden, model.Response{
			Code:    403,
			Message: "权限拦截 (403 Forbidden): 水平越权拦截(IDOR)：禁止查阅非本人的就诊病历",
		})
		return
	}

	// 统一由 DefaultAccessEngine.Evaluate 鉴权并进行风险与合规控制 (RBAC + ABAC + 患者授权 + IDOR 防御)
	decision, err := service.DefaultAccessEngine.Evaluate(service.AccessRequest{
		UserID:      currentUserID,
		Role:        role,
		PatientID:   rec.PatientID,
		RecordID:    rec.ID,
		Action:      "READ",
		Purpose:     "DIAGNOSIS",
		Source:      c.GetHeader("X-Source"),
		IPAddress:   c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
		IsEmergency: false,
	})
	if err != nil || !decision.Allowed {
		c.JSON(http.StatusForbidden, model.Response{
			Code:    403,
			Message: fmt.Sprintf("调阅权限拦截 (403 Forbidden): %s", decision.Reason),
			Data:    decision,
		})
		return
	}

	if role == "doctor" {
		service.DefaultAccessEngine.RecordSuccessfulAccess(currentUserID, rec.PatientID)
	} else if role == "supervisor" {
		// 监管人员进行治理与审计核验，执行严格字段脱敏，分离治理权限与医疗数据查阅权限
		rec.Diagnosis = "【监管审计脱敏屏蔽】"
		rec.Symptoms = "【监管审计脱敏屏蔽】"
		rec.Etiology = "【监管审计脱敏屏蔽】"
		rec.TreatmentPlan = "【监管审计脱敏屏蔽】"
		rec.ChiefComplaint = "【监管审计脱敏屏蔽】"
		rec.PresentIllness = "【监管审计脱敏屏蔽】"
		rec.InitialDiagnosis = "【监管审计脱敏屏蔽】"
		rec.DiagnosticBasis = "【监管审计脱敏屏蔽】"
		rec.ExamReason = "【监管审计脱敏屏蔽】"
		rec.ExamResult = "【监管审计脱敏屏蔽】"
		rec.ExamItems = "【监管审计脱敏屏蔽】"
		rec.VitalSigns = "【监管脱敏屏蔽】"
		rec.InfectionAlert = nil
		rec.Files = nil
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "查询成功", Data: rec})
}

func (ctrl *MedicalController) Download(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	recPtr, err := repository.DefaultMedicalRepository.FindRecordByID(id)
	if err != nil || recPtr == nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "文件不存在"})
		return
	}
	rec := *recPtr

	currentUserID := c.GetUint64("user_id")
	role := c.GetString("role")

	// 统一由 DefaultAccessEngine.Evaluate 鉴权 (针对 patient 校验 IDOR，针对 admin/supervisor 校验下载禁令，针对 doctor 校验授权与风控)
	decision, err := service.DefaultAccessEngine.Evaluate(service.AccessRequest{
		UserID:      currentUserID,
		Role:        role,
		PatientID:   rec.PatientID,
		RecordID:    rec.ID,
		Action:      "DOWNLOAD",
		Purpose:     "EVIDENCE_EXPORT",
		Source:      c.GetHeader("X-Source"),
		IPAddress:   c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
		IsEmergency: false,
	})
	if err != nil || !decision.Allowed {
		c.JSON(http.StatusForbidden, model.Response{
			Code:    403,
			Message: fmt.Sprintf("凭据下载拦截 (403 Forbidden): %s", decision.Reason),
			Data:    decision,
		})
		return
	}

	// 导出带联盟链背书印章的红头 PDF 凭据前，必须强制执行区块链与 IPFS 防篡改验真，严禁为已篡改病历签发可信背书凭据
	if service.DefaultMedicalService != nil {
		service.DefaultMedicalService.VerifyRecord(&rec)
		if rec.IsTampered {
			_ = repository.DefaultMedicalRepository.UpdateIntegrityStatus(rec.ID, "TAMPERED")
			service.DefaultAuditService.Log(currentUserID, "TAMPER_BLOCKED_EXPORT", "RECORD", rec.RecordNo, rec.HospitalID, "INTERCEPTED", "HIGH", c.ClientIP())
			c.JSON(http.StatusConflict, model.Response{
				Code:    409,
				Message: fmt.Sprintf("安全拦截：该病历未通过区块链防篡改完整性核验，禁止导出可信凭据 (%s)", rec.TamperReason),
				Data:    rec,
			})
			return
		}
	}

	if role == "doctor" {
		service.DefaultAccessEngine.RecordSuccessfulAccess(currentUserID, rec.PatientID)
	}

	if pat, err := repository.DefaultUserRepository.FindByID(rec.PatientID); err == nil {
		rec.PatientName = pat.RealName
		rec.PatientIDCard = pat.IDCard
		rec.PatientPhone = pat.Phone
	}
	if doc, err := repository.DefaultUserRepository.FindByID(rec.DoctorID); err == nil {
		rec.DoctorName = doc.RealName
	}
	if hosp, err := repository.DefaultHospitalRepository.FindByID(rec.HospitalID); err == nil {
		rec.HospitalName = hosp.Name
	}

	fileName := fmt.Sprintf("临床规范电子病历凭据_%s.pdf", rec.RecordNo)
	if len(rec.Files) > 0 && rec.Files[0].FileName != "" {
		fileName = rec.Files[0].FileName
	}

	cid := ""
	fileHash := ""
	if len(rec.Files) > 0 {
		cid = rec.Files[0].IPFSCID
		fileHash = rec.Files[0].FileHash
	}

	clinicalHash := service.ComputeRecordHash(&rec)

	pdfBytes, err := pdf.GenerateAttestationPDF(pdf.AttestationData{
		RecordNo:         rec.RecordNo,
		HospitalName:     rec.HospitalName,
		EncounterType:    rec.EncounterType,
		DepartmentName:   rec.DepartmentName,
		PatientName:      rec.PatientName,
		PatientIDCard:    rec.PatientIDCard,
		PatientPhone:     rec.PatientPhone,
		DoctorName:       rec.DoctorName,
		CreatedAt:        rec.CreatedAt,
		ChiefComplaint:   rec.ChiefComplaint,
		PresentIllness:   rec.PresentIllness,
		OnsetTime:        rec.OnsetTime,
		Duration:         rec.Duration,
		VitalSigns:       rec.VitalSigns,
		ExamResult:       rec.ExamResult,
		InitialDiagnosis: rec.InitialDiagnosis,
		DiagnosticBasis:  rec.DiagnosticBasis,
		Diagnosis:        rec.Diagnosis,
		Etiology:         rec.Etiology,
		TreatmentPlan:    rec.TreatmentPlan,
		FabricTxID:       rec.FabricTxID,
		BlockHeight:      rec.BlockHeight,
		IPFSCID:          cid,
		FileHash:         fileHash,
		ClinicalHash:     clinicalHash,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "生成规范 PDF 凭据失败: " + err.Error()})
		return
	}

	encodedName := url.QueryEscape(fileName)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", encodedName, encodedName))
	c.Header("Content-Type", "application/pdf")
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// ListPatients 供医生选择患者或通过姓名/手机号/身份证号检索患者
func (ctrl *MedicalController) ListPatients(c *gin.Context) {
	keyword := c.Query("keyword")
	patients, err := service.DefaultMedicalService.ListPatients(keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: err.Error()})
		return
	}
	for i := range patients {
		patients[i].Phone = maskPhone(patients[i].Phone)
		patients[i].IDCard = maskIDCard(patients[i].IDCard)
	}
	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "查询成功", Data: patients})
}

// CreateEncounterInitial 医生创建初诊记录与开具医技检查
func (ctrl *MedicalController) CreateEncounterInitial(c *gin.Context) {
	doctorID := c.GetUint64("user_id")
	var p service.CreateEncounterInitialParams
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数解析失败: " + err.Error()})
		return
	}
	p.DoctorID = doctorID
	if p.PatientID == 0 {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "缺少必要参数：患者ID"})
		return
	}

	record, orders, err := service.DefaultMedicalService.CreateEncounterInitial(p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: err.Error()})
		return
	}

	msg := "初诊就诊记录创建成功，已联动生成医技检查单"
	if p.DirectComplete || len(orders) == 0 {
		msg = "就诊记录创建成功，已直接完成最终确诊并锚定至联盟链存证"
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: msg,
		Data: gin.H{
			"record": record,
			"orders": orders,
		},
	})
}

// ListExamOrders 查询医技检查申请单列表 (严格限制只能查询本院检查单，禁止跨院调取)
func (ctrl *MedicalController) ListExamOrders(c *gin.Context) {
	status := c.Query("status")
	patientID, _ := strconv.ParseUint(c.Query("patient_id"), 10, 64)

	role := c.GetString("role")
	userHospID := c.GetUint64("hospital_id")
	userID := c.GetUint64("user_id")

	// 零信任 IDOR 防护：患者角色只能查询本人检查单，禁止通过篡改 patient_id 横向越权
	if role == "patient" {
		if patientID != 0 && patientID != userID {
			respondError(c, http.StatusForbidden, 403, "无权查看他人检查申请单")
			return
		}
		patientID = userID
	}

	var hospitalID uint64
	if role == "supervisor" || role == "admin" {
		hospitalID, _ = strconv.ParseUint(c.Query("hospital_id"), 10, 64)
	} else {
		hospitalID = userHospID
		if hospitalID == 0 {
			if u, err := repository.DefaultUserRepository.FindByID(userID); err == nil {
				hospitalID = u.HospitalID
			}
		}
	}

	list, err := service.DefaultMedicalService.ListExamOrders(status, hospitalID, patientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "查询成功", Data: list})
}

// ProcessExamOrder 医技科室接单
func (ctrl *MedicalController) ProcessExamOrder(c *gin.Context) {
	orderID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := c.GetUint64("user_id")
	userHospID := c.GetUint64("hospital_id")
	role := c.GetString("role")

	userName := "检验科技师"
	if u, err := repository.DefaultUserRepository.FindByID(userID); err == nil && u.RealName != "" {
		userName = u.RealName
		if userHospID == 0 {
			userHospID = u.HospitalID
		}
	}

	// 校验单据存在性及本院权限
	orderPtr, err := repository.DefaultMedicalRepository.FindExamOrderByID(orderID)
	if err != nil || orderPtr == nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "检查申请单不存在"})
		return
	}
	order := *orderPtr
	if role != "admin" && role != "supervisor" && userHospID > 0 && order.HospitalID != userHospID {
		c.JSON(http.StatusForbidden, model.Response{Code: 403, Message: "权限不足：医技中心仅可接单处理本院申请单，禁止跨机构越权操作"})
		return
	}

	if err := service.DefaultMedicalService.ProcessExamOrder(orderID, userID, userName); err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "接单成功，检查状态已更新为检查中"})
}

// CompleteExamOrder 医技科室提交检查报告并回传
func (ctrl *MedicalController) CompleteExamOrder(c *gin.Context) {
	orderID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := c.GetUint64("user_id")
	userHospID := c.GetUint64("hospital_id")
	role := c.GetString("role")

	userName := "检验科/影像中心主管"
	if u, err := repository.DefaultUserRepository.FindByID(userID); err == nil && u.RealName != "" {
		userName = u.RealName
		if userHospID == 0 {
			userHospID = u.HospitalID
		}
	}

	// 校验单据存在性及本院权限
	orderPtr, err := repository.DefaultMedicalRepository.FindExamOrderByID(orderID)
	if err != nil || orderPtr == nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "检查申请单不存在"})
		return
	}
	order := *orderPtr
	if role != "admin" && role != "supervisor" && userHospID > 0 && order.HospitalID != userHospID {
		c.JSON(http.StatusForbidden, model.Response{Code: 403, Message: "权限不足：医技中心仅可出具本院检查报告，禁止跨机构越权操作"})
		return
	}

	technicianName := c.DefaultPostForm("technician_name", userName)
	examResult := c.PostForm("exam_result")
	examConclusion := c.PostForm("exam_conclusion")
	fileTitle := strings.TrimSpace(c.PostForm("file_title"))

	var fileBytes []byte
	fileName := ""
	fileType := "jpg"

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 50<<20)
	file, header, err := c.Request.FormFile("file")
	if err == nil && file != nil {
		defer file.Close()
		fb, readErr := io.ReadAll(file)
		if readErr != nil {
			c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "读取检查报告附件失败: " + readErr.Error()})
			return
		}
		if len(fb) > 0 {
			if valErr := security.ValidateMedicalFile(header.Filename, fb); valErr != nil {
				c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "文件安全校验未通过: " + valErr.Error()})
				return
			}
			fileBytes = fb
			fileName = header.Filename
			fileType = strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
			if fileType == "" {
				fileType = "jpg"
			}
		}
	}

	orderRes, err := service.DefaultMedicalService.CompleteExamOrder(service.CompleteExamOrderParams{
		OrderID:        orderID,
		TechnicianID:   userID,
		TechnicianName: technicianName,
		ExamResult:     examResult,
		ExamConclusion: examConclusion,
		FileName:       fileName,
		FileType:       fileType,
		FileData:       fileBytes,
		FileTitle:      fileTitle,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "检查报告提交成功，已完成回传与状态流转",
		Data:    orderRes,
	})
}

// CompleteEncounterFinal 医生下达最终确诊并执行可信存证上链
func (ctrl *MedicalController) CompleteEncounterFinal(c *gin.Context) {
	recordID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || recordID == 0 {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "无效的就诊记录ID"})
		return
	}
	doctorID := c.GetUint64("user_id")

	existing, err := repository.DefaultMedicalRepository.FindRecordByID(recordID)
	if err != nil || existing == nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: service.ErrRecordNotFound.Error()})
		return
	}
	if existing.DoctorID != doctorID {
		service.DefaultAuditService.LogDetailed(service.AuditEntry{
			UserID:        doctorID,
			OperationType: "COMPLETE_ENCOUNTER_IDOR",
			TargetType:    "RECORD",
			TargetID:      existing.RecordNo,
			HospitalID:    c.GetUint64("hospital_id"),
			Result:        "INTERCEPTED",
			RiskLevel:     "HIGH",
			RiskScore:     90,
			Source:        "WEB",
			IPAddress:     c.ClientIP(),
			UserAgent:     c.Request.UserAgent(),
			Reason:        fmt.Sprintf("HTTP 终诊越权拦截：当前医生(ID=%d)非记录经治医生(ID=%d)", doctorID, existing.DoctorID),
		})
		c.JSON(http.StatusForbidden, model.Response{Code: 403, Message: service.ErrEncounterForbidden.Error()})
		return
	}
	if existing.Status == "COMPLETED" && existing.FabricTxID != "" {
		c.JSON(http.StatusConflict, model.Response{Code: 409, Message: service.ErrEncounterAlreadyCompleted.Error()})
		return
	}

	var p service.CompleteEncounterFinalParams
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数解析失败: " + err.Error()})
		return
	}
	p.RecordID = recordID
	p.DoctorID = doctorID

	record, err := service.DefaultMedicalService.CompleteEncounterFinal(p)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: err.Error()})
		case errors.Is(err, service.ErrEncounterForbidden), errors.Is(err, service.ErrEncounterDoctorDisabled):
			c.JSON(http.StatusForbidden, model.Response{Code: 403, Message: err.Error()})
		case errors.Is(err, service.ErrEncounterAlreadyCompleted):
			c.JSON(http.StatusConflict, model.Response{Code: 409, Message: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "就诊病历最终确诊已确认，可信存证上链与 IPFS 存储完成",
		Data:    record,
	})
}

// retrieveAndDecryptMedicalFile 统一附件授权鉴权、IPFS 密文寻址、AES-256-GCM 动态流式解密与内容类型推断 (论文核心流式解密模型)
func (ctrl *MedicalController) retrieveAndDecryptMedicalFile(c *gin.Context, action string) (*model.MedicalFile, *model.MedicalRecord, []byte, string, error) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	filePtr, err := repository.DefaultMedicalRepository.FindFileByID(id)
	if err != nil || filePtr == nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "附件文件不存在"})
		return nil, nil, nil, "", err
	}
	file := *filePtr

	var rec model.MedicalRecord
	if file.RecordID > 0 {
		if r, err := repository.DefaultMedicalRepository.FindRecordByID(file.RecordID); err == nil && r != nil {
			rec = *r
		}
	}

	if rec.ID == 0 {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "无法关联所属病历记录"})
		return nil, nil, nil, "", fmt.Errorf("record not found")
	}

	currentUserID := c.GetUint64("user_id")
	role := c.GetString("role")

	purpose := "FILE_PREVIEW"
	if action == "DOWNLOAD" {
		purpose = "FILE_DOWNLOAD"
	}

	decision, err := service.DefaultAccessEngine.Evaluate(service.AccessRequest{
		UserID:      currentUserID,
		Role:        role,
		PatientID:   rec.PatientID,
		RecordID:    rec.ID,
		Action:      action,
		Purpose:     purpose,
		Source:      c.GetHeader("X-Source"),
		IPAddress:   c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
		IsEmergency: false,
	})
	if err != nil || !decision.Allowed {
		c.JSON(http.StatusForbidden, model.Response{
			Code:    403,
			Message: fmt.Sprintf("医疗附件调阅拦截 (403 Forbidden): %s", decision.Reason),
			Data:    decision,
		})
		return nil, nil, nil, "", fmt.Errorf("access denied")
	}

	if role == "doctor" {
		service.DefaultAccessEngine.RecordSuccessfulAccess(currentUserID, rec.PatientID)
	}

	var plainData []byte
	if file.IPFSCID != "" {
		data, decryptedHash, decErr := service.DefaultMedicalService.DecryptFileAndComputeHash(&rec, &file)
		if decErr != nil {
			// 若已启用信封密钥或已成功拉取 IPFS 密文却验签/解密失败，严禁静默降级为仿真预览或重定向
			if file.EncryptedDEK != "" || !strings.Contains(decErr.Error(), "从 IPFS 拉取密文失败") {
				_ = repository.DefaultMedicalRepository.UpdateIntegrityStatus(rec.ID, "TAMPERED")
				service.DefaultAuditService.Log(currentUserID, "FILE_INTEGRITY_FAILED", "RECORD", rec.RecordNo, rec.HospitalID, "INTERCEPTED", "HIGH", c.ClientIP())
				c.JSON(http.StatusConflict, model.Response{
					Code:    409,
					Message: fmt.Sprintf("安全拦截：附件密文块拉取或 AES-256-GCM 认证标签校验失败，文件可能已遭篡改或损坏 (%v)", decErr),
				})
				return nil, nil, nil, "", fmt.Errorf("attachment integrity check failed: %w", decErr)
			}
		} else {
			if file.FileHash != "" && decryptedHash != file.FileHash {
				_ = repository.DefaultMedicalRepository.UpdateIntegrityStatus(rec.ID, "TAMPERED")
				service.DefaultAuditService.Log(currentUserID, "FILE_HASH_MISMATCH", "RECORD", rec.RecordNo, rec.HospitalID, "INTERCEPTED", "HIGH", c.ClientIP())
				c.JSON(http.StatusConflict, model.Response{
					Code:    409,
					Message: "安全拦截：附件解密明文 SHA-256 指纹与登记指纹不匹配，附件可能已遭替换",
				})
				return nil, nil, nil, "", fmt.Errorf("attachment hash mismatch")
			}
			plainData = data
		}
	}

	ext := strings.ToLower(file.FileType)
	contentType := "application/octet-stream"
	switch ext {
	case "png":
		contentType = "image/png"
	case "jpg", "jpeg":
		contentType = "image/jpeg"
	case "gif":
		contentType = "image/gif"
	case "webp":
		contentType = "image/webp"
	case "svg":
		contentType = "image/svg+xml"
	case "pdf":
		contentType = "application/pdf"
	}

	return &file, &rec, plainData, contentType, nil
}

// ViewMedicalFile 查阅病历附件或影像（支持图片/PDF在线预览，从 IPFS 拉取并在内存中动态流式解密，零磁盘落地）
func (ctrl *MedicalController) ViewMedicalFile(c *gin.Context) {
	file, _, plainData, contentType, err := ctrl.retrieveAndDecryptMedicalFile(c, "VIEW")
	if err != nil {
		return
	}

	if len(plainData) > 0 {
		c.Header("Content-Type", contentType)
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename*=UTF-8''%s", url.QueryEscape(file.FileName)))
		c.Data(http.StatusOK, contentType, plainData)
		return
	}

	// 密文未就绪或初始模板记录时，以医学切片/报告动态流式渲染兜底
	ctrl.serveMockMedicalImage(c, file)
}

// DownloadMedicalFile 下载病历附件 (内存动态解密流式传输)
func (ctrl *MedicalController) DownloadMedicalFile(c *gin.Context) {
	file, rec, plainData, _, err := ctrl.retrieveAndDecryptMedicalFile(c, "DOWNLOAD")
	if err != nil {
		return
	}

	if len(plainData) > 0 {
		encodedFileName := url.QueryEscape(file.FileName)
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", encodedFileName, encodedFileName))
		c.Header("Content-Type", "application/octet-stream")
		c.Data(http.StatusOK, "application/octet-stream", plainData)
		return
	}

	if rec != nil && rec.ID > 0 {
		c.Redirect(http.StatusFound, fmt.Sprintf("/api/v1/medical-records/%d/download", rec.ID))
		return
	}
	ctrl.serveMockMedicalImage(c, file)
}

func (ctrl *MedicalController) serveMockMedicalImage(c *gin.Context, file *model.MedicalFile) {
	var rec model.MedicalRecord
	if file.RecordID > 0 {
		if r, err := repository.DefaultMedicalRepository.FindRecordByID(file.RecordID); err == nil && r != nil {
			rec = *r
		}
	}
	verifyColor := "#f59e0b"
	verifyText := "临床元数据预览切片（未检索到链下加密附件实体，仅作结构化预览）"
	if rec.ID > 0 && service.DefaultMedicalService != nil {
		service.DefaultMedicalService.VerifyRecord(&rec)
		if rec.IsTampered {
			c.JSON(http.StatusConflict, model.Response{
				Code:    409,
				Message: fmt.Sprintf("安全拦截：关联病历未通过防篡改校验，禁止渲染存证预览切片 (%s)", rec.TamperReason),
			})
			return
		}
		if rec.Verified {
			verifyColor = "#22c55e"
			verifyText = "Hyperledger Fabric 账本结构化摘要校验：通过 (元数据预览切片)"
		}
	}
	diag := rec.Diagnosis
	if diag == "" {
		diag = "临床常规辅助检查未见显著器质性病变"
	}
	if len(diag) > 60 {
		diag = diag[:60] + "..."
	}

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="900" height="580" viewBox="0 0 900 580">
  <defs>
    <linearGradient id="bgGrad" x1="0%%" y1="0%%" x2="100%%" y2="100%%">
      <stop offset="0%%" stop-color="#090d16" />
      <stop offset="50%%" stop-color="#0f172a" />
      <stop offset="100%%" stop-color="#020617" />
    </linearGradient>
    <pattern id="pacsGrid" width="30" height="30" patternUnits="userSpaceOnUse">
      <path d="M 30 0 L 0 0 0 30" fill="none" stroke="#1e293b" stroke-width="0.8" opacity="0.6"/>
    </pattern>
  </defs>

  <!-- 背景与 PACS 网格 -->
  <rect width="900" height="580" fill="url(#bgGrad)" />
  <rect width="900" height="580" fill="url(#pacsGrid)" />

  <!-- 头部信息栏 -->
  <rect x="25" y="25" width="850" height="65" rx="8" fill="#1e293b" fill-opacity="0.7" stroke="#334155" stroke-width="1.2" />
  <text x="45" y="55" font-family="'Segoe UI', 'Microsoft YaHei', sans-serif" font-size="18" font-weight="bold" fill="#38bdf8">MedTrust 医疗数据跨机构可信共享 · 医技检验/影像存证切片</text>
  <text x="45" y="76" font-family="'Segoe UI', 'Microsoft YaHei', sans-serif" font-size="12" fill="#94a3b8">业务单号：%s | 存储类型：IPFS 分布式节点 | 格式：%s</text>

  <!-- 影像主视窗 (仿真 PACS 视窗) -->
  <rect x="25" y="105" width="850" height="340" rx="8" fill="#020617" stroke="#38bdf8" stroke-width="1" stroke-opacity="0.4" />

  <!-- 心电 / 生化波形图仿真 -->
  <path d="M 40 280 L 120 280 L 135 250 L 145 320 L 155 210 L 170 340 L 180 270 L 195 290 L 260 280 
           L 320 280 L 335 250 L 345 320 L 355 210 L 370 340 L 380 270 L 395 290 L 460 280
           L 520 280 L 535 250 L 545 320 L 555 210 L 570 340 L 580 270 L 595 290 L 660 280
           L 720 280 L 735 250 L 745 320 L 755 210 L 770 340 L 780 270 L 795 290 L 860 280"
        fill="none" stroke="#22c55e" stroke-width="2.2" stroke-linejoin="round" />

  <!-- 辅助测量标尺与坐标 -->
  <line x1="40" y1="280" x2="860" y2="280" stroke="#15803d" stroke-width="0.8" stroke-dasharray="4,4" opacity="0.5" />
  <text x="50" y="140" font-family="monospace" font-size="14" fill="#38bdf8" font-weight="bold">【文件名称】%s</text>
  <text x="50" y="165" font-family="'Microsoft YaHei', sans-serif" font-size="13" fill="#cbd5e1">【临床结论】%s</text>
  <text x="50" y="190" font-family="'Microsoft YaHei', sans-serif" font-size="12" fill="#94a3b8">【生命体征与参数】%s</text>

  <!-- 密码学与区块链防伪印记 -->
  <rect x="25" y="460" width="850" height="95" rx="8" fill="#1e293b" fill-opacity="0.8" stroke="#334155" stroke-width="1.2" />
  <text x="45" y="488" font-family="monospace" font-size="12" fill="#a78bfa">IPFS CID: %s</text>
  <text x="45" y="512" font-family="monospace" font-size="12" fill="#64748b">原始明文 SHA-256: %s</text>
  <text x="45" y="536" font-family="'Microsoft YaHei', sans-serif" font-size="13" font-weight="bold" fill="%s">%s</text>
</svg>`,
		html.EscapeString(rec.RecordNo), html.EscapeString(strings.ToUpper(file.FileType)),
		html.EscapeString(file.FileName), html.EscapeString(diag), html.EscapeString(rec.VitalSigns),
		html.EscapeString(file.IPFSCID), html.EscapeString(file.FileHash),
		verifyColor, html.EscapeString(verifyText),
	)

	encodedSvgName := url.QueryEscape(file.FileName + ".svg")
	c.Header("Content-Type", "image/svg+xml")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"; filename*=UTF-8''%s", encodedSvgName, encodedSvgName))
	c.Data(http.StatusOK, "image/svg+xml", []byte(svg))
}

// GetPatientInfectionRisks 查询患者高危传染病及医护安全威胁分析
func (ctrl *MedicalController) GetPatientInfectionRisks(c *gin.Context) {
	patientIDStr := c.Param("id")
	patientID, err := strconv.ParseUint(patientIDStr, 10, 64)
	if err != nil || patientID == 0 {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "无效的患者ID"})
		return
	}

	currentUserID := c.GetUint64("user_id")
	role := c.GetString("role")

	// 身份校验：仅允许 doctor 或患者本人访问；admin、supervisor 与非本人患者一律返回 403 Forbidden（治理/管理权限与临床病原学数据隔离）
	if role == "admin" || role == "supervisor" {
		c.JSON(http.StatusForbidden, model.Response{Code: 403, Message: "治理/管理权限与临床病原学数据分离，禁止访问患者高危传染病临床明细"})
		return
	}
	if role == "patient" && currentUserID != patientID {
		c.JSON(http.StatusForbidden, model.Response{Code: 403, Message: "权限拦截：禁止查看其他患者的高危传染病信息"})
		return
	}
	if role != "doctor" && role != "patient" {
		c.JSON(http.StatusForbidden, model.Response{Code: 403, Message: "权限不足，禁止访问"})
		return
	}

	// 若调用者为医生，必须按其实际已授权病历集合 (authSet) 计算风险，严禁穿透暴露患者外院 PRIVATE 锁定的私密病名与单号
	if role == "doctor" {
		recs, _ := service.DefaultMedicalService.GetRecords(patientID, 0, 0, 0, currentUserID, false, "", "", 0, 0)
		authSet := make(map[uint64]bool)
		for _, r := range recs {
			if r.HasAccess {
				authSet[r.ID] = true
			}
		}
		if len(authSet) == 0 {
			// WP-07: 分级脱敏响应——预授权急诊分诊医生不再被 403 硬阻断，改为返回脱敏职业暴露防护等级，
			// 既保护医护人员免遭针刺伤职业暴露，又保守患者具体病原学诊断隐私 (不泄露 HIV/梅毒等病名)
			fullAlert, err := service.GetInfectionService().AnalyzePatientRisks(patientID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "排查患者传染病风险失败: " + err.Error()})
				return
			}
			precautionLevel := "LEVEL_1_STANDARD"
			bannerText := "✅ 常规标准防护：按标准预防原则执行即可，暂未发现需特殊升级防护的提示"
			if fullAlert.HasRisk {
				precautionLevel = "LEVEL_3_BLOOD_BODY_FLUID"
				bannerText = "⚠️ 职业暴露三级防护预警：接诊/穿刺操作请严格佩戴双层手套与护目镜（具体病原学诊断受隐私保护，需患者授权或紧急破窗后解锁）"
			}
			service.DefaultAuditService.LogDetailed(service.AuditEntry{
				UserID:        currentUserID,
				OperationType: "TRIAGE_EXPOSURE_PRECHECK",
				TargetType:    "PATIENT",
				TargetID:      fmt.Sprintf("%d", patientID),
				Result:        "DEIDENTIFIED_PRECAUTION",
				RiskLevel:     "MEDIUM",
				Reason:        "预授权分诊医生触发职业暴露防护等级预检，已返回脱敏防护等级并隐去具体病原学诊断",
				IPAddress:     c.ClientIP(),
				UserAgent:     c.Request.UserAgent(),
				Source:        "INFECTION_RISK",
			})
			c.JSON(http.StatusOK, model.Response{Code: 200, Message: "success", Data: model.InfectionRiskAlert{
				HasRisk:         fullAlert.HasRisk,
				RiskLevel:       "DEIDENTIFIED",
				Summary:         bannerText,
				Diseases:        nil,
				Precautions:     nil,
				ProtectionGear:  nil,
				EmergencySteps:  nil,
				DetectedFrom:    nil,
				PrecautionLevel: precautionLevel,
				BannerText:      bannerText,
			}})
			return
		}

		scopedAlert, err := service.GetInfectionService().AnalyzePatientRisksWithScope(patientID, authSet)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "排查患者传染病风险失败: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, model.Response{Code: 200, Message: "success", Data: scopedAlert})
		return
	}

	alert, err := service.GetInfectionService().AnalyzePatientRisks(patientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "排查患者传染病风险失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "success", Data: alert})
}

// GetRecordInfectionRisks 查询病历涉及的高危传染病及医护安全威胁分析 (防越权保护)
func (ctrl *MedicalController) GetRecordInfectionRisks(c *gin.Context) {
	recordIDStr := c.Param("id")
	recordID, err := strconv.ParseUint(recordIDStr, 10, 64)
	if err != nil || recordID == 0 {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "无效的病历ID"})
		return
	}

	currentUserID := c.GetUint64("user_id")
	role := c.GetString("role")
	if role == "admin" || role == "supervisor" {
		c.JSON(http.StatusForbidden, model.Response{
			Code:    403,
			Message: "治理/管理权限与临床病原学数据分离，禁止访问病历高危传染病临床明细",
		})
		return
	}

	recPtr, err := repository.DefaultMedicalRepository.FindRecordByID(recordID)
	if err != nil || recPtr == nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "病历不存在"})
		return
	}
	rec := *recPtr

	decision, err := service.DefaultAccessEngine.Evaluate(service.AccessRequest{
		UserID:      currentUserID,
		Role:        role,
		PatientID:   rec.PatientID,
		RecordID:    rec.ID,
		Action:      "READ",
		Purpose:     "INFECTION_ALERT",
		Source:      c.GetHeader("X-Source"),
		IPAddress:   c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
		IsEmergency: false,
	})
	if err != nil || !decision.Allowed {
		c.JSON(http.StatusForbidden, model.Response{
			Code:    403,
			Message: fmt.Sprintf("权限拦截 (403 Forbidden): %s", decision.Reason),
		})
		return
	}

	if role == "doctor" {
		service.DefaultAccessEngine.RecordSuccessfulAccess(currentUserID, rec.PatientID)
		authSet := map[uint64]bool{recordID: true}
		if rec.PatientID > 0 {
			recs, _ := service.DefaultMedicalService.GetRecords(rec.PatientID, 0, 0, 0, currentUserID, false, "", "", 0, 0)
			for _, r := range recs {
				if r.HasAccess {
					authSet[r.ID] = true
				}
			}
			alert, err := service.GetInfectionService().AnalyzePatientRisksWithScope(rec.PatientID, authSet)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "排查病历传染病风险失败: " + err.Error()})
				return
			}
			c.JSON(http.StatusOK, model.Response{Code: 200, Message: "success", Data: alert})
			return
		}
	}

	alert, err := service.GetInfectionService().AnalyzeRecordRisks(recordID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "排查病历传染病风险失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "success", Data: alert})
}

// EvaluateInfectionText 即时文本高危传染病评估 (供医生接诊输入时动态校验)
func (ctrl *MedicalController) EvaluateInfectionText(c *gin.Context) {
	var req struct {
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数解析失败"})
		return
	}

	alert := service.GetInfectionService().EvaluateDirectText(req.Text)
	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "success", Data: alert})
}

// DiagnosticAssist 医生工作台：AI 跨期病历关联辅助诊断与鉴别分析
func (ctrl *MedicalController) DiagnosticAssist(c *gin.Context) {
	currentUserID := c.GetUint64("user_id")
	role := c.GetString("role")
	if role != "doctor" {
		c.JSON(http.StatusForbidden, model.Response{Code: 403, Message: "仅执业医师角色可调用 AI 辅助诊断服务"})
		return
	}

	var req model.DiagnosticAssistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误: " + err.Error()})
		return
	}

	result, err := service.GetAIDiagnosticService().AnalyzePatientDiagnosticCorrelation(currentUserID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "AI 辅助诊断执行失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "AI 跨期病历关联辅助诊断推演成功",
		Data:    result,
	})
}
