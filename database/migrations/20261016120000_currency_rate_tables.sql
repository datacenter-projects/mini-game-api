-- currency_rate (docs/modules/currency_rate.md หัวข้อ 6 · lead review รอบ 2 · 2026-10-16)
-- 1) currency_rates: เรทรายเดือน (เรทของวันที่ 1) แทน Redis — usdt_rate ค่าดิบ + thb_rate (1 หน่วย = ? บาท) ปัด 8 ตำแหน่ง
-- 2) cronjobs: เปิด / ปิด job ตามเวลา + ผลรอบล่าสุด (ใช้กับทุก job)
-- ตารางใหม่ ไม่มีข้อมูลเดิม · ไม่ย้ายข้อมูลจาก Redis (job ดึงเดือนนี้ใหม่ตอน boot) — ไม่ต้องแยก 3 ขั้น (CLAUDE.md ข้อ 20)
-- PK (rate_month, currency) รองรับ query หลัก (อ่านแถวเดียว · นับสกุลของเดือน) — ไม่ต้องมี index เพิ่ม (ข้อ 24)

-- +goose Up
CREATE TABLE currency_rates (
    rate_month  DATE             NOT NULL,               -- วันที่ 1 ของเดือน (เวลาไทย)
    currency    VARCHAR(10)      NOT NULL,               -- 27 สกุลของระบบ
    usdt_rate   DOUBLE PRECISION NOT NULL,               -- 1 USDT = ? หน่วยของสกุลนี้ (ค่าดิบ ทศนิยม ≤ 8)
    thb_rate    DOUBLE PRECISION NOT NULL,               -- 1 หน่วยของสกุลนี้ = ? บาท (ปัด 8 ตำแหน่ง คิดครั้งเดียวตอน sync)
    synced_at   TIMESTAMPTZ      NOT NULL,               -- เวลาที่บันทึกแถวนี้จริง
    PRIMARY KEY (rate_month, currency),
    CONSTRAINT ck_currency_rates_positive CHECK (usdt_rate > 0 AND thb_rate > 0),
    CONSTRAINT ck_currency_rates_month_first_day CHECK (rate_month = date_trunc('month', rate_month)::date)
);

CREATE TABLE cronjobs (
    code             VARCHAR(50)  PRIMARY KEY,
    enabled          BOOLEAN      NOT NULL,
    last_started_at  TIMESTAMPTZ,
    last_finished_at TIMESTAMPTZ,
    last_status      VARCHAR(10)  NOT NULL DEFAULT '',   -- '' · SUCCESS · FAILED · SKIPPED
    last_error       VARCHAR(500) NOT NULL DEFAULT '',
    updated_by       VARCHAR(71)  NOT NULL DEFAULT 'system',
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT ck_cronjobs_last_status CHECK (last_status IN ('', 'SUCCESS', 'FAILED', 'SKIPPED'))
);

-- +goose Down
DROP TABLE cronjobs;
DROP TABLE currency_rates;
