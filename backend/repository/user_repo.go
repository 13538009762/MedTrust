package repository

import (
	"errors"
	"medtrust-backend/model"
	"strconv"
	"time"
)

type UserRepository struct{}

var DefaultUserRepository = &UserRepository{}

func (r *UserRepository) FindByID(id uint64) (*model.User, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var u model.User
	if err := DB.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindByUsername(username string) (*model.User, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var u model.User
	if err := DB.Where("username = ?", username).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindByUserNo(userNo string) (*model.User, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var u model.User
	if err := DB.Where("user_no = ?", userNo).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Create(u *model.User) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Create(u).Error
}

func (r *UserRepository) UpdateStatus(id uint64, status string) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	err := DB.Model(&model.User{}).Where("id = ?", id).Update("status", status).Error
	if err == nil {
		InvalidateUserStatus(id)
	}
	return err
}

func (r *UserRepository) ListUsers(role, keyword string, hospitalID uint64) ([]model.User, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	query := DB.Model(&model.User{})
	if role != "" {
		query = query.Where("role = ?", role)
	}
	if hospitalID > 0 {
		query = query.Where("hospital_id = ?", hospitalID)
	}
	if keyword != "" {
		query = query.Where("real_name LIKE ? OR username LIKE ? OR user_no LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	var users []model.User
	err := query.Find(&users).Error
	return users, err
}

func (r *UserRepository) FindHospitalByID(id uint64) (*model.Hospital, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var h model.Hospital
	if err := DB.First(&h, id).Error; err != nil {
		return nil, err
	}
	return &h, nil
}

func (r *UserRepository) ListHospitals() ([]model.Hospital, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var list []model.Hospital
	err := DB.Find(&list).Error
	return list, err
}

func (r *UserRepository) ListDepartments(hospitalID uint64) ([]model.Department, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var list []model.Department
	query := DB.Model(&model.Department{})
	if hospitalID > 0 {
		query = query.Where("hospital_id = ?", hospitalID)
	}
	err := query.Find(&list).Error
	return list, err
}

func (r *UserRepository) FindDepartmentByID(id uint64) (*model.Department, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var d model.Department
	if err := DB.First(&d, id).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *UserRepository) UpdatePassword(id uint64, newHash string) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"password_hash":   newHash,
		"must_change_pwd": false,
	}).Error
}

func (r *UserRepository) UpdateMedicalKeyHash(id uint64, hash string) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Model(&model.User{}).Where("id = ?", id).Update("medical_key_hash", hash).Error
}

func (r *UserRepository) CheckStatus(id uint64) (string, error) {
	u, err := r.FindByID(id)
	if err != nil {
		return "", err
	}
	if u.Status == "" {
		return "NORMAL", nil
	}
	return u.Status, nil
}

func (r *UserRepository) FindByIDs(ids []uint64) ([]model.User, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var users []model.User
	err := DB.Where("id IN ?", ids).Find(&users).Error
	return users, err
}

func (r *UserRepository) FindHospitalsByIDs(ids []uint64) ([]model.Hospital, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var hosps []model.Hospital
	err := DB.Where("id IN ?", ids).Find(&hosps).Error
	return hosps, err
}

func (r *UserRepository) FindUserByIDOrNoOrUsername(paramID string) (*model.User, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var u model.User
	var err error
	if id, parseErr := strconv.ParseUint(paramID, 10, 64); parseErr == nil && id > 0 {
		err = DB.First(&u, id).Error
	} else {
		err = DB.Where("user_no = ? OR username = ?", paramID, paramID).First(&u).Error
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) ListAllUsers() ([]model.User, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var users []model.User
	err := DB.Order("id asc").Find(&users).Error
	return users, err
}

func (r *UserRepository) ListDoctors(keyword string, hospitalID uint64) ([]model.User, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	query := DB.Where("role = 'doctor'")
	if hospitalID > 0 {
		query = query.Where("hospital_id = ?", hospitalID)
	}
	if keyword != "" {
		query = query.Where("real_name LIKE ? OR user_no LIKE ? OR title LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	var doctors []model.User
	err := query.Find(&doctors).Error
	return doctors, err
}

func (r *UserRepository) FindDoctorByID(id uint64) (*model.User, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var doc model.User
	if err := DB.Where("id = ? AND role = 'doctor'", id).First(&doc).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *UserRepository) UserByIDOrNoAndUsername(paramID string) (*model.User, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var u model.User
	var err error
	if id, parseErr := strconv.ParseUint(paramID, 10, 64); parseErr == nil && id > 0 {
		err = DB.First(&u, id).Error
	} else {
		err = DB.Where("user_no = ? OR username = ?", paramID, paramID).First(&u).Error
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// FindUserStatusByID WP-15: 查询用户状态字段，供风控层直接获取医生执业状态
func (r *UserRepository) FindUserStatusByID(id uint64) (string, error) {
	if DB == nil {
		return "", errors.New("database not initialized")
	}
	var u model.User
	if err := DB.Select("status").First(&u, id).Error; err != nil {
		return "", err
	}
	return u.Status, nil
}

// FindHospitalByIDFull 查询医院完整记录（含 Name）
func (r *UserRepository) FindHospitalByIDFull(id uint64) (*model.Hospital, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var h model.Hospital
	if err := DB.First(&h, id).Error; err != nil {
		return nil, err
	}
	return &h, nil
}

// ListPatients 按角色和关键字查询患者列表
func (r *UserRepository) ListPatients(keyword string) ([]model.User, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	query := DB.Model(&model.User{}).Where("role = ?", "patient")
	if keyword != "" {
		query = query.Where("real_name LIKE ? OR phone LIKE ? OR id_card LIKE ? OR user_no LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	var list []model.User
	err := query.Order("id asc").Find(&list).Error
	return list, err
}

// UserExistsByField WP-15: 根据用户名或员工号查询用户记录（供种子数据初始化等场景复用）
func (r *UserRepository) UserExistsByField(field, value string) (*model.User, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var u model.User
	if err := DB.Where(field+" = ? OR user_no = ?", value, value).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// CountByUsername WP-15: 统计指定用户名是否已存在（注册查重用）
func (r *UserRepository) CountByUsername(username string) int64 {
	if DB == nil {
		return 0
	}
	var count int64
	DB.Model(&model.User{}).Where("username = ?", username).Count(&count)
	return count
}

// UsersWithPlainMedicalKey WP-15: 查询所有残留明文医疗密钥的用户（供密钥迁移使用）
func (r *UserRepository) UsersWithPlainMedicalKey() ([]model.User, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var users []model.User
	err := DB.Where("medical_key != '' AND (medical_key_hash = '' OR medical_key_hash IS NULL)").Find(&users).Error
	return users, err
}

// UpdateMedicalKeyMigration WP-15: 更新用户医疗密钥为哈希存储
func (r *UserRepository) UpdateMedicalKeyMigration(userID uint64, newHash string) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"medical_key_hash": newHash,
		"medical_key":      "",
		"updated_at":       time.Now(),
	}).Error
}

// UserExistsByIDCard WP-15: 检查身份证号是否已被其他用户使用
func (r *UserRepository) UserExistsByIDCard(idCard string, excludeID uint64) bool {
	if DB == nil {
		return false
	}
	var count int64
	DB.Model(&model.User{}).Where("id_card = ? AND id != ?", idCard, excludeID).Count(&count)
	return count > 0
}

// UpdateUserFields WP-15: 按用户 ID 批量更新指定字段
func (r *UserRepository) UpdateUserFields(userID uint64, updates map[string]interface{}) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error
}
