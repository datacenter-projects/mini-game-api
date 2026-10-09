package agentmanagement

import (
	"strings"

	"app/pkg/apperr"
)

// รูปแบบที่ใช้แสดงบัญชี — รายชื่อดาวน์ไลน์ · รายละเอียด · agents/list · account Profile (ACC-16, ACC-19)

// PTGroupView — ค่าหุ้นส่วนฝั่ง agent ชุดเดียวต่อระบบ (MGMT-16) · เวลาเป็น RFC 3339
type PTGroupView struct {
	PTFromParent      float64 `json:"pt_from_parent"`
	Force             float64 `json:"force"`
	RemainQuota       float64 `json:"remain_quota"`
	CommissionPercent float64 `json:"commission_percent"`
	Status            bool    `json:"status"`
	CreatedAt         string  `json:"created_at"`
	CreatedBy         string  `json:"created_by"`
	UpdatedAt         string  `json:"updated_at"`
	UpdatedBy         string  `json:"updated_by"`
}

// MemberPTGroupView — pt ที่ผู้สร้างถือสู้กับ Member · remain_quota (ระบบคิด) · Commission (MGMT-21 แก้ 2026-10-09)
type MemberPTGroupView struct {
	PT                float64 `json:"pt"`
	RemainQuota       float64 `json:"remain_quota"`
	CommissionPercent float64 `json:"commission_percent"`
	CreatedAt         string  `json:"created_at"`
	CreatedBy         string  `json:"created_by"`
	UpdatedAt         string  `json:"updated_at"`
	UpdatedBy         string  `json:"updated_by"`
}

// BalanceView — ยอดแยกสกุล (account ACC-19)
type BalanceView struct {
	Currency string  `json:"currency"`
	Amount   float64 `json:"amount"`
}

// DownlineRow — 1 แถวของ POST /manage/downlines/list (MGMT-28) · pt แสดงเสมอ (ไม่มีเมนู pt แล้ว — P4)
// status_game มีเฉพาะแถวฝั่ง agent (lead Q-C1)
type DownlineRow struct {
	ID             uint            `json:"id"`
	Role           string          `json:"role"` // MEMBER = แถว Member
	UserType       string          `json:"user_type"`
	Username       string          `json:"username"`
	Name           string          `json:"name"`
	Phone          string          `json:"phone"`
	Status         string          `json:"status"`          // สถานะที่ใช้งานจริง (ACC-30)
	ParentUsername string          `json:"parent_username"` // ผู้สร้างตรง · ทุกแถว (lead E2-2)
	LastLoginAt    string          `json:"last_login_at"`   // RFC 3339 · ยังไม่เคย login = "" (เพิ่ม 2026-10-09)
	LastLoginIP    string          `json:"last_login_ip"`
	CreatedAt      string          `json:"created_at"`
	PT             any             `json:"pt"`
	StatusGame     map[string]bool `json:"status_game,omitempty"`
	Balances       []BalanceView   `json:"balances"`
}

// AgentDetailResponse — POST /manage/agents/detail/get (MGMT-29)
type AgentDetailResponse struct {
	ID                  uint                   `json:"id"`
	Role                string                 `json:"role"`
	UserType            string                 `json:"user_type"`
	Username            string                 `json:"username"`
	Name                string                 `json:"name"`
	Phone               string                 `json:"phone"`
	Status              string                 `json:"status"`
	ParentUsername      string                 `json:"parent_username"`
	Currencies          []string               `json:"currencies"`
	Balances            []BalanceView          `json:"balances"`
	PT                  map[string]PTGroupView `json:"pt"`
	StatusGame          map[string]bool        `json:"status_game"`           // ค่าที่ตั้งกับบัญชีเอง (เติมฟอร์มแก้)
	StatusGameEffective map[string]bool        `json:"status_game_effective"` // เปิดจริงหลังรวมหัวสาย (แสดงผล — lead E3-1)
	PasscodeSet         bool                   `json:"passcode_set"`
	LastLoginAt         string                 `json:"last_login_at"`
	LastLoginIP         string                 `json:"last_login_ip"`
	CreatedAt           string                 `json:"created_at"`
}

// DownlinesRequest — POST /manage/downlines/list (MGMT-26, MGMT-27 — lead E2) · ทุกค่าไม่บังคับ
// keyword ว่าง = ลูกตรงของ parent_id · ส่ง = ค้น username ทุกชั้นใต้ parent_id · page / limit ใช้ utils.NewPage
type DownlinesRequest struct {
	ParentID uint   `json:"parent_id"` // 0 = ตัวเอง
	Keyword  string `json:"keyword"`
	Page     int    `json:"page"`
	Limit    int    `json:"limit"`
}

func (q *DownlinesRequest) Validate() error {
	return NormalizeKeyword(&q.Keyword)
}

// NormalizeKeyword — keyword ค้น username: ตัดช่องว่าง ตัวเล็ก · ว่าง = ไม่ค้น · มีค่าต้องยาว 4–32 ตัว (MGMT-27 · MGMT-35)
func NormalizeKeyword(k *string) error {
	*k = strings.ToLower(strings.TrimSpace(*k))
	if n := len([]rune(*k)); *k != "" && (n < 4 || n > 32) {
		return apperr.ErrValidation.WithMessage("keyword ต้องยาว 4–32 ตัว (ไม่ค้นให้ส่ง \"\")", "keyword must be 4–32 characters (send \"\" for none)")
	}
	return nil
}

// DetailRequest — POST /manage/agents/detail/get · /manage/members/detail (MGMT-29) · ผู้ดูมาจาก token
type DetailRequest struct {
	ID uint `json:"id"`
}

func (r *DetailRequest) Validate() error { return CheckID(r.ID) }

// AgentListRequest — POST /manage/agents/list (MGMT-35 — lead E6) · keyword ไม่บังคับ
type AgentListRequest struct {
	Keyword string `json:"keyword"`
}

func (r *AgentListRequest) Validate() error { return NormalizeKeyword(&r.Keyword) }

// AgentListRow — 1 รายการของ agents/list (lead E6b)
type AgentListRow struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	UserType string `json:"user_type"`
	Status   string `json:"status"`
}
