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

// Currencies — สกุลเงินที่ระบบรองรับ 27 สกุล (ทศนิยม 2 ตำแหน่งทุกสกุล — ACC-18)
// ระหว่างยังไม่มีตารางสกุลของ module ② Profile ส่งครบทุกสกุลไปก่อน
var Currencies = []string{
	"ARS", "AUD", "BDT", "BOB", "BRL", "CLP", "CNY", "EUR", "GBP", "HKD", "IDR", "INR", "JPY", "KHR",
	"KRW", "LAK", "MMK", "MXN", "MYR", "NGN", "PHP", "PKR", "THB", "TWD", "USD", "USDT", "VND",
}

// เมนูสิทธิ์ (module ② MGMT-51, MGMT-52) — ระดับ off / view / edit
const (
	PermissionOff  = "off"
	PermissionView = "view"
	PermissionEdit = "edit"
)

var (
	menusSuperadmin = []string{"dashboard", "account", "member", "pt", "report", "bet_cancel", "payment", "asset", "rate"}
	menusAgentSide  = []string{"dashboard", "account", "member", "pt", "report", "bet_cancel", "payment", "asset", "announcement"}
)

// MenusForRole — เมนูสิทธิ์ของแต่ละประเภทบัญชี (MGMT-52) · ADMIN ไม่มีเมนูสิทธิ์
func MenusForRole(role models.AgentRole) []string {
	switch role {
	case models.AgentRoleSuperAdmin:
		return menusSuperadmin
	case models.AgentRoleCompany, models.AgentRoleShareholder, models.AgentRoleAgent:
		return menusAgentSide
	default:
		return nil
	}
}

// PlaceholderPermissions — ทุกเมนูของประเภทนั้นเป็น off ระหว่างยังไม่มีระบบสิทธิ์ของ module ②
func PlaceholderPermissions(role models.AgentRole) map[string]string {
	out := map[string]string{}
	for _, m := range MenusForRole(role) {
		out[m] = PermissionOff
	}
	return out
}
