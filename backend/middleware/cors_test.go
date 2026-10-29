package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"medtrust-backend/config"
)

func setupCORSTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORSMiddleware())
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})
	return r
}

func TestCORS_AllowedOrigin(t *testing.T) {
	config.AppConfig.AppEnv = "demo"
	config.AppConfig.Security.CORSAllowedOrigins = ""

	router := setupCORSTestRouter()

	req, _ := http.NewRequest("GET", "/ping", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("预期 200，实际: %d", w.Code)
	}

	allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != "http://localhost:5173" {
		t.Fatalf("预期 Access-Control-Allow-Origin 为 http://localhost:5173，实际: %s", allowOrigin)
	}

	creds := w.Header().Get("Access-Control-Allow-Credentials")
	if creds != "true" {
		t.Fatalf("预期 Access-Control-Allow-Credentials 为 true，实际: %s", creds)
	}
}

func TestCORS_AllowedOrigin_OPTIONS(t *testing.T) {
	config.AppConfig.AppEnv = "demo"
	config.AppConfig.Security.CORSAllowedOrigins = "http://localhost:5173,http://127.0.0.1:5173"

	router := setupCORSTestRouter()

	req, _ := http.NewRequest("OPTIONS", "/ping", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS 预检预期 204 No Content，实际: %d", w.Code)
	}

	allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != "http://localhost:5173" {
		t.Fatalf("预期 Access-Control-Allow-Origin 为 http://localhost:5173，实际: %s", allowOrigin)
	}
}

func TestCORS_DisallowedOrigin_OPTIONS_Forbidden(t *testing.T) {
	config.AppConfig.AppEnv = "demo"
	config.AppConfig.Security.CORSAllowedOrigins = "http://localhost:5173,http://127.0.0.1:5173"

	router := setupCORSTestRouter()

	req, _ := http.NewRequest("OPTIONS", "/ping", nil)
	req.Header.Set("Origin", "http://malicious-attacker.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("未授权源的 OPTIONS 预期 403 Forbidden，实际: %d", w.Code)
	}

	allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != "" {
		t.Fatalf("未授权源不应返回 Access-Control-Allow-Origin，实际: %s", allowOrigin)
	}
}

func TestCORS_DisallowedOrigin_GET_NoHeader(t *testing.T) {
	config.AppConfig.AppEnv = "demo"
	config.AppConfig.Security.CORSAllowedOrigins = "http://localhost:5173,http://127.0.0.1:5173"

	router := setupCORSTestRouter()

	req, _ := http.NewRequest("GET", "/ping", nil)
	req.Header.Set("Origin", "http://malicious-attacker.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 响应不会被后端拦截（因为可能是某些客户端直连），但不带 Access-Control-Allow-Origin 标头，浏览器将拒绝前端 JS 读取
	allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != "" {
		t.Fatalf("未授权源的 GET 请求不应包含 Access-Control-Allow-Origin，实际: %s", allowOrigin)
	}
}
