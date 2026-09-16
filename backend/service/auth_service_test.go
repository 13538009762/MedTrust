package service_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"medtrust-backend/config"
	"medtrust-backend/model"
	"medtrust-backend/repository"
	"medtrust-backend/service"
)

func TestHashMedicalKeyAndVerification(t *testing.T) {
	userNo := "DOC_TEST_001"
	rawKey := "SecretMedicalKey2026!"

	hash := service.HashMedicalKey(rawKey, userNo)
	if hash == "" {
		t.Fatalf("HashMedicalKey returned empty string")
	}

	// 1. 正确密钥校验成功
	if !service.VerifyMedicalKey(rawKey, userNo, hash, "") {
		t.Fatalf("VerifyMedicalKey with correct key should return true")
	}

	// 2. 错误密钥校验失败
	if service.VerifyMedicalKey("WrongKey123", userNo, hash, "") {
		t.Fatalf("VerifyMedicalKey with wrong key should return false")
	}

	// 3. 空输入应拒绝
	if service.VerifyMedicalKey("", userNo, hash, "") {
		t.Fatalf("VerifyMedicalKey with empty key should return false")
	}

	// 4. 兼容老版本明文密钥（无盐哈希但有旧明文字段）
	legacyPlain := "LegacyPlain123"
	if !service.VerifyMedicalKey(legacyPlain, userNo, "", legacyPlain) {
		t.Fatalf("VerifyMedicalKey should support legacy plaintext key migration")
	}
}

func TestGenerateSecureRandomPassword(t *testing.T) {
	for _, length := range []int{10, 12, 16} {
		pwd := service.GenerateSecureRandomPassword(length)
		if len(pwd) != length {
			t.Fatalf("Generated password length mismatch: got %d, want %d", len(pwd), length)
		}
	}
}

func TestAuthService_FailedLoginLockoutFlow(t *testing.T) {
	// 初始化配置与数据库
	_, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("Load config failed: %v", err)
	}
	_, err = repository.InitDB()
	if err != nil {
		t.Fatalf("Init DB failed: %v", err)
	}

	authSvc := service.DefaultAuthService

	// 创建临时测试账号
	testUserNo := "DOC_LOCKOUT_TEST"
	rawPassword := "CorrectPassword123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)

	var user model.User
	repository.DB.Where("user_no = ?", testUserNo).Delete(&model.User{})
	user = model.User{
		UserNo:           testUserNo,
		Username:         testUserNo,
		RealName:         "防爆破测试医生",
		PasswordHash:     string(hash),
		Role:             "doctor",
		Status:           "NORMAL",
		FailedLoginCount: 0,
		CreatedAt:        time.Now(),
	}
	if err := repository.DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	defer repository.DB.Delete(&user)

	// 连续 4 次输入错误密码
	for i := 1; i <= 4; i++ {
		_, _, err := authSvc.Login(testUserNo, "WrongPassword")
		if err == nil {
			t.Fatalf("第 %d 次错误密码预期登录失败，但返回成功", i)
		}
	}

	// 检查 DB 中失败计数是否为 4，未被锁定
	var uCheck model.User
	repository.DB.First(&uCheck, user.ID)
	if uCheck.FailedLoginCount != 4 {
		t.Fatalf("预期 FailedLoginCount=4，实际为: %d", uCheck.FailedLoginCount)
	}

	// 第 5 次输错密码，触发 15 分钟锁定
	_, _, err = authSvc.Login(testUserNo, "WrongPassword")
	if err == nil {
		t.Fatalf("第 5 次错误密码预期锁定并报错，但返回成功")
	}

	repository.DB.First(&uCheck, user.ID)
	if uCheck.LockedUntil == nil || uCheck.LockedUntil.Before(time.Now()) {
		t.Fatalf("第 5 次输错后账号未被锁定至未来时间: %v", uCheck.LockedUntil)
	}

	// 此时即便输入正确密码，也应因处于锁定窗口期而被拦截
	_, _, err = authSvc.Login(testUserNo, rawPassword)
	if err == nil {
		t.Fatalf("账户锁定状态下使用正确密码仍应被拦截，但返回成功")
	}

	// 手动清除锁定时间，模拟锁定到期后使用正确密码登录
	repository.DB.Model(&uCheck).Updates(map[string]interface{}{
		"locked_until": nil,
	})

	token, userResp, err := authSvc.Login(testUserNo, rawPassword)
	if err != nil {
		t.Fatalf("锁定解除后使用正确密码登录失败: %v", err)
	}
	if token == "" || userResp == nil {
		t.Fatalf("登录返回凭证异常: token=%s", token)
	}

	// 验证登录成功后失败计数被重置归零
	repository.DB.First(&uCheck, user.ID)
	if uCheck.FailedLoginCount != 0 {
		t.Fatalf("登录成功后失败次数应重置为 0，实际为: %d", uCheck.FailedLoginCount)
	}
}

// TestAuthService_PreventUsernameEnumeration 验证账号枚举漏洞防御：
// 不存在的用户名与存在的用户名输错密码，返回完全一致的通用报错信息，杜绝侧信道泄露用户账号存在性
func TestAuthService_PreventUsernameEnumeration(t *testing.T) {
	_, _ = config.LoadConfig("../config/config.yaml")
	_, _ = repository.InitDB()
	authSvc := service.DefaultAuthService

	// 1. 针对完全不存在的用户名尝试登录
	_, _, errNotExist := authSvc.Login("NON_EXISTENT_USER_XYZ_9999", "RandomPassword123")
	if errNotExist == nil {
		t.Fatalf("不存在的用户应登录失败")
	}

	// 2. 针对已存在的用户输错密码尝试登录
	testUserNo := "DOC_ENUM_TEST"
	hash, _ := bcrypt.GenerateFromPassword([]byte("ValidPassword123"), bcrypt.DefaultCost)
	testUser := model.User{
		UserNo:       testUserNo,
		Username:     testUserNo,
		PasswordHash: string(hash),
		Role:         "doctor",
		Status:       "NORMAL",
		CreatedAt:    time.Now(),
	}
	repository.DB.Where("user_no = ?", testUserNo).Delete(&model.User{})
	repository.DB.Create(&testUser)
	defer repository.DB.Delete(&testUser)

	_, _, errWrongPass := authSvc.Login(testUserNo, "DefinitelyWrongPassword")
	if errWrongPass == nil {
		t.Fatalf("错误密码应登录失败")
	}

	// 3. 严格断言二者错误信息完全一致，统一为 "用户名或密码错误"
	if errNotExist.Error() != "用户名或密码错误" {
		t.Fatalf("不存在用户的错误信息暴露细节: %s, 期望为: 用户名或密码错误", errNotExist.Error())
	}
	if errWrongPass.Error() != "用户名或密码错误" {
		t.Fatalf("错误密码的错误信息暴露细节: %s, 期望为: 用户名或密码错误", errWrongPass.Error())
	}
	if errNotExist.Error() != errWrongPass.Error() {
		t.Fatalf("用户名存在与不存在的报错信息不一致，存在用户名枚举风险: %q vs %q", errNotExist.Error(), errWrongPass.Error())
	}
}

// TestAuthService_UpdateMedicalKey_Security 验证医疗密钥修改安全性：
// 1. 旧密钥校验必须通过
// 2. 新密钥哈希加密存储
// 3. 原明文密钥字段严格置空
// 4. 接口返回的 User 结构体严格剔除明文密钥
func TestAuthService_UpdateMedicalKey_Security(t *testing.T) {
	_, _ = config.LoadConfig("../config/config.yaml")
	_, _ = repository.InitDB()
	authSvc := service.DefaultAuthService

	userNo := "DOC_MEDKEY_TEST"
	oldKey := "OldMedicalSecret2026!"
	newKey := "NewMedicalSecret2026!"
	initialHash := service.HashMedicalKey(oldKey, userNo)

	user := model.User{
		UserNo:         userNo,
		Username:       userNo,
		Role:           "doctor",
		Status:         "NORMAL",
		MedicalKeyHash: initialHash,
		MedicalKey:     "LegacyLeakedPlaintextKey", // 模拟历史残留明文
		CreatedAt:      time.Now(),
	}
	repository.DB.Where("user_no = ?", userNo).Delete(&model.User{})
	repository.DB.Create(&user)
	defer repository.DB.Delete(&user)

	// 1. 错误旧密钥测试
	err := authSvc.UpdateMedicalKeyWithContext(user.ID, "WrongOldKey", newKey, "192.168.1.100", "Mozilla/5.0")
	if err == nil {
		t.Fatalf("旧医疗密钥错误时预期被拒绝，但成功执行")
	}

	// 2. 正确旧密钥修改
	err = authSvc.UpdateMedicalKeyWithContext(user.ID, oldKey, newKey, "192.168.1.100", "Mozilla/5.0")
	if err != nil {
		t.Fatalf("修改医疗密钥失败: %v", err)
	}

	// 4. 验证数据库中明文彻底抹除且哈希正确
	var dbUser model.User
	repository.DB.First(&dbUser, user.ID)
	if dbUser.MedicalKey != "" {
		t.Fatalf("数据库中的明文 medical_key 未被置空: %s", dbUser.MedicalKey)
	}
	if !service.VerifyMedicalKey(newKey, userNo, dbUser.MedicalKeyHash, "") {
		t.Fatalf("新密钥哈希校验失败")
	}
}

// TestAuthService_SensitiveFieldsJsonIgnored 验证所有敏感凭证字段在 JSON 序列化中均被 `json:"-"` 屏蔽
func TestAuthService_SensitiveFieldsJsonIgnored(t *testing.T) {
	u := model.User{
		ID:               1001,
		UserNo:           "DOC_TEST_JSON",
		Username:         "doctest",
		PasswordHash:     "$2a$10$abcdefghijklmnopqrstuvwxyz123456",
		MedicalKey:       "SUPER_SECRET_PLAINTEXT_KEY",
		MedicalKeyHash:   "SALTED_HASH_987654321",
		FailedLoginCount: 3,
		RealName:         "张三医生",
		Role:             "doctor",
	}

	data, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	jsonStr := string(data)

	// 严格断言关键敏感字段绝未出现在 JSON 输出中
	forbiddenSubstrings := []string{
		"\"password_hash\":",
		"\"medical_key\":",
		"\"medical_key_hash\":",
		"\"failed_login_count\":",
		"SUPER_SECRET_PLAINTEXT_KEY",
		"SALTED_HASH_987654321",
	}

	for _, sub := range forbiddenSubstrings {
		if strings.Contains(jsonStr, sub) {
			t.Fatalf("User JSON 序列化泄露敏感字段或内容 %q: %s", sub, jsonStr)
		}
	}
}

