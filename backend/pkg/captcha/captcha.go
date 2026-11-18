package captcha

import (
	"crypto/hmac"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"medtrust-backend/config"
)

var (
	// 仅在未配置 JWT_SECRET / CAPTCHA_SECRET 时使用的进程级随机兜底密钥（严禁硬编码常量）
	fallbackSecretOnce sync.Once
	fallbackSecret     []byte

	// Set of already-consumed captcha tokens to prevent replay / brute-force attacks
	usedTokensLock sync.Mutex
	usedTokens     = make(map[string]int64)
	maxUsedTokens  = 10000
)

// resolveCaptchaSecret 动态解析验证码 HMAC 密钥：
// 优先从配置/环境变量的 JWT_SECRET / CAPTCHA_SECRET 派生隔离密钥，未配置时生成进程内一次性 CSPRNG 随机密钥。
func resolveCaptchaSecret() []byte {
	baseSecret := strings.TrimSpace(os.Getenv("CAPTCHA_SECRET"))
	if baseSecret == "" {
		baseSecret = strings.TrimSpace(config.AppConfig.Server.JWTSecret)
	}
	if baseSecret == "" {
		baseSecret = strings.TrimSpace(os.Getenv("JWT_SECRET"))
	}
	if baseSecret != "" {
		sum := sha256.Sum256([]byte("MedTrust:CaptchaKey:" + baseSecret))
		return sum[:]
	}
	fallbackSecretOnce.Do(func() {
		fallbackSecret = make([]byte, 32)
		_, _ = crand.Read(fallbackSecret)
	})
	return fallbackSecret
}

func cryptoIntn(max int) int {
	if max <= 1 {
		return 0
	}
	n, err := crand.Int(crand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(n.Int64())
}

func cryptoInt63() int64 {
	n, err := crand.Int(crand.Reader, big.NewInt(1<<62))
	if err != nil {
		return time.Now().UnixNano()
	}
	return n.Int64()
}

func computeAnswerMAC(secret []byte, nonce, expiresAt int64, answer string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(fmt.Sprintf("%d:%d:%s", nonce, expiresAt, answer)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

type CaptchaPayload struct {
	AnswerMAC string `json:"answer_mac"`
	ExpiresAt int64  `json:"exp"`
	Nonce     int64  `json:"nonce"`
}

type CaptchaResult struct {
	CaptchaImg   string `json:"captcha_img"`
	CaptchaToken string `json:"captcha_token"`
}

// GenerateCaptcha creates a random math problem and renders it as an SVG image (base64 data URI).
func GenerateCaptcha() (*CaptchaResult, error) {
	// 运算数 0~99：单 token 错答即作废（见 VerifyCaptcha），扩大范围以压低猜测命中率
	num1 := cryptoIntn(100)
	num2 := cryptoIntn(100)

	op := "+"
	var answer int
	if cryptoIntn(2) == 0 {
		op = "+"
		answer = num1 + num2
	} else {
		op = "-"
		if num1 < num2 {
			num1, num2 = num2, num1
		}
		answer = num1 - num2
	}

	challenge := fmt.Sprintf("%d %s %d = ?", num1, op, num2)

	// Generate SVG
	width, height := 120, 38

	// Noise lines
	var noiseLines strings.Builder
	for i := 0; i < 3; i++ {
		x1 := cryptoIntn(width)
		y1 := cryptoIntn(height)
		x2 := cryptoIntn(width)
		y2 := cryptoIntn(height)
		hue := cryptoIntn(360)
		color := fmt.Sprintf("hsl(%d, 70%%, 50%%)", hue)
		noiseLines.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="1.5" opacity="0.6" />`, x1, y1, x2, y2, color))
	}

	// Noise dots
	var noiseDots strings.Builder
	for i := 0; i < 28; i++ {
		cx := cryptoIntn(width)
		cy := cryptoIntn(height)
		radius := 1.0 + float64(cryptoIntn(15))/10.0
		hue := cryptoIntn(360)
		color := fmt.Sprintf("hsl(%d, 60%%, 60%%)", hue)
		noiseDots.WriteString(fmt.Sprintf(`<circle cx="%d" cy="%d" r="%.1f" fill="%s" opacity="0.5" />`, cx, cy, radius, color))
	}

	// Characters with rotation and emerald/teal/slate hues matching MedTrust theme
	var textContent strings.Builder
	chars := []rune(challenge)
	for i, char := range chars {
		x := 10 + i*13 + (cryptoIntn(3) - 1)
		y := 26 + (cryptoIntn(5) - 2)
		rot := cryptoIntn(24) - 12
		hue := 140 + cryptoIntn(80) // Emerald to teal range
		color := fmt.Sprintf("hsl(%d, 85%%, 35%%)", hue)
		fontSize := 17 + cryptoIntn(4)
		textContent.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-family="monospace, Courier, sans-serif" font-weight="bold" font-size="%d" fill="%s" transform="rotate(%d %d %d)">%c</text>`, x, y, fontSize, color, rot, x, y, char))
	}

	svg := fmt.Sprintf(`<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg" style="background: #f0fdf4; border: 1px solid #bbf7d0; border-radius: 6px; user-select: none;">
		%s
		%s
		%s
	</svg>`, width, height, noiseLines.String(), noiseDots.String(), textContent.String())

	encodedSVG := base64.StdEncoding.EncodeToString([]byte(svg))
	imgDataURI := "data:image/svg+xml;base64," + encodedSVG

	// Create signed token (5 min expiration); 答案仅以带 Nonce 的 HMAC 摘要封存，绝不回传明文答案
	secret := resolveCaptchaSecret()
	expiresAt := time.Now().Add(5 * time.Minute).Unix()
	nonce := cryptoInt63()
	payload := CaptchaPayload{
		AnswerMAC: computeAnswerMAC(secret, nonce, expiresAt, strconv.Itoa(answer)),
		ExpiresAt: expiresAt,
		Nonce:     nonce,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payloadB64))
	sigB64 := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	token := payloadB64 + "." + sigB64

	return &CaptchaResult{
		CaptchaImg:   imgDataURI,
		CaptchaToken: token,
	}, nil
}

// VerifyCaptcha validates that the token is authentic, unexpired, matches the answer, and has not been replayed.
func VerifyCaptcha(token string, answer string) (bool, string) {
	answer = strings.TrimSpace(answer)
	token = strings.TrimSpace(token)
	if token == "" || answer == "" {
		return false, "请输入计算验证码"
	}

	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false, "验证码令牌无效"
	}

	payloadB64, sigB64 := parts[0], parts[1]
	secret := resolveCaptchaSecret()

	// 先验签：拒绝任何伪造或格式错误令牌污染已消费缓存
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payloadB64))
	expectedSig := mac.Sum(nil)

	actualSig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil || !hmac.Equal(actualSig, expectedSig) {
		return false, "验证码签名校验失败"
	}

	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return false, "验证码数据损坏"
	}

	var payload CaptchaPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return false, "验证码格式异常"
	}

	// Check expiration
	if time.Now().Unix() > payload.ExpiresAt {
		return false, "验证码已过期，请刷新"
	}

	// 原子检查并立即消费 token（防并发重放竞态 TOCTOU，且错答即作废杜绝穷举）
	if !consumeTokenAtomic(token, payload.ExpiresAt) {
		return false, "验证码已被使用，请刷新后重试"
	}

	// Check answer MAC in constant time
	expectedAnswerMAC, err1 := base64.RawURLEncoding.DecodeString(computeAnswerMAC(secret, payload.Nonce, payload.ExpiresAt, answer))
	actualAnswerMAC, err2 := base64.RawURLEncoding.DecodeString(payload.AnswerMAC)
	if err1 != nil || err2 != nil || len(actualAnswerMAC) == 0 || !hmac.Equal(actualAnswerMAC, expectedAnswerMAC) {
		return false, "验证码计算错误，请重新输入"
	}

	return true, ""
}

// consumeTokenAtomic 在同一互斥锁内原子校验并标记 token 已消费，消除检查与写入之间的并发重放窗口。
func consumeTokenAtomic(token string, expiresAt int64) bool {
	usedTokensLock.Lock()
	defer usedTokensLock.Unlock()

	if _, used := usedTokens[token]; used {
		return false
	}

	if len(usedTokens) >= maxUsedTokens {
		// Clean up expired ones
		now := time.Now().Unix()
		for k, exp := range usedTokens {
			if exp < now {
				delete(usedTokens, k)
			}
		}
		if len(usedTokens) >= maxUsedTokens {
			// Clear half if still full
			count := 0
			for k := range usedTokens {
				delete(usedTokens, k)
				count++
				if count >= maxUsedTokens/2 {
					break
				}
			}
		}
	}
	usedTokens[token] = expiresAt
	return true
}
