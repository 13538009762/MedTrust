package model

import (
	"time"
)

// User 1. 用户核心信息表 (users)
type User struct {
	ID               uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserNo           string     `gorm:"size:64;uniqueIndex" json:"user_no"`
	Username         string     `gorm:"size:50;uniqueIndex" json:"username"`
	PasswordHash     string     `gorm:"size:255" json:"-"`
	RealName         string     `gorm:"size:50" json:"real_name"`
	Role             string     `gorm:"size:20" json:"role"` // doctor, patient, admin, supervisor
	HospitalID       uint64     `gorm:"default:0;index:idx_hosp_dept" json:"hospital_id"`
	DepartmentID     uint64     `gorm:"default:0;index:idx_hosp_dept" json:"department_id"`
	Title            string     `gorm:"size:50;default:''" json:"title"`
	Phone            string     `gorm:"size:20;default:''" json:"phone"`
	IDCard           string     `gorm:"size:20;default:''" json:"id_card"`
	Avatar           string     `gorm:"type:longtext" json:"avatar"`
	MedicalKey       string     `gorm:"size:64;default:''" json:"-"`
	MedicalKeyHash   string     `gorm:"size:128;default:''" json:"-"`
	MustChangePwd    bool       `gorm:"default:false" json:"must_change_pwd"`
	FailedLoginCount int        `gorm:"default:0" json:"-"`
	LockedUntil      *time.Time `json:"locked_until,omitempty"`
	Status           string     `gorm:"size:20;default:'NORMAL'" json:"status"` // NORMAL, RESTRICTED, DISABLED
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

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
