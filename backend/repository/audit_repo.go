package repository

import (
	"errors"
	"fmt"
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
		Joins("JOIN medical_records ON audit_logs.target_id = medical_records.record_no").
		Where("audit_logs.user_id = ? AND audit_logs.operation_type IN ('ACCESS', 'ACCESS_EVALUATE') AND audit_logs.result IN ('SUCCESS', 'ALLOWED') AND audit_logs.created_at >= ?", doctorID, since).
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
	DB.Model(&model.AuditLog{}).Where("operation_type IN ('ACCESS', 'ACCESS_EVALUATE', 'BREAK_GLASS') AND result IN ('SUCCESS', 'ALLOWED')").Count(&accessCount)
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

type HospStat struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type RiskStat struct {
	RiskLevel string `json:"risk_level"`
	Count     int64  `json:"count"`
}

type DayTrend struct {
	Date           string `json:"date"`
	FlowCount      int64  `json:"flow_count"`
	InterceptCount int64  `json:"intercept_count"`
}

type FlowItem struct {
	Source string `json:"source" gorm:"column:src"`
	Target string `json:"target" gorm:"column:tgt"`
	Value  int64  `json:"value"  gorm:"column:cnt"`
}

type PatientAuditFilter struct {
	PatientID     uint64
	UserNo        string
	RecordTargets []string
	AuthNos       []string
	EventNos      []string
	RequestNos    []string
}

func (r *AuditRepository) CountTotal() int64 {
	if DB == nil {
		return 0
	}
	var count int64
	DB.Model(&model.AuditLog{}).Count(&count)
	return count
}

func (r *AuditRepository) CountIntercepted() int64 {
	if DB == nil {
		return 0
	}
	var count int64
	DB.Model(&model.AuditLog{}).Where("result IN ('INTERCEPTED', 'FAILED')").Count(&count)
	return count
}

func (r *AuditRepository) GetHospStats() []HospStat {
	if DB == nil {
		return nil
	}
	var hospStats []HospStat
	DB.Raw(`
		SELECT h.name, COUNT(r.id) as count 
		FROM hospitals h 
		LEFT JOIN medical_records r ON h.id = r.hospital_id 
		GROUP BY h.id, h.name
	`).Scan(&hospStats)
	return hospStats
}

func (r *AuditRepository) GetRiskStats() []RiskStat {
	if DB == nil {
		return nil
	}
	var riskStats []RiskStat
	DB.Raw(`SELECT risk_level, COUNT(id) as count FROM audit_logs GROUP BY risk_level`).Scan(&riskStats)
	return riskStats
}

func (r *AuditRepository) GetThroughputTrends(days int) []DayTrend {
	if DB == nil || days <= 0 {
		return nil
	}
	now := time.Now()
	loc := now.Location()
	startOfRange := time.Date(now.Year(), now.Month(), now.Day()-(days-1), 0, 0, 0, 0, loc)
	endOfRange := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)

	type dayRow struct {
		DayKey         string `gorm:"column:day_key"`
		FlowCount      int64  `gorm:"column:flow_cnt"`
		InterceptCount int64  `gorm:"column:int_cnt"`
	}
	var rows []dayRow
	DB.Raw(`
		SELECT DATE_FORMAT(created_at, '%Y-%m-%d') AS day_key,
			COUNT(*) AS flow_cnt,
			SUM(CASE WHEN result IN ('INTERCEPTED', 'FAILED') THEN 1 ELSE 0 END) AS int_cnt
		FROM audit_logs
		WHERE created_at >= ? AND created_at < ?
		GROUP BY day_key
	`, startOfRange, endOfRange).Scan(&rows)

	byDay := make(map[string]dayRow, len(rows))
	for _, row := range rows {
		byDay[row.DayKey] = row
	}

	trendList := make([]DayTrend, 0, days)
	for i := days - 1; i >= 0; i-- {
		dayTime := time.Date(now.Year(), now.Month(), now.Day()-i, 0, 0, 0, 0, loc)
		dayKey := dayTime.Format("2006-01-02")
		row := byDay[dayKey]
		trendList = append(trendList, DayTrend{
			Date:           dayTime.Format("01-02"),
			FlowCount:      row.FlowCount,
			InterceptCount: row.InterceptCount,
		})
	}
	return trendList
}

func (r *AuditRepository) GetAccessHeatmapMatrix() [][3]int {
	if DB == nil {
		return nil
	}
	type HeatItem struct {
		Weekday int `gorm:"column:wd"`
		Hour    int `gorm:"column:hr"`
		Count   int `gorm:"column:cnt"`
	}
	var dbItems []HeatItem
	DB.Raw(`
		SELECT WEEKDAY(created_at) AS wd, HOUR(created_at) AS hr, COUNT(*) AS cnt
		FROM audit_logs
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
		GROUP BY wd, hr
	`).Scan(&dbItems)

	matrix := make(map[string]int, 7*24)
	for _, item := range dbItems {
		key := fmt.Sprintf("%d-%d", item.Hour, item.Weekday)
		matrix[key] += item.Count
	}

	result := make([][3]int, 0, 7*24)
	for wd := 0; wd < 7; wd++ {
		for hr := 0; hr < 24; hr++ {
			key := fmt.Sprintf("%d-%d", hr, wd)
			result = append(result, [3]int{hr, wd, matrix[key]})
		}
	}
	return result
}

func (r *AuditRepository) GetHospitalFlows() []FlowItem {
	if DB == nil {
		return nil
	}
	flows := make([]FlowItem, 0)
	DB.Raw(`
		SELECT 
			sh.name AS src,
			th.name AS tgt,
			COUNT(*) AS cnt
		FROM access_requests ar
		JOIN hospitals sh ON ar.source_hospital_id = sh.id
		JOIN hospitals th ON ar.target_hospital_id = th.id
		GROUP BY sh.name, th.name
	`).Scan(&flows)
	return flows
}

func (r *AuditRepository) GetAuthFunnelStats() (total, handled, approved, active int64) {
	if DB == nil {
		return 0, 0, 0, 0
	}
	DB.Model(&model.AccessRequest{}).Count(&total)
	DB.Model(&model.AccessRequest{}).Where("status IN ('APPROVED', 'REJECTED')").Count(&handled)
	DB.Model(&model.AccessRequest{}).Where("status = 'APPROVED'").Count(&approved)
	DB.Model(&model.Authorization{}).Where("status = 'ACTIVE'").Count(&active)
	return total, handled, approved, active
}

func (r *AuditRepository) ListBlockchainTransactions(businessType, txID, status string, offset, limit int) ([]model.BlockchainTransaction, int64, error) {
	if DB == nil {
		return nil, 0, errors.New("database not initialized")
	}
	query := DB.Model(&model.BlockchainTransaction{})
	if businessType != "" {
		query = query.Where("business_type = ?", businessType)
	}
	if txID != "" {
		query = query.Where("tx_id LIKE ?", "%"+txID+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var list []model.BlockchainTransaction
	err := query.Order("id desc").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

func (r *AuditRepository) ListAuditLogsWithFilter(opType, riskLevel string, filter *PatientAuditFilter, limit int) ([]model.AuditLog, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	query := DB.Model(&model.AuditLog{})
	if opType != "" {
		query = query.Where("operation_type = ?", opType)
	}
	if riskLevel != "" {
		query = query.Where("risk_level = ?", riskLevel)
	}
	if filter != nil {
		query = query.Where("user_id = ? OR (target_type = 'RECORD' AND target_id IN (?)) OR (target_type = 'PATIENT' AND (target_id = ? OR target_id = ?)) OR (target_type = 'AUTH' AND target_id IN (?)) OR (target_type = 'EVENT' AND target_id IN (?)) OR (target_type = 'REQUEST' AND target_id IN (?))",
			filter.PatientID, filter.RecordTargets, filter.UserNo, fmt.Sprintf("PATIENT_%d", filter.PatientID), filter.AuthNos, filter.EventNos, filter.RequestNos)
	}
	if limit <= 0 {
		limit = 100
	}
	var list []model.AuditLog
	err := query.Order("id desc").Limit(limit).Find(&list).Error
	return list, err
}

// RecentAuditAccessCount 查询指定用户在窗口期内的有效访问次数（带患者维度 JOIN，替代服务层直接操作 audit_logs 表）
func (r *AuditRepository) RecentAuditAccessCount(doctorID, patientID uint64, duration time.Duration) int64 {
	if DB == nil {
		return 0
	}
	var count int64
	since := time.Now().Add(-duration)
	DB.Table("audit_logs").
		Joins("JOIN medical_records ON audit_logs.target_id = medical_records.record_no").
		Where("audit_logs.user_id = ? AND medical_records.patient_id = ? AND audit_logs.operation_type IN ('ACCESS', 'ACCESS_EVALUATE') AND audit_logs.result IN ('SUCCESS', 'ALLOWED') AND audit_logs.created_at >= ?", doctorID, patientID, since).
		Count(&count)
	return count
}

// DistinctPatientsAccessedByDoctor 查询指定医生在窗口期内访问的不同患者数（替代服务层直接操作 audit_logs 表）
func (r *AuditRepository) DistinctPatientsAccessedByDoctor(doctorID uint64, duration time.Duration) int64 {
	if DB == nil {
		return 0
	}
	var count int64
	since := time.Now().Add(-duration)
	DB.Table("audit_logs").
		Joins("JOIN medical_records ON audit_logs.target_id = medical_records.record_no").
		Where("audit_logs.user_id = ? AND audit_logs.operation_type IN ('ACCESS', 'ACCESS_EVALUATE') AND audit_logs.result IN ('SUCCESS', 'ALLOWED') AND audit_logs.created_at >= ?", doctorID, since).
		Distinct("medical_records.patient_id").
		Count(&count)
	return count
}

// CreateRiskRecord 创建风控记录 WP-15
func (r *AuditRepository) CreateRiskRecord(riskRec *model.RiskRecord) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Create(riskRec).Error
}

// UpdateFabricTxID WP-15: 回填审计日志的 fabric_tx_id 字段
func (r *AuditRepository) UpdateFabricTxID(logID uint64, txID string) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Model(&model.AuditLog{}).Where("id = ?", logID).Update("fabric_tx_id", txID).Error
}

// ExpirePendingAccessRequests WP-15: 自动标记超期未审批调阅申请为 EXPIRED（供清理服务调用）
func (r *AuditRepository) ExpirePendingAccessRequests(now time.Time) (int64, error) {
	if DB == nil {
		return 0, errors.New("database not initialized")
	}
	res := DB.Model(&model.AccessRequest{}).
		Where("status = ? AND expire_at > ? AND expire_at < ?", "PENDING", time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), now).
		Updates(map[string]interface{}{
			"status":   "EXPIRED",
			"decision": "EXPIRED",
		})
	return res.RowsAffected, res.Error
}

// GetActiveMedicalFileCIDs WP-15: 查询所有非空 ipfs_cid（供清理孤儿文件用）
func (r *AuditRepository) GetActiveMedicalFileCIDs() ([]string, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var cids []string
	err := DB.Model(&model.MedicalFile{}).Where("ipfs_cid != ''").Pluck("ipfs_cid", &cids).Error
	return cids, err
}
