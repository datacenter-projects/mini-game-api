package asset

import "testing"

func validTheme() Theme {
	return Theme{FontFamily: "kanit", MainColor1: "#1E90FF", MainColor2: "#FFFFFF",
		StatsBarColor: StatsBarColorMain2, StatsBarFontFamily: "prompt"}
}

func TestNormalizeTheme(t *testing.T) { // AST-33 – AST-36, AST-38
	t.Run("ผ่าน · สีถูกแปลงเป็นตัวพิมพ์เล็ก", func(t *testing.T) {
		got, v := NormalizeTheme(validTheme())
		if v != nil {
			t.Fatalf("ต้องผ่าน: %+v", v)
		}
		if got.MainColor1 != "#1e90ff" || got.MainColor2 != "#ffffff" {
			t.Errorf("สีต้องเป็นตัวพิมพ์เล็ก: %+v", got)
		}
	})

	bad := []struct {
		name   string
		mutate func(*Theme)
		field  string
		reason ThemeReason
	}{
		{"AST-33 ขาด font_family", func(x *Theme) { x.FontFamily = "" }, "font_family", ThemeRequired},
		{"AST-33 ขาด main_color_1", func(x *Theme) { x.MainColor1 = "" }, "main_color_1", ThemeRequired},
		{"AST-33 ขาด main_color_2", func(x *Theme) { x.MainColor2 = "" }, "main_color_2", ThemeRequired},
		{"AST-33 ขาด stats_bar_color", func(x *Theme) { x.StatsBarColor = "" }, "stats_bar_color", ThemeRequired},
		{"AST-33 ขาด stats_bar_font_family", func(x *Theme) { x.StatsBarFontFamily = "" }, "stats_bar_font_family", ThemeRequired},
		{"AST-34 ฟอนต์ตัวพิมพ์ใหญ่", func(x *Theme) { x.FontFamily = "Kanit" }, "font_family", ThemeBadFont},
		{"AST-34 ฟอนต์นอก whitelist", func(x *Theme) { x.FontFamily = "Comic Sans" }, "font_family", ThemeBadFont},
		{"AST-34 CSS injection", func(x *Theme) { x.FontFamily = "noto; color:red" }, "font_family", ThemeBadFont},
		{"AST-34 stats bar font นอก whitelist", func(x *Theme) { x.StatsBarFontFamily = "Arial" }, "stats_bar_font_family", ThemeBadFont},
		{"AST-35 #RGB", func(x *Theme) { x.MainColor1 = "#FFF" }, "main_color_1", ThemeBadColor},
		{"AST-35 ไม่มี #", func(x *Theme) { x.MainColor1 = "FF0000" }, "main_color_1", ThemeBadColor},
		{"AST-35 ไม่ใช่ hex", func(x *Theme) { x.MainColor2 = "#GG0000" }, "main_color_2", ThemeBadColor},
		{"AST-35 ยาวเกิน", func(x *Theme) { x.MainColor2 = "#ff00001" }, "main_color_2", ThemeBadColor},
		{"AST-36 stats_bar_color เป็น hex", func(x *Theme) { x.StatsBarColor = "#ffffff" }, "stats_bar_color", ThemeBadStatsBarColor},
		{"AST-36 stats_bar_color ชื่ออื่น", func(x *Theme) { x.StatsBarColor = "main_color_3" }, "stats_bar_color", ThemeBadStatsBarColor},
	}
	for _, tt := range bad {
		th := validTheme()
		tt.mutate(&th)
		_, v := NormalizeTheme(th)
		if v == nil {
			t.Errorf("%s: ต้องไม่ผ่าน", tt.name)
			continue
		}
		if v.Field != tt.field || v.Reason != tt.reason {
			t.Errorf("%s: got %+v, want field=%s reason=%d", tt.name, *v, tt.field, tt.reason)
		}
	}
}

func TestStatsBarColorHex(t *testing.T) { // AST-36
	th := Theme{MainColor1: "#111111", MainColor2: "#222222", StatsBarColor: StatsBarColorMain1}
	if got := StatsBarColorHex(th); got != "#111111" {
		t.Errorf("main_color_1: got %s", got)
	}
	th.StatsBarColor = StatsBarColorMain2
	if got := StatsBarColorHex(th); got != "#222222" {
		t.Errorf("main_color_2: got %s", got)
	}
	th.MainColor2 = "#333333" // เปลี่ยนสีหลัก แถบสถิติต้องตาม
	if got := StatsBarColorHex(th); got != "#333333" {
		t.Errorf("ต้องตามสีหลักที่เปลี่ยน: got %s", got)
	}
}

func TestDefaultThemePassesRules(t *testing.T) { // AST-37: ค่ากลาง seed ต้องผ่านกฎเดียวกัน (AST-38)
	if _, v := NormalizeTheme(DefaultTheme()); v != nil {
		t.Errorf("DefaultTheme ต้องผ่านกฎ: %+v", *v)
	}
	if d := DefaultTheme(); d.MainColor1 != "#f7447e" || d.FontFamily != "noto" {
		t.Errorf("DefaultTheme ไม่ตรง seed: %+v", d)
	}
}

func TestIsHexColor(t *testing.T) {
	ok := []string{"#000000", "#ffffff", "#AbCdEf", "#123456"}
	no := []string{"", "#", "#fff", "000000", "#00000", "#0000000", "#12345g", " #123456", "#12 456"}
	for _, s := range ok {
		if !IsHexColor(s) {
			t.Errorf("%q ต้องผ่าน", s)
		}
	}
	for _, s := range no {
		if IsHexColor(s) {
			t.Errorf("%q ต้องไม่ผ่าน", s)
		}
	}
}
