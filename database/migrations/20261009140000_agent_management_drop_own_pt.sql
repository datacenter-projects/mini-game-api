-- agent_management (2026-10-09 · boiledegg ตัดสิน): ลบค่าถือ pt ค่าเดียวของบัญชีฝั่ง agent และเส้น update-hold
-- ถือสู้กับ Member ตั้งต่อ Member ที่ user_member_game_settings.pt แล้ว (member_management MGMT-21)

-- +goose Up
ALTER TABLE agent_game_settings
    DROP CONSTRAINT ck_agent_game_settings_pt,
    DROP CONSTRAINT ck_agent_game_settings_pt_range,
    DROP COLUMN pt;

-- +goose Down
-- ค่าเดิมกู้ไม่ได้ · ตั้งเป็นค่าเริ่มต้นแบบตอนสร้าง (ถือทั้งหมดที่ได้รับ · Company Seamless Master = 0)
ALTER TABLE agent_game_settings ADD COLUMN pt DOUBLE PRECISION NOT NULL DEFAULT 0;
UPDATE agent_game_settings s SET pt = s.pt_from_parent
FROM user_agents a
WHERE a.id = s.agent_id AND NOT (a.role = 'COMPANY' AND a.agent_type = 'SEAMLESS_MASTER');
ALTER TABLE agent_game_settings
    ADD CONSTRAINT ck_agent_game_settings_pt_range CHECK (pt BETWEEN 0 AND 100),
    ADD CONSTRAINT ck_agent_game_settings_pt       CHECK (pt <= pt_from_parent);
