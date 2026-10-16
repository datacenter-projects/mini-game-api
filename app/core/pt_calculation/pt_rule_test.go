package ptcalculation

import (
	"errors"
	"reflect"
	"testing"
)

// on — ชั้นที่รับ PT · P = ค่าที่ได้รับ · F / R = Force / Remain
func on(p, f, r float64) Layer { return Layer{Received: p, Force: f, Remain: r, AcceptsPT: true} }

func off(p, f, r float64) Layer { return Layer{Received: p, Force: f, Remain: r} }

func csm(p, f float64) Layer { return Layer{Received: p, Force: f, AcceptsPT: true, IsCSM: true} }

var sa = Layer{Received: FullPT, AcceptsPT: true}

func TestCalculate(t *testing.T) { // PTC-02 – PTC-07 · ชุดข้อมูลตาม spec หัวข้อ 7.1
	tests := []struct {
		name  string
		in    Input
		want  []float64
		warns []Warning
	}{
		{"A reference ข้อ 3", Input{[]Layer{sa, on(95, 0, 0), on(85, 20, 70), on(70, 30, 40)}, 20}, []float64{20, 25, 25, 30}, nil},
		{"B Agent ซ้อน", Input{[]Layer{sa, on(90, 0, 0), on(80, 0, 0), on(70, 0, 60), on(50, 40, 45)}, 30}, []float64{15, 10, 10, 25, 40}, nil},
		{"C Company 1 to 1 สร้าง Member", Input{[]Layer{sa, on(90, 80, 0)}, 60}, []float64{20, 80}, nil},
		{"D ไม่มี Force / Remain", Input{[]Layer{sa, on(90, 0, 0), on(70, 0, 0), on(60, 0, 0)}, 40}, []float64{30, 20, 10, 40}, nil},
		{"E ชั้นกลางไม่รับ PT (lead Q2)", Input{[]Layer{sa, on(95, 0, 0), off(85, 20, 70), on(70, 30, 40)}, 20}, []float64{20, 40, 10, 30}, nil},
		{"E2 ผู้สร้าง Member ไม่รับ PT", Input{[]Layer{sa, on(95, 0, 0), on(85, 20, 70), off(70, 30, 40)}, 20}, []float64{20, 25, 55, 0}, nil},
		{"F CSM มี force ค้าง", Input{[]Layer{sa, csm(80, 60), on(80, 0, 0)}, 50}, []float64{50, 0, 50},
			[]Warning{{Kind: WarnCSMForce, Layer: 1, Want: 60}}},
		{"Force = s พอดี ไม่ทำงาน", Input{[]Layer{sa, on(90, 0, 0), on(60, 40, 0)}, 40}, []float64{30, 30, 40}, nil},
		{"Force แล้ว s ≥ Remain — Remain ไม่ทำงาน", Input{[]Layer{sa, on(90, 0, 0), on(60, 50, 45)}, 40}, []float64{20, 30, 50}, nil},
		{"Member pt = 0 ทุกอย่างคืน Superadmin", Input{[]Layer{sa, on(90, 0, 0), on(60, 0, 0)}, 0}, []float64{70, 30, 0}, nil},
		{"เศษทศนิยม (PTC-07)", Input{[]Layer{sa, on(66.6667, 0, 0), on(33.3333, 22.2222, 0)}, 11.1111}, []float64{44.4444, 33.3334, 22.2222}, nil},
		{"Force เกิน left (ข้อมูลผิด MGMT-18 · PTC-10)", Input{[]Layer{sa, on(90, 0, 0), on(50, 80, 0)}, 30}, []float64{10, 40, 50},
			[]Warning{{Kind: WarnCappedByLeft, Layer: 2, Want: 50, Got: 20}}},
	}
	for _, tt := range tests {
		got, err := Calculate(tt.in)
		if err != nil {
			t.Errorf("%s: err %v", tt.name, err)
			continue
		}
		if !reflect.DeepEqual(got.PT, tt.want) {
			t.Errorf("%s: pt %v want %v", tt.name, got.PT, tt.want)
		}
		if !reflect.DeepEqual(got.Warnings, tt.warns) {
			t.Errorf("%s: warnings %+v want %+v", tt.name, got.Warnings, tt.warns)
		}
	}
}

func TestCalculateInvalidChain(t *testing.T) { // PTC-09
	tests := []struct {
		name string
		in   Input
	}{
		{"สายชั้นเดียว", Input{[]Layer{sa}, 0}},
		{"ลูกได้รับเกินผู้สร้าง", Input{[]Layer{sa, on(60, 0, 0), on(70, 0, 0)}, 10}},
		{"pt ของ Member เกินที่ผู้สร้างได้รับ", Input{[]Layer{sa, on(60, 0, 0)}, 61}},
		{"pt ของ Member ติดลบ", Input{[]Layer{sa, on(60, 0, 0)}, -1}},
		{"force เกิน 100", Input{[]Layer{sa, on(60, 101, 0)}, 10}},
	}
	for _, tt := range tests {
		if _, err := Calculate(tt.in); !errors.Is(err, ErrInvalidChain) {
			t.Errorf("%s: err %v want ErrInvalidChain", tt.name, err)
		}
	}
}
