package currencyrate

import (
	"testing"
	"time"
)

func TestRequestDate(t *testing.T) { // CR-04
	tests := []struct {
		now  time.Time
		want string
	}{
		{time.Date(2026, 10, 7, 23, 30, 0, 0, time.UTC), "2026-10-08"}, // 06:30 ของวันถัดไปเวลาไทย
		{time.Date(2026, 10, 7, 16, 59, 59, 0, time.UTC), "2026-10-07"},
		{time.Date(2026, 10, 7, 17, 0, 0, 0, time.UTC), "2026-10-08"},
	}
	for _, tt := range tests {
		if got := RequestDate(tt.now); got != tt.want {
			t.Errorf("RequestDate(%v) = %s, want %s", tt.now, got, tt.want)
		}
	}
}

func TestParseRate(t *testing.T) { // CR-06
	tests := []struct {
		in   string
		want int64
		ok   bool
	}{
		{"33.62", 3362000000, true},
		{"1", 100000000, true},
		{"25988.07", 2598807000000, true},
		{"0.00000001", 1, true},
		{"6.7", 670000000, true},
		{"0", 0, false},
		{"0.0", 0, false},
		{"-1", 0, false},
		{"0.000000001", 0, false}, // ทศนิยม 9 ตำแหน่ง
		{"1e3", 0, false},
		{"", 0, false},
		{".5", 0, false},
		{"5.", 0, false},
		{"abc", 0, false},
		{"99999999999999999999", 0, false}, // ใหญ่เกิน int64 หลังคูณ 10^8
	}
	for _, tt := range tests {
		got, ok := ParseRate(tt.in)
		if got != tt.want || ok != tt.ok {
			t.Errorf("ParseRate(%q) = %d, %v · want %d, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestCurrencies(t *testing.T) {
	if len(Currencies) != 27 {
		t.Fatalf("ต้องมี 27 สกุล ได้ %d", len(Currencies))
	}
	if !IsSupported("THB") || !IsSupported("USDT") || IsSupported("XYZ") {
		t.Fatal("IsSupported ผิด")
	}
}

func TestConvertMinor(t *testing.T) { // CR-13
	thb, _ := ParseRate("33.62")
	usd, _ := ParseRate("0.99")
	usdt, _ := ParseRate("1")
	vnd, _ := ParseRate("25988.07")
	tests := []struct {
		name             string
		amount, from, to int64
		want             int64
	}{
		{"100.00 THB → USD", 10000, thb, usd, 294}, // 2.9446… → 2.94
		{"1,000.00 USDT → VND", 100000, usdt, vnd, 2598807000},
		{"สกุลเดียวกัน", 12345, thb, thb, 12345},
		{"ปัดครึ่งขึ้น", 1, 2, 1, 1}, // 0.5 → 1
		{"ปัดลง", 1, 3, 1, 0},        // 0.333 → 0
		{"ค่าลบ", -1, 2, 1, -1},      // -0.5 → -1
		{"ศูนย์", 0, thb, usd, 0},
	}
	for _, tt := range tests {
		if got := ConvertMinor(tt.amount, tt.from, tt.to); got != tt.want {
			t.Errorf("%s: got %d, want %d", tt.name, got, tt.want)
		}
	}
}
