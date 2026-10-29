package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

const DefaultMasterSecret = "MedTrust_System_Master_Key_AES256_GCM_2026"

// ComputeSHA256 计算输入字节流的 SHA-256 十六进制摘要
func ComputeSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// CalculateSHA256 向后兼容别名函数
func CalculateSHA256(data []byte) string {
	return ComputeSHA256(data)
}

// DeriveRecordKey 基于主密钥与病历唯一编号派生 256 位对称加密密钥 (HKDF/HMAC-SHA256 机制)
func DeriveRecordKey(masterSecret, recordNo string) []byte {
	if masterSecret == "" {
		masterSecret = DefaultMasterSecret
	}
	mac := hmac.New(sha256.New, []byte(masterSecret))
	mac.Write([]byte("MedTrust:RecordKey:" + recordNo))
	return mac.Sum(nil)
}

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

// EncryptAES256GCM 执行底层 AES-256-GCM 对称加密计算并生成 16 字节认证标签
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

// DecryptAES256GCM 执行底层 AES-256-GCM 解密并在内存中强制校验 AAD 与认证标签 Tag
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

// EncryptRecordFile 高阶流式加密：计算明文 SHA-256 指纹，派生密钥，执行 AAD 绑定加密并输出 [12-byte IV][密文+Tag] 密文包
func EncryptRecordFile(plainData []byte, patientID uint64, recordNo string) (pack []byte, fileHash string, err error) {
	fileHash = CalculateSHA256(plainData)
	key := DeriveRecordKey("", recordNo)
	iv, err := GenerateRandomIV()
	if err != nil {
		return nil, "", fmt.Errorf("生成 IV 失败: %w", err)
	}

	aad := []byte(fmt.Sprintf("%d:%s", patientID, recordNo))
	ciphertext, err := EncryptAES256GCM(key, iv, plainData, aad)
	if err != nil {
		return nil, "", fmt.Errorf("AES-256-GCM 加密失败: %w", err)
	}

	pack = append(iv, ciphertext...)
	return pack, fileHash, nil
}

// DecryptRecordFile 高阶内存流式解密：从密文包解析 IV，重新派生密钥并基于 AAD 校验解密，全程内存运算零磁盘落地
func DecryptRecordFile(ciphertextPack []byte, patientID uint64, recordNo string) ([]byte, error) {
	if len(ciphertextPack) < 12+16 {
		return nil, fmt.Errorf("密文数据包长度不足 (需至少包含 12 字节 IV 与 16 字节 GCM Tag)")
	}

	iv := ciphertextPack[:12]
	ciphertext := ciphertextPack[12:]
	key := DeriveRecordKey("", recordNo)
	aad := []byte(fmt.Sprintf("%d:%s", patientID, recordNo))

	return DecryptAES256GCM(key, iv, ciphertext, aad)
}

// BuildAAD 构造绑定患者ID与病历编号的附加认证数据 (防密文替换重放攻击)
func BuildAAD(patientID uint64, recordNo string) []byte {
	return []byte(fmt.Sprintf("patient:%d|record:%s", patientID, recordNo))
}

// EncryptMedicalFile 生成随机 256-bit DEK 与 96-bit IV，执行带 AAD 的 AES-256-GCM 信封加密模式 (大论文 4.2.1 节核心算法)
func EncryptMedicalFile(plainData []byte, patientID uint64, recordNo string, masterKey []byte) (cipherData []byte, encryptedDEK string, ivBase64 string, fileHash string, err error) {
	if len(masterKey) != 32 {
		return nil, "", "", "", errors.New("masterKey must be 32 bytes for AES-256")
	}
	// 1. 计算原始明文 SHA-256
	fileHash = ComputeSHA256(plainData)

	// 2. 生成 32 字节随机 DEK 与 12 字节随机 Nonce (IV)
	dek := make([]byte, 32)
	if _, err = io.ReadFull(rand.Reader, dek); err != nil {
		return
	}
	iv := make([]byte, 12)
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return
	}

	// 3. 使用 DEK + IV + AAD 加密明文病历
	block, err := aes.NewCipher(dek)
	if err != nil {
		return
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return
	}
	aad := BuildAAD(patientID, recordNo)
	cipherData = aesGCM.Seal(nil, iv, plainData, aad)

	// 4. 使用 MasterKey 加密保护 DEK (信封加密)
	masterBlock, err := aes.NewCipher(masterKey)
	if err != nil {
		return
	}
	masterGCM, err := cipher.NewGCM(masterBlock)
	if err != nil {
		return
	}
	dekNonce := make([]byte, 12)
	if _, err = io.ReadFull(rand.Reader, dekNonce); err != nil {
		return
	}
	sealedDEK := masterGCM.Seal(dekNonce, dekNonce, dek, nil) // 前12字节存dekNonce

	encryptedDEK = base64.StdEncoding.EncodeToString(sealedDEK)
	ivBase64 = base64.StdEncoding.EncodeToString(iv)
	return cipherData, encryptedDEK, ivBase64, fileHash, nil
}

// DecryptMedicalFile 解封 DEK 并校验 AAD 与 AuthTag，还原病历明文
func DecryptMedicalFile(cipherData []byte, encryptedDEK string, ivBase64 string, patientID uint64, recordNo string, masterKey []byte) ([]byte, error) {
	sealedDEK, err := base64.StdEncoding.DecodeString(encryptedDEK)
	if err != nil || len(sealedDEK) < 12 {
		return nil, errors.New("invalid encrypted DEK")
	}
	iv, err := base64.StdEncoding.DecodeString(ivBase64)
	if err != nil {
		return nil, errors.New("invalid IV nonce")
	}

	// 1. 解封 DEK
	masterBlock, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, err
	}
	masterGCM, err := cipher.NewGCM(masterBlock)
	if err != nil {
		return nil, err
	}
	dekNonce := sealedDEK[:12]
	dek, err := masterGCM.Open(nil, dekNonce, sealedDEK[12:], nil)
	if err != nil {
		return nil, errors.New("failed to unseal DEK: " + err.Error())
	}

	// 2. 使用 DEK + IV + AAD 解密密文并验证 GCM AuthTag
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	aad := BuildAAD(patientID, recordNo)
	plainData, err := aesGCM.Open(nil, iv, cipherData, aad)
	if err != nil {
		return nil, errors.New("ciphertext authentication failed (possible tampering or AAD mismatch)")
	}
	return plainData, nil
}

// RewrapDEK 用新主密钥重新封装旧主密钥加密的 DEK (信封加密密匙轮转，零密文搬运热轮换)
func RewrapDEK(oldEncryptedDEK string, oldMasterKey, newMasterKey []byte) (string, error) {
	if len(oldMasterKey) != 32 || len(newMasterKey) != 32 {
		return "", errors.New("master keys must be exactly 32 bytes for AES-256")
	}
	sealedDEK, err := base64.StdEncoding.DecodeString(oldEncryptedDEK)
	if err != nil || len(sealedDEK) < 12 {
		return "", errors.New("invalid encrypted DEK: decode failed or length < 12")
	}

	// 1. 使用旧主密钥解开 DEK
	oldBlock, err := aes.NewCipher(oldMasterKey)
	if err != nil {
		return "", fmt.Errorf("init old cipher failed: %w", err)
	}
	oldGCM, err := cipher.NewGCM(oldBlock)
	if err != nil {
		return "", fmt.Errorf("init old gcm failed: %w", err)
	}
	dekNonce := sealedDEK[:12]
	dek, err := oldGCM.Open(nil, dekNonce, sealedDEK[12:], nil)
	if err != nil {
		return "", fmt.Errorf("failed to unseal DEK with old master key: %w", err)
	}

	// 2. 生成全新 12 字节 Nonce 并用新主密钥重新封装 DEK
	newBlock, err := aes.NewCipher(newMasterKey)
	if err != nil {
		return "", fmt.Errorf("init new cipher failed: %w", err)
	}
	newGCM, err := cipher.NewGCM(newBlock)
	if err != nil {
		return "", fmt.Errorf("init new gcm failed: %w", err)
	}
	newNonce := make([]byte, 12)
	if _, err = io.ReadFull(rand.Reader, newNonce); err != nil {
		return "", fmt.Errorf("generate new nonce failed: %w", err)
	}
	newSealedDEK := newGCM.Seal(newNonce, newNonce, dek, nil)

	return base64.StdEncoding.EncodeToString(newSealedDEK), nil
}

