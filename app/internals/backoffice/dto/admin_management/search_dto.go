package adminmanagement

import (
	agentAuthCore "app/app/core/agent_auth"
	"app/pkg/apperr"
)

// AccountSearchRequest — POST /api/v1/bo/pr/admin/accounts/search (MGMT-27B) · username ตรงทั้งคำ ไม่สนตัวพิมพ์
type AccountSearchRequest struct {
	Username string `json:"username"`
}

func (r *AccountSearchRequest) Validate() error {
	r.Username = agentAuthCore.NormalizeUsername(r.Username)
	if r.Username == "" {
		return apperr.ErrValidation.WithMessage("username ต้องกรอก", "username is required")
	}
	if len([]rune(r.Username)) > 71 {
		return apperr.ErrValidation.WithMessage("username ยาวเกิน 71 ตัว", "username must be at most 71 characters")
	}
	return nil
}

// AccountRow — 1 บัญชีที่พบ (MGMT-27B) · ข้อมูลระบุตัวตนเท่านั้น ไม่มียอดเงิน / PT
// sub: role / user_type = ของเจ้าของ · parent_username = เจ้าของ · SUPERADMIN / ADMIN: parent_username = ""
type AccountRow struct {
	Username       string `json:"username"`
	Role           string `json:"role"`
	UserType       string `json:"user_type"`
	IsSubaccount   bool   `json:"is_subaccount"`
	Status         string `json:"status"`
	ParentUsername string `json:"parent_username"`
	CreatedAt      string `json:"created_at"`
	LastLoginAt    string `json:"last_login_at"`
	LastLoginIP    string `json:"last_login_ip"`
}
