# ฝาก / ถอนเงินให้ลูกตรง (`deposit_withdraw`) — Spec

- สถานะ: **DRAFT** — รอ lead review
- ผู้เขียน: maofoy · วันที่: 2026-10-12
- ชื่อ module ในโค้ด: `deposit_withdraw` (`controllers/deposit_withdraw` · `dto/deposit_withdraw` · `service/deposit_withdraw`)
- rule ID: `DW-xx`
- โค้ดเก่าที่อ้างอิง: ไม่มี — ไม่ได้ extract จาก `askmelotto_minigame-api`

> เอกสารนี้คือ source of truth ของ module — โค้ดและ test ต้องตรงกับเอกสารนี้ ถ้าไม่ตรงให้แก้โค้ด
> หรือแก้เอกสารผ่านการอนุมัติใหม่ ห้ามอ้างเอกสารของโปรเจกต์เก่า

## 1. หน้าที่ของ module

บัญชีหลังบ้าน (Superadmin · Company · Share · Agent และ sub ของบัญชีเหล่านี้) **ฝาก / ถอนเงินให้ลูกตรงของตัวเอง** ทั้งฝั่ง agent และ Member
และดูรายชื่อลูกตรงพร้อมยอดเงินสำหรับหน้าฝากถอน · ทุกรายการเก็บ log ของ module นี้ (`log_deposit_withdraw`) และ `balance_ledger`

**ไม่อยู่ใน module นี้:** ยอดเงินตั้งต้นตอนสร้างบัญชี (agent_management MGMT-15A) · เส้นดูประวัติ log (ยังไม่ทำ) · การหัก / คืนเงินจากการเล่นเกม (module เดิมพัน)

## 2. คำศัพท์

| คำ | ความหมาย |
|---|---|
| แม่ | บัญชีที่ทำรายการ = `Actor.AgentID` (sub = เจ้าของ sub — AUTH-26) |
| ลูกตรง | agent ที่ `user_agents.parent_id = แม่` (ไม่รวม ADMIN) หรือ Member ที่ `user_members.agent_id = แม่` · ไม่รวมหลาน |
| ฝาก (`DEPOSIT`) | แม่ลด · ลูกเพิ่ม |
| ถอน (`WITHDRAW`) | ลูกลด · แม่เพิ่ม |
| ยอดของ agent | `agent_balances.amount` ต่อสกุล |
| ยอดของ Member | `user_members.credit` (Member มี 1 สกุล = `user_members.currency`) |

## 3. Business rules

### ใครทำได้ / ทำกับใคร

| ID | Rule |
|---|---|
| DW-01 | ทำได้เฉพาะ**ลูกตรง**ของแม่ · เช็คด้วย `user_agents.parent_id` / `user_members.agent_id` (ไม่ใช้ `cnf`) · หลาน · นอกสาย · ไม่มี id นี้ = `404401` (ข้อความเดียวกันทุกกรณี ไม่บอกว่ามี id อยู่ในระบบไหม) |
| DW-02 | ระบุบัญชีด้วย `role` + `user_id`: `MEMBER` → `user_members` · `COMPANY` / `SHAREHOLDER` / `AGENT` → `user_agents` และ role ต้องตรงกับในตาราง · ไม่ตรง = `404401` · ค่าอื่น = `422` |
| DW-03 | ADMIN ทำรายการไม่ได้ทั้ง 2 เส้น (`404305`) |
| DW-04 | sub ทำได้ตามสิทธิ์เมนู `payment`: list = view · ฝากถอน = edit (`402303` — เช็คที่ route กฎข้อ 28) |
| DW-05 | แม่ที่ถูกระงับทำไม่ได้ (`PassedGates()` — AUTH-54) · **ลูกที่ `SUSPENDED` / `LOCKED` ฝาก / ถอนไม่ได้** (`404301`) |
| DW-06 | บัญชีฝั่ง Seamless ไม่มียอดเงิน (MGMT-15) ทำรายการไม่ได้ (`404302`) · ตัดสินจากประเภท Company หัวสายของลูก |
| DW-07 | ต้องยืนยัน passcode ของตัวเอง (`RequirePasscode()` — `passcode` ใน body) |

### จำนวนเงินและสกุล

| ID | Rule | ตัวอย่าง |
|---|---|---|
| DW-10 | `credit` มากกว่า 0 · ทศนิยมไม่เกิน 4 ตำแหน่ง · ปัด `utils.Round4` ก่อนบันทึกและหลังบวก / ลบ (กฎข้อ 9) | `0` / `-1` / `1.00001` = `422` |
| DW-11 | `currency` บังคับ · ต้องเป็นสกุลของลูก (agent = `agent_currencies` · Member = `user_members.currency`) **และ**ของแม่ (Superadmin ไม่ต้องเช็คฝั่งแม่) · ไม่ตรง = `404303` | Share B2B (THB, USD) ฝาก JPY ให้ Agent THB = `404303` |
| DW-12 | ฝาก: ยอดแม่ในสกุลนั้นต้อง ≥ `credit` (`404304`) · **Superadmin วงเงินไม่จำกัด** ไม่มีแถวยอดให้หัก | Company 50,000 ฝาก 60,000 = `404304` |
| DW-13 | ถอน: ยอดลูกต้อง ≥ `credit` (`404306`) · **ถอนได้หมดเท่าที่มีในกระเป๋าตอนนั้น** (ไม่หักยอดที่ค้างในเกม — ดูข้อสมมติ A1) | Member 100 ถอน 100 = สำเร็จ เหลือ 0 |
| DW-14 | ถอนจากลูกของ Superadmin: ลูกลดอย่างเดียว ไม่มียอดฝั่ง Superadmin ให้เพิ่ม | |

### Transaction และ idempotent

| ID | Rule |
|---|---|
| DW-20 | ทุกอย่างใน **transaction เดียว** ที่ service เปิด (กฎข้อ 10–11): lock ยอด → เช็คยอด → อัปเดตยอด → `log_deposit_withdraw` 1 แถว → `balance_ledger` |
| DW-21 | **ลำดับ lock ตายตัว** (กันผูกตาย): `agent_balances` ของทุกฝ่ายเรียง `agent_id` น้อย→มาก (`FOR UPDATE`) **ก่อน** แล้วจึง `user_members` (ถ้าลูกเป็น Member) · module อื่นที่ lock 2 ตารางนี้พร้อมกันต้องใช้ลำดับเดียวกัน |
| DW-22 | แม่ไม่มีแถว `agent_balances` ของสกุลนั้น: ฝาก = ยอด 0 → `404304` · ถอน (แม่ได้เงิน) = สร้างแถวใหม่ใน tx เดียวกัน |
| DW-23 | `balance_ledger` (กฎข้อ 11): ฝาก = แม่ `DEPOSIT_OUT` (−) + ลูก `DEPOSIT_IN` (+) · ถอน = ลูก `WITHDRAW_OUT` (−) + แม่ `WITHDRAW_IN` (+) · Superadmin = เฉพาะแถวของลูก · ทุกแถว `ref_type = 'DW'` · `ref_id = log_deposit_withdraw.id` · `request_id` เดียวกับคำขอ · `actor_*` = คนกดจริง |
| DW-24 | **กันยิงซ้ำ** (กฎข้อ 12): `request_id` (UUID · บังคับ) unique ใน `log_deposit_withdraw` · ส่งซ้ำด้วยบัญชีเดิม (`actor_type` + `actor_id`) และค่าเดิมทุก field = ได้ผลเดิม (`200` · `data` จาก log เดิม) ไม่ย้ายเงินซ้ำ · บัญชีอื่น / ค่าไม่ตรง / เคยใช้ในเส้นอื่น (`create_requests`) = `422` "request_id นี้ถูกใช้แล้ว" (แบบเดียวกับ MGMT-15A — lead Q-R4) |
| DW-25 | ยิงพร้อมกันด้วย `request_id` เดียวกัน: แถวที่ insert ไม่ผ่าน unique → rollback แล้วอ่าน log เดิมกลับไปตาม DW-24 |

### รายชื่อลูกตรง

| ID | Rule |
|---|---|
| DW-30 | เฉพาะลูกตรงของแม่ (agent ไม่รวม ADMIN + Member) · `UNION ALL` 2 ตาราง แบบเดียวกับ `downlines/list` · เรียง username A→Z · `page` / `limit` ใน body (`utils.NewPage` · ค่าเริ่มต้น 20 สูงสุด 100) · ไม่มีค้นหา |
| DW-31 | แต่ละแถว: `id` · `role` · `username` · `name` · `status` · `has_balance` · `balances` · `role` ใช้ส่งกลับในเส้นฝากถอนได้ทันที (DW-02) |
| DW-32 | `balances`: agent = 1 รายการต่อสกุลใน `agent_currencies` (ไม่มีแถวยอด = `0`) · Member = 1 รายการ (`currency` + `credit`) · ยอดฝั่ง agent ดึงด้วย `agent_id IN (...)` ครั้งเดียวต่อหน้า (กฎข้อ 23 ห้าม N+1) |
| DW-33 | `has_balance = false` = บัญชีฝั่ง Seamless (DW-06) — หน้าบ้านใช้ปิดปุ่ม · `status` = สถานะของลูกเอง (แม่ผ่าน `PassedGates()` แล้ว สายด้านบนจึงไม่ถูกระงับ) |
| DW-34 | API ส่งยอดตามที่เก็บ ไม่ปัดเพิ่ม (กฎข้อ 9) |

### ข้อสมมติ

| ID | ข้อสมมติ | ถ้าไม่จริง |
|---|---|---|
| A1 | เอนจิ้นเกม (ยังไม่มี) **หักเงินจาก `credit` ทันทีตอนแทง** ยอดที่เหลือใน `credit` จึงถอนได้หมด (DW-13) | ถ้าเกมหักตอน settle ทีหลัง Member ถอนออกก่อนแล้ว settle ไม่ผ่าน CHECK `credit >= 0` — ต้องแก้ DW-13 ก่อนทำ module เดิมพัน |
| A2 | เอนจิ้นเกม lock `user_members` ตามลำดับ DW-21 | เสี่ยง deadlock ระหว่างฝากถอนกับการแทง |

## 4. สิ่งที่พบในโค้ดเก่า และการตัดสินใจ

ไม่ได้ extract จากโค้ดเก่า (repo เก่าไม่มีในเครื่อง) — การตัดสินใจระหว่างเขียน spec:

| # | เรื่อง | การตัดสินใจ |
|---|---|---|
| 1 | ระบุลูกด้วย `user_id` อย่างเดียว | ไม่พอ — id ของ `user_agents` กับ `user_members` ซ้ำกันได้ · เพิ่ม `role` (DW-02) |
| 2 | เช็คลูกตรงด้วย `cnf` หรือ `parent_id` | `parent_id` / `agent_id` (มี index `idx_user_agents_parent_username` · `idx_user_members_agent_username` อยู่แล้ว) |
| 3 | ใช้ `downlines/list` (agent_management E2) แทนเส้น list ใหม่ | แยกเส้น: (1) สิทธิ์คนละเมนู — E2 ใช้ `member` view · หน้านี้ใช้ `payment` view ทำให้ sub ที่มีแค่ `payment` ใช้ได้ (2) field เฉพาะหน้าฝากถอน (`has_balance`) ไม่มี pt / status_game |
| 4 | log ของ module เอง vs ใช้ `balance_ledger` อย่างเดียว | ทำทั้งคู่: `log_deposit_withdraw` = 1 แถวต่อการกด (ใครทำ · ทำกับใคร · ยอดก่อน / หลังทั้ง 2 ฝั่ง) · `balance_ledger` = รายการต่อยอด (กฎข้อ 11 · statement กลาง) |
| 5 | ช่อง `note` | ยังไม่ทำ — เพิ่ม migration ใหม่ได้ภายหลังโดยไม่กระทบข้อมูลเดิม |
| 6 | เส้นดูประวัติ log | ยังไม่ทำ |

**คำถามถึง lead:**

| # | คำถาม | ข้อเสนอ |
|---|---|---|
| Q1 | path ของ 2 เส้น | ตามหัวข้อ 5 |
| Q2 | สิทธิ์ของเส้น list = `payment` view | ตามนั้น |
| Q3 | จอง error module id `04` — code จะเป็น `404xxx` อ่านคล้าย HTTP 404 | ใช้ `04` ตามลำดับ (ไม่ชน `404001` เพราะ `c` ≠ 0) |
| Q4 | `request_id` ต้องเช็คข้ามกับ `create_requests` ไหม (DW-24) | เช็ค — ให้ความหมายเดียวกับ MGMT-15A |

## 5. Endpoints

ใต้ `/bo/pr` ผ่าน `Authenticated` + `PassedGates()` · error ร่วม: `401202`, `401203`, `401301`, `401302`, `401304`, `401306`, `401307`, `401311`, `401312`, `402303`

| Method | Path | สิทธิ์ (route) | หน้าที่ |
|---|---|---|---|
| POST | `/api/v1/bo/pr/deposit-withdraw/downlines/list` | `RequirePermission(payment, view)` | DW1 รายชื่อลูกตรง + ยอด |
| POST | `/api/v1/bo/pr/deposit-withdraw/create` | `RequirePermission(payment, edit)` · `RequirePasscode()` | DW2 ฝาก / ถอน |

ตัวอย่างใช้ข้อมูลชุดเดียวกับ agent_management: `comp01` (id 10, Company Transfer) → `share01` (id 12, Share B2C · THB) → `agent01` (id 15) / `mem01` (Member id 501)

### DW1 · POST /api/v1/bo/pr/deposit-withdraw/downlines/list

Request (ไม่บังคับ):
```json
{ "page": 1, "limit": 20 }
```
Response `data` (share01 login):
```json
{
  "current_page": 1, "total_page": 1, "total_count": 2, "limit": 20, "has_next": false, "has_prev": false,
  "data": [
    { "id": 15,  "role": "AGENT",  "username": "agent01", "name": "agent01", "status": "ACTIVE",
      "has_balance": true, "balances": [ { "currency": "THB", "amount": 5000 } ] },
    { "id": 501, "role": "MEMBER", "username": "mem01",   "name": "ใจดี",    "status": "ACTIVE",
      "has_balance": true, "balances": [ { "currency": "THB", "amount": 100 } ] }
  ]
}
```
Error codes: `402303`, `404305`

### DW2 · POST /api/v1/bo/pr/deposit-withdraw/create

Request (บังคับทุก field):
```json
{
  "request_id": "6f1c2a9e-8b1d-4c1e-9a55-2f0d6b7c1a10",
  "role": "MEMBER",
  "user_id": 501,
  "currency": "THB",
  "credit": 100,
  "transaction_type": "DEPOSIT",
  "passcode": "123456"
}
```

| Field | Rule |
|---|---|
| `request_id` | UUID · ใหม่ทุกครั้งที่กด (DW-24) |
| `role` | `COMPANY` / `SHAREHOLDER` / `AGENT` / `MEMBER` (DW-02) |
| `user_id` | > 0 |
| `currency` | สกุลของลูกและของแม่ (DW-11) |
| `credit` | > 0 · ทศนิยม ≤ 4 ตำแหน่ง · ตัวเลขเท่านั้น (ข้อความ = `400`) (DW-10) |
| `transaction_type` | `DEPOSIT` / `WITHDRAW` (ตัวพิมพ์ใหญ่เท่านั้น) |
| `passcode` | passcode ของคนกด (DW-07) |

Response `data` (share01 ฝาก 100 ให้ mem01):
```json
{
  "id": 9001,
  "transaction_type": "DEPOSIT",
  "role": "MEMBER",
  "user_id": 501,
  "username": "mem01",
  "currency": "THB",
  "credit": 100,
  "parent_balance_after": 4900,
  "target_balance_after": 200,
  "created_at": "2026-10-12T10:00:00+07:00"
}
```
- `parent_balance_after` = `null` เมื่อแม่เป็น Superadmin (DW-12 / 14)
- ส่งซ้ำด้วย `request_id` เดิม + ค่าเดิม = `data` เดิม (DW-24)

Error codes: `422`, `402303`, `404301`, `404302`, `404303`, `404304`, `404305`, `404306`, `404401` + error ของ passcode (`401204`, `401205`, `401304`)

**ลำดับการเช็ค:** รูปแบบ (`422`) → passcode (middleware) → ADMIN (`404305`) → หาลูก (`404401`) → Seamless (`404302`) → สถานะลูก (`404301`) → สกุล (`404303`) → `request_id` (`422` / ผลเดิม) → lock + ยอด (`404304` / `404306`)

## 6. Schema

migration ใหม่ (`make migration name=deposit_withdraw_log`):

```sql
CREATE TABLE log_deposit_withdraw (
    id                    BIGSERIAL        PRIMARY KEY,
    request_id            VARCHAR(64)      NOT NULL,
    transaction_type      VARCHAR(10)      NOT NULL,                       -- DEPOSIT / WITHDRAW
    actor_type            VARCHAR(10)      NOT NULL,                       -- AGENT / SUB (คนกดจริง)
    actor_id              BIGINT           NOT NULL,                       -- user_agents.id หรือ subaccounts.id
    actor_username        VARCHAR(71)      NOT NULL,                       -- sub = comp01@staff
    parent_id             BIGINT           NOT NULL REFERENCES user_agents(id), -- แม่ (sub → เจ้าของ)
    parent_username       VARCHAR(71)      NOT NULL,
    target_type           VARCHAR(10)      NOT NULL,                       -- AGENT / MEMBER
    target_id             BIGINT           NOT NULL,
    target_username       VARCHAR(71)      NOT NULL,
    currency              VARCHAR(4)       NOT NULL,
    amount                DOUBLE PRECISION NOT NULL,
    parent_balance_before DOUBLE PRECISION,                                -- NULL = Superadmin (ไม่จำกัด)
    parent_balance_after  DOUBLE PRECISION,
    target_balance_before DOUBLE PRECISION NOT NULL,
    target_balance_after  DOUBLE PRECISION NOT NULL,
    ip                    VARCHAR(45),
    created_at            TIMESTAMPTZ      NOT NULL DEFAULT now(),
    CONSTRAINT uq_log_deposit_withdraw_request UNIQUE (request_id),        -- DW-24 · กฎข้อ 12
    CONSTRAINT ck_log_deposit_withdraw_type   CHECK (transaction_type IN ('DEPOSIT', 'WITHDRAW')),
    CONSTRAINT ck_log_deposit_withdraw_actor  CHECK (actor_type IN ('AGENT', 'SUB')),
    CONSTRAINT ck_log_deposit_withdraw_target CHECK (target_type IN ('AGENT', 'MEMBER')),
    CONSTRAINT ck_log_deposit_withdraw_amount CHECK (amount > 0)
);
CREATE INDEX idx_log_deposit_withdraw_parent ON log_deposit_withdraw(parent_id, created_at DESC);              -- ประวัติของแม่ (เส้นอนาคต)
CREATE INDEX idx_log_deposit_withdraw_target ON log_deposit_withdraw(target_type, target_id, created_at DESC); -- ประวัติของลูก
```

- เก็บ username เป็น snapshot ในแถว → รายงานในอนาคตไม่ต้อง JOIN 3 ตาราง
- `target_id` ไม่มี FK เพราะชี้ได้ 2 ตาราง (แยกด้วย `target_type`)
- ไม่เก็บ passcode หรือข้อมูลลับ
- ไม่แก้ตารางเดิม: `balance_ledger.reason` (VARCHAR(30)) / `ref_type` (VARCHAR(10)) รองรับค่าใหม่อยู่แล้ว · เพิ่ม enum ใน `app/models`: `DEPOSIT_OUT` · `DEPOSIT_IN` · `WITHDRAW_OUT` · `WITHDRAW_IN`
- index ของ DW1 มีแล้ว: `idx_user_agents_parent_username` · `idx_user_members_agent_username`

## 7. Test cases

| Rule | Input | Expected |
|---|---|---|
| DW-01 | share01 ฝากให้ agent01 · ให้ Member ของ agent01 (หลาน) · id ที่ไม่มี · บัญชีสายอื่น | สำเร็จ · `404401` · `404401` · `404401` |
| DW-02 | `role: "MEMBER", user_id: 15` (id 15 เป็น agent) · `role: "COMPANY"` กับ agent01 · `role: "member"` | `404401` · `404401` · `422` |
| DW-03 | ADMIN เรียก DW1 / DW2 | `404305` |
| DW-04 | sub มี `payment` view เรียก DW1 · เรียก DW2 · sub มีแค่ `member` edit | สำเร็จ · `402303` · `402303` |
| DW-05 | ฝากให้ลูก `SUSPENDED` · `LOCKED` | `404301` ทั้งคู่ · ยอดไม่เปลี่ยน |
| DW-06 | Superadmin ฝากให้ Company Seamless · Share ในสาย Seamless ฝากให้ Agent | `404302` |
| DW-07 | ไม่ส่ง passcode · passcode ผิด | `422` · `401204` |
| DW-10 | `credit` `0` · `-5` · `1.00001` · `"100"` · `25.5` | `422` · `422` · `422` · `400` · สำเร็จ ยอด 25.5 |
| DW-11 | Share B2B (THB, USD) ฝาก USD ให้ Agent THB · ฝาก JPY | `404303` · `404303` |
| DW-12 | Company THB 50,000 ฝาก 10,000 ให้ Share | Company 40,000 · Share +10,000 · ledger 2 แถว · log 1 แถว |
| DW-12 | Company THB 50,000 ฝาก 60,000 | `404304` · ไม่มี log / ledger |
| DW-12 | Superadmin ฝาก 1,000,000 ให้ Company | Company +1,000,000 · ledger 1 แถว (ฝั่ง Company) · log `parent_balance_*` = NULL |
| DW-13 | Member 100 ถอน 100 · ถอน 100.0001 | สำเร็จ `credit` = 0 · `404306` |
| DW-14 | Superadmin ถอน 500 จาก Company | Company −500 · ledger 1 แถว |
| DW-10 | ยอด 0.1 ฝาก 0.2 | ยอด = 0.3 (ไม่ใช่ 0.30000000000000004) |
| DW-22 | แม่ไม่มีแถวยอด USD ถอน USD จากลูก | สร้างแถว USD ให้แม่ · ยอด = ค่าที่ถอน |
| DW-23 | sub `comp01@staff` ฝากให้ share01 | ledger `actor_type = SUB` · log `actor_username = comp01@staff` · `parent_id = 10` |
| DW-24 | ส่งซ้ำ `request_id` เดิม ค่าเดิม · ค่า `credit` ต่าง · บัญชีอื่นใช้ `request_id` นี้ · `request_id` ที่ใช้สร้างบัญชีแล้ว | ผลเดิม ไม่ย้ายเงินซ้ำ · `422` · `422` · `422` |
| DW-25 | ยิงพร้อมกัน 2 คำขอ `request_id` เดียวกัน (integration) | ย้ายเงินครั้งเดียว · log 1 แถว · ทั้งคู่ได้ `data` เดียวกัน |
| DW-21 | ฝากพร้อมกัน 2 คำขอ ยอดพอแค่ 1 คำขอ (integration) | สำเร็จ 1 · อีกอัน `404304` · ยอดไม่ติดลบ |
| DW-21 | share01 ฝากให้ agent01 พร้อมกับถอนจาก agent01 (ทิศเงินสวนกัน · integration) | ไม่ deadlock · ยอดสุดท้ายถูกต้อง |
| DW-30 | share01 มีลูก agent 1 + Member 1 + ADMIN อยู่ในระบบ | ได้ 2 แถว เรียง A→Z · ไม่มี ADMIN · ไม่มีหลาน |
| DW-32 | agent ที่มี 2 สกุลแต่มีแถวยอดแค่ THB | `balances` 2 รายการ USD = 0 |
| DW-33 | Company Seamless ใน list ของ Superadmin | `has_balance = false` |

## 8. Contract changes (สำหรับแจ้ง frontend)

ไม่มี breaking change — เป็นเส้นใหม่ทั้ง 2 เส้น

- หน้าฝากถอนใช้ DW1 แทน `downlines/list` ได้ · ส่ง `role` + `id` + สกุลจากแถวไปที่ DW2 ตรงๆ
- DW2 ต้องสร้าง `request_id` ใหม่ทุกครั้งที่กด (กดซ้ำเพราะเน็ตหลุด = ใช้ค่าเดิม) และส่ง `passcode`
- ปิดปุ่มเมื่อ `has_balance = false` หรือ `status` ≠ `ACTIVE`
