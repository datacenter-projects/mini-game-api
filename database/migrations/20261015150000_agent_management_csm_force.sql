-- +goose Up
-- MGMT-18A (lead pt_calculation Q3 · 2026-10-15): ห้ามตั้ง force ให้ Company Seamless Master — CSM ถือสู้ไม่ได้ (MGMT-19)
-- ปรับ force ของแถว CSM เป็น 0 · ค่าอื่นไม่แตะ · Company ประเภทอื่นไม่แตะ
-- ข้อมูลน้อย (เฉพาะ CSM) · prod เริ่มจาก DB ว่าง — ไม่ต้องแยก 3 ขั้น (กฎข้อ 20)
UPDATE agent_game_settings s
SET force = 0
FROM user_agents a
WHERE a.id = s.agent_id AND a.role = 'COMPANY' AND a.agent_type = 'SEAMLESS_MASTER' AND s.force <> 0;

-- +goose Down
-- ย้อนค่าเดิมไม่ได้ (ไม่ได้เก็บไว้) และค่า 0 ถูกต้องตามกฎ — ไม่ทำอะไร
SELECT 1;
