package service

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"medtrust-backend/middleware"
	"medtrust-backend/model"
	"medtrust-backend/repository"
)

type AuthService struct{}

var DefaultAuthService = &AuthService{}

// HashMedicalKey 使用患者专属编号作为盐值对调阅密钥执行 SHA-256 哈希
func HashMedicalKey(key, userNo string) string {
	h := sha256.New()
	h.Write([]byte("MedTrust:Salt:" + userNo + ":" + strings.TrimSpace(key)))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyMedicalKey 恒定时间校验调阅密钥，优先校验哈希，兼顾旧版平滑过渡
func VerifyMedicalKey(key, userNo, storedHash, legacyPlain string) bool {
	cleanKey := strings.TrimSpace(key)
	if cleanKey == "" {
		return false
	}
	if storedHash != "" {
		calc := HashMedicalKey(cleanKey, userNo)
		if subtle.ConstantTimeCompare([]byte(calc), []byte(storedHash)) == 1 {
			return true
		}
	}
	if legacyPlain != "" && subtle.ConstantTimeCompare([]byte(cleanKey), []byte(strings.TrimSpace(legacyPlain))) == 1 {
		return true
	}
	return false
}

// GenerateSecureRandomPassword 生成 10 位安全随机一次性密码（包含大小写字母、数字及特殊符号）
func GenerateSecureRandomPassword(length int) string {
	if length < 8 {
		length = 10
	}
	const (
		upper    = "ABCDEFGHJKLMNPQRSTUVWXYZ"
		lower    = "abcdefghijkmnopqrstuvwxyz"
		digits   = "23456789"
		specials = "!@#$%^&*"
	)
	all := upper + lower + digits + specials

	res := make([]byte, length)
	res[0] = upper[cryptoRandInt(len(upper))]
	res[1] = lower[cryptoRandInt(len(lower))]
	res[2] = digits[cryptoRandInt(len(digits))]
	res[3] = specials[cryptoRandInt(len(specials))]

	for i := 4; i < length; i++ {
		res[i] = all[cryptoRandInt(len(all))]
	}

	for i := range res {
		j := cryptoRandInt(len(res))
		res[i], res[j] = res[j], res[i]
	}
	return string(res)
}

func cryptoRandInt(max int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(n.Int64())
}

func (s *AuthService) Login(username, password string) (string, *model.User, error) {
	return s.LoginWithContext(username, password, "127.0.0.1", "Internal")
}

func (s *AuthService) LoginWithContext(username, password, clientIP, userAgent string) (string, *model.User, error) {
	if clientIP == "" {
		clientIP = "127.0.0.1"
	}
	if userAgent == "" {
		userAgent = "Unknown"
	}

	var u model.User
	if err := repository.DB.Where("username = ?", username).First(&u).Error; err != nil {
		DefaultAuditService.LogDetailed(AuditEntry{
			UserID:        0,
			OperationType: "LOGIN_FAILED",
			TargetType:    "USER",
			TargetID:      username,
			Result:        "FAILED",
			RiskLevel:     "MEDIUM",
			RiskScore:     30,
			Source:        "WEB",
			Reason:        "用户名或密码错误",
			IPAddress:     clientIP,
			UserAgent:     userAgent,
		})
		return "", nil, errors.New("用户名或密码错误")
	}

	if u.Status == "DISABLED" {
		DefaultAuditService.LogDetailed(AuditEntry{
			UserID:        u.ID,
			OperationType: "LOGIN_BLOCKED",
			TargetType:    "USER",
			TargetID:      u.UserNo,
			HospitalID:    u.HospitalID,
			Result:        "INTERCEPTED",
			RiskLevel:     "HIGH",
			RiskScore:     80,
			Source:        "WEB",
			Reason:        "停用账号尝试登录拦截",
			IPAddress:     clientIP,
			UserAgent:     userAgent,
		})
		return "", nil, errors.New("该账号已被系统停用，请联系管理员")
	}

	// 检查账号防爆破临时锁定
	now := time.Now()
	if u.LockedUntil != nil && u.LockedUntil.After(now) {
		remaining := int(u.LockedUntil.Sub(now).Minutes()) + 1
		DefaultAuditService.LogDetailed(AuditEntry{
			UserID:        u.ID,
			OperationType: "LOGIN_LOCKED_ATTEMPT",
			TargetType:    "USER",
			TargetID:      u.UserNo,
			HospitalID:    u.HospitalID,
			Result:        "INTERCEPTED",
			RiskLevel:     "HIGH",
			RiskScore:     70,
			Source:        "WEB",
			Reason:        fmt.Sprintf("锁定中账号尝试登录（剩余 %d 分钟）", remaining),
			IPAddress:     clientIP,
			UserAgent:     userAgent,
		})
		return "", nil, fmt.Errorf("该账号密码错误次数过多，已被临时锁定保护，请在 %d 分钟后再试", remaining)
	}

	// 安全哈希校验（坚决杜绝任何固定密码绕过漏洞）
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		newFailed := u.FailedLoginCount + 1
		updates := map[string]interface{}{
			"failed_login_count": newFailed,
		}
		if newFailed >= 5 {
			lockTime := now.Add(15 * time.Minute)
			updates["locked_until"] = lockTime
			DefaultAuditService.LogDetailed(AuditEntry{
				UserID:        u.ID,
				OperationType: "LOGIN_LOCKED",
				TargetType:    "USER",
				TargetID:      u.UserNo,
				HospitalID:    u.HospitalID,
				Result:        "INTERCEPTED",
				RiskLevel:     "HIGH",
				RiskScore:     90,
				Source:        "WEB",
				Reason:        "连续密码错误达 5 次触发自动锁定 15 分钟",
				IPAddress:     clientIP,
				UserAgent:     userAgent,
			})
		} else {
			DefaultAuditService.LogDetailed(AuditEntry{
				UserID:        u.ID,
				OperationType: "LOGIN_FAILED",
				TargetType:    "USER",
				TargetID:      u.UserNo,
				HospitalID:    u.HospitalID,
				Result:        "FAILED",
				RiskLevel:     "MEDIUM",
				RiskScore:     30 + newFailed*10,
				Source:        "WEB",
				Reason:        fmt.Sprintf("密码错误第 %d 次", newFailed),
				IPAddress:     clientIP,
				UserAgent:     userAgent,
			})
		}
		_ = repository.DB.Model(&model.User{}).Where("id = ?", u.ID).Updates(updates)

		if newFailed >= 5 {
			return "", nil, errors.New("连续登录失败已达 5 次，该账号已被临时锁定 15 分钟")
		}
		return "", nil, errors.New("用户名或密码错误") // 统一错误信息，不泄露连续失败次数，防范用户名探测枚举
	}

	// 登录成功，重置失败计数与锁定状态
	if u.FailedLoginCount > 0 || u.LockedUntil != nil {
		_ = repository.DB.Model(&model.User{}).Where("id = ?", u.ID).Updates(map[string]interface{}{
			"failed_login_count": 0,
			"locked_until":       nil,
		})
		u.FailedLoginCount = 0
		u.LockedUntil = nil
	}

	// 存量明文医疗密钥自动静默升级为加盐安全哈希存储
	if u.MedicalKey != "" && u.MedicalKeyHash == "" {
		newHash := HashMedicalKey(u.MedicalKey, u.UserNo)
		_ = repository.DB.Model(&model.User{}).Where("id = ?", u.ID).Updates(map[string]interface{}{
			"medical_key_hash": newHash,
			"medical_key":      "",
		})
		u.MedicalKeyHash = newHash
		u.MedicalKey = ""
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

	u.HasMedicalKey = (u.MedicalKeyHash != "" || u.MedicalKey != "")

	token, err := middleware.GenerateToken(&u)
	if err != nil {
		return "", nil, err
	}

	DefaultAuditService.LogDetailed(AuditEntry{
		UserID:        u.ID,
		OperationType: "LOGIN",
		TargetType:    "USER",
		TargetID:      u.UserNo,
		HospitalID:    u.HospitalID,
		Result:        "SUCCESS",
		RiskLevel:     "LOW",
		RiskScore:     0,
		Source:        "WEB",
		Reason:        "身份密码鉴权成功通过",
		IPAddress:     clientIP,
		UserAgent:     userAgent,
	})
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

	u.HasMedicalKey = (u.MedicalKeyHash != "" || u.MedicalKey != "")
	u.MedicalKey = "" // 严格屏蔽明文输出
	return &u, nil
}

func (s *AuthService) RegisterPatient(username, password, realName, idCard, phone string) (*model.User, error) {
	return s.RegisterPatientWithContext(username, password, realName, idCard, phone, "127.0.0.1", "Internal")
}

func (s *AuthService) RegisterPatientWithContext(username, password, realName, idCard, phone, clientIP, userAgent string) (*model.User, error) {
	if clientIP == "" {
		clientIP = "127.0.0.1"
	}
	if userAgent == "" {
		userAgent = "Unknown"
	}
	if len(username) < 3 {
		return nil, errors.New("登录账号名长度至少为 3 个字符")
	}
	if len(password) < 6 {
		return nil, errors.New("登录密码长度至少为 6 个字符")
	}
	if realName == "" {
		return nil, errors.New("请填写就诊人真实姓名")
	}

	var count int64
	repository.DB.Model(&model.User{}).Where("username = ?", username).Count(&count)
	if count > 0 {
		return nil, errors.New("该登录账号名已被占用，请更换其他用户名")
	}

	if idCard != "" {
		repository.DB.Model(&model.User{}).Where("id_card = ?", idCard).Count(&count)
		if count > 0 {
			return nil, errors.New("该居民身份证号已在健康网络中建档，请直接使用原账号登录")
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	randBytes := make([]byte, 3)
	_, _ = rand.Read(randBytes)
	userNo := fmt.Sprintf("PAT_%s%s", time.Now().Format("200601"), hex.EncodeToString(randBytes))

	user := model.User{
		UserNo:       userNo,
		Username:     username,
		PasswordHash: string(hash),
		RealName:     realName,
		Role:         "patient",
		Phone:        phone,
		IDCard:       idCard,
		Status:       "NORMAL",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := repository.DB.Create(&user).Error; err != nil {
		return nil, err
	}

	DefaultAuditService.LogDetailed(AuditEntry{
		UserID:        user.ID,
		OperationType: "REGISTER",
		TargetType:    "USER",
		TargetID:      user.UserNo,
		Result:        "SUCCESS",
		RiskLevel:     "LOW",
		Source:        "WEB",
		Reason:        "新患者账户建档注册",
		IPAddress:     clientIP,
		UserAgent:     userAgent,
	})
	return &user, nil
}

// UpdateProfile 修改个人基本信息（真实姓名、手机号、身份证号、职称）
func (s *AuthService) UpdateProfile(userID uint64, realName, phone, idCard, title string) (*model.User, error) {
	return s.UpdateProfileWithContext(userID, realName, phone, idCard, title, "127.0.0.1", "Internal")
}

func (s *AuthService) UpdateProfileWithContext(userID uint64, realName, phone, idCard, title, clientIP, userAgent string) (*model.User, error) {
	if clientIP == "" {
		clientIP = "127.0.0.1"
	}
	if userAgent == "" {
		userAgent = "Unknown"
	}

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

	DefaultAuditService.LogDetailed(AuditEntry{
		UserID:        u.ID,
		OperationType: "UPDATE",
		TargetType:    "USER",
		TargetID:      u.UserNo,
		HospitalID:    u.HospitalID,
		Result:        "SUCCESS",
		RiskLevel:     "LOW",
		Source:        "WEB",
		Reason:        "个人信息资料更新",
		IPAddress:     clientIP,
		UserAgent:     userAgent,
	})
	return &u, nil
}

// ChangePassword 用户自主修改登录密码
func (s *AuthService) ChangePassword(userID uint64, oldPassword, newPassword string) error {
	return s.ChangePasswordWithContext(userID, oldPassword, newPassword, "127.0.0.1", "Internal")
}

func (s *AuthService) ChangePasswordWithContext(userID uint64, oldPassword, newPassword, clientIP, userAgent string) error {
	if clientIP == "" {
		clientIP = "127.0.0.1"
	}
	if userAgent == "" {
		userAgent = "Unknown"
	}

	if len(newPassword) < 6 {
		return errors.New("新密码长度不能少于 6 位")
	}

	var u model.User
	if err := repository.DB.First(&u, userID).Error; err != nil {
		return errors.New("用户不存在")
	}

	// 严格校验原密码（坚决杜绝任何明文固定密码绕过漏洞）
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(oldPassword)); err != nil {
		DefaultAuditService.LogDetailed(AuditEntry{
			UserID:        u.ID,
			OperationType: "UPDATE_PWD_FAILED",
			TargetType:    "USER",
			TargetID:      u.UserNo,
			HospitalID:    u.HospitalID,
			Result:        "INTERCEPTED",
			RiskLevel:     "HIGH",
			RiskScore:     50,
			Source:        "WEB",
			Reason:        "修改密码原密码校验失败",
			IPAddress:     clientIP,
			UserAgent:     userAgent,
		})
		return errors.New("原登录密码校验错误，请输入正确的当前密码")
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if err := repository.DB.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"password_hash":   string(newHash),
		"must_change_pwd": false,
		"updated_at":      time.Now(),
	}).Error; err != nil {
		return err
	}

	DefaultAuditService.LogDetailed(AuditEntry{
		UserID:        u.ID,
		OperationType: "UPDATE_PWD",
		TargetType:    "USER",
		TargetID:      u.UserNo,
		HospitalID:    u.HospitalID,
		Result:        "SUCCESS",
		RiskLevel:     "LOW",
		Source:        "WEB",
		Reason:        "用户自主更新登录密码",
		IPAddress:     clientIP,
		UserAgent:     userAgent,
	})
	return nil
}

// UpdateMedicalKey 用户/患者修改跨院病历调阅专属密钥（用于现场调阅即时授权解锁）
func (s *AuthService) UpdateMedicalKey(userID uint64, oldKey, newKey string) error {
	return s.UpdateMedicalKeyWithContext(userID, oldKey, newKey, "127.0.0.1", "Internal")
}

func (s *AuthService) UpdateMedicalKeyWithContext(userID uint64, oldKey, newKey, clientIP, userAgent string) error {
	if clientIP == "" {
		clientIP = "127.0.0.1"
	}
	if userAgent == "" {
		userAgent = "Unknown"
	}

	newKey = strings.TrimSpace(newKey)
	if len(newKey) < 6 {
		return errors.New("新调阅密钥长度至少为 6 位字符（建议 6 位数字或安全口令）")
	}

	var u model.User
	if err := repository.DB.First(&u, userID).Error; err != nil {
		return errors.New("用户档案不存在")
	}

	// 若已存在调阅密钥或哈希，强制校验原密钥
	if u.MedicalKeyHash != "" || u.MedicalKey != "" {
		if !VerifyMedicalKey(oldKey, u.UserNo, u.MedicalKeyHash, u.MedicalKey) {
			DefaultAuditService.LogDetailed(AuditEntry{
				UserID:        u.ID,
				OperationType: "UPDATE_MED_KEY_FAILED",
				TargetType:    "USER",
				TargetID:      u.UserNo,
				HospitalID:    u.HospitalID,
				Result:        "INTERCEPTED",
				RiskLevel:     "HIGH",
				RiskScore:     60,
				Source:        "WEB",
				Reason:        "原调阅密钥校验错误",
				IPAddress:     clientIP,
				UserAgent:     userAgent,
			})
			return errors.New("原调阅密钥校验错误，请输入正确的当前密钥")
		}
	}

	newHash := HashMedicalKey(newKey, u.UserNo)

	if err := repository.DB.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"medical_key_hash": newHash,
		"medical_key":      "", // 彻底清空数据库中历史明文残留
		"updated_at":       time.Now(),
	}).Error; err != nil {
		return err
	}

	DefaultAuditService.LogDetailed(AuditEntry{
		UserID:        u.ID,
		OperationType: "UPDATE_MED_KEY",
		TargetType:    "USER",
		TargetID:      u.UserNo,
		HospitalID:    u.HospitalID,
		Result:        "SUCCESS",
		RiskLevel:     "LOW",
		RiskScore:     10,
		Source:        "WEB",
		Reason:        "调阅密钥已加盐更新，旧明文已清空",
		IPAddress:     clientIP,
		UserAgent:     userAgent,
	})
	return nil
}

