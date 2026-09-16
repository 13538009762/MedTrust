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
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
)

func setupTestRouter(userID uint64, role string) (*gin.Engine, *controller.AccessController) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	accessCtrl := &controller.AccessController{}

	// 模拟 JWT 鉴权中间件写入的 context 变量
	r.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("role", role)
		c.Next()
	})

	r.DELETE("/authorizations/:id", accessCtrl.RevokeAuthorization)
	r.DELETE("/authorizations/all", accessCtrl.RevokeAllAuthorizations)
	r.POST("/authorizations/:id/retry-revoke", accessCtrl.RetryRevokeAuthorization)
	r.POST("/authorizations/retry-failed-revokes", accessCtrl.RetryFailedRevocations)

	return r, accessCtrl
}

func TestRevocationConsistency_FabricFailureAndIdempotency(t *testing.T) {
	_, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	_, err = repository.InitDB()
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}

	// 1. 创建测试患者
	patient := model.User{
		UserNo:   "PAT_REVOKE_TEST",
		Username: "PAT_REVOKE_TEST",
		RealName: "撤销测试患者",
		Role:     "patient",
		Status:   "NORMAL",
	}
	repository.DB.Where("user_no = ?", patient.UserNo).Delete(&model.User{})
	repository.DB.Create(&patient)
	defer repository.DB.Delete(&patient)

	router, _ := setupTestRouter(patient.ID, "patient")

	// 2. 创建 ACTIVE 状态的授权
	authNo1 := fmt.Sprintf("AUTH_TEST_REVOKE_FAIL_%d", time.Now().UnixNano())
	authFail := model.Authorization{
		AuthNo:         authNo1,
		PatientID:      patient.ID,
		AuthTargetType: "DOCTOR",
		AuthTargetID:   999,
		ScopeType:      "ALL",
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(24 * time.Hour),
		Status:         "ACTIVE",
		CreatedAt:      time.Now(),
	}
	repository.DB.Create(&authFail)
	defer repository.DB.Delete(&authFail)

	// 3. 模拟 Fabric 服务不可用 (置为 nil)
	originalBlockchain := blockchain.DefaultService
	blockchain.DefaultService = nil

	// 发起撤销请求
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/authorizations/%d", authFail.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 断言：Fabric 不可用时必须返回 503，严禁假成功返回 200
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("Fabric 为 nil 时预期返回 503，实际返回: %d (响应: %s)", w.Code, w.Body.String())
	}

	// 断言数据库状态：严禁被篡改为 REVOKED，必须被标记为 REVOKE_FAILED 并保存错误信息
	var checkAuth model.Authorization
	repository.DB.First(&checkAuth, authFail.ID)
	if checkAuth.Status != "REVOKE_FAILED" {
		t.Fatalf("Fabric 故障时预期数据库状态为 REVOKE_FAILED，实际为: %s", checkAuth.Status)
	}
	if checkAuth.RevokeError == "" {
		t.Fatalf("REVOKE_FAILED 状态下预期保留详细错误信息，但 revoke_error 为空")
	}
	if checkAuth.OperatorID != patient.ID {
		t.Fatalf("预期记录操作者 OperatorID=%d，实际为: %d", patient.ID, checkAuth.OperatorID)
	}

	// 4. 幂等性测试：对于已处于 REVOKED 状态的授权，重复请求应安全幂等返回 200
	authRevoked := model.Authorization{
		AuthNo:         fmt.Sprintf("AUTH_TEST_IDEMP_%d", time.Now().UnixNano()),
		PatientID:      patient.ID,
		AuthTargetType: "DOCTOR",
		AuthTargetID:   999,
		ScopeType:      "ALL",
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(24 * time.Hour),
		Status:         "REVOKED",
		RevokeTxID:     "FABRIC_TX_EXISTING_12345",
		CreatedAt:      time.Now(),
	}
	repository.DB.Create(&authRevoked)
	defer repository.DB.Delete(&authRevoked)

	reqIdemp := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/authorizations/%d", authRevoked.ID), nil)
	wIdemp := httptest.NewRecorder()
	router.ServeHTTP(wIdemp, reqIdemp)

	if wIdemp.Code != http.StatusOK {
		t.Fatalf("对已撤销记录重复撤销应幂等返回 200，实际返回: %d (响应: %s)", wIdemp.Code, wIdemp.Body.String())
	}

	// 5. 恢复 Fabric 服务，测试正常撤销与重试补偿
	if originalBlockchain != nil {
		blockchain.DefaultService = originalBlockchain
	} else {
		blockchain.InitBlockchainService("../ledger_data", blockchain.FabricGatewayConfig{Mode: "local"})
	}

	// 5.1 正常撤销流程 (ACTIVE -> REVOKED)
	authNormal := model.Authorization{
		AuthNo:         fmt.Sprintf("AUTH_TEST_NORMAL_%d", time.Now().UnixNano()),
		PatientID:      patient.ID,
		AuthTargetType: "DOCTOR",
		AuthTargetID:   999,
		ScopeType:      "ALL",
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(24 * time.Hour),
		Status:         "ACTIVE",
		CreatedAt:      time.Now(),
	}
	repository.DB.Create(&authNormal)
	defer repository.DB.Delete(&authNormal)

	reqNormal := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/authorizations/%d", authNormal.ID), nil)
	wNormal := httptest.NewRecorder()
	router.ServeHTTP(wNormal, reqNormal)

	if wNormal.Code != http.StatusOK {
		t.Fatalf("正常撤销预期返回 200，实际返回: %d (响应: %s)", wNormal.Code, wNormal.Body.String())
	}

	var checkNormal model.Authorization
	repository.DB.First(&checkNormal, authNormal.ID)
	if checkNormal.Status != "REVOKED" {
		t.Fatalf("正常撤销后状态预期为 REVOKED，实际为: %s", checkNormal.Status)
	}
	if checkNormal.RevokeTxID == "" {
		t.Fatalf("正常撤销后 Fabric 存证交易 ID 未保存")
	}

	// 5.2 单条重试补偿流程 (REVOKE_FAILED -> RetryRevoke -> REVOKED)
	reqRetry := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/authorizations/%d/retry-revoke", authFail.ID), nil)
	wRetry := httptest.NewRecorder()
	router.ServeHTTP(wRetry, reqRetry)

	if wRetry.Code != http.StatusOK {
		t.Fatalf("重试补偿预期返回 200，实际返回: %d (响应: %s)", wRetry.Code, wRetry.Body.String())
	}

	var checkRetried model.Authorization
	repository.DB.First(&checkRetried, authFail.ID)
	if checkRetried.Status != "REVOKED" {
		t.Fatalf("重试补偿后状态预期转化为 REVOKED，实际为: %s", checkRetried.Status)
	}
	if checkRetried.RevokeTxID == "" {
		t.Fatalf("重试补偿后 RevokeTxID 未被固化")
	}
	if checkRetried.RevokeError != "" {
		t.Fatalf("重试补偿成功后 RevokeError 未被清空: %s", checkRetried.RevokeError)
	}
}

func TestBatchRetryFailedRevocations(t *testing.T) {
	_, _ = config.LoadConfig("../config/config.yaml")
	_, _ = repository.InitDB()
	blockchain.InitBlockchainService("../ledger_data", blockchain.FabricGatewayConfig{Mode: "local"})

	patient := model.User{
		UserNo:   "PAT_BATCH_RETRY",
		Username: "PAT_BATCH_RETRY",
		RealName: "批量补偿测试患者",
		Role:     "patient",
		Status:   "NORMAL",
	}
	repository.DB.Where("user_no = ?", patient.UserNo).Delete(&model.User{})
	repository.DB.Create(&patient)
	defer repository.DB.Delete(&patient)

	router, _ := setupTestRouter(patient.ID, "patient")

	// 创建 2 条处于 REVOKE_FAILED 的记录
	auth1 := model.Authorization{
		AuthNo:         fmt.Sprintf("AUTH_BATCH_1_%d", time.Now().UnixNano()),
		PatientID:      patient.ID,
		AuthTargetType: "DOCTOR",
		AuthTargetID:   888,
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(24 * time.Hour),
		Status:         "REVOKE_FAILED",
		RevokeError:    "Simulated network timeout",
		CreatedAt:      time.Now(),
	}
	auth2 := model.Authorization{
		AuthNo:         fmt.Sprintf("AUTH_BATCH_2_%d", time.Now().UnixNano()),
		PatientID:      patient.ID,
		AuthTargetType: "DOCTOR",
		AuthTargetID:   888,
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(24 * time.Hour),
		Status:         "REVOKE_FAILED",
		RevokeError:    "Simulated peer unavailable",
		CreatedAt:      time.Now(),
	}
	repository.DB.Create(&auth1)
	repository.DB.Create(&auth2)
	defer repository.DB.Delete(&auth1)
	defer repository.DB.Delete(&auth2)

	// 执行批量重试
	reqBatch := httptest.NewRequest(http.MethodPost, "/authorizations/retry-failed-revokes", nil)
	wBatch := httptest.NewRecorder()
	router.ServeHTTP(wBatch, reqBatch)

	if wBatch.Code != http.StatusOK {
		t.Fatalf("批量重试预期返回 200，实际返回: %d (响应: %s)", wBatch.Code, wBatch.Body.String())
	}

	var resp struct {
		Code int `json:"code"`
		Data struct {
			RevokedCount int `json:"revoked_count"`
			FailedCount  int `json:"failed_count"`
			Total        int `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wBatch.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应 JSON 失败: %v", err)
	}

	if resp.Data.RevokedCount < 2 {
		t.Fatalf("预期至少成功重试 2 条记录，实际为: %d (响应: %s)", resp.Data.RevokedCount, wBatch.Body.String())
	}

	// 校验数据库状态均已转为 REVOKED
	var chk1, chk2 model.Authorization
	repository.DB.First(&chk1, auth1.ID)
	repository.DB.First(&chk2, auth2.ID)

	if chk1.Status != "REVOKED" || chk2.Status != "REVOKED" {
		t.Fatalf("批量重试后记录状态未更新为 REVOKED: chk1=%s, chk2=%s", chk1.Status, chk2.Status)
	}
}
