// Package agentauth คือกฎของ backoffice auth แบบ pure function — spec: docs/modules/agent_auth.md
package agentauth

import (
	"strings"
	"time"
)

// NormalizeUsername — rule: AUTH-01
func NormalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

// IsSubaccountUsername — username รูปแบบ "agent@sub" เป็นของ subaccount (spec หัวข้อ 4 ข้อ 10)
func IsSubaccountUsername(username string) bool {
	return strings.Contains(username, "@")
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
