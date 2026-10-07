-- agent_management (module ②) — docs/modules/agent_management.md หัวข้อ 6

-- +goose Up
ALTER TABLE user_agents
    ADD COLUMN name       VARCHAR(32),
    ADD COLUMN phone      VARCHAR(15),
    ADD COLUMN agent_type VARCHAR(20),
    ADD CONSTRAINT ck_user_agents_phone CHECK (phone IS NULL OR phone ~ '^[0-9]{8,15}$');
-- NOT VALID: บัญชี Company / Share ที่มีอยู่ก่อน migration นี้ยังไม่มี agent_type · แถวใหม่ทุกแถวถูกตรวจ
ALTER TABLE user_agents ADD CONSTRAINT ck_user_agents_agent_type CHECK (
    (role = 'COMPANY' AND agent_type IN ('TRANSFER', 'SEAMLESS_RESELLER', 'SEAMLESS_MASTER', 'SEAMLESS_1TO1')) OR
    (role = 'SHAREHOLDER' AND agent_type IN ('B2B', 'B2C', 'RESELLER', 'MASTER')) OR
    (role NOT IN ('COMPANY', 'SHAREHOLDER') AND agent_type IS NULL)) NOT VALID;
CREATE UNIQUE INDEX uq_user_agents_phone ON user_agents(phone) WHERE phone IS NOT NULL;
CREATE INDEX idx_user_agents_parent_username ON user_agents(parent_id, username);  -- รายชื่อลูกตรง เรียง A→Z

CREATE TABLE agent_currencies (
    agent_id BIGINT     NOT NULL REFERENCES user_agents(id),
    currency VARCHAR(4) NOT NULL,
    PRIMARY KEY (agent_id, currency)
);

-- ค่าหุ้นส่วน + เปิด / ปิด ต่อเกม (MGMT-16, MGMT-20) · ค่าในกลุ่มเดียวกันเท่ากันทุกเกม
CREATE TABLE agent_game_settings (
    agent_id          BIGINT      NOT NULL REFERENCES user_agents(id),
    category          VARCHAR(30) NOT NULL,
    game_code         VARCHAR(50) NOT NULL,
    pt_from_parent_bp INT         NOT NULL CHECK (pt_from_parent_bp BETWEEN 0 AND 10000),
    pt_bp             INT         NOT NULL CHECK (pt_bp BETWEEN 0 AND 10000),
    force_bp          INT         NOT NULL DEFAULT 0 CHECK (force_bp BETWEEN 0 AND 10000),
    remain_bp         INT         NOT NULL DEFAULT 0 CHECK (remain_bp BETWEEN 0 AND 10000),
    commission_bp     INT         NOT NULL DEFAULT 0 CHECK (commission_bp BETWEEN 0 AND 100),
    status            BOOLEAN     NOT NULL DEFAULT true,  -- เปิด / ปิดทั้งกลุ่ม
    status_game       BOOLEAN     NOT NULL DEFAULT true,  -- เปิด / ปิดทีละเกม
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (agent_id, game_code),
    CONSTRAINT ck_agent_game_settings_pt CHECK (pt_bp <= pt_from_parent_bp)
);

CREATE TABLE members (
    id            BIGSERIAL    PRIMARY KEY,
    agent_id      BIGINT       NOT NULL REFERENCES user_agents(id),  -- ผู้สร้าง
    username      VARCHAR(32)  NOT NULL,
    password_hash VARCHAR(100) NOT NULL,
    name          VARCHAR(32)  NOT NULL,
    phone         VARCHAR(15),
    currency      VARCHAR(4)   NOT NULL,
    status        VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE',
    last_login_at TIMESTAMPTZ,
    last_login_ip VARCHAR(45),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_members_username UNIQUE (username),
    CONSTRAINT ck_members_username CHECK (username = lower(username) AND username ~ '^[a-z0-9]{3,32}$'),
    CONSTRAINT ck_members_phone CHECK (phone IS NULL OR phone ~ '^[0-9]{8,15}$'),
    CONSTRAINT ck_members_status CHECK (status IN ('ACTIVE', 'SUSPENDED', 'LOCKED'))
);
CREATE INDEX idx_members_agent_username ON members(agent_id, username);
CREATE UNIQUE INDEX uq_members_phone ON members(phone) WHERE phone IS NOT NULL;

-- Member มีแค่ Commission ต่อเกม (MGMT-21)
CREATE TABLE member_game_settings (
    member_id     BIGINT      NOT NULL REFERENCES members(id),
    category      VARCHAR(30) NOT NULL,
    game_code     VARCHAR(50) NOT NULL,
    commission_bp INT         NOT NULL DEFAULT 0 CHECK (commission_bp BETWEEN 0 AND 100),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (member_id, game_code)
);

-- สถานะ INACTIVE ของ sub เก็บเป็น 'SUSPENDED' (MGMT-43) · permissions ไม่มี key = off
ALTER TABLE subaccounts
    ADD COLUMN name        VARCHAR(32),
    ADD COLUMN phone       VARCHAR(15),
    ADD COLUMN permissions JSONB NOT NULL DEFAULT '{}',
    ADD CONSTRAINT ck_subaccounts_phone CHECK (phone IS NULL OR phone ~ '^[0-9]{8,15}$');
CREATE INDEX idx_subaccounts_agent_username ON subaccounts(agent_id, username);
CREATE UNIQUE INDEX uq_subaccounts_phone ON subaccounts(phone) WHERE phone IS NOT NULL;

-- ยอดเงินและ ledger (MGMT-15A) — module การชำระเงินใช้ต่อ · amount = หน่วยย่อย 1/100
CREATE TABLE agent_balances (
    agent_id   BIGINT      NOT NULL REFERENCES user_agents(id),
    currency   VARCHAR(4)  NOT NULL,
    amount     BIGINT      NOT NULL DEFAULT 0 CHECK (amount >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (agent_id, currency)
);

CREATE TABLE member_balances (
    member_id  BIGINT      NOT NULL REFERENCES members(id),
    currency   VARCHAR(4)  NOT NULL,
    amount     BIGINT      NOT NULL DEFAULT 0 CHECK (amount >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (member_id, currency)
);

-- ทุกการเปลี่ยนยอดต้องมีแถวที่นี่ใน tx เดียวกัน (กฎข้อ 11) · unique กันโอนซ้ำ (กฎข้อ 12)
CREATE TABLE balance_ledger (
    id            BIGSERIAL   PRIMARY KEY,
    owner_type    VARCHAR(10) NOT NULL,
    owner_id      BIGINT      NOT NULL,
    currency      VARCHAR(4)  NOT NULL,
    amount        BIGINT      NOT NULL,
    balance_after BIGINT      NOT NULL,
    reason        VARCHAR(30) NOT NULL,
    ref_type      VARCHAR(10),
    ref_id        BIGINT,
    request_id    VARCHAR(64) NOT NULL,
    actor_type    VARCHAR(10) NOT NULL,
    actor_id      BIGINT      NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_balance_ledger_request UNIQUE (request_id, owner_type, owner_id, currency),
    CONSTRAINT ck_balance_ledger_owner_type CHECK (owner_type IN ('AGENT', 'MEMBER'))
);
CREATE INDEX idx_balance_ledger_owner ON balance_ledger(owner_type, owner_id, created_at);

-- กันสร้างซ้ำด้วย request_id (MGMT-15A)
CREATE TABLE create_requests (
    request_id   VARCHAR(64) PRIMARY KEY,
    creator_type VARCHAR(10) NOT NULL,
    creator_id   BIGINT      NOT NULL,
    target_type  VARCHAR(10) NOT NULL,
    target_id    BIGINT      NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ประวัติการเปลี่ยนแปลงบัญชี (MGMT-60) — ห้ามเก็บรหัสผ่าน
CREATE TABLE account_change_logs (
    id              BIGSERIAL   PRIMARY KEY,
    actor_type      VARCHAR(10) NOT NULL,
    actor_id        BIGINT      NOT NULL,
    actor_username  VARCHAR(71) NOT NULL,
    target_type     VARCHAR(10) NOT NULL,
    target_id       BIGINT      NOT NULL,
    target_username VARCHAR(71) NOT NULL,
    action          VARCHAR(30) NOT NULL,
    old_value       JSONB,
    new_value       JSONB,
    ip              VARCHAR(45),
    request_id      VARCHAR(64),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_account_change_logs_target ON account_change_logs(target_type, target_id, created_at);

-- Superadmin ที่มีอยู่: ได้รับ 100% ทุกเกม (MGMT-22) · ครบ 27 สกุล · ไม่มีแถวยอดเงิน (วงเงินไม่จำกัด — MGMT-15A)
INSERT INTO agent_game_settings (agent_id, category, game_code, pt_from_parent_bp, pt_bp)
SELECT a.id, 'minigame', g.code, 10000, 0
FROM user_agents a CROSS JOIN (VALUES ('coin_toss'), ('rock_paper_scissors'), ('scratch_card')) AS g(code)
WHERE a.role = 'SUPERADMIN'
ON CONFLICT DO NOTHING;
INSERT INTO agent_currencies (agent_id, currency)
SELECT a.id, c.code
FROM user_agents a CROSS JOIN (VALUES ('ARS'), ('AUD'), ('BDT'), ('BOB'), ('BRL'), ('CLP'), ('CNY'), ('EUR'), ('GBP'),
    ('HKD'), ('IDR'), ('INR'), ('JPY'), ('KHR'), ('KRW'), ('LAK'), ('MMK'), ('MXN'), ('MYR'), ('NGN'), ('PHP'),
    ('PKR'), ('THB'), ('TWD'), ('USD'), ('USDT'), ('VND')) AS c(code)
WHERE a.role = 'SUPERADMIN'
ON CONFLICT DO NOTHING;

-- +goose Down
DROP TABLE account_change_logs;
DROP TABLE create_requests;
DROP TABLE balance_ledger;
DROP TABLE member_balances;
DROP TABLE agent_balances;
DROP INDEX uq_subaccounts_phone;
DROP INDEX idx_subaccounts_agent_username;
ALTER TABLE subaccounts
    DROP CONSTRAINT ck_subaccounts_phone,
    DROP COLUMN permissions,
    DROP COLUMN phone,
    DROP COLUMN name;
DROP TABLE member_game_settings;
DROP TABLE members;
DROP TABLE agent_game_settings;
DROP TABLE agent_currencies;
DROP INDEX idx_user_agents_parent_username;
DROP INDEX uq_user_agents_phone;
ALTER TABLE user_agents
    DROP CONSTRAINT ck_user_agents_agent_type,
    DROP CONSTRAINT ck_user_agents_phone,
    DROP COLUMN agent_type,
    DROP COLUMN phone,
    DROP COLUMN name;
