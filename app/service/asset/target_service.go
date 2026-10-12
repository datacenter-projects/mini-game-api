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

// CheckAssetTargetService — สิทธิ์ต่อเป้าหมาย (AST-03 / AST-04 / AST-05) · ใช้โดย middleware RequireAssetTarget
// ผู้เรียก = บัญชีหลักที่ request ทำงานในนาม (sub = เจ้าของ) · สิทธิ์เมนู asset (view / edit) ตรวจก่อนหน้านี้แล้ว
//
//	GLOBAL: อ่านได้ · เขียนเฉพาะ SUPERADMIN → ไม่ใช่ = 405301
//	AGENT : ตัวเอง · สายล่าง (ดูได้ แก้ไม่ได้ = 405301) · นอกสาย = 402402
//
// ผ่านแล้วคืนเป้าหมายที่เติม Chain (สายของเป้าหมาย AGENT จาก cnf ที่อ่านมาตรวจสิทธิ์) ให้ service ไล่ค่าต่อโดยไม่อ่านซ้ำ
// service ปลายทางต้องเช็คซ้ำว่า resource (config) เป็นของเป้าหมาย — middleware ไม่ได้พิสูจน์ข้อนี้ (AST-07)
func CheckAssetTargetService(ctx context.Context, actor agentAuthService.Actor, t assetCore.Target, write bool) (assetCore.Target, error) {
	if !t.Resolved {
		return t, errTargetNotResolved
	}
	db := database.DBConn.WithContext(ctx)
	owner, err := loadOwner(db, actor)
	if err != nil {
		return t, err
	}

	rel := assetCore.RelationOutside
	if t.Scope == assetCore.ScopeAgent {
		cnf, err := agentManagementPostgres.GetAgentChainRepository(db, t.AgentID)
		switch {
		case errors.Is(err, apperr.ErrNotFound): // ไม่มีบัญชีนี้ = นอกสาย (ไม่บอกว่ามีหรือไม่)
		case err != nil:
			return t, err
		default:
			parents, err := parentIDs(cnf)
			if err != nil {
				return t, err
			}
			rel = assetCore.RelationOf(owner.ID, t.AgentID, parents)
			t.Chain = assetCore.ChainIDs(t.AgentID, parents)
		}
	}

	switch assetCore.Authorize(t.Scope, write, owner.Role, owner.AgentType, rel) {
	case assetCore.OutcomeAllow:
		return t, nil
	case assetCore.OutcomeOutside:
		return t, apperr.ErrDownlineNotFound
	}
	return t, apperr.ErrAssetNoPermission
}
