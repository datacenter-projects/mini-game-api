-- agent_management: เบอร์โทรของ sub ซ้ำได้ (MGMT-41 — แก้ 2026-10-08) · ฝั่ง agent และ Member ยังห้ามซ้ำ (MGMT-08)
-- ไม่สร้าง index แทน: ไม่มี query หลักที่ค้นตามเบอร์ของ sub

-- +goose Up
DROP INDEX uq_subaccounts_phone;

-- +goose Down
CREATE UNIQUE INDEX uq_subaccounts_phone ON subaccounts(phone) WHERE phone IS NOT NULL;
