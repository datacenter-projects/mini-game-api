package agentauth

import (
	agentAuthDto "app/app/internals/backoffice/dto/agent_auth"
	"app/app/internals/backoffice/middleware"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// ResetPasscodeController — POST /api/v1/bo/pr/admin/passcode/reset
// response มีค่าชั่วคราวที่แสดงครั้งเดียว — ห้าม cache และห้าม log body (AUTH-46)
func ResetPasscodeController(c *fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-store")
	var req agentAuthDto.ResetCredentialRequest
	if err := utils.ParseBody(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := agentAuthService.ResetPasscodeService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}
