// Package redis คือที่เดียวที่ใช้ database.DBRedis — cache, session, distributed lock
//
// แยก folder ตาม module: redis/{module}/ (package = ชื่อ folder ตัด _ · import alias {module}Redis) · root นี้มีแค่ doc.go
//
// กฎ:
//   - ไฟล์: {topic}_repository.go, ฟังก์ชัน: XxxRepository(ctx context.Context, ...)
//   - key ของ module ประกาศเป็นฟังก์ชันใน keys.go ของ folder นั้น (ห้ามต่อ string key กระจายทั่วโค้ด) และต้องมี TTL
//   - prefix ของ key ห้ามซ้ำกับ module อื่น
//   - คำสั่งต่อเนื่องหลายคำสั่งใช้ Pipeline
//   - cache ห้ามเป็น source of truth ของยอดเงิน — ยอดเงินอ่านจาก Postgres เสมอ
package redis
