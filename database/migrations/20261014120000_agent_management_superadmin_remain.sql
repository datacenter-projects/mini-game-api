-- +goose Up
-- MGMT-18A (2026-10-14): Superadmin ไม่ตั้ง Remain ให้ลูก — ส่วนที่เหลือคืน Superadmin อยู่แล้ว (MGMT-22) · การคิดตอน settle เหมือนเดิม
-- ปรับ remain ของแถวที่ผู้สร้างเป็น Superadmin (Company ทุกประเภท รวม CSM) เป็น 0 · ค่าอื่นไม่แตะ
-- ข้อมูลน้อย (เฉพาะ Company) · prod เริ่มจาก DB ว่าง — ไม่ต้องแยก 3 ขั้น (กฎข้อ 20)
UPDATE agent_game_settings s
SET remain = 0
FROM user_agents p
WHERE p.id = s.parent_id AND p.role = 'SUPERADMIN' AND s.remain <> 0;

-- +goose Down
-- ย้อนค่าเดิมไม่ได้ (ไม่ได้เก็บไว้) และค่า 0 ถูกต้องตามกฎ — ไม่ทำอะไร
SELECT 1;
