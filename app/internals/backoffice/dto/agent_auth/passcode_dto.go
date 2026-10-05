package agentauth

import (
	agentAuthCore "app/app/core/agent_auth"
	"app/pkg/apperr"
	"app/pkg/utils"
)

// SetupPasscodeRequest — POST /api/v1/bo/pr/auth/passcode/setup (AUTH-32)
type SetupPasscodeRequest struct {
	Passcode        utils.JSONString `json:"passcode"`
	ConfirmPasscode utils.JSONString `json:"confirm_passcode"`
}

func (r *SetupPasscodeRequest) Validate() error {
	if err := ValidatePasscodeField("passcode", r.Passcode); err != nil {
		return err
	}
	if err := ValidatePasscodeField("confirm_passcode", r.ConfirmPasscode); err != nil {
		return err
	}
	if r.Passcode.Value != r.ConfirmPasscode.Value {
		return apperr.ErrValidation.WithMessage("confirm_passcode ไม่ตรงกับ passcode", "confirm_passcode does not match passcode")
	}
	return nil
}

// ChangePasscodeRequest — POST /api/v1/bo/pr/auth/passcode/change (AUTH-34)
type ChangePasscodeRequest struct {
	OldPasscode     utils.JSONString `json:"old_passcode"`
	NewPasscode     utils.JSONString `json:"new_passcode"`
	ConfirmPasscode utils.JSONString `json:"confirm_passcode"`
}

func (r *ChangePasscodeRequest) Validate() error {
	if err := ValidatePasscodeField("old_passcode", r.OldPasscode); err != nil {
		return err
	}
	if err := ValidatePasscodeField("new_passcode", r.NewPasscode); err != nil {
		return err
	}
	if err := ValidatePasscodeField("confirm_passcode", r.ConfirmPasscode); err != nil {
		return err
	}
	if r.NewPasscode.Value != r.ConfirmPasscode.Value {
		return apperr.ErrValidation.WithMessage("confirm_passcode ไม่ตรงกับ new_passcode", "confirm_passcode does not match new_passcode")
	}
	return nil
}

// ValidatePasscodeField ตรวจรูปแบบ passcode 1 field (AUTH-30) — middleware RequirePasscode ใช้ร่วมด้วย
func ValidatePasscodeField(field string, v utils.JSONString) error {
	switch {
	case !v.Present:
		return apperr.ErrValidation.WithMessage("กรุณากรอก "+field, field+" is required")
	case !v.IsString:
		return apperr.ErrValidation.WithMessage(field+" ต้องส่งเป็นข้อความ (string)", field+" must be a string")
	case !agentAuthCore.IsValidPasscode(v.Value):
		return apperr.ErrValidation.WithMessage(field+" ต้องเป็นตัวเลข 6 หลัก", field+" must be exactly 6 digits")
	}
	return nil
}
