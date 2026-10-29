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

func TestEncryptDecryptMedicalFile_Success(t *testing.T) {
	masterKey := []byte("12345678901234567890123456789012") // 32 bytes
	plain := []byte("信封加密测试：患者心电图检查结论正常。")
	patientID := uint64(1005)
	recordNo := "REC20260901001"

	cipherData, encDEK, ivNonce, hash, err := EncryptMedicalFile(plain, patientID, recordNo, masterKey)
	if err != nil {
		t.Fatalf("信封加密失败: %v", err)
	}

	if encDEK == "" || ivNonce == "" || hash == "" {
		t.Fatalf("加密元数据不完整: dek=%s, iv=%s, hash=%s", encDEK, ivNonce, hash)
	}

	decrypted, err := DecryptMedicalFile(cipherData, encDEK, ivNonce, patientID, recordNo, masterKey)
	if err != nil {
		t.Fatalf("信封解密失败: %v", err)
	}

	if !bytes.Equal(decrypted, plain) {
		t.Fatalf("解密后明文不一致: got %s, want %s", string(decrypted), string(plain))
	}

	// 测试 AAD 不一致防篡改 (更换 patientID)
	_, err = DecryptMedicalFile(cipherData, encDEK, ivNonce, uint64(9999), recordNo, masterKey)
	if err == nil {
		t.Fatalf("预期 AAD patientID 不一致时解密报错，但未报错")
	}

	// 测试密文被篡改 1 字节
	tamperedCipher := make([]byte, len(cipherData))
	copy(tamperedCipher, cipherData)
	tamperedCipher[len(tamperedCipher)-1] ^= 0x01
	_, err = DecryptMedicalFile(tamperedCipher, encDEK, ivNonce, patientID, recordNo, masterKey)
	if err == nil {
		t.Fatalf("预期密文篡改后解密报错，但未报错")
	}
}

func TestRewrapDEK_Success(t *testing.T) {
	oldKey := []byte("old-secret-key-12345678901234567")[:32]
	newKey := []byte("new-secret-key-abcdefghijklmnopq")[:32]
	plain := []byte("密钥轮转测试：在不重新下载和加密 IPFS 原始密文的前提下，完成 DEK 重包装。")
	patientID := uint64(8888)
	recordNo := "REC-REKEY-2026-001"

	// 1. 使用旧主密钥加密
	cipherData, oldEncDEK, ivNonce, _, err := EncryptMedicalFile(plain, patientID, recordNo, oldKey)
	if err != nil {
		t.Fatalf("使用旧密钥加密失败: %v", err)
	}

	// 2. 确认旧主密钥可正常解密
	decOld, err := DecryptMedicalFile(cipherData, oldEncDEK, ivNonce, patientID, recordNo, oldKey)
	if err != nil || !bytes.Equal(decOld, plain) {
		t.Fatalf("旧密钥解密验证失败: %v", err)
	}

	// 3. 执行 DEK 重包装
	newEncDEK, err := RewrapDEK(oldEncDEK, oldKey, newKey)
	if err != nil {
		t.Fatalf("RewrapDEK 失败: %v", err)
	}
	if newEncDEK == oldEncDEK {
		t.Fatalf("重包装后密文应与旧密文不同 (随机 Nonce)")
	}

	// 4. 用新主密钥成功解密原始密文
	decNew, err := DecryptMedicalFile(cipherData, newEncDEK, ivNonce, patientID, recordNo, newKey)
	if err != nil {
		t.Fatalf("使用新密钥和新封装 DEK 解密失败: %v", err)
	}
	if !bytes.Equal(decNew, plain) {
		t.Fatalf("新密钥解密内容不一致: got %s, want %s", string(decNew), string(plain))
	}

	// 5. 反向校验：用旧主密钥解新封装 DEK 应失败
	_, err = DecryptMedicalFile(cipherData, newEncDEK, ivNonce, patientID, recordNo, oldKey)
	if err == nil {
		t.Fatalf("预期使用旧主密钥解新 DEK 报错，但未报错")
	}

	// 6. 错误密钥尝试 RewrapDEK 应失败
	wrongKey := []byte("wrong-secret-key-123456789012345")[:32]
	_, err = RewrapDEK(oldEncDEK, wrongKey, newKey)
	if err == nil {
		t.Fatalf("预期使用错误旧密钥解封装 DEK 报错，但未报错")
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
