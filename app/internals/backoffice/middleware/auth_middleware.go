// Package middleware คือ middleware ที่ใช้เฉพาะหลังบ้าน
package middleware

import (
	"strings"

	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
	"app/pkg/response"

	"github.com/gofiber/fiber/v2"
)

const actorLocalsKey = "bo_actor"

// Authenticated ตรวจ token + session ของทุก route ใต้ /bo/pr — spec: docs/modules/agent_auth.md หัวข้อ 5
// ไม่ได้เช็คด่านหลัง login — route ต้องใส่ PassedGates เองบรรทัดเดียวกับ route
func Authenticated() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, ok := BearerToken(c)
		if !ok {
			return response.Error(c, apperr.ErrInvalidToken)
		}
		actor, err := agentAuthService.AuthenticateService(c.UserContext(), token, c.IP())
		if err != nil {
			return response.Error(c, err)
		}
		c.Locals(actorLocalsKey, actor)
		return c.Next()
	}
}

// GetActor คืนผู้ใช้ของ request — เรียกได้เฉพาะใน route ที่ผ่าน Authenticated แล้ว
func GetActor(c *fiber.Ctx) agentAuthService.Actor {
	actor, ok := c.Locals(actorLocalsKey).(agentAuthService.Actor)
	if !ok {
		panic("middleware.GetActor called on a route without middleware.Authenticated")
	}
	return actor
}

// BearerToken อ่าน token จาก header "Authorization: Bearer <token>"
func BearerToken(c *fiber.Ctx) (string, bool) {
	scheme, token, found := strings.Cut(c.Get(fiber.HeaderAuthorization), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	return token, token != ""
}
