package models

import "time"

// ตารางของ module deposit_withdraw — docs/modules/deposit_withdraw.md หัวข้อ 6

// DWTransactionType — ฝาก / ถอน
type DWTransactionType string

const (
	DWDeposit  DWTransactionType = "DEPOSIT"  // แม่ลด · ลูกเพิ่ม
	DWWithdraw DWTransactionType = "WITHDRAW" // ลูกลด · แม่เพิ่ม
)

// DWStatus — ผลของรายการ (DW-35) · ไม่มี pending (ย้ายเงินจบใน request เดียว)
type DWStatus string

const (
	DWStatusSuccess DWStatus = "SUCCESS"
	DWStatusFailed  DWStatus = "FAILED" // ไม่มี ledger · เงินไม่ขยับ · ไม่จอง request_id
)

// เหตุผลใน balance_ledger ของการฝาก / ถอน (DW-33)
const (
	LedgerDepositOut  LedgerReason = "DEPOSIT_OUT"
	LedgerDepositIn   LedgerReason = "DEPOSIT_IN"
	LedgerWithdrawOut LedgerReason = "WITHDRAW_OUT"
	LedgerWithdrawIn  LedgerReason = "WITHDRAW_IN"

	// เลิกใช้ 2026-10-15 (Superadmin มียอดจริงแล้ว) · คงไว้เพราะมีแถวเดิมใน balance_ledger
	LedgerDepositFromSuperadmin LedgerReason = "DEPOSIT_FROM_SUPERADMIN"
	LedgerWithdrawToSuperadmin  LedgerReason = "WITHDRAW_TO_SUPERADMIN"

	// LedgerSeedSuperadmin — ยอดตั้งต้นของ Superadmin (migration 20261015100000 · script create_superadmin)
	LedgerSeedSuperadmin LedgerReason = "SEED_SUPERADMIN"
)

// SeedSuperadminRequestID — balance_ledger.request_id ของ LedgerSeedSuperadmin (ตรงกับใน migration)
const SeedSuperadminRequestID = "seed-superadmin"

// LedgerRefDW — balance_ledger.ref_type ของการฝาก / ถอน · ref_id = deposit_withdraw_logs.id
const LedgerRefDW = "DW"

// CreateRequestTargetDW — create_requests.target_type ของการฝาก / ถอน (DW-26 ทะเบียน request_id กลาง)
const CreateRequestTargetDW = "DW"

// DepositWithdrawLog — 1 แถวต่อการกดฝาก / ถอน 1 ครั้ง · username / role เป็น snapshot ตอนทำรายการ
type DepositWithdrawLog struct {
	ID                  uint              `gorm:"column:id;primaryKey"`
	RequestID           string            `gorm:"column:request_id"`
	TransactionType     DWTransactionType `gorm:"column:transaction_type"`
	Status              DWStatus          `gorm:"column:status"`     // ต้องตั้งเสมอ — GORM ส่ง "" แทนค่า default ของ DB
	ActorType           AccountType       `gorm:"column:actor_type"` // คนกดจริง
	ActorID             uint              `gorm:"column:actor_id"`
	ActorUsername       string            `gorm:"column:actor_username"`
	ParentID            uint              `gorm:"column:parent_id"` // แม่ (sub → เจ้าของ)
	ParentUsername      string            `gorm:"column:parent_username"`
	TargetType          BalanceOwnerType  `gorm:"column:target_type"` // AGENT / MEMBER
	TargetID            uint              `gorm:"column:target_id"`
	TargetRole          string            `gorm:"column:target_role"` // COMPANY / SHAREHOLDER / AGENT / MEMBER
	TargetUsername      string            `gorm:"column:target_username"`
	Currency            string            `gorm:"column:currency"`
	Amount              float64           `gorm:"column:amount"`
	ParentUnlimited     bool              `gorm:"column:parent_unlimited"` // true = แถวก่อน 2026-10-15 ที่แม่เป็น Superadmin วงเงินไม่จำกัด · รายการใหม่ = false เสมอ
	ParentBalanceBefore float64           `gorm:"column:parent_balance_before"`
	ParentBalanceAfter  float64           `gorm:"column:parent_balance_after"`
	TargetBalanceBefore float64           `gorm:"column:target_balance_before"`
	TargetBalanceAfter  float64           `gorm:"column:target_balance_after"`
	Note                string            `gorm:"column:note"`
	IP                  string            `gorm:"column:ip"`
	CreatedAt           time.Time         `gorm:"column:created_at"`
}

func (DepositWithdrawLog) TableName() string { return "deposit_withdraw_logs" }
