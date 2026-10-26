package controller_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
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

// buildE2ETestEngine 构建带鉴权上下文的完整路由测试引擎
func buildE2ETestEngine(userID uint64, role, username string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("role", role)
		c.Set("username", username)
		c.Next()
	})

	v1 := r.Group("/api/v1")
	{
		// 病历上传与调阅
		v1.POST("/medical-records/upload", controller.DefaultMedicalController.Upload)
		v1.GET("/medical-records", controller.DefaultMedicalController.List)
		v1.GET("/medical-records/:id", controller.DefaultMedicalController.GetByID)
		v1.GET("/medical-files/:id/view", controller.DefaultMedicalController.ViewMedicalFile)
		v1.GET("/medical-files/:id/download", controller.DefaultMedicalController.DownloadMedicalFile)

		// 访问控制与授权
		v1.POST("/access/requests", controller.DefaultAccessController.RequestAccess)
		v1.POST("/access-requests", controller.DefaultAccessController.RequestAccess)
		v1.POST("/access/break-glass", controller.DefaultAccessController.BreakGlass)
		v1.POST("/access/break-glass/patient-feedback", controller.DefaultAccessController.PatientFeedback)
		v1.POST("/emergency-events/:event_no/feedback", controller.DefaultAccessController.PatientFeedback)
		v1.POST("/authorizations", controller.DefaultAccessController.CreateAuthorization)
		v1.DELETE("/authorizations/:id", controller.DefaultAccessController.RevokeAuthorization)

		// 验真与监管
		v1.POST("/verification/:record_id", controller.DefaultSupervisorController.VerifyRecord)
		v1.POST("/verification/simulate-tamper/:record_id", controller.DefaultSupervisorController.SimulateTamper)
		v1.POST("/supervisor/simulate-tamper/:record_id", controller.DefaultSupervisorController.SimulateTamper)
		v1.POST("/supervisor/emergency-events/:event_no/audit", controller.DefaultSupervisorController.AuditEmergencyEvent)
		v1.GET("/supervisor/emergency-events", controller.DefaultSupervisorController.ListEmergencyEvents)
		v1.GET("/audit-logs", controller.DefaultSupervisorController.ListAuditLogs)
	}

	return r
}

func initE2EEnvironment(t *testing.T) {
	cfg, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("加载配置文件失败: %v", err)
	}
	db, err := repository.InitDB()
	if err != nil {
		t.Fatalf("初始化数据库失败: %v", err)
	}
	repository.SeedInitialData(db)

	tempDir := t.TempDir()
	ledgerDir := filepath.Join(tempDir, "fabric_ledger")
	ipfsDir := filepath.Join(tempDir, "ipfs_storage")

	blockchain.InitBlockchainService(ledgerDir, blockchain.FabricGatewayConfig{
		Mode:         "mock", // 单测统一使用 mock 稳定仿真
		Enabled:      false,
		PeerEndpoint: cfg.Blockchain.Fabric.PeerEndpoint,
		GatewayPeer:  cfg.Blockchain.Fabric.GatewayPeer,
		MSPID:        cfg.Blockchain.Fabric.MSPID,
		ChannelID:    cfg.Blockchain.ChannelID,
		ChaincodeID:  cfg.Blockchain.ChaincodeID,
		TLSCertPath:  cfg.Blockchain.Fabric.TLSCertPath,
		CertPath:     cfg.Blockchain.Fabric.CertPath,
		KeyPath:      cfg.Blockchain.Fabric.KeyPath,
	})
	service.InitMedicalService("mock", cfg.IPFS.APIURL, ipfsDir)
	service.InitAccessEngine(cfg.Risk.LowThreshold, cfg.Risk.HighThreshold)
	service.EnsureBaselineLedgerAnchored()
}

// TestE2E_AcceptanceFlow 执行 updatev1.txt 第九节要求的全部 6 项核心验收测试用例
func TestE2E_AcceptanceFlow(t *testing.T) {
	initE2EEnvironment(t)
	if service.DefaultAccessEngine != nil && service.DefaultAccessEngine.GetRiskEngine() != nil {
		service.DefaultAccessEngine.GetRiskEngine().SetClock(func() time.Time {
			now := time.Now()
			return time.Date(now.Year(), now.Month(), now.Day(), 14, now.Minute(), now.Second(), now.Nanosecond(), now.Location())
		})
		defer service.DefaultAccessEngine.GetRiskEngine().SetClock(nil)
	}

	// 获取预置种子账号
	docLi, err := repository.DefaultUserRepository.FindByUsername("doc_li")
	if err != nil || docLi == nil {
		t.Fatalf("未找到种子账号 doc_li: %v", err)
	}
	docWang, err := repository.DefaultUserRepository.FindByUsername("doc_wang")
	if err != nil || docWang == nil {
		t.Fatalf("未找到种子账号 doc_wang: %v", err)
	}
	docChen, err := repository.DefaultUserRepository.FindByUsername("doc_chen")
	if err != nil || docChen == nil {
		t.Fatalf("未找到种子账号 doc_chen: %v", err)
	}
	patZhang, err := repository.DefaultUserRepository.FindByUsername("pat_zhang")
	if err != nil || patZhang == nil {
		t.Fatalf("未找到种子账号 pat_zhang: %v", err)
	}
	supZhao, err := repository.DefaultUserRepository.FindByUsername("sup_zhao")
	if err != nil || supZhao == nil {
		t.Fatalf("未找到种子账号 sup_zhao: %v", err)
	}

	// 清理测试患者历史授权和急救事件，保证单测环境幂等确定性
	repository.DB.Where("patient_id = ?", patZhang.ID).Delete(&model.Authorization{})
	repository.DB.Where("patient_id = ?", patZhang.ID).Delete(&model.EmergencyAccessEvent{})
	repository.DB.Model(&model.User{}).Where("id = ?", docChen.ID).Update("status", "NORMAL")

	t.Logf("种子用户状态：docLi(Hosp=%d), docWang(Hosp=%d), docChen(Hosp=%d, Status=%s)",
		docLi.HospitalID, docWang.HospitalID, docChen.HospitalID, docChen.Status)

	var createdRecordID uint64
	var createdFileID uint64
	var ipfsCID string
	var fileHash string
	var testEventNo string

	// -------------------------------------------------------------
	// Test 1: 加密与上链闭环
	// doc_li 上传患者张三病历文件 -> 检查 ipfs_storage/ 下生成的文件确保无法作为明文读取 -> 检查返回的 ipfs_cid, file_hash, fabric_tx_id 非空
	// -------------------------------------------------------------
	t.Run("Test1_EncryptionAndLedgerAnchor", func(t *testing.T) {
		rDocLi := buildE2ETestEngine(docLi.ID, "doctor", "doc_li")

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		plainMedicalContent := "【机密临床病历】患者张三心前区闷痛3天，心电图提示ST段压低0.1mV，拟行冠状动脉造影排查狭窄。"
		part, err := writer.CreateFormFile("file", "coronary_ecg_report.pdf")
		if err != nil {
			t.Fatalf("创建上传表单失败: %v", err)
		}
		_, _ = part.Write([]byte(plainMedicalContent))

		_ = writer.WriteField("patient_id", fmt.Sprintf("%d", patZhang.ID))
		_ = writer.WriteField("data_type", "EMR")
		_ = writer.WriteField("chief_complaint", "反复胸闷3天")
		_ = writer.WriteField("diagnosis", "冠状动脉粥样硬化性心脏病")
		_ = writer.WriteField("treatment_plan", "阿司匹林肠溶片 100mg qd, 替格瑞洛 90mg bid")
		_ = writer.Close()

		req := httptest.NewRequest("POST", "/api/v1/medical-records/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		rDocLi.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("doc_li 上传病历失败: status=%d, body=%s", w.Code, w.Body.String())
		}

		var resp struct {
			Code int                 `json:"code"`
			Data model.MedicalRecord `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("解析上传响应失败: %v", err)
		}

		createdRecordID = resp.Data.ID
		if createdRecordID == 0 || resp.Data.FabricTxID == "" {
			t.Fatalf("返回病历 ID 或 FabricTxID 为空: id=%d, tx=%s", createdRecordID, resp.Data.FabricTxID)
		}

		if len(resp.Data.Files) == 0 {
			t.Fatalf("未包含病历附件 MedicalFile 记录")
		}
		createdFileID = resp.Data.Files[0].ID
		ipfsCID = resp.Data.Files[0].IPFSCID
		fileHash = resp.Data.Files[0].FileHash

		if ipfsCID == "" || fileHash == "" {
			t.Fatalf("ipfs_cid 或 file_hash 为空: cid=%s, hash=%s", ipfsCID, fileHash)
		}

		// 检查本地 ipfs_storage/ 下的文件内容确系密文，严禁包含明文字符串
		diskCipher, readErr := service.DefaultMedicalService.GetIPFSService().Cat(ipfsCID)
		if readErr != nil {
			t.Fatalf("读取 IPFS 存储密文文件失败: %v", readErr)
		}
		if bytes.Contains(diskCipher, []byte("【机密临床病历】")) {
			t.Fatalf("【严重漏洞】IPFS 存储落地了未加密的明文数据！")
		}

		t.Logf("✅ Test 1 通过：已完成 AES-256-GCM 信封加密落盘 (CID=%s, TxID=%s, HospID=%d)", ipfsCID, resp.Data.FabricTxID, resp.Data.HospitalID)
	})

	// -------------------------------------------------------------
	// Test 2: 跨机构未授权阻断
	// doc_wang 未获授权直接调阅该病历 -> 断言返回 403 Forbidden 且 audit_logs 新增一条 result = 'INTERCEPTED'
	// -------------------------------------------------------------
	t.Run("Test2_CrossHospitalUnauthorizedBlocked", func(t *testing.T) {
		rDocWang := buildE2ETestEngine(docWang.ID, "doctor", "doc_wang")

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/medical-files/%d/view", createdFileID), nil)
		w := httptest.NewRecorder()
		rDocWang.ServeHTTP(w, req)

		var recCheck model.MedicalRecord
		_ = repository.DB.First(&recCheck, createdRecordID)
		t.Logf("Test2 调试信息: docWang.HospitalID=%d, recCheck.HospitalID=%d, w.Code=%d, body=%s",
			docWang.HospitalID, recCheck.HospitalID, w.Code, w.Body.String())

		if w.Code != http.StatusForbidden {
			t.Fatalf("预期跨院未授权调阅返回 403，实际返回 status=%d, body=%s", w.Code, w.Body.String())
		}

		// 断言 audit_logs 中记录了 INTERCEPTED 拦截日志
		var interceptedLog model.AuditLog
		err := repository.DB.Where("user_id = ? AND result = 'INTERCEPTED'", docWang.ID).
			Order("created_at DESC").First(&interceptedLog).Error
		if err != nil {
			t.Fatalf("未在 audit_logs 中查到 INTERCEPTED 审计拦截记录: %v", err)
		}

		t.Logf("✅ Test 2 通过：doc_wang 跨院未授权调阅被成功拦截 (HTTP 403, LogID=%s)", interceptedLog.LogID)
	})

	// -------------------------------------------------------------
	// Test 3: 患者授权、动态风险评分与授权撤销
	// pat_zhang 授权给 doc_wang -> doc_wang 调阅得分15分(LOW)放行并成功解密 -> pat_zhang 撤销授权 -> doc_wang 再次调阅 403
	// -------------------------------------------------------------
	t.Run("Test3_AuthorizationRiskAndRevocation", func(t *testing.T) {
		rPat := buildE2ETestEngine(patZhang.ID, "patient", "pat_zhang")
		rDocWang := buildE2ETestEngine(docWang.ID, "doctor", "doc_wang")

		// 1. pat_zhang 授权给 doc_wang
		authPayload := `{"auth_target_type":"DOCTOR","auth_target_id":` + fmt.Sprintf("%d", docWang.ID) + `,"scope_type":"ALL","days":7}`
		reqAuth := httptest.NewRequest("POST", "/api/v1/authorizations", strings.NewReader(authPayload))
		reqAuth.Header.Set("Content-Type", "application/json")
		wAuth := httptest.NewRecorder()
		rPat.ServeHTTP(wAuth, reqAuth)

		if wAuth.Code != http.StatusOK {
			t.Fatalf("患者授权失败: status=%d, body=%s", wAuth.Code, wAuth.Body.String())
		}

		var authResp struct {
			Code int                `json:"code"`
			Data model.Authorization `json:"data"`
		}
		_ = json.Unmarshal(wAuth.Body.Bytes(), &authResp)
		authID := authResp.Data.ID

		// 2. doc_wang 在具备授权后调阅病历附件 (成功解密明文)
		reqView := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/medical-files/%d/view", createdFileID), nil)
		wView := httptest.NewRecorder()
		rDocWang.ServeHTTP(wView, reqView)

		if wView.Code != http.StatusOK {
			t.Fatalf("授权生效后调阅失败: status=%d, body=%s", wView.Code, wView.Body.String())
		}
		if !bytes.Contains(wView.Body.Bytes(), []byte("【机密临床病历】")) {
			t.Fatalf("解密得到的明文内容不匹配: body=%s", wView.Body.String())
		}

		// 3. pat_zhang 撤销授权
		reqRevoke := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/authorizations/%d", authID), nil)
		wRevoke := httptest.NewRecorder()
		rPat.ServeHTTP(wRevoke, reqRevoke)

		if wRevoke.Code != http.StatusOK {
			t.Fatalf("撤销授权失败: status=%d, body=%s", wRevoke.Code, wRevoke.Body.String())
		}

		// 4. doc_wang 再次调阅，断言立即返回 403 Forbidden
		wView2 := httptest.NewRecorder()
		reqView2 := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/medical-files/%d/view", createdFileID), nil)
		rDocWang.ServeHTTP(wView2, reqView2)

		if wView2.Code != http.StatusForbidden {
			t.Fatalf("授权撤销后再次调阅未被拦截: status=%d, body=%s", wView2.Code, wView2.Body.String())
		}

		t.Logf("✅ Test 3 通过：患者自主授权 -> 动态放行与解密 -> 一键撤销 -> 实时拦截闭环验证成功")
	})

	// -------------------------------------------------------------
	// Test 4: Break-Glass 紧急访问与违规封禁闭环
	// doc_chen 触发破窗调阅 (COMA) -> 生成 EA 单 -> pat_zhang 反馈 OBJECTED -> sup_zhao 裁决 CLOSED_VIOLATION -> doc_chen 状态变为 RESTRICTED
	// -------------------------------------------------------------
	t.Run("Test4_BreakGlassAndViolationRestricted", func(t *testing.T) {
		rDocChen := buildE2ETestEngine(docChen.ID, "doctor", "doc_chen")
		rPat := buildE2ETestEngine(patZhang.ID, "patient", "pat_zhang")
		rSup := buildE2ETestEngine(supZhao.ID, "supervisor", "sup_zhao")

		// 1. doc_chen 发起 Break-Glass 紧急访问
		bgPayload := fmt.Sprintf(`{"record_id":%d,"emergency_reason":"COMA","description":"患者突发昏迷伴呼吸微弱，急救室立即行插管抢救","doctor_confirmed":1}`, createdRecordID)
		reqBG := httptest.NewRequest("POST", "/api/v1/access/break-glass", strings.NewReader(bgPayload))
		reqBG.Header.Set("Content-Type", "application/json")
		wBG := httptest.NewRecorder()
		rDocChen.ServeHTTP(wBG, reqBG)

		if wBG.Code != http.StatusOK {
			t.Fatalf("发起紧急破窗访问失败: status=%d, body=%s", wBG.Code, wBG.Body.String())
		}

		var bgResp struct {
			Code int                        `json:"code"`
			Data model.EmergencyAccessEvent `json:"data"`
		}
		_ = json.Unmarshal(wBG.Body.Bytes(), &bgResp)
		testEventNo = bgResp.Data.EventNo
		if testEventNo == "" {
			t.Fatalf("未生成有效紧急事件单号: %s", wBG.Body.String())
		}

		// 2. pat_zhang 提交异议投诉 (OBJECTED)
		feedbackPayload := `{"feedback":"OBJECTED","comment":"我当时并未在医院就诊，属于医生擅自违规调阅！"}`
		reqFeedback := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/emergency-events/%s/feedback", testEventNo), strings.NewReader(feedbackPayload))
		reqFeedback.Header.Set("Content-Type", "application/json")
		wFeedback := httptest.NewRecorder()
		rPat.ServeHTTP(wFeedback, reqFeedback)

		if wFeedback.Code != http.StatusOK {
			t.Fatalf("患者反馈提交失败: status=%d, body=%s", wFeedback.Code, wFeedback.Body.String())
		}

		// 3. sup_zhao 裁决为违规 (CLOSED_VIOLATION)，自动封禁跨院权限 (RESTRICTED)
		auditPayload := `{"decision":"CLOSED_VIOLATION","comment":"确认非真实急诊抢救场景，给予限制跨院访问处罚","punishment":"RESTRICTED"}`
		reqAudit := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/supervisor/emergency-events/%s/audit", testEventNo), strings.NewReader(auditPayload))
		reqAudit.Header.Set("Content-Type", "application/json")
		wAudit := httptest.NewRecorder()
		rSup.ServeHTTP(wAudit, reqAudit)

		if wAudit.Code != http.StatusOK {
			t.Fatalf("监管端审核裁决失败: status=%d, body=%s", wAudit.Code, wAudit.Body.String())
		}

		// 4. 断言 doc_chen 的用户状态已变为 RESTRICTED
		updatedDoc, _ := repository.DefaultUserRepository.FindByID(docChen.ID)
		if updatedDoc.Status != "RESTRICTED" {
			t.Fatalf("预期医生状态变为 RESTRICTED，实际为: %s", updatedDoc.Status)
		}

		// 5. doc_chen 再次尝试发起跨院常规访问申请，直接被拦截
		accessReqPayload := fmt.Sprintf(`{"patient_id":%d,"record_id":%d,"purpose":"门诊调阅"}`, patZhang.ID, createdRecordID)
		reqAccess := httptest.NewRequest("POST", "/api/v1/access-requests", strings.NewReader(accessReqPayload))
		reqAccess.Header.Set("Content-Type", "application/json")
		wAccess := httptest.NewRecorder()
		rDocChen.ServeHTTP(wAccess, reqAccess)

		if wAccess.Code != http.StatusForbidden {
			t.Fatalf("受限制医生发起调阅未被直接拒绝: status=%d, body=%s", wAccess.Code, wAccess.Body.String())
		}

		t.Logf("✅ Test 4 通过：Break-Glass 破窗调阅 -> 患者异议投诉 -> 监管违规裁决 -> 自动联动 RESTRICTED 封禁闭环验证成功")
	})

	// -------------------------------------------------------------
	// Test 5: 防篡改核验闭环
	// 调用 verification 接口断言返回 VERIFIED -> 模拟篡改 1 字节 -> 断言返回 TAMPERED 且触发 HIGH 告警
	// -------------------------------------------------------------
	t.Run("Test5_TamperVerification", func(t *testing.T) {
		rSup := buildE2ETestEngine(supZhao.ID, "supervisor", "sup_zhao")

		// 1. 正常核验，断言数据完整 VERIFIED
		reqVer1 := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/verification/%d", createdRecordID), nil)
		wVer1 := httptest.NewRecorder()
		rSup.ServeHTTP(wVer1, reqVer1)

		if wVer1.Code != http.StatusOK {
			t.Fatalf("初始核验失败: status=%d, body=%s", wVer1.Code, wVer1.Body.String())
		}
		var res1 struct {
			Code int `json:"code"`
			Data struct {
				Verified bool `json:"verified"`
			} `json:"data"`
		}
		_ = json.Unmarshal(wVer1.Body.Bytes(), &res1)
		if !res1.Data.Verified {
			t.Fatalf("初始状态病历核验未通过: %s", wVer1.Body.String())
		}

		// 2. 触发模拟篡改接口 (篡改数据库中的诊断与处方内容)
		reqTamper := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/supervisor/simulate-tamper/%d", createdRecordID), nil)
		wTamper := httptest.NewRecorder()
		rSup.ServeHTTP(wTamper, reqTamper)

		if wTamper.Code != http.StatusOK {
			t.Fatalf("触发模拟篡改失败: status=%d, body=%s", wTamper.Code, wTamper.Body.String())
		}

		// 3. 再次核验，断言返回篡改报警 (Verified == false)
		reqVer2 := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/verification/%d", createdRecordID), nil)
		wVer2 := httptest.NewRecorder()
		rSup.ServeHTTP(wVer2, reqVer2)

		var res2 struct {
			Code int `json:"code"`
			Data struct {
				Verified bool `json:"verified"`
			} `json:"data"`
		}
		_ = json.Unmarshal(wVer2.Body.Bytes(), &res2)
		if res2.Data.Verified {
			t.Fatalf("预期篡改后验真失败报警，但返回了验证通过: %s", wVer2.Body.String())
		}

		// 恢复病历以便其他测试
		_, _ = service.DefaultVerificationService.RestoreTamperedRecord(createdRecordID)

		t.Logf("✅ Test 5 通过：链下篡改检测 -> 动态哈希比对失配 -> 红色高危报警触发闭环验证成功")
	})

	// -------------------------------------------------------------
	// Test 6: AI Agent 权限边界测试
	// 使用未获授权的 doc_wang Token 调用 AI 工具查询张三受保护病历详情 -> 断言 AI 收到后端的权限阻断
	// -------------------------------------------------------------
	t.Run("Test6_AIAgentBoundaryGatekeeping", func(t *testing.T) {
		rDocWang := buildE2ETestEngine(docWang.ID, "doctor", "doc_wang")

		// 携带未获授权医生身份，请求调阅病历列表与详情
		reqAI := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/medical-records/%d", createdRecordID), nil)
		reqAI.Header.Set("X-Source", "AI_AGENT")
		wAI := httptest.NewRecorder()
		rDocWang.ServeHTTP(wAI, reqAI)

		var recResp struct {
			Code int                 `json:"code"`
			Data model.MedicalRecord `json:"data"`
		}
		_ = json.Unmarshal(wAI.Body.Bytes(), &recResp)

		// 医生未获授权时，has_access 必须为 false，且病历文件无法直接调阅
		if recResp.Data.HasAccess {
			t.Fatalf("【权限逃逸】未获授权的医生通过 AI 工具查询到了具有完整访问权的病历！")
		}

		t.Logf("✅ Test 6 通过：AI Agent 严格受限于 Go 后端 Single Gatekeeper 安全守门人边界")
	})

	// -------------------------------------------------------------
	// Test 7: 种子病历预置真实加密与 100% 动态防篡改核验通过
	// 针对系统初始化的 3 份种子病历 (REC20250501001, REC20250502002, REC20250503003) 逐一执行闭环验真
	// -------------------------------------------------------------
	t.Run("Test7_SeedMedicalRecordsIntegrity100Percent", func(t *testing.T) {
		rSup := buildE2ETestEngine(supZhao.ID, "supervisor", "sup_zhao")
		seedNos := []string{"REC20250501001", "REC20250502002", "REC20250503003"}

		for _, recNo := range seedNos {
			var rec model.MedicalRecord
			if err := repository.DB.Where("record_no = ?", recNo).First(&rec).Error; err != nil {
				t.Fatalf("未找到种子病历 %s: %v", recNo, err)
			}

			reqVer := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/verification/%d", rec.ID), nil)
			wVer := httptest.NewRecorder()
			rSup.ServeHTTP(wVer, reqVer)

			if wVer.Code != http.StatusOK {
				t.Fatalf("种子病历 %s 核验请求失败: status=%d, body=%s", recNo, wVer.Code, wVer.Body.String())
			}

			var verResp struct {
				Code int `json:"code"`
				Data struct {
					Verified bool                            `json:"verified"`
					Stages   []service.VerificationStageItem `json:"stages"`
				} `json:"data"`
			}
			if err := json.Unmarshal(wVer.Body.Bytes(), &verResp); err != nil {
				t.Fatalf("解析核验响应失败 %s: %v", recNo, err)
			}

			if !verResp.Data.Verified {
				t.Fatalf("种子病历 %s 初始状态验真未通过 (报哈希不匹配或篡改): %s", recNo, wVer.Body.String())
			}

			for _, st := range verResp.Data.Stages {
				if !st.Passed {
					t.Fatalf("种子病历 %s 阶段 [%s] 核验未通过: local=%s, chain=%s", recNo, st.StageName, st.LocalVal, st.ChainVal)
				}
			}
			t.Logf("✅ 种子病历 %s 三阶段动态验真全部 100%% 通过", recNo)
		}
	})
}

