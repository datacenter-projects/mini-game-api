package utils

import (
	"math"
	"strconv"
)

// เงินและ % เป็น float64 ทศนิยม ปัด 4 ตำแหน่ง (CLAUDE.md กฎข้อ 9 — แก้ 2026-10-09 · docs/modules/account.md ACC-18)
// JSON ส่งค่าตามที่เก็บ (encoding/json ตัด 0 ท้ายให้เอง เช่น 10000.5) · หน้าบ้านปัด 3 ตำแหน่งตอนแสดง

// Round4 — ปัดเป็น 4 ตำแหน่ง (ครึ่งขึ้น) · ใช้ก่อนบันทึกทุกครั้ง และหลังบวก / ลบ ก่อนเอาไปเทียบ
func Round4(v float64) float64 { return math.Round(v*10000) / 10000 }

// maxDecimal4Abs — ส่วนจำนวนเต็มไม่เกิน 11 หลัก: float64 เก็บได้ตรงทุกหลักเมื่อมีทศนิยม 4 ตำแหน่ง
const maxDecimal4Abs = 1e11

// IsDecimal4 — ตัวเลขจาก request (เงิน / % รับเป็น float64 — ทีมตกลง 2026-10-12) ทศนิยมไม่เกิน 4 ตำแหน่ง (กฎข้อ 9)
// และส่วนจำนวนเต็มไม่เกิน 11 หลัก · ส่งเป็นชนิดอื่น (เช่น "70") body parse ไม่ผ่านตั้งแต่ต้น
func IsDecimal4(v float64) bool { return math.Abs(v) < maxDecimal4Abs && Round4(v) == v }

// FormatNum — ตัวเลขสำหรับข้อความ error เช่น 90 → "90" · 0.5 → "0.5" · 10000.1234 → "10000.1234"
func FormatNum(v float64) string { return strconv.FormatFloat(Round4(v), 'f', -1, 64) }
