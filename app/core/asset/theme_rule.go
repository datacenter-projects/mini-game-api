package asset

import "strings"

// ธีมเกม (AST-33 – AST-38) — ฟอนต์ + สี 5 ค่า เขียนทั้งชุดเสมอ

// Theme — 5 ค่าของธีมเกม
type Theme struct {
	FontFamily         string
	MainColor1         string // #rrggbb ตัวพิมพ์เล็ก
	MainColor2         string
	StatsBarColor      string // ชื่อสี "main_color_1" / "main_color_2" ไม่ใช่ hex (AST-36)
	StatsBarFontFamily string
}

const (
	StatsBarColorMain1 = "main_color_1"
	StatsBarColorMain2 = "main_color_2"
)

// Fonts — whitelist ฟอนต์ ตัวพิมพ์ตรงตัว (AST-34)
var Fonts = []string{"noto", "prompt", "kanit", "mitr", "sarabun"}

// DefaultTheme — ค่ากลางเริ่มต้นที่ seed เป็นแถว GLOBAL (AST-37)
func DefaultTheme() Theme {
	return Theme{FontFamily: "noto", MainColor1: "#f7447e", MainColor2: "#ffffff",
		StatsBarColor: StatsBarColorMain1, StatsBarFontFamily: "prompt"}
}

// ThemeReason — เหตุที่ธีมไม่ผ่าน
type ThemeReason int

const (
	ThemeRequired ThemeReason = iota + 1 // ไม่ส่ง / ว่าง (ไม่มี partial update — AST-33)
	ThemeBadFont
	ThemeBadColor
	ThemeBadStatsBarColor
)

// ThemeViolation — field ที่ผิดและเหตุ (ใช้ทำ msg 422 ที่บอก field)
type ThemeViolation struct {
	Field  string // ชื่อ field ใน JSON
	Reason ThemeReason
}

// IsFont — ฟอนต์อยู่ใน whitelist ตัวพิมพ์ตรงตัว
func IsFont(s string) bool {
	for _, f := range Fonts {
		if f == s {
			return true
		}
	}
	return false
}

// IsHexColor — ตรง ^#[0-9A-Fa-f]{6}$ เท่านั้น ไม่รับ #RGB (AST-35)
func IsHexColor(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	for _, c := range s[1:] {
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f', c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}

// NormalizeTheme — ตรวจครบ 5 ค่าแล้วแปลงสีเป็นตัวพิมพ์เล็ก (AST-33 – AST-36 · AST-38)
// ตรวจตามลำดับ field ใน request · คืน violation ตัวแรกที่เจอ (ค่าที่ผ่านแล้ว = nil)
func NormalizeTheme(in Theme) (Theme, *ThemeViolation) {
	if in.FontFamily == "" {
		return Theme{}, &ThemeViolation{"font_family", ThemeRequired}
	}
	if !IsFont(in.FontFamily) {
		return Theme{}, &ThemeViolation{"font_family", ThemeBadFont}
	}
	if in.MainColor1 == "" {
		return Theme{}, &ThemeViolation{"main_color_1", ThemeRequired}
	}
	if !IsHexColor(in.MainColor1) {
		return Theme{}, &ThemeViolation{"main_color_1", ThemeBadColor}
	}
	if in.MainColor2 == "" {
		return Theme{}, &ThemeViolation{"main_color_2", ThemeRequired}
	}
	if !IsHexColor(in.MainColor2) {
		return Theme{}, &ThemeViolation{"main_color_2", ThemeBadColor}
	}
	if in.StatsBarColor == "" {
		return Theme{}, &ThemeViolation{"stats_bar_color", ThemeRequired}
	}
	if in.StatsBarColor != StatsBarColorMain1 && in.StatsBarColor != StatsBarColorMain2 {
		return Theme{}, &ThemeViolation{"stats_bar_color", ThemeBadStatsBarColor}
	}
	if in.StatsBarFontFamily == "" {
		return Theme{}, &ThemeViolation{"stats_bar_font_family", ThemeRequired}
	}
	if !IsFont(in.StatsBarFontFamily) {
		return Theme{}, &ThemeViolation{"stats_bar_font_family", ThemeBadFont}
	}
	in.MainColor1 = strings.ToLower(in.MainColor1)
	in.MainColor2 = strings.ToLower(in.MainColor2)
	return in, nil
}

// StatsBarColorHex — hex ที่ resolve แล้วของสีแถบสถิติ (AST-36)
func StatsBarColorHex(t Theme) string {
	if t.StatsBarColor == StatsBarColorMain2 {
		return t.MainColor2
	}
	return t.MainColor1
}
