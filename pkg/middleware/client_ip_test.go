package middleware

import (
	"net/http/httptest"
	"net/netip"
	"testing"

	"app/pkg/configs"

	"github.com/gofiber/fiber/v2"
)

func TestClientIPFrom(t *testing.T) {
	trusted := mustParse(t, "10.0.0.0/8", "172.16.5.4", "fd00::/8")
	tests := []struct {
		name  string
		peer  string
		xff   []string
		trust []string // ว่าง = ใช้ trusted ด้านบน · "none" = ไม่ตั้ง TRUSTED_PROXIES
		want  string
	}{
		{"ไม่มี XFF = IP ของ connection", "10.0.0.5", nil, nil, "10.0.0.5"},
		{"proxy ชั้นเดียว", "10.0.0.5", []string{"203.0.113.9"}, nil, "203.0.113.9"},
		{"proxy หลายชั้น (ก้อนเดียว)", "10.0.0.5", []string{"203.0.113.9, 10.0.0.7"}, nil, "203.0.113.9"},
		{"proxy หลายชั้น (หลายบรรทัด)", "10.0.0.5", []string{"203.0.113.9", "10.0.0.7"}, nil, "203.0.113.9"},
		{"client ปลอมซ้ายสุด ต้องไม่ถูกเชื่อ", "10.0.0.5", []string{"1.1.1.1, 203.0.113.9, 10.0.0.7"}, nil, "203.0.113.9"},
		{"client ปลอมหลายค่า ได้ค่าที่ proxy เติมจริง", "10.0.0.5", []string{"6.6.6.6, 7.7.7.7, 203.0.113.9"}, nil, "203.0.113.9"},
		{"IP เดี่ยวใน TRUSTED_PROXIES", "172.16.5.4", []string{"203.0.113.9"}, nil, "203.0.113.9"},
		{"มี port ต่อท้าย", "10.0.0.5", []string{"203.0.113.9:4433, 10.0.0.7"}, nil, "203.0.113.9"},
		{"IPv6 มี [] และ port", "10.0.0.5", []string{"[2001:db8::1]:80"}, nil, "2001:db8::1"},
		{"IPv4-mapped IPv6", "10.0.0.5", []string{"::ffff:203.0.113.9"}, nil, "203.0.113.9"},
		{"peer เป็น IPv6 ที่เชื่อถือ", "fd00::7", []string{"203.0.113.9"}, nil, "203.0.113.9"},
		{"ทุกค่าเป็น proxy ของเรา = ตัวซ้ายสุด", "10.0.0.5", []string{"10.1.1.1, 10.2.2.2"}, nil, "10.1.1.1"},
		{"ช่องว่าง / comma ซ้อน", "10.0.0.5", []string{" 203.0.113.9 ,, 10.0.0.7 ,"}, nil, "203.0.113.9"},
		{"ค่าอ่านไม่ได้แทรก = fail-closed ใช้ IP ของ proxy", "10.0.0.5", []string{"203.0.113.9, garbage, 10.0.0.7"}, nil, "10.0.0.5"},
		{"ค่าอ่านไม่ได้ทางขวาสุด = fail-closed", "10.0.0.5", []string{"203.0.113.9, unknown"}, nil, "10.0.0.5"},
		{"connection ไม่ใช่ proxy ที่เชื่อถือ ไม่เชื่อ XFF", "198.51.100.7", []string{"1.2.3.4"}, nil, "198.51.100.7"},
		{"ไม่ตั้ง TRUSTED_PROXIES ไม่เชื่อ XFF", "10.0.0.5", []string{"1.2.3.4"}, []string{"none"}, "10.0.0.5"},
		{"peer ที่อ่านไม่ได้ คืนตามเดิม", "not-an-ip", []string{"1.2.3.4"}, nil, "not-an-ip"},
	}
	for _, tt := range tests {
		ts := trusted
		switch {
		case len(tt.trust) == 1 && tt.trust[0] == "none":
			ts = nil
		case tt.trust != nil:
			ts = mustParse(t, tt.trust...)
		}
		if got := clientIPFrom(tt.peer, tt.xff, ts); got != tt.want {
			t.Errorf("%s: clientIPFrom(%q, %v) = %q want %q", tt.name, tt.peer, tt.xff, got, tt.want)
		}
	}
}

// ClientIP จริงผ่าน fiber: connection ของ app.Test มาจาก 0.0.0.0
func TestClientIPHandler(t *testing.T) {
	orig := configs.Cfg
	configs.Cfg = &configs.Config{}
	t.Cleanup(func() { configs.Cfg = orig })

	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString(ClientIP(c)) })
	get := func(xff ...string) string {
		req := httptest.NewRequest("GET", "/", nil)
		for _, v := range xff {
			req.Header.Add("X-Forwarded-For", v)
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		buf := make([]byte, 64)
		n, _ := resp.Body.Read(buf)
		return string(buf[:n])
	}

	if got := get("1.2.3.4"); got != "0.0.0.0" { // ไม่ตั้ง TRUSTED_PROXIES: ไม่เชื่อ header
		t.Errorf("ไม่ตั้ง TRUSTED_PROXIES: %q", got)
	}
	configs.Cfg.TrustedProxyPrefixes = mustParse(t, "0.0.0.0/32")
	if got := get("66.66.66.66, 203.0.113.9, 0.0.0.0"); got != "203.0.113.9" {
		t.Errorf("ผ่าน proxy ที่เชื่อถือ: %q", got)
	}
	if got := get("66.66.66.66", "203.0.113.9"); got != "203.0.113.9" { // header หลายบรรทัด
		t.Errorf("header หลายบรรทัด: %q", got)
	}
	configs.Cfg.TrustedProxyPrefixes = mustParse(t, "10.0.0.0/8") // connection (0.0.0.0) ไม่ใช่ proxy ที่เชื่อถือ
	if got := get("203.0.113.9"); got != "0.0.0.0" {
		t.Errorf("connection ไม่ใช่ proxy ที่เชื่อถือ: %q", got)
	}
}

// mustParse แปลงรายการ TRUSTED_PROXIES แบบเดียวกับตอน configs.Load (ผิด = test พัง)
func mustParse(t *testing.T, list ...string) []netip.Prefix {
	t.Helper()
	ps, err := configs.ParseTrustedProxies(list)
	if err != nil {
		t.Fatal(err)
	}
	return ps
}
