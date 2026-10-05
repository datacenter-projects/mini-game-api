package account

import (
	"app/app/internals/backoffice/middleware"
	accountService "app/app/service/account"
	"app/pkg/response"

	"github.com/gofiber/fiber/v2"
)

// GetProfileController — GET /api/v1/bo/pr/account/profile
func GetProfileController(c *fiber.Ctx) error {
	res, err := accountService.GetProfileService(c.UserContext(), middleware.GetActor(c))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}
