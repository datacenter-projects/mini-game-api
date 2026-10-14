package asset

import "app/app/models"

// Scope — เจ้าของค่า: ค่ากลางของระบบ หรือค่าที่ตั้งไว้ที่บัญชีใดบัญชีหนึ่ง (AST-01)
// ใช้ type เดียวกับ models.AssetScope เพื่อไม่ต้องแปลงระหว่าง layer
type Scope = models.AssetScope

const (
	ScopeGlobal = models.AssetScopeGlobal
	ScopeAgent  = models.AssetScopeAgent
)

// Target — เป้าหมายของคำขอ (AST-08) · Resolved = middleware ResolveAssetTarget ทำงานแล้ว (AST-09)
// ค่า zero (Resolved = false) ต้องถูกปฏิเสธเสมอ — ห้ามตีความเป็น GLOBAL
type Target struct {
	Resolved bool
	Scope    Scope
	AgentID  uint // 0 เมื่อ scope = GLOBAL
	// Chain — สายของเป้าหมาย AGENT (ตัวเอง → ชั้นบนจากใกล้ → ไกล) ที่โหลดไว้ตอนตรวจสิทธิ์ต่อเป้าหมาย
	// ให้ service ไล่ค่าต่อโดยไม่อ่าน user_agents ซ้ำ · nil = ยังไม่โหลด (GLOBAL / เรียกจากสคริปต์)
	Chain []uint
}

// Relation — เป้าหมาย AGENT เทียบกับผู้เรียก (บัญชีหลักที่ request ทำงานในนาม)
type Relation int

const (
	RelationOutside  Relation = iota // นอกสาย
	RelationSelf                     // ตัวเอง
	RelationDownline                 // บัญชีในสายล่าง
)

// RelationOf — เป้าหมาย AGENT เทียบกับผู้เรียก · parents = ชั้นบนของเป้าหมายจาก cnf (MGMT-61)
// ผู้เรียกอยู่ใน parents ของเป้าหมาย = เป้าหมายอยู่สายล่าง (กฎเดียวกับ cnf @> parent ของ agent_management)
func RelationOf(callerID, targetID uint, parents []uint) Relation {
	if targetID == callerID {
		return RelationSelf
	}
	for _, id := range parents {
		if id == callerID {
			return RelationDownline
		}
	}
	return RelationOutside
}

// Outcome — ผลตัดสินสิทธิ์ต่อเป้าหมาย
type Outcome int

const (
	OutcomeAllow     Outcome = iota
	OutcomeOutside           // นอกสาย → 402402 (AST-04, AST-05)
	OutcomeForbidden         // ไม่มีสิทธิ์แก้ → 405301 (AST-02, AST-03, AST-04)
)

// IsShareMaster — Share Master ตั้งเองไม่ได้ ใช้ของ CSM (AST-02 · lead AV1)
func IsShareMaster(role models.AgentRole, agentType *models.AgentType) bool {
	return role == models.AgentRoleShareholder && agentType != nil && *agentType == models.AgentTypeShareMaster
}

// CanOwnLook — บัญชีหลักที่ตั้งหน้าตาของตัวเองได้ (AST-02): Company ทุกประเภท · Share (ยกเว้น Master) · Agent
func CanOwnLook(role models.AgentRole, agentType *models.AgentType) bool {
	switch role {
	case models.AgentRoleCompany, models.AgentRoleAgent:
		return true
	case models.AgentRoleShareholder:
		return !IsShareMaster(role, agentType)
	}
	return false
}

// Authorize — สิทธิ์ต่อเป้าหมาย (AST-03 / AST-04 / AST-05) · ผู้เรียก = บัญชีหลักที่ request ทำงานในนาม (sub = เจ้าของ)
// สิทธิ์เมนู asset (view / edit) ตรวจก่อนหน้านี้แล้วที่ RequirePermission
//
//	GLOBAL: อ่านได้ · เขียนเฉพาะ SUPERADMIN
//	AGENT : ตัวเอง อ่านได้ · เขียนได้ถ้า CanOwnLook · สายล่าง อ่านได้ แก้ไม่ได้ · นอกสาย = Outside
func Authorize(scope Scope, write bool, role models.AgentRole, agentType *models.AgentType, rel Relation) Outcome {
	switch scope {
	case ScopeGlobal:
		if !write || role == models.AgentRoleSuperAdmin {
			return OutcomeAllow
		}
		return OutcomeForbidden
	case ScopeAgent:
		switch rel {
		case RelationSelf:
			if !write || CanOwnLook(role, agentType) {
				return OutcomeAllow
			}
			return OutcomeForbidden
		case RelationDownline:
			if !write {
				return OutcomeAllow
			}
			return OutcomeForbidden
		}
		return OutcomeOutside
	}
	return OutcomeForbidden // scope ที่ไม่รู้จัก = ปฏิเสธ (fail-closed — AST-09)
}
