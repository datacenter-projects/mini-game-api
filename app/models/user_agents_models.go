package models

import "time"

type AgentRole string

const (
	AgentRoleSuperAdmin  AgentRole = "SUPERADMIN"
	AgentRoleAdmin       AgentRole = "ADMIN"
	AgentRoleCompany     AgentRole = "COMPANY"
	AgentRoleShareholder AgentRole = "SHAREHOLDER"
	AgentRoleAgent       AgentRole = "AGENT"
)

// AgentType — ประเภทย่อยของ Company / Share (docs/modules/agent_management.md MGMT-01) · บัญชีอื่น = nil
type AgentType string

const (
	AgentTypeTransfer         AgentType = "TRANSFER"
	AgentTypeSeamlessReseller AgentType = "SEAMLESS_RESELLER"
	AgentTypeSeamlessMaster   AgentType = "SEAMLESS_MASTER"
	AgentTypeSeamless1to1     AgentType = "SEAMLESS_1TO1"
	AgentTypeShareB2B         AgentType = "B2B"
	AgentTypeShareB2C         AgentType = "B2C"
	AgentTypeShareReseller    AgentType = "RESELLER" // Share B2C ใต้ Company Seamless Reseller
	AgentTypeShareMaster      AgentType = "MASTER"   // Share B2C ใต้ Company Seamless Master
)

type AgentStatus string

const (
	AgentStatusActive    AgentStatus = "ACTIVE"
	AgentStatusSuspended AgentStatus = "SUSPENDED"
	AgentStatusLocked    AgentStatus = "LOCKED"
)

type UserAgent struct {
	ID                    uint        `gorm:"column:id;primaryKey"`
	ParentID              *uint       `gorm:"column:parent_id"`
	Username              string      `gorm:"column:username"`
	PasswordHash          string      `gorm:"column:password_hash"`
	PreviousPasswordHash  *string     `gorm:"column:previous_password_hash"`
	PasscodeHash          *string     `gorm:"column:passcode_hash"`
	MustChangePassword    bool        `gorm:"column:must_change_password"`
	MustChangePasscode    bool        `gorm:"column:must_change_passcode"`
	TempPasswordExpiresAt *time.Time  `gorm:"column:temp_password_expires_at"`
	TempPasscodeExpiresAt *time.Time  `gorm:"column:temp_passcode_expires_at"`
	Name                  *string     `gorm:"column:name"`  // ชื่อ (module ② MGMT-07)
	Phone                 *string     `gorm:"column:phone"` // nil = ไม่ได้กรอก (MGMT-08)
	AgentType             *AgentType  `gorm:"column:agent_type"`
	Role                  AgentRole   `gorm:"column:role"`
	Status                AgentStatus `gorm:"column:status"`
	LastLoginAt           *time.Time  `gorm:"column:last_login_at"`
	LastLoginIP           *string     `gorm:"column:last_login_ip"`
	CreatedAt             time.Time   `gorm:"column:created_at"`
	UpdatedAt             time.Time   `gorm:"column:updated_at"`
}

func (UserAgent) TableName() string { return "user_agents" }
