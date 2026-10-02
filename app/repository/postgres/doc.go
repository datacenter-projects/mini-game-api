// Package postgres คือที่เดียวที่ query PostgreSQL
//
// กฎ:
//   - ไฟล์: {table หรือ aggregate}_repository.go, ฟังก์ชัน: XxxRepository(db *gorm.DB, ...)
//   - param แรกเป็น db *gorm.DB เสมอ — ห้ามอ้าง database.DBConn ในนี้ (service ส่ง DBConn หรือ tx มาให้)
//   - ห้ามเปิด Transaction และห้ามมี business rule (เช็คสิทธิ์/เช็คยอด ทำใน service)
//   - อ่านเพื่อเขียนเงิน ต้อง lock: db.Clauses(clause.Locking{Strength: "UPDATE"})
//   - Select เฉพาะคอลัมน์ที่ใช้, ห้าม N+1 (ใช้ IN / JOIN / Preload), เขียนหลายแถวใช้ CreateInBatches
//   - ไม่เจอข้อมูล: คืน apperr.ErrNotFound (หรือ error เฉพาะของ module) ไม่คืน struct ว่าง
package postgres
