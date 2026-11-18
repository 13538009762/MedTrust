package controller_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
			authGroup.POST("/auth/logout", controller.DefaultAuthController.Logout)
			authGroup.GET("/auth/profile", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"user_id": c.GetUint64("user_id")})
			})
			authGroup.GET("/notifications/stream", func(c *gin.Context) {
				userID := c.GetUint64("user_id")
				c.JSON(http.StatusOK, gin.H{"user_id": userID, "online": true})
			})
			authGroup.GET("/medical-files/:id/view", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"user_id": c.GetUint64("user_id"), "username": c.GetString("username")})
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

	// 6. 作用域严格隔离：Ticket 不得用于非白名单业务端点 (应 403 且不消耗或越权通过)
	reqTkt := httptest.NewRequest("POST", "/api/v1/auth/ticket", nil)
	reqTkt.Header.Set("Authorization", "Bearer "+token)
	wTkt := httptest.NewRecorder()
	router.ServeHTTP(wTkt, reqTkt)
	var tktResp struct {
		Data struct {
			Ticket string `json:"ticket"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wTkt.Body.Bytes(), &tktResp)

	disallowedReq := httptest.NewRequest("GET", "/api/v1/auth/profile?ticket="+tktResp.Data.Ticket, nil)
	disallowedW := httptest.NewRecorder()
	router.ServeHTTP(disallowedW, disallowedReq)
	if disallowedW.Code != http.StatusForbidden {
		t.Fatalf("非白名单路径携带 ?ticket= 预期返回 403 Forbidden，实际: %d (%s)", disallowedW.Code, disallowedW.Body.String())
	}

	// 同一个未被消耗的合法 Ticket 用于 GET /medical-files/1/view 必须成功，且即使 Username 含 '|' 也不破坏 JSON 解析
	pipeTicketUser := &model.User{
		ID:       10087,
		UserNo:   "NO|PIPE|01",
		Username: "name|with|pipes",
		Role:     "patient",
	}
	pipeToken, _ := middleware.GenerateToken(pipeTicketUser)
	reqPipeTkt := httptest.NewRequest("POST", "/api/v1/auth/ticket", nil)
	reqPipeTkt.Header.Set("Authorization", "Bearer "+pipeToken)
	wPipeTkt := httptest.NewRecorder()
	router.ServeHTTP(wPipeTkt, reqPipeTkt)
	var pipeResp struct {
		Data struct {
			Ticket string `json:"ticket"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wPipeTkt.Body.Bytes(), &pipeResp)

	viewReq := httptest.NewRequest("GET", "/api/v1/medical-files/12/view?ticket="+pipeResp.Data.Ticket, nil)
	viewW := httptest.NewRecorder()
	router.ServeHTTP(viewW, viewReq)
	if viewW.Code != http.StatusOK {
		t.Fatalf("GET /medical-files/:id/view 携带合法 Ticket 预期 200，实际: %d (%s)", viewW.Code, viewW.Body.String())
	}

	// 7. DISABLED 用户在缓存失效时使用 Ticket 也必须经由 MySQL 兜底被 403 拦截
	if _, dbErr := repository.InitDB(); dbErr == nil && repository.DB != nil {
		ts := time.Now().UnixNano()
		disabledPat := model.User{
			UserNo:   fmt.Sprintf("DIS_TKT_%d", ts),
			Username: fmt.Sprintf("dis_tkt_%d", ts),
			RealName: "已封禁Ticket用户",
			Role:     "patient",
			Status:   "DISABLED",
		}
		repository.DB.Create(&disabledPat)
		defer repository.DB.Unscoped().Delete(&disabledPat)

		_ = repository.StoreSSETicket("dis_ticket_"+fmt.Sprintf("%d", ts), fmt.Sprintf(`{"user_id":%d,"user_no":"%s","role":"patient","username":"%s","hospital_id":0,"department_id":0}`, disabledPat.ID, disabledPat.UserNo, disabledPat.Username), time.Minute)
		repository.InvalidateUserStatus(disabledPat.ID)

		disReq := httptest.NewRequest("GET", "/api/v1/notifications/stream?ticket=dis_ticket_"+fmt.Sprintf("%d", ts), nil)
		disW := httptest.NewRecorder()
		router.ServeHTTP(disW, disReq)
		if disW.Code != http.StatusForbidden {
			t.Fatalf("缓存失效时 DISABLED 用户 Ticket 必须被 DB 回源拦截返回 403，实际: %d (%s)", disW.Code, disW.Body.String())
		}
	}

	// 8. Logout 接口拒绝伪造 JWT 并防止黑名单污染
	reqBadLogout := httptest.NewRequest("POST", "/api/v1/auth/logout?token=not.a.valid.jwt", nil)
	wBadLogout := httptest.NewRecorder()
	router.ServeHTTP(wBadLogout, reqBadLogout)
	if wBadLogout.Code != http.StatusUnauthorized {
		t.Fatalf("Logout 伪造 token 预期 401，实际: %d", wBadLogout.Code)
	}
}
