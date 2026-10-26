package service

import (
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
