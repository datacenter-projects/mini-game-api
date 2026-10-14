# ฝาก / ถอนเงินให้ลูกตรง (`deposit_withdraw`) — Spec

- สถานะ: **APPROVED** — lead (zerph) 2026-10-13 · [docs/reviews/deposit-withdraw.md](../reviews/deposit-withdraw.md) (DQ1–DQ13 · แก้ spec S1–S15)
- การตัดสินของ lead (สรุป): ลูกไม่ ACTIVE ฝากไม่ได้แต่ถอนได้ (DQ1) · Redis lock ที่ลูก TTL 10 วินาที + `request_id` (DQ2) ·
  upsert แถวยอดก่อน lock (DQ3) · module เดิมพันหักเงินตอนแทง (DQ4) · reason ของ Superadmin แยก (DQ5) · เพดาน 1,000,000,000 (DQ6) ·
  `parent_unlimited` แทน `null` (DQ7) · `credit` → `amount` (DQ8) · error module `04` (DQ9) · `request_id` เช็คข้ามเส้นทั้ง 2 ทาง (DQ10) ·
  DW1 มี `keyword` (DQ11) · `note` + DW3 `logs/list` ทำรอบนี้ (DQ12) · path / สิทธิ์ตาม spec (DQ13)
- **lead ยืนยันเพิ่ม 2026-10-14** (ข้อที่ maofoy กำหนดตอนแก้ spec): DW3 ≤ 31 วัน + เห็นเฉพาะรายการที่ตัวเองเป็นแม่ (DW-46 / 47) · DW1 `keyword` เฉพาะลูกตรง (DW-41) ·
  `create_requests` เป็นทะเบียน `request_id` กลาง (DW-26) · Redis ล่ม `504801` · tx timeout 8 วินาที (DW-23) · ลำดับ lock ใน ARCHITECTURE.md ใช้ทุก module รวม bet / settle (DW-51)
- ผู้ดูแล: maofoy · PR review: boiledegg
- ชื่อ module ในโค้ด: `deposit_withdraw` (`controllers/deposit_withdraw` · `dto/deposit_withdraw` · `service/deposit_withdraw` ·
  `repository/postgres/deposit_withdraw` · `repository/redis/deposit_withdraw`)
- rule ID: `DW-xx`
- โค้ดเก่าที่อ้างอิง: ไม่มี — ไม่ได้ extract จาก `askmelotto_minigame-api`

> เอกสารนี้คือ source of truth ของ module — โค้ดและ test ต้องตรงกับเอกสารนี้ ถ้าไม่ตรงให้แก้โค้ด
> หรือแก้เอกสารผ่านการอนุมัติใหม่ ห้ามอ้างเอกสารของโปรเจกต์เก่า

## 1. หน้าที่ของ module

ย้ายเครดิตภายในสายกับ**ลูกตรงชั้นเดียว** (agent หรือ Member): บัญชีหลังบ้าน (Superadmin · Company · Share · Agent และ sub ของบัญชีเหล่านี้)
ฝาก / ถอนเงินให้ลูกตรงของตัวเอง · ดูรายชื่อลูกตรงพร้อมยอด · ดูประวัติรายการที่ตัวเองทำ · **สาย Transfer เท่านั้น** ·
Superadmin เป็นแหล่งเงินเดียวของระบบ (วงเงินไม่จำกัด)

**ไม่อยู่ใน module นี้:** ยอดเงินตั้งต้นตอนสร้างบัญชี (agent_management MGMT-15A) · การหัก / คืนเงินจากการเล่นเกม (module เดิมพัน) ·
`payment_type` ของ Member (MGMT-63 — แสดงผลอย่างเดียว ไม่มีผลกับ module นี้ · lead P1)

## 2. คำศัพท์

| คำ | ความหมาย |
|---|---|
| แม่ | บัญชีที่ทำรายการ = `Actor.AgentID` (sub = เจ้าของ sub — AUTH-26) |
| ลูกตรง | agent ที่ `user_agents.parent_id = แม่` (ไม่รวม ADMIN) หรือ Member ที่ `user_members.agent_id = แม่` · ไม่รวมหลาน |
| ฝาก (`DEPOSIT`) | แม่ลด · ลูกเพิ่ม |
| ถอน (`WITHDRAW`) | ลูกลด · แม่เพิ่ม |
| ยอดของ agent | `agent_balances.amount` ต่อสกุล |
| ยอดของ Member | `user_members.credit` (Member มี 1 สกุล = `user_members.currency`) |
| สาย Seamless | Company หัวสายเป็น Seamless Reseller / Seamless Master / Seamless 1 to 1 — รวม Share Reseller / Share Master และทุกบัญชีใต้สายนั้น (`agentManagementCore.IsSeamless`) |

## 3. Business rules

### ใครทำได้ / ทำกับใคร

| ID | Rule |
|---|---|
| DW-01 | ทำได้เฉพาะ**ลูกตรง**ของแม่ · เช็คด้วย `user_agents.parent_id` / `user_members.agent_id` (ไม่ใช้ `cnf`) · หลาน · นอกสาย · ไม่มี id นี้ = `404401` (ข้อความเดียวกันทุกกรณี ไม่บอกว่ามี id อยู่ในระบบไหม) |
| DW-02 | ระบุบัญชีด้วย `role` + `user_id`: `MEMBER` → `user_members` · `COMPANY` / `SHAREHOLDER` / `AGENT` → `user_agents` และ role ต้องตรงกับในตาราง · ไม่ตรง = `404401` · ค่าอื่น = `422` |
| DW-03 | ADMIN ทำรายการไม่ได้ทั้ง 3 เส้น (`404305`) |
| DW-04 | sub ทำได้ตามสิทธิ์เมนู `payment`: DW1 / DW3 = view · DW2 = edit (`402303` — เช็คที่ route กฎข้อ 28) · sub ของ Superadmin ทำได้เหมือน Superadmin |
| DW-05 | แม่ที่ถูกระงับทำไม่ได้ (`PassedGates()` — AUTH-54) · **ลูกที่ `SUSPENDED` / `LOCKED`: ฝากไม่ได้ (`404301`) · ถอนได้** (ล็อกบัญชีที่โกงแล้วต้องดึงเงินคืนได้ — DQ1) |
| DW-06 | **สาย Seamless ทั้งหมดใช้ไม่ได้** (ไม่มียอดเงิน — MGMT-15) = `404302` · ตัดสินจาก Company หัวสายของลูก (`GetChainCompanyTypeRepository` + `IsSeamless`) |
| DW-07 | DW2 ต้องยืนยัน passcode ของตัวเอง (`RequirePasscode()` — `passcode` ใน body) |

### จำนวนเงินและสกุล

| ID | Rule | ตัวอย่าง |
|---|---|---|
| DW-10 | `amount` มากกว่า 0 · **ไม่เกิน 1,000,000,000 ต่อรายการ** (DQ6) · ทศนิยมไม่เกิน 4 ตำแหน่ง · ปัด `utils.Round4` ก่อนบันทึกและหลังบวก / ลบ (กฎข้อ 9) · เทียบยอด / เพดานด้วยค่าที่ปัดแล้ว | `0` / `-1` / `1.00001` / `1000000001` = `422` · `1000000000` = ผ่าน |
| DW-11 | `currency` บังคับ · ต้องเป็นสกุลของลูก (agent = `agent_currencies` · Member = `user_members.currency`) **และ**ของแม่ (Superadmin ไม่ต้องเช็คฝั่งแม่) · ไม่ตรง = `404303` | Share B2B (THB, USD) ฝาก JPY ให้ Agent THB = `404303` |
| DW-12 | ฝาก: ยอดแม่ในสกุลนั้นต้อง ≥ `amount` (`404304`) · **Superadmin วงเงินไม่จำกัด** ไม่มีแถวยอดให้หัก | Company 50,000 ฝาก 60,000 = `404304` |
| DW-13 | ถอน: ยอดลูกต้อง ≥ `amount` (`404306`) · ถอนได้หมดเท่าที่มีในกระเป๋าตอนนั้น (ไม่กระทบรอบเกมค้าง — DW-50) | Member 100 ถอน 100 = สำเร็จ เหลือ 0 |
| DW-14 | ถอนจากลูกของ Superadmin: ลูกลดอย่างเดียว ไม่มียอดฝั่ง Superadmin ให้เพิ่ม | |
| DW-15 | `note` ไม่บังคับ · ≤ 200 ตัวอักษร (นับ rune) · ตัดช่องว่างหัวท้าย · ไม่ส่ง / `""` = `""` · `null` = `422` (ACC-32) | |

### กันกดซ้ำ / ยิงพร้อมกัน (DQ2 · DQ10)

| ID | Rule |
|---|---|
| DW-20 | **Redis lock ที่ลูก** ก่อนเปิด tx: key `dw:lock:{owner_type}:{owner_id}` (`owner_type` = `AGENT` / `MEMBER` จาก `role` · `owner_id` = `user_id`) · `SET key token NX PX 10000` · `token` = ค่าสุ่มต่อ request · ใช้ Redis instance เดียวกับ session (`platform` client เดิม) · repository อยู่ `app/repository/redis/deposit_withdraw` |
| DW-21 | ได้ lock ไม่ได้ = ตอบ**ทันที** `404307` "มีรายการของบัญชีนี้กำลังทำอยู่ กรุณาลองใหม่" ไม่รอ · Redis ใช้ไม่ได้ (error / timeout) = `504801` ไม่ทำรายการ (fail closed) |
| DW-22 | ปลด lock หลัง tx จบ (commit หรือ rollback) ด้วย **compare-and-delete** (Lua: ลบเฉพาะเมื่อค่าตรงกับ `token` ของตัวเอง — ไม่ลบ lock ของคนอื่นที่ได้ไปหลัง TTL หมด) · ปลดไม่สำเร็จ = log warn แล้วปล่อยให้ TTL หมดเอง ไม่ทำให้ request ล้ม |
| DW-23 | tx ต้องจบเร็วกว่า TTL: service ใช้ `context.WithTimeout` **8 วินาที** ครอบ tx (เกิน = rollback · `500`) · ถ้า lock หมดอายุก่อน tx จบ ชั้นสุดท้ายยังเป็น DB lock (DW-30) |
| DW-24 | **เช็ค `request_id` เป็นอย่างแรกหลังได้ lock** (ก่อนเช็คลูก / สถานะ / ยอด): ใน tx `pg_advisory_xact_lock(hashtext('create_request:' + request_id))` (key เดียวกับ `FindReplay` ของ agents / members create) แล้วอ่าน `create_requests` · ไม่พบ = ทำรายการต่อ |
| DW-25 | พบ `request_id` แล้ว: `target_type = 'DW'` + ผู้เรียกเดิม (`creator_type` + `creator_id` = `actor_type` + `actor_id`) + ค่าเดิมทุก field (`role` · `user_id` · `currency` · `amount` · `transaction_type` · `note`) = **คืนผลเดิม** (`200` · `data` จาก log เดิม) ไม่ย้ายเงินซ้ำ — แม้ลูกถูกระงับหลังรายการแรก · อื่นๆ (บัญชีอื่น · ค่าไม่ตรง · ใช้สร้างบัญชีไปแล้ว) = `422` "request_id นี้ถูกใช้แล้ว" |
| DW-26 | **เช็คข้ามเส้นทั้ง 2 ทาง (DQ10)** โดยใช้ตาราง `create_requests` ร่วมกัน: DW2 insert แถว `target_type = 'DW'` · `target_id = deposit_withdraw_logs.id` ใน tx เดียวกัน → `agents/create` / `members/create` เจอ `request_id` นี้ผ่าน `FindReplay` เดิม (`target_type` ไม่ตรง) = `422` **โดยไม่ต้องแก้โค้ดของ agent_management** · DW2 เจอแถว `AGENT` / `MEMBER` = `422` ตาม DW-25 |
| DW-27 | ชั้นสุดท้ายใน DB: `create_requests.request_id` (PK) และ `deposit_withdraw_logs.request_id` (UNIQUE) · insert ชนกัน = rollback แล้วอ่านใหม่ตาม DW-25 |

### Transaction และลำดับ lock (DQ3 · DQ5)

| ID | Rule |
|---|---|
| DW-30 | ทุกอย่างใน **transaction เดียว** ที่ service เปิด (กฎข้อ 10–11) ตามลำดับ: (1) advisory lock + เช็ค `request_id` (DW-24) (2) หาลูก + เช็ค DW-01 – DW-06 / DW-11 (3) upsert แถวยอดที่ขาด (4) lock ยอด (5) เช็คยอด → อัปเดต (6) `deposit_withdraw_logs` 1 แถว (7) `balance_ledger` (8) `create_requests` 1 แถว |
| DW-31 | **upsert ก่อน lock:** แถว `agent_balances (agent_id, currency)` ของทุกฝ่ายที่เป็น agent (ไม่รวม Superadmin) ที่ยังไม่มี → `INSERT … ON CONFLICT DO NOTHING` ยอด 0 · เรียง `agent_id` น้อย→มาก |
| DW-32 | **ลำดับ lock ตายตัว** (กฎกลาง — [ARCHITECTURE.md](../ARCHITECTURE.md) หัวข้อ "ลำดับ lock ของยอดเงิน"): `agent_balances` `FOR UPDATE` เรียง `agent_id` น้อย→มาก **ก่อน** → `user_members` `FOR NO KEY UPDATE` (ถ้าลูกเป็น Member) |
| DW-33 | `balance_ledger` (กฎข้อ 11) — `ref_type = 'DW'` · `ref_id = deposit_withdraw_logs.id` · `request_id` เดียวกับคำขอ · `actor_*` = คนกดจริง: |
| | • ฝาก: แม่ `DEPOSIT_OUT` (−) + ลูก `DEPOSIT_IN` (+) · แม่เป็น Superadmin = ลูก `DEPOSIT_FROM_SUPERADMIN` (+) แถวเดียว |
| | • ถอน: ลูก `WITHDRAW_OUT` (−) + แม่ `WITHDRAW_IN` (+) · แม่เป็น Superadmin = ลูก `WITHDRAW_TO_SUPERADMIN` (−) แถวเดียว |
| DW-34 | **CHECK ยอดติดลบ** (ชั้นกันพลาด — ปกติเช็คยอดในข้อ DW-30 (5) ก่อนแล้ว): repository แยกด้วย `errors.As(err, &*pgconn.PgError)` · `Code = "23514"` · `ConstraintName` = `agent_balances_amount_check` / `user_members_credit_check` → คืน sentinel `ErrBalanceNegative` ของ package repository · service แปลงตามทิศ: ฝาก = `404304` · ถอน = `404306` (รายการเดียวมีฝั่งที่ยอดลดฝั่งเดียว) · **ห้ามเทียบข้อความ error** (กฎข้อ 16) · ชื่อ constraint ยืนยันด้วย integration test (อ่าน `pg_constraint`) |

### รายชื่อลูกตรง (DW1)

| ID | Rule |
|---|---|
| DW-40 | เฉพาะลูกตรงของแม่ (agent ไม่รวม ADMIN + Member) · `UNION ALL` 2 ตาราง แบบเดียวกับ `downlines/list` · เรียง username A→Z · `page` / `limit` ใน body (`utils.NewPage` · ค่าเริ่มต้น 20 สูงสุด 100) |
| DW-41 | `keyword` (DQ11): ไม่ส่ง / `""` = ทั้งหมด · ส่ง = 4–32 ตัว (`422`) ค้น username บางส่วน ไม่สนตัวพิมพ์ **เฉพาะลูกตรง** (ไม่ลงหลาน — ต่างจาก E2) · escape `%` `_` `\` |
| DW-42 | แต่ละแถว: `id` · `role` · `username` · `name` · `status` · `has_balance` · `balances` · `role` ส่งกลับในเส้น DW2 ได้ทันที (DW-02) |
| DW-43 | `balances`: agent = 1 รายการต่อสกุลใน `agent_currencies` (ไม่มีแถวยอด = `0`) · Member = 1 รายการ (`currency` + `credit`) · ยอดฝั่ง agent ดึงด้วย `agent_id IN (...)` ครั้งเดียวต่อหน้า (กฎข้อ 23 ห้าม N+1) |
| DW-44 | `has_balance = false` = สาย Seamless (DW-06) · `status` = สถานะของลูกเอง (แม่ผ่าน `PassedGates()` แล้ว สายด้านบนจึงไม่ถูกระงับ) · หน้าบ้าน: ปุ่มฝากเปิดเมื่อ `has_balance` และ `ACTIVE` · **ปุ่มถอนเปิดเมื่อ `has_balance` (ลูกไม่ ACTIVE ก็ถอนได้ — DW-05)** |
| DW-45 | API ส่งยอดตามที่เก็บ ไม่ปัดเพิ่ม (กฎข้อ 9) |

### ประวัติ (DW3)

| ID | Rule |
|---|---|
| DW-46 | เห็นเฉพาะรายการที่ `parent_id = แม่` (รายการที่บัญชีนี้หรือ sub ของบัญชีนี้ทำ — กฎข้อ 22) · ลูกไม่เห็นรายการที่แม่ทำกับตัวเองในเส้นนี้ (ลูกดูได้ที่ Report / statement — module อื่น ทำทีหลัง · lead 2026-10-14) |
| DW-47 | `date_from` / `date_to` บังคับ รูปแบบ `YYYY-MM-DD` เวลาไทย (+07:00) รวมทั้ง 2 วัน · `date_from` ≤ `date_to` · ช่วงไม่เกิน 31 วัน (`422`) |
| DW-48 | ตัวกรองไม่บังคับ: `transaction_type` (`DEPOSIT` / `WITHDRAW` · `""` = ทั้งหมด) · `role` + `user_id` (ลูกคนเดียว · ส่งคู่กัน · `""` / `0` = ทุกคน) · เรียง `created_at` ใหม่→เก่า แล้ว `id` ใหม่→เก่า · `page` / `limit` ใน body |
| DW-49 | index `idx_deposit_withdraw_logs_parent (parent_id, created_at DESC)` รองรับ query หลัก · กรองลูกใช้ index เดียวกัน + เงื่อนไข `target_*` |

### ข้อบังคับของ module เดิมพัน (DQ4 — open item)

| ID | Rule |
|---|---|
| DW-50 | **module เดิมพันต้องหักเงินจาก `credit` ทันทีตอนแทง** (ข้อบังคับของ Bet) → ถอนไม่กระทบรอบเกมค้าง |
| DW-51 | module เดิมพันที่ lock `agent_balances` + `user_members` พร้อมกันต้องใช้ลำดับ DW-32 (ARCHITECTURE.md) |

## 4. สิ่งที่พบในโค้ดเก่า และการตัดสินใจ

ไม่ได้ extract จากโค้ดเก่า (repo เก่าไม่มีในเครื่อง) — การตัดสินใจทั้งหมดอยู่ใน [docs/reviews/deposit-withdraw.md](../reviews/deposit-withdraw.md)

| # | เรื่อง | การตัดสินใจ |
|---|---|---|
| 1 | ระบุลูกด้วย `user_id` อย่างเดียว | ไม่พอ — id ของ `user_agents` กับ `user_members` ซ้ำกันได้ · เพิ่ม `role` (DW-02) |
| 2 | เช็คลูกตรงด้วย `cnf` หรือ `parent_id` | `parent_id` / `agent_id` (มี index `idx_user_agents_parent_username` · `idx_user_members_agent_username` อยู่แล้ว) |
| 3 | ใช้ `downlines/list` (E2) แทน DW1 | แยกเส้น: สิทธิ์ `payment` view (E2 = `member` view) · field `has_balance` · `keyword` ค้นเฉพาะลูกตรง |
| 4 | log ของ module เอง vs `balance_ledger` อย่างเดียว | ทำทั้งคู่: `deposit_withdraw_logs` = 1 แถวต่อการกด · `balance_ledger` = รายการต่อยอด (กฎข้อ 11) |
| 5 | เช็ค `request_id` ข้ามเส้น (DQ10) | ใช้ `create_requests` เป็นทะเบียน `request_id` กลาง (DW-26) — agents / members create ปฏิเสธได้ด้วยโค้ดเดิม ไม่ต้องแก้ `FindReplay` |
| 6 | Redis lock (DQ2) | ยังไม่มี lock helper ในระบบ — เขียนใหม่ใน `repository/redis/deposit_withdraw` (`SET NX PX` + Lua compare-and-delete) |
| 7 | Redis ล่ม | `504801` (HTTP 503) — lead ไม่ได้ระบุ code · maofoy เลือก `c = 8` (storage) |
| 8 | DW3 ช่วงวัน | lead ไม่ได้ระบุเพดาน · maofoy กำหนด ≤ 31 วัน เพื่อจำกัดขนาด query |

## 5. Endpoints

ใต้ `/bo/pr` ผ่าน `Authenticated` + `PassedGates()` · error ร่วม: `401202`, `401203`, `401301`, `401302`, `401304`, `401306`, `401307`, `401311`, `401312`, `402303`, `404305`

| Method | Path | สิทธิ์ (route) | หน้าที่ |
|---|---|---|---|
| POST | `/api/v1/bo/pr/deposit-withdraw/downlines/list` | `RequirePermission(payment, view)` | DW1 รายชื่อลูกตรง + ยอด |
| POST | `/api/v1/bo/pr/deposit-withdraw/create` | `RequirePermission(payment, edit)` · `RequirePasscode()` | DW2 ฝาก / ถอน |
| POST | `/api/v1/bo/pr/deposit-withdraw/logs/list` | `RequirePermission(payment, view)` | DW3 ประวัติ |

ตัวอย่างใช้ข้อมูลชุดเดียวกับ agent_management: `comp01` (id 10, Company Transfer) → `share01` (id 12, Share B2C · THB) → `agent01` (id 15) / `mem01` (Member id 501) · sub ของ share01 = `share01@staff` (id 31)

### DW1 · POST /api/v1/bo/pr/deposit-withdraw/downlines/list

Request (ไม่บังคับ):
```json
{ "keyword": "", "page": 1, "limit": 20 }
```
Response `data` (share01 login):
```json
{
  "current_page": 1, "total_page": 1, "total_count": 2, "limit": 20, "has_next": false, "has_prev": false,
  "data": [
    { "id": 15,  "role": "AGENT",  "username": "agent01", "name": "agent01", "status": "ACTIVE",
      "has_balance": true, "balances": [ { "currency": "THB", "amount": 5000 } ] },
    { "id": 501, "role": "MEMBER", "username": "mem01",   "name": "ใจดี",    "status": "LOCKED",
      "has_balance": true, "balances": [ { "currency": "THB", "amount": 100 } ] }
  ]
}
```
- `mem01` ถูกล็อก: ปุ่มฝากปิด · ปุ่มถอนเปิด (DW-44)

Error codes: `422`

### DW2 · POST /api/v1/bo/pr/deposit-withdraw/create

Request:
```json
{
  "request_id": "6f1c2a9e-8b1d-4c1e-9a55-2f0d6b7c1a10",
  "role": "MEMBER",
  "user_id": 501,
  "currency": "THB",
  "amount": 100,
  "transaction_type": "DEPOSIT",
  "note": "เติมเครดิตรอบแรก",
  "passcode": "123456"
}
```

| Field | บังคับ | Rule |
|---|---|---|
| `request_id` | ✓ | UUID · ใหม่ทุกครั้งที่กด · กดซ้ำเพราะเน็ตหลุด = ใช้ค่าเดิม (DW-24 – DW-27) |
| `role` | ✓ | `COMPANY` / `SHAREHOLDER` / `AGENT` / `MEMBER` (DW-02) |
| `user_id` | ✓ | > 0 |
| `currency` | ✓ | สกุลของลูกและของแม่ (DW-11) |
| `amount` | ✓ | > 0 · ≤ 1,000,000,000 · ทศนิยม ≤ 4 ตำแหน่ง · JSON number เท่านั้น (ข้อความ = `400`) (DW-10) |
| `transaction_type` | ✓ | `DEPOSIT` / `WITHDRAW` (ตัวพิมพ์ใหญ่เท่านั้น) |
| `note` | | ≤ 200 ตัว · ไม่ส่ง / `""` = ไม่มี · `null` = `422` (DW-15) |
| `passcode` | ✓ | passcode ของคนกด (DW-07) |

Response `data` (share01 ฝาก 100 ให้ mem01):
```json
{
  "id": 9001,
  "request_id": "6f1c2a9e-8b1d-4c1e-9a55-2f0d6b7c1a10",
  "transaction_type": "DEPOSIT",
  "role": "MEMBER",
  "user_id": 501,
  "username": "mem01",
  "currency": "THB",
  "amount": 100,
  "note": "เติมเครดิตรอบแรก",
  "parent_unlimited": false,
  "parent_balance_after": 4900,
  "target_balance_after": 200,
  "created_at": "2026-10-14T10:00:00+07:00"
}
```
- `parent_unlimited = true` เมื่อแม่เป็น Superadmin — `parent_balance_after` = `0` (ไม่มีความหมาย ไม่ต้องแสดง · ACC-32 ไม่มี `null`)
- ส่งซ้ำด้วย `request_id` เดิม + ค่าเดิม = `data` เดิม (DW-25)

Error codes: `422`, `404301`, `404302`, `404303`, `404304`, `404306`, `404307`, `404401`, `504801` + error ของ passcode (`401204`, `401205`, `401304`)

**ลำดับการเช็ค** (ตามลำดับ middleware ใน route — แก้ตอน implement 2026-10-14: ADMIN ไม่มีเมนูใดเลย ถ้าเช็คสิทธิ์ก่อนจะได้ `402303` แทน `404305`):
ADMIN (`404305` — `dwMw.RejectAdmin`) → สิทธิ์ `payment` (`402303`) → passcode (`422` / `401204` …) → รูปแบบ body (`400` / `422`) → Redis lock (`404307` / `504801`) → `request_id` (ผลเดิม / `422`) →
หาลูก (`404401`) → Seamless (`404302`) → สถานะลูก เฉพาะฝาก (`404301`) → สกุล (`404303`) → lock + ยอด (`404304` / `404306`)

### DW3 · POST /api/v1/bo/pr/deposit-withdraw/logs/list

Request:
```json
{
  "date_from": "2026-10-01",
  "date_to": "2026-10-14",
  "transaction_type": "",
  "role": "MEMBER",
  "user_id": 501,
  "page": 1,
  "limit": 20
}
```

| Field | บังคับ | Rule |
|---|---|---|
| `date_from` / `date_to` | ✓ | `YYYY-MM-DD` เวลาไทย · รวมทั้ง 2 วัน · `from` ≤ `to` · ≤ 31 วัน (DW-47) |
| `transaction_type` | | `DEPOSIT` / `WITHDRAW` · ไม่ส่ง / `""` = ทั้งหมด |
| `role` + `user_id` | | ส่งคู่กัน (ส่งค่าเดียว = `422`) · ไม่ส่ง / `""` + `0` = ทุกลูก |
| `page` / `limit` | | ค่าเริ่มต้น 1 / 20 · สูงสุด 100 |

Response `data`:
```json
{
  "current_page": 1, "total_page": 1, "total_count": 2, "limit": 20, "has_next": false, "has_prev": false,
  "data": [
    {
      "id": 9002,
      "request_id": "0b7e9d1c-3a2f-4e8b-9c11-5d6e7f8a9b0c",
      "transaction_type": "WITHDRAW",
      "actor_username": "share01@staff",
      "role": "MEMBER",
      "user_id": 501,
      "username": "mem01",
      "currency": "THB",
      "amount": 50,
      "note": "",
      "parent_unlimited": false,
      "parent_balance_before": 4900,
      "parent_balance_after": 4950,
      "target_balance_before": 200,
      "target_balance_after": 150,
      "created_at": "2026-10-14T11:00:00+07:00"
    },
    {
      "id": 9001,
      "request_id": "6f1c2a9e-8b1d-4c1e-9a55-2f0d6b7c1a10",
      "transaction_type": "DEPOSIT",
      "actor_username": "share01",
      "role": "MEMBER",
      "user_id": 501,
      "username": "mem01",
      "currency": "THB",
      "amount": 100,
      "note": "เติมเครดิตรอบแรก",
      "parent_unlimited": false,
      "parent_balance_before": 5000,
      "parent_balance_after": 4900,
      "target_balance_before": 100,
      "target_balance_after": 200,
      "created_at": "2026-10-14T10:00:00+07:00"
    }
  ]
}
```
- `role` ของแถว = `target_role` ที่เก็บตอนทำรายการ · `username` = snapshot ตอนทำรายการ

Error codes: `422`

## 6. Schema

migration ใหม่ (`make migration name=deposit_withdraw_logs`):

```sql
CREATE TABLE deposit_withdraw_logs (
    id                    BIGSERIAL        PRIMARY KEY,
    request_id            VARCHAR(64)      NOT NULL,
    transaction_type      VARCHAR(10)      NOT NULL,                        -- DEPOSIT / WITHDRAW
    actor_type            VARCHAR(10)      NOT NULL,                        -- AGENT / SUB (คนกดจริง)
    actor_id              BIGINT           NOT NULL,                        -- user_agents.id หรือ subaccounts.id
    actor_username        VARCHAR(71)      NOT NULL,                        -- sub = share01@staff
    parent_id             BIGINT           NOT NULL REFERENCES user_agents(id), -- แม่ (sub → เจ้าของ)
    parent_username       VARCHAR(71)      NOT NULL,
    target_type           VARCHAR(10)      NOT NULL,                        -- AGENT / MEMBER
    target_id             BIGINT           NOT NULL,
    target_role           VARCHAR(20)      NOT NULL,                        -- COMPANY / SHAREHOLDER / AGENT / MEMBER (DW-02)
    target_username       VARCHAR(71)      NOT NULL,
    currency              VARCHAR(4)       NOT NULL,
    amount                DOUBLE PRECISION NOT NULL,
    parent_unlimited      BOOLEAN          NOT NULL DEFAULT false,          -- แม่เป็น Superadmin (DQ7)
    parent_balance_before DOUBLE PRECISION NOT NULL DEFAULT 0,              -- parent_unlimited = 0
    parent_balance_after  DOUBLE PRECISION NOT NULL DEFAULT 0,
    target_balance_before DOUBLE PRECISION NOT NULL,
    target_balance_after  DOUBLE PRECISION NOT NULL,
    note                  VARCHAR(200)     NOT NULL DEFAULT '',             -- DW-15
    ip                    VARCHAR(45)      NOT NULL DEFAULT '',
    created_at            TIMESTAMPTZ      NOT NULL DEFAULT now(),
    CONSTRAINT uq_deposit_withdraw_logs_request UNIQUE (request_id),        -- DW-27 · กฎข้อ 12
    CONSTRAINT ck_deposit_withdraw_logs_type   CHECK (transaction_type IN ('DEPOSIT', 'WITHDRAW')),
    CONSTRAINT ck_deposit_withdraw_logs_actor  CHECK (actor_type IN ('AGENT', 'SUB')),
    CONSTRAINT ck_deposit_withdraw_logs_target CHECK (target_type IN ('AGENT', 'MEMBER')),
    CONSTRAINT ck_deposit_withdraw_logs_amount CHECK (amount > 0 AND amount <= 1000000000)
);
CREATE INDEX idx_deposit_withdraw_logs_parent ON deposit_withdraw_logs(parent_id, created_at DESC);              -- DW3
CREATE INDEX idx_deposit_withdraw_logs_target ON deposit_withdraw_logs(target_type, target_id, created_at DESC); -- ประวัติของลูก (รายงานอนาคต)
```

- เก็บ username / role เป็น snapshot ในแถว → DW3 ไม่ต้อง JOIN
- `target_id` ไม่มี FK เพราะชี้ได้ 2 ตาราง (แยกด้วย `target_type`)
- ไม่เก็บ passcode หรือข้อมูลลับ
- ไม่แก้ตารางเดิม:
  - `balance_ledger.reason` (VARCHAR(30)) / `ref_type` (VARCHAR(10)) รองรับค่าใหม่อยู่แล้ว · เพิ่ม enum ใน `app/models`:
    `DEPOSIT_OUT` · `DEPOSIT_IN` · `DEPOSIT_FROM_SUPERADMIN` · `WITHDRAW_OUT` · `WITHDRAW_IN` · `WITHDRAW_TO_SUPERADMIN`
  - `create_requests.target_type` (VARCHAR(10) · ไม่มี CHECK) รองรับ `'DW'` อยู่แล้ว (DW-26)
- index ของ DW1 มีแล้ว: `idx_user_agents_parent_username` · `idx_user_members_agent_username`
- Redis: key `dw:lock:{AGENT|MEMBER}:{id}` · TTL 10 วินาที · ไม่ใช่ source of truth ของยอด (กฎข้อ 14)

## 7. Test cases

| Rule | Input | Expected |
|---|---|---|
| DW-01 | share01 ฝากให้ agent01 · ให้ Member ของ agent01 (หลาน) · id ที่ไม่มี · บัญชีสายอื่น | สำเร็จ · `404401` · `404401` · `404401` |
| DW-02 | `role: "MEMBER", user_id: 15` (id 15 เป็น agent) · `role: "COMPANY"` กับ agent01 · `role: "member"` | `404401` · `404401` · `422` |
| DW-03 | ADMIN เรียก DW1 / DW2 / DW3 | `404305` |
| DW-04 | sub มี `payment` view เรียก DW1 / DW3 · เรียก DW2 · sub มีแค่ `member` edit | สำเร็จ · `402303` · `402303` |
| DW-04 | sub ของ Superadmin (`payment` edit) ฝาก 1,000 ให้ Company | สำเร็จ · log `actor_type = SUB` · `parent_unlimited = true` · ledger `DEPOSIT_FROM_SUPERADMIN` |
| DW-05 | ฝากให้ลูก `SUSPENDED` · `LOCKED` | `404301` ทั้งคู่ · ยอดไม่เปลี่ยน |
| DW-05 | ถอน 100 จากลูก `LOCKED` ที่มี 300 | สำเร็จ · ลูก 200 · แม่ +100 |
| DW-06 | Superadmin ฝากให้ Company Seamless · Share Master ฝากให้ Agent · Company Seamless Reseller ฝากให้ Share | `404302` ทั้งหมด |
| DW-07 | ไม่ส่ง passcode · passcode ผิด | `422` · `401204` |
| DW-10 | `amount` `0` · `-5` · `1.00001` · `"100"` · `1000000001` · `1000000000` · `25.5` | `422` · `422` · `422` · `400` · `422` · สำเร็จ · สำเร็จ ยอด 25.5 |
| DW-10 | ยอด 0.1 ฝาก 0.2 | ยอด = 0.3 (ไม่ใช่ 0.30000000000000004) |
| DW-11 | Share B2B (THB, USD) ฝาก USD ให้ Agent THB · ฝาก JPY | `404303` · `404303` |
| DW-12 | Company THB 50,000 ฝาก 10,000 ให้ Share | Company 40,000 · Share +10,000 · ledger 2 แถว · log 1 แถว · `create_requests` 1 แถว (`DW`) |
| DW-12 | Company THB 50,000 ฝาก 60,000 | `404304` · ไม่มี log / ledger / `create_requests` |
| DW-12 | Superadmin ฝาก 1,000,000 ให้ Company | Company +1,000,000 · ledger 1 แถว `DEPOSIT_FROM_SUPERADMIN` · log `parent_unlimited = true` · `parent_balance_*` = 0 |
| DW-13 | Member 100 ถอน 100 · ถอน 100.0001 | สำเร็จ `credit` = 0 · `404306` |
| DW-14 | Superadmin ถอน 500 จาก Company | Company −500 · ledger 1 แถว `WITHDRAW_TO_SUPERADMIN` |
| DW-15 | `note` 200 ตัว · 201 ตัว · `null` · `"  ทดสอบ  "` | สำเร็จ · `422` · `422` · เก็บ `"ทดสอบ"` |
| DW-20 / 21 | ยิงพร้อมกัน 2 คำขอ (`request_id` ต่างกัน) ที่ลูกเดียวกัน (integration) | ได้ lock 1 คำขอ · อีกอัน `404307` ทันที · ย้ายเงินครั้งเดียว |
| DW-21 | ปิด Redis แล้วเรียก DW2 | `504801` · ยอดไม่เปลี่ยน |
| DW-22 | lock ของ request A หมดอายุ · request B ได้ lock · A ปลด lock | lock ของ B ยังอยู่ (compare-and-delete) |
| DW-25 | กดซ้ำหลังรายการแรกจบ (`request_id` เดิม ค่าเดิม) | ผลเดิม · ไม่ย้ายเงินซ้ำ |
| DW-25 | รายการแรกสำเร็จ → ลูกถูกระงับ → retry `request_id` เดิม | ผลเดิม (ไม่ใช่ `404301`) |
| DW-25 | `request_id` เดิม `amount` ต่าง · บัญชีอื่นใช้ `request_id` นี้ | `422` · `422` |
| DW-26 | `request_id` ที่ใช้สร้างบัญชีแล้วส่งมา DW2 · `request_id` ที่ DW2 ใช้แล้วส่งไป `agents/create` / `members/create` | `422` · `422` |
| DW-27 | ยิงพร้อมกัน 2 คำขอ `request_id` เดียวกันแต่ลูกต่างกัน (lock คนละ key · integration) | ย้ายเงินครั้งเดียว · อีกอัน `422` |
| DW-31 | ฝากครั้งแรกให้ agent ที่ไม่มีแถวยอด THB | สร้างแถว THB · ยอด = ค่าที่ฝาก · ledger ถูก |
| DW-31 | ถอนจาก agent เข้าแม่ที่ไม่มีแถวยอด USD | สร้างแถว USD ให้แม่ · ยอด = ค่าที่ถอน |
| DW-32 | share01 ฝากให้ agent01 พร้อมกับ comp01 ฝากให้ share01 (lock แถว share01 ร่วมกัน · integration) | ไม่ deadlock · ยอดสุดท้ายถูกต้อง |
| DW-33 | sub `share01@staff` ฝากให้ agent01 | ledger `actor_type = SUB` · log `actor_username = share01@staff` · `parent_id = 12` |
| DW-34 | integration: ชื่อ constraint `agent_balances_amount_check` / `user_members_credit_check` มีใน `pg_constraint` · repository UPDATE ยอดติดลบตรงๆ | มีครบ · ได้ `ErrBalanceNegative` |
| DW-40 / 41 | share01 มีลูก agent 1 + Member 1 + ADMIN ในระบบ · `keyword` `"mem0"` · `"ab"` · keyword ของหลาน | 2 แถว A→Z ไม่มี ADMIN · `mem01` · `422` · ไม่พบ |
| DW-43 | agent ที่มี 2 สกุลแต่มีแถวยอดแค่ THB | `balances` 2 รายการ USD = 0 |
| DW-44 | Company Seamless ใน list ของ Superadmin | `has_balance = false` |
| DW-46 | share01 ดู DW3 · comp01 ดู DW3 | เห็นเฉพาะรายการที่ `parent_id = 12` · comp01 ไม่เห็นรายการของ share01 |
| DW-47 | `date_from` > `date_to` · ช่วง 32 วัน · `"2026/10/01"` | `422` |
| DW-48 | ส่ง `role` ไม่ส่ง `user_id` · `transaction_type: "WITHDRAW"` | `422` · เฉพาะรายการถอน |

## 8. Contract changes (สำหรับแจ้ง frontend)

ไม่มี breaking change — เส้นใหม่ทั้ง 3 เส้น · เอกสารหน้าบ้าน `docs/frontend/deposit_withdraw.md` เขียน**หลังโค้ดเสร็จและผ่าน lead review** (S15)

- หน้าฝากถอนใช้ DW1 · ส่ง `role` + `id` + สกุลจากแถวไปที่ DW2 ตรงๆ
- DW2 สร้าง `request_id` ใหม่ทุกครั้งที่กด (กดซ้ำเพราะเน็ตหลุด = ใช้ค่าเดิม) · ส่ง `passcode`
- ปุ่มฝาก: `has_balance` และ `status = ACTIVE` · ปุ่มถอน: `has_balance`
- `404307` = มีรายการของบัญชีนี้กำลังทำอยู่ ให้ลองใหม่อีกครั้ง (ไม่ต้องสร้าง `request_id` ใหม่)
