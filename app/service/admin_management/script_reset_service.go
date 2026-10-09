package adminmanagement

import (
	"context"

	"app/app/models"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
)

// ScriptResetRequest — สิ่งที่ scripts/reset_credentials ส่งมา (AUTH-51)
type ScriptResetRequest struct {
	Username string
	Password bool   // รีเซ็ตรหัสผ่าน
	Passcode bool   // รีเซ็ต passcode
	Operator string // ชื่อผู้รัน script (user ของเครื่อง) — เก็บเป็น actor_username
}

// ScriptResetCredentialsService — รีเซ็ตรหัสผ่าน / passcode ของ SUPERADMIN หรือ ADMIN จาก script บนเซิร์ฟเวอร์ (AUTH-51)
// กติกาเดียวกับ admin reset (AUTH-46 – AUTH-48) · audit actor = SCRIPT · รีเซ็ตทั้งสองอย่างได้ใน transaction เดียว
// กู้บัญชีได้แม้ถูกล็อก (ไม่มีใครอยู่เหนือ SUPERADMIN / ADMIN ที่จะปลดล็อกให้)
func ScriptResetCredentialsService(ctx context.Context, req ScriptResetRequest) (agentAuthService.ResetResult, error) {
	if !req.Password && !req.Passcode {
		return agentAuthService.ResetResult{}, apperr.ErrValidation.WithMessage("ต้องเลือกรีเซ็ตรหัสผ่านหรือ passcode อย่างน้อย 1 อย่าง",
			"choose to reset the password, the passcode, or both")
	}
	t, err := agentAuthService.FindResetTargetService(ctx, req.Username)
	if err != nil {
		return agentAuthService.ResetResult{}, err
	}
	if t.IsSub() || (t.Role() != models.AgentRoleSuperAdmin && t.Role() != models.AgentRoleAdmin) {
		return agentAuthService.ResetResult{}, apperr.ErrResetNotAllowed // บัญชีอื่นใช้ API admin reset
	}
	return agentAuthService.ResetCredentialsService(ctx, t, agentAuthService.ResetOptions{Password: req.Password, Passcode: req.Passcode},
		agentAuthService.ResetBy{ScriptOperator: req.Operator, Meta: agentAuthService.RequestMeta{UserAgent: "scripts/reset_credentials"}})
}
