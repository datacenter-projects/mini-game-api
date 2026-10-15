package agentauth

import (
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
	if v == agentAuthCore.PasswordOK {
		return nil
	}
	th, en := agentAuthCore.PasswordViolationMessage(v)
	return apperr.ErrValidation.WithMessage(field+" "+th, field+" "+en)
}
