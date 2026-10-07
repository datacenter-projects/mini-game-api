package utils

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
)

// Decimal — ตัวเลขทศนิยมใน request (ค่า % และจำนวนเงิน — ACC-18) เก็บข้อความเดิมไว้
// แปลงเป็นจำนวนเต็มด้วย Fixed2 โดยไม่ผ่าน float (กฎข้อ 9)
// ส่งเป็นชนิดอื่น parse ผ่านแต่ IsNumber = false ให้ Validate() ตอบ 422 พร้อมบอก field ได้
type Decimal struct {
	Raw      string
	Present  bool // มี key นี้ใน body และไม่ใช่ null
	IsNumber bool
}

var jsonNumberRe = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`)

func (d *Decimal) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if string(b) == "null" {
		*d = Decimal{}
		return nil
	}
	*d = Decimal{Raw: string(b), Present: true, IsNumber: jsonNumberRe.Match(b)}
	return nil
}

// maxFixed2IntDigits — กันเลขยาวเกิน int64 (10^15 × 100 ยังอยู่ในช่วง)
const maxFixed2IntDigits = 15

// Fixed2 — ค่า × 100 เป็นจำนวนเต็ม (70 → 7000 · 0.5 → 50 · 1234.56 → 123456)
// ok = false เมื่อไม่ใช่ตัวเลข, ทศนิยมเกิน 2 ตำแหน่ง, ใช้รูป exponent หรือยาวเกิน
func (d Decimal) Fixed2() (int64, bool) {
	if !d.IsNumber {
		return 0, false
	}
	s := d.Raw
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	intPart, frac, _ := strings.Cut(s, ".")
	if len(frac) > 2 || len(intPart) > maxFixed2IntDigits {
		return 0, false
	}
	for len(frac) < 2 {
		frac += "0"
	}
	v, err := strconv.ParseInt(intPart+frac, 10, 64)
	if err != nil {
		return 0, false
	}
	if neg {
		v = -v
	}
	return v, true
}
