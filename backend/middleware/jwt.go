package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"medtrust-backend/config"
	"medtrust-backend/model"
	"medtrust-backend/repository"
)

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
			tokenStr = c.Query("token")
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
		if claims.UserID > 0 {
			status, ok := repository.GetUserStatus(claims.UserID)
			if !ok {
				if repository.DB != nil {
					var u model.User
					if err := repository.DB.Select("id, status").First(&u, claims.UserID).Error; err == nil {
						status = u.Status
						repository.CacheUserStatus(claims.UserID, status, 60*time.Second)
					}
				}
			}
			if status == "DISABLED" {
				c.AbortWithStatusJSON(http.StatusForbidden, model.Response{
					Code:    403,
					Message: "您的账号已被监管机构停用封禁，禁止继续执行系统操作",
				})
				return
			}
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
