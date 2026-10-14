-- asset · ธีมเกม · branding (docs/modules/asset.md หัวข้อ 6) · MVP: catalog, config, used, theme, branding (ชื่อ + โลโก้)
-- เลื่อน (migration ใหม่ตอนทำ): site_brandings.favicon_key · site_brandings.site_theme + site_themes · site_banners · change log

-- +goose Up
CREATE TABLE asset_types (                          -- catalog ของช่อง (AST-11) · เพิ่ม / แก้ / เลิกใช้ด้วย migration เท่านั้น
    id            BIGSERIAL    PRIMARY KEY,
    game_code     VARCHAR(50)  NOT NULL,
    asset_type    VARCHAR(50)  NOT NULL,
    allowed_sizes JSONB        NOT NULL DEFAULT '[]', -- [{"width":544,"height":851}] · [] = ไม่ตรวจขนาด (AST-12)
    is_retired    BOOLEAN      NOT NULL DEFAULT false, -- เลิกใช้ = flag ไม่ลบแถว (AST-13)
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_asset_types_slot UNIQUE (game_code, asset_type),
    CONSTRAINT ck_asset_types_asset_type CHECK (asset_type ~ '^[a-z0-9_]{1,50}$'),
    CONSTRAINT ck_asset_types_sizes CHECK (jsonb_typeof(allowed_sizes) = 'array')
);

CREATE TABLE asset_configs (                        -- ทุกเวอร์ชันที่อัป = คลังรูป (AST-20)
    id            BIGSERIAL    PRIMARY KEY,
    scope         VARCHAR(10)  NOT NULL,
    agent_id      BIGINT       REFERENCES user_agents(id), -- NULL = GLOBAL
    asset_type_id BIGINT       NOT NULL REFERENCES asset_types(id),
    storage_key   VARCHAR(500) NOT NULL,
    created_by    VARCHAR(71)  NOT NULL,              -- username จริงของผู้กด (sub = username ของ sub)
    is_deleted    BOOLEAN      NOT NULL DEFAULT false, -- soft delete ทั้ง 2 scope (AST-28)
    deleted_at    TIMESTAMPTZ,
    deleted_by    VARCHAR(71),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT ck_asset_configs_scope CHECK (
        (scope = 'GLOBAL' AND agent_id IS NULL) OR (scope = 'AGENT' AND agent_id IS NOT NULL)),
    CONSTRAINT ck_asset_configs_deleted CHECK (is_deleted = (deleted_at IS NOT NULL))
);
CREATE INDEX idx_asset_configs_slot  ON asset_configs(asset_type_id, agent_id, created_at DESC) WHERE is_deleted = false;
CREATE INDEX idx_asset_configs_agent ON asset_configs(agent_id) WHERE agent_id IS NOT NULL;

CREATE TABLE asset_used (                           -- 1 แถวต่อ (เป้าหมาย, ช่อง) (AST-22)
    id              BIGSERIAL   PRIMARY KEY,
    scope           VARCHAR(10) NOT NULL,
    agent_id        BIGINT      REFERENCES user_agents(id), -- NULL = GLOBAL
    asset_type_id   BIGINT      NOT NULL REFERENCES asset_types(id),
    asset_config_id BIGINT      NOT NULL REFERENCES asset_configs(id) ON DELETE RESTRICT,
    created_by      VARCHAR(71) NOT NULL,
    updated_by      VARCHAR(71) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ck_asset_used_scope CHECK (
        (scope = 'GLOBAL' AND agent_id IS NULL) OR (scope = 'AGENT' AND agent_id IS NOT NULL))
);
-- Postgres มอง NULL ไม่เท่ากัน → unique ธรรมดากัน GLOBAL ซ้ำไม่ได้ ต้องแยก 2 ตัว (ห้ามรวม asset_config_id — AST-22)
-- ตัวแรกใช้เป็น index ของการอ่านทั้งสายด้วย agent_id = ANY(...) (AST-57)
CREATE UNIQUE INDEX uq_asset_used_agent_slot  ON asset_used(agent_id, asset_type_id) WHERE agent_id IS NOT NULL;
CREATE UNIQUE INDEX uq_asset_used_global_slot ON asset_used(asset_type_id)           WHERE agent_id IS NULL;
CREATE INDEX idx_asset_used_config ON asset_used(asset_config_id);   -- นับผู้ที่ใช้ config ตอนลบ (AST-28, AST-32)

CREATE TABLE game_themes (                          -- GLOBAL = ค่ากลาง · AGENT = override ของบัญชี (AST-37)
    id                    BIGSERIAL   PRIMARY KEY,
    scope                 VARCHAR(10) NOT NULL,
    agent_id              BIGINT      REFERENCES user_agents(id), -- NULL = GLOBAL
    game_code             VARCHAR(50) NOT NULL,
    font_family           VARCHAR(20) NOT NULL,       -- whitelist ตรวจที่ app/core (AST-34)
    main_color_1          CHAR(7)     NOT NULL,
    main_color_2          CHAR(7)     NOT NULL,
    stats_bar_color       VARCHAR(20) NOT NULL,       -- ชื่อสี ไม่ใช่ hex (AST-36)
    stats_bar_font_family VARCHAR(20) NOT NULL,
    created_by            VARCHAR(71) NOT NULL,
    updated_by            VARCHAR(71) NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ck_game_themes_scope CHECK (
        (scope = 'GLOBAL' AND agent_id IS NULL) OR (scope = 'AGENT' AND agent_id IS NOT NULL)),
    CONSTRAINT ck_game_themes_colors CHECK (main_color_1 ~ '^#[0-9a-f]{6}$' AND main_color_2 ~ '^#[0-9a-f]{6}$'),
    CONSTRAINT ck_game_themes_stats_bar_color CHECK (stats_bar_color IN ('main_color_1', 'main_color_2'))
);
CREATE UNIQUE INDEX uq_game_themes_agent  ON game_themes(agent_id, game_code) WHERE agent_id IS NOT NULL;
CREATE UNIQUE INDEX uq_game_themes_global ON game_themes(game_code)           WHERE agent_id IS NULL;

CREATE TABLE site_brandings (                       -- 1:1 กับบัญชี · ไม่มี scope / GLOBAL / ประวัติ (AST-41 · lead GQ8)
    agent_id   BIGINT       PRIMARY KEY REFERENCES user_agents(id),
    site_name  VARCHAR(50)  NOT NULL DEFAULT '',
    logo_key   VARCHAR(500) NOT NULL DEFAULT '',
    created_by VARCHAR(71)  NOT NULL,
    updated_by VARCHAR(71)  NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- seed: catalog ของช่อง (ขนาดมาจาก seed ของโค้ดเก่า — รอทีมเกมยืนยัน · แก้ด้วย migration ใหม่ — lead GQ7)
INSERT INTO asset_types (game_code, asset_type, allowed_sizes) VALUES
    ('scratch_card', 'background',        '[{"width":1920,"height":1080}]'),
    ('scratch_card', 'game_cover',        '[{"width":608,"height":1080}]'),
    ('scratch_card', 'bg_loading',        '[{"width":449,"height":52}]'),
    ('scratch_card', 'frame_loading',     '[{"width":449,"height":52}]'),
    ('scratch_card', 'background_main',   '[{"width":608,"height":1080}]'),
    ('scratch_card', 'main_scratch_area', '[{"width":544,"height":851}]'),
    ('scratch_card', 'outer_frame',       '[{"width":544,"height":851}]'),
    ('scratch_card', 'top_scratch',       '[{"width":544,"height":851}]'),
    ('scratch_card', 'main_play',         '[{"width":186,"height":76}]'),
    ('scratch_card', 'icon_button',       '[{"width":32,"height":32},{"width":40,"height":40}]'),
    ('scratch_card', 'auto_scratchers',   '[{"width":520,"height":347}]'),
    ('scratch_card', 'selection_button',  '[{"width":115,"height":76}]'),
    ('scratch_card', 'bet_limit',         '[{"width":322,"height":322}]'),
    ('scratch_card', 'bet_history',       '[{"width":572,"height":929}]'),
    ('scratch_card', 'bar_bet_history',   '[{"width":540,"height":74}]'),
    ('scratch_card', 'how_to_scratchers', '[{"width":572,"height":929}]'),
    ('scratch_card', 'winner',            '[{"width":502,"height":514}]'),
    ('scratch_card', 'you_lose',          '[{"width":544,"height":287}]'),
    ('scratch_card', 'win_highlight',     '[{"width":425,"height":425}]');

INSERT INTO asset_types (game_code, asset_type) VALUES
    ('coin_toss', 'background'), ('coin_toss', 'coin_heads'), ('coin_toss', 'coin_tails'),
    ('coin_toss', 'selection_button'), ('coin_toss', 'outer_frame'), ('coin_toss', 'bet_limit'),
    ('coin_toss', 'bet_history'), ('coin_toss', 'winner'), ('coin_toss', 'game_cover'),
    ('coin_toss', 'loading'), ('coin_toss', 'win_highlight'),
    ('rock_paper_scissors', 'background'), ('rock_paper_scissors', 'hand_rock'),
    ('rock_paper_scissors', 'hand_paper'), ('rock_paper_scissors', 'hand_scissors'),
    ('rock_paper_scissors', 'selection_button'), ('rock_paper_scissors', 'outer_frame'),
    ('rock_paper_scissors', 'bet_limit'), ('rock_paper_scissors', 'bet_history'),
    ('rock_paper_scissors', 'winner'), ('rock_paper_scissors', 'game_cover'),
    ('rock_paper_scissors', 'loading'), ('rock_paper_scissors', 'win_highlight');

-- seed: ธีมกลาง (แถว GLOBAL) ของทุกเกม (AST-37)
INSERT INTO game_themes (scope, agent_id, game_code, font_family, main_color_1, main_color_2,
                         stats_bar_color, stats_bar_font_family, created_by, updated_by)
SELECT 'GLOBAL', NULL, game_code, 'noto', '#f7447e', '#ffffff', 'main_color_1', 'prompt', 'migration', 'migration'
FROM (VALUES ('coin_toss'), ('rock_paper_scissors'), ('scratch_card')) AS g(game_code);

-- +goose Down
DROP TABLE site_brandings;
DROP TABLE game_themes;
DROP TABLE asset_used;
DROP TABLE asset_configs;
DROP TABLE asset_types;
