package depositwithdraw

import (
	depositWithdrawDto "app/app/internals/backoffice/dto/deposit_withdraw"
	"app/app/internals/backoffice/middleware"
	depositWithdrawService "app/app/service/deposit_withdraw"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// CreateDepositWithdrawController — POST /api/v1/bo/pr/deposit-withdraw/create (DW2)
func CreateDepositWithdrawController(c *fiber.Ctx) error {
	var req depositWithdrawDto.CreateRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := depositWithdrawService.CreateDepositWithdrawService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}
