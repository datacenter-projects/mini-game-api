package depositwithdraw

import (
	"strings"
	"time"

	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/pkg/apperr"
)

// HistoryRequest — POST /api/v1/bo/pr/deposit-withdraw/history/list (DW4 · DW-55 – DW-59)
type HistoryRequest struct {
	Username  string `json:"username"`  // ตรงตัว · Validate ตัดช่องว่าง + ตัวเล็ก (username เก็บตัวเล็ก — MGMT-05)
	StartTime string `json:"starttime"` // RFC3339 มี timezone
	EndTime   string `json:"endtime"`
	Page      int    `json:"page"`
	Limit     int    `json:"limit"`

	Start time.Time `json:"-"` // หลัง Validate
	End   time.Time `json:"-"` // รวมขอบ (created_at <= End)
}

// Validate — username → starttime → endtime → ช่วงเวลา (ไม่จำกัดความยาว — DW-58)
func (r *HistoryRequest) Validate() error {
	r.Username = strings.ToLower(strings.TrimSpace(r.Username))
	if r.Username == "" {
		return apperr.ErrValidation.WithMessage("ต้องระบุชื่อผู้ใช้งาน", "Username is required")
	}
	var err error
	if r.Start, err = time.Parse(time.RFC3339, r.StartTime); err != nil {
		return agentManagementDto.Invalid("starttime", "ต้องเป็นวันเวลารูปแบบ RFC3339 มี timezone (เช่น 2026-10-15T00:00:00+07:00)",
			"must be an RFC3339 date-time with timezone (e.g. 2026-10-15T00:00:00+07:00)")
	}
	if r.End, err = time.Parse(time.RFC3339, r.EndTime); err != nil {
		return agentManagementDto.Invalid("endtime", "ต้องเป็นวันเวลารูปแบบ RFC3339 มี timezone (เช่น 2026-10-15T23:59:59+07:00)",
			"must be an RFC3339 date-time with timezone (e.g. 2026-10-15T23:59:59+07:00)")
	}
	if r.End.Before(r.Start) {
		return agentManagementDto.Invalid("starttime / endtime", "เวลาเริ่มต้นต้องไม่เกินเวลาสิ้นสุด", "start must not be after end")
	}
	return nil
}

// HistoryRow — 1 แถวของ DW4 · agent_* = บัญชีที่ถูกทำรายการ (Member ด้วย) · request_* = คนกดจริง (DW-57)
type HistoryRow struct {
	ID              uint    `json:"id"`
	AgentID         uint    `json:"agent_id"`
	AgentUsername   string  `json:"agent_username"`
	Role            string  `json:"role"`
	RequestBy       uint    `json:"request_by"`
	RequestType     string  `json:"request_type"` // AGENT / SUB — id ของ 2 ตารางซ้ำกันได้
	RequestUsername string  `json:"request_username"`
	Currency        string  `json:"currency"`
	TransactionType string  `json:"transaction_type"`
	Amount          float64 `json:"amount"`
	Remark          string  `json:"remark"`
	CreatedAt       string  `json:"created_at"`
}
