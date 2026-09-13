package middleware

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"medtrust-backend/model"
)

func RequireRoles(roles ...string) gin.HandlerFunc {
	roleMap := make(map[string]struct{})
	for _, r := range roles {
		roleMap[r] = struct{}{}
	}

	return func(c *gin.Context) {
		currentRole, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, model.Response{
				Code:    403,
				Message: "无权访问：未识别的用户角色",
			})
			return
		}
		if _, ok := roleMap[currentRole.(string)]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, model.Response{
				Code:    403,
				Message: "RBAC 权限拦截：您当前角色无法执行此操作",
			})
			return
		}
		c.Next()
	}
}
