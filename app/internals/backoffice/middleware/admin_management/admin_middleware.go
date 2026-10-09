// Package adminmanagement คือ middleware ของเส้น ADMIN (/admin/...) — spec: docs/modules/admin_management.md
// ต้องอยู่หลัง Authenticated และ PassedGates เสมอ
package adminmanagement

import (
	"app/app/internals/backoffice/middleware"
	adminManagementService "app/app/service/admin_management"
	"app/pkg/response"

	"github.com/gofiber/fiber/v2"
)

// RequireAdmin — เฉพาะบัญชีหลัก role ADMIN · อื่น = 401308 (AUTH-44)
//
//	pr.Post("/admin/password/reset", mw.PassedGates(), adminMw.RequireAdmin(), mw.RequirePasscode(), ...)
func RequireAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := adminManagementService.CheckAdminService(middleware.GetActor(c)); err != nil {
			return response.Error(c, err)
		}
		return c.Next()
	}
}
