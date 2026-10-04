package models

import "time"

// AccountType แยกบัญชีหลังบ้าน 2 ตาราง — id ของ user_agents กับ subaccounts ซ้ำกันได้
type AccountType string

const (
	AccountTypeAgent AccountType = "AGENT" // user_agents
	AccountTypeSub   AccountType = "SUB"   // subaccounts
)

type AuthAuditAction string

const (
	AuthAuditResetPasscode AuthAuditAction = "RESET_PASSCODE"
	AuthAuditResetPassword AuthAuditAction = "RESET_PASSWORD"
)

// AuthAuditLog — บันทึกการรีเซ็ตของ admin (docs/modules/agent_auth_phase2.md AUTH-49)
type AuthAuditLog struct {
	ID         uint            `gorm:"column:id;primaryKey"`
	ActorType  AccountType     `gorm:"column:actor_type"`
	ActorID    uint            `gorm:"column:actor_id"`
	TargetType AccountType     `gorm:"column:target_type"`
	TargetID   uint            `gorm:"column:target_id"`
	Action     AuthAuditAction `gorm:"column:action"`
	IP         string          `gorm:"column:ip"`
	CreatedAt  time.Time       `gorm:"column:created_at"`
}

func (AuthAuditLog) TableName() string { return "auth_audit_logs" }
