package captcha

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	// Server secret key for HMAC signing
	captchaSecret = []byte("medtrust-blockchain-captcha-secret-2026")

	// Set of already-consumed captcha tokens to prevent replay / brute-force attacks
	usedTokensLock sync.Mutex
	usedTokens     = make(map[string]int64)
	maxUsedTokens  = 10000
)

type CaptchaPayload struct {
	Answer    string `json:"answer"`
	ExpiresAt int64  `json:"exp"`
	Nonce     int64  `json:"nonce"`
}

type CaptchaResult struct {
	CaptchaImg   string `json:"captcha_img"`
	CaptchaToken string `json:"captcha_token"`
}

// GenerateCaptcha creates a random math problem and renders it as an SVG image (base64 data URI).
func GenerateCaptcha() (*CaptchaResult, error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	num1 := r.Intn(10) + 1
	num2 := r.Intn(10) + 1

	op := "+"
	var answer int
	if r.Intn(2) == 0 {
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
		x1 := r.Intn(width)
		y1 := r.Intn(height)
		x2 := r.Intn(width)
		y2 := r.Intn(height)
		hue := r.Intn(360)
		color := fmt.Sprintf("hsl(%d, 70%%, 50%%)", hue)
		noiseLines.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="1.5" opacity="0.6" />`, x1, y1, x2, y2, color))
	}

	// Noise dots
	var noiseDots strings.Builder
	for i := 0; i < 28; i++ {
		cx := r.Intn(width)
		cy := r.Intn(height)
		radius := 1.0 + r.Float64()*1.5
		hue := r.Intn(360)
		color := fmt.Sprintf("hsl(%d, 60%%, 60%%)", hue)
		noiseDots.WriteString(fmt.Sprintf(`<circle cx="%d" cy="%d" r="%.1f" fill="%s" opacity="0.5" />`, cx, cy, radius, color))
	}

	// Characters with rotation and emerald/teal/slate hues matching MedTrust theme
	var textContent strings.Builder
	chars := []rune(challenge)
	for i, char := range chars {
		x := 10 + i*13 + (r.Intn(3) - 1)
		y := 26 + (r.Intn(5) - 2)
		rot := r.Intn(24) - 12
		hue := 140 + r.Intn(80) // Emerald to teal range
		color := fmt.Sprintf("hsl(%d, 85%%, 35%%)", hue)
		fontSize := 17 + r.Intn(4)
		textContent.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-family="monospace, Courier, sans-serif" font-weight="bold" font-size="%d" fill="%s" transform="rotate(%d %d %d)">%c</text>`, x, y, fontSize, color, rot, x, y, char))
	}

	svg := fmt.Sprintf(`<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg" style="background: #f0fdf4; border: 1px solid #bbf7d0; border-radius: 6px; user-select: none;">
		%s
		%s
		%s
	</svg>`, width, height, noiseLines.String(), noiseDots.String(), textContent.String())

	encodedSVG := base64.StdEncoding.EncodeToString([]byte(svg))
	imgDataURI := "data:image/svg+xml;base64," + encodedSVG

	// Create signed token (5 min expiration)
	payload := CaptchaPayload{
		Answer:    strconv.Itoa(answer),
		ExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
		Nonce:     r.Int63(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)

	mac := hmac.New(sha256.New, captchaSecret)
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

	// Check replay
	usedTokensLock.Lock()
	if _, used := usedTokens[token]; used {
		usedTokensLock.Unlock()
		return false, "验证码已被使用，请刷新后重试"
	}
	usedTokensLock.Unlock()

	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false, "验证码令牌无效"
	}

	payloadB64, sigB64 := parts[0], parts[1]

	// Check signature
	mac := hmac.New(sha256.New, captchaSecret)
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

	// Check answer
	if payload.Answer != answer {
		return false, "验证码计算错误，请重新输入"
	}

	// Mark as consumed to prevent replay attack
	usedTokensLock.Lock()
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
	usedTokens[token] = payload.ExpiresAt
	usedTokensLock.Unlock()

	return true, ""
}
