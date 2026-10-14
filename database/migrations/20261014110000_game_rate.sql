-- game_rate (อัตราแพ้ชนะ · เมนู ⑥) — docs/modules/game_rate.md หัวข้อ 6 (lead review 2026-10-12)
-- ค่ากลางชุดเดียวทั้งระบบ · 1 แถว game_rate_versions ต่อการบันทึกที่ค่าเปลี่ยน (ห้าม UPDATE / DELETE — เป็น log คนแก้ RATE-08)
-- ค่า % / ตัวคูณ DOUBLE PRECISION ปัด 4 ตำแหน่ง (กฎข้อ 9 · lead RR1) · CHECK กันค่าหลุดช่วง · กฎ house edge ตรวจใน core/game_rate
-- รูปแบบอัตรา (RATE-01A): steps → game_rate_steps · prizes → win_percent ในแถวเวอร์ชัน + game_rate_prizes
-- version 20261014110000: เดิม 20261012160000 ชนกับ migration asset ของ carl ที่รันบน RDS dev แล้ว (goose เห็นว่ารันแล้วแต่ไม่มีตาราง) — เปลี่ยนเลขก่อน deploy ครั้งแรก

-- +goose Up
CREATE TABLE game_rate_versions (
    id             BIGSERIAL        PRIMARY KEY,
    game_code      VARCHAR(50)      NOT NULL,
    version        INT              NOT NULL CHECK (version >= 1),
    win_percent    DOUBLE PRECISION CHECK (win_percent > 0 AND win_percent <= 100),   -- รูปแบบ prizes เท่านั้น · steps = NULL
    actor_type     VARCHAR(10)      NOT NULL CHECK (actor_type IN ('AGENT', 'SUB')),
    actor_id       BIGINT           NOT NULL,
    created_by     VARCHAR(71)      NOT NULL,                                         -- username ณ เวลานั้น · sub = owner@name
    ip             VARCHAR(45)      NOT NULL DEFAULT '',
    request_id     VARCHAR(64)      NOT NULL DEFAULT '',
    rollback_from  INT              NOT NULL DEFAULT 0,                               -- RATE-05A · 0 = บันทึกปกติ
    created_at     TIMESTAMPTZ      NOT NULL DEFAULT now(),
    CONSTRAINT uq_game_rate_versions UNIQUE (game_code, version)   -- เลขไม่ซ้ำ + index หาเวอร์ชันล่าสุด / list ใหม่ → เก่า
);

CREATE TABLE game_rate_steps (               -- รูปแบบ steps (Coin Flip / RPS)
    version_id     BIGINT           NOT NULL REFERENCES game_rate_versions(id),
    step           SMALLINT         NOT NULL CHECK (step >= 1),
    multiplier     DOUBLE PRECISION NOT NULL CHECK (multiplier >= 1 AND multiplier <= 1000),
    win_percent    DOUBLE PRECISION NOT NULL CHECK (win_percent > 0 AND win_percent <= 100),
    PRIMARY KEY (version_id, step)
);

CREATE TABLE game_rate_prizes (              -- รูปแบบ prizes (Scratch Card)
    version_id     BIGINT           NOT NULL REFERENCES game_rate_versions(id),
    row_no         SMALLINT         NOT NULL CHECK (row_no BETWEEN 1 AND 10),
    multiplier     DOUBLE PRECISION NOT NULL CHECK (multiplier > 0 AND multiplier <= 1000),
    share_percent  DOUBLE PRECISION NOT NULL CHECK (share_percent > 0 AND share_percent <= 100),
    PRIMARY KEY (version_id, row_no)
);

-- +goose Down
DROP TABLE game_rate_prizes;
DROP TABLE game_rate_steps;
DROP TABLE game_rate_versions;
