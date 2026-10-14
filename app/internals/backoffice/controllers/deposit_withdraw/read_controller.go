package depositwithdraw

import (
	depositWithdrawDto "app/app/internals/backoffice/dto/deposit_withdraw"
	"app/app/internals/backoffice/middleware"
	depositWithdrawService "app/app/service/deposit_withdraw"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// ListDownlinesController — POST /api/v1/bo/pr/deposit-withdraw/downlines/list (DW1)
func ListDownlinesController(c *fiber.Ctx) error {
	var req depositWithdrawDto.DownlinesRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	page := utils.NewPage(req.Page, req.Limit)
	rows, total, err := depositWithdrawService.ListDownlinesService(c.UserContext(), middleware.GetActor(c), req, page)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Page(c, rows, page, total)
}

// ListLogsController — POST /api/v1/bo/pr/deposit-withdraw/logs/list (DW3)
func ListLogsController(c *fiber.Ctx) error {
	var req depositWithdrawDto.LogsRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	page := utils.NewPage(req.Page, req.Limit)
	rows, total, err := depositWithdrawService.ListLogsService(c.UserContext(), middleware.GetActor(c), req, page)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Page(c, rows, page, total)
}
