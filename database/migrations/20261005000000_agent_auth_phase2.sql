-- agent_auth phase 2: subaccount, passcode, เปลี่ยน/รีเซ็ตรหัสผ่าน, ADMIN (docs/modules/agent_auth_phase2.md หัวข้อ 6)

-- +goose Up
ALTER TABLE user_agents
    ADD COLUMN passcode_hash            VARCHAR(100),
    ADD COLUMN previous_password_hash   VARCHAR(100),
    ADD COLUMN must_change_password     BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN must_change_passcode     BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN temp_password_expires_at TIMESTAMPTZ,
    ADD COLUMN temp_passcode_expires_at TIMESTAMPTZ,
    DROP CONSTRAINT ck_user_agents_role,
    ADD CONSTRAINT ck_user_agents_role
        CHECK (role IN ('SUPERADMIN', 'ADMIN', 'COMPANY', 'SHAREHOLDER', 'AGENT')),
    ADD CONSTRAINT ck_user_agents_admin_no_parent CHECK (role <> 'ADMIN' OR parent_id IS NULL); -- AUTH-43

CREATE TABLE subaccounts (
    id                       BIGSERIAL    PRIMARY KEY,
    agent_id                 BIGINT       NOT NULL REFERENCES user_agents(id), -- ผู้สร้าง
    username                 VARCHAR(71)  NOT NULL,                            -- {owner 50}@{name 20}
    password_hash            VARCHAR(100) NOT NULL,
    previous_password_hash   VARCHAR(100),
    passcode_hash            VARCHAR(100),
    must_change_password     BOOLEAN      NOT NULL DEFAULT false,
    must_change_passcode     BOOLEAN      NOT NULL DEFAULT false,
    temp_password_expires_at TIMESTAMPTZ,
    temp_passcode_expires_at TIMESTAMPTZ,
    status                   VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE',
    last_login_at            TIMESTAMPTZ,
    last_login_ip            VARCHAR(45),
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_subaccounts_username UNIQUE (username),
    CONSTRAINT ck_subaccounts_username CHECK (username = lower(username) AND username ~ '^[^@]+@[a-z0-9]{3,20}$'),
    CONSTRAINT ck_subaccounts_status CHECK (status IN ('ACTIVE', 'SUSPENDED', 'LOCKED'))
);

CREATE INDEX idx_subaccounts_agent_id ON subaccounts(agent_id);

-- เหตุการณ์ที่เปลี่ยนข้อมูลบัญชี (AUTH-49) — ห้ามเก็บรหัสผ่าน, passcode, token หรือค่าชั่วคราว
CREATE TABLE auth_audit_logs (
    id              BIGSERIAL    PRIMARY KEY,
    action          VARCHAR(30)  NOT NULL,
    actor_type      VARCHAR(10)  NOT NULL, -- AGENT / SUB = คนทำ · SCRIPT = script บนเซิร์ฟเวอร์ · SYSTEM = ระบบบังคับ (บล็อก)
    actor_id        BIGINT,
    actor_username  VARCHAR(71),           -- ชื่อ ณ เวลานั้น (SCRIPT = user ของเครื่องที่รัน)
    target_type     VARCHAR(10),           -- NULL = ไม่มีบัญชีนี้ในระบบ (เช่น LOGIN_BLOCKED ของ username ที่ไม่มีจริง)
    target_id       BIGINT,
    target_username VARCHAR(71)  NOT NULL,
    ip              VARCHAR(45),
    user_agent      VARCHAR(255),
    request_id      VARCHAR(64),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT ck_auth_audit_logs_action CHECK (action IN (
        'PASSCODE_SETUP', 'PASSCODE_CHANGE', 'PASSWORD_CHANGE',
        'RESET_PASSCODE', 'RESET_PASSWORD', 'PASSCODE_BLOCKED', 'LOGIN_BLOCKED')),
    CONSTRAINT ck_auth_audit_logs_actor_type CHECK (actor_type IN ('AGENT', 'SUB', 'SCRIPT', 'SYSTEM')),
    CONSTRAINT ck_auth_audit_logs_actor_id CHECK ((actor_type IN ('AGENT', 'SUB')) = (actor_id IS NOT NULL)),
    CONSTRAINT ck_auth_audit_logs_target_type CHECK (target_type IS NULL OR target_type IN ('AGENT', 'SUB')),
    CONSTRAINT ck_auth_audit_logs_target_id CHECK ((target_type IS NULL) = (target_id IS NULL))
);

CREATE INDEX idx_auth_audit_logs_target ON auth_audit_logs(target_type, target_id, created_at);
CREATE INDEX idx_auth_audit_logs_actor ON auth_audit_logs(actor_type, actor_id, created_at);
CREATE INDEX idx_auth_audit_logs_action ON auth_audit_logs(action, created_at);
CREATE INDEX idx_auth_audit_logs_created_at ON auth_audit_logs(created_at);

-- +goose Down
DROP TABLE auth_audit_logs;
DROP TABLE subaccounts;

ALTER TABLE user_agents
    DROP CONSTRAINT ck_user_agents_admin_no_parent,
    DROP CONSTRAINT ck_user_agents_role,
    ADD CONSTRAINT ck_user_agents_role CHECK (role IN ('SUPERADMIN', 'COMPANY', 'SHAREHOLDER', 'AGENT')),
    DROP COLUMN temp_passcode_expires_at,
    DROP COLUMN temp_password_expires_at,
    DROP COLUMN must_change_passcode,
    DROP COLUMN must_change_password,
    DROP COLUMN previous_password_hash,
    DROP COLUMN passcode_hash;
