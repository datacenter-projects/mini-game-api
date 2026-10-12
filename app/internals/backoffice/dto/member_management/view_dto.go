package membermanagement

import (
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
)

// DetailRequest — POST /manage/members/detail/get (M2 · MGMT-29) · ผู้ดูมาจาก token · กฎเดียวกับ agent จึงใช้ type เดียวกัน
type DetailRequest = agentManagementDto.DetailRequest

// MemberDetailResponse — POST /manage/members/detail/get (M2 · MGMT-29) · ไม่มี status_game / passcode_set · pt แสดงเสมอ
type MemberDetailResponse struct {
	ID               uint                             `json:"id"`
	Role             string                           `json:"role"`
	UserType         string                           `json:"user_type"`
	Username         string                           `json:"username"`
	Name             string                           `json:"name"`
	PhoneCountryCode string                           `json:"phone_country_code"` // MGMT-08 · ไม่ได้กรอก = ""
	Phone            string                           `json:"phone"`
	Status           string                           `json:"status"` // สถานะที่ใช้งานจริง (MGMT-30)
	ParentUsername   string                           `json:"parent_username"`
	Currencies       []string                         `json:"currencies"`
	Balances         []agentManagementDto.BalanceView `json:"balances"` // ฝั่ง Seamless amount = 0 (MGMT-15B)
	PT               any                              `json:"pt"`
	PaymentType      string                           `json:"payment_type"`      // MGMT-63
	PaymentAccount   PaymentAccount                   `json:"payment_account"`   // MGMT-64 เลขบัญชีเต็ม · ไม่มี = ทุก field ""
	WinLimitPerDay   float64                          `json:"win_limit_per_day"` // MGMT-65 · 0 = ไม่จำกัด
	LastLoginAt      string                           `json:"last_login_at"`
	LastLoginIP      string                           `json:"last_login_ip"`
	CreatedAt        string                           `json:"created_at"`
}

// MemberListRequest — POST /manage/members/list (M5 · MGMT-35M) · keyword ไม่บังคับ 4–32 ตัว · กฎเดียวกับ agents/list จึงใช้ type เดียวกัน
type MemberListRequest = agentManagementDto.AgentListRequest

// MemberListRow — 1 รายการของ members/list (M5)
type MemberListRow struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Status   string `json:"status"` // สถานะที่ใช้งานจริง
}
