-- agent_management MGMT-52 (lead review 2026-10-09 P4): ไม่มีเมนูสิทธิ์ pt แล้ว — ทุกเส้นของ module ใช้ member
-- ล้าง key "pt" ออกจาก subaccounts.permissions ของ sub เดิม (ค่าเดิมกู้ไม่ได้ · ไม่มีผลกับสิทธิ์อื่น)

-- +goose Up
UPDATE subaccounts SET permissions = permissions - 'pt' WHERE permissions ? 'pt';

-- +goose Down
-- ไม่คืน key pt (ค่าเดิมไม่ได้เก็บไว้ · โค้ดที่ใช้เมนู pt ถูกลบแล้ว)
SELECT 1;
