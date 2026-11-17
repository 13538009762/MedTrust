package repository

import (
	"errors"
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

// TxRunner 事务内写操作收口：service 层通过它执行事务写入，不直接触碰 GORM 构建器
type TxRunner struct {
	db *gorm.DB
}

// Create 在事务内创建一条记录
func (t *TxRunner) Create(value interface{}) error {
	return t.db.Create(value).Error
}

// Save 在事务内整体保存一条记录
func (t *TxRunner) Save(value interface{}) error {
	return t.db.Save(value).Error
}

// Model 在事务内指定目标表模型（批量更新场景）
func (t *TxRunner) Model(value interface{}) *TxRunnerQuery {
	return &TxRunnerQuery{db: t.db.Model(value)}
}

// TxRunnerQuery 事务内条件更新链（仅暴露收口的条件构建操作）
type TxRunnerQuery struct {
	db *gorm.DB
}

// Where 追加查询条件
func (q *TxRunnerQuery) Where(query interface{}, args ...interface{}) *TxRunnerQuery {
	return &TxRunnerQuery{db: q.db.Where(query, args...)}
}

// Updates 批量更新字段
func (q *TxRunnerQuery) Updates(values interface{}) error {
	return q.db.Updates(values).Error
}

// Update 更新单个字段
func (q *TxRunnerQuery) Update(column string, value interface{}) error {
	return q.db.Update(column, value).Error
}

// Transaction WP-14/WP-15: 多表原子写事务封装。fn 返回 error 时自动回滚。
func Transaction(fn func(tx *TxRunner) error) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Transaction(func(txDB *gorm.DB) error {
		return fn(&TxRunner{db: txDB})
	})
}

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
		&model.ConsistencyTask{},
	)
	if err != nil {
		log.Printf("[DB] AutoMigrate warning: %v", err)
	}

	// 自动注入幂等标准测试种子数据
	SeedInitialData(DB)

	return DB, nil
}
