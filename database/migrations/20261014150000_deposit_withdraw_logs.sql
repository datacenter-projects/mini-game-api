-- deposit_withdraw (lead zerph 2026-10-13 · docs/reviews/deposit-withdraw.md DQ1–DQ13 · ยืนยันเพิ่ม 2026-10-14)
-- 1 แถวต่อการกดฝาก / ถอน 1 ครั้ง · ยอดต่อบัญชีอยู่ใน balance_ledger (ref_type = 'DW' · ref_id = id ของตารางนี้)
-- request_id ลงทะเบียนใน create_requests (target_type = 'DW') ด้วย — กันใช้ซ้ำข้ามเส้น (DW-26)
-- parent_unlimited = แม่เป็น Superadmin (วงเงินไม่จำกัด) · parent_balance_* = 0 (ACC-32 ไม่มี null)

-- +goose Up
CREATE TABLE deposit_withdraw_logs (
    id                    BIGSERIAL        PRIMARY KEY,
    request_id            VARCHAR(64)      NOT NULL,
    transaction_type      VARCHAR(10)      NOT NULL,
    actor_type            VARCHAR(10)      NOT NULL,
    actor_id              BIGINT           NOT NULL,
    actor_username        VARCHAR(71)      NOT NULL,
    parent_id             BIGINT           NOT NULL REFERENCES user_agents(id),
    parent_username       VARCHAR(71)      NOT NULL,
    target_type           VARCHAR(10)      NOT NULL,
    target_id             BIGINT           NOT NULL,
    target_role           VARCHAR(20)      NOT NULL,
    target_username       VARCHAR(71)      NOT NULL,
    currency              VARCHAR(4)       NOT NULL,
    amount                DOUBLE PRECISION NOT NULL,
    parent_unlimited      BOOLEAN          NOT NULL DEFAULT false,
    parent_balance_before DOUBLE PRECISION NOT NULL DEFAULT 0,
    parent_balance_after  DOUBLE PRECISION NOT NULL DEFAULT 0,
    target_balance_before DOUBLE PRECISION NOT NULL,
    target_balance_after  DOUBLE PRECISION NOT NULL,
    note                  VARCHAR(200)     NOT NULL DEFAULT '',
    ip                    VARCHAR(45)      NOT NULL DEFAULT '',
    created_at            TIMESTAMPTZ      NOT NULL DEFAULT now(),
    CONSTRAINT uq_deposit_withdraw_logs_request UNIQUE (request_id),
    CONSTRAINT ck_deposit_withdraw_logs_type   CHECK (transaction_type IN ('DEPOSIT', 'WITHDRAW')),
    CONSTRAINT ck_deposit_withdraw_logs_actor  CHECK (actor_type IN ('AGENT', 'SUB')),
    CONSTRAINT ck_deposit_withdraw_logs_target CHECK (target_type IN ('AGENT', 'MEMBER')),
    CONSTRAINT ck_deposit_withdraw_logs_amount CHECK (amount > 0 AND amount <= 1000000000)
);
CREATE INDEX idx_deposit_withdraw_logs_parent ON deposit_withdraw_logs(parent_id, created_at DESC);
CREATE INDEX idx_deposit_withdraw_logs_target ON deposit_withdraw_logs(target_type, target_id, created_at DESC);

-- +goose Down
DROP TABLE deposit_withdraw_logs;
