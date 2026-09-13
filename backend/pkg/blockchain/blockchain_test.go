package blockchain

import (
	"os"
	"strings"
	"testing"
)

func TestMockLedgerService(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "medtrust_ledger_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mock := &MockLedgerService{
		ledgerDir:     tempDir,
		currentHeight: 100,
		stateStore:    make(map[string]string),
		history:       make([]ChainRecord, 0),
	}

	// 1. 验证运行状态
	status := mock.GetBlockchainStatus()
	if status.Mode != "mock" {
		t.Errorf("Expected mode 'mock', got '%s'", status.Mode)
	}
	if status.Connected != false {
		t.Errorf("Expected connected=false for mock mode, got true")
	}
	if status.Message != "MockLedger Development Mode" {
		t.Errorf("Expected message 'MockLedger Development Mode', got '%s'", status.Message)
	}

	// 2. 验证存证提交
	payload := map[string]interface{}{
		"record_no":  "REC_TEST_MOCK_001",
		"patient_id": 1,
	}
	txID, height, err := mock.CommitAsset("MEDICAL_RECORD", "REC_TEST_MOCK_001", payload)
	if err != nil {
		t.Fatalf("CommitAsset failed: %v", err)
	}
	if txID == "" || height != 101 {
		t.Errorf("Invalid commit result: txID=%s, height=%d", txID, height)
	}

	// 3. 验证查询
	val, ok := mock.QueryAsset("REC_TEST_MOCK_001")
	if !ok || val["record_no"] != "REC_TEST_MOCK_001" {
		t.Errorf("QueryAsset failed: %v", val)
	}
}

func TestFabricOfflineMode(t *testing.T) {
	// 配置不可达的 Peer 节点以模拟 Fabric 网络停机 / 离线
	offlineCfg := FabricGatewayConfig{
		Mode:         "fabric",
		Enabled:      true,
		PeerEndpoint: "127.0.0.1:19999", // 不存在的端口
		GatewayPeer:  "peer0.org1.example.com",
		MSPID:        "Org1MSP",
		ChannelID:    "medchannel",
		ChaincodeID:  "medical",
		TLSCertPath:  "non_existent_tls.crt",
		CertPath:     "non_existent_cert.crt",
		KeyPath:      "non_existent_key.pem",
	}

	gateway := NewFabricGatewayService(offlineCfg)

	// 1. 验证状态报告 Fabric Unavailable
	if gateway.IsLiveFabric() {
		t.Errorf("Expected IsLiveFabric() to be false when offline")
	}

	status := gateway.GetBlockchainStatus()
	if status.Mode != "fabric" {
		t.Errorf("Expected mode 'fabric', got '%s'", status.Mode)
	}
	if status.Connected != false {
		t.Errorf("Expected connected=false when network is down")
	}
	if !strings.HasPrefix(status.Message, "Fabric Unavailable") {
		t.Errorf("Expected status message to start with 'Fabric Unavailable', got '%s'", status.Message)
	}

	// 2. 验证严格拒绝上链存证，绝不虚假伪造成功或私自回退到 Mock
	_, _, err := gateway.CommitAsset("MEDICAL_RECORD", "REC_FAIL_TEST", map[string]interface{}{"data": 123})
	if err == nil {
		t.Fatalf("Expected error when committing to offline Fabric, but got nil")
	}
	if !strings.Contains(err.Error(), "strictly refusing to falsify on-chain confirmation") {
		t.Errorf("Expected rejection error message, got: %v", err)
	}
}

func TestFabricGatewayLive(t *testing.T) {
	liveCfg := FabricGatewayConfig{
		Mode:         "fabric",
		Enabled:      true,
		PeerEndpoint: "127.0.0.1:7051",
		GatewayPeer:  "peer0.org1.example.com",
		MSPID:        "Org1MSP",
		ChannelID:    "medchannel",
		ChaincodeID:  "medical",
		TLSCertPath:  "../../fabric_crypto/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt",
		CertPath:     "../../fabric_crypto/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp/signcerts/Admin@org1.example.com-cert.pem",
		KeyPath:      "../../fabric_crypto/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp/keystore",
	}

	gateway := NewFabricGatewayService(liveCfg)
	if !gateway.IsLiveFabric() {
		t.Skip("Fabric network not currently reachable on 7051, skipping live test")
	}

	status := gateway.GetBlockchainStatus()
	if status.Mode != "fabric" || !status.Connected || status.Message != "Fabric Network Running" {
		t.Errorf("Unexpected status: %+v", status)
	}

	// 存证真实上链测试
	txID, height, err := gateway.CommitAsset("MEDICAL_RECORD", "TEST_REC_UNIT", map[string]interface{}{
		"record_no":   "TEST_REC_UNIT",
		"cid":         "QmUnitTestingCID",
		"file_hash":   "hash123",
		"hospital_id": "1",
		"data_type":   "EMR",
	})
	if err != nil {
		t.Fatalf("CommitAsset to live Fabric failed: %v", err)
	}
	if txID == "" || height == 0 {
		t.Errorf("Invalid txID or height: txID=%s, height=%d", txID, height)
	}
}

