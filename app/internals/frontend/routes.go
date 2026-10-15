// Package frontend คือ HTTP layer ฝั่งผู้เล่น (player) — โครงสร้างเดียวกับ backoffice:
// controllers/{module}/, dto/{module}/, middleware/, routes.go
//
// path: /api/v1/player/pb/* (public) · /api/v1/player/pr/* (ต้อง login) · ทุกเส้น POST (player_auth PA-01)
// middleware / สิทธิ์ต่อ route เขียนไว้บรรทัดเดียวกับ route เสมอ
package frontend

import (
	playerAuthController "app/app/internals/frontend/controllers/player_auth"
	mw "app/app/internals/frontend/middleware"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(api fiber.Router) {
	player := api.Group("/player")

	// ---- public: ไม่ต้อง login ----
	pb := player.Group("/pb")
	pb.Post("/login", playerAuthController.LoginController) // PA1 · rate limit ต่อ IP + ต่อ username อยู่ใน service

	// ---- private: ต้อง login ----
	// logout ต้องลงทะเบียนก่อน group /pr ที่มี mw.PlayerAuthenticated() เพื่อไม่ให้ถูกบังคับผ่าน middleware
	// (session ที่หลุดไปแล้วต้อง logout สำเร็จได้ — PA-26)
	player.Post("/pr/auth/logout", playerAuthController.LogoutController) // PA3

	pr := player.Group("/pr", mw.PlayerAuthenticated())
	pr.Post("/auth/password/update", playerAuthController.PasswordUpdateController) // PA4 · ผู้เล่นที่ login แล้ว (SUSPENDED ผ่าน)
	pr.Post("/me/get", playerAuthController.MeController)                           // PA5 · SUSPENDED ผ่าน
	pr.Post("/balance/get", playerAuthController.BalanceController)                 // PA6 · SUSPENDED ผ่าน · ใช้เป็น ping กันหลุด
	pr.Post("/games/list", playerAuthController.GamesListController)                // PA7 · SUSPENDED ผ่าน
}
