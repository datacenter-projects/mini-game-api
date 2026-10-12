package membermanagement

import (
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
)

// request ของการแก้ Member — spec: docs/modules/member_management.md หัวข้อ 3 (M3 · M4)

// MemberInfoSection — section info ของ members/detail/update: name · phone_country_code + phone ส่งบาง field ได้
// กฎเดียวกับ section info ของ agents/detail/update จึงใช้ type เดียวกัน (MGMT-32M)
type MemberInfoSection = agentManagementDto.AgentInfoSection

// MemberDetailUpdateRequest — POST /manage/members/detail/update (M3 · MGMT-32M · lead M3-1 / MN2)
// 2 section ไม่บังคับ ส่งเฉพาะที่แก้ (nil = ไม่ส่ง) · ไม่ส่งเลยสัก section = 422 · ตรวจบนลงล่าง info → pt
type MemberDetailUpdateRequest struct {
	ID   uint                       `json:"id"`
	Info *MemberInfoSection         `json:"info"`
	PT   map[string]MemberPTRequest `json:"pt"`
}

func (r *MemberDetailUpdateRequest) Validate() error {
	if err := agentManagementDto.CheckID(r.ID); err != nil {
		return err
	}
	if r.Info == nil && r.PT == nil {
		return agentManagementDto.Invalid("info / pt", "ต้องส่งอย่างน้อย 1 section", "at least 1 section is required")
	}
	if r.Info != nil {
		if err := r.Info.Check(); err != nil {
			return err
		}
	}
	if r.PT != nil {
		if err := agentManagementDto.CheckSomeGroups(r.PT); err != nil {
			return err
		}
		if err := parseMemberPTGroups(r.PT); err != nil {
			return err
		}
	}
	return nil
}

// UpdateStatusRequest — POST /manage/members/status/update (M4 · MGMT-30) · กฎเดียวกับ agent จึงใช้ type เดียวกัน
type UpdateStatusRequest = agentManagementDto.UpdateStatusRequest
