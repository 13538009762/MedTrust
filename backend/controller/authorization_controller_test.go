package controller_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"medtrust-backend/config"
	"medtrust-backend/controller"
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
	"medtrust-backend/service"
)

func initControllerAuthTestDB(t *testing.T) {
	t.Helper()
	if _, err := config.LoadConfig("../config/config.yaml"); err != nil {
		t.Fatalf("load config failed: %v", err)
	}
	if _, err := repository.InitDB(); err != nil {
		t.Fatalf("init db failed: %v", err)
	}
	blockchain.InitBlockchainService(filepath.Join(t.TempDir(), "fabric_ledger"), blockchain.FabricGatewayConfig{
		Mode:    "mock",
		Enabled: false,
	})
}

func TestCreateAuthorization_OwnershipAndValidation(t *testing.T) {
	initControllerAuthTestDB(t)

	nowSuffix := time.Now().UnixNano()
	patA := model.User{
		UserNo:   fmt.Sprintf("PAT_OWN_A_%d", nowSuffix),
		Username: fmt.Sprintf("pat_own_a_%d", nowSuffix),
		RealName: "归属患者A",
		Role:     "patient",
		Status:   "NORMAL",
	}
	patB := model.User{
		UserNo:   fmt.Sprintf("PAT_OWN_B_%d", nowSuffix),
		Username: fmt.Sprintf("pat_own_b_%d", nowSuffix),
		RealName: "归属患者B",
		Role:     "patient",
		Status:   "NORMAL",
	}
	doc := model.User{
		UserNo:     fmt.Sprintf("DOC_OWN_%d", nowSuffix),
		Username:   fmt.Sprintf("doc_own_%d", nowSuffix),
		RealName:   "测试医生",
		Role:       "doctor",
		HospitalID: 1,
		Status:     "NORMAL",
	}
	if err := repository.DB.Create(&patA).Error; err != nil {
		t.Fatalf("create patA failed: %v", err)
	}
	defer repository.DB.Delete(&patA)
	if err := repository.DB.Create(&patB).Error; err != nil {
		t.Fatalf("create patB failed: %v", err)
	}
	defer repository.DB.Delete(&patB)
	if err := repository.DB.Create(&doc).Error; err != nil {
		t.Fatalf("create doc failed: %v", err)
	}
	defer repository.DB.Delete(&doc)

	recB := model.MedicalRecord{
		RecordNo:        fmt.Sprintf("REC-OWN-%d", nowSuffix),
		PatientID:       patB.ID,
		DoctorID:        doc.ID,
		HospitalID:      1,
		DataType:        "EMR",
		Diagnosis:       "测试病历",
		IntegrityStatus: "VERIFIED",
		SyncStatus:      service.SyncStatusCompleted,
		CreatedAt:       time.Now(),
	}
	if err := repository.DB.Create(&recB).Error; err != nil {
		t.Fatalf("create recB failed: %v", err)
	}
	defer repository.DB.Delete(&recB)
	defer repository.DB.Where("record_id = ?", recB.ID).Delete(&model.Authorization{})
	defer repository.DB.Where("user_id IN ?", []uint64{patA.ID, patB.ID}).Delete(&model.AuditLog{})

	newRouter := func(userID uint64, role string) *gin.Engine {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Set("role", role)
			c.Set("hospital_id", uint64(1))
			c.Next()
		})
		r.POST("/api/v1/authorizations", controller.DefaultAccessController.CreateAuthorization)
		return r
	}

	rPatA := newRouter(patA.ID, "patient")
	rPatB := newRouter(patB.ID, "patient")

	// 1. Cross-patient authorization must return 403 Forbidden
	crossPayload := fmt.Sprintf(`{"auth_target_type":"DOCTOR","auth_target_id":%d,"scope_type":"SINGLE","record_id":%d,"days":7}`, doc.ID, recB.ID)
	reqCross := httptest.NewRequest(http.MethodPost, "/api/v1/authorizations", bytes.NewBufferString(crossPayload))
	reqCross.Header.Set("Content-Type", "application/json")
	wCross := httptest.NewRecorder()
	rPatA.ServeHTTP(wCross, reqCross)
	if wCross.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cross-patient CreateAuthorization, got %d body=%s", wCross.Code, wCross.Body.String())
	}

	// 2. Non-existent record must return 404
	missingPayload := fmt.Sprintf(`{"auth_target_type":"DOCTOR","auth_target_id":%d,"scope_type":"SINGLE","record_id":999999999,"days":7}`, doc.ID)
	reqMissing := httptest.NewRequest(http.MethodPost, "/api/v1/authorizations", bytes.NewBufferString(missingPayload))
	reqMissing.Header.Set("Content-Type", "application/json")
	wMissing := httptest.NewRecorder()
	rPatA.ServeHTTP(wMissing, reqMissing)
	if wMissing.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for nonexistent record, got %d body=%s", wMissing.Code, wMissing.Body.String())
	}

	// 3. Invalid AuthTargetType must return 400
	badEnumPayload := `{"auth_target_type":"HACKER","auth_target_id":1,"scope_type":"ALL","days":7}`
	reqBad := httptest.NewRequest(http.MethodPost, "/api/v1/authorizations", bytes.NewBufferString(badEnumPayload))
	reqBad.Header.Set("Content-Type", "application/json")
	wBad := httptest.NewRecorder()
	rPatA.ServeHTTP(wBad, reqBad)
	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid auth_target_type, got %d body=%s", wBad.Code, wBad.Body.String())
	}

	// 4. Owner patient B succeeds with 200
	ownPayload := fmt.Sprintf(`{"auth_target_type":"DOCTOR","auth_target_id":%d,"scope_type":"SINGLE","record_id":%d,"days":7}`, doc.ID, recB.ID)
	reqOwn := httptest.NewRequest(http.MethodPost, "/api/v1/authorizations", bytes.NewBufferString(ownPayload))
	reqOwn.Header.Set("Content-Type", "application/json")
	wOwn := httptest.NewRecorder()
	rPatB.ServeHTTP(wOwn, reqOwn)
	if wOwn.Code != http.StatusOK {
		t.Fatalf("expected 200 for owner CreateAuthorization, got %d body=%s", wOwn.Code, wOwn.Body.String())
	}
}

func TestSystemListUsers_OmitsAvatarAndSupportsPagination(t *testing.T) {
	initControllerAuthTestDB(t)

	nowSuffix := time.Now().UnixNano()
	u := model.User{
		UserNo:   fmt.Sprintf("USR_PAGE_%d", nowSuffix),
		Username: fmt.Sprintf("usr_page_%d", nowSuffix),
		RealName: "分页头像测试",
		Role:     "patient",
		Status:   "NORMAL",
		Avatar:   "data:image/png;base64,FAKE_LARGE_AVATAR_PAYLOAD",
	}
	if err := repository.DB.Create(&u).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	defer repository.DB.Delete(&u)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		c.Set("role", "admin")
		c.Next()
	})
	r.GET("/api/v1/system/users", controller.DefaultSystemController.ListUsers)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/system/users?page=1&page_size=5&keyword=%s", u.Username), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from ListUsers, got %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Code int `json:"code"`
		Data struct {
			List     []model.User `json:"list"`
			Total    int64        `json:"total"`
			Page     int          `json:"page"`
			PageSize int          `json:"page_size"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if len(resp.Data.List) != 1 || resp.Data.Total != 1 || resp.Data.Page != 1 || resp.Data.PageSize != 5 {
		t.Fatalf("unexpected paginated ListUsers data: %+v", resp.Data)
	}
	if resp.Data.List[0].Avatar != "" {
		t.Fatalf("expected avatar to be omitted in ListUsers, got len=%d", len(resp.Data.List[0].Avatar))
	}
}
