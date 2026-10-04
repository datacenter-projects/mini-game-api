package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenTypeBackoffice กันไม่ให้ token ของ context อื่น (เช่น player) ถูกใช้กับ route หลังบ้าน
const TokenTypeBackoffice = "bo"

var ErrInvalidJWT = errors.New("invalid jwt")

// SessionClaims — token เป็นแค่ "บัตรอ้างอิง" ไปหา session ใน Redis
// สิทธิ์/สถานะจริงอ่านจาก DB/Redis ทุกครั้ง ไม่เชื่อค่าใน token
//
// AccountType แยกตาราง (เช่น "AGENT" / "SUB") เพราะ id ของแต่ละตารางซ้ำกันได้
// AgentID มีเฉพาะบัญชีที่ทำงานในนามของ agent อื่น (sub → ผู้สร้าง) — docs/modules/agent_auth_phase2.md หัวข้อ 6
type SessionClaims struct {
	jwt.RegisteredClaims
	Type        string `json:"typ"`
	SessionID   string `json:"sid"`
	AccountType string `json:"account_type"`
	AccountID   uint   `json:"account_id"`
	AgentID     uint   `json:"agent_id,omitempty"`
}

// SessionSubject คือเจ้าของ session ที่ใส่ลงใน token
type SessionSubject struct {
	AccountType string
	AccountID   uint
	AgentID     uint // 0 = ไม่มี
}

func SignSessionToken(secret, tokenType, sessionID string, sub SessionSubject, issuedAt, expiresAt time.Time) (string, error) {
	claims := SessionClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		Type:        tokenType,
		SessionID:   sessionID,
		AccountType: sub.AccountType,
		AccountID:   sub.AccountID,
		AgentID:     sub.AgentID,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ParseSessionToken ตรวจลายเซ็น (HS256 เท่านั้น), วันหมดอายุ, ชนิด token และ claim ที่จำเป็น
// token ที่ไม่มี account_type (ออกก่อน phase 2) ถือว่า invalid
func ParseSessionToken(secret, tokenType, raw string) (*SessionClaims, error) {
	claims := &SessionClaims{}
	_, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return nil, errors.Join(ErrInvalidJWT, err)
	}
	if claims.Type != tokenType || claims.SessionID == "" || claims.AccountType == "" || claims.AccountID == 0 {
		return nil, ErrInvalidJWT
	}
	return claims, nil
}
