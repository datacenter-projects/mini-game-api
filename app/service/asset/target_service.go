package asset

import (
	"context"
	"errors"

	assetCore "app/app/core/asset"
	agentManagementPostgres "app/app/repository/postgres/agent_management"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
	"app/platform/database"
)

// CheckAssetTargetService — เติมเป้าหมายจาก token แล้วตรวจสิทธิ์ต่อ scope (AST-02 – AST-05) · ใช้โดย middleware RequireAssetTarget
// ผู้เรียก = บัญชีหลักที่ request ทำงานในนาม (sub = เจ้าของ) · สิทธิ์เมนู asset (view / edit) ตรวจก่อนหน้านี้แล้ว
// BO ระบุบัญชีอื่นไม่ได้ (lead 2026-10-16): AGENT = บัญชีที่ login เสมอ
//
//	GLOBAL: อ่านได้ · เขียนเฉพาะ SUPERADMIN → ไม่ใช่ = 405301
//	AGENT : อ่านได้ · เขียนได้ถ้าเป็นเจ้าของหน้าตา (Share Master / Superadmin = 405301)
//
// ผ่านแล้วคืนเป้าหมายที่ Resolved + AgentID + Chain (สายของบัญชีจาก cnf) ให้ service ไล่ค่าต่อโดยไม่อ่านซ้ำ
// service ปลายทางต้องเช็คซ้ำว่า resource (config) เป็นของเป้าหมาย — middleware ไม่ได้พิสูจน์ข้อนี้ (AST-07)
func CheckAssetTargetService(ctx context.Context, actor agentAuthService.Actor, t assetCore.Target, write bool) (assetCore.Target, error) {
	if t.Scope == "" { // ไม่ผ่าน ResolveAssetTarget (bug ของ route) — fail-closed (AST-09)
		return assetCore.Target{}, errTargetNotResolved
	}
	db := database.DBConn.WithContext(ctx)
	owner, err := loadOwner(db, actor)
	if err != nil {
		return assetCore.Target{}, err
	}
	if assetCore.Authorize(t.Scope, write, owner.Role, owner.AgentType) != assetCore.OutcomeAllow {
		return assetCore.Target{}, apperr.ErrAssetNoPermission
	}

	out := assetCore.Target{Resolved: true, Scope: t.Scope}
	if t.Scope == assetCore.ScopeAgent {
		cnf, err := agentManagementPostgres.GetAgentChainRepository(db, owner.ID)
		if errors.Is(err, apperr.ErrNotFound) { // บัญชีหายไประหว่าง request
			return assetCore.Target{}, apperr.ErrSessionEnded
		}
		if err != nil {
			return assetCore.Target{}, err
		}
		parents, err := parentIDs(cnf)
		if err != nil {
			return assetCore.Target{}, err
		}
		out.AgentID = owner.ID
		out.Chain = assetCore.ChainIDs(owner.ID, parents)
	}
	return out, nil
}
