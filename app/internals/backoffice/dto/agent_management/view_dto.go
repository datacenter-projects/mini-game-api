package agentmanagement

import (
	"strings"

	"app/pkg/apperr"
	"app/pkg/utils"
)

// รูปแบบที่ใช้แสดงบัญชี — รายชื่อดาวน์ไลน์ · รายละเอียด · copy-sources · account Profile (ACC-16, ACC-19)

// PTGroupView — ค่าหุ้นส่วนฝั่ง agent ชุดเดียวต่อระบบ (MGMT-16) · เวลาเป็น RFC 3339
type PTGroupView struct {
	PTFromParent      utils.Percent `json:"pt_from_parent"`
	PT                utils.Percent `json:"pt"`
	Force             utils.Percent `json:"force"`
	RemainQuota       utils.Percent `json:"remain_quota"`
	CommissionPercent utils.Percent `json:"commission_percent"`
	Status            bool          `json:"status"`
	CreatedAt         string        `json:"created_at"`
	CreatedBy         string        `json:"created_by"`
	UpdatedAt         string        `json:"updated_at"`
	UpdatedBy         string        `json:"updated_by"`
}

// MemberPTGroupView — pt ที่ผู้สร้างถือสู้กับ Member · remain_quota (ระบบคิด) · Commission (MGMT-21 แก้ 2026-10-09)
type MemberPTGroupView struct {
	PT                utils.Percent `json:"pt"`
	RemainQuota       utils.Percent `json:"remain_quota"`
	CommissionPercent utils.Percent `json:"commission_percent"`
	CreatedAt         string        `json:"created_at"`
	CreatedBy         string        `json:"created_by"`
	UpdatedAt         string        `json:"updated_at"`
	UpdatedBy         string        `json:"updated_by"`
}

// BalanceView — ยอดแยกสกุล (account ACC-19)
type BalanceView struct {
	Currency string      `json:"currency"`
	Amount   utils.Money `json:"amount"`
}

// DownlineRow — 1 แถวของ POST /manage/downlines/list (MGMT-28)
// PT = nil เมื่อผู้เรียกไม่มีสิทธิ์ pt ≥ view → ไม่มี field pt (MGMT-51)
type DownlineRow struct {
	ID       uint          `json:"id"`
	Role     string        `json:"role"` // MEMBER = แถว Member
	UserType string        `json:"user_type"`
	Username string        `json:"username"`
	Name     string        `json:"name"`
	Phone    string        `json:"phone"`
	Status   string        `json:"status"` // สถานะที่ใช้งานจริง (ACC-30)
	PT       any           `json:"pt,omitempty"`
	Balances []BalanceView `json:"balances"`
}

// AgentDetailResponse — GET /manage/agents/:id (MGMT-29)
type AgentDetailResponse struct {
	ID             uint            `json:"id"`
	Role           string          `json:"role"`
	UserType       string          `json:"user_type"`
	Username       string          `json:"username"`
	Name           string          `json:"name"`
	Phone          string          `json:"phone"`
	Status         string          `json:"status"`
	ParentUsername string          `json:"parent_username"`
	Currencies     []string        `json:"currencies"`
	Balances       []BalanceView   `json:"balances"`
	PT             any             `json:"pt,omitempty"`
	StatusGame     map[string]bool `json:"status_game"`
	PasscodeSet    bool            `json:"passcode_set"`
	LastLoginAt    string          `json:"last_login_at"`
	LastLoginIP    string          `json:"last_login_ip"`
	CreatedAt      string          `json:"created_at"`
}

// CopySource — 1 รายการของ GET /manage/agents/copy-sources (MGMT-35)
type CopySource struct {
	ID         uint                   `json:"id"`
	Username   string                 `json:"username"`
	UserType   string                 `json:"user_type"`
	PT         map[string]PTGroupView `json:"pt"`
	StatusGame map[string]bool        `json:"status_game"`
}

// DownlinesRequest — POST /manage/downlines/list (MGMT-26, MGMT-27) · ทุกค่าไม่บังคับ — ไม่กรองให้ส่ง {}
// page / limit อยู่ใน body ปรับด้วย utils.NewPage (ค่าเริ่มต้น 20 · สูงสุด 100)
type DownlinesRequest struct {
	ParentID uint   `json:"parent_id"` // 0 = ตัวเอง
	Q        string `json:"q"`
	Page     int    `json:"page"`
	Limit    int    `json:"limit"`
}

func (q *DownlinesRequest) Validate() error {
	q.Q = strings.ToLower(strings.TrimSpace(q.Q))
	if len(q.Q) > 32 {
		return apperr.ErrValidation.WithMessage("q ยาวเกิน 32 ตัวอักษร", "q must not exceed 32 characters")
	}
	return nil
}

// DetailRequest — POST /manage/agents/detail · /manage/members/detail (MGMT-29) · ผู้ดูมาจาก token
type DetailRequest struct {
	ID uint `json:"id"`
}

func (r *DetailRequest) Validate() error { return CheckID(r.ID) }

// DownlineSearchRequest — POST /manage/downlines/search (MGMT-27A) · ค้นทุกชั้นใต้ตัวเอง
type DownlineSearchRequest struct {
	Q     string `json:"q"` // บังคับ 2–32 ตัว
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
}

func (r *DownlineSearchRequest) Validate() error {
	r.Q = strings.ToLower(strings.TrimSpace(r.Q))
	if n := len([]rune(r.Q)); n < 2 || n > 32 {
		return apperr.ErrValidation.WithMessage("q ต้องยาว 2–32 ตัว", "q must be 2–32 characters")
	}
	return nil
}

// DownlineSearchRow — 1 แถวของผลค้นทั้งสาย = แถวรายชื่อ + ผู้สร้างตรง (MGMT-27A)
type DownlineSearchRow struct {
	DownlineRow
	ParentUsername string `json:"parent_username"`
}
