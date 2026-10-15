# Lead review: อัตราแลกเปลี่ยน (`currency_rate`) — zerph · 2026-10-16

- โค้ด: merge เข้า dev แล้ว `72f48d5` (Boiledegg · ยังไม่ผ่าน review ก่อน merge) · spec `docs/modules/currency_rate.md` (เขียน APPROVED 2026-10-07)
- ผลตรวจโค้ด (hub verify): ใช้แสดงผลได้ แต่**ยังไม่ปลอดภัยสำหรับใช้กับเงิน** — must-fix 3 · should-fix 6 · nit 6
- การใช้งานเปลี่ยน: Dashboard ถูกตัด → **ผู้ใช้หลักคือ module เดิมพัน** แปลงยอดชนะของ Member สกุลอื่นเป็นบาทเพื่อเช็ค `win_limit_per_day` (หน่วยบาทเสมอ — member-payment รอบ 2.1) → spec ต้องออกรอบใหม่ (business logic ของ job รีวิวต่อในรอบ 2)

## รอบ 1 — โครงสร้าง + การใช้กับเงิน

### ใช้กับเงิน (เดิม spec เขียน "แสดงผลเท่านั้น")
| # | เรื่อง | ผล |
|---|---|---|
| C1 | ไม่มีเรท / เรทเก่าเกินกำหนด ตอน Member สกุลอื่นแทง | **ปฏิเสธการแทง (fail closed)** · Member THB ไม่ต้องแปลง ไม่โดน |
| C2 | อายุเรทสูงสุด | **24 ชั่วโมง** · เกิน = error (`ErrRateStale`) |
| C3 | เก็บเรทไว้กับรายการเดิมพัน | **เก็บ** เรทที่ใช้ + ยอดชนะที่แปลงเป็นบาทแล้ว (ออกแบบใน spec bet) |
| M1 | ฟังก์ชันแปลงใช้ int64 หน่วย 1/100 ปัด 2 ตำแหน่ง | **float64 + ปัด 4 ตำแหน่ง** ตามกฎข้อ 9 (คำนวณด้วย `big.Rat`) |

### โครงสร้าง cronjob (lead อนุมัติการเปลี่ยนโครงสร้าง)
| # | เรื่อง | ผล |
|---|---|---|
| J1 | งานตามเวลาอยู่ที่ไหน | **`app/cronjob/`** — `cronjob.go` (`Start(ctx)` จุดเดียวที่ `application.go` เรียก) + `{job}_job.go` (ตารางเวลา + Redis lock → เรียก service) · ไม่มี business logic · เรียกได้แค่ service · รันใน process เดียวกับ API |
| J2 | pod ไหนรัน | **flag `IS_CRONJOB` (ค่าเริ่มต้น `false`)** เปิดเฉพาะ deployment เดียว (แบบ askmelotto-api) · ยังคง Redis lock กันซ้ำ · แทน `IS_CURRENCY_RATE_SYNC` |
| J3 | ตัวยิง HTTP ไป askme | **client ใน `platform/askme`** · service เรียกผ่าน client · test ใช้ fake server |
| T1 | ความถี่ของ job | **อยู่ในโค้ด** (ไม่เก็บในตาราง) |
| T2 | เปิด / ปิด job | **ตาราง `cronjobs`**: `code` (PK) · `enabled` · `last_started_at` · `last_finished_at` · `last_status` (`SUCCESS` / `FAILED` / `SKIPPED`) · `last_error` (ตัดความยาว) · `updated_by` · `updated_at` · ทุกรอบอ่าน `enabled` ก่อน (ปิด = ข้ามรอบ) · ไม่มีแถว = สร้างตอน boot ตามค่าเริ่มต้นในโค้ด · **ตอนนี้เปิด / ปิดด้วย SQL (ขอ lead บน env ร่วม)** · เส้น BO ให้ Superadmin ทำทีหลัง |
| T3 | ประวัติทุกรอบ | **ไม่เก็บ** (`cronjob_runs` ยังไม่ทำ) · ดูรอบล่าสุดในตาราง · รอบเก่าดูจาก log / metrics |

แก้ CLAUDE.md (โครงสร้าง `app/cronjob` + กฎ cronjob เรียกได้แค่ service) · ARCHITECTURE.md · `.env.example` ใน PR เดียวกัน

### ควรแก้ (จากผลตรวจโค้ด)
- ลบ const ของ currency_rate ที่ค้างใน `app/repository/redis/agent_auth/keys.go` (จาก merge `1b763f6`)
- `synced_at` อัปเดตแม้ไม่มีสกุลไหนได้เรทใหม่ → แก้คู่กับ C2
- รายการ 27 สกุลซ้ำกับ agent_management → ใช้ที่เดียว
- รายละเอียดอื่น: `reviews/mini-game-api/verify-currency-rate.md` (lead ส่งให้)

## รอบ 2 — business logic + spec ใหม่ (lead zerph 2026-10-16) · **แทน spec เดิมทั้งหมด**
อ้างอิงระบบแม่ (askmelotto-api): ดึงเรทจากต้นทางวันละครั้งตี 1 (cronjob 10701) เก็บรายวันใน `rate_currency_usdts` · **เฉพาะวันที่ 1 ของเดือน** แปลงเป็น "1 หน่วย = ? บาท" ลง `currencies.rate` แล้วใช้ทั้งเดือน · รายงานทุกตัวใช้เรทของ `ปี-เดือน-01` (`controller_set_usdt.go:125` · `controller_winlose_v2.go:179`)

### 1. หน้าที่ / ขอบเขต
- เก็บ**เรทรายเดือน** (เรทของวันที่ 1 ของเดือน · เวลาไทย) ของ 27 สกุล — ตรงกับระบบแม่ (R1)
- ผู้ใช้: module เดิมพัน แปลงยอดของ Member สกุลอื่นเป็นบาทเพื่อเช็ค `win_limit_per_day` (บาทเสมอ)
- ไม่อยู่ใน module: แลก / ย้ายเงินข้ามสกุลจริง · การเก็บเรทไว้กับรายการเดิมพัน (C3 → spec bet)

### 2. แหล่งเรท
`POST {ASKMELOTTO_API_BASE_URL}/api/ex/rate/usdt` body `{"date":"YYYY-MM-01"}` — **ขอด้วยวันที่ 1 ของเดือนเสมอ** (S2-1) · `json_rate` = 1 USDT = ? หน่วย · `430503` = askmelotto ยังไม่มีเรทวันนั้น · ยิงผ่าน client `platform/askmelotto` (J3 — ทุกเส้นของ askmelotto อยู่ที่นี่)

### 3. ตารางเก็บเรท (R2 — แทน Redis ทั้งหมด)
```sql
CREATE TABLE currency_rates (
    rate_month  DATE             NOT NULL,   -- วันที่ 1 ของเดือน
    currency    VARCHAR(10)      NOT NULL,   -- 27 สกุล
    usdt_rate   DOUBLE PRECISION NOT NULL,   -- 1 USDT = ? หน่วยของสกุลนี้ (ค่าดิบ ทศนิยม ≤ 8)
    thb_rate    DOUBLE PRECISION NOT NULL,   -- 1 หน่วยของสกุลนี้ = ? บาท (คิดครั้งเดียวตอน sync)
    synced_at   TIMESTAMPTZ      NOT NULL,
    PRIMARY KEY (rate_month, currency),
    CHECK (usdt_rate > 0 AND thb_rate > 0)
);
```
- `thb_rate = usdt_rate(THB) ÷ usdt_rate(สกุลนั้น)` คำนวณด้วย `big.Rat` **เก็บ 8 ตำแหน่ง** · THB = `1` · เช่น USD 33.62 ÷ 0.99 = `33.95959596`
- เก็บทุกเดือนเป็นประวัติ · อ่านจาก DB ตรง (ไม่มี cache)
- S3-2: ได้ไม่ครบ → บันทึกเฉพาะสกุลที่ได้ · สกุลที่ขาด = ไม่มีเรทของเดือนนี้ (ไม่ใช้เรทเดือนก่อน) · **ไม่มี THB = ไม่บันทึกทั้งเดือน**
- S3-3: เรทของเดือนที่บันทึกแล้ว**ไม่เขียนทับอัตโนมัติ** · แก้ได้ด้วยคำสั่งมือ `-force` (ขอ lead)

### 4. job `currency_rate_sync` (`app/cronjob/currency_rate_job.go`)
- **ทุก 15 นาที + รันทันทีตอน boot** (J4-1) · เฉพาะ pod `IS_CRONJOB=true` · ต้อง `enabled` ในตาราง `cronjobs`
- **คง Redis lock** `SET NX PX 300000` + compare-and-delete (J4-2) · ได้ lock ไม่ได้ = `SKIPPED`
- flow: เดือนปัจจุบัน (เวลาไทย) ครบ 27 สกุลแล้ว = จบ ไม่ยิง (SUCCESS) → ยิงด้วยวันที่ 1 · timeout 10 วิ → ตรวจ → `INSERT … ON CONFLICT DO NOTHING` เฉพาะสกุลที่ยังไม่มี → ยังไม่ครบ = FAILED (บอกสกุลที่ขาด) · ครบ = SUCCESS
- ตรวจ response (J4-3): HTTP 200 + `code = 200` + `base_currency = "USDT"` + **`data.date` = วันที่ที่ขอ** + แต่ละสกุลเป็นตัวเลข > 0 ทศนิยม ≤ 8 และอยู่ใน 27 สกุล · สกุลที่ไม่ผ่านไม่บันทึก (log บอกสกุล)
- log (J4-4): ก่อนวันที่ 1 เวลา 03:00 = info · หลังจากนั้นยังไม่ครบ = warn ทุกรอบ
- คำสั่งมือ (J4-5): `scripts/currency_rate_sync -month 2026-10 [-force]` ใช้ service เดียวกับ job · ไม่ใส่ `-force` = เติมเฉพาะที่ขาด · `-force` = เขียนทับทั้งเดือน · env ร่วมต้องขอ lead

### 5. ฟังก์ชันแปลง (ให้ bet ใช้)
```go
// app/service/currency_rate
func ConvertToTHBService(ctx context.Context, db *gorm.DB, amount float64, currency string, at time.Time) (THBConversion, error)

type THBConversion struct {
    AmountTHB float64   // Round4(amount × THBRate) — ปัดครั้งเดียวตอนท้าย ครึ่งขึ้น
    RateMonth time.Time // วันที่ 1 ของเดือนที่ใช้
    THBRate   float64   // 1 หน่วย = ? บาท ของเดือนนั้น
}
```
- คืนเรทที่ใช้ด้วย (S5-1 — bet เก็บตาม C3) · เดือน = **เดือนของ `at` (เวลาเกิดรายการ) ตามเวลาไทย** (S5-2 — spec bet กำหนดว่าเวลาแทงหรือจ่าย)
- มีแค่แปลงเป็นบาท (S5-3) · THB → THB คืนทันทีไม่อ่าน DB (`THBRate = 1`)
- ไม่มีเรทของเดือนนั้น → **`ErrCurrencyRateUnavailable`** (error module ใหม่ `09`) + log warn (สกุล / เดือน) · bet แปลงเป็น code ของตัวเอง และปฏิเสธการแทง (C1)
- อ่าน 1 แถวด้วย PK · ใช้ `db` / tx ของผู้เรียก · ไม่ lock · ไม่มี endpoint
- ตัวอย่าง: 100 USD × 33.95959596 = **3,395.9596 บาท**
- **C2 (แทน "เรทอายุ 24 ชั่วโมง" ของรอบ 1): ต้องมีเรทของเดือนนั้น** · ไม่มี = ปฏิเสธ (ไม่ใช้เรทเดือนก่อน)

### 6. ตาราง `cronjobs` (ใช้กับทุก job)
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
- `app/cronjob/cronjob.go` ห่อทุก job: boot = `INSERT (code, enabled = ค่าเริ่มต้นในโค้ด) ON CONFLICT DO NOTHING` (ไม่ทับค่าที่ตั้งแล้ว) · แต่ละรอบ: ปิด = **ข้ามเงียบ ไม่แก้แถว** (C6-2) · lock ไม่ได้ = SKIPPED · รัน = บันทึก `last_*`
- ค่าเริ่มต้นกำหนดต่อ job ในโค้ด (C6-1): `currency_rate_sync` = เปิด · job เสี่ยงในอนาคต = ปิด
- เปิด / ปิดด้วย SQL สำเร็จรูปใน DEPLOYMENT.md (C6-3) · env ร่วมขอ lead · `updated_by = 'sql:{ชื่อ}'`
  `UPDATE cronjobs SET enabled = false, updated_by = 'sql:zerph', updated_at = now() WHERE code = 'currency_rate_sync';`
- metrics (C6-4): `cronjob_runs_total{job,status}` · `cronjob_last_success_timestamp{job}` (ตั้ง alert ได้)

### 7. Config / migration / error / test
| env | ค่าเริ่มต้น | กฎ |
|---|---|---|
| `IS_CRONJOB` | `false` | `true` = pod นี้รัน cronjob ทุกตัว · เปิด deployment เดียว · **แทน `IS_CURRENCY_RATE_SYNC` (ลบ)** |
| `ASKMELOTTO_API_BASE_URL` | local / dev = `https://dev-api.amblotto.net` | origin เปล่า · uat / prod บังคับ (ไม่ตั้ง = boot ไม่ขึ้น) · **แทน `CURRENCY_RATE_API_BASE_URL` (ลบ)** · ใช้กับทุกเส้นของ askmelotto (ต้องยืนยันตัวเมื่อไร เพิ่ม `ASKMELOTTO_API_KEY`) |

- migration ใหม่: `currency_rates` + `cronjobs` · ไม่ย้ายข้อมูลจาก Redis (job ดึงเดือนนี้ใหม่ตอน boot) · key Redis เดิม (`currency_rate:usdt`, `:meta`) ปล่อยไว้
- error module `09` = currency_rate ลง `ERROR_CODES.md` (`ErrCurrencyRateUnavailable`)
- test:
  - core: `thb_rate` (USD = 33.95959596 · THB = 1 · ปัด 8) · parse ไม่รับ 0 / ติดลบ / string / ทศนิยม 9
  - job: เดือนครบ = ไม่ยิง · `430503` = FAILED + info ก่อน 03:00 วันที่ 1 / warn หลัง · `data.date` ไม่ตรง = ไม่บันทึก · ขาดบางสกุล = บันทึกที่ได้ + FAILED · ไม่มี THB = ไม่บันทึก · ไม่ทับของเดิม · lock ไม่ได้ = SKIPPED · ปิด job = ไม่ยิง ไม่แก้แถว
  - cronjobs: boot ไม่ทับ `enabled` · `last_*` ครบ · metrics
  - คำสั่งมือ: เติมเฉพาะที่ขาด · `-force` ทับทั้งเดือน
  - แปลง: THB ไม่อ่าน DB · เดือนตาม `at` เวลาไทย (31 ต.ค. 23:59 / 1 พ.ย. 00:01) · ไม่มีเรท = error · 100 USD = 3,395.9596
  - config: `IS_CRONJOB=false` ไม่เริ่ม · URL ผิดรูป / prod ไม่ตั้ง = boot ไม่ขึ้น
- ก่อน deploy dev: migrate · `IS_CRONJOB=true` deployment เดียว · pod ออกไปเรียก askmelotto dev ได้

### งานของ Boiledegg
1. เขียน `docs/modules/currency_rate.md` ใหม่ตามรอบ 1 + รอบ 2 (ตัดส่วน Redis / ×10^8 / "แสดงผลเท่านั้น" / อ้างกฎข้อ 9 แบบเก่า) → lead review
2. รื้อโค้ดตาม: `app/cronjob` (+ `cronjobs` + metrics) · `platform/askmelotto` · ตาราง `currency_rates` แทน Redis · `ConvertToTHBService` · คำสั่งมือ · ลบ const ค้างใน `agent_auth/keys.go` · รายการ 27 สกุลที่เดียว
3. แก้ CLAUDE.md (โครงสร้าง `app/cronjob` + กฎ cronjob เรียกได้แค่ service) · ARCHITECTURE.md · DEPLOYMENT.md · `.env.example`
4. PR + integration test ผ่าน → แจ้ง lead ก่อน merge
