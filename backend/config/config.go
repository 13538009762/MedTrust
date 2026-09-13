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
		APIURL     string `yaml:"api_url"`
		StorageDir string `yaml:"storage_dir"`
	} `yaml:"ipfs"`
	Blockchain struct {
		ChannelID   string `yaml:"channel_id"`
		ChaincodeID string `yaml:"chaincode_id"`
		LedgerDir   string `yaml:"ledger_dir"`
		Fabric      struct {
			Enabled      bool   `yaml:"enabled"`
			PeerEndpoint string `yaml:"peer_endpoint"`
			GatewayPeer  string `yaml:"gateway_peer"`
			MSPID        string `yaml:"msp_id"`
			TLSCertPath  string `yaml:"tls_cert_path"`
			CertPath     string `yaml:"cert_path"`
			KeyPath      string `yaml:"key_path"`
		} `yaml:"fabric"`
	} `yaml:"blockchain"`
	Risk struct {
		LowThreshold  int `yaml:"low_threshold"`
		HighThreshold int `yaml:"high_threshold"`
	} `yaml:"risk"`
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
	return &AppConfig, nil
}
