package utils

import (
	"encoding/json"
	"testing"
)

func TestRound4(t *testing.T) { // กฎข้อ 9
	tests := []struct {
		in, want float64
	}{
		{0.1 + 0.2, 0.3},
		{10000.12346, 10000.1235},
		{10000.12344, 10000.1234},
		{90.5, 90.5},
		{-1.00005, -1.0001},
		{0, 0},
	}
	for _, tt := range tests {
		if got := Round4(tt.in); got != tt.want {
			t.Errorf("Round4(%v) = %v, want %v", tt.in, got, tt.want)
		}
	}
	sum := 0.0
	for i := 0; i < 10; i++ {
		sum = Round4(sum + 0.1)
	}
	if sum != 1 {
		t.Errorf("บวก 0.1 สิบครั้งแบบปัดทุกครั้ง = %v, want 1", sum)
	}
}

func TestIsDecimal4(t *testing.T) { // กฎข้อ 9 · ACC-18 · MGMT-17 (รับเป็น float64 — 2026-10-12)
	tests := []struct {
		json string
		want bool
	}{
		{`70`, true},
		{`0.5`, true},
		{`0`, true},
		{`1234.56`, true},
		{`10000.1234`, true},
		{`10000.00`, true},
		{`-1.5`, true},
		{`0.1`, true},
		{`99999999999.9999`, true},
		{`1e2`, true}, // JSON number รูป exponent = 100
		{`30.12345`, false},
		{`0.00001`, false},
		{`123456789012`, false}, // ส่วนจำนวนเต็ม 12 หลัก
		{`100000000000`, false},
	}
	for _, tt := range tests {
		var v float64
		if err := json.Unmarshal([]byte(tt.json), &v); err != nil {
			t.Fatalf("%s: %v", tt.json, err)
		}
		if got := IsDecimal4(v); got != tt.want {
			t.Errorf("IsDecimal4(%s) = %v, want %v", tt.json, got, tt.want)
		}
	}
}

func TestFloatJSON(t *testing.T) { // API ส่งค่าตามที่เก็บ ตัด 0 ท้าย (ACC-18)
	tests := []struct {
		in   float64
		want string
	}{
		{10000.5, "10000.5"},
		{10000, "10000"},
		{0.1235, "0.1235"},
		{962056.25, "962056.25"},
		{0, "0"},
	}
	for _, tt := range tests {
		b, _ := json.Marshal(tt.in)
		if string(b) != tt.want {
			t.Errorf("json(%v) = %s, want %s", tt.in, b, tt.want)
		}
		if got := FormatNum(tt.in); got != tt.want {
			t.Errorf("FormatNum(%v) = %s, want %s", tt.in, got, tt.want)
		}
	}
}

func TestFormatNumGrouped(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{1000000, "1,000,000"},
		{1234.5, "1,234.5"},
		{999, "999"},
		{100000, "100,000"},
		{0, "0"},
		{-12345.6789, "-12,345.6789"},
	}
	for _, tt := range tests {
		if got := FormatNumGrouped(tt.in); got != tt.want {
			t.Errorf("FormatNumGrouped(%v) = %s, want %s", tt.in, got, tt.want)
		}
	}
}
