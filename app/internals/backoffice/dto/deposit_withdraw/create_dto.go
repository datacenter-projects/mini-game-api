// Package depositwithdraw คือ request / response ของเส้น /deposit-withdraw/* — spec: docs/modules/deposit_withdraw.md หัวข้อ 5
// ไม่มี null ใน API (account ACC-32): controller ใช้ utils.ParseBodyNoNull · ข้อความว่าง = "" · ตัวเลขว่าง = 0
package depositwithdraw

import (
	"regexp"
	"strings"
	"unicode/utf8"

	agentManagementCore "app/app/core/agent_management"
	depositWithdrawCore "app/app/core/deposit_withdraw"
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/app/models"
	"app/pkg/utils"
)

// uuidRe — request_id รูปแบบเดียวกับ agents / members create (MGMT-15A · DW-24)
var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// CreateRequest — POST /api/v1/bo/pr/deposit-withdraw/create (DW2) · passcode ใน body ตรวจโดย middleware RequirePasscode
type CreateRequest struct {
	RequestID       string  `json:"request_id"`
	Role            string  `json:"role"` // COMPANY / SHAREHOLDER / AGENT / MEMBER (DW-02)
	UserID          uint    `json:"user_id"`
	Currency        string  `json:"currency"`
	Amount          float64 `json:"amount"` // เงินรับเป็น float64 (ทีมตกลง 2026-10-12)
	TransactionType string  `json:"transaction_type"`
	Note            string  `json:"note"` // ไม่บังคับ · Validate ตัดช่องว่างหัวท้าย

	Owner models.BalanceOwnerType  `json:"-"` // ตารางของบัญชีปลายทาง หลัง Validate
	Type  models.DWTransactionType `json:"-"` // หลัง Validate
}

// Validate — ไล่ตามลำดับ field ใน body: request_id → role → user_id → currency → amount → transaction_type → note
func (r *CreateRequest) Validate() error {
	if !uuidRe.MatchString(r.RequestID) {
		return agentManagementDto.Invalid("request_id", "ต้องเป็น UUID", "must be a UUID")
	}
	owner, ok := depositWithdrawCore.OwnerOfRole(r.Role)
	if !ok {
		return agentManagementDto.Invalid("role", "ต้องเป็น COMPANY, SHAREHOLDER, AGENT หรือ MEMBER", "must be COMPANY, SHAREHOLDER, AGENT or MEMBER")
	}
	r.Owner = owner
	if r.UserID == 0 {
		return agentManagementDto.Invalid("user_id", "ต้องส่งเป็นจำนวนเต็มบวก", "must be a positive integer")
	}
	if !agentManagementCore.IsSupportedCurrency(r.Currency) {
		return agentManagementDto.Invalid("currency", "สกุลเงินไม่ถูกต้อง", "is not a supported currency")
	}
	if !utils.IsDecimal4(r.Amount) {
		return agentManagementDto.Invalid("amount", "ต้องเป็นตัวเลข ทศนิยมไม่เกิน 4 ตำแหน่ง", "must be a number with at most 4 decimals")
	}
	if r.Amount <= 0 {
		return agentManagementDto.Invalid("amount", "ต้องมากกว่า 0", "must be greater than 0")
	}
	if r.Amount > depositWithdrawCore.MaxAmount {
		max := utils.FormatNumGrouped(depositWithdrawCore.MaxAmount)
		return agentManagementDto.Invalid("amount", "ต้องไม่เกิน "+max, "must not exceed "+max)
	}
	switch models.DWTransactionType(r.TransactionType) {
	case models.DWDeposit, models.DWWithdraw:
		r.Type = models.DWTransactionType(r.TransactionType)
	default:
		return agentManagementDto.Invalid("transaction_type", "ต้องเป็น DEPOSIT หรือ WITHDRAW", "must be DEPOSIT or WITHDRAW")
	}
	r.Note = strings.TrimSpace(r.Note)
	if utf8.RuneCountInString(r.Note) > depositWithdrawCore.NoteMaxRunes {
		return agentManagementDto.Invalid("note", "ต้องยาวไม่เกิน 200 ตัวอักษร", "must be at most 200 characters")
	}
	return nil
}

// CreateResponse — data ของ DW2 · ส่ง request_id เดิมซ้ำ = ค่าเดิมจาก log (DW-25)
type CreateResponse struct {
	ID                 uint    `json:"id"`
	RequestID          string  `json:"request_id"`
	TransactionType    string  `json:"transaction_type"`
	Role               string  `json:"role"`
	UserID             uint    `json:"user_id"`
	Username           string  `json:"username"`
	Currency           string  `json:"currency"`
	Amount             float64 `json:"amount"`
	Note               string  `json:"note"`
	ParentUnlimited    bool    `json:"parent_unlimited"` // true = รายการก่อน 2026-10-15 ที่ Superadmin ไม่มียอด · รายการใหม่ = false เสมอ (DQ7)
	ParentBalanceAfter float64 `json:"parent_balance_after"`
	TargetBalanceAfter float64 `json:"target_balance_after"`
	CreatedAt          string  `json:"created_at"`
}
