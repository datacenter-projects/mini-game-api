-- member_management (2026-10-09 · maofoy · รอ lead อนุมัติ)
-- user_members.cnf = สายชั้นบนของ Member {"parent":[{"id","position"}]} = สายของผู้สร้าง + ผู้สร้าง (รูปแบบเดียวกับ user_agents.cnf — MGMT-61)
-- user_members.credit = ยอดเงินของ Member (float ตามที่ทีมตกลง — CLAUDE.md กฎข้อ 9 ยังต้องแก้) · แทนตาราง user_member_balances (Member มี 1 สกุลเสมอ — MGMT-13)
-- version ตั้งหลัง 20261009120000_agent_management_chain เพราะ backfill อ่าน user_agents.cnf

-- +goose Up
ALTER TABLE user_members
    ADD COLUMN cnf    JSONB            NOT NULL DEFAULT '{"parent": []}',
    ADD COLUMN credit DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (credit >= 0);
CREATE INDEX idx_user_members_cnf ON user_members USING GIN (cnf jsonb_path_ops);

UPDATE user_members m
SET cnf = jsonb_build_object('parent', (a.cnf -> 'parent') || jsonb_build_array(jsonb_build_object('id', a.id, 'position', lower(a.role))))
FROM user_agents a
WHERE a.id = m.agent_id;

UPDATE user_members m
SET credit = b.amount / 100.0
FROM user_member_balances b
WHERE b.user_member_id = m.id AND b.currency = m.currency;

DROP TABLE user_member_balances;

-- +goose Down
CREATE TABLE user_member_balances (
    user_member_id BIGINT      NOT NULL REFERENCES user_members(id),
    currency       VARCHAR(4)  NOT NULL,
    amount         BIGINT      NOT NULL DEFAULT 0 CHECK (amount >= 0),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_member_id, currency)
);
INSERT INTO user_member_balances (user_member_id, currency, amount)
SELECT id, currency, round(credit * 100)::BIGINT FROM user_members WHERE credit > 0;

DROP INDEX idx_user_members_cnf;
ALTER TABLE user_members DROP COLUMN credit, DROP COLUMN cnf;
