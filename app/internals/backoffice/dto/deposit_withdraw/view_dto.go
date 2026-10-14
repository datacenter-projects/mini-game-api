package depositwithdraw

import (
	"time"

	depositWithdrawCore "app/app/core/deposit_withdraw"
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/app/models"
)

// DownlinesRequest — POST /api/v1/bo/pr/deposit-withdraw/downlines/list (DW1) · ทุกค่าไม่บังคับ · page / limit ใช้ utils.NewPage
type DownlinesRequest struct {
	Keyword string `json:"keyword"` // ค้นเฉพาะลูกตรง (DW-41)
	Page    int    `json:"page"`
	Limit   int    `json:"limit"`
}

func (r *DownlinesRequest) Validate() error { return agentManagementDto.NormalizeKeyword(&r.Keyword) }

// CurrencyAmounts — ยอดต่อสกุล ครบทุกสกุลที่ระบบรองรับ (agentManagementCore.Currencies) · key ตัวใหญ่ · ไม่มียอด = 0 (DW-43)
type CurrencyAmounts map[string]float64

// DownlineRow — 1 แถวของ DW1 (DW-42) · role / id ส่งกลับใน DW2 ได้ทันที
type DownlineRow struct {
	ID          uint            `json:"id"`
	Role        string          `json:"role"`
	Username    string          `json:"username"`
	Name        string          `json:"name"`
	Status      string          `json:"status"`
	HasBalance  bool            `json:"has_balance"` // false = สาย Seamless (DW-44)
	Currency    CurrencyAmounts `json:"currency"`
	LastLoginIP string          `json:"last_login_ip"` // "" = ยังไม่เคย login (ACC-32)
	LastLoginAt string          `json:"last_login_at"`
}

// CreditDetail — ยอดของคนที่ login (แม่ · sub = เจ้าของ) (DW-44A) · Superadmin ไม่มียอดของตัวเอง = 0 ทุกสกุล
type CreditDetail struct {
	Currency CurrencyAmounts `json:"currency"`
}

// DownlinesSummary — ส่วนของ data ใน DW1 ที่อยู่นอก PageData (lead อนุมัติยกเว้นกฎข้อ 19 · 2026-10-14)
// controller ประกอบกับ response.PageData เอง — dto / service ไม่ import pkg/response (ดึง fiber มาด้วย)
type DownlinesSummary struct {
	CreditDetail CreditDetail               `json:"credit_detail"`
	TotalSum     map[string]CurrencyAmounts `json:"total_sum"` // role ตัวเล็ก (agent / member …) → ยอดรวมของลูกตรงทั้งหมด · เฉพาะ role ที่มีลูก · ไม่มีลูก = {}
}

// bangkok — วันที่ของ DW3 เป็นเวลาไทย (DW-47) · ไม่ใช้ LoadLocation เพื่อไม่พึ่ง tzdata ของเครื่อง
var bangkok = time.FixedZone("Asia/Bangkok", 7*60*60)

const dateLayout = "2006-01-02"

// LogsRequest — POST /api/v1/bo/pr/deposit-withdraw/logs/list (DW3)
type LogsRequest struct {
	DateFrom        string `json:"date_from"`
	DateTo          string `json:"date_to"`
	TransactionType string `json:"transaction_type"` // "" = ทั้งหมด
	Role            string `json:"role"`             // ส่งคู่กับ user_id · "" = ทุกลูก
	UserID          uint   `json:"user_id"`
	Page            int    `json:"page"`
	Limit           int    `json:"limit"`

	From  time.Time                `json:"-"` // เที่ยงคืนของ date_from (เวลาไทย)
	To    time.Time                `json:"-"` // เที่ยงคืนของวันถัดจาก date_to — ใช้ created_at < To
	Type  models.DWTransactionType `json:"-"`
	Owner models.BalanceOwnerType  `json:"-"` // "" = ทุกลูก
}

// Validate — date_from → date_to → ช่วงวัน → transaction_type → role / user_id (DW-47 / DW-48)
func (r *LogsRequest) Validate() error {
	from, err := time.ParseInLocation(dateLayout, r.DateFrom, bangkok)
	if err != nil {
		return agentManagementDto.Invalid("date_from", "ต้องเป็นวันที่รูปแบบ YYYY-MM-DD", "must be a date in YYYY-MM-DD format")
	}
	to, err := time.ParseInLocation(dateLayout, r.DateTo, bangkok)
	if err != nil {
		return agentManagementDto.Invalid("date_to", "ต้องเป็นวันที่รูปแบบ YYYY-MM-DD", "must be a date in YYYY-MM-DD format")
	}
	if !depositWithdrawCore.DateRangeOK(from, to) {
		return agentManagementDto.Invalid("date_from / date_to", "ต้องเรียงจากวันเริ่มต้นไปวันสิ้นสุด และห่างกันไม่เกิน 31 วัน",
			"must be in order and span at most 31 days")
	}
	r.From, r.To = from, to.AddDate(0, 0, 1)
	switch models.DWTransactionType(r.TransactionType) {
	case "", models.DWDeposit, models.DWWithdraw:
		r.Type = models.DWTransactionType(r.TransactionType)
	default:
		return agentManagementDto.Invalid("transaction_type", "ต้องเป็น DEPOSIT หรือ WITHDRAW", "must be DEPOSIT or WITHDRAW")
	}
	if r.Role == "" && r.UserID == 0 {
		return nil
	}
	if r.Role == "" || r.UserID == 0 {
		return agentManagementDto.Invalid("role / user_id", "ต้องส่งคู่กัน (ดูทุกลูกให้เว้นว่างทั้งคู่)", "must be sent together (leave both empty for all)")
	}
	owner, ok := depositWithdrawCore.OwnerOfRole(r.Role)
	if !ok {
		return agentManagementDto.Invalid("role", "ต้องเป็น COMPANY, SHAREHOLDER, AGENT หรือ MEMBER", "must be COMPANY, SHAREHOLDER, AGENT or MEMBER")
	}
	r.Owner = owner
	return nil
}

// LogRow — 1 แถวของ DW3 · username / role = snapshot ตอนทำรายการ
type LogRow struct {
	ID                  uint    `json:"id"`
	RequestID           string  `json:"request_id"`
	TransactionType     string  `json:"transaction_type"`
	ActorUsername       string  `json:"actor_username"`
	Role                string  `json:"role"`
	UserID              uint    `json:"user_id"`
	Username            string  `json:"username"`
	Currency            string  `json:"currency"`
	Amount              float64 `json:"amount"`
	Note                string  `json:"note"`
	ParentUnlimited     bool    `json:"parent_unlimited"`
	ParentBalanceBefore float64 `json:"parent_balance_before"`
	ParentBalanceAfter  float64 `json:"parent_balance_after"`
	TargetBalanceBefore float64 `json:"target_balance_before"`
	TargetBalanceAfter  float64 `json:"target_balance_after"`
	CreatedAt           string  `json:"created_at"`
}
