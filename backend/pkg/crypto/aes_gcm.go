package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// BuildAAD 构造绑定患者ID与病历编号的附加认证数据 (防密文替换重放攻击)
func BuildAAD(patientID uint64, recordNo string) []byte {
	return []byte(fmt.Sprintf("patient:%d|record:%s", patientID, recordNo))
}

// EncryptMedicalFile 生成随机 256-bit DEK 与 96-bit IV，执行带 AAD 的 AES-256-GCM 加密 (信封加密模式)
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
