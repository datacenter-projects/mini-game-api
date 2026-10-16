-- game_rate · status ต่อขั้น (Coin / RPS) และต่อแถวรางวัล (Scratch) — ค่าที่หน้าบ้านเก็บไว้ (เช่น ล็อกขั้น / แถวตอนเกลี่ย %)
-- backend เก็บ / คืนค่าเท่านั้น ไม่กระทบสูตร และ engine · ไม่ส่ง = false · เวอร์ชันเดิมทั้งหมด = false
-- ตารางเล็ก (ขั้น × เวอร์ชัน) ADD COLUMN พร้อม DEFAULT ไม่ rewrite ตาราง (Postgres 11+)

-- +goose Up
ALTER TABLE game_rate_steps  ADD COLUMN status BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE game_rate_prizes ADD COLUMN status BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE game_rate_prizes DROP COLUMN status;
ALTER TABLE game_rate_steps  DROP COLUMN status;
