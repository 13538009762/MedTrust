package service

import (
	"fmt"
	"strings"
	"time"

	"medtrust-backend/model"
	"medtrust-backend/pkg/risk"
	"medtrust-backend/repository"
)

// AccessRequest 结构化统一访问请求上下文 (对齐行业与论文统一访问控制决策模型)
type AccessRequest struct {
	UserID      uint64 `json:"user_id"`
	Role        string `json:"role"`
	PatientID   uint64 `json:"patient_id"`
	RecordID    uint64 `json:"record_id"`
	Action      string `json:"action"` // "VIEW", "DOWNLOAD", "EXAM", "AUDIT"
	Purpose     string `json:"purpose"`
	Source      string `json:"source"` // "WEB", "API", "AI_AGENT", "BREAK_GLASS"
	IsEmergency bool   `json:"is_emergency"`
	IPAddress   string `json:"ip_address"`
	UserAgent   string `json:"user_agent"`
}

// AccessDecision 统一访问决策输出模型
type AccessDecision struct {
	Allowed         bool                  `json:"allowed"`
	IsEmergency     bool                  `json:"is_emergency"`
	Decision        string                `json:"decision"` // ALLOWED, DENIED, REQUIRE_AUTHORIZATION, REQUIRE_BREAK_GLASS, REQUIRE_SUPERVISOR_APPROVAL, BREAK_GLASS_ALLOWED, PENDING_CONFIRM
	Reason          string                `json:"reason"`
	RiskScore       int                   `json:"risk_score"`
	RiskLevel       string                `json:"risk_level"`
	RiskFactors     []risk.RiskFactorItem `json:"risk_factors,omitempty"`
	AuthorizationOK bool                  `json:"authorization_ok"`
	NeedBreakGlass  bool                  `json:"need_break_glass"`
	RiskEvaluation  risk.RiskEvaluation   `json:"risk_evaluation"`
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

// Evaluate 执行统一的访问控制与动态风控综合评估决策流水线 (RBAC + ABAC + Consent + Risk + Break-Glass + Audit)
func (e *AccessEngine) Evaluate(req AccessRequest) (decision *AccessDecision, err error) {
	var user model.User
	var record model.MedicalRecord

	defer func() {
		if decision != nil && req.UserID > 0 && DefaultAuditService != nil {
			source := req.Source
			if source == "" {
				source = "WEB"
			}
			ip := req.IPAddress
			if ip == "" {
				ip = "127.0.0.1"
			}
			resStr := "ALLOWED"
			if !decision.Allowed {
				resStr = "INTERCEPTED"
			}
			targetID := record.RecordNo
			if targetID == "" {
				targetID = fmt.Sprintf("REC_%d", req.RecordID)
			}
			DefaultAuditService.LogDetailed(AuditEntry{
				UserID:        req.UserID,
				OperationType: "ACCESS_EVALUATE",
				TargetType:    "RECORD",
				TargetID:      targetID,
				HospitalID:    user.HospitalID,
				Result:        resStr,
				RiskLevel:     decision.RiskLevel,
				RiskScore:     decision.RiskScore,
				Source:        source,
				Reason:        decision.Reason,
				IPAddress:     ip,
				UserAgent:     req.UserAgent,
			})
		}
	}()

	if req.UserID == 0 {
		return &AccessDecision{Allowed: false, Decision: "REJECTED", Reason: "未提供有效的调用者身份凭证"}, nil
	}

	// 动作 Action 白名单强校验
	action := strings.ToUpper(strings.TrimSpace(req.Action))
	if action == "" {
		action = "VIEW"
	}
	validActions := map[string]bool{
		"VIEW":     true,
		"READ":     true,
		"DOWNLOAD": true,
		"EXAM":     true,
		"AUDIT":    true,
	}
	if !validActions[action] {
		return &AccessDecision{Allowed: false, Decision: "REJECTED", Reason: fmt.Sprintf("非法请求动作(%s)，不在安全白名单内", req.Action)}, nil
	}

	if err := repository.DB.First(&user, req.UserID).Error; err != nil {
		return &AccessDecision{Allowed: false, Decision: "REJECTED", Reason: "未找到调用者身份信息"}, nil
	}

	// 校验请求声明角色与用户真实角色一致性，防范客户端角色伪造
	if req.Role != "" && !strings.EqualFold(req.Role, user.Role) {
		return &AccessDecision{Allowed: false, Decision: "REJECTED", Reason: "角色伪造拦截：请求声明角色与令牌认证角色不一致"}, nil
	}

	if user.Status == "DISABLED" {
		return &AccessDecision{Allowed: false, Decision: "REJECTED", Reason: "账号已被系统封禁禁用，禁止任何访问操作"}, nil
	}

	// 1. 患者端访问策略 (只允许查阅自己本人的记录，严格防御水平越权 IDOR)
	if user.Role == "patient" {
		if req.RecordID > 0 {
			var rec model.MedicalRecord
			if err := repository.DB.First(&rec, req.RecordID).Error; err != nil {
				return &AccessDecision{Allowed: false, Decision: "REJECTED", Reason: "目标病历档案不存在"}, nil
			}
			if rec.PatientID != user.ID {
				return &AccessDecision{Allowed: false, Decision: "REJECTED", Reason: "水平越权拦截(IDOR)：禁止查阅或下载非本人的就诊病历"}, nil
			}
		}
		return &AccessDecision{
			Allowed:         true,
			Decision:        "ALLOWED",
			Reason:          "患者依法享有本人健康医疗数据完整查阅权限",
			AuthorizationOK: true,
			RiskScore:       0,
			RiskLevel:       "LOW",
			RiskEvaluation:  risk.RiskEvaluation{Level: "LOW", TotalScore: 0, SuggestedAction: "DIRECT_ALLOW"},
		}, nil
	}

	// 2. 管理员与监管人员治理边界隔离策略
	if user.Role == "admin" {
		if req.RecordID > 0 || action == "VIEW" || action == "READ" || action == "DOWNLOAD" {
			return &AccessDecision{Allowed: false, Decision: "REJECTED", Reason: "系统管理权限与临床数据查阅分离：禁止系统管理员直接调阅或下载患者临床隐私明文"}, nil
		}
		return &AccessDecision{
			Allowed:        true,
			Decision:       "ALLOWED",
			Reason:         "管理员系统治理操作核验通过",
			RiskEvaluation: risk.RiskEvaluation{Level: "LOW", TotalScore: 0},
		}, nil
	}

	if user.Role == "supervisor" {
		if action == "DOWNLOAD" {
			return &AccessDecision{Allowed: false, Decision: "REJECTED", Reason: "监管合规权限控制：监管人员禁止下载临床原始诊疗附件"}, nil
		}
		// 监管员调阅详情由上层执行脱敏屏蔽
		return &AccessDecision{
			Allowed:        true,
			Decision:       "REQUIRE_SUPERVISOR_APPROVAL",
			Reason:         "监管审计治理调阅核验通过（临床数据需脱敏）",
			RiskEvaluation: risk.RiskEvaluation{Level: "LOW", TotalScore: 5},
		}, nil
	}

	// 3. 医生角色核心访问控制流水线
	if user.Role != "doctor" {
		return &AccessDecision{Allowed: false, Decision: "REJECTED", Reason: "非法角色身份，系统拒绝访问"}, nil
	}

	if user.Status == "RESTRICTED" && req.IsEmergency {
		return &AccessDecision{Allowed: false, Decision: "REJECTED", Reason: "该医生已被监管下达惩戒限制，禁止发起 Break-Glass 紧急访问"}, nil
	}

	if req.RecordID == 0 {
		return &AccessDecision{Allowed: false, Decision: "REJECTED", Reason: "未指定目标医疗记录编号"}, nil
	}

	if err := repository.DB.First(&record, req.RecordID).Error; err != nil {
		return &AccessDecision{Allowed: false, Decision: "REJECTED", Reason: "目标医疗记录不存在"}, nil
	}

	// IDOR 校验：若请求同时声明了目标 patient_id，必须与病历记录所归属患者严格一致
	if req.PatientID > 0 && record.PatientID != req.PatientID {
		return &AccessDecision{Allowed: false, Decision: "REJECTED", Reason: "数据一致性拦截(IDOR)：病历所属患者与请求患者身份不符"}, nil
	}

	// 规则 1: 经治医生对本人开具的病历直接享有调阅放行权限
	if record.DoctorID == user.ID {
		return &AccessDecision{
			Allowed:         true,
			Decision:        "ALLOWED",
			Reason:          "该病历由当前医生本人开具，依法享有直接诊疗与调阅权限，系统直接放行",
			AuthorizationOK: true,
			RiskScore:       0,
			RiskLevel:       "LOW",
			RiskEvaluation:  risk.RiskEvaluation{Level: "LOW", TotalScore: 0, SuggestedAction: "DIRECT_ALLOW"},
		}, nil
	}

	isCrossHospital := user.HospitalID != record.HospitalID
	if user.Status == "RESTRICTED" && isCrossHospital {
		return &AccessDecision{Allowed: false, Decision: "REJECTED", Reason: "该医生账号已被监管限制跨机构调阅权限"}, nil
	}

	// 规则 2: 本院科室协同与专家会诊 (ABAC 细粒度判定)
	if !isCrossHospital {
		var docDept model.Department
		if user.DepartmentID > 0 {
			_ = repository.DB.First(&docDept, user.DepartmentID)
		}
		isSameDept := user.DepartmentID == 0 || record.DepartmentName == "" ||
			record.DepartmentName == "综合门诊" || (docDept.Name != "" && docDept.Name == record.DepartmentName)

		if isSameDept {
			return &AccessDecision{
				Allowed:         true,
				Decision:        "ALLOWED",
				Reason:          "病历归属当前医生所属医院同科室诊疗协同，系统核验放行",
				AuthorizationOK: true,
				RiskScore:       5,
				RiskLevel:       "LOW",
				RiskEvaluation:  risk.RiskEvaluation{Level: "LOW", TotalScore: 5, SuggestedAction: "DIRECT_ALLOW"},
			}, nil
		}
	}

	// 规则 3: 患者在线显式授权策略核查
	now := time.Now()
	var auths []model.Authorization
	repository.DB.Where("patient_id = ? AND status = 'ACTIVE' AND start_time <= ? AND end_time >= ?", record.PatientID, now, now).Find(&auths)

	hasConsent := false
	isAllDoctorsConsent := false

	var singleAuth *model.Authorization
	for _, a := range auths {
		if a.ScopeType == "SINGLE" && a.RecordID == req.RecordID {
			copyA := a
			singleAuth = &copyA
			break
		}
	}

	if singleAuth != nil {
		if singleAuth.AuthTargetType != "PRIVATE" {
			if singleAuth.AuthTargetType == "ALL_DOCTORS" {
				hasConsent = true
				isAllDoctorsConsent = true
			} else if singleAuth.AuthTargetType == "DOCTOR" && singleAuth.AuthTargetID == user.ID {
				hasConsent = true
			} else if singleAuth.AuthTargetType == "HOSPITAL" && singleAuth.AuthTargetID == user.HospitalID {
				hasConsent = true
			}
		}
	} else {
		for _, a := range auths {
			if a.ScopeType == "ALL" {
				if a.AuthTargetType == "ALL_DOCTORS" {
					hasConsent = true
					isAllDoctorsConsent = true
					break
				} else if a.AuthTargetType == "DOCTOR" && a.AuthTargetID == user.ID {
					hasConsent = true
					break
				} else if a.AuthTargetType == "HOSPITAL" && a.AuthTargetID == user.HospitalID {
					hasConsent = true
					break
				}
			}
		}
	}

	// 同院跨科室专家会诊高级职称放行
	if !isCrossHospital && !hasConsent {
		isSenior := user.Title == "主任医师" || user.Title == "副主任医师"
		if isSenior {
			return &AccessDecision{
				Allowed:         true,
				Decision:        "ALLOWED",
				Reason:          "同院跨科室专家会诊协同调阅，经高级职称属性（ABAC）核验通过，系统放行",
				AuthorizationOK: true,
				RiskScore:       15,
				RiskLevel:       "LOW",
				RiskEvaluation:  risk.RiskEvaluation{Level: "LOW", TotalScore: 15, SuggestedAction: "DIRECT_ALLOW"},
			}, nil
		}
	}

	// 规则 4: 动态多因子风控引擎判定
	eval := e.riskEngine.Evaluate(user.ID, record.PatientID, isCrossHospital, hasConsent, req.IsEmergency)

	// 分支 A: 具备患者有效授权
	if hasConsent {
		reason := "常规授权有效，动态风险评估为低风险，系统直接放行"
		if isAllDoctorsConsent {
			reason = "患者已将该病历设置为向全体执业医生公开可见，系统核验放行"
		}
		if eval.Level == "LOW" {
			return &AccessDecision{
				Allowed:         true,
				Decision:        "ALLOWED",
				Reason:          reason,
				RiskScore:       eval.TotalScore,
				RiskLevel:       eval.Level,
				RiskFactors:     eval.Factors,
				AuthorizationOK: true,
				RiskEvaluation:  eval,
			}, nil
		} else if eval.Level == "MEDIUM" {
			return &AccessDecision{
				Allowed:         false,
				Decision:        "PENDING_CONFIRM",
				Reason:          "检测到中风险操作，等待患者在线确认二次授权",
				RiskScore:       eval.TotalScore,
				RiskLevel:       eval.Level,
				RiskFactors:     eval.Factors,
				AuthorizationOK: true,
				RiskEvaluation:  eval,
			}, nil
		} else {
			return &AccessDecision{
				Allowed:         false,
				Decision:        "REJECTED",
				Reason:          "高风险异常调阅，系统安全规则拦截阻断",
				RiskScore:       eval.TotalScore,
				RiskLevel:       eval.Level,
				RiskFactors:     eval.Factors,
				AuthorizationOK: true,
				RiskEvaluation:  eval,
			}, nil
		}
	}

	// 分支 B: 未取得患者授权，检查过去 24 小时紧急救治随访有效放行
	var pastEmg model.EmergencyAccessEvent
	if err := repository.DB.Where("doctor_id = ? AND record_id = ? AND created_at >= ? AND audit_status != 'CLOSED_VIOLATION' AND patient_feedback != 'OBJECTED'",
		user.ID, req.RecordID, now.Add(-24*time.Hour)).First(&pastEmg).Error; err == nil {
		return &AccessDecision{
			Allowed:         true,
			IsEmergency:     true,
			Decision:        "ALLOWED",
			Reason:          "已在紧急破窗 24 小时急救与随诊有效期内核准放行 (无患者异议与监管违规)，免重复申请直接放行",
			RiskScore:       10,
			RiskLevel:       "LOW",
			AuthorizationOK: false,
			RiskEvaluation:  risk.RiskEvaluation{Level: "LOW", TotalScore: 10, SuggestedAction: "DIRECT_ALLOW"},
		}, nil
	}

	if !req.IsEmergency {
		return &AccessDecision{
			Allowed:         false,
			IsEmergency:     false,
			Decision:        "NEED_BREAK_GLASS",
			Reason:          "未取得患者在线授权，若处于急诊抢救或危急场景可申请 Break-Glass 紧急访问",
			RiskScore:       eval.TotalScore,
			RiskLevel:       eval.Level,
			RiskFactors:     eval.Factors,
			NeedBreakGlass:  true,
			AuthorizationOK: false,
			RiskEvaluation:  eval,
		}, nil
	}

	// 分支 C: 触发 Break-Glass 抢救放行
	return &AccessDecision{
		Allowed:         true,
		IsEmergency:     true,
		Decision:        "BREAK_GLASS_ALLOWED",
		Reason:          "临床急救访问责任声明核验通过，临时放行并生成不可篡改紧急事件单",
		RiskScore:       eval.TotalScore,
		RiskLevel:       eval.Level,
		RiskFactors:     eval.Factors,
		NeedBreakGlass:  false,
		AuthorizationOK: false,
		RiskEvaluation:  eval,
	}, nil
}

// EvaluateAccess 执行统一的权限与紧急访问评估流水线 (兼容旧版调用入口)
func (e *AccessEngine) EvaluateAccess(doctorID, patientID, recordID uint64, isEmergency bool) (*AccessDecision, error) {
	return e.Evaluate(AccessRequest{
		UserID:      doctorID,
		Role:        "doctor",
		PatientID:   patientID,
		RecordID:    recordID,
		Action:      "VIEW",
		IsEmergency: isEmergency,
	})
}

// RecordSuccessfulAccess 记录一次真实且成功的病历调阅行为至时间窗口
func (e *AccessEngine) RecordSuccessfulAccess(doctorID, patientID uint64) {
	if e.riskEngine != nil {
		e.riskEngine.RecordSuccessfulAccess(doctorID, patientID)
	}
}
