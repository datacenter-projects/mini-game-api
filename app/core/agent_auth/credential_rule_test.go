package agentauth

import (
	"strings"
	"testing"
	"time"

	"app/app/models"
)

func TestIsValidPasscode(t *testing.T) { // AUTH-30
	tests := []struct {
		in   string
		want bool
	}{
		{"123456", true},
		{"000000", true},
		{"012345", true},
		{"12345", false},
		{"1234567", false},
		{"12345a", false},
		{"12 456", false},
		{"１２３４５６", false}, // เลขเต็มความกว้าง
		{"", false},
	}
	for _, tt := range tests {
		if got := IsValidPasscode(tt.in); got != tt.want {
			t.Errorf("IsValidPasscode(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestPasscodeCounter(t *testing.T) { // AUTH-35
	tests := []struct {
		fails, limit, left int
		blocked            bool
	}{
		{1, 5, 4, false},
		{3, 5, 2, false},
		{4, 5, 1, false},
		{5, 5, 0, true},
		{7, 5, 0, true},
		{9, 0, 0, false}, // limit 0 = ปิด
	}
	for _, tt := range tests {
		if got := PasscodeAttemptsLeft(tt.fails, tt.limit); got != tt.left {
			t.Errorf("PasscodeAttemptsLeft(%d, %d) = %d, want %d", tt.fails, tt.limit, got, tt.left)
		}
		if got := IsPasscodeBlocked(tt.fails, tt.limit); got != tt.blocked {
			t.Errorf("IsPasscodeBlocked(%d, %d) = %v, want %v", tt.fails, tt.limit, got, tt.blocked)
		}
	}
}

func TestCheckPasswordPolicy(t *testing.T) { // AUTH-36
	tests := []struct {
		name string
		in   string
		want PasswordViolation
	}{
		{"ปกติ", "Zx9!kq2m", PasswordOK},
		{"เรียงขึ้นได้ (ตัดกฎแล้ว)", "abcd1234", PasswordOK},
		{"punctuation ครบชุด", `Zx9~"'{}` + `!#$%&()*+,-./:;<=>?@[\]^_` + "`|", PasswordOK},
		{"ยาว 8 พอดี", "abcdefg1", PasswordOK},
		{"ยาว 64 พอดี", strings.Repeat("ab1", 21) + "a", PasswordOK},
		{"ซ้ำ 3 ได้", "Zx9aaam2", PasswordOK},
		{"7 ตัว", "Zx9!kq2", PasswordTooShort},
		{"65 ตัว", strings.Repeat("ab1", 21) + "ab", PasswordTooLong},
		{"ช่องว่างกลาง", "Zx9 kq2m", PasswordInvalidChar},
		{"ช่องว่างท้าย", "Zx9kq2m ", PasswordInvalidChar},
		{"อักษรไทย", "Zx9กkq2m", PasswordInvalidChar},
		{"tab", "Zx9\tkq2m", PasswordInvalidChar},
		{"ไม่มีตัวเลข", "Zxkqmwpt", PasswordNoDigit},
		{"ไม่มีตัวอักษร", "92837465", PasswordNoLetter},
		{"ซ้ำ 4", "Zx9aaaam", PasswordRepeated},
		{"เลขซ้ำ 4", "Zx1111km", PasswordRepeated},
		{"ตัวพิมพ์ต่างกันไม่นับว่าซ้ำ", "aAaA1bBb", PasswordOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckPasswordPolicy(tt.in); got != tt.want {
				t.Fatalf("CheckPasswordPolicy(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestTempPasswordAlphabet(t *testing.T) { // AUTH-46
	for _, banned := range "0Oo1lI" {
		if strings.ContainsRune(TempPasswordAlphabet, banned) {
			t.Errorf("alphabet ต้องไม่มี %q", banned)
		}
	}
	for _, ch := range TempPasswordAlphabet {
		if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9') {
			t.Errorf("alphabet ต้องมีแค่ตัวอักษรกับตัวเลข พบ %q", ch)
		}
	}
}

func TestIsTempExpired(t *testing.T) { // AUTH-28, AUTH-34, AUTH-41
	now := time.Date(2026, 10, 5, 10, 0, 0, 0, time.UTC)
	past, future := now.Add(-time.Second), now.Add(time.Second)
	tests := []struct {
		name       string
		mustChange bool
		expiresAt  *time.Time
		want       bool
	}{
		{"ยังไม่หมด", true, &future, false},
		{"หมดแล้ว", true, &past, true},
		{"ถึงเวลาพอดี = หมด", true, &now, true},
		{"ไม่ได้ถูกบังคับ", false, &past, false},
		{"ไม่มีวันหมดอายุ", true, nil, false},
	}
	for _, tt := range tests {
		if got := IsTempExpired(tt.mustChange, tt.expiresAt, now); got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIsResettableRole(t *testing.T) { // AUTH-45
	tests := []struct {
		role models.AgentRole
		want bool
	}{
		{models.AgentRoleCompany, true},
		{models.AgentRoleShareholder, true},
		{models.AgentRoleAgent, true},
		{models.AgentRoleSuperAdmin, false},
		{models.AgentRoleAdmin, false},
	}
	for _, tt := range tests {
		if got := IsResettableRole(tt.role); got != tt.want {
			t.Errorf("IsResettableRole(%s) = %v, want %v", tt.role, got, tt.want)
		}
	}
}

func TestCurrentGate(t *testing.T) { // AUTH-29
	tests := []struct {
		name                  string
		mustPw, mustPc, pcSet bool
		want                  Gate
	}{
		{"ผ่านครบ", false, false, true, GateNone},
		{"ยังไม่มี passcode", false, false, false, GateSetupPasscode},
		{"ต้องเปลี่ยน passcode", false, true, true, GateChangePasscode},
		{"ต้องเปลี่ยนรหัสผ่าน", true, false, true, GateChangePassword},
		{"ถูกรีเซ็ตทั้งสองอย่าง → รหัสผ่านก่อน", true, true, true, GateChangePassword},
		{"ต้องเปลี่ยนรหัสผ่าน + ยังไม่มี passcode", true, false, false, GateChangePassword},
	}
	for _, tt := range tests {
		if got := CurrentGate(tt.mustPw, tt.mustPc, tt.pcSet); got != tt.want {
			t.Errorf("%s: got %d, want %d", tt.name, got, tt.want)
		}
	}
}
