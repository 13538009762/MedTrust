package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
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
		AESMasterKey string `yaml:"aes_master_key"`
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

func (c *Config) GetAESMasterKey() []byte {
	k := c.Security.AESMasterKey
	if len(k) == 0 {
		k = "MedTrustMasterKeyAES256GCM2026!"
	}
	for len(k) < 32 {
		k += "!"
	}
	return []byte(k[:32])
}

var AppConfig Config

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, &AppConfig); err != nil {
		return nil, err
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
