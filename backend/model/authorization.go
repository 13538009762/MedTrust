package model

import (
	"time"
)

// Authorization 6. 患者自主授权策略表 (authorizations)
type Authorization struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	AuthNo         string    `gorm:"size:64;uniqueIndex" json:"auth_no"`
	PatientID      uint64    `gorm:"index:idx_patient_target" json:"patient_id"`
	AuthTargetType string    `gorm:"size:20;index:idx_patient_target" json:"auth_target_type"` // DOCTOR, HOSPITAL, ALL_DOCTORS
	AuthTargetID   uint64    `gorm:"index:idx_patient_target" json:"auth_target_id"`
	ScopeType      string    `gorm:"size:20" json:"scope_type"` // ALL, SINGLE
	RecordID       uint64    `gorm:"default:0" json:"record_id"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	Status         string    `gorm:"size:32;default:'ACTIVE';index:idx_patient_target" json:"status"` // ACTIVE, REVOKED, REVOKE_PENDING, FABRIC_REVOKED
	FabricTxID     string    `gorm:"size:128;default:''" json:"fabric_tx_id"`
	RevokeTxID     string    `gorm:"size:128;default:''" json:"revoke_tx_id,omitempty"`
	RevokeError    string    `gorm:"size:255;default:''" json:"revoke_error,omitempty"`
	OperatorID     uint64    `gorm:"default:0" json:"operator_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// 辅助展示
	TargetName         string `gorm:"-" json:"target_name,omitempty"`
	RecordNo           string `gorm:"-" json:"record_no,omitempty"`
	RecordHospitalName string `gorm:"-" json:"record_hospital_name,omitempty"`
	RecordDiagnosis    string `gorm:"-" json:"record_diagnosis,omitempty"`
	RecordDepartment   string `gorm:"-" json:"record_department,omitempty"`
}

func (Authorization) TableName() string {
	return "authorizations"
}

// AccessRequest 7. 跨机构常规访问申请与风险决策表 (access_requests)
type AccessRequest struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	RequestNo        string    `gorm:"size:64;uniqueIndex" json:"request_no"`
	DoctorID         uint64    `gorm:"index:idx_doctor_patient" json:"doctor_id"`
	PatientID        uint64    `gorm:"index:idx_doctor_patient" json:"patient_id"`
	RecordID         uint64    `json:"record_id"`
	SourceHospitalID uint64    `json:"source_hospital_id"`
	TargetHospitalID uint64    `json:"target_hospital_id"`
	Purpose          string    `gorm:"size:255" json:"purpose"`
	RiskScore        int       `gorm:"default:0" json:"risk_score"`
	RiskLevel        string    `gorm:"size:20;default:'LOW'" json:"risk_level"` // LOW, MEDIUM, HIGH
	RiskDetail       string    `gorm:"size:512;default:''" json:"risk_detail"`
	Decision         string    `gorm:"size:20" json:"decision"`                 // ALLOWED, PENDING_CONFIRM, REJECTED
	Status           string    `gorm:"size:20;default:'PENDING'" json:"status"` // PENDING, APPROVED, REJECTED
	ScopeType        string    `gorm:"size:20;default:'SINGLE'" json:"scope_type"`
	Days             int       `gorm:"default:7" json:"days"`
	ExpireAt         time.Time `gorm:"index" json:"expire_at"`
	CreatedAt        time.Time `json:"created_at"`

	// 辅助展示
	DoctorName          string `gorm:"-" json:"doctor_name,omitempty"`
	DoctorHospitalName  string `gorm:"-" json:"doctor_hospital_name,omitempty"`
	DoctorTitle         string `gorm:"-" json:"doctor_title,omitempty"`
	RecordNo            string `gorm:"-" json:"record_no,omitempty"`
	PatientName         string `gorm:"-" json:"patient_name,omitempty"`
	RecordDiagnosis     string `gorm:"-" json:"record_diagnosis,omitempty"`
	RecordDepartment    string `gorm:"-" json:"record_department,omitempty"`
	RecordHospitalName  string `gorm:"-" json:"record_hospital_name,omitempty"`
	RecordEncounterType string `gorm:"-" json:"record_encounter_type,omitempty"`
	RecordCreatedAt     string `gorm:"-" json:"record_created_at,omitempty"`
}

func (AccessRequest) TableName() string {
	return "access_requests"
}
