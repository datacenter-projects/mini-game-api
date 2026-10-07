package account

import "app/pkg/utils"

// ProfileResponse — GET /api/v1/bo/pr/account/profile (docs/modules/account.md ACC-12)
// ไม่มี null (ACC-32): ข้อความว่าง = "" · รายการว่าง = [] · object ว่าง = {}
type ProfileResponse struct {
	Username      string `json:"username"`
	Role          string `json:"role"`      // sub = role ของผู้สร้าง
	UserType      string `json:"user_type"` // Company / Share = "" จนกว่า module ② จะมี agent_type
	Status        string `json:"status"`    // สถานะที่ใช้งานจริง (ACC-30)
	IsSubaccount  bool   `json:"is_subaccount"`
	OwnerUsername string `json:"owner_username"` // "" เมื่อไม่ใช่ sub
	PasscodeSet   bool   `json:"passcode_set"`
	LastLoginAt   string `json:"last_login_at"` // "" ถ้ายังไม่เคยบันทึก
	LastLoginIP   string `json:"last_login_ip"`
	CreatedAt     string `json:"created_at"`

	// ส่วนที่ต้องใช้ตารางของ module ② — ส่งค่าว่างไปก่อน
	Currencies  []string          `json:"currencies"`
	Balances    []ProfileBalance  `json:"balances"`
	PT          map[string]any    `json:"pt"`
	StatusGame  map[string]bool   `json:"status_game"`
	Permissions map[string]string `json:"permissions"`
}

// ProfileBalance — ยอดเงินแยกสกุล (ACC-19)
type ProfileBalance struct {
	Currency string      `json:"currency"`
	Amount   utils.Money `json:"amount"` // int64 หน่วยย่อย ส่งเป็น number ทศนิยม 2 ตำแหน่ง (ACC-18)
}
