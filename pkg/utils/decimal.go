package utils

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
)

// Decimal — ตัวเลขทศนิยมใน request (ค่า % และจำนวนเงิน — ACC-18) เก็บข้อความเดิมไว้
// แปลงด้วย Float4: ทศนิยมไม่เกิน 4 ตำแหน่ง (กฎข้อ 9 — แก้ 2026-10-09)
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

// maxIntDigits — กันเลขยาวเกินที่ float64 เก็บได้ตรงทุกหลักเมื่อมีทศนิยม 4 ตำแหน่ง
const maxIntDigits = 11

// Float4 — ค่าเป็น float64 ปัด 4 ตำแหน่ง (70 → 70 · 0.5 → 0.5 · 10000.1234 → 10000.1234)
// ok = false เมื่อไม่ใช่ตัวเลข, ทศนิยมเกิน 4 ตำแหน่ง, ใช้รูป exponent หรือส่วนจำนวนเต็มยาวเกิน 11 หลัก
func (d Decimal) Float4() (float64, bool) {
	if !d.IsNumber {
		return 0, false
	}
	intPart, frac, _ := strings.Cut(strings.TrimPrefix(d.Raw, "-"), ".")
	if len(frac) > 4 || len(intPart) > maxIntDigits {
		return 0, false
	}
	v, err := strconv.ParseFloat(d.Raw, 64)
	if err != nil {
		return 0, false
	}
	return Round4(v), true
}
