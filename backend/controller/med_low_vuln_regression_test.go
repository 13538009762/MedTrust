package controller_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"medtrust-backend/config"
	"medtrust-backend/controller"
	"medtrust-backend/middleware"
	"medtrust-backend/model"
	"medtrust-backend/repository"
	"medtrust-backend/service"
)

// -------------------------------------------------------------------
// V-06: 解除医生限制权限边界 + 生产模式禁用篡改演练接口
// -------------------------------------------------------------------
func TestVuln_V06_LiftRestrictionGuardAndProdSimulateTamperBlocked(t *testing.T) {
	r := initHighVulnTestEnv(t)
	docLi, _, _, patZhang, supZhao, _ := loadSecuritySeedUsers(t)

	ts := time.Now().UnixNano()
	disabledDoc := model.User{
		UserNo:       fmt.Sprintf("DOC_DIS_%d", ts),
		Username:     fmt.Sprintf("doc_dis_%d", ts),
		RealName:     "已封禁医生",
		Role:         "doctor",
		HospitalID:   1,
		DepartmentID: 1,
		Status:       "DISABLED",
	}
	restrictedDoc := model.User{
		UserNo:       fmt.Sprintf("DOC_RES_%d", ts),
		Username:     fmt.Sprintf("doc_res_%d", ts),
		RealName:     "受限待恢复医生",
		Role:         "doctor",
		HospitalID:   1,
		DepartmentID: 1,
		Status:       "RESTRICTED",
	}
	if err := repository.DB.Create(&disabledDoc).Error; err != nil {
		t.Fatalf("创建测试封禁医生失败: %v", err)
	}
	if err := repository.DB.Create(&restrictedDoc).Error; err != nil {
		t.Fatalf("创建测试受限医生失败: %v", err)
	}
	t.Cleanup(func() {
		repository.DB.Unscoped().Delete(&disabledDoc)
		repository.DB.Unscoped().Delete(&restrictedDoc)
	})

	// 1. 监管员尝试用 unrestrict 解封 DISABLED 医生 -> 必须被 403 拦截，状态仍为 DISABLED
	bodyDis := fmt.Sprintf(`{"doctor_id":%d,"comment":"误操作解封"}`, disabledDoc.ID)
	reqDis := httptest.NewRequest(http.MethodPost, "/api/v1/supervisor/lift-doctor-restriction", strings.NewReader(bodyDis))
	reqDis.Header.Set("Authorization", authHeader(t, supZhao))
	reqDis.Header.Set("Content-Type", "application/json")
	wDis := httptest.NewRecorder()
	r.ServeHTTP(wDis, reqDis)
	if wDis.Code != http.StatusForbidden {
		t.Fatalf("【V-06 漏洞】监管员通过 unrestrict 越权解封 DISABLED 医生未被 403 拦截: status=%d body=%s", wDis.Code, wDis.Body.String())
	}
	var chkDis model.User
	repository.DB.First(&chkDis, disabledDoc.ID)
	if chkDis.Status != "DISABLED" {
		t.Fatalf("【V-06 漏洞】被封禁医生状态已被篡改为 %s", chkDis.Status)
	}

	// 2. 监管员尝试对非医生角色(患者)执行 unrestrict -> 必须被 403 拦截
	bodyPat := fmt.Sprintf(`{"doctor_id":%d,"comment":"越权解封患者"}`, patZhang.ID)
	reqPat := httptest.NewRequest(http.MethodPost, "/api/v1/supervisor/lift-doctor-restriction", strings.NewReader(bodyPat))
	reqPat.Header.Set("Authorization", authHeader(t, supZhao))
	reqPat.Header.Set("Content-Type", "application/json")
	wPat := httptest.NewRecorder()
	r.ServeHTTP(wPat, reqPat)
	if wPat.Code != http.StatusForbidden {
		t.Fatalf("【V-06 漏洞】监管员对患者调用 unrestrict 未被 403 拦截: status=%d body=%s", wPat.Code, wPat.Body.String())
	}

	// 3. 对照组：监管员正常解除 RESTRICTED 医生 -> 200 OK 且恢复 NORMAL
	bodyRes := fmt.Sprintf(`{"doctor_id":%d,"comment":"复核通过恢复执业"}`, restrictedDoc.ID)
	reqRes := httptest.NewRequest(http.MethodPost, "/api/v1/supervisor/lift-doctor-restriction", strings.NewReader(bodyRes))
	reqRes.Header.Set("Authorization", authHeader(t, supZhao))
	reqRes.Header.Set("Content-Type", "application/json")
	wRes := httptest.NewRecorder()
	r.ServeHTTP(wRes, reqRes)
	if wRes.Code != http.StatusOK {
		t.Fatalf("解除 RESTRICTED 医生失败: status=%d body=%s", wRes.Code, wRes.Body.String())
	}

	// 4. 生产模式下禁止调用篡改演练与演练恢复接口 -> 403 Forbidden
	rec, err := service.DefaultMedicalService.UploadRecord(service.UploadRecordParams{
		DoctorID:  docLi.ID,
		PatientID: patZhang.ID,
		DataType:  "EMR",
		Diagnosis: "V06 生产防演练测试诊断",
	})
	if err != nil || rec == nil {
		t.Fatalf("UploadRecord failed: %v", err)
	}
	t.Cleanup(func() {
		repository.DB.Where("record_id = ?", rec.ID).Delete(&model.MedicalFile{})
		repository.DB.Where("id = ?", rec.ID).Delete(&model.MedicalRecord{})
	})

	origEnv := config.AppConfig.AppEnv
	config.AppConfig.AppEnv = "production"
	t.Cleanup(func() { config.AppConfig.AppEnv = origEnv })

	reqSim := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/verification/simulate-tamper/%d", rec.ID), nil)
	reqSim.Header.Set("Authorization", authHeader(t, supZhao))
	wSim := httptest.NewRecorder()
	r.ServeHTTP(wSim, reqSim)
	if wSim.Code != http.StatusForbidden {
		t.Fatalf("【V-06 漏洞】生产环境仍允许执行 simulate-tamper 篡改演练: status=%d body=%s", wSim.Code, wSim.Body.String())
	}

	reqRestore := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/verification/restore/%d", rec.ID), nil)
	reqRestore.Header.Set("Authorization", authHeader(t, supZhao))
	wRestore := httptest.NewRecorder()
	r.ServeHTTP(wRestore, reqRestore)
	if wRestore.Code != http.StatusForbidden {
		t.Fatalf("【V-06 漏洞】生产环境仍允许执行 restore 演练恢复: status=%d body=%s", wRestore.Code, wRestore.Body.String())
	}
}

// -------------------------------------------------------------------
// V-07: 幂等键按医生隔离 + 密钥解锁受限状态/全局防爆破 + 医疗密钥 bcrypt 落库
// -------------------------------------------------------------------
func TestVuln_V07_ScopedIdempotencyKey_KeyUnlockGuard_BcryptKey(t *testing.T) {
	r := initHighVulnTestEnv(t)
	docLi, docWang, _, patZhang, _, _ := loadSecuritySeedUsers(t)

	ts := time.Now().UnixNano()
	sharedIdemKey := fmt.Sprintf("IDEM_SHARED_%d", ts)

	// 1. doc_li 使用 sharedIdemKey 上传病历；随后 doc_wang 使用相同 IdempotencyKey 上传
	//    绝不能返回 doc_li 的既有病历（防止跨医生信息泄露）
	recLi, err := service.DefaultMedicalService.UploadRecord(service.UploadRecordParams{
		DoctorID:       docLi.ID,
		PatientID:      patZhang.ID,
		DataType:       "EMR",
		Diagnosis:      fmt.Sprintf("李医生机密诊断_%d", ts),
		IdempotencyKey: sharedIdemKey,
	})
	if err != nil || recLi == nil {
		t.Fatalf("doc_li UploadRecord failed: %v", err)
	}
	t.Cleanup(func() {
		repository.DB.Where("record_id = ?", recLi.ID).Delete(&model.MedicalFile{})
		repository.DB.Where("id = ?", recLi.ID).Delete(&model.MedicalRecord{})
	})

	recWang, err := service.DefaultMedicalService.UploadRecord(service.UploadRecordParams{
		DoctorID:       docWang.ID,
		PatientID:      patZhang.ID,
		DataType:       "EMR",
		Diagnosis:      fmt.Sprintf("王医生独立诊断_%d", ts),
		IdempotencyKey: sharedIdemKey,
	})
	if err != nil || recWang == nil {
		t.Fatalf("doc_wang UploadRecord failed: %v", err)
	}
	t.Cleanup(func() {
		if recWang.ID != recLi.ID {
			repository.DB.Where("record_id = ?", recWang.ID).Delete(&model.MedicalFile{})
			repository.DB.Where("id = ?", recWang.ID).Delete(&model.MedicalRecord{})
		}
	})
	if recWang.ID == recLi.ID || recWang.DoctorID != docWang.ID {
		t.Fatalf("【V-07 漏洞】全局 IdempotencyKey 碰撞导致 doc_wang 拿到了 doc_li 的病历 (recLi=%d, recWang=%d)", recLi.ID, recWang.ID)
	}

	// 2. 医疗密钥更新必须采用 bcrypt ($2a$/$2b$) 落库而非快速 SHA-256
	patKeyUser := model.User{
		UserNo:   fmt.Sprintf("PAT_V07_%d", ts),
		Username: fmt.Sprintf("pat_v07_%d", ts),
		RealName: "密钥加固测试患者",
		Role:     "patient",
		Status:   "NORMAL",
	}
	if err := repository.DB.Create(&patKeyUser).Error; err != nil {
		t.Fatalf("创建测试患者失败: %v", err)
	}
	t.Cleanup(func() {
		repository.DB.Where("patient_id = ?", patKeyUser.ID).Delete(&model.Authorization{})
		repository.DB.Unscoped().Delete(&patKeyUser)
	})

	if err := service.DefaultAuthService.UpdateMedicalKey(patKeyUser.ID, "", "889977"); err != nil {
		t.Fatalf("UpdateMedicalKey failed: %v", err)
	}
	var updatedPat model.User
	repository.DB.First(&updatedPat, patKeyUser.ID)
	if !strings.HasPrefix(updatedPat.MedicalKeyHash, "$2a$") && !strings.HasPrefix(updatedPat.MedicalKeyHash, "$2b$") {
		t.Fatalf("【V-07 漏洞】新设置的医疗密钥仍使用弱快速哈希而非 bcrypt: %s", updatedPat.MedicalKeyHash)
	}
	if !service.VerifyMedicalKey("889977", updatedPat.UserNo, updatedPat.MedicalKeyHash, updatedPat.MedicalKey) {
		t.Fatalf("bcrypt 医疗密钥校验应通过")
	}

	// 3. RESTRICTED 受限医生严禁通过患者密钥解锁跨院建权 -> 必须返回 403
	resDoc := model.User{
		UserNo:       fmt.Sprintf("DOC_V07_RES_%d", ts),
		Username:     fmt.Sprintf("doc_v07_res_%d", ts),
		RealName:     "受限测试医生",
		Role:         "doctor",
		HospitalID:   2,
		DepartmentID: 3,
		Status:       "RESTRICTED",
	}
	if err := repository.DB.Create(&resDoc).Error; err != nil {
		t.Fatalf("创建受限测试医生失败: %v", err)
	}
	t.Cleanup(func() {
		repository.DB.Unscoped().Delete(&resDoc)
	})

	unlockBody := fmt.Sprintf(`{"patient_id":%d,"medical_key":"889977","days":3}`, patKeyUser.ID)
	reqResUnlock := httptest.NewRequest(http.MethodPost, "/api/v1/access/unlock-patient-key", strings.NewReader(unlockBody))
	reqResUnlock.Header.Set("Authorization", authHeader(t, &resDoc))
	reqResUnlock.Header.Set("Content-Type", "application/json")
	wResUnlock := httptest.NewRecorder()
	r.ServeHTTP(wResUnlock, reqResUnlock)
	if wResUnlock.Code != http.StatusForbidden {
		t.Fatalf("【V-07 漏洞】RESTRICTED 受限医生通过密钥解锁绕过跨院限制: status=%d body=%s", wResUnlock.Code, wResUnlock.Body.String())
	}

	// 4. 按 patientID 全局防爆破限流：多个不同医生累计针对同一患者连续输错 10 次后，新医生也必须被 429 锁定
	for i := 0; i < 10; i++ {
		service.RecordKeyUnlockResult(uint64(900000+i), patKeyUser.ID, false)
	}
	if locked, _ := service.IsKeyUnlockLocked(uint64(999999), patKeyUser.ID); !locked {
		t.Fatalf("【V-07 漏洞】针对同一患者跨医生累计 10 次失败后未触发患者全局防爆破锁定")
	}
}

// -------------------------------------------------------------------
// V-08: 改密/重置密码使旧 JWT 立即失效 + CreateUser 枚举校验
// -------------------------------------------------------------------
func TestVuln_V08_PasswordChangeRevokesOldJWT_AndCreateUserEnumValidation(t *testing.T) {
	r := initHighVulnTestEnv(t)
	_, _, _, _, _, admin := loadSecuritySeedUsers(t)

	ts := time.Now().UnixNano()
	origPwd := "InitStrongPass2026!"
	pwdHash, _ := bcrypt.GenerateFromPassword([]byte(origPwd), bcrypt.DefaultCost)
	testUser := model.User{
		UserNo:       fmt.Sprintf("DOC_V08_%d", ts),
		Username:     fmt.Sprintf("doc_v08_%d", ts),
		PasswordHash: string(pwdHash),
		RealName:     "令牌失效测试医生",
		Role:         "doctor",
		HospitalID:   1,
		DepartmentID: 1,
		Status:       "NORMAL",
	}
	if err := repository.DB.Create(&testUser).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	t.Cleanup(func() {
		repository.DB.Unscoped().Delete(&testUser)
	})

	// 1. 签发旧 JWT 并确认初始可用
	oldToken := authHeader(t, &testUser)
	reqBefore := httptest.NewRequest(http.MethodGet, "/api/v1/auth/profile", nil)
	reqBefore.Header.Set("Authorization", oldToken)
	wBefore := httptest.NewRecorder()
	r.ServeHTTP(wBefore, reqBefore)
	if wBefore.Code != http.StatusOK {
		t.Fatalf("改密前令牌应可用，实际: %d", wBefore.Code)
	}

	// 2. 调用 PUT /api/v1/auth/password 修改密码
	changePayload, _ := json.Marshal(map[string]string{
		"old_password": origPwd,
		"new_password": "ChangedStrongPass2026#",
	})
	reqChg := httptest.NewRequest(http.MethodPut, "/api/v1/auth/password", bytes.NewReader(changePayload))
	reqChg.Header.Set("Authorization", oldToken)
	reqChg.Header.Set("Content-Type", "application/json")
	wChg := httptest.NewRecorder()
	r.ServeHTTP(wChg, reqChg)
	if wChg.Code != http.StatusOK {
		t.Fatalf("修改密码失败: status=%d body=%s", wChg.Code, wChg.Body.String())
	}

	// 3. 改密后继续使用 oldToken 访问 -> 必须被 401 拒绝
	reqOldAfter := httptest.NewRequest(http.MethodGet, "/api/v1/auth/profile", nil)
	reqOldAfter.Header.Set("Authorization", oldToken)
	wOldAfter := httptest.NewRecorder()
	r.ServeHTTP(wOldAfter, reqOldAfter)
	if wOldAfter.Code != http.StatusUnauthorized {
		t.Fatalf("【V-08 漏洞】修改密码后旧 JWT 仍可继续访问接口: status=%d body=%s", wOldAfter.Code, wOldAfter.Body.String())
	}

	// 4. 改密后重新签发 newToken -> 必须正常 200 放行
	newToken := authHeader(t, &testUser)
	reqNew := httptest.NewRequest(http.MethodGet, "/api/v1/auth/profile", nil)
	reqNew.Header.Set("Authorization", newToken)
	wNew := httptest.NewRecorder()
	r.ServeHTTP(wNew, reqNew)
	if wNew.Code != http.StatusOK {
		t.Fatalf("改密后新签发令牌应正常放行，实际: status=%d body=%s", wNew.Code, wNew.Body.String())
	}

	// 5. 管理员重置该用户密码 -> newToken 也必须立即失效 (401)
	resetRouter := gin.New()
	resetRouter.Use(middleware.JWTAuthMiddleware())
	resetRouter.POST("/api/v1/system/users/:id/reset-password", middleware.RequireRoles("admin"), controller.DefaultSystemController.ResetUserPassword)
	reqReset := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/system/users/%d/reset-password", testUser.ID), strings.NewReader(`{}`))
	reqReset.Header.Set("Authorization", authHeader(t, admin))
	reqReset.Header.Set("Content-Type", "application/json")
	wReset := httptest.NewRecorder()
	resetRouter.ServeHTTP(wReset, reqReset)
	if wReset.Code != http.StatusOK {
		t.Fatalf("管理员重置密码失败: status=%d body=%s", wReset.Code, wReset.Body.String())
	}
	reqAfterReset := httptest.NewRequest(http.MethodGet, "/api/v1/auth/profile", nil)
	reqAfterReset.Header.Set("Authorization", newToken)
	wAfterReset := httptest.NewRecorder()
	r.ServeHTTP(wAfterReset, reqAfterReset)
	if wAfterReset.Code != http.StatusUnauthorized {
		t.Fatalf("【V-08 漏洞】管理员重置密码后原有 JWT 未失效: status=%d body=%s", wAfterReset.Code, wAfterReset.Body.String())
	}

	// 6. CreateUser 严禁写入非法 role / status 枚举 -> 必须返回 400
	badRolePayload := fmt.Sprintf(`{"username":"bad_role_%d","real_name":"非法角色","role":"superadmin","status":"NORMAL"}`, ts)
	reqBadRole := httptest.NewRequest(http.MethodPost, "/api/v1/system/users", strings.NewReader(badRolePayload))
	reqBadRole.Header.Set("Authorization", authHeader(t, admin))
	reqBadRole.Header.Set("Content-Type", "application/json")
	wBadRole := httptest.NewRecorder()
	r.ServeHTTP(wBadRole, reqBadRole)
	if wBadRole.Code != http.StatusBadRequest {
		repository.DB.Where("username = ?", fmt.Sprintf("bad_role_%d", ts)).Unscoped().Delete(&model.User{})
		t.Fatalf("【V-08 漏洞】CreateUser 未校验 role 枚举白名单: status=%d body=%s", wBadRole.Code, wBadRole.Body.String())
	}

	badStatusPayload := fmt.Sprintf(`{"username":"bad_status_%d","real_name":"非法状态","role":"doctor","status":"HACKED"}`, ts)
	reqBadStatus := httptest.NewRequest(http.MethodPost, "/api/v1/system/users", strings.NewReader(badStatusPayload))
	reqBadStatus.Header.Set("Authorization", authHeader(t, admin))
	reqBadStatus.Header.Set("Content-Type", "application/json")
	wBadStatus := httptest.NewRecorder()
	r.ServeHTTP(wBadStatus, reqBadStatus)
	if wBadStatus.Code != http.StatusBadRequest {
		repository.DB.Where("username = ?", fmt.Sprintf("bad_status_%d", ts)).Unscoped().Delete(&model.User{})
		t.Fatalf("【V-08 漏洞】CreateUser 未校验 status 枚举白名单: status=%d body=%s", wBadStatus.Code, wBadStatus.Body.String())
	}
}

// -------------------------------------------------------------------
// V-09: 禁止 CORS * + Credentials 反射 + 生产环境 Redis 无口令拦截
// -------------------------------------------------------------------
func TestVuln_V09_CORSWildcardCredentialsBlocked_AndConfigRedisGuard(t *testing.T) {
	origCors := config.AppConfig.Security.CORSAllowedOrigins
	origEnv := config.AppConfig.AppEnv
	t.Cleanup(func() {
		config.AppConfig.Security.CORSAllowedOrigins = origCors
		config.AppConfig.AppEnv = origEnv
	})

	// 1. 即使配置了 CORS_ALLOWED_ORIGINS=*，也不得反射任意 Origin 并同时带 Allow-Credentials: true
	config.AppConfig.AppEnv = "demo"
	config.AppConfig.Security.CORSAllowedOrigins = "*"
	gin.SetMode(gin.TestMode)
	corsRouter := gin.New()
	corsRouter.Use(middleware.CORSMiddleware())
	corsRouter.GET("/ping", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	reqEvil := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	reqEvil.Header.Set("Origin", "http://evil-attacker.test")
	wEvil := httptest.NewRecorder()
	corsRouter.ServeHTTP(wEvil, reqEvil)
	if wEvil.Header().Get("Access-Control-Allow-Origin") == "http://evil-attacker.test" &&
		wEvil.Header().Get("Access-Control-Allow-Credentials") == "true" {
		t.Fatalf("【V-09 漏洞】CORS 配置通配符 * 时反射任意 Origin 且返回 Allow-Credentials: true")
	}

	// 2. 生产模式下配置了 Redis.Addr 但未配置 Redis.Password -> Validate() 必须报错
	cfg := &config.Config{AppEnv: "production"}
	cfg.Server.JWTSecret = "c3VwZXJfc2VjdXJlX2p3dF9zZWNyZXRfa2V5XzIwMjZfcHJvZA=="
	cfg.Security.AESMasterKey = "k7R9vX2mL4pQ8sY1wZ3bN5cE7tH9uI0j"
	cfg.Database.DSN = "medtrust_app:StrongP@ssw0rd2026!@tcp(10.0.0.5:3306)/medtrust_prod"
	cfg.Redis.Addr = "redis:6379"
	cfg.Redis.Password = ""
	if err := cfg.Validate(); err == nil {
		t.Fatalf("【V-09 漏洞】生产环境配置了 Redis 地址但密码为空，Validate() 未拦截")
	}
}

// -------------------------------------------------------------------
// V-10: 响应不暴露 EncryptedDEK/IVNonce + 监管员清空 Files + 头像防 SVG/XSS + 生产 CSP 与 RawPrompt 隔离
// -------------------------------------------------------------------
func TestVuln_V10_DataMinimization_AvatarXSS_ProdCSPAndRawPromptHidden(t *testing.T) {
	r := initHighVulnTestEnv(t)
	docLi, _, _, patZhang, supZhao, _ := loadSecuritySeedUsers(t)

	// 1. 病历详情响应不得泄露 encrypted_dek 与 iv_nonce，且监管员视角 files 必须为空
	rec, err := service.DefaultMedicalService.UploadRecord(service.UploadRecordParams{
		DoctorID:  docLi.ID,
		PatientID: patZhang.ID,
		DataType:  "EMR",
		Diagnosis: "V10 数据最小化测试诊断",
		FileName:  "v10_report.txt",
		FileType:  "txt",
		FileData:  []byte("v10 confidential clinical report"),
	})
	if err != nil || rec == nil {
		t.Fatalf("UploadRecord failed: %v", err)
	}
	t.Cleanup(func() {
		repository.DB.Where("record_id = ?", rec.ID).Delete(&model.MedicalFile{})
		repository.DB.Where("id = ?", rec.ID).Delete(&model.MedicalRecord{})
	})

	reqDoc := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/medical-records/%d", rec.ID), nil)
	reqDoc.Header.Set("Authorization", authHeader(t, docLi))
	wDoc := httptest.NewRecorder()
	r.ServeHTTP(wDoc, reqDoc)
	if wDoc.Code != http.StatusOK {
		t.Fatalf("医生查询病历详情失败: status=%d body=%s", wDoc.Code, wDoc.Body.String())
	}
	if strings.Contains(wDoc.Body.String(), `"encrypted_dek"`) || strings.Contains(wDoc.Body.String(), `"iv_nonce"`) {
		t.Fatalf("【V-10 漏洞】病历详情响应向前端泄露了加密密钥包装字段 encrypted_dek/iv_nonce: %s", wDoc.Body.String())
	}

	reqSup := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/medical-records/%d", rec.ID), nil)
	reqSup.Header.Set("Authorization", authHeader(t, supZhao))
	wSup := httptest.NewRecorder()
	r.ServeHTTP(wSup, reqSup)
	var supResp struct {
		Data model.MedicalRecord `json:"data"`
	}
	_ = json.Unmarshal(wSup.Body.Bytes(), &supResp)
	if len(supResp.Data.Files) > 0 {
		t.Fatalf("【V-10 漏洞】监管员查阅病历详情时仍返回了临床附件元数据列表: %+v", supResp.Data.Files)
	}

	// 2. PUT /api/v1/auth/avatar 拦截 SVG XSS 与 javascript: URI
	svgB64 := base64.StdEncoding.EncodeToString([]byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"></svg>`))
	svgPayload, _ := json.Marshal(map[string]string{
		"avatar": "data:image/svg+xml;base64," + svgB64,
	})
	reqSvg := httptest.NewRequest(http.MethodPut, "/api/v1/auth/avatar", bytes.NewReader(svgPayload))
	reqSvg.Header.Set("Authorization", authHeader(t, docLi))
	reqSvg.Header.Set("Content-Type", "application/json")
	wSvg := httptest.NewRecorder()
	r.ServeHTTP(wSvg, reqSvg)
	if wSvg.Code != http.StatusBadRequest {
		t.Fatalf("【V-10 漏洞】UpdateAvatar 允许写入 SVG XSS Data URI: status=%d body=%s", wSvg.Code, wSvg.Body.String())
	}

	// POST /api/v1/auth/avatar/upload 拦截 .svg 文件上传
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("avatar", "evil.svg")
	_, _ = part.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`))
	_ = writer.Close()
	reqSvgUpload := httptest.NewRequest(http.MethodPost, "/api/v1/auth/avatar/upload", body)
	reqSvgUpload.Header.Set("Authorization", authHeader(t, docLi))
	reqSvgUpload.Header.Set("Content-Type", writer.FormDataContentType())
	wSvgUpload := httptest.NewRecorder()
	r.ServeHTTP(wSvgUpload, reqSvgUpload)
	if wSvgUpload.Code != http.StatusBadRequest {
		t.Fatalf("【V-10 漏洞】UploadAvatar 未拦截恶意 SVG 文件: status=%d body=%s", wSvgUpload.Code, wSvgUpload.Body.String())
	}

	// 3. 生产模式下 CSP 默认切为强阻断 (Content-Security-Policy)，且 AI 诊断隐藏 RawPrompt
	origEnv := config.AppConfig.AppEnv
	origCSP := os.Getenv("CSP_ENFORCE")
	_ = os.Unsetenv("CSP_ENFORCE")
	config.AppConfig.AppEnv = "production"
	t.Cleanup(func() {
		config.AppConfig.AppEnv = origEnv
		if origCSP != "" {
			_ = os.Setenv("CSP_ENFORCE", origCSP)
		}
	})

	secRouter := gin.New()
	secRouter.Use(middleware.SecurityHeadersMiddleware())
	secRouter.GET("/ping", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	wCSP := httptest.NewRecorder()
	secRouter.ServeHTTP(wCSP, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if wCSP.Header().Get("Content-Security-Policy") == "" {
		t.Fatalf("【V-10 漏洞】生产模式默认未启用强阻断 Content-Security-Policy")
	}

	aiRes, err := service.GetAIDiagnosticService().AnalyzePatientDiagnosticCorrelation(docLi.ID, model.DiagnosticAssistRequest{
		PatientID:       patZhang.ID,
		CurrentSymptoms: "复查胸闷",
	})
	if err != nil {
		t.Fatalf("AnalyzePatientDiagnosticCorrelation failed: %v", err)
	}
	if aiRes.RawPrompt != "" {
		t.Fatalf("【V-10 漏洞】生产模式 AI 辅助诊断仍向前端暴露内部 RawPrompt")
	}
}
