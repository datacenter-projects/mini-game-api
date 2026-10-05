package account

import "time"

// ProfileResponse — GET /api/v1/bo/pr/account/profile (docs/modules/account.md ACC-12)
type ProfileResponse struct {
	Username        string     `json:"username"`
	Role            string     `json:"role"` // sub = role ของผู้สร้าง
	Status          string     `json:"status"`
	EffectiveStatus string     `json:"effective_status"`
	IsSubaccount    bool       `json:"is_subaccount"`
	OwnerUsername   *string    `json:"owner_username"` // เฉพาะ sub
	PasscodeSet     bool       `json:"passcode_set"`
	LastLoginAt     *time.Time `json:"last_login_at"`
	LastLoginIP     *string    `json:"last_login_ip"`
	CreatedAt       time.Time  `json:"created_at"`
}
