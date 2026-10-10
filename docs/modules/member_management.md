# การจัดการ Member (`member_management`) — Spec

- สถานะ: **APPROVED** · กฎอนุมัติพร้อม spec `agent_management` (lead (zerph) 2026-10-07) · แยก module ในโค้ด 2026-10-08 · แยกเอกสาร 2026-10-09
- **แก้ตาม lead review 2026-10-09** ([docs/reviews/member-management.md](../reviews/member-management.md) — MA1–MA6 · M1–M5 · MN1–MN4 · M3-1 · MQ1–MQ7):
  lead อนุมัติ MA1 (agent ถือ PT สู้กับ Member แต่ละคน) · MA2 (`remain_quota` ระบบคิดและ sync) · MA3 (`pt` + `commission_percent`) · MA4 (`user_members.cnf`) ·
  MA5 (ยอดที่ `user_members.credit` + DROP `user_member_balances`) · MA6 (เลข migration) · โครงเส้นใหม่ M1–M5 (หัวข้อ 3) · เบอร์ 2 field ห้ามซ้ำภายใต้ผู้สร้างเดียวกัน (MQ1)
- `credit` / % / ledger เป็นทศนิยม float ปัด 4 ตำแหน่ง (CLAUDE.md กฎข้อ 9 — lead อนุมัติ 2026-10-09)
- ผู้ดูแล: maofoy · งานตาม review รอบนี้ boiledegg ทำแทน (2026-10-11 · branch `boiledegg/bo/member_management`)
- ชื่อ module ในโค้ด: `member_management` (`controllers/member_management`, `dto/member_management`, `service/member_management`) · กฎ business ร่วมอยู่ที่ `core/agent_management`
- rule ID ใช้ `MGMT-xx` ชุดเดียวกับ [agent_management.md](agent_management.md) (ไม่ตั้งเลขใหม่ — โค้ดและ test อ้างเลขเดิม) · error ใช้ `402xxx` ของ module `agent_management`
- เมนู: 2 การจัดการสมาชิก — ส่วนของ Member (สร้าง · รายละเอียด · แก้ข้อมูล / PT · แก้สถานะ · dropdown Member ลูกตรง)

> เอกสารนี้คือ source of truth ของเส้น Member — กฎที่ใช้ร่วมกับฝั่ง agent อยู่ใน agent_management.md และอ้างด้วย ID

## 1. หน้าที่ของ module

สร้างและจัดการ **Member** (ผู้เล่น — ตาราง `user_members` แยกจากฝั่ง agent): สร้าง · ดูรายละเอียด · แก้ชื่อ / เบอร์ / PT / Commission · แก้สถานะ · รายชื่อ Member ลูกตรง (dropdown)

**ไม่อยู่ใน module นี้:** รายชื่อ / ค้นหา Member ทั้งสาย (`downlines/list` ของ `agent_management` — MGMT-26 – MGMT-28 · Member ขึ้นปนกับฝั่ง agent ด้วย `role = MEMBER`) ·
ADMIN ค้นหาบัญชี (MGMT-27B) · การเติม / ถอนเงินหลังสร้าง (module การชำระเงิน) · login ของ Member (module Player Auth)

## 2. Business rules ที่ใช้กับ Member

กฎเต็มอยู่ใน [agent_management.md](agent_management.md) หัวข้อ 3 — ตารางนี้สรุปส่วนที่เกี่ยวกับ Member

| ID | สรุปส่วนของ Member |
|---|---|
| MGMT-02 | ผู้สร้าง Member ได้: Company Seamless 1 to 1 · Share B2C · **Share Reseller · Share Master** (สร้างได้แค่ Member — lead H1) · Agent · Share B2B สร้างไม่ได้ (`402301`) |
| MGMT-03 | ผู้สร้างต้อง `ACTIVE` · Member ใหม่ `status = ACTIVE` |
| MGMT-05 | username 3–32 ตัว `a-z` `0-9` (ส่งตัวใหญ่ได้ เก็บตัวเล็ก) · **ห้ามซ้ำทั้งระบบ** ข้ามตาราง `user_agents` และ `user_members` (`402401`) |
| MGMT-06 | รหัสผ่านตาม AUTH-36 |
| MGMT-07 | ชื่อบังคับ 3–32 ตัวอักษร ไทย / อังกฤษ / ตัวเลข ไม่มีช่องว่าง |
| MGMT-08 | **เบอร์ 2 field** (lead E1-5 · MQ1): `phone_country_code` = รหัสโทรออกที่มีจริง 1–3 หลัก ไม่มี `+` · `phone` = ตัวเลขล้วน ไม่มี 0 นำหน้า · รวมไม่เกิน 15 หลัก · ไม่กรอก = `""` ทั้งคู่ · ส่งค่าเดียว = `422` · **ห้ามซ้ำที่คู่ (code + phone) เฉพาะภายใต้ผู้สร้างเดียวกัน** (`402403`) — Member คนเดียวกันอาจเล่นหลายเว็บ / หลายสาย ใช้เบอร์เดียวกันได้ถ้าคนละผู้สร้าง · Member กับ agent ใช้เบอร์เดียวกันได้ |
| MGMT-08A | migration เบอร์เดิมของ Member (lead B1 · แบบเดียวกับ agent): ขึ้นต้น `0` → `66` + ตัด 0 · ขึ้นต้น `66` และยาว 11 หลัก → `66` + ส่วนที่เหลือ · แบบอื่นล้างเป็น `""` · แปลงแล้วซ้ำภายใต้ผู้สร้างเดียวกัน เก็บแถว `id` น้อยสุด แถวอื่นล้าง · ทุกรายการที่ถูกล้างบันทึกใน `migration_phone_cleared` (`table_name = user_members` · `reason` `FORMAT` / `DUPLICATE` · `kept_member_id` = Member ที่ได้เก็บเบอร์ไว้) |
| MGMT-09A | แก้ได้: ชื่อ · เบอร์ · PT / Commission (เส้นเดียว `members/detail/update` — M3) · สถานะ (`members/status/update` — M4) · แก้ไม่ได้: username · สกุลเงิน · ผู้สร้าง |
| MGMT-13 | สกุลเงิน = สกุลของผู้สร้าง (1 สกุล) ไม่ต้องส่ง |
| MGMT-15A | ยอดเงินตั้งต้น `balance` ไม่บังคับ · ฝั่ง Transfer เท่านั้น — **สายที่เป็น Seamless ส่งแล้วได้ `422`** · โอนจากผู้สร้าง (ยอดไม่พอ `402312`) · ต้องมี `request_id` (UUID) กันส่งซ้ำ — `request_id` เดิมของตัวเองที่ใช้สร้าง Member = ได้ `id` เดิม ไม่โอนซ้ำ · ของคนอื่น / ที่เคยใช้สร้างฝั่ง agent = `422` (lead Q-R4) · ยอดของ Member เก็บที่ `user_members.credit` (Member มี 1 สกุล) · ทุกการแก้ `credit` ต้อง lock แถว `user_members` `FOR UPDATE` ก่อน (lead MA5) · `balance_ledger` `owner_type = MEMBER` |
| MGMT-15B | Member ฝั่ง Seamless ไม่มียอด: `balances` แสดง `amount: 0` เสมอ (ACC-19 · lead MQ6) — หน้าบ้านซ่อนเอง |
| MGMT-61 | `user_members.cnf` = สายชั้นบนของ Member `{"parent": [{"id", "position"}, …]}` เรียงจาก Superadmin ลงมาถึง**ผู้สร้าง** (= `cnf` ของผู้สร้าง + ผู้สร้าง) · ตั้งตอนสร้าง (lead MA4) |
| MGMT-62 | `user_member_game_settings.parent_id` = id ของ agent ผู้สร้าง Member (ค่าเดียวกับ `user_members.agent_id`) · ตั้งตอนสร้าง · NOT NULL |
| MGMT-16 | `created_by` / `updated_by` ของค่า PT = username ของคนกดจริง (sub = `comp01@staff` — lead Q-R2) |
| MGMT-21 | ผู้สร้างตั้งให้ Member ต่อระบบ 2 ค่า (lead MA1 / MA3): `pt` = ผู้สร้าง**ถือสู้กับ Member คนนี้** (0 ถึงค่าที่ผู้สร้างได้รับ) · `commission_percent` 0–1% · Member แต่ละคน `pt` ต่างกันได้ · ส่ง `pt_from_parent` / `force` / `remain_quota` / `status` = `422` |
| | • ขั้น (lead MQ3): `pt` ทีละ 0.5 · commission ทีละ 0.1 · ไม่ลงขั้นหรือทศนิยมเกิน 4 ตำแหน่ง = `422` · ลงขั้นแต่เกินเพดาน = `402305` (`pt` เกินค่าที่ผู้สร้างได้รับ) / `402309` (commission เกิน 1) |
| | • `remain_quota` = ค่าที่ผู้สร้างได้รับ − `pt` · **ระบบคิดและเก็บ** ต่อเกม (lead MA2) · คิดใหม่ทุกครั้งที่สร้าง / แก้ `pt` · ความหมายของ Remain ตอน settle ตัดสินพร้อม module Settle |
| | • ค่าที่ผู้สร้างได้รับเปลี่ยน → sync `remain_quota` ใน tx เดียวกันทุกทาง: ชั้นบนแก้ค่าที่ให้ agent (`agents/detail/update` — agent_management MGMT-24 R1–R3) · Superadmin แก้ CSM แล้ว Share Master ตาม (MGMT-19) · ลดต่ำกว่า `pt` ของ Member ไม่ได้ (`402306`) |
| | • Share Master ตั้ง `pt` ของ Member เองต่อ Member (lead H4) |
| | • ผู้สร้างไม่มีแถว PT ของกลุ่มนั้น (ข้อมูลก่อน module ②) → ค่าที่ได้รับ = 0 → ตั้งได้แค่ `pt` 0 |
| MGMT-29 | รายละเอียด: ทุก field ยกเว้น password / hash · ไม่มี `passcode_set` / `status_game` · **`pt` แสดงเสมอ** (ไม่มีเมนูสิทธิ์ `pt` แล้ว — lead P4) |
| MGMT-30 | สถานะ `ACTIVE` / `SUSPENDED` / `LOCKED` แก้ได้เฉพาะผู้สร้างโดยตรง (`402304`) · ตั้งกลับ `ACTIVE` ได้ · ไม่ต้อง passcode · **ส่งสถานะเดิม = `200` ไม่เขียน log** (lead MQ5) · `status` ที่แสดง = สถานะที่ใช้งานจริง (หัวสายถูกระงับ = `SUSPENDED`) |
| MGMT-30A | ผล `LOCKED` / `SUSPENDED` ของ Member ต่อ login / เดิมพัน: **open item** → กำหนดตอน review Player Auth (lead MQ7 · ร่าง: LOCKED = login ไม่ได้ + ตัด session · SUSPENDED = login / ดูได้ เดิมพันไม่ได้) |
| MGMT-32M | `members/detail/update` (lead M3-1 · MN2): section `info` / `pt` ไม่บังคับ แต่**ต้องมีอย่างน้อย 1** (`422` "ต้องส่งอย่างน้อย 1 section") · `info` ส่งบาง field ได้ (ไม่ส่ง = คงเดิม · เบอร์ต้องส่งคู่) · `pt.{กลุ่ม}` ต้องครบ `pt` + `commission_percent` · ผู้สร้างโดยตรงเท่านั้น (หลาน `402304` · นอกสาย `402402`) · tx เดียว (section ไหนผิดไม่บันทึกเลย) · log แยกแถว `UPDATE_INFO` / `UPDATE_PT` เฉพาะ section ที่เปลี่ยนจริง · ค่าเดิม = ไม่เขียน log · ไม่ต้อง passcode |
| MGMT-32N | Member ไม่มีแถวค่าตั้งของกลุ่มที่ส่ง (ข้อมูลก่อน module ②) → ข้ามกลุ่มนั้นเงียบ ไม่บันทึก ไม่เขียน log (lead MQ4) |
| MGMT-35M | `members/list` (lead M5): Member ที่เป็น**ลูกตรงของตัวเอง** ไว้ทำ dropdown "คัดลอกการตั้งค่าจาก" → เลือกแล้วเรียก `members/detail/get` · `keyword` ไม่บังคับ 4–32 ตัว (ค้น username บางส่วน ไม่สนตัวพิมพ์) · A→Z · ไม่แบ่งหน้า สูงสุด 500 · ไม่มี = `[]` |
| MGMT-51 | สิทธิ์ของ sub (lead MN4 · P4): ดู (`detail/get` · `list`) = `member` view · สร้าง / แก้ / สถานะ = `member` edit · ส่ง `balance` ต้องมี `payment` edit เพิ่ม (เช็คใน service หลังตรวจรูปแบบ — lead U3) · ไม่มีเมนู `pt` แล้ว |
| MGMT-60 | ทุกการสร้าง / แก้ บันทึก `account_change_logs` (`target_type = MEMBER`) · log ยอดตั้งต้น `INITIAL_BALANCE` ใช้ key `amounts` (lead E1-4) |

### 2.1 ลำดับเช็ค (lead U3 · MQ2 — แบบเดียวกับ agent_management 7.1)

ไล่บนลงล่างตามลำดับ field ใน body 2 รอบ · ตอบ error ของ field แรกที่ผิด · ค่าที่เป็น object (`pt`, `balance`) ไล่ key ตามตัวอักษร

1. **รูปแบบ** (DTO) → `422`
2. **กฎที่ต้องดู DB** — `members/create`: ผู้สร้างสร้าง Member ได้ไหม (`402301`) → `username` ซ้ำ (`402401`) → เบอร์ซ้ำ (`402403`) → `balance` (Seamless `422` · ยอดไม่พอ `402312`) → `pt` (`402305`) → `commission_percent` (`402309`)
   · `members/detail/update`: `info` (เบอร์ซ้ำ `402403`) → `pt` (`402305` → `402309`)
3. สิทธิ์ sub (`402303`) เช็คที่ middleware ก่อน · ข้อยกเว้น `payment` ตอนส่ง `balance` เช็คใน service หลังรอบรูปแบบ

## 3. Endpoints (lead MN1 · M1–M5)

ใช้แค่ `POST` · ระบุ Member ด้วย `id` ใน body · ผู้เรียกมาจาก token · ห้ามส่ง `null` (`422`) · path เดิม (`members/detail` · `update-info` · `update-status` · `update-pt`) ลบแล้ว ได้ `404`
error ร่วม: `401202`, `401203`, `401301`, `401302`, `401304`, `401306`, `401307`, `401311`, `402303`

| # | Path | สิทธิ์ (sub) | หน้าที่ |
|---|---|---|---|
| M1 | `/api/v1/bo/pr/manage/members/create` | `member` edit (+ `payment` edit ถ้าส่ง `balance`) | สร้าง Member |
| M2 | `/api/v1/bo/pr/manage/members/detail/get` | `member` view | รายละเอียด Member |
| M3 | `/api/v1/bo/pr/manage/members/detail/update` | `member` edit | แก้ชื่อ / เบอร์ (`info`) · PT / Commission (`pt`) |
| M4 | `/api/v1/bo/pr/manage/members/status/update` | `member` edit | แก้สถานะ |
| M5 | `/api/v1/bo/pr/manage/members/list` | `member` view | Member ลูกตรง (dropdown) |

### M1 · POST /api/v1/bo/pr/manage/members/create

Request:
```json
{
  "request_id": "0b9c5a77-2d1e-4f3a-8c61-5e4d3c2b1a09",
  "username": "Mem01",
  "password": "aA4b4c4d4e4f",
  "name": "ใจดี",
  "phone_country_code": "66",
  "phone": "812345678",
  "balance": { "THB": 100.00 },
  "pt": { "minigame": { "pt": 50, "commission_percent": 0.3 } }
}
```
Response:
```json
{ "code": 200, "msg": "สำเร็จ", "data": { "id": 501, "username": "mem01" } }
```

| field | กติกา |
|---|---|
| `request_id` | UUID บังคับ (MGMT-15A) |
| `username` · `password` · `name` | MGMT-05 – MGMT-07 |
| `phone_country_code` · `phone` | MGMT-08 · ไม่กรอก `""` ทั้งคู่ |
| `balance` | ไม่บังคับ · key = สกุลของผู้สร้าง · ฝั่ง Seamless ห้ามส่ง (MGMT-15A) |
| `pt.minigame` | `pt` + `commission_percent` บังคับทั้งคู่ · ต้องส่งครบทุกกลุ่ม (MGMT-21) |

- ไม่มี `user_type` · `currencies` · `status_game` · สกุลเงิน = สกุลของผู้สร้าง (MGMT-13)

Error codes: `422`, `402301`, `402303`, `402305`, `402309`, `402312`, `402401`, `402403`

### M2 · POST /api/v1/bo/pr/manage/members/detail/get

Request: `{ "id": 501 }`

Response `data`:
```json
{
  "id": 501, "role": "MEMBER", "user_type": "MEMBER", "username": "mem01", "name": "ใจดี",
  "phone_country_code": "66", "phone": "812345678",
  "status": "ACTIVE", "parent_username": "agent01", "currencies": ["THB"],
  "balances": [ { "currency": "THB", "amount": 100 } ],
  "pt": {
    "minigame": { "pt": 50, "remain_quota": 10, "commission_percent": 0.3, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "agent01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "agent01" }
  },
  "last_login_at": "", "last_login_ip": "", "created_at": "2026-10-02T14:30:00+07:00"
}
```
- ดูได้: ผู้สร้าง Member และชั้นบนในสาย (อื่น = `402402`) · `pt` แสดงเสมอ (MGMT-29) · ไม่มี `passcode_set` / `status_game` · ฝั่ง Seamless `amount` = 0 (MGMT-15B)

Error codes: `422`, `402303`, `402402`

### M3 · POST /api/v1/bo/pr/manage/members/detail/update

Request (ส่งเฉพาะ section ที่แก้ · อย่างน้อย 1 section):
```json
{
  "id": 501,
  "info": { "name": "ใจดี", "phone_country_code": "66", "phone": "822222222" },
  "pt": { "minigame": { "pt": 40, "commission_percent": 0.3 } }
}
```
Response: `{ "code": 200, "msg": "สำเร็จ" }`

| section | กติกา |
|---|---|
| `info` | `name` · `phone_country_code` + `phone` — ส่งบาง field ได้ (ไม่ส่ง = คงเดิม) · เบอร์ต้องส่ง 2 field คู่กัน · ล้างเบอร์ส่ง `""` ทั้งคู่ · `{}` = `422` |
| `pt` | ส่งเฉพาะกลุ่มที่แก้ · ในกลุ่มต้องครบ `pt` + `commission_percent` · `pt` ไม่เกินค่าที่ผู้สร้างได้รับ (`402305`) · `remain_quota` คิดใหม่ · ส่ง `pt_from_parent` / `force` / `remain_quota` / `status` = `422` |

- ไม่ส่งเลยสัก section (`{ "id": 501 }`) = `422` "ต้องส่งอย่างน้อย 1 section" · กติกาอื่น MGMT-32M / MGMT-32N

Error codes: `422`, `402303`, `402304`, `402305`, `402309`, `402402`, `402403`

### M4 · POST /api/v1/bo/pr/manage/members/status/update

Request: `{ "id": 501, "status": "LOCKED" }` · Response: `{ "code": 200, "msg": "สำเร็จ" }`
- `ACTIVE` / `SUSPENDED` / `LOCKED` · ผู้สร้างโดยตรงเท่านั้น · ส่งสถานะเดิม = `200` ไม่เขียน log (MGMT-30)

Error codes: `422`, `402303`, `402304`, `402402`

### M5 · POST /api/v1/bo/pr/manage/members/list

Request: `{ "keyword": "" }` (`{}` ได้)

Response `data`:
```json
[
  { "id": 501, "username": "mem01", "name": "ใจดี", "status": "ACTIVE" },
  { "id": 502, "username": "mem02", "name": "mem02", "status": "LOCKED" }
]
```
- Member ลูกตรงของตัวเอง · `status` = สถานะที่ใช้งานจริง · กติกา MGMT-35M

Error codes: `422`, `402303`

## 4. Schema

ตารางของ Member (เปลี่ยนชื่อจาก `members` / `member_game_settings` / `member_balances` — migration `20261009000000_member_tables_rename` · lead U4) ·
ยอดอยู่ที่ `user_members.credit` (DROP `user_member_balances` — migration `20261009120100`) · การเปลี่ยนยอดเขียน `balance_ledger` (`owner_type = MEMBER`) ซึ่งอยู่ใน agent_management.md หัวข้อ 6

```sql
CREATE TABLE user_members (
    id                 BIGSERIAL    PRIMARY KEY,
    agent_id           BIGINT       NOT NULL REFERENCES user_agents(id),  -- ผู้สร้าง
    username           VARCHAR(32)  NOT NULL,
    password_hash      VARCHAR(100) NOT NULL,
    name               VARCHAR(32)  NOT NULL,
    phone_country_code VARCHAR(3),                                         -- MGMT-08 · NULL คู่กับ phone
    phone              VARCHAR(15),                                        -- เบอร์ในประเทศ ไม่มี 0 นำหน้า
    currency           VARCHAR(4)   NOT NULL,
    credit             DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (credit >= 0),   -- ยอดเงิน หน่วยสกุล (MGMT-15A)
    cnf                JSONB        NOT NULL DEFAULT '{"parent": []}',            -- สายชั้นบน (MGMT-61)
    status             VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','SUSPENDED','LOCKED')),
    last_login_at      TIMESTAMPTZ,
    last_login_ip      VARCHAR(45),
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_user_members_username UNIQUE (username),
    CONSTRAINT ck_user_members_username CHECK (username = lower(username) AND username ~ '^[a-z0-9]{3,32}$'),
    CONSTRAINT ck_user_members_phone CHECK (
        (phone IS NULL AND phone_country_code IS NULL) OR
        (phone_country_code ~ '^[1-9][0-9]{0,2}$' AND phone ~ '^[1-9][0-9]*$' AND length(phone_country_code) + length(phone) <= 15))
);
CREATE INDEX idx_user_members_agent_username ON user_members(agent_id, username);
CREATE UNIQUE INDEX uq_user_members_phone ON user_members(agent_id, phone_country_code, phone) WHERE phone IS NOT NULL;  -- MQ1 แทน unique(phone) เดิม
CREATE INDEX idx_user_members_cnf ON user_members USING GIN (cnf jsonb_path_ops);

CREATE TABLE user_member_game_settings (        -- PT ที่ผู้สร้างถือสู้ + Commission ต่อเกม (MGMT-21)
    user_member_id BIGINT      NOT NULL REFERENCES user_members(id),
    parent_id      BIGINT      NOT NULL REFERENCES user_agents(id),   -- agent ผู้สร้าง Member (MGMT-62)
    category       VARCHAR(30) NOT NULL,
    game_code      VARCHAR(50) NOT NULL,
    pt             DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (pt BETWEEN 0 AND 100),      -- ผู้สร้างถือสู้กับ Member คนนี้
    remain         DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (remain BETWEEN 0 AND 100),  -- ค่าที่ผู้สร้างได้รับ − pt (ระบบคิด · sync ตาม MGMT-21)
    commission     DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (commission BETWEEN 0 AND 1),
    created_by     VARCHAR(71) NOT NULL DEFAULT '',   -- คนกดจริง (MGMT-16)
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by     VARCHAR(71) NOT NULL DEFAULT '',
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_member_id, game_code)
);
CREATE INDEX idx_user_member_game_settings_parent ON user_member_game_settings(parent_id, game_code);

-- migration 20261011120000_member_management_phone_country_code (MGMT-08 / 08A): ADD COLUMN phone_country_code · แปลงเบอร์เดิม · CHECK ใหม่ · unique (agent_id, code, phone) แทน uq_user_members_phone
--   migration_phone_cleared ADD COLUMN kept_member_id (Member ที่ได้เก็บเบอร์ไว้เมื่อ DUPLICATE)
```

## 5. Test cases

test ของเส้น Member อยู่ในชุด integration test ของ `app/internals/backoffice` (ไฟล์ `member_management_*_integration_test.go` และของ agent_management ที่ใช้ร่วม)

| Rule ID | Input | Expected |
|---|---|---|
| MGMT-02 | Share B2B สร้าง Member | `402301` |
| MGMT-02 | Share B2C / Agent / Share Reseller / Share Master สร้าง Member (H1) | สำเร็จ |
| MGMT-05 | สร้าง Member ชื่อเดียวกับ Agent ที่มีอยู่ | `402401` |
| MGMT-08 | `"66"` + `"812345678"` · ส่งค่าเดียว · `phone` ขึ้นต้น 0 · code ไม่มีจริง | สำเร็จ · `422` · `422` · `422` |
| MGMT-08 | เบอร์ซ้ำกับ Member อื่น**ของผู้สร้างเดียวกัน** · ซ้ำกับ Member ของผู้สร้างอื่น · ซ้ำกับ agent | `402403` · สำเร็จ · สำเร็จ |
| MGMT-08A | migration: `0812345678` ×2 ใต้ผู้สร้างเดียวกัน · `0812345678` ใต้ผู้สร้างอื่น · `12345678` | แถวแรก `66`/`812345678` แถวที่สอง `""` (DUPLICATE · `kept_member_id`) · `66`/`812345678` · `""` (FORMAT) |
| MGMT-13 | สร้าง Member | `currencies` = สกุลของผู้สร้าง |
| MGMT-15A | สร้างพร้อม `balance` · ส่งซ้ำด้วย `request_id` เดิม · `request_id` ของคนอื่น / ที่ใช้สร้าง agent | ผู้สร้างยอดลด · Member ได้ยอด · ledger 2 แถว · ได้ `id` เดิม ไม่โอนซ้ำ · `422` |
| MGMT-15A | ผู้สร้างฝั่ง Seamless ส่ง `balance` · `balance` 25.5 | `422` · `credit` = 25.5 · ledger 25.5 |
| MGMT-15A | log ยอดตั้งต้น | `INITIAL_BALANCE` `new_value` key `amounts` |
| MGMT-61 | Agent (ใต้ Share ใต้ Company) สร้าง Member | `cnf.parent` = superadmin → company → share → agent |
| MGMT-21 | Commission 0.3 · 1.1 · 0.15 · ส่ง `pt_from_parent` | สำเร็จ · `402309` · `422` · `422` |
| MGMT-21 | ผู้สร้างได้รับ 60: `pt` 50 · 60.5 · 30.25 · ไม่ส่ง `pt` · ส่ง `remain_quota` | สำเร็จ (`remain_quota` 10) · `402305` · `422` · `422` · `422` |
| MGMT-29 | ผู้สร้าง / ชั้นบนดู · sub `member` view ดู · บัญชีสายอื่นดู | สำเร็จ มี `pt` เสมอ ไม่มี `passcode_set` · สำเร็จ มี `pt` · `402402` |
| MGMT-30 | ผู้สร้างตั้ง `LOCKED` แล้วกลับ `ACTIVE` · ชั้นบนที่ไม่ใช่ผู้สร้างแก้ · ส่งสถานะเดิม | สำเร็จ · `402304` · `200` ไม่มี log เพิ่ม |
| MGMT-30 | หัวสายถูกระงับ | รายละเอียด Member แสดง `status = SUSPENDED` |
| MGMT-32M | `{ "id": 501 }` · `info: {}` · `info` ชื่อ + `pt` เปลี่ยน · ส่งค่าเดิมทั้งหมด · `pt` ผิดหลัง `info` ถูก | `422` · `422` · log 2 แถว `UPDATE_INFO` + `UPDATE_PT` · `200` ไม่มี log · `402305` ชื่อไม่เปลี่ยน |
| MGMT-32M | `pt` ผู้สร้างได้รับ 60: `pt` 60 · 60.5 · ชั้นบนที่ไม่ใช่ผู้สร้าง · ไม่ส่ง `commission_percent` | สำเร็จ (`remain_quota` 0) · `402305` · `402304` · `422` |
| MGMT-32N | Member ไม่มีแถวค่าตั้ง แก้ `pt` | `200` ไม่มี log |
| MGMT-35M | `members/list` · `keyword` 3 ตัว · ไม่มีลูก | Member ลูกตรง A→Z · `422` · `[]` |
| MGMT-51 | sub `member` view เรียก `detail/get` · `list` · `detail/update` | สำเร็จ · สำเร็จ · `402303` |
| MGMT-60 | สร้าง / แก้ข้อมูล / PT / สถานะ | `account_change_logs` `target_type = MEMBER` ค่าเก่า / ใหม่ |
| — | เรียก path เดิม (`members/detail` · `update-info` · `update-status` · `update-pt`) | `404` |

## 6. Error codes

ใช้ `402xxx` ของ module `agent_management` (agent_management.md หัวข้อ 9) — ที่เส้น Member ตอบได้:

| Code | ความหมาย |
|---|---|
| 402301 | สร้างบัญชีประเภทนี้ไม่ได้ |
| 402303 | ไม่มีสิทธิ์ใช้งานเมนูนี้ |
| 402304 | แก้ไขได้เฉพาะผู้สร้างของบัญชีนี้ |
| 402305 | `pt` ของ Member เกินค่าที่ผู้สร้างได้รับ |
| 402309 | Commission เกินที่กำหนด |
| 402312 | ยอดเงินของคุณไม่พอสำหรับยอดเงินตั้งต้น |
| 402401 | username นี้ถูกใช้แล้ว |
| 402402 | ไม่พบบัญชีในสายของคุณ |
| 402403 | เบอร์โทรนี้ถูกใช้แล้ว (ภายใต้ผู้สร้างเดียวกัน) |

## 7. Open items

| # | เรื่อง | ตัดสินเมื่อ |
|---|---|---|
| MQ7 | ผล `LOCKED` / `SUSPENDED` ของ Member ต่อ login / เดิมพัน (MGMT-30A) | review Player Auth |
| MA2 | ความหมายของ Remain ตอน settle | module Settle |

## 8. Contract changes (แจ้ง frontend · 2026-10-11)

- path ใหม่: `members/detail/get` · `members/detail/update` (รวม `update-info` + `update-pt`) · `members/status/update` · เส้นใหม่ `members/list` · path เดิมได้ `404`
- เบอร์ 2 field `phone_country_code` + `phone` ทุก request / response · เบอร์เดิมแปลงให้แล้ว บางเบอร์ถูกล้างเป็น `""`
- สิทธิ์ sub ใช้แค่ `member` (+ `payment` เมื่อส่ง `balance`) · `pt` ในรายละเอียดแสดงเสมอ
- ส่งสถานะเดิม = `200` ไม่มีประวัติเพิ่ม
