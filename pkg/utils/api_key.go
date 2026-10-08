package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// Key ของข้อมูลรับรอง API — docs/modules/account.md ACC-03, ACC-04

// GenerateAPIKey สุ่ม 32 byte ด้วย crypto/rand แล้วคืนเป็น hex ตัวพิมพ์เล็ก 64 ตัว (ACC-03)
func GenerateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
