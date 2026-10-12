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
