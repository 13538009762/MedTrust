package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

const DefaultMasterSecret = "MedTrust_System_Master_Key_AES256_GCM_2026"

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

// EncryptAES256GCM 执行 AES-256-GCM 对称加密计算并生成 16 字节认证标签
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

// DecryptAES256GCM 执行 AES-256-GCM 解密并在内存中强制校验 AAD 与 认证标签 Tag
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

func CalculateSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
