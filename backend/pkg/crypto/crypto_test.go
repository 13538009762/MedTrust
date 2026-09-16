package crypto

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptRecordFile_Success(t *testing.T) {
	plain := []byte("临床电子病历敏感数据：患者确诊高血压，遵医嘱口服氨氯地平片 5mg QD。")
	patientID := uint64(1001)
	recordNo := "ENC20260916001"

	pack, fileHash, err := EncryptRecordFile(plain, patientID, recordNo)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	if len(pack) <= len(plain) {
		t.Fatalf("密文数据包长度异常，至少应包含 12 字节 IV + 16 字节 Tag")
	}

	expectedHash := CalculateSHA256(plain)
	if fileHash != expectedHash {
		t.Fatalf("计算明文哈希不匹配: got %s, want %s", fileHash, expectedHash)
	}

	decrypted, err := DecryptRecordFile(pack, patientID, recordNo)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}

	if !bytes.Equal(decrypted, plain) {
		t.Fatalf("解密明文与原始数据不一致: got %s, want %s", string(decrypted), string(plain))
	}
}

func TestDecryptRecordFile_TamperedCiphertext(t *testing.T) {
	plain := []byte("敏感医疗诊断与过敏史数据")
	patientID := uint64(1002)
	recordNo := "ENC20260916002"

	pack, _, err := EncryptRecordFile(plain, patientID, recordNo)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	// 恶意篡改密文包中的字节（模拟中间人攻击或存储介质篡改）
	tamperedPack := make([]byte, len(pack))
	copy(tamperedPack, pack)
	tamperedPack[len(tamperedPack)-5] ^= 0xFF

	_, err = DecryptRecordFile(tamperedPack, patientID, recordNo)
	if err == nil {
		t.Fatalf("预期篡改密文后解密失败，但解密成功 (GCM 认证标签验证失效)")
	}
}

func TestDecryptRecordFile_MismatchedPatientID(t *testing.T) {
	plain := []byte("患者隐私就诊记录")
	patientID := uint64(1003)
	wrongPatientID := uint64(9999)
	recordNo := "ENC20260916003"

	pack, _, err := EncryptRecordFile(plain, patientID, recordNo)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	// 使用错误的 patientID 尝试解密（AAD 校验不通过）
	_, err = DecryptRecordFile(pack, wrongPatientID, recordNo)
	if err == nil {
		t.Fatalf("预期 AAD patientID 不一致时解密失败，但解密成功")
	}
}

func TestDecryptRecordFile_MismatchedRecordNo(t *testing.T) {
	plain := []byte("患者隐私就诊记录")
	patientID := uint64(1004)
	recordNo := "ENC20260916004"
	wrongRecordNo := "ENC20260916999"

	pack, _, err := EncryptRecordFile(plain, patientID, recordNo)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	// 使用错误的 recordNo 尝试解密（密钥派生与 AAD 均不一致）
	_, err = DecryptRecordFile(pack, patientID, wrongRecordNo)
	if err == nil {
		t.Fatalf("预期 recordNo 不一致时解密失败，但解密成功")
	}
}

func TestCalculateSHA256(t *testing.T) {
	data := []byte("MedTrust-Blockchain-Security-2026")
	hash := CalculateSHA256(data)
	if len(hash) != 64 {
		t.Fatalf("SHA-256 哈希长度错误: %d (需为 64 位十六进制)", len(hash))
	}
}

func BenchmarkEncryptRecordFile(b *testing.B) {
	data := bytes.Repeat([]byte("A"), 1024*64) // 64KB 典型结构化病历与检验报告
	patientID := uint64(2001)
	recordNo := "ENC-BENCH-001"

	b.ResetTimer()
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		_, _, err := EncryptRecordFile(data, patientID, recordNo)
		if err != nil {
			b.Fatalf("benchmark encrypt failed: %v", err)
		}
	}
}

func BenchmarkDecryptRecordFile(b *testing.B) {
	data := bytes.Repeat([]byte("B"), 1024*64) // 64KB
	patientID := uint64(2002)
	recordNo := "ENC-BENCH-002"
	pack, _, _ := EncryptRecordFile(data, patientID, recordNo)

	b.ResetTimer()
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		_, err := DecryptRecordFile(pack, patientID, recordNo)
		if err != nil {
			b.Fatalf("benchmark decrypt failed: %v", err)
		}
	}
}
