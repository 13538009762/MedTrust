package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"medtrust-backend/pkg/sse"
)

type NotificationController struct{}

var DefaultNotificationController = &NotificationController{}

// Stream 建立客户端 SSE 长连接
// GET /api/v1/notifications/stream?token=...
func (ctrl *NotificationController) Stream(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.Status(http.StatusUnauthorized)
		return
	}
	userID := userIDVal.(uint64)

	// 设置 SSE 响应标头
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	// 订阅用户通知通道
	ch := sse.Default.Subscribe(userID)
	defer sse.Default.Unsubscribe(userID, ch)

	// 发送初次握手确认
	_, _ = fmt.Fprintf(c.Writer, "event: connected\ndata: {\"online\":true,\"user_id\":%d}\n\n", userID)
	c.Writer.Flush()

	// 启动心跳定时器（每25秒一次，保持 TCP 连接存活）
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	notifyClose := c.Request.Context().Done()

	for {
		select {
		case <-notifyClose:
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(event)
			if err == nil {
				_, _ = fmt.Fprintf(c.Writer, "event: notify\ndata: %s\n\n", string(data))
				c.Writer.Flush()
			}
		case <-ticker.C:
			_, _ = fmt.Fprintf(c.Writer, "event: heartbeat\ndata: {\"time\":%d}\n\n", time.Now().Unix())
			c.Writer.Flush()
		}
	}
}
