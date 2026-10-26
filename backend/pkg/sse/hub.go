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
	clients map[uint64]map[chan NotifyEvent]bool
}

var Default = &Hub{
	clients: make(map[uint64]map[chan NotifyEvent]bool),
}

// Subscribe 注册一个用户的 SSE 接收通道
func (h *Hub) Subscribe(userID uint64) chan NotifyEvent {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan NotifyEvent, 32)
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[chan NotifyEvent]bool)
	}
	h.clients[userID][ch] = true
	return ch
}

// Unsubscribe 注销并关闭用户的 SSE 通道 (带重复关闭保护)
func (h *Hub) Unsubscribe(userID uint64, ch chan NotifyEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if userClients, ok := h.clients[userID]; ok {
		if userClients[ch] {
			delete(userClients, ch)
			close(ch)
		}
		if len(userClients) == 0 {
			delete(h.clients, userID)
		}
	}
}

// Publish 向指定用户的所有在线客户端广播事件 (读锁快照 + recover 安全投递)
func (h *Hub) Publish(userID uint64, event NotifyEvent) {
	h.mu.RLock()
	var targets []chan NotifyEvent
	if userClients, ok := h.clients[userID]; ok {
		targets = make([]chan NotifyEvent, 0, len(userClients))
		for ch := range userClients {
			targets = append(targets, ch)
		}
	}
	h.mu.RUnlock()

	for _, ch := range targets {
		safeSend(ch, event)
	}
}

// BroadcastAll 向全平台所有在线连接广播（例如全局公告或高危预警）
func (h *Hub) BroadcastAll(event NotifyEvent) {
	h.mu.RLock()
	var targets []chan NotifyEvent
	for _, userClients := range h.clients {
		for ch := range userClients {
			targets = append(targets, ch)
		}
	}
	h.mu.RUnlock()

	for _, ch := range targets {
		safeSend(ch, event)
	}
}

// safeSend 规避向关闭通道发送引发的进程级崩溃
func safeSend(ch chan NotifyEvent, event NotifyEvent) {
	defer func() {
		if r := recover(); r != nil {
			// 安全吸收并发极少数情况下通道关闭后的发送 panic，保证主业务进程绝对不崩溃
		}
	}()
	select {
	case ch <- event:
	default:
		// 缓冲区已满时非阻塞丢弃，避免阻塞业务主协程
	}
}

// OnlineCount 当前在线客户端总数
func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	total := 0
	for _, userClients := range h.clients {
		total += len(userClients)
	}
	return total
}
