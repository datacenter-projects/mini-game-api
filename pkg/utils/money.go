package utils

import (
	"strconv"
)

// Money — จำนวนเงินหน่วยย่อยที่สุด (1/100 ของสกุล) เก็บเป็น int64 ตามกฎข้อ 9
// ตอนส่งออก JSON เป็น number ทศนิยม 2 ตำแหน่ง เช่น 96205600 → 962056.00 (docs/modules/account.md ACC-18)
// แปลงด้วยการต่อข้อความ ไม่ผ่าน float
type Money int64

func (m Money) MarshalJSON() ([]byte, error) {
	v := int64(m)
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	frac := v % 100
	s := sign + strconv.FormatInt(v/100, 10) + "."
	if frac < 10 {
		s += "0"
	}
	return []byte(s + strconv.FormatInt(frac, 10)), nil
}
