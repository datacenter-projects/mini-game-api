package adminmanagement

import (
	"strings"
	"time"

	"app/pkg/apperr"
)

// ResetCredentialRequest — POST /api/v1/bo/pr/admin/{passcode,password}/reset (AUTH-45)
// passcode ของ admin ตรวจที่ middleware RequirePasscode ไม่ใช่ที่นี่
type ResetCredentialRequest struct {
	Username string `json:"username"`
}

func (r *ResetCredentialRequest) Validate() error {
	if strings.TrimSpace(r.Username) == "" {
		return apperr.ErrValidation.WithMessage("กรุณากรอก username", "Username is required")
	}
	if len(r.Username) > 100 {
		return apperr.ErrValidation.WithMessage("username ยาวเกิน 100 ตัวอักษร", "Username must not exceed 100 characters")
	}
	return nil
}

// ResetPasscodeResponse — แสดงครั้งเดียว (AUTH-46)
type ResetPasscodeResponse struct {
	Username      string    `json:"username"`
	TempPasscode  string    `json:"temp_passcode"`
	TempExpiresAt time.Time `json:"temp_expires_at"`
}

// ResetPasswordResponse — แสดงครั้งเดียว (AUTH-46)
type ResetPasswordResponse struct {
	Username      string    `json:"username"`
	TempPassword  string    `json:"temp_password"`
	TempExpiresAt time.Time `json:"temp_expires_at"`
}
