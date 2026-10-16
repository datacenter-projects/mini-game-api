package asset

import (
	assetCore "app/app/core/asset"
	assetDto "app/app/internals/backoffice/dto/asset"
	"app/app/internals/backoffice/middleware"
	assetService "app/app/service/asset"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// GetBrandingController — POST /api/v1/bo/pr/asset/branding/detail/get (GN12) · เป้าหมาย = บัญชีที่ login (ไม่มี scope · ไม่รับ agent_id — AST-08)
func GetBrandingController(c *fiber.Ctx) error {
	res, err := assetService.GetBrandingService(c.UserContext(), middleware.GetActor(c), middleware.GetAssetTarget(c))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}

// UpdateBrandingController — POST /api/v1/bo/pr/asset/branding/detail/update (GN13)
func UpdateBrandingController(c *fiber.Ctx) error {
	var req assetDto.BrandingUpdateRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	if err := assetService.UpdateBrandingService(c.UserContext(), middleware.GetActor(c), middleware.GetAssetTarget(c), req); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, nil)
}

// UploadLogoController — POST /api/v1/bo/pr/asset/branding/logo/upload (GN14) · multipart/form-data (file) · ของบัญชีที่ login
func UploadLogoController(c *fiber.Ctx) error {
	file, err := utils.ReadFormFile(c, "file", assetCore.MaxFileSize+1)
	if err != nil {
		return response.Error(c, err)
	}
	res, err := assetService.UploadLogoService(c.UserContext(), middleware.GetActor(c), middleware.GetAssetTarget(c), file)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}

// ResetBrandingController — POST /api/v1/bo/pr/asset/branding/reset (GN16)
func ResetBrandingController(c *fiber.Ctx) error {
	if err := assetService.ResetBrandingService(c.UserContext(), middleware.GetActor(c), middleware.GetAssetTarget(c)); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, nil)
}
