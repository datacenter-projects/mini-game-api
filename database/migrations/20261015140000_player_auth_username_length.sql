-- player_auth (docs/modules/player_auth.md PA-06) · ผู้เล่นจาก operator ใช้ username {agent}@{player} (32 + @ + 32 = 65)
-- 1) ขยายคอลัมน์เป็น VARCHAR(65) (เปลี่ยน metadata อย่างเดียว Postgres ไม่เขียนตารางใหม่ · unique เดิม uq_user_members_username ใช้ต่อได้)
-- 2) CHECK เดิมรับแค่ ^[a-z0-9]{3,32}$ → รับเพิ่มรูปแบบ {a-z0-9 3–32}@{a-z0-9 3–32} · แถวเดิมทุกแถว (ไม่มี @) ผ่านเหมือนเดิม
-- ADD CONSTRAINT ตรวจทั้งตาราง — ตอนนี้ยังไม่มีข้อมูลจริงบน prod จึงรันตรงได้ไม่ต้องแยก 3 ขั้น (docs/DEPLOYMENT.md · CLAUDE.md ข้อ 20)

-- +goose Up
ALTER TABLE user_members ALTER COLUMN username TYPE VARCHAR(65);
ALTER TABLE user_members DROP CONSTRAINT ck_user_members_username;
ALTER TABLE user_members ADD CONSTRAINT ck_user_members_username
    CHECK (username = lower(username) AND username ~ '^[a-z0-9]{3,32}(@[a-z0-9]{3,32})?$');

-- +goose Down
-- ย้อนกลับไม่ได้ถ้ามีผู้เล่นแบบ agent@player แล้ว (ผิด CHECK เดิมและยาวเกิน 32) — Postgres จะ error ไม่ตัดข้อมูลเงียบๆ
-- ต้องลบแถวผู้เล่น operator ก่อนจึงย้อนได้
ALTER TABLE user_members DROP CONSTRAINT ck_user_members_username;
ALTER TABLE user_members ADD CONSTRAINT ck_user_members_username
    CHECK (username = lower(username) AND username ~ '^[a-z0-9]{3,32}$');
ALTER TABLE user_members ALTER COLUMN username TYPE VARCHAR(32);
