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

// FindRecentByDoctorAndPatient 查询某医生对某患者近窗口期内全部有效紧急访问事件 (批量破窗冷却判定的数据基础)
func (r *EmergencyRepository) FindRecentByDoctorAndPatient(doctorID, patientID uint64, windowStart time.Time) ([]model.EmergencyAccessEvent, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var list []model.EmergencyAccessEvent
	err := DB.Where("doctor_id = ? AND patient_id = ? AND created_at >= ? AND audit_status != 'CLOSED_VIOLATION'",
		doctorID, patientID, windowStart).Find(&list).Error
	return list, err
}

// CountRecentActiveBreakGlass WP-15: 统计某医生对某病历在窗口期内有效（未违规关闭、患者未异议）的紧急破窗次数
func (r *EmergencyRepository) CountRecentActiveBreakGlass(doctorID, recordID uint64, windowStart time.Time) int64 {
	if DB == nil {
		return 0
	}
	var count int64
	DB.Model(&model.EmergencyAccessEvent{}).Where(
		"doctor_id = ? AND record_id = ? AND created_at >= ? AND audit_status != 'CLOSED_VIOLATION' AND patient_feedback != 'OBJECTED'",
		doctorID, recordID, windowStart,
	).Count(&count)
	return count
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

// CountRecentBreakGlassBursts 统计某医生在窗口期内的破窗次数（用于 EvaluateRulePreScreen）
func (r *EmergencyRepository) CountRecentBreakGlassBursts(doctorID uint64, eventTime time.Time) int64 {
	if DB == nil {
		return 0
	}
	var count int64
	windowStart := eventTime.Add(-1 * time.Hour)
	DB.Model(&model.EmergencyAccessEvent{}).
		Where("doctor_id = ? AND created_at >= ? AND created_at <= ?", doctorID, windowStart, eventTime).
		Count(&count)
	return count
}

// CountHistoricalViolations 统计某医生近期被裁定违规的破窗历史次数（WP-15 风控引擎数据源收口）
func (r *EmergencyRepository) CountHistoricalViolations(doctorID uint64, days int) int64 {
	if DB == nil {
		return 0
	}
	var count int64
	since := time.Now().AddDate(0, 0, -days)
	DB.Model(&model.EmergencyAccessEvent{}).
		Where("doctor_id = ? AND audit_status = 'CLOSED_VIOLATION' AND created_at >= ?", doctorID, since).
		Count(&count)
	return count
}

// ListRecentActiveEmgs 查询某用户在窗口期内全部有效的紧急破窗事件（跨院调阅绿色通道判定，WP-15 收口）
func (r *EmergencyRepository) ListRecentActiveEmgs(userID uint64, windowStart time.Time) ([]model.EmergencyAccessEvent, error) {
	var list []model.EmergencyAccessEvent
	if DB == nil {
		return list, errors.New("database not initialized")
	}
	err := DB.Where("doctor_id = ? AND created_at >= ? AND audit_status != 'CLOSED_VIOLATION' AND patient_feedback != 'OBJECTED'", userID, windowStart).Find(&list).Error
	return list, err
}
