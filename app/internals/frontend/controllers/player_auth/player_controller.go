// Package playerauth คือ controller ฝั่งผู้เล่น — parse → เรียก service → response เท่านั้น (ไม่มี business logic)
package playerauth

import (
	playerAuthDto "app/app/internals/frontend/dto/player_auth"
	"app/app/internals/frontend/middleware"
	playerAuthService "app/app/service/player_auth"
	"app/pkg/apperr"
	pkgMiddleware "app/pkg/middleware"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// LoginController — PA1 POST /api/v1/player/pb/login
func LoginController(c *fiber.Ctx) error {
	var req playerAuthDto.LoginRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := playerAuthService.LoginService(c.UserContext(), req, middleware.RequestMeta(c))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}

// LogoutController — PA3 POST /api/v1/player/pr/auth/logout
// ไม่ผ่าน middleware.PlayerAuthenticated (ดูเหตุผลที่ LogoutService)
func LogoutController(c *fiber.Ctx) error {
	token, ok := pkgMiddleware.BearerToken(c)
	if !ok {
		return response.Error(c, apperr.ErrPlayerInvalidToken)
	}
	if err := playerAuthService.LogoutService(c.UserContext(), token); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, nil)
}

// PasswordUpdateController — PA4 POST /api/v1/player/pr/auth/password/update
func PasswordUpdateController(c *fiber.Ctx) error {
	var req playerAuthDto.PasswordUpdateRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	if err := playerAuthService.PasswordUpdateService(c.UserContext(), middleware.GetPlayer(c), req); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, nil)
}

// MeController — PA5 POST /api/v1/player/pr/me/get (ไม่รับ field)
func MeController(c *fiber.Ctx) error {
	return response.OK(c, playerAuthService.MeService(middleware.GetPlayer(c)))
}

// BalanceController — PA6 POST /api/v1/player/pr/balance/get (ไม่รับ field)
func BalanceController(c *fiber.Ctx) error {
	res, err := playerAuthService.BalanceService(c.UserContext(), middleware.GetPlayer(c))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}

// GamesListController — PA7 POST /api/v1/player/pr/games/list (ไม่รับ field)
func GamesListController(c *fiber.Ctx) error {
	res, err := playerAuthService.GamesListService(c.UserContext(), middleware.GetPlayer(c))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}
