package main

import (
	"crypto/x509"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/pkg/cid"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeClientIdentity struct {
	cid.ClientIdentity
	mspID string
	id    string
}

func (f *fakeClientIdentity) GetMSPID() (string, error) { return f.mspID, nil }
func (f *fakeClientIdentity) GetID() (string, error)    { return f.id, nil }
func (f *fakeClientIdentity) GetX509Certificate() (*x509.Certificate, error) {
	return nil, nil
}

type fakeStateIterator struct {
	items []*queryresult.KV
	idx   int
}

func (it *fakeStateIterator) HasNext() bool { return it.idx < len(it.items) }
func (it *fakeStateIterator) Next() (*queryresult.KV, error) {
	item := it.items[it.idx]
	it.idx++
	return item, nil
}
func (it *fakeStateIterator) Close() error { return nil }

type fakeStub struct {
	shim.ChaincodeStubInterface
	state map[string][]byte
	ts    *timestamppb.Timestamp
}

func (s *fakeStub) GetState(key string) ([]byte, error) {
	return s.state[key], nil
}
func (s *fakeStub) PutState(key string, value []byte) error {
	cp := make([]byte, len(value))
	copy(cp, value)
	s.state[key] = cp
	return nil
}
func (s *fakeStub) GetTxTimestamp() (*timestamppb.Timestamp, error) {
	return s.ts, nil
}
func (s *fakeStub) GetStateByRange(startKey, endKey string) (shim.StateQueryIteratorInterface, error) {
	var items []*queryresult.KV
	for k, v := range s.state {
		if k >= startKey && k <= endKey {
			items = append(items, &queryresult.KV{Key: k, Value: v})
		}
	}
	return &fakeStateIterator{items: items}, nil
}

type fakeTxContext struct {
	contractapi.TransactionContextInterface
	stub *fakeStub
	id   *fakeClientIdentity
}

func (c *fakeTxContext) GetStub() shim.ChaincodeStubInterface { return c.stub }
func (c *fakeTxContext) GetClientIdentity() cid.ClientIdentity {
	return c.id
}

func newTestCtx(mspID, identityID string) *fakeTxContext {
	return &fakeTxContext{
		stub: &fakeStub{
			state: make(map[string][]byte),
			ts:    &timestamppb.Timestamp{Seconds: 1770000000},
		},
		id: &fakeClientIdentity{mspID: mspID, id: identityID},
	}
}

func TestChaincodeSchemaAllowsLegacyAssetsWithoutIdentityFields(t *testing.T) {
	cc, err := contractapi.NewChaincode(new(MedicalContract))
	if err != nil {
		t.Fatalf("NewChaincode failed: %v", err)
	}
	if cc == nil {
		t.Fatal("expected non-nil chaincode")
	}

	// Ensure legacy JSON without endorser_msp_id / client_identity_id unmarshals cleanly
	// and marshals cleanly with metadata:",optional" schema compatibility.
	legacyJSON := `{"record_id":"REC202609001","patient_id":"4","cid":"Qm123","file_hash":"abc","clinical_hash":"def","hospital_id":"1","creator_id":"1","data_type":"EMR","create_time":"2026-09-01T00:00:00Z"}`
	var asset MedicalAsset
	if err := json.Unmarshal([]byte(legacyJSON), &asset); err != nil {
		t.Fatalf("unmarshal legacy MedicalAsset failed: %v", err)
	}
	if asset.EndorserMSPID != "" || asset.ClientIdentityID != "" {
		t.Fatalf("expected empty identity fields for legacy asset, got %+v", asset)
	}
}

func TestChaincode_UnifiedContractsStampIdentityAndRejectDuplicates(t *testing.T) {
	contract := new(MedicalContract)
	ctx := newTestCtx("HospitalAMSP", "x509::CN=doc_li")

	// 1. 伪造身份字段必须在链码侧被覆写为真实提交方 MSP，且重复创建同一 RecordID 被拒绝
	err := contract.CreateMedicalAsset(ctx, `{"record_id":"REC-1","patient_id":"4","cid":"Qm1","file_hash":"fh1","clinical_hash":"ch1","endorser_msp_id":"SpoofedMSP"}`)
	if err != nil {
		t.Fatalf("CreateMedicalAsset failed: %v", err)
	}
	asset, err := contract.QueryMedicalAsset(ctx, "REC-1")
	if err != nil {
		t.Fatalf("QueryMedicalAsset failed: %v", err)
	}
	if asset.EndorserMSPID != "HospitalAMSP" {
		t.Fatalf("expected stamped EndorserMSPID=HospitalAMSP, got %q", asset.EndorserMSPID)
	}
	if err := contract.CreateMedicalRecord(ctx, "REC-1", "Qm2", "fh2", "1", "EMR", ""); err == nil || !strings.Contains(err.Error(), "已存在") {
		t.Fatalf("expected duplicate CreateMedicalRecord error, got %v", err)
	}

	// 2. CreateAuthorization 与 RevokeAuthorization 统一使用 AuthorizationRecord 并强制 MSP 盖章与判重
	if err := contract.CreateAuthorization(ctx, "AUTH-1", "4", "1", "1", "SINGLE", "2027-01-01T00:00:00Z", "ACTIVE"); err != nil {
		t.Fatalf("CreateAuthorization failed: %v", err)
	}
	if err := contract.CreateAuthorizationRecord(ctx, `{"auth_no":"AUTH-1","patient_id":"4"}`); err == nil {
		t.Fatalf("expected duplicate CreateAuthorizationRecord error")
	}
	if err := contract.RevokeAuthorization(ctx, "AUTH-1"); err != nil {
		t.Fatalf("RevokeAuthorization failed: %v", err)
	}
	var savedAuth AuthorizationRecord
	if err := json.Unmarshal(ctx.stub.state["AUTH_AUTH-1"], &savedAuth); err != nil {
		t.Fatalf("unmarshal AUTH-1 failed: %v", err)
	}
	if savedAuth.Status != "REVOKED" || savedAuth.EndorserMSPID != "HospitalAMSP" {
		t.Fatalf("expected REVOKED with HospitalAMSP stamp, got %+v", savedAuth)
	}

	// 3. CreateAuditRecord / CreateAuditLog 统一被 QueryAuditLogs 解析且拒绝空 ID/重复 ID
	badCtx := newTestCtx("InvalidMSPName", "bad")
	if err := contract.CreateAuditLog(badCtx, "LOG-BAD", "1", "ACCESS", "REC-1", "LOW", ""); err == nil {
		t.Fatalf("expected CreateAuditLog to enforce MSP format validation")
	}
	if err := contract.CreateAuditRecord(ctx, `{"log_id":"LOG-1","user_id":"1","op_type":"ACCESS_EVALUATE","target_id":"REC-1","risk_level":"LOW"}`); err != nil {
		t.Fatalf("CreateAuditRecord failed: %v", err)
	}
	logs, err := contract.QueryAuditLogs(ctx, "REC-1")
	if err != nil {
		t.Fatalf("QueryAuditLogs failed: %v", err)
	}
	if len(logs) != 1 || logs[0].AuditID != "LOG-1" || logs[0].Action != "ACCESS_EVALUATE" || logs[0].RecordID != "REC-1" {
		t.Fatalf("unexpected QueryAuditLogs result: %+v", logs)
	}
}
