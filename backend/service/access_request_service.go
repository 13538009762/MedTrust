package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"medtrust-backend/model"
	"medtrust-backend/pkg/sse"
	"medtrust-backend/repository"
)

// AccessRequestService WP-14: 访问申请/知情同意领域服务
// 承载原 access_request_controller.go 中的风控评估、申请单落库、链上授权存证、
// DB 事务与 SSE 实时通知等业务逻辑，控制器退化为薄 HTTP 适配层。
type AccessRequestService struct{}

var DefaultAccessRequestService = &AccessRequestService{}

// AccessRequestInput 常规访问申请入参
type AccessRequestInput struct {
	PatientID uint64
	RecordID  uint64
	Purpose   string
}

// SubmitAccessRequest 医生发起常规访问申请：风控评估 + 申请单落库 + 审计
func (s *AccessRequestService) SubmitAccessRequest(ctx context.Context, doctorID uint64, input AccessRequestInput) (*AccessDecision, error) {
	decision, err := DefaultAccessEngine.EvaluateAccess(doctorID, input.PatientID, input.RecordID, false)
	if err != nil {
		return nil, err
	}

	// 记录访问申请与系统决策
	var doctor model.User
	if d, err := repository.DefaultUserRepository.FindByID(doctorID); err == nil && d != nil {
		doctor = *d
	}
	var record model.MedicalRecord
	if r, err := repository.DefaultMedicalRepository.FindRecordByID(input.RecordID); err == nil && r != nil {
		record = *r
	}

	accessRec := model.AccessRequest{
		// 与既有单号格式保持一致：REQ{yyyyMMdd}{rand}
		RequestNo:        fmt.Sprintf("REQ%s%s", time.Now().Format("20060102"), randomHex3()),
		DoctorID:         doctorID,
		PatientID:        input.PatientID,
		RecordID:         input.RecordID,
		SourceHospitalID: doctor.HospitalID,
		TargetHospitalID: record.HospitalID,
		Purpose:          input.Purpose,
		RiskScore:        decision.RiskEvaluation.TotalScore,
		RiskLevel:        decision.RiskEvaluation.Level,
		Decision:         decision.Decision,
		Status:           "PENDING",
		ScopeType:        "SINGLE",
		Days:             7,
		ExpireAt:         time.Now().AddDate(0, 0, 7),
		CreatedAt:        time.Now(),
	}
	_ = repository.DefaultAuthPolicyRepository.CreateAccessRequest(&accessRec)

	result := "INTERCEPTED"
	if decision.Allowed {
		result = "SUCCESS"
	}
	DefaultAuditService.Log(doctorID, "ACCESS", "RECORD", record.RecordNo, doctor.HospitalID, result, decision.RiskEvaluation.Level, clientIPFromContext(ctx))
	return decision, nil
}

// ApplyConsentInput 知情同意申请入参
type ApplyConsentInput struct {
	RecordID  uint64
	PatientID uint64
	Purpose   string
	ScopeType string
	Days      int
}

// ApplyConsent 医生发起知情授权申请（通知患者在线核准）
func (s *AccessRequestService) ApplyConsent(ctx context.Context, doctorID uint64, input ApplyConsentInput) (*model.AccessRequest, error) {
	var targetPatientID uint64
	var targetHospitalID uint64
	if input.RecordID > 0 {
		rec, err := repository.DefaultMedicalRepository.FindRecordByID(input.RecordID)
		if err != nil || rec == nil {
			return nil, ErrRecordNotFound
		}
		targetPatientID = rec.PatientID
		targetHospitalID = rec.HospitalID
	} else if input.PatientID > 0 {
		pat, err := repository.DefaultUserRepository.FindByID(input.PatientID)
		if err != nil || pat == nil {
			return nil, ErrPatientNotFound
		}
		targetPatientID = pat.ID
		input.ScopeType = "ALL"
	} else {
		return nil, ErrMissingTarget
	}

	var doctor model.User
	if d, err := repository.DefaultUserRepository.FindByID(doctorID); err == nil && d != nil {
		doctor = *d
	}

	if input.Purpose == "" {
		input.Purpose = "门诊专科联合随访评估与既往慢病复核，需调阅外院历史健康档案"
	}
	if input.ScopeType == "" {
		input.ScopeType = "SINGLE"
	}
	days, clampErr := ClampAuthDays(input.Days, 7)
	if clampErr != nil {
		return nil, clampErr
	}
	input.Days = days

	reqNo := generateRequestNo("REQ-CONSENT")
	accessRec := model.AccessRequest{
		RequestNo:        reqNo,
		DoctorID:         doctorID,
		PatientID:        targetPatientID,
		RecordID:         input.RecordID,
		SourceHospitalID: doctor.HospitalID,
		TargetHospitalID: targetHospitalID,
		Purpose:          input.Purpose,
		RiskScore:        15,
		RiskLevel:        "LOW",
		Decision:         "PENDING_CONFIRM",
		Status:           "PENDING",
		ScopeType:        input.ScopeType,
		Days:             input.Days,
		ExpireAt:         time.Now().AddDate(0, 0, input.Days),
		CreatedAt:        time.Now(),
	}
	_ = repository.DefaultAuthPolicyRepository.CreateAccessRequest(&accessRec)

	DefaultAuditService.Log(doctorID, "CONSENT_APPLY", "REQUEST", reqNo, doctor.HospitalID, "PENDING", "LOW", clientIPFromContext(ctx))

	// 通过 SSE 实时推送通知患者
	docName := doctorDisplayName(doctor)
	hospName := hospitalDisplayName(doctor.HospitalID)
	sse.Default.Publish(targetPatientID, sse.NotifyEvent{
		Type:    "access_request",
		Title:   "收到新的跨院调阅申请",
		Message: fmt.Sprintf("%s（%s）向您申请跨院调阅医疗病历，请及时在线知情核准", docName, hospName),
		Payload: map[string]interface{}{
			"request_id":  accessRec.ID,
			"request_no":  reqNo,
			"doctor_name": docName,
			"hospital":    hospName,
			"scope_type":  input.ScopeType,
		},
	})
	return &accessRec, nil
}

// BatchApplyConsentInput 批量知情同意申请入参
type BatchApplyConsentInput struct {
	PatientID uint64
	RecordIDs []uint64
	Purpose   string
	Days      int
}

// BatchApplyConsent 医生分批/勾选申请多份病历的调阅知情同意
func (s *AccessRequestService) BatchApplyConsent(ctx context.Context, doctorID uint64, input BatchApplyConsentInput) ([]model.AccessRequest, error) {
	if len(input.RecordIDs) == 0 {
		return nil, ErrEmptyRecordList
	}

	var doctor model.User
	if d, err := repository.DefaultUserRepository.FindByID(doctorID); err == nil && d != nil {
		doctor = *d
	}

	if input.Purpose == "" {
		input.Purpose = "急救临床协同诊疗，批量申请调阅患者既往受控病历档案"
	}
	days, clampErr := ClampAuthDays(input.Days, 7)
	if clampErr != nil {
		return nil, clampErr
	}
	input.Days = days

	records, err := repository.DefaultMedicalRepository.FindRecordsByIDs(input.RecordIDs)
	if err != nil || len(records) != len(input.RecordIDs) {
		return nil, ErrRecordListMismatch
	}
	for _, rec := range records {
		if rec.PatientID != input.PatientID {
			return nil, fmt.Errorf("越权拦截：病历记录(ID:%d)不属于患者(ID:%d)，禁止跨患者批量申请", rec.ID, input.PatientID)
		}
	}

	var createdRequests []model.AccessRequest
	now := time.Now()
	expireAt := now.AddDate(0, 0, input.Days)

	for _, rec := range records {
		accessRec := model.AccessRequest{
			RequestNo:        generateRequestNo("REQ-CONSENT"),
			DoctorID:         doctorID,
			PatientID:        input.PatientID,
			RecordID:         rec.ID,
			SourceHospitalID: doctor.HospitalID,
			TargetHospitalID: rec.HospitalID,
			Purpose:          input.Purpose,
			RiskScore:        15,
			RiskLevel:        "LOW",
			Decision:         "PENDING_CONFIRM",
			Status:           "PENDING",
			ScopeType:        "SINGLE",
			Days:             input.Days,
			ExpireAt:         expireAt,
			CreatedAt:        now,
		}
		_ = repository.DefaultAuthPolicyRepository.CreateAccessRequest(&accessRec)
		createdRequests = append(createdRequests, accessRec)
	}

	DefaultAuditService.Log(doctorID, "BATCH_CONSENT_APPLY", "REQUEST_BATCH", fmt.Sprintf("PATIENT_%d", input.PatientID), doctor.HospitalID, "PENDING", "LOW", clientIPFromContext(ctx))

	if len(createdRequests) > 0 {
		docName := doctorDisplayName(doctor)
		hospName := hospitalDisplayName(doctor.HospitalID)
		sse.Default.Publish(input.PatientID, sse.NotifyEvent{
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
	return createdRequests, nil
}

// ListPendingRequests 患者视角待处理调阅申请列表（批量预加载医生/病历/医院展示字段）
func (s *AccessRequestService) ListPendingRequests(ctx context.Context, patientID uint64) ([]model.AccessRequest, error) {
	list, _ := repository.DefaultAuthPolicyRepository.FindPendingRequestsByPatient(patientID, 100)
	if len(list) == 0 {
		return list, nil
	}

	docIDs := make([]uint64, 0, len(list))
	recIDs := make([]uint64, 0, len(list))
	for _, item := range list {
		docIDs = append(docIDs, item.DoctorID)
		recIDs = append(recIDs, item.RecordID)
	}

	docMap := loadUsersBatchService(docIDs)
	recMap := loadRecordsBatchService(recIDs)

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
	hospMap := loadHospitalsBatchService(hospIDs)

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
	return list, nil
}

// ApproveRequest 患者核准同意调阅申请：链上存证 + 事务化建立授权 + SSE 通知医生
func (s *AccessRequestService) ApproveRequest(ctx context.Context, patientID uint64, req *model.AccessRequest) (*model.Authorization, error) {
	days := req.Days
	if days <= 0 {
		days = 7
	}
	startTime := time.Now()
	endTime := startTime.Add(time.Duration(days) * 24 * time.Hour)

	authNo := fmt.Sprintf("AUTH%s%s", time.Now().Format("20060102"), randomHex3())

	txID, _, cErr := CommitAssetWithCompensation("AUTHORIZATION", authNo, map[string]interface{}{
		"auth_no":     authNo,
		"patient_id":  patientID,
		"target_type": "DOCTOR",
		"target_id":   req.DoctorID,
		"scope":       req.ScopeType,
		"record_id":   req.RecordID,
		"status":      "ACTIVE",
		"timestamp":   time.Now().Format(time.RFC3339),
	})
	if cErr != nil {
		// 修复 5：知情同意授权存证失败不得静默，落显式审计（授权仍按 DB 生效，补偿队列负责重放）
		AuditChainAnchoringPending(patientID, "APPROVE_CONSENT_ANCHOR_PENDING", "AUTH", authNo, 0, cErr)
	}

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

	// WP-10: 申请单状态更新与授权建立必须原子生效，杜绝"已批准但授权缺失"的中间态
	req.Status = "APPROVED"
	req.Decision = "ALLOWED"
	if err := repository.Transaction(func(tx *repository.TxRunner) error {
		if err := tx.Create(&auth); err != nil {
			return err
		}
		return tx.Save(req)
	}); err != nil {
		return nil, err
	}

	DefaultAuditService.Log(patientID, "APPROVE_CONSENT", "AUTH", authNo, 0, "SUCCESS", "LOW", clientIPFromContext(ctx))

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
	return &auth, nil
}

// RejectRequest 患者驳回调阅申请
func (s *AccessRequestService) RejectRequest(ctx context.Context, patientID uint64, req *model.AccessRequest) error {
	req.Status = "REJECTED"
	req.Decision = "REJECTED"
	_ = repository.DefaultAuthPolicyRepository.SaveAccessRequest(req)

	DefaultAuditService.Log(patientID, "REJECT_CONSENT", "REQUEST", req.RequestNo, 0, "SUCCESS", "LOW", clientIPFromContext(ctx))

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
	return nil
}

// BatchConsentProcessResult 批量审批/驳回结果汇总
type BatchConsentProcessResult struct {
	TotalCount   int      `json:"total_count"`
	SuccessCount int      `json:"success_count"`
	FailedCount  int      `json:"failed_count"`
	AuthNos      []string `json:"auth_nos,omitempty"`
	Errors       []string `json:"errors,omitempty"`
}

// BatchApproveRequests 批量或一键全部核准调阅申请，释放访问权限并自动建立上链授权
func (s *AccessRequestService) BatchApproveRequests(ctx context.Context, patientID uint64, requestIDs []uint64, all bool) (*BatchConsentProcessResult, error) {
	var reqs []model.AccessRequest
	var err error
	if all {
		reqs, err = repository.DefaultAuthPolicyRepository.FindPendingRequestsByPatient(patientID, 500)
	} else {
		reqs, err = repository.DefaultAuthPolicyRepository.FindPendingRequestsByIDs(patientID, requestIDs)
	}
	if err != nil {
		return nil, err
	}

	result := &BatchConsentProcessResult{
		TotalCount: len(reqs),
		AuthNos:    make([]string, 0, len(reqs)),
		Errors:     make([]string, 0),
	}

	for i := range reqs {
		req := &reqs[i]
		if req.Status != "PENDING" {
			continue
		}
		auth, approveErr := s.ApproveRequest(ctx, patientID, req)
		if approveErr != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Sprintf("申请单 %s: %v", req.RequestNo, approveErr))
		} else {
			result.SuccessCount++
			if auth != nil {
				result.AuthNos = append(result.AuthNos, auth.AuthNo)
			}
		}
	}
	return result, nil
}

// BatchRejectRequests 批量或一键全部驳回调阅申请
func (s *AccessRequestService) BatchRejectRequests(ctx context.Context, patientID uint64, requestIDs []uint64, all bool) (*BatchConsentProcessResult, error) {
	var reqs []model.AccessRequest
	var err error
	if all {
		reqs, err = repository.DefaultAuthPolicyRepository.FindPendingRequestsByPatient(patientID, 500)
	} else {
		reqs, err = repository.DefaultAuthPolicyRepository.FindPendingRequestsByIDs(patientID, requestIDs)
	}
	if err != nil {
		return nil, err
	}

	result := &BatchConsentProcessResult{
		TotalCount: len(reqs),
		Errors:     make([]string, 0),
	}

	for i := range reqs {
		req := &reqs[i]
		if req.Status != "PENDING" {
			continue
		}
		rejectErr := s.RejectRequest(ctx, patientID, req)
		if rejectErr != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Sprintf("申请单 %s: %v", req.RequestNo, rejectErr))
		} else {
			result.SuccessCount++
		}
	}
	return result, nil
}

// --- 内部辅助 ---

func generateRequestNo(prefix string) string {
	return fmt.Sprintf("%s-%s-%s", prefix, time.Now().Format("20060102"), randomHex3())
}

func randomHex3() string {
	randBytes := make([]byte, 3)
	_, _ = rand.Read(randBytes)
	return hex.EncodeToString(randBytes)
}

// doctorDisplayName 医生展示名（实名优先，回退用户名）
func doctorDisplayName(doctor model.User) string {
	if doctor.RealName != "" {
		return doctor.RealName
	}
	return doctor.Username
}

// hospitalDisplayName 医院展示名（查询失败回退默认值）
func hospitalDisplayName(hospitalID uint64) string {
	if hospPtr, err := repository.DefaultHospitalRepository.FindByID(hospitalID); err == nil && hospPtr != nil && hospPtr.Name != "" {
		return hospPtr.Name
	}
	return "外院医疗机构"
}

// clientIPFromContext 从上下文提取客户端 IP（缺失时回退空串）
func clientIPFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if ip, ok := ctx.Value(ClientIPContextKey{}).(string); ok {
		return ip
	}
	return ""
}

// ClientIPContextKey gin 上下文中客户端 IP 的键类型
type ClientIPContextKey struct{}