package models

import "time"

// ตารางของ UserMember (user_members · user_member_game_settings) — docs/modules/member_management.md หัวข้อ 4

// UserMember — ผู้เล่น (ตารางแยกจากฝั่ง agent)
type UserMember struct {
	ID           uint        `gorm:"column:id;primaryKey"`
	AgentID      uint        `gorm:"column:agent_id"` // ผู้สร้าง
	Username     string      `gorm:"column:username"`
	PasswordHash string      `gorm:"column:password_hash"`
	Name         string      `gorm:"column:name"`
	Phone        *string     `gorm:"column:phone"`
	Currency     string      `gorm:"column:currency"`
	Credit       float64     `gorm:"column:credit"` // ยอดเงิน หน่วยสกุล (เช่น 25.5) — float ตามที่ทีมตกลง 2026-10-09
	Status       AgentStatus `gorm:"column:status"`
	Cnf          string      `gorm:"column:cnf;default:'{\"parent\": []}'"` // JSONB สายชั้นบน {"parent":[{"id","position"}]} = สายของผู้สร้าง + ผู้สร้าง
	LastLoginAt  *time.Time  `gorm:"column:last_login_at"`
	LastLoginIP  *string     `gorm:"column:last_login_ip"`
	CreatedAt    time.Time   `gorm:"column:created_at"`
	UpdatedAt    time.Time   `gorm:"column:updated_at"`
}

func (UserMember) TableName() string { return "user_members" }

// UserMemberGameSetting — PT ที่ผู้สร้างถือสู้กับ UserMember คนนี้ + Commission ต่อเกม (MGMT-21 แก้ 2026-10-09)
type UserMemberGameSetting struct {
	UserMemberID uint      `gorm:"column:user_member_id;primaryKey"`
	GameCode     string    `gorm:"column:game_code;primaryKey"`
	Category     string    `gorm:"column:category"`
	PTBP         int       `gorm:"column:pt_bp"`     // ผู้สร้างถือสู้กับ Member คนนี้
	RemainBP     int       `gorm:"column:remain_bp"` // ค่าที่ผู้สร้างได้รับ − pt_bp (ระบบคิด)
	CommissionBP int       `gorm:"column:commission_bp"`
	CreatedBy    string    `gorm:"column:created_by"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedBy    string    `gorm:"column:updated_by"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (UserMemberGameSetting) TableName() string { return "user_member_game_settings" }
