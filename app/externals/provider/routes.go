// Package provider คือ endpoint ที่ partner ภายนอก (provider/COM) เรียกเข้ามา
// เป็น trust boundary แยกจาก internals — auth คนละแบบ (per-agent key) ห้ามใช้ middleware ของ backoffice
// โครงสร้าง: controllers/, dto/, middleware/, routes.go
package provider

import "github.com/gofiber/fiber/v2"

func RegisterRoutes(api fiber.Router) {
	_ = api.Group("/provider")
}
