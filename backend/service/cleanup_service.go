package service

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"medtrust-backend/config"
	"medtrust-backend/model"
	"medtrust-backend/repository"
)

// StartAccessRequestCleanup 启动后台定时任务：定期扫描并自动标记超期的调阅申请
func StartAccessRequestCleanup() {
	go func() {
		// 启动时延迟 5 秒先执行一次补扫
		time.Sleep(5 * time.Second)
		cleanExpiredRequests()

		// 后续每 30 分钟轮询检测一次
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			cleanExpiredRequests()
		}
	}()
}

func cleanExpiredRequests() {
	if repository.DB == nil {
		return
	}

	now := time.Now()
	// 只更新状态为 PENDING、且明确设置了过期时间（大于 2000 年）并且当前时间已超期的记录
	res := repository.DB.Model(&model.AccessRequest{}).
		Where("status = ? AND expire_at > ? AND expire_at < ?", "PENDING", time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), now).
		Updates(map[string]interface{}{
			"status":   "EXPIRED",
			"decision": "EXPIRED",
		})

	if res.Error != nil {
		log.Printf("[CleanupService] 扫描过期调阅申请出错: %v", res.Error)
		return
	}

	if res.RowsAffected > 0 {
		log.Printf("[CleanupService] 已自动将 %d 笔超时未审批的跨院调阅申请置为 EXPIRED 状态", res.RowsAffected)
	}
}

// CleanOrphanedIPFSFiles 扫描并清理 IPFS 本地存储目录下与当前业务数据库已存病历无关联的历史遗留/无效临时大文件
func CleanOrphanedIPFSFiles() (int, int64, error) {
	storageDir := config.AppConfig.IPFS.StorageDir
	if storageDir == "" {
		storageDir = "./data/ipfs_storage"
	}

	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return 0, 0, err
	}

	// 确保存储目录下存在 .gitkeep 文件以保留目录结构
	gitkeepPath := filepath.Join(storageDir, ".gitkeep")
	if _, err := os.Stat(gitkeepPath); os.IsNotExist(err) {
		_ = os.WriteFile(gitkeepPath, []byte(""), 0644)
	}

	entries, err := os.ReadDir(storageDir)
	if err != nil {
		return 0, 0, err
	}

	// 1. 从数据库读取当前所有有效病历附件的 ipfs_cid
	validCIDs := make(map[string]bool)
	if repository.DB != nil {
		var activeCIDs []string
		_ = repository.DB.Model(&model.MedicalFile{}).Where("ipfs_cid != ''").Pluck("ipfs_cid", &activeCIDs).Error
		for _, cid := range activeCIDs {
			validCIDs[strings.TrimSpace(cid)] = true
		}
	}

	// 2. 遍历 IPFS 存储目录，清理与数据库无关联的历史遗留块
	cleanedCount := 0
	var freedBytes int64 = 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == ".gitkeep" || strings.HasPrefix(name, ".") {
			continue
		}

		if !validCIDs[name] {
			info, err := entry.Info()
			fileSize := int64(0)
			if err == nil {
				fileSize = info.Size()
			}
			filePath := filepath.Join(storageDir, name)
			if err := os.Remove(filePath); err == nil {
				cleanedCount++
				freedBytes += fileSize
			}
		}
	}

	if cleanedCount > 0 {
		log.Printf("[IPFS-Cleanup] ✅ 已成功清理 %d 份与当前数据库无关联的历史遗留/测试 IPFS 密文块，释放磁盘空间: %.2f MB", cleanedCount, float64(freedBytes)/(1024*1024))
	} else {
		log.Printf("[IPFS-Cleanup] ✅ IPFS 本地存储块与业务数据库一致，无需清理 (有效密文块: %d)", len(validCIDs))
	}

	return cleanedCount, freedBytes, nil
}
