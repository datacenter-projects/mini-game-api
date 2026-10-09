package agentauth

import (
	agentAuthCore "app/app/core/agent_auth"
	"app/app/models"
	"app/pkg/apperr"
)

// CheckNotSuspendedService — AUTH-54: บัญชีที่ EffectiveStatus = SUSPENDED (ตัวเอง ผู้สร้าง หรือ upline ถูกระงับ)
// ใช้ได้เฉพาะ route ที่ประกาศ PassedGatesAllowSuspended · LOCKED ถูกปฏิเสธตั้งแต่ Authenticated แล้ว (AUTH-27)
func CheckNotSuspendedService(actor Actor) error {
	if actor.EffectiveStatus == models.AgentStatusSuspended {
		return apperr.ErrAccountSuspended
	}
	return nil
}

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
