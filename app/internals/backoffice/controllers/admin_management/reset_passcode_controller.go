package adminmanagement

import (
	adminManagementDto "app/app/internals/backoffice/dto/admin_management"
	"app/app/internals/backoffice/middleware"
	adminManagementService "app/app/service/admin_management"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// ResetPasscodeController — POST /api/v1/bo/pr/admin/passcode/reset
// response มีค่าชั่วคราวที่แสดงครั้งเดียว — ห้าม cache และห้าม log body (AUTH-46)
func ResetPasscodeController(c *fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-store")
	var req adminManagementDto.ResetCredentialRequest
	if err := utils.ParseBody(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := adminManagementService.ResetPasscodeService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}
