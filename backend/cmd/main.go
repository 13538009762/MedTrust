package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Config security validation failed: %v", err)
	}
	fmt.Printf("[MedTrust] Configuration validated successfully (Mode: %s)\n", cfg.AppEnv)

	// 1. 初始化数据库与自动迁移（内部已完成幂等标准测试种子数据注入）
	_, err = repository.InitDB()
	if err != nil {
		log.Fatalf("Failed to init DB: %v", err)
	}
	fmt.Println("[MedTrust] MySQL Database Connected & AutoMigrated Successfully")

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

	// 2. 初始化 Redis 客户端并检测连通性 (若无 Redis 则自动平滑降级至 MemoryCache 内存缓存)
	repository.InitRedis()

	// 3. 双模驱动初始化 (Blockchain: Fabric Live vs. Local Mock, IPFS: Kubo Live vs. Local Mock)
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

	// 4. 固化标准种子病历至 Fabric 账本并清理孤立历史文件
	service.EnsureBaselineLedgerAnchored()
	_, _, _ = service.CleanOrphanedIPFSFiles()
	fmt.Println("[MedTrust] Fabric Ledger & IPFS Engine Initialized with Baseline Assets")

	r := router.SetupRouter()
	_ = r.SetTrustedProxies([]string{"127.0.0.1", "::1"})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:           addr,
		Handler:        r,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   60 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB HTTP 标头限制 (防 DOS)
	}

	go func() {
		fmt.Printf("[MedTrust] Hardened HTTP Server listening on http://127.0.0.1%s\n", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server ListenAndServe failed: %v", err)
		}
	}()

	// 监听系统中断信号实现优雅关机 (Graceful Shutdown)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	fmt.Println("\n[MedTrust] Shutting down server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	fmt.Println("[MedTrust] Server exited cleanly.")
}
