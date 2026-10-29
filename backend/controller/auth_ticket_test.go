package controller_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"medtrust-backend/config"
	"medtrust-backend/controller"
	"medtrust-backend/middleware"
	"medtrust-backend/model"
	"medtrust-backend/repository"
)

func setupTicketTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.CORSMiddleware())

	v1 := r.Group("/api/v1")
	{
		authGroup := v1.Group("")
		authGroup.Use(middleware.JWTAuthMiddleware())
		{
			authGroup.POST("/auth/ticket", controller.DefaultAuthController.CreateSSETicket)
			authGroup.GET("/notifications/stream", func(c *gin.Context) {
				userID := c.GetUint64("user_id")
				c.JSON(http.StatusOK, gin.H{"user_id": userID, "online": true})
			})
		}
	}
	return r
}

func TestSSETicket_Lifecycle_CreateAndSingleUse(t *testing.T) {
	config.AppConfig.Server.JWTSecret = "medtrust_secure_jwt_secret_key_2026"
	config.AppConfig.Server.JWTExpireHours = 24
	repository.InitRedis()

	router := setupTicketTestRouter()

	// 1. 生成合法测试用户 JWT
	testUser := &model.User{
		ID:       10086,
		UserNo:   "USER_TICKET_TEST",
		Username: "ticket_user",
		Role:     "doctor",
	}
	token, err := middleware.GenerateToken(testUser)
	if err != nil {
		t.Fatalf("生成测试 Token 失败: %v", err)
	}

	// 2. 申请一次性 SSE Ticket: POST /api/v1/auth/ticket
	req := httptest.NewRequest("POST", "/api/v1/auth/ticket", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("预期 200，实际: %d, body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Ticket    string `json:"ticket"`
			ExpiresIn int    `json:"expires_in"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	ticket := resp.Data.Ticket
	if ticket == "" || resp.Data.ExpiresIn != 60 {
		t.Fatalf("Ticket 格式不正确: %+v", resp.Data)
	}

	// 3. 第一次使用 Ticket 访问 SSE: GET /api/v1/notifications/stream?ticket=<ticket>
	sseReq1 := httptest.NewRequest("GET", "/api/v1/notifications/stream?ticket="+ticket, nil)
	sseW1 := httptest.NewRecorder()
	router.ServeHTTP(sseW1, sseReq1)

	if sseW1.Code != http.StatusOK {
		t.Fatalf("首次使用 Ticket 应鉴权成功返回 200，实际: %d", sseW1.Code)
	}

	// 4. 重放攻击测试：第二次使用同一个 Ticket 应被拦截 (401 Unauthorized)
	sseReq2 := httptest.NewRequest("GET", "/api/v1/notifications/stream?ticket="+ticket, nil)
	sseW2 := httptest.NewRecorder()
	router.ServeHTTP(sseW2, sseReq2)

	if sseW2.Code != http.StatusUnauthorized {
		t.Fatalf("二次使用已失效的 Ticket 预期返回 401 Unauthorized，实际: %d", sseW2.Code)
	}

	// 5. 使用伪造 Ticket 访问应被拦截
	sseReqFake := httptest.NewRequest("GET", "/api/v1/notifications/stream?ticket=fake_invalid_ticket_xyz", nil)
	sseWFake := httptest.NewRecorder()
	router.ServeHTTP(sseWFake, sseReqFake)

	if sseWFake.Code != http.StatusUnauthorized {
		t.Fatalf("伪造 Ticket 预期返回 401 Unauthorized，实际: %d", sseWFake.Code)
	}
}
