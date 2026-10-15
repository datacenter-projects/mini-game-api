// Package currencyrate คือ Redis ของ currency_rate — key ทั้งหมดประกาศที่นี่ที่เดียว · prefix: currency_rate:
// spec: docs/modules/currency_rate.md CR-13 · เหลือ key เดียว (lock) — key เรทเดิม `currency_rate:usdt` / `:meta` เลิกใช้แล้ว ปล่อยไว้ไม่ลบ ไม่อ่าน
package currencyrate

// keyLock — lock ของ job (SET NX PX) กันหลาย pod ดึงเรทพร้อมกัน (CR-13) · มี TTL เสมอ (ผู้เรียกส่ง ttl)
func keyLock() string { return "currency_rate:sync:lock" }
