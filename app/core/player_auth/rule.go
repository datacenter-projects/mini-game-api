// Package playerauth คือกฎของ Player Auth — pure function ไม่แตะ DB / Redis / config (spec: docs/modules/player_auth.md)
package playerauth

import (
	"crypto/subtle"
	"net/netip"
	"net/url"
	"regexp"
	"strings"

	agentAuthCore "app/app/core/agent_auth"
	agentManagementCore "app/app/core/agent_management"
	"app/app/models"
)

// PlayerUsernameMax — ส่วน player ยาวสุดกี่ตัว (PA-05) · username ที่เก็บ = agent (≤ 32) + "@" + player (≤ 32) = 65 (PA-06)
const (
	PlayerUsernameMin = 3
	PlayerUsernameMax = 32
	StoredUsernameMax = 65
)

var playerUsernameRe = regexp.MustCompile(`^[a-z0-9]{3,32}$`)

// NormalizeUsername — trim + ตัวพิมพ์เล็ก (PA-04) ใช้กับ username ที่รับทางตรงและ player_username
func NormalizeUsername(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// IsPlayerUsername — ส่วน player ของผู้เล่นจาก operator: a-z0-9 ยาว 3–32 (PA-05) · ส่งมาเป็น normalized แล้ว
func IsPlayerUsername(s string) bool { return playerUsernameRe.MatchString(s) }

// ComposeUsername — username ที่เก็บของผู้เล่นจาก operator = {agent}@{player} (PA-05)
// agentUsername ของ agent ยาวไม่เกิน 32 และไม่มี "@" (a-z0-9) จึงได้ไม่เกิน StoredUsernameMax เสมอ
func ComposeUsername(agentUsername, player string) string {
	return NormalizeUsername(agentUsername) + "@" + player
}

// CanUseTransfer — ผู้สร้างที่เรียก transfer/* ได้: Share B2C และ Agent เท่านั้น (PA-14 · lead Q1)
func CanUseTransfer(t agentManagementCore.UserType) bool {
	return t == agentManagementCore.UserTypeShareB2C || t == agentManagementCore.UserTypeAgent
}

// EffectiveStatus — สถานะที่ใช้งานจริง = แย่ที่สุดของ Member + ผู้สร้าง + สายบนทุกคน (หัวข้อ 3.4)
func EffectiveStatus(own models.AgentStatus, upline ...models.AgentStatus) models.AgentStatus {
	return agentAuthCore.WorstStatus(append([]models.AgentStatus{own}, upline...)...)
}

// CanBet — เดิมพันได้เมื่อสถานะที่ใช้งานจริงเป็น ACTIVE และ win_limit_per_day > 0 (PA-35 · MGMT-65: 0 = เล่นไม่ได้)
func CanBet(effective models.AgentStatus, winLimitPerDay float64) bool {
	return effective == models.AgentStatusActive && winLimitPerDay > 0
}

// สถานะของ agent ที่ operator ใช้เรียก transfer/* (PA-13)
type OperatorGate int

const (
	OperatorOK        OperatorGate = iota
	OperatorLocked                 // agent หรือสายบน LOCKED → 407302 ทุกเส้น
	OperatorSuspended              // SUSPENDED → ผู้เล่นเดิมใช้ได้ · สร้างผู้เล่นใหม่ไม่ได้ (407308)
)

// OperatorGateOf — สถานะรวมของ agent เจ้าของ Key (ตัวเอง + สายบน)
func OperatorGateOf(own models.AgentStatus, upline ...models.AgentStatus) OperatorGate {
	switch EffectiveStatus(own, upline...) {
	case models.AgentStatusLocked:
		return OperatorLocked
	case models.AgentStatusSuspended:
		return OperatorSuspended
	}
	return OperatorOK
}

// KeyEqual — เทียบ Key แบบ constant-time (PA-11)
func KeyEqual(stored, given string) bool {
	return subtle.ConstantTimeCompare([]byte(stored), []byte(given)) == 1
}

// IPAllowed — ip อยู่ใน whitelist (IPv4 / CIDR) ไหม · whitelist ว่าง = ปฏิเสธทุก IP (PA-10 · ACC-07)
// ip ที่แปลงไม่ได้ หรือ cidr ที่เสีย = ไม่ผ่าน (fail-closed)
func IPAllowed(ip string, cidrs []string) bool {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	addr = addr.Unmap()
	for _, c := range cidrs {
		p, err := netip.ParsePrefix(c)
		if err != nil {
			continue
		}
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

// Language ที่รับใน transfer/login (L2)
const (
	LangTH = "th"
	LangEN = "en"
)

// NormalizeLanguage — ว่าง = th · th / en เท่านั้น · อื่น ok false (422)
func NormalizeLanguage(s string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", LangTH:
		return LangTH, true
	case LangEN:
		return LangEN, true
	}
	return "", false
}

// LobbyLink — link เข้า lobby: base + ?token=<token>&lang=<lang> (LQ-c · lead Q12)
// base ที่มี query อยู่แล้วต่อ & ให้ · base ที่อ่านไม่ได้ ok false
func LobbyLink(base, token, lang string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(base))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", false
	}
	extra := "token=" + url.QueryEscape(token) + "&lang=" + url.QueryEscape(lang) // เรียง token ก่อน lang ตาม spec
	if u.RawQuery != "" {
		u.RawQuery += "&"
	}
	u.RawQuery += extra
	return u.String(), true
}
