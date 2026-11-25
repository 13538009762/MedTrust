package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"medtrust-backend/model"
	"medtrust-backend/repository"
)

// KeyUnlockService WP-14: 患者密钥解锁领域服务
// 承载原 key_unlock_controller.go 中的密钥核验、防暴锁定、
// 链上授权存证与授权落库逻辑，控制器退化为薄 HTTP 适配层。
type KeyUnlockService struct{}

var DefaultKeyUnlockService = &KeyUnlockService{}

// --- 防暴锁定（Redis + 本地 TTL 内存双写，消除 sync.Map 内存泄漏并增加患者全局上限） ---

const (
	keyUnlockDoctorLimit  = 5
	keyUnlockPatientLimit = 10
	keyUnlockLockDuration = 15 * time.Minute
)

func keyUnlockDoctorScope(doctorID, patientID uint64) string {
	return fmt.Sprintf("doc:%d:pat:%d", doctorID, patientID)
}

func keyUnlockPatientScope(patientID uint64) string {
	return fmt.Sprintf("pat:%d", patientID)
}

// IsKeyUnlockLocked 检查 (doctorID, patientID) 或 patientID 全局通道是否处于防暴锁定，返回剩余分钟数
func IsKeyUnlockLocked(doctorID, patientID uint64) (bool, int) {
	if locked, rem := repository.GetKeyUnlockLockRemaining(keyUnlockDoctorScope(doctorID, patientID)); locked {
		return true, rem
	}
	if locked, rem := repository.GetKeyUnlockLockRemaining(keyUnlockPatientScope(patientID)); locked {
		return true, rem
	}
	return false, 0
}

// RecordKeyUnlockResult 记录一次核验结果；单医生失败累计 5 次或跨医生对同一患者累计 10 次后锁定 15 分钟
func RecordKeyUnlockResult(doctorID, patientID uint64, success bool) int {
	docScope := keyUnlockDoctorScope(doctorID, patientID)
	patScope := keyUnlockPatientScope(patientID)
	if success {
		repository.ClearKeyUnlockFailure(docScope)
		return 0
	}
	docCnt := repository.IncrementKeyUnlockFailure(docScope, keyUnlockLockDuration)
	if docCnt >= keyUnlockDoctorLimit {
		repository.SetKeyUnlockLock(docScope, keyUnlockLockDuration)
	}
	patCnt := repository.IncrementKeyUnlockFailure(patScope, keyUnlockLockDuration)
	if patCnt >= keyUnlockPatientLimit {
		repository.SetKeyUnlockLock(patScope, keyUnlockLockDuration)
		return keyUnlockDoctorLimit // 触发统一 429 锁定上限响应
	}
	return docCnt
}

// validateKeyUnlockDoctorEligibility 校验发起密钥解锁的医生账号与医疗机构是否正常执业（防 RESTRICTED/DISABLED/停用医院绕过）
func validateKeyUnlockDoctorEligibility(doctorID uint64, targetRecordHospitalID uint64) (*model.User, error) {
	docPtr, err := repository.DefaultUserRepository.FindByID(doctorID)
	if err != nil || docPtr == nil {
		return nil, ErrDoctorNotFound
	}
	if docPtr.Role != "doctor" || docPtr.Status == "DISABLED" || docPtr.Status == "RESTRICTED" {
		return nil, ErrEncounterDoctorDisabled
	}
	if docPtr.HospitalID > 0 {
		if hosp, hErr := repository.DefaultHospitalRepository.FindByID(docPtr.HospitalID); hErr == nil && hosp != nil && hosp.Status == 0 {
			return nil, ErrEncounterDoctorDisabled
		}
	}
	if targetRecordHospitalID > 0 && targetRecordHospitalID != docPtr.HospitalID {
		if hosp, hErr := repository.DefaultHospitalRepository.FindByID(targetRecordHospitalID); hErr == nil && hosp != nil && hosp.Status == 0 {
			return nil, ErrEncounterDoctorDisabled
		}
	}
	return docPtr, nil
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
	docPtr, eligErr := validateKeyUnlockDoctorEligibility(doctorID, 0)
	if eligErr != nil {
		return nil, nil, eligErr
	}
	doctor := *docPtr

	if locked, rem := IsKeyUnlockLocked(doctorID, input.PatientID); locked {
		_ = rem // 锁定剩余分钟数由控制器组装文案
		return nil, nil, ErrKeyUnlockLocked
	}

	patientPtr, err := repository.DefaultUserRepository.FindByID(input.PatientID)
	if err != nil || patientPtr == nil {
		return nil, nil, ErrPatientNotFound
	}
	patient := *patientPtr

	inputKey := strings.TrimSpace(input.MedicalKey)
	if !VerifyMedicalKey(inputKey, patient.UserNo, patient.MedicalKeyHash, patient.MedicalKey) {
		failCnt := RecordKeyUnlockResult(doctorID, patient.ID, false)
		DefaultAuditService.Log(doctorID, "KEY_UNLOCK_FAILED", "PATIENT", patient.UserNo, doctor.HospitalID, "INTERCEPTED", "HIGH", clientIPFromContext(ctx))
		if failCnt >= keyUnlockDoctorLimit {
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

	docPtr, eligErr := validateKeyUnlockDoctorEligibility(doctorID, rec.HospitalID)
	if eligErr != nil {
		return nil, eligErr
	}
	doctor := *docPtr

	if locked, _ := IsKeyUnlockLocked(doctorID, rec.PatientID); locked {
		return nil, ErrKeyUnlockLocked
	}

	patientPtr, err := repository.DefaultUserRepository.FindByID(rec.PatientID)
	if err != nil || patientPtr == nil {
		return nil, ErrPatientNotFound
	}
	patient := *patientPtr

	inputKey := strings.TrimSpace(input.MedicalKey)
	if !VerifyMedicalKey(inputKey, patient.UserNo, patient.MedicalKeyHash, patient.MedicalKey) {
		failCnt := RecordKeyUnlockResult(doctorID, patient.ID, false)
		DefaultAuditService.Log(doctorID, "KEY_UNLOCK_FAILED", "RECORD", rec.RecordNo, doctor.HospitalID, "INTERCEPTED", "HIGH", clientIPFromContext(ctx))
		if failCnt >= keyUnlockDoctorLimit {
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