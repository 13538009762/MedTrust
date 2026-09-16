package risk

import (
	"fmt"
	"sync"
	"time"
)

type AccessRecordItem struct {
	DoctorID  uint64
	PatientID uint64
	Time      time.Time
}

// AccessHistoryProvider 访问历史持久化提供接口 (支持跨重启持久化查询)
type AccessHistoryProvider interface {
	GetRecentAccessCount(doctorID, patientID uint64, duration time.Duration) int
	GetDistinctPatientsAccessed(doctorID uint64, duration time.Duration) int
	GetHistoricalViolationCount(doctorID uint64, days int) int
	GetDoctorStatus(doctorID uint64) string
}

type RiskEngine struct {
	lowThreshold    int
	highThreshold   int
	accessWindow    []AccessRecordItem
	historyProvider AccessHistoryProvider
	mu              sync.Mutex
}

func NewRiskEngine(low, high int) *RiskEngine {
	if low <= 0 {
		low = 30
	}
	if high <= 0 {
		high = 60
	}
	return &RiskEngine{
		lowThreshold:  low,
		highThreshold: high,
		accessWindow:  make([]AccessRecordItem, 0),
	}
}

func (e *RiskEngine) SetHistoryProvider(p AccessHistoryProvider) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.historyProvider = p
}

// RiskFactorItem 结构化风控评估规则因子细分
type RiskFactorItem struct {
	RuleID      string `json:"rule_id"`
	FactorName  string `json:"factor_name"`
	Weight      int    `json:"weight"`
	Score       int    `json:"score"`
	Triggered   bool   `json:"triggered"`
	Description string `json:"description"`
}

// RiskEvaluation 结构化风控评估结果 (基于规则加权引擎，杜绝黑盒虚假模型)
type RiskEvaluation struct {
	TotalScore      int              `json:"total_score"`
	Level           string           `json:"level"`
	Strategy        string           `json:"strategy"`         // "RULE_ENGINE_WEIGHTED" 或 "BREAK_GLASS_EXEMPTION"
	SuggestedAction string           `json:"suggested_action"` // "DIRECT_ALLOW", "NEED_PATIENT_CONFIRM", "NEED_BREAK_GLASS", "REJECT_AND_ALARM"
	ActionDesc      string           `json:"action_desc"`      // 建议处理动作详细说明
	RulesFired      []string         `json:"rules_fired"`
	Factors         []RiskFactorItem `json:"factors"`
}

func (e *RiskEngine) Evaluate(doctorID, patientID uint64, isCrossHospital, hasConsent, isEmergency bool) RiskEvaluation {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-10 * time.Minute)
	valid := make([]AccessRecordItem, 0)
	for _, item := range e.accessWindow {
		if item.Time.After(cutoff) {
			valid = append(valid, item)
		}
	}
	e.accessWindow = valid

	// 紧急访问（Break-Glass）走专用绿色通道，豁免常规风险拦截阻断
	if isEmergency {
		factors := []RiskFactorItem{
			{
				RuleID:      "RULE-E1",
				FactorName:  "急诊危重抢救绿色通道",
				Weight:      0,
				Score:       0,
				Triggered:   true,
				Description: "触发 Break-Glass 紧急救治专属通道，医生签署法律声明，豁免常规加权评分阻断",
			},
		}
		return RiskEvaluation{
			TotalScore:      0,
			Level:           "EMERGENCY",
			Strategy:        "BREAK_GLASS_EXEMPTION",
			SuggestedAction: "NEED_BREAK_GLASS",
			ActionDesc:      "触发 Break-Glass 临床抢救专属通道，签署法律免责声明后放行，实施全流程审计留痕",
			RulesFired:      []string{"RULE-E1: 触发 Break-Glass 临床抢救专属放行通道 (豁免常规评分阻断)"},
			Factors:         factors,
		}
	}

	score := 0
	rules := make([]string, 0)
	factors := make([]RiskFactorItem, 0)

	// 1. RULE-D1: 跨医疗机构调阅行为
	d1Triggered := isCrossHospital
	d1Score := 0
	if d1Triggered {
		d1Score = 15
		score += d1Score
		rules = append(rules, "RULE-D1: 跨医疗机构调阅 (+15)")
	}
	factors = append(factors, RiskFactorItem{
		RuleID:      "RULE-D1",
		FactorName:  "跨医疗机构调阅行为",
		Weight:      15,
		Score:       d1Score,
		Triggered:   d1Triggered,
		Description: func() string {
			if d1Triggered {
				return "调阅对象归属外部医疗机构，触发跨组织数据协同风控基线"
			}
			return "属于本医疗机构内部就诊记录，未触发跨院风险加分"
		}(),
	})

	// 2. RULE-A1: 患者在线授权状态
	a1Triggered := !hasConsent
	a1Score := 0
	if a1Triggered {
		a1Score = 30
		score += a1Score
		rules = append(rules, "RULE-A1: 患者未在线显式授权 (+30)")
	}
	factors = append(factors, RiskFactorItem{
		RuleID:      "RULE-A1",
		FactorName:  "患者知情同意授权状态",
		Weight:      30,
		Score:       a1Score,
		Triggered:   a1Triggered,
		Description: func() string {
			if a1Triggered {
				return "患者尚未在区块链建立生效中的知情授权凭证，存在隐私越权合规风险"
			}
			return "患者已在链上颁发有效授权策略，凭证合法有效"
		}(),
	})

	// 3. RULE-T1: 访问时间窗口合规性
	hour := now.Hour()
	t1Triggered := (hour < 8 || hour >= 18)
	t1Score := 0
	if t1Triggered {
		t1Score = 20
		score += t1Score
		rules = append(rules, "RULE-T1: 非工作时间调阅 (+20)")
	}
	factors = append(factors, RiskFactorItem{
		RuleID:      "RULE-T1",
		FactorName:  "访问时间窗口合规性",
		Weight:      20,
		Score:       t1Score,
		Triggered:   t1Triggered,
		Description: func() string {
			if t1Triggered {
				return fmt.Sprintf("当前时间 %02d:%02d 处于常规门诊工作时段(08:00-18:00)之外，触发夜间防泄露监控", hour, now.Minute())
			}
			return fmt.Sprintf("当前时间 %02d:%02d 属于正常门诊执业时段，符合常规业务基线", hour, now.Minute())
		}(),
	})

	// 4. RULE-F1: 5分钟窗口同患者高频调阅
	fiveMinCutoff := now.Add(-5 * time.Minute)
	samePatientCount := 0
	for _, item := range e.accessWindow {
		if item.DoctorID == doctorID && item.PatientID == patientID && item.Time.After(fiveMinCutoff) {
			samePatientCount++
		}
	}
	if e.historyProvider != nil {
		dbCount := e.historyProvider.GetRecentAccessCount(doctorID, patientID, 5*time.Minute)
		if dbCount > samePatientCount {
			samePatientCount = dbCount
		}
	}
	f1Triggered := (samePatientCount >= 3)
	f1Score := 0
	if f1Triggered {
		f1Score = 25
		score += f1Score
		rules = append(rules, "RULE-F1: 5分钟窗口高频调阅 (+25)")
	}
	factors = append(factors, RiskFactorItem{
		RuleID:      "RULE-F1",
		FactorName:  "同患者短周期高频调阅",
		Weight:      25,
		Score:       f1Score,
		Triggered:   f1Triggered,
		Description: func() string {
			if f1Triggered {
				return fmt.Sprintf("5分钟内针对该患者已发起 %d 次调阅，疑似爬取或脚本异常访问", samePatientCount)
			}
			return fmt.Sprintf("5分钟内调阅频次 %d 次，处于正常频率阈值以内", samePatientCount)
		}(),
	})

	// 5. RULE-F2: 短期批量跨患者调阅
	distinctPatients := make(map[uint64]struct{})
	for _, item := range e.accessWindow {
		if item.DoctorID == doctorID {
			distinctPatients[item.PatientID] = struct{}{}
		}
	}
	distinctCount := len(distinctPatients)
	if e.historyProvider != nil {
		dbDistinct := e.historyProvider.GetDistinctPatientsAccessed(doctorID, 10*time.Minute)
		if dbDistinct > distinctCount {
			distinctCount = dbDistinct
		}
	}
	f2Triggered := (distinctCount >= 5)
	f2Score := 0
	if f2Triggered {
		f2Score = 20
		score += f2Score
		rules = append(rules, "RULE-F2: 批量跨患者调阅 (+20)")
	}
	factors = append(factors, RiskFactorItem{
		RuleID:      "RULE-F2",
		FactorName:  "短周期批量多患者调阅",
		Weight:      20,
		Score:       f2Score,
		Triggered:   f2Triggered,
		Description: func() string {
			if f2Triggered {
				return fmt.Sprintf("10分钟滑动窗口内累计调阅 %d 位不同患者档案，触发批量泄露保护预警", distinctCount)
			}
			return fmt.Sprintf("10分钟滑动窗口内调阅 %d 位患者档案，行为指标正常", distinctCount)
		}(),
	})

	// 6. RULE-H1: 历史违规/异常访问追溯
	h1Triggered := false
	h1Score := 0
	if e.historyProvider != nil {
		violations := e.historyProvider.GetHistoricalViolationCount(doctorID, 30)
		if violations > 0 {
			h1Triggered = true
			h1Score = 35
			score += h1Score
			rules = append(rules, fmt.Sprintf("RULE-H1: 30天内存在历史违规记录(%d次) (+35)", violations))
		}
	}
	factors = append(factors, RiskFactorItem{
		RuleID:      "RULE-H1",
		FactorName:  "历史违规行为追溯",
		Weight:      35,
		Score:       h1Score,
		Triggered:   h1Triggered,
		Description: func() string {
			if h1Triggered {
				return "调阅医生在过去30天内存在被监管裁定违规的紧急调阅记录，提高风险权值"
			}
			return "调阅医生近30天无违规通报记录，合规信用良好"
		}(),
	})

	// 7. RULE-S1: 医生账号状态异常判定
	s1Triggered := false
	s1Score := 0
	if e.historyProvider != nil {
		docStatus := e.historyProvider.GetDoctorStatus(doctorID)
		if docStatus == "RESTRICTED" {
			s1Triggered = true
			s1Score = 30
			score += s1Score
			rules = append(rules, "RULE-S1: 医生账号处于监管受限状态 (+30)")
		}
	}
	factors = append(factors, RiskFactorItem{
		RuleID:      "RULE-S1",
		FactorName:  "医生资质与账号监管状态",
		Weight:      30,
		Score:       s1Score,
		Triggered:   s1Triggered,
		Description: func() string {
			if s1Triggered {
				return "调阅医生账号当前处于监管受限状态(RESTRICTED)，触发高权值防范"
			}
			return "调阅医生执业资格与系统账号状态正常"
		}(),
	})

	if score > 100 {
		score = 100
	}

	level := "LOW"
	suggestedAction := "DIRECT_ALLOW"
	actionDesc := "多因子风控综合评分较低，处于正常业务基线内，允许直接调阅"

	if score >= e.highThreshold {
		level = "HIGH"
		suggestedAction = "REJECT_AND_ALARM"
		actionDesc = "高危风险访问行为：触发多项关键安全规则或批量访问特征，系统实施强制阻断并上报安全预警"
	} else if score >= e.lowThreshold {
		level = "MEDIUM"
		suggestedAction = "NEED_PATIENT_CONFIRM"
		actionDesc = "触发中风险预警，需引导患者在客户端进行即时二次确认或在线颁发授权凭证"
	}

	return RiskEvaluation{
		TotalScore:      score,
		Level:           level,
		Strategy:        "RULE_ENGINE_WEIGHTED",
		SuggestedAction: suggestedAction,
		ActionDesc:      actionDesc,
		RulesFired:      rules,
		Factors:         factors,
	}
}

// RecordSuccessfulAccess 仅在病历被真正成功授权调阅并解密呈现后记录，杜绝重复刷新或被拦截请求污染频率窗口
func (e *RiskEngine) RecordSuccessfulAccess(doctorID, patientID uint64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-10 * time.Minute)
	valid := make([]AccessRecordItem, 0)
	for _, item := range e.accessWindow {
		if item.Time.After(cutoff) {
			valid = append(valid, item)
		}
	}
	valid = append(valid, AccessRecordItem{DoctorID: doctorID, PatientID: patientID, Time: now})
	e.accessWindow = valid
}
