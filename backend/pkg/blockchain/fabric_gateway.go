package blockchain

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// FabricGatewayConfig Fabric 2.5 Gateway 连接配置
type FabricGatewayConfig struct {
	Mode         string `yaml:"mode"` // "fabric" or "mock"
	Enabled      bool   `yaml:"enabled"`
	PeerEndpoint string `yaml:"peer_endpoint"`
	GatewayPeer  string `yaml:"gateway_peer"`
	MSPID        string `yaml:"msp_id"`
	ChannelID    string `yaml:"channel_id"`
	ChaincodeID  string `yaml:"chaincode_id"`
	TLSCertPath  string `yaml:"tls_cert_path"`
	CertPath     string `yaml:"cert_path"`
	KeyPath      string `yaml:"key_path"`
}

// FabricGatewayService 官方 Fabric 2.5 Gateway 真实连接服务实现
type FabricGatewayService struct {
	cfg           FabricGatewayConfig
	connected     bool
	statusMessage string
	gw            *client.Gateway
	network       *client.Network
	contract      *client.Contract
	clientConn    *grpc.ClientConn
	blockCount    uint64
	txCount       int
	mu            sync.RWMutex
}

// NewFabricGatewayService 实例化 Fabric 官方网关服务 (严格连接与防伪装校验)
func NewFabricGatewayService(cfg FabricGatewayConfig) *FabricGatewayService {
	s := &FabricGatewayService{
		cfg:           cfg,
		connected:     false,
		statusMessage: "Initializing Fabric Gateway",
		blockCount:    1,
		txCount:       0,
	}

	if s.cfg.ChannelID == "" {
		s.cfg.ChannelID = "medchannel"
	}
	if s.cfg.ChaincodeID == "" {
		s.cfg.ChaincodeID = "medical"
	}
	if s.cfg.GatewayPeer == "" {
		s.cfg.GatewayPeer = "peer0.org1.example.com"
	}
	if s.cfg.MSPID == "" {
		s.cfg.MSPID = "Org1MSP"
	}

	log.Printf("[FabricGateway] 正在建立与 Hyperledger Fabric 2.5 Gateway 真实连接: Peer=%s, Channel=%s, Chaincode=%s, MSP=%s",
		cfg.PeerEndpoint, s.cfg.ChannelID, s.cfg.ChaincodeID, s.cfg.MSPID)

	// 1. TCP 探活
	conn, err := net.DialTimeout("tcp", cfg.PeerEndpoint, 1500*time.Millisecond)
	if err != nil {
		s.connected = false
		s.statusMessage = fmt.Sprintf("connection refused at %s (%v)", cfg.PeerEndpoint, err)
		log.Printf("[FabricGateway] ❌ Fabric Peer 节点网络不可达 (%s): %v。已明确标记 Fabric Unavailable，拒绝伪装上链", cfg.PeerEndpoint, err)
		return s
	}
	_ = conn.Close()

	// 2. 读取证书与私钥
	id, err := loadIdentity(s.cfg.CertPath, s.cfg.MSPID)
	if err != nil {
		s.connected = false
		s.statusMessage = fmt.Sprintf("Load identity failed: %v", err)
		log.Printf("[FabricGateway] ❌ 读取 MSP 证书失败 (%s): %v", s.cfg.CertPath, err)
		return s
	}

	sign, err := loadSigner(s.cfg.KeyPath)
	if err != nil {
		s.connected = false
		s.statusMessage = fmt.Sprintf("Load private key failed: %v", err)
		log.Printf("[FabricGateway] ❌ 读取私钥失败 (%s): %v", s.cfg.KeyPath, err)
		return s
	}

	// 3. 读取 TLS CA 证书建立 gRPC 连接
	grpcConn, err := newGrpcConnection(s.cfg.PeerEndpoint, s.cfg.TLSCertPath, s.cfg.GatewayPeer)
	if err != nil {
		s.connected = false
		s.statusMessage = fmt.Sprintf("gRPC connection failed: %v", err)
		log.Printf("[FabricGateway] ❌ 建立 TLS gRPC 连接失败: %v", err)
		return s
	}
	s.clientConn = grpcConn

	// 4. 初始化 Fabric Gateway Client
	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithClientConnection(grpcConn),
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(15*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		s.connected = false
		s.statusMessage = fmt.Sprintf("Gateway client connect failed: %v", err)
		log.Printf("[FabricGateway] ❌ Fabric Gateway 连接失败: %v", err)
		return s
	}
	s.gw = gw

	// 5. 获取 Network 与 Contract
	network := gw.GetNetwork(s.cfg.ChannelID)
	contract := network.GetContractWithName(s.cfg.ChaincodeID, "MedicalContract")
	if contract == nil {
		contract = network.GetContract(s.cfg.ChaincodeID)
	}
	s.network = network
	s.contract = contract

	s.connected = true
	s.statusMessage = "Connected to Fabric Gateway successfully"

	log.Printf("[FabricGateway] ✅ 成功连接至 Hyperledger Fabric 2.5 网关通道: %s, 智能合约: %s, 节点: %s",
		s.cfg.ChannelID, s.cfg.ChaincodeID, s.cfg.GatewayPeer)

	return s
}

// loadIdentity 从 PEM 证书读取 X.509 身份
func loadIdentity(certPath, mspID string) (*identity.X509Identity, error) {
	info, err := os.Stat(certPath)
	if err == nil && info.IsDir() {
		files, _ := os.ReadDir(certPath)
		for _, f := range files {
			if strings.HasSuffix(f.Name(), ".pem") {
				certPath = filepath.Join(certPath, f.Name())
				break
			}
		}
	} else if os.IsNotExist(err) {
		dir := filepath.Dir(certPath)
		if files, rErr := os.ReadDir(dir); rErr == nil {
			for _, f := range files {
				if strings.HasSuffix(f.Name(), ".pem") {
					certPath = filepath.Join(dir, f.Name())
					break
				}
			}
		}
	}

	certificatePEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file (%s): %w", certPath, err)
	}
	cert, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}
	return identity.NewX509Identity(mspID, cert)
}


// loadSigner 从文件或 keystore 目录读取私钥并构建数字签名器
func loadSigner(keyPath string) (identity.Sign, error) {
	var keyFile string
	info, err := os.Stat(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat key path: %w", err)
	}

	if info.IsDir() {
		files, err := os.ReadDir(keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read keystore directory: %w", err)
		}
		for _, file := range files {
			if !file.IsDir() && (strings.HasSuffix(file.Name(), "_sk") || strings.HasSuffix(file.Name(), ".pem") || strings.HasSuffix(file.Name(), ".key")) {
				keyFile = filepath.Join(keyPath, file.Name())
				break
			}
		}
		if keyFile == "" {
			return nil, fmt.Errorf("no private key found in keystore directory: %s", keyPath)
		}
	} else {
		keyFile = keyPath
	}

	privateKeyPEM, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	privateKey, err := identity.PrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return identity.NewPrivateKeySign(privateKey)
}

// newGrpcConnection 建立带 TLS 证书校验的 gRPC 连接
func newGrpcConnection(endpoint, tlsCertPath, gatewayPeer string) (*grpc.ClientConn, error) {
	tlsCert, err := os.ReadFile(tlsCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read TLS CA cert %s: %w", tlsCertPath, err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(tlsCert) {
		return nil, fmt.Errorf("failed to append TLS CA cert to pool")
	}

	transportCreds := credentials.NewClientTLSFromCert(certPool, gatewayPeer)
	return grpc.Dial(endpoint, grpc.WithTransportCredentials(transportCreds))
}

func (s *FabricGatewayService) IsLiveFabric() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected
}

func (s *FabricGatewayService) GetBlockchainStatus() BlockchainStatusDTO {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := BlockchainStatusDTO{
		Mode:        "fabric",
		Connected:   s.connected,
		Network:     s.cfg.ChannelID,
		Chaincode:   s.cfg.ChaincodeID,
		Peer:        s.cfg.GatewayPeer,
		LatestBlock: s.blockCount,
	}

	if s.connected {
		status.Message = "Fabric Network Running"
	} else {
		status.Message = "Fabric Unavailable: " + s.statusMessage
	}

	return status
}

func (s *FabricGatewayService) CommitAsset(assetType, assetID string, payload map[string]interface{}) (txID string, height uint64, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 严禁在 Fabric 模式下静默伪装上链成功
	if !s.connected || s.contract == nil {
		return "", 0, fmt.Errorf("Fabric network unavailable (%s): transaction %s rejected, strictly refusing to falsify on-chain confirmation", s.statusMessage, assetID)
	}

	nowStr := time.Now().Format(time.RFC3339)
	var methodName string
	var args []string

	switch assetType {
	case "MEDICAL_RECORD":
		methodName = "CreateMedicalRecord"
		recordID := assetID
		if r, ok := payload["record_no"].(string); ok && r != "" {
			recordID = r
		}
		cid := fmt.Sprintf("%v", payload["cid"])
		// 优先取综合临床指纹 clinical_hash（绑定全量病史、主诉、诱因过敏史、确诊、用药方案及附件指纹），确保数据库任何文字篡改都能被 Fabric 账本即刻阻断
		targetHash := fmt.Sprintf("%v", payload["file_hash"])
		if ch, ok := payload["clinical_hash"].(string); ok && ch != "" && ch != "<nil>" {
			targetHash = ch
		} else if ch, ok := payload["clinical_hash"]; ok && fmt.Sprintf("%v", ch) != "" && fmt.Sprintf("%v", ch) != "<nil>" {
			targetHash = fmt.Sprintf("%v", ch)
		}
		hospID := fmt.Sprintf("%v", payload["hospital_id"])
		dataType := fmt.Sprintf("%v", payload["data_type"])
		args = []string{recordID, cid, targetHash, hospID, dataType, nowStr}

	case "AUTHORIZATION":
		methodName = "CreateAuthorization"
		patientID := fmt.Sprintf("%v", payload["patient_id"])
		doctorID := fmt.Sprintf("%v", payload["doctor_id"])
		hospID := fmt.Sprintf("%v", payload["hospital_id"])
		scope := fmt.Sprintf("%v", payload["scope"])
		expireTime := fmt.Sprintf("%v", payload["expire_time"])
		status := fmt.Sprintf("%v", payload["status"])
		args = []string{assetID, patientID, doctorID, hospID, scope, expireTime, status}

	case "REVOKE_AUTH":
		methodName = "RevokeAuthorization"
		args = []string{assetID}

	case "AUDIT":
		methodName = "CreateAuditLog"
		userID := fmt.Sprintf("%v", payload["user_id"])
		action := fmt.Sprintf("%v", payload["op_type"])
		targetID := fmt.Sprintf("%v", payload["target_id"])
		riskLevel := fmt.Sprintf("%v", payload["risk_level"])
		args = []string{assetID, userID, action, targetID, riskLevel, nowStr}

	case "EMERGENCY_ACCESS":
		methodName = "CreateEmergencyRecord"
		jsonBytes, _ := json.Marshal(payload)
		args = []string{string(jsonBytes)}

	case "EMERGENCY_AUDIT":
		methodName = "UpdateEmergencyStatus"
		eventNo := fmt.Sprintf("%v", payload["event_no"])
		auditStatus := fmt.Sprintf("%v", payload["audit_status"])
		supervisorID := fmt.Sprintf("%v", payload["supervisor_id"])
		comment := fmt.Sprintf("%v", payload["comment"])
		punishment := fmt.Sprintf("%v", payload["punishment"])
		args = []string{eventNo, auditStatus, supervisorID, comment, punishment, nowStr}

	default:
		methodName = "CreateAuditRecord"
		jsonBytes, _ := json.Marshal(payload)
		args = []string{string(jsonBytes)}
	}

	// 真实提交 Proposal 并获取真实 Fabric Transaction ID 与 BlockNumber
	proposal, err := s.contract.NewProposal(methodName, client.WithArguments(args...))
	if err != nil {
		return "", 0, fmt.Errorf("failed to create Fabric proposal for %s: %w", methodName, err)
	}

	txID = proposal.TransactionID()
	endorsed, err := proposal.Endorse()
	if err != nil {
		return "", 0, fmt.Errorf("failed to endorse Fabric transaction %s: %w", txID, err)
	}

	commit, err := endorsed.Submit()
	if err != nil {
		return "", 0, fmt.Errorf("failed to submit Fabric transaction %s: %w", txID, err)
	}

	status, err := commit.Status()
	if err != nil {
		return txID, 0, fmt.Errorf("failed to obtain commit status for %s: %w", txID, err)
	}

	if !status.Successful {
		return txID, 0, fmt.Errorf("Fabric commit rejected for %s, status code: %d", txID, status.Code)
	}

	height = status.BlockNumber
	s.blockCount = height
	s.txCount++

	log.Printf("[FabricGateway] ✅ 交易已成功上链: TxID=%s, BlockHeight=%d, Contract=%s, Method=%s",
		txID, height, s.cfg.ChaincodeID, methodName)

	return txID, height, nil
}

func (s *FabricGatewayService) QueryAsset(assetID string) (map[string]interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected || s.contract == nil {
		return nil, false
	}

	// 优先以 QueryMedicalRecord 查询
	result, err := s.contract.EvaluateTransaction("QueryMedicalRecord", assetID)
	if err != nil {
		// 备用 QueryMedicalAsset
		result, err = s.contract.EvaluateTransaction("QueryMedicalAsset", assetID)
		if err != nil {
			return nil, false
		}
	}

	var res map[string]interface{}
	if err := json.Unmarshal(result, &res); err != nil {
		return nil, false
	}
	if _, ok := res["clinical_hash"]; !ok {
		if fh, ok := res["file_hash"]; ok {
			res["clinical_hash"] = fh
		}
	}
	return res, true
}

func (s *FabricGatewayService) GetStats() (totalTx int, currentHeight uint64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.txCount, s.blockCount
}

func (s *FabricGatewayService) GetHistory(assetID string) ([]ChainRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.connected || s.contract == nil {
		return nil, nil
	}
	result, err := s.contract.EvaluateTransaction("QueryAuditLogs", assetID)
	if err != nil {
		return nil, err
	}
	var logs []*AuditLog
	_ = json.Unmarshal(result, &logs)
	var records []ChainRecord
	for _, l := range logs {
		records = append(records, ChainRecord{
			TxID:        l.AuditID,
			BlockHeight: s.blockCount,
			Timestamp:   l.Timestamp,
			AssetType:   "AUDIT",
			AssetID:     l.RecordID,
			Payload: map[string]interface{}{
				"audit_id":   l.AuditID,
				"user_id":    l.UserID,
				"action":     l.Action,
				"risk_level": l.RiskLevel,
				"timestamp":  l.Timestamp,
			},
		})
	}
	return records, nil
}

