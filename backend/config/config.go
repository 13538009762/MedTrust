package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type Config struct {
	AppEnv string `yaml:"app_env"` // "demo" (default) or "production"

	Server struct {
		Port           int    `yaml:"port"`
		JWTSecret      string `yaml:"jwt_secret"`
		JWTExpireHours int    `yaml:"jwt_expire_hours"`
	} `yaml:"server"`
	Database struct {
		DSN string `yaml:"dsn"`
	} `yaml:"database"`
	IPFS struct {
		Mode       string `yaml:"mode"` // "live" or "mock"
		APIURL     string `yaml:"api_url"`
		StorageDir string `yaml:"storage_dir"`
	} `yaml:"ipfs"`
	Blockchain struct {
		Mode        string `yaml:"mode"` // "fabric" or "mock"
		ChannelID   string `yaml:"channel_id"`
		ChaincodeID string `yaml:"chaincode_id"`
		LedgerDir   string `yaml:"ledger_dir"`
		Fabric      struct {
			Mode         string `yaml:"mode"`
			Enabled      bool   `yaml:"enabled"`
			PeerEndpoint string `yaml:"peer_endpoint"`
			GatewayPeer  string `yaml:"gateway_peer"`
			MSPID        string `yaml:"msp_id"`
			TLSCertPath  string `yaml:"tls_cert_path"`
			CertPath     string `yaml:"cert_path"`
			KeyPath      string `yaml:"key_path"`
		} `yaml:"fabric"`
	} `yaml:"blockchain"`

	Security struct {
		AESMasterKey       string `yaml:"aes_master_key"`
		CORSAllowedOrigins string `yaml:"cors_allowed_origins"`
	} `yaml:"security"`

	Risk struct {
		LowThreshold  int `yaml:"low_threshold"`
		HighThreshold int `yaml:"high_threshold"`
	} `yaml:"risk"`

	Redis struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"`

	AI struct {
		Enabled     bool    `yaml:"enabled"`
		APIURL      string  `yaml:"api_url"`
		APIKey      string  `yaml:"api_key"`
		Model       string  `yaml:"model"`
		Temperature float64 `yaml:"temperature"`
	} `yaml:"ai"`
}

// IsProduction 判断当前是否处于严格生产模式
func (c *Config) IsProduction() bool {
	return strings.ToLower(strings.TrimSpace(c.AppEnv)) == "production"
}

var (
	ephemeralKeyOnce sync.Once
	ephemeralKey     []byte
)

// GetAESMasterKey 获取 32-byte AES 主密钥（支持 base64: 前缀编码或 32 字节明文）
//
// 密钥来源优先级：配置/AES_MASTER_KEY 环境变量 > (demo) 进程内一次性随机密钥。
// 生产模式不允许静默回退：Validate() 已 fail-fast；此处再兜一层直接返回 nil 由调用方拒绝。
func (c *Config) GetAESMasterKey() []byte {
	k := strings.TrimSpace(c.Security.AESMasterKey)
	if strings.HasPrefix(k, "base64:") {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(k, "base64:"))
		if err == nil && len(decoded) == 32 {
			return decoded
		}
	}
	if len(k) == 32 {
		return []byte(k)
	}
	// 非 32 字节的显式配置仍按历史行为补齐（demo 容灾），但会打印警告
	if len(k) > 0 {
		log.Printf("[Security WARNING] AES 主密钥长度为 %d 字节，非 32 字节标准长度，已在内存中补齐/截断使用", len(k))
		for len(k) < 32 {
			k += "!"
		}
		return []byte(k[:32])
	}
	if c.IsProduction() {
		return nil
	}
	// demo 且未配置密钥：生成进程内一次性随机密钥（重启后旧密文不可解）
	ephemeralKeyOnce.Do(func() {
		ephemeralKey = make([]byte, 32)
		if _, err := rand.Read(ephemeralKey); err != nil {
			log.Fatalf("[Security FATAL] 无法生成一次性 AES 主密钥: %v", err)
		}
		log.Println("[Security WARNING (Demo Mode)] 未配置 AES_MASTER_KEY，已生成进程内一次性随机主密钥；重启后既有密文将无法解密，请在 .env 中显式配置")
	})
	return ephemeralKey
}

// Validate 校验启动配置的安全性
func (c *Config) Validate() error {
	isProd := c.IsProduction()
	var errs []string

	// 1. JWT 密钥校验
	jwtSec := strings.TrimSpace(c.Server.JWTSecret)
	if len(jwtSec) < 32 {
		msg := fmt.Sprintf("JWTSecret 长度不足 32 字符 (当前: %d)", len(jwtSec))
		if isProd {
			errs = append(errs, msg)
		} else {
			log.Printf("[Security WARNING (Demo Mode)] %s", msg)
		}
	}
	if isKnownWeakSecret(jwtSec) {
		msg := "JWTSecret 正使用模板默认值或已知弱密钥"
		if isProd {
			errs = append(errs, msg)
		} else {
			log.Printf("[Security WARNING (Demo Mode)] %s", msg)
		}
	}

	// 2. AES 主密钥校验
	rawKey := strings.TrimSpace(c.Security.AESMasterKey)
	var keyBytes []byte
	if strings.HasPrefix(rawKey, "base64:") {
		b, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(rawKey, "base64:"))
		if err == nil && len(b) == 32 {
			keyBytes = b
		}
	} else if len(rawKey) == 32 {
		keyBytes = []byte(rawKey)
	}

	if isProd {
		if len(keyBytes) != 32 {
			errs = append(errs, fmt.Sprintf("生产模式下 AESMasterKey 必须为恰好 32 字节或有效 base64: 32字节编码 (当前: %d)", len(rawKey)))
		}
		if isKnownWeakAESKey(rawKey) {
			errs = append(errs, "AESMasterKey 正使用模板默认弱密钥 (MedTrustMasterKeyAES256GCM2026!)")
		}
	} else {
		if len(keyBytes) != 32 {
			log.Printf("[Security WARNING (Demo Mode)] AES 主密钥长度为 %d 字节，已在内存中平滑适配为 32 字节", len(rawKey))
		}
		if isKnownWeakAESKey(rawKey) {
			log.Printf("[Security WARNING (Demo Mode)] AESMasterKey 正使用模板默认弱密钥 (%s)", rawKey)
		}
	}

	// 3. 生产环境数据库敏感配置校验
	if isProd {
		dsn := strings.TrimSpace(c.Database.DSN)
		if dsn == "" {
			errs = append(errs, "生产环境必须通过 DB_DSN 显式配置数据库连接串")
		} else if lower := strings.ToLower(dsn); strings.Contains(lower, "root:123456") || strings.Contains(lower, ":@tcp") {
			errs = append(errs, "生产环境数据库 DSN 严禁使用默认 root 口令或空口令")
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("配置安全校验未通过 (Security Validation Failed):\n - %s", strings.Join(errs, "\n - "))
	}
	return nil
}

func isKnownWeakSecret(s string) bool {
	weak := []string{
		"medtrust_secure_jwt_secret_key_2026",
		"secret",
		"123456",
		"jwt_secret",
	}
	for _, w := range weak {
		if s == w {
			return true
		}
	}
	return false
}

func isKnownWeakAESKey(s string) bool {
	weak := []string{
		"MedTrustMasterKeyAES256GCM2026!",
		"12345678901234567890123456789012",
	}
	for _, w := range weak {
		if s == w {
			return true
		}
	}
	return false
}

var AppConfig Config

func loadDotEnv() {
	// 测试也需加载 .env：密钥与 DB_DSN 已从 config.yaml 外置，否则 go test 会因缺少 DSN/密钥失败。
	paths := []string{".env", "../.env", "../../.env"}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					val := strings.TrimSpace(parts[1])
					val = strings.Trim(val, `"'`)
					if os.Getenv(key) == "" {
						_ = os.Setenv(key, val)
					}
				}
			}
			break
		}
	}
}

func LoadConfig(path string) (*Config, error) {
	loadDotEnv()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, &AppConfig); err != nil {
		return nil, err
	}
	if AppConfig.AppEnv == "" {
		AppConfig.AppEnv = "demo"
	}
	if v := os.Getenv("APP_ENV"); v != "" {
		AppConfig.AppEnv = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		AppConfig.Server.JWTSecret = v
	}
	if v := os.Getenv("DB_DSN"); v != "" {
		AppConfig.Database.DSN = v
	}
	if v := os.Getenv("AES_MASTER_KEY"); v != "" {
		AppConfig.Security.AESMasterKey = v
	}
	if v := os.Getenv("MEDTRUST_CRYPTO_MASTER_KEY"); v != "" {
		AppConfig.Security.AESMasterKey = v
	}
	if v := os.Getenv("CORS_ALLOWED_ORIGINS"); v != "" {
		AppConfig.Security.CORSAllowedOrigins = v
	}
	if v := os.Getenv("AI_API_KEY"); v != "" {
		AppConfig.AI.APIKey = v
		AppConfig.AI.Enabled = true
	}
	if v := os.Getenv("AI_API_URL"); v != "" {
		AppConfig.AI.APIURL = v
	}
	if v := os.Getenv("AI_MODEL"); v != "" {
		AppConfig.AI.Model = v
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		AppConfig.Redis.Addr = v
	}
	if v := os.Getenv("FABRIC_PEER_ENDPOINT"); v != "" {
		AppConfig.Blockchain.Fabric.PeerEndpoint = v
	}
	if v := os.Getenv("IPFS_API_URL"); v != "" {
		AppConfig.IPFS.APIURL = v
	}
	if v := os.Getenv("BLOCKCHAIN_MODE"); v != "" {
		AppConfig.Blockchain.Mode = v
	}
	return &AppConfig, nil
}
