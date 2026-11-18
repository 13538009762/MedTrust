package captcha

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestCaptchaExpiredTokenRejected(t *testing.T) {
	secret := resolveCaptchaSecret()
	expiredAt := time.Now().Add(-10 * time.Minute).Unix()
	nonce := int64(123456789)

	expiredPayload := CaptchaPayload{
		AnswerMAC: computeAnswerMAC(secret, nonce, expiredAt, "7"),
		ExpiresAt: expiredAt,
		Nonce:     nonce,
	}
	payloadBytes, _ := json.Marshal(expiredPayload)
	expiredB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(expiredB64))
	expiredSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	expiredToken := expiredB64 + "." + expiredSig

	if ok, msg := VerifyCaptcha(expiredToken, "7"); ok {
		t.Fatalf("Expected expired token to be rejected")
	} else {
		t.Logf("Expired token properly rejected: %s", msg)
	}
}
