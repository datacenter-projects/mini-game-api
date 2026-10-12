package asset

import (
	"strings"
	"testing"
)

func TestAssetObjectKey(t *testing.T) { // AST-49
	tests := []struct {
		name  string
		scope Scope
		agent uint
		want  string
	}{
		{"GLOBAL", ScopeGlobal, 0, "games/scratch_card/global/icon_button/1759900000000000000-a1b2c3d4.png"},
		{"AGENT", ScopeAgent, 12, "games/scratch_card/agents/12/icon_button/1759900000000000000-a1b2c3d4.png"},
	}
	for _, tt := range tests {
		got := AssetObjectKey(tt.scope, tt.agent, "scratch_card", "icon_button", 1759900000000000000, "a1b2c3d4", "png")
		if got != tt.want {
			t.Errorf("%s: got %s, want %s", tt.name, got, tt.want)
		}
	}
}

func TestLogoObjectKey(t *testing.T) {
	got := LogoObjectKey(12, 1759900000000000000, "0a1b2c3d", "webp")
	if got != "sites/12/logo/1759900000000000000-0a1b2c3d.webp" {
		t.Errorf("got %s", got)
	}
}

func TestObjectKeysStartWithIAMPrefixes(t *testing.T) { // IAM อนุญาตเฉพาะ games/* และ sites/* (lead 7.3)
	for _, k := range []string{
		AssetObjectKey(ScopeGlobal, 0, "coin_toss", "background", 1, "x", "png"),
		AssetObjectKey(ScopeAgent, 5, "coin_toss", "background", 1, "x", "png"),
		LogoObjectKey(5, 1, "x", "png"),
	} {
		if !strings.HasPrefix(k, "games/") && !strings.HasPrefix(k, "sites/") {
			t.Errorf("key ออกนอก prefix ที่ IAM อนุญาต: %s", k)
		}
	}
}

func TestObjectURL(t *testing.T) { // AST-49
	key := "games/scratch_card/global/icon_button/1-a.png"
	tests := []struct {
		name, base, key, want string
	}{
		{"ปกติ", "https://cdn.example.com", key, "https://cdn.example.com/" + key},
		{"base มี / ท้าย ไม่ซ้อน //", "https://cdn.example.com/", key, "https://cdn.example.com/" + key},
		{"key มี / หน้า", "https://cdn.example.com", "/" + key, "https://cdn.example.com/" + key},
		{"ทั้งคู่มี /", "https://cdn.example.com//", "//" + key, "https://cdn.example.com/" + key},
		{"ไม่ตั้ง CDN คืน key", "", key, key},
		{"key ว่าง = \"\" (ACC-32)", "https://cdn.example.com", "", ""},
	}
	for _, tt := range tests {
		if got := ObjectURL(tt.base, tt.key); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestNormalizeSiteName(t *testing.T) { // AST-42 · lead GE13
	tests := []struct {
		name string
		in   string
		want string
		v    NameViolation
	}{
		{"ปกติ", "Share One", "Share One", NameOK},
		{"ตัดช่องว่างหน้า-หลัง", "  Share One  ", "Share One", NameOK},
		{"ตัด tab / newline หน้า-หลัง", "\tShare One\n", "Share One", NameOK},
		{"ภาษาไทย", "ร้านหนึ่ง", "ร้านหนึ่ง", NameOK},
		{"ช่องว่างกลางชื่อได้", "Share  One", "Share  One", NameOK},
		{"ว่าง", "", "", NameEmpty},
		{"มีแต่ช่องว่าง", "   ", "", NameEmpty},
		{"50 ตัวอักษรพอดี", strings.Repeat("a", 50), strings.Repeat("a", 50), NameOK},
		{"51 ตัวอักษร", strings.Repeat("a", 51), "", NameTooLong},
		{"ไทย 50 ตัวอักษร (เกิน 50 byte) ผ่าน", strings.Repeat("ก", 50), strings.Repeat("ก", 50), NameOK},
		{"ไทย 51 ตัวอักษร", strings.Repeat("ก", 51), "", NameTooLong},
		{"ตัดช่องว่างแล้วจึงนับ (50 + ช่องว่างท้าย)", strings.Repeat("a", 50) + "  ", strings.Repeat("a", 50), NameOK},
		{"newline กลางชื่อ", "Share\nOne", "", NameControlChar},
		{"tab กลางชื่อ", "Share\tOne", "", NameControlChar},
		{"NUL", "Share\x00One", "", NameControlChar},
		{"DEL", "Share\x7fOne", "", NameControlChar},
		{"C1 control", "Share\u0085One", "", NameControlChar},
	}
	for _, tt := range tests {
		got, v := NormalizeSiteName(tt.in)
		if v != tt.v || got != tt.want {
			t.Errorf("%s: got (%q,%d), want (%q,%d)", tt.name, got, v, tt.want, tt.v)
		}
	}
}

func TestGameCodes(t *testing.T) { // AST-10
	want := []string{"coin_toss", "rock_paper_scissors", "scratch_card"}
	got := GameCodes()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %v, want %v", got, want)
		}
	}
	for _, c := range want {
		if !IsGameCode(c) {
			t.Errorf("%s ต้องเป็นรหัสเกม", c)
		}
	}
	for _, c := range []string{"", "plinko", "Coin_Toss", "coin toss", "scratch_card "} {
		if IsGameCode(c) {
			t.Errorf("%q ต้องไม่ใช่รหัสเกม (เทียบตรงทุกตัวอักษร — AST-14)", c)
		}
	}
}
