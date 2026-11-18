package ipfs

import "testing"

func TestIPFSClient_RejectsTraversalCIDAndRoundtripsValidCID(t *testing.T) {
	dir := t.TempDir()
	mockClient := NewLocalMockIPFSClient(dir)
	liveClient := NewKuboLiveClient("http://127.0.0.1:59999", dir)

	badCIDs := []string{"", ".", "..", "../../etc/passwd", `..\..\windows\win.ini`, "Qm123?arg=evil", "cid/slash"}
	for _, bad := range badCIDs {
		if _, err := mockClient.Cat(bad); err == nil {
			t.Fatalf("LocalMockIPFSClient.Cat 应拒绝非法/路径穿越 CID %q", bad)
		}
		if _, err := liveClient.Cat(bad); err == nil {
			t.Fatalf("KuboLiveClient.Cat 应拒绝非法/路径穿越 CID %q", bad)
		}
		if bad != "" {
			if err := mockClient.Delete(bad); err == nil {
				t.Fatalf("LocalMockIPFSClient.Delete 应拒绝非法/路径穿越 CID %q", bad)
			}
			if err := liveClient.Delete(bad); err == nil {
				t.Fatalf("KuboLiveClient.Delete 应拒绝非法/路径穿越 CID %q", bad)
			}
		}
	}

	payload := []byte("ENCRYPTED_TEST_CIPHER_BLOCK")
	cid, err := liveClient.Add(payload)
	if err != nil {
		t.Fatalf("live fallback Add failed: %v", err)
	}
	got, err := liveClient.Cat(cid)
	if err != nil || string(got) != string(payload) {
		t.Fatalf("live fallback Cat mismatch: err=%v got=%q", err, string(got))
	}
}
