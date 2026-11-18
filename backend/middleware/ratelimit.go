package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"medtrust-backend/model"
)

const maxRateLimitBuckets = 10000

type ipBucket struct {
	count     int
	resetTime time.Time
}

type MemoryRateLimiter struct {
	mu        sync.Mutex
	buckets   map[string]*ipBucket
	limit     int
	window    time.Duration
	stopChan  chan struct{}
	closeOnce sync.Once
}

func NewMemoryRateLimiter(limit int, window time.Duration) *MemoryRateLimiter {
	limiter := &MemoryRateLimiter{
		buckets:  make(map[string]*ipBucket),
		limit:    limit,
		window:   window,
		stopChan: make(chan struct{}),
	}

	// 启动定期清理过期 IP 桶的后台协程，防止内存泄露
	go func() {
		ticker := time.NewTicker(2 * window)
		defer ticker.Stop()
		for {
			select {
			case <-limiter.stopChan:
				return
			case <-ticker.C:
				limiter.mu.Lock()
				now := time.Now()
				for ip, b := range limiter.buckets {
					if now.After(b.resetTime) {
						delete(limiter.buckets, ip)
					}
				}
				limiter.mu.Unlock()
			}
		}
	}()

	return limiter
}

// Stop 优雅终止限流器的后台清理协程
func (limiter *MemoryRateLimiter) Stop() {
	if limiter == nil {
		return
	}
	limiter.closeOnce.Do(func() {
		if limiter.stopChan != nil {
			close(limiter.stopChan)
		}
	})
}

func (limiter *MemoryRateLimiter) Allow(ip string) (bool, int, time.Duration) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := time.Now()
	b, exists := limiter.buckets[ip]

	if !exists || now.After(b.resetTime) {
		if !exists && len(limiter.buckets) >= maxRateLimitBuckets {
			// 先清扫已过期条目，仍满时淘汰最早过期的桶，严防 IP 伪造攻击耗尽内存
			for k, item := range limiter.buckets {
				if now.After(item.resetTime) {
					delete(limiter.buckets, k)
				}
			}
			for len(limiter.buckets) >= maxRateLimitBuckets {
				var oldestIP string
				var earliest time.Time
				first := true
				for k, item := range limiter.buckets {
					if first || item.resetTime.Before(earliest) {
						earliest = item.resetTime
						oldestIP = k
						first = false
					}
				}
				if oldestIP == "" {
					break
				}
				delete(limiter.buckets, oldestIP)
			}
		}
		limiter.buckets[ip] = &ipBucket{
			count:     1,
			resetTime: now.Add(limiter.window),
		}
		return true, limiter.limit - 1, limiter.window
	}

	if b.count < limiter.limit {
		b.count++
		remaining := limiter.limit - b.count
		return true, remaining, b.resetTime.Sub(now)
	}

	retryAfter := b.resetTime.Sub(now)
	if retryAfter < time.Second {
		retryAfter = time.Second
	}
	return false, 0, retryAfter
}

// RateLimitMiddleware 基于客户端 IP 的限流中间件 (防暴力破解与 API 洪水攻击)
func RateLimitMiddleware(limit int, window time.Duration) gin.HandlerFunc {
	limiter := NewMemoryRateLimiter(limit, window)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			ip = "unknown"
		}

		allowed, remaining, retryAfter := limiter.Allow(ip)
		c.Writer.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		c.Writer.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

		if !allowed {
			seconds := int(retryAfter.Seconds())
			if seconds <= 0 {
				seconds = 1
			}
			c.Writer.Header().Set("Retry-After", fmt.Sprintf("%d", seconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, model.Response{
				Code:    429,
				Message: fmt.Sprintf("请求过于频繁，系统已触发安全限流，请 %d 秒后重试", seconds),
			})
			return
		}

		c.Next()
	}
}
