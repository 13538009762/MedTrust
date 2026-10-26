package model

// Response 统一 API 响应结果模型
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// LoginRequest 登录请求入参
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest 用户注册入参
type RegisterRequest struct {
	Username     string `json:"username" binding:"required"`
	Password     string `json:"password" binding:"required"`
	RealName     string `json:"real_name" binding:"required"`
	Role         string `json:"role" binding:"required"`
	HospitalID   uint64 `json:"hospital_id"`
	DepartmentID uint64 `json:"department_id"`
	Title        string `json:"title"`
	Phone        string `json:"phone"`
	IDCard       string `json:"id_card"`
}

// ProfileDTO 用户画像与状态信息
type ProfileDTO struct {
	User
	Token string `json:"token,omitempty"`
}

// VerificationResultDTO 动态完整性比对结果响应
type VerificationResultDTO struct {
	RecordID        uint64 `json:"record_id"`
	RecordNo        string `json:"record_no"`
	IntegrityStatus string `json:"integrity_status"` // VERIFIED, TAMPERED
	IsIntact        bool   `json:"is_intact"`
	CalcHash        string `json:"calc_hash"`
	DBHash          string `json:"db_hash"`
	ChainHash       string `json:"chain_hash"`
	IPFSCID         string `json:"ipfs_cid"`
	ChainCID        string `json:"chain_cid"`
	FabricTxID      string `json:"fabric_tx_id"`
	BlockHeight     uint64 `json:"block_height"`
	TamperReason    string `json:"tamper_reason,omitempty"`
	VerifyTime      string `json:"verify_time"`
}

// CreateAuthorizationDTO 创建患者自主授权入参
type CreateAuthorizationDTO struct {
	AuthTargetType string `json:"auth_target_type" binding:"required"` // DOCTOR, HOSPITAL
	AuthTargetID   uint64 `json:"auth_target_id" binding:"required"`
	ScopeType      string `json:"scope_type" binding:"required"` // ALL, SINGLE
	RecordID       uint64 `json:"record_id"`
	Days           int    `json:"days"` // 有效天数 (如 7, 30)
}

// BreakGlassRequestDTO 医生触发破窗急诊调阅入参
type BreakGlassRequestDTO struct {
	RecordID        uint64 `json:"record_id" binding:"required"`
	EmergencyReason string `json:"emergency_reason" binding:"required"` // RESCUE, COMA, CRITICAL
	Description     string `json:"description" binding:"required"`      // 临床急救详细说明 (不少于15字)
	DoctorConfirmed int8   `json:"doctor_confirmed" binding:"required"` // 1-勾选法律责任声明
}

// PatientFeedbackDTO 患者对紧急访问知情反馈入参
type PatientFeedbackDTO struct {
	Feedback string `json:"feedback" binding:"required"` // CONFIRMED, OBJECTED
	Comment  string `json:"comment"`
}

// SupervisorAuditDTO 监管端紧急访问审核裁决入参
type SupervisorAuditDTO struct {
	Decision   string `json:"decision" binding:"required"` // CLOSED_APPROVED, CLOSED_VIOLATION
	Comment    string `json:"comment"`
	Punishment string `json:"punishment"` // NORMAL, RESTRICTED, DISABLED
}

// DashboardStatsDTO 监管大屏统计数据聚合对象
type DashboardStatsDTO struct {
	TotalTransactions  int64            `json:"total_transactions"`
	CurrentHeight      uint64           `json:"current_height"`
	TotalSharedCount   int64            `json:"total_shared_count"`
	HighRiskAlerts     int64            `json:"high_risk_alerts"`
	PendingEmergencies int64            `json:"pending_emergencies"`
	OrgStats           map[string]int64 `json:"org_stats"`
	RiskDistribution   map[string]int64 `json:"risk_distribution"`
}

// ── AI 跨期病历关联辅助诊断 DTO ───────────────────────────────────

// DiagnosticAssistRequest 医生发起跨院病历 SOAP 汇总与隐匿线索提取入参
type DiagnosticAssistRequest struct {
	PatientID       uint64 `json:"patient_id" binding:"required"`
	CurrentSymptoms string `json:"current_symptoms"`
	PresentIllness  string `json:"present_illness"`
	VitalSigns      string `json:"vital_signs"`
	DepartmentName  string `json:"department_name"`
}

// SOAPSummaryDTO 跨院病程 SOAP 结构化接诊摘要
type SOAPSummaryDTO struct {
	Subjective string `json:"subjective"` // S (患者主诉与既往病史汇总)
	Objective  string `json:"objective"`  // O (跨院客观检查与生命体征摘要)
	Assessment string `json:"assessment"` // A (各院既有临床诊断客观汇总，严禁做新诊断)
	Plan       string `json:"plan"`       // P (历史处置与已用药物记录)
}

// HiddenClueDTO 非结构化隐匿过敏与高危线索
type HiddenClueDTO struct {
	ClueType       string `json:"clue_type"`        // "疑似药物过敏史", "既往高危合并症", "特殊用药风险"
	Content        string `json:"content"`          // 详细提炼内容
	SourceRecordNo string `json:"source_record_no"` // 来源病历单号
	SourceHospital string `json:"source_hospital"`  // 来源机构
}

// SemanticContradictionDTO 跨院记录语义矛盾提示
type SemanticContradictionDTO struct {
	ContradictionType  string   `json:"contradiction_type"`  // "既往病史陈述矛盾", "用药依从性矛盾", "过敏史自述差异"
	Description        string   `json:"description"`         // 矛盾具体描述
	InvolvedRecords    []string `json:"involved_records"`    // 涉及的病历单号或机构
	VerificationAdvice string   `json:"verification_advice"` // 建议接诊医生当面核实要点
}

// TimelineCorrelationItem 跨期时序演变与时间间隔分析项
type TimelineCorrelationItem struct {
	RecordID         uint64 `json:"record_id"`
	RecordNo         string `json:"record_no"`
	VisitDate        string `json:"visit_date"`
	DaysAgo          int    `json:"days_ago"`
	IntervalFromPrev string `json:"interval_from_prev"` // 例如："相隔 120 天"、"相隔 18 天 (急剧缩短)"
	IntervalDays     int    `json:"interval_days"`
	HospitalName     string `json:"hospital_name"`
	DepartmentName   string `json:"department_name"`
	ChiefComplaint   string `json:"chief_complaint"`
	Diagnosis        string `json:"diagnosis"`
	TreatmentPlan    string `json:"treatment_plan"`
	ExamSummary      string `json:"exam_summary"`
	EvolutionStatus  string `json:"evolution_status"` // "平稳代偿", "间歇复发", "频发加剧", "急性恶化"
}

// SymptomCorrelationDetail 症状群聚与动态演进关联
type SymptomCorrelationDetail struct {
	CurrentAnalysis    string   `json:"current_analysis"`
	HistoricalLinkage  string   `json:"historical_linkage"`
	ProgressionPattern string   `json:"progression_pattern"`
	KeySymptomsTrack   []string `json:"key_symptoms_track"`
}

// MedicationCorrelationDetail 用药史与耐药/不良反应关联
type MedicationCorrelationDetail struct {
	PastMedications      []string `json:"past_medications"`
	DrugResponseAnalysis string   `json:"drug_response_analysis"`
	InteractionsAndRisks string   `json:"interactions_and_risks"`
}

// LabIndicatorItem 化验单与检查指标对比项
type LabIndicatorItem struct {
	ItemName      string `json:"item_name"`
	HistoryValue  string `json:"history_value"`
	Status        string `json:"status"` // "ABNORMAL", "WARNING", "NORMAL"
	ClinicalAlert string `json:"clinical_alert"`
}

// LabCorrelationDetail 检查化验报告交叉比对
type LabCorrelationDetail struct {
	AbnormalIndicators []LabIndicatorItem `json:"abnormal_indicators"`
	Summary            string             `json:"summary"`
}

// PredictedDiseaseItem 疑似疾病与鉴别诊断预测项
type PredictedDiseaseItem struct {
	DiseaseName        string   `json:"disease_name"`
	ICDCode            string   `json:"icd_code"`
	Probability        int      `json:"probability"`      // 0-100
	ConfidenceLevel    string   `json:"confidence_level"` // "高置信度", "重点疑似", "鉴别排除"
	RiskLevel          string   `json:"risk_level"`       // "CRITICAL", "HIGH", "MEDIUM", "LOW"
	ClinicalBasis      string   `json:"clinical_basis"`
	DifferentialPoints string   `json:"differential_points"`
	SuggestedExams     []string `json:"suggested_exams"`
}

// PrescriptionSuggestionItem 推荐处方用药项
type PrescriptionSuggestionItem struct {
	Category string `json:"category"`
	DrugName string `json:"drug_name"`
	Usage    string `json:"usage"`
}

// RecommendedTreatmentPlan 建议处置与用药
type RecommendedTreatmentPlan struct {
	Prescriptions     []PrescriptionSuggestionItem `json:"prescriptions"`
	TreatmentPlanText string                       `json:"treatment_plan_text"`
	SuggestedOrders   []string                     `json:"suggested_orders"`
}

// DiagnosticAssistResult 跨机构病历 SOAP 结构化摘要与隐匿线索提取综合输出
type DiagnosticAssistResult struct {
	PatientID              uint64                      `json:"patient_id"`
	PatientName            string                      `json:"patient_name"`
	PatientGender          string                      `json:"patient_gender"`
	PatientAge             int                         `json:"patient_age"`
	PatientNo              string                      `json:"patient_no"`
	AuthorizedRecordCount  int                         `json:"authorized_record_count"`
	TotalExamOrderCount    int                         `json:"total_exam_order_count"`
	AnalysisSummary        string                      `json:"analysis_summary"`
	SOAPSummary            SOAPSummaryDTO              `json:"soap_summary"`
	HiddenAllergyClues     []HiddenClueDTO             `json:"hidden_allergy_clues"`
	SemanticContradictions []SemanticContradictionDTO  `json:"semantic_contradictions"`
	PostVerificationGuard  string                      `json:"post_verification_guard"`
	TimelineCorrelation    []TimelineCorrelationItem   `json:"timeline_correlation"`
	SymptomCorrelation     SymptomCorrelationDetail    `json:"symptom_correlation"`
	MedicationCorrelation  MedicationCorrelationDetail `json:"medication_correlation"`
	LabCorrelation         LabCorrelationDetail        `json:"lab_correlation"`
	PredictedDiseases      []PredictedDiseaseItem      `json:"predicted_diseases"`
	RecommendedTreatment   RecommendedTreatmentPlan    `json:"recommended_treatment"`
	ClinicalWarnings       []string                    `json:"clinical_warnings"`
	RawPrompt              string                      `json:"raw_prompt"`
	EngineMode             string                      `json:"engine_mode"` // "EXTERNAL_API" or "BUILTIN_KNOWLEDGE_ENGINE"
	ModelName              string                      `json:"model_name"`
}

// InfectionRiskAlert 医护职业安全高危传染病预警结构体
type InfectionRiskAlert struct {
	HasRisk        bool                   `json:"has_risk"`
	RiskLevel      string                 `json:"risk_level"`      // CRITICAL (极高危), HIGH (高危), MEDIUM_HIGH (重点中高危), SAFE (安全)
	Summary        string                 `json:"summary"`         // 简述
	Diseases       []InfectionDiseaseInfo `json:"diseases"`        // 命中高危疾病明细
	Precautions    []string               `json:"precautions"`     // 医务人员标准及专项防护指引
	ProtectionGear []string               `json:"protection_gear"` // 必备个人防护装备 (PPE)
	EmergencySteps []string               `json:"emergency_steps"` // 职业暴露发生后的紧急处置处置规程 (一挤二冲三消毒 / 2h内PEP)
	DetectedFrom   []string               `json:"detected_from"`   // 检出来源病历号或检验报告摘要
}

// InfectionDiseaseInfo 单项传染病临床特征
type InfectionDiseaseInfo struct {
	Name           string `json:"name"`             // 艾滋病 (HIV/AIDS)
	Category       string `json:"category"`         // 血液/体液传染病
	Persistence    string `json:"persistence"`      // 终身携带 / 长期慢性携带
	ThreatToDoctor string `json:"threat_to_doctor"` // 锐器针刺伤暴露、血液黏膜喷溅
	Transmission   string `json:"transmission"`     // 血液、体液、锐器刺伤
	RiskLevel      string `json:"risk_level"`       // CRITICAL, HIGH, MEDIUM_HIGH
	MatchedKeyword string `json:"matched_keyword"`  // 命中的病历关键词
	RecordNo       string `json:"record_no"`        // 来源就诊编号
	HospitalName   string `json:"hospital_name"`    // 来源机构
	RecordDate     string `json:"record_date"`      // 诊断/确诊时间
}
