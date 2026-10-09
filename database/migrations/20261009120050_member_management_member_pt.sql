-- member_management: agent ถือ PT สู้กับ Member แต่ละคนแยกกัน (แก้ 2026-10-09 · maofoy · รอ lead อนุมัติ)
-- pt_bp = ผู้สร้างถือสู้กับ Member คนนี้ · remain_bp = ค่าที่ผู้สร้างได้รับ − pt_bp (ระบบคิด · sync เมื่อค่าที่ผู้สร้างได้รับเปลี่ยน)
-- แถวเดิม: pt_bp = pt_bp ของผู้สร้าง (ค่าที่ใช้กับ Member ทุกคนก่อน migration นี้) · ผู้สร้างไม่มีแถวของเกมนั้น = 0

-- +goose Up
ALTER TABLE user_member_game_settings
    ADD COLUMN pt_bp     INT NOT NULL DEFAULT 0 CHECK (pt_bp BETWEEN 0 AND 10000),
    ADD COLUMN remain_bp INT NOT NULL DEFAULT 0 CHECK (remain_bp BETWEEN 0 AND 10000);

UPDATE user_member_game_settings s
SET pt_bp = a.pt_bp, remain_bp = a.pt_from_parent_bp - a.pt_bp
FROM user_members m
JOIN agent_game_settings a ON a.agent_id = m.agent_id
WHERE m.id = s.user_member_id AND a.game_code = s.game_code;

-- +goose Down
ALTER TABLE user_member_game_settings DROP COLUMN remain_bp, DROP COLUMN pt_bp;
