package agentmanagement

import (
	"testing"

	"app/app/models"
)

func TestLevelAllows(t *testing.T) { // MGMT-50
	tests := []struct {
		have, need Level
		want       bool
	}{
		{LevelEdit, LevelView, true},
		{LevelEdit, LevelEdit, true},
		{LevelView, LevelView, true},
		{LevelView, LevelEdit, false},
		{LevelOff, LevelView, false},
		{"", LevelView, false},
	}
	for _, tt := range tests {
		if got := tt.have.Allows(tt.need); got != tt.want {
			t.Errorf("%q.Allows(%q) = %v, want %v", tt.have, tt.need, got, tt.want)
		}
	}
}

func TestMenusForRole(t *testing.T) { // MGMT-52
	has := func(ms []Menu, m Menu) bool {
		for _, x := range ms {
			if x == m {
				return true
			}
		}
		return false
	}
	sa := MenusForRole(models.AgentRoleSuperAdmin)
	if len(sa) != 8 || !has(sa, MenuRate) || has(sa, MenuAnnouncement) || has(sa, MenuAPICredential) { // Superadmin ไม่มี Key (account ACC-01)
		t.Errorf("Superadmin menus = %v", sa)
	}
	for _, r := range []models.AgentRole{models.AgentRoleCompany, models.AgentRoleShareholder, models.AgentRoleAgent} {
		ms := MenusForRole(r)
		if len(ms) != 9 || has(ms, MenuRate) || !has(ms, MenuAnnouncement) || !has(ms, MenuAPICredential) { // api_credential — lead A6
			t.Errorf("%s menus = %v", r, ms)
		}
	}
	if MenusForRole(models.AgentRoleAdmin) != nil {
		t.Error("ADMIN should have no menus")
	}
}

func TestFullPermissions(t *testing.T) { // MGMT-53
	p := FullPermissions(models.AgentRoleCompany)
	if len(p) != 9 || p[MenuAPICredential] != LevelEdit || p[MenuMember] != LevelEdit || p[MenuDashboard] != LevelEdit || p[MenuReport] != LevelEdit {
		t.Errorf("FullPermissions(COMPANY) = %v", p)
	}
}

func TestNormalizeSubPermissions(t *testing.T) { // MGMT-50, MGMT-52
	tests := []struct {
		name    string
		role    models.AgentRole
		in      map[string]string
		ok      bool
		badMenu string
		reason  string
	}{
		{"ส่งแค่ member", models.AgentRoleCompany, map[string]string{"member": "edit"}, true, "", ""},
		{"ไม่ส่งเลย", models.AgentRoleCompany, nil, true, "", ""},
		{"VIEW ตัวใหญ่", models.AgentRoleCompany, map[string]string{"member": "VIEW"}, false, "member", "level"},
		{"none", models.AgentRoleCompany, map[string]string{"member": "none"}, false, "member", "level"},
		{"dashboard edit", models.AgentRoleCompany, map[string]string{"dashboard": "edit"}, false, "dashboard", "level"},
		{"sub ของ Superadmin ได้ announcement", models.AgentRoleSuperAdmin, map[string]string{"announcement": "view"}, false, "announcement", "menu"},
		{"Superadmin ให้ rate", models.AgentRoleSuperAdmin, map[string]string{"rate": "edit"}, true, "", ""},
		{"Company ให้ rate", models.AgentRoleCompany, map[string]string{"rate": "view"}, false, "rate", "menu"},
		{"เมนูไม่รู้จัก", models.AgentRoleAgent, map[string]string{"foo": "view"}, false, "foo", "menu"},
		{"Company ให้ api_credential edit", models.AgentRoleCompany, map[string]string{"api_credential": "edit"}, true, "", ""},
		{"sub ของ Superadmin ได้ api_credential", models.AgentRoleSuperAdmin, map[string]string{"api_credential": "view"}, false, "api_credential", "menu"},
	}
	for _, tt := range tests {
		out, v, ok := NormalizeSubPermissions(tt.role, tt.in)
		if ok != tt.ok {
			t.Errorf("%s: ok = %v, want %v", tt.name, ok, tt.ok)
			continue
		}
		if !ok {
			if v.Menu != tt.badMenu || v.Reason != tt.reason {
				t.Errorf("%s: violation = %+v", tt.name, v)
			}
			continue
		}
		if len(out) != len(MenusForRole(tt.role)) {
			t.Errorf("%s: %d menus, want %d", tt.name, len(out), len(MenusForRole(tt.role)))
		}
		for m, l := range out {
			want := LevelOff
			if s, ok := tt.in[string(m)]; ok {
				want = Level(s)
			}
			if l != want {
				t.Errorf("%s: %s = %q, want %q", tt.name, m, l, want)
			}
		}
	}
}

func TestSubPermissionsView(t *testing.T) { // MGMT-50
	got := SubPermissionsView(models.AgentRoleCompany, map[string]string{"member": "view", "pt": "bogus", "rate": "edit", "report": "edit", "account": "edit"})
	if len(got) != 9 || got[MenuAPICredential] != LevelOff {
		t.Fatalf("len = %d (want 9) api_credential = %q (want off — sub เดิมไม่มี key นี้)", len(got), got[MenuAPICredential])
	}
	if got[MenuMember] != LevelView || got[MenuPT] != LevelOff || got[MenuReport] != LevelOff {
		t.Errorf("SubPermissionsView = %v", got)
	}
	if _, ok := got[MenuRate]; ok {
		t.Error("Company sub should not have rate")
	}
	if _, ok := got["account"]; ok {
		t.Error("ไม่มีเมนู account แล้ว — ค่าเก่าที่เก็บไว้ต้องไม่แสดง")
	}
}
