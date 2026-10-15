package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// BearerToken อ่าน token จาก header "Authorization: Bearer <token>" — ใช้ร่วมทุก context (backoffice / frontend)
// อยู่ที่ pkg เพราะ frontend ห้าม import backoffice (CLAUDE.md ข้อ 29)
func BearerToken(c *fiber.Ctx) (string, bool) {
	scheme, token, found := strings.Cut(c.Get(fiber.HeaderAuthorization), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	return token, token != ""
}
