package repository

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"medtrust-backend/config"
	"medtrust-backend/model"
)

var DB *gorm.DB

func InitDB() (*gorm.DB, error) {
	dsn := config.AppConfig.Database.DSN
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL database: %w", err)
	}

	_ = DB.AutoMigrate(
		&model.User{},
		&model.Hospital{},
		&model.Department{},
		&model.MedicalRecord{},
		&model.MedicalFile{},
		&model.MedicalExamOrder{},
		&model.Authorization{},
		&model.AccessRequest{},
		&model.EmergencyAccessEvent{},
		&model.AuditLog{},
	)

	return DB, nil
}
