package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
)

type EmergencyService struct{}

var DefaultEmergencyService = &EmergencyService{}

// SubmitEmergencyAccess 提交 Break-Glass 紧急访问申请并直接生成事件上链
func (s *EmergencyService) SubmitEmergencyAccess(doctorID, recordID uint64, reason, desc string, confirmed bool, clientIP ...string) (*model.EmergencyAccessEvent, error) {
	ip := "SYSTEM_INTERNAL"
	if len(clientIP) > 0 && clientIP[0] != "" {
		ip = clientIP[0]
	}

	var doctor model.User
	if err := repository.DB.First(&doctor, doctorID).Error; err != nil {
		return nil, fmt.Errorf("医生不存在: %w", err)
	}

	var record model.MedicalRecord
	if err := repository.DB.First(&record, recordID).Error; err != nil {
		return nil, fmt.Errorf("病历不存在: %w", err)
	}

	// 生成事件单号
	randBytes := make([]byte, 3)
	_, _ = rand.Read(randBytes)
	eventNo := fmt.Sprintf("EA%s%s", time.Now().Format("20060102"), hex.EncodeToString(randBytes))

	// 上链存证
	txID, _, err := blockchain.DefaultService.CommitAsset("EMERGENCY_ACCESS", eventNo, map[string]interface{}{
		"event_no":    eventNo,
		"doctor_id":   fmt.Sprintf("%d", doctorID),
		"patient_id":  fmt.Sprintf("%d", record.PatientID),
		"record_id":   fmt.Sprintf("%d", recordID),
		"hospital_id": fmt.Sprintf("%d", doctor.HospitalID),
		"reason":      reason,
		"description": desc,
		"timestamp":   time.Now().Format(time.RFC3339),
		"status":      "PENDING_AUDIT",
	})
	if err != nil {
		return nil, fmt.Errorf("区块链记录失败: %w", err)
	}

	event := model.EmergencyAccessEvent{
		EventNo:          eventNo,
		DoctorID:         doctorID,
		PatientID:        record.PatientID,
		RecordID:         recordID,
		SourceHospitalID: doctor.HospitalID,
		TargetHospitalID: record.HospitalID,
		EmergencyReason:  reason,
		Description:      desc,
		DoctorConfirmed:  1,
		PatientFeedback:  "PENDING",
		AuditStatus:      "PENDING_AUDIT",
		FabricTxID:       txID,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := repository.DB.Create(&event).Error; err != nil {
		return nil, err
	}

	// 自动生成 24 小时紧急破窗临时通行授权，确保急救期间及随诊期间直接放行，重启服务依然生效
	authNo := fmt.Sprintf("AUTH_EMG%s%s", time.Now().Format("20060102"), hex.EncodeToString(randBytes))
	emgAuth := model.Authorization{
		AuthNo:         authNo,
		PatientID:      record.PatientID,
		AuthTargetType: "DOCTOR",
		AuthTargetID:   doctorID,
		ScopeType:      "SINGLE",
		RecordID:       recordID,
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(24 * time.Hour),
		Status:         "ACTIVE",
		FabricTxID:     txID,
		CreatedAt:      time.Now(),
	}
	_ = repository.DB.Create(&emgAuth)

	DefaultAuditService.Log(doctorID, "BREAK_GLASS", "EVENT", eventNo, doctor.HospitalID, "SUCCESS", "HIGH", ip)
	return &event, nil
}

// SubmitPatientFeedback 患者对紧急调阅做出知情确认或投诉
func (s *EmergencyService) SubmitPatientFeedback(patientID uint64, eventNo, feedback, comment string, clientIP ...string) error {
	ip := "SYSTEM_INTERNAL"
	if len(clientIP) > 0 && clientIP[0] != "" {
		ip = clientIP[0]
	}

	var event model.EmergencyAccessEvent
	if err := repository.DB.Where("event_no = ? AND patient_id = ?", eventNo, patientID).First(&event).Error; err != nil {
		return fmt.Errorf("未找到该紧急事件单或无权操作")
	}

	event.PatientFeedback = feedback
	if comment != "" {
		event.PatientComment = comment
	}
	event.UpdatedAt = time.Now()
	if err := repository.DB.Save(&event).Error; err != nil {
		return fmt.Errorf("更新事件反馈失败: %w", err)
	}

	// 若患者提出异议 (OBJECTED)，立即收回关联的 24h 紧急授权记录
	if feedback == "OBJECTED" {
		_ = repository.DB.Model(&model.Authorization{}).
			Where("patient_id = ? AND auth_target_id = ? AND record_id = ? AND auth_no LIKE 'AUTH_EMG%' AND status = 'ACTIVE'",
				event.PatientID, event.DoctorID, event.RecordID).
			Updates(map[string]interface{}{"status": "REVOKED", "revoke_reason": "患者对破窗行为提出异议，紧急放行即时终止"}).Error
	}

	DefaultAuditService.Log(patientID, "FEEDBACK", "EVENT", eventNo, event.TargetHospitalID, "SUCCESS", "LOW", ip)
	return nil
}

// AuditEvent 监管人员执行审核裁决并执行阶梯式惩戒
func (s *EmergencyService) AuditEvent(supervisorID uint64, eventNo, auditStatus, comment, punishment string, clientIP ...string) error {
	ip := "SYSTEM_INTERNAL"
	if len(clientIP) > 0 && clientIP[0] != "" {
		ip = clientIP[0]
	}

	var event model.EmergencyAccessEvent
	if err := repository.DB.Where("event_no = ?", eventNo).First(&event).Error; err != nil {
		return fmt.Errorf("未找到紧急事件单: %w", err)
	}

	event.AuditStatus = auditStatus
	event.SupervisorID = supervisorID
	event.AuditComment = comment
	event.UpdatedAt = time.Now()
	if err := repository.DB.Save(&event).Error; err != nil {
		return fmt.Errorf("保存审核结果失败: %w", err)
	}

	// 更新 Fabric 链上状态
	_, _, _ = blockchain.DefaultService.CommitAsset("EMERGENCY_AUDIT", eventNo, map[string]interface{}{
		"event_no":      eventNo,
		"supervisor_id": supervisorID,
		"audit_status":  auditStatus,
		"comment":       comment,
		"punishment":    punishment,
		"timestamp":     time.Now().Format(time.RFC3339),
	})

	// 执行惩戒联动 (NORMAL, RESTRICTED, DISABLED)
	if auditStatus == "CLOSED_VIOLATION" || (punishment != "" && punishment != "NORMAL") {
		_ = repository.DB.Model(&model.Authorization{}).
			Where("patient_id = ? AND auth_target_id = ? AND record_id = ? AND auth_no LIKE 'AUTH_EMG%' AND status = 'ACTIVE'",
				event.PatientID, event.DoctorID, event.RecordID).
			Updates(map[string]interface{}{"status": "REVOKED", "revoke_reason": "监管裁定违规破窗，收回紧急放行权限"}).Error
	}
	if punishment != "" && punishment != "NORMAL" {
		_ = repository.DB.Model(&model.User{}).Where("id = ?", event.DoctorID).Update("status", punishment).Error
		repository.InvalidateUserStatus(event.DoctorID)
	}

	DefaultAuditService.Log(supervisorID, "AUDIT_CLOSE", "EVENT", eventNo, 0, "SUCCESS", "LOW", ip)
	return nil
}

// LiftDoctorRestriction 监管人员一键解除对医生的处罚限制，恢复正常执业状态
func (s *EmergencyService) LiftDoctorRestriction(supervisorID, doctorID uint64, comment string, clientIP ...string) error {
	ip := "SYSTEM_INTERNAL"
	if len(clientIP) > 0 && clientIP[0] != "" {
		ip = clientIP[0]
	}

	var doc model.User
	if err := repository.DB.First(&doc, doctorID).Error; err != nil {
		return fmt.Errorf("医生不存在: %w", err)
	}

	if err := repository.DB.Model(&model.User{}).Where("id = ?", doctorID).Update("status", "NORMAL").Error; err != nil {
		return err
	}
	repository.InvalidateUserStatus(doctorID)

	DefaultAuditService.Log(supervisorID, "UNRESTRICT_DOCTOR", "USER", fmt.Sprintf("%d", doctorID), doc.HospitalID, "SUCCESS", "LOW", ip)
	return nil
}

// EvaluateRulePreScreen 关卡一：Go 纯代码客观合理性初筛 (0 Token 消耗，毫秒级判定)
func (s *EmergencyService) EvaluateRulePreScreen(event *model.EmergencyAccessEvent) (string, string) {
	reasonUpper := strings.ToUpper(strings.TrimSpace(event.EmergencyReason))
	descUpper := strings.ToUpper(strings.TrimSpace(event.Description))

	// 判断是否为明确的危急重症指征 (COMA, SHOCK, CARDIAC_ARREST, RESCUE, CRITICAL, 昏迷, 休克, 骤停)
	isCritical := strings.Contains(reasonUpper, "COMA") || strings.Contains(reasonUpper, "SHOCK") ||
		strings.Contains(reasonUpper, "CARDIAC") || strings.Contains(reasonUpper, "ARREST") ||
		strings.Contains(reasonUpper, "RESCUE") || strings.Contains(reasonUpper, "CRITICAL") ||
		strings.Contains(reasonUpper, "昏迷") || strings.Contains(reasonUpper, "休克") ||
		strings.Contains(reasonUpper, "呼吸衰竭") || strings.Contains(reasonUpper, "心跳骤停") ||
		strings.Contains(descUpper, "抢救") || strings.Contains(descUpper, "意识丧失") ||
		strings.Contains(descUpper, "心室颤动") || strings.Contains(descUpper, "大出血")

	// 检查是否在短期内跨多名患者频繁发起破窗 (防滥用刷单)
	var recentBursts int64
	if repository.DB != nil {
		oneHourAgo := event.CreatedAt.Add(-1 * time.Hour)
		repository.DB.Model(&model.EmergencyAccessEvent{}).
			Where("doctor_id = ? AND created_at >= ? AND created_at <= ?", event.DoctorID, oneHourAgo, event.CreatedAt).
			Count(&recentBursts)
	}

	// 若理由过于笼统 (如 OTHER, 其它, 调阅) 或 1小时内超过 8 次破窗，或患者已提出异议
	isSuspicious := strings.Contains(reasonUpper, "OTHER") || strings.Contains(reasonUpper, "其它") ||
		strings.Contains(reasonUpper, "其他") || strings.EqualFold(reasonUpper, "NORMAL") ||
		recentBursts > 8 || event.PatientFeedback == "OBJECTED"

	if isCritical && !isSuspicious {
		return "RULE_PASS_CRITICAL", "[规则初筛：符合急诊危急重症指征 (抢救指征明确，客观体征吻合)]"
	}

	if isSuspicious {
		if event.PatientFeedback == "OBJECTED" {
			return "RULE_WARN_SUSPICIOUS", "[规则初筛：患者已提出正式异议投诉，存在医患陈述重大分歧，建议重点复核]"
		}
		if recentBursts > 8 {
			return "RULE_WARN_SUSPICIOUS", fmt.Sprintf("[规则初筛：触发短时高频破窗预警 (1小时内发起 %d 次)，建议核实抢救真实性]", recentBursts)
		}
		return "RULE_WARN_SUSPICIOUS", "[规则初筛：体征/场景不典型或破窗原因笼统，建议重点复核临床记录]"
	}

	return "RULE_PASS_CRITICAL", "[规则初筛：符合临床紧急调阅常规避险指征]"
}

// GenerateAuditComparisonSummary 关卡二：AI 医患陈述争议对比辅助摘要 (仅客观列举争议焦点，裁决权保留在监管专员手中)
func (s *EmergencyService) GenerateAuditComparisonSummary(event *model.EmergencyAccessEvent) string {
	if event == nil {
		return ""
	}

	docClaim := event.Description
	if docClaim == "" {
		docClaim = fmt.Sprintf("紧急破窗原因登记为：%s，执行紧急调阅抢救。", event.EmergencyReason)
	}

	patientClaim := event.PatientComment
	if patientClaim == "" {
		if event.PatientFeedback == "OBJECTED" {
			patientClaim = "患者对本次紧急破窗调阅提出异议，声明就诊时段自身意识清醒未授权或未在该院急诊接受抢救。"
		} else {
			patientClaim = "患者暂未补充文字陈述（状态：等待知情核验）。"
		}
	}

	var sb strings.Builder
	sb.WriteString("【AI 医患陈述争议焦点客观对比】\n")
	sb.WriteString(fmt.Sprintf("• 【医生抢救陈述】：%s\n", docClaim))
	sb.WriteString(fmt.Sprintf("• 【患者异议申诉】：%s\n", patientClaim))
	sb.WriteString("• 【核心合规争议焦点】：\n")

	if strings.Contains(patientClaim, "未就诊") || strings.Contains(patientClaim, "未去") || strings.Contains(patientClaim, "没去") {
		sb.WriteString("  1. 就诊真实性分歧：医生主张急诊抢救调阅，患者声称当日并未前往调阅方医院，存在账号冒用或借道查阅嫌疑；\n")
	} else if strings.Contains(patientClaim, "清醒") || strings.Contains(patientClaim, "未告知") || strings.Contains(patientClaim, "不知情") {
		sb.WriteString("  1. 破窗程序合规分歧：患者主张当时神志清楚具备自主授权能力，医生未先行申请知情同意直接破窗；\n")
	} else {
		sb.WriteString("  1. 调阅必要性争议：患者对病历调阅的急迫性与范围提出异议，需复核当时生命体征；\n")
	}
	sb.WriteString("  2. 临床证据链要求：请监管人员核验当日急诊抢救流水号、急查心电图/生化检验开立时间戳与病程签名；\n")
	sb.WriteString("• 【监管裁决提示】：AI 严禁自动实施封号惩戒。请依据真实抢救证据，人工裁定【合规结案】或【确认违规并执行 RESTRICTED 权限降级】。")

	return sb.String()
}
