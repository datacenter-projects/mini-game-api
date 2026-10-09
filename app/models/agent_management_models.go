package models

import "time"

// ตารางของ module ② — docs/modules/agent_management.md หัวข้อ 6

// AgentCurrency — สกุลที่บัญชีฝั่ง agent ใช้ได้ (MGMT-10 – MGMT-14)
type AgentCurrency struct {
	AgentID  uint   `gorm:"column:agent_id;primaryKey"`
	Currency string `gorm:"column:currency;primaryKey"`
}

func (AgentCurrency) TableName() string { return "agent_currencies" }

// AgentGameSetting — ค่าหุ้นส่วนและเปิด / ปิดเกม ต่อเกม (MGMT-16, MGMT-20) · ค่า % เป็น float64 ปัด 4 ตำแหน่ง (กฎข้อ 9)
type AgentGameSetting struct {
	AgentID      uint      `gorm:"column:agent_id;primaryKey"`
	ParentID     *uint     `gorm:"column:parent_id"` // ผู้สร้างของเจ้าของแถว · Superadmin = nil (MGMT-62)
	GameCode     string    `gorm:"column:game_code;primaryKey"`
	Category     string    `gorm:"column:category"`
	PTFromParent float64   `gorm:"column:pt_from_parent"` // ได้รับจากผู้สร้าง
	Force        float64   `gorm:"column:force"`
	Remain       float64   `gorm:"column:remain"`
	Commission   float64   `gorm:"column:commission"`
	Status       bool      `gorm:"column:status"`      // รับ PT ไหม (MGMT-20)
	StatusGame   bool      `gorm:"column:status_game"` // เปิด / ปิดทีละเกม
	CreatedBy    string    `gorm:"column:created_by"`  // username ผู้สร้างค่า PT
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedBy    string    `gorm:"column:updated_by"` // username คนที่แก้ค่า PT ล่าสุด (sub = owner@name)
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (AgentGameSetting) TableName() string { return "agent_game_settings" }

// AgentBalance — ยอดเงินของบัญชีฝั่ง agent ต่อสกุล ทศนิยม ปัด 4 ตำแหน่ง (MGMT-15A · กฎข้อ 9)
type AgentBalance struct {
	AgentID   uint      `gorm:"column:agent_id;primaryKey"`
	Currency  string    `gorm:"column:currency;primaryKey"`
	Amount    float64   `gorm:"column:amount"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (AgentBalance) TableName() string { return "agent_balances" }

// BalanceOwnerType — เจ้าของยอดใน ledger
type BalanceOwnerType string

const (
	BalanceOwnerAgent  BalanceOwnerType = "AGENT"
	BalanceOwnerMember BalanceOwnerType = "MEMBER"
)

// LedgerReason — เหตุผลของการเปลี่ยนยอด
type LedgerReason string

const (
	LedgerInitialTransferOut    LedgerReason = "INITIAL_TRANSFER_OUT"
	LedgerInitialTransferIn     LedgerReason = "INITIAL_TRANSFER_IN"
	LedgerInitialFromSuperadmin LedgerReason = "INITIAL_FROM_SUPERADMIN"
)

// BalanceLedger — ทุกการเปลี่ยนยอดมี 1 แถว (กฎข้อ 11)
type BalanceLedger struct {
	ID           uint             `gorm:"column:id;primaryKey"`
	OwnerType    BalanceOwnerType `gorm:"column:owner_type"`
	OwnerID      uint             `gorm:"column:owner_id"`
	Currency     string           `gorm:"column:currency"`
	Amount       float64          `gorm:"column:amount"` // + เข้า / − ออก
	BalanceAfter float64          `gorm:"column:balance_after"`
	Reason       LedgerReason     `gorm:"column:reason"`
	RefType      *string          `gorm:"column:ref_type"`
	RefID        *uint            `gorm:"column:ref_id"`
	RequestID    string           `gorm:"column:request_id"`
	ActorType    AccountType      `gorm:"column:actor_type"`
	ActorID      uint             `gorm:"column:actor_id"`
	CreatedAt    time.Time        `gorm:"column:created_at"`
}

func (BalanceLedger) TableName() string { return "balance_ledger" }

// CreateRequest — request_id ของเส้นสร้างที่ใช้ไปแล้ว (MGMT-15A)
type CreateRequest struct {
	RequestID   string      `gorm:"column:request_id;primaryKey"`
	CreatorType AccountType `gorm:"column:creator_type"`
	CreatorID   uint        `gorm:"column:creator_id"`
	TargetType  string      `gorm:"column:target_type"` // AGENT / MEMBER
	TargetID    uint        `gorm:"column:target_id"`
	CreatedAt   time.Time   `gorm:"column:created_at"`
}

func (CreateRequest) TableName() string { return "create_requests" }

// AccountChangeAction — ประเภทการเปลี่ยนแปลงใน account_change_logs (MGMT-60)
type AccountChangeAction string

const (
	ChangeCreate         AccountChangeAction = "CREATE"
	ChangeUpdateInfo     AccountChangeAction = "UPDATE_INFO"
	ChangeUpdateStatus   AccountChangeAction = "UPDATE_STATUS"
	ChangeUpdatePT       AccountChangeAction = "UPDATE_PT"
	ChangeUpdateGames    AccountChangeAction = "UPDATE_GAMES"
	ChangeSyncFromCSM    AccountChangeAction = "SYNC_FROM_CSM" // ระบบปรับ Share Master ตาม Company Seamless Master (MGMT-19)
	ChangeSubStatus      AccountChangeAction = "SUB_STATUS"
	ChangeInitialBalance AccountChangeAction = "INITIAL_BALANCE"
)

// AccountChangeLog — ประวัติการเปลี่ยนแปลงบัญชี (MGMT-60) · ห้ามเก็บรหัสผ่าน
type AccountChangeLog struct {
	ID             uint                `gorm:"column:id;primaryKey"`
	ActorType      AccountType         `gorm:"column:actor_type"`
	ActorID        uint                `gorm:"column:actor_id"`
	ActorUsername  string              `gorm:"column:actor_username"`
	TargetType     string              `gorm:"column:target_type"` // AGENT / MEMBER / SUB
	TargetID       uint                `gorm:"column:target_id"`
	TargetUsername string              `gorm:"column:target_username"`
	Action         AccountChangeAction `gorm:"column:action"`
	OldValue       *string             `gorm:"column:old_value"` // JSON
	NewValue       *string             `gorm:"column:new_value"` // JSON
	IP             *string             `gorm:"column:ip"`
	RequestID      *string             `gorm:"column:request_id"`
	CreatedAt      time.Time           `gorm:"column:created_at"`
}

func (AccountChangeLog) TableName() string { return "account_change_logs" }
