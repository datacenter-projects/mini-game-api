-- agent_management MGMT-61, MGMT-62 (2026-10-09 · lead อนุมัติ)
-- user_agents.cnf = สายชั้นบน {"parent":[{"id":..,"position":"superadmin|company|shareholder|agent"}, ...]} เรียงบนลงล่าง ไม่รวมตัวเอง
-- agent_game_settings.parent_id = ผู้สร้างของบัญชีเจ้าของแถว (= user_agents.parent_id) · Superadmin = NULL

-- +goose Up
ALTER TABLE user_agents ADD COLUMN cnf JSONB NOT NULL DEFAULT '{"parent": []}';
CREATE INDEX idx_user_agents_cnf ON user_agents USING GIN (cnf jsonb_path_ops);

ALTER TABLE agent_game_settings ADD COLUMN parent_id BIGINT REFERENCES user_agents(id);
CREATE INDEX idx_agent_game_settings_parent ON agent_game_settings(parent_id, game_code);

-- backfill บัญชีเดิม
WITH RECURSIVE chain AS (
    SELECT id, '[]'::jsonb AS parent
    FROM user_agents
    WHERE parent_id IS NULL
    UNION ALL
    SELECT c.id, ch.parent || jsonb_build_array(jsonb_build_object('id', p.id, 'position', lower(p.role)))
    FROM user_agents c
    JOIN chain ch ON ch.id = c.parent_id
    JOIN user_agents p ON p.id = c.parent_id
)
UPDATE user_agents u
SET cnf = jsonb_build_object('parent', chain.parent)
FROM chain
WHERE chain.id = u.id AND u.parent_id IS NOT NULL;

UPDATE agent_game_settings s
SET parent_id = u.parent_id
FROM user_agents u
WHERE u.id = s.agent_id AND u.parent_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_agent_game_settings_parent;
ALTER TABLE agent_game_settings DROP COLUMN parent_id;
DROP INDEX IF EXISTS idx_user_agents_cnf;
ALTER TABLE user_agents DROP COLUMN cnf;
