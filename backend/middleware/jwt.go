package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"medtrust-backend/config"
	"medtrust-backend/model"
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
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.Response{
				Code:    401,
				Message: "未提供认证 Token，请先登录",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.Response{
				Code:    401,
				Message: "Token 格式错误 (Bearer {token})",
			})
			return
		}

		tokenStr := parts[1]
		claims := &CustomClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(config.AppConfig.Server.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.Response{
				Code:    401,
				Message: "无效或已过期的身份令牌",
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
