package controller

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"medtrust-backend/config"
	"medtrust-backend/middleware"
	"medtrust-backend/model"
	"medtrust-backend/pkg/captcha"
	"medtrust-backend/repository"
	"medtrust-backend/service"
)

type AuthController struct{}

var DefaultAuthController = &AuthController{}

type LoginRequest struct {
	Username      string `json:"username" binding:"required"`
	Password      string `json:"password" binding:"required"`
	CaptchaAnswer string `json:"captcha_answer"`
	CaptchaToken  string `json:"captcha_token"`
}

func (ctrl *AuthController) GetCaptcha(c *gin.Context) {
	res, err := captcha.GenerateCaptcha()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "生成验证码失败"})
		return
	}
	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "获取验证码成功",
		Data:    res,
	})
}

func (ctrl *AuthController) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误，请输入用户名与密码"})
		return
	}

	// 验证图形计算验证码，防御针对账号密码的自动化碰撞与爆破攻击
	// 若当前调用者已持有合法 JWT (如答辩演示向导切换角色)，且未携带验证码参数，允许平滑切换
	if gin.Mode() != gin.TestMode {
		bypassCaptcha := false
		if strings.TrimSpace(req.CaptchaToken) == "" && strings.TrimSpace(req.CaptchaAnswer) == "" {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
				claims := &middleware.CustomClaims{}
				if token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
					return []byte(config.AppConfig.Server.JWTSecret), nil
				}); err == nil && token.Valid && !repository.IsTokenBlacklisted(tokenStr) {
					bypassCaptcha = true
				}
			}
		}
		if !bypassCaptcha {
			if ok, errMsg := captcha.VerifyCaptcha(req.CaptchaToken, req.CaptchaAnswer); !ok {
				c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: errMsg})
				return
			}
		}
	}

	token, user, err := service.DefaultAuthService.LoginWithContext(req.Username, req.Password, c.ClientIP(), c.Request.UserAgent())
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

	user, err := service.DefaultAuthService.RegisterPatientWithContext(req.Username, req.Password, req.RealName, req.IDCard, req.Phone, c.ClientIP(), c.Request.UserAgent())
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
	Avatar   string `json:"avatar"`
}

func (ctrl *AuthController) UpdateProfile(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误"})
		return
	}

	user, err := service.DefaultAuthService.UpdateProfileWithContext(userID, req.RealName, req.Phone, req.IDCard, req.Title, req.Avatar, c.ClientIP(), c.Request.UserAgent())
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

type UpdateAvatarRequest struct {
	Avatar string `json:"avatar"`
}

func (ctrl *AuthController) UpdateAvatar(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req UpdateAvatarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误"})
		return
	}

	user, err := service.DefaultAuthService.UpdateAvatar(userID, req.Avatar, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "头像更新成功",
		Data:    user,
	})
}

func (ctrl *AuthController) UploadAvatar(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 5<<20)
	userID := c.GetUint64("user_id")
	file, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "请选择要上传的头像图片"})
		return
	}

	if file.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "头像文件大小不能超过 5MB"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "打开上传文件失败"})
		return
	}
	defer f.Close()

	buf := make([]byte, file.Size)
	if _, err := f.Read(buf); err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "读取图片内容失败"})
		return
	}

	contentType := http.DetectContentType(buf)
	if !strings.HasPrefix(contentType, "image/") {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "上传的文件不是有效的图片格式"})
		return
	}

	base64Str := base64.StdEncoding.EncodeToString(buf)
	avatarData := fmt.Sprintf("data:%s;base64,%s", contentType, base64Str)

	user, err := service.DefaultAuthService.UpdateAvatar(userID, avatarData, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "头像上传成功",
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

	if err := service.DefaultAuthService.ChangePasswordWithContext(userID, req.OldPassword, req.NewPassword, c.ClientIP(), c.Request.UserAgent()); err != nil {
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

	if err := service.DefaultAuthService.UpdateMedicalKeyWithContext(userID, req.OldKey, req.NewKey, c.ClientIP(), c.Request.UserAgent()); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "医疗密钥修改成功",
		Data:    nil,
	})
}

// Logout 处理用户登出请求，将当前 JWT 加入 Redis / 内存降级黑名单中实现即时失效
func (ctrl *AuthController) Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	tokenStr := ""
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			tokenStr = parts[1]
		}
	}
	if tokenStr == "" {
		tokenStr = c.Query("token")
	}

	if tokenStr == "" {
		c.JSON(http.StatusBadRequest, model.Response{
			Code:    400,
			Message: "未提供需要注销的 Token",
		})
		return
	}

	// 解析 Token 剩余有效期，动态设置黑名单 TTL
	claims := &middleware.CustomClaims{}
	ttl := 24 * time.Hour
	token, _ := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.AppConfig.Server.JWTSecret), nil
	})
	if token != nil && claims.ExpiresAt != nil {
		remaining := time.Until(claims.ExpiresAt.Time)
		if remaining > 0 {
			ttl = remaining
		}
	}

	// 写入 Redis 与 MemoryCache 双写黑名单
	_ = repository.BlacklistToken(tokenStr, ttl)

	// 获取操作人信息并记录登出审计日志
	userID := c.GetUint64("user_id")
	userNo := c.GetString("user_no")
	hospitalID := c.GetUint64("hospital_id")
	if userID == 0 && claims.UserID > 0 {
		userID = claims.UserID
		userNo = claims.UserNo
		hospitalID = claims.HospitalID
	}

	service.DefaultAuditService.Log(userID, "LOGOUT", "USER", userNo, hospitalID, "SUCCESS", "LOW", c.ClientIP())

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "登出成功，身份令牌已加入黑名单失效",
		Data:    nil,
	})
}

// CreateSSETicket 生成单次有效 (60s) 的 SSE 接入票据 (解决 URL 携带长效 JWT 导致的历史记录与日志泄露问题)
// POST /api/v1/auth/ticket
func (ctrl *AuthController) CreateSSETicket(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, model.Response{Code: 401, Message: "未授权"})
		return
	}
	userID := userIDVal.(uint64)
	userNo, _ := c.Get("user_no")
	role, _ := c.Get("role")
	username, _ := c.Get("username")
	hospitalID, _ := c.Get("hospital_id")
	departmentID, _ := c.Get("department_id")

	// 生成 24 字节高熵随机票据 (48 位十六进制字符)
	ticketBytes := make([]byte, 24)
	if _, err := rand.Read(ticketBytes); err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "生成票据失败"})
		return
	}
	ticket := hex.EncodeToString(ticketBytes)

	// 序列化用户上下文
	ticketData := fmt.Sprintf("%d|%v|%v|%v|%v|%v", userID, userNo, role, username, hospitalID, departmentID)

	if err := repository.StoreSSETicket(ticket, ticketData, 60*time.Second); err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "保存票据失败"})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "创建 SSE 接入票据成功",
		Data: gin.H{
			"ticket":     ticket,
			"expires_in": 60,
		},
	})
}

