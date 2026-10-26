package controller

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
	"medtrust-backend/service"
	"net/http"
	"time"
)

type CreateAuthRequest struct {
	AuthTargetType string    `json:"auth_target_type" binding:"required"` // DOCTOR, HOSPITAL, ALL_DOCTORS
	AuthTargetID   uint64    `json:"auth_target_id"`                      // 当 ALL_DOCTORS 时可为 0
	ScopeType      string    `json:"scope_type" binding:"required"`       // ALL, SINGLE
	RecordID       uint64    `json:"record_id"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	Days           int       `json:"days"`
}

func (ctrl *AccessController) CreateAuthorization(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	var req CreateAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误"})
		return
	}

	if req.AuthTargetType != "ALL_DOCTORS" && req.AuthTargetID == 0 {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "请选择具体的授权目标医生或医疗机构"})
		return
	}
	if req.ScopeType == "SINGLE" && req.RecordID == 0 {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "指定单份病历授权时必须选择具体病历"})
		return
	}

	if req.Days > 0 {
		req.StartTime = time.Now()
		req.EndTime = req.StartTime.Add(time.Duration(req.Days) * 24 * time.Hour)
	} else {
		if req.StartTime.IsZero() {
			req.StartTime = time.Now()
		}
		if req.EndTime.IsZero() {
			req.EndTime = time.Now().Add(365 * 24 * time.Hour)
		}
	}

	randBytes := make([]byte, 3)
	_, _ = rand.Read(randBytes)
	authNo := fmt.Sprintf("AUTH%s%s", time.Now().Format("20060102"), hex.EncodeToString(randBytes))

	txID, _, _ := blockchain.DefaultService.CommitAsset("AUTHORIZATION", authNo, map[string]interface{}{
		"auth_no":     authNo,
		"patient_id":  patientID,
		"target_type": req.AuthTargetType,
		"target_id":   req.AuthTargetID,
		"scope":       req.ScopeType,
		"record_id":   req.RecordID,
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

	service.DefaultAuditService.Log(patientID, "AUTHORIZE", "AUTH", authNo, 0, "SUCCESS", "LOW", c.ClientIP())
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

	docIDs := make([]uint64, 0, len(list))
	hospIDs := make([]uint64, 0, len(list))
	recIDs := make([]uint64, 0, len(list))

	for _, a := range list {
		if a.AuthTargetType == "DOCTOR" && a.AuthTargetID > 0 {
			docIDs = append(docIDs, a.AuthTargetID)
		} else if a.AuthTargetType == "HOSPITAL" && a.AuthTargetID > 0 {
			hospIDs = append(hospIDs, a.AuthTargetID)
		}
		if a.RecordID > 0 {
			recIDs = append(recIDs, a.RecordID)
		}
	}

	docMap := loadUsersBatch(docIDs)
	recMap := loadRecordsBatch(recIDs)

	for _, d := range docMap {
		if d.HospitalID > 0 {
			hospIDs = append(hospIDs, d.HospitalID)
		}
	}
	for _, r := range recMap {
		if r.HospitalID > 0 {
			hospIDs = append(hospIDs, r.HospitalID)
		}
	}
	hospMap := loadHospitalsBatch(hospIDs)

	for i := range list {
		if list[i].AuthTargetType == "ALL_DOCTORS" {
			list[i].TargetName = "全体执业医生 (全联盟公开)"
		} else if list[i].AuthTargetType == "DOCTOR" {
			if doc, ok := docMap[list[i].AuthTargetID]; ok {
				hospSuffix := ""
				if hosp, hOk := hospMap[doc.HospitalID]; hOk && hosp.Name != "" {
					hospSuffix = " (" + hosp.Name + ")"
				}
				list[i].TargetName = doc.RealName + " 医生" + hospSuffix
			}
		} else if list[i].AuthTargetType == "HOSPITAL" {
			if hosp, ok := hospMap[list[i].AuthTargetID]; ok {
				list[i].TargetName = hosp.Name
			}
		}

		if list[i].RecordID > 0 {
			if rec, ok := recMap[list[i].RecordID]; ok {
				list[i].RecordNo = rec.RecordNo
				list[i].RecordDiagnosis = rec.Diagnosis
				list[i].RecordDepartment = rec.DepartmentName
				if hosp, ok := hospMap[rec.HospitalID]; ok {
					list[i].RecordHospitalName = hosp.Name
				}
			}
		}
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "查询成功", Data: list})
}
