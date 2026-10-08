package models

import "time"

// Subaccount — บัญชีย่อยที่ agent สร้าง (docs/modules/agent_auth_phase2.md AUTH-17)
// status ใช้ชุดเดียวกับ agent (AgentStatus)
type Subaccount struct {
	ID                    uint        `gorm:"column:id;primaryKey"`
	AgentID               uint        `gorm:"column:agent_id"` // ผู้สร้าง
	Username              string      `gorm:"column:username"` // {username ผู้สร้าง}@{name}
	PasswordHash          string      `gorm:"column:password_hash"`
	PreviousPasswordHash  *string     `gorm:"column:previous_password_hash"`
	PasscodeHash          *string     `gorm:"column:passcode_hash"`
	MustChangePassword    bool        `gorm:"column:must_change_password"`
	MustChangePasscode    bool        `gorm:"column:must_change_passcode"`
	TempPasswordExpiresAt *time.Time  `gorm:"column:temp_password_expires_at"`
	TempPasscodeExpiresAt *time.Time  `gorm:"column:temp_passcode_expires_at"`
	Name                  *string     `gorm:"column:name"` // ชื่อเล่น (module ② MGMT-41)
	Phone                 *string     `gorm:"column:phone"`
	Permissions           string      `gorm:"column:permissions;default:'{}'"` // JSONB {"menu":"view"|"edit"} · ไม่มี key = off (MGMT-50)
	Status                AgentStatus `gorm:"column:status"`
	LastLoginAt           *time.Time  `gorm:"column:last_login_at"`
	LastLoginIP           *string     `gorm:"column:last_login_ip"`
	CreatedAt             time.Time   `gorm:"column:created_at"`
	UpdatedAt             time.Time   `gorm:"column:updated_at"`
}

func (Subaccount) TableName() string { return "subaccounts" }
