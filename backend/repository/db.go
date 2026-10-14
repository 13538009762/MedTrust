package repository

import (
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"medtrust-backend/config"
	"medtrust-backend/model"
)

var DB *gorm.DB

// MemoryCache 用于在无 Redis 环境下的内存降级容灾实现
type MemoryCache struct {
	data sync.Map
}

var DefaultCache = &MemoryCache{}

func (c *MemoryCache) Set(key string, val interface{}, ttl time.Duration) {
	expireAt := time.Now().Add(ttl)
	c.data.Store(key, struct {
		val      interface{}
		expireAt time.Time
	}{val: val, expireAt: expireAt})
}

func (c *MemoryCache) Get(key string) (interface{}, bool) {
	v, ok := c.data.Load(key)
	if !ok {
		return nil, false
	}
	item := v.(struct {
		val      interface{}
		expireAt time.Time
	})
	if time.Now().After(item.expireAt) {
		c.data.Delete(key)
		return nil, false
	}
	return item.val, true
}

func (c *MemoryCache) Delete(key string) {
	c.data.Delete(key)
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
	)
	if err != nil {
		log.Printf("[DB] AutoMigrate warning: %v", err)
	}

	// 自动注入幂等标准测试种子数据
	SeedInitialData(DB)

	return DB, nil
}

// SeedInitialData 启动时自动幂等写入指令书标准测试数据
func SeedInitialData(db *gorm.DB) {
	if db == nil {
		return
	}

	// 1. 机构数据
	hospitals := []model.Hospital{
		{ID: 1, HospitalNo: "HOSP_A", Name: "广州第一附属医院", Level: "三甲", Address: "广州市越秀区中山路1号", Status: 1},
		{ID: 2, HospitalNo: "HOSP_B", Name: "深圳中心医院", Level: "三甲", Address: "深圳市福田区深南大道88号", Status: 1},
		{ID: 3, HospitalNo: "HOSP_C", Name: "珠海人民医院", Level: "三甲", Address: "珠海市香洲区人民路66号", Status: 1},
	}
	for _, h := range hospitals {
		db.Where("id = ? OR hospital_no = ?", h.ID, h.HospitalNo).FirstOrCreate(&h)
	}

	// 2. 科室数据
	depts := []model.Department{
		{ID: 1, DeptNo: "DEPT_A01", HospitalID: 1, Name: "心内科", Description: "心血管疾病诊疗中心"},
		{ID: 2, DeptNo: "DEPT_A02", HospitalID: 1, Name: "急诊科", Description: "急危重症抢救中心"},
		{ID: 3, DeptNo: "DEPT_B01", HospitalID: 2, Name: "心内科", Description: "心血管介入科室"},
		{ID: 4, DeptNo: "DEPT_B02", HospitalID: 2, Name: "急诊科", Description: "创伤急救中心"},
		{ID: 5, DeptNo: "DEPT_C01", HospitalID: 3, Name: "神经内科", Description: "脑血管与神经病变"},
	}
	for _, d := range depts {
		db.Where("id = ? OR dept_no = ?", d.ID, d.DeptNo).FirstOrCreate(&d)
	}

	// 3. 用户密码同一哈希 (123456)
	pwdHash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	hashStr := string(pwdHash)

	// 标准测试账号清单 (updatev1.txt 第九节)
	seedUsers := []model.User{
		// doc_li: 李医生，医院A-心内科，主任医师，NORMAL
		{UserNo: "DOC_LI_001", Username: "doc_li", PasswordHash: hashStr, RealName: "李建国医生", Role: "doctor", HospitalID: 1, DepartmentID: 1, Title: "主任医师", Phone: "13800000001", Status: "NORMAL"},
		// doc_wang: 王医生，医院B-心内科，主治医师，NORMAL
		{UserNo: "DOC_WANG_001", Username: "doc_wang", PasswordHash: hashStr, RealName: "王明德医生", Role: "doctor", HospitalID: 2, DepartmentID: 3, Title: "主治医师", Phone: "13800000002", Status: "NORMAL"},
		// doc_chen: 陈医生，医院B-急诊科，主治医师，NORMAL
		{UserNo: "DOC_CHEN_001", Username: "doc_chen", PasswordHash: hashStr, RealName: "陈建明医生", Role: "doctor", HospitalID: 2, DepartmentID: 4, Title: "主治医师", Phone: "13800000003", Status: "NORMAL"},
		// pat_zhang: 患者张三，user_no=PAT_001
		{UserNo: "PAT_001", Username: "pat_zhang", PasswordHash: hashStr, RealName: "张三", Role: "patient", HospitalID: 0, DepartmentID: 0, Title: "患者", Phone: "13900000001", IDCard: "110101198805122315", Status: "NORMAL"},
		// sup_zhao: 监管员赵主任，卫健委监管端
		{UserNo: "SUP_ZHAO_001", Username: "sup_zhao", PasswordHash: hashStr, RealName: "赵主任", Role: "supervisor", HospitalID: 0, DepartmentID: 0, Title: "监管主任", Phone: "13600000001", Status: "NORMAL"},
		// admin: 系统管理员
		{UserNo: "ADM_001", Username: "admin", PasswordHash: hashStr, RealName: "系统管理员", Role: "admin", HospitalID: 1, DepartmentID: 0, Title: "技术主管", Phone: "13700000001", Status: "NORMAL"},
	}

	for _, u := range seedUsers {
		var existing model.User
		if err := db.Where("username = ?", u.Username).First(&existing).Error; err != nil {
			_ = db.Create(&u)
		} else {
			// 更新关键属性保持同步
			db.Model(&existing).Updates(map[string]interface{}{
				"role":          u.Role,
				"hospital_id":   u.HospitalID,
				"department_id": u.DepartmentID,
				"title":         u.Title,
				"status":        u.Status,
			})
		}
	}
	log.Printf("[DB-Seed] MedTrust 标准测试种子账号与机构初始化完成")
}
