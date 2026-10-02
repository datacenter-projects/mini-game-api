package agentauth

import (
	"strings"
	"time"

	"app/pkg/apperr"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (r *LoginRequest) Validate() error {
	if strings.TrimSpace(r.Username) == "" {
		return apperr.ErrValidation.WithMessage("กรุณากรอก username", "Username is required")
	}
	if len(r.Username) > 100 {
		return apperr.ErrValidation.WithMessage("username ยาวเกิน 100 ตัวอักษร", "Username must not exceed 100 characters")
	}
	if r.Password == "" {
		return apperr.ErrValidation.WithMessage("กรุณากรอกรหัสผ่าน", "Password is required")
	}
	if len(r.Password) > 72 { // bcrypt รับได้สูงสุด 72 byte
		return apperr.ErrValidation.WithMessage("รหัสผ่านยาวเกิน 72 ตัวอักษร", "Password must not exceed 72 characters")
	}
	return nil
}

type LoginResponse struct {
	Token        string    `json:"token"`
	Username     string    `json:"username"`
	Role         string    `json:"role"`
	IsSubaccount bool      `json:"is_subaccount"`
	PasscodeSet  bool      `json:"passcode_set"` // false เสมอจนกว่าจะทำ module passcode (spec หัวข้อ 9 ข้อ 5)
	ExpiresAt    time.Time `json:"expires_at"`
}
