package account

import agentManagementDto "app/app/internals/backoffice/dto/agent_management"

// ProfileResponse — GET /api/v1/bo/pr/account/profile (docs/modules/account.md ACC-12)
// ไม่มี null (ACC-32): ข้อความว่าง = "" · รายการว่าง = [] · object ว่าง = {}
// รูปแบบ pt / balances ใช้ของ module ② ชุดเดียวกับรายชื่อและรายละเอียดบัญชี
type ProfileResponse struct {
	Username      string `json:"username"`
	Role          string `json:"role"`      // sub = role ของผู้สร้าง
	UserType      string `json:"user_type"` // sub = ของผู้สร้าง (ACC-15)
	Status        string `json:"status"`    // สถานะที่ใช้งานจริง (ACC-30)
	IsSubaccount  bool   `json:"is_subaccount"`
	OwnerUsername string `json:"owner_username"` // "" เมื่อไม่ใช่ sub
	PasscodeSet   bool   `json:"passcode_set"`
	LastLoginAt   string `json:"last_login_at"` // "" ถ้ายังไม่เคยบันทึก
	LastLoginIP   string `json:"last_login_ip"`
	CreatedAt     string `json:"created_at"`

	Currencies  []string                                  `json:"currencies"`
	Balances    []agentManagementDto.BalanceView          `json:"balances"`    // ACC-19
	PT          map[string]agentManagementDto.PTGroupView `json:"pt"`          // กลุ่ม PT → ค่าชุดเดียว (ACC-16)
	StatusGame  map[string]bool                           `json:"status_game"` // รหัสเกม → เปิด / ปิด
	Permissions map[string]string                         `json:"permissions"` // เมนู → off / view / edit
}
