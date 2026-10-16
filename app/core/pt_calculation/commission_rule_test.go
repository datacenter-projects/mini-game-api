package ptcalculation

import (
	"math"
	"reflect"
	"testing"
)

func TestNetCommission(t *testing.T) { // PTC-13 แบบ ก · ตัวเลขตาม spec หัวข้อ 7.2
	tests := []struct {
		name      string
		pt        []float64
		com       []float64 // L0 … Lk (L0 ไม่ใช้)
		memberCom float64
		want      []float64 // L0 … Lk + Member
	}{
		{"reference ข้อ 4", []float64{20, 25, 25, 30}, []float64{0, 1, 0.7, 0.3}, 1, []float64{-0.2, -0.115, 0.105, -0.79, 1}},
		{"Agent ให้ส่วนลด Member อย่างเดียว", []float64{77, 14, 4, 5}, []float64{0, 0, 0, 0}, 0.1, []float64{0, 0, 0, -0.1, 0.1}},
		{"Share ให้ Agent 0.3 · Agent ให้ Member 0.1", []float64{77, 14, 4, 5}, []float64{0, 0, 0, 0.3}, 0.1, []float64{0, 0, -0.285, 0.185, 0.1}},
		{"ตั้งครบทุกชั้น", []float64{77, 14, 4, 5}, []float64{0, 0.5, 0.4, 0.3}, 0.1, []float64{-0.385, 0.021, 0.079, 0.185, 0.1}},
		{"Agent ซ้อน", []float64{10, 10, 15, 25, 40}, []float64{0, 1, 0.7, 0.3, 0.2}, 0.5, []float64{-0.1, -0.04, 0.035, -0.015, -0.38, 0.5}},
		{"Company 1 to 1 สร้าง Member", []float64{20, 80}, []float64{0, 0.5}, 0.3, []float64{-0.1, -0.2, 0.3}},
		{"ไม่มี commission เลย", []float64{30, 70}, []float64{0, 0}, 0, []float64{0, 0, 0}},
	}
	for _, tt := range tests {
		got := NetCommission(tt.pt, tt.com, tt.memberCom)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: %v want %v", tt.name, got, tt.want)
		}
		sum := 0.0
		for _, v := range got {
			sum += v
			if v == 0 && math.Signbit(v) {
				t.Errorf("%s: มี -0 ใน %v", tt.name, got)
			}
		}
		if round4(sum) != 0 {
			t.Errorf("%s: รวม %v ≠ 0", tt.name, round4(sum))
		}
	}
}
