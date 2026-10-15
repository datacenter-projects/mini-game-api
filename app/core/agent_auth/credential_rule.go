package agentauth

import (
	"fmt"
	"time"
	"unicode/utf8"

	"app/app/models"
)

// ---- passcode (AUTH-30, AUTH-35) ----

const PasscodeLength = 6

// IsValidPasscode — ตัวเลข 0-9 ครบ 6 หลัก (AUTH-30)
func IsValidPasscode(s string) bool {
	if len(s) != PasscodeLength {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// PasscodeAttemptsLeft — จำนวนครั้งที่ยังใส่ผิดได้ก่อนถูกบล็อก (AUTH-35) ไม่ติดลบ
func PasscodeAttemptsLeft(failCount, limit int) int {
	if left := limit - failCount; left > 0 {
		return left
	}
	return 0
}

// IsPasscodeBlocked — ผิดครบ limit แล้ว (AUTH-35) · limit 0 = ปิด
func IsPasscodeBlocked(failCount, limit int) bool {
	return limit > 0 && failCount >= limit
}

// ---- password (AUTH-36) ----

const (
	PasswordMinLen = 8
	PasswordMaxLen = 64
	// ห้ามตัวเดียวกันซ้ำติดกันตั้งแต่ 4 ตัว
	PasswordMaxRepeat = 3
)

type PasswordViolation int

const (
	PasswordOK PasswordViolation = iota
	PasswordInvalidChar
	PasswordTooShort
	PasswordTooLong
	PasswordNoLetter
	PasswordNoDigit
	PasswordRepeated
)

// CheckPasswordPolicy — rule: AUTH-36
// อักขระที่ใช้ได้ = ASCII ที่พิมพ์ได้ยกเว้นช่องว่าง (0x21–0x7E) คือ a-z, A-Z, 0-9 และ ASCII punctuation
func CheckPasswordPolicy(pw string) PasswordViolation {
	var hasLetter, hasDigit bool
	for i := 0; i < len(pw); i++ {
		ch := pw[i]
		switch {
		case ch < 0x21 || ch > 0x7E:
			return PasswordInvalidChar
		case (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z'):
			hasLetter = true
		case ch >= '0' && ch <= '9':
			hasDigit = true
		}
	}
	n := utf8.RuneCountInString(pw) // ผ่านด่านอักขระแล้ว = ASCII ล้วน
	switch {
	case n < PasswordMinLen:
		return PasswordTooShort
	case n > PasswordMaxLen:
		return PasswordTooLong
	case !hasLetter:
		return PasswordNoLetter
	case !hasDigit:
		return PasswordNoDigit
	case hasRepeatRun(pw, PasswordMaxRepeat+1):
		return PasswordRepeated
	}
	return PasswordOK
}

func hasRepeatRun(s string, run int) bool {
	count := 1
	for i := 1; i < len(s); i++ {
		if s[i] == s[i-1] {
			count++
			if count >= run {
				return true
			}
		} else {
			count = 1
		}
	}
	return false
}

// ---- ค่าชั่วคราวจาก admin reset (AUTH-46) ----

const (
	TempPasswordLength = 12
	// ตัวอักษรกับตัวเลข ไม่มี 0 O o 1 l I
	TempPasswordAlphabet = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

// IsTempExpired — ค่าชั่วคราวหมดอายุแล้ว (AUTH-28, AUTH-34, AUTH-41)
// ใช้ได้เฉพาะตอนยังถูกบังคับเปลี่ยน · ไม่มีวันหมดอายุ = ไม่หมด
func IsTempExpired(mustChange bool, expiresAt *time.Time, now time.Time) bool {
	return mustChange && expiresAt != nil && !now.Before(*expiresAt)
}

// IsResettableRole — admin รีเซ็ตได้เฉพาะ COMPANY / SHAREHOLDER / AGENT (AUTH-45)
func IsResettableRole(role models.AgentRole) bool {
	switch role {
	case models.AgentRoleCompany, models.AgentRoleShareholder, models.AgentRoleAgent:
		return true
	}
	return false
}

// ---- ด่านหลัง login (AUTH-29) ----

type Gate int

const (
	GateNone           Gate = iota // ผ่านครบแล้ว
	GateChangePassword             // ข้อ 1
	GateChangePasscode             // ข้อ 2
	GateSetupPasscode              // ข้อ 3
)

// CurrentGate คืนด่านแรกที่ session ยังไม่ผ่าน ตามลำดับ AUTH-29
func CurrentGate(mustChangePassword, mustChangePasscode, passcodeSet bool) Gate {
	switch {
	case mustChangePassword:
		return GateChangePassword
	case mustChangePasscode:
		return GateChangePasscode
	case !passcodeSet:
		return GateSetupPasscode
	}
	return GateNone
}

// PasswordViolationMessage — ข้อความ (ไทย / อังกฤษ) ของผลตรวจรหัสผ่านที่ไม่ผ่าน (AUTH-36) ไม่รวมชื่อ field · PasswordOK = ว่าง
// ใช้ร่วม backoffice และ player_auth (frontend ห้าม import dto ของ backoffice)
func PasswordViolationMessage(v PasswordViolation) (th, en string) {
	switch v {
	case PasswordOK:
		return "", ""
	case PasswordInvalidChar:
		return "ใช้ได้เฉพาะ a-z, A-Z, 0-9 และอักขระพิเศษ ห้ามมีช่องว่าง", "may contain only a-z, A-Z, 0-9 and ASCII symbols, no spaces"
	case PasswordTooShort:
		return fmt.Sprintf("ต้องยาวอย่างน้อย %d ตัว", PasswordMinLen), fmt.Sprintf("must be at least %d characters", PasswordMinLen)
	case PasswordTooLong:
		return fmt.Sprintf("ต้องยาวไม่เกิน %d ตัว", PasswordMaxLen), fmt.Sprintf("must not exceed %d characters", PasswordMaxLen)
	case PasswordNoLetter:
		return "ต้องมีตัวอักษรอย่างน้อย 1 ตัว", "must contain at least 1 letter"
	case PasswordNoDigit:
		return "ต้องมีตัวเลขอย่างน้อย 1 ตัว", "must contain at least 1 digit"
	case PasswordRepeated:
		return fmt.Sprintf("ห้ามมีตัวเดียวกันซ้ำติดกัน %d ตัวขึ้นไป", PasswordMaxRepeat+1), fmt.Sprintf("must not repeat the same character %d or more times in a row", PasswordMaxRepeat+1)
	}
	return "รูปแบบไม่ถูกต้อง", "is invalid"
}
