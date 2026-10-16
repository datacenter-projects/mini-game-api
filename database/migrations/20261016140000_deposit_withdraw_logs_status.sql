-- deposit_withdraw · status ของรายการฝาก / ถอน (maofoy 2026-10-16 · lead รับทราบ — DW-35 / DW-36)
-- SUCCESS = ย้ายเงินสำเร็จ (แถวเดิมทั้งหมด) · FAILED = ถูกปฏิเสธหลังหาลูกเจอ หรือระบบล้มระหว่างย้ายเงิน (ไม่มี ledger · เงินไม่ขยับ)
-- request_id unique เฉพาะแถว SUCCESS — แถว FAILED ไม่จอง request_id (กดซ้ำด้วยค่าเดิมได้ · DW-25 ดูจาก create_requests)
-- ตารางยังเล็ก (เพิ่ง deploy 14 ต.ค.) — ALTER / index ปกติได้ ไม่ต้องแยกขั้น (กฎข้อ 20)

-- +goose Up
ALTER TABLE deposit_withdraw_logs
    ADD COLUMN status VARCHAR(10) NOT NULL DEFAULT 'SUCCESS',
    ADD CONSTRAINT ck_deposit_withdraw_logs_status CHECK (status IN ('SUCCESS', 'FAILED')),
    DROP CONSTRAINT uq_deposit_withdraw_logs_request;
CREATE UNIQUE INDEX uq_deposit_withdraw_logs_request ON deposit_withdraw_logs(request_id) WHERE status = 'SUCCESS';

-- +goose Down
-- แถว FAILED ลบทิ้ง (request_id อาจซ้ำกับแถวอื่น — คืน unique เต็มไม่ได้ถ้าเก็บไว้)
DELETE FROM deposit_withdraw_logs WHERE status = 'FAILED';
DROP INDEX uq_deposit_withdraw_logs_request;
ALTER TABLE deposit_withdraw_logs
    ADD CONSTRAINT uq_deposit_withdraw_logs_request UNIQUE (request_id),
    DROP CONSTRAINT ck_deposit_withdraw_logs_status,
    DROP COLUMN status;
