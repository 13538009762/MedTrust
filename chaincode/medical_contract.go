package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// deterministicTimestamp 获取确定性账本交易时间戳，杜绝背书节点因 time.Now() 不一致导致背书失败
func deterministicTimestamp(ctx contractapi.TransactionContextInterface, provided string) string {
	cleaned := strings.TrimSpace(provided)
	if cleaned != "" && cleaned != "<nil>" {
		return cleaned
	}
	if stub := ctx.GetStub(); stub != nil {
		if ts, err := stub.GetTxTimestamp(); err == nil && ts != nil {
			return time.Unix(ts.Seconds, int64(ts.Nanos)).UTC().Format(time.RFC3339)
		}
	}
	return "1970-01-01T00:00:00Z"
}

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


// AuthorizationRecord 患者授权上链存证（统一授权账本结构）
type AuthorizationRecord struct {
	AuthNo     string `json:"auth_no"`
	PatientID  string `json:"patient_id"`
	TargetType string `json:"target_type"` // DOCTOR, HOSPITAL, ALL_DOCTORS, PRIVATE
	TargetID   string `json:"target_id"`
	Scope      string `json:"scope"` // ALL, SINGLE
	RecordID   string `json:"record_id"`
	Status     string `json:"status"` // ACTIVE, REVOKED
	Timestamp  string `json:"timestamp"`
	HospitalID string `json:"hospital_id,omitempty" metadata:",optional"`
	ExpireTime string `json:"expire_time,omitempty" metadata:",optional"`

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
	asset.RecordID = strings.TrimSpace(asset.RecordID)
	if asset.RecordID == "" {
		return fmt.Errorf("病历资产 record_id 不能为空")
	}

	// WP-11: 拒绝客户端载荷自带的伪造 MSP 归属字段，身份以链码侧提取为准
	asset.EndorserMSPID = ""
	asset.ClientIdentityID = ""
	asset.CreateTime = deterministicTimestamp(ctx, asset.CreateTime)

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

	assetBytes, err := json.Marshal(asset)
	if err != nil {
		return fmt.Errorf("序列化病历资产失败: %v", err)
	}
	return ctx.GetStub().PutState("RECORD_"+asset.RecordID, assetBytes)
}

// QueryMedicalAsset 查询病历链上存证摘要
func (c *MedicalContract) QueryMedicalAsset(ctx contractapi.TransactionContextInterface, recordID string) (*MedicalAsset, error) {
	recordID = strings.TrimSpace(recordID)
	if recordID == "" {
		return nil, fmt.Errorf("record_id 不能为空")
	}
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
	auth.AuthNo = strings.TrimSpace(auth.AuthNo)
	if auth.AuthNo == "" {
		return fmt.Errorf("授权流水号 auth_no 不能为空")
	}

	existing, err := ctx.GetStub().GetState("AUTH_" + auth.AuthNo)
	if err != nil {
		return fmt.Errorf("读取授权状态失败: %v", err)
	}
	if existing != nil {
		return fmt.Errorf("授权记录 %s 已存在于账本中，不可重复创建", auth.AuthNo)
	}

	if strings.TrimSpace(auth.Status) == "" {
		auth.Status = "ACTIVE"
	}
	auth.Timestamp = deterministicTimestamp(ctx, auth.Timestamp)

	// WP-11: 拒绝客户端伪造，链码侧盖章调用者身份
	auth.EndorserMSPID = ""
	auth.ClientIdentityID = ""
	if err := stampIdentityFromContext(ctx, &auth.EndorserMSPID, &auth.ClientIdentityID); err != nil {
		return err
	}

	authBytes, err := json.Marshal(auth)
	if err != nil {
		return fmt.Errorf("序列化授权记录失败: %v", err)
	}
	return ctx.GetStub().PutState("AUTH_"+auth.AuthNo, authBytes)
}

// RevokeAuthorizationRecord 撤销患者授权策略
func (c *MedicalContract) RevokeAuthorizationRecord(ctx contractapi.TransactionContextInterface, authNo string) error {
	authNo = strings.TrimSpace(authNo)
	if authNo == "" {
		return fmt.Errorf("授权流水号不能为空")
	}
	authBytes, err := ctx.GetStub().GetState("AUTH_" + authNo)
	if err != nil {
		return fmt.Errorf("读取授权条目失败: %v", err)
	}
	if authBytes == nil {
		return fmt.Errorf("授权条目 %s 不存在", authNo)
	}

	var auth AuthorizationRecord
	if err := json.Unmarshal(authBytes, &auth); err != nil {
		// 兼容早期 Authorization JSON 结构迁移
		var legacy Authorization
		if legErr := json.Unmarshal(authBytes, &legacy); legErr != nil {
			return fmt.Errorf("反序列化授权条目 %s 失败: %v", authNo, err)
		}
		auth = AuthorizationRecord{
			AuthNo:     legacy.AuthorizationID,
			PatientID:  legacy.PatientID,
			TargetType: "DOCTOR",
			TargetID:   legacy.DoctorID,
			HospitalID: legacy.HospitalID,
			Scope:      legacy.AuthorizationScope,
			ExpireTime: legacy.ExpireTime,
			Status:     legacy.Status,
		}
		if auth.AuthNo == "" {
			auth.AuthNo = authNo
		}
	}
	auth.Status = "REVOKED"

	// WP-11: 撤销操作重新盖章当前调用者身份
	auth.EndorserMSPID = ""
	auth.ClientIdentityID = ""
	if err := stampIdentityFromContext(ctx, &auth.EndorserMSPID, &auth.ClientIdentityID); err != nil {
		return err
	}

	updatedBytes, err := json.Marshal(auth)
	if err != nil {
		return fmt.Errorf("序列化撤销后授权记录失败: %v", err)
	}
	return ctx.GetStub().PutState("AUTH_"+authNo, updatedBytes)
}

// CreateEmergencyRecord 记录 Break-Glass 紧急访问事件单
func (c *MedicalContract) CreateEmergencyRecord(ctx contractapi.TransactionContextInterface, eventJSON string) error {
	var record EmergencyAccessRecord
	if err := json.Unmarshal([]byte(eventJSON), &record); err != nil {
		return fmt.Errorf("反序列化紧急访问记录失败: %v", err)
	}
	record.EventNo = strings.TrimSpace(record.EventNo)
	if record.EventNo == "" {
		return fmt.Errorf("紧急事件编号 event_no 不能为空")
	}

	existing, err := ctx.GetStub().GetState("EMERGENCY_" + record.EventNo)
	if err != nil {
		return fmt.Errorf("读取紧急事件状态失败: %v", err)
	}
	if existing != nil {
		return fmt.Errorf("紧急事件单 %s 已存在于账本中，不可重复创建", record.EventNo)
	}

	record.Timestamp = deterministicTimestamp(ctx, record.Timestamp)
	if strings.TrimSpace(record.Status) == "" {
		record.Status = "PENDING_AUDIT"
	}

	// WP-11: 链码侧盖章调用者身份
	record.EndorserMSPID = ""
	record.ClientIdentityID = ""
	if err := stampIdentityFromContext(ctx, &record.EndorserMSPID, &record.ClientIdentityID); err != nil {
		return err
	}

	recordBytes, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("序列化紧急访问记录失败: %v", err)
	}
	return ctx.GetStub().PutState("EMERGENCY_"+record.EventNo, recordBytes)
}

// UpdateEmergencyStatus 监管人员更新紧急访问审核裁决状态
func (c *MedicalContract) UpdateEmergencyStatus(ctx contractapi.TransactionContextInterface, eventNo, status, supervisorID, comment, punishment, auditTime string) error {
	eventNo = strings.TrimSpace(eventNo)
	if eventNo == "" {
		return fmt.Errorf("紧急事件编号不能为空")
	}
	recordBytes, err := ctx.GetStub().GetState("EMERGENCY_" + eventNo)
	if err != nil {
		return fmt.Errorf("读取紧急事件单失败: %v", err)
	}
	if recordBytes == nil {
		return fmt.Errorf("紧急事件单 %s 不存在", eventNo)
	}

	var record EmergencyAccessRecord
	if err := json.Unmarshal(recordBytes, &record); err != nil {
		return fmt.Errorf("反序列化紧急事件单 %s 失败: %v", eventNo, err)
	}
	if strings.TrimSpace(status) == "" {
		return fmt.Errorf("审核裁决目标状态不能为空")
	}
	record.Status = status
	record.SupervisorID = supervisorID
	record.AuditComment = comment
	record.Punishment = punishment
	record.AuditTime = deterministicTimestamp(ctx, auditTime)

	// WP-11: 监管裁决操作重新盖章当前调用者身份
	record.EndorserMSPID = ""
	record.ClientIdentityID = ""
	if err := stampIdentityFromContext(ctx, &record.EndorserMSPID, &record.ClientIdentityID); err != nil {
		return err
	}

	updatedBytes, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("序列化更新后紧急事件单失败: %v", err)
	}
	return ctx.GetStub().PutState("EMERGENCY_"+eventNo, updatedBytes)
}

// CreateMedicalRecord 提交符合链上原则的轻量病历存证 (统一映射至 CreateMedicalAsset 校验链与判重逻辑)
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
	raw, err := json.Marshal(asset)
	if err != nil {
		return fmt.Errorf("序列化病历资产失败: %v", err)
	}
	return c.CreateMedicalAsset(ctx, string(raw))
}

// QueryMedicalRecord 查询病历链上存证摘要 (统一返回 MedicalAsset 结构)
func (c *MedicalContract) QueryMedicalRecord(ctx contractapi.TransactionContextInterface, recordID string) (*MedicalAsset, error) {
	return c.QueryMedicalAsset(ctx, recordID)
}

// CreateAuthorization 记录患者知情授权策略 (统一收口至带 MSP 盖章与判重校验的 CreateAuthorizationRecord)
func (c *MedicalContract) CreateAuthorization(ctx contractapi.TransactionContextInterface, authorizationID, patientID, doctorID, hospitalID, authorizationScope, expireTime, status string) error {
	auth := AuthorizationRecord{
		AuthNo:     authorizationID,
		PatientID:  patientID,
		TargetType: "DOCTOR",
		TargetID:   doctorID,
		HospitalID: hospitalID,
		Scope:      authorizationScope,
		ExpireTime: expireTime,
		Status:     status,
	}
	raw, err := json.Marshal(auth)
	if err != nil {
		return fmt.Errorf("序列化授权记录失败: %v", err)
	}
	return c.CreateAuthorizationRecord(ctx, string(raw))
}

// RevokeAuthorization 撤销患者授权策略 (统一收口至带 MSP 盖章的 RevokeAuthorizationRecord)
func (c *MedicalContract) RevokeAuthorization(ctx contractapi.TransactionContextInterface, authorizationID string) error {
	return c.RevokeAuthorizationRecord(ctx, authorizationID)
}

// CreateAuditLog 记录系统敏感操作全量审计存证 (统一收口至带 MSP 盖章的 CreateAuditRecord)
func (c *MedicalContract) CreateAuditLog(ctx contractapi.TransactionContextInterface, auditID, userID, action, recordID, riskLevel, timestamp string) error {
	audit := AuditRecord{
		LogID:      auditID,
		UserID:     userID,
		OpType:     action,
		TargetType: "RECORD",
		TargetID:   recordID,
		RiskLevel:  riskLevel,
		Timestamp:  timestamp,
	}
	raw, err := json.Marshal(audit)
	if err != nil {
		return fmt.Errorf("序列化审计记录失败: %v", err)
	}
	return c.CreateAuditRecord(ctx, string(raw))
}

// QueryAuditLogs 查询链上审计日志 (统一兼容 AuditRecord 与历史 AuditLog 账本模型，支持按 recordID 过滤或返回全部)
func (c *MedicalContract) QueryAuditLogs(ctx contractapi.TransactionContextInterface, recordID string) ([]*AuditLog, error) {
	iterator, err := ctx.GetStub().GetStateByRange("AUDIT_", "AUDIT_\uffff")
	if err != nil {
		return nil, fmt.Errorf("查询审计日志失败: %v", err)
	}
	defer iterator.Close()

	var logs []*AuditLog
	for iterator.HasNext() {
		item, iterErr := iterator.Next()
		if iterErr != nil {
			return nil, fmt.Errorf("遍历审计账本失败: %v", iterErr)
		}
		var combined struct {
			AuditID   string `json:"audit_id"`
			LogID     string `json:"log_id"`
			UserID    string `json:"user_id"`
			Action    string `json:"action"`
			OpType    string `json:"op_type"`
			RecordID  string `json:"record_id"`
			TargetID  string `json:"target_id"`
			RiskLevel string `json:"risk_level"`
			Timestamp string `json:"timestamp"`
		}
		if err := json.Unmarshal(item.Value, &combined); err != nil {
			continue
		}
		entry := AuditLog{
			AuditID:   combined.AuditID,
			UserID:    combined.UserID,
			Action:    combined.Action,
			RecordID:  combined.RecordID,
			RiskLevel: combined.RiskLevel,
			Timestamp: combined.Timestamp,
		}
		if entry.AuditID == "" {
			entry.AuditID = combined.LogID
		}
		if entry.Action == "" {
			entry.Action = combined.OpType
		}
		if entry.RecordID == "" {
			entry.RecordID = combined.TargetID
		}
		if recordID == "" || recordID == "ALL" || entry.RecordID == recordID {
			logs = append(logs, &entry)
		}
	}
	return logs, nil
}

// CreateAuditRecord 记录系统敏感操作全量审计存证
func (c *MedicalContract) CreateAuditRecord(ctx contractapi.TransactionContextInterface, auditJSON string) error {
	var audit AuditRecord
	if err := json.Unmarshal([]byte(auditJSON), &audit); err != nil {
		return fmt.Errorf("反序列化审计日志记录失败: %v", err)
	}
	audit.LogID = strings.TrimSpace(audit.LogID)
	if audit.LogID == "" {
		return fmt.Errorf("审计日志编号 log_id 不能为空")
	}

	existing, err := ctx.GetStub().GetState("AUDIT_" + audit.LogID)
	if err != nil {
		return fmt.Errorf("读取审计状态失败: %v", err)
	}
	if existing != nil {
		return fmt.Errorf("审计日志 %s 已存在于账本中，禁止覆写", audit.LogID)
	}

	audit.Timestamp = deterministicTimestamp(ctx, audit.Timestamp)

	// WP-11: 链码侧盖章调用者身份
	audit.EndorserMSPID = ""
	audit.ClientIdentityID = ""
	if err := stampIdentityFromContext(ctx, &audit.EndorserMSPID, &audit.ClientIdentityID); err != nil {
		return err
	}

	auditBytes, err := json.Marshal(audit)
	if err != nil {
		return fmt.Errorf("序列化审计日志记录失败: %v", err)
	}
	return ctx.GetStub().PutState("AUDIT_"+audit.LogID, auditBytes)
}
