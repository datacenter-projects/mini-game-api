package configs

import (
	"app/pkg/response"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
)

// FiberConfig สร้าง config ของ Fiber จาก Cfg
//
// c.IP() เชื่อ X-Forwarded-For เฉพาะเมื่อ request มาจาก proxy ใน TRUSTED_PROXIES เท่านั้น
// ไม่ตั้ง TRUSTED_PROXIES = ใช้ IP ของ connection ตรงๆ (กัน client ปลอม header เพื่อหลบ rate limit)
func FiberConfig() fiber.Config {
	proxyHeader := ""
	if len(Cfg.TrustedProxies) > 0 {
		proxyHeader = fiber.HeaderXForwardedFor
	}
	return fiber.Config{
		AppName:                 "mini-game-api",
		ReadTimeout:             Cfg.ServerReadTimeout,
		WriteTimeout:            Cfg.ServerWriteTimeout,
		BodyLimit:               10 * 1024 * 1024,
		EnableTrustedProxyCheck: len(Cfg.TrustedProxies) > 0,
		TrustedProxies:          Cfg.TrustedProxies,
		ProxyHeader:             proxyHeader,
		JSONEncoder:             json.Marshal,
		JSONDecoder:             json.Unmarshal,
		// error ที่หลุดมาถึง Fiber (เช่น 404/405 จาก router) ให้ตอบ envelope เดียวกันทั้งระบบ
		ErrorHandler: response.FiberErrorHandler,
	}
}
