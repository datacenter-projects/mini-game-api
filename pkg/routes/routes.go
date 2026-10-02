package routes

import (
	"app/app/externals/provider"
	"app/app/health"
	"app/app/internals/backoffice"
	"app/app/internals/frontend"
	"app/app/internals/game"
	"app/pkg/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupRoutes คือจุดเดียวที่ mount ทุก context — route จริงของแต่ละ context อยู่ใน
// app/internals/{context}/routes.go และ app/externals/{partner}/routes.go
func SetupRoutes(a *fiber.App) {
	h := a.Group("/health")
	h.Get("/live", health.LiveController)
	h.Get("/ready", health.ReadyController)

	api := a.Group("/api/v1")
	backoffice.RegisterRoutes(api) // /api/v1/bo/...
	frontend.RegisterRoutes(api)   // /api/v1/player/...
	game.RegisterRoutes(api)       // /api/v1/{scratch,cointoss,rockpaperscissors}/...
	provider.RegisterRoutes(api)   // /api/v1/provider/...

	middleware.NotFoundRoute(a)
}
