package main

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// mspIDPattern 联盟成员 MSP 标识格式约定 (如 Org1MSP / HospitalAMSP)
var mspIDPattern = regexp.MustCompile(`^[A-Za-z0-9]+MSP$`)

// stampIdentityFromContext (WP-11) 从交易上下文提取提交方 MSP 归属与客户端身份，
// 校验 MSP 非空且符合联盟注册格式后盖章到账本状态，客户端载荷无法 spoof 组织来源。
func stampIdentityFromContext(ctx contractapi.TransactionContextInterface, endorserMSPID, clientIdentityID *string) error {
	mspID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("提取提交方 MSP 身份失败: %v", err)
	}
	if mspID == "" {
		return fmt.Errorf("交易提交方 MSP ID 为空，拒绝确权存证")
	}
	if !mspIDPattern.MatchString(mspID) {
		return fmt.Errorf("提交方 MSP ID %q 不符合联盟注册 MSP 格式 (^[A-Za-z0-9]+MSP$)，拒绝确权存证", mspID)
	}
	*endorserMSPID = mspID

	if clientID, idErr := ctx.GetClientIdentity().GetID(); idErr == nil && clientID != "" {
		*clientIdentityID = clientID
	}
	return nil
}

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

	// WP-11: 链上 MSP 背书身份确权——由链码从 ClientIdentity 提取并盖章，
	// 客户端载荷无法伪造组织来源归属
	EndorserMSPID    string `json:"endorser_msp_id,omitempty" metadata:",optional"`
	ClientIdentityID string `json:"client_identity_id,omitempty" metadata:",optional"`
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

	// WP-11: 链码侧调用者证书身份强绑定，客户端载荷无法伪造归属
	EndorserMSPID    string `json:"endorser_msp_id,omitempty" metadata:",optional"`
	ClientIdentityID string `json:"client_identity_id,omitempty" metadata:",optional"`
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
	SupervisorID string `json:"supervisor_id,omitempty" metadata:",optional"`
	AuditComment string `json:"audit_comment,omitempty" metadata:",optional"`
	Punishment   string `json:"punishment,omitempty" metadata:",optional"`
	AuditTime    string `json:"audit_time,omitempty" metadata:",optional"`

	// WP-11: 链码侧调用者证书身份强绑定
	EndorserMSPID    string `json:"endorser_msp_id,omitempty" metadata:",optional"`
	ClientIdentityID string `json:"client_identity_id,omitempty" metadata:",optional"`
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

	// WP-11: 链码侧调用者证书身份强绑定
	EndorserMSPID    string `json:"endorser_msp_id,omitempty" metadata:",optional"`
	ClientIdentityID string `json:"client_identity_id,omitempty" metadata:",optional"`
}

// CreateMedicalAsset 提交病历存证至分布式账本世界状态
func (c *MedicalContract) CreateMedicalAsset(ctx contractapi.TransactionContextInterface, assetJSON string) error {
	var asset MedicalAsset
	if err := json.Unmarshal([]byte(assetJSON), &asset); err != nil {
		return fmt.Errorf("反序列化病历资产失败: %v", err)
	}

	// WP-11: 拒绝客户端载荷自带的伪造 MSP 归属字段，身份以链码侧提取为准
	asset.EndorserMSPID = ""
	asset.ClientIdentityID = ""

	exists, err := ctx.GetStub().GetState("RECORD_" + asset.RecordID)
	if err != nil {
		return fmt.Errorf("读取账本状态失败: %v", err)
	}
	if exists != nil {
		return fmt.Errorf("病历资产 %s 已存在于账本中，不可重复创建", asset.RecordID)
	}

	// WP-11: 从交易提交方 ClientIdentity 提取并盖章 MSP 归属，客户端无法伪造
	if err := stampIdentityFromContext(ctx, &asset.EndorserMSPID, &asset.ClientIdentityID); err != nil {
		return err
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

	// WP-11: 拒绝客户端伪造，链码侧盖章调用者身份
	auth.EndorserMSPID = ""
	auth.ClientIdentityID = ""
	if err := stampIdentityFromContext(ctx, &auth.EndorserMSPID, &auth.ClientIdentityID); err != nil {
		return err
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

	// WP-11: 撤销操作重新盖章当前调用者身份
	auth.EndorserMSPID = ""
	auth.ClientIdentityID = ""
	if err := stampIdentityFromContext(ctx, &auth.EndorserMSPID, &auth.ClientIdentityID); err != nil {
		return err
	}

	updatedBytes, _ := json.Marshal(auth)
	return ctx.GetStub().PutState("AUTH_"+authNo, updatedBytes)
}

// CreateEmergencyRecord 记录 Break-Glass 紧急访问事件单
func (c *MedicalContract) CreateEmergencyRecord(ctx contractapi.TransactionContextInterface, eventJSON string) error {
	var record EmergencyAccessRecord
	if err := json.Unmarshal([]byte(eventJSON), &record); err != nil {
		return fmt.Errorf("反序列化紧急访问记录失败: %v", err)
	}

	// WP-11: 链码侧盖章调用者身份
	record.EndorserMSPID = ""
	record.ClientIdentityID = ""
	if err := stampIdentityFromContext(ctx, &record.EndorserMSPID, &record.ClientIdentityID); err != nil {
		return err
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

	// WP-11: 监管裁决操作重新盖章当前调用者身份
	record.EndorserMSPID = ""
	record.ClientIdentityID = ""
	if err := stampIdentityFromContext(ctx, &record.EndorserMSPID, &record.ClientIdentityID); err != nil {
		return err
	}

	updatedBytes, _ := json.Marshal(record)
	return ctx.GetStub().PutState("EMERGENCY_"+eventNo, updatedBytes)
}

// CreateMedicalRecord 提交符合链上原则的轻量病历存证 (统一映射至 MedicalAsset 统一资产模型)
func (c *MedicalContract) CreateMedicalRecord(ctx contractapi.TransactionContextInterface, recordID, cid, fileHash, hospitalID, dataType, timestamp string) error {
	asset := MedicalAsset{
		RecordID:     recordID,
		CID:          cid,
		FileHash:     fileHash,
		ClinicalHash: fileHash,
		HospitalID:   hospitalID,
		DataType:     dataType,
		CreateTime:   timestamp,
	}
	// WP-11: 盖章链码侧提取的 MSP 背书身份
	if err := stampIdentityFromContext(ctx, &asset.EndorserMSPID, &asset.ClientIdentityID); err != nil {
		return err
	}
	assetBytes, _ := json.Marshal(asset)
	return ctx.GetStub().PutState("RECORD_"+recordID, assetBytes)
}

// QueryMedicalRecord 查询病历链上存证摘要 (统一返回 MedicalAsset 结构)
func (c *MedicalContract) QueryMedicalRecord(ctx contractapi.TransactionContextInterface, recordID string) (*MedicalAsset, error) {
	return c.QueryMedicalAsset(ctx, recordID)
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

	// WP-11: 链码侧盖章调用者身份
	audit.EndorserMSPID = ""
	audit.ClientIdentityID = ""
	if err := stampIdentityFromContext(ctx, &audit.EndorserMSPID, &audit.ClientIdentityID); err != nil {
		return err
	}

	auditBytes, _ := json.Marshal(audit)
	return ctx.GetStub().PutState("AUDIT_"+audit.LogID, auditBytes)
}
