package health

import (
	"context"
	"time"

	"app/pkg/response"
	"app/platform/database"

	"github.com/gofiber/fiber/v2"
)

// LiveController — process ยังตอบได้ (ไม่เช็ค dependency) ใช้กับ liveness probe
func LiveController(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

// ReadyController — DB และ Redis พร้อม ใช้กับ readiness probe / load balancer
func ReadyController(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
	defer cancel()

	checks := fiber.Map{"postgres": "ok", "redis": "ok"}
	ready := true
	if err := database.PingPostgreSQL(ctx); err != nil {
		checks["postgres"], ready = err.Error(), false
	}
	if err := database.PingRedis(ctx); err != nil {
		checks["redis"], ready = err.Error(), false
	}

	status := fiber.StatusOK
	if !ready {
		status = fiber.StatusServiceUnavailable
	}
	return c.Status(status).JSON(fiber.Map{"ready": ready, "checks": checks})
}

// RootController — GET/HEAD / ให้ load balancer / uptime check และคนเปิด domain ได้ 200
// ห้ามใส่ version, commit, env หรือ hostname ใน body · k8s probe ยังใช้ /health/live และ /health/ready
func RootController(c *fiber.Ctx) error {
	return response.OK(c, nil)
}
