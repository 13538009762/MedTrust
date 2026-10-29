package middleware

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"medtrust-backend/config"
)

// SecurityHeadersMiddleware 注入业界标准 HTTP 纵深防御安全标头
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 防点击劫持 (允许同源 iframe 预览 PDF/影像，防御第三方恶意网站嵌套)
		c.Writer.Header().Set("X-Frame-Options", "SAMEORIGIN")

		// 2. 防 MIME 混淆与嗅探攻击
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")

		// 3. 启用浏览器原生 XSS 过滤器防护
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")

		// 4. 推荐严格 Referrer 策略 (跨域不泄露敏感查询参数与路径)
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// 5. 权限策略 (禁用未使用的敏感硬件接口)
		c.Writer.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		// 6. 生产环境或 TLS 下启用 HSTS (强制一年 HTTPS 传输)
		if config.AppConfig.IsProduction() || c.Request.TLS != nil {
			c.Writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// 7. 内容安全策略 (CSP)
		// 适配 ECharts 动态渲染、Canvas、Blob 附件预览与本地 SSE/WebSocket 调试连接
		cspPolicy := strings.Join([]string{
			"default-src 'self'",
			"script-src 'self' 'unsafe-eval' 'unsafe-inline'",
			"style-src 'self' 'unsafe-inline'",
			"img-src 'self' data: blob:",
			"font-src 'self' data:",
			"connect-src 'self' http://127.0.0.1:* ws://127.0.0.1:* http://localhost:* ws://localhost:*",
			"object-src 'self' blob:",
			"frame-src 'self' blob:",
			"report-uri /api/v1/csp-report",
		}, "; ")

		// 决策点 D4：先默认使用 Content-Security-Policy-Report-Only 观察，支持通过环境变量 CSP_ENFORCE=true 切为强阻断
		enforceCSP := strings.ToLower(os.Getenv("CSP_ENFORCE")) == "true"
		if enforceCSP {
			c.Writer.Header().Set("Content-Security-Policy", cspPolicy)
		} else {
			c.Writer.Header().Set("Content-Security-Policy-Report-Only", cspPolicy)
		}

		c.Next()
	}
}
