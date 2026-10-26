package model

import (
	"time"
)

// MedicalRecord 4. 医疗数据记录主表 (medical_records)
type MedicalRecord struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	RecordNo        string    `gorm:"size:64;uniqueIndex" json:"record_no"`
	PatientID       uint64    `gorm:"index" json:"patient_id"`
	DoctorID        uint64    `gorm:"index:idx_doctor_hosp" json:"doctor_id"`
	HospitalID      uint64    `gorm:"index:idx_doctor_hosp" json:"hospital_id"`
	DepartmentID    uint64    `gorm:"default:0" json:"department_id"`
	DataType        string    `gorm:"size:32" json:"data_type"` // EMR, REPORT, IMAGE
	Title           string    `gorm:"size:128;default:''" json:"title"`
	Diagnosis       string    `gorm:"type:text" json:"diagnosis"`
	FabricTxID      string    `gorm:"size:128;default:''" json:"fabric_tx_id"`
	BlockHeight     uint64    `gorm:"default:0" json:"block_height"`
	IntegrityStatus string    `gorm:"size:20;default:'VERIFIED'" json:"integrity_status"` // VERIFIED, TAMPERED
	CreatedAt       time.Time `json:"created_at"`

	// 临床详细扩展字段
	OnsetTime        string `gorm:"size:50;default:''" json:"onset_time"`
	Duration         string `gorm:"size:50;default:''" json:"duration"`
	Symptoms         string `gorm:"type:text" json:"symptoms"`
	Etiology         string `gorm:"type:text" json:"etiology"`
	TreatmentPlan    string `gorm:"type:text" json:"treatment_plan"`
	VitalSigns       string `gorm:"size:255;default:''" json:"vital_signs"`
	EncounterType    string `gorm:"size:32;default:'OUTPATIENT'" json:"encounter_type"`
	DepartmentName   string `gorm:"size:64;default:'综合门诊'" json:"department_name"`
	Status           string `gorm:"size:32;default:'COMPLETED'" json:"status"`
	ChiefComplaint   string `gorm:"type:text" json:"chief_complaint"`
	PresentIllness   string `gorm:"type:text" json:"present_illness"`
	InitialDiagnosis string `gorm:"type:text" json:"initial_diagnosis"`
	DiagnosticBasis  string `gorm:"type:text" json:"diagnostic_basis"`
	NeedExam         bool   `gorm:"default:false" json:"need_exam"`
	ExamItems        string `gorm:"size:255;default:''" json:"exam_items"`
	ExamReason       string `gorm:"type:text" json:"exam_reason"`
	ExamResult       string `gorm:"type:text" json:"exam_result"`
	ExamDoctor       string `gorm:"size:50;default:''" json:"exam_doctor"`
	ExamTime         string `gorm:"size:50;default:''" json:"exam_time"`

	// 链上链下数据流转状态机
	SyncStatus     string `gorm:"size:32;default:'COMPLETED';index" json:"sync_status"`
	IdempotencyKey string `gorm:"size:64;index" json:"idempotency_key,omitempty"`
	SyncError      string `gorm:"type:text" json:"sync_error,omitempty"`
	RetryCount     int    `gorm:"default:0" json:"retry_count"`

	// 辅助展示
	PatientName          string              `gorm:"-" json:"patient_name,omitempty"`
	PatientIDCard        string              `gorm:"-" json:"patient_id_card,omitempty"`
	PatientPhone         string              `gorm:"-" json:"patient_phone,omitempty"`
	DoctorName           string              `gorm:"-" json:"doctor_name,omitempty"`
	HospitalName         string              `gorm:"-" json:"hospital_name,omitempty"`
	Files                []MedicalFile       `gorm:"foreignKey:RecordID" json:"files,omitempty"`
	IsTampered           bool                `gorm:"-" json:"is_tampered"`
	Verified             bool                `gorm:"-" json:"verified"`
	ChainHash            string              `gorm:"-" json:"chain_hash,omitempty"`
	CurrentHash          string              `gorm:"-" json:"current_hash,omitempty"`
	TamperReason         string              `gorm:"-" json:"tamper_reason,omitempty"`
	HasAccess            bool                `gorm:"-" json:"has_access"`
	AccessType           string              `gorm:"-" json:"access_type,omitempty"`
	SharingScope         string              `gorm:"-" json:"sharing_scope,omitempty"`
	SharingSummary       string              `gorm:"-" json:"sharing_summary,omitempty"`
	ActiveAuthID         uint64              `gorm:"-" json:"active_auth_id,omitempty"`
	ActiveAuthTargetType string              `gorm:"-" json:"active_auth_target_type,omitempty"`
	ActiveAuthTargetID   uint64              `gorm:"-" json:"active_auth_target_id,omitempty"`
	ActiveAuthTargetName string              `gorm:"-" json:"active_auth_target_name,omitempty"`
	InfectionAlert       *InfectionRiskAlert `gorm:"-" json:"infection_alert,omitempty"`
}

func (MedicalRecord) TableName() string {
	return "medical_records"
}

// MedicalFile 5. 医疗文件与链下存储映射表 (medical_files)
type MedicalFile struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	RecordID     uint64    `gorm:"index" json:"record_id"`
	FileName     string    `gorm:"size:255" json:"file_name"`
	FileType     string    `gorm:"size:32" json:"file_type"`
	FileSize     uint64    `json:"file_size"`
	IPFSCID      string    `gorm:"column:ipfs_cid;size:128;index" json:"ipfs_cid"`
	FileHash     string    `gorm:"size:64" json:"file_hash"`
	EncryptedDEK string    `gorm:"size:255;default:''" json:"encrypted_dek"`
	IVNonce      string    `gorm:"size:64;default:''" json:"iv_nonce"`
	CreatedAt    time.Time `json:"created_at"`
}

func (MedicalFile) TableName() string {
	return "medical_files"
}

// MedicalExamOrder 12. 医技检查申请与回传报告单 (medical_exam_orders，支持检验科、影像科等流程解耦)
type MedicalExamOrder struct {
	ID             uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderNo        string     `gorm:"size:64;uniqueIndex" json:"order_no"`       // 申请单号 (ORD...)
	RecordID       uint64     `gorm:"index" json:"record_id"`                    // 关联就诊记录 ID
	PatientID      uint64     `gorm:"index" json:"patient_id"`                    // 患者 ID
	DoctorID       uint64     `gorm:"index" json:"doctor_id"`                    // 开单医生 ID
	HospitalID     uint64     `gorm:"index" json:"hospital_id"`                  // 机构 ID
	DepartmentName string     `gorm:"size:64;default:''" json:"department_name"` // 就诊科室
	ExamItem       string     `gorm:"size:128" json:"exam_item"`                 // 检查项目
	ExamReason     string     `gorm:"type:text" json:"exam_reason"`              // 检查目的与临床指征
	Status         string     `gorm:"size:32;default:'PENDING'" json:"status"`   // PENDING, PROCESSING, COMPLETED

	// 医技科室执行与回传
	TechnicianID   uint64     `gorm:"default:0" json:"technician_id"`
	TechnicianName string     `gorm:"size:64;default:''" json:"technician_name"`
	ExamResult     string     `gorm:"type:text" json:"exam_result"`              // 测量参数与所见描述
	ExamConclusion string     `gorm:"type:text" json:"exam_conclusion"`          // 报告结论
	ExecutedAt     *time.Time `json:"executed_at"`

	// 影像或报告附件 (IPFS 存证)
	ReportFileName string     `gorm:"size:255;default:''" json:"report_file_name"`
	ReportFileType string     `gorm:"size:32;default:''" json:"report_file_type"`
	IPFSCID        string     `gorm:"size:128;default:''" json:"ipfs_cid"`
	FileHash       string     `gorm:"size:64;default:''" json:"file_hash"`

	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// 辅助展示字段
	RecordNo                string `gorm:"-" json:"record_no,omitempty"`
	PatientName             string `gorm:"-" json:"patient_name,omitempty"`
	PatientIDCard           string `gorm:"-" json:"patient_id_card,omitempty"`
	PatientPhone            string `gorm:"-" json:"patient_phone,omitempty"`
	DoctorName              string `gorm:"-" json:"doctor_name,omitempty"`
	HospitalName            string `gorm:"-" json:"hospital_name,omitempty"`
	TechnicianHospitalName  string `gorm:"-" json:"technician_hospital_name,omitempty"`
	PatientChiefComplaint   string `gorm:"-" json:"patient_chief_complaint,omitempty"`
	PatientInitialDiagnosis string `gorm:"-" json:"patient_initial_diagnosis,omitempty"`
	PatientDiagnosticBasis  string `gorm:"-" json:"patient_diagnostic_basis,omitempty"`
	PatientVitalSigns       string `gorm:"-" json:"patient_vital_signs,omitempty"`
	EncounterType           string `gorm:"-" json:"encounter_type,omitempty"`
	FileID                  uint64 `gorm:"-" json:"file_id,omitempty"`
}

func (MedicalExamOrder) TableName() string {
	return "medical_exam_orders"
}
