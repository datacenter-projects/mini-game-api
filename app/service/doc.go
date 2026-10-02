// Package service เป็นที่รวม business logic — แยก sub-package ตาม module: app/service/{module}/
//
// กฎ:
//   - ชื่อ module ต้องตรงกับ app/internals/{context}/controllers/{module} และ dto/{module} ทุกตัวอักษร
//   - ไฟล์: {operation}_service.go, ฟังก์ชัน: XxxService(ctx context.Context, ...) (result, error)
//   - service เป็นเจ้าของ transaction: database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {...})
//     แล้วส่ง tx ลง repository — repository ห้ามเปิด transaction เอง
//   - error ที่ client ต้องเห็นให้คืน apperr.ErrXxx — ห้ามคืน code แยกเป็นค่าที่สอง
//   - คำนวณล้วนๆ (ไม่แตะ DB) ให้แยกไปไว้ที่ app/core/{topic} แล้วเขียน table test
//   - service เรียก service ของ module อื่นได้ แต่ห้าม import package ใน app/internals หรือ app/externals
package service
