package controller_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"medtrust-backend/config"
	"medtrust-backend/controller"
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
	"medtrust-backend/service"
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
	blockchain.InitBlockchainService(filepath.Join(t.TempDir(), "ledger_data"), blockchain.FabricGatewayConfig{Mode: "mock"})

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
		blockchain.InitBlockchainService(filepath.Join(t.TempDir(), "ledger_data"), blockchain.FabricGatewayConfig{Mode: "local"})
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
	blockchain.InitBlockchainService(filepath.Join(t.TempDir(), "ledger_data"), blockchain.FabricGatewayConfig{Mode: "local"})

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

type countedBlockchainService struct {
	blockchain.BlockchainService
	revokeAuthCount  int32
	totalCommitCount int32
}

func (c *countedBlockchainService) CommitAsset(assetType, assetID string, payload map[string]interface{}) (string, uint64, error) {
	atomic.AddInt32(&c.totalCommitCount, 1)
	if assetType == "REVOKE_AUTH" {
		atomic.AddInt32(&c.revokeAuthCount, 1)
	}
	return c.BlockchainService.CommitAsset(assetType, assetID, payload)
}

func TestRevocation_100GoroutinesConcurrency(t *testing.T) {
	_, _ = config.LoadConfig("../config/config.yaml")
	_, _ = repository.InitDB()
	blockchain.InitBlockchainService(filepath.Join(t.TempDir(), "ledger_data"), blockchain.FabricGatewayConfig{Mode: "mock"})

	origBC := blockchain.DefaultService
	mockBC := &countedBlockchainService{
		BlockchainService: origBC,
	}
	blockchain.DefaultService = mockBC
	defer func() {
		blockchain.DefaultService = origBC
	}()

	patient := model.User{
		UserNo:   fmt.Sprintf("PAT_CONCUR_%d", time.Now().UnixNano()),
		Username: fmt.Sprintf("PAT_CONCUR_%d", time.Now().UnixNano()),
		RealName: "并发撤销测试患者",
		Role:     "patient",
		Status:   "NORMAL",
	}
	repository.DB.Create(&patient)
	defer repository.DB.Delete(&patient)

	router, _ := setupTestRouter(patient.ID, "patient")

	auth := model.Authorization{
		AuthNo:         fmt.Sprintf("AUTH_CONCUR_%d", time.Now().UnixNano()),
		PatientID:      patient.ID,
		AuthTargetType: "DOCTOR",
		AuthTargetID:   888,
		ScopeType:      "ALL",
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(24 * time.Hour),
		Status:         "ACTIVE",
		CreatedAt:      time.Now(),
	}
	repository.DB.Create(&auth)
	defer repository.DB.Delete(&auth)

	var wg sync.WaitGroup
	numRequests := 100
	statusCodeChan := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/authorizations/%d", auth.ID), nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			statusCodeChan <- w.Code
		}()
	}

	wg.Wait()
	close(statusCodeChan)

	statusCounts := make(map[int]int)
	for code := range statusCodeChan {
		statusCounts[code]++
	}

	// 核心断言：Fabric CommitAsset("REVOKE_AUTH") 必须被且仅被执行 1 次！
	commits := atomic.LoadInt32(&mockBC.revokeAuthCount)
	if commits != 1 {
		t.Fatalf("预期区块链 CommitAsset(\"REVOKE_AUTH\") 恰好执行 1 次，实际执行了 %d 次 (并发竞争拦截失败)", commits)
	}

	var finalAuth model.Authorization
	repository.DB.First(&finalAuth, auth.ID)
	if finalAuth.Status != "REVOKED" {
		t.Fatalf("预期最终授权状态为 REVOKED，实际为: %s", finalAuth.Status)
	}
	if finalAuth.RevokeTxID == "" {
		t.Fatalf("预期存证交易 ID 非空")
	}

	// 所有 100 个并发请求要么是 200 (首个成功或后续幂等成功)，要么是 409 (处理中冲突)
	for code, count := range statusCounts {
		if code != http.StatusOK && code != http.StatusConflict {
			t.Fatalf("并发测试出现非预期状态码: %d (出现次数: %d)", code, count)
		}
	}
}

func TestRevocation_FabricSuccessDbFailure_Reconciliation(t *testing.T) {
	_, _ = config.LoadConfig("../config/config.yaml")
	_, _ = repository.InitDB()
	blockchain.InitBlockchainService(filepath.Join(t.TempDir(), "ledger_data"), blockchain.FabricGatewayConfig{Mode: "mock"})

	patient := model.User{
		UserNo:   fmt.Sprintf("PAT_RECON_%d", time.Now().UnixNano()),
		Username: fmt.Sprintf("PAT_RECON_%d", time.Now().UnixNano()),
		RealName: "对账修复测试患者",
		Role:     "patient",
		Status:   "NORMAL",
	}
	repository.DB.Create(&patient)
	defer repository.DB.Delete(&patient)

	router, _ := setupTestRouter(patient.ID, "patient")

	auth := model.Authorization{
		AuthNo:         fmt.Sprintf("AUTH_RECON_%d", time.Now().UnixNano()),
		PatientID:      patient.ID,
		AuthTargetType: "DOCTOR",
		AuthTargetID:   888,
		ScopeType:      "ALL",
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(24 * time.Hour),
		Status:         "ACTIVE",
		CreatedAt:      time.Now(),
	}
	repository.DB.Create(&auth)
	defer repository.DB.Delete(&auth)

	// 注入 GORM Callback 模拟：在 Fabric 存证成功后，更新 status 为 REVOKED 时发生数据库故障
	callbackName := "sim_db_failure_revoked_state"
	repository.DB.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement != nil {
			if m, ok := tx.Statement.Dest.(map[string]interface{}); ok {
				if m["status"] == "REVOKED" {
					_ = tx.AddError(errors.New("simulated disk I/O failure on updating status to REVOKED"))
				}
			}
		}
	})

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/authorizations/%d", auth.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 移除故障模拟回调
	repository.DB.Callback().Update().Remove(callbackName)

	// 断言：严禁假装成功返回 200！必须返回 500
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("当 Fabric 成功但本地 DB 持久化失败时，预期返回 500，实际返回: %d (响应: %s)", w.Code, w.Body.String())
	}

	// 断言数据库状态：必须安全流转至 NEEDS_RECONCILIATION 并固化 FabricTxID
	var intermediateAuth model.Authorization
	repository.DB.First(&intermediateAuth, auth.ID)
	if intermediateAuth.Status != "NEEDS_RECONCILIATION" {
		t.Fatalf("预期状态转换为 NEEDS_RECONCILIATION，实际为: %s", intermediateAuth.Status)
	}
	if intermediateAuth.RevokeTxID == "" {
		t.Fatalf("预期已留存链上确权生成的 RevokeTxID，但实际为空")
	}
	if !strings.Contains(intermediateAuth.RevokeError, "本地数据库状态持久化失败") {
		t.Fatalf("预期保留详细对账错误信息，实际为: %s", intermediateAuth.RevokeError)
	}

	// 追踪重试补偿流程：针对处于 NEEDS_RECONCILIATION 的授权发起 retry-revoke
	origBC := blockchain.DefaultService
	mockBC := &countedBlockchainService{BlockchainService: origBC}
	blockchain.DefaultService = mockBC
	defer func() {
		blockchain.DefaultService = origBC
	}()

	reqRetry := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/authorizations/%d/retry-revoke", auth.ID), nil)
	wRetry := httptest.NewRecorder()
	router.ServeHTTP(wRetry, reqRetry)

	if wRetry.Code != http.StatusOK {
		t.Fatalf("对账重试补偿预期返回 200，实际返回: %d (响应: %s)", wRetry.Code, wRetry.Body.String())
	}

	// 验证：因为先前链上已成功确权，本次重试严禁重复调用 Fabric CommitAsset！
	if atomic.LoadInt32(&mockBC.revokeAuthCount) != 0 {
		t.Fatalf("NEEDS_RECONCILIATION 对账重试时预期直接对账本地终态，严禁重复提交 Fabric 交易，实际调用次数: %d", mockBC.revokeAuthCount)
	}

	var finalAuth model.Authorization
	repository.DB.First(&finalAuth, auth.ID)
	if finalAuth.Status != "REVOKED" {
		t.Fatalf("对账重试后预期状态变为 REVOKED，实际为: %s", finalAuth.Status)
	}
	if finalAuth.RevokeError != "" {
		t.Fatalf("对账重试成功后预期清空 RevokeError，实际为: %s", finalAuth.RevokeError)
	}
}

func setupMedicalFullRouter(userID uint64, role string, hospID uint64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("role", role)
		c.Set("hospital_id", hospID)
		c.Next()
	})
	r.GET("/medical-records/:id", controller.DefaultMedicalController.GetByID)
	r.GET("/medical-records/:id/download", controller.DefaultMedicalController.Download)
	r.GET("/medical-files/:id/view", controller.DefaultMedicalController.ViewMedicalFile)
	r.GET("/medical-files/:id/download", controller.DefaultMedicalController.DownloadMedicalFile)
	r.GET("/medical-records/:id/infection-risks", controller.DefaultMedicalController.GetRecordInfectionRisks)
	return r
}

func TestMedicalData_AccessEngineUnifiedCoverage(t *testing.T) {
	_, _ = config.LoadConfig("../config/config.yaml")
	_, _ = repository.InitDB()
	blockchain.InitBlockchainService(filepath.Join(t.TempDir(), "ledger_data"), blockchain.FabricGatewayConfig{Mode: "mock"})
	service.InitAccessEngine(20, 60)
	service.GetInfectionService()

	// 1. 创建机构与测试用户
	hospA := model.Hospital{HospitalNo: fmt.Sprintf("HOSP_A_%d", time.Now().UnixNano()), Name: "医院A"}
	hospB := model.Hospital{HospitalNo: fmt.Sprintf("HOSP_B_%d", time.Now().UnixNano()+1), Name: "医院B"}
	repository.DB.Create(&hospA)
	repository.DB.Create(&hospB)
	defer repository.DB.Delete(&hospA)
	defer repository.DB.Delete(&hospB)

	patA := model.User{UserNo: fmt.Sprintf("PAT_A_%d", time.Now().UnixNano()), Username: fmt.Sprintf("PAT_A_%d", time.Now().UnixNano()), RealName: "患者张三", Role: "patient", Status: "NORMAL"}
	patB := model.User{UserNo: fmt.Sprintf("PAT_B_%d", time.Now().UnixNano()), Username: fmt.Sprintf("PAT_B_%d", time.Now().UnixNano()), RealName: "患者李四", Role: "patient", Status: "NORMAL"}
	docA := model.User{UserNo: fmt.Sprintf("DOC_A_%d", time.Now().UnixNano()), Username: fmt.Sprintf("DOC_A_%d", time.Now().UnixNano()), RealName: "医生王五", Role: "doctor", HospitalID: hospA.ID, Status: "NORMAL"}
	adminUser := model.User{UserNo: fmt.Sprintf("ADM_%d", time.Now().UnixNano()), Username: fmt.Sprintf("ADM_%d", time.Now().UnixNano()), RealName: "管理赵六", Role: "admin", Status: "NORMAL"}
	supervisorUser := model.User{UserNo: fmt.Sprintf("SUP_%d", time.Now().UnixNano()), Username: fmt.Sprintf("SUP_%d", time.Now().UnixNano()), RealName: "监管钱七", Role: "supervisor", Status: "NORMAL"}

	repository.DB.Create(&patA)
	repository.DB.Create(&patB)
	repository.DB.Create(&docA)
	repository.DB.Create(&adminUser)
	repository.DB.Create(&supervisorUser)
	defer repository.DB.Delete(&patA)
	defer repository.DB.Delete(&patB)
	defer repository.DB.Delete(&docA)
	defer repository.DB.Delete(&adminUser)
	defer repository.DB.Delete(&supervisorUser)

	// 创建属于患者 A 的病历与附件
	recA := model.MedicalRecord{
		RecordNo:       fmt.Sprintf("REC_A_%d", time.Now().UnixNano()),
		PatientID:      patA.ID,
		DoctorID:       docA.ID,
		HospitalID:     hospA.ID,
		Diagnosis:      "急性化脓性扁桃体炎伴高热",
		Symptoms:       "发热咽痛",
		ChiefComplaint: "咽痛3天伴畏寒发热",
		Status:         "COMPLETED",
		CreatedAt:      time.Now(),
	}
	repository.DB.Create(&recA)
	defer repository.DB.Delete(&recA)

	fileA := model.MedicalFile{
		RecordID:  recA.ID,
		FileName:  "咽部拭子生化报告.pdf",
		FileType:  "pdf",
		FileSize:  1024,
		IPFSCID:   "QmMockCID1234567890",
		FileHash:  "mockhash123456",
		CreatedAt: time.Now(),
	}
	repository.DB.Create(&fileA)
	defer repository.DB.Delete(&fileA)

	// 场景 1: 患者越权 (IDOR) - 患者 B 试图调阅患者 A 的病历、下载凭据、查阅传染病档案
	rPatB := setupMedicalFullRouter(patB.ID, "patient", 0)

	w := httptest.NewRecorder()
	rPatB.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/medical-records/%d", recA.ID), nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("患者越权查阅病历预期 403，实际: %d", w.Code)
	}

	w = httptest.NewRecorder()
	rPatB.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/medical-records/%d/download", recA.ID), nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("患者越权下载病历预期 403，实际: %d", w.Code)
	}

	w = httptest.NewRecorder()
	rPatB.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/medical-files/%d/view", fileA.ID), nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("患者越权预览附件预期 403，实际: %d", w.Code)
	}

	w = httptest.NewRecorder()
	rPatB.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/medical-records/%d/infection-risks", recA.ID), nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("患者越权查看传染病分析预期 403，实际: %d", w.Code)
	}

	// 场景 2: 系统管理员职能隔离 - 管理员调阅临床病历明文与下载凭据均应拦截
	rAdmin := setupMedicalFullRouter(adminUser.ID, "admin", 0)

	w = httptest.NewRecorder()
	rAdmin.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/medical-records/%d", recA.ID), nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("管理员调阅临床病历明文预期 403，实际: %d", w.Code)
	}

	w = httptest.NewRecorder()
	rAdmin.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/medical-records/%d/download", recA.ID), nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("管理员下载临床凭据预期 403，实际: %d", w.Code)
	}

	// 场景 3: 监管人员权限边界 - 监管员查阅病历时敏感字段必须脱敏，且禁止下载临床凭据与附件
	rSup := setupMedicalFullRouter(supervisorUser.ID, "supervisor", 0)

	w = httptest.NewRecorder()
	rSup.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/medical-records/%d/download", recA.ID), nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("监管员下载临床凭据预期 403，实际: %d", w.Code)
	}

	w = httptest.NewRecorder()
	rSup.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/medical-files/%d/download", fileA.ID), nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("监管员下载临床附件原件预期 403，实际: %d", w.Code)
	}

	// 监管员查阅病历详情应返回 200，但临床诊断、症状等敏感信息已执行字段级脱敏
	w = httptest.NewRecorder()
	rSup.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/medical-records/%d", recA.ID), nil))
	if w.Code != http.StatusOK {
		t.Fatalf("监管员查阅病历详情预期 200，实际: %d", w.Code)
	}
	var supResp struct {
		Code int                 `json:"code"`
		Data model.MedicalRecord `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &supResp)
	if !strings.Contains(supResp.Data.Diagnosis, "【监管审计脱敏屏蔽】") {
		t.Fatalf("监管员查阅病历未完成诊断脱敏: %s", supResp.Data.Diagnosis)
	}

	// 场景 4: 患者本人查阅自己本人的病历与凭据，应正常放行 200
	rPatA := setupMedicalFullRouter(patA.ID, "patient", 0)
	w = httptest.NewRecorder()
	rPatA.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/medical-records/%d", recA.ID), nil))
	if w.Code != http.StatusOK {
		t.Fatalf("患者本人查阅自身病历预期 200，实际: %d", w.Code)
	}
}
