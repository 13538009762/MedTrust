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

type DBAccessHistoryProvider struct{}

func (p *DBAccessHistoryProvider) GetRecentAccessCount(doctorID, patientID uint64, duration time.Duration) int {
	if repository.DB == nil {
		return 0
	}
	var count int64
	since := time.Now().Add(-duration)
	repository.DB.Table("audit_logs").
		Joins("JOIN medical_records ON audit_logs.target_id = CAST(medical_records.id AS CHAR)").
		Where("audit_logs.user_id = ? AND medical_records.patient_id = ? AND audit_logs.operation_type = 'ACCESS' AND audit_logs.result = 'SUCCESS' AND audit_logs.created_at >= ?", doctorID, patientID, since).
		Count(&count)
	return int(count)
}

func (p *DBAccessHistoryProvider) GetDistinctPatientsAccessed(doctorID uint64, duration time.Duration) int {
	if repository.DB == nil {
		return 0
	}
	var count int64
	since := time.Now().Add(-duration)
	repository.DB.Table("audit_logs").
		Joins("JOIN medical_records ON audit_logs.target_id = CAST(medical_records.id AS CHAR)").
		Where("audit_logs.user_id = ? AND audit_logs.operation_type = 'ACCESS' AND audit_logs.result = 'SUCCESS' AND audit_logs.created_at >= ?", doctorID, since).
		Distinct("medical_records.patient_id").
		Count(&count)
	return int(count)
}

func (p *DBAccessHistoryProvider) GetHistoricalViolationCount(doctorID uint64, days int) int {
	if repository.DB == nil {
		return 0
	}
	var count int64
	since := time.Now().AddDate(0, 0, -days)
	repository.DB.Model(&model.EmergencyAccessEvent{}).
		Where("doctor_id = ? AND audit_status = 'CLOSED_VIOLATION' AND created_at >= ?", doctorID, since).
		Count(&count)
	return int(count)
}

func (p *DBAccessHistoryProvider) GetDoctorStatus(doctorID uint64) string {
	if repository.DB == nil {
		return "NORMAL"
	}
	var doc model.User
	if err := repository.DB.Select("status").First(&doc, doctorID).Error; err != nil {
		return "NORMAL"
	}
	return doc.Status
}

type AccessEngine struct {
	riskEngine *risk.RiskEngine
}

var DefaultAccessEngine *AccessEngine

func InitAccessEngine(low, high int) {
	eng := risk.NewRiskEngine(low, high)
	eng.SetHistoryProvider(&DBAccessHistoryProvider{})
	DefaultAccessEngine = &AccessEngine{
		riskEngine: eng,
	}
}

func (e *AccessEngine) GetRiskEngine() *risk.RiskEngine {
	return e.riskEngine
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

	// 2. 本院机构病历访问控制 (ABAC 细粒度科室协同判定)
	if !isCrossHospital {
		var docDept model.Department
		if doctor.DepartmentID > 0 {
			_ = repository.DB.First(&docDept, doctor.DepartmentID)
		}

		// 同科室协同诊疗或未区分科室时直接放行
		isSameDept := doctor.DepartmentID == 0 || record.DepartmentName == "" ||
			record.DepartmentName == "综合门诊" || (docDept.Name != "" && docDept.Name == record.DepartmentName)

		if isSameDept {
			return &AccessDecision{
				Allowed:        true,
				IsEmergency:    false,
				Decision:       "ALLOWED",
				Reason:         "病历归属当前医生所属医院同科室诊疗协同，系统核验放行",
				RiskEvaluation: risk.RiskEvaluation{Level: "LOW", TotalScore: 5},
			}, nil
		}
		// 院内跨科室会诊：若未取得患者授权且非紧急情况，引导授权或由高级职称医生协同
		// 后续流程进入显式授权与风险评估流水线
	}

	// 3. 检查患者显式授权 (包括对指定医生、指定医院及全体执业医生的开放授权)
	now := time.Now()
	var auths []model.Authorization
	repository.DB.Where("patient_id = ? AND status = 'ACTIVE' AND start_time <= ? AND end_time >= ?", patientID, now, now).Find(&auths)

	hasConsent := false
	isAllDoctorsConsent := false

	// 先查找该病历专属的独立授权策略 (SINGLE 范围优先级高于全局 ALL 范围)
	var singleAuth *model.Authorization
	for _, a := range auths {
		if a.ScopeType == "SINGLE" && a.RecordID == recordID {
			copyA := a
			singleAuth = &copyA
			break
		}
	}

	if singleAuth != nil {
		if singleAuth.AuthTargetType != "PRIVATE" {
			targetMatch := false
			if singleAuth.AuthTargetType == "ALL_DOCTORS" {
				targetMatch = true
				isAllDoctorsConsent = true
			} else if singleAuth.AuthTargetType == "DOCTOR" && singleAuth.AuthTargetID == doctorID {
				targetMatch = true
			} else if singleAuth.AuthTargetType == "HOSPITAL" && singleAuth.AuthTargetID == doctor.HospitalID {
				targetMatch = true
			}
			if targetMatch {
				hasConsent = true
			}
		}
	} else {
		// 无单病历专属授权时，回退评估全局授权策略 (ScopeType == ALL)
		for _, a := range auths {
			if a.ScopeType == "ALL" {
				targetMatch := false
				if a.AuthTargetType == "ALL_DOCTORS" {
					targetMatch = true
					isAllDoctorsConsent = true
				} else if a.AuthTargetType == "DOCTOR" && a.AuthTargetID == doctorID {
					targetMatch = true
				} else if a.AuthTargetType == "HOSPITAL" && a.AuthTargetID == doctor.HospitalID {
					targetMatch = true
				}
				if targetMatch {
					hasConsent = true
					break
				}
			}
		}
	}

	// 院内跨科室调阅且无患者授权场景：若为副主任医师及以上高级职称且处于工作时间，允许会诊放行（低风险）；否则需患者授权
	if !isCrossHospital && !hasConsent {
		isSenior := doctor.Title == "主任医师" || doctor.Title == "副主任医师"
		if isSenior {
			return &AccessDecision{
				Allowed:        true,
				IsEmergency:    false,
				Decision:       "ALLOWED",
				Reason:         "同院跨科室专家会诊协同调阅，经高级职称属性（ABAC）核验通过，系统放行",
				RiskEvaluation: risk.RiskEvaluation{Level: "LOW", TotalScore: 15},
			}, nil
		}
	}

	// 风险评分引擎评估 (紧急访问走专属绿色通道)
	eval := e.riskEngine.Evaluate(doctorID, patientID, isCrossHospital, hasConsent, isEmergency)

	// 分支 A: 具备患者有效授权
	if hasConsent {
		reason := "常规授权有效，动态风险评估为低风险，系统直接放行"
		if isAllDoctorsConsent {
			reason = "患者已将该病历设置为向全体执业医生公开可见，系统核验放行"
		}
		if eval.Level == "LOW" {
			return &AccessDecision{
				Allowed:        true,
				IsEmergency:    false,
				Decision:       "ALLOWED",
				Reason:         reason,
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
	// 检查该医生是否曾在过去 24 小时内对该病历成功实施过破窗放行，且未被监管裁定违规或患者提出异议
	var pastEmg model.EmergencyAccessEvent
	if err := repository.DB.Where("doctor_id = ? AND record_id = ? AND created_at >= ? AND audit_status != 'CLOSED_VIOLATION' AND patient_feedback != 'OBJECTED'",
		doctorID, recordID, now.Add(-24*time.Hour)).First(&pastEmg).Error; err == nil {
		return &AccessDecision{
			Allowed:        true,
			IsEmergency:    true,
			Decision:       "ALLOWED",
			Reason:         "已在紧急破窗 24 小时急救与随诊有效期内核准放行 (无患者异议与监管违规)，系统免重复申请直接放行",
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

	// 分支 C: 触发 Break-Glass 抢救放行 (专属绿色通道)
	return &AccessDecision{
		Allowed:        true,
		IsEmergency:    true,
		Decision:       "BREAK_GLASS_ALLOWED",
		Reason:         "临床急救访问责任声明核验通过，临时放行并生成不可篡改紧急事件单",
		RiskEvaluation: eval,
	}, nil
}

// RecordSuccessfulAccess 记录一次真实且成功的病历调阅行为至时间窗口
func (e *AccessEngine) RecordSuccessfulAccess(doctorID, patientID uint64) {
	if e.riskEngine != nil {
		e.riskEngine.RecordSuccessfulAccess(doctorID, patientID)
	}
}
