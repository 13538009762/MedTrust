package controller

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"

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
	repository.DB.Model(&model.User{}).Where("id = ?", id).Update("status", req.Status)

	service.DefaultAuditService.Log(c.GetUint64("user_id"), "UPDATE_USER", "USER", u.UserNo, u.HospitalID, "SUCCESS", "LOW", "127.0.0.1")
	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "账号状态变更成功"})
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

	pwd := dto.Password
	if pwd == "" {
		pwd = "123456"
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)

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
		UserNo:       userNo,
		Username:     dto.Username,
		PasswordHash: string(hash),
		RealName:     dto.RealName,
		Role:         dto.Role,
		HospitalID:   dto.HospitalID,
		DepartmentID: dto.DepartmentID,
		Title:        dto.Title,
		Phone:        dto.Phone,
		IDCard:       dto.IDCard,
		Status:       status,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := repository.DB.Create(&u).Error; err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "创建用户失败: " + err.Error()})
		return
	}

	service.DefaultAuditService.Log(c.GetUint64("user_id"), "CREATE_USER", "USER", u.UserNo, u.HospitalID, "SUCCESS", "LOW", "127.0.0.1")
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
