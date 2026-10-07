package agentmanagement

import (
	"testing"

	"app/app/models"
)

func TestUserTypeOf(t *testing.T) { // MGMT-01
	at := func(a models.AgentType) *models.AgentType { return &a }
	tests := []struct {
		role models.AgentRole
		typ  *models.AgentType
		want UserType
	}{
		{models.AgentRoleSuperAdmin, nil, UserTypeSuperadmin},
		{models.AgentRoleAdmin, nil, UserTypeAdmin},
		{models.AgentRoleAgent, nil, UserTypeAgent},
		{models.AgentRoleCompany, at(models.AgentTypeTransfer), UserTypeCompanyTransfer},
		{models.AgentRoleCompany, at(models.AgentTypeSeamless1to1), UserTypeCompanySeamless1to1},
		{models.AgentRoleShareholder, at(models.AgentTypeShareReseller), UserTypeShareReseller},
		{models.AgentRoleShareholder, at(models.AgentTypeShareB2B), UserTypeShareB2B},
		{models.AgentRoleCompany, nil, ""}, // บัญชีเก่าก่อน migration
	}
	for _, tt := range tests {
		if got := UserTypeOf(tt.role, tt.typ); got != tt.want {
			t.Errorf("UserTypeOf(%s) = %q, want %q", tt.role, got, tt.want)
		}
	}
}

func TestResolveNewAgent(t *testing.T) { // MGMT-02
	tests := []struct {
		creator, requested UserType
		ok                 bool
		role               models.AgentRole
		stored             UserType
	}{
		{UserTypeSuperadmin, UserTypeCompanyTransfer, true, models.AgentRoleCompany, UserTypeCompanyTransfer},
		{UserTypeSuperadmin, UserTypeCompanySeamlessMaster, true, models.AgentRoleCompany, UserTypeCompanySeamlessMaster},
		{UserTypeSuperadmin, UserTypeShareB2B, false, "", ""},
		{UserTypeCompanyTransfer, UserTypeShareB2B, true, models.AgentRoleShareholder, UserTypeShareB2B},
		{UserTypeCompanyTransfer, UserTypeShareB2C, true, models.AgentRoleShareholder, UserTypeShareB2C},
		{UserTypeCompanyTransfer, UserTypeAgent, false, "", ""},
		{UserTypeCompanySeamlessReseller, UserTypeShareB2C, true, models.AgentRoleShareholder, UserTypeShareReseller},
		{UserTypeCompanySeamlessMaster, UserTypeShareB2C, true, models.AgentRoleShareholder, UserTypeShareMaster},
		{UserTypeCompanySeamlessReseller, UserTypeShareB2B, false, "", ""},
		{UserTypeCompanySeamlessReseller, UserTypeShareReseller, false, "", ""}, // ต้องส่ง SHARE_B2C
		{UserTypeCompanySeamless1to1, UserTypeShareB2C, false, "", ""},
		{UserTypeShareB2B, UserTypeAgent, true, models.AgentRoleAgent, UserTypeAgent},
		{UserTypeShareB2C, UserTypeAgent, true, models.AgentRoleAgent, UserTypeAgent},
		{UserTypeShareMaster, UserTypeAgent, true, models.AgentRoleAgent, UserTypeAgent},
		{UserTypeAgent, UserTypeAgent, true, models.AgentRoleAgent, UserTypeAgent},
		{UserTypeAgent, UserTypeShareB2C, false, "", ""},
		{UserTypeAdmin, UserTypeCompanyTransfer, false, "", ""}, // AUTH-43
	}
	for _, tt := range tests {
		got, ok := ResolveNewAgent(tt.creator, tt.requested)
		if ok != tt.ok || got.Role != tt.role || got.UserType != tt.stored {
			t.Errorf("ResolveNewAgent(%s, %s) = %+v, %v", tt.creator, tt.requested, got, ok)
		}
		if ok && got.Role != models.AgentRoleAgent && got.AgentType == nil {
			t.Errorf("ResolveNewAgent(%s, %s): agent_type nil", tt.creator, tt.requested)
		}
		if ok && got.Role == models.AgentRoleAgent && got.AgentType != nil {
			t.Errorf("ResolveNewAgent(%s, %s): Agent must not have agent_type", tt.creator, tt.requested)
		}
	}
}

func TestCanCreateMember(t *testing.T) { // MGMT-02
	tests := []struct {
		creator UserType
		want    bool
	}{
		{UserTypeSuperadmin, false},
		{UserTypeAdmin, false},
		{UserTypeCompanyTransfer, false},
		{UserTypeCompanySeamlessReseller, false},
		{UserTypeCompanySeamless1to1, true},
		{UserTypeShareB2B, false},
		{UserTypeShareB2C, true},
		{UserTypeShareReseller, true},
		{UserTypeShareMaster, true},
		{UserTypeAgent, true},
	}
	for _, tt := range tests {
		if got := CanCreateMember(tt.creator); got != tt.want {
			t.Errorf("CanCreateMember(%s) = %v, want %v", tt.creator, got, tt.want)
		}
	}
}

func TestIsSeamless(t *testing.T) { // MGMT-15
	tests := []struct {
		in   UserType
		want bool
	}{
		{UserTypeCompanyTransfer, false},
		{UserTypeCompanySeamlessReseller, true},
		{UserTypeCompanySeamlessMaster, true},
		{UserTypeCompanySeamless1to1, true},
		{UserTypeSuperadmin, false},
	}
	for _, tt := range tests {
		if got := IsSeamless(tt.in); got != tt.want {
			t.Errorf("IsSeamless(%s) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestUsernameRules(t *testing.T) { // MGMT-05
	tests := []struct {
		in   string
		norm string
		ok   bool
	}{
		{"Share01", "share01", true},
		{"  abc ", "abc", true},
		{"ab", "ab", false},
		{"abcdefghijabcdefghijabcdefghijab", "abcdefghijabcdefghijabcdefghijab", true}, // 32
		{"abcdefghijabcdefghijabcdefghijabc", "abcdefghijabcdefghijabcdefghijabc", false},
		{"share_01", "share_01", false},
		{"ชื่อไทย", "ชื่อไทย", false},
	}
	for _, tt := range tests {
		n := NormalizeUsername(tt.in)
		if n != tt.norm || IsValidUsername(n) != tt.ok {
			t.Errorf("username %q → %q valid=%v, want %q valid=%v", tt.in, n, IsValidUsername(n), tt.norm, tt.ok)
		}
	}
}

func TestIsValidName(t *testing.T) { // MGMT-07, MGMT-41
	tests := []struct {
		in   string
		want bool
	}{
		{"Somchai", true},
		{"ab", false},
		{"Som chai", false},
		{"", false},
		{"ABCDEFGHIJabcdefghij0123456789AB", true},
	}
	for _, tt := range tests {
		if got := IsValidName(tt.in); got != tt.want {
			t.Errorf("IsValidName(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestIsValidPhone(t *testing.T) { // MGMT-08
	tests := []struct {
		in   string
		want bool
	}{
		{"", true},
		{"0812345678", true},
		{"12345678", true},
		{"1234567", false},
		{"1234567890123456", false},
		{"+66812345678", false},
		{"081-234-5678", false},
	}
	for _, tt := range tests {
		if got := IsValidPhone(tt.in); got != tt.want {
			t.Errorf("IsValidPhone(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestIsValidSubName(t *testing.T) { // MGMT-41
	tests := []struct {
		in   string
		want bool
	}{
		{NormalizeUsername("Staff"), true},
		{"ab", false},
		{"abcdefghijabcdefghij", true},
		{"abcdefghijabcdefghijk", false},
		{"st@ff", false},
	}
	for _, tt := range tests {
		if got := IsValidSubName(tt.in); got != tt.want {
			t.Errorf("IsValidSubName(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
