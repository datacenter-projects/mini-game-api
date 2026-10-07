// Package agentmanagement คือ request / response ของ module ② — spec: docs/modules/agent_management.md หัวข้อ 5
// ไม่มี null ใน API (account ACC-32): controller ใช้ utils.ParseBodyNoNull · ข้อความว่าง = "" · ตัวเลขว่าง = 0
package agentmanagement

import (
	"fmt"
	"regexp"
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

// MemberPTRequest — Member มีแค่ commission_percent ต่อกลุ่ม (MGMT-21)
type MemberPTRequest struct {
	CommissionPercent utils.Decimal `json:"commission_percent"`

	CommissionBP int `json:"-"` // bp หลัง Validate
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

// CreateMemberRequest — POST /api/v1/bo/pr/manage/members/create
type CreateMemberRequest struct {
	RequestID string                     `json:"request_id"`
	Username  string                     `json:"username"`
	Password  string                     `json:"password"`
	Name      string                     `json:"name"`
	Phone     string                     `json:"phone"`
	Balance   map[string]utils.Decimal   `json:"balance"`
	PT        map[string]MemberPTRequest `json:"pt"`

	BalanceMinor map[string]int64 `json:"-"`
}

type CreateAgentResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	UserType string `json:"user_type"`
}

type CreateMemberResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func invalid(field, th, en string) error {
	return apperr.ErrValidation.WithMessage(field+" "+th, field+" "+en)
}

// validateAccountFields — ข้อมูลร่วมของเส้นสร้าง agent / Member (MGMT-05 – MGMT-08, MGMT-15A)
func validateAccountFields(requestID string, username *string, password, name string, phone *string) error {
	if !uuidRe.MatchString(requestID) {
		return invalid("request_id", "ต้องเป็น UUID", "must be a UUID")
	}
	*username = agentManagementCore.NormalizeUsername(*username)
	if !agentManagementCore.IsValidUsername(*username) {
		return invalid("username", "ต้องยาว 3–32 ตัว ใช้ได้เฉพาะ a-z และ 0-9", "must be 3–32 characters of a-z and 0-9")
	}
	if password == "" {
		return invalid("password", "ต้องกรอก", "is required")
	}
	if err := agentAuthDto.PasswordPolicyError("password", agentAuthCore.CheckPasswordPolicy(password)); err != nil {
		return err
	}
	if !agentManagementCore.IsValidName(name) {
		return invalid("name", "ต้องยาว 3–32 ตัวอักษร ใช้ได้เฉพาะภาษาไทย อังกฤษ และตัวเลข ไม่มีช่องว่าง", "must be 3–32 characters of Thai, English letters or digits, without spaces")
	}
	*phone = strings.TrimSpace(*phone)
	if !agentManagementCore.IsValidPhone(*phone) {
		return invalid("phone", "ต้องเป็นตัวเลข 8–15 ตัว (ไม่กรอกให้ส่ง \"\")", `must be 8–15 digits (send "" to leave it empty)`)
	}
	return nil
}

// parseBalance — ยอดเงินตั้งต้นต่อสกุล (MGMT-15A) · มากกว่า 0 · ทศนิยมไม่เกิน 2 ตำแหน่ง
func parseBalance(in map[string]utils.Decimal) (map[string]int64, error) {
	out := make(map[string]int64, len(in))
	for cur, d := range in {
		field := "balance." + cur
		if !agentManagementCore.IsSupportedCurrency(cur) {
			return nil, invalid(field, "สกุลเงินไม่ถูกต้อง", "is not a supported currency")
		}
		v, ok := d.Fixed2()
		if !ok {
			return nil, invalid(field, "ต้องเป็นตัวเลข ทศนิยมไม่เกิน 2 ตำแหน่ง", "must be a number with at most 2 decimals")
		}
		if v <= 0 {
			return nil, invalid(field, "ต้องมากกว่า 0", "must be greater than 0")
		}
		out[cur] = v
	}
	return out, nil
}

// percentBP — ค่า % ใน API → bp (MGMT-17) · ต้องส่ง ต้องเป็นตัวเลข ทศนิยมไม่เกิน 2 ตำแหน่ง
func percentBP(field string, d utils.Decimal) (int, error) {
	if !d.Present {
		return 0, invalid(field, "ต้องส่ง", "is required")
	}
	v, ok := d.Fixed2()
	if !ok {
		return 0, invalid(field, "ต้องเป็นตัวเลข ทศนิยมไม่เกิน 2 ตำแหน่ง", "must be a number with at most 2 decimals")
	}
	if v < 0 || v > agentManagementCore.FullPTBP {
		return 0, invalid(field, "ต้องอยู่ระหว่าง 0 ถึง 100", "must be between 0 and 100")
	}
	return int(v), nil
}

// checkGroups — ต้องส่งครบทุกกลุ่มที่มี และไม่มีกลุ่มที่ไม่รู้จัก (MGMT-21)
func checkGroups[T any](pt map[string]T) error {
	for k := range pt {
		if agentManagementCore.GamesOf(agentManagementCore.PTGroup(k)) == nil {
			return invalid("pt."+k, "ไม่มีกลุ่มนี้", "is not a known group")
		}
	}
	for _, g := range agentManagementCore.Groups() {
		if _, ok := pt[string(g)]; !ok {
			return invalid("pt."+string(g), "ต้องส่ง", "is required")
		}
	}
	return nil
}

func (r *CreateAgentRequest) Validate() error {
	if err := validateAccountFields(r.RequestID, &r.Username, r.Password, r.Name, &r.Phone); err != nil {
		return err
	}
	if r.UserType == "" {
		return invalid("user_type", "ต้องส่ง", "is required")
	}
	for i, c := range r.Currencies {
		if !agentManagementCore.IsSupportedCurrency(c) {
			return invalid(fmt.Sprintf("currencies[%d]", i), "สกุลเงินไม่ถูกต้อง", "is not a supported currency")
		}
	}
	var err error
	if r.BalanceMinor, err = parseBalance(r.Balance); err != nil {
		return err
	}
	if err := checkGroups(r.PT); err != nil {
		return err
	}
	for g, v := range r.PT {
		if err := parseChildPT("pt."+g+".", &v); err != nil {
			return err
		}
		r.PT[g] = v
	}
	for code := range r.StatusGame {
		if _, ok := agentManagementCore.GroupOfGame(code); !ok {
			return invalid("status_game."+code, "ไม่มีเกมนี้", "is not a known game")
		}
	}
	return nil
}

func (r *CreateMemberRequest) Validate() error {
	if err := validateAccountFields(r.RequestID, &r.Username, r.Password, r.Name, &r.Phone); err != nil {
		return err
	}
	var err error
	if r.BalanceMinor, err = parseBalance(r.Balance); err != nil {
		return err
	}
	if err := checkGroups(r.PT); err != nil {
		return err
	}
	for g, v := range r.PT {
		if v.CommissionBP, err = percentBP("pt."+g+".commission_percent", v.CommissionPercent); err != nil {
			return err
		}
		r.PT[g] = v
	}
	return nil
}

// parseChildPT — ค่าที่ผู้สร้างตั้งให้ลูก 1 กลุ่ม ต้องครบ 5 ค่า → bp ใน v.Parsed (MGMT-16, MGMT-17)
func parseChildPT(prefix string, v *ChildPTRequest) error {
	var err error
	if v.Parsed.PTFromParentBP, err = percentBP(prefix+"pt_from_parent", v.PTFromParent); err != nil {
		return err
	}
	if v.Parsed.ForceBP, err = percentBP(prefix+"force", v.Force); err != nil {
		return err
	}
	if v.Parsed.RemainBP, err = percentBP(prefix+"remain_quota", v.RemainQuota); err != nil {
		return err
	}
	if v.Parsed.CommissionBP, err = percentBP(prefix+"commission_percent", v.CommissionPercent); err != nil {
		return err
	}
	if v.Status == nil {
		return invalid(prefix+"status", "ต้องส่ง", "is required")
	}
	return nil
}
