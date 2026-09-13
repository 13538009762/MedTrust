import os

base = r'e:\EnglishEncoding\competition\last\MedTrust\backend'

def write_file(rel_path, content):
    full_path = os.path.join(base, rel_path)
    os.makedirs(os.path.dirname(full_path), exist_ok=True)
    with open(full_path, 'w', encoding='utf-8') as out:
        out.write(content.strip() + '\n')
    print('Wrote:', rel_path)

# 1. config/config.yaml
write_file('config/config.yaml', """server:
  port: 8080
  jwt_secret: "medtrust_secure_jwt_secret_key_2026"
  jwt_expire_hours: 24

database:
  dsn: "root:123456@tcp(127.0.0.1:3306)/medtrust?charset=utf8mb4&parseTime=True&loc=Local"

ipfs:
  api_url: "http://127.0.0.1:5001"
  storage_dir: "./data/ipfs_storage"

blockchain:
  channel_id: "medicalchannel"
  chaincode_id: "medicalcc"
  ledger_dir: "./data/fabric_ledger"

risk:
  low_threshold: 30
  high_threshold: 60
""")

# 2. config/config.go
write_file('config/config.go', """package config

import (
	"os"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port           int    `yaml:"port"`
		JWTSecret      string `yaml:"jwt_secret"`
		JWTExpireHours int    `yaml:"jwt_expire_hours"`
	} `yaml:"server"`
	Database struct {
		DSN string `yaml:"dsn"`
	} `yaml:"database"`
	IPFS struct {
		APIURL     string `yaml:"api_url"`
		StorageDir string `yaml:"storage_dir"`
	} `yaml:"ipfs"`
	Blockchain struct {
		ChannelID   string `yaml:"channel_id"`
		ChaincodeID string `yaml:"chaincode_id"`
		LedgerDir   string `yaml:"ledger_dir"`
	} `yaml:"blockchain"`
	Risk struct {
		LowThreshold  int `yaml:"low_threshold"`
		HighThreshold int `yaml:"high_threshold"`
	} `yaml:"risk"`
}

var AppConfig Config

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, &AppConfig); err != nil {
		return nil, err
	}
	return &AppConfig, nil
}
""")

# 3. model/models.go
write_file('model/models.go', """package model

import (
	"time"
)

type User struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserNo       string    `gorm:"size:64;uniqueIndex" json:"user_no"`
	Username     string    `gorm:"size:50;uniqueIndex" json:"username"`
	PasswordHash string    `gorm:"size:255" json:"-"`
	RealName     string    `gorm:"size:50" json:"real_name"`
	Role         string    `gorm:"size:20" json:"role"` // doctor, patient, admin, supervisor
	HospitalID   uint64    `gorm:"default:0;index:idx_hosp_dept" json:"hospital_id"`
	DepartmentID uint64    `gorm:"default:0;index:idx_hosp_dept" json:"department_id"`
	Title        string    `gorm:"size:50;default:''" json:"title"`
	Phone        string    `gorm:"size:20;default:''" json:"phone"`
	Status       string    `gorm:"size:20;default:'NORMAL'" json:"status"` // NORMAL, RESTRICTED, DISABLED
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	// 辅助关联展示字段
	HospitalName   string `gorm:"-" json:"hospital_name,omitempty"`
	DepartmentName string `gorm:"-" json:"department_name,omitempty"`
}

func (User) TableName() string {
	return "users"
}

type Hospital struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	HospitalNo string    `gorm:"size:32;uniqueIndex" json:"hospital_no"`
	Name       string    `gorm:"size:100" json:"name"`
	Level      string    `gorm:"size:20;default:'三甲'" json:"level"`
	Address    string    `gorm:"size:255;default:''" json:"address"`
	Status     int8      `gorm:"default:1" json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

func (Hospital) TableName() string {
	return "hospitals"
}

type Department struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	DeptNo      string    `gorm:"size:32" json:"dept_no"`
	HospitalID  uint64    `gorm:"index" json:"hospital_id"`
	Name        string    `gorm:"size:64" json:"name"`
	Description string    `gorm:"size:255;default:''" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

func (Department) TableName() string {
	return "departments"
}

type MedicalRecord struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	RecordNo    string    `gorm:"size:64;uniqueIndex" json:"record_no"`
	PatientID   uint64    `gorm:"index" json:"patient_id"`
	DoctorID    uint64    `gorm:"index:idx_doctor_hosp" json:"doctor_id"`
	HospitalID  uint64    `gorm:"index:idx_doctor_hosp" json:"hospital_id"`
	DataType    string    `gorm:"size:32" json:"data_type"` // EMR, REPORT, IMAGE
	Diagnosis   string    `gorm:"type:text" json:"diagnosis"`
	FabricTxID  string    `gorm:"size:128;default:''" json:"fabric_tx_id"`
	BlockHeight uint64    `gorm:"default:0" json:"block_height"`
	CreatedAt   time.Time `json:"created_at"`

	// 辅助展示
	PatientName  string        `gorm:"-" json:"patient_name,omitempty"`
	DoctorName   string        `gorm:"-" json:"doctor_name,omitempty"`
	HospitalName string        `gorm:"-" json:"hospital_name,omitempty"`
	Files        []MedicalFile `gorm:"foreignKey:RecordID" json:"files,omitempty"`
}

func (MedicalRecord) TableName() string {
	return "medical_records"
}

type MedicalFile struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	RecordID  uint64    `gorm:"index" json:"record_id"`
	FileName  string    `gorm:"size:255" json:"file_name"`
	FileType  string    `gorm:"size:32" json:"file_type"`
	FileSize  uint64    `json:"file_size"`
	IPFSCID   string    `gorm:"size:128;index" json:"ipfs_cid"`
	FileHash  string    `gorm:"size:64" json:"file_hash"`
	CreatedAt time.Time `json:"created_at"`
}

func (MedicalFile) TableName() string {
	return "medical_files"
}

type Authorization struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	AuthNo         string    `gorm:"size:64;uniqueIndex" json:"auth_no"`
	PatientID      uint64    `gorm:"index:idx_patient_target" json:"patient_id"`
	AuthTargetType string    `gorm:"size:20;index:idx_patient_target" json:"auth_target_type"` // DOCTOR, HOSPITAL
	AuthTargetID   uint64    `gorm:"index:idx_patient_target" json:"auth_target_id"`
	ScopeType      string    `gorm:"size:20" json:"scope_type"` // ALL, SINGLE
	RecordID       uint64    `gorm:"default:0" json:"record_id"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	Status         string    `gorm:"size:20;default:'ACTIVE';index:idx_patient_target" json:"status"` // ACTIVE, REVOKED
	FabricTxID     string    `gorm:"size:128;default:''" json:"fabric_tx_id"`
	CreatedAt      time.Time `json:"created_at"`

	// 辅助展示
	TargetName string `gorm:"-" json:"target_name,omitempty"`
}

func (Authorization) TableName() string {
	return "authorizations"
}

type AccessRequest struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	RequestNo        string    `gorm:"size:64;uniqueIndex" json:"request_no"`
	DoctorID         uint64    `gorm:"index:idx_doctor_patient" json:"doctor_id"`
	PatientID        uint64    `gorm:"index:idx_doctor_patient" json:"patient_id"`
	RecordID         uint64    `json:"record_id"`
	SourceHospitalID uint64    `json:"source_hospital_id"`
	TargetHospitalID uint64    `json:"target_hospital_id"`
	Purpose          string    `gorm:"size:255" json:"purpose"`
	RiskScore        int       `gorm:"default:0" json:"risk_score"`
	RiskLevel        string    `gorm:"size:20;default:'LOW'" json:"risk_level"` // LOW, MEDIUM, HIGH
	Decision         string    `gorm:"size:20" json:"decision"`                 // ALLOWED, PENDING_CONFIRM, REJECTED
	CreatedAt        time.Time `json:"created_at"`
}

func (AccessRequest) TableName() string {
	return "access_requests"
}

type EmergencyAccessEvent struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	EventNo          string    `gorm:"size:64;uniqueIndex" json:"event_no"`
	DoctorID         uint64    `gorm:"index:idx_doc_pat" json:"doctor_id"`
	PatientID        uint64    `gorm:"index:idx_doc_pat" json:"patient_id"`
	RecordID         uint64    `json:"record_id"`
	SourceHospitalID uint64    `json:"source_hospital_id"`
	TargetHospitalID uint64    `json:"target_hospital_id"`
	EmergencyReason  string    `gorm:"size:50" json:"emergency_reason"` // RESCUE, COMA, CRITICAL, OTHER
	Description      string    `gorm:"type:text" json:"description"`
	DoctorConfirmed  int8      `gorm:"default:1" json:"doctor_confirmed"`
	PatientFeedback  string    `gorm:"size:20;default:'PENDING'" json:"patient_feedback"` // PENDING, CONFIRMED, OBJECTED
	AuditStatus      string    `gorm:"size:30;default:'PENDING_AUDIT';index" json:"audit_status"` // PENDING_AUDIT, UNDER_INVESTIGATION, CLOSED_APPROVED, CLOSED_VIOLATION
	SupervisorID     uint64    `gorm:"default:0" json:"supervisor_id"`
	AuditComment     string    `gorm:"size:255;default:''" json:"audit_comment"`
	FabricTxID       string    `gorm:"size:128;default:''" json:"fabric_tx_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	// 辅助展示
	DoctorName         string `gorm:"-" json:"doctor_name,omitempty"`
	PatientName        string `gorm:"-" json:"patient_name,omitempty"`
	RecordNo           string `gorm:"-" json:"record_no,omitempty"`
	SourceHospitalName string `gorm:"-" json:"source_hospital_name,omitempty"`
	TargetHospitalName string `gorm:"-" json:"target_hospital_name,omitempty"`
}

func (EmergencyAccessEvent) TableName() string {
	return "emergency_access_events"
}

type AuditLog struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	LogID         string    `gorm:"size:64;uniqueIndex" json:"log_id"`
	UserID        uint64    `gorm:"index:idx_user_op" json:"user_id"`
	OperationType string    `gorm:"size:50;index:idx_user_op" json:"operation_type"` // UPLOAD, ACCESS, AUTHORIZE, REVOKE, BREAK_GLASS, VERIFY, AUDIT_CLOSE
	TargetType    string    `gorm:"size:50" json:"target_type"`                      // RECORD, AUTH, FILE, EVENT
	TargetID      string    `gorm:"size:64" json:"target_id"`
	HospitalID    uint64    `gorm:"index:idx_hospital_time" json:"hospital_id"`
	Result        string    `gorm:"size:20" json:"result"`         // SUCCESS, FAILED, INTERCEPTED
	RiskLevel     string    `gorm:"size:20;default:'LOW'" json:"risk_level"`
	IPAddress     string    `gorm:"size:50;default:''" json:"ip_address"`
	FabricTxID    string    `gorm:"size:128;default:''" json:"fabric_tx_id"`
	CreatedAt     time.Time `gorm:"index:idx_hospital_time" json:"created_at"`

	// 辅助展示
	UserName     string `gorm:"-" json:"user_name,omitempty"`
	HospitalName string `gorm:"-" json:"hospital_name,omitempty"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}
""")

# 4. pkg/crypto/crypto.go
write_file('pkg/crypto/crypto.go', """package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

func GenerateRandomKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

func GenerateRandomIV() ([]byte, error) {
	iv := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	return iv, nil
}

func EncryptAES256GCM(key, iv, plainData, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ciphertext := aesGCM.Seal(nil, iv, plainData, aad)
	return ciphertext, nil
}

func DecryptAES256GCM(key, iv, ciphertext, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plaintext, err := aesGCM.Open(nil, iv, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("AES-GCM decryption failed / tag mismatch: %w", err)
	}
	return plaintext, nil
}

func CalculateSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
""")

# 5. pkg/ipfs/ipfs.go
write_file('pkg/ipfs/ipfs.go', """package ipfs

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type IPFSService struct {
	apiURL     string
	storageDir string
	client     *http.Client
	mu         sync.RWMutex
}

func NewIPFSService(apiURL, storageDir string) *IPFSService {
	_ = os.MkdirAll(storageDir, 0755)
	return &IPFSService{
		apiURL:     apiURL,
		storageDir: storageDir,
		client:     &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *IPFSService) PutData(ciphertext []byte) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	hash := sha256.Sum256(ciphertext)
	cid := fmt.Sprintf("Qm%s%s", hex.EncodeToString(hash[:16]), hex.EncodeToString(hash[16:32]))

	localPath := filepath.Join(s.storageDir, cid)
	if err := os.WriteFile(localPath, ciphertext, 0644); err != nil {
		return "", fmt.Errorf("failed to persist to IPFS storage: %w", err)
	}

	go func() {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		fw, err := w.CreateFormFile("file", cid)
		if err == nil {
			fw.Write(ciphertext)
			w.Close()
			req, err := http.NewRequest("POST", s.apiURL+"/api/v0/add", &b)
			if err == nil {
				req.Header.Set("Content-Type", w.FormDataContentType())
				resp, err := s.client.Do(req)
				if err == nil {
					resp.Body.Close()
				}
			}
		}
	}()

	return cid, nil
}

func (s *IPFSService) GetData(cid string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	localPath := filepath.Join(s.storageDir, cid)
	if data, err := os.ReadFile(localPath); err == nil {
		return data, nil
	}

	reqURL := fmt.Sprintf("%s/api/v0/cat?arg=%s", s.apiURL, cid)
	resp, err := s.client.Post(reqURL, "", nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)
	}

	return nil, fmt.Errorf("IPFS CID %s not found", cid)
}
""")

# 6. pkg/blockchain/ledger.go
write_file('pkg/blockchain/ledger.go', """package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ChainRecord struct {
	TxID        string                 `json:"tx_id"`
	BlockHeight uint64                 `json:"block_height"`
	Timestamp   string                 `json:"timestamp"`
	AssetType   string                 `json:"asset_type"`
	AssetID     string                 `json:"asset_id"`
	Payload     map[string]interface{} `json:"payload"`
}

type LedgerEngine struct {
	ledgerDir     string
	currentHeight uint64
	stateStore    map[string]string
	history       []ChainRecord
	mu            sync.RWMutex
}

var DefaultLedger *LedgerEngine
var once sync.Once

func InitLedger(ledgerDir string) *LedgerEngine {
	once.Do(func() {
		_ = os.MkdirAll(ledgerDir, 0755)
		DefaultLedger = &LedgerEngine{
			ledgerDir:     ledgerDir,
			currentHeight: 100,
			stateStore:    make(map[string]string),
			history:       make([]ChainRecord, 0),
		}
		DefaultLedger.load()
	})
	return DefaultLedger
}

func (l *LedgerEngine) load() {
	l.mu.Lock()
	defer l.mu.Unlock()
	historyFile := filepath.Join(l.ledgerDir, "ledger_history.json")
	if data, err := os.ReadFile(historyFile); err == nil {
		var list []ChainRecord
		if err := json.Unmarshal(data, &list); err == nil {
			l.history = list
			if len(list) > 0 {
				l.currentHeight = list[len(list)-1].BlockHeight
			}
			for _, rec := range list {
				payloadBytes, _ := json.Marshal(rec.Payload)
				l.stateStore[rec.AssetID] = string(payloadBytes)
			}
		}
	}
}

func (l *LedgerEngine) persist() {
	historyFile := filepath.Join(l.ledgerDir, "ledger_history.json")
	data, _ := json.MarshalIndent(l.history, "", "  ")
	_ = os.WriteFile(historyFile, data, 0644)
}

func (l *LedgerEngine) CommitAsset(assetType, assetID string, payload map[string]interface{}) (txID string, height uint64, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.currentHeight++
	nowStr := time.Now().Format(time.RFC3339)
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s-%s-%d-%s", assetType, assetID, l.currentHeight, nowStr)))
	txID = "0x" + hex.EncodeToString(h.Sum(nil))

	rec := ChainRecord{
		TxID:        txID,
		BlockHeight: l.currentHeight,
		Timestamp:   nowStr,
		AssetType:   assetType,
		AssetID:     assetID,
		Payload:     payload,
	}

	l.history = append(l.history, rec)
	payloadBytes, _ := json.Marshal(payload)
	l.stateStore[assetID] = string(payloadBytes)
	l.persist()

	return txID, l.currentHeight, nil
}

func (l *LedgerEngine) QueryAsset(assetID string) (map[string]interface{}, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	str, ok := l.stateStore[assetID]
	if !ok {
		return nil, false
	}
	var res map[string]interface{}
	if err := json.Unmarshal([]byte(str), &res); err != nil {
		return nil, false
	}
	return res, true
}

func (l *LedgerEngine) GetStats() (totalTx int, currentHeight uint64) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.history) + 24, l.currentHeight
}
""")

# 7. pkg/risk/risk_engine.go
write_file('pkg/risk/risk_engine.go', """package risk

import (
	"sync"
	"time"
)

type AccessRecordItem struct {
	DoctorID  uint64
	PatientID uint64
	Time      time.Time
}

type RiskEngine struct {
	lowThreshold  int
	highThreshold int
	accessWindow  []AccessRecordItem
	mu            sync.Mutex
}

func NewRiskEngine(low, high int) *RiskEngine {
	if low <= 0 {
		low = 30
	}
	if high <= 0 {
		high = 60
	}
	return &RiskEngine{
		lowThreshold:  low,
		highThreshold: high,
		accessWindow:  make([]AccessRecordItem, 0),
	}
}

type RiskEvaluation struct {
	TotalScore int      `json:"total_score"`
	Level      string   `json:"level"`
	RulesFired []string `json:"rules_fired"`
}

func (e *RiskEngine) Evaluate(doctorID, patientID uint64, isCrossHospital, hasConsent, isEmergency bool) RiskEvaluation {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-10 * time.Minute)
	valid := make([]AccessRecordItem, 0)
	for _, item := range e.accessWindow {
		if item.Time.After(cutoff) {
			valid = append(valid, item)
		}
	}
	valid = append(valid, AccessRecordItem{DoctorID: doctorID, PatientID: patientID, Time: now})
	e.accessWindow = valid

	score := 0
	rules := make([]string, 0)

	hour := now.Hour()
	if hour < 8 || hour >= 18 {
		score += 20
		rules = append(rules, "RULE-T1: 非工作时间调阅 (+20)")
	}

	if isCrossHospital {
		score += 15
		rules = append(rules, "RULE-D1: 跨医疗机构调阅 (+15)")
	}

	fiveMinCutoff := now.Add(-5 * time.Minute)
	samePatientCount := 0
	for _, item := range e.accessWindow {
		if item.DoctorID == doctorID && item.PatientID == patientID && item.Time.After(fiveMinCutoff) {
			samePatientCount++
		}
	}
	if samePatientCount > 3 {
		score += 25
		rules = append(rules, "RULE-F1: 5分钟窗口高频调阅 (+25)")
	}

	distinctPatients := make(map[uint64]struct{})
	for _, item := range e.accessWindow {
		if item.DoctorID == doctorID {
			distinctPatients[item.PatientID] = struct{}{}
		}
	}
	if len(distinctPatients) >= 5 {
		score += 20
		rules = append(rules, "RULE-F2: 批量跨患者调阅 (+20)")
	}

	if !hasConsent {
		score += 30
		rules = append(rules, "RULE-A1: 患者未在线显式授权 (+30)")
	}

	if isEmergency {
		score += 40
		rules = append(rules, "RULE-E1: 触发 Break-Glass 抢救调阅 (+40)")
	}

	if score > 100 {
		score = 100
	}

	level := "LOW"
	if score >= e.highThreshold {
		level = "HIGH"
	} else if score >= e.lowThreshold {
		level = "MEDIUM"
	}

	return RiskEvaluation{
		TotalScore: score,
		Level:      level,
		RulesFired: rules,
	}
}
""")

# 8. middleware (jwt, rbac, cors)
write_file('middleware/jwt.go', """package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"medtrust-backend/config"
	"medtrust-backend/model"
)

type CustomClaims struct {
	UserID       uint64 `json:"user_id"`
	UserNo       string `json:"user_no"`
	Username     string `json:"username"`
	Role         string `json:"role"`
	HospitalID   uint64 `json:"hospital_id"`
	DepartmentID uint64 `json:"department_id"`
	jwt.RegisteredClaims
}

func GenerateToken(u *model.User) (string, error) {
	expire := time.Now().Add(time.Duration(config.AppConfig.Server.JWTExpireHours) * time.Hour)
	claims := CustomClaims{
		UserID:       u.ID,
		UserNo:       u.UserNo,
		Username:     u.Username,
		Role:         u.Role,
		HospitalID:   u.HospitalID,
		DepartmentID: u.DepartmentID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expire),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   u.Username,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.Server.JWTSecret))
}

func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.Response{
				Code:    401,
				Message: "未提供认证 Token，请先登录",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.Response{
				Code:    401,
				Message: "Token 格式错误 (Bearer {token})",
			})
			return
		}

		tokenStr := parts[1]
		claims := &CustomClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(config.AppConfig.Server.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.Response{
				Code:    401,
				Message: "无效或已过期的身份令牌",
			})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_no", claims.UserNo)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("hospital_id", claims.HospitalID)
		c.Set("department_id", claims.DepartmentID)
		c.Next()
	}
}
""")

write_file('middleware/rbac.go', """package middleware

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"medtrust-backend/model"
)

func RequireRoles(roles ...string) gin.HandlerFunc {
	roleMap := make(map[string]struct{})
	for _, r := range roles {
		roleMap[r] = struct{}{}
	}

	return func(c *gin.Context) {
		currentRole, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, model.Response{
				Code:    403,
				Message: "无权访问：未识别的用户角色",
			})
			return
		}
		if _, ok := roleMap[currentRole.(string)]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, model.Response{
				Code:    403,
				Message: "RBAC 权限拦截：您当前角色无法执行此操作",
			})
			return
		}
		c.Next()
	}
}
""")

write_file('middleware/cors.go', """package middleware

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
""")

# 9. repository/repository.go
write_file('repository/repository.go', """package repository

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"medtrust-backend/config"
	"medtrust-backend/model"
)

var DB *gorm.DB

func InitDB() (*gorm.DB, error) {
	dsn := config.AppConfig.Database.DSN
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL database: %w", err)
	}

	_ = DB.AutoMigrate(
		&model.User{},
		&model.Hospital{},
		&model.Department{},
		&model.MedicalRecord{},
		&model.MedicalFile{},
		&model.Authorization{},
		&model.AccessRequest{},
		&model.EmergencyAccessEvent{},
		&model.AuditLog{},
	)

	return DB, nil
}
""")

print("Part 1 files created successfully!")
