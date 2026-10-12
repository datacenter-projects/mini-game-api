-- เงินและ % เป็นทศนิยม DOUBLE PRECISION ปัด 4 ตำแหน่ง (CLAUDE.md กฎข้อ 9 — ทีมตกลง 2026-10-09 · agent_management หัวข้อ 6)
-- แปลงค่าเดิม ÷100 (bp → % · หน่วยย่อย 1/100 → หน่วยสกุล) · column ของ % ตัด _bp
-- CHECK ช่วงค่าเดิมเปลี่ยนเป็นหน่วย % (0–100 · commission 0–1)

-- +goose Up
ALTER TABLE agent_game_settings
    DROP CONSTRAINT agent_game_settings_pt_from_parent_bp_check,
    DROP CONSTRAINT agent_game_settings_pt_bp_check,
    DROP CONSTRAINT agent_game_settings_force_bp_check,
    DROP CONSTRAINT agent_game_settings_remain_bp_check,
    DROP CONSTRAINT agent_game_settings_commission_bp_check,
    DROP CONSTRAINT ck_agent_game_settings_pt;
ALTER TABLE agent_game_settings RENAME COLUMN pt_from_parent_bp TO pt_from_parent;
ALTER TABLE agent_game_settings RENAME COLUMN pt_bp TO pt;
ALTER TABLE agent_game_settings RENAME COLUMN force_bp TO force;
ALTER TABLE agent_game_settings RENAME COLUMN remain_bp TO remain;
ALTER TABLE agent_game_settings RENAME COLUMN commission_bp TO commission;
ALTER TABLE agent_game_settings
    ALTER COLUMN pt_from_parent TYPE DOUBLE PRECISION USING pt_from_parent / 100.0,
    ALTER COLUMN pt             TYPE DOUBLE PRECISION USING pt / 100.0,
    ALTER COLUMN force          TYPE DOUBLE PRECISION USING force / 100.0,
    ALTER COLUMN remain         TYPE DOUBLE PRECISION USING remain / 100.0,
    ALTER COLUMN commission     TYPE DOUBLE PRECISION USING commission / 100.0,
    ADD CONSTRAINT ck_agent_game_settings_pt_from_parent CHECK (pt_from_parent BETWEEN 0 AND 100),
    ADD CONSTRAINT ck_agent_game_settings_pt_range       CHECK (pt BETWEEN 0 AND 100),
    ADD CONSTRAINT ck_agent_game_settings_force          CHECK (force BETWEEN 0 AND 100),
    ADD CONSTRAINT ck_agent_game_settings_remain         CHECK (remain BETWEEN 0 AND 100),
    ADD CONSTRAINT ck_agent_game_settings_commission     CHECK (commission BETWEEN 0 AND 1),
    ADD CONSTRAINT ck_agent_game_settings_pt             CHECK (pt <= pt_from_parent);

ALTER TABLE user_member_game_settings
    DROP CONSTRAINT user_member_game_settings_pt_bp_check,
    DROP CONSTRAINT user_member_game_settings_remain_bp_check,
    DROP CONSTRAINT user_member_game_settings_commission_bp_check;
ALTER TABLE user_member_game_settings RENAME COLUMN pt_bp TO pt;
ALTER TABLE user_member_game_settings RENAME COLUMN remain_bp TO remain;
ALTER TABLE user_member_game_settings RENAME COLUMN commission_bp TO commission;
ALTER TABLE user_member_game_settings
    ALTER COLUMN pt         TYPE DOUBLE PRECISION USING pt / 100.0,
    ALTER COLUMN remain     TYPE DOUBLE PRECISION USING remain / 100.0,
    ALTER COLUMN commission TYPE DOUBLE PRECISION USING commission / 100.0,
    ADD CONSTRAINT ck_user_member_game_settings_pt         CHECK (pt BETWEEN 0 AND 100),
    ADD CONSTRAINT ck_user_member_game_settings_remain     CHECK (remain BETWEEN 0 AND 100),
    ADD CONSTRAINT ck_user_member_game_settings_commission CHECK (commission BETWEEN 0 AND 1);

ALTER TABLE agent_balances ALTER COLUMN amount TYPE DOUBLE PRECISION USING amount / 100.0;
ALTER TABLE balance_ledger
    ALTER COLUMN amount        TYPE DOUBLE PRECISION USING amount / 100.0,
    ALTER COLUMN balance_after TYPE DOUBLE PRECISION USING balance_after / 100.0;

-- +goose Down
ALTER TABLE balance_ledger
    ALTER COLUMN amount        TYPE BIGINT USING round(amount * 100)::BIGINT,
    ALTER COLUMN balance_after TYPE BIGINT USING round(balance_after * 100)::BIGINT;
ALTER TABLE agent_balances ALTER COLUMN amount TYPE BIGINT USING round(amount * 100)::BIGINT;

ALTER TABLE user_member_game_settings
    DROP CONSTRAINT ck_user_member_game_settings_pt,
    DROP CONSTRAINT ck_user_member_game_settings_remain,
    DROP CONSTRAINT ck_user_member_game_settings_commission;
ALTER TABLE user_member_game_settings
    ALTER COLUMN pt         TYPE INT USING round(pt * 100)::INT,
    ALTER COLUMN remain     TYPE INT USING round(remain * 100)::INT,
    ALTER COLUMN commission TYPE INT USING round(commission * 100)::INT;
ALTER TABLE user_member_game_settings RENAME COLUMN pt TO pt_bp;
ALTER TABLE user_member_game_settings RENAME COLUMN remain TO remain_bp;
ALTER TABLE user_member_game_settings RENAME COLUMN commission TO commission_bp;
ALTER TABLE user_member_game_settings
    ADD CONSTRAINT user_member_game_settings_pt_bp_check         CHECK (pt_bp BETWEEN 0 AND 10000),
    ADD CONSTRAINT user_member_game_settings_remain_bp_check     CHECK (remain_bp BETWEEN 0 AND 10000),
    ADD CONSTRAINT user_member_game_settings_commission_bp_check CHECK (commission_bp BETWEEN 0 AND 100);

ALTER TABLE agent_game_settings
    DROP CONSTRAINT ck_agent_game_settings_pt_from_parent,
    DROP CONSTRAINT ck_agent_game_settings_pt_range,
    DROP CONSTRAINT ck_agent_game_settings_force,
    DROP CONSTRAINT ck_agent_game_settings_remain,
    DROP CONSTRAINT ck_agent_game_settings_commission,
    DROP CONSTRAINT ck_agent_game_settings_pt;
ALTER TABLE agent_game_settings
    ALTER COLUMN pt_from_parent TYPE INT USING round(pt_from_parent * 100)::INT,
    ALTER COLUMN pt             TYPE INT USING round(pt * 100)::INT,
    ALTER COLUMN force          TYPE INT USING round(force * 100)::INT,
    ALTER COLUMN remain         TYPE INT USING round(remain * 100)::INT,
    ALTER COLUMN commission     TYPE INT USING round(commission * 100)::INT;
ALTER TABLE agent_game_settings RENAME COLUMN pt_from_parent TO pt_from_parent_bp;
ALTER TABLE agent_game_settings RENAME COLUMN pt TO pt_bp;
ALTER TABLE agent_game_settings RENAME COLUMN force TO force_bp;
ALTER TABLE agent_game_settings RENAME COLUMN remain TO remain_bp;
ALTER TABLE agent_game_settings RENAME COLUMN commission TO commission_bp;
ALTER TABLE agent_game_settings
    ADD CONSTRAINT agent_game_settings_pt_from_parent_bp_check CHECK (pt_from_parent_bp BETWEEN 0 AND 10000),
    ADD CONSTRAINT agent_game_settings_pt_bp_check             CHECK (pt_bp BETWEEN 0 AND 10000),
    ADD CONSTRAINT agent_game_settings_force_bp_check          CHECK (force_bp BETWEEN 0 AND 10000),
    ADD CONSTRAINT agent_game_settings_remain_bp_check         CHECK (remain_bp BETWEEN 0 AND 10000),
    ADD CONSTRAINT agent_game_settings_commission_bp_check     CHECK (commission_bp BETWEEN 0 AND 100),
    ADD CONSTRAINT ck_agent_game_settings_pt                   CHECK (pt_bp <= pt_from_parent_bp);
