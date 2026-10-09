package membermanagement

import (
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
)

// DetailRequest — POST /manage/members/detail (MGMT-29) · ผู้ดูมาจาก token · กฎเดียวกับ agent จึงใช้ type เดียวกัน
type DetailRequest = agentManagementDto.DetailRequest

// MemberDetailResponse — POST /manage/members/detail (MGMT-29) · ไม่มี status_game / passcode_set
type MemberDetailResponse struct {
	ID             uint                             `json:"id"`
	Role           string                           `json:"role"`
	UserType       string                           `json:"user_type"`
	Username       string                           `json:"username"`
	Name           string                           `json:"name"`
	Phone          string                           `json:"phone"`
	Status         string                           `json:"status"`
	ParentUsername string                           `json:"parent_username"`
	Currencies     []string                         `json:"currencies"`
	Balances       []agentManagementDto.BalanceView `json:"balances"`
	PT             any                              `json:"pt,omitempty"`
	LastLoginAt    string                           `json:"last_login_at"`
	LastLoginIP    string                           `json:"last_login_ip"`
	CreatedAt      string                           `json:"created_at"`
}
