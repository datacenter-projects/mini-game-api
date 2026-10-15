package configs

import (
	"strings"
	"testing"
)

func TestParseTrustedProxies(t *testing.T) { // S1-a
	got, err := ParseTrustedProxies([]string{"10.0.0.0/8", " 172.16.5.4 ", "", "fd00::/8", "::ffff:192.168.1.1", "10.1.2.3/8"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"10.0.0.0/8", "172.16.5.4/32", "fd00::/8", "192.168.1.1/32", "10.0.0.0/8"} // IP เดี่ยว = /32 · mapped IPv6 แปลงเป็น IPv4 · CIDR ถูก mask
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i, p := range got {
		if p.String() != want[i] {
			t.Errorf("[%d] = %s want %s", i, p, want[i])
		}
	}
	if got, err := ParseTrustedProxies(nil); err != nil || len(got) != 0 {
		t.Errorf("ไม่ตั้ง = ว่างไม่มี error: %v %v", got, err)
	}
	for _, bad := range []string{"bogus", "999.1.1.1/8", "10.0.0.0/99", "10.0.0.0/8 extra", "10.0.0"} {
		if _, err := ParseTrustedProxies([]string{"10.0.0.0/8", bad}); err == nil || !strings.Contains(err.Error(), bad) {
			t.Errorf("%q ต้องคืน error ที่บอกค่าที่ผิด: %v", bad, err)
		}
	}
}

func TestLoadRejectsBadTrustedProxiesAndLobbyURL(t *testing.T) { // S1-a · N5: ค่าพิมพ์ผิด = boot ไม่ขึ้น
	orig := Cfg
	t.Cleanup(func() { Cfg = orig })

	tests := []struct {
		name, proxies, lobby string
		wantErr              string // ข้อความที่ error ต้องมี · ว่าง = ต้องผ่าน
	}{
		{"ไม่ตั้งอะไร", "", "", ""},
		{"proxies ถูก", "10.0.0.0/8, 172.16.5.4", "https://lobby.example.com/", ""},
		{"proxies พิมพ์ผิด", "10.0.0.0/8,bogus", "", "TRUSTED_PROXIES"},
		{"CIDR ผิด", "10.0.0.0/99", "", "TRUSTED_PROXIES"},
		{"lobby ไม่มี scheme", "", "lobby.example.com", "PLAYER_LOBBY_URL"},
		{"lobby ไม่ใช่ http(s)", "", "ftp://lobby.example.com/", "PLAYER_LOBBY_URL"},
		{"lobby ไม่มี host", "", "https:///path", "PLAYER_LOBBY_URL"},
		{"lobby เป็น path", "", "/relative", "PLAYER_LOBBY_URL"},
		{"lobby มี query อยู่แล้ว", "", "http://localhost:3000/?x=1", ""},
	}
	for _, tt := range tests {
		setRequiredEnv(t)
		t.Setenv("APP_ENV", "dev")
		t.Setenv("TRUSTED_PROXIES", tt.proxies)
		t.Setenv("PLAYER_LOBBY_URL", tt.lobby)
		err := Load()
		switch {
		case tt.wantErr == "" && err != nil:
			t.Errorf("%s: ไม่ควร error: %v", tt.name, err)
		case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
			t.Errorf("%s: ต้อง error ที่มี %q ได้ %v", tt.name, tt.wantErr, err)
		}
	}

	setRequiredEnv(t) // ค่าถูกต้องถูก parse ไว้ครั้งเดียวใน Cfg
	t.Setenv("APP_ENV", "dev")
	t.Setenv("TRUSTED_PROXIES", "10.0.0.0/8,172.16.5.4")
	t.Setenv("PLAYER_LOBBY_URL", "")
	if err := Load(); err != nil || len(Cfg.TrustedProxyPrefixes) != 2 {
		t.Fatalf("prefixes = %v err %v", Cfg.TrustedProxyPrefixes, err)
	}
}
