# Backoffice Auth (`agent_auth`) ระยะ 2 — Subaccount login · Passcode · Password · Admin reset — Spec

- สถานะ: **DRAFT**
- อนุมัติโดย: — · วันที่: —
- ชื่อ module ในโค้ด: `agent_auth` (ต่อจาก [agent_auth.md](agent_auth.md) — rule `AUTH-01`–`AUTH-16` ยังใช้ทั้งหมด)
- ที่มาของ rule: lead ให้ rule โดยตรง (2026-10-02 / 2026-10-03) ไม่ได้ extract จากโค้ดเก่า

> เอกสารนี้คือ source of truth ของงานระยะ 2 ของ module — โค้ดและ test ต้องตรงกับเอกสารนี้ ถ้าไม่ตรงให้แก้โค้ด
> หรือแก้เอกสารผ่านการอนุมัติใหม่ ห้ามอ้างเอกสารของโปรเจกต์เก่า

## 1. หน้าที่ของ module (ส่วนที่เพิ่ม)

**อยู่ใน scope รอบนี้:**
- subaccount login
- passcode: ตั้งครั้งแรก, ยืนยันหลัง login, เปลี่ยนเอง (จำได้), middleware `RequirePasscode`
- เปลี่ยนรหัสผ่านเอง (จำได้)
- role `ADMIN`: รีเซ็ต passcode / รหัสผ่านให้คนที่จำไม่ได้ + `scripts/create_admin` + middleware `RequireRole`
- บังคับเปลี่ยนรหัสผ่าน / passcode หลังถูก admin รีเซ็ต

**แยกออกไปเป็นงานอื่น:** quick-start, การเข้าถึง agent ในสาย (`/agents/:agent_id`), role/permission guard
(รอบนี้มีแค่ `RequireRole(ADMIN)`), สร้าง/จัดการ subaccount

## 2. คำศัพท์

| คำ | ความหมาย |
|---|---|
| subaccount (sub) | บัญชีย่อยที่ Company, Share หรือ Agent สร้าง ทำงานในนามของผู้สร้างตามสิทธิ์ที่ได้รับ |
| ผู้สร้าง | agent ที่เป็นเจ้าของ sub (`subaccounts.agent_id`) |
| passcode | ตัวเลข 6 หลัก ใช้ยืนยันทุกครั้งหลัง login และตอนทำรายการบางอย่าง |
| ผ่านขั้น passcode | session นั้นตั้งหรือยืนยัน passcode สำเร็จแล้ว |
| admin | บัญชี role `ADMIN` ใน `user_agents` (ทีม support) รีเซ็ต passcode / รหัสผ่านให้ผู้ใช้ที่จำไม่ได้ |
| ถูกบังคับเปลี่ยน | บัญชีที่ admin เพิ่งรีเซ็ต (`must_change_password` / `must_change_passcode` = true) |

## 3. Business rules

### Subaccount login

| ID | Rule |
|---|---|
| AUTH-17 | sub เก็บในตาราง `subaccounts` แยกจาก `user_agents` และเก็บ `agent_id` ของผู้สร้าง |
| AUTH-18 | username ของ sub = `{username ผู้สร้าง}@{ชื่อที่ตั้ง}` · username มี `@` → ค้นใน `subaccounts` · ไม่มี `@` → ค้นใน `user_agents` · username ของ agent ห้ามมี `@` · ใช้ AUTH-01/02/03 เหมือน agent |
| AUTH-19 | sub มี status ของตัวเอง (`ACTIVE` / `SUSPENDED` / `LOCKED`) ผู้สร้างล็อก sub ทีละตัวได้ · sub ที่ `LOCKED` login ไม่ได้ (`401301`) — เช็คหลังรหัสผ่านถูกแล้วเท่านั้น |
| AUTH-20 | ผู้สร้างหรือ upline คนใดของผู้สร้างเป็น `LOCKED` → sub login ไม่ได้ (`401302`) |
| AUTH-21 | sub หรือผู้สร้างเป็น `SUSPENDED` → login ได้ แต่ทำอะไรไม่ได้เหมือนผู้สร้าง (ความหมายของ SUSPENDED กำหนดใน module สายงาน) |
| AUTH-22 | session แยกต่อคน — AUTH-06/07/08/09 ใช้กับ sub แต่ละตัว · sub login ไม่ทำให้ผู้สร้างหลุด และผู้สร้าง login ไม่ทำให้ sub หลุด |
| AUTH-23 | AUTH-10/11 ใช้กติกาเดียวกับ agent · ตัวนับคิดตาม username เต็มของ sub |
| AUTH-24 | response login เหมือน agent · `role` = role ของผู้สร้าง · `is_subaccount = true` |
| AUTH-25 | sub ใช้งานในนามของผู้สร้าง — ขอบเขตข้อมูลเป็นของผู้สร้าง (การจำกัดสิทธิ์อยู่ในงาน permission) |
| AUTH-26 | middleware: sub ถูกเปลี่ยนเป็น `LOCKED` ระหว่างใช้งาน → request ถัดไป `401301` และ session ถูกลบ (เหมือน AUTH-15) · คำสั่ง LOCK ผู้สร้าง (module สายงาน) ต้องลบ session ของ sub ทั้งหมดของผู้สร้างด้วย |

### Passcode

| ID | Rule |
|---|---|
| AUTH-27 | passcode เป็นตัวเลข 6 หลัก (`0`–`9`) ไม่มีเลขต้องห้าม · เก็บเป็น bcrypt hash เท่านั้น |
| AUTH-28 | agent, sub และ admin มี passcode ของตัวเองแยกกัน |
| AUTH-29 | **ด่านหลัง login** — session ต้องผ่านด่านตามลำดับนี้ก่อนเรียก route อื่นใต้ `/bo/pr` ได้ (logout เรียกได้เสมอ): |
| | 1. `must_change_password` → เรียกได้เฉพาะ `password/change` อื่นๆ ตอบ `401306` |
| | 2. `must_change_passcode` → เรียกได้เฉพาะ `passcode/change` อื่นๆ ตอบ `401307` |
| | 3. ยังไม่เคยตั้ง passcode → เรียกได้เฉพาะ `passcode/setup` อื่นๆ ตอบ `401304` |
| | 4. ตั้งแล้วแต่ session นี้ยังไม่ยืนยัน → เรียกได้เฉพาะ `passcode/verify` อื่นๆ ตอบ `401305` |
| | ผ่านครบแล้วใช้ได้จนจบ session |
| AUTH-30 | ตั้ง passcode ครั้งแรก: ส่ง `passcode` และ `confirm_passcode` ตรงกัน · ใช้ได้เฉพาะตอนยังไม่เคยตั้ง — ตั้งแล้วตอบ `401401` |
| AUTH-31 | ตั้ง passcode สำเร็จ = ผ่านขั้น passcode ของ session นั้นแล้ว ไม่ต้องยืนยันซ้ำ |
| AUTH-32 | route ที่ต้องยืนยันตอนทำรายการ ส่ง `passcode` มาใน body และผ่าน `mw.RequirePasscode` — แต่ละ module เลือกเองว่า route ไหนต้องใช้ |
| AUTH-33 | passcode ผิด (ตอนยืนยันหลัง login, ตอนทำรายการ และ `old_passcode` ตอนเปลี่ยน passcode — นับรวมกัน) → `401204` · ผิดครบ **5 ครั้ง** → session ถูกลบ ตอบ `401205` · ตัวนับอยู่ใน session (login ใหม่เริ่มนับ 0) · ไม่เปลี่ยน status ของบัญชี |
| AUTH-39 | เปลี่ยน passcode เอง: ส่ง `old_passcode`, `new_passcode`, `confirm_passcode` (ไม่ต้องใช้รหัสผ่าน) · agent, sub และ admin เปลี่ยนของตัวเองได้ · ต้องผ่านขั้น passcode ก่อน ยกเว้นตอนถูกบังคับเปลี่ยน (`old_passcode` = ตัวที่ admin ตั้งให้) |
| AUTH-40 | passcode ใหม่ห้ามซ้ำกับตัวปัจจุบัน (`401403`) |
| AUTH-41 | เปลี่ยน passcode สำเร็จ → `must_change_passcode = false` · session ปัจจุบันถูกลบ ต้อง login ใหม่และยืนยันด้วย passcode ใหม่ |

### รหัสผ่าน

| ID | Rule |
|---|---|
| AUTH-34 | เปลี่ยนรหัสผ่านเอง: ส่ง `old_password`, `new_password`, `confirm_password` และ `passcode` (ผ่าน `mw.RequirePasscode`) · agent, sub และ admin เปลี่ยนของตัวเองได้ · **ตอนถูกบังคับเปลี่ยน** ไม่ต้องผ่านขั้น passcode และไม่ต้องส่ง `passcode` (`old_password` = รหัสที่ admin ตั้งให้) |
| AUTH-35 | กติการหัสผ่าน — ใช้ร่วมกันทุกที่ที่ตั้งรหัส (อยู่ใน `app/core`): ยาว **8–12** ตัว · ใช้ได้เฉพาะ `a-z`, `A-Z`, `0-9` และอักขระพิเศษ · ห้ามมีช่องว่าง · ต้องมีตัวอักษร (`a-z`/`A-Z`) อย่างน้อย 1 ตัว และตัวเลขอย่างน้อย 1 ตัว · ตัวพิมพ์เล็ก/ใหญ่ถือเป็นคนละตัว · ห้ามตัวเดียวกันซ้ำติดกัน 4 ตัวขึ้นไป (`aaaa`, `1111`) · ห้ามเรียง **ขึ้น** ติดกัน 4 ตัวขึ้นไป (`1234`, `abcd`) — เรียงลง (`4321`) และตัวพิมพ์ใหญ่ (`ABCD`) ไม่นับ · ไม่เช็ค username · ไม่เทียบรายการรหัสยอดนิยม |
| AUTH-36 | รหัสใหม่ห้ามซ้ำกับรหัสปัจจุบันและรหัสก่อนหน้า 1 รหัส (`401402`) — ใช้ทั้งตอนเปลี่ยนเองและตอน admin รีเซ็ต |
| AUTH-37 | `old_password` ผิด → `401206` และนับรวมกับตัวนับ AUTH-10 · ครบ 5 ครั้ง → login ไม่ได้ 15 นาที (`401303`) และ session ปัจจุบันถูกลบ |
| AUTH-38 | เปลี่ยนรหัสผ่านสำเร็จ → `must_change_password = false` · session ปัจจุบันถูกลบ ต้อง login ใหม่ |

### Admin รีเซ็ต (ผู้ใช้จำไม่ได้)

| ID | Rule |
|---|---|
| AUTH-42 | admin เป็นบัญชีใน `user_agents` role `ADMIN` · สร้างด้วย `scripts/create_admin` เท่านั้น (รหัสผ่านอ่านจาก stdin และตรวจตาม AUTH-35) · login, passcode และด่าน AUTH-29 เหมือนบัญชีอื่น |
| AUTH-43 | route ของ admin ใช้ได้เฉพาะ role `ADMIN` (`mw.RequireRole(ADMIN)`) — role อื่นตอบ `401308` · ต้องผ่านขั้น passcode และส่ง `passcode` ของ admin เองใน body (`mw.RequirePasscode`) |
| AUTH-44 | ระบุเป้าหมายด้วย `username` (มี `@` = sub, ไม่มี = `user_agents` ตาม AUTH-18 · ผ่าน AUTH-01) · ไม่พบ → `401404` · admin รีเซ็ตให้ได้ทุกบัญชี: ทุก agent, ทุก sub, Superadmin, admin ด้วยกัน และตัวเอง |
| AUTH-45 | รีเซ็ต passcode: admin ส่ง `new_passcode` + `confirm_passcode` (AUTH-27) → บันทึก passcode ใหม่ · `must_change_passcode = true` · session ของเป้าหมายถูกลบทันที |
| AUTH-46 | รีเซ็ตรหัสผ่าน: admin ส่ง `new_password` + `confirm_password` ตาม AUTH-35 และ AUTH-36 (เทียบกับรหัสของเป้าหมาย) → บันทึกรหัสใหม่ · `must_change_password = true` · session ของเป้าหมายถูกลบทันที |
| AUTH-47 | ถูกบังคับเปลี่ยน: login แล้วเปลี่ยนได้ทันทีโดยไม่ต้องยืนยัน passcode ก่อน (AUTH-29 ข้อ 1–2) · ถูกรีเซ็ตทั้งสองอย่าง → เปลี่ยนรหัสผ่านก่อน → login ใหม่ → เปลี่ยน passcode → login ใหม่ → ยืนยัน passcode |

## 4. สิ่งที่พบในโค้ดเก่า และการตัดสินใจ

ไม่มี — รอบนี้ lead ให้ rule โดยตรง ไม่ได้ extract จากโค้ดเก่า

## 5. Endpoints

| Method | Path | ต้อง login | ด่าน AUTH-29 | สิทธิ์ | Contract |
|---|---|---|---|---|---|
| POST | `/api/v1/bo/pb/auth/login` | ไม่ | — | — | เส้นเดิม รับ sub + field ใหม่ |
| POST | `/api/v1/bo/pr/auth/logout` | ใช่ | ไม่ต้องผ่าน | — | เส้นเดิม ไม่เปลี่ยน |
| POST | `/api/v1/bo/pr/auth/passcode/setup` | ใช่ | ข้อ 3 | — | **ใหม่** |
| POST | `/api/v1/bo/pr/auth/passcode/verify` | ใช่ | ข้อ 4 | — | **ใหม่** |
| POST | `/api/v1/bo/pr/auth/passcode/change` | ใช่ | ผ่านครบ หรือข้อ 2 | — | **ใหม่** |
| POST | `/api/v1/bo/pr/auth/password/change` | ใช่ | ผ่านครบ + `RequirePasscode` หรือข้อ 1 | — | **ใหม่** |
| POST | `/api/v1/bo/pr/admin/passcode/reset` | ใช่ | ผ่านครบ | `RequireRole(ADMIN)` + `RequirePasscode` | **ใหม่** |
| POST | `/api/v1/bo/pr/admin/password/reset` | ใช่ | ผ่านครบ | `RequireRole(ADMIN)` + `RequirePasscode` | **ใหม่** |

ลำดับ middleware ใน `routes.go`: `Authenticated` → ด่าน AUTH-29 → `RequireRole` (ถ้ามี) → `RequirePasscode` (ถ้ามี)
— เขียนไว้บรรทัดเดียวกับ route ทุกเส้น

### POST /api/v1/bo/pb/auth/login (เส้นเดิม — รับ sub + field ใหม่)

Request (sub):
```json
{ "username": "agent01@staff", "password": "••••••••" }
```
Response `data`:
```json
{
  "token": "<jwt>",
  "username": "agent01@staff",
  "role": "AGENT",
  "is_subaccount": true,
  "passcode_set": false,
  "must_change_password": false,
  "must_change_passcode": false,
  "expires_at": "2026-10-02T22:00:00+07:00"
}
```
- `role` = role ของผู้สร้าง (กรณี sub)
- `passcode_set` เป็นค่าจริงแล้ว (เดิมส่ง `false` เสมอ)
- `must_change_password` / `must_change_passcode` — **ใหม่** frontend ใช้เลือกหน้าถัดไปตาม AUTH-29

Error codes: `422`, `401201`, `401301`, `401302`, `401303`, `429`, `500`

### POST /api/v1/bo/pr/auth/passcode/setup (ใหม่ — ตั้ง passcode ครั้งแรก)

Header `Authorization: Bearer <token>`

Request:
```json
{ "passcode": "123456", "confirm_passcode": "123456" }
```
Response:
```json
{ "code": 200, "msg": "สำเร็จ" }
```
ไม่มี `data`

Error codes: `422` (ไม่ใช่ตัวเลข 6 หลัก / สองช่องไม่ตรงกัน — msg บอกว่าผิดที่ field ไหน), `401401`, `401306`,
`401307`, `401202`, `401203`

### POST /api/v1/bo/pr/auth/passcode/verify (ใหม่ — ยืนยัน passcode หลัง login)

Header `Authorization: Bearer <token>`

Request:
```json
{ "passcode": "123456" }
```
Response:
```json
{ "code": 200, "msg": "สำเร็จ" }
```
ไม่มี `data`

Error codes: `422`, `401204`, `401205`, `401304`, `401306`, `401307`, `401202`, `401203`

### POST /api/v1/bo/pr/auth/passcode/change (ใหม่ — เปลี่ยน passcode เอง)

Header `Authorization: Bearer <token>`

Request:
```json
{ "old_passcode": "123456", "new_passcode": "654321", "confirm_passcode": "654321" }
```
Response:
```json
{ "code": 200, "msg": "สำเร็จ" }
```
ไม่มี `data` — session ถูกลบแล้ว frontend ต้องพาไปหน้า login

Error codes: `422`, `401204`, `401205`, `401403`, `401304`, `401305`, `401306`, `401202`, `401203`

### POST /api/v1/bo/pr/auth/password/change (ใหม่ — เปลี่ยนรหัสผ่านเอง)

Header `Authorization: Bearer <token>`

Request (ปกติ):
```json
{
  "old_password": "••••••••",
  "new_password": "••••••••",
  "confirm_password": "••••••••",
  "passcode": "123456"
}
```
Request (ถูกบังคับเปลี่ยน — ไม่มี `passcode`):
```json
{ "old_password": "••••••••", "new_password": "••••••••", "confirm_password": "••••••••" }
```
Response:
```json
{ "code": 200, "msg": "สำเร็จ" }
```
ไม่มี `data` — session ถูกลบแล้ว frontend ต้องพาไปหน้า login

Error codes: `422` (ไม่กรอก / `confirm_password` ไม่ตรง / ผิดกติกา AUTH-35 — msg บอกว่าผิดกฎข้อไหน), `401204`,
`401205`, `401206`, `401303`, `401402`, `401305`, `401307`, `401202`, `401203`

### POST /api/v1/bo/pr/admin/passcode/reset (ใหม่ — admin รีเซ็ต passcode)

Header `Authorization: Bearer <token ของ admin>`

Request:
```json
{
  "username": "agent01@staff",
  "new_passcode": "112233",
  "confirm_passcode": "112233",
  "passcode": "123456"
}
```
`passcode` = passcode ของ admin เอง

Response:
```json
{ "code": 200, "msg": "สำเร็จ" }
```
ไม่มี `data`

Error codes: `422`, `401308`, `401404`, `401204`, `401205`, `401305`, `401202`, `401203`

### POST /api/v1/bo/pr/admin/password/reset (ใหม่ — admin รีเซ็ตรหัสผ่าน)

Header `Authorization: Bearer <token ของ admin>`

Request:
```json
{
  "username": "agent01",
  "new_password": "••••••••",
  "confirm_password": "••••••••",
  "passcode": "123456"
}
```
`passcode` = passcode ของ admin เอง

Response:
```json
{ "code": 200, "msg": "สำเร็จ" }
```
ไม่มี `data`

Error codes: `422` (ผิดกติกา AUTH-35 — msg บอกว่าผิดกฎข้อไหน), `401308`, `401404`, `401402`, `401204`, `401205`,
`401305`, `401202`, `401203`

## 6. Schema

migration ไฟล์ใหม่ (ไม่แก้ `20261002000000_agent_auth_create_user_agents.sql`)

```sql
ALTER TABLE user_agents
    ADD COLUMN passcode_hash          VARCHAR(100),
    ADD COLUMN previous_password_hash VARCHAR(100),
    ADD COLUMN must_change_password   BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN must_change_passcode   BOOLEAN NOT NULL DEFAULT false,
    DROP CONSTRAINT ck_user_agents_role,
    ADD CONSTRAINT ck_user_agents_role
        CHECK (role IN ('SUPERADMIN','ADMIN','COMPANY','SHAREHOLDER','AGENT'));

CREATE TABLE subaccounts (
    id                     BIGSERIAL PRIMARY KEY,
    agent_id               BIGINT       NOT NULL REFERENCES user_agents(id),  -- ผู้สร้าง
    username               VARCHAR(50)  NOT NULL,                             -- {owner}@{name}
    password_hash          VARCHAR(100) NOT NULL,
    previous_password_hash VARCHAR(100),
    passcode_hash          VARCHAR(100),
    must_change_password   BOOLEAN      NOT NULL DEFAULT false,
    must_change_passcode   BOOLEAN      NOT NULL DEFAULT false,
    status                 VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE',
    last_login_at          TIMESTAMPTZ,
    last_login_ip          VARCHAR(45),
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_subaccounts_username UNIQUE (username),
    CONSTRAINT ck_subaccounts_status CHECK (status IN ('ACTIVE','SUSPENDED','LOCKED'))
);
CREATE INDEX idx_subaccounts_agent_id ON subaccounts(agent_id);
```

- `passcode_hash IS NULL` = `passcode_set = false`
- `previous_password_hash` เก็บรหัสก่อนหน้า 1 รหัสสำหรับ AUTH-36 (ไม่ทำตารางประวัติแยก เพราะเก็บแค่ 1 รหัส)
- admin ไม่อยู่ในสายงาน (`parent_id = NULL`)
- `subaccounts` มีเฉพาะคอลัมน์ที่ auth ใช้ — สิทธิ์และคอลัมน์อื่นเพิ่มตอนทำ module subaccount/permission
- middleware โหลด `status`, `passcode_hash IS NULL`, `must_change_password`, `must_change_passcode` จากแถวของบัญชี
  ทุก request (ต่อจากขั้นที่ 3 ของ middleware เดิม) เพื่อเช็คด่าน AUTH-29
- การทดสอบ: integration test insert แถว sub ลง DB เอง ไม่มี script สร้าง sub

Redis:

| Key | ค่า | TTL |
|---|---|---|
| `bo:sess:{sid}` | JSON `{account_type, account_id, agent_id, ip, created_at, passcode_ok, passcode_fail}` | idle 60 นาที (เหมือนเดิม) |
| `bo:sess:agent:{agent_id}` | `sid` ปัจจุบันของ agent / admin | absolute 12 ชม. (เดิม) |
| `bo:sess:sub:{subaccount_id}` | `sid` ปัจจุบันของ sub | absolute 12 ชม. |
| `bo:login:fail:{username}` | จำนวนครั้งที่ผิด (login + `old_password` ผิด) | 15 นาที (เดิม) |
| `bo:login:ip:{ip}` | จำนวนครั้งที่เรียก | 1 นาที (เดิม) |

- `account_type` = `AGENT` / `SUB` · `agent_id` = id ของ agent เอง หรือของผู้สร้างถ้าเป็น sub (AUTH-25)
- token ระบุชนิดบัญชีและ id ด้วย · key ชี้ session แยก agent/sub เพราะ id มาจากคนละตารางและซ้ำกันได้
- `scripts/create_superadmin` และ `scripts/create_admin` ตรวจรหัสตาม AUTH-35

## 7. Test cases

| Rule | Input | Expected |
|---|---|---|
| AUTH-18 | login `" Agent01@Staff "` (บัญชี `agent01@staff`) | สำเร็จ, `is_subaccount = true` |
| AUTH-18 | `agent01@ไม่มีจริง` | `401201` |
| AUTH-19 | sub `LOCKED` + รหัสถูก | `401301` |
| AUTH-19 | sub `LOCKED` + รหัสผิด | `401201` |
| AUTH-20 | ผู้สร้าง `LOCKED` | `401302` |
| AUTH-20 | upline ของผู้สร้าง `LOCKED` | `401302` |
| AUTH-21 | ผู้สร้าง `SUSPENDED` | login สำเร็จ |
| AUTH-22 | sub login แล้วผู้สร้าง login | token ของทั้งสองยังใช้ได้ |
| AUTH-23 | sub ผิด 5 ครั้ง แล้วใส่ถูก | `401303` · ผู้สร้างยัง login ได้ |
| AUTH-24 | sub ของ Agent | `role = AGENT`, `is_subaccount = true` |
| AUTH-26 | sub ถูก LOCK ระหว่างใช้งาน | request ถัดไป `401301` และ session ถูกลบ |
| AUTH-29 | login แล้วเรียก route อื่นโดยยังไม่เคยตั้ง passcode | `401304` |
| AUTH-29 | ตั้งแล้ว login ใหม่ เรียก route อื่นโดยยังไม่ยืนยัน | `401305` |
| AUTH-29 | ยังไม่ผ่านด่านแล้ว logout | สำเร็จ |
| AUTH-29 | `must_change_password` แล้วเรียก `passcode/verify` | `401306` |
| AUTH-29 | `must_change_passcode` แล้วเรียก `passcode/verify` | `401307` |
| AUTH-30 | `confirm_passcode` ไม่ตรง / `12345` / `12345a` | `422` |
| AUTH-30 | ตั้งซ้ำครั้งที่ 2 | `401401` |
| AUTH-31 | ตั้งสำเร็จแล้วเรียก route อื่น | ผ่าน |
| AUTH-33 | ยืนยันผิด 4 ครั้ง แล้วถูก | ผ่าน |
| AUTH-33 | ยืนยันผิด 5 ครั้ง | ครั้งที่ 5 ได้ `401205` · token เดิมได้ `401203` · status บัญชีไม่เปลี่ยน |
| AUTH-33 | ผิดตอนยืนยัน 3 ครั้ง + ผิดตอนทำรายการ 2 ครั้ง | ครั้งที่ 5 ได้ `401205` |
| AUTH-33 | ผิด 4 ครั้ง → login ใหม่ → ผิดอีก 4 ครั้ง | ยังไม่ถูกตัด |
| AUTH-39 | sub เปลี่ยน passcode ของตัวเอง | สำเร็จ |
| AUTH-39 | ยังไม่ยืนยัน passcode แล้วเรียก `passcode/change` (ไม่ได้ถูกบังคับ) | `401305` |
| AUTH-39 | `old_passcode` ผิด | `401204` (นับรวม AUTH-33) |
| AUTH-40 | `new_passcode` = ตัวปัจจุบัน | `401403` |
| AUTH-41 | เปลี่ยนสำเร็จ แล้วใช้ token เดิม | `401203` · login ใหม่ยืนยันด้วยตัวใหม่ได้ · ตัวเก่าได้ `401204` |
| AUTH-35 | `Zx9!kq2` (7 ตัว) / `Zx9!kq2mWp7r4` (13 ตัว) | `422` |
| AUTH-35 | `Zx9 kq2m` (ช่องว่าง) / `Zx9กkq2m` (อักษรไทย) | `422` |
| AUTH-35 | `Zxkqmwpt` (ไม่มีตัวเลข) / `92837465` (ไม่มีตัวอักษร) | `422` |
| AUTH-35 | `Zx9aaaam` (ซ้ำ 4) / `Zx1234km` / `Zxabcd9m` (เรียงขึ้น 4) | `422` |
| AUTH-35 | `Zx9!kq2m` / `4321dcba` / `ABCD1x9z` | ผ่าน |
| AUTH-36 | ใช้รหัสปัจจุบัน / รหัสก่อนหน้า | `401402` |
| AUTH-37 | `old_password` ผิด 1 ครั้ง | `401206` |
| AUTH-37 | `old_password` ผิด 5 ครั้ง | session ถูกลบ · login ไม่ได้ 15 นาที (`401303`) |
| AUTH-38 | เปลี่ยนสำเร็จ แล้วใช้ token เดิม | `401203` · login ด้วยรหัสใหม่ได้ · รหัสเดิมไม่ได้ |
| AUTH-43 | agent (ไม่ใช่ admin) เรียก `admin/password/reset` | `401308` |
| AUTH-43 | admin ส่ง `passcode` ของตัวเองผิด | `401204` |
| AUTH-44 | `username` ไม่มีในระบบ | `401404` |
| AUTH-44 | admin รีเซ็ตให้ Superadmin / admin คนอื่น / ตัวเอง | สำเร็จ |
| AUTH-45 | admin รีเซ็ต passcode ของ sub ที่ login อยู่ | token ของ sub ได้ `401203` · login ใหม่ได้ `must_change_passcode = true` |
| AUTH-46 | admin รีเซ็ตรหัสผ่านเป็นรหัสปัจจุบันของเป้าหมาย | `401402` |
| AUTH-46 | admin รีเซ็ตรหัสผ่าน | session เป้าหมายถูกลบ · login ด้วยรหัสใหม่ได้ `must_change_password = true` |
| AUTH-47 | ถูกรีเซ็ตรหัสผ่าน → login → `password/change` ไม่ส่ง `passcode` | สำเร็จ · login ใหม่แล้วต้องยืนยัน passcode |
| AUTH-47 | ถูกรีเซ็ต passcode → login → `passcode/change` โดย `old_passcode` = ตัวที่ admin ตั้ง | สำเร็จ |
| AUTH-47 | ถูกรีเซ็ตทั้งสองอย่าง → login → เรียก `passcode/change` ก่อน | `401306` |

## 8. Contract changes (แจ้ง frontend)

- login response เพิ่ม `must_change_password`, `must_change_passcode` · `passcode_set` เป็นค่าจริง
- หลัง login เลือกหน้าตามลำดับ AUTH-29: เปลี่ยนรหัสผ่าน → เปลี่ยน passcode → ตั้ง passcode → ยืนยัน passcode
- route ใต้ `/bo/pr` ตอบ `401304` / `401305` / `401306` / `401307` จนกว่าจะผ่านด่าน
- username ที่มี `@` login ได้จริงถ้าเป็น sub (เดิมตอบ `401201` เสมอ)
- เพิ่ม 6 route ใหม่ (หัวข้อ 5) และ error code ใหม่ (หัวข้อ 9 ข้อ 3)
- `401205`, เปลี่ยนรหัสผ่านสำเร็จ, เปลี่ยน passcode สำเร็จ → พาไปหน้า login
- route ที่ต้องยืนยันตอนทำรายการ ต้องส่ง `passcode` ใน body

## 9. การตัดสินใจ (2026-10-02 / 2026-10-03)

1. **ตัวนับ passcode ผิดนับต่อ session** — ยอมรับว่าคนที่รู้รหัสผ่านเดา passcode ได้ 5 ครั้งต่อการ login 1 ครั้ง
   โดยติดเพียง AUTH-10/11
2. **จำได้ → เปลี่ยนเอง · จำไม่ได้ → admin รีเซ็ต** ทั้ง passcode และรหัสผ่าน · admin เป็น role ใหม่ (ทีม support)
   สร้างด้วย script · admin กรอกค่าใหม่ให้ แล้วเป้าหมายถูกบังคับเปลี่ยนเองหลัง login (AUTH-45–47)
3. **Error code ใหม่** (module `agent_auth` = `bb=01`)

| Code | HTTP | ความหมาย |
|---|---|---|
| 401204 | 200 | passcode ไม่ถูกต้อง |
| 401205 | 401 | ใส่ passcode ผิดครบ 5 ครั้ง session ถูกตัด |
| 401206 | 200 | รหัสผ่านเดิมไม่ถูกต้อง |
| 401304 | 403 | ต้องตั้ง passcode ก่อน |
| 401305 | 403 | ต้องยืนยัน passcode ก่อน |
| 401306 | 403 | ต้องเปลี่ยนรหัสผ่านก่อน |
| 401307 | 403 | ต้องเปลี่ยน passcode ก่อน |
| 401308 | 403 | ไม่มีสิทธิ์ใช้งานส่วนของ admin |
| 401401 | 200 | ตั้ง passcode ไว้แล้ว |
| 401402 | 200 | รหัสผ่านใหม่ซ้ำกับรหัสที่เคยใช้ |
| 401403 | 200 | passcode ใหม่ซ้ำกับตัวเดิม |
| 401404 | 200 | ไม่พบบัญชีที่ต้องการรีเซ็ต |

`401204` / `401206` ตอบ HTTP 200 เพราะ frontend พาไปหน้า login เมื่อได้ 401 — ใส่ผิดครั้งเดียวไม่ควรถูกเด้งออก

4. **`create_superadmin` / `create_admin`** ตรวจรหัสตาม AUTH-35
5. **บัญชีทดสอบ sub** — integration test insert ลง DB เอง ไม่ทำ script
6. **role/permission guard เต็มรูปแบบ** แยกเป็นงานอื่น — รอบนี้มีแค่ `RequireRole(ADMIN)` เพื่อให้ route admin
   ระบุสิทธิ์บรรทัดเดียวกับ route ได้ตามกฎข้อ 28
