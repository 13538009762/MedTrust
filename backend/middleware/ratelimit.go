package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"medtrust-backend/model"
)

type ipBucket struct {
	count      int
	resetTime  time.Time
}

type MemoryRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*ipBucket
	limit   int
	window  time.Duration
}

func NewMemoryRateLimiter(limit int, window time.Duration) *MemoryRateLimiter {
	limiter := &MemoryRateLimiter{
		buckets: make(map[string]*ipBucket),
		limit:   limit,
		window:  window,
	}

	// 启动定期清理过期 IP 桶的后台协程，防止内存泄露
	go func() {
		ticker := time.NewTicker(2 * window)
		for range ticker.C {
			limiter.mu.Lock()
			now := time.Now()
			for ip, b := range limiter.buckets {
				if now.After(b.resetTime) {
					delete(limiter.buckets, ip)
				}
			}
			limiter.mu.Unlock()
		}
	}()

	return limiter
}

func (limiter *MemoryRateLimiter) Allow(ip string) (bool, int, time.Duration) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := time.Now()
	b, exists := limiter.buckets[ip]

	if !exists || now.After(b.resetTime) {
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
