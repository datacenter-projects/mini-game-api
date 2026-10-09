package utils

import (
	"math"
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

// MinorToCredit / CreditToMinor — แปลงหน่วยย่อย 1/100 (int64 · ledger / ยอดฝั่ง agent) ↔ user_members.credit (float64 หน่วยสกุล)
// credit เป็น float ตามที่ทีมตกลง 2026-10-09 · แปลงกลับปัดเป็นหน่วยย่อยที่ใกล้ที่สุด
func MinorToCredit(minor int64) float64 { return float64(minor) / 100 }

func CreditToMinor(credit float64) int64 { return int64(math.Round(credit * 100)) }
