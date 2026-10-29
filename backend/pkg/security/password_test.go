package security

import (
	"testing"
)

func TestValidatePasswordStrength_Valid(t *testing.T) {
	validPasswords := []string{
		"MedTrust2026",
		"Doctor#Pass1",
		"abcde12345",
		"SecureP@ss88",
	}
	for _, pwd := range validPasswords {
		if err := ValidatePasswordStrength(pwd); err != nil {
			t.Errorf("密码 %s 预期合法，但报错: %v", pwd, err)
		}
	}
}

func TestValidatePasswordStrength_Invalid(t *testing.T) {
	testCases := []struct {
		pwd string
		msg string
	}{
		{"123456", "弱密码字典拦截"},
		{"admin123", "弱密码字典拦截"},
		{"abcdef", "长度少于8位"},
		{"12345678", "纯数字无字母"},
		{"abcdefgh", "纯字母无数字"},
	}

	for _, tc := range testCases {
		if err := ValidatePasswordStrength(tc.pwd); err == nil {
			t.Errorf("密码 %s (%s) 预期报错，但未报错", tc.pwd, tc.msg)
		}
	}
}
