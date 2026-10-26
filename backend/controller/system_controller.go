package controller

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
	"medtrust-backend/service"
)

type SystemController struct{}

var DefaultSystemController = &SystemController{}

func (ctrl *SystemController) ListUsers(c *gin.Context) {
	var users []model.User
	repository.DB.Order("id asc").Find(&users)

	for i := range users {
		users[i].HasMedicalKey = (users[i].MedicalKeyHash != "" || users[i].MedicalKey != "")
		users[i].PasswordHash = ""
		users[i].MedicalKey = ""
		users[i].MedicalKeyHash = ""

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
	paramID := c.Param("id")
	id, _ := strconv.ParseUint(paramID, 10, 64)
	var req UpdateStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误: 缺少或无效的 status 字段"})
		return
	}

	status := strings.ToUpper(strings.TrimSpace(req.Status))
	if status != "NORMAL" && status != "RESTRICTED" && status != "DISABLED" {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "状态值无效，仅支持 NORMAL, RESTRICTED, DISABLED"})
		return
	}

	var u model.User
	var err error
	if id > 0 {
		err = repository.DB.First(&u, id).Error
	} else {
		err = repository.DB.Where("user_no = ? OR username = ?", paramID, paramID).First(&u).Error
	}
	if err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "用户不存在"})
		return
	}

	u.Status = status
	repository.DB.Model(&model.User{}).Where("id = ?", u.ID).Update("status", status)
	repository.InvalidateUserStatus(u.ID)

	if service.DefaultAuditService != nil {
		service.DefaultAuditService.Log(c.GetUint64("user_id"), "UPDATE_USER_STATUS", "USER", u.UserNo, u.HospitalID, "SUCCESS", "LOW", c.ClientIP())
	}
	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: fmt.Sprintf("用户 %s 账号状态已成功变更为 %s", u.Username, status),
		Data: gin.H{
			"id":       u.ID,
			"user_no":  u.UserNo,
			"username": u.Username,
			"status":   status,
		},
	})
}

type CreateUserDTO struct {
	Username     string `json:"username" binding:"required"`
	Password     string `json:"password"`
	UserNo       string `json:"user_no"`
	RealName     string `json:"real_name" binding:"required"`
	Role         string `json:"role" binding:"required"`
	HospitalID   uint64 `json:"hospital_id"`
	DepartmentID uint64 `json:"department_id"`
	Title        string `json:"title"`
	Phone        string `json:"phone"`
	IDCard       string `json:"id_card"`
	Status       string `json:"status"`
}

func (ctrl *SystemController) CreateUser(c *gin.Context) {
	var dto CreateUserDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误，请填写用户名、姓名和角色"})
		return
	}

	pwd := strings.TrimSpace(dto.Password)
	isRandomTemp := false
	if pwd == "" {
		pwd = service.GenerateSecureRandomPassword(10)
		isRandomTemp = true
	} else if len(pwd) < 6 {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "密码长度至少需要 6 个字符"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "密码哈希生成失败"})
		return
	}

	userNo := dto.UserNo
	if userNo == "" {
		randBytes := make([]byte, 3)
		_, _ = rand.Read(randBytes)
		prefix := "USR"
		if dto.Role == "doctor" {
			prefix = "DOC"
		} else if dto.Role == "patient" {
			prefix = "PAT"
		} else if dto.Role == "supervisor" {
			prefix = "SUP"
		} else if dto.Role == "admin" {
			prefix = "ADM"
		}
		userNo = fmt.Sprintf("%s_%s%s", prefix, time.Now().Format("200601"), hex.EncodeToString(randBytes))
	}

	status := dto.Status
	if status == "" {
		status = "NORMAL"
	}

	u := model.User{
		UserNo:        userNo,
		Username:      dto.Username,
		PasswordHash:  string(hash),
		RealName:      dto.RealName,
		Role:          dto.Role,
		HospitalID:    dto.HospitalID,
		DepartmentID:  dto.DepartmentID,
		Title:         dto.Title,
		Phone:         dto.Phone,
		IDCard:        dto.IDCard,
		MustChangePwd: isRandomTemp,
		Status:        status,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := repository.DB.Create(&u).Error; err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "创建用户失败: " + err.Error()})
		return
	}

	if service.DefaultAuditService != nil {
		service.DefaultAuditService.Log(c.GetUint64("user_id"), "CREATE_USER", "USER", u.UserNo, u.HospitalID, "SUCCESS", "LOW", c.ClientIP())
	}

	u.PasswordHash = ""
	u.MedicalKey = ""
	u.MedicalKeyHash = ""

	respData := gin.H{
		"user": u,
	}
	if isRandomTemp {
		respData["temp_password"] = pwd
		respData["notice"] = "已为新用户生成高强度临时随机初始密码，用户首次登录时必须修改密码"
	}
	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "用户创建成功",
		Data:    respData,
	})
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

func (ctrl *SystemController) ListDoctors(c *gin.Context) {
	hospID, _ := strconv.ParseUint(c.Query("hospital_id"), 10, 64)
	var doctors []model.User
	query := repository.DB.Where("role = 'doctor'")
	if hospID > 0 {
		query = query.Where("hospital_id = ?", hospID)
	}
	query.Order("id asc").Find(&doctors)

	for i := range doctors {
		if doctors[i].HospitalID > 0 {
			var hosp model.Hospital
			if err := repository.DB.First(&hosp, doctors[i].HospitalID).Error; err == nil {
				doctors[i].HospitalName = hosp.Name
			}
		}
		if doctors[i].DepartmentID > 0 {
			var dept model.Department
			if err := repository.DB.First(&dept, doctors[i].DepartmentID).Error; err == nil {
				doctors[i].DepartmentName = dept.Name
			}
		}
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "查询成功", Data: doctors})
}

func (ctrl *SystemController) GetBlockchainStatus(c *gin.Context) {
	status := blockchain.DefaultService.GetBlockchainStatus()
	c.JSON(http.StatusOK, model.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    status,
	})
}

type CreateHospitalDTO struct {
	HospitalNo string `json:"hospital_no" binding:"required"`
	Name       string `json:"name" binding:"required"`
	Level      string `json:"level"`
	Address    string `json:"address"`
}

type UpdateHospitalDTO struct {
	Name    string `json:"name" binding:"required"`
	Level   string `json:"level"`
	Address string `json:"address"`
}

type UpdateHospitalStatusDTO struct {
	Status *int8 `json:"status" binding:"required"`
}

type CreateDepartmentDTO struct {
	DeptNo      string `json:"dept_no" binding:"required"`
	HospitalID  uint64 `json:"hospital_id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type UpdateDepartmentDTO struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

func (ctrl *SystemController) CreateHospital(c *gin.Context) {
	var req CreateHospitalDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误: 机构代码和机构名称为必填项"})
		return
	}

	hospitalNo := strings.TrimSpace(req.HospitalNo)
	name := strings.TrimSpace(req.Name)
	if hospitalNo == "" || name == "" {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "机构代码与机构名称不能为空"})
		return
	}

	var existing model.Hospital
	if err := repository.DB.Where("hospital_no = ?", hospitalNo).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, model.Response{Code: 409, Message: "机构代码已存在，请勿重复添加"})
		return
	}

	level := strings.TrimSpace(req.Level)
	if level == "" {
		level = "三甲"
	}

	h := model.Hospital{
		HospitalNo: hospitalNo,
		Name:       name,
		Level:      level,
		Address:    strings.TrimSpace(req.Address),
		Status:     1,
		CreatedAt:  time.Now(),
	}

	if err := repository.DB.Create(&h).Error; err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "创建医疗机构失败: " + err.Error()})
		return
	}

	if service.DefaultAuditService != nil {
		service.DefaultAuditService.Log(c.GetUint64("user_id"), "CREATE_HOSPITAL", "HOSPITAL", h.HospitalNo, h.ID, "SUCCESS", "LOW", c.ClientIP())
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "医疗机构接入成功", Data: h})
}

func (ctrl *SystemController) UpdateHospital(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "无效的机构ID"})
		return
	}

	var req UpdateHospitalDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误: 机构名称为必填项"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "机构名称不能为空"})
		return
	}

	var h model.Hospital
	if err := repository.DB.First(&h, id).Error; err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "医疗机构不存在"})
		return
	}

	updates := map[string]interface{}{
		"name": name,
	}
	if req.Level != "" {
		updates["level"] = strings.TrimSpace(req.Level)
	}
	if req.Address != "" {
		updates["address"] = strings.TrimSpace(req.Address)
	}

	if err := repository.DB.Model(&h).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "更新医疗机构失败: " + err.Error()})
		return
	}

	repository.DB.First(&h, id)

	if service.DefaultAuditService != nil {
		service.DefaultAuditService.Log(c.GetUint64("user_id"), "UPDATE_HOSPITAL", "HOSPITAL", h.HospitalNo, h.ID, "SUCCESS", "LOW", c.ClientIP())
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "机构信息更新成功", Data: h})
}

func (ctrl *SystemController) UpdateHospitalStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "无效的机构ID"})
		return
	}

	var req UpdateHospitalStatusDTO
	if err := c.ShouldBindJSON(&req); err != nil || req.Status == nil || (*req.Status != 0 && *req.Status != 1) {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误: 状态值仅支持 0(停用) 或 1(正常)"})
		return
	}

	var h model.Hospital
	if err := repository.DB.First(&h, id).Error; err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "医疗机构不存在"})
		return
	}

	newStatus := *req.Status
	if err := repository.DB.Model(&model.Hospital{}).Where("id = ?", h.ID).Update("status", newStatus).Error; err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "更新机构状态失败: " + err.Error()})
		return
	}

	if service.DefaultAuditService != nil {
		service.DefaultAuditService.Log(c.GetUint64("user_id"), "UPDATE_HOSPITAL_STATUS", "HOSPITAL", h.HospitalNo, h.ID, "SUCCESS", "LOW", c.ClientIP())
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: fmt.Sprintf("机构状态已成功更新为 %d", newStatus),
		Data: gin.H{
			"id":          h.ID,
			"hospital_no": h.HospitalNo,
			"status":      newStatus,
		},
	})
}

func (ctrl *SystemController) CreateDepartment(c *gin.Context) {
	var req CreateDepartmentDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误: 科室代码、所属机构及科室名称为必填项"})
		return
	}

	deptNo := strings.TrimSpace(req.DeptNo)
	name := strings.TrimSpace(req.Name)
	if deptNo == "" || name == "" || req.HospitalID == 0 {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "科室代码、所属机构与科室名称均不能为空"})
		return
	}

	var hosp model.Hospital
	if err := repository.DB.First(&hosp, req.HospitalID).Error; err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "指定的所属医疗机构不存在"})
		return
	}

	var existing model.Department
	if err := repository.DB.Where("hospital_id = ? AND dept_no = ?", req.HospitalID, deptNo).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, model.Response{Code: 409, Message: "该机构下已存在相同科室代码，请勿重复创建"})
		return
	}

	dept := model.Department{
		DeptNo:      deptNo,
		HospitalID:  req.HospitalID,
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		CreatedAt:   time.Now(),
	}

	if err := repository.DB.Create(&dept).Error; err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "创建科室失败: " + err.Error()})
		return
	}

	if service.DefaultAuditService != nil {
		service.DefaultAuditService.Log(c.GetUint64("user_id"), "CREATE_DEPARTMENT", "DEPARTMENT", dept.DeptNo, dept.HospitalID, "SUCCESS", "LOW", c.ClientIP())
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "科室创建成功", Data: dept})
}

func (ctrl *SystemController) UpdateDepartment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "无效的科室ID"})
		return
	}

	var req UpdateDepartmentDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误: 科室名称为必填项"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "科室名称不能为空"})
		return
	}

	var dept model.Department
	if err := repository.DB.First(&dept, id).Error; err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "科室不存在"})
		return
	}

	updates := map[string]interface{}{
		"name":        name,
		"description": strings.TrimSpace(req.Description),
	}

	if err := repository.DB.Model(&dept).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "更新科室失败: " + err.Error()})
		return
	}

	repository.DB.First(&dept, id)

	if service.DefaultAuditService != nil {
		service.DefaultAuditService.Log(c.GetUint64("user_id"), "UPDATE_DEPARTMENT", "DEPARTMENT", dept.DeptNo, dept.HospitalID, "SUCCESS", "LOW", c.ClientIP())
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "科室信息更新成功", Data: dept})
}

func (ctrl *SystemController) DeleteDepartment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "无效的科室ID"})
		return
	}

	var dept model.Department
	if err := repository.DB.First(&dept, id).Error; err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "科室不存在"})
		return
	}

	// 级联防灾校验 1: 检查是否存在关联医生/医护人员
	var userCount int64
	repository.DB.Model(&model.User{}).Where("department_id = ?", dept.ID).Count(&userCount)
	if userCount > 0 {
		c.JSON(http.StatusConflict, model.Response{
			Code:    409,
			Message: fmt.Sprintf("该科室下尚有关联的 %d 位医生/医护人员，禁止删除", userCount),
		})
		return
	}

	// 级联防灾校验 2: 检查是否存在历史开立病历
	var recordCount int64
	repository.DB.Model(&model.MedicalRecord{}).Where("department_id = ?", dept.ID).Count(&recordCount)
	if recordCount > 0 {
		c.JSON(http.StatusConflict, model.Response{
			Code:    409,
			Message: fmt.Sprintf("该科室已产生 %d 份历史诊疗病历档案，禁止直接删除", recordCount),
		})
		return
	}

	if err := repository.DB.Delete(&dept).Error; err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "删除科室失败: " + err.Error()})
		return
	}

	if service.DefaultAuditService != nil {
		service.DefaultAuditService.Log(c.GetUint64("user_id"), "DELETE_DEPARTMENT", "DEPARTMENT", dept.DeptNo, dept.HospitalID, "SUCCESS", "LOW", c.ClientIP())
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "科室删除成功"})
}
