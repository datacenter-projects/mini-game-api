package asset

import (
	assetDto "app/app/internals/backoffice/dto/asset"
	"app/app/internals/backoffice/middleware"
	assetService "app/app/service/asset"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// ListUsedController — POST /api/v1/bo/pr/asset/used/list (GN6)
func ListUsedController(c *fiber.Ctx) error {
	var req assetDto.GameRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := assetService.ListUsedService(c.UserContext(), middleware.GetActor(c), middleware.GetAssetTarget(c), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}

// UpdateUsedController — POST /api/v1/bo/pr/asset/used/update (GN7)
func UpdateUsedController(c *fiber.Ctx) error {
	var req assetDto.UsedUpdateRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	if err := assetService.UpdateUsedService(c.UserContext(), middleware.GetActor(c), middleware.GetAssetTarget(c), req); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, nil)
}

// ResetUsedController — POST /api/v1/bo/pr/asset/used/reset (GN8)
func ResetUsedController(c *fiber.Ctx) error {
	var req assetDto.UsedResetRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	if err := assetService.ResetUsedService(c.UserContext(), middleware.GetActor(c), middleware.GetAssetTarget(c), req); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, nil)
}
