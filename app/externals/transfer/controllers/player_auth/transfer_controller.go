// Package playerauth คือ controller ของ transfer/* (operator) — parse → เรียก service → response เท่านั้น
package playerauth

import (
	transferDto "app/app/externals/transfer/dto/player_auth"
	"app/app/externals/transfer/middleware"
	playerAuthService "app/app/service/player_auth"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// TransferLoginController — L2 POST /api/v1/transfer/login (หลัง middleware.OperatorAuthenticated)
func TransferLoginController(c *fiber.Ctx) error {
	var req transferDto.TransferLoginRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := playerAuthService.TransferLoginService(c.UserContext(), middleware.GetOperator(c), req, middleware.RequestMeta(c))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}

// TransferRegisterController — TR POST /api/v1/transfer/register (หลัง middleware.OperatorAuthenticated)
func TransferRegisterController(c *fiber.Ctx) error {
	var req transferDto.TransferRegisterRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := playerAuthService.TransferRegisterService(c.UserContext(), middleware.GetOperator(c), req, middleware.RequestMeta(c))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}
