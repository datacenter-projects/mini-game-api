-- agent_management MGMT-08 / MGMT-08A (lead review 2026-10-09 E1-5 / B1): เบอร์ 2 field phone_country_code + phone
-- ใช้กับ user_agents และ subaccounts · user_members ทำใน member_management
-- แปลงเบอร์เดิม: ขึ้นต้น 0 → 66 + ตัด 0 · ขึ้นต้น 66 และยาว 11 หลัก → 66 + ส่วนที่เหลือ · แบบอื่นล้างเป็น NULL (ไม่เดาประเทศ)
-- รายการที่ถูกล้าง (รวมเบอร์ที่ซ้ำกันหลังแปลง — เก็บแถวที่ id น้อยสุด) บันทึกใน migration_phone_cleared

-- +goose Up
CREATE TABLE migration_phone_cleared (
    id         BIGSERIAL   PRIMARY KEY,
    table_name VARCHAR(30) NOT NULL,
    row_id     BIGINT      NOT NULL,
    old_phone  VARCHAR(20) NOT NULL,   -- FORMAT = เบอร์เดิมก่อนแปลง · DUPLICATE = รหัส + เบอร์หลังแปลง
    reason     VARCHAR(20) NOT NULL,   -- FORMAT / DUPLICATE
    cleared_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---- user_agents ----
ALTER TABLE user_agents DROP CONSTRAINT ck_user_agents_phone;
DROP INDEX uq_user_agents_phone;
ALTER TABLE user_agents ADD COLUMN phone_country_code VARCHAR(3);

UPDATE user_agents SET phone_country_code = '66', phone = substr(phone, 2)
WHERE phone ~ '^0[1-9][0-9]*$' AND length(phone) <= 14;
UPDATE user_agents SET phone_country_code = '66', phone = substr(phone, 3)
WHERE phone_country_code IS NULL AND phone ~ '^66[1-9][0-9]{8}$';

INSERT INTO migration_phone_cleared (table_name, row_id, old_phone, reason)
SELECT 'user_agents', id, phone, 'FORMAT' FROM user_agents WHERE phone IS NOT NULL AND phone_country_code IS NULL;
UPDATE user_agents SET phone = NULL WHERE phone IS NOT NULL AND phone_country_code IS NULL;

INSERT INTO migration_phone_cleared (table_name, row_id, old_phone, reason)
SELECT 'user_agents', id, phone_country_code || phone, 'DUPLICATE' FROM (
    SELECT id, phone_country_code, phone, row_number() OVER (PARTITION BY phone_country_code, phone ORDER BY id) AS rn
    FROM user_agents WHERE phone IS NOT NULL
) d WHERE rn > 1;
UPDATE user_agents SET phone = NULL, phone_country_code = NULL
WHERE id IN (SELECT row_id FROM migration_phone_cleared WHERE table_name = 'user_agents' AND reason = 'DUPLICATE');

ALTER TABLE user_agents ADD CONSTRAINT ck_user_agents_phone CHECK (
    (phone IS NULL AND phone_country_code IS NULL) OR
    (phone_country_code ~ '^[1-9][0-9]{0,2}$' AND phone ~ '^[1-9][0-9]*$' AND length(phone_country_code) + length(phone) <= 15));
CREATE UNIQUE INDEX uq_user_agents_phone ON user_agents(phone_country_code, phone) WHERE phone IS NOT NULL;

-- ---- subaccounts (เบอร์ซ้ำได้ — MGMT-41 · ไม่มี unique) ----
ALTER TABLE subaccounts DROP CONSTRAINT ck_subaccounts_phone;
ALTER TABLE subaccounts ADD COLUMN phone_country_code VARCHAR(3);

UPDATE subaccounts SET phone_country_code = '66', phone = substr(phone, 2)
WHERE phone ~ '^0[1-9][0-9]*$' AND length(phone) <= 14;
UPDATE subaccounts SET phone_country_code = '66', phone = substr(phone, 3)
WHERE phone_country_code IS NULL AND phone ~ '^66[1-9][0-9]{8}$';

INSERT INTO migration_phone_cleared (table_name, row_id, old_phone, reason)
SELECT 'subaccounts', id, phone, 'FORMAT' FROM subaccounts WHERE phone IS NOT NULL AND phone_country_code IS NULL;
UPDATE subaccounts SET phone = NULL WHERE phone IS NOT NULL AND phone_country_code IS NULL;

ALTER TABLE subaccounts ADD CONSTRAINT ck_subaccounts_phone CHECK (
    (phone IS NULL AND phone_country_code IS NULL) OR
    (phone_country_code ~ '^[1-9][0-9]{0,2}$' AND phone ~ '^[1-9][0-9]*$' AND length(phone_country_code) + length(phone) <= 15));

-- +goose Down
-- คืนรูปแบบเดิม (เบอร์เดียว 8–15 หลัก): 66 → 0 + เบอร์ · รหัสอื่น → รหัส + เบอร์ · ที่เข้ารูปแบบเดิมไม่ได้ = NULL · เบอร์ที่ถูกล้างตอน Up ไม่คืน
ALTER TABLE subaccounts DROP CONSTRAINT ck_subaccounts_phone;
UPDATE subaccounts SET phone = CASE WHEN phone_country_code = '66' THEN '0' || phone ELSE phone_country_code || phone END WHERE phone IS NOT NULL;
UPDATE subaccounts SET phone = NULL WHERE phone IS NOT NULL AND phone !~ '^[0-9]{8,15}$';
ALTER TABLE subaccounts DROP COLUMN phone_country_code;
ALTER TABLE subaccounts ADD CONSTRAINT ck_subaccounts_phone CHECK (phone IS NULL OR phone ~ '^[0-9]{8,15}$');

DROP INDEX uq_user_agents_phone;
ALTER TABLE user_agents DROP CONSTRAINT ck_user_agents_phone;
UPDATE user_agents SET phone = CASE WHEN phone_country_code = '66' THEN '0' || phone ELSE phone_country_code || phone END WHERE phone IS NOT NULL;
UPDATE user_agents SET phone = NULL WHERE phone IS NOT NULL AND phone !~ '^[0-9]{8,15}$';
ALTER TABLE user_agents DROP COLUMN phone_country_code;
ALTER TABLE user_agents ADD CONSTRAINT ck_user_agents_phone CHECK (phone IS NULL OR phone ~ '^[0-9]{8,15}$');
CREATE UNIQUE INDEX uq_user_agents_phone ON user_agents(phone) WHERE phone IS NOT NULL;

DROP TABLE migration_phone_cleared;
