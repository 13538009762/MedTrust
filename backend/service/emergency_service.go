package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/repository"
)

type EmergencyService struct{}

var DefaultEmergencyService = &EmergencyService{}

// SubmitEmergencyAccess 提交 Break-Glass 紧急访问申请并直接生成事件上链
func (s *EmergencyService) SubmitEmergencyAccess(doctorID, recordID uint64, reason, desc string, confirmed bool) (*model.EmergencyAccessEvent, error) {
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
		"doctor_id":   doctorID,
		"patient_id":  record.PatientID,
		"record_id":   recordID,
		"hospital_id": doctor.HospitalID,
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

	DefaultAuditService.Log(doctorID, "BREAK_GLASS", "EVENT", eventNo, doctor.HospitalID, "SUCCESS", "HIGH", "127.0.0.1")
	return &event, nil
}

// SubmitPatientFeedback 患者对紧急调阅做出知情确认或投诉
func (s *EmergencyService) SubmitPatientFeedback(patientID uint64, eventNo, feedback, comment string) error {
	var event model.EmergencyAccessEvent
	if err := repository.DB.Where("event_no = ? AND patient_id = ?", eventNo, patientID).First(&event).Error; err != nil {
		return fmt.Errorf("未找到该紧急事件单或无权操作")
	}

	event.PatientFeedback = feedback
	if comment != "" {
		event.Description = fmt.Sprintf("%s | [患者反馈: %s]", event.Description, comment)
	}
	event.UpdatedAt = time.Now()
	repository.DB.Save(&event)

	DefaultAuditService.Log(patientID, "FEEDBACK", "EVENT", eventNo, event.TargetHospitalID, "SUCCESS", "LOW", "127.0.0.1")
	return nil
}

// AuditEvent 监管人员执行审核裁决并执行阶梯式惩戒
func (s *EmergencyService) AuditEvent(supervisorID uint64, eventNo, auditStatus, comment, punishment string) error {
	var event model.EmergencyAccessEvent
	if err := repository.DB.Where("event_no = ?", eventNo).First(&event).Error; err != nil {
		return fmt.Errorf("未找到紧急事件单: %w", err)
	}

	event.AuditStatus = auditStatus
	event.SupervisorID = supervisorID
	event.AuditComment = comment
	event.UpdatedAt = time.Now()
	repository.DB.Save(&event)

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
	if punishment != "" && punishment != "NORMAL" {
		repository.DB.Model(&model.User{}).Where("id = ?", event.DoctorID).Update("status", punishment)
	}

	DefaultAuditService.Log(supervisorID, "AUDIT_CLOSE", "EVENT", eventNo, 0, "SUCCESS", "LOW", "127.0.0.1")
	return nil
}

// LiftDoctorRestriction 监管人员一键解除对医生的处罚限制，恢复正常执业状态
func (s *EmergencyService) LiftDoctorRestriction(supervisorID, doctorID uint64, comment string) error {
	var doc model.User
	if err := repository.DB.First(&doc, doctorID).Error; err != nil {
		return fmt.Errorf("医生不存在: %w", err)
	}

	if err := repository.DB.Model(&model.User{}).Where("id = ?", doctorID).Update("status", "NORMAL").Error; err != nil {
		return err
	}

	DefaultAuditService.Log(supervisorID, "UNRESTRICT_DOCTOR", "USER", fmt.Sprintf("%d", doctorID), doc.HospitalID, "SUCCESS", "LOW", "127.0.0.1")
	return nil
}

