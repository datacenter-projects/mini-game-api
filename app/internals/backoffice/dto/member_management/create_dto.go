// Package membermanagement คือ request / response ของเส้น /manage/members/* — spec: docs/modules/member_management.md หัวข้อ 3
// ไม่มี null ใน API (account ACC-32): controller ใช้ utils.ParseBodyNoNull · ข้อความว่าง = "" · ตัวเลขว่าง = 0
// helper ตรวจค่าใช้ร่วมกับ agent จาก dto/agent_management
package membermanagement

import (
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/app/models"
)

// MemberPTRequest — ผู้สร้างตั้งให้ Member ต่อกลุ่ม: pt (ถือสู้กับ Member คนนี้) · commission_percent (MGMT-21 · lead MA1 / MA3)
// remain_quota ระบบคิดเอง · field อื่นของฝั่ง agent = 422 · ใช้ทั้ง members/create และ members/detail/update
// % รับเป็น float64 (ทีมตกลง 2026-10-12) · pointer เพื่อแยก "ไม่ส่ง" (pt_from_parent / force / remain_quota / status มีไว้ตอบ 422)
type MemberPTRequest struct {
	OwnPT             *float64 `json:"pt"`
	CommissionPercent *float64 `json:"commission_percent"`
	PTFromParent      *float64 `json:"pt_from_parent"`
	Force             *float64 `json:"force"`
	RemainQuota       *float64 `json:"remain_quota"`
	Status            *bool    `json:"status"`

	PT         float64 `json:"-"` // % หลัง Validate
	Commission float64 `json:"-"`
}

// parseMemberPT — ตรวจ 1 กลุ่ม: field ที่ Member ไม่มี → pt → commission_percent
func parseMemberPT(group string, v *MemberPTRequest) error {
	p := "pt." + group
	if v.PTFromParent != nil || v.Force != nil || v.RemainQuota != nil || v.Status != nil {
		return agentManagementDto.Invalid(p, "Member มีแค่ pt และ commission_percent (remain_quota ระบบคิดให้)",
			"a Member only has pt and commission_percent (remain_quota is computed)")
	}
	var err error
	if v.PT, err = agentManagementDto.Percent(p+".pt", v.OwnPT); err != nil {
		return err
	}
	if v.Commission, err = agentManagementDto.Percent(p+".commission_percent", v.CommissionPercent); err != nil {
		return err
	}
	return nil
}

// parseMemberPTGroups — ตรวจทุกกลุ่มเรียงตามชื่อกลุ่ม (error ออกตัวเดียวกันทุกครั้ง — หัวข้อ 2.1)
func parseMemberPTGroups(pt map[string]MemberPTRequest) error {
	for _, g := range agentManagementDto.SortedKeys(pt) {
		v := pt[g]
		if err := parseMemberPT(g, &v); err != nil {
			return err
		}
		pt[g] = v
	}
	return nil
}

// CreateMemberRequest — POST /api/v1/bo/pr/manage/members/create (M1)
type CreateMemberRequest struct {
	RequestID        string                     `json:"request_id"`
	Username         string                     `json:"username"` // Validate แปลงเป็นตัวเล็ก (MGMT-05)
	Password         string                     `json:"password"`
	Name             string                     `json:"name"`
	PhoneCountryCode string                     `json:"phone_country_code"` // MGMT-08 · ไม่กรอก = "" คู่กับ phone
	Phone            string                     `json:"phone"`
	Balance          map[string]float64         `json:"balance"`
	PT               map[string]MemberPTRequest `json:"pt"`
	PaymentType      string                     `json:"payment_type"`      // MGMT-63 บังคับ
	PaymentAccount   *PaymentAccount            `json:"payment_account"`   // MGMT-64 ไม่บังคับ
	WinLimitPerDay   float64                    `json:"win_limit_per_day"` // MGMT-65 ไม่บังคับ (ไม่ส่ง = 0 ไม่จำกัด)

	BalanceAmounts map[string]float64          `json:"-"` // ยอดต่อสกุล (ปัด 4 ตำแหน่ง) หลัง Validate
	Payment        models.PaymentType          `json:"-"` // หลัง Validate (ก่อนปรับตาม Seamless)
	Account        models.MemberPaymentAccount `json:"-"` // หลัง Validate · ว่างทั้งหมด = ไม่มีบัญชี
}

type CreateMemberResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

// Validate — ไล่ตามลำดับ field ใน body: request_id → username → password → name → phone_country_code → phone → balance → pt
// → payment_type → payment_account → win_limit_per_day (หัวข้อ 2.1)
func (r *CreateMemberRequest) Validate() error {
	if err := agentManagementDto.ValidateIdentityFields(r.RequestID, &r.Username, r.Password, r.Name); err != nil {
		return err
	}
	if err := agentManagementDto.ValidatePhonePair("", &r.PhoneCountryCode, &r.Phone); err != nil {
		return err
	}
	var err error
	if r.BalanceAmounts, err = agentManagementDto.ParseBalance(r.Balance); err != nil {
		return err
	}
	if err := agentManagementDto.CheckGroups(r.PT); err != nil {
		return err
	}
	if err := parseMemberPTGroups(r.PT); err != nil {
		return err
	}
	if r.Payment, err = parsePaymentType("payment_type", r.PaymentType); err != nil {
		return err
	}
	if r.Account, err = parsePaymentAccount("payment_account.", r.PaymentAccount); err != nil {
		return err
	}
	return checkWinLimit("win_limit_per_day", r.WinLimitPerDay)
}
