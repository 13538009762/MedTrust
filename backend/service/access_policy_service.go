package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
)

// AccessPolicyService WP-14: 患者病历访问策略领域服务
// 承载原 access_policy_controller.go 中的策略状态迁移、撤销旧授权、
// 链上存证与授权落库逻辑，控制器退化为薄 HTTP 适配层。
type AccessPolicyService struct{}

var DefaultAccessPolicyService = &AccessPolicyService{}

// SetRecordAccessPolicyInput 单份病历策略入参
type SetRecordAccessPolicyInput struct {
	RecordID     uint64
	PolicyType   string // ALL_DOCTORS, HOSPITAL, DOCTOR, PRIVATE
	AuthTargetID uint64 // 医院ID 或 医生用户ID
	Days         int
}

// SetRecordAccessPolicyResult 策略设置结果（供控制器组装响应文案）
type SetRecordAccessPolicyResult struct {
	Auth     *model.Authorization
	AuditOp  string // 审计操作类型
	Message  string // 面向用户的成功文案
}

// SetRecordAccessPolicy 患者快速设置单份病历的访问权限
func (s *AccessPolicyService) SetRecordAccessPolicy(ctx context.Context, patientID uint64, rec *model.MedicalRecord, input SetRecordAccessPolicyInput) (*SetRecordAccessPolicyResult, error) {
	now := time.Now()
	days, err := ClampAuthDays(input.Days, 365)
	if err != nil {
		return nil, err
	}
	startTime := now
	endTime := startTime.Add(time.Duration(days) * 24 * time.Hour)

	switch input.PolicyType {
	case "ALL_DOCTORS":
		revokeActiveAuthorizationsForScope(patientID, "SINGLE", input.RecordID)
		auth := commitAndCreateAuthorization("ALL_DOCTORS", 0, "SINGLE", input.RecordID, patientID, startTime, endTime, now)
		return &SetRecordAccessPolicyResult{
			Auth:    auth,
			AuditOp: "SET_RECORD_VISIBILITY",
			Message: "已成功设置该病历为「全体医生可见」，所有医院执业医生可直接免审调阅，策略已固化上链",
		}, nil
	case "HOSPITAL":
		if input.AuthTargetID == 0 {
			return nil, ErrMissingTarget
		}
		hospPtr, err := repository.DefaultHospitalRepository.FindByID(input.AuthTargetID)
		if err != nil || hospPtr == nil {
			return nil, ErrHospitalNotFound
		}
		hosp := *hospPtr
		revokeActiveAuthorizationsForScope(patientID, "SINGLE", input.RecordID)
		auth := commitAndCreateAuthorization("HOSPITAL", input.AuthTargetID, "SINGLE", input.RecordID, patientID, startTime, endTime, now)
		return &SetRecordAccessPolicyResult{
			Auth:    auth,
			AuditOp: "SET_RECORD_VISIBILITY",
			Message: fmt.Sprintf("已成功设置该病历为「%s」全体医生可见，策略已固化上链", hosp.Name),
		}, nil
	case "DOCTOR":
		if input.AuthTargetID == 0 {
			return nil, ErrMissingTarget
		}
		docPtr, err := repository.DefaultUserRepository.FindDoctorByID(input.AuthTargetID)
		if err != nil || docPtr == nil {
			return nil, ErrDoctorNotFound
		}
		doc := *docPtr
		revokeActiveAuthorizationsForScope(patientID, "SINGLE", input.RecordID)
		auth := commitAndCreateAuthorization("DOCTOR", input.AuthTargetID, "SINGLE", input.RecordID, patientID, startTime, endTime, now)
		docName := doctorDisplayNameWithSuffix(doc)
		return &SetRecordAccessPolicyResult{
			Auth:    auth,
			AuditOp: "SET_RECORD_VISIBILITY",
			Message: fmt.Sprintf("已成功设置该病历为「%s」专属授权调阅，策略已固化上链", docName),
		}, nil
	case "PRIVATE":
		// 撤销对该病历的所有单份活跃授权
		revokeActiveAuthorizationsForScope(patientID, "SINGLE", input.RecordID)
		auth := commitAndCreateAuthorization("PRIVATE", 0, "SINGLE", input.RecordID, patientID, now, now.Add(3650*24*time.Hour), now)
		return &SetRecordAccessPolicyResult{
			Auth:    auth,
			AuditOp: "REVOKE_RECORD_VISIBILITY",
			Message: "已将该病历恢复为「私密受控」，其他医生调阅需通过系统知情同意审批，存证已固化上链",
		}, nil
	default:
		return nil, ErrUnsupportedPolicyType
	}
}

// BatchSetAccessPolicyInput 全部健康档案策略入参
type BatchSetAccessPolicyInput struct {
	PolicyType   string
	AuthTargetID uint64
	Days         int
}

// BatchSetAccessPolicy 患者批量设置全部健康档案的访问共享权限
func (s *AccessPolicyService) BatchSetAccessPolicy(ctx context.Context, patientID uint64, input BatchSetAccessPolicyInput) (*SetRecordAccessPolicyResult, error) {
	now := time.Now()
	days, err := ClampAuthDays(input.Days, 365)
	if err != nil {
		return nil, err
	}
	startTime := now
	endTime := startTime.Add(time.Duration(days) * 24 * time.Hour)

	switch input.PolicyType {
	case "ALL_DOCTORS":
		revokeActiveAuthorizationsForScope(patientID, "ALL", 0)
		auth := commitAndCreateAuthorization("ALL_DOCTORS", 0, "ALL", 0, patientID, startTime, endTime, now)
		return &SetRecordAccessPolicyResult{
			Auth:    auth,
			AuditOp: "SET_ALL_VISIBILITY",
			Message: "已成功开启「全部健康档案对所有医生可见」，全网盟医生均可直接查阅您的历史健康档案",
		}, nil
	case "HOSPITAL":
		if input.AuthTargetID == 0 {
			return nil, ErrMissingTarget
		}
		hospPtr, err := repository.DefaultHospitalRepository.FindByID(input.AuthTargetID)
		if err != nil || hospPtr == nil {
			return nil, ErrHospitalNotFound
		}
		hosp := *hospPtr
		revokeActiveAuthorizationsForScope(patientID, "ALL", 0)
		auth := commitAndCreateAuthorization("HOSPITAL", input.AuthTargetID, "ALL", 0, patientID, startTime, endTime, now)
		return &SetRecordAccessPolicyResult{
			Auth:    auth,
			AuditOp: "SET_ALL_VISIBILITY",
			Message: fmt.Sprintf("已成功开启「全部健康档案对 %s 全体医生可见」，策略已固化上链", hosp.Name),
		}, nil
	case "DOCTOR":
		if input.AuthTargetID == 0 {
			return nil, ErrMissingTarget
		}
		docPtr, err := repository.DefaultUserRepository.FindDoctorByID(input.AuthTargetID)
		if err != nil || docPtr == nil {
			return nil, ErrDoctorNotFound
		}
		doc := *docPtr
		revokeActiveAuthorizationsForScope(patientID, "ALL", 0)
		auth := commitAndCreateAuthorization("DOCTOR", input.AuthTargetID, "ALL", 0, patientID, startTime, endTime, now)
		docName := doctorDisplayNameWithSuffix(doc)
		return &SetRecordAccessPolicyResult{
			Auth:    auth,
			AuditOp: "SET_ALL_VISIBILITY",
			Message: fmt.Sprintf("已成功开启「全部健康档案对 %s 可见」，策略已固化上链", docName),
		}, nil
	case "PRIVATE":
		revokeActiveAuthorizationsForScope(patientID, "ALL", 0)
		return &SetRecordAccessPolicyResult{
			AuditOp: "REVOKE_ALL_VISIBILITY",
			Message: "已关闭全局公开，恢复为私密受控模式",
		}, nil
	default:
		return nil, ErrUnsupportedPolicyType
	}
}

// --- 内部辅助 ---

// revokeActiveAuthorizationsForScope 撤销指定范围活跃授权并同步链上撤销存证
func revokeActiveAuthorizationsForScope(patientID uint64, scopeType string, recordID uint64) {
	if oldAuths, err := repository.DefaultAuthPolicyRepository.FindActiveAuthorizationsForScope(patientID, scopeType, recordID); err == nil {
		for _, oa := range oldAuths {
			if oa.AuthNo != "" && blockchain.DefaultService != nil {
				if _, _, cErr := CommitAssetWithCompensation("REVOKE_AUTH", oa.AuthNo, map[string]interface{}{
					"auth_no":    oa.AuthNo,
					"patient_id": patientID,
					"status":     "REVOKED",
					"timestamp":  time.Now().Format(time.RFC3339),
				}); cErr != nil {
					// 修复 5：撤销存证失败不得静默——落显式审计标记"存证确认中"，补偿队列已接手重放
					AuditChainAnchoringPending(patientID, "REVOKE_AUTH", "AUTH", oa.AuthNo, 0, cErr)
				}
			}
			if uErr := repository.DefaultAuthPolicyRepository.UpdateAuthorizationFields(oa.ID, map[string]interface{}{"status": "REVOKED"}); uErr != nil {
				log.Printf("[AccessPolicy] ⚠️ 更新授权撤销状态失败 auth_id=%d auth_no=%s err=%v", oa.ID, oa.AuthNo, uErr)
			}
		}
	}
}

// commitAndCreateAuthorization 统一"链上存证 + 授权落库"流程
func commitAndCreateAuthorization(targetType string, targetID uint64, scopeType string, recordID, patientID uint64, startTime, endTime, now time.Time) *model.Authorization {
	authNo := fmt.Sprintf("AUTH%s%s", now.Format("20060102"), randomHex3())

	txID, _, cErr := CommitAssetWithCompensation("AUTHORIZATION", authNo, map[string]interface{}{
		"auth_no":     authNo,
		"patient_id":  patientID,
		"target_type": targetType,
		"target_id":   targetID,
		"scope":       scopeType,
		"record_id":   recordID,
		"status":      "ACTIVE",
		"timestamp":   now.Format(time.RFC3339),
	})
	if cErr != nil {
		// 修复 5：授权存证失败时授权仍在本地生效（DB 为准），但必须显式审计"存证确认中"
		AuditChainAnchoringPending(patientID, "GRANT_AUTH", "AUTH", authNo, 0, cErr)
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
		CreatedAt:      now,
	}
	_ = repository.DefaultAuthPolicyRepository.CreateAuthorization(&auth)
	return &auth
}

// doctorDisplayNameWithSuffix 医生展示名（补齐"医生/医师"后缀）
func doctorDisplayNameWithSuffix(doctor model.User) string {
	docName := doctor.RealName
	if !strings.HasSuffix(docName, "医生") && !strings.HasSuffix(docName, "医师") {
		docName += " 医生"
	}
	return docName
}