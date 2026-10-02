package agentauth

import (
	"app/app/internals/backoffice/middleware"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
	"app/pkg/response"

	"github.com/gofiber/fiber/v2"
)

// LogoutController — POST /api/v1/bo/pr/auth/logout
// ไม่ผ่าน middleware.Authenticated (ดูเหตุผลที่ LogoutService)
func LogoutController(c *fiber.Ctx) error {
	token, ok := middleware.BearerToken(c)
	if !ok {
		return response.Error(c, apperr.ErrInvalidToken)
	}
	if err := agentAuthService.LogoutService(c.UserContext(), token); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, nil)
}
