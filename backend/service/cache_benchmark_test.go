package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"medtrust-backend/config"
	"medtrust-backend/middleware"
	"medtrust-backend/model"
	"medtrust-backend/repository"
)

// TestUserStatusCache_ImmediateForbiddenOnDisable 验证核心安全性：
// 当管理员或监管员封禁用户时，缓存即时失效，下一次请求必须立即返回 403 Forbidden，杜绝因缓存滞后造成的越权访问窗口
func TestUserStatusCache_ImmediateForbiddenOnDisable(t *testing.T) {
	_, _ = config.LoadConfig("../config/config.yaml")
	_, _ = repository.InitDB()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.JWTAuthMiddleware())
	r.GET("/test/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "access granted"})
	})

	nanos := time.Now().UnixNano()
	user := model.User{
		UserNo:    fmt.Sprintf("USER_CACHE_%d", nanos),
		Username:  fmt.Sprintf("cache_user_%d", nanos),
		Role:      "doctor",
		Status:    "NORMAL",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repository.DB.Create(&user)
	defer repository.DB.Delete(&user)

	// 生成该用户的有效 JWT Token
	token, err := middleware.GenerateToken(&user)
	if err != nil {
		t.Fatalf("Generate token failed: %v", err)
	}

	// 1. 第一次请求：回源查库并回填状态缓存 -> 预期 200 OK
	req1 := httptest.NewRequest("GET", "/test/protected", nil)
	req1.Header.Set("Authorization", "Bearer "+token)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("初始请求预期 200, 实际: %d, body: %s", w1.Code, w1.Body.String())
	}

	// 2. 验证缓存已命中 (此时缓存状态为 NORMAL)
	cachedStatus, hit := repository.GetUserStatus(user.ID)
	if !hit || cachedStatus != "NORMAL" {
		t.Fatalf("状态未能正确缓存: hit=%v, status=%s", hit, cachedStatus)
	}

	// 3. 第二次请求：走状态缓存命中路径 -> 预期依然 200 OK
	req2 := httptest.NewRequest("GET", "/test/protected", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("缓存命中请求预期 200, 实际: %d", w2.Code)
	}

	// 4. 管理员在后台将该用户状态更新为 DISABLED 并主动失效缓存
	repository.DB.Model(&model.User{}).Where("id = ?", user.ID).Update("status", "DISABLED")
	repository.InvalidateUserStatus(user.ID)

	// 5. 核心断言：下一次请求必须立即拦截并返回 403 Forbidden，绝无缓存越权滞后！
	req3 := httptest.NewRequest("GET", "/test/protected", nil)
	req3.Header.Set("Authorization", "Bearer "+token)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)

	if w3.Code != http.StatusForbidden {
		t.Fatalf("禁用后请求预期立即返回 403 Forbidden, 实际返回: %d, body: %s", w3.Code, w3.Body.String())
	}

	t.Logf("✅ 核心安全用例通过：用户状态缓存写后即时失效，封禁拦截 100%% 秒级生效 (HTTP %d)", w3.Code)
}

// TestUserStatusCache_DirectLifecycle 验证缓存读写、命中与清理的底层行为
func TestUserStatusCache_DirectLifecycle(t *testing.T) {
	_, _ = config.LoadConfig("../config/config.yaml")

	userID := uint64(999998)
	repository.InvalidateUserStatus(userID)

	// 1. 未写入前，未命中
	_, ok := repository.GetUserStatus(userID)
	if ok {
		t.Fatalf("未写入前不应命中缓存")
	}

	// 2. 写入缓存并读取
	repository.CacheUserStatus(userID, "RESTRICTED", 10*time.Second)
	status, ok := repository.GetUserStatus(userID)
	if !ok || status != "RESTRICTED" {
		t.Fatalf("写入后获取缓存失败: ok=%v, status=%s", ok, status)
	}

	// 3. 主动失效
	repository.InvalidateUserStatus(userID)
	_, ok = repository.GetUserStatus(userID)
	if ok {
		t.Fatalf("失效后依然命中缓存")
	}
}

// BenchmarkDirectDBQuery 模拟直连数据库查询用户状态的性能开销
func BenchmarkDirectDBQuery(b *testing.B) {
	_, _ = config.LoadConfig("../config/config.yaml")
	_, _ = repository.InitDB()

	var u model.User
	if err := repository.DB.First(&u, 1).Error; err != nil {
		u = model.User{
			UserNo:    "BENCH_USER_1",
			Username:  "bench_user_1",
			Role:      "doctor",
			Status:    "NORMAL",
			CreatedAt: time.Now(),
		}
		repository.DB.Create(&u)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var user model.User
		_ = repository.DB.Select("id, status").First(&user, u.ID).Error
	}
}

// BenchmarkCachedQuery 模拟启用状态缓存后的性能吞吐表现
func BenchmarkCachedQuery(b *testing.B) {
	_, _ = config.LoadConfig("../config/config.yaml")
	userID := uint64(1)
	repository.CacheUserStatus(userID, "NORMAL", 10*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = repository.GetUserStatus(userID)
	}
}
