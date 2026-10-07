// Package currencyrate คือกฎของอัตราแลกเปลี่ยนแบบ pure function — spec: docs/modules/currency_rate.md
package currencyrate

import (
	"math/big"
	"strings"
	"time"
)

// RateScale — เรทเก็บเป็นจำนวนเต็มคูณ 10^8 (CR-06) · RateDecimals = ทศนิยมสูงสุดที่รับ
const (
	RateScale    int64 = 100_000_000
	RateDecimals       = 8
)

// Currencies — 27 สกุลของระบบ (CR-06 รับเฉพาะสกุลเหล่านี้)
var Currencies = []string{
	"ARS", "AUD", "BDT", "BOB", "BRL", "CLP", "CNY", "EUR", "GBP", "HKD", "IDR", "INR", "JPY", "KHR",
	"KRW", "LAK", "MMK", "MXN", "MYR", "NGN", "PHP", "PKR", "THB", "TWD", "USD", "USDT", "VND",
}

var bangkok = time.FixedZone("Asia/Bangkok", 7*3600) // ไทยไม่มี daylight saving

// RequestDate — วันที่ที่ส่งให้ askme = วันปัจจุบันตามเวลาไทย รูปแบบ YYYY-MM-DD (CR-04)
func RequestDate(now time.Time) string {
	return now.In(bangkok).Format("2006-01-02")
}

// ParseRate แปลง JSON number (ข้อความ) เป็นจำนวนเต็ม ×10^8 โดยไม่ผ่าน float (CR-06)
// ok = false เมื่อไม่ใช่ตัวเลขทศนิยมบวก, เป็น 0, ทศนิยมเกิน 8 ตำแหน่ง หรือใหญ่เกิน int64
func ParseRate(s string) (int64, bool) {
	intPart, frac, hasDot := strings.Cut(s, ".")
	if intPart == "" || (hasDot && frac == "") || len(frac) > RateDecimals || !allDigits(intPart) || !allDigits(frac) {
		return 0, false
	}
	n, ok := new(big.Int).SetString(intPart+frac+strings.Repeat("0", RateDecimals-len(frac)), 10)
	if !ok || n.Sign() <= 0 || !n.IsInt64() {
		return 0, false
	}
	return n.Int64(), true
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// IsSupported — สกุลนี้อยู่ใน 27 สกุลของระบบไหม
func IsSupported(currency string) bool {
	for _, c := range Currencies {
		if c == currency {
			return true
		}
	}
	return false
}

// ConvertMinor แปลงยอด (หน่วยย่อย 1/100) จากสกุลที่มีเรท rateFrom ไปสกุลที่มีเรท rateTo (CR-13)
// ยอดใหม่ = ยอด × rateTo ÷ rateFrom · คำนวณด้วย big.Int · ปัดครึ่งขึ้น (ค่าลบปัดออกจากศูนย์)
// ใช้แสดงผลเท่านั้น ห้ามใช้ย้ายเงิน
func ConvertMinor(amount, rateFrom, rateTo int64) int64 {
	num := new(big.Int).Mul(big.NewInt(amount), big.NewInt(rateTo))
	den := big.NewInt(rateFrom)
	neg := num.Sign() < 0
	num.Abs(num)
	q, r := new(big.Int).QuoRem(num, den, new(big.Int))
	if new(big.Int).Mul(r, big.NewInt(2)).Cmp(den) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	if neg {
		q.Neg(q)
	}
	return q.Int64()
}
