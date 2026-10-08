package models

import "time"

// ตารางของ UserMember (user_members · user_member_game_settings · user_member_balances) — docs/modules/agent_management.md หัวข้อ 6

// UserMember — ผู้เล่น (ตารางแยกจากฝั่ง agent)
type UserMember struct {
	ID           uint        `gorm:"column:id;primaryKey"`
	AgentID      uint        `gorm:"column:agent_id"` // ผู้สร้าง
	Username     string      `gorm:"column:username"`
	PasswordHash string      `gorm:"column:password_hash"`
	Name         string      `gorm:"column:name"`
	Phone        *string     `gorm:"column:phone"`
	Currency     string      `gorm:"column:currency"`
	Status       AgentStatus `gorm:"column:status"`
	LastLoginAt  *time.Time  `gorm:"column:last_login_at"`
	LastLoginIP  *string     `gorm:"column:last_login_ip"`
	CreatedAt    time.Time   `gorm:"column:created_at"`
	UpdatedAt    time.Time   `gorm:"column:updated_at"`
}

func (UserMember) TableName() string { return "user_members" }

// UserMemberGameSetting — Commission ของ UserMember ต่อเกม (MGMT-21)
type UserMemberGameSetting struct {
	UserMemberID uint      `gorm:"column:user_member_id;primaryKey"`
	GameCode     string    `gorm:"column:game_code;primaryKey"`
	Category     string    `gorm:"column:category"`
	CommissionBP int       `gorm:"column:commission_bp"`
	CreatedBy    string    `gorm:"column:created_by"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedBy    string    `gorm:"column:updated_by"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (UserMemberGameSetting) TableName() string { return "user_member_game_settings" }

// UserMemberBalance — ยอดเงินของ UserMember ต่อสกุล หน่วยย่อย 1/100 (MGMT-15A)
type UserMemberBalance struct {
	UserMemberID uint      `gorm:"column:user_member_id;primaryKey"`
	Currency     string    `gorm:"column:currency;primaryKey"`
	Amount       int64     `gorm:"column:amount"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (UserMemberBalance) TableName() string { return "user_member_balances" }
