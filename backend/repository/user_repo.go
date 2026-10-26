package repository

import (
	"errors"
	"medtrust-backend/model"
	"strconv"
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
		query = query.Where("real_name LIKE ? OR user_no LIKE ? OR professional_title LIKE ?",
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
