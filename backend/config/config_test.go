package config

import (
	"testing"
)

func TestConfigValidate_DemoMode(t *testing.T) {
	cfg := &Config{
		AppEnv: "demo",
	}
	cfg.Server.JWTSecret = "medtrust_secure_jwt_secret_key_2026"
	cfg.Security.AESMasterKey = "MedTrustMasterKeyAES256GCM2026!"
	cfg.Database.DSN = "root:123456@tcp(127.0.0.1:3306)/medtrust"

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Demo 模式下即使有弱密钥警告，也不应阻止启动: %v", err)
	}
}

func TestConfigValidate_ProductionMode_RejectsWeakSecrets(t *testing.T) {
	cfg := &Config{
		AppEnv: "production",
	}
	cfg.Server.JWTSecret = "medtrust_secure_jwt_secret_key_2026" // weak
	cfg.Security.AESMasterKey = "MedTrustMasterKeyAES256GCM2026!"   // weak
	cfg.Database.DSN = "root:123456@tcp(127.0.0.1:3306)/medtrust" // weak dsn

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("生产模式下应拒绝弱密钥与默认口令，但 Validate() 未报错")
	}
}

func TestConfigValidate_ProductionMode_AcceptsStrongSecrets(t *testing.T) {
	cfg := &Config{
		AppEnv: "production",
	}
	cfg.Server.JWTSecret = "c3VwZXJfc2VjdXJlX2p3dF9zZWNyZXRfa2V5XzIwMjZfcHJvZA==" // strong 44 chars
	cfg.Security.AESMasterKey = "k7R9vX2mL4pQ8sY1wZ3bN5cE7tH9uI0j"                 // strong 32 chars
	cfg.Database.DSN = "medtrust_app:StrongP@ssw0rd2026!@tcp(10.0.0.5:3306)/medtrust_prod"

	if err := cfg.Validate(); err != nil {
		t.Fatalf("强配置在生产模式下应验证通过，但报错: %v", err)
	}
}

// 生产模式下密钥为空（已从 config.yaml 外置）必须 fail-fast，不得静默回退。
func TestConfigValidate_ProductionMode_RejectsEmptySecrets(t *testing.T) {
	cfg := &Config{AppEnv: "production"}
	cfg.Server.JWTSecret = ""
	cfg.Security.AESMasterKey = ""
	cfg.Database.DSN = ""

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("生产模式下密钥与 DSN 全空时应拒绝启动")
	}
}

// demo 模式下未配置主密钥时，GetAESMasterKey 返回进程内一次性随机 32 字节密钥，
// 且不得再回退到硬编码的 "MedTrustMasterKeyAES256GCM2026!"。
func TestGetAESMasterKey_DemoGeneratesEphemeralKey(t *testing.T) {
	cfg := &Config{AppEnv: "demo"}
	cfg.Security.AESMasterKey = ""

	k1 := cfg.GetAESMasterKey()
	if len(k1) != 32 {
		t.Fatalf("一次性 demo 主密钥长度应为 32 字节，实际: %d", len(k1))
	}
	if string(k1) == "MedTrustMasterKeyAES256GCM2026!" {
		t.Fatalf("demo 模式下不得回退到硬编码模板弱密钥")
	}
	// 同一进程内应保持稳定（否则已加密的密文当次请求内都无法解开）
	if k2 := cfg.GetAESMasterKey(); string(k2) != string(k1) {
		t.Fatalf("同进程内一次性主密钥应保持一致")
	}
}

// 生产模式未配置主密钥时 GetAESMasterKey 返回 nil，由调用方/启动校验拒绝。
func TestGetAESMasterKey_ProductionWithoutKeyReturnsNil(t *testing.T) {
	cfg := &Config{AppEnv: "production"}
	cfg.Security.AESMasterKey = ""

	if k := cfg.GetAESMasterKey(); k != nil {
		t.Fatalf("生产模式未配置主密钥时应返回 nil，实际长度: %d", len(k))
	}
}
