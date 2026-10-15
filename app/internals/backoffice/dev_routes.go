package backoffice

import (
	ptCalculationController "app/app/internals/backoffice/controllers/pt_calculation"
	mw "app/app/internals/backoffice/middleware"
	adminMw "app/app/internals/backoffice/middleware/admin_management"

	"github.com/gofiber/fiber/v2"
)

// RegisterDevRoutes — เส้นให้ dev / QA เช็คผลเอง (/api/v1/bo/pr/dev/...) · pkg/routes เรียกเฉพาะเมื่อ DEV_ROUTES_ENABLED=true
// (prod เปิดไม่ได้ — configs.Load ไม่ผ่าน) · ปิดอยู่ = เส้นไม่มีจริง ได้ 404001 (pt_calculation PTC-40 – PTC-43)
// ADMIN เท่านั้น เห็นทุกสาย (ข้อยกเว้นกฎข้อ 22 เฉพาะเส้น dev — PTC-42)
//
// ต้องเรียกหลัง RegisterRoutes: mw.Authenticated() ของ group /bo/pr ครอบทุก path ใต้ /bo/pr อยู่แล้ว
// จึงไม่ใส่ซ้ำที่นี่ (ใส่ซ้ำ = ตรวจ token + แตะ session สองรอบ)
func RegisterDevRoutes(api fiber.Router) {
	dev := api.Group("/bo/pr/dev")

	// pt_calculation — docs/modules/pt_calculation.md หัวข้อ 5
	dev.Post("/pt/calculate", mw.PassedGates(), adminMw.RequireAdmin(), ptCalculationController.CalculateController)
}
