package controller

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
	"medtrust-backend/model"
	"medtrust-backend/repository"
	"medtrust-backend/service"
	"strings"
)

func setupSystemTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// 模拟已登录 admin 身份上下文
	authAdmin := func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		c.Set("role", "admin")
		c.Next()
	}

	v1 := r.Group("/api/v1/system", authAdmin)
	{
		v1.GET("/hospitals", DefaultSystemController.ListHospitals)
		v1.POST("/hospitals", DefaultSystemController.CreateHospital)
		v1.PUT("/hospitals/:id", DefaultSystemController.UpdateHospital)
		v1.PUT("/hospitals/:id/status", DefaultSystemController.UpdateHospitalStatus)

		v1.GET("/departments", DefaultSystemController.ListDepartments)
		v1.POST("/departments", DefaultSystemController.CreateDepartment)
		v1.PUT("/departments/:id", DefaultSystemController.UpdateDepartment)
		v1.DELETE("/departments/:id", DefaultSystemController.DeleteDepartment)
	}

	return r
}

func TestHospitalCRUD_Lifecycle(t *testing.T) {
	_, _ = config.LoadConfig("../config/config.yaml")
	_, _ = repository.InitDB()
	r := setupSystemTestRouter()

	nanos := time.Now().UnixNano()
	hospNo := fmt.Sprintf("HOSP_TEST_%d", nanos)

	// 1. 创建医疗机构 - 成功
	createPayload := fmt.Sprintf(`{"hospital_no":"%s","name":"测试三甲医院_%d","level":"三级甲等","address":"广州市天河区科技路1号"}`, hospNo, nanos)
	req1 := httptest.NewRequest("POST", "/api/v1/system/hospitals", bytes.NewBufferString(createPayload))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("创建机构失败, status=%d, body=%s", w1.Code, w1.Body.String())
	}

	var resp1 struct {
		Code int            `json:"code"`
		Data model.Hospital `json:"data"`
	}
	_ = json.Unmarshal(w1.Body.Bytes(), &resp1)
	hospID := resp1.Data.ID
	if hospID == 0 || resp1.Data.Status != 1 {
		t.Fatalf("创建机构返回值异常: %+v", resp1.Data)
	}

	// 2. 查重防冲突 - 相同 hospital_no 预期返回 409 Conflict
	reqDup := httptest.NewRequest("POST", "/api/v1/system/hospitals", bytes.NewBufferString(createPayload))
	reqDup.Header.Set("Content-Type", "application/json")
	wDup := httptest.NewRecorder()
	r.ServeHTTP(wDup, reqDup)

	if wDup.Code != http.StatusConflict {
		t.Fatalf("重复 hospital_no 预期 409, 实际 status=%d, body=%s", wDup.Code, wDup.Body.String())
	}

	// 3. 更新机构信息 - 成功
	updatePayload := fmt.Sprintf(`{"name":"测试三甲医院(已更名)_%d","level":"三甲特等","address":"深圳市福田区金融中心88号"}`, nanos)
	req2 := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/system/hospitals/%d", hospID), bytes.NewBufferString(updatePayload))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("更新机构失败, status=%d, body=%s", w2.Code, w2.Body.String())
	}

	// 4. 停用机构 - 成功
	statusPayload := `{"status":0}`
	req3 := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/system/hospitals/%d/status", hospID), bytes.NewBufferString(statusPayload))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Fatalf("停用机构失败, status=%d, body=%s", w3.Code, w3.Body.String())
	}

	var hospCheck model.Hospital
	repository.DB.First(&hospCheck, hospID)
	if hospCheck.Status != 0 {
		t.Fatalf("机构状态未变为 0: %d", hospCheck.Status)
	}

	// 5. 重新启用机构 - 幂等验证
	statusPayloadActive := `{"status":1}`
	req4 := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/system/hospitals/%d/status", hospID), bytes.NewBufferString(statusPayloadActive))
	req4.Header.Set("Content-Type", "application/json")
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)

	if w4.Code != http.StatusOK {
		t.Fatalf("重新启用机构失败, status=%d, body=%s", w4.Code, w4.Body.String())
	}

	repository.DB.First(&hospCheck, hospID)
	if hospCheck.Status != 1 {
		t.Fatalf("机构状态未恢复为 1: %d", hospCheck.Status)
	}

	// 清理测试机构
	repository.DB.Delete(&model.Hospital{}, hospID)
}

func TestDepartmentCRUD_AndCascadeSafety(t *testing.T) {
	_, _ = config.LoadConfig("../config/config.yaml")
	_, _ = repository.InitDB()
	r := setupSystemTestRouter()

	nanos := time.Now().UnixNano()
	deptNo := fmt.Sprintf("DEPT_%d", nanos)

	// 1. 创建科室 - 成功 (归属 HOSP_A, id=1)
	createPayload := fmt.Sprintf(`{"dept_no":"%s","hospital_id":1,"name":"自动化测试专科_%d","description":"专治疑难杂症与系统自动化回归"}`, deptNo, nanos)
	req1 := httptest.NewRequest("POST", "/api/v1/system/departments", bytes.NewBufferString(createPayload))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("创建科室失败, status=%d, body=%s", w1.Code, w1.Body.String())
	}

	var resp1 struct {
		Code int              `json:"code"`
		Data model.Department `json:"data"`
	}
	_ = json.Unmarshal(w1.Body.Bytes(), &resp1)
	deptID := resp1.Data.ID
	if deptID == 0 {
		t.Fatalf("创建科室返回 ID 为空: %+v", resp1.Data)
	}

	// 2. 同院同编码查重 - 预期 409 Conflict
	reqDup := httptest.NewRequest("POST", "/api/v1/system/departments", bytes.NewBufferString(createPayload))
	reqDup.Header.Set("Content-Type", "application/json")
	wDup := httptest.NewRecorder()
	r.ServeHTTP(wDup, reqDup)

	if wDup.Code != http.StatusConflict {
		t.Fatalf("同院重复科室代码预期 409, 实际 status=%d, body=%s", wDup.Code, wDup.Body.String())
	}

	// 3. 更新科室信息 - 成功
	updatePayload := fmt.Sprintf(`{"name":"自动化测试专科(已更名)_%d","description":"国家临床重点专科建设项目"}`, nanos)
	req2 := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/system/departments/%d", deptID), bytes.NewBufferString(updatePayload))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("更新科室失败, status=%d, body=%s", w2.Code, w2.Body.String())
	}

	// 4. 安全级联删除拦截 - 模拟有关联医生时禁止删除 (预期 409 Conflict)
	mockUser := model.User{
		UserNo:       fmt.Sprintf("USER_DOC_%d", nanos),
		Username:     fmt.Sprintf("doc_test_%d", nanos),
		RealName:     "测试关联医生",
		Role:         "doctor",
		HospitalID:   1,
		DepartmentID: deptID,
		Status:       "NORMAL",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	repository.DB.Create(&mockUser)

	reqDel1 := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/system/departments/%d", deptID), nil)
	wDel1 := httptest.NewRecorder()
	r.ServeHTTP(wDel1, reqDel1)

	if wDel1.Code != http.StatusConflict {
		t.Fatalf("有关联医生时删除科室预期 409, 实际 status=%d, body=%s", wDel1.Code, wDel1.Body.String())
	}

	// 解除关联并清理医生
	repository.DB.Delete(&mockUser)

	// 5. 模拟产生历史病历时禁止删除 (预期 409 Conflict)
	mockRecord := model.MedicalRecord{
		RecordNo:     fmt.Sprintf("REC_TEST_%d", nanos),
		PatientID:    1,
		DoctorID:     1,
		HospitalID:   1,
		DepartmentID: deptID,
		DataType:     "EMR",
		Title:        "测试关联病历",
		CreatedAt:    time.Now(),
	}
	repository.DB.Create(&mockRecord)

	reqDel2 := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/system/departments/%d", deptID), nil)
	wDel2 := httptest.NewRecorder()
	r.ServeHTTP(wDel2, reqDel2)

	if wDel2.Code != http.StatusConflict {
		t.Fatalf("有历史病历时删除科室预期 409, 实际 status=%d, body=%s", wDel2.Code, wDel2.Body.String())
	}

	// 清理病历
	repository.DB.Delete(&mockRecord)

	// 6. 无关联情况下执行删除 - 成功 (200 OK)
	reqDel3 := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/system/departments/%d", deptID), nil)
	wDel3 := httptest.NewRecorder()
	r.ServeHTTP(wDel3, reqDel3)

	if wDel3.Code != http.StatusOK {
		t.Fatalf("无关联时删除科室失败, status=%d, body=%s", wDel3.Code, wDel3.Body.String())
	}

	var deptCheck model.Department
	err := repository.DB.First(&deptCheck, deptID).Error
	if err == nil {
		t.Fatalf("科室删除后仍在数据库中存在")
	}
}

func TestHospitalDisabled_DecisionD1_AccessEngineRejection(t *testing.T) {
	cfg, _ := config.LoadConfig("../config/config.yaml")
	_, _ = repository.InitDB()
	low := 30
	high := 60
	if cfg != nil && cfg.Risk.LowThreshold > 0 {
		low = cfg.Risk.LowThreshold
		high = cfg.Risk.HighThreshold
	}
	service.InitAccessEngine(low, high)
	r := setupSystemTestRouter()

	nanos := time.Now().UnixNano()
	hospNo := fmt.Sprintf("HOSP_D1_%d", nanos)

	// 1. 创建机构 (默认 status=1)
	createPayload := fmt.Sprintf(`{"hospital_no":"%s","name":"停用拦截测试医院_%d","level":"三甲","address":"测试地址"}`, hospNo, nanos)
	req1 := httptest.NewRequest("POST", "/api/v1/system/hospitals", bytes.NewBufferString(createPayload))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("创建机构失败: %d, %s", w1.Code, w1.Body.String())
	}
	var resp1 struct {
		Data model.Hospital `json:"data"`
	}
	_ = json.Unmarshal(w1.Body.Bytes(), &resp1)
	hospID := resp1.Data.ID

	// 2. 创建归属于该机构的医生与病历
	doc := model.User{
		UserNo:     fmt.Sprintf("DOC_D1_%d", nanos),
		Username:   fmt.Sprintf("doc_d1_%d", nanos),
		Role:       "doctor",
		HospitalID: hospID,
		Status:     "NORMAL",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	repository.DB.Create(&doc)
	defer repository.DB.Delete(&doc)

	rec := model.MedicalRecord{
		RecordNo:   fmt.Sprintf("REC_D1_%d", nanos),
		PatientID:  1,
		DoctorID:   doc.ID,
		HospitalID: hospID,
		DataType:   "EMR",
		Title:      "机构停用测试病历",
		CreatedAt:  time.Now(),
	}
	repository.DB.Create(&rec)
	defer repository.DB.Delete(&rec)
	defer repository.DB.Delete(&model.Hospital{}, hospID)

	// 3. 初始状态下(status=1)，医生调阅本人开具病历放行
	dec1, err := service.DefaultAccessEngine.EvaluateAccess(doc.ID, 1, rec.ID, false)
	if err != nil || !dec1.Allowed {
		t.Fatalf("机构正常时医生调阅本人病历预期放行, 实际: allowed=%v, dec=%+v", dec1.Allowed, dec1)
	}

	// 4. 管理员将该机构停用 (status=0)
	statusPayload := `{"status":0}`
	reqStatus := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/system/hospitals/%d/status", hospID), bytes.NewBufferString(statusPayload))
	reqStatus.Header.Set("Content-Type", "application/json")
	wStatus := httptest.NewRecorder()
	r.ServeHTTP(wStatus, reqStatus)
	if wStatus.Code != http.StatusOK {
		t.Fatalf("停用机构请求失败: %d", wStatus.Code)
	}

	// 5. 再次调阅，断言访问控制引擎（AccessEngine）直接拦截且原因包含“机构已停用”
	dec2, _ := service.DefaultAccessEngine.EvaluateAccess(doc.ID, 1, rec.ID, false)
	if dec2.Allowed || dec2.Decision != "REJECTED" || !strings.Contains(dec2.Reason, "机构已停用") {
		t.Fatalf("机构停用后调阅病历预期拦截(REJECTED), 实际: allowed=%v, decision=%s, reason=%s", dec2.Allowed, dec2.Decision, dec2.Reason)
	}
	t.Logf("✅ 决策 D1 闭环验证通过: 机构停用 -> AccessEngine ABAC 环境属性拦截成功 (Reason: %s)", dec2.Reason)
}
