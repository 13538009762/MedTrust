package model

import (
	"time"
)

// EmergencyAccessEvent 8. 紧急访问事件单表 (emergency_access_events)
type EmergencyAccessEvent struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	EventNo          string    `gorm:"size:64;uniqueIndex" json:"event_no"`
	DoctorID         uint64    `gorm:"index:idx_doc_pat" json:"doctor_id"`
	PatientID        uint64    `gorm:"index:idx_doc_pat" json:"patient_id"`
	RecordID         uint64    `json:"record_id"`
	SourceHospitalID uint64    `json:"source_hospital_id"`
	TargetHospitalID uint64    `json:"target_hospital_id"`
	EmergencyReason  string    `gorm:"size:50" json:"emergency_reason"` // RESCUE, COMA, CRITICAL
	Description      string    `gorm:"type:text" json:"description"`
	DoctorConfirmed  int8      `gorm:"default:1" json:"doctor_confirmed"`
	PatientFeedback  string    `gorm:"size:20;default:'PENDING'" json:"patient_feedback"` // PENDING, CONFIRMED, OBJECTED
	PatientComment   string    `gorm:"size:255;default:''" json:"patient_comment"`
	AuditStatus      string    `gorm:"size:30;default:'PENDING_AUDIT';index" json:"audit_status"` // PENDING_AUDIT, CLOSED_APPROVED, CLOSED_VIOLATION
	SupervisorID     uint64    `gorm:"default:0" json:"supervisor_id"`
	AuditComment     string    `gorm:"size:255;default:''" json:"audit_comment"`
	FabricTxID       string    `gorm:"size:128;default:''" json:"fabric_tx_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	// 辅助展示
	DoctorName             string `gorm:"-" json:"doctor_name,omitempty"`
	PatientName            string `gorm:"-" json:"patient_name,omitempty"`
	RecordNo               string `gorm:"-" json:"record_no,omitempty"`
	SourceHospitalName     string `gorm:"-" json:"source_hospital_name,omitempty"`
	TargetHospitalName     string `gorm:"-" json:"target_hospital_name,omitempty"`
	RulePreScreen          string `gorm:"-" json:"rule_pre_screen,omitempty"`
	RulePreScreenDesc      string `gorm:"-" json:"rule_pre_screen_desc,omitempty"`
	AuditComparisonSummary string `gorm:"-" json:"audit_comparison_summary,omitempty"`
}

func (EmergencyAccessEvent) TableName() string {
	return "emergency_access_events"
}
