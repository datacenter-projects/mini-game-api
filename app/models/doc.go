// Package models คือ GORM struct ที่ map กับตาราง — ไฟล์: {table}_models.go
//
// กฎ:
//   - มีได้แค่ field, TableName() และ constant ของค่า enum ในตารางนั้น — ห้ามมี business logic
//     (logic ไปอยู่ app/core หรือ app/service)
//   - schema จริงมาจาก database/migrations (goose) เท่านั้น — ห้ามใช้ AutoMigrate
//   - เงินและ % เป็น float64 ปัด 4 ตำแหน่ง (CLAUDE.md กฎข้อ 9 — แก้ 2026-10-09)
//   - ห้ามมี global state
package models
