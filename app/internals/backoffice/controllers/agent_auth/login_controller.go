package agentauth

import (
	agentAuthDto "app/app/internals/backoffice/dto/agent_auth"
	"app/app/internals/backoffice/middleware"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// LoginController — POST /api/v1/bo/pb/auth/login
func LoginController(c *fiber.Ctx) error {
	var req agentAuthDto.LoginRequest
	if err := utils.ParseBody(c, &req); err != nil {
		return response.Error(c, err)
	}

	res, err := agentAuthService.LoginService(c.UserContext(), req, middleware.RequestMeta(c))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}
