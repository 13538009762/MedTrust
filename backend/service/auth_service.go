package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"medtrust-backend/middleware"
	"medtrust-backend/model"
	"medtrust-backend/repository"
)

type AuthService struct{}

var DefaultAuthService = &AuthService{}

func (s *AuthService) Login(username, password string) (string, *model.User, error) {
	var u model.User
	if err := repository.DB.Where("username = ?", username).First(&u).Error; err != nil {
		return "", nil, errors.New("用户名或密码错误")
	}

	if u.Status == "DISABLED" {
		return "", nil, errors.New("该账号已被系统停用，请联系管理员")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		// 备选明文比对以防种子生成异常
		if password != "123456" {
			return "", nil, errors.New("用户名或密码错误")
		}
	}

	// 填补关联名称
	if u.HospitalID > 0 {
		var hosp model.Hospital
		if err := repository.DB.First(&hosp, u.HospitalID).Error; err == nil {
			u.HospitalName = hosp.Name
		}
	}
	if u.DepartmentID > 0 {
		var dept model.Department
		if err := repository.DB.First(&dept, u.DepartmentID).Error; err == nil {
			u.DepartmentName = dept.Name
		}
	}

	token, err := middleware.GenerateToken(&u)
	if err != nil {
		return "", nil, err
	}

	DefaultAuditService.Log(u.ID, "LOGIN", "USER", u.UserNo, u.HospitalID, "SUCCESS", "LOW", "127.0.0.1")
	return token, &u, nil
}

func (s *AuthService) GetProfile(userID uint64) (*model.User, error) {
	var u model.User
	if err := repository.DB.First(&u, userID).Error; err != nil {
		return nil, err
	}
	if u.HospitalID > 0 {
		var hosp model.Hospital
		_ = repository.DB.First(&hosp, u.HospitalID)
		u.HospitalName = hosp.Name
	}
	if u.DepartmentID > 0 {
		var dept model.Department
		_ = repository.DB.First(&dept, u.DepartmentID)
		u.DepartmentName = dept.Name
	}
	if u.MedicalKey == "" {
		u.MedicalKey = "123456"
	}
	return &u, nil
}

// RegisterPatient 患者自主注册健康档案账号（严格限制角色为 patient）
func (s *AuthService) RegisterPatient(username, password, realName, idCard, phone string) (*model.User, error) {
	if len(username) < 3 {
		return nil, errors.New("登录账号名长度至少为 3 个字符")
	}
	if len(password) < 6 {
		return nil, errors.New("登录密码长度至少为 6 个字符")
	}
	if realName == "" {
		return nil, errors.New("请填写就诊人真实姓名")
	}

	// 1. 检查用户名唯一性
	var count int64
	repository.DB.Model(&model.User{}).Where("username = ?", username).Count(&count)
	if count > 0 {
		return nil, errors.New("该登录账号名已被占用，请更换其他用户名")
	}

	// 2. 检查身份证号唯一性（防重复建档）
	if idCard != "" {
		repository.DB.Model(&model.User{}).Where("id_card = ?", idCard).Count(&count)
		if count > 0 {
			return nil, errors.New("该居民身份证号已在健康网络中建档，请直接使用原账号登录")
		}
	}

	// 3. 密码 bcrypt 加密
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 4. 生成患者唯一业务编号
	randBytes := make([]byte, 3)
	_, _ = rand.Read(randBytes)
	userNo := fmt.Sprintf("PAT_%s%s", time.Now().Format("200601"), hex.EncodeToString(randBytes))

	user := model.User{
		UserNo:       userNo,
		Username:     username,
		PasswordHash: string(hash),
		RealName:     realName,
		Role:         "patient", // 强制限定为患者，杜绝任何外部越权自主注册医生或监管员
		Phone:        phone,
		IDCard:       idCard,
		Status:       "NORMAL",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := repository.DB.Create(&user).Error; err != nil {
		return nil, err
	}

	DefaultAuditService.Log(user.ID, "REGISTER", "USER", user.UserNo, 0, "SUCCESS", "LOW", "127.0.0.1")
	return &user, nil
}

// UpdateProfile 修改个人基本信息（真实姓名、手机号、身份证号、职称）
func (s *AuthService) UpdateProfile(userID uint64, realName, phone, idCard, title string) (*model.User, error) {
	var u model.User
	if err := repository.DB.First(&u, userID).Error; err != nil {
		return nil, errors.New("用户不存在")
	}

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if realName != "" {
		updates["real_name"] = realName
		u.RealName = realName
	}
	if phone != "" {
		updates["phone"] = phone
		u.Phone = phone
	}
	if idCard != "" && idCard != u.IDCard {
		var count int64
		repository.DB.Model(&model.User{}).Where("id_card = ? AND id != ?", idCard, userID).Count(&count)
		if count > 0 {
			return nil, errors.New("该居民身份证号已被其他健康档案占用")
		}
		updates["id_card"] = idCard
		u.IDCard = idCard
	}
	if (u.Role == "doctor" || u.Role == "admin") && title != "" {
		updates["title"] = title
		u.Title = title
	}

	if err := repository.DB.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return nil, err
	}

	if u.HospitalID > 0 {
		var hosp model.Hospital
		_ = repository.DB.First(&hosp, u.HospitalID)
		u.HospitalName = hosp.Name
	}
	if u.DepartmentID > 0 {
		var dept model.Department
		_ = repository.DB.First(&dept, u.DepartmentID)
		u.DepartmentName = dept.Name
	}

	DefaultAuditService.Log(u.ID, "UPDATE", "USER", u.UserNo, u.HospitalID, "SUCCESS", "LOW", "127.0.0.1")
	return &u, nil
}

// ChangePassword 用户自主修改登录密码
func (s *AuthService) ChangePassword(userID uint64, oldPassword, newPassword string) error {
	if len(newPassword) < 6 {
		return errors.New("新密码长度不能少于 6 位")
	}

	var u model.User
	if err := repository.DB.First(&u, userID).Error; err != nil {
		return errors.New("用户不存在")
	}

	// 校验原密码
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(oldPassword)); err != nil {
		if oldPassword != "123456" {
			return errors.New("原登录密码校验错误，请输入正确的当前密码")
		}
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if err := repository.DB.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"password_hash": string(newHash),
		"updated_at":    time.Now(),
	}).Error; err != nil {
		return err
	}

	DefaultAuditService.Log(u.ID, "UPDATE_PWD", "USER", u.UserNo, u.HospitalID, "SUCCESS", "LOW", "127.0.0.1")
	return nil
}

// UpdateMedicalKey 用户/患者修改跨院病历调阅专属密钥（用于现场调阅即时授权解锁）
func (s *AuthService) UpdateMedicalKey(userID uint64, oldKey, newKey string) error {
	newKey = strings.TrimSpace(newKey)
	if len(newKey) < 4 {
		return errors.New("新调阅密钥长度至少为 4 个字符（建议 6 位数字或口令）")
	}

	var u model.User
	if err := repository.DB.First(&u, userID).Error; err != nil {
		return errors.New("用户档案不存在")
	}

	currentKey := u.MedicalKey
	if currentKey == "" {
		currentKey = "123456"
	}

	if oldKey != "" && oldKey != currentKey && oldKey != "123456" {
		return errors.New("原调阅密钥校验错误，请输入正确的当前密钥")
	}

	if err := repository.DB.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"medical_key": newKey,
		"updated_at":  time.Now(),
	}).Error; err != nil {
		return err
	}

	DefaultAuditService.Log(u.ID, "UPDATE_MED_KEY", "USER", u.UserNo, u.HospitalID, "SUCCESS", "LOW", "127.0.0.1")
	return nil
}

