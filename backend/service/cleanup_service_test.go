package service_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"medtrust-backend/config"
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
	"medtrust-backend/service"
)

func TestCleanOrphanedIPFSFiles(t *testing.T) {
	_, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	db, err := repository.InitDB()
	if err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}
	repository.SeedInitialData(db)

	tempDir := t.TempDir()
	config.AppConfig.IPFS.StorageDir = tempDir

	// 创建一个超过宽限期（1 小时前）的孤立历史测试大文件，以及一个刚写入的在途上传文件
	dummyOrphan := filepath.Join(tempDir, "QmDummyOrphanFileForTesting123456789")
	_ = os.WriteFile(dummyOrphan, []byte("large historical invalid ciphertext chunk"), 0644)
	oldTime := time.Now().Add(-1 * time.Hour)
	_ = os.Chtimes(dummyOrphan, oldTime, oldTime)

	inFlightFile := filepath.Join(tempDir, "QmInFlightUploadChunk123456789")
	_ = os.WriteFile(inFlightFile, []byte("brand new ciphertext being uploaded"), 0644)

	// 创建一个 .gitkeep
	gitkeep := filepath.Join(tempDir, ".gitkeep")
	_ = os.WriteFile(gitkeep, []byte(""), 0644)

	// 执行清理
	cleaned, bytesFreed, err := service.CleanOrphanedIPFSFiles()
	if err != nil {
		t.Fatalf("清理执行异常: %v", err)
	}

	if cleaned != 1 {
		t.Fatalf("期望仅清理 1 个历史孤立文件，实际清理 %d", cleaned)
	}
	if bytesFreed <= 0 {
		t.Fatalf("期望释放的字节数 > 0，实际 %d", bytesFreed)
	}

	// 确认历史孤立文件已被移除
	if _, err := os.Stat(dummyOrphan); !os.IsNotExist(err) {
		t.Fatalf("历史孤立文件未被成功删除!")
	}

	// 确认在途刚写入文件受到 15 分钟宽限期保护不被误删
	if _, err := os.Stat(inFlightFile); err != nil {
		t.Fatalf("【竞态漏洞】在途刚上传文件被孤儿清理误删: %v", err)
	}

	// 确认 .gitkeep 得到保护
	if _, err := os.Stat(gitkeep); err != nil {
		t.Fatalf(".gitkeep 文件应当被保留!")
	}

	t.Logf("✅ CleanOrphanedIPFSFiles 成功清理 %d 个历史孤立文件并保护在途上传文件，释放 %d 字节", cleaned, bytesFreed)
}

func TestCleanRealIPFSStorage(t *testing.T) {
	_, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	db, err := repository.InitDB()
	if err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}
	repository.SeedInitialData(db)

	simulatedDir := t.TempDir()
	config.AppConfig.IPFS.StorageDir = simulatedDir
	_ = os.WriteFile(filepath.Join(simulatedDir, ".gitkeep"), []byte(""), 0644)
	orphanPath := filepath.Join(simulatedDir, "QmSimulatedOrphanFile987654321")
	_ = os.WriteFile(orphanPath, []byte("orphan"), 0644)
	oldTime := time.Now().Add(-1 * time.Hour)
	_ = os.Chtimes(orphanPath, oldTime, oldTime)

	cleaned, bytesFreed, err := service.CleanOrphanedIPFSFiles()
	if err != nil {
		t.Fatalf("存储目录清理出错: %v", err)
	}
	if cleaned != 1 {
		t.Fatalf("期望清理 1 个历史文件，实际 %d", cleaned)
	}
	t.Logf("✅ 存储目录清理测试完成: 清理文件数=%d, 释放空间=%d 字节", cleaned, bytesFreed)
}

func TestCompensation_RevokeAuthPreservesGrantTxIDAndAdvancesRetry(t *testing.T) {
	_, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if _, err := repository.InitDB(); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	blockchain.InitBlockchainService(filepath.Join(t.TempDir(), "ledger"), blockchain.FabricGatewayConfig{Mode: "mock"})

	authNo := fmt.Sprintf("AUTH-COMP-REV-%d", time.Now().UnixNano())
	auth := model.Authorization{
		AuthNo:         authNo,
		PatientID:      4,
		AuthTargetType: "DOCTOR",
		AuthTargetID:   2,
		ScopeType:      "ALL",
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(24 * time.Hour),
		Status:         "REVOKE_FAILED",
		FabricTxID:     "ORIG_GRANT_TX_KEEP_ME",
		RevokeError:    "temp fabric error",
		CreatedAt:      time.Now(),
	}
	if err := repository.DB.Create(&auth).Error; err != nil {
		t.Fatalf("Create auth failed: %v", err)
	}
	defer repository.DB.Where("auth_no = ?", authNo).Delete(&model.Authorization{})

	task := model.ConsistencyTask{
		BusinessType: "REVOKE_AUTH",
		BusinessID:   authNo,
		PayloadJSON:  fmt.Sprintf(`{"auth_no":%q,"status":"REVOKED"}`, authNo),
		Status:       "PENDING_RETRY",
	}
	if err := repository.DB.Create(&task).Error; err != nil {
		t.Fatalf("Create task failed: %v", err)
	}
	defer repository.DB.Where("business_id = ?", authNo).Delete(&model.ConsistencyTask{})

	_, recovered := service.ProcessPendingConsistencyTasks()
	if recovered == 0 {
		t.Fatalf("expected REVOKE_AUTH task to recover")
	}

	var updated model.Authorization
	if err := repository.DB.Where("auth_no = ?", authNo).First(&updated).Error; err != nil {
		t.Fatalf("Reload auth failed: %v", err)
	}
	if updated.FabricTxID != "ORIG_GRANT_TX_KEEP_ME" {
		t.Fatalf("【数据覆盖】REVOKE_AUTH 补偿覆盖了原授权 FabricTxID: got=%s", updated.FabricTxID)
	}
	if updated.Status != "REVOKED" || updated.RevokeTxID == "" || updated.RevokeError != "" {
		t.Fatalf("预期 Status=REVOKED, RevokeTxID 非空且 RevokeError 清空，实际: %+v", updated)
	}

	// 验证失败任务（如 Fabric 离线）不会因 continue 跳过 retry_count 递进
	origBC := blockchain.DefaultService
	blockchain.DefaultService = blockchain.NewFabricGatewayService(blockchain.FabricGatewayConfig{
		Mode:         "fabric",
		Enabled:      true,
		PeerEndpoint: "127.0.0.1:19997",
		TLSCertPath:  "none",
		CertPath:     "none",
		KeyPath:      "none",
	})
	defer func() {
		if c, ok := blockchain.DefaultService.(interface{ Close() error }); ok {
			_ = c.Close()
		}
		blockchain.DefaultService = origBC
	}()

	failID := fmt.Sprintf("AUTH-COMP-FAIL-%d", time.Now().UnixNano())
	failTask := model.ConsistencyTask{
		BusinessType: "AUTHORIZATION",
		BusinessID:   failID,
		PayloadJSON:  `{"auth_no":"fail"}`,
		Status:       "PENDING_RETRY",
		RetryCount:   service.MaxChainCompensationRetries - 1,
	}
	repository.DB.Create(&failTask)
	defer repository.DB.Where("business_id = ?", failID).Delete(&model.ConsistencyTask{})

	service.ProcessPendingConsistencyTasks()
	var afterFail model.ConsistencyTask
	repository.DB.First(&afterFail, failTask.ID)
	if afterFail.Status != "FAILED" || afterFail.RetryCount != service.MaxChainCompensationRetries {
		t.Fatalf("达到最大重试次数后应置为 FAILED，实际 status=%s retry=%d", afterFail.Status, afterFail.RetryCount)
	}
}
