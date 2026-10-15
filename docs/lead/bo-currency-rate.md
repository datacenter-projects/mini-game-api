# สถานะ — `boiledegg/bo/currency-rate` · spec APPROVED 2026-10-16 · รื้อโค้ดเสร็จ รอแจ้ง lead ก่อน merge

## 1. ภาพรวม

| | |
|---|---|
| branch | `boiledegg/bo/currency-rate` (ตรงกับ `dev` @ `6e9cd0a` + commit spec ใหม่) |
| spec | `docs/modules/currency_rate.md` — **APPROVED 2026-10-16 (zerph)** (แทนฉบับ APPROVED 2026-10-07 ทั้งหมด · ตาม [review](../reviews/currency-rate.md) รอบ 1–2) |
| โค้ด | โค้ดเดิม merge เข้า `dev` แล้ว (`72f48d5` ก่อน review) — **ใช้กับเงินไม่ได้** · รื้อตาม spec ใหม่แล้ว (ยังไม่ merge) |
| deploy | **ยังไม่ deploy** — ถ้าจะ deploy dev ก่อนรื้อเสร็จ ต้องตั้ง `IS_CURRENCY_RATE_SYNC=false` (ค่าเริ่มต้น `true` ยิง askmelotto ทุก 15 นาทีตอน boot) |

## 2. งาน

| ส่วน | สถานะ |
|---|---|
| 1. เขียน `docs/modules/currency_rate.md` ใหม่ตามรอบ 1 + 2 | ✅ lead อนุมัติ 2026-10-16 |
| 2. รื้อโค้ด: `app/cronjob` (+ `cronjobs` + metrics) · `platform/askmelotto` · ตาราง `currency_rates` แทน Redis · `ConvertToTHBService` · คำสั่งมือ · ลบ const ค้างใน `agent_auth/keys.go` · รายการ 27 สกุลที่เดียว | ✅ เสร็จ · test ผ่านในเครื่อง |
| 3. แก้ CLAUDE.md (`app/cronjob` + กฎ cronjob เรียกได้แค่ service) · ARCHITECTURE.md · DEPLOYMENT.md · `.env.example` | ✅ เสร็จ |
| 4. PR + integration ผ่าน → แจ้ง lead ก่อน merge | ⏸ |

สิ่งที่ spec ฉบับใหม่ทำ: เรทรายเดือน (เรทวันที่ 1 · เก็บ `usdt_rate` + `thb_rate` 8 ตำแหน่ง ในตาราง) · ขอ askmelotto ด้วยวันที่ 1 เสมอ · job ทุก 15 นาที + ตอน boot เฉพาะ pod `IS_CRONJOB=true` + ตาราง `cronjobs` เปิด/ปิด + Redis lock + metrics · คำสั่งมือ `-force` · `ConvertToTHBService` → `{AmountTHB Round4, RateMonth, THBRate}` · ไม่มีเรท = `ErrCurrencyRateUnavailable` (module 09) · ตัดส่วน Redis / ×10^8 / "แสดงผลเท่านั้น" ของเดิม

## 3. ข้อเสนอที่ lead รับแล้ว

spec หัวข้อ 10 — lead รับทั้ง 8 ข้อ (2026-10-16):

| # | เรื่อง | ข้อเสนอ |
|---|---|---|
| 1 | รายการ 27 สกุลที่เดียว | ใช้ `agentManagementCore.Currencies` · ลบของ currency_rate |
| 2 | code ของ `ErrCurrencyRateUnavailable` | `409401` (module 09 · ชนิด 4 ไม่พบ) HTTP 200 |
| 3 | ปัด `thb_rate` 8 ตำแหน่ง | ปัดครึ่งขึ้น |
| 4 | `data.date` ใน response | ต้องตรง `YYYY-MM-01` ที่ขอเท่านั้น |
| 5 | `synced_at` ของแถวที่ไม่ถูกเขียนใหม่ | คงค่าเดิม (แก้บั๊กของโค้ดเดิม) |
| 6 | CHECK กันแถวที่ `rate_month` ไม่ใช่วันที่ 1 | เพิ่ม (เกินจากร่างของ lead) |
| 7 | `amount` ติดลบ / NaN / Inf | error ภายใน ไม่ใช่ `ErrCurrencyRateUnavailable` |
| 8 | `IS_CRONJOB=true` บน dev | deployment เดียวของ dev ตั้งเป็น true หลังรื้อเสร็จ |

## 4. ประวัติอัปเดต

| วันที่ | commit | อัปเดต |
|---|---|---|
| 2026-10-07 | `3abaf04` | ร่าง spec ฉบับแรก (ยิงเส้น askme จริงเพื่อดู response) |
| 2026-10-07 | `574a188` | spec อนุมัติ · โค้ดครบตาม spec + test |
| 2026-10-15 | `72f48d5` | merge เข้า dev (ก่อน lead review) |
| 2026-10-16 | — | lead review รอบ 1–2: ใช้กับเงินไม่ได้ · ออกแบบใหม่ทั้งหมด · แทน spec เดิม |
| 2026-10-16 | `ed3d38c` | เขียน spec ใหม่ + จอง error module 09 |
| 2026-10-16 | lead | อนุมัติ spec · รับทั้ง 8 ข้อ |
| 2026-10-16 | (commit นี้) | รื้อโค้ดตาม spec + แก้ CLAUDE.md / ARCHITECTURE.md / DEPLOYMENT.md / ERROR_CODES.md |
