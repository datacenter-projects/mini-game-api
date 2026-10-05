package utils

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-at-least-32-characters!!"

var agentSubject = SessionSubject{AccountType: "AGENT", AccountID: 7}

func TestSessionTokenRoundTrip(t *testing.T) {
	now := time.Now()
	tests := []SessionSubject{
		agentSubject,
		{AccountType: "SUB", AccountID: 3, AgentID: 7},
	}
	for _, sub := range tests {
		tok, err := SignSessionToken(testSecret, TokenTypeBackoffice, "sid-1", sub, now, now.Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		c, err := ParseSessionToken(testSecret, TokenTypeBackoffice, tok)
		if err != nil {
			t.Fatal(err)
		}
		if c.SessionID != "sid-1" || c.AccountType != sub.AccountType || c.AccountID != sub.AccountID || c.AgentID != sub.AgentID {
			t.Fatalf("got %+v", c)
		}
	}
}

func TestParseSessionTokenRejects(t *testing.T) {
	now := time.Now()
	valid, _ := SignSessionToken(testSecret, TokenTypeBackoffice, "sid", agentSubject, now, now.Add(time.Hour))
	expired, _ := SignSessionToken(testSecret, TokenTypeBackoffice, "sid", agentSubject, now.Add(-2*time.Hour), now.Add(-time.Hour))
	otherType, _ := SignSessionToken(testSecret, "player", "sid", agentSubject, now, now.Add(time.Hour))
	noSID, _ := SignSessionToken(testSecret, TokenTypeBackoffice, "", agentSubject, now, now.Add(time.Hour))
	noAccountType, _ := SignSessionToken(testSecret, TokenTypeBackoffice, "sid", SessionSubject{AccountID: 7}, now, now.Add(time.Hour))
	noAccountID, _ := SignSessionToken(testSecret, TokenTypeBackoffice, "sid", SessionSubject{AccountType: "AGENT"}, now, now.Add(time.Hour))
	// token รูปแบบ phase 1: มีแค่ agent_id ไม่มี account_type / account_id
	phase1, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"typ": TokenTypeBackoffice, "sid": "sid", "agent_id": 7, "exp": now.Add(time.Hour).Unix(),
	}).SignedString([]byte(testSecret))
	noneAlg, _ := jwt.NewWithClaims(jwt.SigningMethodNone, SessionClaims{Type: TokenTypeBackoffice, SessionID: "sid", AccountType: "AGENT", AccountID: 1,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))}}).SignedString(jwt.UnsafeAllowNoneSignatureType)

	tests := []struct {
		name, secret, token string
	}{
		{"ลายเซ็นผิด secret", "another-secret-at-least-32-characters", valid},
		{"หมดอายุ", testSecret, expired},
		{"token ของ context อื่น", testSecret, otherType},
		{"ไม่มี sid", testSecret, noSID},
		{"ไม่มี account_type", testSecret, noAccountType},
		{"ไม่มี account_id", testSecret, noAccountID},
		{"token ของ phase 1", testSecret, phase1},
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
