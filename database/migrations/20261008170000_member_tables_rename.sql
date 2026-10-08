-- member_management: ตารางของ Member ใช้ชื่อขึ้นต้น user_member ให้ตรงกับ models.UserMember* (2026-10-08)
-- members → user_members · member_game_settings → user_member_game_settings · member_balances → user_member_balances
-- เปลี่ยนชื่อเท่านั้น ไม่แตะข้อมูล · FK / sequence / index ตามตารางไปเอง (อ้างด้วย OID) — rename ชื่อให้ตรงกันด้วย
-- constraint ที่ไม่ได้ตั้งชื่อเองใช้ชื่อที่ Postgres ตั้งให้ ({table}_pkey · {table}_{column}_fkey · {table}_{column}_check · {table}_id_seq)

-- +goose Up
ALTER TABLE members RENAME TO user_members;
ALTER SEQUENCE members_id_seq RENAME TO user_members_id_seq;
ALTER TABLE user_members RENAME CONSTRAINT members_pkey TO user_members_pkey;
ALTER TABLE user_members RENAME CONSTRAINT members_agent_id_fkey TO user_members_agent_id_fkey;
ALTER TABLE user_members RENAME CONSTRAINT uq_members_username TO uq_user_members_username;
ALTER TABLE user_members RENAME CONSTRAINT ck_members_username TO ck_user_members_username;
ALTER TABLE user_members RENAME CONSTRAINT ck_members_phone TO ck_user_members_phone;
ALTER TABLE user_members RENAME CONSTRAINT ck_members_status TO ck_user_members_status;
ALTER INDEX idx_members_agent_username RENAME TO idx_user_members_agent_username;
ALTER INDEX uq_members_phone RENAME TO uq_user_members_phone;

ALTER TABLE member_game_settings RENAME TO user_member_game_settings;
ALTER TABLE user_member_game_settings RENAME COLUMN member_id TO user_member_id;
ALTER TABLE user_member_game_settings RENAME CONSTRAINT member_game_settings_pkey TO user_member_game_settings_pkey;
ALTER TABLE user_member_game_settings RENAME CONSTRAINT member_game_settings_member_id_fkey TO user_member_game_settings_user_member_id_fkey;
ALTER TABLE user_member_game_settings RENAME CONSTRAINT member_game_settings_commission_bp_check TO user_member_game_settings_commission_bp_check;

ALTER TABLE member_balances RENAME TO user_member_balances;
ALTER TABLE user_member_balances RENAME COLUMN member_id TO user_member_id;
ALTER TABLE user_member_balances RENAME CONSTRAINT member_balances_pkey TO user_member_balances_pkey;
ALTER TABLE user_member_balances RENAME CONSTRAINT member_balances_member_id_fkey TO user_member_balances_user_member_id_fkey;
ALTER TABLE user_member_balances RENAME CONSTRAINT member_balances_amount_check TO user_member_balances_amount_check;

-- +goose Down
ALTER TABLE user_member_balances RENAME CONSTRAINT user_member_balances_amount_check TO member_balances_amount_check;
ALTER TABLE user_member_balances RENAME CONSTRAINT user_member_balances_user_member_id_fkey TO member_balances_member_id_fkey;
ALTER TABLE user_member_balances RENAME CONSTRAINT user_member_balances_pkey TO member_balances_pkey;
ALTER TABLE user_member_balances RENAME COLUMN user_member_id TO member_id;
ALTER TABLE user_member_balances RENAME TO member_balances;

ALTER TABLE user_member_game_settings RENAME CONSTRAINT user_member_game_settings_commission_bp_check TO member_game_settings_commission_bp_check;
ALTER TABLE user_member_game_settings RENAME CONSTRAINT user_member_game_settings_user_member_id_fkey TO member_game_settings_member_id_fkey;
ALTER TABLE user_member_game_settings RENAME CONSTRAINT user_member_game_settings_pkey TO member_game_settings_pkey;
ALTER TABLE user_member_game_settings RENAME COLUMN user_member_id TO member_id;
ALTER TABLE user_member_game_settings RENAME TO member_game_settings;

ALTER INDEX uq_user_members_phone RENAME TO uq_members_phone;
ALTER INDEX idx_user_members_agent_username RENAME TO idx_members_agent_username;
ALTER TABLE user_members RENAME CONSTRAINT ck_user_members_status TO ck_members_status;
ALTER TABLE user_members RENAME CONSTRAINT ck_user_members_phone TO ck_members_phone;
ALTER TABLE user_members RENAME CONSTRAINT ck_user_members_username TO ck_members_username;
ALTER TABLE user_members RENAME CONSTRAINT uq_user_members_username TO uq_members_username;
ALTER TABLE user_members RENAME CONSTRAINT user_members_agent_id_fkey TO members_agent_id_fkey;
ALTER TABLE user_members RENAME CONSTRAINT user_members_pkey TO members_pkey;
ALTER SEQUENCE user_members_id_seq RENAME TO members_id_seq;
ALTER TABLE user_members RENAME TO members;
