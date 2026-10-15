package ptcalculation

import (
	"strings"

	"app/app/core/game"
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
)

// CalculateRequest — POST /api/v1/bo/pr/dev/pt/calculate (PTC-20)
type CalculateRequest struct {
	MemberID uint   `json:"member_id"`
	GameCode string `json:"game_code"`
}

// Validate — member_id → game_code
func (r *CalculateRequest) Validate() error {
	if r.MemberID == 0 {
		return agentManagementDto.Invalid("member_id", "ต้องระบุ (มากกว่า 0)", "is required (greater than 0)")
	}
	r.GameCode = strings.TrimSpace(r.GameCode)
	if !game.IsGame(r.GameCode) {
		return agentManagementDto.Invalid("game_code", "ไม่มีในระบบ", "does not exist")
	}
	return nil
}

// LayerRow — 1 ชั้นในผล เรียง Superadmin → … → Member (PTC-22) · แถว Member ไม่มี pt
type LayerRow struct {
	ID         uint     `json:"id"`
	Position   string   `json:"position"`
	PT         *float64 `json:"pt,omitempty"`
	Commission float64  `json:"commission"`
}
