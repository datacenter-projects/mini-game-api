// Package account คือกฎของหน้าบัญชีหลังบ้านแบบ pure function
// spec: docs/modules/account.md
package account

import (
	"time"

	"app/app/models"
)

// UserTypeFromRole — ประเภทบัญชี (ACC-12) ที่บอกได้จาก role อย่างเดียว
// Company / Share ต้องใช้ประเภทย่อย (agent_type) ของ module ② ซึ่งยังไม่มี → ส่ง "" ไปก่อน (ACC-32)
func UserTypeFromRole(role models.AgentRole) string {
	switch role {
	case models.AgentRoleSuperAdmin, models.AgentRoleAdmin, models.AgentRoleAgent:
		return string(role)
	default:
		return ""
	}
}

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
