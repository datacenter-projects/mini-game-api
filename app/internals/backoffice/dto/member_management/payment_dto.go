package membermanagement

import (
	memberManagementCore "app/app/core/member_management"
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/app/models"
	"app/pkg/utils"
)

// วิธีจ่ายเงิน · บัญชีธนาคาร · ลิมิตชนะต่อวัน — spec: docs/modules/member_management.md MGMT-63 – MGMT-65 (lead review P1–P15)

// PaymentAccount — payment_account ใน request (members/create · section payment) และ response (members/detail/get)
// ไม่ส่ง key = "" · ไม่มีบัญชี = ทุก field "" (ACC-32)
type PaymentAccount struct {
	BankCode      string `json:"bank_code"`
	AccountNumber string `json:"account_number"`
	NameTH        string `json:"name_th"`
	MidnameTH     string `json:"midname_th"`
	SurnameTH     string `json:"surname_th"`
	NameEN        string `json:"name_en"`
	MidnameEN     string `json:"midname_en"`
	SurnameEN     string `json:"surname_en"`
}

// PaymentAccountView — แปลงค่าที่เก็บเป็น response (เลขบัญชีเต็ม — lead P8)
func PaymentAccountView(a models.MemberPaymentAccount) PaymentAccount {
	return PaymentAccount{BankCode: a.BankCode, AccountNumber: a.AccountNumber, NameTH: a.NameTH, MidnameTH: a.MidnameTH, SurnameTH: a.SurnameTH,
		NameEN: a.NameEN, MidnameEN: a.MidnameEN, SurnameEN: a.SurnameEN}
}

// parsePaymentType — บังคับ CREDIT / CASH (MGMT-63) · field = path ใน body
func parsePaymentType(field, s string) (models.PaymentType, error) {
	t, ok := memberManagementCore.ParsePaymentType(s)
	if !ok {
		return "", agentManagementDto.Invalid(field, "ต้องเป็น CREDIT หรือ CASH", "must be CREDIT or CASH")
	}
	return t, nil
}

// parsePaymentAccount — ตัดช่องว่าง ตรวจรูปแบบ (MGMT-64) · nil = ไม่มีบัญชี · prefix = path ของ object เช่น "payment_account."
func parsePaymentAccount(prefix string, in *PaymentAccount) (models.MemberPaymentAccount, error) {
	if in == nil {
		return models.MemberPaymentAccount{}, nil
	}
	a := memberManagementCore.NormalizePaymentAccount(models.MemberPaymentAccount{BankCode: in.BankCode, AccountNumber: in.AccountNumber,
		NameTH: in.NameTH, MidnameTH: in.MidnameTH, SurnameTH: in.SurnameTH, NameEN: in.NameEN, MidnameEN: in.MidnameEN, SurnameEN: in.SurnameEN})
	issue := memberManagementCore.CheckPaymentAccount(a)
	switch issue.Violation {
	case memberManagementCore.PaymentAccountOK:
		return a, nil
	case memberManagementCore.AccountNotPaired:
		return a, agentManagementDto.Invalid(prefix+"bank_code / "+prefix+"account_number", "ต้องกรอกคู่กัน (ไม่มีบัญชีให้เว้นว่างทั้งคู่)",
			"must be filled in together (leave both empty when there is no account)")
	case memberManagementCore.BankCodeInvalid:
		return a, agentManagementDto.Invalid(prefix+"bank_code", "ต้องเป็นรหัสธนาคารไทยในรายการ (เช่น KBANK, SCB)", "must be a Thai bank code from the list (e.g. KBANK, SCB)")
	case memberManagementCore.AccountNumberInvalid:
		return a, agentManagementDto.Invalid(prefix+"account_number", "ต้องเป็นตัวเลขล้วน 10–15 หลัก", "must be 10–15 digits")
	case memberManagementCore.NameWithoutAccount:
		th, en := utils.SubFieldLabels("payment_account", "bank_code", "account_number")
		return a, agentManagementDto.Invalid(prefix+issue.Field, "กรอกได้เมื่อมี"+th[0]+"และ"+th[1], "is allowed only with "+en[0]+" and "+en[1])
	default:
		if issue.Field == "name_th" || issue.Field == "midname_th" || issue.Field == "surname_th" {
			return a, agentManagementDto.Invalid(prefix+issue.Field, "ต้องเป็นอักษรไทย มีช่องว่างหรือ - ได้ ไม่เกิน 50 ตัว",
				"must be Thai letters (spaces or - allowed), at most 50 characters")
		}
		return a, agentManagementDto.Invalid(prefix+issue.Field, "ต้องเป็น A–Z a–z มีช่องว่างหรือ - ได้ ไม่เกิน 50 ตัว",
			"must be letters A–Z a–z (spaces or - allowed), at most 50 characters")
	}
}

// checkWinLimit — 0 ถึง 1,000,000 ทศนิยมไม่เกิน 4 ตำแหน่ง (MGMT-65 · กฎข้อ 9) · ยอดเงินรับเป็น float64 (ทีมตกลง 2026-10-12)
func checkWinLimit(field string, v float64) error {
	if v >= 0 && !memberManagementCore.WinLimitWithinMax(v) {
		maxText := utils.FormatNum(memberManagementCore.MaxWinLimitPerDay)
		return agentManagementDto.Invalid(field, "ต้องไม่เกิน "+maxText+" (0 = ไม่จำกัด)", "must not exceed "+maxText+" (0 = no limit)")
	}
	if v < 0 || !utils.IsDecimal4(v) {
		return agentManagementDto.Invalid(field, "ต้องเป็นตัวเลข ≥ 0 ทศนิยมไม่เกิน 4 ตำแหน่ง (0 = ไม่จำกัด)",
			"must be a number ≥ 0 with at most 4 decimals (0 = no limit)")
	}
	return nil
}

// MemberPaymentSection — section payment ของ members/detail/update (MGMT-32M · lead P13) · ไม่ส่ง field = คงเดิม
// payment_account ที่ส่งแทนทั้งก้อน · {} = ล้างบัญชี
type MemberPaymentSection struct {
	PaymentType    utils.JSONString `json:"payment_type"`
	PaymentAccount *PaymentAccount  `json:"payment_account"`

	Type    models.PaymentType           `json:"-"` // หลัง Validate · "" = ไม่ส่ง
	Account *models.MemberPaymentAccount `json:"-"` // หลัง Validate · nil = ไม่ส่ง
}

func (s *MemberPaymentSection) check() error {
	if !s.PaymentType.Present && s.PaymentAccount == nil {
		th, en := utils.SubFieldLabels("payment", "payment_type", "payment_account")
		return agentManagementDto.Invalid("payment", "ต้องมีอย่างน้อย 1 อย่าง ("+th[0]+" / "+th[1]+")",
			"must contain at least 1 of ("+en[0]+" / "+en[1]+")")
	}
	if s.PaymentType.Present {
		if !s.PaymentType.IsString {
			return agentManagementDto.Invalid("payment.payment_type", "ต้องส่งเป็นข้อความ", "must be a string")
		}
		t, err := parsePaymentType("payment.payment_type", s.PaymentType.Value)
		if err != nil {
			return err
		}
		s.Type = t
	}
	if s.PaymentAccount != nil {
		a, err := parsePaymentAccount("payment.payment_account.", s.PaymentAccount)
		if err != nil {
			return err
		}
		s.Account = &a
	}
	return nil
}

// MemberLimitSection — section limit ของ members/detail/update (MGMT-65 · lead P13) · win_limit_per_day บังคับใน section
// pointer เพื่อแยก "ไม่ส่ง" (422) ออกจาก 0 (ไม่จำกัด)
type MemberLimitSection struct {
	WinLimitPerDay *float64 `json:"win_limit_per_day"`
}

func (s *MemberLimitSection) check() error {
	if s.WinLimitPerDay == nil {
		return agentManagementDto.Invalid("limit.win_limit_per_day", "ต้องกรอก", "is required")
	}
	return checkWinLimit("limit.win_limit_per_day", *s.WinLimitPerDay)
}
