import os

base = r'e:\EnglishEncoding\competition\last\MedTrust\backend'

def write_file(rel_path, content):
    full_path = os.path.join(base, rel_path)
    os.makedirs(os.path.dirname(full_path), exist_ok=True)
    with open(full_path, 'w', encoding='utf-8') as out:
        out.write(content.strip() + '\n')
    print('Wrote:', rel_path)

# 1. controller/auth_controller.go
write_file('controller/auth_controller.go', """package controller

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"medtrust-backend/model"
	"medtrust-backend/service"
)

type AuthController struct{}

var DefaultAuthController = &AuthController{}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (ctrl *AuthController) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误，请输入用户名与密码"})
		return
	}

	token, user, err := service.DefaultAuthService.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.Response{Code: 401, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "登录成功",
		Data: gin.H{
			"token": token,
			"user":  user,
		},
	})
}

func (ctrl *AuthController) Profile(c *gin.Context) {
	userID := c.GetUint64("user_id")
	user, err := service.DefaultAuthService.GetProfile(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "获取成功", Data: user})
}
""")

# 2. controller/medical_controller.go
write_file('controller/medical_controller.go', """package controller

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"medtrust-backend/model"
	"medtrust-backend/repository"
	"medtrust-backend/service"
)

type MedicalController struct{}

var DefaultMedicalController = &MedicalController{}

func (ctrl *MedicalController) Upload(c *gin.Context) {
	doctorID := c.GetUint64("user_id")
	patientIDStr := c.PostForm("patient_id")
	dataType := c.PostForm("data_type")
	diagnosis := c.PostForm("diagnosis")

	patientID, _ := strconv.ParseUint(patientIDStr, 10, 64)
	if patientID == 0 || dataType == "" {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "缺少必要参数：患者ID与数据类型"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "请上传病历附件或医学影像切片"})
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "读取文件失败"})
		return
	}

	fileType := "pdf"
	if len(header.Filename) > 4 {
		fileType = header.Filename[len(header.Filename)-3:]
	}

	rec, err := service.DefaultMedicalService.UploadRecord(doctorID, patientID, dataType, diagnosis, header.Filename, fileType, fileBytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "医疗数据加密存储并上链存证成功",
		Data:    rec,
	})
}

func (ctrl *MedicalController) List(c *gin.Context) {
	patientID, _ := strconv.ParseUint(c.Query("patient_id"), 10, 64)
	doctorID, _ := strconv.ParseUint(c.Query("doctor_id"), 10, 64)
	hospitalID, _ := strconv.ParseUint(c.Query("hospital_id"), 10, 64)
	keyword := c.Query("keyword")

	// 患者角色限制只能查自己的
	role := c.GetString("role")
	if role == "patient" {
		patientID = c.GetUint64("user_id")
	}

	list, err := service.DefaultMedicalService.GetRecords(patientID, doctorID, hospitalID, keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "查询成功", Data: list})
}

func (ctrl *MedicalController) GetByID(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	rec, err := service.DefaultMedicalService.GetRecordByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "记录不存在"})
		return
	}
	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "查询成功", Data: rec})
}

func (ctrl *MedicalController) Download(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var rec model.MedicalRecord
	if err := repository.DB.Preload("Files").First(&rec, id).Error; err != nil || len(rec.Files) == 0 {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "文件不存在"})
		return
	}

	file := rec.Files[0]
	c.Header("Content-Disposition", "attachment; filename="+file.FileName)
	c.Header("Content-Type", "application/octet-stream")
	c.String(http.StatusOK, "Decrypted Medical Record Data Placeholder for "+file.FileName)
}
""")

# 3. controller/access_controller.go
write_file('controller/access_controller.go', """package controller

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
	"medtrust-backend/service"
)

type AccessController struct{}

var DefaultAccessController = &AccessController{}

type AccessRequestDTO struct {
	PatientID uint64 `json:"patient_id" binding:"required"`
	RecordID  uint64 `json:"record_id" binding:"required"`
	Purpose   string `json:"purpose"`
}

// RequestAccess 医生发起常规访问申请
func (ctrl *AccessController) RequestAccess(c *gin.Context) {
	doctorID := c.GetUint64("user_id")
	var req AccessRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误"})
		return
	}

	decision, err := service.DefaultAccessEngine.EvaluateAccess(doctorID, req.PatientID, req.RecordID, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: err.Error()})
		return
	}

	// 记录访问申请与系统决策
	var doctor model.User
	_ = repository.DB.First(&doctor, doctorID)
	var record model.MedicalRecord
	_ = repository.DB.First(&record, req.RecordID)

	randBytes := make([]byte, 3)
	_, _ = rand.Read(randBytes)
	reqNo := fmt.Sprintf("REQ%s%s", time.Now().Format("20060102"), hex.EncodeToString(randBytes))

	accessRec := model.AccessRequest{
		RequestNo:        reqNo,
		DoctorID:         doctorID,
		PatientID:        req.PatientID,
		RecordID:         req.RecordID,
		SourceHospitalID: doctor.HospitalID,
		TargetHospitalID: record.HospitalID,
		Purpose:          req.Purpose,
		RiskScore:        decision.RiskEvaluation.TotalScore,
		RiskLevel:        decision.RiskEvaluation.Level,
		Decision:         decision.Decision,
		CreatedAt:        time.Now(),
	}
	repository.DB.Create(&accessRec)

	service.DefaultAuditService.Log(doctorID, "ACCESS", "RECORD", record.RecordNo, doctor.HospitalID, func() string {
		if decision.Allowed {
			return "SUCCESS"
		}
		return "INTERCEPTED"
	}(), decision.RiskEvaluation.Level, "127.0.0.1")

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "权限评估完成",
		Data:    decision,
	})
}

type BreakGlassRequest struct {
	RecordID        uint64 `json:"record_id" binding:"required"`
	EmergencyReason string `json:"emergency_reason" binding:"required"` // RESCUE, COMA, CRITICAL, OTHER
	Description     string `json:"description" binding:"required"`
	DoctorConfirmed bool   `json:"doctor_confirmed" binding:"required"`
}

// BreakGlass 医生发起紧急抢救调阅
func (ctrl *AccessController) BreakGlass(c *gin.Context) {
	doctorID := c.GetUint64("user_id")
	var req BreakGlassRequest
	if err := c.ShouldBindJSON(&req); err != nil || !req.DoctorConfirmed {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "必须如实填写抢救说明并勾选临床法律责任免责声明"})
		return
	}

	var rec model.MedicalRecord
	if err := repository.DB.First(&rec, req.RecordID).Error; err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "病历不存在"})
		return
	}

	// 统一权限网关的前置核验 (医生身份、是否被处罚 RESTRICTED)
	decision, _ := service.DefaultAccessEngine.EvaluateAccess(doctorID, rec.PatientID, req.RecordID, true)
	if !decision.Allowed {
		c.JSON(http.StatusForbidden, model.Response{Code: 403, Message: decision.Reason, Data: decision})
		return
	}

	// 生成事件单并上链
	event, err := service.DefaultEmergencyService.SubmitEmergencyAccess(doctorID, req.RecordID, req.EmergencyReason, req.Description, req.DoctorConfirmed)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: err.Error()})
		return
	}

	// 解密放行病历数据
	fullRecord, _ := service.DefaultMedicalService.GetRecordByID(req.RecordID)

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "紧急访问申请已核准放行，生成待审核事件单并全量存证",
		Data: gin.H{
			"event":  event,
			"record": fullRecord,
		},
	})
}

type PatientFeedbackRequest struct {
	EventNo  string `json:"event_no" binding:"required"`
	Feedback string `json:"feedback" binding:"required"` // CONFIRMED, OBJECTED
	Comment  string `json:"comment"`
}

func (ctrl *AccessController) PatientFeedback(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	var req PatientFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误"})
		return
	}

	if err := service.DefaultEmergencyService.SubmitPatientFeedback(patientID, req.EventNo, req.Feedback, req.Comment); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "反馈已成功提交并归档"})
}

type CreateAuthRequest struct {
	AuthTargetType string    `json:"auth_target_type" binding:"required"` // DOCTOR, HOSPITAL
	AuthTargetID   uint64    `json:"auth_target_id" binding:"required"`
	ScopeType      string    `json:"scope_type" binding:"required"`       // ALL, SINGLE
	RecordID       uint64    `json:"record_id"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
}

func (ctrl *AccessController) CreateAuthorization(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	var req CreateAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误"})
		return
	}

	if req.StartTime.IsZero() {
		req.StartTime = time.Now()
	}
	if req.EndTime.IsZero() {
		req.EndTime = time.Now().Add(365 * 24 * time.Hour)
	}

	randBytes := make([]byte, 3)
	_, _ = rand.Read(randBytes)
	authNo := fmt.Sprintf("AUTH%s%s", time.Now().Format("20060102"), hex.EncodeToString(randBytes))

	txID, _, _ := blockchain.DefaultLedger.CommitAsset("AUTHORIZATION", authNo, map[string]interface{}{
		"auth_no":     authNo,
		"patient_id":  patientID,
		"target_type": req.AuthTargetType,
		"target_id":   req.AuthTargetID,
		"scope":       req.ScopeType,
		"status":      "ACTIVE",
		"timestamp":   time.Now().Format(time.RFC3339),
	})

	auth := model.Authorization{
		AuthNo:         authNo,
		PatientID:      patientID,
		AuthTargetType: req.AuthTargetType,
		AuthTargetID:   req.AuthTargetID,
		ScopeType:      req.ScopeType,
		RecordID:       req.RecordID,
		StartTime:      req.StartTime,
		EndTime:        req.EndTime,
		Status:         "ACTIVE",
		FabricTxID:     txID,
		CreatedAt:      time.Now(),
	}
	repository.DB.Create(&auth)

	service.DefaultAuditService.Log(patientID, "AUTHORIZE", "AUTH", authNo, 0, "SUCCESS", "LOW", "127.0.0.1")
	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "授权策略建立并上链成功", Data: auth})
}

func (ctrl *AccessController) ListAuthorizations(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	role := c.GetString("role")

	var list []model.Authorization
	query := repository.DB.Model(&model.Authorization{})
	if role == "patient" {
		query = query.Where("patient_id = ?", patientID)
	}
	query.Order("id desc").Find(&list)

	for i := range list {
		if list[i].AuthTargetType == "DOCTOR" {
			var doc model.User
			if err := repository.DB.First(&doc, list[i].AuthTargetID).Error; err == nil {
				list[i].TargetName = doc.RealName + " (医生)"
			}
		} else if list[i].AuthTargetType == "HOSPITAL" {
			var hosp model.Hospital
			if err := repository.DB.First(&hosp, list[i].AuthTargetID).Error; err == nil {
				list[i].TargetName = hosp.Name
			}
		}
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "查询成功", Data: list})
}

func (ctrl *AccessController) RevokeAuthorization(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var auth model.Authorization
	if err := repository.DB.Where("id = ? AND patient_id = ?", id, patientID).First(&auth).Error; err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "未找到该授权条目"})
		return
	}

	auth.Status = "REVOKED"
	repository.DB.Save(&auth)

	_, _, _ = blockchain.DefaultLedger.CommitAsset("REVOKE_AUTH", auth.AuthNo, map[string]interface{}{
		"auth_no":   auth.AuthNo,
		"status":    "REVOKED",
		"timestamp": time.Now().Format(time.RFC3339),
	})

	service.DefaultAuditService.Log(patientID, "REVOKE", "AUTH", auth.AuthNo, 0, "SUCCESS", "LOW", "127.0.0.1")
	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "授权已撤回上链"})
}
""")

# 4. controller/supervisor_controller.go
write_file('controller/supervisor_controller.go', """package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
	"medtrust-backend/service"
)

type SupervisorController struct{}

var DefaultSupervisorController = &SupervisorController{}

func (ctrl *SupervisorController) Overview(c *gin.Context) {
	var totalRecords int64
	repository.DB.Model(&model.MedicalRecord{}).Count(&totalRecords)

	var totalEmergencies int64
	repository.DB.Model(&model.EmergencyAccessEvent{}).Count(&totalEmergencies)

	var pendingAudits int64
	repository.DB.Model(&model.EmergencyAccessEvent{}).Where("audit_status = 'PENDING_AUDIT'").Count(&pendingAudits)

	var totalLogs int64
	repository.DB.Model(&model.AuditLog{}).Count(&totalLogs)

	totalTx, blockHeight := blockchain.DefaultLedger.GetStats()

	// 统计不同机构病历占比
	type HospStat struct {
		Name  string `json:"name"`
		Count int64  `json:"count"`
	}
	var hospStats []HospStat
	repository.DB.Raw(`
		SELECT h.name, COUNT(r.id) as count 
		FROM hospitals h 
		LEFT JOIN medical_records r ON h.id = r.hospital_id 
		GROUP BY h.id, h.name
	`).Scan(&hospStats)

	// 统计风险等级分布
	type RiskStat struct {
		RiskLevel string `json:"risk_level"`
		Count     int64  `json:"count"`
	}
	var riskStats []RiskStat
	repository.DB.Raw(`SELECT risk_level, COUNT(id) as count FROM audit_logs GROUP BY risk_level`).Scan(&riskStats)

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "查询成功",
		Data: gin.H{
			"total_records":     totalRecords,
			"total_emergencies": totalEmergencies,
			"pending_audits":    pendingAudits,
			"total_logs":        totalLogs,
			"total_tx":          totalTx,
			"block_height":      blockHeight,
			"hosp_stats":        hospStats,
			"risk_stats":        riskStats,
		},
	})
}

func (ctrl *SupervisorController) ListEmergencyEvents(c *gin.Context) {
	status := c.Query("status")
	query := repository.DB.Model(&model.EmergencyAccessEvent{})
	if status != "" {
		query = query.Where("audit_status = ?", status)
	}

	var list []model.EmergencyAccessEvent
	query.Order("id desc").Find(&list)

	for i := range list {
		var doc model.User
		if err := repository.DB.First(&doc, list[i].DoctorID).Error; err == nil {
			list[i].DoctorName = doc.RealName
		}
		var pat model.User
		if err := repository.DB.First(&pat, list[i].PatientID).Error; err == nil {
			list[i].PatientName = pat.RealName
		}
		var rec model.MedicalRecord
		if err := repository.DB.First(&rec, list[i].RecordID).Error; err == nil {
			list[i].RecordNo = rec.RecordNo
		}
		var h1, h2 model.Hospital
		if err := repository.DB.First(&h1, list[i].SourceHospitalID).Error; err == nil {
			list[i].SourceHospitalName = h1.Name
		}
		if err := repository.DB.First(&h2, list[i].TargetHospitalID).Error; err == nil {
			list[i].TargetHospitalName = h2.Name
		}
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "查询成功", Data: list})
}

type AuditEventRequest struct {
	AuditStatus  string `json:"audit_status" binding:"required"` // CLOSED_APPROVED, CLOSED_VIOLATION, UNDER_INVESTIGATION
	AuditComment string `json:"audit_comment"`
	Punishment   string `json:"punishment"` // NORMAL, RESTRICTED, DISABLED
}

func (ctrl *SupervisorController) AuditEmergencyEvent(c *gin.Context) {
	supervisorID := c.GetUint64("user_id")
	eventNo := c.Param("event_no")

	var req AuditEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误"})
		return
	}

	if err := service.DefaultEmergencyService.AuditEvent(supervisorID, eventNo, req.AuditStatus, req.AuditComment, req.Punishment); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "审核裁决与处置已固化生效"})
}

func (ctrl *SupervisorController) ListAuditLogs(c *gin.Context) {
	opType := c.Query("operation_type")
	riskLevel := c.Query("risk_level")

	query := repository.DB.Model(&model.AuditLog{})
	if opType != "" {
		query = query.Where("operation_type = ?", opType)
	}
	if riskLevel != "" {
		query = query.Where("risk_level = ?", riskLevel)
	}

	var list []model.AuditLog
	query.Order("id desc").Limit(100).Find(&list)

	for i := range list {
		var u model.User
		if err := repository.DB.First(&u, list[i].UserID).Error; err == nil {
			list[i].UserName = u.RealName
		}
		var h model.Hospital
		if err := repository.DB.First(&h, list[i].HospitalID).Error; err == nil {
			list[i].HospitalName = h.Name
		}
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "查询成功", Data: list})
}

func (ctrl *SupervisorController) VerifyRecord(c *gin.Context) {
	recordID, _ := strconv.ParseUint(c.Param("record_id"), 10, 64)
	res, err := service.DefaultVerificationService.Verify(recordID)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "防篡改核验完成", Data: res})
}
""")

# 5. controller/system_controller.go
write_file('controller/system_controller.go', """package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"medtrust-backend/model"
	"medtrust-backend/repository"
	"medtrust-backend/service"
)

type SystemController struct{}

var DefaultSystemController = &SystemController{}

func (ctrl *SystemController) ListUsers(c *gin.Context) {
	var users []model.User
	repository.DB.Order("id asc").Find(&users)

	for i := range users {
		if users[i].HospitalID > 0 {
			var hosp model.Hospital
			if err := repository.DB.First(&hosp, users[i].HospitalID).Error; err == nil {
				users[i].HospitalName = hosp.Name
			}
		}
		if users[i].DepartmentID > 0 {
			var dept model.Department
			if err := repository.DB.First(&dept, users[i].DepartmentID).Error; err == nil {
				users[i].DepartmentName = dept.Name
			}
		}
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "查询成功", Data: users})
}

type UpdateStatusReq struct {
	Status string `json:"status" binding:"required"` // NORMAL, RESTRICTED, DISABLED
}

func (ctrl *SystemController) UpdateUserStatus(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req UpdateStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误"})
		return
	}

	var u model.User
	if err := repository.DB.First(&u, id).Error; err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "用户不存在"})
		return
	}

	u.Status = req.Status
	repository.DB.Save(&u)

	service.DefaultAuditService.Log(c.GetUint64("user_id"), "UPDATE_USER", "USER", u.UserNo, u.HospitalID, "SUCCESS", "LOW", "127.0.0.1")
	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "账号状态变更成功"})
}

func (ctrl *SystemController) CreateUser(c *gin.Context) {
	var u model.User
	if err := c.ShouldBindJSON(&u); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误"})
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	u.PasswordHash = string(hash)
	if u.Status == "" {
		u.Status = "NORMAL"
	}

	if err := repository.DB.Create(&u).Error; err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "创建用户失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "用户创建成功", Data: u})
}

func (ctrl *SystemController) ListHospitals(c *gin.Context) {
	var list []model.Hospital
	repository.DB.Find(&list)
	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "查询成功", Data: list})
}

func (ctrl *SystemController) ListDepartments(c *gin.Context) {
	hospID, _ := strconv.ParseUint(c.Query("hospital_id"), 10, 64)
	var list []model.Department
	query := repository.DB.Model(&model.Department{})
	if hospID > 0 {
		query = query.Where("hospital_id = ?", hospID)
	}
	query.Find(&list)
	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "查询成功", Data: list})
}
""")

# 6. router/router.go
write_file('router/router.go', """package router

import (
	\"github.com/gin-gonic/gin\"
	\"medtrust-backend/controller\"
	\"medtrust-backend/middleware\"
	\"medtrust-backend/model\"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORSMiddleware())

	r.GET(\"/api/v1/health\", func(c *gin.Context) {
		c.JSON(200, model.Response{Code: 200, Message: \"MedTrust Blockchain Backend Gateway is Running\", Data: nil})
	})

	v1 := r.Group(\"/api/v1\")
	{
		// 认证模块 (公开)
		v1.POST(\"/auth/login\", controller.DefaultAuthController.Login)

		// 需登录鉴权保护路由
		authGroup := v1.Group(\"\")
		authGroup.Use(middleware.JWTAuthMiddleware())
		{
			authGroup.GET(\"/auth/profile\", controller.DefaultAuthController.Profile)

			// 医疗记录模块
			authGroup.POST(\"/medical-records/upload\", middleware.RequireRoles(\"doctor\"), controller.DefaultMedicalController.Upload)
			authGroup.GET(\"/medical-records\", controller.DefaultMedicalController.List)
			authGroup.GET(\"/medical-records/:id\", controller.DefaultMedicalController.GetByID)
			authGroup.GET(\"/medical-records/:id/download\", controller.DefaultMedicalController.Download)

			// 跨院与访问控制模块
			authGroup.POST(\"/access/requests\", middleware.RequireRoles(\"doctor\"), controller.DefaultAccessController.RequestAccess)
			authGroup.POST(\"/access/break-glass\", middleware.RequireRoles(\"doctor\"), controller.DefaultAccessController.BreakGlass)
			authGroup.POST(\"/access/break-glass/patient-feedback\", middleware.RequireRoles(\"patient\"), controller.DefaultAccessController.PatientFeedback)

			// 患者自主授权模块
			authGroup.POST(\"/authorizations\", middleware.RequireRoles(\"patient\"), controller.DefaultAccessController.CreateAuthorization)
			authGroup.GET(\"/authorizations\", controller.DefaultAccessController.ListAuthorizations)
			authGroup.DELETE(\"/authorizations/:id\", middleware.RequireRoles(\"patient\"), controller.DefaultAccessController.RevokeAuthorization)

			// 监管看板模块
			authGroup.GET(\"/supervisor/overview\", middleware.RequireRoles(\"supervisor\", \"admin\"), controller.DefaultSupervisorController.Overview)
			authGroup.GET(\"/supervisor/emergency-events\", middleware.RequireRoles(\"supervisor\", \"admin\", \"patient\"), controller.DefaultSupervisorController.ListEmergencyEvents)
			authGroup.POST(\"/supervisor/emergency-events/:event_no/audit\", middleware.RequireRoles(\"supervisor\"), controller.DefaultSupervisorController.AuditEmergencyEvent)
			authGroup.GET(\"/audit-logs\", middleware.RequireRoles(\"supervisor\", \"admin\"), controller.DefaultSupervisorController.ListAuditLogs)
			authGroup.POST(\"/verification/:record_id\", middleware.RequireRoles(\"supervisor\", \"admin\", \"doctor\"), controller.DefaultSupervisorController.VerifyRecord)

			// 系统管理员模块
			authGroup.GET(\"/system/users\", middleware.RequireRoles(\"admin\", \"supervisor\"), controller.DefaultSystemController.ListUsers)
			authGroup.PUT(\"/system/users/:id/status\", middleware.RequireRoles(\"admin\"), controller.DefaultSystemController.UpdateUserStatus)
			authGroup.POST(\"/system/users\", middleware.RequireRoles(\"admin\"), controller.DefaultSystemController.CreateUser)
			authGroup.GET(\"/system/hospitals\", controller.DefaultSystemController.ListHospitals)
			authGroup.GET(\"/system/departments\", controller.DefaultSystemController.ListDepartments)
		}
	}

	return r
}
""")

# 7. cmd/main.go
write_file('cmd/main.go', """package main

import (
	\"fmt\"
	\"log\"

	\"medtrust-backend/config\"
	\"medtrust-backend/pkg/blockchain\"
	\"medtrust-backend/repository\"
	\"medtrust-backend/router\"
	\"medtrust-backend/service\"
)

func main() {
	cfg, err := config.LoadConfig(\"./config/config.yaml\")
	if err != nil {
		log.Fatalf(\"Failed to load config: %v\", err)
	}

	// 初始化数据库
	_, err = repository.InitDB()
	if err != nil {
		log.Fatalf(\"Failed to init DB: %v\", err)
	}
	fmt.Println(\"[MedTrust] MySQL Database Connected & AutoMigrated Successfully\")

	// 初始化区块链与 IPFS 服务
	blockchain.InitLedger(cfg.Blockchain.LedgerDir)
	service.InitMedicalService(cfg.IPFS.APIURL, cfg.IPFS.StorageDir)
	service.InitAccessEngine(cfg.Risk.LowThreshold, cfg.Risk.HighThreshold)
	fmt.Println(\"[MedTrust] Fabric Ledger & IPFS Engine Initialized\")

	r := router.SetupRouter()
	addr := fmt.Sprintf(\":%d\", cfg.Server.Port)
	fmt.Printf(\"[MedTrust] Server listening on http://127.0.0.1%s\\n\", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf(\"Failed to run server: %v\", err)
	}
}
""")

print("Part 3 files created successfully!")
