package configs

import (
	"app/pkg/response"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
)

// FiberConfig สร้าง config ของ Fiber จาก Cfg
//
// IP ของ client ต้องอ่านผ่าน middleware.ClientIP(c) เท่านั้น (ไล่ X-Forwarded-For จากขวาข้าม proxy ใน TRUSTED_PROXIES) — ไม่ใช้ c.IP()
// ProxyHeader เว้นว่างโดยตั้งใจ: ถ้าตั้ง X-Forwarded-For c.IP() จะคืนค่า header ทั้งก้อน ("client, proxy") และ client ปลอมได้
// EnableTrustedProxyCheck + TrustedProxies ยังใช้กับการอ่าน scheme / host ที่มาจาก proxy
func FiberConfig() fiber.Config {
	return fiber.Config{
		AppName:                 "mini-game-api",
		ReadTimeout:             Cfg.ServerReadTimeout,
		WriteTimeout:            Cfg.ServerWriteTimeout,
		BodyLimit:               10 * 1024 * 1024,
		EnableTrustedProxyCheck: len(Cfg.TrustedProxies) > 0,
		TrustedProxies:          Cfg.TrustedProxies,
		JSONEncoder:             json.Marshal,
		JSONDecoder:             json.Unmarshal,
		// error ที่หลุดมาถึง Fiber (เช่น 404/405 จาก router) ให้ตอบ envelope เดียวกันทั้งระบบ
		ErrorHandler: response.FiberErrorHandler,
	}
}
