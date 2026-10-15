// Package asset คือ handler ของ module asset — parse → เรียก service → ตอบ (docs/modules/asset.md หัวข้อ 5)
package asset

import (
	assetDto "app/app/internals/backoffice/dto/asset"
	"app/app/internals/backoffice/middleware"
	assetService "app/app/service/asset"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// ListGamesController — POST /api/v1/bo/pr/asset/games/list (GN1) · ไม่มี field ใน request จึงไม่ parse body
func ListGamesController(c *fiber.Ctx) error {
	res, err := assetService.GamesListService(c.UserContext(), middleware.GetActor(c))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}

// ListSlotsController — POST /api/v1/bo/pr/asset/slots/list (GN2)
func ListSlotsController(c *fiber.Ctx) error {
	var req assetDto.GameRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := assetService.SlotsListService(c.UserContext(), req.GameCode)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}
