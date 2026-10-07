package account

import (
	"testing"
	"time"

	"app/app/models"
)

func TestUserTypeFromRole(t *testing.T) { // ACC-12
	tests := []struct {
		role models.AgentRole
		want string
	}{
		{models.AgentRoleSuperAdmin, "SUPERADMIN"},
		{models.AgentRoleAdmin, "ADMIN"},
		{models.AgentRoleAgent, "AGENT"},
		{models.AgentRoleCompany, ""},     // ต้องใช้ agent_type ของ module ②
		{models.AgentRoleShareholder, ""}, // ต้องใช้ agent_type ของ module ②
		{models.AgentRole("UNKNOWN"), ""},
	}
	for _, tt := range tests {
		if got := UserTypeFromRole(tt.role); got != tt.want {
			t.Errorf("UserTypeFromRole(%q) = %q, want %q", tt.role, got, tt.want)
		}
	}
}

func TestOptionalTime(t *testing.T) { // ACC-32
	at := time.Date(2026, 10, 5, 10, 0, 0, 0, time.FixedZone("ICT", 7*3600))
	tests := []struct {
		name string
		in   *time.Time
		want string
	}{
		{"ไม่มีค่า", nil, ""},
		{"มีค่า", &at, "2026-10-05T10:00:00+07:00"},
	}
	for _, tt := range tests {
		if got := OptionalTime(tt.in); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestOptionalString(t *testing.T) { // ACC-32
	ip := "203.0.113.10"
	tests := []struct {
		name string
		in   *string
		want string
	}{
		{"ไม่มีค่า", nil, ""},
		{"มีค่า", &ip, "203.0.113.10"},
	}
	for _, tt := range tests {
		if got := OptionalString(tt.in); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}
