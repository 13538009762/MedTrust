package model

import (
	"time"
)

// User 1. 用户核心信息表 (users)
type User struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserNo       string     `gorm:"size:64;uniqueIndex" json:"user_no"`
	Username     string     `gorm:"size:50;uniqueIndex" json:"username"`
	PasswordHash string     `gorm:"size:255" json:"-"`
	RealName     string     `gorm:"size:50" json:"real_name"`
	Role         string     `gorm:"size:20" json:"role"` // doctor, patient, admin, supervisor
	HospitalID   uint64     `gorm:"default:0;index:idx_hosp_dept" json:"hospital_id"`
	DepartmentID uint64     `gorm:"default:0;index:idx_hosp_dept" json:"department_id"`
	Title        string     `gorm:"size:50;default:''" json:"title"`
	Phone        string     `gorm:"size:20;default:''" json:"phone"`
	IDCard       string     `gorm:"size:20;default:''" json:"id_card"`
	Avatar       string     `gorm:"type:longtext" json:"avatar"`
	MedicalKey   string     `gorm:"size:64;default:''" json:"-"`
	MedicalKeyHash string   `gorm:"size:128;default:''" json:"-"`
	MustChangePwd bool      `gorm:"default:false" json:"must_change_pwd"`
	FailedLoginCount int    `gorm:"default:0" json:"-"`
	LockedUntil  *time.Time `json:"locked_until,omitempty"`
	Status       string     `gorm:"size:20;default:'NORMAL'" json:"status"` // NORMAL, RESTRICTED, DISABLED
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// 辅助关联展示字段
	HasMedicalKey    bool                `gorm:"-" json:"has_medical_key"`
	HospitalName     string              `gorm:"-" json:"hospital_name,omitempty"`
	DepartmentName   string              `gorm:"-" json:"department_name,omitempty"`
	InfectionAlert   *InfectionRiskAlert `gorm:"-" json:"infection_alert,omitempty"`
	Gender           string              `gorm:"-" json:"gender,omitempty"`
	Age              int                 `gorm:"-" json:"age,omitempty"`
	BloodType        string              `gorm:"-" json:"blood_type,omitempty"`
	Allergies        string              `gorm:"-" json:"allergies,omitempty"`
	HasAllergies     bool                `gorm:"-" json:"has_allergies"`
	EmergencyContact string              `gorm:"-" json:"emergency_contact,omitempty"`
	EmergencyPhone   string              `gorm:"-" json:"emergency_phone,omitempty"`
	ChronicDiseases  string              `gorm:"-" json:"chronic_diseases,omitempty"`
	LatestVitalSigns string              `gorm:"-" json:"latest_vital_signs,omitempty"`
}

func (User) TableName() string {
	return "users"
}

// Hospital 2. 医疗机构信息表 (hospitals)
type Hospital struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	HospitalNo string    `gorm:"size:32;uniqueIndex" json:"hospital_no"`
	Name       string    `gorm:"size:100" json:"name"`
	Level      string    `gorm:"size:20;default:'三甲'" json:"level"`
	Address    string    `gorm:"size:255;default:''" json:"address"`
	Status     int8      `gorm:"default:1" json:"status"` // 1-正常, 0-停用
	CreatedAt  time.Time `json:"created_at"`
}

func (Hospital) TableName() string {
	return "hospitals"
}

// Department 3. 医院科室信息表 (departments)
type Department struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	DeptNo      string    `gorm:"size:32" json:"dept_no"`
	HospitalID  uint64    `gorm:"index" json:"hospital_id"`
	Name        string    `gorm:"size:64" json:"name"`
	Description string    `gorm:"size:255;default:''" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

func (Department) TableName() string {
	return "departments"
}

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
	DoctorName         string `gorm:"-" json:"doctor_name,omitempty"`
	PatientName        string `gorm:"-" json:"patient_name,omitempty"`
	RecordNo           string `gorm:"-" json:"record_no,omitempty"`
	SourceHospitalName string `gorm:"-" json:"source_hospital_name,omitempty"`
	TargetHospitalName string `gorm:"-" json:"target_hospital_name,omitempty"`
}

func (EmergencyAccessEvent) TableName() string {
	return "emergency_access_events"
}

// AuditLog 9. 系统全量可信审计日志表 (audit_logs)
type AuditLog struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	LogID         string    `gorm:"size:64;uniqueIndex" json:"log_id"`
	UserID        uint64    `gorm:"index:idx_user_op" json:"user_id"`
	OperationType string    `gorm:"size:50;index:idx_user_op" json:"operation_type"` // UPLOAD, ACCESS, AUTHORIZE, REVOKE, BREAK_GLASS, VERIFY, AUDIT_DECISION, AUDIT_CLOSE
	TargetType    string    `gorm:"size:50" json:"target_type"`                      // RECORD, AUTH, EVENT, USER
	TargetID      string    `gorm:"size:64" json:"target_id"`
	HospitalID    uint64    `gorm:"index:idx_hospital_time" json:"hospital_id"`
	Result        string    `gorm:"size:20" json:"result"` // SUCCESS, FAILED, INTERCEPTED
	RiskScore     int       `gorm:"default:0" json:"risk_score"`
	RiskLevel     string    `gorm:"size:20;default:'LOW'" json:"risk_level"`
	Source        string    `gorm:"size:30;default:'WEB'" json:"source"`
	Reason        string    `gorm:"size:255;default:''" json:"reason"`
	IPAddress     string    `gorm:"size:50;default:''" json:"ip_address"`
	UserAgent     string    `gorm:"size:255;default:''" json:"user_agent"`
	Detail        string    `gorm:"size:512;default:''" json:"detail"`
	FabricTxID    string    `gorm:"size:128;default:''" json:"fabric_tx_id"`
	CreatedAt     time.Time `gorm:"index:idx_hospital_time" json:"created_at"`

	// 辅助展示
	UserName     string `gorm:"-" json:"user_name,omitempty"`
	HospitalName string `gorm:"-" json:"hospital_name,omitempty"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

// RiskRecord 10. 访问风险评估记录表 (risk_records，对应大论文 4.3.2 节)
type RiskRecord struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	AccessRequestID uint64    `gorm:"default:0;index" json:"access_request_id"`
	AccessFrequency int       `gorm:"default:0" json:"access_frequency"`
	TimeRisk        int       `gorm:"default:0" json:"time_risk"`
	BehaviorRisk    int       `gorm:"default:0" json:"behavior_risk"`
	TotalScore      int       `gorm:"default:0" json:"total_score"`
	AnomalyFlag     int8      `gorm:"default:0" json:"anomaly_flag"` // 0-正常, 1-异常
	AnalysisResult  string    `gorm:"size:255;default:''" json:"analysis_result"`
	CreatedAt       time.Time `gorm:"index" json:"created_at"`
}

func (RiskRecord) TableName() string {
	return "risk_records"
}

// BlockchainTransaction 11. 区块链交易存证表 (blockchain_transactions，对应大论文 4.3.2 节)
type BlockchainTransaction struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	BusinessType  string    `gorm:"size:50;index" json:"business_type"` // RECORD_COMMIT, AUTH_POLICY, EMERGENCY_ACCESS, AUDIT_ANCHOR, etc.
	BusinessID    string    `gorm:"size:64;index" json:"business_id"`
	TxID          string    `gorm:"size:128;uniqueIndex" json:"tx_id"`
	BlockNo       uint64    `gorm:"default:0;index" json:"block_no"`
	ChannelName   string    `gorm:"size:64;default:'medchannel'" json:"channel_name"`
	ChaincodeName string    `gorm:"size:64;default:'medical_contract'" json:"chaincode_name"`
	Status        string    `gorm:"size:20;default:'VALID'" json:"status"` // VALID, INVALID
	CreatedAt     time.Time `gorm:"index" json:"created_at"`
}

func (BlockchainTransaction) TableName() string {
	return "blockchain_transactions"
}
