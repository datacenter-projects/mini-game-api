package ptcalculation

import (
	"strings"

	"app/app/core/game"
	"app/pkg/apperr"
	"app/pkg/utils"
)

// CalculateRequest — POST /api/v1/dev/pt/calculate (PTC-20)
type CalculateRequest struct {
	MemberID uint   `json:"member_id"`
	GameCode string `json:"game_code"`
}

// Validate — member_id → game_code
func (r *CalculateRequest) Validate() error {
	if r.MemberID == 0 {
		return invalid("member_id", "ต้องระบุ (มากกว่า 0)", "is required (greater than 0)")
	}
	r.GameCode = strings.TrimSpace(r.GameCode)
	if !game.IsGame(r.GameCode) {
		return invalid("game_code", "ไม่มีในระบบ", "does not exist")
	}
	return nil
}

// invalid — 422 พร้อมชื่อ field ที่ผู้ใช้อ่านได้ (กฎข้อ 19)
func invalid(field, th, en string) error {
	labelTH, labelEN := utils.FieldLabel(field)
	return apperr.ErrValidation.WithMessage(labelTH+" "+th, labelEN+" "+en)
}

// LayerRow — 1 ชั้นในผล เรียง Superadmin → … → Member (PTC-22) · แถว Member ไม่มี pt
type LayerRow struct {
	ID            uint     `json:"id"`
	Position      string   `json:"position"`
	PT            *float64 `json:"pt,omitempty"`
	CommissionNet float64  `json:"commission_net"` // % ของยอดแทงที่ได้ (+) / เสีย (−) สุทธิ (PTC-13)
}
