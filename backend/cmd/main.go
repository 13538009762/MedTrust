package main

import (
	"fmt"
	"log"
	"time"

	"medtrust-backend/config"
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
	"medtrust-backend/router"
	"medtrust-backend/service"
)

func main() {
	cfg, err := config.LoadConfig("./config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 1. 初始化数据库与自动迁移
	db, err := repository.InitDB()
	if err != nil {
		log.Fatalf("Failed to init DB: %v", err)
	}
	fmt.Println("[MedTrust] MySQL Database Connected & AutoMigrated Successfully")

	// 显式执行标准测试种子数据注入
	repository.SeedInitialData(db)

	// 注册区块链底层存证交易与本地数据双写记录器 (支持论文 4.3.2 节 blockchain_transactions)
	blockchain.TxRecorder = func(bType, bID, txID string, blockNo uint64, channel, cc, status string) {
		if repository.DB != nil {
			rec := model.BlockchainTransaction{
				BusinessType:  bType,
				BusinessID:    bID,
				TxID:          txID,
				BlockNo:       blockNo,
				ChannelName:   channel,
				ChaincodeName: cc,
				Status:        status,
				CreatedAt:     time.Now(),
			}
			_ = repository.DB.Create(&rec).Error
		}
	}

	// 2. 双模驱动初始化 (Blockchain: Fabric Live vs. Local Mock, IPFS: Kubo Live vs. Local Mock)
	blockchain.InitBlockchainService(cfg.Blockchain.LedgerDir, blockchain.FabricGatewayConfig{
		Mode:         cfg.Blockchain.Mode,
		Enabled:      cfg.Blockchain.Fabric.Enabled,
		PeerEndpoint: cfg.Blockchain.Fabric.PeerEndpoint,
		GatewayPeer:  cfg.Blockchain.Fabric.GatewayPeer,
		MSPID:        cfg.Blockchain.Fabric.MSPID,
		ChannelID:    cfg.Blockchain.ChannelID,
		ChaincodeID:  cfg.Blockchain.ChaincodeID,
		TLSCertPath:  cfg.Blockchain.Fabric.TLSCertPath,
		CertPath:     cfg.Blockchain.Fabric.CertPath,
		KeyPath:      cfg.Blockchain.Fabric.KeyPath,
	})
	service.InitMedicalService(cfg.IPFS.Mode, cfg.IPFS.APIURL, cfg.IPFS.StorageDir)
	service.InitAccessEngine(cfg.Risk.LowThreshold, cfg.Risk.HighThreshold)
	service.StartAccessRequestCleanup()
	go service.EnsureBaselineLedgerAnchored()
	fmt.Println("[MedTrust] Fabric Ledger & IPFS Engine Initialized with Baseline Assets")

	r := router.SetupRouter()
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Printf("[MedTrust] Server listening on http://127.0.0.1%s\n", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
