// Package middleware คือ middleware ที่ใช้เฉพาะฝั่งผู้เล่น (frontend) — ห้าม import ของ backoffice (CLAUDE.md ข้อ 29)
package middleware

import (
	playerAuthService "app/app/service/player_auth"
	"app/pkg/apperr"
	pkgMiddleware "app/pkg/middleware"
	"app/pkg/response"

	"github.com/gofiber/fiber/v2"
)

const playerLocalsKey = "pl_player"

// PlayerAuthenticated ตรวจ token + session + สถานะของทุก route ใต้ /player/pr — spec: docs/modules/player_auth.md หัวข้อ 5
// error ของ middleware (ทุกเส้น pr/* อ้างถึงที่นี่): 407202 token · 407203 session สิ้นสุด · 407204 ถูก login จากที่อื่น · 407301 ล็อก · 407302 สายบนล็อก
// SUSPENDED ผ่านได้ (ดูข้อมูลได้ แทงไม่ได้ — can_bet) · logout ไม่ผ่านตัวนี้ (ต้อง logout สำเร็จได้แม้ session หลุดแล้ว — PA-26)
func PlayerAuthenticated() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, ok := pkgMiddleware.BearerToken(c)
		if !ok {
			return response.Error(c, apperr.ErrPlayerInvalidToken)
		}
		player, err := playerAuthService.AuthenticateService(c.UserContext(), token)
		if err != nil {
			return response.Error(c, err)
		}
		c.Locals(playerLocalsKey, player)
		return c.Next()
	}
}

// GetPlayer คืนผู้เล่นของ request — เรียกได้เฉพาะใน route ที่ผ่าน PlayerAuthenticated แล้ว
func GetPlayer(c *fiber.Ctx) playerAuthService.Player {
	player, ok := c.Locals(playerLocalsKey).(playerAuthService.Player)
	if !ok {
		panic("middleware.GetPlayer called on a route without middleware.PlayerAuthenticated")
	}
	return player
}

// RequestMeta — ข้อมูลของ request ที่ service ใช้ (IP)
func RequestMeta(c *fiber.Ctx) playerAuthService.RequestMeta {
	return playerAuthService.RequestMeta{IP: c.IP()}
}
