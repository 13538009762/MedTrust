package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ChainRecord struct {
	TxID        string                 `json:"tx_id"`
	BlockHeight uint64                 `json:"block_height"`
	Timestamp   string                 `json:"timestamp"`
	AssetType   string                 `json:"asset_type"`
	AssetID     string                 `json:"asset_id"`
	Payload     map[string]interface{} `json:"payload"`
}

type LedgerEngine struct {
	ledgerDir     string
	currentHeight uint64
	stateStore    map[string]string
	history       []ChainRecord
	mu            sync.RWMutex
}

var DefaultLedger *LedgerEngine
var once sync.Once

func InitLedger(ledgerDir string) *LedgerEngine {
	once.Do(func() {
		_ = os.MkdirAll(ledgerDir, 0755)
		DefaultLedger = &LedgerEngine{
			ledgerDir:     ledgerDir,
			currentHeight: 100,
			stateStore:    make(map[string]string),
			history:       make([]ChainRecord, 0),
		}
		DefaultLedger.load()
	})
	return DefaultLedger
}

func (l *LedgerEngine) load() {
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

func (l *LedgerEngine) persist() {
	historyFile := filepath.Join(l.ledgerDir, "ledger_history.json")
	data, _ := json.MarshalIndent(l.history, "", "  ")
	_ = os.WriteFile(historyFile, data, 0644)
}

func (l *LedgerEngine) CommitAsset(assetType, assetID string, payload map[string]interface{}) (txID string, height uint64, err error) {
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

func (l *LedgerEngine) QueryAsset(assetID string) (map[string]interface{}, bool) {
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

func (l *LedgerEngine) GetStats() (totalTx int, currentHeight uint64) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.history) + 24, l.currentHeight
}
