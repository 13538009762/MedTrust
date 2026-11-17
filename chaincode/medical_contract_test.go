package main

import (
	"encoding/json"
	"testing"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

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
