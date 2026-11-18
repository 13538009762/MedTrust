package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"medtrust-backend/config"
	"medtrust-backend/model"
	"medtrust-backend/repository"
)

// TicketPayload 一次性接入票据序列化载荷（JSON 结构化存储，杜绝分隔符注入）
type TicketPayload struct {
	UserID       uint64 `json:"user_id"`
	UserNo       string `json:"user_no"`
	Role         string `json:"role"`
	Username     string `json:"username"`
	HospitalID   uint64 `json:"hospital_id"`
	DepartmentID uint64 `json:"department_id"`
}

var ticketAllowedMedicalFilePathRe = regexp.MustCompile(`^/api/v1/medical-files/[0-9]+/(view|download)$`)

func isTicketAllowedEndpoint(method, path string) bool {
	if method != http.MethodGet {
		return false
	}
	return path == "/api/v1/notifications/stream" || ticketAllowedMedicalFilePathRe.MatchString(path)
}

func isUserDisabled(userID uint64) bool {
	if userID == 0 {
		return false
	}
	status, ok := repository.GetUserStatus(userID)
	if !ok && repository.DB != nil {
		var u model.User
		if err := repository.DB.Select("id, status").First(&u, userID).Error; err == nil {
			status = u.Status
			repository.CacheUserStatus(userID, status, 60*time.Second)
		}
	}
	return status == "DISABLED"
}

type CustomClaims struct {
	UserID       uint64 `json:"user_id"`
	UserNo       string `json:"user_no"`
	Username     string `json:"username"`
	Role         string `json:"role"`
	HospitalID   uint64 `json:"hospital_id"`
	DepartmentID uint64 `json:"department_id"`
	jwt.RegisteredClaims
}

func GenerateToken(u *model.User) (string, error) {
	expire := time.Now().Add(time.Duration(config.AppConfig.Server.JWTExpireHours) * time.Hour)
	claims := CustomClaims{
		UserID:       u.ID,
		UserNo:       u.UserNo,
		Username:     u.Username,
		Role:         u.Role,
		HospitalID:   u.HospitalID,
		DepartmentID: u.DepartmentID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expire),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   u.Username,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.Server.JWTSecret))
}

func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 优先检查单次有效票据 Ticket (仅限 GET /notifications/stream 及 GET /medical-files/:id/(view|download))
		if ticket := c.Query("ticket"); ticket != "" {
			if !isTicketAllowedEndpoint(c.Request.Method, c.Request.URL.Path) {
				c.AbortWithStatusJSON(http.StatusForbidden, model.Response{
					Code:    403,
					Message: "安全拦截：一次性接入票据仅允许用于 GET 实时通知流与附件浏览/下载接口",
				})
				return
			}
			if data, ok := repository.ConsumeSSETicket(ticket); ok {
				var tp TicketPayload
				if err := json.Unmarshal([]byte(data), &tp); err == nil && tp.UserID > 0 && tp.Role != "" {
					// 检查用户封禁状态 (含缓存未命中时回源 MySQL 兜底)
					if isUserDisabled(tp.UserID) {
						c.AbortWithStatusJSON(http.StatusForbidden, model.Response{
							Code:    403,
							Message: "您的账号已被监管机构停用封禁，禁止继续执行系统操作",
						})
						return
					}

					c.Set("user_id", tp.UserID)
					c.Set("user_no", tp.UserNo)
					c.Set("username", tp.Username)
					c.Set("role", tp.Role)
					c.Set("hospital_id", tp.HospitalID)
					c.Set("department_id", tp.DepartmentID)
					c.Next()
					return
				}
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.Response{
				Code:    401,
				Message: "SSE 接入票据无效或已过期，请重新申请",
			})
			return
		}

		authHeader := c.GetHeader("Authorization")
		tokenStr := ""

		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenStr = parts[1]
			} else {
				c.AbortWithStatusJSON(http.StatusUnauthorized, model.Response{
					Code:    401,
					Message: "Token 格式错误 (Bearer {token})",
				})
				return
			}
		} else {
			// 支持 URL Query token (供 <img> / 浏览器附件直接下载安全鉴权)
			// 生产模式一律拒绝：URL 中的长效 JWT 会经访问日志、Referer、浏览器历史泄露，
			// 前端已改为先 POST /auth/ticket 取一次性票据，再以 ?ticket= 访问。
			tokenStr = c.Query("token")
			if tokenStr != "" {
				if config.AppConfig.IsProduction() {
					c.AbortWithStatusJSON(http.StatusForbidden, model.Response{
						Code:    403,
						Message: "生产环境禁止在 URL 中明文传递 Token，请先 POST /api/v1/auth/ticket 获取一次性 Ticket 后以 ?ticket= 访问",
					})
					return
				}
				log.Printf("[Security WARNING (Demo Mode)] 检测到通过 URL 携带 JWT Token 访问 %s，建议升级为一次性 Ticket 鉴权", c.Request.URL.Path)
			}
		}

		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.Response{
				Code:    401,
				Message: "未提供认证 Token，请先登录",
			})
			return
		}

		// 检查当前 Token 是否已被注销并加入黑名单 (Redis / MemoryCache 容灾黑名单)
		if repository.IsTokenBlacklisted(tokenStr) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.Response{
				Code:    401,
				Message: "身份令牌已登出失效，请重新登录",
			})
			return
		}

		claims := &CustomClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(config.AppConfig.Server.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.Response{
				Code:    401,
				Message: "无效或已过期的身份令牌",
			})
			return
		}

		// 实时核验账号状态 (通过 Redis / MemoryCache 高频削峰，仅在缓存未命中时回源 MySQL)
		if isUserDisabled(claims.UserID) {
			c.AbortWithStatusJSON(http.StatusForbidden, model.Response{
				Code:    403,
				Message: "您的账号已被监管机构停用封禁，禁止继续执行系统操作",
			})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_no", claims.UserNo)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("hospital_id", claims.HospitalID)
		c.Set("department_id", claims.DepartmentID)
		c.Next()
	}
}
