package controller

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/pkg/captcha"
	"medtrust-backend/repository"
	"medtrust-backend/service"
	"net/http"
	"strings"
	"sync"
	"time"
)

type keyUnlockFailState struct {
	count       int
	lockedUntil time.Time
}

var keyUnlockLimiter sync.Map

func checkKeyUnlockLocked(doctorID, patientID uint64) (bool, int) {
	k := fmt.Sprintf("key_unlock:%d:%d", doctorID, patientID)
	if v, ok := keyUnlockLimiter.Load(k); ok {
		st := v.(keyUnlockFailState)
		if st.lockedUntil.After(time.Now()) {
			rem := int(st.lockedUntil.Sub(time.Now()).Minutes()) + 1
			return true, rem
		}
	}
	return false, 0
}

func recordKeyUnlockResult(doctorID, patientID uint64, success bool) int {
	k := fmt.Sprintf("key_unlock:%d:%d", doctorID, patientID)
	if success {
		keyUnlockLimiter.Delete(k)
		return 0
	}
	st := keyUnlockFailState{count: 1}
	if v, ok := keyUnlockLimiter.Load(k); ok {
		prev := v.(keyUnlockFailState)
		st.count = prev.count + 1
	}
	if st.count >= 5 {
		st.lockedUntil = time.Now().Add(15 * time.Minute)
	}
	keyUnlockLimiter.Store(k, st)
	return st.count
}

type UnlockPatientByKeyDTO struct {
	PatientID     uint64 `json:"patient_id" binding:"required"`
	MedicalKey    string `json:"medical_key" binding:"required"`
	CaptchaAnswer string `json:"captcha_answer"`
	CaptchaToken  string `json:"captcha_token"`
	Purpose       string `json:"purpose"`
	Days          int    `json:"days"`
}

// UnlockPatientByKey 医生在现场输入患者专属密钥，核验通过后即时解密放行该患者全套病历并固化上链
func (ctrl *AccessController) UnlockPatientByKey(c *gin.Context) {
	doctorID := c.GetUint64("user_id")
	var req UnlockPatientByKeyDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误：请输入患者ID与专属授权密钥"})
		return
	}
	// 校验图形防爆破验证码
	if gin.Mode() != gin.TestMode {
		if ok, errMsg := captcha.VerifyCaptcha(req.CaptchaToken, req.CaptchaAnswer); !ok {
			c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: errMsg})
			return
		}
	}

	if locked, rem := checkKeyUnlockLocked(doctorID, req.PatientID); locked {
		c.JSON(http.StatusTooManyRequests, model.Response{
			Code:    429,
			Message: fmt.Sprintf("针对该患者的调阅密钥连续核验失败已达上限，安全防暴破锁定中（剩余 %d 分钟）", rem),
		})
		return
	}

	var patient model.User
	if err := repository.DB.First(&patient, req.PatientID).Error; err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "目标患者档案不存在"})
		return
	}

	var doctor model.User
	_ = repository.DB.First(&doctor, doctorID)

	inputKey := strings.TrimSpace(req.MedicalKey)
	if !service.VerifyMedicalKey(inputKey, patient.UserNo, patient.MedicalKeyHash, patient.MedicalKey) {
		failCnt := recordKeyUnlockResult(doctorID, patient.ID, false)
		service.DefaultAuditService.Log(doctorID, "KEY_UNLOCK_FAILED", "PATIENT", patient.UserNo, doctor.HospitalID, "INTERCEPTED", "HIGH", c.ClientIP())
		if failCnt >= 5 {
			c.JSON(http.StatusTooManyRequests, model.Response{
				Code:    429,
				Message: "连续密钥错误已达 5 次，已触发防暴破安全保护，锁定该患者的密钥解锁通道 15 分钟",
			})
			return
		}
		c.JSON(http.StatusForbidden, model.Response{
			Code:    403,
			Message: "患者授权密钥校验失败，密码不正确！请由患者在个人中心核实或重新设置调阅密钥",
		})
		return
	}
	recordKeyUnlockResult(doctorID, patient.ID, true)

	// 密钥核验通过：生成对该患者全部档案全局生效的授权凭证
	days := req.Days
	if days <= 0 {
		days = 7
	}
	startTime := time.Now()
	endTime := startTime.Add(time.Duration(days) * 24 * time.Hour)

	randBytes := make([]byte, 3)
	_, _ = rand.Read(randBytes)
	authNo := fmt.Sprintf("AUTH-KEY-ALL-%s-%s", startTime.Format("20060102"), hex.EncodeToString(randBytes))

	txID, _, _ := blockchain.DefaultService.CommitAsset("AUTHORIZATION_KEY", authNo, map[string]interface{}{
		"auth_no":     authNo,
		"patient_id":  patient.ID,
		"target_type": "DOCTOR",
		"target_id":   doctorID,
		"scope":       "ALL",
		"record_id":   0,
		"status":      "ACTIVE",
		"timestamp":   startTime.Format(time.RFC3339),
	})

	auth := model.Authorization{
		AuthNo:         authNo,
		PatientID:      patient.ID,
		AuthTargetType: "DOCTOR",
		AuthTargetID:   doctorID,
		ScopeType:      "ALL",
		RecordID:       0,
		StartTime:      startTime,
		EndTime:        endTime,
		Status:         "ACTIVE",
		FabricTxID:     txID,
		CreatedAt:      startTime,
	}
	repository.DB.Create(&auth)

	service.DefaultAuditService.Log(doctorID, "KEY_UNLOCK_SUCCESS", "PATIENT", patient.UserNo, doctor.HospitalID, "SUCCESS", "LOW", c.ClientIP())

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: fmt.Sprintf("患者现场密钥核验通过！已即时解密放行「%s」名下全部历史就诊与检验影像档案，授权策略已固化上链", patient.RealName),
		Data: gin.H{
			"auth": auth,
		},
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

// UnlockByKey 医生在就诊现场输入患者病历专属密钥，核验通过后即时解密放行并存证上链
func (ctrl *AccessController) UnlockByKey(c *gin.Context) {
	doctorID := c.GetUint64("user_id")
	var req UnlockByKeyDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误：请输入病历ID与患者密钥"})
		return
	}
	// 校验图形防爆破验证码
	if gin.Mode() != gin.TestMode {
		if ok, errMsg := captcha.VerifyCaptcha(req.CaptchaToken, req.CaptchaAnswer); !ok {
			c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: errMsg})
			return
		}
	}

	var rec model.MedicalRecord
	if err := repository.DB.First(&rec, req.RecordID).Error; err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "目标病历档案不存在"})
		return
	}

	if locked, rem := checkKeyUnlockLocked(doctorID, rec.PatientID); locked {
		c.JSON(http.StatusTooManyRequests, model.Response{
			Code:    429,
			Message: fmt.Sprintf("针对该患者的调阅密钥连续核验失败已达上限，安全防暴破锁定中（剩余 %d 分钟）", rem),
		})
		return
	}

	var patient model.User
	if err := repository.DB.First(&patient, rec.PatientID).Error; err != nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "患者用户档案不存在"})
		return
	}

	var doctor model.User
	_ = repository.DB.First(&doctor, doctorID)

	inputKey := strings.TrimSpace(req.MedicalKey)
	if !service.VerifyMedicalKey(inputKey, patient.UserNo, patient.MedicalKeyHash, patient.MedicalKey) {
		failCnt := recordKeyUnlockResult(doctorID, patient.ID, false)
		service.DefaultAuditService.Log(doctorID, "KEY_UNLOCK_FAILED", "RECORD", rec.RecordNo, doctor.HospitalID, "INTERCEPTED", "HIGH", c.ClientIP())
		if failCnt >= 5 {
			c.JSON(http.StatusTooManyRequests, model.Response{
				Code:    429,
				Message: "连续密钥错误已达 5 次，已触发防暴破安全保护，锁定该患者的密钥解锁通道 15 分钟",
			})
			return
		}
		c.JSON(http.StatusForbidden, model.Response{
			Code:    403,
			Message: "患者授权密钥校验失败，密码不正确！请由患者在个人中心核实或重新设置调阅密钥",
		})
		return
	}
	recordKeyUnlockResult(doctorID, patient.ID, true)

	// 密钥核验通过：生成即时生效的授权凭证
	days := req.Days
	if days <= 0 {
		days = 7
	}
	startTime := time.Now()
	endTime := startTime.Add(time.Duration(days) * 24 * time.Hour)

	randBytes := make([]byte, 3)
	_, _ = rand.Read(randBytes)
	authNo := fmt.Sprintf("AUTH-KEY-%s-%s", time.Now().Format("20060102"), hex.EncodeToString(randBytes))

	txID, _, _ := blockchain.DefaultService.CommitAsset("AUTHORIZATION_KEY", authNo, map[string]interface{}{
		"auth_no":     authNo,
		"patient_id":  patient.ID,
		"target_type": "DOCTOR",
		"target_id":   doctorID,
		"scope":       "SINGLE",
		"record_id":   req.RecordID,
		"status":      "ACTIVE",
		"auth_method": "PATIENT_KEY_VERIFIED",
		"timestamp":   time.Now().Format(time.RFC3339),
	})

	auth := model.Authorization{
		AuthNo:         authNo,
		PatientID:      patient.ID,
		AuthTargetType: "DOCTOR",
		AuthTargetID:   doctorID,
		ScopeType:      "SINGLE",
		RecordID:       req.RecordID,
		StartTime:      startTime,
		EndTime:        endTime,
		Status:         "ACTIVE",
		FabricTxID:     txID,
		CreatedAt:      time.Now(),
	}
	repository.DB.Create(&auth)

	// 若有关于此病历与医生的待处理申请单，顺便标记为已核准
	repository.DB.Model(&model.AccessRequest{}).
		Where("doctor_id = ? AND record_id = ? AND status = 'PENDING'", doctorID, req.RecordID).
		Updates(map[string]interface{}{
			"status":   "APPROVED",
			"decision": "ALLOWED",
		})

	service.DefaultAuditService.Log(doctorID, "KEY_UNLOCK_SUCCESS", "RECORD", rec.RecordNo, doctor.HospitalID, "SUCCESS", "LOW", c.ClientIP())

	// 加载完整解密病历
	fullRecord, err := service.DefaultMedicalService.GetRecordByID(req.RecordID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "解密病历失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "患者现场密钥核验通过！已即时解密放行该份跨院病历并完成区块链存证",
		Data: gin.H{
			"record": fullRecord,
			"auth":   auth,
		},
	})
}
