package repository

import (
	"errors"
	"time"

	"medtrust-backend/model"
)

type EmergencyRepository struct{}

var DefaultEmergencyRepository = &EmergencyRepository{}

func (r *EmergencyRepository) CreateEvent(ev *model.EmergencyAccessEvent) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Create(ev).Error
}

func (r *EmergencyRepository) FindByEventNo(eventNo string) (*model.EmergencyAccessEvent, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var ev model.EmergencyAccessEvent
	if err := DB.Where("event_no = ?", eventNo).First(&ev).Error; err != nil {
		return nil, err
	}
	return &ev, nil
}

func (r *EmergencyRepository) FindByID(id uint64) (*model.EmergencyAccessEvent, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var ev model.EmergencyAccessEvent
	if err := DB.First(&ev, id).Error; err != nil {
		return nil, err
	}
	return &ev, nil
}

func (r *EmergencyRepository) ListEvents(doctorID, patientID uint64, auditStatus string) ([]model.EmergencyAccessEvent, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	query := DB.Model(&model.EmergencyAccessEvent{})
	if doctorID > 0 {
		query = query.Where("doctor_id = ?", doctorID)
	}
	if patientID > 0 {
		query = query.Where("patient_id = ?", patientID)
	}
	if auditStatus != "" {
		query = query.Where("audit_status = ?", auditStatus)
	}
	var list []model.EmergencyAccessEvent
	err := query.Order("created_at DESC").Find(&list).Error
	return list, err
}

func (r *EmergencyRepository) UpdatePatientFeedback(eventNo string, patientID uint64, feedback, comment string) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Model(&model.EmergencyAccessEvent{}).
		Where("event_no = ? AND patient_id = ?", eventNo, patientID).
		Updates(map[string]interface{}{
			"patient_feedback": feedback,
			"patient_comment":  comment,
			"updated_at":       time.Now(),
		}).Error
}

func (r *EmergencyRepository) UpdateAuditDecision(eventNo string, supervisorID uint64, auditStatus, comment string) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Model(&model.EmergencyAccessEvent{}).
		Where("event_no = ?", eventNo).
		Updates(map[string]interface{}{
			"audit_status":  auditStatus,
			"supervisor_id": supervisorID,
			"audit_comment": comment,
			"updated_at":    time.Now(),
		}).Error
}
