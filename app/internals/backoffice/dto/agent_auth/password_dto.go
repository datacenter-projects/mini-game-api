package agentauth

import (
	"fmt"

	agentAuthCore "app/app/core/agent_auth"
	"app/pkg/apperr"
)

// ChangePasswordRequest — POST /api/v1/bo/pr/auth/password/change (AUTH-37, AUTH-41)
// passcode (ถ้าต้องส่ง) ตรวจที่ middleware RequirePasscodeUnlessMustChangePassword ไม่ใช่ที่นี่
type ChangePasswordRequest struct {
	OldPassword     string `json:"old_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

func (r *ChangePasswordRequest) Validate() error {
	if r.OldPassword == "" {
		return apperr.ErrValidation.WithMessage("กรุณากรอก old_password", "old_password is required")
	}
	if len(r.OldPassword) > 72 { // bcrypt รับได้สูงสุด 72 byte
		return apperr.ErrValidation.WithMessage("old_password ยาวเกิน 72 ตัวอักษร", "old_password must not exceed 72 characters")
	}
	if r.NewPassword == "" {
		return apperr.ErrValidation.WithMessage("กรุณากรอก new_password", "new_password is required")
	}
	if err := PasswordPolicyError("new_password", agentAuthCore.CheckPasswordPolicy(r.NewPassword)); err != nil {
		return err
	}
	if r.ConfirmPassword != r.NewPassword {
		return apperr.ErrValidation.WithMessage("confirm_password ไม่ตรงกับ new_password", "confirm_password does not match new_password")
	}
	return nil
}

// PasswordPolicyError แปลงผลของ CheckPasswordPolicy (AUTH-36) เป็น 422 ที่บอกว่าผิดกฎข้อไหน — nil = ผ่าน
func PasswordPolicyError(field string, v agentAuthCore.PasswordViolation) error {
	var th, en string
	switch v {
	case agentAuthCore.PasswordOK:
		return nil
	case agentAuthCore.PasswordInvalidChar:
		th, en = "ใช้ได้เฉพาะ a-z, A-Z, 0-9 และอักขระพิเศษ ห้ามมีช่องว่าง", "may contain only a-z, A-Z, 0-9 and ASCII symbols, no spaces"
	case agentAuthCore.PasswordTooShort:
		th, en = fmt.Sprintf("ต้องยาวอย่างน้อย %d ตัว", agentAuthCore.PasswordMinLen), fmt.Sprintf("must be at least %d characters", agentAuthCore.PasswordMinLen)
	case agentAuthCore.PasswordTooLong:
		th, en = fmt.Sprintf("ต้องยาวไม่เกิน %d ตัว", agentAuthCore.PasswordMaxLen), fmt.Sprintf("must not exceed %d characters", agentAuthCore.PasswordMaxLen)
	case agentAuthCore.PasswordNoLetter:
		th, en = "ต้องมีตัวอักษรอย่างน้อย 1 ตัว", "must contain at least 1 letter"
	case agentAuthCore.PasswordNoDigit:
		th, en = "ต้องมีตัวเลขอย่างน้อย 1 ตัว", "must contain at least 1 digit"
	case agentAuthCore.PasswordRepeated:
		th, en = fmt.Sprintf("ห้ามมีตัวเดียวกันซ้ำติดกัน %d ตัวขึ้นไป", agentAuthCore.PasswordMaxRepeat+1), fmt.Sprintf("must not repeat the same character %d or more times in a row", agentAuthCore.PasswordMaxRepeat+1)
	default:
		th, en = "รูปแบบไม่ถูกต้อง", "is invalid"
	}
	return apperr.ErrValidation.WithMessage(field+" "+th, field+" "+en)
}
