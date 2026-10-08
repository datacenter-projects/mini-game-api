-- account 1.3 ข้อมูลรับรอง API (docs/modules/account.md หัวข้อ 6) · เลขเวอร์ชันเปลี่ยนจาก 20261007142913 (2026-10-08) ให้ลงหลัง migration ของ agent_management ที่เข้า dev ก่อน

-- +goose Up
CREATE TABLE api_credentials (
    agent_id       BIGINT       PRIMARY KEY REFERENCES user_agents(id),  -- เจ้าของ Key (ACC-01)
    key_ciphertext BYTEA        NOT NULL,                               -- AES-256-GCM (nonce + ciphertext) — ACC-04
    key_hash       CHAR(64)     NOT NULL,                               -- sha256(Key) hex — ค้นตอนลูกค้าเรียก API
    callback_url   VARCHAR(500),                                        -- NULL = ยังไม่ตั้ง (ACC-06)
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_api_credentials_key_hash UNIQUE (key_hash)
);

-- uq_api_allowed_ips ขึ้นต้นด้วย agent_id จึงใช้เป็น index ของการค้นตามเจ้าของได้
CREATE TABLE api_allowed_ips (
    id         BIGSERIAL   PRIMARY KEY,
    agent_id   BIGINT      NOT NULL REFERENCES api_credentials(agent_id) ON DELETE CASCADE,
    cidr       CIDR        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_api_allowed_ips UNIQUE (agent_id, cidr),
    CONSTRAINT ck_api_allowed_ips_v4 CHECK (family(cidr) = 4)
);

-- ประวัติการบันทึก (ACC-09) — ห้ามเก็บ Key
CREATE TABLE api_credential_logs (
    id               BIGSERIAL    PRIMARY KEY,
    agent_id         BIGINT       NOT NULL REFERENCES user_agents(id),
    actor_type       VARCHAR(10)  NOT NULL,  -- AGENT / SUB
    actor_id         BIGINT       NOT NULL,
    actor_username   VARCHAR(71)  NOT NULL,
    old_callback_url VARCHAR(500),
    new_callback_url VARCHAR(500),
    old_ips          TEXT[]       NOT NULL,
    new_ips          TEXT[]       NOT NULL,
    ip               VARCHAR(45),
    request_id       VARCHAR(64),
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT ck_api_credential_logs_actor_type CHECK (actor_type IN ('AGENT', 'SUB'))
);
CREATE INDEX idx_api_credential_logs_agent ON api_credential_logs(agent_id, created_at);

-- +goose Down
DROP TABLE api_credential_logs;
DROP TABLE api_allowed_ips;
DROP TABLE api_credentials;
