package asset

import (
	assetDto "app/app/internals/backoffice/dto/asset"
	"app/app/internals/backoffice/middleware"
	assetService "app/app/service/asset"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// GetThemeController — POST /api/v1/bo/pr/asset/theme/detail/get (GN9)
func GetThemeController(c *fiber.Ctx) error {
	var req assetDto.GameRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := assetService.GetThemeService(c.UserContext(), middleware.GetActor(c), middleware.GetAssetTarget(c), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}

// UpdateThemeController — POST /api/v1/bo/pr/asset/theme/detail/update (GN10) · แทนทั้งชุด (ครบ 5 ค่า)
func UpdateThemeController(c *fiber.Ctx) error {
	var req assetDto.ThemeUpdateRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	if err := assetService.UpdateThemeService(c.UserContext(), middleware.GetActor(c), middleware.GetAssetTarget(c), req); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, nil)
}

// ResetThemeController — POST /api/v1/bo/pr/asset/theme/reset (GN11)
func ResetThemeController(c *fiber.Ctx) error {
	var req assetDto.GameRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	if err := assetService.ResetThemeService(c.UserContext(), middleware.GetActor(c), middleware.GetAssetTarget(c), req); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, nil)
}
