package risk

import (
	"sync"
	"time"
)

type AccessRecordItem struct {
	DoctorID  uint64
	PatientID uint64
	Time      time.Time
}

type RiskEngine struct {
	lowThreshold  int
	highThreshold int
	accessWindow  []AccessRecordItem
	mu            sync.Mutex
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

type RiskEvaluation struct {
	TotalScore int      `json:"total_score"`
	Level      string   `json:"level"`
	RulesFired []string `json:"rules_fired"`
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
	valid = append(valid, AccessRecordItem{DoctorID: doctorID, PatientID: patientID, Time: now})
	e.accessWindow = valid

	score := 0
	rules := make([]string, 0)

	hour := now.Hour()
	if hour < 8 || hour >= 18 {
		score += 20
		rules = append(rules, "RULE-T1: 非工作时间调阅 (+20)")
	}

	if isCrossHospital {
		score += 15
		rules = append(rules, "RULE-D1: 跨医疗机构调阅 (+15)")
	}

	fiveMinCutoff := now.Add(-5 * time.Minute)
	samePatientCount := 0
	for _, item := range e.accessWindow {
		if item.DoctorID == doctorID && item.PatientID == patientID && item.Time.After(fiveMinCutoff) {
			samePatientCount++
		}
	}
	if samePatientCount > 3 {
		score += 25
		rules = append(rules, "RULE-F1: 5分钟窗口高频调阅 (+25)")
	}

	distinctPatients := make(map[uint64]struct{})
	for _, item := range e.accessWindow {
		if item.DoctorID == doctorID {
			distinctPatients[item.PatientID] = struct{}{}
		}
	}
	if len(distinctPatients) >= 5 {
		score += 20
		rules = append(rules, "RULE-F2: 批量跨患者调阅 (+20)")
	}

	if !hasConsent {
		score += 30
		rules = append(rules, "RULE-A1: 患者未在线显式授权 (+30)")
	}

	if isEmergency {
		score += 40
		rules = append(rules, "RULE-E1: 触发 Break-Glass 抢救调阅 (+40)")
	}

	if score > 100 {
		score = 100
	}

	level := "LOW"
	if score >= e.highThreshold {
		level = "HIGH"
	} else if score >= e.lowThreshold {
		level = "MEDIUM"
	}

	return RiskEvaluation{
		TotalScore: score,
		Level:      level,
		RulesFired: rules,
	}
}
