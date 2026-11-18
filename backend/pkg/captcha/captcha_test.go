package captcha_test

import (
	"encoding/base64"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"medtrust-backend/pkg/captcha"
)

// solveSVGChallenge 从生成返回的 SVG 图像中提取算术题明文并计算正确答案（模拟真实人眼识别）。
func solveSVGChallenge(t *testing.T, imgDataURI string) string {
	t.Helper()
	const prefix = "data:image/svg+xml;base64,"
	if !strings.HasPrefix(imgDataURI, prefix) {
		t.Fatalf("Expected SVG data URI, got %s", imgDataURI)
	}
	svgBytes, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(imgDataURI, prefix))
	if err != nil {
		t.Fatalf("Failed to decode SVG base64: %v", err)
	}
	re := regexp.MustCompile(`<text[^>]*>(.)</text>`)
	matches := re.FindAllStringSubmatch(string(svgBytes), -1)
	var expr strings.Builder
	for _, m := range matches {
		expr.WriteString(m[1])
	}
	// expr 形如 "42 + 17 = ?"
	parts := strings.Fields(expr.String())
	if len(parts) < 3 {
		t.Fatalf("Unexpected SVG expression: %q", expr.String())
	}
	a, err1 := strconv.Atoi(parts[0])
	op := parts[1]
	b, err2 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil {
		t.Fatalf("Failed to parse operands from %q", expr.String())
	}
	switch op {
	case "+":
		return strconv.Itoa(a + b)
	case "-":
		return strconv.Itoa(a - b)
	default:
		t.Fatalf("Unexpected operator %q in %q", op, expr.String())
		return ""
	}
}

func TestCaptchaLifecycle_NoPlaintextAnswerLeak(t *testing.T) {
	res, err := captcha.GenerateCaptcha()
	if err != nil {
		t.Fatalf("GenerateCaptcha failed: %v", err)
	}

	correctAns := solveSVGChallenge(t, res.CaptchaImg)

	// 关键安全断言：token 前半段 payload 不得包含明文答案或 "answer" 字段
	parts := strings.Split(res.CaptchaToken, ".")
	if len(parts) != 2 {
		t.Fatalf("Invalid token format: %s", res.CaptchaToken)
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("Failed to decode token payload: %v", err)
	}
	var rawMap map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &rawMap); err != nil {
		t.Fatalf("Failed to unmarshal payload JSON: %v", err)
	}
	if _, hasPlainAnswer := rawMap["answer"]; hasPlainAnswer {
		t.Fatalf("【安全漏洞】captcha_token 中仍在明文泄露 answer 字段: %s", string(payloadBytes))
	}
	if macStr, ok := rawMap["answer_mac"].(string); !ok || macStr == "" || macStr == correctAns {
		t.Fatalf("期望 payload 使用非空 answer_mac 摘要，实际: %v", rawMap)
	}

	// 1. 错答拦截 + 单 token 错答即作废
	ok, msg := captcha.VerifyCaptcha(res.CaptchaToken, "wrong_answer_999")
	if ok {
		t.Fatalf("Expected wrong answer to fail, but succeeded")
	}
	t.Logf("Wrong answer properly rejected: %s", msg)

	ok, msg = captcha.VerifyCaptcha(res.CaptchaToken, correctAns)
	if ok {
		t.Fatalf("【安全】错答后的 token 仍可用正确答案通过，存在暴力穷举风险")
	}
	t.Logf("Token invalidated after wrong answer: %s", msg)

	// 2. 新 token 正确答案放行
	res2, err := captcha.GenerateCaptcha()
	if err != nil {
		t.Fatalf("GenerateCaptcha failed: %v", err)
	}
	correctAns2 := solveSVGChallenge(t, res2.CaptchaImg)

	ok, msg = captcha.VerifyCaptcha(res2.CaptchaToken, correctAns2)
	if !ok {
		t.Fatalf("Expected correct answer to succeed, but failed: %s", msg)
	}

	// 3. 重放拦截
	ok, msg = captcha.VerifyCaptcha(res2.CaptchaToken, correctAns2)
	if ok {
		t.Fatalf("Expected replayed token to be rejected, but succeeded")
	}
	t.Logf("Replay attack prevented: %s", msg)
}

func TestCaptchaConcurrentReplayProtection(t *testing.T) {
	res, err := captcha.GenerateCaptcha()
	if err != nil {
		t.Fatalf("GenerateCaptcha failed: %v", err)
	}
	ans := solveSVGChallenge(t, res.CaptchaImg)

	const goroutines = 32
	var wg sync.WaitGroup
	var successCount int32
	start := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if ok, _ := captcha.VerifyCaptcha(res.CaptchaToken, ans); ok {
				atomic.AddInt32(&successCount, 1)
			}
		}()
	}
	close(start)
	wg.Wait()

	if successCount != 1 {
		t.Fatalf("并发重放防护失效：期望恰好 1 次通过，实际通过 %d 次", successCount)
	}
}

func TestCaptchaTamperSignature(t *testing.T) {
	res, err := captcha.GenerateCaptcha()
	if err != nil {
		t.Fatalf("GenerateCaptcha failed: %v", err)
	}
	ans := solveSVGChallenge(t, res.CaptchaImg)

	parts := strings.Split(res.CaptchaToken, ".")
	tamperedToken := parts[0] + ".tampered_signature"

	if ok, _ := captcha.VerifyCaptcha(tamperedToken, ans); ok {
		t.Fatalf("Expected tampered token to fail")
	}
}
