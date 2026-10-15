// Package middleware คือ middleware ของ transfer/* (operator สาย Transfer) — trust boundary แยก ห้ามใช้ middleware auth ของ internals (CLAUDE.md ข้อ 29)
package middleware

import (
	transferDto "app/app/externals/transfer/dto/player_auth"
	playerAuthService "app/app/service/player_auth"
	pkgMiddleware "app/pkg/middleware"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

const operatorLocalsKey = "transfer_operator"

// OperatorAuthenticated ยืนยัน operator ของ transfer/* ทุกเส้น — spec: docs/modules/player_auth.md หัวข้อ 5 (PA-10 – PA-14 · PA-42)
// ลำดับ: rate limit ต่อ IP → อ่าน agent_username + key จาก body → Key → IP whitelist → สถานะ → ประเภทผู้สร้าง
// ห้าม log body ดิบ (มี key / password) — เส้นนี้ไม่ log body เลย
func OperatorAuthenticated() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := playerAuthService.OperatorRateLimitService(c.UserContext(), pkgMiddleware.ClientIP(c)); err != nil {
			return response.Error(c, err)
		}
		var creds transferDto.OperatorCreds
		if err := utils.ParseBodyNoNull(c, &creds); err != nil {
			return response.Error(c, err)
		}
		op, err := playerAuthService.OperatorAuthenticateService(c.UserContext(), creds, pkgMiddleware.ClientIP(c))
		if err != nil {
			return response.Error(c, err)
		}
		c.Locals(operatorLocalsKey, op)
		return c.Next()
	}
}

// GetOperator คืน operator ของ request — เรียกได้เฉพาะใน route ที่ผ่าน OperatorAuthenticated แล้ว
func GetOperator(c *fiber.Ctx) playerAuthService.Operator {
	op, ok := c.Locals(operatorLocalsKey).(playerAuthService.Operator)
	if !ok {
		panic("middleware.GetOperator called on a route without middleware.OperatorAuthenticated")
	}
	return op
}

// RequestMeta — ข้อมูลของ request ที่ service ใช้ (IP)
func RequestMeta(c *fiber.Ctx) playerAuthService.RequestMeta {
	return playerAuthService.RequestMeta{IP: pkgMiddleware.ClientIP(c)}
}
