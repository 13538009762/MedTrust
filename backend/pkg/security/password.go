package security

import (
	"errors"
	"strings"
	"unicode"
)

var CommonWeakPasswords = map[string]bool{
	"123456":    true,
	"12345678":  true,
	"123456789": true,
	"password":  true,
	"admin123":  true,
	"admin":     true,
	"root":      true,
	"qwerty":    true,
}

// IsWeakPassword 检查是否属于常见弱密码字典
func IsWeakPassword(password string) bool {
	return CommonWeakPasswords[strings.ToLower(strings.TrimSpace(password))]
}

// ValidatePasswordStrength 校验密码复杂度强度 (长度>=8，包含字母与数字，非弱口令黑名单)
func ValidatePasswordStrength(password string) error {
	p := strings.TrimSpace(password)
	if len(p) < 8 {
		return errors.New("密码长度不能少于 8 个字符")
	}
	if IsWeakPassword(p) {
		return errors.New("所选密码过于简单，属于常见弱密码字典，请使用更复杂的密码组合")
	}

	var hasLetter, hasDigit bool
	for _, r := range p {
		if unicode.IsLetter(r) {
			hasLetter = true
		} else if unicode.IsDigit(r) {
			hasDigit = true
		}
	}

	if !hasLetter || !hasDigit {
		return errors.New("密码必须同时包含英文字母和数字组合")
	}

	return nil
}
