package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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

// HashAPIKey — sha256 ของ Key เป็น hex 64 ตัว ใช้ค้นตอนลูกค้าเรียก API (ACC-04)
func HashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// EncryptAPIKey เข้ารหัส Key ด้วย AES-256-GCM คืน nonce + ciphertext (ACC-04) — encKey ต้องยาว 32 byte
func EncryptAPIKey(encKey []byte, key string) ([]byte, error) {
	gcm, err := newGCM(encKey)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, []byte(key), nil), nil
}

// DecryptAPIKey ถอดรหัสค่าจาก EncryptAPIKey (ACC-04)
func DecryptAPIKey(encKey, ciphertext []byte) (string, error) {
	gcm, err := newGCM(encKey)
	if err != nil {
		return "", err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return "", errors.New("api key ciphertext too short")
	}
	nonce, data := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func newGCM(encKey []byte) (cipher.AEAD, error) {
	if len(encKey) != 32 {
		return nil, errors.New("api key encryption key must be 32 bytes")
	}
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
