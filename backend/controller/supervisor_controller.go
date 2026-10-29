package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
	"medtrust-backend/service"
)

type SupervisorController struct{}

var DefaultSupervisorController = &SupervisorController{}

func (ctrl *SupervisorController) Overview(c *gin.Context) {
	totalRecords := repository.DefaultMedicalRepository.CountTotal()
	totalEmergencies := repository.DefaultEmergencyRepository.CountTotal()
	pendingAudits := repository.DefaultEmergencyRepository.CountPendingAudits()
	totalLogs := repository.DefaultAuditRepository.CountTotal()

	totalTx, blockHeight := blockchain.DefaultService.GetStats()

	// 1. 统计不同机构病历占比
	hospStats := repository.DefaultAuditRepository.GetHospStats()

	// 2. 统计风险等级分布
	riskStats := repository.DefaultAuditRepository.GetRiskStats()

	// 3. 统计近7天全局流转量与风险拦截走势 (ECharts 折线/柱状图数据源)
	trendList := repository.DefaultAuditRepository.GetThroughputTrends(7)

	// 4. 统计防篡改核验通过率与拦截总量
	var tamperedCount int64
	allRecs, _ := repository.DefaultMedicalRepository.ListAllRecords()
	for i := range allRecs {
		service.DefaultMedicalService.VerifyRecord(&allRecs[i])
		if allRecs[i].IsTampered {
			tamperedCount++
		}
	}
	verifiedCount := totalRecords - tamperedCount
	passRate := 100.0
	if totalRecords > 0 {
		passRate = float64(verifiedCount) / float64(totalRecords) * 100.0
		if passRate < 0 {
			passRate = 0
		}
	}

	totalIntercepted := repository.DefaultAuditRepository.CountIntercepted()

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "查询成功",
		Data: gin.H{
			"total_records":          totalRecords,
			"total_emergencies":      totalEmergencies,
			"pending_audits":         pendingAudits,
			"total_logs":             totalLogs,
			"total_tx":               totalTx,
			"block_height":           blockHeight,
			"hosp_stats":             hospStats,
			"risk_stats":             riskStats,
			"throughput_trend":       trendList,
			"total_intercepted":      totalIntercepted,
			"tampered_count":         tamperedCount,
			"verified_count":         verifiedCount,
			"verification_pass_rate": passRate,
		},
	})
}

func (ctrl *SupervisorController) ListEmergencyEvents(c *gin.Context) {
	status := c.Query("status")
	role := c.GetString("role")
	userID := c.GetUint64("user_id")
	rawList, _ := repository.DefaultEmergencyRepository.ListEventsWithFilter(status)

	var list []model.EmergencyAccessEvent
	for _, ev := range rawList {
		if role == "patient" && ev.PatientID != userID {
			continue
		}
		list = append(list, ev)
	}

	for i := range list {
		if doc, err := repository.DefaultUserRepository.FindByID(list[i].DoctorID); err == nil {
			list[i].DoctorName = doc.RealName
		}
		if pat, err := repository.DefaultUserRepository.FindByID(list[i].PatientID); err == nil {
			list[i].PatientName = pat.RealName
		}
		if rec, err := repository.DefaultMedicalRepository.FindRecordByID(list[i].RecordID); err == nil {
			list[i].RecordNo = rec.RecordNo
		}
		if h1, err := repository.DefaultHospitalRepository.FindByID(list[i].SourceHospitalID); err == nil {
			list[i].SourceHospitalName = h1.Name
		}
		if h2, err := repository.DefaultHospitalRepository.FindByID(list[i].TargetHospitalID); err == nil {
			list[i].TargetHospitalName = h2.Name
		}

		// 第一道关卡：Go 纯代码客观合理性初筛
		code, desc := service.DefaultEmergencyService.EvaluateRulePreScreen(&list[i])
		list[i].RulePreScreen = code
		list[i].RulePreScreenDesc = desc

		// 第二道关卡：当触发异议或存疑时，生成 AI 医患陈述争议对比摘要
		if list[i].PatientFeedback == "OBJECTED" || code == "RULE_WARN_SUSPICIOUS" {
			list[i].AuditComparisonSummary = service.DefaultEmergencyService.GenerateAuditComparisonSummary(&list[i])
		}
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "查询成功", Data: list})
}

type AuditEmergencyRequest struct {
	AuditStatus  string `json:"audit_status"`
	Decision     string `json:"decision"`
	AuditComment string `json:"audit_comment"`
	Comment      string `json:"comment"`
	Punishment   string `json:"punishment"` // NORMAL, RESTRICTED, DISABLED
}

func (ctrl *SupervisorController) AuditEmergencyEvent(c *gin.Context) {
	supervisorID := c.GetUint64("user_id")
	eventNo := c.Param("event_no")

	var req AuditEmergencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误: " + err.Error()})
		return
	}

	status := req.AuditStatus
	if status == "" {
		status = req.Decision
	}
	comment := req.AuditComment
	if comment == "" {
		comment = req.Comment
	}
	if status == "" {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "缺少审核裁决状态 (audit_status 或 decision)"})
		return
	}

	punishment := req.Punishment
	if punishment == "" && status == "CLOSED_VIOLATION" {
		punishment = "RESTRICTED"
	}

	if err := service.DefaultEmergencyService.AuditEvent(supervisorID, eventNo, status, comment, punishment, c.ClientIP()); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "审核裁决与处置已固化生效"})
}

func (ctrl *SupervisorController) ListAuditLogs(c *gin.Context) {
	opType := c.Query("operation_type")
	riskLevel := c.Query("risk_level")
	currentUserID := c.GetUint64("user_id")
	role := c.GetString("role")

	var filter *repository.PatientAuditFilter
	// 患者角色数据隔离: 仅允许调阅涉及本人的访问痕迹与全生命周期追踪 ("谁访问过我的数据")
	if role == "patient" {
		pat, _ := repository.DefaultUserRepository.FindByID(currentUserID)
		userNo := ""
		if pat != nil {
			userNo = pat.UserNo
		}

		recordNos, recordIDs := repository.DefaultMedicalRepository.GetPatientRecordIdentifiers(currentUserID)
		authNos := repository.DefaultAuthPolicyRepository.GetPatientAuthNos(currentUserID)
		eventNos := repository.DefaultEmergencyRepository.GetPatientEventNos(currentUserID)
		requestNos := repository.DefaultAuthPolicyRepository.GetPatientRequestNos(currentUserID)

		var recIdStrs []string
		for _, rid := range recordIDs {
			recIdStrs = append(recIdStrs, fmt.Sprintf("%d", rid), fmt.Sprintf("REC_%d", rid))
		}
		allRecordTargets := append(recordNos, recIdStrs...)

		if len(allRecordTargets) == 0 {
			allRecordTargets = []string{"__EMPTY__"}
		}
		if len(authNos) == 0 {
			authNos = []string{"__EMPTY__"}
		}
		if len(eventNos) == 0 {
			eventNos = []string{"__EMPTY__"}
		}
		if len(requestNos) == 0 {
			requestNos = []string{"__EMPTY__"}
		}

		filter = &repository.PatientAuditFilter{
			PatientID:     currentUserID,
			UserNo:        userNo,
			RecordTargets: allRecordTargets,
			AuthNos:       authNos,
			EventNos:      eventNos,
			RequestNos:    requestNos,
		}
	}

	list, _ := repository.DefaultAuditRepository.ListAuditLogsWithFilter(opType, riskLevel, filter, 100)

	if len(list) > 0 {
		userIDs := make([]uint64, 0, len(list))
		hospIDs := make([]uint64, 0, len(list))
		for _, item := range list {
			if item.UserID > 0 {
				userIDs = append(userIDs, item.UserID)
			}
			if item.HospitalID > 0 {
				hospIDs = append(hospIDs, item.HospitalID)
			}
		}

		userMap := loadUsersBatch(userIDs)
		hospMap := loadHospitalsBatch(hospIDs)

		for i := range list {
			if u, ok := userMap[list[i].UserID]; ok {
				list[i].UserName = u.RealName
			}
			if h, ok := hospMap[list[i].HospitalID]; ok {
				list[i].HospitalName = h.Name
			}
		}
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "查询成功", Data: list})
}

func (ctrl *SupervisorController) VerifyRecord(c *gin.Context) {
	recordID, _ := strconv.ParseUint(c.Param("record_id"), 10, 64)
	res, err := service.DefaultVerificationService.Verify(recordID, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "防篡改核验完成", Data: res})
}

// SimulateTamper 真实触发数据库篡改演练 (答辩核心演示亮点)
func (ctrl *SupervisorController) SimulateTamper(c *gin.Context) {
	recordID, _ := strconv.ParseUint(c.Param("record_id"), 10, 64)
	res, err := service.DefaultVerificationService.SimulateTamper(recordID, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "【演示警报】数据恶意篡改演练已触发！系统动态验真检测到哈希不匹配，已亮起红标报警！",
		Data:    res,
	})
}

// RestoreTamperedRecord 一键恢复真实病历数据
func (ctrl *SupervisorController) RestoreTamperedRecord(c *gin.Context) {
	recordID, _ := strconv.ParseUint(c.Param("record_id"), 10, 64)
	res, err := service.DefaultVerificationService.RestoreTamperedRecord(recordID, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "数据已成功一键恢复！重新计算哈希与 Fabric 链上指纹恢复一致，绿标通过！",
		Data:    res,
	})
}

type LiftDoctorRestrictionDTO struct {
	DoctorID uint64 `json:"doctor_id" binding:"required"`
	Comment  string `json:"comment"`
}

// LiftDoctorRestriction 监管人员一键解除医生受限惩戒
func (ctrl *SupervisorController) LiftDoctorRestriction(c *gin.Context) {
	supervisorID := c.GetUint64("user_id")
	var req LiftDoctorRestrictionDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误: " + err.Error()})
		return
	}

	if err := service.DefaultEmergencyService.LiftDoctorRestriction(supervisorID, req.DoctorID, req.Comment, c.ClientIP()); err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "已成功解除该医生的权限限制，状态恢复为 NORMAL 正常"})
}

// AccessHeatmap 统计近7天×24小时的全网调阅行为热力图矩阵
func (ctrl *SupervisorController) AccessHeatmap(c *gin.Context) {
	result := repository.DefaultAuditRepository.GetAccessHeatmapMatrix()

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "获取全网调阅行为时空热力分布成功",
		Data:    result,
	})
}

// HospitalFlow 统计机构间跨院医疗数据共享流向矩阵 (供 Sankey 桑基图)
func (ctrl *SupervisorController) HospitalFlow(c *gin.Context) {
	flows := repository.DefaultAuditRepository.GetHospitalFlows()

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "获取跨机构数据流向图数据成功",
		Data:    flows,
	})
}

// AuthFunnel 统计患者自主知情授权闭环漏斗数据
func (ctrl *SupervisorController) AuthFunnel(c *gin.Context) {
	totalRequests, handledRequests, approvedRequests, activeAuths := repository.DefaultAuditRepository.GetAuthFunnelStats()

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "获取授权闭环转化漏斗数据成功",
		Data: gin.H{
			"total":    totalRequests,
			"handled":  handledRequests,
			"approved": approvedRequests,
			"active":   activeAuths,
		},
	})
}

// ListBlockchainTransactions 获取区块链浏览器存证交易列表 (支撑监管端区块链浏览器真实流水)
func (ctrl *SupervisorController) ListBlockchainTransactions(c *gin.Context) {
	page, pageSize, offset := parsePagination(c, 20)

	businessType := c.Query("business_type")
	txID := c.Query("tx_id")
	status := c.Query("status")

	list, total, _ := repository.DefaultAuditRepository.ListBlockchainTransactions(businessType, txID, status, offset, pageSize)

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "获取区块链浏览器存证交易流水成功",
		Data: gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"list":      list,
		},
	})
}
