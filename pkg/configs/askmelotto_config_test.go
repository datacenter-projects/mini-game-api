package configs

import "testing"

func TestLoadAskmelotto(t *testing.T) { // currency_rate หัวข้อ 6.3 · config test
	tests := []struct {
		name        string
		env, url    string
		wantErr     bool
		wantMissing bool
		wantURL     string
	}{
		{"local ไม่ตั้ง = ค่าเริ่มต้น", "local", "", false, false, defaultAskmelottoBaseURL},
		{"dev ไม่ตั้ง = ค่าเริ่มต้น", "dev", "", false, false, defaultAskmelottoBaseURL},
		{"dev ตั้งเอง", "dev", "https://dev-api.example", false, false, "https://dev-api.example"},
		{"มี port", "local", "http://localhost:9000", false, false, "http://localhost:9000"},
		{"prod ไม่ตั้ง", "prod", "", false, true, ""},
		{"uat ไม่ตั้ง", "uat", "", false, true, ""},
		{"uat ตั้ง", "uat", "https://api.example", false, false, "https://api.example"},
		{"prod ตั้ง", "prod", "https://api.example", false, false, "https://api.example"},
		{"มี path", "local", "https://api.example/api", true, false, ""},
		{"มี / ท้าย", "local", "https://api.example/", true, false, ""},
		{"มี query", "local", "https://api.example?x=1", true, false, ""},
		{"มี user", "local", "https://u:p@api.example", true, false, ""},
		{"ไม่มี scheme", "local", "api.example", true, false, ""},
		{"scheme อื่น", "local", "ftp://api.example", true, false, ""},
		{"prod ตั้งผิดรูปแบบ", "prod", "https://api.example/x", true, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ASKMELOTTO_API_BASE_URL", tt.url)
			var missing []string
			required := func(k string) string {
				if tt.url == "" {
					missing = append(missing, k)
				}
				return tt.url
			}
			cfg := &Config{AppEnv: tt.env}
			err := loadAskmelotto(cfg, required)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if (len(missing) > 0) != tt.wantMissing {
				t.Fatalf("missing = %v", missing)
			}
			if !tt.wantErr && cfg.Askmelotto.BaseURL != tt.wantURL {
				t.Fatalf("url = %q, want %q", cfg.Askmelotto.BaseURL, tt.wantURL)
			}
		})
	}
}

func TestLoadIsCronjob(t *testing.T) { // IS_CRONJOB ค่าเริ่มต้น false (CR-11)
	orig := Cfg
	t.Cleanup(func() { Cfg = orig })
	tests := []struct {
		val  string
		want bool
	}{{"", false}, {"false", false}, {"true", true}}
	for _, tt := range tests {
		setRequiredEnv(t)
		t.Setenv("APP_ENV", "dev")
		t.Setenv("IS_CRONJOB", tt.val)
		if err := Load(); err != nil {
			t.Fatalf("IS_CRONJOB=%q: %v", tt.val, err)
		}
		if Cfg.IsCronjob != tt.want {
			t.Errorf("IS_CRONJOB=%q: IsCronjob = %v want %v", tt.val, Cfg.IsCronjob, tt.want)
		}
	}
}
