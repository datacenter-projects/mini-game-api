package utils

import (
	"strconv"
	"strings"
)

// Percent — ค่า % เก็บเป็น bp (1% = 100) ตามกฎข้อ 9 · ส่งออก JSON เป็น number ทศนิยมไม่เกิน 2 ตำแหน่ง
// เช่น 9550 → 95.5 · 50 → 0.5 · 7000 → 70 (account ACC-18 · agent_management MGMT-17)
// แปลงด้วยการต่อข้อความ ไม่ผ่าน float
type Percent int

func (p Percent) MarshalJSON() ([]byte, error) {
	v := int64(p)
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	s := sign + strconv.FormatInt(v/100, 10)
	if frac := v % 100; frac != 0 {
		f := strconv.FormatInt(frac, 10)
		if frac < 10 {
			f = "0" + f
		}
		s += "." + strings.TrimRight(f, "0")
	}
	return []byte(s), nil
}
