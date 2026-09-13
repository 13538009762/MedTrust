package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// MedicalContract 医疗数据存证与共享控制智能合约
type MedicalContract struct {
	contractapi.Contract
}

// MedicalRecord 符合极简链上存证原则的病历记录 (密文存 IPFS)
type MedicalRecord struct {
	RecordID   string `json:"record_id"`
	CID        string `json:"cid"`
	FileHash   string `json:"file_hash"`
	HospitalID string `json:"hospital_id"`
	DataType   string `json:"data_type"`
	Timestamp  string `json:"timestamp"`
}

// Authorization 符合链上原则的患者授权存证
type Authorization struct {
	AuthorizationID    string `json:"authorization_id"`
	PatientID          string `json:"patient_id"`
	DoctorID           string `json:"doctor_id"`
	HospitalID         string `json:"hospital_id"`
	AuthorizationScope string `json:"authorization_scope"`
	ExpireTime         string `json:"expire_time"`
	Status             string `json:"status"`
}

// AuditLog 符合链上原则的审计日志存证
type AuditLog struct {
	AuditID   string `json:"audit_id"`
	UserID    string `json:"user_id"`
	Action    string `json:"action"`
	RecordID  string `json:"record_id"`
	RiskLevel string `json:"risk_level"`
	Timestamp string `json:"timestamp"`
}

// MedicalAsset 医疗数据上链存证资产 (向前兼容)
type MedicalAsset struct {
	RecordID     string `json:"record_id"`     // 病历业务流水号 (如 ENC2026...)
	PatientID    string `json:"patient_id"`    // 患者编号
	CID          string `json:"cid"`           // IPFS 密文寻址哈希 (如 Qm...)
	FileHash     string `json:"file_hash"`     // 原始文件明文 SHA-256 指纹
	ClinicalHash string `json:"clinical_hash"` // 临床病历多维结构化 SHA-256 综合摘要
	HospitalID   string `json:"hospital_id"`   // 开具医院标识
	CreatorID    string `json:"creator_id"`    // 签署医生标识
	DataType     string `json:"data_type"`     // EMR, REPORT, IMAGE
	CreateTime   string `json:"create_time"`   // 上链时间戳
}


// AuthorizationRecord 患者授权上链存证
type AuthorizationRecord struct {
	AuthNo     string `json:"auth_no"`
	PatientID  string `json:"patient_id"`
	TargetType string `json:"target_type"` // DOCTOR, HOSPITAL
	TargetID   string `json:"target_id"`
	Scope      string `json:"scope"` // ALL, SINGLE
	RecordID   string `json:"record_id"`
	Status     string `json:"status"` // ACTIVE, REVOKED
	Timestamp  string `json:"timestamp"`
}

// EmergencyAccessRecord 紧急破窗访问上链凭据
type EmergencyAccessRecord struct {
	EventNo      string `json:"event_no"`
	DoctorID     string `json:"doctor_id"`
	PatientID    string `json:"patient_id"`
	RecordID     string `json:"record_id"`
	HospitalID   string `json:"hospital_id"`
	Reason       string `json:"reason"`
	Description  string `json:"description"`
	Timestamp    string `json:"timestamp"`
	Status       string `json:"status"` // PENDING_AUDIT, UNDER_INVESTIGATION, CLOSED_APPROVED, CLOSED_VIOLATION
	SupervisorID string `json:"supervisor_id,omitempty"`
	AuditComment string `json:"audit_comment,omitempty"`
	Punishment   string `json:"punishment,omitempty"`
	AuditTime    string `json:"audit_time,omitempty"`
}

// AuditRecord 全局不可篡改审计凭据
type AuditRecord struct {
	LogID      string `json:"log_id"`
	UserID     string `json:"user_id"`
	OpType     string `json:"op_type"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	HospitalID string `json:"hospital_id"`
	Result     string `json:"result"`
	RiskLevel  string `json:"risk_level"`
	Timestamp  string `json:"timestamp"`
}

// CreateMedicalAsset 提交病历存证至分布式账本世界状态
func (c *MedicalContract) CreateMedicalAsset(ctx contractapi.TransactionContextInterface, assetJSON string) error {
	var asset MedicalAsset
	if err := json.Unmarshal([]byte(assetJSON), &asset); err != nil {
		return fmt.Errorf("反序列化病历资产失败: %v", err)
	}

	exists, err := ctx.GetStub().GetState("RECORD_" + asset.RecordID)
	if err != nil {
		return fmt.Errorf("读取账本状态失败: %v", err)
	}
	if exists != nil {
		return fmt.Errorf("病历资产 %s 已存在于账本中，不可重复创建", asset.RecordID)
	}

	assetBytes, _ := json.Marshal(asset)
	return ctx.GetStub().PutState("RECORD_"+asset.RecordID, assetBytes)
}

// QueryMedicalAsset 查询病历链上存证摘要
func (c *MedicalContract) QueryMedicalAsset(ctx contractapi.TransactionContextInterface, recordID string) (*MedicalAsset, error) {
	assetBytes, err := ctx.GetStub().GetState("RECORD_" + recordID)
	if err != nil {
		return nil, fmt.Errorf("读取账本失败: %v", err)
	}
	if assetBytes == nil {
		return nil, fmt.Errorf("病历资产 %s 不存在于账本中", recordID)
	}

	var asset MedicalAsset
	if err := json.Unmarshal(assetBytes, &asset); err != nil {
		return nil, err
	}
	return &asset, nil
}

// CreateAuthorizationRecord 记录患者知情授权策略
func (c *MedicalContract) CreateAuthorizationRecord(ctx contractapi.TransactionContextInterface, authJSON string) error {
	var auth AuthorizationRecord
	if err := json.Unmarshal([]byte(authJSON), &auth); err != nil {
		return fmt.Errorf("反序列化授权记录失败: %v", err)
	}

	authBytes, _ := json.Marshal(auth)
	return ctx.GetStub().PutState("AUTH_"+auth.AuthNo, authBytes)
}

// RevokeAuthorizationRecord 撤销患者授权策略
func (c *MedicalContract) RevokeAuthorizationRecord(ctx contractapi.TransactionContextInterface, authNo string) error {
	authBytes, err := ctx.GetStub().GetState("AUTH_" + authNo)
	if err != nil || authBytes == nil {
		return fmt.Errorf("授权条目 %s 不存在", authNo)
	}

	var auth AuthorizationRecord
	_ = json.Unmarshal(authBytes, &auth)
	auth.Status = "REVOKED"

	updatedBytes, _ := json.Marshal(auth)
	return ctx.GetStub().PutState("AUTH_"+authNo, updatedBytes)
}

// CreateEmergencyRecord 记录 Break-Glass 紧急访问事件单
func (c *MedicalContract) CreateEmergencyRecord(ctx contractapi.TransactionContextInterface, eventJSON string) error {
	var record EmergencyAccessRecord
	if err := json.Unmarshal([]byte(eventJSON), &record); err != nil {
		return fmt.Errorf("反序列化紧急访问记录失败: %v", err)
	}

	recordBytes, _ := json.Marshal(record)
	return ctx.GetStub().PutState("EMERGENCY_"+record.EventNo, recordBytes)
}

// UpdateEmergencyStatus 监管人员更新紧急访问审核裁决状态
func (c *MedicalContract) UpdateEmergencyStatus(ctx contractapi.TransactionContextInterface, eventNo, status, supervisorID, comment, punishment, auditTime string) error {
	recordBytes, err := ctx.GetStub().GetState("EMERGENCY_" + eventNo)
	if err != nil || recordBytes == nil {
		return fmt.Errorf("紧急事件单 %s 不存在", eventNo)
	}

	var record EmergencyAccessRecord
	_ = json.Unmarshal(recordBytes, &record)
	record.Status = status
	record.SupervisorID = supervisorID
	record.AuditComment = comment
	record.Punishment = punishment
	record.AuditTime = auditTime

	updatedBytes, _ := json.Marshal(record)
	return ctx.GetStub().PutState("EMERGENCY_"+eventNo, updatedBytes)
}

// CreateMedicalRecord 提交符合链上原则的轻量病历存证 (仅存证摘要与指纹，密文存 IPFS)
func (c *MedicalContract) CreateMedicalRecord(ctx contractapi.TransactionContextInterface, recordID, cid, fileHash, hospitalID, dataType, timestamp string) error {
	rec := MedicalRecord{
		RecordID:   recordID,
		CID:        cid,
		FileHash:   fileHash,
		HospitalID: hospitalID,
		DataType:   dataType,
		Timestamp:  timestamp,
	}
	recBytes, _ := json.Marshal(rec)
	return ctx.GetStub().PutState("RECORD_"+recordID, recBytes)
}

// QueryMedicalRecord 查询病历链上存证摘要
func (c *MedicalContract) QueryMedicalRecord(ctx contractapi.TransactionContextInterface, recordID string) (*MedicalRecord, error) {
	val, err := ctx.GetStub().GetState("RECORD_" + recordID)
	if err != nil {
		return nil, fmt.Errorf("读取账本记录失败: %v", err)
	}
	if val == nil {
		return nil, fmt.Errorf("病历记录 %s 在账本中不存在", recordID)
	}
	var rec MedicalRecord
	if err := json.Unmarshal(val, &rec); err != nil {
		return nil, fmt.Errorf("解析病历记录失败: %v", err)
	}
	return &rec, nil
}

// CreateAuthorization 记录患者知情授权策略
func (c *MedicalContract) CreateAuthorization(ctx contractapi.TransactionContextInterface, authorizationID, patientID, doctorID, hospitalID, authorizationScope, expireTime, status string) error {
	auth := Authorization{
		AuthorizationID:    authorizationID,
		PatientID:          patientID,
		DoctorID:           doctorID,
		HospitalID:         hospitalID,
		AuthorizationScope: authorizationScope,
		ExpireTime:         expireTime,
		Status:             status,
	}
	authBytes, _ := json.Marshal(auth)
	return ctx.GetStub().PutState("AUTH_"+authorizationID, authBytes)
}

// RevokeAuthorization 撤销患者授权策略
func (c *MedicalContract) RevokeAuthorization(ctx contractapi.TransactionContextInterface, authorizationID string) error {
	val, err := ctx.GetStub().GetState("AUTH_" + authorizationID)
	if err != nil || val == nil {
		return fmt.Errorf("授权条目 %s 不存在", authorizationID)
	}
	var auth Authorization
	_ = json.Unmarshal(val, &auth)
	auth.Status = "REVOKED"
	updatedBytes, _ := json.Marshal(auth)
	return ctx.GetStub().PutState("AUTH_"+authorizationID, updatedBytes)
}

// CreateAuditLog 记录系统敏感操作全量审计存证
func (c *MedicalContract) CreateAuditLog(ctx contractapi.TransactionContextInterface, auditID, userID, action, recordID, riskLevel, timestamp string) error {
	logEntry := AuditLog{
		AuditID:   auditID,
		UserID:    userID,
		Action:    action,
		RecordID:  recordID,
		RiskLevel: riskLevel,
		Timestamp: timestamp,
	}
	logBytes, _ := json.Marshal(logEntry)
	return ctx.GetStub().PutState("AUDIT_"+auditID, logBytes)
}

// QueryAuditLogs 查询链上审计日志 (支持按 recordID 过滤或返回全部)
func (c *MedicalContract) QueryAuditLogs(ctx contractapi.TransactionContextInterface, recordID string) ([]*AuditLog, error) {
	iterator, err := ctx.GetStub().GetStateByRange("AUDIT_", "AUDIT_\uffff")
	if err != nil {
		return nil, fmt.Errorf("查询审计日志失败: %v", err)
	}
	defer iterator.Close()

	var logs []*AuditLog
	for iterator.HasNext() {
		item, err := iterator.Next()
		if err != nil {
			continue
		}
		var logEntry AuditLog
		if err := json.Unmarshal(item.Value, &logEntry); err == nil {
			if recordID == "" || recordID == "ALL" || logEntry.RecordID == recordID {
				logs = append(logs, &logEntry)
			}
		}
	}
	return logs, nil
}

// CreateAuditRecord 记录系统敏感操作全量审计存证 (向前兼容)
func (c *MedicalContract) CreateAuditRecord(ctx contractapi.TransactionContextInterface, auditJSON string) error {
	var audit AuditRecord
	if err := json.Unmarshal([]byte(auditJSON), &audit); err != nil {
		return fmt.Errorf("反序列化审计日志记录失败: %v", err)
	}

	auditBytes, _ := json.Marshal(audit)
	return ctx.GetStub().PutState("AUDIT_"+audit.LogID, auditBytes)
}

