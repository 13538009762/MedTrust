package risk

import (
	"testing"
	"time"
)

type mockHistoryProvider struct {
	recentCount      int
	distinctPatients int
	violations       int
	doctorStatus     string
}

func (m *mockHistoryProvider) GetRecentAccessCount(doctorID, patientID uint64, duration time.Duration) int {
	return m.recentCount
}

func (m *mockHistoryProvider) GetDistinctPatientsAccessed(doctorID uint64, duration time.Duration) int {
	return m.distinctPatients
}

func (m *mockHistoryProvider) GetHistoricalViolationCount(doctorID uint64, days int) int {
	return m.violations
}

func (m *mockHistoryProvider) GetDoctorStatus(doctorID uint64) string {
	if m.doctorStatus == "" {
		return "NORMAL"
	}
	return m.doctorStatus
}

func TestEvaluate_EmergencyBreakGlass(t *testing.T) {
	engine := NewRiskEngine(30, 60)
	eval := engine.Evaluate(1, 100, true, false, true)

	if eval.Level != "EMERGENCY" {
		t.Fatalf("预期紧急访问 Level 为 EMERGENCY，实际为: %s", eval.Level)
	}
	if eval.Strategy != "BREAK_GLASS_EXEMPTION" {
		t.Fatalf("预期策略为 BREAK_GLASS_EXEMPTION，实际为: %s", eval.Strategy)
	}
	if eval.SuggestedAction != "NEED_BREAK_GLASS" {
		t.Fatalf("预期建议动作为 NEED_BREAK_GLASS，实际为: %s", eval.SuggestedAction)
	}
	if eval.TotalScore != 0 {
		t.Fatalf("预期急诊免评分 TotalScore=0，实际为: %d", eval.TotalScore)
	}
}

func TestEvaluate_BaselineLowRisk(t *testing.T) {
	engine := NewRiskEngine(30, 60)
	// 本院、有授权、非紧急
	eval := engine.Evaluate(1, 100, false, true, false)

	// 非工作时间可能 +20，但在无其他扣分下 score 仍然 <= 20 < 30 (LOW)
	if eval.TotalScore >= 30 {
		t.Fatalf("基准低风险评分超出预期: %d", eval.TotalScore)
	}
	if eval.Level != "LOW" {
		t.Fatalf("预期等级为 LOW，实际为: %s", eval.Level)
	}
	if eval.SuggestedAction != "DIRECT_ALLOW" {
		t.Fatalf("预期动作为 DIRECT_ALLOW，实际为: %s", eval.SuggestedAction)
	}
}

func TestEvaluate_CrossHospitalAndNoConsent(t *testing.T) {
	engine := NewRiskEngine(30, 60)
	// 跨院 (+15), 无授权 (+30) -> 基础分至少 45 (MEDIUM 或 HIGH)
	eval := engine.Evaluate(1, 100, true, false, false)

	if eval.TotalScore < 45 {
		t.Fatalf("预期跨院且无授权总分至少 45 分，实际为: %d", eval.TotalScore)
	}
	if eval.Level == "LOW" {
		t.Fatalf("预期非低风险，实际为: %s", eval.Level)
	}
	if eval.SuggestedAction != "NEED_PATIENT_CONFIRM" && eval.SuggestedAction != "REJECT_AND_ALARM" {
		t.Fatalf("预期动作为 NEED_PATIENT_CONFIRM 或 REJECT_AND_ALARM，实际为: %s", eval.SuggestedAction)
	}
}

func TestEvaluate_HighFrequencyRULE_F1(t *testing.T) {
	engine := NewRiskEngine(30, 60)
	doctorID := uint64(10)
	patientID := uint64(20)

	// 模拟 5 分钟内连续 3 次调阅
	engine.RecordSuccessfulAccess(doctorID, patientID)
	engine.RecordSuccessfulAccess(doctorID, patientID)
	engine.RecordSuccessfulAccess(doctorID, patientID)

	eval := engine.Evaluate(doctorID, patientID, false, true, false)
	foundF1 := false
	for _, f := range eval.Factors {
		if f.RuleID == "RULE-F1" && f.Triggered {
			foundF1 = true
			if f.Score != 25 {
				t.Fatalf("RULE-F1 评分不符，预期 25，实际: %d", f.Score)
			}
			break
		}
	}
	if !foundF1 {
		t.Fatalf("预期触发 RULE-F1 高频规则，但未触发")
	}
}

func TestEvaluate_DistinctPatientsBurstRULE_F2(t *testing.T) {
	engine := NewRiskEngine(30, 60)
	doctorID := uint64(10)

	// 模拟调阅 5 个不同患者
	for p := uint64(1); p <= 5; p++ {
		engine.RecordSuccessfulAccess(doctorID, p)
	}

	eval := engine.Evaluate(doctorID, 1, false, true, false)
	foundF2 := false
	for _, f := range eval.Factors {
		if f.RuleID == "RULE-F2" && f.Triggered {
			foundF2 = true
			if f.Score != 20 {
				t.Fatalf("RULE-F2 评分不符，预期 20，实际: %d", f.Score)
			}
			break
		}
	}
	if !foundF2 {
		t.Fatalf("预期触发 RULE-F2 批量多患者调阅规则，但未触发")
	}
}

func TestEvaluate_PersistentHistoryProviderRules(t *testing.T) {
	engine := NewRiskEngine(30, 60)
	mock := &mockHistoryProvider{
		violations:   2,
		doctorStatus: "RESTRICTED",
	}
	engine.SetHistoryProvider(mock)

	eval := engine.Evaluate(10, 20, false, true, false)

	foundH1 := false
	foundS1 := false
	for _, f := range eval.Factors {
		if f.RuleID == "RULE-H1" && f.Triggered {
			foundH1 = true
			if f.Score != 35 {
				t.Fatalf("RULE-H1 评分不符: got %d, want 35", f.Score)
			}
		}
		if f.RuleID == "RULE-S1" && f.Triggered {
			foundS1 = true
			if f.Score != 30 {
				t.Fatalf("RULE-S1 评分不符: got %d, want 30", f.Score)
			}
		}
	}

	if !foundH1 {
		t.Fatalf("预期持久化历史违规触发 RULE-H1，但未触发")
	}
	if !foundS1 {
		t.Fatalf("预期医生受限状态触发 RULE-S1，但未触发")
	}

	// 历史违规(35) + 受限状态(30) = 65 >= 60 (HIGH)
	if eval.Level != "HIGH" {
		t.Fatalf("预期风险等级为 HIGH，实际为: %s", eval.Level)
	}
	if eval.SuggestedAction != "REJECT_AND_ALARM" {
		t.Fatalf("预期建议动作为 REJECT_AND_ALARM，实际为: %s", eval.SuggestedAction)
	}
}

func TestEvaluate_ClampingMaxScore(t *testing.T) {
	engine := NewRiskEngine(30, 60)
	mock := &mockHistoryProvider{
		violations:   3,
		doctorStatus: "RESTRICTED",
	}
	engine.SetHistoryProvider(mock)

	// 跨院(+15) + 无授权(+30) + 历史违规(+35) + 受限(+30) > 100
	eval := engine.Evaluate(10, 20, true, false, false)
	if eval.TotalScore != 100 {
		t.Fatalf("风险总分未做 100 分封顶截断: %d", eval.TotalScore)
	}
	if eval.Level != "HIGH" {
		t.Fatalf("满分风险等级应为 HIGH")
	}
}

func BenchmarkRiskEvaluate(b *testing.B) {
	engine := NewRiskEngine(30, 60)
	for i := uint64(1); i <= 10; i++ {
		engine.RecordSuccessfulAccess(1, i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.Evaluate(1, 2, true, false, false)
	}
}
