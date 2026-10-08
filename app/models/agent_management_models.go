package models

import "time"

// ตารางของ module ② — docs/modules/agent_management.md หัวข้อ 6

// AgentCurrency — สกุลที่บัญชีฝั่ง agent ใช้ได้ (MGMT-10 – MGMT-14)
type AgentCurrency struct {
	AgentID  uint   `gorm:"column:agent_id;primaryKey"`
	Currency string `gorm:"column:currency;primaryKey"`
}

func (AgentCurrency) TableName() string { return "agent_currencies" }

// AgentGameSetting — ค่าหุ้นส่วนและเปิด / ปิดเกม ต่อเกม (MGMT-16, MGMT-20) · ค่า % เป็น bp
type AgentGameSetting struct {
	AgentID        uint      `gorm:"column:agent_id;primaryKey"`
	GameCode       string    `gorm:"column:game_code;primaryKey"`
	Category       string    `gorm:"column:category"`
	PTFromParentBP int       `gorm:"column:pt_from_parent_bp"` // ได้รับจากผู้สร้าง
	PTBP           int       `gorm:"column:pt_bp"`             // ถือจาก Member ใต้ตัวเอง
	ForceBP        int       `gorm:"column:force_bp"`
	RemainBP       int       `gorm:"column:remain_bp"`
	CommissionBP   int       `gorm:"column:commission_bp"`
	Status         bool      `gorm:"column:status"`      // status ของ PT — ❓ ความหมายรอ lead (MGMT-20)
	StatusGame     bool      `gorm:"column:status_game"` // เปิด / ปิดทีละเกม
	CreatedBy      string    `gorm:"column:created_by"`  // username ผู้สร้างค่า PT
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedBy      string    `gorm:"column:updated_by"` // username คนที่แก้ค่า PT ล่าสุด (sub = owner@name)
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (AgentGameSetting) TableName() string { return "agent_game_settings" }

// Member — ผู้เล่น (ตารางแยกจากฝั่ง agent)
type Member struct {
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

func (Member) TableName() string { return "members" }

// MemberGameSetting — Commission ของ Member ต่อเกม (MGMT-21)
type MemberGameSetting struct {
	MemberID     uint      `gorm:"column:member_id;primaryKey"`
	GameCode     string    `gorm:"column:game_code;primaryKey"`
	Category     string    `gorm:"column:category"`
	CommissionBP int       `gorm:"column:commission_bp"`
	CreatedBy    string    `gorm:"column:created_by"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedBy    string    `gorm:"column:updated_by"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (MemberGameSetting) TableName() string { return "member_game_settings" }

// AgentBalance / MemberBalance — ยอดเงินต่อสกุล หน่วยย่อย 1/100 (MGMT-15A)
type AgentBalance struct {
	AgentID   uint      `gorm:"column:agent_id;primaryKey"`
	Currency  string    `gorm:"column:currency;primaryKey"`
	Amount    int64     `gorm:"column:amount"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (AgentBalance) TableName() string { return "agent_balances" }

type MemberBalance struct {
	MemberID  uint      `gorm:"column:member_id;primaryKey"`
	Currency  string    `gorm:"column:currency;primaryKey"`
	Amount    int64     `gorm:"column:amount"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (MemberBalance) TableName() string { return "member_balances" }

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
	Amount       int64            `gorm:"column:amount"` // + เข้า / − ออก
	BalanceAfter int64            `gorm:"column:balance_after"`
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
