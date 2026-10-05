package agentauth

import (
	agentAuthDto "app/app/internals/backoffice/dto/agent_auth"
	"app/app/internals/backoffice/middleware"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// SetupPasscodeController — POST /api/v1/bo/pr/auth/passcode/setup
func SetupPasscodeController(c *fiber.Ctx) error {
	var req agentAuthDto.SetupPasscodeRequest
	if err := utils.ParseBody(c, &req); err != nil {
		return response.Error(c, err)
	}
	if err := agentAuthService.SetupPasscodeService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c)); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, nil)
}
