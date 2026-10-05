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
	AuthAuditPasscodeSetup   AuthAuditAction = "PASSCODE_SETUP"
	AuthAuditPasscodeChange  AuthAuditAction = "PASSCODE_CHANGE"
	AuthAuditPasswordChange  AuthAuditAction = "PASSWORD_CHANGE"
	AuthAuditResetPasscode   AuthAuditAction = "RESET_PASSCODE"
	AuthAuditResetPassword   AuthAuditAction = "RESET_PASSWORD"
	AuthAuditPasscodeBlocked AuthAuditAction = "PASSCODE_BLOCKED"
	AuthAuditLoginBlocked    AuthAuditAction = "LOGIN_BLOCKED"
)

// AuthAuditActorType — ผู้ทำ: บัญชี (AGENT / SUB) หรือไม่ใช่บัญชี (SCRIPT / SYSTEM)
type AuthAuditActorType string

const (
	AuthAuditActorAgent  AuthAuditActorType = "AGENT"
	AuthAuditActorSub    AuthAuditActorType = "SUB"
	AuthAuditActorScript AuthAuditActorType = "SCRIPT" // script บนเซิร์ฟเวอร์
	AuthAuditActorSystem AuthAuditActorType = "SYSTEM" // ระบบบังคับเอง เช่น บล็อกเพราะใส่ผิดครบ
)

// AuthAuditLog — เหตุการณ์ที่เปลี่ยนข้อมูลบัญชี (docs/modules/agent_auth_phase2.md AUTH-49)
type AuthAuditLog struct {
	ID             uint               `gorm:"column:id;primaryKey"`
	Action         AuthAuditAction    `gorm:"column:action"`
	ActorType      AuthAuditActorType `gorm:"column:actor_type"`
	ActorID        *uint              `gorm:"column:actor_id"`
	ActorUsername  *string            `gorm:"column:actor_username"`
	TargetType     *AccountType       `gorm:"column:target_type"`
	TargetID       *uint              `gorm:"column:target_id"`
	TargetUsername string             `gorm:"column:target_username"`
	IP             *string            `gorm:"column:ip"`
	UserAgent      *string            `gorm:"column:user_agent"`
	RequestID      *string            `gorm:"column:request_id"`
	CreatedAt      time.Time          `gorm:"column:created_at"`
}

func (AuthAuditLog) TableName() string { return "auth_audit_logs" }
