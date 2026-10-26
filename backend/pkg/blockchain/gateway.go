package blockchain

import (
	"encoding/json"
	"fmt"
	"time"
)

// MedicalAsset 医疗数据统一上链存证资产模型 (规范契合 Fabric 智能合约)
type MedicalAsset struct {
	RecordID     string `json:"record_id"`
	PatientID    string `json:"patient_id"`
	CID          string `json:"cid"`
	FileHash     string `json:"file_hash"`
	ClinicalHash string `json:"clinical_hash"`
	HospitalID   string `json:"hospital_id"`
	CreatorID    string `json:"creator_id"`
	DataType     string `json:"data_type"`
	CreateTime   string `json:"create_time"`
}

// EmergencyAccessRecord 紧急破窗访问上链凭据 (与智能合约严格对齐)
type EmergencyAccessRecord struct {
	EventNo      string `json:"event_no"`
	DoctorID     string `json:"doctor_id"`
	PatientID    string `json:"patient_id"`
	RecordID     string `json:"record_id"`
	HospitalID   string `json:"hospital_id"`
	Reason       string `json:"reason"`
	Description  string `json:"description"`
	Timestamp    string `json:"timestamp"`
	Status       string `json:"status"`
	SupervisorID string `json:"supervisor_id,omitempty"`
	AuditComment string `json:"audit_comment,omitempty"`
	Punishment   string `json:"punishment,omitempty"`
	AuditTime    string `json:"audit_time,omitempty"`
}

// ChainRecord 账本流转存证记录
type ChainRecord struct {
	TxID        string                 `json:"tx_id"`
	BlockHeight uint64                 `json:"block_height"`
	Timestamp   string                 `json:"timestamp"`
	AssetType   string                 `json:"asset_type"`
	AssetID     string                 `json:"asset_id"`
	Payload     map[string]interface{} `json:"payload"`
}

// BlockchainStatusDTO 区块链底层引擎运行状态
type BlockchainStatusDTO struct {
	Mode        string `json:"mode"`
	Connected   bool   `json:"connected"`
	Network     string `json:"network,omitempty"`
	Chaincode   string `json:"chaincode,omitempty"`
	Peer        string `json:"peer,omitempty"`
	LatestBlock uint64 `json:"latest_block"`
	Message     string `json:"message,omitempty"`
}

// AuditLog 链上审计结构
type AuditLog struct {
	AuditID   string `json:"audit_id"`
	UserID    string `json:"user_id"`
	Action    string `json:"action"`
	RecordID  string `json:"record_id"`
	RiskLevel string `json:"risk_level"`
	Timestamp string `json:"timestamp"`
}

// BlockchainService 统一区块链服务抽象接口 (满足正式 Fabric 2.5 Gateway 与本地容灾 Mock 互换)
type BlockchainService interface {
	CommitAsset(assetType, assetID string, payload map[string]interface{}) (txID string, height uint64, err error)
	QueryAsset(assetID string) (map[string]interface{}, bool)
	GetStats() (totalTx int, currentHeight uint64)
	GetHistory(assetID string) ([]ChainRecord, error)
	IsLiveFabric() bool
	GetBlockchainStatus() BlockchainStatusDTO

	// 高阶规范方法
	CreateMedicalAsset(asset MedicalAssetPayload) (txID string, blockHeight uint64, err error)
	QueryMedicalAsset(recordNo string) (*MedicalAssetPayload, error)
	CreateAuthorizationRecord(auth AuthorizationPayload) (txID string, err error)
	RevokeAuthorizationRecord(authNo string) (txID string, err error)
	CreateEmergencyRecord(ev EmergencyPayload) (txID string, err error)
	UpdateEmergencyStatus(eventNo string, newStatus string) (txID string, err error)
	CreateAuditRecord(audit AuditPayload) (txID string, err error)
}

// DefaultService 全局单例区块链服务实例
var DefaultService BlockchainService

// TxRecorder 交易落库钩子函数 (支持链上链下双重校验与区块链交易存证表记录)
var TxRecorder func(businessType, businessID, txID string, blockNo uint64, channelName, chaincodeName, status string)

// MedicalAssetPayload 链上医疗数据存证负载
type MedicalAssetPayload struct {
	RecordID   string `json:"record_id"`
	PatientID  string `json:"patient_id"`
	CID        string `json:"cid"`
	FileHash   string `json:"file_hash"`
	HospitalID string `json:"hospital_id"`
	CreatorID  string `json:"creator_id"`
	CreateTime string `json:"create_time"`
}

// AuthorizationPayload 链上授权流转负载
type AuthorizationPayload struct {
	AuthID     string `json:"auth_id"`
	PatientID  string `json:"patient_id"`
	TargetType string `json:"target_type"` // DOCTOR / HOSPITAL
	TargetID   string `json:"target_id"`
	ScopeType  string `json:"scope_type"` // ALL / SINGLE
	RecordID   string `json:"record_id"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	Status     string `json:"status"` // ACTIVE / REVOKED
}

// EmergencyPayload 链上紧急访问事件凭据
type EmergencyPayload struct {
	EventNo    string `json:"event_no"`
	DoctorID   string `json:"doctor_id"`
	PatientID  string `json:"patient_id"`
	RecordID   string `json:"record_id"`
	HospitalID string `json:"hospital_id"`
	Reason     string `json:"reason"`
	Timestamp  string `json:"timestamp"`
	Status     string `json:"status"` // PENDING_AUDIT / CLOSED_APPROVED / CLOSED_VIOLATION
}

// AuditPayload 链上可信审计日志载荷
type AuditPayload struct {
	LogID      string `json:"log_id"`
	UserID     string `json:"user_id"`
	Action     string `json:"action"` // UPLOAD / ACCESS / AUTHORIZE / REVOKE / BREAK_GLASS / VERIFY
	ResourceID string `json:"resource_id"`
	Result     string `json:"result"` // SUCCESS / INTERCEPTED / TAMPERED
	Timestamp  string `json:"timestamp"`
}

// FabricGatewayClient 别名映射
type FabricGatewayClient = FabricGatewayService

// MockLedgerClient 别名映射
type MockLedgerClient = MockLedgerService

// --- MockLedgerService 高阶方法实现 ---

func (m *MockLedgerService) CreateMedicalAsset(asset MedicalAssetPayload) (txID string, blockHeight uint64, err error) {
	if asset.CreateTime == "" {
		asset.CreateTime = time.Now().Format(time.RFC3339)
	}
	payload := map[string]interface{}{
		"record_id":   asset.RecordID,
		"patient_id":  asset.PatientID,
		"cid":         asset.CID,
		"file_hash":   asset.FileHash,
		"hospital_id": asset.HospitalID,
		"creator_id":  asset.CreatorID,
		"create_time": asset.CreateTime,
	}
	txID, blockHeight, err = m.CommitAsset("MEDICAL_ASSET", asset.RecordID, payload)
	if err == nil {
		// 同时记录带 ASSET_ 前缀的世界状态以便快速按键查验
		m.mu.Lock()
		b, _ := json.Marshal(asset)
		m.stateStore["ASSET_"+asset.RecordID] = string(b)
		m.mu.Unlock()
	}
	return txID, blockHeight, err
}

func (m *MockLedgerService) QueryMedicalAsset(recordNo string) (*MedicalAssetPayload, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 优先查询 ASSET_ 前缀
	if raw, ok := m.stateStore["ASSET_"+recordNo]; ok {
		var payload MedicalAssetPayload
		if err := json.Unmarshal([]byte(raw), &payload); err == nil && payload.RecordID != "" {
			return &payload, nil
		}
	}

	// 备用通过 QueryAsset
	if data, ok := m.stateStore[recordNo]; ok {
		var cr ChainRecord
		if err := json.Unmarshal([]byte(data), &cr); err == nil && cr.Payload != nil {
			pb, _ := json.Marshal(cr.Payload)
			var payload MedicalAssetPayload
			_ = json.Unmarshal(pb, &payload)
			if payload.RecordID == "" {
				if rNo, ok := cr.Payload["record_no"].(string); ok {
					payload.RecordID = rNo
				}
			}
			if payload.RecordID != "" {
				return &payload, nil
			}
		}
	}

	return nil, fmt.Errorf("medical asset %s not found on chain", recordNo)
}

func (m *MockLedgerService) CreateAuthorizationRecord(auth AuthorizationPayload) (txID string, err error) {
	payload := map[string]interface{}{
		"auth_id":     auth.AuthID,
		"patient_id":  auth.PatientID,
		"target_type": auth.TargetType,
		"target_id":   auth.TargetID,
		"scope_type":  auth.ScopeType,
		"record_id":   auth.RecordID,
		"start_time":  auth.StartTime,
		"end_time":    auth.EndTime,
		"status":      auth.Status,
	}
	txID, _, err = m.CommitAsset("AUTH_RECORD", auth.AuthID, payload)
	if err == nil {
		m.mu.Lock()
		b, _ := json.Marshal(auth)
		m.stateStore["AUTH_"+auth.AuthID] = string(b)
		m.mu.Unlock()
	}
	return txID, err
}

func (m *MockLedgerService) RevokeAuthorizationRecord(authNo string) (txID string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	raw, ok := m.stateStore["AUTH_"+authNo]
	if !ok {
		// 尝试不带前缀
		raw, ok = m.stateStore[authNo]
	}
	var auth AuthorizationPayload
	if ok {
		_ = json.Unmarshal([]byte(raw), &auth)
	}
	auth.Status = "REVOKED"
	b, _ := json.Marshal(auth)
	m.stateStore["AUTH_"+authNo] = string(b)

	payload := map[string]interface{}{
		"auth_no": authNo,
		"status":  "REVOKED",
		"time":    time.Now().Format(time.RFC3339),
	}
	txID, _, err = m.CommitAsset("AUTH_REVOKE", authNo, payload)
	return txID, err
}

func (m *MockLedgerService) CreateEmergencyRecord(ev EmergencyPayload) (txID string, err error) {
	payload := map[string]interface{}{
		"event_no":    ev.EventNo,
		"doctor_id":   ev.DoctorID,
		"patient_id":  ev.PatientID,
		"record_id":   ev.RecordID,
		"hospital_id": ev.HospitalID,
		"reason":      ev.Reason,
		"timestamp":   ev.Timestamp,
		"status":      ev.Status,
	}
	txID, _, err = m.CommitAsset("EMERGENCY_RECORD", ev.EventNo, payload)
	if err == nil {
		m.mu.Lock()
		b, _ := json.Marshal(ev)
		m.stateStore["EMERG_"+ev.EventNo] = string(b)
		m.mu.Unlock()
	}
	return txID, err
}

func (m *MockLedgerService) UpdateEmergencyStatus(eventNo string, newStatus string) (txID string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	raw, ok := m.stateStore["EMERG_"+eventNo]
	var ev EmergencyPayload
	if ok {
		_ = json.Unmarshal([]byte(raw), &ev)
	}
	ev.Status = newStatus
	b, _ := json.Marshal(ev)
	m.stateStore["EMERG_"+eventNo] = string(b)

	payload := map[string]interface{}{
		"event_no":   eventNo,
		"new_status": newStatus,
		"time":       time.Now().Format(time.RFC3339),
	}
	txID, _, err = m.CommitAsset("EMERGENCY_STATUS", eventNo, payload)
	return txID, err
}

func (m *MockLedgerService) CreateAuditRecord(audit AuditPayload) (txID string, err error) {
	payload := map[string]interface{}{
		"log_id":      audit.LogID,
		"user_id":     audit.UserID,
		"action":      audit.Action,
		"resource_id": audit.ResourceID,
		"result":      audit.Result,
		"timestamp":   audit.Timestamp,
	}
	txID, _, err = m.CommitAsset("AUDIT_RECORD", audit.LogID, payload)
	if err == nil {
		m.mu.Lock()
		b, _ := json.Marshal(audit)
		m.stateStore["AUDIT_"+audit.LogID] = string(b)
		m.mu.Unlock()
	}
	return txID, err
}

// --- FabricGatewayService 高阶方法实现 ---

func (s *FabricGatewayService) CreateMedicalAsset(asset MedicalAssetPayload) (string, uint64, error) {
	payload := map[string]interface{}{
		"record_id":   asset.RecordID,
		"patient_id":  asset.PatientID,
		"cid":         asset.CID,
		"file_hash":   asset.FileHash,
		"hospital_id": asset.HospitalID,
		"creator_id":  asset.CreatorID,
		"create_time": asset.CreateTime,
	}
	return s.CommitAsset("MEDICAL_ASSET", asset.RecordID, payload)
}

func (s *FabricGatewayService) QueryMedicalAsset(recordNo string) (*MedicalAssetPayload, error) {
	data, ok := s.QueryAsset(recordNo)
	if !ok {
		return nil, fmt.Errorf("medical asset %s not found on live fabric", recordNo)
	}
	b, _ := json.Marshal(data)
	var payload MedicalAssetPayload
	if err := json.Unmarshal(b, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func (s *FabricGatewayService) CreateAuthorizationRecord(auth AuthorizationPayload) (string, error) {
	payload := map[string]interface{}{
		"auth_id":     auth.AuthID,
		"patient_id":  auth.PatientID,
		"target_type": auth.TargetType,
		"target_id":   auth.TargetID,
		"scope_type":  auth.ScopeType,
		"record_id":   auth.RecordID,
		"start_time":  auth.StartTime,
		"end_time":    auth.EndTime,
		"status":      auth.Status,
	}
	txID, _, err := s.CommitAsset("AUTH_RECORD", auth.AuthID, payload)
	return txID, err
}

func (s *FabricGatewayService) RevokeAuthorizationRecord(authNo string) (string, error) {
	payload := map[string]interface{}{
		"auth_no": authNo,
		"status":  "REVOKED",
	}
	txID, _, err := s.CommitAsset("AUTH_REVOKE", authNo, payload)
	return txID, err
}

func (s *FabricGatewayService) CreateEmergencyRecord(ev EmergencyPayload) (string, error) {
	payload := map[string]interface{}{
		"event_no":    ev.EventNo,
		"doctor_id":   ev.DoctorID,
		"patient_id":  ev.PatientID,
		"record_id":   ev.RecordID,
		"hospital_id": ev.HospitalID,
		"reason":      ev.Reason,
		"timestamp":   ev.Timestamp,
		"status":      ev.Status,
	}
	txID, _, err := s.CommitAsset("EMERGENCY_RECORD", ev.EventNo, payload)
	return txID, err
}

func (s *FabricGatewayService) UpdateEmergencyStatus(eventNo string, newStatus string) (string, error) {
	payload := map[string]interface{}{
		"event_no":   eventNo,
		"new_status": newStatus,
	}
	txID, _, err := s.CommitAsset("EMERGENCY_STATUS", eventNo, payload)
	return txID, err
}

func (s *FabricGatewayService) CreateAuditRecord(audit AuditPayload) (string, error) {
	payload := map[string]interface{}{
		"log_id":      audit.LogID,
		"user_id":     audit.UserID,
		"action":      audit.Action,
		"resource_id": audit.ResourceID,
		"result":      audit.Result,
		"timestamp":   audit.Timestamp,
	}
	txID, _, err := s.CommitAsset("AUDIT_RECORD", audit.LogID, payload)
	return txID, err
}
