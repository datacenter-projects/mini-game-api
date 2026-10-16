package ptcalculation

import (
	ptCalculationDto "app/app/internals/developer/dto/pt_calculation"
	ptCalculationService "app/app/service/pt_calculation"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// CalculateController — POST /api/v1/dev/pt/calculate (เส้น dev · PTC-40 – PTC-43)
func CalculateController(c *fiber.Ctx) error {
	var req ptCalculationDto.CalculateRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	layers, err := ptCalculationService.CalculateMemberPTService(c.UserContext(), req.MemberID, req.GameCode)
	if err != nil {
		return response.Error(c, err)
	}
	rows := make([]ptCalculationDto.LayerRow, len(layers))
	for i, l := range layers {
		rows[i] = ptCalculationDto.LayerRow{ID: l.ID, Position: l.Position, PT: l.PT, CommissionNet: l.CommissionNet}
	}
	return response.OK(c, rows)
}
