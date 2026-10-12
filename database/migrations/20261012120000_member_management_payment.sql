-- member_management MGMT-63 – MGMT-65 (lead review 2026-10-12 · docs/reviews/member-payment.md P1–P15)
-- payment_type = วิธีจ่ายเงิน (แสดงผลอย่างเดียว) · Member เดิม = CASH (P2)
-- payment_account = บัญชีธนาคาร JSONB {"bank_code","account_number","name_th",…,"surname_en"} · NULL = ไม่มี
-- ห้ามซ้ำที่คู่ (bank_code + account_number) เฉพาะภายใต้ผู้สร้างเดียวกัน (P4 · 402405)
-- win_limit_per_day = ลิมิตชนะต่อวัน ค่าเดียวทุกเกม หน่วยสกุลของ Member · 0 = ไม่จำกัด (P9)

-- +goose Up
ALTER TABLE user_members
    ADD COLUMN payment_type      VARCHAR(10)      NOT NULL DEFAULT 'CASH',
    ADD COLUMN payment_account   JSONB,
    ADD COLUMN win_limit_per_day DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD CONSTRAINT ck_user_members_payment_type CHECK (payment_type IN ('CREDIT', 'CASH')),
    ADD CONSTRAINT ck_user_members_win_limit CHECK (win_limit_per_day >= 0);
CREATE UNIQUE INDEX uq_user_members_payment_account ON user_members
    (agent_id, (payment_account ->> 'bank_code'), (payment_account ->> 'account_number')) WHERE payment_account IS NOT NULL;

-- +goose Down
DROP INDEX uq_user_members_payment_account;
ALTER TABLE user_members
    DROP CONSTRAINT ck_user_members_win_limit,
    DROP CONSTRAINT ck_user_members_payment_type,
    DROP COLUMN win_limit_per_day,
    DROP COLUMN payment_account,
    DROP COLUMN payment_type;
