package model

import (
	"time"
)

// InfectionRiskAlert 医护职业安全高危传染病预警结构体
type InfectionRiskAlert struct {
	HasRisk        bool                   `json:"has_risk"`
	RiskLevel      string                 `json:"risk_level"`      // CRITICAL (极高危), HIGH (高危), MEDIUM_HIGH (重点中高危), SAFE (安全)
	Summary        string                 `json:"summary"`         // 简述
	Diseases       []InfectionDiseaseInfo `json:"diseases"`        // 命中高危疾病明细
	Precautions    []string               `json:"precautions"`     // 医务人员标准及专项防护指引
	ProtectionGear []string               `json:"protection_gear"` // 必备个人防护装备 (PPE)
	EmergencySteps []string               `json:"emergency_steps"` // 职业暴露发生后的紧急处置处置规程 (一挤二冲三消毒 / 2h内PEP)
	DetectedFrom   []string               `json:"detected_from"`   // 检出来源病历号或检验报告摘要
}

// InfectionDiseaseInfo 单项传染病临床特征
type InfectionDiseaseInfo struct {
	Name           string `json:"name"`             // 艾滋病 (HIV/AIDS)
	Category       string `json:"category"`         // 血液/体液传染病
	Persistence    string `json:"persistence"`      // 必定终身携带 / 长期慢性携带
	ThreatToDoctor string `json:"threat_to_doctor"` // 锐器针刺伤暴露、血液黏膜喷溅
	Transmission   string `json:"transmission"`     // 血液、体液、锐器刺伤
	RiskLevel      string `json:"risk_level"`       // CRITICAL, HIGH, MEDIUM_HIGH
	MatchedKeyword string `json:"matched_keyword"`  // 命中的病历关键词
	RecordNo       string `json:"record_no"`        // 来源就诊编号
	HospitalName   string `json:"hospital_name"`    // 来源机构
	RecordDate     string `json:"record_date"`      // 诊断/确诊时间
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
