package asset

import (
	"testing"

	"app/app/models"
)

func typ(t models.AgentType) *models.AgentType { return &t }

func TestAuthorize(t *testing.T) { // AST-02 – AST-05 · เป้าหมาย AGENT = ผู้เรียกเองเสมอ (lead 2026-10-16)
	tests := []struct {
		name  string
		scope Scope
		write bool
		role  models.AgentRole
		typ   *models.AgentType
		want  Outcome
	}{
		// GLOBAL
		{"AST-03 superadmin เขียน GLOBAL", ScopeGlobal, true, models.AgentRoleSuperAdmin, nil, OutcomeAllow},
		{"AST-03 share เขียน GLOBAL", ScopeGlobal, true, models.AgentRoleShareholder, typ(models.AgentTypeShareB2C), OutcomeForbidden},
		{"AST-03 company เขียน GLOBAL", ScopeGlobal, true, models.AgentRoleCompany, typ(models.AgentTypeTransfer), OutcomeForbidden},
		{"AST-05 agent อ่าน GLOBAL", ScopeGlobal, false, models.AgentRoleAgent, nil, OutcomeAllow},
		{"AST-05 share master อ่าน GLOBAL", ScopeGlobal, false, models.AgentRoleShareholder, typ(models.AgentTypeShareMaster), OutcomeAllow},

		// AGENT (ของตัวเอง) · เขียน
		{"AST-02 company transfer ตั้งของตัวเอง", ScopeAgent, true, models.AgentRoleCompany, typ(models.AgentTypeTransfer), OutcomeAllow},
		{"AST-02 CSM ตั้งของตัวเอง", ScopeAgent, true, models.AgentRoleCompany, typ(models.AgentTypeSeamlessMaster), OutcomeAllow},
		{"AST-02 company 1to1 ตั้งของตัวเอง", ScopeAgent, true, models.AgentRoleCompany, typ(models.AgentTypeSeamless1to1), OutcomeAllow},
		{"AST-02 share b2b ตั้งของตัวเอง", ScopeAgent, true, models.AgentRoleShareholder, typ(models.AgentTypeShareB2B), OutcomeAllow},
		{"AST-02 share b2c ตั้งของตัวเอง", ScopeAgent, true, models.AgentRoleShareholder, typ(models.AgentTypeShareB2C), OutcomeAllow},
		{"AST-02 share reseller ตั้งของตัวเอง", ScopeAgent, true, models.AgentRoleShareholder, typ(models.AgentTypeShareReseller), OutcomeAllow},
		{"AST-02 agent ตั้งของตัวเอง", ScopeAgent, true, models.AgentRoleAgent, nil, OutcomeAllow},
		{"AST-02 share master ตั้งเองไม่ได้ (ใช้ของ CSM)", ScopeAgent, true, models.AgentRoleShareholder, typ(models.AgentTypeShareMaster), OutcomeForbidden},
		{"AST-02 superadmin ตั้ง AGENT ของตัวเองไม่ได้", ScopeAgent, true, models.AgentRoleSuperAdmin, nil, OutcomeForbidden},
		{"AST-06 admin ไม่ใช่เจ้าของหน้าตา", ScopeAgent, true, models.AgentRoleAdmin, nil, OutcomeForbidden},

		// AGENT (ของตัวเอง) · อ่าน
		{"AST-05 share master อ่านของตัวเอง (ได้ของ CSM)", ScopeAgent, false, models.AgentRoleShareholder, typ(models.AgentTypeShareMaster), OutcomeAllow},
		{"AST-05 agent อ่านของตัวเอง", ScopeAgent, false, models.AgentRoleAgent, nil, OutcomeAllow},

		// fail-closed
		{"AST-09 scope ที่ไม่รู้จัก", Scope(""), false, models.AgentRoleSuperAdmin, nil, OutcomeForbidden},
	}
	for _, tt := range tests {
		if got := Authorize(tt.scope, tt.write, tt.role, tt.typ); got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIsShareMaster(t *testing.T) {
	if !IsShareMaster(models.AgentRoleShareholder, typ(models.AgentTypeShareMaster)) {
		t.Error("share master ต้องเป็น true")
	}
	if IsShareMaster(models.AgentRoleShareholder, typ(models.AgentTypeShareB2C)) || IsShareMaster(models.AgentRoleShareholder, nil) {
		t.Error("share ประเภทอื่นต้องเป็น false")
	}
	if IsShareMaster(models.AgentRoleCompany, typ(models.AgentTypeSeamlessMaster)) {
		t.Error("CSM ไม่ใช่ Share Master")
	}
}

func TestTargetZeroValueIsNotResolved(t *testing.T) { // AST-09: ค่า zero ต้องไม่ถูกตีความเป็น GLOBAL
	var tg Target
	if tg.Resolved || tg.Scope == ScopeGlobal {
		t.Errorf("zero Target ต้องไม่ resolved และไม่ใช่ GLOBAL: %+v", tg)
	}
}
