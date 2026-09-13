package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)


// MockLedgerService 高仿真本地区块链账本引擎 (满足无 Docker 容器环境下的答辩与单测演示)
type MockLedgerService struct {
	ledgerDir     string
	currentHeight uint64
	stateStore    map[string]string
	history       []ChainRecord
	mu            sync.RWMutex
}

// 保持向前兼容别名
type LedgerEngine = MockLedgerService

var DefaultLedger *MockLedgerService
var once sync.Once

func InitLedger(ledgerDir string) *MockLedgerService {
	once.Do(func() {
		_ = os.MkdirAll(ledgerDir, 0755)
		DefaultLedger = &MockLedgerService{
			ledgerDir:     ledgerDir,
			currentHeight: 100,
			stateStore:    make(map[string]string),
			history:       make([]ChainRecord, 0),
		}
		DefaultLedger.load()
		DefaultService = DefaultLedger
	})
	return DefaultLedger
}

// InitBlockchainService 初始化区块链服务引擎 (支持 Fabric 2.5 官方网关与 Mock 引擎无缝切换)
func InitBlockchainService(ledgerDir string, fabricCfg FabricGatewayConfig) BlockchainService {
	mock := InitLedger(ledgerDir)

	// 优先读取环境变量 BLOCKCHAIN_MODE (支持 fabric 或 mock)
	mode := strings.ToLower(os.Getenv("BLOCKCHAIN_MODE"))
	if mode == "" {
		mode = strings.ToLower(fabricCfg.Mode)
	}
	if mode == "" {
		if fabricCfg.Enabled {
			mode = "fabric"
		} else {
			mode = "mock"
		}
	}

	if mode == "fabric" {
		gateway := NewFabricGatewayService(fabricCfg)
		DefaultService = gateway
		return gateway
	}

	DefaultService = mock
	return mock
}


func (l *MockLedgerService) IsLiveFabric() bool {
	return false
}

func (l *MockLedgerService) GetBlockchainStatus() BlockchainStatusDTO {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return BlockchainStatusDTO{
		Mode:        "mock",
		Connected:   false,
		Message:     "MockLedger Development Mode",
		Network:     "medchannel",
		Chaincode:   "medical",
		Peer:        "peer0.org1.example.com (Mock)",
		LatestBlock: l.currentHeight,
	}
}


func (l *MockLedgerService) load() {
	l.mu.Lock()
	defer l.mu.Unlock()
	historyFile := filepath.Join(l.ledgerDir, "ledger_history.json")
	if data, err := os.ReadFile(historyFile); err == nil {
		var list []ChainRecord
		if err := json.Unmarshal(data, &list); err == nil {
			l.history = list
			if len(list) > 0 {
				l.currentHeight = list[len(list)-1].BlockHeight
			}
			for _, rec := range list {
				payloadBytes, _ := json.Marshal(rec.Payload)
				l.stateStore[rec.AssetID] = string(payloadBytes)
			}
		}
	}
}

func (l *MockLedgerService) persist() {
	historyFile := filepath.Join(l.ledgerDir, "ledger_history.json")
	data, _ := json.MarshalIndent(l.history, "", "  ")
	_ = os.WriteFile(historyFile, data, 0644)
}

func (l *MockLedgerService) CommitAsset(assetType, assetID string, payload map[string]interface{}) (txID string, height uint64, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.currentHeight++
	nowStr := time.Now().Format(time.RFC3339)
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s-%s-%d-%s", assetType, assetID, l.currentHeight, nowStr)))
	txID = "0x" + hex.EncodeToString(h.Sum(nil))

	rec := ChainRecord{
		TxID:        txID,
		BlockHeight: l.currentHeight,
		Timestamp:   nowStr,
		AssetType:   assetType,
		AssetID:     assetID,
		Payload:     payload,
	}

	l.history = append(l.history, rec)
	payloadBytes, _ := json.Marshal(payload)
	l.stateStore[assetID] = string(payloadBytes)
	l.persist()

	return txID, l.currentHeight, nil
}

func (l *MockLedgerService) QueryAsset(assetID string) (map[string]interface{}, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	str, ok := l.stateStore[assetID]
	if !ok {
		return nil, false
	}
	var res map[string]interface{}
	if err := json.Unmarshal([]byte(str), &res); err != nil {
		return nil, false
	}
	return res, true
}

func (l *MockLedgerService) GetStats() (totalTx int, currentHeight uint64) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.history) + 24, l.currentHeight
}

func (l *MockLedgerService) GetHistory(assetID string) ([]ChainRecord, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var matches []ChainRecord
	for _, rec := range l.history {
		if rec.AssetID == assetID {
			matches = append(matches, rec)
		}
	}
	return matches, nil
}
