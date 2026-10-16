// Package currencyrate คือกฎของอัตราแลกเปลี่ยนแบบ pure function — spec: docs/modules/currency_rate.md
// ไม่แตะ DB / Redis / config / logger · คำนวณเงินด้วย big.Rat ปัดครั้งเดียวตอนท้ายแบบครึ่งขึ้น (กฎข้อ 9)
package currencyrate

import (
	"encoding/json"
	"math/big"
	"strconv"
	"strings"
	"time"

	agentManagementCore "app/app/core/agent_management"
)

// ทศนิยมของเรท (CR-03 / CR-15) และของยอดที่แปลงแล้ว (CR-22)
const (
	RateDecimals   = 8
	AmountDecimals = 4
)

// สกุลที่ใช้เป็นฐานของการแปลง
const (
	CurrencyTHB  = "THB"
	CurrencyUSDT = "USDT"
)

// Currencies — 27 สกุลของระบบ: ใช้รายการเดียวกับ agent_management (MGMT-10 · spec หัวข้อ 10 ข้อ 1) ไม่ประกาศซ้ำ
func Currencies() []string { return agentManagementCore.Currencies }

// IsSupported — สกุลนี้อยู่ใน 27 สกุลของระบบไหม (ตัวพิมพ์ใหญ่เท่านั้น)
func IsSupported(currency string) bool { return agentManagementCore.IsSupportedCurrency(currency) }

var bangkok = time.FixedZone("Asia/Bangkok", 7*3600) // ไทยไม่มี daylight saving

// Bangkok คือ time zone ที่ใช้ตัดเดือนของเรท (CR-01)
func Bangkok() *time.Location { return bangkok }

// MonthDate — วันที่ 1 ของเดือนที่ t อยู่ (เวลาไทย) เก็บเป็นเที่ยงคืน UTC ของวันนั้น — ใช้เป็น key / ค่าของคอลัมน์ DATE
// (ไม่ใช้เที่ยงคืนเวลาไทย เพราะ driver แปลงเป็น UTC แล้ววันอาจเลื่อนถอยหลัง 1 วัน)
func MonthDate(t time.Time) time.Time {
	b := t.In(bangkok)
	return time.Date(b.Year(), b.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// ParseMonth แปลง "YYYY-MM" (ของคำสั่งมือ -month) เป็น MonthDate · ผิดรูปแบบ ok = false
func ParseMonth(s string) (time.Time, bool) {
	t, err := time.Parse("2006-01", s)
	if err != nil {
		return time.Time{}, false
	}
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), true
}

// RequestDate — วันที่ที่ส่งให้ askmelotto = วันที่ 1 ของเดือนเสมอ รูปแบบ YYYY-MM-01 (CR-02)
func RequestDate(month time.Time) string { return month.Format("2006-01-02") }

// ก่อนเวลานี้ (ชั่วโมง เวลาไทย) ของวันที่ 1 askmelotto อาจยังสร้างเรทใหม่ไม่เสร็จ (cronjob ของระบบแม่ตี 1) — CR-16
const earlyWindowHours = 3

// IsEarlyMonth — now ยังอยู่ก่อน 03:00 ของวันที่ 1 ของเดือน (เวลาไทย) → ยังไม่ครบเป็นเรื่องปกติ (log info แทน warn — CR-16)
func IsEarlyMonth(now time.Time) bool {
	b := now.In(bangkok)
	return b.Day() == 1 && b.Hour() < earlyWindowHours
}

// ---- ตัวเลขเรท ----

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// ParseRate แปลง JSON number (ข้อความ) เป็น big.Rat โดยไม่ผ่าน float (CR-03 / CR-15)
// ok = false เมื่อไม่ใช่ตัวเลขทศนิยมบวก, เป็น 0, ติดลบ, มีเลขยกกำลัง, ทศนิยมเกิน 8 ตำแหน่ง
func ParseRate(s string) (*big.Rat, bool) {
	intPart, frac, hasDot := strings.Cut(s, ".")
	if intPart == "" || (hasDot && frac == "") || len(frac) > RateDecimals || !allDigits(intPart) || !allDigits(frac) {
		return nil, false
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok || r.Sign() <= 0 {
		return nil, false
	}
	return r, true
}

// roundHalfUp ปัด r เป็น decimals ตำแหน่งแบบครึ่งขึ้น (r ≥ 0) แล้วคืนเป็นข้อความทศนิยมเต็มตำแหน่ง
func roundHalfUp(r *big.Rat, decimals int) string {
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	num := new(big.Int).Mul(r.Num(), scale)
	q, rem := new(big.Int).QuoRem(num, r.Denom(), new(big.Int))
	if new(big.Int).Mul(rem, big.NewInt(2)).Cmp(r.Denom()) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	s := q.String()
	if len(s) <= decimals {
		s = strings.Repeat("0", decimals-len(s)+1) + s
	}
	return s[:len(s)-decimals] + "." + s[len(s)-decimals:]
}

func toFloat(decimal string) float64 {
	f, _ := strconv.ParseFloat(decimal, 64) // ข้อความทศนิยมสั้น (≤ 8 ตำแหน่ง) → double ที่ใกล้ที่สุดเสมอ
	return f
}

// RateToFloat — เรทดิบ (≤ 8 ตำแหน่ง) เป็น float64 สำหรับเก็บ usdt_rate
func RateToFloat(r *big.Rat) float64 { return toFloat(roundHalfUp(r, RateDecimals)) }

// THBRate — thb_rate = usdt_rate(THB) ÷ usdt_rate(สกุลนั้น) ปัด 8 ตำแหน่งครึ่งขึ้น (CR-03)
// เช่น THB 33.62 · USD 0.99 → 33.95959596
func THBRate(usdtTHB, usdtCurrency *big.Rat) float64 {
	return toFloat(roundHalfUp(new(big.Rat).Quo(usdtTHB, usdtCurrency), RateDecimals))
}

// ---- ตรวจ response ของ askmelotto (CR-15) ----

// Response คือส่วนของ response ที่ต้องใช้ — ดึงมาจาก JSON แล้วส่งเข้า core ให้ตรวจแบบ pure
type Response struct {
	Code         int
	BaseCurrency string
	Date         string
	JSONRate     map[string]json.RawMessage
}

// ResponseProblem — ผลตรวจ response ทั้งก้อน (ผิด = ไม่บันทึกอะไรเลย)
type ResponseProblem int

const (
	ResponseOK           ResponseProblem = iota
	ResponseBadCode                      // code ≠ 200 (เช่น 430503 ยังไม่มีเรทของวันนั้น)
	ResponseBadBase                      // base_currency ≠ USDT
	ResponseDateMismatch                 // data.date ≠ วันที่ที่ขอ
	ResponseNoRates                      // json_rate ไม่มี / ไม่ใช่ object
)

// CheckResponse ตรวจระดับ response (CR-15): code = 200 · base = USDT · date ตรงกับที่ขอ · มี json_rate
func CheckResponse(r Response, requestedDate string) ResponseProblem {
	switch {
	case r.Code != 200:
		return ResponseBadCode
	case r.BaseCurrency != CurrencyUSDT:
		return ResponseBadBase
	case r.Date != requestedDate:
		return ResponseDateMismatch
	case r.JSONRate == nil:
		return ResponseNoRates
	}
	return ResponseOK
}

// ParseRates แยกสกุลที่ผ่านตรวจ (JSON number > 0 · ทศนิยม ≤ 8 · อยู่ใน 27 สกุล) ออกจากสกุลที่ไม่ผ่าน (CR-15)
// สกุลนอกรายการไม่สนใจ · rejected = สกุลของระบบที่มีใน response แต่ค่าผิด · สกุลที่ไม่มีใน response เลยไม่อยู่ใน rejected (ดู MissingOf)
func ParseRates(raw map[string]json.RawMessage) (accepted map[string]*big.Rat, rejected []string) {
	accepted = map[string]*big.Rat{}
	for _, c := range Currencies() {
		v, present := raw[c]
		if !present {
			continue
		}
		// ต้องเป็น JSON number (ไม่รับ string / null / object) แล้วค่อยแปลงตามกฎทศนิยม
		if len(v) == 0 || v[0] < '0' || v[0] > '9' {
			rejected = append(rejected, c)
			continue
		}
		r, ok := ParseRate(string(v))
		if !ok {
			rejected = append(rejected, c)
			continue
		}
		accepted[c] = r
	}
	return accepted, rejected
}

// Rate คือแถวที่พร้อมบันทึก (ยังไม่มี rate_month / synced_at)
type Rate struct {
	Currency string
	USDTRate float64
	THBRate  float64
}

// BuildRates คิด thb_rate ของทุกสกุลที่ผ่านตรวจ (CR-03 · CR-04) เรียงตามรายการ 27 สกุล
// ไม่มี THB ใน accepted → ok = false (ไม่มีฐานคิด — ไม่บันทึกทั้งเดือน)
func BuildRates(accepted map[string]*big.Rat) (rates []Rate, ok bool) {
	thb, has := accepted[CurrencyTHB]
	if !has {
		return nil, false
	}
	for _, c := range Currencies() {
		r, has := accepted[c]
		if !has {
			continue
		}
		rate := Rate{Currency: c, USDTRate: RateToFloat(r), THBRate: THBRate(thb, r)}
		if c == CurrencyTHB {
			rate.THBRate = 1
		}
		rates = append(rates, rate)
	}
	return rates, true
}

// MissingOf — สกุลของระบบที่ยังไม่มีในรายการที่มีแล้ว (have) · เรียงตามรายการ 27 สกุล
func MissingOf(have []string) []string {
	set := make(map[string]bool, len(have))
	for _, c := range have {
		set[c] = true
	}
	var out []string
	for _, c := range Currencies() {
		if !set[c] {
			out = append(out, c)
		}
	}
	return out
}

// ---- แปลงเป็นบาท (CR-20 – CR-22) ----

// decimalRat แปลง float64 เป็น big.Rat ผ่านข้อความทศนิยมสั้นสุดของมัน (ไม่ใช้ค่าไบนารีตรงๆ — กันศูนย์ตกค้างของ 0.1 เป็นต้น)
func decimalRat(f float64) (*big.Rat, bool) {
	return new(big.Rat).SetString(strconv.FormatFloat(f, 'f', -1, 64))
}

// ConvertTHB — ยอดเป็นบาท = Round4(amount × thbRate) · คำนวณด้วย big.Rat ปัดครั้งเดียวตอนท้ายครึ่งขึ้น (CR-22)
// ok = false เมื่อ amount / thbRate ไม่ใช่เลขจำกัด (NaN / Inf) · amount ติดลบ · thbRate ไม่เป็นบวก
func ConvertTHB(amount, thbRate float64) (float64, bool) {
	if !isFinite(amount) || !isFinite(thbRate) || amount < 0 || thbRate <= 0 {
		return 0, false
	}
	a, ok1 := decimalRat(amount)
	r, ok2 := decimalRat(thbRate)
	if !ok1 || !ok2 {
		return 0, false
	}
	return toFloat(roundHalfUp(new(big.Rat).Mul(a, r), AmountDecimals)), true
}

func isFinite(f float64) bool { return f-f == 0 } // NaN และ ±Inf ให้ผลเป็น NaN
