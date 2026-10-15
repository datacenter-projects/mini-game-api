-- asset AST-51: games/list หาช่อง game_cover ของทุกเกมด้วยชื่อช่องอย่างเดียว (ไม่ระบุ game_code) — uq_asset_types_slot (game_code, asset_type) ใช้ไม่ได้
-- แยกจาก 20261012160000_asset_create_tables เพราะไฟล์นั้นรันบน dev DB ไปแล้ว (แก้ไฟล์เดิมจะไม่ถูกรันซ้ำ)

-- +goose Up
CREATE INDEX idx_asset_types_asset_type ON asset_types(asset_type);

-- +goose Down
DROP INDEX idx_asset_types_asset_type;
