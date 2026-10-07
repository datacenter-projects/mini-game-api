package account

import (
	"fmt"

	accountCore "app/app/core/account"
	"app/pkg/apperr"
	"app/pkg/utils"
)

// APICredentialResponse — GET /api/v1/bo/pr/account/api-credential (docs/modules/account.md หัวข้อ 5)
// ไม่มี null (ACC-32): ยังไม่ตั้งลิงก์ = "" · ไม่มี IP = []
type APICredentialResponse struct {
	Username    string   `json:"username"` // username ของเจ้าของ Key (ACC-10)
	Key         string   `json:"key"`
	CallbackURL string   `json:"callback_url"`
	AllowedIPs  []string `json:"allowed_ips"`
}

// SaveAPICredentialRequest — POST /api/v1/bo/pr/account/api-credential (ACC-06–ACC-08)
// passcode อ่านโดย middleware RequirePasscode · บันทึกแทนทั้งชุด จึงต้องส่งครบทั้ง 2 field
type SaveAPICredentialRequest struct {
	CallbackURL utils.JSONString `json:"callback_url"`
	AllowedIPs  *[]string        `json:"allowed_ips"`

	// ค่าหลังตรวจและแปลงรูปแบบแล้ว — service ใช้ค่านี้
	NormalizedCallbackURL string   `json:"-"` // "" = ไม่ตั้ง
	NormalizedIPs         []string `json:"-"` // CIDR ของ IPv4
}

func (r *SaveAPICredentialRequest) Validate() error {
	if !r.CallbackURL.Present || !r.CallbackURL.IsString { // ไม่ส่ง หรือส่ง null = 422 (ACC-32)
		return apperr.ErrValidation.WithMessage(`ต้องส่ง callback_url เป็นข้อความ (ไม่ตั้งให้ส่ง "" · ห้าม null)`, `callback_url must be a string (send "" to leave it unset, null is not allowed)`)
	}
	url, v := accountCore.NormalizeCallbackURL(r.CallbackURL.Value)
	switch v {
	case accountCore.CallbackTooLong:
		return apperr.ErrValidation.WithMessage(
			fmt.Sprintf("callback_url ยาวเกิน %d ตัวอักษร", accountCore.CallbackURLMaxLen),
			fmt.Sprintf("callback_url must not exceed %d characters", accountCore.CallbackURLMaxLen))
	case accountCore.CallbackInvalidURL:
		return apperr.ErrValidation.WithMessage("callback_url ไม่ใช่ URL ที่ถูกต้อง", "callback_url is not a valid URL")
	case accountCore.CallbackNotHTTPS:
		return apperr.ErrValidation.WithMessage("callback_url ต้องขึ้นต้นด้วย https://", "callback_url must start with https://")
	}

	if r.AllowedIPs == nil {
		return apperr.ErrValidation.WithMessage("ต้องส่ง allowed_ips เป็นรายการ (ไม่มีให้ส่ง [] · ห้าม null)", "allowed_ips must be a list (send [] for none, null is not allowed)")
	}
	ips, iv, idx := accountCore.NormalizeAllowedIPs(*r.AllowedIPs)
	field := fmt.Sprintf("allowed_ips[%d]", idx)
	switch iv {
	case accountCore.IPTooMany:
		return apperr.ErrValidation.WithMessage(
			fmt.Sprintf("allowed_ips เกิน %d รายการ", accountCore.AllowedIPsMax),
			fmt.Sprintf("allowed_ips must not exceed %d entries", accountCore.AllowedIPsMax))
	case accountCore.IPInvalid:
		return apperr.ErrValidation.WithMessage(field+" ต้องเป็น IPv4 หรือช่วง CIDR ของ IPv4", field+" must be an IPv4 address or IPv4 CIDR")
	case accountCore.IPHostBitsSet:
		return apperr.ErrValidation.WithMessage(field+" ต้องเป็นที่อยู่เครือข่าย เช่น 198.51.100.0/24", field+" must be a network address such as 198.51.100.0/24")
	case accountCore.IPDuplicate:
		return apperr.ErrValidation.WithMessage(field+" ซ้ำกับรายการก่อนหน้า", field+" duplicates an earlier entry")
	}

	r.NormalizedCallbackURL, r.NormalizedIPs = url, ips
	return nil
}
