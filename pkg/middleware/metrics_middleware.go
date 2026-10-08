package middleware

import (
	"strconv"
	"strings"
	"time"

	"app/platform/metrics"

	"github.com/gofiber/fiber/v2"
)

// localsUnmatched ถูกตั้งโดย NotFoundRoute — request ที่ไม่ตรง route ไหนเลย
const localsUnmatched = "metrics_unmatched"

// unmatchedPath คือ label path ของ request ที่ไม่ตรง route ไหนเลย (404) — ไม่ใช้ path จริงกัน cardinality ระเบิด
const unmatchedPath = "unmatched"

// HTTPMetrics นับ request และเวลาตอบ (label method, path = route pattern, status) — ไม่นับ / และ /health/* (LB poll)
// ต้องเป็น middleware ตัวแรก เวลาที่วัดจะได้ครอบทุก middleware
func HTTPMetrics(c *fiber.Ctx) error {
	if p := c.Path(); p == "/" || strings.HasPrefix(p, "/health/") {
		return c.Next()
	}

	start := time.Now()
	if err := c.Next(); err != nil {
		// ErrorHandler ของ Fiber ทำงานหลัง middleware คืนค่า — เรียกเองตรงนี้ status ที่นับจะได้ตรงกับที่ client ได้รับ
		if herr := c.App().ErrorHandler(c, err); herr != nil {
			_ = c.SendStatus(fiber.StatusInternalServerError)
		}
	}

	// ค่าจาก fiber.Ctx ชี้ไปที่ buffer ของ fasthttp ซึ่งถูกใช้ซ้ำใน request ถัดไป ต้อง strings.Clone
	// ก่อนเก็บเป็น label ไม่งั้น label ที่ Prometheus ถือไว้จะเพี้ยนเป็นค่าของ request อื่น
	method := strings.Clone(c.Method())
	path := routeLabel(c)
	status := strconv.Itoa(c.Response().StatusCode())

	metrics.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
	metrics.HTTPRequestDuration.WithLabelValues(method, path, status).Observe(time.Since(start).Seconds())
	return nil
}

// routeLabel คืน route pattern (เช่น /api/v1/bo/agents/:id) ไม่ใช่ path จริง
//   - ไม่ตรง route ไหน (จบที่ NotFoundRoute) → "unmatched"
//   - ถูก middleware ของ group ตอบกลับก่อนถึง handler (เช่น 401) → prefix ของ group
//   - ถูก middleware ระดับ app ตอบเอง (เช่น CORS preflight) → "/"
func routeLabel(c *fiber.Ctx) string {
	r := c.Route()
	if unmatched, _ := c.Locals(localsUnmatched).(bool); unmatched || r == nil || r.Path == "" {
		return unmatchedPath
	}
	return strings.Clone(r.Path)
}
