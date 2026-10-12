// Package agentmanagement คือ request / response ของ module ② — spec: docs/modules/agent_management.md หัวข้อ 5
// ไม่มี null ใน API (account ACC-32): controller ใช้ utils.ParseBodyNoNull · ข้อความว่าง = "" · ตัวเลขว่าง = 0
package agentmanagement

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	agentAuthCore "app/app/core/agent_auth"
	agentManagementCore "app/app/core/agent_management"
	agentAuthDto "app/app/internals/backoffice/dto/agent_auth"
	"app/pkg/apperr"
	"app/pkg/utils"
)

// ChildPTRequest — ค่าที่ผู้สร้างตั้งให้ลูก 1 กลุ่ม (MGMT-16) · ครบ 5 ค่า หรือแค่ commission_percent (Share Master ใต้ CSM — MGMT-19)
// % รับเป็น float64 (ทีมตกลง 2026-10-12) · pointer เพื่อแยก "ไม่ส่ง" ออกจาก 0
type ChildPTRequest struct {
	PTFromParent      *float64 `json:"pt_from_parent"`
	Force             *float64 `json:"force"`
	RemainQuota       *float64 `json:"remain_quota"`
	CommissionPercent *float64 `json:"commission_percent"`
	Status            *bool    `json:"status"`

	// CommissionOnly — ส่งมาแค่ commission_percent · ใช้ได้เฉพาะ CSM ตั้งให้ Share Master (service ตัดสิน — 7.1 ข้อ 4)
	CommissionOnly bool `json:"-"`
	// Missing — field แรกที่ไม่ได้ส่ง (ไม่ใช่ CommissionOnly) · service คืน error นี้เมื่อลูกไม่ใช่ Share Master ใต้ CSM
	// (ถ้าเป็น Share Master ต้องตอบว่าส่งได้แค่ commission_percent แทน "ต้องส่ง" — MGMT-19)
	Missing error `json:"-"`
}

// CreateAgentRequest — POST /api/v1/bo/pr/manage/agents/create
type CreateAgentRequest struct {
	RequestID        string                    `json:"request_id"`
	UserType         string                    `json:"user_type"`
	Username         string                    `json:"username"` // Validate แปลงเป็นตัวเล็ก (MGMT-05)
	Password         string                    `json:"password"`
	Name             string                    `json:"name"`
	PhoneCountryCode string                    `json:"phone_country_code"` // MGMT-08 · ไม่กรอก = "" คู่กับ phone
	Phone            string                    `json:"phone"`
	Currencies       []string                  `json:"currencies"`
	Balance          map[string]float64        `json:"balance"`
	PT               map[string]ChildPTRequest `json:"pt"`
	StatusGame       map[string]bool           `json:"status_game"`

	BalanceAmounts map[string]float64 `json:"-"` // ยอดต่อสกุล (ปัด 4 ตำแหน่ง) หลัง Validate
}

type CreateAgentResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	UserType string `json:"user_type"`
}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Invalid — 422 ที่ขึ้นต้นด้วยชื่อ field · path มีจุด (เช่น pt.minigame.force) แปลงเป็นชื่อที่ผู้ใช้อ่านได้ (utils.FieldLabel)
func Invalid(field, th, en string) error {
	labelTH, labelEN := utils.FieldLabel(field)
	return apperr.ErrValidation.WithMessage(labelTH+" "+th, labelEN+" "+en)
}

// ValidateIdentityFields — request_id · username · password · name (MGMT-05 – MGMT-07) ตามลำดับ field ใน body
func ValidateIdentityFields(requestID string, username *string, password, name string) error {
	if !uuidRe.MatchString(requestID) {
		return Invalid("request_id", "ต้องเป็น UUID", "must be a UUID")
	}
	*username = agentManagementCore.NormalizeUsername(*username)
	if !agentManagementCore.IsValidUsername(*username) {
		return Invalid("username", "ต้องยาว 3–32 ตัว ใช้ได้เฉพาะ a-z และ 0-9", "must be 3–32 characters of a-z and 0-9")
	}
	if password == "" {
		return Invalid("password", "ต้องกรอก", "is required")
	}
	if err := agentAuthDto.PasswordPolicyError("password", agentAuthCore.CheckPasswordPolicy(password)); err != nil {
		return err
	}
	if !agentManagementCore.IsValidName(name) {
		return Invalid("name", "ต้องยาว 3–32 ตัวอักษร ใช้ได้เฉพาะภาษาไทย อังกฤษ และตัวเลข ไม่มีช่องว่าง", "must be 3–32 characters of Thai, English letters or digits, without spaces")
	}
	return nil
}

// ValidatePhonePair — เบอร์ 2 field (MGMT-08 · lead E1-5) · ตัดช่องว่าง · prefix = path ของ section เช่น "info."
func ValidatePhonePair(prefix string, code, phone *string) error {
	*code, *phone = strings.TrimSpace(*code), strings.TrimSpace(*phone)
	switch agentManagementCore.CheckPhone(*code, *phone) {
	case agentManagementCore.PhoneNotPaired:
		return Invalid(prefix+"phone_country_code / "+prefix+"phone", `ต้องส่งคู่กัน (ไม่กรอกให้ส่ง "" ทั้งคู่)`, `must be sent together (send "" for both to leave empty)`)
	case agentManagementCore.PhoneCodeInvalid:
		return Invalid(prefix+"phone_country_code", "ต้องเป็นรหัสโทรออกของประเทศที่มีจริง ตัวเลข 1–3 หลัก ไม่มี + (เช่น 66)", "must be a real country calling code of 1–3 digits without + (e.g. 66)")
	case agentManagementCore.PhoneNumberInvalid:
		return Invalid(prefix+"phone", "ต้องเป็นตัวเลขล้วน ไม่มี 0 นำหน้า (เช่น 812345678)", "must be digits only without a leading 0 (e.g. 812345678)")
	case agentManagementCore.PhoneTooLong:
		return Invalid(prefix+"phone", fmt.Sprintf("รหัสประเทศ + เบอร์ รวมไม่เกิน %d หลัก", agentManagementCore.PhoneMaxDigits),
			fmt.Sprintf("country code + phone must not exceed %d digits", agentManagementCore.PhoneMaxDigits))
	}
	return nil
}

// ParseBalance — ยอดเงินตั้งต้นต่อสกุล (MGMT-15A) · มากกว่า 0 · ทศนิยมไม่เกิน 4 ตำแหน่ง (กฎข้อ 9)
func ParseBalance(in map[string]float64) (map[string]float64, error) {
	out := make(map[string]float64, len(in))
	for _, cur := range SortedKeys(in) {
		v := in[cur]
		field := "balance." + cur
		if !agentManagementCore.IsSupportedCurrency(cur) {
			return nil, Invalid(field, "สกุลเงินไม่ถูกต้อง", "is not a supported currency")
		}
		if !utils.IsDecimal4(v) {
			return nil, Invalid(field, "ต้องเป็นตัวเลข ทศนิยมไม่เกิน 4 ตำแหน่ง", "must be a number with at most 4 decimals")
		}
		if v <= 0 {
			return nil, Invalid(field, "ต้องมากกว่า 0", "must be greater than 0")
		}
		out[cur] = v
	}
	return out, nil
}

// Percent — ค่า % ใน API (MGMT-17) · ต้องส่ง ต้องเป็นตัวเลข ทศนิยมไม่เกิน 4 ตำแหน่ง · 0 – 100
func Percent(field string, d *float64) (float64, error) {
	if d == nil {
		return 0, Invalid(field, "ต้องส่ง", "is required")
	}
	v := *d
	if !utils.IsDecimal4(v) {
		return 0, Invalid(field, "ต้องเป็นตัวเลข ทศนิยมไม่เกิน 4 ตำแหน่ง", "must be a number with at most 4 decimals")
	}
	if v < 0 || v > agentManagementCore.FullPT {
		return 0, Invalid(field, "ต้องอยู่ระหว่าง 0 ถึง 100", "must be between 0 and 100")
	}
	return v, nil
}

// CheckGroups — ต้องส่งครบทุกกลุ่มที่มี และไม่มีกลุ่มที่ไม่รู้จัก (MGMT-21)
func CheckGroups[T any](pt map[string]T) error {
	for k := range pt {
		if agentManagementCore.GamesOf(agentManagementCore.PTGroup(k)) == nil {
			return Invalid("pt."+k, "ไม่มีกลุ่มนี้", "is not a known group")
		}
	}
	for _, g := range agentManagementCore.Groups() {
		if _, ok := pt[string(g)]; !ok {
			return Invalid("pt."+string(g), "ต้องส่ง", "is required")
		}
	}
	return nil
}

func (r *CreateAgentRequest) Validate() error {
	// ไล่ตามลำดับ field ใน body: request_id → user_type → username → password → name → phone_country_code → phone → currencies → balance → pt → status_game
	if !uuidRe.MatchString(r.RequestID) {
		return Invalid("request_id", "ต้องเป็น UUID", "must be a UUID")
	}
	if r.UserType == "" {
		return Invalid("user_type", "ต้องส่ง", "is required")
	}
	if err := ValidateIdentityFields(r.RequestID, &r.Username, r.Password, r.Name); err != nil {
		return err
	}
	if err := ValidatePhonePair("", &r.PhoneCountryCode, &r.Phone); err != nil {
		return err
	}
	for i, c := range r.Currencies {
		if !agentManagementCore.IsSupportedCurrency(c) {
			return Invalid(fmt.Sprintf("currencies[%d]", i), "สกุลเงินไม่ถูกต้อง", "is not a supported currency")
		}
	}
	var err error
	if r.BalanceAmounts, err = ParseBalance(r.Balance); err != nil {
		return err
	}
	if err := CheckGroups(r.PT); err != nil {
		return err
	}
	for _, g := range SortedKeys(r.PT) {
		v := r.PT[g]
		if err := parseChildPT("pt."+g+".", &v); err != nil {
			return err
		}
		r.PT[g] = v
	}
	for _, code := range SortedKeys(r.StatusGame) {
		if _, ok := agentManagementCore.GroupOfGame(code); !ok {
			return Invalid("status_game."+code, "ไม่มีเกมนี้", "is not a known game")
		}
	}
	return nil
}

// Values — ค่าที่ตั้งให้ลูก 1 กลุ่มเป็น core.ChildPT (MGMT-16) · ใช้หลัง Validate · field ที่ไม่ส่ง = 0
// (CommissionOnly / Missing — service ตัดสินว่าใช้ได้ไหม · Share Master ใต้ CSM service แทนค่าด้วยของ CSM)
func (v ChildPTRequest) Values() agentManagementCore.ChildPT {
	return agentManagementCore.ChildPT{PTFromParent: floatOrZero(v.PTFromParent), Force: floatOrZero(v.Force),
		Remain: floatOrZero(v.RemainQuota), Commission: floatOrZero(v.CommissionPercent)}
}

func floatOrZero(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

// parseChildPT — ตรวจค่าที่ผู้สร้างตั้งให้ลูก 1 กลุ่ม (MGMT-16, MGMT-17) · อ่านค่าด้วย v.Values()
// รูปแบบผิดตอบทันที · ไม่ครบ 5 ค่าเก็บไว้ใน v.Missing ให้ service ตัดสิน (ขึ้นกับว่าลูกเป็น Share Master ใต้ CSM ไหม — MGMT-19)
func parseChildPT(prefix string, v *ChildPTRequest) error {
	if v.PTFromParent == nil && v.Force == nil && v.RemainQuota == nil && v.Status == nil && v.CommissionPercent != nil {
		v.CommissionOnly = true
		_, err := Percent(prefix+"commission_percent", v.CommissionPercent)
		return err
	}
	for _, f := range []struct {
		name string
		in   *float64
	}{
		{"pt_from_parent", v.PTFromParent},
		{"force", v.Force},
		{"remain_quota", v.RemainQuota},
		{"commission_percent", v.CommissionPercent},
	} {
		if f.in == nil {
			if v.Missing == nil {
				v.Missing = Invalid(prefix+f.name, "ต้องส่ง", "is required")
			}
			continue
		}
		if _, err := Percent(prefix+f.name, f.in); err != nil {
			return err
		}
	}
	if v.Status == nil && v.Missing == nil {
		v.Missing = Invalid(prefix+"status", "ต้องส่ง", "is required")
	}
	return nil
}

// SortedKeys — เรียง key ให้ error ออกตัวเดียวกันทุกครั้ง (map ใน Go วนไม่เรียง) · ใช้ร่วมกับ dto/member_management
func SortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
