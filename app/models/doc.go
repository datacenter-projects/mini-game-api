// Package models คือ GORM struct ที่ map กับตาราง — ไฟล์: {table}_models.go
//
// กฎ:
//   - มีได้แค่ field, TableName() และ constant ของค่า enum ในตารางนั้น — ห้ามมี business logic
//     (logic ไปอยู่ app/core หรือ app/service)
//   - schema จริงมาจาก database/migrations (goose) เท่านั้น — ห้ามใช้ AutoMigrate
//   - เงินเป็น int64 หน่วยย่อยที่สุด
//   - ห้ามมี global state
package models
