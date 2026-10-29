package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func setupRateLimitTestRouter(limit int, window time.Duration) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimitMiddleware(limit, window))
	r.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return r
}

func TestRateLimit_UnderLimit(t *testing.T) {
	router := setupRateLimitTestRouter(5, time.Minute)

	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("GET", "/api/test", nil)
		req.RemoteAddr = "192.168.1.100:1234"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("第 %d 次请求应为 200 OK，实际为: %d", i+1, w.Code)
		}
	}
}

func TestRateLimit_ExceedLimit_Returns429(t *testing.T) {
	router := setupRateLimitTestRouter(3, time.Minute)

	// 前 3 次请求成功
	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest("GET", "/api/test", nil)
		req.RemoteAddr = "192.168.1.101:5678"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("前 3 次应返回 200，实际: %d", w.Code)
		}
	}

	// 第 4 次请求应被拦截 (429 Too Many Requests)
	req4, _ := http.NewRequest("GET", "/api/test", nil)
	req4.RemoteAddr = "192.168.1.101:5678"
	w4 := httptest.NewRecorder()
	router.ServeHTTP(w4, req4)

	if w4.Code != http.StatusTooManyRequests {
		t.Fatalf("超额请求预期返回 429 Too Many Requests，实际: %d", w4.Code)
	}

	retryAfter := w4.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Fatalf("429 响应头必须包含 Retry-After")
	}
}
