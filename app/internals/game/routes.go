// Package game คือ HTTP layer ของเอนจิ้นเกม (scratch card, coin toss, rock-paper-scissors)
// โครงสร้าง: controllers/{game}/, dto/{game}/, routes.go
//
// route ฝั่ง admin ของเกม (ตั้งค่าเกม) ใช้ middleware จาก backoffice/middleware
// logic ของเกมอยู่ที่ app/service/{game} และ app/core/{rng,...} — ไม่อยู่ใน package นี้
package game

import "github.com/gofiber/fiber/v2"

func RegisterRoutes(api fiber.Router) {
	_ = api.Group("/scratch")
	_ = api.Group("/cointoss")
	_ = api.Group("/rockpaperscissors")
}
