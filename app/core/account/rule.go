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

// IsAPIKeyOwner — บัญชีที่มี Key ของ 1.3 = Company / Share / Agent ทุกประเภท (ACC-01 แก้ 2026-10-08) · SUPERADMIN / ADMIN / Member ไม่มี
func IsAPIKeyOwner(t agentManagementCore.UserType) bool {
	switch t {
	case agentManagementCore.UserTypeSuperadmin, agentManagementCore.UserTypeAdmin, agentManagementCore.UserTypeMember, "":
		return false
	}
	return true
}

// SuspendedPermissions — สิทธิ์ที่ใช้ได้จริงตอน status = SUSPENDED (ACC-12 · ACC-31 · AUTH-54): เหลือ report และ api_credential ไม่เกิน view · เมนูอื่น off
// ใช้แสดงใน Profile · การกันจริงอยู่ที่ middleware PassedGates
func SuspendedPermissions(perms map[agentManagementCore.Menu]agentManagementCore.Level) map[agentManagementCore.Menu]agentManagementCore.Level {
	out := make(map[agentManagementCore.Menu]agentManagementCore.Level, len(perms))
	for m, l := range perms {
		out[m] = agentManagementCore.LevelOff
		if (m == agentManagementCore.MenuReport || m == agentManagementCore.MenuAPICredential) && l.Allows(agentManagementCore.LevelView) {
			out[m] = agentManagementCore.LevelView
		}
	}
	return out
}
