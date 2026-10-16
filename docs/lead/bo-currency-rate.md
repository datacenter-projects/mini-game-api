# สถานะ — `boiledegg/bo/currency-rate` · spec APPROVED · โค้ดตรวจรอบ 2 ผ่าน (lead 2026-10-16) · แก้ 3 ข้อก่อน merge แล้ว

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

## 5. ผล test (แนบตามที่ lead ขอ · ตรวจรอบ 2 — 2026-10-16)

รันบน test env ในเครื่อง (Postgres localhost:55432 / miniredis · DB สร้างใหม่ migrate ครบทุกไฟล์รวม `20261016120000`) · branch นี้ merge `origin/dev` แล้ว (รวม migration game_rate `20261016110000`)

- แก้ตามรอบ 2: **S1** `-force` ต้องใส่ `-yes` ทุกครั้งไม่ว่า `APP_ENV` (`TestRunForceAlwaysNeedsYes` ลอง local / dev / prod / ไม่ตั้ง) · **RateMonth** ที่ `ConvertToTHBService` คืน = เที่ยงคืน UTC ของวันที่ 1 เหมือน `rate_month` ใน repo / model (`TestConvertRateMonthMatchesStoredRow` เทียบกับแถวที่เก็บ + ลองเวลา 30 ก.ย. 17:30 UTC)
- **เลข migration เปลี่ยน** `20261016100000` → **`20261016120000`** (ใหม่กว่าของ game_rate `20261016110000` ที่เข้า dev แล้ว — กัน goose ฟ้อง missing migration ถ้า RDS dev รันของ game_rate ไปก่อน) · ยังไม่เคยรันบน RDS

```
# go test (unit) — 2026-10-16 12:51 +0700 · branch boiledegg/bo/currency-rate
ok  	app/app/core/account	0.728s
ok  	app/app/core/agent_auth	0.747s
ok  	app/app/core/agent_management	0.813s
ok  	app/app/core/asset	0.806s
ok  	app/app/core/currency_rate	0.687s
ok  	app/app/core/deposit_withdraw	0.646s
ok  	app/app/core/game	0.409s
ok  	app/app/core/game_rate	0.593s
ok  	app/app/core/member_management	0.665s
ok  	app/app/core/player_auth	0.913s
ok  	app/app/core/pt_calculation	0.364s
ok  	app/app/externals/transfer/dto/player_auth	0.944s
ok  	app/app/internals/backoffice/dto/account	0.821s
ok  	app/app/internals/backoffice/dto/admin_management	0.790s
ok  	app/app/internals/backoffice/dto/agent_auth	0.801s
ok  	app/app/internals/backoffice/dto/asset	0.848s
ok  	app/app/internals/frontend/dto/player_auth	0.815s
ok  	app/app/repository/redis/agent_auth	1.435s
ok  	app/app/repository/redis/currency_rate	1.161s
ok  	app/app/repository/redis/player_auth	1.311s
ok  	app/pkg/apperr	0.732s
ok  	app/pkg/configs	0.707s
ok  	app/pkg/middleware	0.753s
ok  	app/pkg/response	0.679s
ok  	app/pkg/routes	2.271s
ok  	app/pkg/utils	0.807s
ok  	app/platform/metrics	1.128s
ok  	app/platform/storage	1.734s

# go test -tags=integration -p 1 (test env: postgres localhost:55432/minigame_test · miniredis)
--- PASS: TestStartDisabledOnThisPod (1.14s)
--- PASS: TestStartRunsImmediatelyAndRecords (0.57s)
--- PASS: TestBootDoesNotOverwriteEnabled (0.98s)
--- PASS: TestDisabledJobSkipsSilently (0.88s)
--- PASS: TestMissingRowCountsAsDisabled (0.85s)
--- PASS: TestFailedRunRecordsError (0.84s)
--- PASS: TestIncompleteRunIsFailed (0.85s)
--- PASS: TestCompleteMonthIsSuccessWithoutCall (0.84s)
--- PASS: TestLockHeldIsSkipped (0.86s)
--- PASS: TestLastErrorTruncated (0.85s)
--- PASS: TestRunOnceUnknownJob (0.84s)
--- PASS: TestStartStopsOnContextCancel (1.18s)
PASS
ok  	app/app/cronjob	12.588s
--- PASS: TestSyncRequestsFirstDayAndStoresAll (0.56s)
--- PASS: TestSyncMonthIsBangkokMonth (0.52s)
--- PASS: TestSyncCompleteMonthDoesNotCall (0.82s)
--- PASS: TestSyncNeverOverwritesAndFillsMissing (0.86s)
--- PASS: TestSyncNoTHBSavesNothing (0.84s)
--- PASS: TestSyncResponseProblemsSaveNothing (0.87s)
--- PASS: TestSyncRejectsBadValues (0.89s)
--- PASS: TestSyncForceOverwritesWholeMonth (0.87s)
--- PASS: TestSyncSpecificMonth (0.84s)
--- PASS: TestSyncLockHeldSkips (0.91s)
--- PASS: TestSyncReleasesLock (0.84s)
--- PASS: TestConvertToTHB (0.86s)
--- PASS: TestConvertTHBSkipsDB (0.86s)
--- PASS: TestConvertUsesCallerTx (0.84s)
--- PASS: TestConvertUnavailable (0.85s)
--- PASS: TestConvertInvalidAmountIsInternalError (0.83s)
--- PASS: TestConvertRoundsOnceHalfUp (0.83s)
--- PASS: TestSyncLogLevelByTime (0.86s)
--- PASS: TestSyncMissingCurrencyLogsWarnWithNames (0.86s)
--- PASS: TestConvertRateMonthMatchesStoredRow (0.83s)
PASS
ok  	app/app/service/currency_rate	18.489s
--- PASS: TestRunRejectsBadMonth (0.57s)
--- PASS: TestRunFillsOnlyMissing (0.63s)
--- PASS: TestRunForceOverwritesWholeMonth (0.61s)
--- PASS: TestRunForceAlwaysNeedsYes (0.91s)
--- PASS: TestRunDefaultsToCurrentThaiMonth (0.88s)
```
