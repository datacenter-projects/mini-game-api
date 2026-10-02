// Package redis คือที่เดียวที่ใช้ database.DBRedis — cache, session, distributed lock
//
// กฎ:
//   - ไฟล์: {topic}_repository.go, ฟังก์ชัน: XxxRepository(ctx context.Context, ...)
//   - key ทุกตัวประกาศเป็นฟังก์ชันใน keys.go (ห้ามต่อ string key กระจายทั่วโค้ด) และต้องมี TTL
//   - คำสั่งต่อเนื่องหลายคำสั่งใช้ Pipeline
//   - cache ห้ามเป็น source of truth ของยอดเงิน — ยอดเงินอ่านจาก Postgres เสมอ
package redis
