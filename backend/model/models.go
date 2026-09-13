package model

import (
	"time"
)

type User struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserNo       string    `gorm:"size:64;uniqueIndex" json:"user_no"`
	Username     string    `gorm:"size:50;uniqueIndex" json:"username"`
	PasswordHash string    `gorm:"size:255" json:"-"`
	RealName     string    `gorm:"size:50" json:"real_name"`
	Role         string    `gorm:"size:20" json:"role"` // doctor, patient, admin, supervisor
	HospitalID   uint64    `gorm:"default:0;index:idx_hosp_dept" json:"hospital_id"`
	DepartmentID uint64    `gorm:"default:0;index:idx_hosp_dept" json:"department_id"`
	Title        string    `gorm:"size:50;default:''" json:"title"`
	Phone        string    `gorm:"size:20;default:''" json:"phone"`
	IDCard       string    `gorm:"size:20;default:''" json:"id_card"`
	MedicalKey   string    `gorm:"size:64;default:'123456'" json:"medical_key,omitempty"` // 患者跨院病历调阅专属密码/现场授权密钥
	Status       string    `gorm:"size:20;default:'NORMAL'" json:"status"` // NORMAL, RESTRICTED, DISABLED
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	// 辅助关联展示字段
	HospitalName   string `gorm:"-" json:"hospital_name,omitempty"`
	DepartmentName string `gorm:"-" json:"department_name,omitempty"`
}

func (User) TableName() string {
	return "users"
}

type Hospital struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	HospitalNo string    `gorm:"size:32;uniqueIndex" json:"hospital_no"`
	Name       string    `gorm:"size:100" json:"name"`
	Level      string    `gorm:"size:20;default:'三甲'" json:"level"`
	Address    string    `gorm:"size:255;default:''" json:"address"`
	Status     int8      `gorm:"default:1" json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

func (Hospital) TableName() string {
	return "hospitals"
}

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

type MedicalRecord struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	RecordNo      string    `gorm:"size:64;uniqueIndex" json:"record_no"`
	PatientID     uint64    `gorm:"index" json:"patient_id"`
	DoctorID      uint64    `gorm:"index:idx_doctor_hosp" json:"doctor_id"`
	HospitalID    uint64    `gorm:"index:idx_doctor_hosp" json:"hospital_id"`
	DataType      string    `gorm:"size:32" json:"data_type"` // EMR, REPORT, IMAGE
	OnsetTime     string    `gorm:"size:50;default:''" json:"onset_time"`
	Duration      string    `gorm:"size:50;default:''" json:"duration"`
	Symptoms      string    `gorm:"type:text" json:"symptoms"`
	Etiology      string    `gorm:"type:text" json:"etiology"`
	TreatmentPlan string    `gorm:"type:text" json:"treatment_plan"`
	VitalSigns    string    `gorm:"size:255;default:''" json:"vital_signs"`
	Diagnosis     string    `gorm:"type:text" json:"diagnosis"`
	FabricTxID    string    `gorm:"size:128;default:''" json:"fabric_tx_id"`
	BlockHeight   uint64    `gorm:"default:0" json:"block_height"`
	CreatedAt     time.Time `json:"created_at"`

	// 就诊事件 (Encounter) 核心业务扩展字段
	EncounterType    string `gorm:"size:32;default:'OUTPATIENT'" json:"encounter_type"` // OUTPATIENT, EMERGENCY, INPATIENT
	DepartmentName   string `gorm:"size:64;default:'综合门诊'" json:"department_name"`
	Status           string `gorm:"size:32;default:'COMPLETED'" json:"status"`           // INITIAL_DIAGNOSIS, WAITING_FOR_EXAM, EXAM_COMPLETED, COMPLETED
	ChiefComplaint   string `gorm:"type:text" json:"chief_complaint"`                   // 主诉
	PresentIllness   string `gorm:"type:text" json:"present_illness"`                   // 现病史
	InitialDiagnosis string `gorm:"type:text" json:"initial_diagnosis"`                 // 初步诊断
	DiagnosticBasis  string `gorm:"type:text" json:"diagnostic_basis"`                  // 初步诊断依据
	NeedExam         bool   `gorm:"default:false" json:"need_exam"`                     // 是否开具检查
	ExamItems        string `gorm:"size:255;default:''" json:"exam_items"`              // 检查项目
	ExamReason       string `gorm:"type:text" json:"exam_reason"`                       // 检查原因
	ExamResult       string `gorm:"type:text" json:"exam_result"`                       // 检验科/影像科报告结果描述
	ExamDoctor       string `gorm:"size:50;default:''" json:"exam_doctor"`              // 检查出具医生/技师
	ExamTime         string `gorm:"size:50;default:''" json:"exam_time"`                // 检查完成时间

	// 辅助展示
	PatientName   string        `gorm:"-" json:"patient_name,omitempty"`
	PatientIDCard string        `gorm:"-" json:"patient_id_card,omitempty"`
	PatientPhone  string        `gorm:"-" json:"patient_phone,omitempty"`
	DoctorName    string        `gorm:"-" json:"doctor_name,omitempty"`
	HospitalName  string        `gorm:"-" json:"hospital_name,omitempty"`
	Files         []MedicalFile `gorm:"foreignKey:RecordID" json:"files,omitempty"`

	// 区块链防篡改动态核验状态
	IsTampered   bool   `gorm:"-" json:"is_tampered"`
	Verified     bool   `gorm:"-" json:"verified"`
	ChainHash    string `gorm:"-" json:"chain_hash,omitempty"`
	CurrentHash  string `gorm:"-" json:"current_hash,omitempty"`
	TamperReason string `gorm:"-" json:"tamper_reason,omitempty"`

	// 跨院访问权限状态 (针对当前调用接口的医生上下文)
	HasAccess  bool   `gorm:"-" json:"has_access"`
	AccessType string `gorm:"-" json:"access_type,omitempty"` // OWNER, HOSPITAL, AUTHORIZED, BREAK_GLASS, UNAUTHORIZED
}

func (MedicalRecord) TableName() string {
	return "medical_records"
}

type MedicalFile struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	RecordID  uint64    `gorm:"index" json:"record_id"`
	FileName  string    `gorm:"size:255" json:"file_name"`
	FileType  string    `gorm:"size:32" json:"file_type"`
	FileSize  uint64    `json:"file_size"`
	IPFSCID   string    `gorm:"column:ipfs_cid;size:128;index" json:"ipfs_cid"`
	FileHash  string    `gorm:"size:64" json:"file_hash"`
	CreatedAt time.Time `json:"created_at"`
}

func (MedicalFile) TableName() string {
	return "medical_files"
}

// MedicalExamOrder 医技检查申请与回传报告单 (支持检验科、影像科等流程解耦)
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

type Authorization struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	AuthNo         string    `gorm:"size:64;uniqueIndex" json:"auth_no"`
	PatientID      uint64    `gorm:"index:idx_patient_target" json:"patient_id"`
	AuthTargetType string    `gorm:"size:20;index:idx_patient_target" json:"auth_target_type"` // DOCTOR, HOSPITAL
	AuthTargetID   uint64    `gorm:"index:idx_patient_target" json:"auth_target_id"`
	ScopeType      string    `gorm:"size:20" json:"scope_type"` // ALL, SINGLE
	RecordID       uint64    `gorm:"default:0" json:"record_id"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	Status         string    `gorm:"size:20;default:'ACTIVE';index:idx_patient_target" json:"status"` // ACTIVE, REVOKED
	FabricTxID     string    `gorm:"size:128;default:''" json:"fabric_tx_id"`
	CreatedAt      time.Time `json:"created_at"`

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
	Decision         string    `gorm:"size:20" json:"decision"`                 // ALLOWED, PENDING_CONFIRM, REJECTED
	Status           string    `gorm:"size:20;default:'PENDING'" json:"status"` // PENDING, APPROVED, REJECTED
	ScopeType        string    `gorm:"size:20;default:'SINGLE'" json:"scope_type"` // SINGLE, ALL
	Days             int       `gorm:"default:7" json:"days"`
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

type EmergencyAccessEvent struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	EventNo          string    `gorm:"size:64;uniqueIndex" json:"event_no"`
	DoctorID         uint64    `gorm:"index:idx_doc_pat" json:"doctor_id"`
	PatientID        uint64    `gorm:"index:idx_doc_pat" json:"patient_id"`
	RecordID         uint64    `json:"record_id"`
	SourceHospitalID uint64    `json:"source_hospital_id"`
	TargetHospitalID uint64    `json:"target_hospital_id"`
	EmergencyReason  string    `gorm:"size:50" json:"emergency_reason"` // RESCUE, COMA, CRITICAL, OTHER
	Description      string    `gorm:"type:text" json:"description"`
	DoctorConfirmed  int8      `gorm:"default:1" json:"doctor_confirmed"`
	PatientFeedback  string    `gorm:"size:20;default:'PENDING'" json:"patient_feedback"` // PENDING, CONFIRMED, OBJECTED
	AuditStatus      string    `gorm:"size:30;default:'PENDING_AUDIT';index" json:"audit_status"` // PENDING_AUDIT, UNDER_INVESTIGATION, CLOSED_APPROVED, CLOSED_VIOLATION
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

type AuditLog struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	LogID         string    `gorm:"size:64;uniqueIndex" json:"log_id"`
	UserID        uint64    `gorm:"index:idx_user_op" json:"user_id"`
	OperationType string    `gorm:"size:50;index:idx_user_op" json:"operation_type"` // UPLOAD, ACCESS, AUTHORIZE, REVOKE, BREAK_GLASS, VERIFY, AUDIT_CLOSE
	TargetType    string    `gorm:"size:50" json:"target_type"`                      // RECORD, AUTH, FILE, EVENT
	TargetID      string    `gorm:"size:64" json:"target_id"`
	HospitalID    uint64    `gorm:"index:idx_hospital_time" json:"hospital_id"`
	Result        string    `gorm:"size:20" json:"result"`         // SUCCESS, FAILED, INTERCEPTED
	RiskLevel     string    `gorm:"size:20;default:'LOW'" json:"risk_level"`
	IPAddress     string    `gorm:"size:50;default:''" json:"ip_address"`
	FabricTxID    string    `gorm:"size:128;default:''" json:"fabric_tx_id"`
	CreatedAt     time.Time `gorm:"index:idx_hospital_time" json:"created_at"`

	// 辅助展示
	UserName     string `gorm:"-" json:"user_name,omitempty"`
	HospitalName string `gorm:"-" json:"hospital_name,omitempty"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}
