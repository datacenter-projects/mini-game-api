-- account ACC-04 แก้ 2026-10-08 (lead): เก็บ Key ตรงๆ ไม่เข้ารหัส · ไม่ใช้ API_KEY_ENCRYPTION_KEY
-- แถวเดิมถอด Key ไม่ได้แล้ว (ไม่มีกุญแจ) จึงลบทิ้ง — ตอนแก้ยังไม่มี env ไหนมีข้อมูลจริง (ตารางยังไม่ถูก deploy)
-- เจ้าของที่ถูกลบแถวได้ Key ใหม่ตอนเปิดหน้า 1.3 ครั้งแรก (ACC-05)

-- +goose Up
DELETE FROM api_credentials;  -- api_allowed_ips ลบตาม ON DELETE CASCADE
ALTER TABLE api_credentials
    DROP CONSTRAINT uq_api_credentials_key_hash,
    DROP COLUMN key_ciphertext,
    DROP COLUMN key_hash,
    ADD COLUMN api_key CHAR(64) NOT NULL,  -- hex ตัวพิมพ์เล็ก 64 ตัว (ACC-03)
    ADD CONSTRAINT uq_api_credentials_api_key UNIQUE (api_key);  -- index สำหรับค้นตอนลูกค้าเรียก API

-- +goose Down
DELETE FROM api_credentials;
ALTER TABLE api_credentials
    DROP CONSTRAINT uq_api_credentials_api_key,
    DROP COLUMN api_key,
    ADD COLUMN key_ciphertext BYTEA NOT NULL,
    ADD COLUMN key_hash CHAR(64) NOT NULL,
    ADD CONSTRAINT uq_api_credentials_key_hash UNIQUE (key_hash);
