package service

import (
	"fmt"
	"strings"
	"sync"

	"medtrust-backend/model"
	"medtrust-backend/repository"
)

type InfectionRule struct {
	Name           string   // 疾病通用全称
	Category       string   // 传播分类
	Persistence    string   // 携带性质 (必定终身携带 / 长期慢性携带 / 活动期持续排菌)
	ThreatToDoctor string   // 医护职业暴露主要威胁
	Transmission   string   // 传播途径
	RiskLevel      string   // CRITICAL (极高危), HIGH (高危), MEDIUM_HIGH (重点中高危)
	Keywords       []string // 诊断与病历匹配关键词库 (不区分大小写)
	Precautions    []string // 医务人员专项防护要点
	ProtectionGear []string // 必备防护装备 (PPE)
	EmergencySteps []string // 意外暴露应急处置步骤
}

type InfectionService struct {
	rules []InfectionRule
}

var (
	DefaultInfectionService *InfectionService
	infectionOnce           sync.Once
)

func GetInfectionService() *InfectionService {
	infectionOnce.Do(func() {
		DefaultInfectionService = &InfectionService{
			rules: initInfectionRules(),
		}
	})
	return DefaultInfectionService
}

func initInfectionRules() []InfectionRule {
	return []InfectionRule{
		{
			Name:           "获得性免疫缺陷综合征 / 艾滋病 (HIV / AIDS)",
			Category:       "血源与体液高危传播",
			Persistence:    "终身携带 (Lifelong Carriage)",
			ThreatToDoctor: "锐器针刺伤暴露、血液黏膜喷溅暴露导致医护感染HIV，致死性极高",
			Transmission:   "血液、体液、黏膜破损、利器穿刺",
			RiskLevel:      "CRITICAL",
			Keywords: []string{
				"艾滋病", "hiv", "aids", "获得性免疫缺陷", "人类免疫缺陷病毒",
				"hiv抗体阳性", "hiv-ab(+)", "hiv阳性", "hiv感染", "免疫缺陷病毒",
			},
			Precautions: []string{
				"严格执行标准预防与双层屏障防护，侵入性或抽血操作必须穿戴双层手套与防护面屏/护目镜",
				"严禁双手回套针帽，注射与穿刺完成后必须立即将锐器弃入就近防穿刺利器收集盒",
				"手术或有创操作中禁止徒手直接传递刀片或缝针，必须使用无接触中转盘传递",
				"若发生针刺伤或破损皮肤黏膜暴露，立即执行“一挤二冲三消毒”并在2小时内启动PEP药物阻断",
			},
			ProtectionGear: []string{
				"医用防护口罩 (N95/KN95)",
				"双层无粉灭菌乳胶手套",
				"防喷溅防护面屏 / 护目镜",
				"一次性防渗透隔离衣",
				"自毁型安全留置针与防针刺针具",
			},
			EmergencySteps: []string{
				"挤血：从近心端向远心端轻轻挤压伤口，尽量挤出损伤处血液，严禁局部直接按压挤捏",
				"冲洗：立即使用流动的生理盐水或流动肥皂清水反复彻底冲洗伤口至少 5 分钟",
				"消毒：用 75% 医用乙醇或 0.5% 聚维酮碘消毒伤口局部，并包扎保护",
				"上报与用药：立即向院感科紧急报备，并在暴露后 2 小时内尽早服用 PEP 阻断抗病毒药物 (最长不超 72 小时)",
			},
		},
		{
			Name:           "慢性乙型病毒性肝炎 (Chronic Hepatitis B / HBV)",
			Category:       "血源与体液高危传播",
			Persistence:    "长期/终身慢性携带 (Chronic Persistent Carriage)",
			ThreatToDoctor: "针刺伤传播率高达 6%~30%，易导致医护人员爆发性肝炎或慢性肝脏疾病",
			Transmission:   "血液、体液、注射针刺、黏膜破损",
			RiskLevel:      "HIGH",
			Keywords: []string{
				"乙型肝炎", "乙肝", "hbv", "hbsag(+)", "hbsag阳性", "乙肝表面抗原阳性",
				"大三阳", "小三阳", "乙型病毒性肝炎", "慢性乙肝", "乙肝病毒携带",
			},
			Precautions: []string{
				"医护人员接诊前应确认自身已完成乙肝疫苗接种并具备抗-HBs保护性抗体 (滴度 ≥ 10 mIU/mL)",
				"进行静脉采血、穿刺、拔针或换药操作时必须规范佩戴乳胶手套，避免徒手接触患者血液",
				"锐器操作全程防范刺伤，所有污染利器严格定点入盒",
				"一旦发生职业暴露且抗体阴性，应在24小时内注射乙肝高效价免疫球蛋白(HBIG)并补种疫苗",
			},
			ProtectionGear: []string{
				"医用外科/防护口罩",
				"双层乳胶医用检查手套",
				"防溅眼罩 (有创喷溅风险操作时)",
				"防刺锐器收集盒",
			},
			EmergencySteps: []string{
				"立即挤出伤口伤处污血，流动清水及肥皂液冲洗 10 分钟",
				"0.5% 聚维酮碘或 75% 酒精涂抹消毒",
				"急查暴露医护人员抗-HBs抗体水平",
				"若抗体滴度不足 10 mIU/mL，24 小时内肌注乙肝免疫球蛋白 (HBIG) 200~400 IU 并续种疫苗",
			},
		},
		{
			Name:           "慢性丙型病毒性肝炎 (Chronic Hepatitis C / HCV)",
			Category:       "血源与体液传播",
			Persistence:    "长期慢性携带 (70%~80%慢性化率)",
			ThreatToDoctor: "锐器刺伤或破损皮肤接触含有病毒血液，易引起慢性肝损伤",
			Transmission:   "血液、深部穿刺针刺暴露",
			RiskLevel:      "HIGH",
			Keywords: []string{
				"丙型肝炎", "丙肝", "hcv", "抗-hcv(+)", "抗-hcv阳性", "hcv-rna阳性",
				"丙型病毒性肝炎", "慢性丙肝", "丙肝抗体阳性",
			},
			Precautions: []string{
				"严格落实侵入性操作标准防护，接触血液及体液必须佩戴手套，若破损立即更换",
				"强化利器规范化传递流程，杜绝危险徒手操作",
				"发生暴露后由于尚无疫苗与免疫球蛋白，需立即冲洗消毒并在4~6周内监测HCV RNA与ALT",
			},
			ProtectionGear: []string{
				"医用外科口罩",
				"医用乳胶手套",
				"护目镜 (喷溅高危环境)",
			},
			EmergencySteps: []string{
				"近心端向远心端挤血，清水充分冲洗伤口",
				"碘伏消毒并贴无菌防水辅料",
				"暴露当时、第2周、第4周分别检测基线与动态 HCV RNA 载量，必要时启动直接抗病毒药物 (DAA)",
			},
		},
		{
			Name:           "活动性/开放性肺结核 (Active Pulmonary TB / MDR-TB)",
			Category:       "空气与呼吸道气溶胶烈性传播",
			Persistence:    "活动期持续排菌/病程迁延",
			ThreatToDoctor: "患者咳嗽、咳痰、打喷嚏产生飞沫气溶胶，诊室密闭环境下医护极易吸入感染",
			Transmission:   "空气飞沫微粒吸入、气溶胶传播",
			RiskLevel:      "HIGH",
			Keywords: []string{
				"肺结核", "开放性结核", "活动性结核", "结核分枝杆菌", "耐药结核",
				"mdr-tb", "涂阳肺结核", "结核性胸膜炎", "空洞型肺结核",
			},
			Precautions: []string{
				"就诊与查体期间指导患者佩戴医用外科口罩，减少飞沫扩散",
				"接诊医护人员必须严格佩戴密合性优良的 N95 / KN95 医用防护口罩",
				"诊室必须保持强化机械通风或开启负压层流通风设施 (换气次数 ≥ 12 ACH)",
				"尽量避免在非负压环境下开展咽拭子、吸痰或诱导排痰等高气溶胶操作",
			},
			ProtectionGear: []string{
				"医用防护口罩 (N95/KN95 级，气密合格)",
				"医用隔离衣 / 防护面屏",
				"一次性医用检查手套",
			},
			EmergencySteps: []string{
				"若发生面部无防护近距离剧烈呛咳飞沫暴露，立即移步通风良好处流动水清洗眼部及口鼻",
				"向院感与疾控科登记报备，建立潜伏感染随访档案",
				"暴露后第 8~12 周行结核菌素皮肤试验 (PPD) 或 γ-干扰素释放试验 (IGRA)，异常时行预防性抗结核治疗",
			},
		},
		{
			Name:           "梅毒 (Syphilis / 现症或晚期活动期)",
			Category:       "接触与体液黏膜传播",
			Persistence:    "血清固定/持续感染 (Untreated or Chronic Infection)",
			ThreatToDoctor: "直接接触患者破损硬下疳、梅毒疹溃疡或血液导致医源性感染",
			Transmission:   "血液、皮损渗出液、黏膜直接接触",
			RiskLevel:      "MEDIUM_HIGH",
			Keywords: []string{
				"梅毒", "梅毒螺旋体", "tppa阳性", "rpr阳性", "trust阳性",
				"一期梅毒", "二期梅毒", "神经梅毒", "硬下疳", "苍白螺旋体",
			},
			Precautions: []string{
				"体格检查、视触诊或伤口换药全程佩戴医用乳胶手套，医护人员手部有皮损者严禁直接操作",
				"接触患者病变皮肤或分泌物后，脱手套后必须严格使用皂液洗手并速干手消毒",
				"严防穿刺针刺伤",
			},
			ProtectionGear: []string{
				"医用乳胶手套 (查体接触必备)",
				"医用外科口罩",
				"防水围裙/隔离衣 (脓液创面处理时)",
			},
			EmergencySteps: []string{
				"针刺或创面接触后立即挤血、流动清水肥皂水刷洗 5 分钟",
				"75% 酒精消毒局部",
				"立即行基线 TPPA/RPR 筛查，必要时肌注苄星青霉素 240 万单位进行预防性干预",
			},
		},
		{
			Name:           "多重耐药超级细菌定植与感染 (CRE / MRSA / VRE / CRAB)",
			Category:       "接触传播与院感环境定植",
			Persistence:    "体内长期定植 (Persistent Colonization)",
			ThreatToDoctor: "医护人员手部或工作服携带转移，易造成自身带菌及病区严重暴发流行",
			Transmission:   "接触传播、器械交叉接触、环境物表污染",
			RiskLevel:      "MEDIUM_HIGH",
			Keywords: []string{
				"耐碳青霉烯", "cre", "mrsa", "耐甲氧西林", "vre", "耐万古霉素",
				"多重耐药菌", "泛耐药鲍曼", "多重耐药", "carbapenem-resistant",
			},
			Precautions: []string{
				"实施严格的接触隔离措施，进入诊疗区域需穿戴一次性隔离衣与手套",
				"诊疗结束后严格执行卫生洗手与手部快速消毒",
				"听诊器、血压计袖带、体温表等器械必须专人专用，使用后 1000mg/L 含氯消毒剂彻底擦拭",
			},
			ProtectionGear: []string{
				"一次性医用防护隔离衣",
				"医用检查手套",
				"医用外科口罩",
				"专用诊疗器具包",
			},
			EmergencySteps: []string{
				"接触患者排泄物或伤口渗液后立即肥皂水洗手加手消毒液彻底搓揉",
				"污染的工作服立即更换入黄色医疗废物袋高温高压消杀",
			},
		},
		{
			Name:           "狂犬病 (Rabies / 狂犬病毒感染)",
			Category:       "唾液与中枢神经烈性感染",
			Persistence:    "急性烈性感染/致死率100%",
			ThreatToDoctor: "患者躁动恐水，咬伤、抓伤或唾液喷溅接触医护破损黏膜极度致命",
			Transmission:   "唾液喷溅、咬伤抓伤、黏膜接触",
			RiskLevel:      "CRITICAL",
			Keywords: []string{
				"狂犬病", "狂犬病毒", "恐水症", "rabies",
			},
			Precautions: []string{
				"严格执行三级个人防护，穿戴防撕咬防护装备、防护面罩与厚橡胶手套",
				"就诊需设专人监护防范患者躁动攻击",
			},
			ProtectionGear: []string{
				"全脸防溅防护面罩",
				"防刺防咬特种防护手套",
				"防护服与防水围裙",
			},
			EmergencySteps: []string{
				"若发生咬伤抓伤，立即使用 20% 肥皂水和流动清水交替冲洗伤口至少 15 分钟",
				"稀碘伏彻底消毒，伤口原则上不予缝合",
				"立即注射狂犬病被动免疫制剂 (狂犬病人免疫球蛋白) 并在当天启动狂犬疫苗全程接种",
			},
		},
		{
			Name:           "克雅氏病 (Creutzfeldt-Jakob Disease / 朊病毒 CJD)",
			Category:       "神经组织医源性接触极高危",
			Persistence:    "终身致死性存在 (朊毒体常规高温高压无法灭活)",
			ThreatToDoctor: "眼科、神经外科、腰穿等操作接触脑脊液或神经组织，极难消毒致医护严重感染",
			Transmission:   "神经组织接触、脑脊液穿刺、侵入性器械交叉污染",
			RiskLevel:      "CRITICAL",
			Keywords: []string{
				"克雅氏病", "克-雅氏病", "cjd", "朊毒体", "朊病毒", "海绵状脑病",
			},
			Precautions: []string{
				"凡涉及脑脊液穿刺、眼科或神经外科检查必须全程使用一次性专属器械",
				"使用后的器械严禁与普通手术器械混淆，必须密闭封装后送专业焚烧炉彻底销毁",
			},
			ProtectionGear: []string{
				"双层耐穿刺乳胶手套",
				"防渗透全封闭防护面屏",
				"一次性防渗透连体防护服",
			},
			EmergencySteps: []string{
				"发生锐器针刺或脑脊液黏膜污染后，立即用 1M 氢氧化钠或 2.5% 次氯酸钠溶液冲洗创口 5 分钟",
				"大量清水反复彻底清洗，紧急上报疾控与院感专科评估",
			},
		},
	}
}

func isASCIIAlphaNumByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// findPositiveKeywordMatch 在否定剥离后的肯定性子句中定位传染病关键词，兼顾 ASCII 词边界与邻近否定窗口保护
func findPositiveKeywordMatch(clause, lowerKw string) bool {
	offset := 0
	for {
		idx := strings.Index(clause[offset:], lowerKw)
		if idx < 0 {
			return false
		}
		absIdx := offset + idx
		endIdx := absIdx + len(lowerKw)

		// 纯/首尾 ASCII 词边界校验，避免 "cre" 误中 "creatinine" 等无关英文词
		validBoundary := true
		if len(lowerKw) > 0 && isASCIIAlphaNumByte(lowerKw[0]) && absIdx > 0 && isASCIIAlphaNumByte(clause[absIdx-1]) {
			validBoundary = false
		}
		if len(lowerKw) > 0 && isASCIIAlphaNumByte(lowerKw[len(lowerKw)-1]) && endIdx < len(clause) && isASCIIAlphaNumByte(clause[endIdx]) {
			validBoundary = false
		}

		if validBoundary {
			// 后缀邻近窗口检测（最多 24 字节，排除 "HBsAg 阴性" / "HIV (-)" / "TPPA未检出"）
			sufEnd := endIdx + 24
			if sufEnd > len(clause) {
				sufEnd = len(clause)
			}
			suf := clause[endIdx:sufEnd]
			posInSuf := strings.Contains(suf, "阳性") || strings.Contains(suf, "(+)")
			negInSuf := strings.Contains(suf, "阴性") || strings.Contains(suf, "(-)") ||
				strings.Contains(suf, "negative") || strings.Contains(suf, "non-reactive") ||
				strings.Contains(suf, "未检出") || strings.Contains(suf, "未见") || strings.Contains(suf, "排除")

			// 前缀邻近窗口检测（最多 24 字节，排除 "无明确结核" / "否认HIV"）
			preStart := absIdx - 24
			if preStart < 0 {
				preStart = 0
			}
			pre := clause[preStart:absIdx]
			negInPre := strings.Contains(pre, "否认") || strings.Contains(pre, "未见") ||
				strings.Contains(pre, "排除") || strings.Contains(pre, "无既往")

			if !(negInSuf && !posInSuf) && !negInPre {
				return true
			}
		}

		offset = endIdx
		if offset >= len(clause) {
			return false
		}
	}
}

// matchText 针对单段文本（诊断、主诉、既往史、现病史等）进行关键词扫描（自动剥离阴性/否认表述，防止假阳性告警）
func (s *InfectionService) matchText(text string) []model.InfectionDiseaseInfo {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	affirmative := filterAffirmativeClinicalText(text)
	if strings.TrimSpace(affirmative) == "" {
		return nil
	}
	clauses := clinicalClauseSepRe.Split(strings.ToLower(affirmative), -1)
	var matched []model.InfectionDiseaseInfo

	for _, rule := range s.rules {
	ruleLoop:
		for _, kw := range rule.Keywords {
			lowerKw := strings.ToLower(kw)
			for _, cl := range clauses {
				if cl == "" {
					continue
				}
				if findPositiveKeywordMatch(cl, lowerKw) {
					matched = append(matched, model.InfectionDiseaseInfo{
						Name:           rule.Name,
						Category:       rule.Category,
						Persistence:    rule.Persistence,
						ThreatToDoctor: rule.ThreatToDoctor,
						Transmission:   rule.Transmission,
						RiskLevel:      rule.RiskLevel,
						MatchedKeyword: kw,
					})
					break ruleLoop
				}
			}
		}
	}
	return matched
}

// AnalyzePatientRisks 查询该患者在全网/所有机构的历史病历与检验单，进行医护安全高危传染病排查
func (s *InfectionService) AnalyzePatientRisks(patientID uint64) (*model.InfectionRiskAlert, error) {
	return s.AnalyzePatientRisksWithScope(patientID, nil)
}

// AnalyzePatientRisksWithScope 在患者全景职业暴露检测的基础上，按调用者已获授权病历集合对未授权来源单号与机构名进行脱敏
// (authorizedRecordIDs == nil 表示内部全量检测或后续自行整体脱敏的场景)
func (s *InfectionService) AnalyzePatientRisksWithScope(patientID uint64, authorizedRecordIDs map[uint64]bool) (*model.InfectionRiskAlert, error) {
	if patientID == 0 {
		return &model.InfectionRiskAlert{HasRisk: false, RiskLevel: "SAFE", Summary: "未选择就诊患者"}, nil
	}

	// 1. 查询患者名下所有历史病历 (跨机构联合排查)
	records, err := repository.DefaultMedicalRepository.FindRecordsByPatient(patientID)
	if err != nil {
		return nil, err
	}

	// 2. 查询患者名下所有检验单
	examOrders, err := repository.DefaultExamOrderRepository.FindExamOrdersByPatientID(patientID)
	if err != nil {
		examOrders = nil
	}

	return s.evaluateRisksWithScope(records, examOrders, authorizedRecordIDs)
}

// AnalyzeRecordRisks 分析某单张病历与其患者历史背景（自动对未获该单据以外的跨院来源单号/机构脱敏，防止越权元数据泄露）
func (s *InfectionService) AnalyzeRecordRisks(recordID uint64) (*model.InfectionRiskAlert, error) {
	record, err := repository.DefaultMedicalRepository.FindRecordSimpleByID(recordID)
	if err != nil {
		return nil, err
	}

	if record.PatientID > 0 {
		return s.AnalyzePatientRisksWithScope(record.PatientID, map[uint64]bool{recordID: true})
	}

	return s.evaluateRisksWithScope([]model.MedicalRecord{*record}, nil, map[uint64]bool{recordID: true})
}

// EvaluateDirectText 即时评估医生在界面上填写的文本 (支持前端实时推断)
func (s *InfectionService) EvaluateDirectText(text string) *model.InfectionRiskAlert {
	matched := s.matchText(text)
	if len(matched) == 0 {
		return &model.InfectionRiskAlert{
			HasRisk:   false,
			RiskLevel: "SAFE",
			Summary:   "未检出高危传染病携带记录",
		}
	}
	return s.buildAlertFromMatched(matched, []string{"现场接诊录入文本"})
}

// evaluateRisks 汇总病历与检验单做综合风险研判
func (s *InfectionService) evaluateRisks(records []model.MedicalRecord, examOrders []model.MedicalExamOrder) (*model.InfectionRiskAlert, error) {
	return s.evaluateRisksWithScope(records, examOrders, nil)
}

// evaluateRisksWithScope 按调用者已授权病历边界汇总传染病风险；未授权来源档案仅返回脱敏职业暴露防护等级，严禁泄露具体病原学病名与单号
func (s *InfectionService) evaluateRisksWithScope(records []model.MedicalRecord, examOrders []model.MedicalExamOrder, authorizedRecordIDs map[uint64]bool) (*model.InfectionRiskAlert, error) {
	var allMatched []model.InfectionDiseaseInfo
	var detectedSources []string
	seenRule := make(map[string]bool)
	hasUnauthorizedRisk := false

	// 扫描病历字段（优先扫描已授权记录，确保若已授权记录与未授权记录同时记载时，优先展示已授权来源明细）
	orderedRecords := records
	if authorizedRecordIDs != nil && len(records) > 1 {
		orderedRecords = make([]model.MedicalRecord, 0, len(records))
		for _, rec := range records {
			if authorizedRecordIDs[rec.ID] {
				orderedRecords = append(orderedRecords, rec)
			}
		}
		for _, rec := range records {
			if !authorizedRecordIDs[rec.ID] {
				orderedRecords = append(orderedRecords, rec)
			}
		}
	}

	for _, rec := range orderedRecords {
		fullText := strings.Join([]string{
			rec.Diagnosis,
			rec.InitialDiagnosis,
			rec.ChiefComplaint,
			rec.PresentIllness,
			rec.Symptoms,
			rec.Etiology,
			rec.ExamResult,
			rec.ExamItems,
		}, "，")

		matches := s.matchText(fullText)
		isAuthorizedSource := authorizedRecordIDs == nil || authorizedRecordIDs[rec.ID]
		if !isAuthorizedSource {
			if len(matches) > 0 {
				hasUnauthorizedRisk = true
			}
			continue
		}
		for _, m := range matches {
			if !seenRule[m.Name] {
				seenRule[m.Name] = true
				m.RecordNo = rec.RecordNo
				m.HospitalName = rec.HospitalName
				if rec.HospitalName == "" {
					m.HospitalName = formatHospName(rec.HospitalID)
				}
				m.RecordDate = rec.CreatedAt.Format("2006-01-02 15:04")
				sourceDesc := fmt.Sprintf("就诊编号 %s (%s, %s): 命中关键词【%s】",
					rec.RecordNo, m.HospitalName, m.RecordDate, m.MatchedKeyword)
				detectedSources = append(detectedSources, sourceDesc)
				allMatched = append(allMatched, m)
			}
		}
	}

	// 扫描检验单字段
	for _, ord := range examOrders {
		examText := strings.Join([]string{ord.ExamItem, ord.ExamReason, ord.ExamResult}, "，")
		matches := s.matchText(examText)
		isAuthorizedOrder := authorizedRecordIDs == nil || (ord.RecordID > 0 && authorizedRecordIDs[ord.RecordID])
		if !isAuthorizedOrder {
			if len(matches) > 0 {
				hasUnauthorizedRisk = true
			}
			continue
		}
		for _, m := range matches {
			if !seenRule[m.Name] {
				seenRule[m.Name] = true
				m.RecordNo = ord.OrderNo
				m.HospitalName = formatHospName(ord.HospitalID)
				m.RecordDate = ord.CreatedAt.Format("2006-01-02 15:04")
				sourceDesc := fmt.Sprintf("检验单号 %s (%s): 命中关键词【%s】",
					ord.OrderNo, m.HospitalName, m.MatchedKeyword)
				detectedSources = append(detectedSources, sourceDesc)
				allMatched = append(allMatched, m)
			}
		}
	}

	if len(allMatched) == 0 {
		if hasUnauthorizedRisk {
			banner := "⚠️ 职业暴露三级防护预警：接诊/穿刺操作请严格佩戴双层手套与护目镜（具体病原学诊断受隐私锁保护，需患者显式授权或紧急破窗后解锁）"
			return &model.InfectionRiskAlert{
				HasRisk:         true,
				RiskLevel:       "DEIDENTIFIED",
				Summary:         banner,
				Diseases:        nil,
				Precautions:     nil,
				ProtectionGear:  nil,
				EmergencySteps:  nil,
				DetectedFrom:    []string{"既往关联受控隐私档案（未获该单明细授权，已隐去机构、单号与具体病原学病名）"},
				PrecautionLevel: "LEVEL_3_BLOOD_BODY_FLUID",
				BannerText:      banner,
			}, nil
		}
		return &model.InfectionRiskAlert{
			HasRisk:   false,
			RiskLevel: "SAFE",
			Summary:   "未检出高危传染病携带记录",
		}, nil
	}

	if hasUnauthorizedRisk {
		detectedSources = append(detectedSources, "另存在受隐私锁保护的未授权关联档案提示（已按隐私策略隐去具体病原学病名与单号）")
	}

	return s.buildAlertFromMatched(allMatched, detectedSources), nil
}

// buildAlertFromMatched 聚合命中项构建医护预警体
func (s *InfectionService) buildAlertFromMatched(matched []model.InfectionDiseaseInfo, detectedSources []string) *model.InfectionRiskAlert {
	highestLevel := "MEDIUM_HIGH"
	var names []string

	precautionsMap := make(map[string]bool)
	gearMap := make(map[string]bool)
	emergencyMap := make(map[string]bool)

	for _, m := range matched {
		names = append(names, m.Name)
		if m.RiskLevel == "CRITICAL" {
			highestLevel = "CRITICAL"
		} else if m.RiskLevel == "HIGH" && highestLevel != "CRITICAL" {
			highestLevel = "HIGH"
		}

		// 关联规则获取细则
		for _, rule := range s.rules {
			if rule.Name == m.Name {
				for _, p := range rule.Precautions {
					precautionsMap[p] = true
				}
				for _, g := range rule.ProtectionGear {
					gearMap[g] = true
				}
				for _, e := range rule.EmergencySteps {
					emergencyMap[e] = true
				}
				break
			}
		}
	}

	var precautions []string
	for p := range precautionsMap {
		precautions = append(precautions, p)
	}
	var gear []string
	for g := range gearMap {
		gear = append(gear, g)
	}
	var emergency []string
	for e := range emergencyMap {
		emergency = append(emergency, e)
	}

	summaryText := fmt.Sprintf("检测到患者有【%s】既往/长期携带病史，接诊查体请务必落实职业暴露严格防护！",
		strings.Join(names, "、"))

	return &model.InfectionRiskAlert{
		HasRisk:        true,
		RiskLevel:      highestLevel,
		Summary:        summaryText,
		Diseases:       matched,
		Precautions:    precautions,
		ProtectionGear: gear,
		EmergencySteps: emergency,
		DetectedFrom:   detectedSources,
	}
}

func formatHospName(hId uint64) string {
	switch hId {
	case 1:
		return "第一人民医院"
	case 2:
		return "省立中心医院"
	case 3:
		return "协和医学中心"
	default:
		return "医疗中心"
	}
}
