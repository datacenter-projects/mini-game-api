package models

import "time"

type AgentRole string

const (
	AgentRoleSuperAdmin  AgentRole = "SUPERADMIN"
	AgentRoleCompany     AgentRole = "COMPANY"
	AgentRoleShareholder AgentRole = "SHAREHOLDER"
	AgentRoleAgent       AgentRole = "AGENT"
)

type AgentStatus string

const (
	AgentStatusActive    AgentStatus = "ACTIVE"
	AgentStatusSuspended AgentStatus = "SUSPENDED"
	AgentStatusLocked    AgentStatus = "LOCKED"
)

type UserAgent struct {
	ID           uint        `gorm:"column:id;primaryKey"`
	ParentID     *uint       `gorm:"column:parent_id"`
	Username     string      `gorm:"column:username"`
	PasswordHash string      `gorm:"column:password_hash"`
	Role         AgentRole   `gorm:"column:role"`
	Status       AgentStatus `gorm:"column:status"`
	LastLoginAt  *time.Time  `gorm:"column:last_login_at"`
	LastLoginIP  *string     `gorm:"column:last_login_ip"`
	CreatedAt    time.Time   `gorm:"column:created_at"`
	UpdatedAt    time.Time   `gorm:"column:updated_at"`
}

func (UserAgent) TableName() string { return "user_agents" }
