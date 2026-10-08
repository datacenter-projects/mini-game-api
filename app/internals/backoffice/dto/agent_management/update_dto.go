package agentmanagement

import (
	"strings"

	agentManagementCore "app/app/core/agent_management"
	"app/app/models"
	"app/pkg/utils"
)

// request ของการแก้บัญชี (phase 4) — spec: docs/modules/agent_management.md หัวข้อ 5
// field ที่ไม่ใช่ของเส้นนั้น (เช่น ผู้สร้างส่ง pt ของลูก) = 422 (MGMT-23)

// UpdateInfoRequest — POST /manage/agents/update-info · /manage/members/update-info (MGMT-09, MGMT-09A)
// แทนทั้งชุด: ต้องส่งทั้ง name และ phone ("" = ไม่ตั้ง)
type UpdateInfoRequest struct {
	ID    uint             `json:"id"` // บัญชีที่จะแก้ (ลูกตรง) — ผู้แก้มาจาก token
	Name  string           `json:"name"`
	Phone utils.JSONString `json:"phone"`
}

func (r *UpdateInfoRequest) Validate() error {
	if err := CheckID(r.ID); err != nil {
		return err
	}
	if !agentManagementCore.IsValidName(r.Name) {
		return Invalid("name", "ต้องยาว 3–32 ตัวอักษร ใช้ได้เฉพาะภาษาไทย อังกฤษ และตัวเลข ไม่มีช่องว่าง", "must be 3–32 characters of Thai, English letters or digits, without spaces")
	}
	if !r.Phone.Present || !r.Phone.IsString {
		return Invalid("phone", `ต้องส่งเป็นข้อความ (ไม่ตั้งให้ส่ง "")`, `must be a string (send "" to leave it empty)`)
	}
	r.Phone.Value = strings.TrimSpace(r.Phone.Value)
	if !agentManagementCore.IsValidPhone(r.Phone.Value) {
		return Invalid("phone", "ต้องเป็นตัวเลข 8–15 ตัว (ไม่กรอกให้ส่ง \"\")", `must be 8–15 digits (send "" to leave it empty)`)
	}
	return nil
}

// UpdateStatusRequest — POST /manage/agents/update-status · /manage/members/update-status (MGMT-30)
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

// UpdateChildPTGroup — ค่าที่ผู้สร้างตั้งให้ลูก 1 กลุ่ม ครบ 5 ค่า · ห้ามส่ง pt (ค่าถือที่ลูกตั้งเอง)
type UpdateChildPTGroup struct {
	ChildPTRequest
	OwnPT utils.Decimal `json:"pt"`
}

// UpdateChildPTRequest — POST /manage/agents/update-pt (MGMT-23 – MGMT-25) · ส่งเฉพาะกลุ่มที่จะแก้
type UpdateChildPTRequest struct {
	ID uint                          `json:"id"`
	PT map[string]UpdateChildPTGroup `json:"pt"`
}

func (r *UpdateChildPTRequest) Validate() error {
	if err := CheckID(r.ID); err != nil {
		return err
	}
	if err := CheckSomeGroups(r.PT); err != nil {
		return err
	}
	for g, v := range r.PT {
		p := "pt." + g + "."
		if v.OwnPT.Present {
			return Invalid(p+"pt", "แก้ที่เส้นนี้ไม่ได้ (บัญชีตั้งค่าถือเอง)", "cannot be set here (each account sets its own pt)")
		}
		if err := parseChildPT(p, &v.ChildPTRequest); err != nil {
			return err
		}
		r.PT[g] = v
	}
	return nil
}

// UpdateOwnPTGroup — ค่าถือของตัวเอง 1 กลุ่ม · ค่าที่ผู้สร้างตั้งให้ = 422
type UpdateOwnPTGroup struct {
	OwnPT             utils.Decimal `json:"pt"`
	PTFromParent      utils.Decimal `json:"pt_from_parent"`
	Force             utils.Decimal `json:"force"`
	RemainQuota       utils.Decimal `json:"remain_quota"`
	CommissionPercent utils.Decimal `json:"commission_percent"`
	Status            *bool         `json:"status"`

	PTBP int `json:"-"`
}

// UpdateOwnPTRequest — POST /manage/agents/update-hold (ไม่มี id — บัญชีของ token) (MGMT-22)
type UpdateOwnPTRequest struct {
	PT map[string]UpdateOwnPTGroup `json:"pt"`
}

func (r *UpdateOwnPTRequest) Validate() error {
	if err := CheckSomeGroups(r.PT); err != nil {
		return err
	}
	for g, v := range r.PT {
		p := "pt." + g + "."
		if v.PTFromParent.Present || v.Force.Present || v.RemainQuota.Present || v.CommissionPercent.Present || v.Status != nil {
			return Invalid("pt."+g, "เส้นนี้แก้ได้แค่ pt (ค่าอื่นผู้สร้างเป็นคนตั้ง)", "only pt can be set here (other values are set by the creator)")
		}
		var err error
		if v.PTBP, err = PercentBP(p+"pt", v.OwnPT); err != nil {
			return err
		}
		r.PT[g] = v
	}
	return nil
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

// UpdateGamesRequest — POST /manage/agents/update-games (MGMT-20) · ส่งเฉพาะเกมที่จะเปลี่ยน
type UpdateGamesRequest struct {
	ID         uint            `json:"id"`
	StatusGame map[string]bool `json:"status_game"`
}

func (r *UpdateGamesRequest) Validate() error {
	if err := CheckID(r.ID); err != nil {
		return err
	}
	if len(r.StatusGame) == 0 {
		return Invalid("status_game", "ต้องส่งอย่างน้อย 1 เกม", "must contain at least 1 game")
	}
	for code := range r.StatusGame {
		if _, ok := agentManagementCore.GroupOfGame(code); !ok {
			return Invalid("status_game."+code, "ไม่มีเกมนี้", "is not a known game")
		}
	}
	return nil
}
