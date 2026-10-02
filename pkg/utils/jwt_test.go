package utils

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-at-least-32-characters!!"

func TestSessionTokenRoundTrip(t *testing.T) {
	now := time.Now()
	tok, err := SignSessionToken(testSecret, TokenTypeBackoffice, "sid-1", 7, now, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	c, err := ParseSessionToken(testSecret, TokenTypeBackoffice, tok)
	if err != nil {
		t.Fatal(err)
	}
	if c.SessionID != "sid-1" || c.AgentID != 7 {
		t.Fatalf("got %+v", c)
	}
}

func TestParseSessionTokenRejects(t *testing.T) {
	now := time.Now()
	valid, _ := SignSessionToken(testSecret, TokenTypeBackoffice, "sid", 1, now, now.Add(time.Hour))
	expired, _ := SignSessionToken(testSecret, TokenTypeBackoffice, "sid", 1, now.Add(-2*time.Hour), now.Add(-time.Hour))
	otherType, _ := SignSessionToken(testSecret, "player", "sid", 1, now, now.Add(time.Hour))
	noSID, _ := SignSessionToken(testSecret, TokenTypeBackoffice, "", 1, now, now.Add(time.Hour))
	noneAlg, _ := jwt.NewWithClaims(jwt.SigningMethodNone, SessionClaims{Type: TokenTypeBackoffice, SessionID: "sid", AgentID: 1,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))}}).SignedString(jwt.UnsafeAllowNoneSignatureType)

	tests := []struct {
		name, secret, token string
	}{
		{"ลายเซ็นผิด secret", "another-secret-at-least-32-characters", valid},
		{"หมดอายุ", testSecret, expired},
		{"token ของ context อื่น", testSecret, otherType},
		{"ไม่มี sid", testSecret, noSID},
		{"alg none", testSecret, noneAlg},
		{"ขยะ", testSecret, "not-a-jwt"},
		{"ว่าง", testSecret, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseSessionToken(tt.secret, TokenTypeBackoffice, tt.token); !errors.Is(err, ErrInvalidJWT) {
				t.Fatalf("want ErrInvalidJWT, got %v", err)
			}
		})
	}
}
