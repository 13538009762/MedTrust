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
	ScopeType      string `json:"scope_type" binding:"required"`       // ALL, SINGLE
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
	TotalTransactions int64            `json:"total_transactions"`
	CurrentHeight     uint64           `json:"current_height"`
	TotalSharedCount  int64            `json:"total_shared_count"`
	HighRiskAlerts    int64            `json:"high_risk_alerts"`
	PendingEmergencies int64           `json:"pending_emergencies"`
	OrgStats          map[string]int64 `json:"org_stats"`
	RiskDistribution  map[string]int64 `json:"risk_distribution"`
}
