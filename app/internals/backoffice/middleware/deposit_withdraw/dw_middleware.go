// Package depositwithdraw คือ middleware ของเส้น /deposit-withdraw/* — spec: docs/modules/deposit_withdraw.md
// ต้องอยู่หลัง Authenticated และ PassedGates เสมอ
package depositwithdraw

import (
	"app/app/internals/backoffice/middleware"
	depositWithdrawService "app/app/service/deposit_withdraw"
	"app/pkg/response"

	"github.com/gofiber/fiber/v2"
)

// RejectAdmin — ADMIN ใช้เส้นฝาก / ถอนไม่ได้ = 404305 (DW-03) · วางก่อน RequirePermission (ADMIN ไม่มีเมนู — ไม่งั้นได้ 402303)
//
//	pr.Post("/deposit-withdraw/create", mw.PassedGates(), dwMw.RejectAdmin(), paymentEdit, mw.RequirePasscode(), ...)
func RejectAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := depositWithdrawService.CheckNotAdminService(middleware.GetActor(c)); err != nil {
			return response.Error(c, err)
		}
		return c.Next()
	}
}
