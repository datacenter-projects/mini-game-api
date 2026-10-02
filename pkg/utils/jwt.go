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
type SessionClaims struct {
	jwt.RegisteredClaims
	Type      string `json:"typ"`
	SessionID string `json:"sid"`
	AgentID   uint   `json:"agent_id"`
}

func SignSessionToken(secret, tokenType, sessionID string, agentID uint, issuedAt, expiresAt time.Time) (string, error) {
	claims := SessionClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		Type:      tokenType,
		SessionID: sessionID,
		AgentID:   agentID,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ParseSessionToken ตรวจลายเซ็น (HS256 เท่านั้น), วันหมดอายุ และชนิด token
func ParseSessionToken(secret, tokenType, raw string) (*SessionClaims, error) {
	claims := &SessionClaims{}
	_, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return nil, errors.Join(ErrInvalidJWT, err)
	}
	if claims.Type != tokenType || claims.SessionID == "" || claims.AgentID == 0 {
		return nil, ErrInvalidJWT
	}
	return claims, nil
}
