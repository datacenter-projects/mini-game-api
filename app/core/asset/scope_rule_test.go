package asset

import (
	"testing"

	"app/app/models"
)

func typ(t models.AgentType) *models.AgentType { return &t }

func TestAuthorize(t *testing.T) { // AST-02 / AST-03 / AST-04 / AST-05
	tests := []struct {
		name  string
		scope Scope
		write bool
		role  models.AgentRole
		typ   *models.AgentType
		rel   Relation
		want  Outcome
	}{
		// GLOBAL
		{"AST-03 superadmin เขียน GLOBAL", ScopeGlobal, true, models.AgentRoleSuperAdmin, nil, RelationOutside, OutcomeAllow},
		{"AST-03 share เขียน GLOBAL", ScopeGlobal, true, models.AgentRoleShareholder, typ(models.AgentTypeShareB2C), RelationSelf, OutcomeForbidden},
		{"AST-03 company เขียน GLOBAL", ScopeGlobal, true, models.AgentRoleCompany, typ(models.AgentTypeTransfer), RelationSelf, OutcomeForbidden},
		{"AST-05 agent อ่าน GLOBAL", ScopeGlobal, false, models.AgentRoleAgent, nil, RelationSelf, OutcomeAllow},
		{"AST-05 share master อ่าน GLOBAL", ScopeGlobal, false, models.AgentRoleShareholder, typ(models.AgentTypeShareMaster), RelationSelf, OutcomeAllow},

		// AGENT · ตัวเอง · เขียน
		{"AST-02 company transfer ตั้งของตัวเอง", ScopeAgent, true, models.AgentRoleCompany, typ(models.AgentTypeTransfer), RelationSelf, OutcomeAllow},
		{"AST-02 CSM ตั้งของตัวเอง", ScopeAgent, true, models.AgentRoleCompany, typ(models.AgentTypeSeamlessMaster), RelationSelf, OutcomeAllow},
		{"AST-02 company 1to1 ตั้งของตัวเอง", ScopeAgent, true, models.AgentRoleCompany, typ(models.AgentTypeSeamless1to1), RelationSelf, OutcomeAllow},
		{"AST-02 share b2b ตั้งของตัวเอง", ScopeAgent, true, models.AgentRoleShareholder, typ(models.AgentTypeShareB2B), RelationSelf, OutcomeAllow},
		{"AST-02 share b2c ตั้งของตัวเอง", ScopeAgent, true, models.AgentRoleShareholder, typ(models.AgentTypeShareB2C), RelationSelf, OutcomeAllow},
		{"AST-02 share reseller ตั้งของตัวเอง", ScopeAgent, true, models.AgentRoleShareholder, typ(models.AgentTypeShareReseller), RelationSelf, OutcomeAllow},
		{"AST-02 agent ตั้งของตัวเอง", ScopeAgent, true, models.AgentRoleAgent, nil, RelationSelf, OutcomeAllow},
		{"AST-02 share master ตั้งเองไม่ได้ (ใช้ของ CSM)", ScopeAgent, true, models.AgentRoleShareholder, typ(models.AgentTypeShareMaster), RelationSelf, OutcomeForbidden},
		{"AST-02 superadmin ตั้ง AGENT ของตัวเองไม่ได้", ScopeAgent, true, models.AgentRoleSuperAdmin, nil, RelationSelf, OutcomeForbidden},
		{"AST-06 admin ไม่ใช่เจ้าของหน้าตา", ScopeAgent, true, models.AgentRoleAdmin, nil, RelationSelf, OutcomeForbidden},

		// AGENT · ตัวเอง · อ่าน
		{"AST-05 share master อ่านของตัวเอง", ScopeAgent, false, models.AgentRoleShareholder, typ(models.AgentTypeShareMaster), RelationSelf, OutcomeAllow},

		// AGENT · สายล่าง
		{"AST-05 ชั้นบนดูสายล่าง", ScopeAgent, false, models.AgentRoleShareholder, typ(models.AgentTypeShareB2C), RelationDownline, OutcomeAllow},
		{"AST-04 ชั้นบนแก้สายล่างไม่ได้", ScopeAgent, true, models.AgentRoleShareholder, typ(models.AgentTypeShareB2C), RelationDownline, OutcomeForbidden},
		{"AST-04 CSM แก้ Share Master ใต้ตัวเองไม่ได้ (สืบทอดเอง)", ScopeAgent, true, models.AgentRoleCompany, typ(models.AgentTypeSeamlessMaster), RelationDownline, OutcomeForbidden},
		{"AST-05 superadmin ดูสายล่าง", ScopeAgent, false, models.AgentRoleSuperAdmin, nil, RelationDownline, OutcomeAllow},

		// AGENT · นอกสาย
		{"AST-05 อ่านนอกสาย", ScopeAgent, false, models.AgentRoleShareholder, typ(models.AgentTypeShareB2C), RelationOutside, OutcomeOutside},
		{"AST-04 เขียนนอกสาย", ScopeAgent, true, models.AgentRoleShareholder, typ(models.AgentTypeShareB2C), RelationOutside, OutcomeOutside},

		// fail-closed
		{"AST-09 scope ที่ไม่รู้จัก", Scope(""), false, models.AgentRoleSuperAdmin, nil, RelationSelf, OutcomeForbidden},
	}
	for _, tt := range tests {
		if got := Authorize(tt.scope, tt.write, tt.role, tt.typ, tt.rel); got != tt.want {
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
