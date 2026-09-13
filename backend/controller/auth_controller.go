package controller

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

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	RealName string `json:"real_name" binding:"required"`
	IDCard   string `json:"id_card" binding:"required"`
	Phone    string `json:"phone" binding:"required"`
}

func (ctrl *AuthController) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误，请完整填写真实姓名、身份证号、手机号、账号名及密码"})
		return
	}

	user, err := service.DefaultAuthService.RegisterPatient(req.Username, req.Password, req.RealName, req.IDCard, req.Phone)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "患者健康档案账户建档注册成功，请使用新账号登录",
		Data:    user,
	})
}

type UpdateProfileRequest struct {
	RealName string `json:"real_name"`
	Phone    string `json:"phone"`
	IDCard   string `json:"id_card"`
	Title    string `json:"title"`
}

func (ctrl *AuthController) UpdateProfile(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误"})
		return
	}

	user, err := service.DefaultAuthService.UpdateProfile(userID, req.RealName, req.Phone, req.IDCard, req.Title)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "个人信息更新成功",
		Data:    user,
	})
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

func (ctrl *AuthController) ChangePassword(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "请输入原密码与新密码"})
		return
	}

	if err := service.DefaultAuthService.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "密码修改成功，请牢记新密码",
		Data:    nil,
	})
}

type ChangeMedicalKeyRequest struct {
	OldKey string `json:"old_key"`
	NewKey string `json:"new_key" binding:"required"`
}

func (ctrl *AuthController) ChangeMedicalKey(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req ChangeMedicalKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "请输入新的病历调阅专属密钥"})
		return
	}

	if err := service.DefaultAuthService.UpdateMedicalKey(userID, req.OldKey, req.NewKey); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "病历调阅专属密钥已成功更新！请妥善保管并在外院就诊时向医生提供或现场输入",
		Data:    gin.H{"medical_key": req.NewKey},
	})
}

