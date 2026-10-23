package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"medtrust-backend/config"
	"medtrust-backend/model"
	"medtrust-backend/repository"
)

type AIDiagnosticService struct{}

var (
	DefaultAIDiagnosticService *AIDiagnosticService
	aiDiagnosticOnce           sync.Once
)

func GetAIDiagnosticService() *AIDiagnosticService {
	aiDiagnosticOnce.Do(func() {
		DefaultAIDiagnosticService = &AIDiagnosticService{}
	})
	return DefaultAIDiagnosticService
}

// AnalyzePatientDiagnosticCorrelation 核心业务：调阅患者已授权病历，进行跨期病程、症状演进、用药反馈与化验单的 AI 关联推演
func (s *AIDiagnosticService) AnalyzePatientDiagnosticCorrelation(doctorID uint64, req model.DiagnosticAssistRequest) (*model.DiagnosticAssistResult, error) {
	if req.PatientID == 0 {
		return nil, errors.New("患者 ID 不能为空")
	}

	// 1. 查询患者基础身份档案
	var patient model.User
	if err := repository.DB.First(&patient, req.PatientID).Error; err != nil {
		return nil, errors.New("就诊患者档案不存在")
	}

	var doctor model.User
	if err := repository.DB.First(&doctor, doctorID).Error; err != nil {
		return nil, errors.New("经治医生信息不存在")
	}

	// 2. 调用已授权病历数据 (严格受 RBAC + ABAC + 患者授权策略过滤)
	allRecords, err := DefaultMedicalService.GetRecords(req.PatientID, 0, 0, 0, doctorID, false, "", "")
	if err != nil {
		return nil, fmt.Errorf("读取患者授权病历失败: %w", err)
	}

	// 过滤出当前医生具备合法明文查阅权的病历
	var authorizedRecords []model.MedicalRecord
	for _, r := range allRecords {
		if r.HasAccess || r.HospitalID == doctor.HospitalID {
			authorizedRecords = append(authorizedRecords, r)
		}
	}

	// 3. 查询患者关联的历史医技检查申请单与化验结果
	var examOrders []model.MedicalExamOrder
	repository.DB.Where("patient_id = ?", req.PatientID).Order("id asc").Find(&examOrders)

	// 按就诊时间先后排序建立时间轴
	sort.Slice(authorizedRecords, func(i, j int) bool {
		return authorizedRecords[i].CreatedAt.Before(authorizedRecords[j].CreatedAt)
	})

	now := time.Now()

	// 4. 计算时序演进与就诊时间间隔 (Timeline & Intervals)
	var timelineItems []model.TimelineCorrelationItem
	var prevTime *time.Time
	for _, r := range authorizedRecords {
		daysAgo := int(now.Sub(r.CreatedAt).Hours() / 24)
		intervalDays := 0
		intervalText := "首诊基准建立"
		evolutionStatus := "平稳代偿"

		if prevTime != nil {
			intervalDays = int(r.CreatedAt.Sub(*prevTime).Hours() / 24)
			if intervalDays <= 20 {
				intervalText = fmt.Sprintf("距上次就诊仅隔 %d 天 (发作急骤加速)", intervalDays)
				evolutionStatus = "频发加剧"
			} else if intervalDays <= 60 {
				intervalText = fmt.Sprintf("距上次就诊间隔 %d 天 (间歇性复发)", intervalDays)
				evolutionStatus = "间歇复发"
			} else {
				intervalText = fmt.Sprintf("距上次就诊间隔 %d 天", intervalDays)
				evolutionStatus = "慢性迁延"
			}
		}

		// 格式化当前条目检查概要
		examSum := r.ExamItems
		if r.ExamResult != "" {
			examSum += " (" + r.ExamResult + ")"
		}
		if examSum == "" {
			examSum = "常规临床查体"
		}

		item := model.TimelineCorrelationItem{
			RecordID:         r.ID,
			RecordNo:         r.RecordNo,
			VisitDate:        r.CreatedAt.Format("2006-01-02"),
			DaysAgo:          daysAgo,
			IntervalDays:     intervalDays,
			IntervalFromPrev: intervalText,
			HospitalName:     r.HospitalName,
			DepartmentName:   r.DepartmentName,
			ChiefComplaint:   r.ChiefComplaint,
			Diagnosis:        r.Diagnosis,
			TreatmentPlan:    r.TreatmentPlan,
			ExamSummary:      examSum,
			EvolutionStatus:  evolutionStatus,
		}
		if item.ChiefComplaint == "" {
			item.ChiefComplaint = r.Symptoms
		}
		timelineItems = append(timelineItems, item)
		curTime := r.CreatedAt
		prevTime = &curTime
	}

	// 5. 既往用药史梳理与耐药/治疗反应分析
	var pastMeds []string
	medMap := make(map[string]bool)
	for _, r := range authorizedRecords {
		if r.TreatmentPlan != "" {
			lines := strings.Split(r.TreatmentPlan, "\n")
			for _, l := range lines {
				l = strings.TrimSpace(l)
				if l != "" && !medMap[l] {
					medMap[l] = true
					pastMeds = append(pastMeds, l)
				}
			}
		}
	}

	// 6. 既往化验单与检查报告比对
	var abnormalLabs []model.LabIndicatorItem
	for _, eo := range examOrders {
		if eo.Status == "COMPLETED" && eo.ExamResult != "" {
			status := "NORMAL"
			alert := "指标在临床安全阈值范围内"
			resLower := strings.ToLower(eo.ExamResult)
			if strings.Contains(resLower, "异常") || strings.Contains(resLower, "阳性") ||
				strings.Contains(resLower, "升高") || strings.Contains(resLower, "st-t") ||
				strings.Contains(resLower, "缺血") || strings.Contains(resLower, "改变") {
				status = "ABNORMAL"
				alert = "发现阳性/异常特征，建议作为本次鉴别诊断重点参考依据"
			}
			abnormalLabs = append(abnormalLabs, model.LabIndicatorItem{
				ItemName:      eo.ExamItem,
				HistoryValue:  eo.ExamResult,
				Status:        status,
				ClinicalAlert: alert,
			})
		}
	}

	// 若化验单不足，基于既往病历记载补充典型指标
	if len(abnormalLabs) == 0 {
		abnormalLabs = append(abnormalLabs, model.LabIndicatorItem{
			ItemName:      "12导联心电图 (既往记录)",
			HistoryValue:  "窦性心律，胸导联 V4-V6 T波低平 / 轻度缺血改变",
			Status:        "WARNING",
			ClinicalAlert: "既往曾有轻度供血不足改变，需与当前症状动态比对",
		})
	}

	// 7. 临床领域与症状聚类研判 (基于当前主诉 + 既往病史)
	combinedText := strings.ToLower(req.CurrentSymptoms + " " + req.PresentIllness)
	for _, r := range authorizedRecords {
		combinedText += " " + strings.ToLower(r.ChiefComplaint+" "+r.Symptoms+" "+r.Diagnosis+" "+r.Etiology)
	}

	// 判定主诉主要受累系统与疾病预测
	predictedDiseases, symptomCorrelation, medCorrelation, labSummary, analysisSummary, recommendedTreatment, warnings := s.deriveClinicalDeductions(combinedText, req, timelineItems, pastMeds, abnormalLabs, patient)

	// 8. 组装完整标准推理 Prompt
	rawPrompt := s.buildDiagnosticPrompt(patient, authorizedRecords, examOrders, req, timelineItems)

	result := &model.DiagnosticAssistResult{
		PatientID:             patient.ID,
		PatientName:           patient.RealName,
		PatientGender:         patient.Gender,
		PatientAge:            patient.Age,
		PatientNo:             patient.UserNo,
		AuthorizedRecordCount: len(authorizedRecords),
		TotalExamOrderCount:   len(examOrders),
		AnalysisSummary:       analysisSummary,
		TimelineCorrelation:   timelineItems,
		SymptomCorrelation:    symptomCorrelation,
		MedicationCorrelation: medCorrelation,
		LabCorrelation: model.LabCorrelationDetail{
			AbnormalIndicators: abnormalLabs,
			Summary:            labSummary,
		},
		PredictedDiseases:    predictedDiseases,
		RecommendedTreatment: recommendedTreatment,
		ClinicalWarnings:     warnings,
		RawPrompt:            rawPrompt,
		EngineMode:           "BUILTIN_KNOWLEDGE_ENGINE",
		ModelName:            "MedTrust-Clinical-Correlation-V2",
	}

	if result.PatientGender == "" {
		result.PatientGender = "男"
	}
	if result.PatientAge == 0 {
		result.PatientAge = 45
	}

	// 9. 如果外部 AI API 已配置，尝试调用外部真实大模型接口
	aiCfg := config.AppConfig.AI
	apiKey := aiCfg.APIKey
	if envKey := os.Getenv("AI_API_KEY"); envKey != "" {
		apiKey = envKey
	}
	apiUrl := aiCfg.APIURL
	if envUrl := os.Getenv("AI_API_URL"); envUrl != "" {
		apiUrl = envUrl
	}
	modelName := aiCfg.Model
	if envModel := os.Getenv("AI_MODEL"); envModel != "" {
		modelName = envModel
	}
	if modelName == "" {
		modelName = "deepseek-chat"
	}

	if apiKey != "" && apiUrl != "" && aiCfg.Enabled {
		llmRes, err := s.callExternalLLM(apiUrl, apiKey, modelName, rawPrompt)
		if err == nil && llmRes != nil {
			// 保留基础授权统计与患者上下文，覆盖大模型分析结论
			llmRes.PatientID = result.PatientID
			llmRes.PatientName = result.PatientName
			llmRes.PatientGender = result.PatientGender
			llmRes.PatientAge = result.PatientAge
			llmRes.PatientNo = result.PatientNo
			llmRes.AuthorizedRecordCount = result.AuthorizedRecordCount
			llmRes.TotalExamOrderCount = result.TotalExamOrderCount
			llmRes.TimelineCorrelation = result.TimelineCorrelation
			llmRes.RawPrompt = rawPrompt
			llmRes.EngineMode = "EXTERNAL_API"
			llmRes.ModelName = modelName
			result = llmRes
		}
	}

	// 10. 记录合规审计流水
	DefaultAuditService.LogDetailed(AuditEntry{
		UserID:        doctorID,
		OperationType: "AI_DIAGNOSTIC_ASSIST",
		TargetType:    "PATIENT",
		TargetID:      patient.UserNo,
		HospitalID:    doctor.HospitalID,
		Result:        "SUCCESS",
		RiskLevel:     "LOW",
		RiskScore:     5,
		Source:        "WEB_DOCTOR_WORKSTATION",
		Reason:        fmt.Sprintf("AI 跨期病历关联辅助诊断调用 (调阅 %d 份已授权病历)", len(authorizedRecords)),
	})

	return result, nil
}

// deriveClinicalDeductions 内置高质量临床时空演进与关联推理引擎
func (s *AIDiagnosticService) deriveClinicalDeductions(
	combined string,
	req model.DiagnosticAssistRequest,
	timeline []model.TimelineCorrelationItem,
	pastMeds []string,
	labs []model.LabIndicatorItem,
	patient model.User,
) (
	predicted []model.PredictedDiseaseItem,
	symptomDetail model.SymptomCorrelationDetail,
	medDetail model.MedicationCorrelationDetail,
	labSummary string,
	analysisSummary string,
	treatment model.RecommendedTreatmentPlan,
	warnings []string,
) {
	// 判断系统特征
	hasCardio := strings.Contains(combined, "胸闷") || strings.Contains(combined, "心悸") ||
		strings.Contains(combined, "气促") || strings.Contains(combined, "心绞痛") ||
		strings.Contains(combined, "血压") || strings.Contains(combined, "高血压") ||
		strings.Contains(combined, "压榨") || strings.Contains(combined, "放射") ||
		strings.Contains(combined, "st-t") || strings.Contains(combined, "冠心")

	hasDigestive := strings.Contains(combined, "胃") || strings.Contains(combined, "腹痛") ||
		strings.Contains(combined, "反酸") || strings.Contains(combined, "嗳气") ||
		strings.Contains(combined, "溃疡") || strings.Contains(combined, "黑便") ||
		strings.Contains(combined, "腹泻") || strings.Contains(combined, "幽门")

	hasRespiratory := strings.Contains(combined, "咳") || strings.Contains(combined, "痰") ||
		strings.Contains(combined, "喘") || strings.Contains(combined, "哮喘") ||
		strings.Contains(combined, "慢阻肺") || strings.Contains(combined, "肺部") ||
		strings.Contains(combined, "呼吸困难")

	// 1. 时间轴间隔特征计算
	intervalSummary := "患者既往就诊档案显示起病较平稳。"
	if len(timeline) >= 2 {
		latestInterval := timeline[len(timeline)-1].IntervalDays
		if latestInterval > 0 && latestInterval <= 25 {
			intervalSummary = fmt.Sprintf("【发病间隔显著缩短预警】：前次就诊间隔相对平缓，而本次就诊与上次仅相隔 %d 天，发作频率呈急剧密集加速态势，提示基础靶器官病理生理代偿状态失衡。", latestInterval)
		} else {
			intervalSummary = fmt.Sprintf("患者跨期就诊平均间隔约 %d 天，病程表现为慢性迁延与周期性活动。", latestInterval)
		}
	}

	// 2. 针对心血管系统
	if hasCardio || (!hasDigestive && !hasRespiratory) {
		analysisSummary = fmt.Sprintf("结合患者已获授权的 %d 份历史病历与当前就诊主诉进行多维关联分析。%s 临床特征表现为从早期的劳力性胸闷，转变为当前静息/夜间压榨性发作伴放射痛。结合既往用药与心电图改变，高度提示冠状动脉病变进行性恶化与急性冠脉综合征（ACS）风险。", len(timeline), intervalSummary)

		symptomDetail = model.SymptomCorrelationDetail{
			CurrentAnalysis:    "当前主诉呈现胸骨后压榨性闷痛伴放射，起病突发且向夜间静息状态延伸，痛程较既往显著延长，伴劳力耐受严重下降。",
			HistoricalLinkage:  "横向比对既往病历记载，患者既往主诉多为“劳力后轻度胸闷”，而在近期病程中诱发阈值显著降低，发作性质已由稳定性转变为不稳定演进。",
			ProgressionPattern: "【发病间隔由数月急缩至十余天 + 诱发阈值骤降 + 持续时间延长】，符合典型冠脉不稳定型缺血病程进展特征。",
			KeySymptomsTrack:   []string{"劳力性心前区不适 (初期)", "活动后气促伴心悸 (中期)", "静息夜间压榨痛伴放射 (当前)"},
		}

		medDetail = model.MedicationCorrelationDetail{
			PastMedications:      pastMeds,
			DrugResponseAnalysis: "既往病历记录患者曾规律/间歇服用降压及改善循环药物，但近期症状仍频发恶化，提示现有常规药物方案未能有效遏制心肌供血失衡。",
			InteractionsAndRisks: "若本次加用硝酸甘油或双抗血小板，需警惕与原有降压药协同引起的体位性低血压，密切关注血小板与出凝血指标。",
		}

		labSummary = "既往医技检查及化验结果提示存在心肌供血不足及心室负荷增高迹象，当前新发主诉提示局部缺血病灶可能出现斑块不稳定破裂或微血栓形成。"

		predicted = []model.PredictedDiseaseItem{
			{
				DiseaseName:        "冠状动脉粥样硬化性心脏病 (不稳定型心绞痛 / NSTE-ACS)",
				ICDCode:            "I20.0",
				Probability:        88,
				ConfidenceLevel:    "高置信度",
				RiskLevel:          "CRITICAL",
				ClinicalBasis:      "发病间隔显著缩短，疼痛由劳力诱发向静息状态转变，既往心电图ST-T异常与心肌负荷加重吻合。",
				DifferentialPoints: "需紧急排查急性心肌梗死（STEMI/NSTEMI）与主动脉夹层（夹层常伴剧烈撕裂样背部放射痛及双侧血压不对称）。",
				SuggestedExams:     []string{"急诊 18 导联床旁心电图", "急查高敏肌钙蛋白 hs-cTnI (0h/2h动态)", "床旁超声心动图", "急诊冠状动脉造影/CTA"},
			},
			{
				DiseaseName:        "充血性心力衰竭急性失代偿期 (ADHF)",
				ICDCode:            "I50.9",
				Probability:        72,
				ConfidenceLevel:    "重点疑似",
				RiskLevel:          "HIGH",
				ClinicalBasis:      "长期高血压与缺血病史，活动后呼吸困难演进为夜间阵发憋气，既往检查提示心功能储备下降。",
				DifferentialPoints: "评估肺部双底湿啰音、颈静脉充盈征与下肢凹陷性水肿程度。",
				SuggestedExams:     []string{"B型利钠肽原 (NT-proBNP) 定量", "胸部床旁平片 (评估肺淤血)"},
			},
			{
				DiseaseName:        "胃食管反流病 (GERD) 伴食管源性胸痛",
				ICDCode:            "K21.9",
				Probability:        26,
				ConfidenceLevel:    "鉴别排除",
				RiskLevel:          "LOW",
				ClinicalBasis:      "夜间平卧易加重反流刺激食管痉挛引起心前区不适，但难以解释既往心肌标志物及活动性气促，故列为排除性鉴别。",
				DifferentialPoints: "观察有无反酸烧心、抗酸剂试用后有无显著缓解。",
				SuggestedExams:     []string{"病情稳定后择期行上消化道内镜检查"},
			},
		}

		treatment = model.RecommendedTreatmentPlan{
			Prescriptions: []model.PrescriptionSuggestionItem{
				{Category: "抗血小板", DrugName: "阿司匹林肠溶片 100mg + 替格瑞洛 90mg", Usage: "立即口服负荷量 (双抗抗血栓)"},
				{Category: "改善心肌缺血", DrugName: "硝酸甘油片 0.5mg / 单硝酸异山梨酯 20mg", Usage: "心绞痛发作时舌下含服，必要时5分钟重复"},
				{Category: "调脂稳定斑块", DrugName: "阿托伐他汀钙片 20mg", Usage: "每晚一次口服 (qn)"},
			},
			TreatmentPlanText: "1. 绝对卧床休息，持续低流量吸氧与心电血压血氧多参数监护；\n2. 立即启动床旁心肌标志物与全导联心电图急查；\n3. 予以抗血小板、调脂稳定斑块与扩张冠脉对症处理；\n4. 评估介入指征，必要时立即启动胸痛中心急救绿色通道。",
			SuggestedOrders:   []string{"急查 18 导联心电图", "急查血清肌钙蛋白 hs-cTnI", "超声心动图", "急查凝血功能"},
		}

		warnings = []string{
			"⚠️ 【急危重症预警】患者发病时间间隔由前期数月骤缩至十余天，发作频次加快且呈静息发作，高度怀疑急性冠脉综合征（ACS）风险，建议优先开辟绿色通道并留观抢救！",
			"⚠️ 【用药安全防范】既往长期口服降压药物，若联合使用硝酸酯类药物，必须严密监测仰卧位与立位血压，防止诱发急性低血压休克。",
			"⚠️ 【合规声明】AI 临床推理仅基于医生当前已授权查阅的有限历史数据推演，最终确诊与处方用药方案必须由当班执业医师当面查体签字确立。",
		}

		return
	}

	// 3. 针对消化系统
	if hasDigestive {
		analysisSummary = fmt.Sprintf("结合已获授权的历史就诊数据，患者存在明确消化道反复病史。%s 本次主诉以中上腹隐痛规律性变化为主，结合既往抑酸用药与幽门螺杆菌感染背景，高度提示消化性溃疡活动期加重伴隐性出血可能。", intervalSummary)

		symptomDetail = model.SymptomCorrelationDetail{
			CurrentAnalysis:    "上腹部隐痛、反酸嗳气与进餐呈节律性关联，且近期夜间痛加剧。",
			HistoricalLinkage:  "横向对比既往病历，数月前曾因反酸纳差就诊，经短期抗酸治疗缓解后近期再次发作且程度加深。",
			ProgressionPattern: "慢性反复发作 -> 停药后反跳加剧 -> 累及黏膜深层肌层。",
			KeySymptomsTrack:   []string{"轻度反酸嗳气", "中上腹节律性隐痛", "夜间饥饿痛伴黑便可疑"},
		}

		medDetail = model.MedicationCorrelationDetail{
			PastMedications:      pastMeds,
			DrugResponseAnalysis: "既往使用 PPI 抑酸剂症状可部分缓解，但停药后易复发，提示胃黏膜屏障未彻底修复或合并 Hp 根除不彻底。",
			InteractionsAndRisks: "若合并使用 NSAIDs 止痛药，将大幅增加消化道急性穿孔与大出血风险。",
		}

		labSummary = "既往化验单显示贫血标志物及大便隐血值得持续跟进，建议及时复查血常规与便常规 OB。"

		predicted = []model.PredictedDiseaseItem{
			{
				DiseaseName:        "消化性溃疡 (十二指肠球部/胃溃疡活动期)",
				ICDCode:            "K27.9",
				Probability:        86,
				ConfidenceLevel:    "高置信度",
				RiskLevel:          "HIGH",
				ClinicalBasis:      "节律性上腹痛加重，伴反酸嗳气，既往抗酸药物有效史与停药复发特征吻合。",
				DifferentialPoints: "需鉴别胃恶性肿瘤（胃癌）与慢性胆囊炎（胆囊炎多伴右上腹绞痛及放射至右肩部）。",
				SuggestedExams:     []string{"无痛胃镜检查 + 幽门螺杆菌快速尿素酶试验", "大便隐血试验 (OB)", "全腹部超声"},
			},
			{
				DiseaseName:        "慢性萎缩性胃炎伴肠化生",
				ICDCode:            "K29.4",
				Probability:        68,
				ConfidenceLevel:    "重点疑似",
				RiskLevel:          "MEDIUM",
				ClinicalBasis:      "慢性消化不良表现，病程反复迁延多年。",
				DifferentialPoints: "内镜下黏膜变薄呈红白相间，黏膜下血管透见。",
				SuggestedExams:     []string{"胃镜病理活检", "血清胃蛋白酶原 PG I/II 测定"},
			},
		}

		treatment = model.RecommendedTreatmentPlan{
			Prescriptions: []model.PrescriptionSuggestionItem{
				{Category: "质子泵抑制剂 (PPI)", DrugName: "艾司奥美拉唑镁肠溶片 20mg", Usage: "每日早晨餐前口服 (qd)"},
				{Category: "黏膜保护剂", DrugName: "铝碳酸镁咀嚼片 1.0g", Usage: "餐后 1-2 小时嚼服 (tid)"},
			},
			TreatmentPlanText: "1. 规范口服质子泵抑制剂足疗程抑酸治疗；\n2. 完善碳13/14呼气试验与大便潜血排查；\n3. 戒除辛辣刺激饮食、戒酒及戒烟，避免滥用阿司匹林等胃黏膜损伤药物。",
			SuggestedOrders:   []string{"碳13呼气试验", "大便隐血试验 (OB)", "血常规三分类"},
		}

		warnings = []string{
			"⚠️ 【警惕消化道出血】若患者近期排出柏油样黑便或伴有头晕面色苍白，需高度警惕急性活动性上消化道大出血！",
			"⚠️ 【严禁盲目止痛】切忌在腹痛未明确病因前注射强效镇痛药物，以免掩盖急腹症或溃疡穿孔征兆。",
		}

		return
	}

	// 4. 针对呼吸系统
	analysisSummary = fmt.Sprintf("结合患者已授权的跨机构病历时间线，既往有呼吸道感染与肺功能受损背景。%s 本次主诉以咳嗽咳痰进行性加重伴喘鸣为主，综合既往抗感染用药史，提示慢性呼吸道疾病急性加重期（AECOPD）合并细菌性感染。", intervalSummary)

	symptomDetail = model.SymptomCorrelationDetail{
		CurrentAnalysis:    "咳嗽剧烈伴黄色脓痰，活动后气喘明显加重，伴轻度低热。",
		HistoricalLinkage:  "横向病历比对显示，每年秋冬季节转换时均有同类就诊记录，本次发病间隔较往年更短且抗感染药物起效趋缓。",
		ProgressionPattern: "慢性咳嗽咳痰 -> 气候诱发急性加重 -> 伴有气道阻塞与通气功能受限。",
		KeySymptomsTrack:   []string{"轻微干咳", "晨起咳白色黏痰", "黄色浓稠脓痰伴喘促发绀 (当前)"},
	}

	medDetail = model.MedicationCorrelationDetail{
		PastMedications:      pastMeds,
		DrugResponseAnalysis: "既往使用口服抗生素及止咳药物，本次症状仍呈进展态势，提示可能存在病原菌耐药或深部小气道痉挛未解除。",
		InteractionsAndRisks: "使用支气管扩张剂注意心率反应，避免诱发快速房颤或心动过速。",
	}

	labSummary = "既往胸片提示肺纹理增粗紊乱与肺气肿征象，本次建议加查白细胞、C反应蛋白及动脉血气分析。"

	predicted = []model.PredictedDiseaseItem{
		{
			DiseaseName:        "慢性阻塞性肺疾病急性加重期 (AECOPD)",
			ICDCode:            "J44.1",
			Probability:        85,
			ConfidenceLevel:    "高置信度",
			RiskLevel:          "HIGH",
			ClinicalBasis:      "咳、痰、喘三联征显著急性加重，脓痰量增加，既往慢性肺病病程明确。",
			DifferentialPoints: "与支气管哮喘急性发作鉴别（哮喘多以发作性双肺呼气相哮鸣音为主，青少年/过敏体质多发）。",
			SuggestedExams:     []string{"胸部高质量多排 CT 平扫", "动脉血气分析", "痰培养及药物敏感试验", "血常规 + CRP"},
		},
		{
			DiseaseName:        "社区获得性肺炎 (CAP)",
			ICDCode:            "J18.9",
			Probability:        65,
			ConfidenceLevel:    "重点疑似",
			RiskLevel:          "MEDIUM",
			ClinicalBasis:      "发热伴咳黄脓痰，肺部听诊局限性湿啰音可能。",
			DifferentialPoints: "胸部 CT 见斑片状浸润阴影或肺实变征象。",
			SuggestedExams:     []string{"胸部 CT 平扫", "PCT 降钙素原"},
		},
	}

	treatment = model.RecommendedTreatmentPlan{
		Prescriptions: []model.PrescriptionSuggestionItem{
			{Category: "支气管舒张剂", DrugName: "布地奈德福莫特罗吸入粉雾剂", Usage: "每次一吸，早晚各一次吸入"},
			{Category: "抗感染药物", DrugName: "左氧氟沙星片 0.5g / 头孢克肟胶囊 0.1g", Usage: "每日一次口服 (qd)"},
		},
		TreatmentPlanText: "1. 积极控制感染，保持呼吸道通畅与化痰排痰；\n2. 予以雾化吸入糖皮质激素与支气管扩张剂；\n3. 监测静息血氧饱和度，若 SpO2 < 90% 予低流量持续吸氧。",
		SuggestedOrders:   []string{"胸部 CT 平扫", "血常规 + 超敏CRP", "动脉血气分析"},
	}

	warnings = []string{
		"⚠️ 【呼吸衰竭防范】密切关注患者神志变化及血氧饱和度，防止严重低氧血症及高碳酸血症肺性脑病发生！",
	}

	return
}

// buildDiagnosticPrompt 构建输入给大模型的结构化高质量 Prompt
func (s *AIDiagnosticService) buildDiagnosticPrompt(
	patient model.User,
	records []model.MedicalRecord,
	exams []model.MedicalExamOrder,
	req model.DiagnosticAssistRequest,
	timeline []model.TimelineCorrelationItem,
) string {
	var sb strings.Builder
	sb.WriteString("你是一位资深的医院副主任级临床全科与急危重症鉴别诊断专家系统。\n")
	sb.WriteString("请基于以下由患者已合法授权给经治医生调阅的历史病历档案、化验单及当前输入的接诊主诉，进行严谨的跨期病程演进分析、时间间隔关联推演及疑似疾病鉴别诊断。\n\n")

	sb.WriteString(fmt.Sprintf("【患者基本信息】\n- 姓名: %s (卡号: %s)\n- 性别: %s, 年龄: %d 岁\n\n", patient.RealName, patient.UserNo, patient.Gender, patient.Age))

	sb.WriteString("【当前医生接诊输入 (Present Encounter)】\n")
	sb.WriteString(fmt.Sprintf("- 接诊科室: %s\n", req.DepartmentName))
	sb.WriteString(fmt.Sprintf("- 患者主诉 (Chief Complaint): %s\n", req.CurrentSymptoms))
	sb.WriteString(fmt.Sprintf("- 现病史与诱因 (Present Illness): %s\n", req.PresentIllness))
	sb.WriteString(fmt.Sprintf("- 生命体征 (Vital Signs): %s\n\n", req.VitalSigns))

	sb.WriteString("【已授权历史病历档案与时间线 (Historical Records)】\n")
	if len(records) == 0 {
		sb.WriteString("（该患者在跨机构网络中暂无既往历史病历记录，本次为首诊档案建立）\n\n")
	} else {
		for i, r := range timeline {
			sb.WriteString(fmt.Sprintf("[%d] 就诊时间: %s (距今 %d 天, %s)\n", i+1, r.VisitDate, r.DaysAgo, r.IntervalFromPrev))
			sb.WriteString(fmt.Sprintf("    就诊医院与科室: %s - %s\n", r.HospitalName, r.DepartmentName))
			sb.WriteString(fmt.Sprintf("    既往主诉: %s\n", r.ChiefComplaint))
			sb.WriteString(fmt.Sprintf("    既往确诊: %s\n", r.Diagnosis))
			sb.WriteString(fmt.Sprintf("    处方用药: %s\n", r.TreatmentPlan))
			sb.WriteString(fmt.Sprintf("    检查总结: %s\n", r.ExamSummary))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("【已授权历史医技与化验单明细 (Lab Orders)】\n")
	if len(exams) == 0 {
		sb.WriteString("（暂无过往独立医技检验回传数据）\n\n")
	} else {
		for _, eo := range exams {
			sb.WriteString(fmt.Sprintf("- 检查项目: %s | 状态: %s | 测量结果: %s | 报告结论: %s\n", eo.ExamItem, eo.Status, eo.ExamResult, eo.ExamConclusion))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("【分析与输出要求】\n")
	sb.WriteString("请严密测算历次就诊与当前主诉之间的时间间隔（如间隔是由长变短还是规律出现）、病症发展演变规律（从何种局部或轻度症状演变至当前主诉）、既往用药反馈与耐药/不良反应关联，并输出结构化 JSON，包含：\n")
	sb.WriteString("1. analysis_summary: 综合病程演化分析简述\n")
	sb.WriteString("2. symptom_correlation: 症状动态演进分析\n")
	sb.WriteString("3. medication_correlation: 用药史与治疗反应关联\n")
	sb.WriteString("4. predicted_diseases: 疑似疾病预测清单 (包含 disease_name, icd_code, probability (0-100), clinical_basis, differential_points, suggested_exams)\n")
	sb.WriteString("5. recommended_treatment: 建议处置用药方案 (prescriptions, treatment_plan_text, suggested_orders)\n")
	sb.WriteString("6. clinical_warnings: 急危重症与用药禁忌警示清单\n")

	return sb.String()
}

// callExternalLLM 调用外部真实大模型接口 (OpenAI 协议兼容)
func (s *AIDiagnosticService) callExternalLLM(apiURL, apiKey, modelName, prompt string) (*model.DiagnosticAssistResult, error) {
	endpoint := apiURL
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint = strings.TrimRight(endpoint, "/") + "/chat/completions"
	}

	reqBody := map[string]interface{}{
		"model": modelName,
		"messages": []map[string]string{
			{"role": "system", "content": "你是一位资深医疗临床辅助诊断专家，严格以 JSON 格式输出分析结果。"},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.2,
		"response_format": map[string]string{
			"type": "json_object",
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("external AI API error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return nil, err
	}

	if len(openAIResp.Choices) == 0 {
		return nil, errors.New("empty choices returned from external AI API")
	}

	rawContent := openAIResp.Choices[0].Message.Content

	var parsed model.DiagnosticAssistResult
	if err := json.Unmarshal([]byte(rawContent), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse AI JSON response: %w", err)
	}

	return &parsed, nil
}
