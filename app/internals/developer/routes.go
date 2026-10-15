// Package developer คือ HTTP layer ของเครื่องมือ dev / QA — โครงสร้างเดียวกับ context อื่น:
// controllers/{module}/, dto/{module}/, routes.go
//
// path: /api/v1/dev/* · **ไม่ต้อง login** (public เฉพาะ env ที่เปิด DEV_ROUTES_ENABLED — CLAUDE.md กฎข้อ 28)
// pkg/routes เรียก RegisterRoutes เฉพาะเมื่อเปิด flag · ปิด = เส้นไม่มี (404001) · APP_ENV=prod + เปิด flag = boot ไม่ผ่าน
// ห้ามใช้ middleware auth ของ context อื่น · เครื่องมือ dev ที่เพิ่มในอนาคตอยู่ใน context นี้
package developer

import (
	ptCalculationController "app/app/internals/developer/controllers/pt_calculation"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(api fiber.Router) {
	dev := api.Group("/dev")

	// pt_calculation — docs/modules/pt_calculation.md หัวข้อ 5 (PTC-40 – PTC-43) · public (flag เปิด)
	dev.Post("/pt/calculate", ptCalculationController.CalculateController)
}
