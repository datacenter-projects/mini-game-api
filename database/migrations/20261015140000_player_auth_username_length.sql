-- player_auth (docs/modules/player_auth.md PA-06) · ผู้เล่นจาก operator ใช้ username {agent}@{player} (32 + @ + 32 = 65)
-- ขยาย VARCHAR เป็นการเปลี่ยน metadata อย่างเดียว (Postgres ไม่เขียนตารางใหม่) · unique เดิม uq_user_members_username ใช้ต่อได้

-- +goose Up
ALTER TABLE user_members ALTER COLUMN username TYPE VARCHAR(65);

-- +goose Down
-- ย่อกลับเป็น 32 ไม่ได้ถ้ามีผู้เล่นแบบ agent@player แล้ว — ต้องลบแถวผู้เล่น operator ก่อนจึงย่อได้ (ถ้ายาวเกิน Postgres จะ error ไม่ตัดข้อมูลเงียบๆ)
ALTER TABLE user_members ALTER COLUMN username TYPE VARCHAR(32);
