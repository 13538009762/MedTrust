package captcha_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"medtrust-backend/pkg/captcha"
	"strings"
	"testing"
	"time"
)

func TestCaptchaLifecycle(t *testing.T) {
	res, err := captcha.GenerateCaptcha()
	if err != nil {
		t.Fatalf("GenerateCaptcha failed: %v", err)
	}

	if !strings.HasPrefix(res.CaptchaImg, "data:image/svg+xml;base64,") {
		t.Fatalf("Expected SVG data URI, got %s", res.CaptchaImg[:30])
	}

	payload := decodePayload(t, res.CaptchaToken)

	// 1. Wrong answer should fail
	ok, msg := captcha.VerifyCaptcha(res.CaptchaToken, "wrong_answer_999")
	if ok {
		t.Fatalf("Expected wrong answer to fail, but succeeded")
	}
	t.Logf("Wrong answer properly rejected: %s", msg)

	// 1b. 错答即作废：同一 token 即使给出正确答案也不得再通过（杜绝暴力穷举）
	ok, msg = captcha.VerifyCaptcha(res.CaptchaToken, payload.Answer)
	if ok {
		t.Fatalf("【安全】错答后的 token 仍可用正确答案通过，存在暴力穷举风险")
	}
	t.Logf("Token invalidated after wrong answer: %s", msg)

	// 2. Correct answer on a fresh token should succeed
	res2, err := captcha.GenerateCaptcha()
	if err != nil {
		t.Fatalf("GenerateCaptcha failed: %v", err)
	}
	payload2 := decodePayload(t, res2.CaptchaToken)

	ok, msg = captcha.VerifyCaptcha(res2.CaptchaToken, payload2.Answer)
	if !ok {
		t.Fatalf("Expected correct answer to succeed, but failed: %s", msg)
	}
	t.Logf("Correct answer validated successfully!")

	// 3. Replay attack with the same token should fail
	ok, msg = captcha.VerifyCaptcha(res2.CaptchaToken, payload2.Answer)
	if ok {
		t.Fatalf("Expected replayed token to be rejected, but succeeded")
	}
	t.Logf("Replay attack prevented: %s", msg)
}

// decodePayload 从 token 中解出（未验签的）payload，仅用于测试取正确答案。
func decodePayload(t *testing.T, token string) captcha.CaptchaPayload {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		t.Fatalf("Invalid token format: %s", token)
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("Failed to decode token payload: %v", err)
	}
	var payload captcha.CaptchaPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}
	return payload
}

func TestCaptchaTamperAndExpiry(t *testing.T) {
	res, err := captcha.GenerateCaptcha()
	if err != nil {
		t.Fatalf("GenerateCaptcha failed: %v", err)
	}

	parts := strings.Split(res.CaptchaToken, ".")
	payloadB64 := parts[0]
	// Tamper signature
	tamperedToken := payloadB64 + ".tampered_signature"

	ok, _ := captcha.VerifyCaptcha(tamperedToken, "5")
	if ok {
		t.Fatalf("Expected tampered token to fail")
	}

	// Test expired token
	expiredPayload := captcha.CaptchaPayload{
		Answer:    "7",
		ExpiresAt: time.Now().Add(-10 * time.Minute).Unix(),
		Nonce:     12345,
	}
	payloadBytes, _ := json.Marshal(expiredPayload)
	expiredB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)

	mac := hmac.New(sha256.New, []byte("medtrust-blockchain-captcha-secret-2026"))
	mac.Write([]byte(expiredB64))
	expiredSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	expiredToken := expiredB64 + "." + expiredSig

	ok, msg := captcha.VerifyCaptcha(expiredToken, "7")
	if ok {
		t.Fatalf("Expected expired token to fail")
	}
	t.Logf("Expired token properly rejected: %s", msg)
}
