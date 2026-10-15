// Package core เป็นที่รวม business rule แบบ pure function — แยก sub-package ตามหัวข้อ: app/core/{topic}/
// เช่น commission, rng, ladder
//
// กฎ:
//   - ห้าม import database, redis, fiber, logger, configs หรือ package ใน app/service / app/repository
//     (รับ input เป็นค่า คืน output เป็นค่า — ไม่มี side effect)
//   - ทุกฟังก์ชันต้องมี table test ที่ใส่ตัวเลข input/output ชัดเจน ให้คนอ่าน spec เช็คเองได้
//   - rule ทุกข้อต้องอ้างถึง spec ที่อนุมัติแล้วใน docs/modules/{module}.md
//   - เงินและ % เป็น float64 ปัด 4 ตำแหน่ง (CLAUDE.md กฎข้อ 9 — แก้ 2026-10-09)
package core
