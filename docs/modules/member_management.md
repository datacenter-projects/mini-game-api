# การจัดการ Member (`member_management`) — Spec

- สถานะ: **APPROVED** — กฎอนุมัติพร้อม spec `agent_management` (lead (zerph) 2026-10-07 · แก้ 2026-10-08) · แยก module ในโค้ด lead อนุมัติ 2026-10-08 · แยกเอกสารออกจาก [agent_management.md](agent_management.md) 2026-10-09
- ผู้ดูแล: maofoy
- ชื่อ module ในโค้ด: `member_management` (`controllers/member_management`, `dto/member_management`, `service/member_management`) · กฎ business ร่วมอยู่ที่ `core/agent_management`
- rule ID ใช้ `MGMT-xx` ชุดเดียวกับ [agent_management.md](agent_management.md) (ไม่ตั้งเลขใหม่ — โค้ดและ test อ้างเลขเดิม) · error ใช้ `402xxx` ของ module `agent_management`
- เมนู: 2 การจัดการสมาชิก — ส่วนของ Member (สร้าง · รายละเอียด · แก้ข้อมูล · แก้สถานะ · แก้ Commission)

> เอกสารนี้คือ source of truth ของเส้น Member — กฎที่ใช้ร่วมกับฝั่ง agent อยู่ใน agent_management.md และอ้างด้วย ID

## 1. หน้าที่ของ module

สร้างและจัดการ **Member** (ผู้เล่น — ตาราง `user_members` แยกจากฝั่ง agent): สร้าง, ดูรายละเอียด, แก้ชื่อ / เบอร์, แก้สถานะ, แก้ Commission

**ไม่อยู่ใน module นี้:** รายชื่อ / ค้นหา Member (อยู่ในเส้นรายชื่อดาวน์ไลน์ของ `agent_management` — MGMT-26 – MGMT-28 · Member ขึ้นปนกับฝั่ง agent ด้วย `role = MEMBER`) ·
ADMIN ค้นหาบัญชี (MGMT-27B) · การเติม / ถอนเงินหลังสร้าง (module การชำระเงิน) · login ของ Member (module หน้าบ้าน)

## 2. Business rules ที่ใช้กับ Member

กฎเต็มอยู่ใน [agent_management.md](agent_management.md) หัวข้อ 3 — ตารางนี้สรุปส่วนที่เกี่ยวกับ Member

| ID | สรุปส่วนของ Member |
|---|---|
| MGMT-02 | ผู้สร้าง Member ได้: Company Seamless 1 to 1 · Share B2C / Reseller / Master · Agent (Share B2B สร้างไม่ได้ — `402301`) |
| MGMT-03 | ผู้สร้างต้อง `ACTIVE` · Member ใหม่ `status = ACTIVE` |
| MGMT-05 | username 3–32 ตัว `a-z` `0-9` (ส่งตัวใหญ่ได้ เก็บตัวเล็ก) · **ห้ามซ้ำทั้งระบบ** ข้ามตาราง `user_agents` และ `user_members` (`402401`) |
| MGMT-06 | รหัสผ่านตาม AUTH-36 |
| MGMT-07 | ชื่อบังคับ 3–32 ตัวอักษร ไทย / อังกฤษ / ตัวเลข ไม่มีช่องว่าง |
| MGMT-08 | เบอร์โทรไม่บังคับ (`""` = ไม่ตั้ง) ตัวเลข 8–15 ตัว · ห้ามซ้ำภายใน `user_members` (`402403`) |
| MGMT-09A | แก้ได้: ชื่อ · เบอร์ · สถานะ · Commission (คนละเส้น) · แก้ไม่ได้: username · สกุลเงิน · ผู้สร้าง |
| MGMT-13 | สกุลเงิน = สกุลของผู้สร้าง (1 สกุล) ไม่ต้องส่ง |
| MGMT-15A | ยอดเงินตั้งต้น (`balance` ไม่บังคับ · ฝั่ง Transfer เท่านั้น — สายที่เป็น Seamless ส่งแล้วได้ `422`) โอนจากผู้สร้าง · ต้องมี `request_id` (UUID) กันส่งซ้ำ · ผู้สร้างยอดไม่พอ `402312` |
| MGMT-21 | Member มีแค่ `commission_percent` ต่อระบบ (ไม่ถือ PT) · 0–1% ทีละ 0.1% (`402309`) · หลังบ้านเก็บต่อเกม |
| MGMT-29 | รายละเอียด: ทุก field ยกเว้น password / hash · ไม่มี `passcode_set` · `pt` มีแค่ `commission_percent` + `created_*` / `updated_*` |
| MGMT-30 | สถานะ `ACTIVE` / `SUSPENDED` / `LOCKED` แก้ได้เฉพาะผู้สร้างโดยตรง (`402304`) · ตั้งกลับ `ACTIVE` ได้ · ไม่ต้อง passcode · `status` ที่แสดง = สถานะที่ใช้งานจริง (หัวสายถูกระงับ = `SUSPENDED`) |
| MGMT-51 | สิทธิ์ของ sub: สร้าง = `member` edit + `pt` edit (+ `payment` edit ถ้าส่ง `balance`) · ดู = `member` view · แก้ข้อมูล / สถานะ = `member` edit · แก้ Commission = `pt` edit |
| MGMT-60 | ทุกการสร้าง / แก้ บันทึก `account_change_logs` (`target_type = MEMBER`) |

## 3. Endpoints

ใช้แค่ `POST` · ระบุ Member ด้วย `id` ใน body · ผู้เรียกมาจาก token · ห้ามส่ง `null` (`422`)
error ร่วม: `401202`, `401203`, `401301`, `401302`, `401304`, `401306`, `401307`, `401311`, `402303`

| Method | Path | สิทธิ์ (sub) | หมายเหตุ |
|---|---|---|---|
| POST | `/api/v1/bo/pr/manage/members/create` | `member` edit · `pt` edit (+ `payment` edit ถ้าส่ง `balance`) | สร้าง Member |
| POST | `/api/v1/bo/pr/manage/members/detail` | `member` view | รายละเอียด Member |
| POST | `/api/v1/bo/pr/manage/members/update-info` | `member` edit | แก้ชื่อ · เบอร์โทร |
| POST | `/api/v1/bo/pr/manage/members/update-status` | `member` edit | แก้สถานะ |
| POST | `/api/v1/bo/pr/manage/members/update-commission` | `pt` edit | แก้ Commission |

### POST /api/v1/bo/pr/manage/members/create

Request:
```json
{
  "request_id": "0b9c5a77-2d1e-4f3a-8c61-5e4d3c2b1a09",
  "username": "Mem01",
  "password": "aA4b4c4d4e4f",
  "name": "ใจดี",
  "phone": "",
  "balance": { "THB": 100.00 },
  "pt": { "minigame": { "commission_percent": 0.3 } }
}
```
Response:
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": { "id": 501, "username": "mem01" }
}
```
- ไม่มี `user_type` · `currencies` · `status_game` · สกุลเงิน = สกุลของผู้สร้าง (MGMT-13)
- `pt` กลุ่มละ `commission_percent` เท่านั้น (MGMT-21)

Error codes: `422`, `402301`, `402303`, `402309`, `402312`, `402401`, `402403`

### POST /api/v1/bo/pr/manage/members/detail

Request:
```json
{ "id": 501 }
```
Response:
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "id": 501,
    "role": "MEMBER",
    "user_type": "MEMBER",
    "username": "mem01",
    "name": "ใจดี",
    "phone": "",
    "status": "ACTIVE",
    "parent_username": "agent01",
    "currencies": ["THB"],
    "balances": [ { "currency": "THB", "amount": 100.00 } ],
    "pt": {
      "minigame": { "commission_percent": 0.3, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "agent01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "agent01" }
    },
    "last_login_at": "",
    "last_login_ip": "",
    "created_at": "2026-10-02T14:30:00+07:00"
  }
}
```
- ดูได้: ผู้สร้าง Member และชั้นบนในสาย · ไม่มี `passcode_set` · `pt` ต่อเกมมีแค่ `commission_percent` · `created_at` · `created_by` · `updated_at` · `updated_by`

Error codes: `422`, `402303`, `402402`

### POST /api/v1/bo/pr/manage/members/update-info

Request (แทนทั้งชุด — ต้องส่งครบ):
```json
{ "id": 501, "name": "ใจดี", "phone": "0822222222" }
```
Response:
```json
{ "code": 200, "msg": "สำเร็จ" }
```
- `id` = Member ที่จะแก้ · แก้ได้เฉพาะผู้สร้างโดยตรง (MGMT-30 / `402304`) · `phone` `""` = ไม่ตั้ง · ส่ง `null` หรือไม่ส่ง field = `422`

Error codes: `422`, `402303`, `402304`, `402402`, `402403`

### POST /api/v1/bo/pr/manage/members/update-status

Request:
```json
{ "id": 501, "status": "LOCKED" }
```
Response:
```json
{ "code": 200, "msg": "สำเร็จ" }
```
- `status`: `ACTIVE` / `SUSPENDED` / `LOCKED` · ตั้งกลับ `ACTIVE` ได้ · แก้ได้เฉพาะผู้สร้างโดยตรง · ไม่ต้อง passcode (MGMT-30)

Error codes: `422`, `402303`, `402304`, `402402`

### POST /api/v1/bo/pr/manage/members/update-commission

Request:
```json
{ "id": 501, "pt": { "minigame": { "commission_percent": 0.3 } } }
```
Response:
```json
{ "code": 200, "msg": "สำเร็จ" }
```
- Member มีแค่ `commission_percent` (MGMT-21) · ส่ง field อื่นในกลุ่ม = `422`

Error codes: `422`, `402303`, `402304`, `402309`, `402402`

## 4. Schema

ตารางของ Member (เปลี่ยนชื่อจาก `members` / `member_game_settings` / `member_balances` ด้วย migration `20261009000000_member_tables_rename` — 2026-10-08) ·
`user_member_balances` ใช้ร่วมกับ `balance_ledger` (`owner_type = MEMBER`) ซึ่งอยู่ใน agent_management.md หัวข้อ 6

```sql
CREATE TABLE user_members (                     -- เปลี่ยนชื่อจาก members 2026-10-08
    id            BIGSERIAL    PRIMARY KEY,
    agent_id      BIGINT       NOT NULL REFERENCES user_agents(id),  -- ผู้สร้าง
    username      VARCHAR(32)  NOT NULL,
    password_hash VARCHAR(100) NOT NULL,
    name          VARCHAR(32)  NOT NULL,
    phone         VARCHAR(15),
    currency      VARCHAR(4)   NOT NULL,
    status        VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','SUSPENDED','LOCKED')),
    last_login_at TIMESTAMPTZ,
    last_login_ip VARCHAR(45),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_user_members_username UNIQUE (username),
    CONSTRAINT ck_user_members_username CHECK (username = lower(username) AND username ~ '^[a-z0-9]{3,32}$'),
    CONSTRAINT ck_user_members_phone CHECK (phone IS NULL OR phone ~ '^[0-9]{8,15}$')
);
CREATE INDEX idx_user_members_agent_username ON user_members(agent_id, username);
CREATE UNIQUE INDEX uq_user_members_phone ON user_members(phone) WHERE phone IS NOT NULL;

CREATE TABLE user_member_game_settings (        -- Member มีแค่ Commission ต่อเกม (MGMT-21) · เปลี่ยนชื่อจาก member_game_settings 2026-10-08
    user_member_id BIGINT      NOT NULL REFERENCES user_members(id),
    category      VARCHAR(30) NOT NULL,
    game_code     VARCHAR(50) NOT NULL,
    commission_bp INT         NOT NULL DEFAULT 0 CHECK (commission_bp BETWEEN 0 AND 100),
    created_by    VARCHAR(71) NOT NULL DEFAULT '',   -- username ผู้สร้าง (migration 20261008120000)
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by    VARCHAR(71) NOT NULL DEFAULT '',   -- username คนที่แก้ Commission ล่าสุด
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_member_id, game_code)
);

CREATE TABLE user_member_balances (             -- เปลี่ยนชื่อจาก member_balances 2026-10-08
    user_member_id BIGINT      NOT NULL REFERENCES user_members(id),
    currency   VARCHAR(4)  NOT NULL,
    amount     BIGINT      NOT NULL DEFAULT 0 CHECK (amount >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_member_id, currency)
);
```

## 5. Test cases

test ของเส้น Member อยู่ในชุด test ของ agent_management (กฎเดียวกัน) — แถวที่เกี่ยวกับ Member:

| Rule ID | Input | Expected |
|---|---|---|
| MGMT-02 | Share B2B สร้าง Member | `402301` |
| MGMT-02 | Share B2C / Agent สร้าง Member | สำเร็จ |
| MGMT-05 | สร้าง Member ชื่อเดียวกับ Agent ที่มีอยู่ | `402401` |
| MGMT-13 | สร้าง Member | `currencies` = สกุลของผู้สร้าง |
| MGMT-15A | สร้าง Member พร้อม `balance` · ส่งซ้ำด้วย `request_id` เดิม | ผู้สร้างยอดลด · Member ได้ยอด · ledger 2 แถว · ส่งซ้ำได้ `id` เดิม ไม่โอนซ้ำ |
| MGMT-21 | Commission 0.3 · 1.1 · ส่ง `pt_from_parent` มาด้วย | สำเร็จ · `402309` · `422` |
| MGMT-29 | ผู้สร้าง / ชั้นบนดูรายละเอียด · บัญชีสายอื่นดู | สำเร็จ ไม่มี `passcode_set` · `402402` |
| MGMT-30 | ผู้สร้างตั้ง `LOCKED` แล้วกลับ `ACTIVE` · ชั้นบนที่ไม่ใช่ผู้สร้างแก้ | สำเร็จ · `402304` |
| MGMT-30 | หัวสายถูกระงับ | รายละเอียด Member แสดง `status = SUSPENDED` |
| MGMT-60 | แก้ข้อมูล / Commission | มี `account_change_logs` `target_type = MEMBER` ค่าเก่า / ใหม่ |

## 6. Error codes

ใช้ `402xxx` ของ module `agent_management` (agent_management.md หัวข้อ 9) — ที่เส้น Member ตอบได้:

| Code | ความหมาย |
|---|---|
| 402301 | สร้างบัญชีประเภทนี้ไม่ได้ |
| 402303 | ไม่มีสิทธิ์ใช้งานเมนูนี้ |
| 402304 | แก้ไขได้เฉพาะผู้สร้างของบัญชีนี้ |
| 402309 | Commission เกินที่กำหนด |
| 402312 | ยอดเงินของคุณไม่พอสำหรับยอดเงินตั้งต้น |
| 402401 | username นี้ถูกใช้แล้ว |
| 402402 | ไม่พบบัญชีในสายของคุณ |
| 402403 | เบอร์โทรนี้ถูกใช้แล้ว |
