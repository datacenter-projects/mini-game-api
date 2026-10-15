package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
		token  string
		ok     bool
	}{
		{"ปกติ", "Bearer abc.def", "abc.def", true},
		{"ตัวพิมพ์ของ scheme ไม่สำคัญ", "bearer abc", "abc", true},
		{"ตัดช่องว่างท้าย", "Bearer  abc  ", "abc", true},
		{"ไม่มี header", "", "", false},
		{"scheme อื่น", "Basic abc", "", false},
		{"ไม่มี token", "Bearer ", "", false},
		{"มีแต่ scheme", "Bearer", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			var gotToken string
			var gotOK bool
			app.Get("/", func(c *fiber.Ctx) error {
				gotToken, gotOK = BearerToken(c)
				return nil
			})
			req := httptest.NewRequest("GET", "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			if _, err := app.Test(req); err != nil {
				t.Fatal(err)
			}
			if gotToken != tt.token || gotOK != tt.ok {
				t.Errorf("BearerToken(%q) = %q, %v want %q, %v", tt.header, gotToken, gotOK, tt.token, tt.ok)
			}
		})
	}
}
