// Package backoffice คือ HTTP layer ของหลังบ้าน agent/subaccount
//
//	controllers/{module}/  handler — parse request → เรียก service → ตอบ response
//	dto/{module}/          request/response struct
//	middleware/            auth + สิทธิ์ที่ใช้เฉพาะหลังบ้าน
//	routes.go              ประกาศทุก route ของหลังบ้าน (ไฟล์นี้)
//
// อ่านไฟล์นี้ไฟล์เดียวต้องรู้ได้ว่า endpoint ไหนต้องผ่าน middleware/สิทธิ์อะไรบ้าง
// จึงให้ใส่ middleware ต่อ route ไว้บรรทัดเดียวกับ route เสมอ
package backoffice

import (
	agentAuthController "app/app/internals/backoffice/controllers/agent_auth"
	mw "app/app/internals/backoffice/middleware"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(api fiber.Router) {
	bo := api.Group("/bo")

	// ---- public: ไม่ต้อง login ----
	pb := bo.Group("/pb")
	pb.Post("/auth/login", agentAuthController.LoginController)

	// ---- private: ต้อง login ----
	// logout ต้องลงทะเบียนก่อน group /pr ที่มี mw.Authenticated() เพื่อไม่ให้ถูกบังคับผ่าน middleware
	// (session ที่หลุดไปแล้วต้อง logout สำเร็จได้ — AUTH-09)
	bo.Post("/pr/auth/logout", agentAuthController.LogoutController)

	pr := bo.Group("/pr", mw.Authenticated())
	_ = pr
}
