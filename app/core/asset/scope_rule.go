package asset

import "app/app/models"

// Scope — เจ้าของค่า: ค่ากลางของระบบ หรือค่าที่ตั้งไว้ที่บัญชีใดบัญชีหนึ่ง (AST-01)
// ใช้ type เดียวกับ models.AssetScope เพื่อไม่ต้องแปลงระหว่าง layer
type Scope = models.AssetScope

const (
	ScopeGlobal = models.AssetScopeGlobal
	ScopeAgent  = models.AssetScopeAgent
)

// Target — เป้าหมายของคำขอ (AST-08) · Resolved = ผ่าน RequireAssetTarget แล้ว (เติมบัญชีจาก token + สาย — AST-09)
// ค่า zero (Resolved = false) ต้องถูกปฏิเสธเสมอ — ห้ามตีความเป็น GLOBAL
// BO ระบุบัญชีอื่นไม่ได้ (lead 2026-10-16): AGENT = บัญชีที่ login เสมอ (sub = เจ้าของ)
type Target struct {
	Resolved bool
	Scope    Scope
	AgentID  uint // บัญชีที่ login เมื่อ scope = AGENT · 0 เมื่อ scope = GLOBAL
	// Chain — สายของเป้าหมาย AGENT (ตัวเอง → ชั้นบนจากใกล้ → ไกล) ที่โหลดไว้ตอนตรวจสิทธิ์
	// ให้ service ไล่ค่าต่อโดยไม่อ่าน user_agents ซ้ำ · nil = ยังไม่โหลด (GLOBAL / เรียกจากสคริปต์)
	Chain []uint
}

// Outcome — ผลตัดสินสิทธิ์ต่อ scope
type Outcome int

const (
	OutcomeAllow     Outcome = iota
	OutcomeForbidden         // ไม่มีสิทธิ์แก้ → 405301 (AST-02, AST-03)
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

// Authorize — สิทธิ์ต่อ scope (AST-02 – AST-05) · ผู้เรียก = บัญชีหลักที่ request ทำงานในนาม (sub = เจ้าของ)
// เป้าหมาย AGENT คือผู้เรียกเองเสมอ (ไม่มีการระบุบัญชีอื่น — lead 2026-10-16) · สิทธิ์เมนู asset (view / edit) ตรวจก่อนหน้านี้แล้ว
//
//	GLOBAL: อ่านได้ · เขียนเฉพาะ SUPERADMIN
//	AGENT : อ่านได้ · เขียนได้ถ้า CanOwnLook (Share Master / Superadmin / ADMIN ไม่ได้)
func Authorize(scope Scope, write bool, role models.AgentRole, agentType *models.AgentType) Outcome {
	switch scope {
	case ScopeGlobal:
		if !write || role == models.AgentRoleSuperAdmin {
			return OutcomeAllow
		}
		return OutcomeForbidden
	case ScopeAgent:
		if !write || CanOwnLook(role, agentType) {
			return OutcomeAllow
		}
		return OutcomeForbidden
	}
	return OutcomeForbidden // scope ที่ไม่รู้จัก = ปฏิเสธ (fail-closed — AST-09)
}
