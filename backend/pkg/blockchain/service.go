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

// BlockchainService 统一区块链服务抽象接口 (满足正式 Fabric 2.5 Gateway 与本地容灾 Mock 互换)
type BlockchainService interface {
	CommitAsset(assetType, assetID string, payload map[string]interface{}) (txID string, height uint64, err error)
	QueryAsset(assetID string) (map[string]interface{}, bool)
	GetStats() (totalTx int, currentHeight uint64)
	GetHistory(assetID string) ([]ChainRecord, error)
	IsLiveFabric() bool
}

// DefaultService 全局单例区块链服务实例
var DefaultService BlockchainService
