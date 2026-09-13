package blockchain

// ChainRecord 账本流转存证记录
type ChainRecord struct {
	TxID        string                 `json:"tx_id"`
	BlockHeight uint64                 `json:"block_height"`
	Timestamp   string                 `json:"timestamp"`
	AssetType   string                 `json:"asset_type"`
	AssetID     string                 `json:"asset_id"`
	Payload     map[string]interface{} `json:"payload"`
}

// BlockchainStatusDTO 区块链底层引擎运行状态
type BlockchainStatusDTO struct {
	Mode        string `json:"mode"`
	Connected   bool   `json:"connected"`
	Network     string `json:"network,omitempty"`
	Chaincode   string `json:"chaincode,omitempty"`
	Peer        string `json:"peer,omitempty"`
	LatestBlock uint64 `json:"latest_block"`
	Message     string `json:"message,omitempty"`
}

// AuditLog 链上审计结构
type AuditLog struct {
	AuditID   string `json:"audit_id"`
	UserID    string `json:"user_id"`
	Action    string `json:"action"`
	RecordID  string `json:"record_id"`
	RiskLevel string `json:"risk_level"`
	Timestamp string `json:"timestamp"`
}


// BlockchainService 统一区块链服务抽象接口 (满足正式 Fabric 2.5 Gateway 与本地容灾 Mock 互换)
type BlockchainService interface {
	CommitAsset(assetType, assetID string, payload map[string]interface{}) (txID string, height uint64, err error)
	QueryAsset(assetID string) (map[string]interface{}, bool)
	GetStats() (totalTx int, currentHeight uint64)
	GetHistory(assetID string) ([]ChainRecord, error)
	IsLiveFabric() bool
	GetBlockchainStatus() BlockchainStatusDTO
}

// DefaultService 全局单例区块链服务实例
var DefaultService BlockchainService

