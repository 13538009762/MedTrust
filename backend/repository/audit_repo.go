package repository

import (
	"errors"
	"time"

	"medtrust-backend/model"
)

type AuditRepository struct{}

var DefaultAuditRepository = &AuditRepository{}

func (r *AuditRepository) Create(log *model.AuditLog) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Create(log).Error
}

func (r *AuditRepository) List(userID, hospitalID uint64, opType, riskLevel string, limit, offset int) ([]model.AuditLog, int64, error) {
	if DB == nil {
		return nil, 0, errors.New("database not initialized")
	}
	query := DB.Model(&model.AuditLog{})
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if hospitalID > 0 {
		query = query.Where("hospital_id = ?", hospitalID)
	}
	if opType != "" {
		query = query.Where("operation_type = ?", opType)
	}
	if riskLevel != "" {
		query = query.Where("risk_level = ?", riskLevel)
	}

	var total int64
	_ = query.Count(&total)

	if limit <= 0 {
		limit = 50
	}
	var logs []model.AuditLog
	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error
	return logs, total, err
}

func (r *AuditRepository) CountRecentAccess(doctorID uint64, duration time.Duration) int64 {
	if DB == nil {
		return 0
	}
	var count int64
	since := time.Now().Add(-duration)
	DB.Model(&model.AuditLog{}).
		Where("user_id = ? AND created_at >= ?", doctorID, since).
		Count(&count)
	return count
}

func (r *AuditRepository) CountDistinctPatients(doctorID uint64, duration time.Duration) int64 {
	if DB == nil {
		return 0
	}
	var count int64
	since := time.Now().Add(-duration)
	DB.Table("audit_logs").
		Joins("JOIN medical_records ON audit_logs.target_id = CAST(medical_records.id AS CHAR)").
		Where("audit_logs.user_id = ? AND audit_logs.created_at >= ?", doctorID, since).
		Distinct("medical_records.patient_id").
		Count(&count)
	return count
}

func (r *AuditRepository) CountHistoricalViolations(doctorID uint64, days int) int64 {
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

func (r *AuditRepository) GetDashboardStats() (*model.DashboardStatsDTO, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}

	stats := &model.DashboardStatsDTO{
		OrgStats:         make(map[string]int64),
		RiskDistribution: make(map[string]int64),
	}

	// 跨院共享与调阅总数
	var accessCount int64
	DB.Model(&model.AuditLog{}).Where("operation_type IN ('ACCESS', 'ACCESS_EVALUATE', 'BREAK_GLASS') AND result = 'SUCCESS'").Count(&accessCount)
	stats.TotalSharedCount = accessCount

	// 高危拦截
	var highRiskCount int64
	DB.Model(&model.AuditLog{}).Where("risk_level = 'HIGH' OR result = 'INTERCEPTED'").Count(&highRiskCount)
	stats.HighRiskAlerts = highRiskCount

	// 待审急诊事件
	var pendingEmg int64
	DB.Model(&model.EmergencyAccessEvent{}).Where("audit_status = 'PENDING_AUDIT'").Count(&pendingEmg)
	stats.PendingEmergencies = pendingEmg

	// 风险分布
	type riskCount struct {
		RiskLevel string
		Count     int64
	}
	var rcList []riskCount
	DB.Model(&model.AuditLog{}).Select("risk_level, count(*) as count").Group("risk_level").Scan(&rcList)
	for _, rc := range rcList {
		stats.RiskDistribution[rc.RiskLevel] = rc.Count
	}

	return stats, nil
}
