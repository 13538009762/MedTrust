package service

import (
	"strings"
	"testing"
	"time"

	"medtrust-backend/model"
)

func TestAIDiagnostic_RuleAndSemanticExtraction(t *testing.T) {
	s := GetAIDiagnosticService()

	// 1. 测试从非结构化病历中抽取隐匿过敏线索
	testRecords := []model.MedicalRecord{
		{
			RecordNo:       "REC2026_TEST_001",
			HospitalName:   "上海市第一人民医院",
			ChiefComplaint: "主诉反复咳嗽，既往服用消炎药后有皮疹伴瘙痒，自行停药",
			Symptoms:       "发热伴红斑",
			Diagnosis:      "上呼吸道感染",
			TreatmentPlan:  "抗感冒治疗",
			CreatedAt:      time.Now().Add(-10 * 24 * time.Hour),
		},
		{
			RecordNo:       "REC2026_TEST_002",
			HospitalName:   "华山医院",
			ChiefComplaint: "中上腹隐痛，伴黑便可疑",
			Symptoms:       "胃溃疡复发",
			Diagnosis:      "慢性胃溃疡活动期",
			TreatmentPlan:  "奥美拉唑抑酸治疗",
			CreatedAt:      time.Now().Add(-5 * 24 * time.Hour),
		},
	}

	req := model.DiagnosticAssistRequest{
		PatientID:       1,
		CurrentSymptoms: "发热头痛",
		PresentIllness:  "今日起病",
		VitalSigns:      "BP 120/80mmHg, HR 78bpm",
		DepartmentName:  "全科门诊",
	}

	clues := s.extractHiddenAllergyClues(testRecords, req)
	if len(clues) == 0 {
		t.Fatalf("预期检出隐匿过敏或溃疡线索，但结果为空")
	}

	hasAllergyClue := false
	hasBleedClue := false
	for _, c := range clues {
		if c.ClueType == "疑似药物过敏史" {
			hasAllergyClue = true
		}
		if c.ClueType == "消化道出血与溃疡高危史" {
			hasBleedClue = true
		}
	}
	if !hasAllergyClue {
		t.Errorf("未能成功提取【疑似药物过敏史】隐匿线索: %+v", clues)
	}
	if !hasBleedClue {
		t.Errorf("未能成功提取【消化道出血与溃疡高危史】隐匿线索: %+v", clues)
	}

	// 2. 测试 SOAP 接诊结构化摘要生成
	soap := s.generateSOAPSummary(req, model.User{RealName: "测试患者"}, testRecords, nil)
	if soap.Subjective == "" || soap.Objective == "" || soap.Assessment == "" || soap.Plan == "" {
		t.Errorf("SOAP 摘要四段式未完整填充: %+v", soap)
	}

	// 3. 测试跨院语义矛盾探测
	contradictionRecords := []model.MedicalRecord{
		{
			RecordNo:       "REC_A",
			HospitalName:   "医院A",
			ChiefComplaint: "主诉头晕，否认高血压病史，否认药物过敏",
		},
		{
			RecordNo:       "REC_B",
			HospitalName:   "医院B",
			ChiefComplaint: "高血压3级极高危，服降压药多年，曾有青霉素过敏起红疹",
			TreatmentPlan:  "硝苯地平控释片 30mg qd",
		},
	}
	contradictions := s.detectSemanticContradictions(contradictionRecords, req)
	if len(contradictions) < 2 {
		t.Errorf("预期检出慢病自述矛盾与过敏史差异，实际检出数: %d", len(contradictions))
	}
}

func TestEmergencyService_RulePreScreenAndAuditSummary(t *testing.T) {
	emg := DefaultEmergencyService

	// 1. 测试危急重症初筛放行 (RULE_PASS_CRITICAL)
	criticalEvent := &model.EmergencyAccessEvent{
		EmergencyReason: "COMA_RESCUE",
		Description:     "患者突发意识丧失心搏骤停，紧急开辟抢救绿色通道",
		DoctorID:        1,
		PatientFeedback: "PENDING",
		CreatedAt:       time.Now(),
	}
	code, desc := emg.EvaluateRulePreScreen(criticalEvent)
	if code != "RULE_PASS_CRITICAL" {
		t.Errorf("危急重症抢救预期为 RULE_PASS_CRITICAL，实际得到: %s (%s)", code, desc)
	}

	// 2. 测试存疑与异议初筛预警 (RULE_WARN_SUSPICIOUS)
	objectedEvent := &model.EmergencyAccessEvent{
		EmergencyReason: "OTHER",
		Description:     "调阅既往病历",
		DoctorID:        1,
		PatientFeedback: "OBJECTED",
		PatientComment:  "我当天根本没有去过该医院急诊，纯属医生违规盗看！",
		CreatedAt:       time.Now(),
	}
	warnCode, warnDesc := emg.EvaluateRulePreScreen(objectedEvent)
	if warnCode != "RULE_WARN_SUSPICIOUS" {
		t.Errorf("患者提出异议投诉预期为 RULE_WARN_SUSPICIOUS，实际得到: %s (%s)", warnCode, warnDesc)
	}

	// 3. 测试 AI 医患陈述争议对比卡生成
	summary := emg.GenerateAuditComparisonSummary(objectedEvent)
	if summary == "" {
		t.Fatalf("预期生成争议焦点对比摘要，实际为空")
	}
	if !testing.Short() {
		t.Logf("生成的医患陈述对比摘要:\n%s", summary)
	}
}

// TestAIDiagnostic_Fix4_NoHallucinatedLabsAndPIISanitized 回归测试 Fix 4：杜绝伪造心电图与人口学推测，严格脱敏 Prompt PII
func TestAIDiagnostic_Fix4_NoHallucinatedLabsAndPIISanitized(t *testing.T) {
	s := GetAIDiagnosticService()

	// 1. 测试当化验项为空时，SOAP Objective 段如实说明，绝不伪造心电缺血结论
	req := model.DiagnosticAssistRequest{
		PatientID:       100,
		CurrentSymptoms: "轻微咳嗽",
		PresentIllness:  "起病1天",
		VitalSigns:      "BP 115/75mmHg, HR 72bpm",
	}
	soap := s.generateSOAPSummary(req, model.User{RealName: "张三"}, nil, []model.LabIndicatorItem{
		{
			ItemName:     "检验/检查",
			HistoryValue: "本次授权范围内未见异常检验/检查结果记录",
			Status:       "NORMAL",
		},
	})
	for _, forbidden := range []string{"12导联心电图", "V4-V6", "T波低平", "轻度缺血改变"} {
		if strings.Contains(soap.Objective, forbidden) {
			t.Fatalf("【真实性违规】无异常化验单时仍伪造了心电图指标: %s", forbidden)
		}
	}
	if !strings.Contains(soap.Objective, "未见异常或危急值记录") {
		t.Errorf("SOAP Objective 段未如实说明无异常化验: %s", soap.Objective)
	}

	// 2. 测试 buildDiagnosticPrompt 隐私脱敏与无性别年龄推测
	patUnknown := model.User{
		RealName: "李晓明",
		UserNo:   "P20261102",
		Gender:   "", // 未登记性别
		Age:      0,  // 未登记年龄
	}
	prompt := s.buildDiagnosticPrompt(patUnknown, nil, nil, req, nil)

	// 真实明文姓名与卡号不得出现
	if strings.Contains(prompt, "李晓明") {
		t.Fatalf("【PII 泄露】患者真实明文姓名出现在 Prompt 中")
	}
	if strings.Contains(prompt, "P20261102") {
		t.Fatalf("【PII 泄露】患者全量病历号出现在 Prompt 中")
	}
	// 掩码内容必须存在
	if !strings.Contains(prompt, "李**") {
		t.Errorf("Prompt 未正确包含脱敏后的姓名")
	}
	if !strings.Contains(prompt, "P2****02") {
		t.Errorf("Prompt 未正确包含脱敏后的病历号")
	}
	// 缺失的性别与年龄不得被猜测为 男/45
	if strings.Contains(prompt, "45 岁") {
		t.Fatalf("【数据伪造】未登记年龄被静默猜测为 45 岁")
	}
	if !strings.Contains(prompt, "未登记") {
		t.Errorf("未登记的性别或年龄未显示未登记标识")
	}

	// 3. 针对 mask 函数的边界用例测试
	if got := maskRealName("王"); got != "王*" {
		t.Errorf("单字名脱敏失败: got %s, want 王*", got)
	}
	if got := maskRealName("诸葛孔明"); got != "诸***" {
		t.Errorf("四字名脱敏失败: got %s, want 诸***", got)
	}
	if got := maskUserNo("P12"); got != "****" {
		t.Errorf("超短号脱敏失败: got %s, want ****", got)
	}
}

// TestAIDiagnostic_NegationHandlingNoSelfContradiction 验证否定词（否认高血压/无药物过敏/青霉素皮试阴性）不会在同一份病历内触发自相矛盾或误报过敏
func TestAIDiagnostic_NegationHandlingNoSelfContradiction(t *testing.T) {
	s := GetAIDiagnosticService()
	negatedOnly := []model.MedicalRecord{
		{
			RecordNo:       "REC_NEG_001",
			HospitalName:   "测试医院",
			ChiefComplaint: "主诉间断头痛3天，否认高血压病史，否认药物过敏",
			Symptoms:       "无皮疹，头晕，青霉素皮试阴性，不过敏",
			TreatmentPlan:  "建议休息复查",
		},
	}
	req := model.DiagnosticAssistRequest{
		PatientID:       999,
		CurrentSymptoms: "轻微头痛，无药物过敏",
		PresentIllness:  "无黑便，否认胃溃疡及晕厥史",
		DepartmentName:  "神经内科",
	}

	contradictions := s.detectSemanticContradictions(negatedOnly, req)
	for _, c := range contradictions {
		if c.ContradictionType != "跨院陈述一致性核验正常" {
			t.Fatalf("纯否定语句不应引发自相矛盾告警: %+v", c)
		}
	}

	clues := s.extractHiddenAllergyClues(negatedOnly, req)
	for _, cl := range clues {
		if cl.ClueType != "过敏与高危线索排查正常" {
			t.Fatalf("纯否定陈述不应误抽出过敏/溃疡/晕厥线索: %+v", cl)
		}
	}
}
