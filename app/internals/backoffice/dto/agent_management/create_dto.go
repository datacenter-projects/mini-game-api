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

// ChildPTRequest — ค่าที่ผู้สร้างตั้งให้ลูก 1 กลุ่ม (MGMT-16) · ทุกค่าบังคับ
type ChildPTRequest struct {
	PTFromParent      utils.Decimal `json:"pt_from_parent"`
	Force             utils.Decimal `json:"force"`
	RemainQuota       utils.Decimal `json:"remain_quota"`
	CommissionPercent utils.Decimal `json:"commission_percent"`
	Status            *bool         `json:"status"`

	Parsed agentManagementCore.ChildPT `json:"-"` // bp หลัง Validate
}

// CreateAgentRequest — POST /api/v1/bo/pr/manage/agents/create
type CreateAgentRequest struct {
	RequestID  string                    `json:"request_id"`
	UserType   string                    `json:"user_type"`
	Username   string                    `json:"username"` // Validate แปลงเป็นตัวเล็ก (MGMT-05)
	Password   string                    `json:"password"`
	Name       string                    `json:"name"`
	Phone      string                    `json:"phone"`
	Currencies []string                  `json:"currencies"`
	Balance    map[string]utils.Decimal  `json:"balance"`
	PT         map[string]ChildPTRequest `json:"pt"`
	StatusGame map[string]bool           `json:"status_game"`

	BalanceMinor map[string]int64 `json:"-"` // หน่วยย่อย 1/100 หลัง Validate
}

type CreateAgentResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	UserType string `json:"user_type"`
}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func Invalid(field, th, en string) error {
	return apperr.ErrValidation.WithMessage(field+" "+th, field+" "+en)
}

// ValidateAccountFields — ข้อมูลร่วมของเส้นสร้าง agent / Member (MGMT-05 – MGMT-08, MGMT-15A)
func ValidateAccountFields(requestID string, username *string, password, name string, phone *string) error {
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
	*phone = strings.TrimSpace(*phone)
	if !agentManagementCore.IsValidPhone(*phone) {
		return Invalid("phone", "ต้องเป็นตัวเลข 8–15 ตัว (ไม่กรอกให้ส่ง \"\")", `must be 8–15 digits (send "" to leave it empty)`)
	}
	return nil
}

// ParseBalance — ยอดเงินตั้งต้นต่อสกุล (MGMT-15A) · มากกว่า 0 · ทศนิยมไม่เกิน 2 ตำแหน่ง
func ParseBalance(in map[string]utils.Decimal) (map[string]int64, error) {
	out := make(map[string]int64, len(in))
	for _, cur := range sortedKeys(in) {
		d := in[cur]
		field := "balance." + cur
		if !agentManagementCore.IsSupportedCurrency(cur) {
			return nil, Invalid(field, "สกุลเงินไม่ถูกต้อง", "is not a supported currency")
		}
		v, ok := d.Fixed2()
		if !ok {
			return nil, Invalid(field, "ต้องเป็นตัวเลข ทศนิยมไม่เกิน 2 ตำแหน่ง", "must be a number with at most 2 decimals")
		}
		if v <= 0 {
			return nil, Invalid(field, "ต้องมากกว่า 0", "must be greater than 0")
		}
		out[cur] = v
	}
	return out, nil
}

// PercentBP — ค่า % ใน API → bp (MGMT-17) · ต้องส่ง ต้องเป็นตัวเลข ทศนิยมไม่เกิน 2 ตำแหน่ง
func PercentBP(field string, d utils.Decimal) (int, error) {
	if !d.Present {
		return 0, Invalid(field, "ต้องส่ง", "is required")
	}
	v, ok := d.Fixed2()
	if !ok {
		return 0, Invalid(field, "ต้องเป็นตัวเลข ทศนิยมไม่เกิน 2 ตำแหน่ง", "must be a number with at most 2 decimals")
	}
	if v < 0 || v > agentManagementCore.FullPTBP {
		return 0, Invalid(field, "ต้องอยู่ระหว่าง 0 ถึง 100", "must be between 0 and 100")
	}
	return int(v), nil
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
	// ไล่ตามลำดับ field ใน body: request_id → user_type → username → password → name → phone → currencies → balance → pt → status_game
	if !uuidRe.MatchString(r.RequestID) {
		return Invalid("request_id", "ต้องเป็น UUID", "must be a UUID")
	}
	if r.UserType == "" {
		return Invalid("user_type", "ต้องส่ง", "is required")
	}
	if err := ValidateAccountFields(r.RequestID, &r.Username, r.Password, r.Name, &r.Phone); err != nil {
		return err
	}
	for i, c := range r.Currencies {
		if !agentManagementCore.IsSupportedCurrency(c) {
			return Invalid(fmt.Sprintf("currencies[%d]", i), "สกุลเงินไม่ถูกต้อง", "is not a supported currency")
		}
	}
	var err error
	if r.BalanceMinor, err = ParseBalance(r.Balance); err != nil {
		return err
	}
	if err := CheckGroups(r.PT); err != nil {
		return err
	}
	for _, g := range sortedKeys(r.PT) {
		v := r.PT[g]
		if err := parseChildPT("pt."+g+".", &v); err != nil {
			return err
		}
		r.PT[g] = v
	}
	for _, code := range sortedKeys(r.StatusGame) {
		if _, ok := agentManagementCore.GroupOfGame(code); !ok {
			return Invalid("status_game."+code, "ไม่มีเกมนี้", "is not a known game")
		}
	}
	return nil
}

// parseChildPT — ค่าที่ผู้สร้างตั้งให้ลูก 1 กลุ่ม ต้องครบ 5 ค่า → bp ใน v.Parsed (MGMT-16, MGMT-17)
func parseChildPT(prefix string, v *ChildPTRequest) error {
	var err error
	if v.Parsed.PTFromParentBP, err = PercentBP(prefix+"pt_from_parent", v.PTFromParent); err != nil {
		return err
	}
	if v.Parsed.ForceBP, err = PercentBP(prefix+"force", v.Force); err != nil {
		return err
	}
	if v.Parsed.RemainBP, err = PercentBP(prefix+"remain_quota", v.RemainQuota); err != nil {
		return err
	}
	if v.Parsed.CommissionBP, err = PercentBP(prefix+"commission_percent", v.CommissionPercent); err != nil {
		return err
	}
	if v.Status == nil {
		return Invalid(prefix+"status", "ต้องส่ง", "is required")
	}
	return nil
}

// sortedKeys — เรียง key ให้ error ออกตัวเดียวกันทุกครั้ง (map ใน Go วนไม่เรียง)
func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
