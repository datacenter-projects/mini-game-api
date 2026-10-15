-- account ACC-01 / ACC-05 แก้ 2026-10-08: ทุกบัญชี Company / Share / Agent มี Key ของ 1.3
-- สร้าง Key ให้บัญชีที่มีอยู่แล้วแต่ยังไม่มี (บัญชีใหม่ได้ Key ตอนสร้าง — MGMT-04)
-- Key = gen_random_uuid() 2 ค่าต่อกันแบบไม่มีขีด = hex ตัวเล็ก 64 ตัว (ACC-03) · สุ่มจาก pg_strong_random

-- +goose Up
INSERT INTO api_credentials (agent_id, api_key)
SELECT a.id, replace(gen_random_uuid()::text, '-', '') || replace(gen_random_uuid()::text, '-', '')
FROM user_agents a
WHERE a.role IN ('COMPANY', 'SHAREHOLDER', 'AGENT')
ON CONFLICT (agent_id) DO NOTHING;

-- +goose Down
-- ไม่ลบ Key ที่สร้างไว้ (ลูกค้าอาจนำไปใช้แล้ว)
SELECT 1;
