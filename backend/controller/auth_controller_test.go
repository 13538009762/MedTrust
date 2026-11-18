package controller_test

import (
	"bytes"
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
	"medtrust-backend/service"
)

// buildAuthTestEngine 构建完整带有 JWTAuthMiddleware 和 AuthController 的测试路由引擎
func buildAuthTestEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/login", controller.DefaultAuthController.Login)
		v1.POST("/auth/register", controller.DefaultAuthController.Register)

		authGroup := v1.Group("")
		authGroup.Use(middleware.JWTAuthMiddleware())
		{
			authGroup.POST("/auth/logout", controller.DefaultAuthController.Logout)
			authGroup.GET("/auth/profile", controller.DefaultAuthController.Profile)
		}
	}
	return r
}

func TestAuthLoginLogoutAndBlacklist(t *testing.T) {
	_, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	db, err := repository.InitDB()
	if err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}
	repository.SeedInitialData(db)
	repository.InitRedis()

	engine := buildAuthTestEngine()

	// 1. 测试登录 doc_li / 123456
	loginPayload, _ := json.Marshal(map[string]string{
		"username": "doc_li",
		"password": "123456",
	})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginPayload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("登录请求失败, 期望 200, 实际 %d: %s", w.Code, w.Body.String())
	}

	var loginResp struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &loginResp); err != nil || loginResp.Data.Token == "" {
		t.Fatalf("解析登录响应 Token 失败: %v", err)
	}
	token := loginResp.Data.Token

	// 2. 携带有效 Token 访问 /auth/profile
	reqProfile := httptest.NewRequest("GET", "/api/v1/auth/profile", nil)
	reqProfile.Header.Set("Authorization", "Bearer "+token)
	wProfile := httptest.NewRecorder()
	engine.ServeHTTP(wProfile, reqProfile)
	if wProfile.Code != http.StatusOK {
		t.Fatalf("携带有效 Token 获取个人信息失败, 期望 200, 实际 %d: %s", wProfile.Code, wProfile.Body.String())
	}

	// 3. 执行登出 POST /auth/logout
	reqLogout := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	reqLogout.Header.Set("Authorization", "Bearer "+token)
	wLogout := httptest.NewRecorder()
	engine.ServeHTTP(wLogout, reqLogout)
	if wLogout.Code != http.StatusOK {
		t.Fatalf("执行登出失败, 期望 200, 实际 %d: %s", wLogout.Code, wLogout.Body.String())
	}

	// 4. 验证 Token 是否已被加入黑名单
	if !repository.IsTokenBlacklisted(token) {
		t.Fatalf("登出后 Token 未在黑名单中命中!")
	}

	// 5. 再次携带已注销的 Token 访问 /auth/profile，必须被拦截并返回 401
	reqProfileAfterLogout := httptest.NewRequest("GET", "/api/v1/auth/profile", nil)
	reqProfileAfterLogout.Header.Set("Authorization", "Bearer "+token)
	wProfileAfter := httptest.NewRecorder()
	engine.ServeHTTP(wProfileAfter, reqProfileAfterLogout)

	if wProfileAfter.Code != http.StatusUnauthorized {
		t.Fatalf("已注销的 Token 未被中间件拦截, 期望 401 Unauthorized, 实际 %d: %s", wProfileAfter.Code, wProfileAfter.Body.String())
	}

	var errResp model.Response
	_ = json.Unmarshal(wProfileAfter.Body.Bytes(), &errResp)
	if errResp.Code != 401 {
		t.Fatalf("响应结构体 Code 期望 401, 实际 %d", errResp.Code)
	}

	t.Logf("✅ 登出与 Token 黑名单拦截闭环验证通过: 登录 -> 鉴权通过 -> 登出加黑名单 -> 再次访问被拒(401)")
}

func TestAuthPasswordPolicyAndAdminSelfDisableGuard(t *testing.T) {
	if _, err := config.LoadConfig("../config/config.yaml"); err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if _, err := repository.InitDB(); err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}

	// 1. AuthService 注册与改密必须执行强密码校验（防止 6 位弱口令绕过）
	if _, err := service.DefaultAuthService.RegisterPatient("weak_pwd_user", "123456", "弱口令", "", ""); err == nil {
		t.Fatalf("RegisterPatient 应拒绝 6 位弱口令 '123456'")
	}
	if err := service.DefaultAuthService.ChangePassword(1, "old", "123456"); err == nil {
		t.Fatalf("ChangePassword 应拒绝 6 位弱口令 '123456'")
	}

	// 2. 管理员不得停用正在使用的自身账号
	ts := time.Now().UnixNano()
	adminUser := model.User{
		UserNo:   fmt.Sprintf("ADM_SELF_%d", ts),
		Username: fmt.Sprintf("adm_self_%d", ts),
		RealName: "自保管理员",
		Role:     "admin",
		Status:   "NORMAL",
	}
	repository.DB.Create(&adminUser)
	defer repository.DB.Unscoped().Delete(&adminUser)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PUT("/api/v1/system/users/:id/status", func(c *gin.Context) {
		c.Set("user_id", adminUser.ID)
		c.Next()
	}, controller.DefaultSystemController.UpdateUserStatus)

	body, _ := json.Marshal(map[string]string{"status": "DISABLED"})
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/system/users/%d/status", adminUser.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("管理员自停用应返回 400，实际: %d (%s)", w.Code, w.Body.String())
	}
}
