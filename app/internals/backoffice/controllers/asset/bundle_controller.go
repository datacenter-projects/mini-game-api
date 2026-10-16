package asset

import (
	assetDto "app/app/internals/backoffice/dto/asset"
	"app/app/internals/backoffice/middleware"
	assetService "app/app/service/asset"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// GetBundleController — POST /api/v1/bo/pr/asset/bundle/detail/get (GN22) · preview ของบัญชีที่ login (ไม่รับ agent_id — AST-08)
// เส้นฝั่งผู้เล่น (GN23) เรียก assetService.GetBundleService ตัวเดียวกันตอนทำ Player Auth
func GetBundleController(c *fiber.Ctx) error {
	var req assetDto.GameRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := assetService.GetBundleService(c.UserContext(), middleware.GetActor(c), middleware.GetAssetTarget(c), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}
