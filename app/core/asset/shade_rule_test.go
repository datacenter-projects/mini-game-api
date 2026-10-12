package asset

import "testing"

// ค่าคาดหวังคำนวณตามสูตร AST-39 (RGB · หาร 10 · round-half-up · clamp)
// ต้อง pin เทียบกับไฟล์ palette จริงของหน้าบ้านเมื่อได้ไฟล์ (lead GQ7 · AST-40)
func TestShades(t *testing.T) {
	tests := []struct {
		name string
		hex  string
		want map[string]string
	}{
		{"#f7447e (สีกลางตั้งต้น)", "#f7447e", map[string]string{
			"50": "#feecf2", "900": "#f7447e", "1800": "#19070d",
		}},
		{"#1e90ff ครบ 19 ระดับ", "#1e90ff", map[string]string{
			"50": "#e9f4ff", "100": "#d2e9ff", "200": "#bcdeff", "300": "#a5d3ff", "400": "#8fc8ff",
			"500": "#78bcff", "600": "#62b1ff", "700": "#4ba6ff", "800": "#359bff", "900": "#1e90ff",
			"1000": "#1b82e6", "1100": "#1873cc", "1200": "#1565b3", "1300": "#125699", "1400": "#0f4880",
			"1500": "#0c3a66", "1600": "#092b4d", "1700": "#061d33", "1800": "#030e1a",
		}},
		{"#ffffff ขาวล้วน", "#ffffff", map[string]string{
			"50": "#ffffff", "800": "#ffffff", "900": "#ffffff",
			"1000": "#e6e6e6", "1100": "#cccccc", "1200": "#b3b3b3", "1300": "#999999", "1400": "#808080",
			"1500": "#666666", "1600": "#4d4d4d", "1700": "#333333", "1800": "#1a1a1a",
		}},
		{"#000000 ดำล้วน (ปลายไม่ถึงขาวสนิท)", "#000000", map[string]string{
			"50": "#e6e6e6", "100": "#cccccc", "800": "#1a1a1a", "900": "#000000", "1800": "#000000",
		}},
		{"ตัวพิมพ์ใหญ่ ผลเป็นตัวพิมพ์เล็ก", "#1E90FF", map[string]string{"900": "#1e90ff", "50": "#e9f4ff"}},
	}
	for _, tt := range tests {
		got, err := Shades(tt.hex)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if len(got) != 19 {
			t.Errorf("%s: ต้องได้ 19 ระดับ ได้ %d", tt.name, len(got))
		}
		for step, want := range tt.want {
			if got[step] != want {
				t.Errorf("%s: step %s got %s, want %s", tt.name, step, got[step], want)
			}
		}
	}
}

func TestShadesBaseIsOriginal(t *testing.T) { // AST-39: 900 = สีหลักตรงตัว
	for _, hex := range []string{"#f7447e", "#123456", "#0a0b0c", "#fedcba"} {
		got, err := Shades(hex)
		if err != nil {
			t.Fatal(err)
		}
		if got["900"] != hex {
			t.Errorf("900 ของ %s = %s", hex, got["900"])
		}
	}
}

func TestShadesStepNames(t *testing.T) {
	got, _ := Shades("#336699")
	for _, s := range ShadeSteps {
		if _, ok := got[s]; !ok {
			t.Errorf("ขาด step %s", s)
		}
	}
	if len(ShadeSteps) != 19 || ShadeSteps[0] != "50" || ShadeSteps[9] != "900" || ShadeSteps[18] != "1800" {
		t.Errorf("ชื่อ step ผิด: %v", ShadeSteps)
	}
}

func TestShadesInvalidColor(t *testing.T) {
	for _, hex := range []string{"", "#fff", "f7447e", "#gg0000", "#f7447e0"} {
		if _, err := Shades(hex); err == nil {
			t.Errorf("%q ต้อง error", hex)
		}
	}
}
