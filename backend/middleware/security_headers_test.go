package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupSecurityHeadersTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityHeadersMiddleware())
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return r
}

func TestSecurityHeaders_DefaultReportOnly(t *testing.T) {
	_ = os.Setenv("CSP_ENFORCE", "false")
	defer os.Unsetenv("CSP_ENFORCE")

	router := setupSecurityHeadersTestRouter()
	req, _ := http.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("预期 200，实际: %d", w.Code)
	}

	if w.Header().Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Errorf("X-Frame-Options 缺失或不为 SAMEORIGIN: %s", w.Header().Get("X-Frame-Options"))
	}
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("X-Content-Type-Options 缺失或不为 nosniff: %s", w.Header().Get("X-Content-Type-Options"))
	}
	if w.Header().Get("X-XSS-Protection") != "1; mode=block" {
		t.Errorf("X-XSS-Protection 缺失: %s", w.Header().Get("X-XSS-Protection"))
	}
	if w.Header().Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
		t.Errorf("Referrer-Policy 缺失: %s", w.Header().Get("Referrer-Policy"))
	}
	if w.Header().Get("Content-Security-Policy-Report-Only") == "" {
		t.Errorf("默认模式下应输出 Content-Security-Policy-Report-Only 标头")
	}
	if w.Header().Get("Content-Security-Policy") != "" {
		t.Errorf("默认模式下不应输出 Content-Security-Policy 强拦截标头")
	}
}

func TestSecurityHeaders_EnforceMode(t *testing.T) {
	_ = os.Setenv("CSP_ENFORCE", "true")
	defer os.Unsetenv("CSP_ENFORCE")

	router := setupSecurityHeadersTestRouter()
	req, _ := http.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("Content-Security-Policy") == "" {
		t.Errorf("强阻断模式下应输出 Content-Security-Policy 标头")
	}
	if w.Header().Get("Content-Security-Policy-Report-Only") != "" {
		t.Errorf("强阻断模式下不应输出 Content-Security-Policy-Report-Only")
	}
}
