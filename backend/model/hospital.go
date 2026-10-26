package model

import (
	"time"
)

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
