package agentmanagement

import (
	agentManagementCore "app/app/core/agent_management"
	"app/app/models"
	"app/pkg/utils"
)

// request ของการแก้บัญชี (phase 4) — spec: docs/modules/agent_management.md หัวข้อ 5
// field ที่ไม่ใช่ของเส้นนั้น (เช่น ผู้สร้างส่ง pt ของลูก) = 422 (MGMT-23)

// UpdateStatusRequest — POST /manage/agents/status/update · /manage/members/status/update (MGMT-30)
type UpdateStatusRequest struct {
	ID     uint   `json:"id"`
	Status string `json:"status"`
}

func (r *UpdateStatusRequest) Validate() error {
	if err := CheckID(r.ID); err != nil {
		return err
	}
	switch models.AgentStatus(r.Status) {
	case models.AgentStatusActive, models.AgentStatusSuspended, models.AgentStatusLocked:
		return nil
	}
	return Invalid("status", "ต้องเป็น ACTIVE / SUSPENDED / LOCKED", "must be ACTIVE, SUSPENDED or LOCKED")
}

// UpdateChildPTGroup — ค่าที่ผู้สร้างตั้งให้ลูก 1 กลุ่ม ครบ 5 ค่า · ส่ง pt = 422 (ไม่มีในเส้นนี้)
type UpdateChildPTGroup struct {
	ChildPTRequest
	OwnPT *float64 `json:"pt"` // มีไว้ตอบ 422 เมื่อส่งมา
}

// CheckSomeGroups — ต้องส่งอย่างน้อย 1 กลุ่ม และทุกกลุ่มต้องมีอยู่จริง (MGMT-23)
func CheckSomeGroups[T any](pt map[string]T) error {
	if len(pt) == 0 {
		return Invalid("pt", "ต้องส่งอย่างน้อย 1 กลุ่ม", "must contain at least 1 group")
	}
	for k := range pt {
		if agentManagementCore.GamesOf(agentManagementCore.PTGroup(k)) == nil {
			return Invalid("pt."+k, "ไม่มีกลุ่มนี้", "is not a known group")
		}
	}
	return nil
}

// CheckID — id ของบัญชีที่จะแก้ ต้องส่งเป็นจำนวนเต็มบวก
func CheckID(id uint) error {
	if id == 0 {
		return Invalid("id", "ต้องส่งเป็นจำนวนเต็มบวก", "must be a positive integer")
	}
	return nil
}

// AgentInfoSection — section info ของ agents/detail/update และ members/detail/update · ส่งบาง field ได้ (MGMT-32 · MGMT-32M)
type AgentInfoSection struct {
	Name             *string          `json:"name"`
	PhoneCountryCode utils.JSONString `json:"phone_country_code"` // ต้องส่งคู่กับ phone (MGMT-08)
	Phone            utils.JSONString `json:"phone"`
}

// AgentDetailUpdateRequest — POST /manage/agents/detail/update (MGMT-32 – MGMT-34 · lead E4)
// 3 section ไม่บังคับ ส่งเฉพาะที่แก้ (nil = ไม่ส่ง) · ไม่ส่งเลยสัก section = 422 · ตรวจบนลงล่าง info → pt → status_game
type AgentDetailUpdateRequest struct {
	ID         uint                          `json:"id"`
	Info       *AgentInfoSection             `json:"info"`
	PT         map[string]UpdateChildPTGroup `json:"pt"`
	StatusGame map[string]bool               `json:"status_game"`
}

func (r *AgentDetailUpdateRequest) Validate() error {
	if err := CheckID(r.ID); err != nil {
		return err
	}
	if r.Info == nil && r.PT == nil && r.StatusGame == nil {
		return Invalid("info / pt / status_game", "ต้องส่งอย่างน้อย 1 section", "at least 1 section is required")
	}
	if r.Info != nil {
		if err := r.Info.Check(); err != nil {
			return err
		}
	}
	if r.PT != nil {
		if err := CheckSomeGroups(r.PT); err != nil {
			return err
		}
		for _, g := range SortedKeys(r.PT) {
			v := r.PT[g]
			p := "pt." + g + "."
			if v.OwnPT != nil {
				return Invalid(p+"pt", "ไม่มีในเส้นนี้ (ถือสู้กับ Member ตั้งต่อ Member)", "is not part of this request (hold against Members is set per Member)")
			}
			if err := parseChildPT(p, &v.ChildPTRequest); err != nil {
				return err
			}
			r.PT[g] = v
		}
	}
	if r.StatusGame != nil {
		if len(r.StatusGame) == 0 {
			return Invalid("status_game", "ต้องส่งอย่างน้อย 1 เกม", "must contain at least 1 game")
		}
		for _, code := range SortedKeys(r.StatusGame) {
			if _, ok := agentManagementCore.GroupOfGame(code); !ok {
				return Invalid("status_game."+code, "ไม่มีเกมนี้", "is not a known game")
			}
		}
	}
	return nil
}

// Check — ตรวจ section info (ใช้ร่วมกับ members/detail/update — member_management MGMT-32M)
func (s *AgentInfoSection) Check() error {
	if s.Name == nil && !s.Phone.Present && !s.PhoneCountryCode.Present {
		th, en := utils.SubFieldLabels("info", "name", "phone_country_code", "phone")
		return Invalid("info", "ต้องมีอย่างน้อย 1 อย่าง ("+th[0]+" / "+th[1]+" + "+th[2]+")",
			"must contain at least 1 of ("+en[0]+" / "+en[1]+" + "+en[2]+")")
	}
	if s.Name != nil && !agentManagementCore.IsValidName(*s.Name) {
		return Invalid("info.name", "ต้องยาว 3–32 ตัวอักษร ใช้ได้เฉพาะภาษาไทย อังกฤษ และตัวเลข ไม่มีช่องว่าง", "must be 3–32 characters of Thai, English letters or digits, without spaces")
	}
	if s.Phone.Present || s.PhoneCountryCode.Present {
		if !s.Phone.Present || !s.PhoneCountryCode.Present {
			return Invalid("info.phone_country_code / info.phone", "ต้องกรอกคู่กัน (ลบเบอร์ให้เว้นว่างทั้งคู่)", "must be filled in together (leave both empty to remove the phone)")
		}
		if !s.PhoneCountryCode.IsString {
			return Invalid("info.phone_country_code", "ต้องส่งเป็นข้อความ", "must be a string")
		}
		if !s.Phone.IsString {
			return Invalid("info.phone", "ต้องส่งเป็นข้อความ", "must be a string")
		}
		return ValidatePhonePair("info.", &s.PhoneCountryCode.Value, &s.Phone.Value)
	}
	return nil
}
