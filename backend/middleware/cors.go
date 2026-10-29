package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"medtrust-backend/config"
)

// CORSMiddleware 生产级严格 CORS 白名单中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// 构造当前允许的 Origin 白名单集合
		allowedMap := make(map[string]bool)

		// 1. 读取配置中的白名单列表
		cfgOrigins := strings.TrimSpace(config.AppConfig.Security.CORSAllowedOrigins)
		if cfgOrigins != "" {
			for _, o := range strings.Split(cfgOrigins, ",") {
				o = strings.TrimSpace(o)
				if o != "" {
					allowedMap[o] = true
				}
			}
		}

		// 2. 在 demo/开发模式下，若未配置则注入安全的本地域名默认白名单
		if len(allowedMap) == 0 && !config.AppConfig.IsProduction() {
			defaultOrigins := []string{
				"http://localhost:5173",
				"http://127.0.0.1:5173",
				"http://localhost:3000",
				"http://127.0.0.1:3000",
				"http://localhost:8080",
				"http://127.0.0.1:8080",
			}
			for _, d := range defaultOrigins {
				allowedMap[d] = true
			}
		}

		// 如果未提供 Origin 标头 (同源请求或命令行工具)，直接放行
		if origin == "" {
			c.Next()
			return
		}

		isAllowed := allowedMap["*"] || allowedMap[origin]

		if isAllowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Source, X-Idempotency-Key")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
			c.Writer.Header().Set("Access-Control-Max-Age", "86400")
			c.Writer.Header().Set("Vary", "Origin")

			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}
			c.Next()
			return
		}

		// 跨域来源不在白名单内：OPTIONS 预检请求直接拦截返回 403
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		// 非预检跨域请求不设置 Access-Control-Allow-Origin，浏览器将强制遵循同源策略阻断响应读取
		c.Next()
	}
}
