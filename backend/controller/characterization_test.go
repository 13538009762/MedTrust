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
	"medtrust-backend/middleware"
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
	"medtrust-backend/router"
	"medtrust-backend/service"
)

func initCharEnv(t *testing.T) (*gin.Engine, *model.User, *model.User, *model.User, *model.User) {
	gin.SetMode(gin.TestMode)
	cfg, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	db, err := repository.InitDB()
	if err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}
	repository.SeedInitialData(db)

	tempDir := t.TempDir()
	blockchain.InitBlockchainService(filepath.Join(tempDir, "ledger_data"), blockchain.FabricGatewayConfig{Mode: "mock"})
	service.InitMedicalService(cfg.IPFS.APIURL, filepath.Join(tempDir, "ipfs_storage"))
	service.InitAccessEngine(cfg.Risk.LowThreshold, cfg.Risk.HighThreshold)

	r := router.SetupRouter()

	admin, _ := repository.DefaultUserRepository.FindByUsername("admin")
	docLi, _ := repository.DefaultUserRepository.FindByUsername("doc_li")
	patZhang, _ := repository.DefaultUserRepository.FindByUsername("pat_zhang")
	supZhao, _ := repository.DefaultUserRepository.FindByUsername("sup_zhao")

	return r, admin, docLi, patZhang, supZhao
}

func authHeader(t *testing.T, u *model.User) string {
	token, err := middleware.GenerateToken(u)
	if err != nil {
		t.Fatalf("生成 token 失败: %v", err)
	}
	return "Bearer " + token
}

func TestCharacterization_Notifications(t *testing.T) {
	r, _, docLi, patZhang, _ := initCharEnv(t)
	patToken := authHeader(t, patZhang)

	// 在 DB 中插入一条待确认的加急访问申请
	reqNo := fmt.Sprintf("REQ_CHAR_%d", time.Now().UnixNano())
	testReq := model.AccessRequest{
		RequestNo: reqNo,
		DoctorID:  docLi.ID,
		PatientID: patZhang.ID,
		RecordID:  1,
		Purpose:   "特征测试加急申请",
		ScopeType: "ALL",
		Status:    "PENDING",
		ExpireAt:  time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	repository.DB.Create(&testReq)
	defer repository.DB.Delete(&testReq)

	// 1. GET /api/v1/notifications
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	req.Header.Set("Authorization", patToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /notifications 状态码期望 200，实际: %d, body: %s", w.Code, w.Body.String())
	}

	var resp model.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != 200 {
		t.Fatalf("业务码期望 200，实际: %d", resp.Code)
	}

	// 2. 患者确认同意加急申请 POST /api/v1/access-requests/:id/confirm
	reqConfirm := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/access-requests/%d/confirm", testReq.ID), nil)
	reqConfirm.Header.Set("Authorization", patToken)
	wConfirm := httptest.NewRecorder()
	r.ServeHTTP(wConfirm, reqConfirm)
	if wConfirm.Code != http.StatusOK {
		t.Fatalf("POST /access-requests/:req_no/confirm 状态码期望 200，实际: %d, body: %s", wConfirm.Code, wConfirm.Body.String())
	}

	// 校验 DB 行更新
	var checkReq model.AccessRequest
	repository.DB.First(&checkReq, testReq.ID)
	if checkReq.Status != "APPROVED" {
		t.Fatalf("申请状态期望 APPROVED，实际: %s", checkReq.Status)
	}
}

func TestCharacterization_SystemStatus(t *testing.T) {
	r, admin, _, _, _ := initCharEnv(t)

	// 1. 公开接口 GET /system/blockchain/status
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/v1/system/blockchain/status", nil)
	wStatus := httptest.NewRecorder()
	r.ServeHTTP(wStatus, reqStatus)
	if wStatus.Code != http.StatusOK {
		t.Fatalf("GET /system/blockchain/status 期望 200，实际: %d", wStatus.Code)
	}

	// 2. 管理员更新用户状态 PUT /system/users/:id/status
	token := authHeader(t, admin)
	testUser := model.User{
		UserNo:       "TEST_STATUS_USER_01",
		Username:     "test_status_user_01",
		RealName:     "状态测试用户",
		Role:         "doctor",
		HospitalID:   1,
		DepartmentID: 1,
		Status:       "NORMAL",
	}
	repository.DB.Where("user_no = ?", testUser.UserNo).Delete(&model.User{})
	repository.DB.Create(&testUser)
	defer repository.DB.Delete(&testUser)

	bodyBytes, _ := json.Marshal(map[string]string{"status": "DISABLED"})
	reqUpdate := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/system/users/%d/status", testUser.ID), bytes.NewReader(bodyBytes))
	reqUpdate.Header.Set("Authorization", token)
	reqUpdate.Header.Set("Content-Type", "application/json")
	wUpdate := httptest.NewRecorder()
	r.ServeHTTP(wUpdate, reqUpdate)
	if wUpdate.Code != http.StatusOK {
		t.Fatalf("PUT /system/users/:id/status 期望 200，实际: %d, body: %s", wUpdate.Code, wUpdate.Body.String())
	}

	// 校验 DB 状态已变为 DISABLED
	var updatedUser model.User
	repository.DB.First(&updatedUser, testUser.ID)
	if updatedUser.Status != "DISABLED" {
		t.Fatalf("用户状态更新失败，期望 DISABLED，实际: %s", updatedUser.Status)
	}
}

func TestCharacterization_SupervisorAudit(t *testing.T) {
	r, _, _, patZhang, supZhao := initCharEnv(t)
	token := authHeader(t, supZhao)

	// 1. GET /audit-logs
	reqLogs := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs", nil)
	reqLogs.Header.Set("Authorization", token)
	wLogs := httptest.NewRecorder()
	r.ServeHTTP(wLogs, reqLogs)
	if wLogs.Code != http.StatusOK {
		t.Fatalf("GET /audit-logs 期望 200，实际: %d", wLogs.Code)
	}

	// 2. 插入一条待审核紧急事件
	eventNo := fmt.Sprintf("EVT_CHAR_%d", time.Now().UnixNano())
	testEvt := model.EmergencyAccessEvent{
		EventNo:          eventNo,
		DoctorID:         2001,
		PatientID:        patZhang.ID,
		RecordID:         1,
		SourceHospitalID: 1,
		TargetHospitalID: 1,
		EmergencyReason:  "RESCUE",
		Description:      "特征测试紧急破窗",
		AuditStatus:      "PENDING_AUDIT",
		CreatedAt:        time.Now(),
	}
	repository.DB.Create(&testEvt)
	defer repository.DB.Delete(&testEvt)

	// 3. GET /supervisor/emergency-events
	reqEvts := httptest.NewRequest(http.MethodGet, "/api/v1/supervisor/emergency-events", nil)
	reqEvts.Header.Set("Authorization", token)
	wEvts := httptest.NewRecorder()
	r.ServeHTTP(wEvts, reqEvts)
	if wEvts.Code != http.StatusOK {
		t.Fatalf("GET /supervisor/emergency-events 期望 200，实际: %d", wEvts.Code)
	}

	// 4. POST /supervisor/emergency-events/:event_no/audit
	auditPayload, _ := json.Marshal(map[string]interface{}{
		"audit_status":  "CLOSED_APPROVED",
		"audit_comment": "审核通过，符合急救规范",
	})
	reqAudit := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/supervisor/emergency-events/%s/audit", eventNo), bytes.NewReader(auditPayload))
	reqAudit.Header.Set("Authorization", token)
	reqAudit.Header.Set("Content-Type", "application/json")
	wAudit := httptest.NewRecorder()
	r.ServeHTTP(wAudit, reqAudit)
	if wAudit.Code != http.StatusOK {
		t.Fatalf("POST /supervisor/emergency-events/:event_no/audit 期望 200，实际: %d, body: %s", wAudit.Code, wAudit.Body.String())
	}

	// 校验 DB 行更新
	var checkEvt model.EmergencyAccessEvent
	repository.DB.Where("event_no = ?", eventNo).First(&checkEvt)
	if checkEvt.AuditStatus != "CLOSED_APPROVED" {
		t.Fatalf("审核状态未更新: status=%s", checkEvt.AuditStatus)
	}
}

func TestCharacterization_AccessRequestLifecycle(t *testing.T) {
	r, _, docLi, patZhang, _ := initCharEnv(t)
	docToken := authHeader(t, docLi)
	patToken := authHeader(t, patZhang)

	// 1. 医生发起常规访问申请 POST /access/requests
	reqBody, _ := json.Marshal(map[string]interface{}{
		"patient_id": patZhang.ID,
		"record_id":  1,
		"purpose":    "特征测试门诊调阅申请",
	})
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/access/requests", bytes.NewReader(reqBody))
	reqCreate.Header.Set("Authorization", docToken)
	reqCreate.Header.Set("Content-Type", "application/json")
	wCreate := httptest.NewRecorder()
	r.ServeHTTP(wCreate, reqCreate)
	if wCreate.Code != http.StatusOK {
		t.Fatalf("POST /access/requests 期望 200，实际: %d, body: %s", wCreate.Code, wCreate.Body.String())
	}

	var latestReq model.AccessRequest
	repository.DB.Where("doctor_id = ? AND patient_id = ?", docLi.ID, patZhang.ID).Order("id DESC").First(&latestReq)
	reqID := latestReq.ID
	if reqID == 0 {
		t.Fatalf("未在数据库查到刚刚创建的访问申请")
	}
	defer repository.DB.Where("id = ?", reqID).Delete(&model.AccessRequest{})

	// 2. 患者查询待审批申请 GET /access/requests/pending
	reqPending := httptest.NewRequest(http.MethodGet, "/api/v1/access/requests/pending", nil)
	reqPending.Header.Set("Authorization", patToken)
	wPending := httptest.NewRecorder()
	r.ServeHTTP(wPending, reqPending)
	if wPending.Code != http.StatusOK {
		t.Fatalf("GET /access/requests/pending 期望 200，实际: %d", wPending.Code)
	}

	// 3. 患者拒绝或审批申请 POST /access/requests/:id/reject
	reqReject := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/access/requests/%d/reject", reqID), bytes.NewReader([]byte(`{"reject_reason":"测试拒绝"}`)))
	reqReject.Header.Set("Authorization", patToken)
	reqReject.Header.Set("Content-Type", "application/json")
	wReject := httptest.NewRecorder()
	r.ServeHTTP(wReject, reqReject)
	if wReject.Code != http.StatusOK {
		t.Fatalf("POST /access/requests/:id/reject 期望 200，实际: %d", wReject.Code)
	}

	var checkReq model.AccessRequest
	repository.DB.First(&checkReq, reqID)
	if checkReq.Status != "REJECTED" {
		t.Fatalf("申请状态未更新为 REJECTED，实际: %s", checkReq.Status)
	}
}
