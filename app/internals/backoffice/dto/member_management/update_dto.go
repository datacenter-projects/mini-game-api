package membermanagement

import (
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
)

// request ของการแก้ Member (phase 4) — spec: docs/modules/agent_management.md หัวข้อ 5

// UpdateInfoRequest — POST /manage/members/update-info (MGMT-09A) · กฎเดียวกับ agent จึงใช้ type เดียวกัน
type UpdateInfoRequest = agentManagementDto.UpdateInfoRequest

// UpdateStatusRequest — POST /manage/members/update-status (MGMT-30) · กฎเดียวกับ agent จึงใช้ type เดียวกัน
type UpdateStatusRequest = agentManagementDto.UpdateStatusRequest

// UpdateMemberPTRequest — POST /manage/members/update-pt (MGMT-21 แก้ 2026-10-09) · แก้เฉพาะกลุ่มที่ส่ง · ในกลุ่มต้องครบ pt + commission_percent
type UpdateMemberPTRequest struct {
	ID uint                       `json:"id"`
	PT map[string]MemberPTRequest `json:"pt"`
}

func (r *UpdateMemberPTRequest) Validate() error {
	if err := agentManagementDto.CheckID(r.ID); err != nil {
		return err
	}
	if err := agentManagementDto.CheckSomeGroups(r.PT); err != nil {
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
