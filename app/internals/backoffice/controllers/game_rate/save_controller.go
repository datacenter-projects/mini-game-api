package gamerate

import (
	gameRateDto "app/app/internals/backoffice/dto/game_rate"
	"app/app/internals/backoffice/middleware"
	gameRateService "app/app/service/game_rate"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// UpdateRateController — RE4 POST /api/v1/bo/pr/rate/detail/update (+ passcode)
func UpdateRateController(c *fiber.Ctx) error {
	var req gameRateDto.UpdateRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := gameRateService.UpdateRateService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}

// RollbackRateController — RE7 POST /api/v1/bo/pr/rate/versions/rollback (+ passcode)
func RollbackRateController(c *fiber.Ctx) error {
	var req gameRateDto.RollbackRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := gameRateService.RollbackRateService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}
