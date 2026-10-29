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
