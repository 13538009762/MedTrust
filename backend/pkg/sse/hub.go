package sse

import (
	"sync"
)

// NotifyEvent 实时通知事件
type NotifyEvent struct {
	Type    string      `json:"type"`    // "access_request" | "request_approved" | "request_rejected" | "break_glass"
	Title   string      `json:"title"`   // 通知标题
	Message string      `json:"message"` // 通知具体内容
	Payload interface{} `json:"payload"` // 附带业务载荷
}

// Hub 全局 SSE 客户端连接管理器
type Hub struct {
	mu      sync.RWMutex
	clients map[uint64][]chan NotifyEvent
}

var Default = &Hub{
	clients: make(map[uint64][]chan NotifyEvent),
}

// Subscribe 注册一个用户的 SSE 接收通道
func (h *Hub) Subscribe(userID uint64) chan NotifyEvent {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan NotifyEvent, 16)
	h.clients[userID] = append(h.clients[userID], ch)
	return ch
}

// Unsubscribe 注销并关闭用户的 SSE 通道
func (h *Hub) Unsubscribe(userID uint64, ch chan NotifyEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	channels := h.clients[userID]
	for i, c := range channels {
		if c == ch {
			h.clients[userID] = append(channels[:i], channels[i+1:]...)
			close(ch)
			break
		}
	}
	if len(h.clients[userID]) == 0 {
		delete(h.clients, userID)
	}
}

// Publish 向指定用户的所有在线客户端广播事件
func (h *Hub) Publish(userID uint64, event NotifyEvent) {
	h.mu.RLock()
	channels := h.clients[userID]
	h.mu.RUnlock()

	for _, ch := range channels {
		select {
		case ch <- event:
		default:
			// 缓冲区已满时非阻塞丢弃，避免阻塞业务主协程
		}
	}
}

// BroadcastAll 向全平台所有在线连接广播（例如全局公告或高危预警）
func (h *Hub) BroadcastAll(event NotifyEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, channels := range h.clients {
		for _, ch := range channels {
			select {
			case ch <- event:
			default:
			}
		}
	}
}

// OnlineCount 当前在线客户端总数
func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	total := 0
	for _, channels := range h.clients {
		total += len(channels)
	}
	return total
}
