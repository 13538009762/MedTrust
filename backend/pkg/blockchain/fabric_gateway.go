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
	stopChan      chan struct{}
	reconnectOnce sync.Once
	closeOnce     sync.Once
}

// resolveCryptoPath 智能多级候选路径解析，适配宿主机、测试目录、跨工作区与 Docker 多种运行环境
func resolveCryptoPath(path string) string {
	if path == "" {
		return path
	}
	if _, err := os.Stat(path); err == nil {
		return path
	}
	clean := filepath.Clean(path)
	candidates := []string{
		filepath.Join(".", clean),
		filepath.Join("backend", clean),
		filepath.Join("..", clean),
		filepath.Join("..", "..", clean),
	}
	if idx := strings.Index(clean, "fabric_crypto"); idx != -1 {
		sub := clean[idx+len("fabric_crypto"):]
		sub = strings.TrimPrefix(sub, string(filepath.Separator))
		sub = strings.TrimPrefix(sub, "/")
		candidates = append(candidates,
			filepath.Join("deploy", "fabric", "crypto-config", sub),
			filepath.Join("..", "deploy", "fabric", "crypto-config", sub),
			filepath.Join("..", "..", "deploy", "fabric", "crypto-config", sub),
			filepath.Join("backend", "fabric_crypto", sub),
			filepath.Join(".", "fabric_crypto", sub),
		)
	}
	for _, cand := range candidates {
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
	}
	return path
}

func isNetworkErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "transport is closing") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "code = Unavailable") ||
		strings.Contains(msg, "all SubConns are in TransientFailure")
}

// tryConnect 尝试建立与 Fabric 网关的真实连接 (具备双向端点嗅探与证书自适应加载)
func (s *FabricGatewayService) tryConnect() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.connected && s.clientConn != nil {
		return nil
	}

	// 清理旧资源
	if s.gw != nil {
		_ = s.gw.Close()
		s.gw = nil
	}
	if s.clientConn != nil {
		_ = s.clientConn.Close()
		s.clientConn = nil
	}
	s.network = nil
	s.contract = nil

	// 1. TCP 探活与双向端点回退 (支持宿主机 127.0.0.1 与容器内 peer0.org1.example.com 双向自适应探测)
	endpoint := s.cfg.PeerEndpoint
	conn, err := net.DialTimeout("tcp", endpoint, 1500*time.Millisecond)
	if err != nil {
		host, port, splitErr := net.SplitHostPort(endpoint)
		if splitErr == nil {
			var fallbacks []string
			if host != "127.0.0.1" && host != "localhost" {
				fallbacks = append(fallbacks, fmt.Sprintf("127.0.0.1:%s", port))
			} else {
				fallbacks = append(fallbacks, fmt.Sprintf("%s:%s", s.cfg.GatewayPeer, port))
			}

			for _, fb := range fallbacks {
				fbConn, fbErr := net.DialTimeout("tcp", fb, 1500*time.Millisecond)
				if fbErr == nil {
					log.Printf("[FabricGateway] 💡 成功探测到候选端点 %s，自动切换 PeerEndpoint -> %s", fb, fb)
					endpoint = fb
					s.cfg.PeerEndpoint = fb
					conn = fbConn
					err = nil
					break
				}
			}
		}
	}

	if err != nil {
		s.connected = false
		s.statusMessage = fmt.Sprintf("connection refused at %s (%v)", s.cfg.PeerEndpoint, err)
		return fmt.Errorf("fabric peer unreachable at %s: %w", s.cfg.PeerEndpoint, err)
	}
	if conn != nil {
		_ = conn.Close()
	}

	// 2. 智能候选路径加载证书与私钥
	certPath := resolveCryptoPath(s.cfg.CertPath)
	id, err := loadIdentity(certPath, s.cfg.MSPID)
	if err != nil {
		s.connected = false
		s.statusMessage = fmt.Sprintf("Load identity failed: %v", err)
		log.Printf("[FabricGateway] ❌ 读取 MSP 证书失败 (%s): %v", certPath, err)
		return err
	}

	keyPath := resolveCryptoPath(s.cfg.KeyPath)
	sign, err := loadSigner(keyPath)
	if err != nil {
		s.connected = false
		s.statusMessage = fmt.Sprintf("Load private key failed: %v", err)
		log.Printf("[FabricGateway] ❌ 读取私钥失败 (%s): %v", keyPath, err)
		return err
	}

	// 3. 读取 TLS CA 证书建立 gRPC 连接
	tlsCertPath := resolveCryptoPath(s.cfg.TLSCertPath)
	grpcConn, err := newGrpcConnection(endpoint, tlsCertPath, s.cfg.GatewayPeer)
	if err != nil {
		s.connected = false
		s.statusMessage = fmt.Sprintf("gRPC connection failed: %v", err)
		log.Printf("[FabricGateway] ❌ 建立 TLS gRPC 连接失败: %v", err)
		return err
	}

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
		_ = grpcConn.Close()
		s.connected = false
		s.statusMessage = fmt.Sprintf("Gateway client connect failed: %v", err)
		log.Printf("[FabricGateway] ❌ Fabric Gateway 连接失败: %v", err)
		return err
	}

	// 5. 获取 Network 与 Contract
	network := gw.GetNetwork(s.cfg.ChannelID)
	contract := network.GetContractWithName(s.cfg.ChaincodeID, "MedicalContract")
	if contract == nil {
		contract = network.GetContract(s.cfg.ChaincodeID)
	}

	s.clientConn = grpcConn
	s.gw = gw
	s.network = network
	s.contract = contract
	s.connected = true
	s.statusMessage = "Connected to Fabric Gateway successfully"

	log.Printf("[FabricGateway] ✅ 成功连接至 Hyperledger Fabric 2.5 网关通道: %s, 智能合约: %s, 节点: %s",
		s.cfg.ChannelID, s.cfg.ChaincodeID, s.cfg.GatewayPeer)

	return nil
}

// startReconnectLoop 启动常驻后台自愈协程 (每 3 秒检测连通状态，失联自动重试)
func (s *FabricGatewayService) startReconnectLoop() {
	s.reconnectOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(3 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-s.stopChan:
					return
				case <-ticker.C:
					s.mu.RLock()
					connected := s.connected
					s.mu.RUnlock()

					if !connected {
						if err := s.tryConnect(); err == nil {
							log.Printf("[FabricGateway] 🔄 后台自愈引擎已成功恢复与 Hyperledger Fabric 2.5 Gateway 连接")
						}
					}
				}
			}
		}()
	})
}

// NewFabricGatewayService 实例化 Fabric 官方网关服务 (严格连接与防伪装校验)
func NewFabricGatewayService(cfg FabricGatewayConfig) *FabricGatewayService {
	s := &FabricGatewayService{
		cfg:           cfg,
		connected:     false,
		statusMessage: "Initializing Fabric Gateway",
		blockCount:    1,
		txCount:       0,
		stopChan:      make(chan struct{}),
	}

	if s.cfg.ChannelID == "" {
		s.cfg.ChannelID = "medicalchannel"
	}
	if s.cfg.ChaincodeID == "" {
		s.cfg.ChaincodeID = "medical_contract"
	}
	if s.cfg.GatewayPeer == "" {
		s.cfg.GatewayPeer = "peer0.org1.example.com"
	}
	if s.cfg.MSPID == "" {
		s.cfg.MSPID = "Org1MSP"
	}

	log.Printf("[FabricGateway] 正在建立与 Hyperledger Fabric 2.5 Gateway 真实连接: Peer=%s, Channel=%s, Chaincode=%s, MSP=%s",
		cfg.PeerEndpoint, s.cfg.ChannelID, s.cfg.ChaincodeID, s.cfg.MSPID)

	if err := s.tryConnect(); err != nil {
		log.Printf("[FabricGateway] ⚠️ 初始连接 Fabric 暂不可达 (%v)，后台自愈重试协程已启动...", err)
	}

	s.startReconnectLoop()
	return s
}

// loadIdentity 从 PEM 证书读取 X.509 身份
func loadIdentity(certPath, mspID string) (*identity.X509Identity, error) {
	certPath = resolveCryptoPath(certPath)
	info, err := os.Stat(certPath)
	if err == nil && info.IsDir() {
		files, _ := os.ReadDir(certPath)
		for _, f := range files {
			if strings.HasSuffix(f.Name(), ".pem") || strings.HasSuffix(f.Name(), ".crt") {
				certPath = filepath.Join(certPath, f.Name())
				break
			}
		}
	} else if os.IsNotExist(err) {
		dir := filepath.Dir(certPath)
		if files, rErr := os.ReadDir(dir); rErr == nil {
			for _, f := range files {
				if strings.HasSuffix(f.Name(), ".pem") || strings.HasSuffix(f.Name(), ".crt") {
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
	keyPath = resolveCryptoPath(keyPath)
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
		if keyFile == "" && len(files) == 1 && !files[0].IsDir() {
			keyFile = filepath.Join(keyPath, files[0].Name())
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
	tlsCertPath = resolveCryptoPath(tlsCertPath)
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

// markDisconnected 安全标记失联状态：仅当失败时的 contract 仍是当前活跃实例时才降级，防止覆盖刚恢复重连的新连接。
func (s *FabricGatewayService) markDisconnected(failedContract *client.Contract, msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.contract == failedContract {
		s.connected = false
		s.statusMessage = msg
	}
}

// Close 优雅停止后台重连协程并关闭 Fabric Gateway 与底层 gRPC 连接。
func (s *FabricGatewayService) Close() error {
	s.closeOnce.Do(func() {
		if s.stopChan != nil {
			close(s.stopChan)
		}
	})
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.gw != nil {
		_ = s.gw.Close()
		s.gw = nil
	}
	if s.clientConn != nil {
		_ = s.clientConn.Close()
		s.clientConn = nil
	}
	s.network = nil
	s.contract = nil
	s.connected = false
	s.statusMessage = "Fabric Gateway closed"
	return nil
}

func (s *FabricGatewayService) GetBlockchainStatus() BlockchainStatusDTO {
	if !s.IsLiveFabric() {
		_ = s.tryConnect()
	}

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

func payloadStr(payload map[string]interface{}, keys ...string) string {
	if payload == nil {
		return ""
	}
	for _, k := range keys {
		v, ok := payload[k]
		if !ok || v == nil {
			continue
		}
		s := strings.TrimSpace(fmt.Sprintf("%v", v))
		if s != "" && s != "<nil>" {
			return s
		}
	}
	return ""
}

func (s *FabricGatewayService) CommitAsset(assetType, assetID string, payload map[string]interface{}) (txID string, height uint64, err error) {
	if !s.IsLiveFabric() {
		_ = s.tryConnect()
	}

	s.mu.RLock()
	connected := s.connected
	contract := s.contract
	statusMsg := s.statusMessage
	s.mu.RUnlock()

	// 严禁在 Fabric 模式下静默伪装上链成功
	if !connected || contract == nil {
		return "", 0, fmt.Errorf("Fabric network unavailable (%s): transaction %s rejected, strictly refusing to falsify on-chain confirmation", statusMsg, assetID)
	}

	nowStr := time.Now().Format(time.RFC3339)
	var methodName string
	var args []string

	switch assetType {
	case "MEDICAL_RECORD", "MEDICAL_ASSET":
		recordID := payloadStr(payload, "record_no", "record_id")
		if recordID == "" {
			recordID = assetID
		}
		patientID := payloadStr(payload, "patient_id")
		cid := payloadStr(payload, "cid")
		fileHash := payloadStr(payload, "file_hash")
		clinicalHash := payloadStr(payload, "clinical_hash")
		if clinicalHash == "" {
			clinicalHash = fileHash
		}
		hospID := payloadStr(payload, "hospital_id")
		creatorID := payloadStr(payload, "doctor_id", "creator_id")
		dataType := payloadStr(payload, "data_type")
		if dataType == "" {
			dataType = "EMR"
		}

		asset := MedicalAsset{
			RecordID:     recordID,
			PatientID:    patientID,
			CID:          cid,
			FileHash:     fileHash,
			ClinicalHash: clinicalHash,
			HospitalID:   hospID,
			CreatorID:    creatorID,
			DataType:     dataType,
			CreateTime:   nowStr,
		}
		assetBytes, _ := json.Marshal(asset)
		methodName = "CreateMedicalAsset"
		args = []string{string(assetBytes)}

	case "AUTHORIZATION", "AUTHORIZATION_KEY", "AUTH_RECORD":
		methodName = "CreateAuthorizationRecord"
		authNo := payloadStr(payload, "auth_no", "auth_id")
		if authNo == "" {
			authNo = assetID
		}
		status := payloadStr(payload, "status")
		if status == "" {
			status = "ACTIVE"
		}
		ts := payloadStr(payload, "timestamp", "start_time")
		if ts == "" {
			ts = nowStr
		}
		authPayload := map[string]interface{}{
			"auth_no":     authNo,
			"patient_id":  payloadStr(payload, "patient_id"),
			"target_type": payloadStr(payload, "target_type"),
			"target_id":   payloadStr(payload, "target_id", "doctor_id"),
			"hospital_id": payloadStr(payload, "hospital_id"),
			"scope":       payloadStr(payload, "scope", "scope_type"),
			"record_id":   payloadStr(payload, "record_id"),
			"expire_time": payloadStr(payload, "expire_time", "end_time"),
			"status":      status,
			"timestamp":   ts,
		}
		jsonBytes, _ := json.Marshal(authPayload)
		args = []string{string(jsonBytes)}

	case "REVOKE_AUTH", "AUTH_REVOKE":
		methodName = "RevokeAuthorizationRecord"
		authNo := payloadStr(payload, "auth_no", "auth_id")
		if authNo == "" {
			authNo = assetID
		}
		args = []string{authNo}

	case "AUDIT", "AUDIT_LOG", "AUDIT_RECORD":
		methodName = "CreateAuditRecord"
		logID := payloadStr(payload, "log_id", "audit_id")
		if logID == "" {
			logID = assetID
		}
		ts := payloadStr(payload, "timestamp")
		if ts == "" {
			ts = nowStr
		}
		auditPayload := map[string]interface{}{
			"log_id":      logID,
			"user_id":     payloadStr(payload, "user_id"),
			"op_type":     payloadStr(payload, "op_type", "action"),
			"target_type": payloadStr(payload, "target_type"),
			"target_id":   payloadStr(payload, "target_id", "resource_id", "record_id"),
			"hospital_id": payloadStr(payload, "hospital_id"),
			"result":      payloadStr(payload, "result"),
			"risk_level":  payloadStr(payload, "risk_level"),
			"timestamp":   ts,
		}
		jsonBytes, _ := json.Marshal(auditPayload)
		args = []string{string(jsonBytes)}

	case "EMERGENCY", "EMERGENCY_ACCESS", "EMERGENCY_RECORD":
		methodName = "CreateEmergencyRecord"
		eventNo := payloadStr(payload, "event_no")
		if eventNo == "" {
			eventNo = assetID
		}
		rec := EmergencyAccessRecord{
			EventNo:     eventNo,
			DoctorID:    payloadStr(payload, "doctor_id"),
			PatientID:   payloadStr(payload, "patient_id"),
			RecordID:    payloadStr(payload, "record_id"),
			HospitalID:  payloadStr(payload, "hospital_id"),
			Reason:      payloadStr(payload, "reason"),
			Description: payloadStr(payload, "description"),
			Timestamp:   payloadStr(payload, "timestamp"),
			Status:      payloadStr(payload, "status"),
		}
		if rec.Timestamp == "" {
			rec.Timestamp = nowStr
		}
		if rec.Status == "" {
			rec.Status = "PENDING_AUDIT"
		}
		jsonBytes, _ := json.Marshal(rec)
		args = []string{string(jsonBytes)}

	case "EMERGENCY_AUDIT", "EMERGENCY_STATUS":
		methodName = "UpdateEmergencyStatus"
		eventNo := payloadStr(payload, "event_no")
		if eventNo == "" {
			eventNo = assetID
		}
		auditStatus := payloadStr(payload, "audit_status", "new_status", "status")
		supervisorID := payloadStr(payload, "supervisor_id")
		comment := payloadStr(payload, "comment")
		punishment := payloadStr(payload, "punishment")
		auditTime := payloadStr(payload, "timestamp")
		if auditTime == "" {
			auditTime = nowStr
		}
		args = []string{eventNo, auditStatus, supervisorID, comment, punishment, auditTime}

	default:
		methodName = "CreateAuditRecord"
		logID := payloadStr(payload, "log_id", "audit_id")
		if logID == "" {
			logID = assetID
		}
		auditPayload := map[string]interface{}{
			"log_id":      logID,
			"user_id":     payloadStr(payload, "user_id"),
			"op_type":     payloadStr(payload, "op_type", "action", "business_type"),
			"target_type": payloadStr(payload, "target_type"),
			"target_id":   payloadStr(payload, "target_id", "resource_id", "record_id"),
			"hospital_id": payloadStr(payload, "hospital_id"),
			"result":      payloadStr(payload, "result"),
			"risk_level":  payloadStr(payload, "risk_level"),
			"timestamp":   nowStr,
		}
		jsonBytes, _ := json.Marshal(auditPayload)
		args = []string{string(jsonBytes)}
	}

	// 锁外执行网络背书/提交/确认（client.Contract 线程安全），避免持全局写锁长达 90s 阻塞全系统读/写
	proposal, err := contract.NewProposal(methodName, client.WithArguments(args...))
	if err != nil {
		if isNetworkErr(err) {
			s.markDisconnected(contract, fmt.Sprintf("proposal error: %v", err))
		}
		return "", 0, fmt.Errorf("failed to create Fabric proposal for %s: %w", methodName, err)
	}

	txID = proposal.TransactionID()
	endorsed, err := proposal.Endorse()
	if err != nil {
		if isNetworkErr(err) {
			s.markDisconnected(contract, fmt.Sprintf("endorse error: %v", err))
		}
		return "", 0, fmt.Errorf("failed to endorse Fabric transaction %s: %w", txID, err)
	}

	commit, err := endorsed.Submit()
	if err != nil {
		if isNetworkErr(err) {
			s.markDisconnected(contract, fmt.Sprintf("submit error: %v", err))
		}
		return "", 0, fmt.Errorf("failed to submit Fabric transaction %s: %w", txID, err)
	}

	status, err := commit.Status()
	if err != nil {
		if isNetworkErr(err) {
			s.markDisconnected(contract, fmt.Sprintf("commit status error: %v", err))
		}
		return txID, 0, fmt.Errorf("failed to obtain commit status for %s: %w", txID, err)
	}

	if !status.Successful {
		return txID, 0, fmt.Errorf("Fabric commit rejected for %s, status code: %d", txID, status.Code)
	}

	height = status.BlockNumber
	s.mu.Lock()
	if height > s.blockCount {
		s.blockCount = height
	}
	s.txCount++
	s.mu.Unlock()

	log.Printf("[FabricGateway] ✅ 交易已成功上链: TxID=%s, BlockHeight=%d, Contract=%s, Method=%s",
		txID, height, s.cfg.ChaincodeID, methodName)

	if TxRecorder != nil {
		TxRecorder(assetType, assetID, txID, height, s.cfg.ChannelID, s.cfg.ChaincodeID, "VALID")
	}

	return txID, height, nil
}

func (s *FabricGatewayService) QueryAsset(assetID string) (map[string]interface{}, bool) {
	if !s.IsLiveFabric() {
		_ = s.tryConnect()
	}

	s.mu.RLock()
	connected := s.connected
	contract := s.contract
	s.mu.RUnlock()

	if !connected || contract == nil {
		return nil, false
	}

	// 锁外执行 EvaluateTransaction，避免在 RLock 下写状态导致 Data Race
	result, err := contract.EvaluateTransaction("QueryMedicalAsset", assetID)
	if err != nil {
		if isNetworkErr(err) {
			s.markDisconnected(contract, fmt.Sprintf("evaluate error: %v", err))
		}
		// 备用 QueryMedicalRecord
		result, err = contract.EvaluateTransaction("QueryMedicalRecord", assetID)
		if err != nil {
			if isNetworkErr(err) {
				s.markDisconnected(contract, fmt.Sprintf("evaluate error: %v", err))
			}
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
	if !s.IsLiveFabric() {
		_ = s.tryConnect()
	}

	s.mu.RLock()
	connected := s.connected
	contract := s.contract
	blockCount := s.blockCount
	s.mu.RUnlock()

	if !connected || contract == nil {
		return nil, nil
	}
	result, err := contract.EvaluateTransaction("QueryAuditLogs", assetID)
	if err != nil {
		if isNetworkErr(err) {
			s.markDisconnected(contract, fmt.Sprintf("evaluate error: %v", err))
		}
		return nil, err
	}
	var logs []*AuditLog
	if err := json.Unmarshal(result, &logs); err != nil {
		return nil, fmt.Errorf("failed to parse chaincode audit logs: %w", err)
	}
	var records []ChainRecord
	for _, l := range logs {
		records = append(records, ChainRecord{
			TxID:        l.AuditID,
			BlockHeight: blockCount,
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
