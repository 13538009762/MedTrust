package repository

import (
	"errors"
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
