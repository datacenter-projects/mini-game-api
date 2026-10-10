-- member_management MGMT-08 / MGMT-08A (lead review 2026-10-09 MQ1 · E1-5 / B1): เบอร์ 2 field phone_country_code + phone ของ user_members
-- แปลงเบอร์เดิมแบบเดียวกับ agent (20261009160000): ขึ้นต้น 0 → 66 + ตัด 0 · ขึ้นต้น 66 และยาว 11 หลัก → 66 + ส่วนที่เหลือ · แบบอื่นล้างเป็น NULL
-- ห้ามซ้ำที่คู่ (รหัส + เบอร์) เฉพาะภายใต้ผู้สร้างเดียวกัน (agent_id) · แปลงแล้วซ้ำ: Member id น้อยสุดเก็บเบอร์ไว้ แถวอื่นล้าง
-- รายการที่ถูกล้างบันทึกใน migration_phone_cleared (table_name = user_members · kept_member_id = Member ที่ได้เก็บเบอร์ไว้)

-- +goose Up
ALTER TABLE migration_phone_cleared ADD COLUMN kept_member_id BIGINT; -- DUPLICATE ของ user_members: Member ที่ได้เก็บเบอร์ไว้ · FORMAT = NULL

ALTER TABLE user_members DROP CONSTRAINT ck_user_members_phone;
DROP INDEX uq_user_members_phone;
ALTER TABLE user_members ADD COLUMN phone_country_code VARCHAR(3);

UPDATE user_members SET phone_country_code = '66', phone = substr(phone, 2)
WHERE phone ~ '^0[1-9][0-9]*$' AND length(phone) <= 14;
UPDATE user_members SET phone_country_code = '66', phone = substr(phone, 3)
WHERE phone_country_code IS NULL AND phone ~ '^66[1-9][0-9]{8}$';

INSERT INTO migration_phone_cleared (table_name, row_id, old_phone, reason)
SELECT 'user_members', id, phone, 'FORMAT' FROM user_members WHERE phone IS NOT NULL AND phone_country_code IS NULL;
UPDATE user_members SET phone = NULL WHERE phone IS NOT NULL AND phone_country_code IS NULL;

INSERT INTO migration_phone_cleared (table_name, row_id, old_phone, reason, kept_member_id)
SELECT 'user_members', id, phone_country_code || phone, 'DUPLICATE', kept FROM (
    SELECT id, phone_country_code, phone, row_number() OVER (PARTITION BY agent_id, phone_country_code, phone ORDER BY id) AS rn,
           min(id) OVER (PARTITION BY agent_id, phone_country_code, phone) AS kept
    FROM user_members WHERE phone IS NOT NULL
) d WHERE rn > 1;
UPDATE user_members SET phone = NULL, phone_country_code = NULL
WHERE id IN (SELECT row_id FROM migration_phone_cleared WHERE table_name = 'user_members' AND reason = 'DUPLICATE');

ALTER TABLE user_members ADD CONSTRAINT ck_user_members_phone CHECK (
    (phone IS NULL AND phone_country_code IS NULL) OR
    (phone_country_code ~ '^[1-9][0-9]{0,2}$' AND phone ~ '^[1-9][0-9]*$' AND length(phone_country_code) + length(phone) <= 15));
CREATE UNIQUE INDEX uq_user_members_phone ON user_members(agent_id, phone_country_code, phone) WHERE phone IS NOT NULL;

-- +goose Down
-- คืนรูปแบบเดิม (เบอร์เดียว 8–15 หลัก ห้ามซ้ำทั้งตาราง): 66 → 0 + เบอร์ · รหัสอื่น → รหัส + เบอร์ · เข้ารูปแบบเดิมไม่ได้ / ซ้ำข้ามผู้สร้าง = NULL
-- เบอร์ที่ถูกล้างตอน Up ไม่คืน
DROP INDEX uq_user_members_phone;
ALTER TABLE user_members DROP CONSTRAINT ck_user_members_phone;
UPDATE user_members SET phone = CASE WHEN phone_country_code = '66' THEN '0' || phone ELSE phone_country_code || phone END WHERE phone IS NOT NULL;
UPDATE user_members SET phone = NULL WHERE phone IS NOT NULL AND phone !~ '^[0-9]{8,15}$';
UPDATE user_members SET phone = NULL WHERE id IN (
    SELECT id FROM (SELECT id, row_number() OVER (PARTITION BY phone ORDER BY id) AS rn FROM user_members WHERE phone IS NOT NULL) d WHERE rn > 1);
ALTER TABLE user_members DROP COLUMN phone_country_code;
ALTER TABLE user_members ADD CONSTRAINT ck_user_members_phone CHECK (phone IS NULL OR phone ~ '^[0-9]{8,15}$');
CREATE UNIQUE INDEX uq_user_members_phone ON user_members(phone) WHERE phone IS NOT NULL;

DELETE FROM migration_phone_cleared WHERE table_name = 'user_members';
ALTER TABLE migration_phone_cleared DROP COLUMN kept_member_id;
