// Package currencyrate คือ Redis ของ currency_rate — key ทั้งหมดประกาศที่นี่ที่เดียว · prefix: currency_rate:
// spec: docs/modules/currency_rate.md CR-03, CR-10
package currencyrate

// keyLock — lock ของ worker (SET NX PX) กันหลาย instance ดึงเรทพร้อมกัน (CR-03)
func keyLock() string { return "currency_rate:sync:lock" }

// keyRate — hash: สกุล → เรท ×10^8 (CR-10)
func keyRate() string { return "currency_rate:usdt" }

// keyMeta — hash: date, synced_at (CR-10)
func keyMeta() string { return "currency_rate:usdt:meta" }
