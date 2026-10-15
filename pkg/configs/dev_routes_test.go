package configs

import (
	"strings"
	"testing"
)

// setRequiredEnv — key ที่ Load บังคับ (ค่าทดสอบ)
func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DB_HOST", "127.0.0.1")
	t.Setenv("DB_USER", "test")
	t.Setenv("DB_NAME", "test")
	t.Setenv("REDIS_ADDR", "127.0.0.1:6379")
	t.Setenv("JWT_SECRET", strings.Repeat("x", 32))
	t.Setenv("METRICS_ADDR", "")
	t.Setenv("ASKMELOTTO_API_BASE_URL", "https://askmelotto.example.com") // บังคับใน uat / prod (currency_rate)
}

func TestLoadDevRoutes(t *testing.T) { // pt_calculation PTC-41: DEV_ROUTES_ENABLED ค่าเริ่มต้น false · prod เปิดไม่ได้
	orig := Cfg
	t.Cleanup(func() { Cfg = orig })

	tests := []struct {
		env, flag string
		wantErr   bool
		want      bool
	}{
		{"prod", "true", true, false},
		{"prod", "false", false, false},
		{"dev", "true", false, true},
		{"dev", "", false, false},
	}
	for _, tt := range tests {
		setRequiredEnv(t)
		t.Setenv("APP_ENV", tt.env)
		t.Setenv("DEV_ROUTES_ENABLED", tt.flag)
		err := Load()
		if (err != nil) != tt.wantErr {
			t.Fatalf("APP_ENV=%s DEV_ROUTES_ENABLED=%q: err %v", tt.env, tt.flag, err)
		}
		if err == nil && Cfg.DevRoutesEnabled != tt.want {
			t.Fatalf("APP_ENV=%s DEV_ROUTES_ENABLED=%q: enabled %v want %v", tt.env, tt.flag, Cfg.DevRoutesEnabled, tt.want)
		}
	}
}
