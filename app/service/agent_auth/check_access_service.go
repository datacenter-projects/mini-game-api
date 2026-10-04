package agentauth

import (
	agentAuthCore "app/app/core/agent_auth"
	"app/app/models"
	"app/pkg/apperr"
)

// CheckGateService — ด่านหลัง login (AUTH-29) ใช้โดย middleware PassedGates
// allow = ด่านที่ route นี้เป็นทางผ่าน (เช่น route เปลี่ยนรหัสผ่านเรียกได้ระหว่างติดด่านเปลี่ยนรหัสผ่าน)
func CheckGateService(actor Actor, allow ...agentAuthCore.Gate) error {
	gate := actor.Gate()
	if gate == agentAuthCore.GateNone {
		return nil
	}
	for _, g := range allow {
		if g == gate {
			return nil
		}
	}
	switch gate {
	case agentAuthCore.GateChangePassword:
		return apperr.ErrMustChangePassword
	case agentAuthCore.GateChangePasscode:
		return apperr.ErrMustChangePasscode
	default:
		return apperr.ErrPasscodeNotSet
	}
}

// CheckRoleService — route ที่จำกัด role ของบัญชี agent (sub ไม่ผ่านเสมอ เพราะ Role ของ sub คือของผู้สร้าง)
// รอบนี้ใช้กับ ADMIN (AUTH-44) — role/permission guard เต็มรูปแบบอยู่ในงานอื่น
func CheckRoleService(actor Actor, role models.AgentRole) error {
	if actor.AccountType == models.AccountTypeAgent && actor.Role == role {
		return nil
	}
	if role == models.AgentRoleAdmin {
		return apperr.ErrAdminOnly
	}
	return apperr.ErrForbidden
}
