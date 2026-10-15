// Package transfer คือ endpoint ที่ operator สาย Transfer เรียกเข้ามา (server → server)
// เป็น trust boundary แยกจาก internals — auth คือ agent_username + key ใน body + IP whitelist (ไม่มี sign) ห้ามใช้ middleware ของ backoffice / frontend
// โครงสร้าง: controllers/, dto/, middleware/, routes.go · ทุกเส้น POST · /api/v1/transfer/* (ไม่มี prefix ext — lead PQ1)
package transfer

import (
	playerAuthController "app/app/externals/transfer/controllers/player_auth"
	mw "app/app/externals/transfer/middleware"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(api fiber.Router) {
	tr := api.Group("/transfer", mw.OperatorAuthenticated())              // ทุกเส้นใต้ /transfer ต้องผ่าน operator: Key + IP + สถานะ + ประเภทผู้สร้าง
	tr.Post("/login", playerAuthController.TransferLoginController)       // L2
	tr.Post("/register", playerAuthController.TransferRegisterController) // TR
}
