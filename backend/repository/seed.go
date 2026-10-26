package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"medtrust-backend/model"
)

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
			// 仅补齐基础组织与职称属性并解除临时登录锁，保留用户已修改的密码及监管处置状态 (RESTRICTED/DISABLED)
			updates := map[string]interface{}{
				"role":               u.Role,
				"hospital_id":        u.HospitalID,
				"department_id":      u.DepartmentID,
				"title":              u.Title,
				"failed_login_count": 0,
				"locked_until":       nil,
			}
			if existing.PasswordHash == "" {
				updates["password_hash"] = u.PasswordHash
			}
			if existing.Status == "" {
				updates["status"] = u.Status
			}
			db.Model(&existing).Updates(updates)
		}
	}

	// 4. 确保演示辅助账号无密码时补齐初始密码 123456 并解除临时暴破锁定（不覆盖已封禁/受限状态）
	demoUsernames := []string{
		"admin", "pat_zhang", "doc_li", "doc_wang", "doc_chen", "sup_zhao",
		"doc_a", "doc_b", "doc_c", "doc_a2", "doc_a3", "doc_b2", "doc_b3", "doc_b4", "doc_c2", "doc_c3", "doc_c4",
		"pat_li", "pat_wang", "pat_zhao", "pat_qian", "pat_sun", "pat_zhou", "pat_chen", "pat_zhouhq", "pat_linxz",
		"tech_lab_a", "tech_pacs_a", "tech_lab_b", "tech_pacs_b", "tech_lab_c", "tech_pacs_c", "supervisor",
	}
	db.Model(&model.User{}).Where("username IN ? AND (password_hash = '' OR password_hash IS NULL)", demoUsernames).Update("password_hash", hashStr)
	db.Model(&model.User{}).Where("username IN ?", demoUsernames).Updates(map[string]interface{}{
		"failed_login_count": 0,
		"locked_until":       nil,
	})

	// 主动迁移：扫描 users 表中所有长度小于 32 的明文 medical_key，自动哈希加密并清空明文
	var plainKeyUsers []model.User
	if err := db.Where("medical_key != '' AND (medical_key_hash = '' OR medical_key_hash IS NULL)").Find(&plainKeyUsers).Error; err == nil {
		for _, pu := range plainKeyUsers {
			cleanKey := strings.TrimSpace(pu.MedicalKey)
			if len(cleanKey) > 0 && len(cleanKey) < 32 {
				h := sha256.New()
				h.Write([]byte("MedTrust:Salt:" + pu.UserNo + ":" + cleanKey))
				newHash := hex.EncodeToString(h.Sum(nil))
				_ = db.Model(&model.User{}).Where("id = ?", pu.ID).Updates(map[string]interface{}{
					"medical_key_hash": newHash,
					"medical_key":      "",
				})
			}
		}
	}

	log.Printf("[DB-Seed] MedTrust 标准测试种子账号与机构初始化完成")
}
