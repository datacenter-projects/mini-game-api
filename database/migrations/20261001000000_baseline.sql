-- Baseline: จุดเริ่มต้นของ schema — ตารางของแต่ละ module จะถูกเพิ่มเป็นไฟล์ใหม่ตอน port module นั้น

-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- +goose Down
DROP EXTENSION IF EXISTS pgcrypto;
