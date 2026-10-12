package models

import "time"

// ตารางของ UserMember (user_members · user_member_game_settings) — docs/modules/member_management.md หัวข้อ 4

// UserMember — ผู้เล่น (ตารางแยกจากฝั่ง agent)
type UserMember struct {
	ID               uint        `gorm:"column:id;primaryKey"`
	AgentID          uint        `gorm:"column:agent_id"` // ผู้สร้าง
	Username         string      `gorm:"column:username"`
	PasswordHash     string      `gorm:"column:password_hash"`
	Name             string      `gorm:"column:name"`
	PhoneCountryCode *string     `gorm:"column:phone_country_code"` // เบอร์ 2 field (MGMT-08) · NULL คู่กับ phone · ห้ามซ้ำที่คู่ภายใต้ผู้สร้างเดียวกัน
	Phone            *string     `gorm:"column:phone"`
	Currency         string      `gorm:"column:currency"`
	Credit           float64     `gorm:"column:credit"` // ยอดเงิน หน่วยสกุล (เช่น 25.5) — float ตามที่ทีมตกลง 2026-10-09
	Status           AgentStatus `gorm:"column:status"`
	Cnf              string      `gorm:"column:cnf;default:'{\"parent\": []}'"` // JSONB สายชั้นบน {"parent":[{"id","position"}]} = สายของผู้สร้าง + ผู้สร้าง
	PaymentType      PaymentType `gorm:"column:payment_type;default:'CASH'"`    // วิธีจ่ายเงิน แสดงผลอย่างเดียว (MGMT-63) · ไม่ตั้ง = ค่า default ของ DB
	PaymentAccount   *string     `gorm:"column:payment_account"`                // JSONB ของ MemberPaymentAccount · NULL = ไม่มีบัญชี (MGMT-64)
	WinLimitPerDay   float64     `gorm:"column:win_limit_per_day"`              // ลิมิตชนะต่อวัน หน่วยสกุล · 0 = ไม่จำกัด (MGMT-65)
	LastLoginAt      *time.Time  `gorm:"column:last_login_at"`
	LastLoginIP      *string     `gorm:"column:last_login_ip"`
	CreatedAt        time.Time   `gorm:"column:created_at"`
	UpdatedAt        time.Time   `gorm:"column:updated_at"`
}

func (UserMember) TableName() string { return "user_members" }

// PaymentType — วิธีจ่ายเงินของ Member (MGMT-63) · ข้อมูลแสดงผลอย่างเดียว
type PaymentType string

const (
	PaymentTypeCredit PaymentType = "CREDIT"
	PaymentTypeCash   PaymentType = "CASH"
)

// MemberPaymentAccount — ค่าใน user_members.payment_account (MGMT-64) · ชื่อไม่กรอก = ""
type MemberPaymentAccount struct {
	BankCode      string `json:"bank_code"`
	AccountNumber string `json:"account_number"`
	NameTH        string `json:"name_th"`
	MidnameTH     string `json:"midname_th"`
	SurnameTH     string `json:"surname_th"`
	NameEN        string `json:"name_en"`
	MidnameEN     string `json:"midname_en"`
	SurnameEN     string `json:"surname_en"`
}

// UserMemberGameSetting — PT ที่ผู้สร้างถือสู้กับ UserMember คนนี้ + Commission ต่อเกม (MGMT-21 แก้ 2026-10-09)
type UserMemberGameSetting struct {
	UserMemberID uint      `gorm:"column:user_member_id;primaryKey"`
	GameCode     string    `gorm:"column:game_code;primaryKey"`
	ParentID     uint      `gorm:"column:parent_id"` // agent ผู้สร้าง Member (= user_members.agent_id)
	Category     string    `gorm:"column:category"`
	PT           float64   `gorm:"column:pt"`     // ผู้สร้างถือสู้กับ Member คนนี้
	Remain       float64   `gorm:"column:remain"` // ค่าที่ผู้สร้างได้รับ − pt (ระบบคิด)
	Commission   float64   `gorm:"column:commission"`
	CreatedBy    string    `gorm:"column:created_by"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedBy    string    `gorm:"column:updated_by"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (UserMemberGameSetting) TableName() string { return "user_member_game_settings" }
