// Package agentauth คือ business logic ของ backoffice auth
// spec: docs/modules/agent_auth.md, docs/modules/agent_auth_phase2.md
package agentauth

import (
	agentAuthCore "app/app/core/agent_auth"
	"app/app/models"
)

// Actor คือผู้ใช้หลังบ้านที่ยืนยันตัวตนแล้วของ request นี้
// middleware Authenticated เป็นคนสร้าง แล้ว controller อ่านด้วย middleware.GetActor(c)
type Actor struct {
	AccountType models.AccountType
	// AgentID = agent ที่ request นี้ทำงานในนาม: agent เอง หรือผู้สร้างถ้าเป็น sub (AUTH-26)
	// query ของหลังบ้านใช้ค่านี้ scope ข้อมูล
	AgentID      uint
	SubaccountID uint // 0 = ไม่ใช่ sub
	ParentID     *uint
	Username     string
	Role         models.AgentRole   // sub = role ของผู้สร้าง
	Status       models.AgentStatus // สถานะของบัญชีนี้เอง
	// EffectiveStatus = สถานะที่ใช้ตัดสินสิทธิ์ (sub = เข้มที่สุดระหว่าง sub กับผู้สร้าง)
	EffectiveStatus    models.AgentStatus
	MustChangePassword bool
	MustChangePasscode bool
	PasscodeSet        bool
	SessionID          string
}

// AccountID = id ในตารางของบัญชีนี้ (user_agents.id หรือ subaccounts.id)
func (a Actor) AccountID() uint {
	if a.AccountType == models.AccountTypeSub {
		return a.SubaccountID
	}
	return a.AgentID
}

// Gate = ด่านหลัง login ที่ยังไม่ผ่าน (AUTH-29)
func (a Actor) Gate() agentAuthCore.Gate {
	return agentAuthCore.CurrentGate(a.MustChangePassword, a.MustChangePasscode, a.PasscodeSet)
}

func newActor(a account, sid string) Actor {
	act := Actor{
		AccountType:        a.Type,
		AgentID:            a.AgentID,
		ParentID:           a.ParentID,
		Username:           a.Username,
		Role:               a.Role,
		Status:             a.Status,
		EffectiveStatus:    a.EffectiveStatus(),
		MustChangePassword: a.MustChangePassword,
		MustChangePasscode: a.MustChangePasscode,
		PasscodeSet:        a.PasscodeSet(),
		SessionID:          sid,
	}
	if a.IsSub() {
		act.SubaccountID = a.ID
	}
	return act
}
