package service_test

import (
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
