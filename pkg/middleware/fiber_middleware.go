package middleware

import (
	"app/pkg/apperr"
	"app/pkg/configs"
	"app/pkg/response"
	"app/platform/logger"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

// FiberMiddleware คือ middleware ระดับทั้งระบบ — middleware ของแต่ละ context (auth/สิทธิ์)
// อยู่ที่ app/internals/{context}/middleware
func FiberMiddleware(a *fiber.App) {
	a.Use(
		cors.New(cors.Config{AllowOrigins: configs.Cfg.CORSAllowOrigins}),
		requestid.New(),
		requestContext,
		recover.New(recover.Config{
			EnableStackTrace: true,
			StackTraceHandler: func(c *fiber.Ctx, e any) {
				logger.Ctx(c.UserContext()).Errorw("panic recovered", "method", c.Method(), "path", c.OriginalURL(), "panic", e)
			},
		}),
		fiberlogger.New(fiberlogger.Config{
			Format: "${time} | ${status} | ${latency} | ${ip} | ${method} | ${path} | reqid=${locals:requestid}\n",
		}),
	)
}

// requestContext ผูก request_id เข้า c.UserContext() — controller ส่ง c.UserContext() ต่อให้ service
// แล้ว logger.Ctx(ctx) ใน service/repository จะได้ request_id อัตโนมัติ
func requestContext(c *fiber.Ctx) error {
	rid, _ := c.Locals(requestid.ConfigDefault.ContextKey).(string)
	c.SetUserContext(logger.WithRequestID(c.UserContext(), rid))
	return c.Next()
}

// NotFoundRoute ลงทะเบียนเป็นตัวสุดท้ายเสมอ
func NotFoundRoute(a *fiber.App) {
	a.Use(func(c *fiber.Ctx) error {
		return response.Error(c, apperr.ErrRouteNotFound)
	})
}
