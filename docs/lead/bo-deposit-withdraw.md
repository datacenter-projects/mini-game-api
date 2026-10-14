# Deposit / Withdraw (ฝาก / ถอนเงินให้ลูกตรง) — ทำตาม lead review (DQ1–DQ13 · S1–S15) · 2026-10-14

จาก maofoy · review: [docs/reviews/deposit-withdraw.md](../reviews/deposit-withdraw.md) · spec: [docs/modules/deposit_withdraw.md](../modules/deposit_withdraw.md) (APPROVED 2026-10-13 · ยืนยันเพิ่ม 6 ข้อ 2026-10-14)
branch `maofoy/bo/deposit-withdraw` · โค้ด `c59fcdc` · merge `dev` เข้า branch `330b6c0` · **boiledegg merge เข้า `dev` แล้ว** (`3709cd3` · แก้ lint `3a80c7b`) ·
deploy tag `dev-202610140819` · migration `20261014150000_deposit_withdraw_logs` รันบน dev แล้ว (maofoy แจ้ง 14 ต.ค.)

**lead อนุมัติโค้ดแล้ว (แจ้งผ่าน maofoy 2026-10-14)** · เอกสารหน้าบ้าน [docs/frontend/deposit_withdraw.md](../frontend/deposit_withdraw.md) (S15) ทำแล้ว

## 1. ผลแต่ละข้อ

| ข้อ | ผล | ที่อยู่ในโค้ด |
|---|---|---|
| DQ1 | ลูก `SUSPENDED` / `LOCKED`: ฝาก `404301` · ถอนได้ · Member เช็คสถานะซ้ำหลัง lock แถว | `core/deposit_withdraw.CanDeposit` · `service/deposit_withdraw/create_service.go` |
| DQ2 | Redis lock `dw:lock:{AGENT\|MEMBER}:{id}` · `SET NX PX 10000` + token สุ่ม (`crypto/rand`) · ปลดด้วย Lua compare-and-delete · ได้ lock ไม่ได้ = `404307` ทันที · Redis ใช้ไม่ได้ = `504801` (HTTP 503) · หลังได้ lock เช็ค `request_id` ก่อนอย่างอื่น · tx timeout 8 วินาที | `repository/redis/deposit_withdraw` · `CreateDepositWithdrawService` |
| DQ3 | upsert แถว `agent_balances` ที่ขาด (`ON CONFLICT DO NOTHING` เรียง `agent_id`) → `FOR UPDATE` เรียง `agent_id` → `user_members FOR NO KEY UPDATE` · กฎกลางย้ายไป ARCHITECTURE.md หัวข้อ "ลำดับ lock ของยอดเงิน" | `repository/postgres/deposit_withdraw` |
| DQ4 | เขียนเป็นข้อบังคับของ module เดิมพันใน spec (DW-50 / DW-51) | spec |
| DQ5 | ledger: `DEPOSIT_OUT` / `DEPOSIT_IN` · `WITHDRAW_OUT` / `WITHDRAW_IN` · Superadmin = `DEPOSIT_FROM_SUPERADMIN` / `WITHDRAW_TO_SUPERADMIN` แถวเดียว · `ref_type = 'DW'` · `ref_id` = id ของ log | `core.LedgerReasons` · `ledgerRows` |
| DQ6 | เพดาน 1,000,000,000 ต่อรายการ (`422`) + CHECK ใน DB | `dto/deposit_withdraw/create_dto.go` · migration |
| DQ7 | `parent_unlimited` แทน `null` · `parent_balance_*` = 0 เมื่อแม่เป็น Superadmin | response DW2 / DW3 |
| DQ8 | field `amount` | DTO |
| DQ9 | error module `04` จองใน ERROR_CODES.md · 9 code | `pkg/apperr/deposit_withdraw_errors.go` |
| DQ10 | `create_requests` เป็นทะเบียน `request_id` กลาง: DW2 insert แถว `target_type = 'DW'` · advisory lock key เดียวกับ `FindReplay` → `agents/create` / `members/create` ปฏิเสธด้วยโค้ดเดิม (**ไม่ได้แก้โค้ด agent_management**) | `findReplay` |
| DQ11 | DW1 `keyword` 4–32 ค้นเฉพาะลูกตรง | `ListDWDownlinesRepository` |
| DQ12 | `note` ≤ 200 ตัว (ตัดช่องว่าง · `null` = `422`) · DW3 `logs/list` ช่วงวันเวลาไทย ≤ 31 วัน · กรองประเภท / ลูก · เห็นเฉพาะรายการที่ตัวเองเป็นแม่ | `view_dto.go` · `ListLogsService` |
| DQ13 | path ตาม spec: `deposit-withdraw/downlines/list` · `/create` · `/logs/list` · สิทธิ์ `payment` view / edit · DW2 ต้องใส่ passcode | `routes.go` |

S1–S14 แก้ spec ครบ · S15 (เอกสารหน้าบ้าน) รอหลัง lead ตรวจโค้ด

## 2. ที่ต่างจาก spec / ตัดสินตอน implement

| # | เรื่อง | ผล |
|---|---|---|
| 1 | **ADMIN ได้ `402303` แทน `404305`** ถ้าเช็คสิทธิ์ก่อน (ADMIN ไม่มีเมนูใดเลย — `MenusForRole` = nil) | เพิ่ม middleware `dwMw.RejectAdmin()` ก่อน `RequirePermission` ในบรรทัด route (กฎข้อ 28) · แก้ลำดับการเช็คใน spec หัวข้อ DW2 ให้ตรงกับ middleware จริง |
| 2 | CHECK ยอดติดลบ (DW-34) | repository แยกด้วย `pgconn.PgError` (`23514` + ชื่อ constraint) → `ErrBalanceNegative` · service แปลงตามทิศ (ฝาก `404304` · ถอน `404306`) · `pgx` เป็น direct dependency ใน go.mod |
| 3 | สถานะของลูกฝั่ง agent อ่านแบบไม่ lock (ไม่มีกฎ lock `user_agents` ในลำดับ lock) | ถ้า agent ถูกระงับระหว่างเช็คกับ commit การฝากครั้งนั้นยังผ่าน — ช่องแคบมาก · Member เช็คซ้ำหลัง lock แล้ว |
| 4 | `created_at` ตัดเหลือ microsecond ตอนบันทึก | ส่ง `request_id` ซ้ำได้ `created_at` เดิมทุกตัวอักษร |
| 5 | regex UUID ของ `request_id` | ประกาศใน dto ของ module เอง (ตัวของ agent_management เป็น private · ไม่แก้ไฟล์ของ boiledegg) |

## 3. ผล test

- structure OK · build / vet ผ่าน · unit test ผ่านทุก package · `core/deposit_withdraw` table test (`OwnerOfRole` · `CanDeposit` · `Settle` · `LedgerReasons` · `SignedAmounts` · `DateRangeOK`)
- integration test `app/internals/backoffice/deposit_withdraw_integration_test.go` 10 ชุด: ฝาก / ถอน / ส่งซ้ำ · Superadmin + sub · กฎ DW-01 – DW-15 · แถวยอดครั้งแรก · `request_id` ข้ามเส้น · Redis lock / ยิงพร้อมกัน / ไม่ deadlock · Redis ล่ม · ชื่อ CHECK · DW1 · DW3
- **เครื่อง maofoy รัน integration ไม่ได้** (ไม่มี Docker / PostgreSQL) — compile ผ่าน `-tags=integration` · ผลจริงคือ ci ของ release tag `dev-202610140819` (ยังไม่ได้ยืนยันผลเองในหน้า Actions)

## 4. คำถามที่รอ lead

1. ตรวจโค้ดตามข้อ 1–2 · ผ่านแล้ว maofoy จะเขียน `docs/frontend/deposit_withdraw.md` (S15)
2. ข้อ 2.3 — ยอมรับช่องที่สถานะ agent ไม่ถูก lock ไหม · ถ้าไม่ ต้องเพิ่ม `user_agents` เข้าลำดับ lock ใน ARCHITECTURE.md
3. งานนี้ boiledegg merge เข้า `dev` ด้วย merge commit ในเครื่อง (ไม่ผ่านปุ่ม merge ของ PR) — ถือว่าผ่าน PR review ตามที่ตกลงไหม

## 5. ประวัติอัปเดต

- 2026-10-14 — สร้างไฟล์ · โค้ด DW1–DW3 merge เข้า dev + deploy `dev-202610140819` · migrate แล้ว · ขอ lead ตรวจโค้ด
- 2026-10-14 — lead อนุมัติโค้ด (แจ้งผ่าน maofoy) · เขียน `docs/frontend/deposit_withdraw.md` (S15) · คำถามข้อ 2–3 ในหัวข้อ 4 ยังไม่ได้คำตอบเป็นลายลักษณ์อักษร
- 2026-10-14 — **แก้ DW1 response (lead อนุมัติ)**: `currency` map 27 สกุล (key ตัวใหญ่) แทน `balances` · `last_login_ip` / `last_login_at` · `credit_detail` (ยอดของแม่) · `total_sum` (ยอดรวมลูกตรงทั้งหมดแยก role) · ยกเว้นกฎข้อ 19 — controller ประกอบ `DownlinesSummary` + `response.PageData` · แก้ `likeEscaper` ที่แปลง `\` ผิด (keyword ที่มี `\` ไม่ถูก escape) · ⚠️ map 27 สกุลทำให้หน้าบ้านแยกไม่ได้ว่าลูกมีสกุลไหน (ค่า 0 = ไม่มีสกุล หรือยอด 0) — ฝากสกุลที่ลูกไม่มีได้ `404303`
