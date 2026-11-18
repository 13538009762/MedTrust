package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"medtrust-backend/config"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
)

// DefaultOrphanedIPFSGracePeriod 运行期清理孤儿 IPFS 文件时的最小修改宽限期，
// 防止误删尚处于「阶段一 IPFS 落盘 -> 阶段二 Fabric 背书 -> 阶段三 MySQL 事务」途中的在途密文。
const DefaultOrphanedIPFSGracePeriod = 15 * time.Minute

// StartAccessRequestCleanup 启动后台定时维护任务（兼容旧入口，无显式取消上下文时使用 Background）
func StartAccessRequestCleanup() {
	StartBackgroundMaintenance(context.Background())
}

// StartBackgroundMaintenance 启动后台定时维护协程：
//   1. 每 1 分钟推进链上存证补偿队列 (ProcessPendingConsistencyTasks) 与未终态病历对账；
//   2. 每 30 分钟扫描并自动标记超期未审批的调阅申请为 EXPIRED；
//   3. 每 1 小时清理超过宽限期的孤儿 IPFS 密文文件；
// 监控 ctx.Done() 实现优雅退出，杜绝 goroutine 泄漏。
func StartBackgroundMaintenance(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	go func() {
		// 启动时延迟 5 秒先执行一次补扫（响应关闭信号立即退出）
		initTimer := time.NewTimer(5 * time.Second)
		select {
		case <-ctx.Done():
			initTimer.Stop()
			return
		case <-initTimer.C:
		}
		cleanExpiredRequests()
		runCompensationMaintenanceTick()

		compTicker := time.NewTicker(1 * time.Minute)
		expireTicker := time.NewTicker(30 * time.Minute)
		ipfsTicker := time.NewTicker(1 * time.Hour)
		defer compTicker.Stop()
		defer expireTicker.Stop()
		defer ipfsTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-compTicker.C:
				runCompensationMaintenanceTick()
			case <-expireTicker.C:
				cleanExpiredRequests()
			case <-ipfsTicker.C:
				_, _, _ = CleanOrphanedIPFSFiles()
			}
		}
	}()
}

func runCompensationMaintenanceTick() {
	if repository.DB == nil || blockchain.DefaultService == nil {
		return
	}
	processed, recovered := ProcessPendingConsistencyTasks()
	if processed > 0 {
		log.Printf("[CompensationWorker] 补偿队列本轮处理 %d 笔，成功恢复 %d 笔", processed, recovered)
	}
	scanned, reconciled, recErr := DefaultConsistencyService.ReconcileAllPendingOrFailed()
	if recErr != nil {
		log.Printf("[CompensationWorker] 病历一致性扫描异常: %v", recErr)
	} else if scanned > 0 {
		log.Printf("[CompensationWorker] 病历一致性巡检扫描 %d 份，自动对齐 %d 份", scanned, reconciled)
	}
}

func cleanExpiredRequests() {
	now := time.Now()
	// 只更新状态为 PENDING、且明确设置了过期时间（大于 2000 年）并且当前时间已超期的记录
	rowsAffected, err := repository.DefaultAuditRepository.ExpirePendingAccessRequests(now)
	if err != nil {
		log.Printf("[CleanupService] 扫描过期调阅申请出错: %v", err)
		return
	}

	if rowsAffected > 0 {
		log.Printf("[CleanupService] 已自动将 %d 笔超时未审批的跨院调阅申请置为 EXPIRED 状态", rowsAffected)
	}
}

// CleanOrphanedIPFSFiles 使用默认 15 分钟安全宽限期清理孤儿 IPFS 密文块，保护在途上传不被误删。
func CleanOrphanedIPFSFiles() (int, int64, error) {
	return CleanOrphanedIPFSFilesWithGrace(DefaultOrphanedIPFSGracePeriod)
}

// CleanOrphanedIPFSFilesWithGrace 扫描并清理 IPFS 本地存储目录下与当前业务数据库无关联的孤儿密文文件。
// 当 gracePeriod > 0 时，修改时间在宽限期以内的新文件会被跳过，防止并发上传在写库前被误删。
func CleanOrphanedIPFSFilesWithGrace(gracePeriod time.Duration) (int, int64, error) {
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

	// 1. 先从数据库读取当前所有有效病历附件的 ipfs_cid；若查询失败严禁继续清理，防止空集合误删全部密文
	activeCIDs, err := repository.DefaultAuditRepository.GetActiveMedicalFileCIDs()
	if err != nil {
		return 0, 0, fmt.Errorf("查询活跃病历 IPFS CID 失败，中止孤儿清理: %w", err)
	}
	validCIDs := make(map[string]bool, len(activeCIDs))
	for _, cid := range activeCIDs {
		if trimmed := strings.TrimSpace(cid); trimmed != "" {
			validCIDs[trimmed] = true
		}
	}

	entries, err := os.ReadDir(storageDir)
	if err != nil {
		return 0, 0, err
	}

	// 2. 遍历 IPFS 存储目录，清理超过安全宽限期且与数据库无关联的历史遗留块
	cleanedCount := 0
	var freedBytes int64 = 0
	now := time.Now()

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == ".gitkeep" || strings.HasPrefix(name, ".") {
			continue
		}

		if !validCIDs[name] {
			info, infoErr := entry.Info()
			if infoErr != nil {
				continue
			}
			if gracePeriod > 0 && now.Sub(info.ModTime()) < gracePeriod {
				continue
			}
			fileSize := info.Size()
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
