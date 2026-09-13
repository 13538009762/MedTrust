package main

import (
	"fmt"
	"log"

	"medtrust-backend/config"
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

	// 初始化数据库
	_, err = repository.InitDB()
	if err != nil {
		log.Fatalf("Failed to init DB: %v", err)
	}
	fmt.Println("[MedTrust] MySQL Database Connected & AutoMigrated Successfully")

	// 初始化区块链与 IPFS 服务
	blockchain.InitLedger(cfg.Blockchain.LedgerDir)
	service.InitMedicalService(cfg.IPFS.APIURL, cfg.IPFS.StorageDir)
	service.InitAccessEngine(cfg.Risk.LowThreshold, cfg.Risk.HighThreshold)
	service.EnsureBaselineLedgerAnchored()
	fmt.Println("[MedTrust] Fabric Ledger & IPFS Engine Initialized with Baseline Assets")

	r := router.SetupRouter()
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Printf("[MedTrust] Server listening on http://127.0.0.1%s\n", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
