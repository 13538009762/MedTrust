package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"medtrust-backend/model"
	"medtrust-backend/repository"
)

// KeyUnlockService WP-14: 患者密钥解锁领域服务
// 承载原 key_unlock_controller.go 中的密钥核验、防暴锁定、
// 链上授权存证与授权落库逻辑，控制器退化为薄 HTTP 适配层。
type KeyUnlockService struct{}

var DefaultKeyUnlockService = &KeyUnlockService{}

// --- 防暴锁定（内存级限流） ---

type keyUnlockFailState struct {
	count       int
	lockedUntil time.Time
}

var keyUnlockLimiter sync.Map

// IsKeyUnlockLocked 检查 (doctorID, patientID) 是否处于防暴锁定，返回剩余分钟数
func IsKeyUnlockLocked(doctorID, patientID uint64) (bool, int) {
	k := fmt.Sprintf("key_unlock:%d:%d", doctorID, patientID)
	if v, ok := keyUnlockLimiter.Load(k); ok {
		st := v.(keyUnlockFailState)
		if st.lockedUntil.After(time.Now()) {
			rem := int(st.lockedUntil.Sub(time.Now()).Minutes()) + 1
			return true, rem
		}
	}
	return false, 0
}

// RecordKeyUnlockResult 记录一次核验结果；失败累计 5 次后锁定 15 分钟，返回累计失败次数
func RecordKeyUnlockResult(doctorID, patientID uint64, success bool) int {
	k := fmt.Sprintf("key_unlock:%d:%d", doctorID, patientID)
	if success {
		keyUnlockLimiter.Delete(k)
		return 0
	}
	st := keyUnlockFailState{count: 1}
	if v, ok := keyUnlockLimiter.Load(k); ok {
		prev := v.(keyUnlockFailState)
		st.count = prev.count + 1
	}
	if st.count >= 5 {
		st.lockedUntil = time.Now().Add(15 * time.Minute)
	}
	keyUnlockLimiter.Store(k, st)
	return st.count
}

// UnlockPatientByKeyInput 患者全档密钥解锁入参
type UnlockPatientByKeyInput struct {
	PatientID  uint64
	MedicalKey string
	Days       int
}

// UnlockPatientByKey 医生输入患者专属密钥，核验通过后即时解锁该患者全套病历并固化上链。
// 返回建立的授权凭证；密钥错误时返回 ErrKeyVerifyFailed / ErrKeyUnlockLimit / ErrKeyUnlockLocked。
func (s *KeyUnlockService) UnlockPatientByKey(ctx context.Context, doctorID uint64, input UnlockPatientByKeyInput) (*model.Authorization, *model.User, error) {
	if locked, rem := IsKeyUnlockLocked(doctorID, input.PatientID); locked {
		_ = rem // 锁定剩余分钟数由控制器组装文案
		return nil, nil, ErrKeyUnlockLocked
	}

	patientPtr, err := repository.DefaultUserRepository.FindByID(input.PatientID)
	if err != nil || patientPtr == nil {
		return nil, nil, ErrPatientNotFound
	}
	patient := *patientPtr

	var doctor model.User
	if d, err := repository.DefaultUserRepository.FindByID(doctorID); err == nil && d != nil {
		doctor = *d
	}

	inputKey := strings.TrimSpace(input.MedicalKey)
	if !VerifyMedicalKey(inputKey, patient.UserNo, patient.MedicalKeyHash, patient.MedicalKey) {
		failCnt := RecordKeyUnlockResult(doctorID, patient.ID, false)
		DefaultAuditService.Log(doctorID, "KEY_UNLOCK_FAILED", "PATIENT", patient.UserNo, doctor.HospitalID, "INTERCEPTED", "HIGH", clientIPFromContext(ctx))
		if failCnt >= 5 {
			return nil, nil, ErrKeyUnlockLimit
		}
		return nil, nil, ErrKeyVerifyFailed
	}
	RecordKeyUnlockResult(doctorID, patient.ID, true)

	// 密钥核验通过：生成对该患者全部档案全局生效的授权凭证
	days, clampErr := ClampAuthDays(input.Days, 7)
	if clampErr != nil {
		return nil, nil, clampErr
	}
	startTime := time.Now()
	endTime := startTime.Add(time.Duration(days) * 24 * time.Hour)

	authNo := fmt.Sprintf("AUTH-KEY-ALL-%s-%s", startTime.Format("20060102"), randomHex3())
	auth := commitAndCreateAuthorizationWithMethod("AUTHORIZATION_KEY", authNo, "DOCTOR", doctorID, "ALL", 0, patient.ID, startTime, endTime, "")

	DefaultAuditService.Log(doctorID, "KEY_UNLOCK_SUCCESS", "PATIENT", patient.UserNo, doctor.HospitalID, "SUCCESS", "LOW", clientIPFromContext(ctx))
	return auth, &patient, nil
}

// UnlockByKeyInput 单份病历密钥解锁入参
type UnlockByKeyInput struct {
	RecordID   uint64
	MedicalKey string
	Days       int
}

// UnlockByKeyResult 单份解锁结果：授权 + 完整解锁病历
type UnlockByKeyResult struct {
	Auth   *model.Authorization
	Record *model.MedicalRecord
}

// UnlockByKey 医生输入病历专属密钥，核验通过后即时解锁该病历、核准关联待处理申请并加载完整内容
func (s *KeyUnlockService) UnlockByKey(ctx context.Context, doctorID uint64, input UnlockByKeyInput) (*UnlockByKeyResult, error) {
	recPtr, err := repository.DefaultMedicalRepository.FindRecordByID(input.RecordID)
	if err != nil || recPtr == nil {
		return nil, ErrRecordNotFound
	}
	rec := *recPtr

	if locked, _ := IsKeyUnlockLocked(doctorID, rec.PatientID); locked {
		return nil, ErrKeyUnlockLocked
	}

	patientPtr, err := repository.DefaultUserRepository.FindByID(rec.PatientID)
	if err != nil || patientPtr == nil {
		return nil, ErrPatientNotFound
	}
	patient := *patientPtr

	var doctor model.User
	if d, err := repository.DefaultUserRepository.FindByID(doctorID); err == nil && d != nil {
		doctor = *d
	}

	inputKey := strings.TrimSpace(input.MedicalKey)
	if !VerifyMedicalKey(inputKey, patient.UserNo, patient.MedicalKeyHash, patient.MedicalKey) {
		failCnt := RecordKeyUnlockResult(doctorID, patient.ID, false)
		DefaultAuditService.Log(doctorID, "KEY_UNLOCK_FAILED", "RECORD", rec.RecordNo, doctor.HospitalID, "INTERCEPTED", "HIGH", clientIPFromContext(ctx))
		if failCnt >= 5 {
			return nil, ErrKeyUnlockLimit
		}
		return nil, ErrKeyVerifyFailed
	}
	RecordKeyUnlockResult(doctorID, patient.ID, true)

	// 密钥核验通过：生成即时生效的授权凭证
	days, clampErr := ClampAuthDays(input.Days, 7)
	if clampErr != nil {
		return nil, clampErr
	}
	startTime := time.Now()
	endTime := startTime.Add(time.Duration(days) * 24 * time.Hour)

	authNo := fmt.Sprintf("AUTH-KEY-%s-%s", time.Now().Format("20060102"), randomHex3())
	auth := commitAndCreateAuthorizationWithMethod("AUTHORIZATION_KEY", authNo, "DOCTOR", doctorID, "SINGLE", input.RecordID, patient.ID, startTime, endTime, "PATIENT_KEY_VERIFIED")

	// 若有关于此病历与医生的待处理申请单，顺便利标记为已核准
	_ = repository.DefaultAuthPolicyRepository.ApprovePendingRequests(input.RecordID, doctorID)

	DefaultAuditService.Log(doctorID, "KEY_UNLOCK_SUCCESS", "RECORD", rec.RecordNo, doctor.HospitalID, "SUCCESS", "LOW", clientIPFromContext(ctx))

	// 加载完整解锁病历
	fullRecord, err := DefaultMedicalService.GetRecordByID(input.RecordID)
	if err != nil {
		return nil, fmt.Errorf("解锁病历失败: %w", err)
	}

	return &UnlockByKeyResult{Auth: auth, Record: fullRecord}, nil
}

// --- 内部辅助 ---

// commitAndCreateAuthorizationWithMethod 密钥解锁专用的"链上存证 + 授权落库"（支持 auth_method 扩展字段）
func commitAndCreateAuthorizationWithMethod(businessType, authNo, targetType string, targetID uint64, scopeType string, recordID, patientID uint64, startTime, endTime time.Time, authMethod string) *model.Authorization {
	payload := map[string]interface{}{
		"auth_no":     authNo,
		"patient_id":  patientID,
		"target_type": targetType,
		"target_id":   targetID,
		"scope":       scopeType,
		"record_id":   recordID,
		"status":      "ACTIVE",
		"timestamp":   time.Now().Format(time.RFC3339),
	}
	if authMethod != "" {
		payload["auth_method"] = authMethod
	}
	txID, _, cErr := CommitAssetWithCompensation(businessType, authNo, payload)
	if cErr != nil {
		// 修复 5：密钥解锁授权存证失败时必须显式审计（授权仍在本地生效），补偿队列已接手重放
		AuditChainAnchoringPending(patientID, "GRANT_AUTH_KEY", "AUTH", authNo, 0, cErr)
	}

	auth := model.Authorization{
		AuthNo:         authNo,
		PatientID:      patientID,
		AuthTargetType: targetType,
		AuthTargetID:   targetID,
		ScopeType:      scopeType,
		RecordID:       recordID,
		StartTime:      startTime,
		EndTime:        endTime,
		Status:         "ACTIVE",
		FabricTxID:     txID,
		CreatedAt:      time.Now(),
	}
	_ = repository.DefaultAuthPolicyRepository.CreateAuthorization(&auth)
	return &auth
}