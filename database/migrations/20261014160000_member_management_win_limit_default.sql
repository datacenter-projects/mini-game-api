-- member_management MGMT-65 (maofoy 2026-10-14 · แทน lead P9 — รอ lead รับทราบ)
-- ความหมายใหม่ของ win_limit_per_day: 0 = เล่นไม่ได้ (เดิม 0 = ไม่จำกัด) · ตั้งได้ 0 – 10,000,000 · ไม่ส่งตอนสร้าง = 10,000,000
-- Member เดิมที่เป็น 0 (ไม่ได้ตั้ง = ไม่จำกัด ตามความหมายเดิม) ปรับเป็น 10,000,000 เพื่อให้ยังเล่นได้เหมือนเดิม
-- เพดาน 10,000,000 ตรวจในแอป (core/member_management) · DB คง CHECK >= 0

-- +goose Up
UPDATE user_members SET win_limit_per_day = 10000000 WHERE win_limit_per_day = 0;
ALTER TABLE user_members ALTER COLUMN win_limit_per_day SET DEFAULT 10000000;

-- +goose Down
-- คืน default เดิมเท่านั้น · แถวที่ถูกปรับเป็น 10,000,000 ตอน Up ไม่คืนเป็น 0 (แยกไม่ออกจากแถวที่ตั้ง 10,000,000 เอง)
ALTER TABLE user_members ALTER COLUMN win_limit_per_day SET DEFAULT 0;
