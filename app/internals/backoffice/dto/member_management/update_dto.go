package membermanagement

import (
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/pkg/utils"
)

// request ของการแก้ Member (phase 4) — spec: docs/modules/agent_management.md หัวข้อ 5

// UpdateInfoRequest — POST /manage/members/update-info (MGMT-09A) · กฎเดียวกับ agent จึงใช้ type เดียวกัน
type UpdateInfoRequest = agentManagementDto.UpdateInfoRequest

// UpdateStatusRequest — POST /manage/members/update-status (MGMT-30) · กฎเดียวกับ agent จึงใช้ type เดียวกัน
type UpdateStatusRequest = agentManagementDto.UpdateStatusRequest

// UpdateMemberPTGroup — Member มีแค่ commission_percent (MGMT-21) · field อื่น = 422
type UpdateMemberPTGroup struct {
	CommissionPercent utils.Decimal `json:"commission_percent"`
	PTFromParent      utils.Decimal `json:"pt_from_parent"`
	OwnPT             utils.Decimal `json:"pt"`
	Force             utils.Decimal `json:"force"`
	RemainQuota       utils.Decimal `json:"remain_quota"`
	Status            *bool         `json:"status"`

	CommissionBP int `json:"-"`
}

// UpdateMemberPTRequest — POST /manage/members/update-commission
type UpdateMemberPTRequest struct {
	ID uint                           `json:"id"`
	PT map[string]UpdateMemberPTGroup `json:"pt"`
}

func (r *UpdateMemberPTRequest) Validate() error {
	if err := agentManagementDto.CheckID(r.ID); err != nil {
		return err
	}
	if err := agentManagementDto.CheckSomeGroups(r.PT); err != nil {
		return err
	}
	for g, v := range r.PT {
		p := "pt." + g + "."
		if v.PTFromParent.Present || v.OwnPT.Present || v.Force.Present || v.RemainQuota.Present || v.Status != nil {
			return agentManagementDto.Invalid("pt."+g, "Member มีแค่ commission_percent", "a Member only has commission_percent")
		}
		var err error
		if v.CommissionBP, err = agentManagementDto.PercentBP(p+"commission_percent", v.CommissionPercent); err != nil {
			return err
		}
		r.PT[g] = v
	}
	return nil
}
