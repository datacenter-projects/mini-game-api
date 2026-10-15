# อัตราแลกเปลี่ยน (`currency_rate`) — Spec (ฉบับใหม่ · แทนฉบับ 2026-10-07 ทั้งหมด)

- สถานะ: **DRAFT** (เขียนตาม lead review รอบ 1 + รอบ 2 — รอ lead review ก่อนรื้อโค้ด)
- อนุมัติโดย: — · วันที่: —
- ผู้ทำ: Boiledegg · branch `boiledegg/bo/currency-rate`
- ที่มาของการตัดสิน: [docs/reviews/currency-rate.md](../reviews/currency-rate.md) (lead zerph 2026-10-16 · รอบ 1 โครงสร้าง + การใช้กับเงิน · รอบ 2 business logic) — **รอบ 2 แทน spec เดิมทั้งหมด**
- ชื่อ module ในโค้ด: `currency_rate` (`app/core/currency_rate` · `app/service/currency_rate` · `app/repository/postgres/currency_rate` · `app/cronjob` · `platform/askmelotto`)
- ที่มาของ rule: คำตัดสินของ lead + พฤติกรรมของระบบแม่ askmelotto-api ที่ lead สรุปไว้ (ดึงต้นทางวันละครั้งตี 1 · ใช้เรทของวันที่ 1 ของเดือนทั้งเดือน) + ผลยิงเส้นจริงบน `https://dev-api.amblotto.net` (2026-10-07) · ไม่ได้ extract จากโค้ดเก่าของโปรเจกต์ minigame เดิม

> เอกสารนี้คือ source of truth ของ module — โค้ดและ test ต้องตรงกับเอกสารนี้ ถ้าไม่ตรงให้แก้โค้ด
> หรือแก้เอกสารผ่านการอนุมัติใหม่ · ข้อที่ผู้ร่างเสนอเองมีเครื่องหมาย **[เสนอ]** รอ lead ยืนยัน (รวมไว้ในหัวข้อ 10)

## 1. หน้าที่ของ module

เก็บ**เรทรายเดือน** (เรทของวันที่ 1 ของเดือน เวลาไทย) ของ 27 สกุล โดยดึงจากระบบ askmelotto ตรงกับระบบแม่ (CR-01) แล้วให้ **module เดิมพัน** แปลงยอดของ Member สกุลอื่นเป็นบาท เพื่อเช็ค `win_limit_per_day` (หน่วยบาทเสมอ — member-payment รอบ 2.1)

**ผู้ใช้หลัก = module เดิมพัน (Bet)** — เรทนี้ใช้กับเงิน จึงต้อง fail closed: ไม่มีเรท = ปฏิเสธการแทง (CR-24) · Dashboard ถูกตัดออกจาก scope แล้ว

**ไม่อยู่ใน module นี้:** แลก / ย้ายเงินข้ามสกุลจริง · การเก็บเรทที่ใช้ไว้กับรายการเดิมพัน (lead C3 → spec Bet) · endpoint ของเราเอง (ไม่มี — job ภายในและฟังก์ชันให้ service อื่นเรียก) · เส้น BO เปิด / ปิด cronjob (Superadmin ทำทีหลัง)

## 2. คำศัพท์

| คำ | ความหมาย |
|---|---|
| เรทรายเดือน | เรทของ `rate_month` = วันที่ 1 ของเดือน (เวลาไทย) ใช้ทั้งเดือน — ไม่ใช่เรทรายวัน |
| `usdt_rate` | 1 USDT = กี่หน่วยของสกุลนั้น (ค่าดิบจาก askmelotto เช่น THB `33.62`) |
| `thb_rate` | **1 หน่วยของสกุลนั้น = กี่บาท** (คิดครั้งเดียวตอน sync) เช่น USD = `33.95959596` |
| askmelotto | ระบบภายนอกที่เป็นแหล่งเรท (`POST {ASKMELOTTO_API_BASE_URL}/api/ex/rate/usdt`) |
| รอบ sync | การทำงานของ job `currency_rate_sync` 1 ครั้ง |
| cronjob | งานตามเวลาใน `app/cronjob/` รันใน process เดียวกับ API เฉพาะ pod ที่ `IS_CRONJOB=true` |

## 3. Business rules

### 3.1 เรทและการเก็บ

| ID | Rule |
|---|---|
| CR-01 | **เรทรายเดือน**: เรทของวันที่ 1 ของเดือน (เวลาไทย `Asia/Bangkok`) ใช้ทั้งเดือน ตรงกับระบบแม่ (R1) · เก็บทุกเดือนเป็นประวัติ |
| CR-02 | **ขอ askmelotto ด้วยวันที่ 1 ของเดือนเสมอ** (`date = YYYY-MM-01` ของเดือนปัจจุบันตามเวลาไทย) ไม่ใช่วันปัจจุบัน (S2-1) · timeout 10 วินาที · ไม่ต้องยืนยันตัวตน |
| CR-03 | `thb_rate` = `usdt_rate(THB) ÷ usdt_rate(สกุลนั้น)` คำนวณด้วย `big.Rat` จากข้อความทศนิยมของ response (ไม่ผ่าน float) แล้ว**เก็บ 8 ตำแหน่ง** · THB = `1` · เช่น USD `33.62 ÷ 0.99 = 33.95959596` · USDT (`usdt_rate = 1`) `thb_rate` = `usdt_rate(THB)` |
| CR-04 | ได้ response ไม่ครบ → บันทึก**เฉพาะสกุลที่ได้** · สกุลที่ขาดคือไม่มีเรทของเดือนนี้ (**ไม่ใช้เรทเดือนก่อน** — S3-2) · **ไม่มี THB = ไม่บันทึกทั้งเดือน** (ไม่มีฐานคิด `thb_rate`) |
| CR-05 | เรทของเดือนที่บันทึกแล้ว**ไม่เขียนทับอัตโนมัติ** (`INSERT … ON CONFLICT DO NOTHING`) · แก้ได้ด้วยคำสั่งมือ `-force` เท่านั้น (ขอ lead — CR-17) (S3-3) |
| CR-06 | เก็บใน **ตาราง `currency_rates`** (Postgres) แทน Redis ทั้งหมด (R2) · อ่านจาก DB ตรง ไม่มี cache (กฎข้อ 14) · Redis เหลือใช้แค่ lock ของ job (CR-13) |
| CR-07 | ตัวเลขเรททุกตัว (ทั้ง `usdt_rate` / `thb_rate` / ยอดแปลง) เป็น `float64` / `DOUBLE PRECISION` ตามกฎข้อ 9 — ค่าที่เก็บผ่านการปัด 8 ตำแหน่งด้วย `big.Rat` แล้วก่อนแปลงเป็น float · ยอดที่แปลงแล้วปัด 4 ตำแหน่ง (`utils.Round4`) (M1) |

### 3.2 job `currency_rate_sync`

| ID | Rule |
|---|---|
| CR-10 | **ทุก 15 นาที + รันทันทีตอน boot** (J4-1) — ความถี่อยู่ในโค้ด ไม่เก็บในตาราง (T1) · รอบที่เดือนปัจจุบันมีครบ 27 สกุลแล้วไม่ยิง askmelotto (CR-14) |
| CR-11 | รันเฉพาะ pod ที่ **`IS_CRONJOB=true`** (ค่าเริ่มต้น `false`) เปิดเฉพาะ deployment เดียว (แบบ askmelotto-api) (J2) · **แทน `IS_CURRENCY_RATE_SYNC` (ลบ)** |
| CR-12 | ทุกรอบอ่าน `enabled` ของแถว `currency_rate_sync` ในตาราง `cronjobs` ก่อน — ปิด = **ข้ามเงียบ ไม่ยิง ไม่แก้แถว** (C6-2 · หัวข้อ 6.2) |
| CR-13 | **คง Redis lock** กันหลาย pod ซ้ำ (J4-2): `SET currency_rate:sync:lock {token} NX PX 300000` (5 นาที) · ได้ lock = รัน · ไม่ได้ = `SKIPPED` · ปล่อย lock เฉพาะเมื่อ token ยังเป็นของตัวเอง (Lua compare-and-delete) |
| CR-14 | flow: (1) เดือนปัจจุบัน (เวลาไทย) ครบ 27 สกุลแล้ว → จบ ไม่ยิง (`SUCCESS`) (2) ยิง askmelotto ด้วยวันที่ 1 (3) ตรวจ response (CR-15) (4) `INSERT … ON CONFLICT DO NOTHING` เฉพาะสกุลที่ยังไม่มี (5) ยังไม่ครบ 27 → `FAILED` บอกสกุลที่ขาด · ครบ → `SUCCESS` · ทุกสกุลที่ผ่านตรวจถูกบันทึกไม่ว่ารอบนั้นจะ `FAILED` หรือไม่ |
| CR-15 | **ตรวจ response** (J4-3): HTTP 200 **และ** `code = 200` **และ** `data.base_currency = "USDT"` **และ** `data.date` = วันที่ที่ขอ (ไม่ตรง = ไม่บันทึกอะไรเลย) **และ** `data.json_rate` เป็น object · แต่ละสกุล: JSON number > 0 · ทศนิยมไม่เกิน 8 ตำแหน่ง · อยู่ใน 27 สกุลของระบบ (สกุลอื่นไม่สนใจ) · สกุลที่ไม่ผ่านไม่บันทึกและ log บอกสกุล · error ของ askmelotto ตอบ HTTP 200 แต่ `code` ≠ 200 (`430502` รูปแบบวันที่ผิด · `430503` ยังไม่มีเรทของวันนั้น) |
| CR-16 | **log** (J4-4): ยังไม่ครบก่อนวันที่ 1 เวลา **03:00** (เวลาไทย) ของเดือน = `info` (askmelotto สร้างเรทใหม่ตอนตี 1 ช้าได้) · หลังจากนั้นยังไม่ครบ = `warn` ทุกรอบ · log มี `code` / `msg` ของ askmelotto และสกุลที่ขาด/ผิด · **ไม่ log URL ที่มี secret / ไม่ log body ดิบเกินจำเป็น** |
| CR-17 | **คำสั่งมือ** `scripts/currency_rate_sync -month 2026-10 [-force]` (J4-5): ใช้ **service เดียวกับ job** · ไม่ใส่ `-force` = เติมเฉพาะสกุลที่ขาด · `-force` = เขียนทับทั้งเดือน · รันบน env ร่วม (dev / uat / prod) **ต้องขอ lead ก่อน** · `-month` ต้องเป็นเดือนที่ askmelotto มีเรท (ขอด้วยวันที่ 1 ของเดือนนั้น) |
| CR-18 | job เรียกได้เฉพาะ **service** (ไม่มี business logic ใน `app/cronjob`) · ตัวยิง HTTP ไป askmelotto อยู่ที่ `platform/askmelotto` (client เดียวของทุกเส้น askmelotto — J3) · goroutine มีขอบเขต รับ ctx หยุดตอน server ปิด (กฎข้อ 27) |

### 3.3 ฟังก์ชันแปลง (ให้ module เดิมพันใช้)

| ID | Rule |
|---|---|
| CR-20 | `ConvertToTHBService(ctx, db, amount, currency, at) (THBConversion, error)` — แปลง**เป็นบาทอย่างเดียว** (S5-3) · ผลมี `AmountTHB` · `RateMonth` · `THBRate` (หัวข้อ 5) |
| CR-21 | **เดือนที่ใช้ = เดือนของ `at` (เวลาเกิดรายการ) ตามเวลาไทย** (S5-2) เช่น `at` = 31 ต.ค. 23:59 ใช้เรทเดือน ต.ค. · 1 พ.ย. 00:01 ใช้เรทเดือน พ.ย. · spec Bet กำหนดว่า `at` คือเวลาแทงหรือเวลาจ่าย |
| CR-22 | `AmountTHB = Round4(amount × THBRate)` — คำนวณด้วย `big.Rat` **ปัดครั้งเดียวตอนท้าย** แบบปัดครึ่งขึ้น (กฎข้อ 9 · M1) ตัวอย่าง: `100 USD × 33.95959596 = 3,395.9596 บาท` |
| CR-23 | สกุล `THB` → คืนทันที **ไม่อ่าน DB** (`THBRate = 1` · `RateMonth` = วันที่ 1 ของเดือนของ `at`) — Member THB ไม่ต้องมีเรท ไม่โดนปฏิเสธ |
| CR-24 | **ไม่มีเรทของเดือนนั้น → `ErrCurrencyRateUnavailable`** (error module `09`) + log `warn` (สกุล / เดือน) · **ไม่ใช้เรทเดือนก่อน** · module เดิมพันแปลงเป็น code ของตัวเองและ **ปฏิเสธการแทง (fail closed — lead C1)** · สกุลที่ระบบไม่รู้จัก = error เดียวกัน |
| CR-25 | **แทน "เรทอายุ 24 ชั่วโมง" ของรอบ 1 (C2): ต้องมีเรทของเดือนนั้น** — ไม่มี = ปฏิเสธ · ไม่มีเกณฑ์อายุเรทแบบชั่วโมงอีก (เรทรายเดือนคงที่ทั้งเดือน) |
| CR-26 | อ่าน **1 แถวด้วย primary key** `(rate_month, currency)` · ใช้ `db` / tx ของผู้เรียก · ไม่ lock · ไม่มี endpoint · ผู้เรียกเก็บ `THBRate` + `RateMonth` ไว้กับรายการเดิมพัน (C3 — ออกแบบใน spec Bet) |
| CR-27 | `amount` ต้องเป็นเลขจำกัด (ไม่ NaN / Inf) และไม่ติดลบ — ผิด = error ภายใน ไม่ใช่ `ErrCurrencyRateUnavailable` (ผู้เรียกส่งค่าผิด ไม่ใช่เรทหาย) |

## 4. สิ่งที่พบในโค้ดเดิม (merge เข้า dev `72f48d5` ก่อน review) และการตัดสินใจ

| # | โค้ดเดิมทำอะไร | ผล review | การตัดสินใจ (ที่ต้องรื้อ) |
|---|---|---|---|
| 1 | เรทรายวัน (วันปัจจุบัน) ดึงทุก 15 นาที ทับค่าเดิมเสมอ | ไม่ตรงระบบแม่ (ใช้เรทวันที่ 1 ทั้งเดือน) | CR-01 / CR-02 / CR-05 |
| 2 | เก็บใน Redis hash เป็นจำนวนเต็ม ×10^8 ไม่มี TTL | Redis หาย = ไม่มีเรท · ไม่ใช่แหล่งข้อมูลถาวร | CR-06 ตาราง `currency_rates` (float 8 ตำแหน่ง) |
| 3 | `ConvertMinor` ใช้ int64 หน่วย 1/100 ปัด 2 ตำแหน่ง · "แสดงผลเท่านั้น" | ฝ่าฝืนกฎข้อ 9 (float64 + Round4) · ใช้กับเงินไม่ได้ (M1) | CR-20 – CR-22 |
| 4 | ไม่มีเรท = `errors.New("currency rate unavailable")` | ไม่ใช่ `apperr` · bet แยก error ไม่ได้ (กฎข้อ 15) | CR-24 `ErrCurrencyRateUnavailable` module `09` |
| 5 | worker เป็น goroutine ใน service เริ่มจาก `application.go` ผ่าน `IS_CURRENCY_RATE_SYNC` | ทุก pod ยิงได้ · ปิด/เปิดต้องแก้ env | CR-11 / CR-12 `app/cronjob` + `IS_CRONJOB` + ตาราง `cronjobs` |
| 6 | ยิง HTTP อยู่ใน service | ไม่มีที่กลางของ client ภายนอก | CR-18 `platform/askmelotto` (J3) |
| 7 | `synced_at` อัปเดตแม้ไม่มีสกุลไหนได้เรทใหม่ | ผิด — แยกไม่ออกว่าข้อมูลสด | `synced_at` = เวลาที่บันทึกแถวนั้นจริง (หัวข้อ 6) · ไม่แตะเมื่อไม่ได้เขียน |
| 8 | รายการ 27 สกุลซ้ำกับ `agent_management` | ต้องมีที่เดียว | **[เสนอ]** ใช้ `agentManagementCore.Currencies` (หัวข้อ 10 ข้อ 1) |
| 9 | const key ของ currency_rate ค้างใน `app/repository/redis/agent_auth/keys.go` (เกิดจาก merge `1b763f6`) | ผิดที่ผิดทาง | ลบ · key เหลือแค่ lock ใน `redis/currency_rate/keys.go` |
| 10 | env `IS_CURRENCY_RATE_SYNC` · `CURRENCY_RATE_API_BASE_URL` | ชื่อไม่ตรงโครงใหม่ (ใช้กับทุกเส้น askmelotto) | `IS_CRONJOB` · `ASKMELOTTO_API_BASE_URL` (ลบชื่อเดิม) |

## 5. ฟังก์ชันแปลง (สัญญากับ module เดิมพัน)

```go
// app/service/currency_rate
func ConvertToTHBService(ctx context.Context, db *gorm.DB, amount float64, currency string, at time.Time) (THBConversion, error)

type THBConversion struct {
    AmountTHB float64   // Round4(amount × THBRate) — ปัดครั้งเดียวตอนท้าย ครึ่งขึ้น
    RateMonth time.Time // วันที่ 1 ของเดือนที่ใช้ (เวลาไทย)
    THBRate   float64   // 1 หน่วยของสกุลนั้น = ? บาท ของเดือนนั้น
}
```

| ตัวอย่าง | currency | at (เวลาไทย) | เรทในตาราง | ผล |
|---|---|---|---|---|
| ปกติ | USD | 2026-10-16 | `2026-10-01` USD `thb_rate 33.95959596` | `AmountTHB 3395.9596` (100 USD) · `RateMonth 2026-10-01` · `THBRate 33.95959596` |
| บาท | THB | ใดๆ | — (ไม่อ่าน DB) | `AmountTHB = amount` · `THBRate 1` |
| ข้ามเดือน | USD | 2026-10-31 23:59 / 2026-11-01 00:01 | ต.ค. / พ.ย. คนละแถว | ใช้เรทเดือน ต.ค. / พ.ย. ตามลำดับ |
| ไม่มีเรท | VND | 2026-11-01 00:10 (เรท พ.ย. ยังไม่มา) | ไม่มี | `ErrCurrencyRateUnavailable` (ไม่ใช้เรท ต.ค.) |

## 6. Schema · cronjobs · Config

### 6.1 ตาราง `currency_rates` (R2)

```sql
CREATE TABLE currency_rates (
    rate_month  DATE             NOT NULL,   -- วันที่ 1 ของเดือน (เวลาไทย)
    currency    VARCHAR(10)      NOT NULL,   -- 27 สกุล
    usdt_rate   DOUBLE PRECISION NOT NULL,   -- 1 USDT = ? หน่วยของสกุลนี้ (ค่าดิบ ทศนิยม ≤ 8)
    thb_rate    DOUBLE PRECISION NOT NULL,   -- 1 หน่วยของสกุลนี้ = ? บาท (ปัด 8 ตำแหน่ง คิดครั้งเดียวตอน sync)
    synced_at   TIMESTAMPTZ      NOT NULL,   -- เวลาที่บันทึกแถวนี้จริง
    PRIMARY KEY (rate_month, currency),
    CHECK (usdt_rate > 0 AND thb_rate > 0),
    CHECK (rate_month = date_trunc('month', rate_month)::date)
);
```
- PK `(rate_month, currency)` รองรับ query หลักของ `ConvertToTHBService` (อ่านแถวเดียว) และ "นับสกุลของเดือนนี้" ของ job — ไม่ต้องมี index เพิ่ม (กฎข้อ 24)
- ไม่มีคอลัมน์ `updated_at` เพราะเรทที่บันทึกแล้วไม่ถูกแก้ (CR-05) — `-force` เขียนทับพร้อมตั้ง `synced_at` ใหม่

### 6.2 ตาราง `cronjobs` (ใช้กับทุก job)

```sql
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
    CHECK (last_status IN ('', 'SUCCESS', 'FAILED', 'SKIPPED'))
);
```

| ID | Rule |
|---|---|
| CJ-01 | `app/cronjob/cronjob.go` ห่อทุก job: `Start(ctx)` เป็นจุดเดียวที่ `application.go` เรียก · เรียกได้เฉพาะ service · ไม่มี business logic (J1) |
| CJ-02 | boot: `INSERT (code, enabled = ค่าเริ่มต้นในโค้ด) ON CONFLICT DO NOTHING` — **ไม่ทับค่าที่ตั้งไว้แล้ว** |
| CJ-03 | ค่าเริ่มต้นกำหนดต่อ job ในโค้ด (C6-1): `currency_rate_sync` = **เปิด** · job เสี่ยงในอนาคต = ปิด |
| CJ-04 | แต่ละรอบ: อ่าน `enabled` — ปิด = **ข้ามเงียบ ไม่แก้แถว** (C6-2) · lock ไม่ได้ = `SKIPPED` · รัน = บันทึก `last_started_at` / `last_finished_at` / `last_status` / `last_error` (ตัดความยาวไม่เกิน 500) |
| CJ-05 | เปิด / ปิดด้วย **SQL สำเร็จรูปใน DEPLOYMENT.md** (C6-3) · บน env ร่วม**ขอ lead ก่อน** · ตั้ง `updated_by = 'sql:{ชื่อ}'` เช่น<br>`UPDATE cronjobs SET enabled = false, updated_by = 'sql:zerph', updated_at = now() WHERE code = 'currency_rate_sync';` |
| CJ-06 | **metrics** (C6-4): `cronjob_runs_total{job,status}` · `cronjob_last_success_timestamp{job}` (ตั้ง alert ได้) · ไม่เก็บประวัติทุกรอบ (T3 — `cronjob_runs` ยังไม่ทำ · ดูรอบเก่าจาก log / metrics) |

### 6.3 Config

| env | ค่าเริ่มต้น | กฎ |
|---|---|---|
| `IS_CRONJOB` | `false` | `true` = pod นี้รัน cronjob ทุกตัว · เปิด deployment เดียว · **แทน `IS_CURRENCY_RATE_SYNC` (ลบ)** |
| `ASKMELOTTO_API_BASE_URL` | local / dev = `https://dev-api.amblotto.net` | ต้องเป็น **origin เปล่า** (`scheme://host[:port]`) ไม่มี path / query / `/` ท้าย ไม่งั้น server ไม่เริ่ม · **uat / prod บังคับตั้ง** (ไม่ตั้ง = boot ไม่ขึ้น) · **แทน `CURRENCY_RATE_API_BASE_URL` (ลบ)** · ใช้กับทุกเส้นของ askmelotto · ถ้าต้องยืนยันตัวตนภายหลังเพิ่ม `ASKMELOTTO_API_KEY` |

- เพิ่ม / ลบ key ใน `pkg/configs` และ `.env.example` ใน PR เดียวกัน (กฎข้อ 25) · `docs/DEPLOYMENT.md` อัปเดตตาม (ตาราง env · SQL เปิด/ปิด cronjob · deployment ไหนตั้ง `IS_CRONJOB`)

### 6.4 Migration

- migration ใหม่ (ไฟล์เดียว `make migration name=currency_rate_tables`): สร้าง `currency_rates` + `cronjobs` · มี Up / Down ครบ (Down ลบทั้งสองตาราง)
- **ไม่ย้ายข้อมูลจาก Redis** (job ดึงเดือนนี้ใหม่ตอน boot ภายในไม่กี่วินาที) · key Redis เดิม `currency_rate:usdt` · `currency_rate:usdt:meta` **ปล่อยไว้** ไม่ลบ ไม่อ่าน
- ตารางใหม่ ไม่มีข้อมูลเดิม ไม่ต้องแยก 3 ขั้น (กฎข้อ 20)

## 7. เส้นภายนอกที่เรียก (askmelotto)

`POST {ASKMELOTTO_API_BASE_URL}/api/ex/rate/usdt` · header `Content-Type: application/json` · **`date` = วันที่ 1 ของเดือนเสมอ** (CR-02)

Request:
```json
{ "date": "2026-10-01" }
```

Response สำเร็จ (ยิงจริง 2026-10-07 ด้วยวันปัจจุบัน · 0.8–1.8 วินาที — รูปแบบเดียวกับวันที่ 1):
```json
{
  "code": 200,
  "data": {
    "date": "2026-10-01",
    "base_currency": "USDT",
    "json_rate": {
      "THB": 33.62, "MMK": 2099.53, "CNY": 6.7, "EUR": 0.88, "GBP": 0.75, "IDR": 17849.27, "INR": 96.37,
      "JPY": 158.15, "KHR": 4046.24, "KRW": 1337.9, "LAK": 22394.92, "PHP": 62.77, "USD": 0.99, "VND": 25988.07,
      "AUD": 1.43, "TWD": 31.78, "MYR": 4.08, "HKD": 7.84, "MXN": 17.97, "USDT": 1, "BRL": 4.98, "BDT": 122.75,
      "PKR": 276.6, "CLP": 973.67, "ARS": 1520.54, "BOB": 11.5, "NGN": 1324.42
    }
  },
  "msg": { "en": "Success", "th": "สำเร็จ" }
}
```
`json_rate` = 1 USDT = ? หน่วยของสกุลนั้น · ค่าตัวอย่างข้างบนเป็นของเรทที่ยิงจริง (ไม่ใช่เรทของวันที่ 1 เดือนใดโดยเฉพาะ)

Response error (HTTP 200 ทุกกรณี):

| `code` | เมื่อ | ผลของเรา |
|---|---|---|
| 430502 | ไม่ส่ง `date` หรือรูปแบบผิด (ต้อง `YYYY-MM-DD`) | `FAILED` · บั๊กฝั่งเรา (log error) |
| 430503 | askmelotto ยังไม่มีเรทของวันนั้น (เช่น ก่อนตี 1 ของวันที่ 1 หรือยังไม่สร้าง) | `FAILED` · log ตาม CR-16 (info ก่อน 03:00 / warn หลังจากนั้น) · รอรอบถัดไป |

error ที่เรา (ผู้เรียกฟังก์ชันแปลง) ส่งออก: `ErrCurrencyRateUnavailable` (หัวข้อ 8)

## 8. Error code (module `09` = currency_rate)

| Code | HTTP | ความหมาย |
|---|---|---|
| **[เสนอ]** 409401 | 200 | ไม่มีเรทของเดือน / สกุลนี้ (`ErrCurrencyRateUnavailable`) — module เดิมพันแปลงเป็น code ของตัวเอง ไม่ส่งเลขนี้ถึง client ตรงๆ |

ไฟล์ `pkg/apperr/currency_rate_errors.go` · จองใน `docs/ERROR_CODES.md` ก่อนประกาศ (กฎข้อ 15) · ไม่มี endpoint จึงไม่มี error validation (422)

## 9. Test cases

| Rule | Input | Expected |
|---|---|---|
| CR-02 | job รันวันที่ 16 ต.ค. | ส่ง `date = 2026-10-01` (ไม่ใช่ 2026-10-16) |
| CR-02 / CR-21 | 31 ต.ค. 17:30 UTC (= 1 พ.ย. 00:30 ไทย) | เดือนที่ขอ = `2026-11-01` · แปลงด้วย `at` นี้ใช้เรท พ.ย. |
| CR-03 | USD `0.99` · THB `33.62` | `thb_rate = 33.95959596` · THB = `1` · USDT = `33.62` |
| CR-03 | ทศนิยมยาวที่ปัดต้องเลื่อนหลัก (เช่น ผลหาร `…999999995`) | ปัดครึ่งขึ้นที่ 8 ตำแหน่งด้วย `big.Rat` ไม่ใช่ float |
| CR-04 | response ขาด `KRW` | บันทึก 26 สกุล · `FAILED` บอก KRW · สกุลอื่นไม่ถูกแตะ · KRW ไม่ใช้เรทเดือนก่อน |
| CR-04 | response ไม่มี `THB` | ไม่บันทึกอะไรเลย · `FAILED` |
| CR-05 | เดือนมีเรทแล้ว ยิงรอบใหม่ได้ค่าต่างออกไป | ไม่เขียนทับ · แถวเดิมคงเดิม (รวม `synced_at`) |
| CR-05 / CR-17 | คำสั่งมือไม่ใส่ `-force` / ใส่ `-force` | เติมเฉพาะที่ขาด / เขียนทับทั้งเดือน |
| CR-10 / CR-14 | เดือนนี้มีครบ 27 สกุลแล้ว | ไม่ยิง · `SUCCESS` |
| CR-11 | `IS_CRONJOB=false` | ไม่เริ่ม cronjob · ไม่ยิง · ไม่สร้างแถว `cronjobs` |
| CR-12 / CJ-04 | `enabled = false` | ข้ามรอบเงียบ · ไม่ยิง · `last_*` ไม่เปลี่ยน |
| CR-13 | 2 pod เริ่มรอบพร้อมกัน | ยิง 1 ครั้ง · อีกตัว `SKIPPED` |
| CR-13 | pod ที่ถือ lock ค้างจน lock หมดอายุ แล้วอีก pod ได้ lock | pod แรกไม่ลบ lock ของ pod หลัง |
| CR-15 | `code = 430503` | `FAILED` · ไม่บันทึก · log info ถ้าก่อน 03:00 วันที่ 1 / warn ถ้าหลังจากนั้น |
| CR-15 | `data.date` ≠ วันที่ที่ขอ · `base_currency = "USD"` · HTTP 500 / timeout / body ไม่ใช่ JSON | ไม่บันทึกอะไรเลย · `FAILED` |
| CR-15 | `"THB": 0` / `-1` / `"33.62"` (string) / ทศนิยม 9 ตำแหน่ง · มีสกุล `XYZ` | สกุลนั้นไม่บันทึก (log บอกสกุล) · `XYZ` ไม่สนใจ |
| CJ-02 | boot ครั้งที่ 2 หลังปิด job ด้วย SQL | ไม่ทับ `enabled = false` |
| CJ-04 / CJ-06 | รอบสำเร็จ / ล้ม / ข้าม | `last_status` = SUCCESS / FAILED / SKIPPED · `last_error` ตัด ≤ 500 · metrics `cronjob_runs_total` + `cronjob_last_success_timestamp` เปลี่ยนตาม |
| CR-20 – CR-22 | 100 USD (`thb_rate 33.95959596`) | `AmountTHB 3395.9596` · `RateMonth` · `THBRate` ครบ |
| CR-22 | ยอดที่ผลคูณลงท้าย `…5` ที่ตำแหน่งที่ 5 | ปัดครึ่งขึ้น ที่ 4 ตำแหน่ง · ปัดครั้งเดียว |
| CR-23 | THB 250.5 | `AmountTHB 250.5` · `THBRate 1` · **ไม่มี query ต่อ DB** |
| CR-24 | VND ไม่มีเรทเดือนนี้ (มีแต่เดือนก่อน) | `ErrCurrencyRateUnavailable` · ไม่ใช้เรทเดือนก่อน · log warn มีสกุล / เดือน |
| CR-24 | สกุล `XYZ` | `ErrCurrencyRateUnavailable` |
| CR-26 | แปลง 1 ครั้ง | query เดียวด้วย PK · ใช้ tx ของผู้เรียก (ไม่เปิด tx เอง) |
| CR-27 | `amount` = NaN / Inf / ติดลบ | error ภายใน (ไม่ใช่ `ErrCurrencyRateUnavailable`) |
| config | `IS_CRONJOB=false` (ค่าเริ่มต้น) | server ไม่เริ่ม cronjob |
| config | `ASKMELOTTO_API_BASE_URL=https://x.example/api` / `https://x.example/` | server ไม่เริ่ม (ต้องเป็น origin เปล่า) |
| config | `APP_ENV=prod` / `uat` ไม่ตั้ง `ASKMELOTTO_API_BASE_URL` | server ไม่เริ่ม |

ต้องมี table test ของ `core/currency_rate` (คิด `thb_rate` · ตรวจ response · เดือนไทยของ `at` · ปัด) · integration test ของ job (ใช้ fake askmelotto server — `httptest`) · cronjobs · คำสั่งมือ · `ConvertToTHBService`

## 10. จุดที่ผู้ร่างเสนอเพิ่ม — รอ lead ยืนยันตอน review

| # | เรื่อง | ข้อเสนอ |
|---|---|---|
| 1 | **รายการ 27 สกุลที่เดียว** (review รอบ 1 "ใช้ที่เดียว" ไม่ได้ระบุว่าที่ไหน) | ใช้ `agentManagementCore.Currencies` (แหล่งเดียวที่ MGMT-10 กำหนด) · `core/currency_rate` ลบรายการของตัวเอง แล้วอ้างตัวนั้น — ไม่สร้างแพ็กเกจกลางใหม่ |
| 2 | code ของ `ErrCurrencyRateUnavailable` | `409401` (module 09 · ชนิด 4 = ไม่พบ) HTTP 200 |
| 3 | ปัด `thb_rate` 8 ตำแหน่ง | ปัดครึ่งขึ้น (ตรงกับกฎปัด 4 ตำแหน่งของยอด) |
| 4 | `data.date` ที่ตรวจ (J4-3) | เทียบกับ `YYYY-MM-01` ที่ขอเท่านั้น (ไม่รับวันอื่น แม้ askmelotto จะตอบ 200) |
| 5 | `synced_at` ของแถวที่ไม่ถูกเขียนใหม่ (CR-05) | คงค่าเดิม — แก้บั๊กที่รอบเดิมอัปเดตแม้ไม่มีสกุลใหม่ (หัวข้อ 4 ข้อ 7) |
| 6 | `CHECK (rate_month = date_trunc('month', rate_month)::date)` | เพิ่มเพื่อกันแถวที่ไม่ใช่วันที่ 1 (เกินจากร่างของ lead) |
| 7 | `amount` ติดลบ / NaN / Inf ใน `ConvertToTHBService` (CR-27) | error ภายใน — bet ไม่ควรส่งยอดติดลบ (ยอดชนะ ≥ 0) |
| 8 | pod ที่ตั้ง `IS_CRONJOB=true` บน dev | deployment เดียวของ dev ตั้งเป็น `true` ก่อนเปิดใช้ · ระหว่างรื้อโค้ด (โค้ดเดิมบน dev ยังเปิด worker เมื่อ `IS_CURRENCY_RATE_SYNC` เป็นค่าเริ่มต้น `true`) ให้ตั้ง `false` ก่อน deploy ตามที่ lead สั่ง |

## 11. Contract changes (แจ้ง frontend / infra)

- ไม่มี endpoint ใหม่ · หน้าที่แสดงยอดแปลง (Dashboard / รายงาน) ไม่อยู่ใน scope แล้ว
- **infra:** env เปลี่ยนชื่อ — `IS_CURRENCY_RATE_SYNC` → `IS_CRONJOB` (เปิดเฉพาะ deployment เดียว) · `CURRENCY_RATE_API_BASE_URL` → `ASKMELOTTO_API_BASE_URL` (uat / prod บังคับ) · pod ที่รัน job ต้องออกไปเรียก askmelotto ได้ (HTTPS ออก)
- **spec Bet:** ใช้ `ConvertToTHBService` · กำหนดว่า `at` = เวลาแทงหรือเวลาจ่าย · เก็บ `THBRate` + `RateMonth` + ยอดบาทไว้กับรายการ (C3) · ไม่มีเรท = ปฏิเสธการแทง (C1)
- **ทีม:** แก้ CLAUDE.md (โครงสร้าง `app/cronjob` + กฎ "cronjob เรียกได้แค่ service") และ `docs/ARCHITECTURE.md` ใน PR เดียวกับโค้ด — เป็นข้อตกลงของทีม ต้องผ่าน review (CLAUDE.md Meta rule 2)

## 12. Checklist ก่อนเปิด PR (ลำดับตามที่ lead กำหนด)

1. lead review spec นี้ → ปิดหัวข้อ 10 → เปลี่ยนสถานะเป็น APPROVED
2. รื้อโค้ด: `app/cronjob` (+ `cronjobs` + metrics) · `platform/askmelotto` · ตาราง `currency_rates` แทน Redis · `ConvertToTHBService` · คำสั่งมือ `scripts/currency_rate_sync` · ลบ const ค้างใน `agent_auth/keys.go` · รายการ 27 สกุลที่เดียว · error module 09
3. แก้ CLAUDE.md · ARCHITECTURE.md · DEPLOYMENT.md · `.env.example`
4. migration `currency_rates` + `cronjobs` (up / down / up) · `make check` + integration test ผ่าน (แนบผล)
5. PR + integration ผ่าน → **แจ้ง lead ก่อน merge**
