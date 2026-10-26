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

func (r *EmergencyRepository) FindPendingFeedbackEventsByPatient(patientID uint64, limit int) ([]model.EmergencyAccessEvent, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	if limit <= 0 {
		limit = 10
	}
	var emgs []model.EmergencyAccessEvent
	err := DB.Where("patient_id = ? AND patient_feedback = 'PENDING'", patientID).Order("id desc").Limit(limit).Find(&emgs).Error
	return emgs, err
}

func (r *EmergencyRepository) CountTotal() int64 {
	if DB == nil {
		return 0
	}
	var count int64
	DB.Model(&model.EmergencyAccessEvent{}).Count(&count)
	return count
}

func (r *EmergencyRepository) CountPendingAudits() int64 {
	if DB == nil {
		return 0
	}
	var count int64
	DB.Model(&model.EmergencyAccessEvent{}).Where("audit_status = 'PENDING_AUDIT'").Count(&count)
	return count
}

func (r *EmergencyRepository) ListEventsWithFilter(status string) ([]model.EmergencyAccessEvent, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	query := DB.Model(&model.EmergencyAccessEvent{})
	if status != "" {
		query = query.Where("audit_status = ?", status)
	}
	var list []model.EmergencyAccessEvent
	err := query.Order("id desc").Find(&list).Error
	return list, err
}

func (r *EmergencyRepository) GetPatientEventNos(patientID uint64) []string {
	if DB == nil {
		return nil
	}
	var eventNos []string
	DB.Model(&model.EmergencyAccessEvent{}).Where("patient_id = ?", patientID).Pluck("event_no", &eventNos)
	return eventNos
}

func (r *EmergencyRepository) FindRecentActiveEvent(doctorID, patientID, recordID uint64, windowStart time.Time) (*model.EmergencyAccessEvent, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var activeEvt model.EmergencyAccessEvent
	if err := DB.Where("doctor_id = ? AND patient_id = ? AND record_id = ? AND created_at >= ? AND audit_status != 'CLOSED_VIOLATION'",
		doctorID, patientID, recordID, windowStart).First(&activeEvt).Error; err != nil {
		return nil, err
	}
	return &activeEvt, nil
}
