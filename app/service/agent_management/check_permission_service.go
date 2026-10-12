// Package agentmanagement คือ business logic ของการจัดการสมาชิก — spec: docs/modules/agent_management.md
package agentmanagement

import (
	"context"
	"encoding/json"

	agentManagementCore "app/app/core/agent_management"
	"app/app/models"
	agentAuthPostgres "app/app/repository/postgres/agent_auth"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
	"app/platform/database"
)

// PermissionsOf — สิทธิ์ของผู้เรียก (MGMT-50, MGMT-52) · บัญชีหลัก = ระดับสูงสุดทุกเมนูของ role · sub = ตามที่เจ้าของให้
func PermissionsOf(ctx context.Context, actor agentAuthService.Actor) (map[agentManagementCore.Menu]agentManagementCore.Level, error) {
	if actor.AccountType != models.AccountTypeSub {
		return agentManagementCore.FullPermissions(actor.Role), nil
	}
	raw, err := agentAuthPostgres.GetSubaccountPermissionsRepository(database.DBConn.WithContext(ctx), actor.SubaccountID)
	if err != nil {
		return nil, err
	}
	stored := map[string]string{}
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &stored); err != nil {
			return nil, apperr.ErrInternal.Wrap(err)
		}
	}
	return agentManagementCore.SubPermissionsView(actor.Role, stored), nil
}

// CheckPermissionService — ใช้โดย middleware RequirePermission (MGMT-51) · ไม่พอ = 402303
func CheckPermissionService(ctx context.Context, actor agentAuthService.Actor, menu agentManagementCore.Menu, need agentManagementCore.Level) error {
	perms, err := PermissionsOf(ctx, actor)
	if err != nil {
		return err
	}
	if !perms[menu].Allows(need) {
		return apperr.ErrNoMenuPermission
	}
	return nil
}

// CheckSubPermissionService — เช็คสิทธิ์เฉพาะ sub · บัญชีหลักผ่านเสมอ (account ACC-02 · lead A5 / A6 2026-10-09)
// ใช้กับเส้นที่บัญชีหลักทุกประเภทเรียกได้ แต่ service ตัดสินเองว่าประเภทไหนใช้ได้ (เช่น Superadmin ไม่มี Key → 403301)
func CheckSubPermissionService(ctx context.Context, actor agentAuthService.Actor, menu agentManagementCore.Menu, need agentManagementCore.Level) error {
	if actor.AccountType != models.AccountTypeSub {
		return nil
	}
	return CheckPermissionService(ctx, actor, menu, need)
}
