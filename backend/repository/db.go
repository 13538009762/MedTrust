package repository

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"medtrust-backend/config"
	"medtrust-backend/model"
)

var DB *gorm.DB

// InitDB 初始化 MySQL 数据库连接、自动迁移表结构并幂等写入标准测试种子数据
func InitDB() (*gorm.DB, error) {
	dsn := config.AppConfig.Database.DSN
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL database: %w", err)
	}

	// 配置连接池参数，防止高并发场景下耗尽系统 TCP 端口与 MySQL 最大连接数
	if sqlDB, err := DB.DB(); err == nil {
		sqlDB.SetMaxIdleConns(20)
		sqlDB.SetMaxOpenConns(40)
		sqlDB.SetConnMaxLifetime(10 * time.Minute)
	}

	// 自动迁移 9 张核心表 + 扩展表
	err = DB.AutoMigrate(
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
		&model.RiskRecord{},
		&model.BlockchainTransaction{},
	)
	if err != nil {
		log.Printf("[DB] AutoMigrate warning: %v", err)
	}

	// 自动注入幂等标准测试种子数据
	SeedInitialData(DB)

	return DB, nil
}
