// Package agentauth คือกฎของ backoffice auth แบบ pure function
// spec: docs/modules/agent_auth.md, docs/modules/agent_auth_phase2.md
package agentauth

import (
	"regexp"
	"strings"
	"time"

	"app/app/models"
)

// NormalizeUsername — rule: AUTH-01
func NormalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

// IsSubaccountUsername — username ที่มี "@" เป็นของ subaccount เสมอ (AUTH-18)
func IsSubaccountUsername(username string) bool {
	return strings.Contains(username, "@")
}

var subaccountNameRe = regexp.MustCompile(`^[a-z0-9]{3,20}$`)

// SplitSubaccountUsername แยก "{owner}@{name}" — ok=false เมื่อรูปแบบผิด (AUTH-18)
// รับ username ที่ผ่าน NormalizeUsername แล้ว
func SplitSubaccountUsername(username string) (owner, name string, ok bool) {
	owner, name, found := strings.Cut(username, "@")
	if !found || owner == "" || strings.Contains(name, "@") || !subaccountNameRe.MatchString(name) {
		return "", "", false
	}
	return owner, name, true
}

var statusRank = map[models.AgentStatus]int{
	models.AgentStatusActive:    0,
	models.AgentStatusSuspended: 1,
	models.AgentStatusLocked:    2,
}

// EffectiveStatus — สถานะที่ใช้ตัดสินสิทธิ์ของ sub = สถานะที่เข้มที่สุดระหว่าง sub กับผู้สร้าง
// (LOCKED > SUSPENDED > ACTIVE) · agent ส่ง creator เป็นค่าว่างแล้วได้สถานะของตัวเอง
// (นิยามที่เสนอใน review รอบ 2 ยังรอ lead ยืนยัน)
func EffectiveStatus(own, creator models.AgentStatus) models.AgentStatus {
	if statusRank[creator] > statusRank[own] {
		return creator
	}
	return own
}

// SessionTTL คืนอายุ idle ที่ควรตั้งให้ session ตอนนี้ — rule: AUTH-07, AUTH-08
// = idle timeout แต่ไม่เกินเวลาที่เหลือถึง absolute deadline · คืน <= 0 แปลว่า session หมดอายุแล้ว
func SessionTTL(now, absoluteDeadline time.Time, idle time.Duration) time.Duration {
	remaining := absoluteDeadline.Sub(now)
	if remaining < idle {
		return remaining
	}
	return idle
}

// IsLoginBlocked — rule: AUTH-10 · failCount = จำนวนครั้งที่ผิดในหน้าต่างเวลาปัจจุบัน
func IsLoginBlocked(failCount, limit int) bool {
	return limit > 0 && failCount >= limit
}
