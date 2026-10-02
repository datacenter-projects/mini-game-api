// Package frontend คือ HTTP layer ฝั่งผู้เล่น (player) — โครงสร้างเดียวกับ backoffice:
// controllers/{module}/, dto/{module}/, middleware/, routes.go
package frontend

import "github.com/gofiber/fiber/v2"

func RegisterRoutes(api fiber.Router) {
	player := api.Group("/player")
	_ = player
}
