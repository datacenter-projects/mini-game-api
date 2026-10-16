# ฝาก / ถอนเงินให้ลูกตรง (`deposit_withdraw`) — Spec

- สถานะ: **APPROVED** — lead (zerph) 2026-10-13 · [docs/reviews/deposit-withdraw.md](../reviews/deposit-withdraw.md) (DQ1–DQ13 · แก้ spec S1–S15)
- การตัดสินของ lead (สรุป): ลูกไม่ ACTIVE ฝากไม่ได้แต่ถอนได้ (DQ1) · Redis lock ที่ลูก TTL 10 วินาที + `request_id` (DQ2) ·
  upsert แถวยอดก่อน lock (DQ3) · module เดิมพันหักเงินตอนแทง (DQ4) · reason ของ Superadmin แยก (DQ5) · เพดาน 1,000,000,000 (DQ6) ·
  `parent_unlimited` แทน `null` (DQ7) · `credit` → `amount` (DQ8) · error module `04` (DQ9) · `request_id` เช็คข้ามเส้นทั้ง 2 ทาง (DQ10) ·
  DW1 มี `keyword` (DQ11) · `note` + DW3 `logs/list` ทำรอบนี้ (DQ12) · path / สิทธิ์ตาม spec (DQ13)
- **lead ยืนยันเพิ่ม 2026-10-14** (ข้อที่ maofoy กำหนดตอนแก้ spec): DW3 ≤ 31 วัน + เห็นเฉพาะรายการที่ตัวเองเป็นแม่ (DW-46 / 47) · DW1 `keyword` เฉพาะลูกตรง (DW-41) ·
  `create_requests` เป็นทะเบียน `request_id` กลาง (DW-26) · Redis ล่ม `504801` · tx timeout 8 วินาที (DW-23) · ลำดับ lock ใน ARCHITECTURE.md ใช้ทุก module รวม bet / settle (DW-51)
- **แก้ DW1 response 2026-10-14 (lead อนุมัติ)**: `currency` map 27 สกุล · `last_login_*` · `credit_detail` · `total_sum` แยก role (DW-42 – DW-44A · ยกเว้นกฎข้อ 19)
- **Superadmin มียอดจริง 2026-10-15 (lead อนุมัติ · seed เต็มใน migration)**: DW-12 / DW-14 / DW-33 / DW-44A · migration `20261015100000` · `parent_unlimited` = `false` เสมอสำหรับรายการใหม่ · MGMT-15A (Superadmin สร้าง Company Transfer พร้อมยอดตั้งต้น) หักยอด Superadmin ด้วย — ledger `INITIAL_TRANSFER_OUT` / `IN` (`agent_management/creator.go` · `field_errors.go`)
- **DW4 `history/list` (lead อนุมัติ 2026-10-15)**: ค้นประวัติที่ทำให้ `username` หนึ่ง (ตรงตัว) ใต้ตัวเองทุกชั้น · ผู้ทำรายการจริง · `starttime` / `endtime` ไม่จำกัดช่วง · `404402` (DW-55 – DW-60)
- **สาย Seamless มียอด 2026-10-15 (lead อนุมัติ)**: สร้างพร้อม `balance` ได้ (MGMT-15 / 15A) · ฝาก / ถอนได้ (DW-06) · DW1 `has_balance` = `true` เสมอ · `404302` เลิกใช้ · ยอดใช้แทงไม่ได้ (DW-52)
- **status ของรายการ 2026-10-16 (lead รับทราบ)**: `SUCCESS` / `FAILED` (ไม่มี pending — หัวข้อ 4 ข้อ 10) · บันทึก FAILED เมื่อล้มหลังหาลูกเจอ (DW-35 / DW-36) · DW3 ส่ง `status` + `ip` · DW4 ส่ง `status`
- ผู้ดูแล: maofoy · PR review: boiledegg
- ชื่อ module ในโค้ด: `deposit_withdraw` (`controllers/deposit_withdraw` · `dto/deposit_withdraw` · `service/deposit_withdraw` ·
  `repository/postgres/deposit_withdraw` · `repository/redis/deposit_withdraw`)
- rule ID: `DW-xx`
- โค้ดเก่าที่อ้างอิง: ไม่มี — ไม่ได้ extract จาก `askmelotto_minigame-api`

> เอกสารนี้คือ source of truth ของ module — โค้ดและ test ต้องตรงกับเอกสารนี้ ถ้าไม่ตรงให้แก้โค้ด
> หรือแก้เอกสารผ่านการอนุมัติใหม่ ห้ามอ้างเอกสารของโปรเจกต์เก่า

## 1. หน้าที่ของ module

ย้ายเครดิตภายในสายกับ**ลูกตรงชั้นเดียว** (agent หรือ Member): บัญชีหลังบ้าน (Superadmin · Company · Share · Agent และ sub ของบัญชีเหล่านี้)
ฝาก / ถอนเงินให้ลูกตรงของตัวเอง · ดูรายชื่อลูกตรงพร้อมยอด · ดูประวัติรายการที่ตัวเองทำ · **ทั้งสาย Transfer และ Seamless** (Seamless เพิ่ม 2026-10-15 · lead อนุมัติ — ยอดของสาย Seamless ใช้แทงไม่ได้) ·
Superadmin มียอดจริงใน `agent_balances` (seed สกุลละ 100,000,000 — lead 2026-10-15 · เดิม "วงเงินไม่จำกัด")

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
| DW-06 | **แก้ 2026-10-15 (lead อนุมัติ)**: สาย Seamless ทุกประเภท (Reseller / Master / 1 to 1) **ฝาก / ถอนได้** ตามกฎลูกตรงเดิม (DW-01) — คู่ที่ทำได้ตามกฎการสร้าง MGMT-02 (Superadmin → Company Seamless · Company Seamless Reseller / Master → Share · Company Seamless 1 to 1 → Member · Share Reseller / Master → Member) · **ยอดนี้ใช้แทงไม่ได้** (DW-52) · `404302` เลิกใช้ · เดิม: Seamless ใช้ไม่ได้ = `404302` |
| DW-07 | DW2 ต้องยืนยัน passcode ของตัวเอง (`RequirePasscode()` — `passcode` ใน body) |

### จำนวนเงินและสกุล

| ID | Rule | ตัวอย่าง |
|---|---|---|
| DW-10 | `amount` มากกว่า 0 · **ไม่เกิน 1,000,000,000 ต่อรายการ** (DQ6) · ทศนิยมไม่เกิน 4 ตำแหน่ง · ปัด `utils.Round4` ก่อนบันทึกและหลังบวก / ลบ (กฎข้อ 9) · เทียบยอด / เพดานด้วยค่าที่ปัดแล้ว | `0` / `-1` / `1.00001` / `1000000001` = `422` · `1000000000` = ผ่าน |
| DW-11 | `currency` บังคับ · ต้องเป็นสกุลของลูก (agent = `agent_currencies` · Member = `user_members.currency`) **และ**ของแม่ (Superadmin ไม่ต้องเช็คฝั่งแม่) · ไม่ตรง = `404303` | Share B2B (THB, USD) ฝาก JPY ให้ Agent THB = `404303` |
| DW-12 | ฝาก: ยอดแม่ในสกุลนั้นต้อง ≥ `amount` (`404304`) · **แม่ทุกประเภทรวม Superadmin หักจาก `agent_balances`** (แก้ 2026-10-15 · lead — เดิม Superadmin วงเงินไม่จำกัด) | Company 50,000 ฝาก 60,000 = `404304` |
| DW-13 | ถอน: ยอดลูกต้อง ≥ `amount` (`404306`) · ถอนได้หมดเท่าที่มีในกระเป๋าตอนนั้น (ไม่กระทบรอบเกมค้าง — DW-50) | Member 100 ถอน 100 = สำเร็จ เหลือ 0 |
| DW-14 | **Superadmin มียอดจริง** (lead 2026-10-15): migration `20261015100000` seed ทุกบัญชี `SUPERADMIN` ครบ 27 สกุลใน `agent_currencies` + `agent_balances` สกุลละ 100,000,000 (seed เต็ม ไม่หักยอดที่เคยปล่อยไปแล้ว) + ledger `SEED_SUPERADMIN` · script `create_superadmin` ทำแบบเดียวกัน · ถอนจากลูกของ Superadmin = ยอด Superadmin เพิ่ม | |
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
| | • ฝาก: แม่ `DEPOSIT_OUT` (−) + ลูก `DEPOSIT_IN` (+) |
| | • ถอน: ลูก `WITHDRAW_OUT` (−) + แม่ `WITHDRAW_IN` (+) · `DEPOSIT_FROM_SUPERADMIN` / `WITHDRAW_TO_SUPERADMIN` เลิกใช้ 2026-10-15 (มีเฉพาะแถวเก่า) |
| DW-34 | **CHECK ยอดติดลบ** (ชั้นกันพลาด — ปกติเช็คยอดในข้อ DW-30 (5) ก่อนแล้ว): repository แยกด้วย `errors.As(err, &*pgconn.PgError)` · `Code = "23514"` · `ConstraintName` = `agent_balances_amount_check` / `user_members_credit_check` → คืน sentinel `ErrBalanceNegative` ของ package repository · service แปลงตามทิศ: ฝาก = `404304` · ถอน = `404306` (รายการเดียวมีฝั่งที่ยอดลดฝั่งเดียว) · **ห้ามเทียบข้อความ error** (กฎข้อ 16) · ชื่อ constraint ยืนยันด้วย integration test (อ่าน `pg_constraint`) |
| DW-35 | **status ของรายการ** (maofoy 2026-10-16 · lead รับทราบ): `SUCCESS` / `FAILED` · ไม่มี pending (เหตุผล — หัวข้อ 4 ข้อ 10) · บันทึก `FAILED` เมื่อล้ม**หลังหาลูกเจอแล้ว** (แบบ B): ถูกปฏิเสธ `404301` / `404303` / `404304` / `404306` หรือระบบล้มระหว่างย้ายเงิน (5xx / timeout / error ที่ไม่ได้ประกาศ) · **ไม่บันทึก**: body ผิด · ไม่มีสิทธิ์ · passcode · `request_id` ซ้ำ · ติด lock `404307` / Redis `504801` · ไม่ใช่ลูกตรง `404401` |
| DW-36 | แถว `FAILED` เขียนใน tx ใหม่หลัง tx หลัก rollback (ไม่ใช้ ctx ของ tx · รอสูงสุด 3 วินาที · บันทึกไม่ได้ = log ไม่เปลี่ยน error ที่ตอบ) · **C-S1 (lead review รอบ 3)**: ใน tx ใหม่ทำ advisory lock key `create_request:{request_id}` (เดียวกับ tx หลัก — รอจน tx หลักจบจริง) แล้วเช็ค `create_requests` · มีแล้ว = COMMIT สำเร็จแม้ client ได้ error (timeout ตอน commit) → **ไม่เขียน FAILED** · ยอดก่อน = ยอดหลัง = ยอด ณ ตอนบันทึก · **ไม่มี ledger · ไม่ลง `create_requests`** (`request_id` เดิมกดซ้ำได้) · `note` = note ที่พิมพ์ + `" · ล้มเหลว: "` + ข้อความภาษาไทยของ error (ระบบล้ม = "เกิดข้อผิดพลาดภายในระบบ") · ยาวเกิน 200 ตัดส่วนที่พิมพ์ก่อน (`core.FailedNote`) · response ของ DW2 ยังเป็น error code เดิม |

### รายชื่อลูกตรง (DW1)

| ID | Rule |
|---|---|
| DW-40 | เฉพาะลูกตรงของแม่ (agent ไม่รวม ADMIN + Member) · `UNION ALL` 2 ตาราง แบบเดียวกับ `downlines/list` · เรียง username A→Z · `page` / `limit` ใน body (`utils.NewPage` · ค่าเริ่มต้น 20 สูงสุด 100) |
| DW-41 | `keyword` (DQ11): ไม่ส่ง / `""` = ทั้งหมด · ส่ง = 4–32 ตัว (`422`) ค้น username บางส่วน ไม่สนตัวพิมพ์ **เฉพาะลูกตรง** (ไม่ลงหลาน — ต่างจาก E2) · escape `%` `_` `\` |
| DW-42 | แต่ละแถว: `id` · `role` · `username` · `name` · `status` · `has_balance` · `currency` · `last_login_ip` · `last_login_at` (ยังไม่เคย login = `""`) · `role` ส่งกลับในเส้น DW2 ได้ทันที (DW-02) — **แก้ 2026-10-14 (lead อนุมัติ): `balances` (array) → `currency` (map) + last login** |
| DW-43 | `currency` = map **ครบ 27 สกุลที่ระบบรองรับ** (`agentManagementCore.Currencies`) · key ตัวใหญ่ · สกุลที่ไม่มียอด / บัญชีไม่มี = `0` · agent = `agent_balances` · Member = `credit` ที่สกุลของตัวเอง · ยอดของแม่ + ลูกฝั่ง agent ในหน้าดึงด้วย `agent_id IN (...)` ครั้งเดียว (กฎข้อ 23 ห้าม N+1) |
| DW-44 | `has_balance` = `true` ทุกแถว (สาย Seamless มียอดแล้ว — DW-06 แก้ 2026-10-15 · คง field ไว้ไม่ให้ contract แตก · เดิม Seamless = `false` + ยอด 0) · `status` = สถานะของลูกเอง (แม่ผ่าน `PassedGates()` แล้ว สายด้านบนจึงไม่ถูกระงับ) · หน้าบ้าน: ปุ่มฝากเปิดเมื่อ `has_balance` และ `ACTIVE` · **ปุ่มถอนเปิดเมื่อ `has_balance` (ลูกไม่ ACTIVE ก็ถอนได้ — DW-05)** |
| DW-44A | **นอก PageData (lead อนุมัติยกเว้นกฎข้อ 19 · 2026-10-14)**: `credit_detail.currency` = ยอดของคนที่ login (แม่ · sub = เจ้าของ) map 27 สกุล (ซ้ำกับ profile `balances` — ACC-19 · สาย Seamless = `0` ทุกสกุล · Superadmin = ยอดจริง) · `total_sum` = ยอดรวมของ**ลูกตรงทั้งหมด** (ทุกหน้า · ตาม `keyword` เดียวกับรายชื่อ) แยกตาม `role` **key ตัวเล็ก** (`agent` · `member` · `shareholder` · `company` — แก้ 2026-10-14 จากตัวใหญ่ · field `role` ในแถวยังตัวใหญ่) → map 27 สกุล · ใส่เฉพาะ role ที่มีลูก · ไม่มีลูก = `{}` · รวมด้วย query SUM 1 ครั้ง (`GROUP BY role, currency`) · controller ประกอบกับ `response.PageData` (dto / service ไม่ import `pkg/response`) |
| DW-45 | API ส่งยอดตามที่เก็บ ไม่ปัดเพิ่ม (กฎข้อ 9) |

### ประวัติ (DW3)

| ID | Rule |
|---|---|
| DW-46 | เห็นเฉพาะรายการที่ `parent_id = แม่` (รายการที่บัญชีนี้หรือ sub ของบัญชีนี้ทำ — กฎข้อ 22) · ลูกไม่เห็นรายการที่แม่ทำกับตัวเองในเส้นนี้ (ลูกดูได้ที่ Report / statement — module อื่น ทำทีหลัง · lead 2026-10-14) · มีทั้งแถว `SUCCESS` และ `FAILED` (DW-35) · แต่ละแถวมี `status` และ `ip` (ip ของคนกด) |
| DW-47 | `date_from` / `date_to` บังคับ รูปแบบ `YYYY-MM-DD` เวลาไทย (+07:00) รวมทั้ง 2 วัน · `date_from` ≤ `date_to` · ช่วงไม่เกิน 31 วัน (`422`) |
| DW-48 | ตัวกรองไม่บังคับ: `transaction_type` (`DEPOSIT` / `WITHDRAW` · `""` = ทั้งหมด) · `role` + `user_id` (ลูกคนเดียว · ส่งคู่กัน · `""` / `0` = ทุกคน) · เรียง `created_at` ใหม่→เก่า แล้ว `id` ใหม่→เก่า · `page` / `limit` ใน body |
| DW-49 | index `idx_deposit_withdraw_logs_parent (parent_id, created_at DESC)` รองรับ query หลัก · กรองลูกใช้ index เดียวกัน + เงื่อนไข `target_*` |

### ข้อบังคับของ module เดิมพัน (DQ4 — open item)

| ID | Rule |
|---|---|
| DW-50 | **module เดิมพันต้องหักเงินจาก `credit` ทันทีตอนแทง** (ข้อบังคับของ Bet) → ถอนไม่กระทบรอบเกมค้าง |
| DW-51 | module เดิมพันที่ lock `agent_balances` + `user_members` พร้อมกันต้องใช้ลำดับ DW-32 (ARCHITECTURE.md) |
| DW-52 | **ยอดของสาย Seamless ใช้แทงไม่ได้** (lead 2026-10-15): Member สาย Seamless แทงจาก wallet ของ operator (seamless callback) · ยอดใน `agent_balances` / `user_members.credit` ของสาย Seamless เป็นแค่วงเงินที่ย้ายในสาย — module เดิมพันต้องไม่ตัดยอดนี้ |

### ประวัติของบัญชีในสาย (DW4) — lead อนุมัติ 2026-10-15

| ID | Rule |
|---|---|
| DW-55 | ค้นประวัติฝาก / ถอนที่**ทำให้บัญชีหนึ่ง** (ฝั่งที่ถูกทำรายการ) ด้วย `username` **ตรงตัว** (ตัดช่องว่าง · ตัวเล็ก — username เก็บตัวเล็ก MGMT-05) · ไม่ส่ง / `""` = `422` "ต้องระบุชื่อผู้ใช้งาน" |
| DW-56 | บัญชีที่ค้นได้: agent หรือ Member ที่อยู่**ใต้ตัวเองทุกชั้น** (`cnf @> {"parent":[{"id": แม่}]}` — MGMT-61 · กฎข้อ 22 · sub = ใต้เจ้าของ) · **ค้นตัวเองไม่ได้** · ไม่มี username นี้ / นอกสาย / ตัวเอง / ADMIN = `404402` "ไม่พบข้อมูลบัญชี" (ข้อความเดียวกันทุกกรณี) |
| DW-57 | ผลรวมรายการที่ทำโดยใครก็ได้ในสาย (แม่ตรงของบัญชีนั้น หรือ sub ของแม่) · แสดง**คนกดจริง** `request_by` + `request_type` (`AGENT` / `SUB` — id ซ้ำกันได้) + `request_username` (sub = `share01@staff`) |
| DW-58 | `starttime` / `endtime` บังคับ · RFC3339 มี timezone (เช่น `2026-10-15T00:00:00+07:00`) · `starttime` ≤ `endtime` · รวมทั้ง 2 ขอบ (`created_at >= starttime AND created_at <= endtime`) · **ไม่จำกัดช่วง** (query อยู่ที่บัญชีเดียว — DW-60) |
| DW-59 | เรียง `created_at` ใหม่→เก่า แล้ว `id` ใหม่→เก่า · `page` / `limit` ใน body (`utils.NewPage` · ค่าเริ่มต้น 20 สูงสุด 100) · response = PageData มาตรฐาน · `currency` / `transaction_type` ตัวใหญ่ · `remark` = `note` · ไม่มี `updated_at` (log แก้ไม่ได้) · มีแถว `FAILED` ด้วย + field `status` (DW-35) |
| DW-60 | หา username 1 query ต่อตาราง (`user_agents` / `user_members` · username ไม่ซ้ำข้าม 2 ตาราง — MGMT-05) แล้วดึง log ด้วย `target_type` + `target_id` + ช่วงเวลา → index `idx_deposit_withdraw_logs_target (target_type, target_id, created_at DESC)` ที่มีอยู่แล้ว · ไม่ต้องทำ migration |

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
| 9 | DW4 แยกเส้นจาก DW3 (lead อนุมัติ 2026-10-15) | DW3 = รายการที่ตัวเองเป็นแม่ · DW4 = รายการที่ทำให้บัญชีหนึ่งในสาย (ทุกชั้น · ผู้ทำรายการเป็นใครก็ได้ในสาย) · payload ตามหน้าจอ (`username` · `starttime` / `endtime`) แต่ response ตาม `response.Page` ของทีม (`limit` แทน `perpage`) |
| 10 | status ของรายการมีแค่ `SUCCESS` / `FAILED` ไม่มี `pending` (DW-35 · 2026-10-16) | ดูเหตุผลด้านล่าง |

**ทำไมไม่มี `pending`** (DW-35 — ใช้ตอบคำถามว่า "ควรเพิ่ม pending ไหม")

1. **ย้ายเงินจบใน transaction เดียว (DW-30):** หักแม่ · เพิ่มลูก · log · ledger · `create_requests` commit พร้อมกันหรือ rollback พร้อมกัน — ไม่มีจังหวะที่เงินขยับไปครึ่งทาง จึงไม่มีสถานะ "กำลังทำ" ให้ใครเห็น
2. **pending ต้องเขียนแยก tx ก่อนย้ายเงิน:** ถ้า server ล่ม / timeout / deploy ระหว่างนั้น แถวจะค้าง `pending` ตลอดไป ต้องมี job ตามเก็บ (reconcile) เพิ่ม — ซับซ้อนขึ้นแต่ไม่ได้ข้อมูลอะไรเพิ่ม
3. **pending มีความหมายเฉพาะงานที่รอระบบภายนอก** (เช่น เรียก wallet ของ operator / ธนาคาร แล้วรอผล) — DW ไม่ได้เรียกระบบภายนอก ยอดอยู่ใน DB ของเราทั้งหมด
4. **กันกดซ้ำ / เน็ตหลุดมีอยู่แล้ว:** Redis lock (DW-20 – 23) กันยิงพร้อมกัน · `request_id` (DW-24 – 27) ทำให้ client ส่งซ้ำแล้วได้ผลเดิม — ไม่ต้องใช้ pending บอกว่ารายการยังไม่จบ
5. **รายการที่ไม่สำเร็จยังเห็นได้:** บันทึกแถว `FAILED` พร้อมเหตุผลหลัง rollback (DW-36)

**เมื่อไรควรเพิ่ม `pending`:** ถ้าวันหนึ่งการฝาก / ถอนต้องเรียกระบบภายนอกแล้วรอผล (เช่น โอนเข้า wallet ของ operator ฝั่ง Seamless) — ต้องเพิ่มพร้อม job reconcile และกติกา timeout ผ่าน spec ใหม่

## 5. Endpoints

ใต้ `/bo/pr` ผ่าน `Authenticated` + `PassedGates()` · error ร่วม: `401202`, `401203`, `401301`, `401302`, `401304`, `401306`, `401307`, `401311`, `401312`, `402303`, `404305`

| Method | Path | สิทธิ์ (route) | หน้าที่ |
|---|---|---|---|
| POST | `/api/v1/bo/pr/deposit-withdraw/downlines/list` | `RequirePermission(payment, view)` | DW1 รายชื่อลูกตรง + ยอด |
| POST | `/api/v1/bo/pr/deposit-withdraw/create` | `RequirePermission(payment, edit)` · `RequirePasscode()` | DW2 ฝาก / ถอน |
| POST | `/api/v1/bo/pr/deposit-withdraw/logs/list` | `RequirePermission(payment, view)` | DW3 ประวัติ |
| POST | `/api/v1/bo/pr/deposit-withdraw/history/list` | `dwMw.RejectAdmin()` · `RequirePermission(payment, view)` | DW4 ประวัติของบัญชีในสาย |

ตัวอย่างใช้ข้อมูลชุดเดียวกับ agent_management: `comp01` (id 10, Company Transfer) → `share01` (id 12, Share B2C · THB) → `agent01` (id 15) / `mem01` (Member id 501) · sub ของ share01 = `share01@staff` (id 31)

### DW1 · POST /api/v1/bo/pr/deposit-withdraw/downlines/list

Request (ไม่บังคับ):
```json
{ "keyword": "", "page": 1, "limit": 20 }
```
Response `data` (share01 login · map สกุลย่อเหลือบางตัว — ของจริงมีครบ 27 สกุล):
```json
{
  "credit_detail": { "currency": { "THB": 900, "USD": 0, "CNY": 0 } },
  "total_sum": {
    "agent":  { "THB": 5000, "USD": 0, "CNY": 0 },
    "member": { "THB": 100,  "USD": 0, "CNY": 0 }
  },
  "current_page": 1, "total_page": 1, "total_count": 2, "limit": 20, "has_next": false, "has_prev": false,
  "data": [
    { "id": 15,  "role": "AGENT",  "username": "agent01", "name": "agent01", "status": "ACTIVE", "has_balance": true,
      "currency": { "THB": 5000, "USD": 0, "CNY": 0 },
      "last_login_ip": "203.0.113.10", "last_login_at": "2026-10-08T21:00:00+07:00" },
    { "id": 501, "role": "MEMBER", "username": "mem01",   "name": "ใจดี",    "status": "LOCKED", "has_balance": true,
      "currency": { "THB": 100, "USD": 0, "CNY": 0 },
      "last_login_ip": "", "last_login_at": "" }
  ]
}
```
- `mem01` ถูกล็อก: ปุ่มฝากปิด · ปุ่มถอนเปิด (DW-44)
- `total_sum` รวมลูกตรงทุกคน (ไม่ใช่แค่หน้านี้) · มีเฉพาะ role ที่มีลูก (DW-44A)

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
- `parent_unlimited` = `false` เสมอสำหรับรายการใหม่ (Superadmin มียอดจริงตั้งแต่ 2026-10-15) · `true` เฉพาะรายการเก่าที่ Superadmin ยังไม่มียอด — `parent_balance_*` = `0`
- ส่งซ้ำด้วย `request_id` เดิม + ค่าเดิม = `data` เดิม (DW-25)

Error codes: `422`, `404301`, `404303`, `404304`, `404306`, `404307`, `404401`, `504801` + error ของ passcode (`401204`, `401205`, `401304`)

**ลำดับการเช็ค** (ตามลำดับ middleware ใน route — แก้ตอน implement 2026-10-14: ADMIN ไม่มีเมนูใดเลย ถ้าเช็คสิทธิ์ก่อนจะได้ `402303` แทน `404305`):
ADMIN (`404305` — `dwMw.RejectAdmin`) → สิทธิ์ `payment` (`402303`) → passcode (`422` / `401204` …) → รูปแบบ body (`400` / `422`) → Redis lock (`404307` / `504801`) → `request_id` (ผลเดิม / `422`) →
หาลูก (`404401`) → สถานะลูก เฉพาะฝาก (`404301`) → สกุล (`404303`) → lock + ยอด (`404304` / `404306`)

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
      "status": "SUCCESS",
      "actor_username": "share01@staff",
      "role": "MEMBER",
      "user_id": 501,
      "username": "mem01",
      "currency": "THB",
      "amount": 50,
      "note": "",
      "ip": "203.0.113.10",
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
      "status": "SUCCESS",
      "actor_username": "share01",
      "role": "MEMBER",
      "user_id": 501,
      "username": "mem01",
      "currency": "THB",
      "amount": 100,
      "note": "เติมเครดิตรอบแรก",
      "ip": "203.0.113.10",
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

### DW4 · POST /api/v1/bo/pr/deposit-withdraw/history/list

ใช้กับหน้าจอ "ประวัติฝาก / ถอน": ค้น `username` แล้วดูว่าใครทำรายการให้บัญชีนั้นบ้าง (DW-55 – DW-60) · ตัวอย่าง: comp01 ค้น `agent01` (หลาน) → เห็นรายการที่ share01 / `share01@staff` ทำให้ agent01

Request:
```json
{
  "username": "agent01",
  "starttime": "2026-10-15T00:00:00+07:00",
  "endtime": "2026-10-15T23:59:59+07:00",
  "page": 1,
  "limit": 100
}
```

| Field | บังคับ | Rule |
|---|---|---|
| `username` | ✓ | ตรงตัว (ไม่สนตัวพิมพ์) · agent หรือ Member ใต้ตัวเองทุกชั้น · ตัวเอง / นอกสาย / ไม่มี = `404402` |
| `starttime` / `endtime` | ✓ | RFC3339 มี timezone · `starttime` ≤ `endtime` · รวมทั้ง 2 ขอบ · ไม่จำกัดช่วง |
| `page` / `limit` | | ค่าเริ่มต้น 1 / 20 · สูงสุด 100 |

Response `data` (เรียงใหม่ → เก่า):
```json
{
  "current_page": 1, "total_page": 1, "total_count": 1, "limit": 100, "has_next": false, "has_prev": false,
  "data": [
    {
      "id": 9003,
      "agent_id": 15,
      "agent_username": "agent01",
      "role": "AGENT",
      "request_by": 31,
      "request_type": "SUB",
      "request_username": "share01@staff",
      "currency": "THB",
      "transaction_type": "DEPOSIT",
      "status": "SUCCESS",
      "amount": 1000,
      "remark": "",
      "created_at": "2026-10-15T11:58:24.123456+07:00"
    }
  ]
}
```
- `agent_id` / `agent_username` / `role` = บัญชีที่ถูกทำรายการ (Member ก็ใช้ชื่อ field นี้ · `role = MEMBER`) · `request_*` = คนกดจริง
- `endtime` ที่ลงท้าย `23:59:59` ไม่รวมรายการในเสี้ยววินาทีสุดท้าย (`23:59:59.5`) — หน้าบ้านส่ง `23:59:59.999999` ได้ถ้าต้องการให้ครบวัน

Error codes: `422`, `404402`

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
    `DEPOSIT_OUT` · `DEPOSIT_IN` · `WITHDRAW_OUT` · `WITHDRAW_IN` · `SEED_SUPERADMIN` (migration `20261015100000`) · `DEPOSIT_FROM_SUPERADMIN` / `WITHDRAW_TO_SUPERADMIN` เลิกใช้ 2026-10-15
  - `create_requests.target_type` (VARCHAR(10) · ไม่มี CHECK) รองรับ `'DW'` อยู่แล้ว (DW-26)
- index ของ DW1 มีแล้ว: `idx_user_agents_parent_username` · `idx_user_members_agent_username`
- migration `20261015100000_deposit_withdraw_superadmin_balances` (lead 2026-10-15): seed ยอด Superadmin ตาม DW-14 · ไม่แก้ schema · `parent_unlimited` ยังอยู่เพื่อแถวเก่า
- migration `20261016140000_deposit_withdraw_logs_status` (DW-35): เพิ่ม `status VARCHAR(10) NOT NULL DEFAULT 'SUCCESS'` + CHECK · unique `request_id` เปลี่ยนเป็น partial unique index `WHERE status = 'SUCCESS'` (แถว FAILED ไม่จอง) · โค้ดต้องตั้ง `status` เอง (GORM ส่ง `""` แทนค่า default)
- Redis: key `dw:lock:{AGENT|MEMBER}:{id}` · TTL 10 วินาที · ไม่ใช่ source of truth ของยอด (กฎข้อ 14)

## 7. Test cases

| Rule | Input | Expected |
|---|---|---|
| DW-01 | share01 ฝากให้ agent01 · ให้ Member ของ agent01 (หลาน) · id ที่ไม่มี · บัญชีสายอื่น | สำเร็จ · `404401` · `404401` · `404401` |
| DW-02 | `role: "MEMBER", user_id: 15` (id 15 เป็น agent) · `role: "COMPANY"` กับ agent01 · `role: "member"` | `404401` · `404401` · `422` |
| DW-03 | ADMIN เรียก DW1 / DW2 / DW3 | `404305` |
| DW-04 | sub มี `payment` view เรียก DW1 / DW3 · เรียก DW2 · sub มีแค่ `member` edit | สำเร็จ · `402303` · `402303` |
| DW-04 | sub ของ Superadmin (`payment` edit) ฝาก 1,000 ให้ Company | สำเร็จ · log `actor_type = SUB` · `parent_id` = Superadmin · ยอด Superadmin ลด 1,000 |
| DW-05 | ฝากให้ลูก `SUSPENDED` · `LOCKED` | `404301` ทั้งคู่ · ยอดไม่เปลี่ยน |
| DW-05 | ถอน 100 จากลูก `LOCKED` ที่มี 300 | สำเร็จ · ลูก 200 · แม่ +100 |
| DW-06 | Superadmin (THB 100) ฝาก 5 ให้ Company Seamless Reseller แล้วถอน 2 · Company Seamless 1 to 1 สร้าง Member พร้อมยอด | Reseller 3 · Superadmin 97 · สำเร็จ |
| DW-07 | ไม่ส่ง passcode · passcode ผิด | `422` · `401204` |
| DW-10 | `amount` `0` · `-5` · `1.00001` · `"100"` · `1000000001` · `1000000000` · `25.5` | `422` · `422` · `422` · `400` · `422` · สำเร็จ · สำเร็จ ยอด 25.5 |
| DW-10 | ยอด 0.1 ฝาก 0.2 | ยอด = 0.3 (ไม่ใช่ 0.30000000000000004) |
| DW-11 | Share B2B (THB, USD) ฝาก USD ให้ Agent THB · ฝาก JPY | `404303` · `404303` |
| DW-12 | Company THB 50,000 ฝาก 10,000 ให้ Share | Company 40,000 · Share +10,000 · ledger 2 แถว · log 1 แถว · `create_requests` 1 แถว (`DW`) |
| DW-12 | Company THB 50,000 ฝาก 60,000 | `404304` · ไม่มี log / ledger / `create_requests` |
| DW-12 | Superadmin (THB 100,000,000) ฝาก 1,000,000 ให้ Company · ฝากเกินยอดที่เหลือ | Superadmin 99,000,000 · Company +1,000,000 · ledger `DEPOSIT_OUT` + `DEPOSIT_IN` · `parent_unlimited = false` · `404304` |
| DW-13 | Member 100 ถอน 100 · ถอน 100.0001 | สำเร็จ `credit` = 0 · `404306` |
| DW-14 | Superadmin ถอน 500 จาก Company · migration: Superadmin ขาด 2 สกุล + มีแถวยอด JPY อยู่แล้ว | Company −500 · Superadmin +500 · ledger `WITHDRAW_IN` + `WITHDRAW_OUT` · ครบ 27 สกุล · ยอด 100,000,000 26 แถว (JPY คงเดิม) · ledger `SEED_SUPERADMIN` 26 แถว |
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
| DW-35 / 36 | ฝากเกินยอด (note "ลองฝาก") · กดซ้ำ `request_id` เดิมหลังเติมยอด · ส่งซ้ำอีกครั้ง · ฝากให้ลูกที่ถูกล็อก · body ผิด / ไม่ใช่ลูกตรง / ติด lock | `404304` + แถว FAILED note "ลองฝาก · ล้มเหลว: ยอดเงินของคุณไม่พอ" ยอดก่อน = หลัง ไม่มี ledger / create_requests · สำเร็จ (แถว SUCCESS) · ผลเดิม (ไม่เพิ่มแถว) · `404301` + FAILED · ไม่มีแถวเพิ่ม |
| DW-36 (C-S1) | เรียก `recordFailure` ตรง: `create_requests` มีแล้ว · ไม่มี · tx อื่นถือ advisory lock แล้ว commit ช้า 500 ms | ไม่เขียน FAILED · เขียน FAILED (ยอด ณ ตอนบันทึก) · รอแล้วไม่เขียน FAILED |
| DW-40 / 41 | share01 มีลูก agent 1 + Member 1 + ADMIN ในระบบ · `keyword` `"mem0"` · `"ab"` · keyword ของหลาน | 2 แถว A→Z ไม่มี ADMIN · `mem01` · `422` · ไม่พบ |
| DW-43 | แม่ / ลูก / Member ที่มียอดสกุลเดียว · Share B2B THB 7 + USD 3 | `currency` ครบ 27 key · สกุลอื่น = 0 · THB 7 · USD 3 |
| DW-44 | Company Seamless (THB 33) ใน list ของ Superadmin | `has_balance = true` · `currency` THB 33 |
| DW-44A | share01 (THB 900) มี agent01 9,900 + Member 100.5 · `limit 1` · `keyword` เฉพาะ Member · keyword ไม่เจอ · comp01 ดู (share01 900 + shareb2b 7 / USD 3) · Superadmin ดู | `credit_detail` THB 900 · `total_sum` `{agent: 9900, member: 100.5}` · ยังรวม 2 role · มีแค่ `member` · `{}` + `credit_detail` ยังมี · `shareholder` THB 907 / USD 3 · `credit_detail` 0 ทุกสกุล |
| DW-46 | share01 ดู DW3 · comp01 ดู DW3 | เห็นเฉพาะรายการที่ `parent_id = 12` · comp01 ไม่เห็นรายการของ share01 |
| DW-47 | `date_from` > `date_to` · ช่วง 32 วัน · `"2026/10/01"` | `422` |
| DW-48 | ส่ง `role` ไม่ส่ง `user_id` · `transaction_type: "WITHDRAW"` | `422` · เฉพาะรายการถอน |
| DW-55 | ไม่ส่ง `username` · `""` · `" AGENT01 "` | `422` · `422` · ค้น `agent01` ได้ |
| DW-56 | comp01 ค้น `agent01` (หลาน) · ค้น Member ของ agent01 · ค้น `comp01` (ตัวเอง) · ค้นบัญชีสายอื่น · ค้น username ที่ไม่มี · ค้น ADMIN | เห็นรายการ · เห็นรายการ · `404402` ทุกกรณีที่เหลือ (ข้อความเดียวกัน) |
| DW-57 | `share01@staff` ฝากให้ agent01 แล้ว comp01 ค้น `agent01` | แถวมี `request_type = SUB` · `request_username = share01@staff` · `request_by` = id ของ sub |
| DW-58 | `starttime` > `endtime` · ไม่มี timezone · `"2026-10-15"` · ช่วง 1 ปี · รายการที่ `created_at` = `endtime` พอดี | `422` · `422` · `422` · สำเร็จ · อยู่ในผล |
| DW-59 | รายการ 3 แถว `limit 2` | หน้า 1 = 2 แถวใหม่สุด · `total_page = 2` · `currency` / `transaction_type` ตัวใหญ่ |

## 8. Contract changes (สำหรับแจ้ง frontend)

เส้นใหม่ทั้ง 3 เส้น · เอกสารหน้าบ้าน [docs/frontend/deposit_withdraw.md](../frontend/deposit_withdraw.md)

**เปลี่ยนพฤติกรรม 2026-10-15 (lead อนุมัติ)**: Superadmin ฝากเกินยอดที่มีได้ `404304` · `parent_unlimited` = `false` และ `parent_balance_*` เป็นยอดจริงของ Superadmin · `credit_detail` ของ Superadmin เป็นยอดจริง

**Breaking change ของ DW1 (2026-10-14 · lead อนุมัติ · หลัง deploy `dev-202610140819`)**: แถวเปลี่ยน `balances` (array) → `currency` (map 27 สกุล) ·
เพิ่ม `last_login_ip` / `last_login_at` · `data` เพิ่ม `credit_detail` และ `total_sum` (field แบ่งหน้าเดิมอยู่ที่เดิม)

- หน้าฝากถอนใช้ DW1 · ส่ง `role` + `id` + สกุลจากแถวไปที่ DW2 ตรงๆ
- DW2 สร้าง `request_id` ใหม่ทุกครั้งที่กด (กดซ้ำเพราะเน็ตหลุด = ใช้ค่าเดิม) · ส่ง `passcode`
- ปุ่มฝาก: `has_balance` และ `status = ACTIVE` · ปุ่มถอน: `has_balance`
- `404307` = มีรายการของบัญชีนี้กำลังทำอยู่ ให้ลองใหม่อีกครั้ง (ไม่ต้องสร้าง `request_id` ใหม่)
