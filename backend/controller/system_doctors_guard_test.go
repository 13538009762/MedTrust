package controller_test

// 次级修复回归：/system/doctors 医生目录的访问控制与 PII 脱敏
//   1. 患者（授权选医生场景）可访问，且 phone/id_card 必须脱敏、密钥类字段必须清空
//   2. 未声明角色 / 未知角色被 RequireRoles 拦截 403

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"medtrust-backend/controller"
	"medtrust-backend/middleware"
	"medtrust-backend/model"
)

// buildDoctorCatalogEngine 构建仅挂载 /system/doctors 的最小引擎（带身份上下文）
func buildDoctorCatalogEngine(userID uint64, role string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("role", role)
		c.Next()
	})
	v1 := r.Group("/api/v1")
	v1.GET("/system/doctors", middleware.RequireRoles("admin", "supervisor", "doctor", "patient"), controller.DefaultSystemController.ListDoctors)
	return r
}

func TestSecurity_DoctorCatalogRoleGuardAndMasking(t *testing.T) {
	initSecurityTestEnv(t)
	_, _, _, patZhang, _, _ := loadSecuritySeedUsers(t)

	// 1. 患者访问：应放行
	r := buildDoctorCatalogEngine(patZhang.ID, "patient")
	req := httptest.NewRequest("GET", "/api/v1/system/doctors", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("患者应可访问医生目录（授权选医生依赖此接口），实际 %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code int           `json:"code"`
		Data []model.User `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != 200 || len(resp.Data) == 0 {
		t.Fatalf("医生目录应返回非空列表，实际 code=%d len=%d", resp.Code, len(resp.Data))
	}

	for _, d := range resp.Data {
		if d.PasswordHash != "" || d.MedicalKey != "" || d.MedicalKeyHash != "" {
			t.Fatalf("【安全缺口】医生目录泄露密钥类字段: id=%d", d.ID)
		}
		// 脱敏后必含掩码字符；原始 PII 若为长串则不得原样出现
		if d.Phone != "" && len(d.Phone) > 4 && !containsMask(d.Phone) {
			t.Fatalf("【安全缺口】医生 ID=%d 手机号未脱敏: %s", d.ID, d.Phone)
		}
		if d.IDCard != "" && len(d.IDCard) > 6 && !containsMask(d.IDCard) {
			t.Fatalf("【安全缺口】医生 ID=%d 身份证未脱敏: %s", d.ID, d.IDCard)
		}
	}

	// 2. 未携带角色：应被 403 拦截
	rNoRole := buildDoctorCatalogEngine(patZhang.ID, "")
	wNoRole := httptest.NewRecorder()
	rNoRole.ServeHTTP(wNoRole, httptest.NewRequest("GET", "/api/v1/system/doctors", nil))
	if wNoRole.Code != http.StatusForbidden {
		t.Fatalf("无角色访问医生目录应返回 403，实际 %d", wNoRole.Code)
	}

	// 3. 未知角色：应被 403 拦截
	rUnknown := buildDoctorCatalogEngine(patZhang.ID, "anonymous")
	wUnknown := httptest.NewRecorder()
	rUnknown.ServeHTTP(wUnknown, httptest.NewRequest("GET", "/api/v1/system/doctors", nil))
	if wUnknown.Code != http.StatusForbidden {
		t.Fatalf("未知角色访问医生目录应返回 403，实际 %d", wUnknown.Code)
	}
}

func containsMask(s string) bool {
	for _, r := range s {
		if r == '*' || r == '＊' {
			return true
		}
	}
	return false
}