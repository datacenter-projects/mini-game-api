-- agent_management: ผู้สร้าง / ผู้แก้ล่าสุดของค่า PT (MGMT-16 — แก้ 2026-10-08)
-- created_by / updated_by = username ณ เวลานั้น (sub = owner@name) · ไม่มี FK เพราะ sub ลบจริงได้
-- แถวเดิม: ผู้สร้าง = ผู้สร้างบัญชี (Superadmin ไม่มีผู้สร้าง = ตัวเอง) · เวลา = เวลาสร้างบัญชี

-- +goose Up
ALTER TABLE agent_game_settings
    ADD COLUMN created_by VARCHAR(71) NOT NULL DEFAULT '',
    ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN updated_by VARCHAR(71) NOT NULL DEFAULT '';

ALTER TABLE member_game_settings
    ADD COLUMN created_by VARCHAR(71) NOT NULL DEFAULT '',
    ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN updated_by VARCHAR(71) NOT NULL DEFAULT '';

UPDATE agent_game_settings s
SET created_by = COALESCE(p.username, a.username), updated_by = COALESCE(p.username, a.username), created_at = a.created_at
FROM user_agents a
LEFT JOIN user_agents p ON p.id = a.parent_id
WHERE a.id = s.agent_id;

UPDATE member_game_settings s
SET created_by = a.username, updated_by = a.username, created_at = m.created_at
FROM members m
JOIN user_agents a ON a.id = m.agent_id
WHERE m.id = s.member_id;

-- +goose Down
ALTER TABLE member_game_settings DROP COLUMN updated_by, DROP COLUMN created_at, DROP COLUMN created_by;
ALTER TABLE agent_game_settings DROP COLUMN updated_by, DROP COLUMN created_at, DROP COLUMN created_by;
