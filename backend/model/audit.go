package model

import (
	"time"
)

// AuditLog 9. 系统全量可信审计日志表 (audit_logs)
type AuditLog struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	LogID         string    `gorm:"size:64;uniqueIndex" json:"log_id"`
	UserID        uint64    `gorm:"index:idx_user_op" json:"user_id"`
	OperationType string    `gorm:"size:50;index:idx_user_op" json:"operation_type"` // UPLOAD, ACCESS, AUTHORIZE, REVOKE, BREAK_GLASS, VERIFY, AUDIT_DECISION, AUDIT_CLOSE
	TargetType    string    `gorm:"size:50" json:"target_type"`                      // RECORD, AUTH, EVENT, USER
	TargetID      string    `gorm:"size:64" json:"target_id"`
	HospitalID    uint64    `gorm:"index:idx_hospital_time" json:"hospital_id"`
	Result        string    `gorm:"size:20" json:"result"` // SUCCESS, FAILED, INTERCEPTED
	RiskScore     int       `gorm:"default:0" json:"risk_score"`
	RiskLevel     string    `gorm:"size:20;default:'LOW'" json:"risk_level"`
	Source        string    `gorm:"size:30;default:'WEB'" json:"source"`
	Reason        string    `gorm:"size:255;default:''" json:"reason"`
	IPAddress     string    `gorm:"size:50;default:''" json:"ip_address"`
	UserAgent     string    `gorm:"size:255;default:''" json:"user_agent"`
	Detail        string    `gorm:"size:512;default:''" json:"detail"`
	FabricTxID    string    `gorm:"size:128;default:''" json:"fabric_tx_id"`
	CreatedAt     time.Time `gorm:"index:idx_hospital_time" json:"created_at"`

	// 辅助展示
	UserName     string `gorm:"-" json:"user_name,omitempty"`
	HospitalName string `gorm:"-" json:"hospital_name,omitempty"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

// RiskRecord 10. 访问风险评估记录表 (risk_records，对应大论文 4.3.2 节)
type RiskRecord struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	AccessRequestID uint64    `gorm:"default:0;index" json:"access_request_id"`
	AccessFrequency int       `gorm:"default:0" json:"access_frequency"`
	TimeRisk        int       `gorm:"default:0" json:"time_risk"`
	BehaviorRisk    int       `gorm:"default:0" json:"behavior_risk"`
	TotalScore      int       `gorm:"default:0" json:"total_score"`
	AnomalyFlag     int8      `gorm:"default:0" json:"anomaly_flag"` // 0-正常, 1-异常
	AnalysisResult  string    `gorm:"size:255;default:''" json:"analysis_result"`
	CreatedAt       time.Time `gorm:"index" json:"created_at"`
}

func (RiskRecord) TableName() string {
	return "risk_records"
}

// BlockchainTransaction 11. 区块链交易存证表 (blockchain_transactions，对应大论文 4.3.2 节)
type BlockchainTransaction struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	BusinessType  string    `gorm:"size:50;index" json:"business_type"` // RECORD_COMMIT, AUTH_POLICY, EMERGENCY_ACCESS, AUDIT_ANCHOR, etc.
	BusinessID    string    `gorm:"size:64;index" json:"business_id"`
	TxID          string    `gorm:"size:128;uniqueIndex" json:"tx_id"`
	BlockNo       uint64    `gorm:"default:0;index" json:"block_no"`
	ChannelName   string    `gorm:"size:64;default:'medicalchannel'" json:"channel_name"`
	ChaincodeName string    `gorm:"size:64;default:'medical_contract'" json:"chaincode_name"`
	Status        string    `gorm:"size:20;default:'VALID'" json:"status"` // VALID, INVALID
	CreatedAt     time.Time `gorm:"index" json:"created_at"`
}

func (BlockchainTransaction) TableName() string {
	return "blockchain_transactions"
}
