package configs

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestLoadAPIKeyEncryptionKey(t *testing.T) { // account ACC-04
	key32 := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))
	key16 := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 16)))
	tests := []struct {
		name        string
		env, value  string
		wantErr     bool
		wantMissing bool
		wantLen     int
	}{
		{"local ไม่ตั้ง", "local", "", false, false, 0},
		{"prod ไม่ตั้ง", "prod", "", false, true, 0},
		{"dev ตั้งถูก", "dev", key32, false, false, 32},
		{"local ตั้งถูก", "local", key32, false, false, 32},
		{"ไม่ใช่ base64", "local", "not-base64!!", true, false, 0},
		{"ยาวไม่ใช่ 32 byte", "uat", key16, true, false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("API_KEY_ENCRYPTION_KEY", tt.value)
			var missing []string
			required := func(k string) string {
				v := tt.value
				if v == "" {
					missing = append(missing, k)
				}
				return v
			}
			cfg := &Config{AppEnv: tt.env}
			err := loadAPIKeyEncryptionKey(cfg, required)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if (len(missing) > 0) != tt.wantMissing {
				t.Fatalf("missing = %v, wantMissing %v", missing, tt.wantMissing)
			}
			if len(cfg.Account.APIKeyEncryptionKey) != tt.wantLen {
				t.Fatalf("key len = %d, want %d", len(cfg.Account.APIKeyEncryptionKey), tt.wantLen)
			}
		})
	}
}
