package account

import (
	accountDto "app/app/internals/backoffice/dto/account"
	"app/app/internals/backoffice/middleware"
	accountService "app/app/service/account"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// GetAPICredentialController — GET /api/v1/bo/pr/account/api-credential
// response มี Key จึงห้าม cache (ACC-04)
func GetAPICredentialController(c *fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-store")
	res, err := accountService.GetAPICredentialService(c.UserContext(), middleware.GetActor(c))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}

// SaveAPICredentialController — POST /api/v1/bo/pr/account/update-credential
func SaveAPICredentialController(c *fiber.Ctx) error {
	var req accountDto.SaveAPICredentialRequest
	if err := utils.ParseBody(c, &req); err != nil {
		return response.Error(c, err)
	}
	if err := accountService.SaveAPICredentialService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c)); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, nil)
}
