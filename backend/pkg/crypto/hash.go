package crypto

import (
	"crypto/sha256"
	"encoding/hex"
)

// ComputeSHA256 计算字节流 SHA-256 十六进制哈希
func ComputeSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
