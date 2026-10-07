// Package account คือกฎของหน้าบัญชีหลังบ้านแบบ pure function
// spec: docs/modules/account.md · ประเภทบัญชี สกุลเงิน และสิทธิ์ ใช้ของ module ② (app/core/agent_management)
package account

import (
	"time"

	agentManagementCore "app/app/core/agent_management"
)

// OptionalTime — เวลาที่อาจไม่มีค่า ส่งเป็น "" แทน null (ACC-32)
// รูปแบบเดียวกับที่ encoding/json ใช้กับ time.Time
func OptionalTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}

// OptionalString — ข้อความที่อาจไม่มีค่า ส่งเป็น "" แทน null (ACC-32)
func OptionalString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// IsAPIKeyOwner — เจ้าของ Key ของ 1.3 = Company Seamless 1 to 1 · Share Master · Share Reseller (ACC-01)
func IsAPIKeyOwner(t agentManagementCore.UserType) bool {
	switch t {
	case agentManagementCore.UserTypeCompanySeamless1to1, agentManagementCore.UserTypeShareMaster, agentManagementCore.UserTypeShareReseller:
		return true
	}
	return false
}

// SuspendedPermissions — สิทธิ์ที่ใช้ได้จริงตอน status = SUSPENDED (ACC-12 · AUTH-54): เหลือ report ไม่เกิน view · เมนูอื่น off
// ใช้แสดงใน Profile · การกันจริงอยู่ที่ middleware PassedGates
func SuspendedPermissions(perms map[agentManagementCore.Menu]agentManagementCore.Level) map[agentManagementCore.Menu]agentManagementCore.Level {
	out := make(map[agentManagementCore.Menu]agentManagementCore.Level, len(perms))
	for m, l := range perms {
		out[m] = agentManagementCore.LevelOff
		if m == agentManagementCore.MenuReport && l.Allows(agentManagementCore.LevelView) {
			out[m] = agentManagementCore.LevelView
		}
	}
	return out
}
