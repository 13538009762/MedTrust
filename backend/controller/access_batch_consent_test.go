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

func setupBatchConsentTestEnv(t *testing.T) {
	_, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	db, err := repository.InitDB()
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	repository.SeedInitialData(db)
	blockchain.InitBlockchainService(filepath.Join(t.TempDir(), "ledger_data"), blockchain.FabricGatewayConfig{Mode: "mock"})
}

func setupBatchConsentRouter(patientID uint64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", patientID)
		c.Set("role", "patient")
		c.Next()
	})
	v1 := r.Group("/api/v1")
	{
		v1.POST("/access/requests/batch-approve", controller.DefaultAccessController.BatchApproveRequests)
		v1.POST("/access/requests/batch-reject", controller.DefaultAccessController.BatchRejectRequests)
		v1.GET("/access/requests/pending", controller.DefaultAccessController.ListPendingRequests)
	}
	return r
}

func TestBatchConsent_ApproveSelected(t *testing.T) {
	setupBatchConsentTestEnv(t)
	patientID := uint64(4)
	docID := uint64(1)

	// 创建 3 个处于 PENDING 的访问申请
	req1 := model.AccessRequest{
		RequestNo: fmt.Sprintf("REQ-TEST-SEL-1-%d", time.Now().UnixNano()),
		DoctorID:  docID,
		PatientID: patientID,
		RecordID:  40,
		ScopeType: "SINGLE",
		Status:    "PENDING",
		Days:      7,
		ExpireAt:  time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}
	req2 := model.AccessRequest{
		RequestNo: fmt.Sprintf("REQ-TEST-SEL-2-%d", time.Now().UnixNano()),
		DoctorID:  docID,
		PatientID: patientID,
		RecordID:  40,
		ScopeType: "SINGLE",
		Status:    "PENDING",
		Days:      14,
		ExpireAt:  time.Now().Add(14 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}
	req3 := model.AccessRequest{
		RequestNo: fmt.Sprintf("REQ-TEST-SEL-3-%d", time.Now().UnixNano()),
		DoctorID:  docID,
		PatientID: patientID,
		RecordID:  40,
		ScopeType: "SINGLE",
		Status:    "PENDING",
		Days:      30,
		ExpireAt:  time.Now().Add(30 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}
	_ = repository.DB.Create(&req1)
	_ = repository.DB.Create(&req2)
	_ = repository.DB.Create(&req3)

	r := setupBatchConsentRouter(patientID)

	// 1. 批量同意选中的 req1 和 req2
	bodyObj := map[string]interface{}{
		"request_ids": []uint64{req1.ID, req2.ID},
		"all":         false,
	}
	bodyBytes, _ := json.Marshal(bodyObj)
	req := httptest.NewRequest("POST", "/api/v1/access/requests/batch-approve", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("batch approve failed: %d, body=%s", w.Code, w.Body.String())
	}

	var res struct {
		Code int `json:"code"`
		Data struct {
			TotalCount   int      `json:"total_count"`
			SuccessCount int      `json:"success_count"`
			FailedCount  int      `json:"failed_count"`
			AuthNos      []string `json:"auth_nos"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res.Data.SuccessCount != 2 {
		t.Fatalf("expected 2 successes, got %d", res.Data.SuccessCount)
	}
	if len(res.Data.AuthNos) != 2 {
		t.Fatalf("expected 2 authNos, got %d", len(res.Data.AuthNos))
	}

	// 验证数据库中 req1, req2 为 APPROVED，req3 仍为 PENDING
	var check1, check2, check3 model.AccessRequest
	repository.DB.First(&check1, req1.ID)
	repository.DB.First(&check2, req2.ID)
	repository.DB.First(&check3, req3.ID)

	if check1.Status != "APPROVED" || check2.Status != "APPROVED" {
		t.Fatalf("req1 or req2 status not APPROVED: req1=%s, req2=%s", check1.Status, check2.Status)
	}
	if check3.Status != "PENDING" {
		t.Fatalf("req3 status should still be PENDING, got %s", check3.Status)
	}
}

func TestBatchConsent_ApproveAll(t *testing.T) {
	setupBatchConsentTestEnv(t)
	// 使用独立测试患者
	patientID := uint64(888)
	docID := uint64(1)

	// 创建 3 个处于 PENDING 的申请
	for i := 1; i <= 3; i++ {
		r := model.AccessRequest{
			RequestNo: fmt.Sprintf("REQ-TEST-ALL-%d-%d", i, time.Now().UnixNano()),
			DoctorID:  docID,
			PatientID: patientID,
			RecordID:  40,
			ScopeType: "SINGLE",
			Status:    "PENDING",
			Days:      7,
			ExpireAt:  time.Now().Add(7 * 24 * time.Hour),
			CreatedAt: time.Now(),
		}
		_ = repository.DB.Create(&r)
	}

	router := setupBatchConsentRouter(patientID)

	// 一键全部同意
	bodyObj := map[string]interface{}{
		"all": true,
	}
	bodyBytes, _ := json.Marshal(bodyObj)
	req := httptest.NewRequest("POST", "/api/v1/access/requests/batch-approve", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("one-click approve all failed: %d, body=%s", w.Code, w.Body.String())
	}

	var res struct {
		Code int `json:"code"`
		Data struct {
			SuccessCount int `json:"success_count"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res.Data.SuccessCount != 3 {
		t.Fatalf("expected 3 successes, got %d", res.Data.SuccessCount)
	}

	// 确认该患者名下无 PENDING 记录
	pending, _ := service.DefaultAccessRequestService.ListPendingRequests(nil, patientID)
	if len(pending) != 0 {
		t.Fatalf("expected 0 pending requests, got %d", len(pending))
	}
}

func TestBatchConsent_RejectSelectedAndAll(t *testing.T) {
	setupBatchConsentTestEnv(t)
	patientID := uint64(999)
	docID := uint64(1)

	r1 := model.AccessRequest{
		RequestNo: fmt.Sprintf("REQ-TEST-REJ-1-%d", time.Now().UnixNano()),
		DoctorID:  docID,
		PatientID: patientID,
		RecordID:  40,
		ScopeType: "SINGLE",
		Status:    "PENDING",
		Days:      7,
		ExpireAt:  time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}
	r2 := model.AccessRequest{
		RequestNo: fmt.Sprintf("REQ-TEST-REJ-2-%d", time.Now().UnixNano()),
		DoctorID:  docID,
		PatientID: patientID,
		RecordID:  40,
		ScopeType: "SINGLE",
		Status:    "PENDING",
		Days:      7,
		ExpireAt:  time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}
	r3 := model.AccessRequest{
		RequestNo: fmt.Sprintf("REQ-TEST-REJ-3-%d", time.Now().UnixNano()),
		DoctorID:  docID,
		PatientID: patientID,
		RecordID:  40,
		ScopeType: "SINGLE",
		Status:    "PENDING",
		Days:      7,
		ExpireAt:  time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}
	_ = repository.DB.Create(&r1)
	_ = repository.DB.Create(&r2)
	_ = repository.DB.Create(&r3)

	router := setupBatchConsentRouter(patientID)

	// 批量驳回 r1
	bodyObj1 := map[string]interface{}{
		"request_ids": []uint64{r1.ID},
		"all":         false,
	}
	b1, _ := json.Marshal(bodyObj1)
	req1 := httptest.NewRequest("POST", "/api/v1/access/requests/batch-reject", bytes.NewReader(b1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("batch reject r1 failed: %d", w1.Code)
	}

	var check1 model.AccessRequest
	repository.DB.First(&check1, r1.ID)
	if check1.Status != "REJECTED" {
		t.Fatalf("r1 should be REJECTED, got %s", check1.Status)
	}

	// 一键全部驳回剩余 (r2, r3)
	bodyObj2 := map[string]interface{}{
		"all": true,
	}
	b2, _ := json.Marshal(bodyObj2)
	req2 := httptest.NewRequest("POST", "/api/v1/access/requests/batch-reject", bytes.NewReader(b2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("one-click reject all failed: %d", w2.Code)
	}

	var check2, check3 model.AccessRequest
	repository.DB.First(&check2, r2.ID)
	repository.DB.First(&check3, r3.ID)
	if check2.Status != "REJECTED" || check3.Status != "REJECTED" {
		t.Fatalf("r2 or r3 not REJECTED: r2=%s, r3=%s", check2.Status, check3.Status)
	}
}

func TestBatchConsent_IDORProtection(t *testing.T) {
	setupBatchConsentTestEnv(t)
	patientA := uint64(1001)
	patientB := uint64(1002)
	docID := uint64(1)

	// patient B 的申请单
	reqB := model.AccessRequest{
		RequestNo: fmt.Sprintf("REQ-TEST-IDOR-%d", time.Now().UnixNano()),
		DoctorID:  docID,
		PatientID: patientB,
		RecordID:  40,
		ScopeType: "SINGLE",
		Status:    "PENDING",
		Days:      7,
		ExpireAt:  time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}
	_ = repository.DB.Create(&reqB)

	// patient A 尝试越权核准 patient B 的申请单
	routerA := setupBatchConsentRouter(patientA)
	bodyObj := map[string]interface{}{
		"request_ids": []uint64{reqB.ID},
		"all":         false,
	}
	b, _ := json.Marshal(bodyObj)
	req := httptest.NewRequest("POST", "/api/v1/access/requests/batch-approve", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	routerA.ServeHTTP(w, req)

	// 结果应为 0 成功（因为查不到属于 patient A 的该 ID）
	var res struct {
		Code int `json:"code"`
		Data struct {
			SuccessCount int `json:"success_count"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res.Data.SuccessCount != 0 {
		t.Fatalf("IDOR security breach: patient A should not be able to approve patient B's request!")
	}

	// 检查数据库中 reqB 仍为 PENDING
	var checkB model.AccessRequest
	repository.DB.First(&checkB, reqB.ID)
	if checkB.Status != "PENDING" {
		t.Fatalf("reqB should remain PENDING, got %s", checkB.Status)
	}
}
