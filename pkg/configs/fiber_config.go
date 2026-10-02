package configs

import (
	"app/pkg/response"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
)

// FiberConfig สร้าง config ของ Fiber จาก Cfg
func FiberConfig() fiber.Config {
	return fiber.Config{
		AppName:                 "mini-game-api",
		ReadTimeout:             Cfg.ServerReadTimeout,
		WriteTimeout:            Cfg.ServerWriteTimeout,
		BodyLimit:               10 * 1024 * 1024,
		EnableTrustedProxyCheck: len(Cfg.TrustedProxies) > 0,
		TrustedProxies:          Cfg.TrustedProxies,
		ProxyHeader:             fiber.HeaderXForwardedFor,
		JSONEncoder:             json.Marshal,
		JSONDecoder:             json.Unmarshal,
		// error ที่หลุดมาถึง Fiber (เช่น 404/405 จาก router) ให้ตอบ envelope เดียวกันทั้งระบบ
		ErrorHandler: response.FiberErrorHandler,
	}
}
