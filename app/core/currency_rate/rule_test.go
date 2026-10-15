package currencyrate

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"
)

func TestParseRate(t *testing.T) { // rule: CR-03 / CR-15
	tests := []struct {
		in string
		ok bool
	}{
		{"33.62", true}, {"1", true}, {"0.00000001", true}, {"22394.92", true}, {"1337.9", true},
		{"0", false}, {"0.0", false}, {"-1", false}, {"+1", false}, {"", false}, {".5", false}, {"5.", false},
		{"0.123456789", false}, // ทศนิยม 9 ตำแหน่ง
		{"1e3", false}, {"1E-2", false}, {"abc", false}, {"1.2.3", false}, {"١٢", false}, {" 1", false},
	}
	for _, tt := range tests {
		r, ok := ParseRate(tt.in)
		if ok != tt.ok || (ok && r.Sign() <= 0) {
			t.Errorf("ParseRate(%q) ok = %v want %v", tt.in, ok, tt.ok)
		}
	}
}

func TestTHBRate(t *testing.T) { // rule: CR-03
	thb, _ := ParseRate("33.62")
	tests := []struct {
		cur  string
		want float64
	}{
		{"0.99", 33.95959596},  // ตัวอย่างใน spec: 33.62 ÷ 0.99
		{"1", 33.62},           // USDT
		{"33.62", 1},           // THB เทียบตัวเอง
		{"158.15", 0.21258299}, // JPY: 33.62 ÷ 158.15 = 0.2125829…
		{"3", 11.20666667},     // ปัดครึ่งขึ้นที่ตำแหน่งที่ 8 (11.2066666…)
		{"7", 4.80285714},      // 4.802857142… → ...14
		{"0.003", 11206.66666667},
	}
	for _, tt := range tests {
		c, ok := ParseRate(tt.cur)
		if !ok {
			t.Fatalf("ParseRate(%q)", tt.cur)
		}
		got := THBRate(thb, c)
		if math.Abs(got-tt.want) > 5e-9 {
			t.Errorf("THBRate(33.62 / %s) = %.10f want %.8f", tt.cur, got, tt.want)
		}
	}
	// ปัดครึ่งขึ้นจริง ไม่ใช่ปัดลง / banker: 1 ÷ 8 = 0.125 → 8 ตำแหน่ง 0.12500000 (ไม่มีเศษ) · ใช้ 1/3 ตรวจเศษ
	one, _ := ParseRate("1")
	three, _ := ParseRate("3")
	if got := THBRate(one, three); got != 0.33333333 {
		t.Errorf("1/3 = %v want 0.33333333", got)
	}
	two, _ := ParseRate("2")
	if got := THBRate(two, three); got != 0.66666667 { // 0.666… ปัดขึ้น
		t.Errorf("2/3 = %v want 0.66666667", got)
	}
}

func TestBuildRates(t *testing.T) { // rule: CR-03 / CR-04
	raw := map[string]json.RawMessage{}
	for c, v := range map[string]string{"THB": "33.62", "USD": "0.99", "USDT": "1", "EUR": "0.88", "XYZ": "5"} {
		raw[c] = json.RawMessage(v)
	}
	accepted, rejected := ParseRates(raw)
	if len(rejected) != 0 || len(accepted) != 4 { // XYZ ไม่สนใจ
		t.Fatalf("accepted %d rejected %v", len(accepted), rejected)
	}
	rates, ok := BuildRates(accepted)
	if !ok || len(rates) != 4 {
		t.Fatalf("BuildRates ok %v len %d", ok, len(rates))
	}
	byCur := map[string]Rate{}
	for _, r := range rates {
		byCur[r.Currency] = r
	}
	if byCur["THB"].THBRate != 1 || byCur["THB"].USDTRate != 33.62 {
		t.Errorf("THB = %+v", byCur["THB"])
	}
	if byCur["USD"].THBRate != 33.95959596 || byCur["USD"].USDTRate != 0.99 {
		t.Errorf("USD = %+v", byCur["USD"])
	}
	if byCur["USDT"].THBRate != 33.62 {
		t.Errorf("USDT = %+v", byCur["USDT"])
	}
	// เรียงตามรายการ 27 สกุลของระบบ (EUR ก่อน THB ก่อน USD ก่อน USDT)
	order := []string{rates[0].Currency, rates[1].Currency, rates[2].Currency, rates[3].Currency}
	if strings.Join(order, ",") != "EUR,THB,USD,USDT" {
		t.Errorf("order = %v", order)
	}

	// ไม่มี THB = ไม่มีฐานคิด → ไม่บันทึกทั้งเดือน (CR-04)
	delete(raw, "THB")
	accepted, _ = ParseRates(raw)
	if rates, ok := BuildRates(accepted); ok || rates != nil {
		t.Errorf("ไม่มี THB ต้อง ok = false: %v %v", ok, rates)
	}
}

func TestParseRatesRejects(t *testing.T) { // rule: CR-15
	raw := map[string]json.RawMessage{
		"THB": json.RawMessage(`33.62`),
		"USD": json.RawMessage(`"0.99"`),      // string
		"EUR": json.RawMessage(`null`),        // null
		"JPY": json.RawMessage(`0`),           // ศูนย์
		"KRW": json.RawMessage(`-1`),          // ติดลบ
		"IDR": json.RawMessage(`1.123456789`), // ทศนิยม 9
		"GBP": json.RawMessage(`1e2`),         // เลขยกกำลัง
		"CNY": json.RawMessage(`{"a":1}`),     // object
		"XYZ": json.RawMessage(`5`),           // นอกรายการ — ไม่สนใจ
	}
	accepted, rejected := ParseRates(raw)
	if len(accepted) != 1 || accepted["THB"] == nil {
		t.Errorf("accepted = %v", accepted)
	}
	want := map[string]bool{"USD": true, "EUR": true, "JPY": true, "KRW": true, "IDR": true, "GBP": true, "CNY": true}
	if len(rejected) != len(want) {
		t.Fatalf("rejected = %v", rejected)
	}
	for _, c := range rejected {
		if !want[c] {
			t.Errorf("rejected ผิด: %s", c)
		}
	}
}

func TestCheckResponse(t *testing.T) { // rule: CR-15
	ok := Response{Code: 200, BaseCurrency: "USDT", Date: "2026-10-01", JSONRate: map[string]json.RawMessage{}}
	with := func(f func(r *Response)) Response {
		r := ok
		f(&r)
		return r
	}
	tests := []struct {
		name string
		r    Response
		want ResponseProblem
	}{
		{"ปกติ", ok, ResponseOK},
		{"430503 ยังไม่มีเรท", with(func(r *Response) { r.Code = 430503 }), ResponseBadCode},
		{"430502 รูปแบบวันที่ผิด", with(func(r *Response) { r.Code = 430502 }), ResponseBadCode},
		{"base ไม่ใช่ USDT", with(func(r *Response) { r.BaseCurrency = "USD" }), ResponseBadBase},
		{"date ไม่ตรงที่ขอ", with(func(r *Response) { r.Date = "2026-10-16" }), ResponseDateMismatch},
		{"date ว่าง", with(func(r *Response) { r.Date = "" }), ResponseDateMismatch},
		{"ไม่มี json_rate", with(func(r *Response) { r.JSONRate = nil }), ResponseNoRates},
	}
	for _, tt := range tests {
		if got := CheckResponse(tt.r, "2026-10-01"); got != tt.want {
			t.Errorf("%s: got %v want %v", tt.name, got, tt.want)
		}
	}
}

func TestMissingOf(t *testing.T) { // rule: CR-14
	if got := MissingOf(Currencies()); len(got) != 0 {
		t.Errorf("ครบแล้ว = ไม่ขาด: %v", got)
	}
	if got := MissingOf(nil); len(got) != 27 {
		t.Errorf("ไม่มีเลข = ขาด 27: %d", len(got))
	}
	have := []string{}
	for _, c := range Currencies() {
		if c != "KRW" && c != "THB" {
			have = append(have, c)
		}
	}
	if got := MissingOf(have); strings.Join(got, ",") != "KRW,THB" {
		t.Errorf("MissingOf = %v", got)
	}
	if len(Currencies()) != 27 || !IsSupported("USDT") || IsSupported("XYZ") || IsSupported("thb") {
		t.Error("รายการ 27 สกุลผิด")
	}
}

func TestMonthDate(t *testing.T) { // rule: CR-01 · CR-21 — เดือนตามเวลาไทย
	tests := []struct {
		name string
		at   time.Time
		want string
	}{
		{"กลางเดือน", time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC), "2026-10-01"},
		{"31 ต.ค. 23:59 ไทย", time.Date(2026, 10, 31, 23, 59, 0, 0, bangkok), "2026-10-01"},
		{"1 พ.ย. 00:01 ไทย", time.Date(2026, 11, 1, 0, 1, 0, 0, bangkok), "2026-11-01"},
		{"31 ต.ค. 17:30 UTC = 1 พ.ย. 00:30 ไทย", time.Date(2026, 10, 31, 17, 30, 0, 0, time.UTC), "2026-11-01"},
		{"31 ต.ค. 16:59 UTC = 31 ต.ค. 23:59 ไทย", time.Date(2026, 10, 31, 16, 59, 0, 0, time.UTC), "2026-10-01"},
		{"ปีใหม่ 31 ธ.ค. 17:00 UTC = 1 ม.ค. ไทย", time.Date(2026, 12, 31, 17, 0, 0, 0, time.UTC), "2027-01-01"},
	}
	for _, tt := range tests {
		got := MonthDate(tt.at)
		if got.Format("2006-01-02") != tt.want || got.Location() != time.UTC || got.Hour() != 0 {
			t.Errorf("%s: MonthDate = %v want %s (UTC เที่ยงคืน)", tt.name, got, tt.want)
		}
		if RequestDate(got) != tt.want {
			t.Errorf("%s: RequestDate = %s want %s", tt.name, RequestDate(got), tt.want)
		}
	}
}

func TestParseMonth(t *testing.T) { // rule: CR-17
	if m, ok := ParseMonth("2026-10"); !ok || m.Format("2006-01-02") != "2026-10-01" {
		t.Errorf("2026-10 → %v %v", m, ok)
	}
	for _, bad := range []string{"", "2026", "2026-13", "2026-1", "2026-10-01", "oct", "26-10"} {
		if _, ok := ParseMonth(bad); ok {
			t.Errorf("ParseMonth(%q) ต้องไม่ผ่าน", bad)
		}
	}
}

func TestIsEarlyMonth(t *testing.T) { // rule: CR-16 — ก่อน 03:00 ของวันที่ 1 เวลาไทย
	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{"วันที่ 1 00:00 ไทย", time.Date(2026, 11, 1, 0, 0, 0, 0, bangkok), true},
		{"วันที่ 1 02:59 ไทย", time.Date(2026, 11, 1, 2, 59, 0, 0, bangkok), true},
		{"วันที่ 1 03:00 ไทย", time.Date(2026, 11, 1, 3, 0, 0, 0, bangkok), false},
		{"วันที่ 1 12:00 ไทย", time.Date(2026, 11, 1, 12, 0, 0, 0, bangkok), false},
		{"วันที่ 2 01:00 ไทย", time.Date(2026, 11, 2, 1, 0, 0, 0, bangkok), false},
		{"31 ต.ค. 17:30 UTC = 1 พ.ย. 00:30 ไทย", time.Date(2026, 10, 31, 17, 30, 0, 0, time.UTC), true},
		{"กลางเดือน", time.Date(2026, 10, 16, 1, 0, 0, 0, bangkok), false},
	}
	for _, tt := range tests {
		if got := IsEarlyMonth(tt.now); got != tt.want {
			t.Errorf("%s: IsEarlyMonth = %v want %v", tt.name, got, tt.want)
		}
	}
}

func TestConvertTHB(t *testing.T) { // rule: CR-22
	tests := []struct {
		name   string
		amount float64
		rate   float64
		want   float64
	}{
		{"100 USD ตัวอย่างใน spec", 100, 33.95959596, 3395.9596},
		{"บาทต่อบาท", 250.5, 1, 250.5},
		{"ศูนย์", 0, 33.95959596, 0},
		{"ทศนิยม 4 ตำแหน่งของยอด", 0.1234, 33.62, 4.1487}, // 4.148708 → 4.1487
		{"ปัดครึ่งขึ้น (…5 ที่ตำแหน่งที่ 5)", 1, 0.00005, 0.0001},
		{"ปัดลงเมื่อไม่ถึงครึ่ง", 1, 0.000049, 0},
		{"ยอด 0.1 ไม่มีศูนย์ตกค้างจากไบนารี", 0.1, 0.5, 0.05},
		{"ยอดใหญ่", 1_000_000, 33.95959596, 33959595.96},
		{"เรทน้อย (IDR)", 1_000_000, 0.00193, 1930},
	}
	for _, tt := range tests {
		got, ok := ConvertTHB(tt.amount, tt.rate)
		if !ok || math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("%s: ConvertTHB(%v, %v) = %v, %v want %v", tt.name, tt.amount, tt.rate, got, ok, tt.want)
		}
	}
	// ปัดครั้งเดียวตอนท้าย: 3 ใบ ใบละ 0.00005 รวมกัน = ปัดครั้งเดียว (ไม่ปัดทีละใบแล้วบวก)
	if got, _ := ConvertTHB(3, 0.00005); got != 0.0002 { // 0.00015 → 0.0002 (ครึ่งขึ้น)
		t.Errorf("3 × 0.00005 = %v want 0.0002", got)
	}
	for _, bad := range [][2]float64{{math.NaN(), 1}, {math.Inf(1), 1}, {-1, 33}, {1, math.NaN()}, {1, math.Inf(-1)}, {1, 0}, {1, -2}} {
		if got, ok := ConvertTHB(bad[0], bad[1]); ok {
			t.Errorf("ConvertTHB(%v, %v) = %v ต้อง ok = false", bad[0], bad[1], got)
		}
	}
}
