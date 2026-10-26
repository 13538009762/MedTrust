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
	err := query.Order("created_at DESC").Find(&list).Error
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
