package adminmanagement

import (
	"app/app/models"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
)

// CheckAdminService — ใช้โดย middleware admin_management.RequireAdmin (AUTH-44)
// เฉพาะบัญชีหลัก role ADMIN · sub (แม้เจ้าของเป็น ADMIN) และ role อื่น = 401308
func CheckAdminService(actor agentAuthService.Actor) error {
	if actor.AccountType == models.AccountTypeAgent && actor.Role == models.AgentRoleAdmin {
		return nil
	}
	return apperr.ErrAdminOnly
}
