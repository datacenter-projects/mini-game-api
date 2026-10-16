package configs

import "testing"

func TestLoadCurrencyRate(t *testing.T) { // currency_rate หัวข้อ 6
	tests := []struct {
		name        string
		env, url    string
		sync        string
		wantErr     bool
		wantMissing bool
		wantURL     string
		wantSync    bool
	}{
		{"local ไม่ตั้ง = ค่าเริ่มต้น", "local", "", "", false, false, defaultCurrencyRateBaseURL, true},
		{"dev ตั้งเอง", "dev", "https://dev-api.example", "", false, false, "https://dev-api.example", true},
		{"มี port", "local", "http://localhost:9000", "", false, false, "http://localhost:9000", true},
		{"ปิด sync", "local", "", "false", false, false, defaultCurrencyRateBaseURL, false},
		{"prod ไม่ตั้ง", "prod", "", "", false, true, "", true},
		{"uat ตั้ง", "uat", "https://api.example", "", false, false, "https://api.example", true},
		{"มี path", "local", "https://api.example/api", "", true, false, "", true},
		{"มี / ท้าย", "local", "https://api.example/", "", true, false, "", true},
		{"มี query", "local", "https://api.example?x=1", "", true, false, "", true},
		{"ไม่มี scheme", "local", "api.example", "", true, false, "", true},
		{"scheme อื่น", "local", "ftp://api.example", "", true, false, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CURRENCY_RATE_API_BASE_URL", tt.url)
			t.Setenv("IS_CURRENCY_RATE_SYNC", tt.sync)
			var missing []string
			required := func(k string) string {
				if tt.url == "" {
					missing = append(missing, k)
				}
				return tt.url
			}
			cfg := &Config{AppEnv: tt.env}
			err := loadCurrencyRate(cfg, required)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if (len(missing) > 0) != tt.wantMissing {
				t.Fatalf("missing = %v", missing)
			}
			if !tt.wantErr && cfg.CurrencyRate.BaseURL != tt.wantURL {
				t.Fatalf("url = %q, want %q", cfg.CurrencyRate.BaseURL, tt.wantURL)
			}
			if cfg.CurrencyRate.SyncEnabled != tt.wantSync {
				t.Fatalf("sync = %v, want %v", cfg.CurrencyRate.SyncEnabled, tt.wantSync)
			}
		})
	}
}
