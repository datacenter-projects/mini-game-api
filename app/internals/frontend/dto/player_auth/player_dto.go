// Package playerauth คือ request / response ของเส้น /player/* — spec: docs/modules/player_auth.md หัวข้อ 5
// ทุกเส้น POST ที่รับ field ใช้ utils.ParseBodyNoNull (ห้าม null) + Validate() · เส้นที่ไม่รับ field (me/get · balance/get · games/list · logout) ไม่ parse body
package playerauth

import (
	"strings"
	"time"

	agentAuthCore "app/app/core/agent_auth"
	"app/pkg/apperr"
)

// maxPasswordBytes — bcrypt รับได้สูงสุด 72 byte
const maxPasswordBytes = 72

// MemberView — ข้อมูลผู้เล่นใน PA1 / PA5 (ไม่มี credit / id — lead รอบ 3)
type MemberView struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
	Status   string `json:"status"` // สถานะที่ใช้งานจริง ACTIVE / SUSPENDED (LOCKED login ไม่ผ่าน)
	CanBet   bool   `json:"can_bet"`
}

// LoginRequest — PA1 POST /api/v1/player/pb/login
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (r *LoginRequest) Validate() error {
	if strings.TrimSpace(r.Username) == "" {
		return apperr.ErrValidation.WithMessage("กรุณากรอก username", "username is required")
	}
	if len(r.Username) > 100 {
		return apperr.ErrValidation.WithMessage("username ยาวเกิน 100 ตัวอักษร", "username must not exceed 100 characters")
	}
	if r.Password == "" {
		return apperr.ErrValidation.WithMessage("กรุณากรอก password", "password is required")
	}
	if len(r.Password) > maxPasswordBytes {
		return apperr.ErrValidation.WithMessage("password ยาวเกิน 72 ตัวอักษร", "password must not exceed 72 characters")
	}
	return nil
}

// LoginResponse — PA1 · ExpiresAt = absolute (login + 4 ชม.)
type LoginResponse struct {
	Token     string     `json:"token"`
	ExpiresAt time.Time  `json:"expires_at"`
	Member    MemberView `json:"member"`
}

// PasswordUpdateRequest — PA4 POST /api/v1/player/pr/auth/password/update
type PasswordUpdateRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (r *PasswordUpdateRequest) Validate() error {
	if r.OldPassword == "" {
		return apperr.ErrValidation.WithMessage("กรุณากรอก old_password", "old_password is required")
	}
	if len(r.OldPassword) > maxPasswordBytes {
		return apperr.ErrValidation.WithMessage("old_password ยาวเกิน 72 ตัวอักษร", "old_password must not exceed 72 characters")
	}
	if r.NewPassword == "" {
		return apperr.ErrValidation.WithMessage("กรุณากรอก new_password", "new_password is required")
	}
	return PasswordPolicyError("new_password", r.NewPassword)
}

// PasswordPolicyError — ผลตรวจ AUTH-36 เป็น 422 ที่บอก field (ใช้ฟังก์ชันตรวจตัวเดียวกับ BO ใน app/core) · ผ่าน = nil
func PasswordPolicyError(field, password string) error {
	v := agentAuthCore.CheckPasswordPolicy(password)
	if v == agentAuthCore.PasswordOK {
		return nil
	}
	th, en := agentAuthCore.PasswordViolationMessage(v)
	return apperr.ErrValidation.WithMessage(field+" "+th, field+" "+en)
}

// BalanceResponse — PA6 POST /api/v1/player/pr/balance/get · credit ส่งค่าที่เก็บตามจริง ไม่ปัดเพิ่ม (กฎข้อ 9)
type BalanceResponse struct {
	Currency string  `json:"currency"`
	Credit   float64 `json:"credit"`
}

// GameItem — 1 เกมใน PA7 · ไม่มีค่า rate
type GameItem struct {
	GameCode  string `json:"game_code"`
	NameTH    string `json:"name_th"`
	NameEN    string `json:"name_en"`
	Open      bool   `json:"open"`
	GameCover string `json:"game_cover"` // "" = ยังไม่มีปก
}

// GamesListResponse — PA7 POST /api/v1/player/pr/games/list
type GamesListResponse struct {
	Items []GameItem `json:"items"`
}
