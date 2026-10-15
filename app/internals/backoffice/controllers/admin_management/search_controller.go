package adminmanagement

import (
	adminManagementDto "app/app/internals/backoffice/dto/admin_management"
	adminManagementService "app/app/service/admin_management"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// SearchAccountsController — POST /api/v1/bo/pr/admin/accounts/search (ADMIN เท่านั้น — สิทธิ์อยู่ที่ route)
func SearchAccountsController(c *fiber.Ctx) error {
	var req adminManagementDto.AccountSearchRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	rows, err := adminManagementService.SearchAccountsService(c.UserContext(), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, rows)
}
