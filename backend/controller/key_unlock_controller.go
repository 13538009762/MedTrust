package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"medtrust-backend/model"
	"medtrust-backend/pkg/captcha"
	"medtrust-backend/service"
)

type UnlockPatientByKeyDTO struct {
	PatientID     uint64 `json:"patient_id" binding:"required"`
	MedicalKey    string `json:"medical_key" binding:"required"`
	CaptchaAnswer string `json:"captcha_answer"`
	CaptchaToken  string `json:"captcha_token"`
	Purpose       string `json:"purpose"`
	Days          int    `json:"days"`
}

// UnlockPatientByKey 医生在现场输入患者专属密钥，核验通过后即时解锁放行该患者全套病历并固化上链
func (ctrl *AccessController) UnlockPatientByKey(c *gin.Context) {
	doctorID := c.GetUint64("user_id")
	var req UnlockPatientByKeyDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误：请输入患者ID与专属授权密钥"})
		return
	}
	// 校验图形防分码验证码
	if gin.Mode() != gin.TestMode {
		if ok, errMsg := captcha.VerifyCaptcha(req.CaptchaToken, req.CaptchaAnswer); !ok {
			c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: errMsg})
			return
		}
	}

	auth, patient, err := service.DefaultKeyUnlockService.UnlockPatientByKey(serviceCtx(c), doctorID, service.UnlockPatientByKeyInput{
		PatientID:  req.PatientID,
		MedicalKey: req.MedicalKey,
		Days:       req.Days,
	})
	if err != nil {
		respondKeyUnlockError(c, err)
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "患者现场密钥核验通过！已即时解锁放行「" + patient.RealName + "」名下全部历史就诊与检验影像档案，授权策略已固化上链",
		Data:    gin.H{"auth": auth},
	})
}

type UnlockByKeyDTO struct {
	RecordID      uint64 `json:"record_id" binding:"required"`
	MedicalKey    string `json:"medical_key" binding:"required"`
	CaptchaAnswer string `json:"captcha_answer"`
	CaptchaToken  string `json:"captcha_token"`
	Purpose       string `json:"purpose"`
	Days          int    `json:"days"`
}

// UnlockByKey 医生在就诊现场输入患者病历专属密钥，核验通过后即时解锁放行并存证上链
func (ctrl *AccessController) UnlockByKey(c *gin.Context) {
	doctorID := c.GetUint64("user_id")
	var req UnlockByKeyDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误：请输入病历ID与患者密钥"})
		return
	}
	// 校验图形防分码验证码
	if gin.Mode() != gin.TestMode {
		if ok, errMsg := captcha.VerifyCaptcha(req.CaptchaToken, req.CaptchaAnswer); !ok {
			c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: errMsg})
			return
		}
	}

	result, err := service.DefaultKeyUnlockService.UnlockByKey(serviceCtx(c), doctorID, service.UnlockByKeyInput{
		RecordID:   req.RecordID,
		MedicalKey: req.MedicalKey,
		Days:       req.Days,
	})
	if err != nil {
		respondKeyUnlockError(c, err)
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "患者现场密钥核验通过！已即时解锁放行该份跨院病历并完成区块链存证",
		Data:    gin.H{"record": result.Record, "auth": result.Auth},
	})
}

// respondKeyUnlockError 依据领域错误类型映射 HTTP 状态码
func respondKeyUnlockError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrKeyVerifyFailed):
		c.JSON(http.StatusForbidden, model.Response{Code: 403, Message: err.Error()})
	case errors.Is(err, service.ErrKeyUnlockLimit):
		c.JSON(http.StatusTooManyRequests, model.Response{Code: 429, Message: err.Error()})
	case errors.Is(err, service.ErrKeyUnlockLocked):
		c.JSON(http.StatusTooManyRequests, model.Response{Code: 429, Message: err.Error()})
	case errors.Is(err, service.ErrPatientNotFound):
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "患者用户档案不存在"})
	case errors.Is(err, service.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "目标病历档案不存在"})
	default:
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: err.Error()})
	}
}
