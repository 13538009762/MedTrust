package repository

import (
	"errors"
	"strconv"
	"time"

	"medtrust-backend/model"
)

type AuthPolicyRepository struct{}

var DefaultAuthPolicyRepository = &AuthPolicyRepository{}

func (r *AuthPolicyRepository) CreateAuthorization(auth *model.Authorization) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Create(auth).Error
}

func (r *AuthPolicyRepository) FindAuthorizationByID(id uint64) (*model.Authorization, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var auth model.Authorization
	if err := DB.First(&auth, id).Error; err != nil {
		return nil, err
	}
	return &auth, nil
}

func (r *AuthPolicyRepository) FindAuthorizationByNo(authNo string) (*model.Authorization, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var auth model.Authorization
	if err := DB.Where("auth_no = ?", authNo).First(&auth).Error; err != nil {
		return nil, err
	}
	return &auth, nil
}

func (r *AuthPolicyRepository) ListAuthorizations(patientID uint64, status string) ([]model.Authorization, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	query := DB.Model(&model.Authorization{})
	if patientID > 0 {
		query = query.Where("patient_id = ?", patientID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var list []model.Authorization
	err := query.Order("created_at DESC, id DESC").Find(&list).Error
	return list, err
}

// ListAuthorizationsForDoctor 医生视角：仅返回授权目标为该医生本人或其所在医疗机构的授权策略 (行级租户过滤)
func (r *AuthPolicyRepository) ListAuthorizationsForDoctor(doctorID, hospitalID uint64, status string) ([]model.Authorization, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	query := DB.Model(&model.Authorization{}).
		Where("(auth_target_type = 'DOCTOR' AND auth_target_id = ?) OR (auth_target_type = 'HOSPITAL' AND auth_target_id = ?) OR auth_target_type = 'ALL_DOCTORS'",
			doctorID, hospitalID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var list []model.Authorization
	err := query.Order("created_at DESC, id DESC").Find(&list).Error
	return list, err
}

func (r *AuthPolicyRepository) UpdateAuthorizationStatus(id uint64, status, revokeTxID, revokeError string) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if revokeTxID != "" {
		updates["revoke_tx_id"] = revokeTxID
	}
	if revokeError != "" {
		updates["revoke_error"] = revokeError
	}
	return DB.Model(&model.Authorization{}).Where("id = ?", id).Updates(updates).Error
}

func (r *AuthPolicyRepository) FindActiveAuthorization(patientID uint64, targetType string, targetID uint64, recordID uint64) (*model.Authorization, bool) {
	if DB == nil {
		return nil, false
	}
	var auth model.Authorization
	now := time.Now()
	err := DB.Where("patient_id = ? AND status = 'ACTIVE' AND ? BETWEEN start_time AND end_time", patientID, now).
		Where("((auth_target_type = ? AND auth_target_id = ?) OR auth_target_type = 'ALL_DOCTORS')", targetType, targetID).
		Where("(scope_type = 'ALL' OR (scope_type = 'SINGLE' AND record_id = ?))", recordID).
		First(&auth).Error
	if err == nil && auth.ID > 0 {
		return &auth, true
	}
	return nil, false
}

func (r *AuthPolicyRepository) CreateAccessRequest(req *model.AccessRequest) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Create(req).Error
}

func (r *AuthPolicyRepository) FindAccessRequestByNo(requestNo string) (*model.AccessRequest, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var req model.AccessRequest
	if err := DB.Where("request_no = ?", requestNo).First(&req).Error; err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *AuthPolicyRepository) ListAccessRequests(doctorID, patientID uint64, status string) ([]model.AccessRequest, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	query := DB.Model(&model.AccessRequest{})
	if doctorID > 0 {
		query = query.Where("doctor_id = ?", doctorID)
	}
	if patientID > 0 {
		query = query.Where("patient_id = ?", patientID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var list []model.AccessRequest
	err := query.Order("created_at DESC").Find(&list).Error
	return list, err
}

func (r *AuthPolicyRepository) UpdateAccessRequestDecision(requestNo string, decision, status string) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Model(&model.AccessRequest{}).Where("request_no = ?", requestNo).Updates(map[string]interface{}{
		"decision": decision,
		"status":   status,
	}).Error
}

func (r *AuthPolicyRepository) FindPatientAccessRequest(patientID uint64, reqNoOrID string) (*model.AccessRequest, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var req model.AccessRequest
	var err error
	if id, parseErr := strconv.ParseUint(reqNoOrID, 10, 64); parseErr == nil && id > 0 {
		err = DB.Where("(id = ? OR request_no = ?) AND patient_id = ?", id, reqNoOrID, patientID).First(&req).Error
	} else {
		err = DB.Where("request_no = ? AND patient_id = ?", reqNoOrID, patientID).First(&req).Error
	}
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *AuthPolicyRepository) SaveAccessRequest(req *model.AccessRequest) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Save(req).Error
}

func (r *AuthPolicyRepository) FindPendingRequestsByPatient(patientID uint64, limit int) ([]model.AccessRequest, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	if limit <= 0 {
		limit = 10
	}
	var reqs []model.AccessRequest
	err := DB.Where("patient_id = ? AND status = 'PENDING'", patientID).Order("id desc").Limit(limit).Find(&reqs).Error
	return reqs, err
}

func (r *AuthPolicyRepository) FindPendingRequestsByIDs(patientID uint64, ids []uint64) ([]model.AccessRequest, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	if len(ids) == 0 {
		return nil, nil
	}
	var reqs []model.AccessRequest
	err := DB.Where("patient_id = ? AND id IN (?) AND status = 'PENDING'", patientID, ids).Order("id desc").Find(&reqs).Error
	return reqs, err
}

func (r *AuthPolicyRepository) GetPatientAuthNos(patientID uint64) []string {
	if DB == nil {
		return nil
	}
	var authNos []string
	DB.Model(&model.Authorization{}).Where("patient_id = ?", patientID).Pluck("auth_no", &authNos)
	return authNos
}

func (r *AuthPolicyRepository) GetPatientRequestNos(patientID uint64) []string {
	if DB == nil {
		return nil
	}
	var requestNos []string
	DB.Model(&model.AccessRequest{}).Where("patient_id = ?", patientID).Pluck("request_no", &requestNos)
	return requestNos
}

func (r *AuthPolicyRepository) ApprovePendingRequests(recordID, doctorID uint64) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Model(&model.AccessRequest{}).
		Where("record_id = ? AND doctor_id = ? AND status = 'PENDING'", recordID, doctorID).
		Updates(map[string]interface{}{"status": "APPROVED", "decision": "ALLOWED"}).Error
}

func (r *AuthPolicyRepository) FindPatientAuthorization(id uint64, patientID uint64) (*model.Authorization, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var auth model.Authorization
	if err := DB.Where("id = ? AND patient_id = ?", id, patientID).First(&auth).Error; err != nil {
		return nil, err
	}
	return &auth, nil
}

func (r *AuthPolicyRepository) RevokeConflictingAuthorizations(patientID uint64, scopeType string, recordID uint64, targetType string, targetID uint64) ([]string, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	q := DB.Where("patient_id = ? AND scope_type = ? AND status = 'ACTIVE'", patientID, scopeType)
	if scopeType == "SINGLE" {
		q = q.Where("record_id = ?", recordID)
	}
	var oldAuths []model.Authorization
	q.Find(&oldAuths)

	var supersededNos []string
	for _, oa := range oldAuths {
		if oa.AuthTargetType == targetType && oa.AuthTargetID == targetID {
			DB.Model(&model.Authorization{}).Where("id = ?", oa.ID).Update("status", "REVOKED")
			supersededNos = append(supersededNos, oa.AuthNo)
		}
	}
	return supersededNos, nil
}

func (r *AuthPolicyRepository) TryClaimRevoking(id uint64, patientID uint64, deadline time.Time) (bool, error) {
	if DB == nil {
		return false, errors.New("database not initialized")
	}
	res := DB.Model(&model.Authorization{}).
		Where("id = ? AND patient_id = ? AND (status = 'ACTIVE' OR (status = 'REVOKING' AND updated_at < ?))", id, patientID, deadline).
		Updates(map[string]interface{}{
			"status":     "REVOKING",
			"updated_at": time.Now(),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *AuthPolicyRepository) TryClaimRetryRevoking(id uint64, patientID uint64, deadline time.Time) (bool, error) {
	if DB == nil {
		return false, errors.New("database not initialized")
	}
	res := DB.Model(&model.Authorization{}).
		Where("id = ? AND patient_id = ? AND (status IN ('REVOKE_FAILED', 'NEEDS_RECONCILIATION') OR (status = 'REVOKE_PENDING' AND updated_at < ?))", id, patientID, deadline).
		Updates(map[string]interface{}{
			"status":     "REVOKE_PENDING",
			"updated_at": time.Now(),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *AuthPolicyRepository) UpdateAuthorizationFields(id uint64, updates map[string]interface{}) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Model(&model.Authorization{}).Where("id = ?", id).Updates(updates).Error
}

func (r *AuthPolicyRepository) FindActiveAuthorizationsByPatient(patientID uint64) ([]model.Authorization, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var activeAuths []model.Authorization
	err := DB.Where("patient_id = ? AND status = 'ACTIVE'", patientID).Find(&activeAuths).Error
	return activeAuths, err
}

func (r *AuthPolicyRepository) FindFailedRevokeAuthorizationsByPatient(patientID uint64) ([]model.Authorization, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var failedAuths []model.Authorization
	err := DB.Where("patient_id = ? AND status IN ('REVOKE_FAILED', 'REVOKE_PENDING', 'NEEDS_RECONCILIATION')", patientID).Find(&failedAuths).Error
	return failedAuths, err
}

func (r *AuthPolicyRepository) FindActiveAuthorizationsForScope(patientID uint64, scopeType string, recordID uint64) ([]model.Authorization, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	q := DB.Where("patient_id = ? AND scope_type = ? AND status = 'ACTIVE'", patientID, scopeType)
	if scopeType == "SINGLE" {
		q = q.Where("record_id = ?", recordID)
	}
	var oldAuths []model.Authorization
	err := q.Find(&oldAuths).Error
	return oldAuths, err
}

// IsAuthorizationNewerThan 比较两条策略的生效顺序：优先按 CreatedAt，同秒或零值时按自增 ID 严格定序，
// 用于判定后续患者审批/密钥解锁是否覆盖先前的 PRIVATE 锁定，以及后续 PRIVATE 锁定是否覆盖先前的全局授权。
func IsAuthorizationNewerThan(a, priv model.Authorization) bool {
	if !a.CreatedAt.IsZero() && !priv.CreatedAt.IsZero() && !a.CreatedAt.Equal(priv.CreatedAt) {
		return a.CreatedAt.After(priv.CreatedAt)
	}
	return a.ID > priv.ID
}

// EvaluateConsentInAuthorizations 统一判定患者活跃授权列表在 now 时刻是否对指定 (doctorID, hospitalID, recordID) 授予有效同意。
// 规则：
//   1. 仅统计处于有效期 (!StartTime.After(now) && EndTime.After(now)) 的 ACTIVE 记录；
//   2. 若该病历存在有效 PRIVATE 锁定记录，只有比最新 PRIVATE 记录更晚授予的显式同意（如患者审批或密钥解锁）才有效；
//   3. SINGLE 范围匹配优先于 ALL 全局范围。
func EvaluateConsentInAuthorizations(auths []model.Authorization, doctorID, hospitalID, recordID uint64, now time.Time) (hasConsent bool, isAllDoctors bool) {
	var active []model.Authorization
	var latestPriv *model.Authorization

	for _, a := range auths {
		if a.Status != "" && a.Status != "ACTIVE" {
			continue
		}
		if a.StartTime.After(now) || !a.EndTime.After(now) {
			continue
		}
		active = append(active, a)
		if a.ScopeType == "SINGLE" && a.RecordID == recordID && a.AuthTargetType == "PRIVATE" {
			copyA := a
			if latestPriv == nil || IsAuthorizationNewerThan(copyA, *latestPriv) {
				latestPriv = &copyA
			}
		}
	}

	matchesTarget := func(a model.Authorization) bool {
		switch a.AuthTargetType {
		case "ALL_DOCTORS":
			return true
		case "DOCTOR":
			return a.AuthTargetID == doctorID
		case "HOSPITAL":
			return hospitalID > 0 && a.AuthTargetID == hospitalID
		default:
			return false
		}
	}

	// 1. 优先检查单份病历显式授权 (ScopeType == SINGLE)
	for _, a := range active {
		if a.ScopeType != "SINGLE" || a.RecordID != recordID || a.AuthTargetType == "PRIVATE" {
			continue
		}
		if latestPriv != nil && !IsAuthorizationNewerThan(a, *latestPriv) {
			continue
		}
		if matchesTarget(a) {
			hasConsent = true
			if a.AuthTargetType == "ALL_DOCTORS" {
				isAllDoctors = true
			}
		}
	}
	if hasConsent {
		return true, isAllDoctors
	}

	// 2. 再检查全局范围授权 (ScopeType == ALL)，同样受更新时刻的 PRIVATE 标记约束
	for _, a := range active {
		if a.ScopeType != "ALL" {
			continue
		}
		if latestPriv != nil && !IsAuthorizationNewerThan(a, *latestPriv) {
			continue
		}
		if matchesTarget(a) {
			hasConsent = true
			if a.AuthTargetType == "ALL_DOCTORS" {
				isAllDoctors = true
			}
		}
	}
	return hasConsent, isAllDoctors
}

// CountActiveConsentAuthorizations WP-15: 统计患者在指定时间点对 (医生/医院/全体医生) 且覆盖指定病历的有效授权数（严格遵循 PRIVATE 优先级）
func (r *AuthPolicyRepository) CountActiveConsentAuthorizations(patientID, doctorID, hospitalID, recordID uint64, now time.Time) int64 {
	if DB == nil {
		return 0
	}
	auths, err := r.ListAuthorizations(patientID, "ACTIVE")
	if err != nil {
		return 0
	}
	if ok, _ := EvaluateConsentInAuthorizations(auths, doctorID, hospitalID, recordID, now); ok {
		return 1
	}
	return 0
}

func (r *AuthPolicyRepository) TryClaimRevokingActive(id, patientID uint64) (int64, error) {
	if DB == nil {
		return 0, errors.New("database not initialized")
	}
	res := DB.Model(&model.Authorization{}).
		Where("id = ? AND patient_id = ? AND status = 'ACTIVE'", id, patientID).
		Updates(map[string]interface{}{
			"status":       "REVOKE_PENDING",
			"revoke_error": "",
			"operator_id":  patientID,
			"updated_at":   time.Now(),
		})
	return res.RowsAffected, res.Error
}

func (r *AuthPolicyRepository) TryClaimRetryRevokingFailed(id, patientID uint64) (int64, error) {
	if DB == nil {
		return 0, errors.New("database not initialized")
	}
	res := DB.Model(&model.Authorization{}).
		Where("id = ? AND patient_id = ? AND status IN ('REVOKE_FAILED', 'REVOKE_PENDING')", id, patientID).
		Updates(map[string]interface{}{
			"status":      "REVOKE_PENDING",
			"operator_id": patientID,
			"updated_at":  time.Now(),
		})
	return res.RowsAffected, res.Error
}

// FindAuthorizationsByMultiField WP-15: 多字段查询授权记录（含 WHERE + Updates）
func (r *AuthPolicyRepository) FindAuthorizationsByMultiField(where string, args ...interface{}) ([]model.Authorization, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var list []model.Authorization
	err := DB.Where(where, args...).Find(&list).Error
	return list, err
}

// UpdateAuthorizationByMultiField 通过多条件更新授权状态
func (r *AuthPolicyRepository) UpdateAuthorizationByMultiField(where string, updates map[string]interface{}, args ...interface{}) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Model(&model.Authorization{}).Where(where, args...).Updates(updates).Error
}

// RevokeAuthorizationByPattern 根据 LIKE 模式撤销授权（紧急场景 revoke EMG 授权用）
func (r *AuthPolicyRepository) RevokeAuthorizationByPattern(where string, updates map[string]interface{}, args ...interface{}) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Model(&model.Authorization{}).Where(where, args...).Updates(updates).Error
}

// ListPrivAuthsByTime 查询指定时间窗口内活跃的 PRIVATE 范围授权
func (r *AuthPolicyRepository) ListPrivAuthsByTime(startTime, endTime time.Time) ([]model.Authorization, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var list []model.Authorization
	err := DB.Where("scope_type = 'SINGLE' AND auth_target_type = 'PRIVATE' AND status = 'ACTIVE' AND start_time <= ? AND end_time > ?", startTime, endTime).
		Order("created_at DESC, id DESC").Find(&list).Error
	return list, err
}

// ListPatientAuthsAtTime 查询指定患者在给定时间点的有效授权
func (r *AuthPolicyRepository) ListPatientAuthsAtTime(patientID uint64, startTime, endTime time.Time) ([]model.Authorization, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var list []model.Authorization
	err := DB.Where("patient_id = ? AND status = 'ACTIVE' AND start_time <= ? AND end_time > ?", patientID, startTime, endTime).
		Order("created_at DESC, id DESC").Find(&list).Error
	return list, err
}

// ListRecentActiveEmgs 查询最近一段时间内有效的紧急事件 (保留以兼容)
func (r *AuthPolicyRepository) ListRecentActiveEmgs(doctorID uint64, since time.Time) ([]model.EmergencyAccessEvent, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var emgs []model.EmergencyAccessEvent
	err := DB.Where("doctor_id = ? AND created_at >= ? AND audit_status != 'CLOSED_VIOLATION' AND patient_feedback != 'OBJECTED'", doctorID, since).Find(&emgs).Error
	return emgs, err
}
