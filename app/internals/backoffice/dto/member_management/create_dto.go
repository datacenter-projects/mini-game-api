// Package membermanagement คือ request / response ของเส้น /manage/members/* — spec: docs/modules/agent_management.md หัวข้อ 5
// ไม่มี null ใน API (account ACC-32): controller ใช้ utils.ParseBodyNoNull · ข้อความว่าง = "" · ตัวเลขว่าง = 0
// helper ตรวจค่าใช้ร่วมกับ agent จาก dto/agent_management
package membermanagement

import (
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/pkg/utils"
)

// MemberPTRequest — ผู้สร้างตั้งให้ Member ต่อกลุ่ม: pt (ถือสู้กับ Member คนนี้) · commission_percent (MGMT-21 แก้ 2026-10-09)
// remain_quota ระบบคิดเอง · field อื่นของฝั่ง agent = 422 · ใช้ทั้งเส้นสร้างและ update-pt
type MemberPTRequest struct {
	OwnPT             utils.Decimal `json:"pt"`
	CommissionPercent utils.Decimal `json:"commission_percent"`
	PTFromParent      utils.Decimal `json:"pt_from_parent"`
	Force             utils.Decimal `json:"force"`
	RemainQuota       utils.Decimal `json:"remain_quota"`
	Status            *bool         `json:"status"`

	PT         float64 `json:"-"` // % หลัง Validate
	Commission float64 `json:"-"`
}

// parseMemberPT — ตรวจ 1 กลุ่ม: field ที่ Member ไม่มี → pt → commission_percent
func parseMemberPT(group string, v *MemberPTRequest) error {
	p := "pt." + group
	if v.PTFromParent.Present || v.Force.Present || v.RemainQuota.Present || v.Status != nil {
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

// CreateMemberRequest — POST /api/v1/bo/pr/manage/members/create
type CreateMemberRequest struct {
	RequestID string                     `json:"request_id"`
	Username  string                     `json:"username"`
	Password  string                     `json:"password"`
	Name      string                     `json:"name"`
	Phone     string                     `json:"phone"`
	Balance   map[string]utils.Decimal   `json:"balance"`
	PT        map[string]MemberPTRequest `json:"pt"`

	BalanceAmounts map[string]float64 `json:"-"` // ยอดต่อสกุล (ปัด 4 ตำแหน่ง) หลัง Validate
}

type CreateMemberResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

func (r *CreateMemberRequest) Validate() error {
	if err := agentManagementDto.ValidateAccountFields(r.RequestID, &r.Username, r.Password, r.Name, &r.Phone); err != nil {
		return err
	}
	var err error
	if r.BalanceAmounts, err = agentManagementDto.ParseBalance(r.Balance); err != nil {
		return err
	}
	if err := agentManagementDto.CheckGroups(r.PT); err != nil {
		return err
	}
	for g, v := range r.PT {
		if err := parseMemberPT(g, &v); err != nil {
			return err
		}
		r.PT[g] = v
	}
	return nil
}
