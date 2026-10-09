package account

import (
	"testing"
	"time"

	agentManagementCore "app/app/core/agent_management"
)

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

func TestIsAPIKeyOwner(t *testing.T) { // ACC-01
	tests := []struct {
		in   agentManagementCore.UserType
		want bool
	}{
		{agentManagementCore.UserTypeCompanySeamless1to1, true},
		{agentManagementCore.UserTypeShareMaster, true},
		{agentManagementCore.UserTypeShareReseller, true},
		{agentManagementCore.UserTypeCompanySeamlessReseller, true},
		{agentManagementCore.UserTypeCompanySeamlessMaster, true},
		{agentManagementCore.UserTypeCompanyTransfer, true},
		{agentManagementCore.UserTypeShareB2B, true},
		{agentManagementCore.UserTypeShareB2C, true},
		{agentManagementCore.UserTypeAgent, true},
		{agentManagementCore.UserTypeSuperadmin, false},
		{agentManagementCore.UserTypeAdmin, false},
		{agentManagementCore.UserTypeMember, false},
	}
	for _, tt := range tests {
		if got := IsAPIKeyOwner(tt.in); got != tt.want {
			t.Errorf("IsAPIKeyOwner(%s) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestSuspendedPermissions(t *testing.T) { // ACC-12 · AUTH-54
	full := agentManagementCore.FullPermissions("COMPANY")
	got := SuspendedPermissions(full)
	if len(got) != len(full) || got[agentManagementCore.MenuReport] != agentManagementCore.LevelView {
		t.Fatalf("full → %v", got)
	}
	for m, l := range got {
		if m != agentManagementCore.MenuReport && l != agentManagementCore.LevelOff {
			t.Fatalf("%s = %s, want off", m, l)
		}
	}
	noReport := map[agentManagementCore.Menu]agentManagementCore.Level{
		agentManagementCore.MenuReport: agentManagementCore.LevelOff, agentManagementCore.MenuMember: agentManagementCore.LevelEdit}
	if got := SuspendedPermissions(noReport); got[agentManagementCore.MenuReport] != agentManagementCore.LevelOff || got[agentManagementCore.MenuMember] != agentManagementCore.LevelOff {
		t.Fatalf("no report → %v", got)
	}
	if got := SuspendedPermissions(map[agentManagementCore.Menu]agentManagementCore.Level{}); len(got) != 0 {
		t.Fatal("ADMIN ต้องได้ {}")
	}
}
