package service

import (
	"time"

	"medtrust-backend/model"
	"medtrust-backend/pkg/risk"
	"medtrust-backend/repository"
)

type AccessDecision struct {
	Allowed        bool                `json:"allowed"`
	IsEmergency    bool                `json:"is_emergency"`
	Decision       string              `json:"decision"` // ALLOWED, NEED_BREAK_GLASS, PENDING_CONFIRM, REJECTED, BREAK_GLASS_ALLOWED
	Reason         string              `json:"reason"`
	RiskEvaluation risk.RiskEvaluation `json:"risk_evaluation"`
}

type AccessEngine struct {
	riskEngine *risk.RiskEngine
}

var DefaultAccessEngine *AccessEngine

func InitAccessEngine(low, high int) {
	DefaultAccessEngine = &AccessEngine{
		riskEngine: risk.NewRiskEngine(low, high),
	}
}

// EvaluateAccess 执行统一的权限与紧急访问评估流水线
func (e *AccessEngine) EvaluateAccess(doctorID, patientID, recordID uint64, isEmergency bool) (*AccessDecision, error) {
	// 1. RBAC & ABAC: 基础身份前置安全核验
	var doctor model.User
	if err := repository.DB.First(&doctor, doctorID).Error; err != nil {
		return &AccessDecision{Allowed: false, Decision: "REJECTED", Reason: "未找到该医生身份信息"}, nil
	}
	if doctor.Role != "doctor" {
		return &AccessDecision{Allowed: false, Decision: "REJECTED", Reason: "非合法医生角色，拒绝访问"}, nil
	}
	if doctor.Status == "DISABLED" {
		return &AccessDecision{Allowed: false, Decision: "REJECTED", Reason: "医生账号已被系统封禁禁用，禁止任何调阅操作"}, nil
	}
	if doctor.Status == "RESTRICTED" && isEmergency {
		return &AccessDecision{Allowed: false, Decision: "REJECTED", Reason: "该医生账号已被监管部门下达惩戒，禁止使用紧急访问权限"}, nil
	}

	// 检查病历归属机构与开具医生
	var record model.MedicalRecord
	if err := repository.DB.First(&record, recordID).Error; err != nil {
		return &AccessDecision{Allowed: false, Decision: "REJECTED", Reason: "目标医疗记录不存在"}, nil
	}

	// 1. 本人开具的病历：医生对本人开具的病历依法享有直接诊疗与调阅权限，免除授权申请
	if record.DoctorID == doctorID {
		return &AccessDecision{
			Allowed:        true,
			IsEmergency:    false,
			Decision:       "ALLOWED",
			Reason:         "该病历由当前医生本人开具，依法享有直接诊疗与调阅权限，系统直接放行",
			RiskEvaluation: risk.RiskEvaluation{Level: "LOW", TotalScore: 0},
		}, nil
	}

	isCrossHospital := doctor.HospitalID != record.HospitalID
	if doctor.Status == "RESTRICTED" && isCrossHospital {
		return &AccessDecision{Allowed: false, Decision: "REJECTED", Reason: "该医生账号已被监管限制跨机构调阅权限"}, nil
	}

	// 2. 本院同机构病历：同一医疗机构内诊疗延续与医嘱互认，系统直接放行
	if !isCrossHospital {
		return &AccessDecision{
			Allowed:        true,
			IsEmergency:    false,
			Decision:       "ALLOWED",
			Reason:         "病历归属当前医生所属医疗机构，属于院内诊疗互通，系统直接放行",
			RiskEvaluation: risk.RiskEvaluation{Level: "LOW", TotalScore: 5},
		}, nil
	}

	// 2. 检查患者显式授权
	now := time.Now()
	var auths []model.Authorization
	repository.DB.Where("patient_id = ? AND status = 'ACTIVE' AND start_time <= ? AND end_time >= ?", patientID, now, now).Find(&auths)

	hasConsent := false
	for _, a := range auths {
		targetMatch := false
		if a.AuthTargetType == "DOCTOR" && a.AuthTargetID == doctorID {
			targetMatch = true
		} else if a.AuthTargetType == "HOSPITAL" && a.AuthTargetID == doctor.HospitalID {
			targetMatch = true
		}
		if targetMatch {
			if a.ScopeType == "ALL" || (a.ScopeType == "SINGLE" && a.RecordID == recordID) {
				hasConsent = true
				break
			}
		}
	}

	// 风险评分引擎评估
	eval := e.riskEngine.Evaluate(doctorID, patientID, isCrossHospital, hasConsent, isEmergency)

	// 分支 A: 具备患者有效授权
	if hasConsent {
		if eval.Level == "LOW" {
			return &AccessDecision{
				Allowed:        true,
				IsEmergency:    false,
				Decision:       "ALLOWED",
				Reason:         "常规授权有效，动态风险评估为低风险，系统直接放行",
				RiskEvaluation: eval,
			}, nil
		} else if eval.Level == "MEDIUM" {
			return &AccessDecision{
				Allowed:        false,
				IsEmergency:    false,
				Decision:       "PENDING_CONFIRM",
				Reason:         "检测到中风险操作，等待患者在线确认二次授权",
				RiskEvaluation: eval,
			}, nil
		} else {
			return &AccessDecision{
				Allowed:        false,
				IsEmergency:    false,
				Decision:       "REJECTED",
				Reason:         "高风险异常调阅，系统安全规则拦截阻断",
				RiskEvaluation: eval,
			}, nil
		}
	}

	// 分支 B: 未取得患者显式授权
	// 检查该医生是否曾在过去 24 小时内对该病历成功实施过破窗放行
	var pastEmg model.EmergencyAccessEvent
	if err := repository.DB.Where("doctor_id = ? AND record_id = ? AND created_at >= ?", doctorID, recordID, now.Add(-24*time.Hour)).First(&pastEmg).Error; err == nil {
		return &AccessDecision{
			Allowed:        true,
			IsEmergency:    true,
			Decision:       "ALLOWED",
			Reason:         "已在紧急破窗 24 小时急救与随诊有效期内核准放行，系统免重复申请直接放行",
			RiskEvaluation: risk.RiskEvaluation{Level: "LOW", TotalScore: 10},
		}, nil
	}

	if !isEmergency {
		return &AccessDecision{
			Allowed:        false,
			IsEmergency:    false,
			Decision:       "NEED_BREAK_GLASS",
			Reason:         "未取得患者在线授权，若处于急诊抢救或危急场景可申请 Break-Glass 紧急访问",
			RiskEvaluation: eval,
		}, nil
	}

	// 分支 C: 触发 Break-Glass 抢救放行
	return &AccessDecision{
		Allowed:        true,
		IsEmergency:    true,
		Decision:       "BREAK_GLASS_ALLOWED",
		Reason:         "临床急救访问责任声明核验通过，临时放行并生成不可篡改紧急事件单",
		RiskEvaluation: eval,
	}, nil
}
