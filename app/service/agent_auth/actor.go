// Package agentauth คือ business logic ของ backoffice auth — spec: docs/modules/agent_auth.md
package agentauth

import "app/app/models"

// Actor คือผู้ใช้หลังบ้านที่ยืนยันตัวตนแล้วของ request นี้
// middleware Authenticated เป็นคนสร้าง แล้ว controller อ่านด้วย middleware.GetActor(c)
type Actor struct {
	AgentID   uint
	ParentID  *uint
	Username  string
	Role      models.AgentRole
	Status    models.AgentStatus
	SessionID string
}
