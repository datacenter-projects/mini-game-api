package middleware

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"app/pkg/response"
	"app/platform/metrics"

	"github.com/gofiber/fiber/v2"
)

// newMetricsApp ประกอบ app แบบเดียวกับ application.go: HTTPMetrics ตัวแรก → route → NotFoundRoute ตัวสุดท้าย
func newMetricsApp() *fiber.App {
	a := fiber.New(fiber.Config{ErrorHandler: response.FiberErrorHandler})
	a.Use(HTTPMetrics)
	a.Get("/health/live", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	a.Get("/items/:id", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	a.Post("/items/:id", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusCreated) })
	a.Get("/fail", func(c *fiber.Ctx) error { return errors.New("boom") })
	g := a.Group("/private", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusUnauthorized) })
	g.Get("/secret/:id", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	NotFoundRoute(a)
	return a
}

func do(t *testing.T, a *fiber.App, method, path string) int {
	t.Helper()
	resp, err := a.Test(httptest.NewRequest(method, path, nil))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

// requestCounts คืน "method path status" → จำนวน จาก mini_game_api_http_requests_total
func requestCounts(t *testing.T) map[string]float64 {
	t.Helper()
	families, err := metrics.Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]float64{}
	for _, f := range families {
		if f.GetName() != "mini_game_api_http_requests_total" {
			continue
		}
		for _, m := range f.GetMetric() {
			l := map[string]string{}
			for _, p := range m.GetLabel() {
				l[p.GetName()] = p.GetValue()
			}
			out[l["method"]+" "+l["path"]+" "+l["status"]] = m.GetCounter().GetValue()
		}
	}
	return out
}

func TestHTTPMetricsLabels(t *testing.T) {
	metrics.HTTPRequestsTotal.Reset()
	metrics.HTTPRequestDuration.Reset()
	a := newMetricsApp()

	do(t, a, "GET", "/items/1")
	do(t, a, "POST", "/items/2")
	// request ถัดๆ ไปใช้ buffer ของ fasthttp ซ้ำ — ถ้าไม่ clone label ของ 2 request แรกจะเพี้ยน
	for _, p := range []string{"/items/333333", "/nope/aaaaaaaaaaaa", "/health/live", "/health/ready", "/fail", "/private/secret/9"} {
		do(t, a, "DELETE", p)
		do(t, a, "GET", p)
	}

	want := map[string]float64{
		"GET /items/:id 200":   2, // /items/1 + /items/333333
		"POST /items/:id 201":  1,
		"DELETE unmatched 404": 3, // ไม่มี route DELETE (/health/* ไม่นับ ทั้งที่ match และไม่ match)
		"DELETE /private 401":  1,
		"GET unmatched 404":    1, // /nope/...
		"GET /fail 500":        1, // error จาก handler → status ตาม ErrorHandler
		"GET /private 401":     1, // middleware ของ group ตอบก่อนถึง handler → prefix ของ group
	}
	got := requestCounts(t)
	if len(got) != len(want) {
		t.Fatalf("series = %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%q = %v, want %v (all: %v)", k, got[k], v, got)
		}
	}
}

func TestHTTPMetricsKeepsResponse(t *testing.T) {
	a := newMetricsApp()
	if s := do(t, a, "GET", "/fail"); s != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", s)
	}
	if s := do(t, a, "GET", "/nope"); s != fiber.StatusNotFound {
		t.Fatalf("status = %d, want 404", s)
	}
}

func TestRouteLabelNeverRawPath(t *testing.T) {
	metrics.HTTPRequestsTotal.Reset()
	a := newMetricsApp()
	do(t, a, "GET", "/items/12345")
	do(t, a, "GET", "/random/12345")
	for k := range requestCounts(t) {
		if strings.Contains(k, "12345") {
			t.Fatalf("raw path leaked into label: %q", k)
		}
	}
}
