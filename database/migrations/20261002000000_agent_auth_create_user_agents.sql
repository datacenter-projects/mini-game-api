-- agent_auth: ตาราง user_agents เฉพาะคอลัมน์ที่ auth ใช้ (docs/modules/agent_auth.md หัวข้อ 6)
-- คอลัมน์อื่นของ agent จะเพิ่มด้วย migration ใหม่ตอนทำ module สายงาน

-- +goose Up
CREATE TABLE user_agents (
    id            BIGSERIAL    PRIMARY KEY,
    parent_id     BIGINT       REFERENCES user_agents(id),
    username      VARCHAR(50)  NOT NULL,
    password_hash VARCHAR(100) NOT NULL,
    role          VARCHAR(20)  NOT NULL,
    status        VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE',
    last_login_at TIMESTAMPTZ,
    last_login_ip VARCHAR(45),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_user_agents_username UNIQUE (username),
    CONSTRAINT ck_user_agents_username CHECK (username = lower(username) AND username ~ '^[a-z0-9]+$'),
    CONSTRAINT ck_user_agents_status CHECK (status IN ('ACTIVE', 'SUSPENDED', 'LOCKED')),
    CONSTRAINT ck_user_agents_role CHECK (role IN ('SUPERADMIN', 'COMPANY', 'SHAREHOLDER', 'AGENT')),
    CONSTRAINT ck_user_agents_not_self_parent CHECK (parent_id IS NULL OR parent_id <> id)
);

CREATE INDEX idx_user_agents_parent_id ON user_agents(parent_id);

-- +goose Down
DROP TABLE user_agents;
