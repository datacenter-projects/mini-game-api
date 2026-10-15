package asset

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ShadeSteps — ชื่อ 19 ระดับของเฉดสี เรียงจากอ่อนไปเข้ม (AST-39) · index 9 ("900") = สีหลักตรงตัว
var ShadeSteps = []string{
	"50", "100", "200", "300", "400", "500", "600", "700", "800",
	"900",
	"1000", "1100", "1200", "1300", "1400", "1500", "1600", "1700", "1800",
}

const (
	baseShadeIndex = 9
	shadeDivisor   = 10.0 // หาร 10 ไม่ใช่ 9 — ปลายทั้งสองข้างไม่ถึงขาว / ดำสนิท (AST-39)
)

// Shades — ชุดเฉด 19 ระดับจากสีหลัก #RRGGBB (AST-39)
//   - index < 9: ผสมขาว t = (9 − i) / 10 → c + (255 − c) × t
//   - index > 9: ผสมดำ k = (10 − (i − 9)) / 10 → c × k
//
// คำนวณบน RGB (ไม่ใช่ HSL) · ปัดแบบ round-half-up แล้ว clamp 0–255 · hex ตัวพิมพ์เล็ก
func Shades(hex string) (map[string]string, error) {
	r, g, b, err := hexToRGB(hex)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(ShadeSteps))
	for i, step := range ShadeSteps {
		switch {
		case i == baseShadeIndex:
			out[step] = rgbToHex(r, g, b)
		case i < baseShadeIndex:
			t := float64(baseShadeIndex-i) / shadeDivisor
			out[step] = rgbToHex(mixToWhite(r, t), mixToWhite(g, t), mixToWhite(b, t))
		default:
			k := (shadeDivisor - float64(i-baseShadeIndex)) / shadeDivisor
			out[step] = rgbToHex(r*k, g*k, b*k)
		}
	}
	return out, nil
}

func mixToWhite(v, t float64) float64 { return v + (255-v)*t }

func hexToRGB(hex string) (r, g, b float64, err error) {
	if !IsHexColor(hex) {
		return 0, 0, 0, fmt.Errorf("invalid hex color %q: expected #RRGGBB", hex)
	}
	h := strings.TrimPrefix(hex, "#")
	vals := [3]float64{}
	for i := range vals {
		v, perr := strconv.ParseUint(h[i*2:i*2+2], 16, 8)
		if perr != nil {
			return 0, 0, 0, perr
		}
		vals[i] = float64(v)
	}
	return vals[0], vals[1], vals[2], nil
}

func rgbToHex(r, g, b float64) string {
	return fmt.Sprintf("#%02x%02x%02x", channel(r), channel(g), channel(b))
}

// channel — math.Round ปัดครึ่งออกจาก 0 = round-half-up สำหรับค่าบวก
func channel(v float64) int {
	n := int(math.Round(v))
	if n < 0 {
		return 0
	}
	if n > 255 {
		return 255
	}
	return n
}
