-- deposit_withdraw · Superadmin มียอดจริงใน agent_balances (lead อนุมัติ 2026-10-15 · แทน "วงเงินไม่จำกัด" ของ DW-12 / DW-14)
-- ทุกบัญชี role SUPERADMIN: ครบ 27 สกุลใน agent_currencies + ยอดสกุลละ 100,000,000 (seed เต็ม ไม่หักยอดที่เคยปล่อยไปแล้ว)
-- ทุกการเปลี่ยนยอดมี ledger (กฎข้อ 11): reason SEED_SUPERADMIN · request_id 'seed-superadmin' · actor = Superadmin เอง
-- แถวที่มีอยู่แล้วไม่แก้ (ON CONFLICT DO NOTHING) · ledger เขียนเฉพาะแถวที่ insert จริง
-- รายการ 27 สกุลตรงกับ agentManagementCore.Currencies ณ วันที่เขียน (migration ห้ามแก้ภายหลัง)

-- +goose Up
INSERT INTO agent_currencies (agent_id, currency)
SELECT a.id, c.currency
FROM user_agents a
CROSS JOIN (VALUES ('ARS'), ('AUD'), ('BDT'), ('BOB'), ('BRL'), ('CLP'), ('CNY'), ('EUR'), ('GBP'),
                   ('HKD'), ('IDR'), ('INR'), ('JPY'), ('KHR'), ('KRW'), ('LAK'), ('MMK'), ('MXN'), ('MYR'), ('NGN'), ('PHP'),
                   ('PKR'), ('THB'), ('TWD'), ('USD'), ('USDT'), ('VND')) AS c(currency)
WHERE a.role = 'SUPERADMIN'
ON CONFLICT (agent_id, currency) DO NOTHING;

WITH seeded AS (
    INSERT INTO agent_balances (agent_id, currency, amount, updated_at)
    SELECT ac.agent_id, ac.currency, 100000000, now()
    FROM agent_currencies ac
    JOIN user_agents a ON a.id = ac.agent_id
    WHERE a.role = 'SUPERADMIN'
    ON CONFLICT (agent_id, currency) DO NOTHING
    RETURNING agent_id, currency, amount
)
INSERT INTO balance_ledger (owner_type, owner_id, currency, amount, balance_after, reason, request_id, actor_type, actor_id, created_at)
SELECT 'AGENT', agent_id, currency, amount, amount, 'SEED_SUPERADMIN', 'seed-superadmin', 'AGENT', agent_id, now()
FROM seeded;

-- +goose Down
-- ลบเฉพาะยอดที่ seed ไว้ (ถ้า Superadmin ใช้ยอดไปแล้ว ยอดที่เหลือถูกลบตาม) · agent_currencies ที่เพิ่มไม่ลบ (แยกไม่ได้ว่าแถวไหนเพิ่มในรอบนี้)
DELETE FROM balance_ledger WHERE reason = 'SEED_SUPERADMIN' AND request_id = 'seed-superadmin';
DELETE FROM agent_balances b USING user_agents a WHERE a.id = b.agent_id AND a.role = 'SUPERADMIN';
