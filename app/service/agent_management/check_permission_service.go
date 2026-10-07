// Package agentmanagement คือ business logic ของการจัดการสมาชิก — spec: docs/modules/agent_management.md
package agentmanagement

import (
	"context"
	"encoding/json"

	agentManagementCore "app/app/core/agent_management"
	"app/app/models"
	"app/app/repository/postgres"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
	"app/platform/database"
)

// PermissionsOf — สิทธิ์ของผู้เรียก (MGMT-50, MGMT-52) · บัญชีหลัก = ระดับสูงสุดทุกเมนูของ role · sub = ตามที่เจ้าของให้
func PermissionsOf(ctx context.Context, actor agentAuthService.Actor) (map[agentManagementCore.Menu]agentManagementCore.Level, error) {
	if actor.AccountType != models.AccountTypeSub {
		return agentManagementCore.FullPermissions(actor.Role), nil
	}
	raw, err := postgres.GetSubaccountPermissionsRepository(database.DBConn.WithContext(ctx), actor.SubaccountID)
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
