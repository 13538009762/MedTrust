package service_test

import (
	"os"
	"path/filepath"
	"testing"

	"medtrust-backend/config"
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

	// 创建一个孤立的历史无关联测试大文件
	dummyOrphan := filepath.Join(tempDir, "QmDummyOrphanFileForTesting123456789")
	_ = os.WriteFile(dummyOrphan, []byte("large historical invalid ciphertext chunk"), 0644)

	// 创建一个 .gitkeep
	gitkeep := filepath.Join(tempDir, ".gitkeep")
	_ = os.WriteFile(gitkeep, []byte(""), 0644)

	// 执行清理
	cleaned, bytesFreed, err := service.CleanOrphanedIPFSFiles()
	if err != nil {
		t.Fatalf("清理执行异常: %v", err)
	}

	if cleaned < 1 {
		t.Fatalf("期望至少清理 1 个孤立文件，实际清理 %d", cleaned)
	}
	if bytesFreed <= 0 {
		t.Fatalf("期望释放的字节数 > 0，实际 %d", bytesFreed)
	}

	// 确认孤立文件已被移除
	if _, err := os.Stat(dummyOrphan); !os.IsNotExist(err) {
		t.Fatalf("孤立文件未被成功删除!")
	}

	// 确认 .gitkeep 得到保护
	if _, err := os.Stat(gitkeep); err != nil {
		t.Fatalf(".gitkeep 文件应当被保留!")
	}

	t.Logf("✅ CleanOrphanedIPFSFiles 成功清理 %d 个孤立文件，释放 %d 字节，.gitkeep 安全保留", cleaned, bytesFreed)
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
	_ = os.WriteFile(filepath.Join(simulatedDir, "QmSimulatedOrphanFile987654321"), []byte("orphan"), 0644)

	cleaned, bytesFreed, err := service.CleanOrphanedIPFSFiles()
	if err != nil {
		t.Fatalf("存储目录清理出错: %v", err)
	}
	t.Logf("✅ 存储目录清理测试完成: 清理文件数=%d, 释放空间=%d 字节", cleaned, bytesFreed)
}

