package agentmanagement

import (
	"strings"

	agentManagementCore "app/app/core/agent_management"
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/app/models"
	"app/pkg/apperr"
	"app/pkg/utils"
)

// Company Seamless Master (CSM) กับ Share Master — MGMT-19 (lead H2 / H3 / Q-R1 / E1-1 / B2 2026-10-09)
// Share Master ใช้ PT · Force / Remain (= 0) · pt.status · status_game ตาม CSM อัตโนมัติ · CSM ตั้งได้แค่ Commission แยกต่อคน

func isCSM(role models.AgentRole, agentType *models.AgentType) bool {
	return agentManagementCore.UserTypeOf(role, agentType) == agentManagementCore.UserTypeCompanySeamlessMaster
}

func isShareMaster(role models.AgentRole, agentType *models.AgentType) bool {
	return agentManagementCore.UserTypeOf(role, agentType) == agentManagementCore.UserTypeShareMaster
}

// shareMasterFieldError — ส่งค่าที่ระบบตั้งเองให้ Share Master (422 ตาม lead E1-1)
func shareMasterFieldError(field string) error {
	th, en := utils.FieldLabel(field)
	cTH, cEN := utils.SubFieldLabels("pt", "commission_percent")
	return apperr.ErrValidation.WithMessage(th+": Share Master ใช้ค่าตาม Company Seamless Master (ส่งได้แค่ "+cTH[0]+")",
		en+": Share Master follows its Company Seamless Master (only "+cEN[0]+" can be sent)")
}

// fullPTRequiredError — ส่งแค่ commission_percent ให้บัญชีที่ไม่ใช่ Share Master ใต้ CSM (ต้องครบ 5 ค่า — MGMT-23)
func fullPTRequiredError(group string) error {
	th, en := utils.FieldLabel("pt." + group + ".pt_from_parent")
	allTH, allEN := utils.SubFieldLabels("pt", "pt_from_parent", "force", "remain_quota", "commission_percent", "status")
	return apperr.ErrValidation.WithMessage(th+" ต้องส่ง (ต้องครบ 5 ค่า: "+strings.Join(allTH, " · ")+")",
		en+" is required (all 5 values: "+strings.Join(allEN, ", ")+")")
}

// checkPTShape — Share Master ใต้ CSM ต้องส่งแค่ commission · บัญชีอื่นต้องครบ 5 ค่า (7.1 ข้อ 4) · ไล่กลุ่มตามตัวอักษร
func checkPTShape(pt map[string]agentManagementDto.ChildPTRequest, shareMaster bool) error {
	for _, g := range SortedGroups(pt) {
		if shareMaster && !pt[g].CommissionOnly {
			return shareMasterFieldError("pt." + g)
		}
		if !shareMaster && pt[g].CommissionOnly {
			return fullPTRequiredError(g)
		}
		if !shareMaster && pt[g].Missing != nil {
			return pt[g].Missing
		}
	}
	return nil
}

// followCSM — ค่าที่ Share Master ได้จาก CSM ของกลุ่มนี้ (pt_from_parent = ที่ CSM ได้รับ · force / remain = 0) + commission ที่ CSM ตั้ง
func followCSM(csm []models.AgentGameSetting, group agentManagementCore.PTGroup, commission float64) (agentManagementCore.ChildPT, bool) {
	cur, _ := groupSetting(csm, group)
	return agentManagementCore.ChildPT{PTFromParent: cur.PTFromParent, Commission: commission}, cur.Status
}

// statusGameOf — status_game ต่อเกมของบัญชี
func statusGameOf(settings []models.AgentGameSetting) map[string]bool {
	out := make(map[string]bool, len(settings))
	for _, s := range settings {
		out[s.GameCode] = s.StatusGame
	}
	return out
}
