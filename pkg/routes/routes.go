package routes

import (
	"app/app/externals/provider"
	"app/app/externals/transfer"
	"app/app/health"
	"app/app/internals/backoffice"
	"app/app/internals/developer"
	"app/app/internals/frontend"
	"app/app/internals/game"
	"app/pkg/configs"
	"app/pkg/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupRoutes คือจุดเดียวที่ mount ทุก context — route จริงของแต่ละ context อยู่ใน
// app/internals/{context}/routes.go และ app/externals/{partner}/routes.go
func SetupRoutes(a *fiber.App) {
	a.Get("/", health.RootController) // Get ลงทะเบียน HEAD ให้ด้วย

	h := a.Group("/health")
	h.Get("/live", health.LiveController)
	h.Get("/ready", health.ReadyController)

	api := a.Group("/api/v1")
	backoffice.RegisterRoutes(api) // /api/v1/bo/...
	frontend.RegisterRoutes(api)   // /api/v1/player/...
	game.RegisterRoutes(api)       // /api/v1/{scratch,cointoss,rockpaperscissors}/...
	provider.RegisterRoutes(api)   // /api/v1/provider/...
	transfer.RegisterRoutes(api)   // /api/v1/transfer/...
	// เครื่องมือ dev — public · เปิดเฉพาะ DEV_ROUTES_ENABLED=true (prod เปิดไม่ได้ — configs.Load) · Cfg nil (unit test ของ routes) = ปิด
	if configs.Cfg != nil && configs.Cfg.DevRoutesEnabled {
		developer.RegisterRoutes(api) // /api/v1/dev/...
	}

	middleware.NotFoundRoute(a)
}
