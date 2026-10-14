package service

import (
	"log"
	"time"

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
