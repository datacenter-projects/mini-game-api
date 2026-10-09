-- member_management (2026-10-09 · maofoy · lead อนุมัติ 2026-10-09): user_member_game_settings.parent_id = agent ผู้สร้าง Member เจ้าของแถว
-- ค่าเดียวกับ user_members.agent_id (แบบเดียวกับ agent_game_settings.parent_id — agent_management MGMT-62) · Member มีผู้สร้างเสมอ = NOT NULL
-- version ตั้งเองหลัง 20261009140000 ที่รันบน dev แล้ว (make migration ได้เลขที่น้อยกว่า goose จะปฏิเสธ)

-- +goose Up
ALTER TABLE user_member_game_settings ADD COLUMN parent_id BIGINT REFERENCES user_agents(id);

UPDATE user_member_game_settings s
SET parent_id = m.agent_id
FROM user_members m
WHERE m.id = s.user_member_id;

ALTER TABLE user_member_game_settings ALTER COLUMN parent_id SET NOT NULL;
CREATE INDEX idx_user_member_game_settings_parent ON user_member_game_settings(parent_id, game_code);

-- +goose Down
DROP INDEX IF EXISTS idx_user_member_game_settings_parent;
ALTER TABLE user_member_game_settings DROP COLUMN parent_id;
