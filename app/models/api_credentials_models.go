package models

import (
	"time"

	"app/pkg/utils"
)

// APICredential — Key, ลิงก์ตอบกลับของเจ้าของ Key 1 บัญชี (docs/modules/account.md ACC-01–ACC-06)
type APICredential struct {
	AgentID     uint      `gorm:"column:agent_id;primaryKey"` // เจ้าของ Key
	APIKey      string    `gorm:"column:api_key"`             // Key ตรงๆ hex 64 ตัว (ACC-03, ACC-04)
	CallbackURL *string   `gorm:"column:callback_url"`        // nil = ยังไม่ตั้ง
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (APICredential) TableName() string { return "api_credentials" }

// APIAllowedIP — IP / CIDR ของ IPv4 ที่เรียก API ได้ (ACC-07)
type APIAllowedIP struct {
	ID        uint      `gorm:"column:id;primaryKey"`
	AgentID   uint      `gorm:"column:agent_id"`
	CIDR      string    `gorm:"column:cidr"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (APIAllowedIP) TableName() string { return "api_allowed_ips" }

// APICredentialLog — ประวัติการบันทึกลิงก์ / IP (ACC-09) — ห้ามมี Key
type APICredentialLog struct {
	ID             uint            `gorm:"column:id;primaryKey"`
	AgentID        uint            `gorm:"column:agent_id"`
	ActorType      AccountType     `gorm:"column:actor_type"`
	ActorID        uint            `gorm:"column:actor_id"`
	ActorUsername  string          `gorm:"column:actor_username"`
	OldCallbackURL *string         `gorm:"column:old_callback_url"`
	NewCallbackURL *string         `gorm:"column:new_callback_url"`
	OldIPs         utils.TextArray `gorm:"column:old_ips"`
	NewIPs         utils.TextArray `gorm:"column:new_ips"`
	IP             *string         `gorm:"column:ip"`
	RequestID      *string         `gorm:"column:request_id"`
	CreatedAt      time.Time       `gorm:"column:created_at"`
}

func (APICredentialLog) TableName() string { return "api_credential_logs" }
