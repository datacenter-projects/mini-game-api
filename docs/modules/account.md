# บัญชี (`account`) — Dashboard · Profile · ข้อมูลรับรอง API — Spec

- สถานะ: **DRAFT** (แก้ตาม review 2026-10-05 — รออนุมัติใหม่)
- อนุมัติโดย: — · วันที่: — (ฉบับก่อนอนุมัติ 2026-10-05 · review เปลี่ยน Profile, 1.3 และเพิ่ม Dashboard)
- ชื่อ module ในโค้ด: `account` (`controllers/account`, `dto/account`, `service/account`, `core/account`)
- เมนู: **1.1 ภาพรวม (Dashboard)** · **1.2 ประวัติของฉัน (Profile)** · **1.3 ข้อมูลรับรอง API**
- ที่มาของ rule: ภาพหน้าจอตัวอย่าง + เอกสาร Seamless API Flow ของ lead + คำตอบของ boiledegg + review comments (2026-10-05) ·
  ไม่ได้ extract จากโค้ดเก่า

> เอกสารนี้คือ source of truth ของ module — โค้ดและ test ต้องตรงกับเอกสารนี้ ถ้าไม่ตรงให้แก้โค้ด
> หรือแก้เอกสารผ่านการอนุมัติใหม่ ห้ามอ้างเอกสารของโปรเจกต์เก่า

## 1. หน้าที่ของ module

หน้าข้อมูลบัญชีของผู้ใช้หลังบ้าน:
- **Dashboard (1.1)** — สรุปรายการเล่นของ Member ในสาย และสรุปรายได้ของคนที่ login
- **Profile (1.2)** — ข้อมูลบัญชีของตัวเอง รวมประเภท, สกุลเงิน, ค่าหุ้นส่วน และสิทธิ์ — หน้าบ้านใช้ต่อในส่วนอื่น
  (เช่น ตอนสร้างบัญชีชั้นถัดไป) · เปลี่ยนรหัสผ่าน / passcode ใช้ endpoint ของ `agent_auth`
- **1.3 ข้อมูลรับรอง API** — Key, ลิงก์ตอบกลับ และ IP ที่อนุญาต ที่ลูกค้าแบบ Seamless ใช้เชื่อมระบบกับเรา (Seamless API Flow)

**ต้องมีก่อน implement:**

| ส่วน | ต้องมี | มาจาก |
|---|---|---|
| Profile | ประเภทบัญชี, สกุลเงิน, ค่าหุ้นส่วน, ระบบสิทธิ์ (รายการสิทธิ์ + สิทธิ์ของ sub) | module ② User Management |
| 1.3 | ประเภทบัญชี (หาเจ้าของ Key) · ระบบสิทธิ์ (สิทธิ์ดู / แก้ไขของ sub) | module ② User Management |
| Dashboard | bet, ผลได้เสียตาม PT, Commission | module เดิมพัน / Settle |

**ไม่อยู่ใน module นี้:** เครดิตแยกสกุล (module การชำระเงิน — เพิ่มเข้า Profile ตอนทำ module นั้น) · การใช้ Key / IP / ลิงก์ตอนลูกค้า
เรียก API (module provider `app/externals/provider`)

## 2. คำศัพท์

| คำ | ความหมาย |
|---|---|
| ประเภทบัญชี (`user_type`) | role + ประเภทย่อยตาม Company Hierarchy เช่น Company Transfer, Share B2C, Agent |
| ค่าหุ้นส่วน | PT ถือ · PT ปล่อย · Remain (เอาส่วนที่เหลือ) · Force (บังคับถือสู้) · Commission — ระบบเก็บต่อเกม แต่หน้าบ้านตั้ง**ชุดเดียว**แล้วใช้กับทุกเกม (module ②) |
| สิทธิ์ | รายการสิ่งที่บัญชีทำได้ (กำหนดใน module ② ครั้งเดียว) · บัญชีหลักได้ตามประเภท · sub ได้ตามที่ผู้สร้างให้ |
| เจ้าของ Key | Company Seamless 1 to 1 · Share Master · Share Reseller — บัญชีละ 1 Key (ACC-01) |
| Key | ค่าลับที่ลูกค้าใช้ระบุตัวและคำนวณ `sign` ทุกคำขอ (Seamless API Flow) |
| ลิงก์ตอบกลับ | URL ของระบบลูกค้าที่ระบบเราเรียก (GetMember, GetBalance, Bet, Settle, Cancel) |
| IP ที่อนุญาต | IP ของลูกค้าที่เรียก API ของเราได้ |

## 3. Business rules

### สถานะ (ใช้กับทุก module)

| ID | Rule |
|---|---|
| ACC-30 | `status` = สถานะที่ตั้งกับบัญชีนี้โดยตรง (หัวสายระงับ / ล็อกบัญชีนี้เอง) · `effective_status` = สถานะที่ใช้งานจริงเมื่อรวมผลจากหัวสาย (เข้มที่สุดของตัวเอง, ผู้สร้างกรณี sub และ upline ทั้งสาย — AUTH-53) เช่น Company ระงับ Share → Agent ใต้ Share นั้น `effective_status = SUSPENDED` |
| ACC-31 | การเข้าใช้งานตัดสินจาก `effective_status`: `ACTIVE` ใช้งานได้ปกติ · `SUSPENDED` เข้าได้**เฉพาะหน้า Profile และ Report** และดูได้อย่างเดียว · `LOCKED` เข้าใช้งานไม่ได้เลย (AUTH-27) — บังคับทุก module ที่ middleware กลาง (AUTH-54) |

### Dashboard (1.1)

| ID | Rule |
|---|---|
| ACC-20 | เปิดได้: บัญชีหลังบ้านที่มีสิทธิ์ Dashboard (สิทธิ์จาก module ②) · `effective_status` ต้องเป็น `ACTIVE` (ACC-31) |
| ACC-21 | ข้อมูลเฉพาะสายของตัวเอง (ตัวเอง + สายล่าง) · sub เห็นของผู้สร้าง (AUTH-26) · ไม่เห็นข้ามสาย ไม่เห็นชั้นบน (System Overview) |
| ACC-22 | ตัวกรอง: เกม (ทั้งหมด หรือเกมเดียว) · เดือน + ปี · ทางลัด ก่อนหน้า / เดือนนี้ / ปีนี้ / ถัดไป · สกุลเงิน 1 สกุลต่อครั้ง (เฉพาะสกุลที่บัญชีใช้ได้) |
| ACC-23 | ข้อมูลเป็นยอดสรุปรายวัน อัปเดตวันละครั้ง ช่วง 00:15–01:00 น. (ไม่ใช่ real-time) · ช่วงเวลาที่ยังไม่สรุปไม่นับ |
| ACC-24 | การ์ด: สมาชิกที่กำลังใช้งาน · สมาชิกที่สมัครใหม่ · สมาชิกทั้งหมด · ยอดเล่นทั้งหมด · ยอดเล่นที่ถูกรางวัล · ยอดเล่นจริง · ยอดเล่นทั้งหมดตาม PT · ยอดถูกรางวัลตาม PT · ยอดแพ้ / ชนะตาม PT · ทุกการ์ด (ยกเว้นสมาชิกที่กำลังใช้งาน) แสดง % เปลี่ยนแปลงเทียบช่วงก่อนหน้าที่ยาวเท่ากัน |
| ACC-25 | "ตาม PT" = ยอดคูณ PT ที่**คนที่ login ถือ** (ค่า ณ ตอน bet) · รายได้ของคนที่ login = ได้เสียตาม PT ที่ตัวเองถือ + Commission ที่ได้ − Commission ที่จ่าย |
| ACC-26 | ตาราง Top 10 สมาชิกในสาย: ยอดเล่นมากสุด · ยอดถูกมากสุด · ยอดเสียมากสุด (ลำดับ, username, จำนวนเงิน) |
| ACC-27 | เงินเป็น `int64` หน่วยย่อยที่สุดตามสกุล (กฎข้อ 9) · % เปลี่ยนแปลงทศนิยม 2 ตำแหน่ง · ช่วงก่อนหน้าเป็น 0 → แสดง `null` |

### Profile (1.2)

| ID | Rule |
|---|---|
| ACC-11 | ทุกบัญชีหลังบ้านเรียกได้ (SUPERADMIN, ADMIN, Company, Share, Agent และ sub) · ต้องผ่านด่านหลัง login (`PassedGates()`) · เปิดได้แม้ `effective_status = SUSPENDED` (ACC-31) · แสดงข้อมูลของ**ตัวเอง** |
| ACC-12 | ข้อมูลที่แสดง: username · role · **ประเภทบัญชี** · `status` · `effective_status` · เป็น sub ไหม · username ของผู้สร้าง (เฉพาะ sub) · ตั้ง passcode แล้วหรือยัง · login ล่าสุด (เวลา, IP) · วันที่สร้างบัญชี · **สกุลเงิน** · **ค่าหุ้นส่วน** (ชุดเดียว — ACC-16) · **สิทธิ์** |
| ACC-15 | sub: role, ประเภทบัญชี, สกุลเงิน และค่าหุ้นส่วน = ของผู้สร้าง (AUTH-25) · สิทธิ์ = ที่ผู้สร้างให้ sub นั้น |
| ACC-16 | ค่าหุ้นส่วนส่งเป็น key `pt` เดียว เป็น object **ชุดเดียว** (ถือ / ปล่อย / Remain / Force / Commission หน่วย bp) — หน้าบ้านตั้งที่เดียวแล้วระบบใช้ค่านั้นกับทุกเกม (ระบบเก็บต่อเกมตาม module ②) · SUPERADMIN / ADMIN ไม่มีค่าหุ้นส่วน (`pt = null`) |
| ACC-17 | Profile มีไว้ให้หน้าบ้านแสดงผล / เช็คเบื้องต้นในฟอร์ม (เช่น สร้างบัญชีชั้นถัดไปที่ PT เกินของตัวเองไม่ได้) · **หลังบ้านต้องเช็คซ้ำทุกครั้ง** ใน module ที่ทำรายการ |
| ACC-13 | อ่านอย่างเดียว · เปลี่ยนรหัสผ่าน / passcode ของตัวเองใช้เส้นเดิม `POST /bo/pr/auth/password/change` และ `/auth/passcode/change` |
| ACC-14 | login ล่าสุด = ค่าที่บันทึกตอน login (AUTH-12) จึงเป็นเวลาของ session ปัจจุบัน |

### 1.3 ข้อมูลรับรอง API

| ID | Rule |
|---|---|
| ACC-01 | เจ้าของ Key = **Company Seamless 1 to 1** · **Share Master** · **Share Reseller** · แต่ละบัญชีมี Key, ลิงก์ตอบกลับ และรายการ IP **ของตัวเอง** · บัญชีอื่นทั้งหมด (Superadmin, ADMIN, Company Transfer, Company Seamless Reseller / Master, Share B2B / B2C, Agent) เปิดหน้านี้ไม่ได้ (`403301`) |
| ACC-02 | เจ้าของเปิดดู / บันทึกได้ · sub ของเจ้าของ: **สิทธิ์ดู** → GET ได้ · **สิทธิ์แก้ไข** → POST ได้ · ไม่ได้รับสิทธิ์ → เข้าไม่ได้ (error สิทธิ์ของ module ②) · บันทึกต้อง `effective_status = ACTIVE` (`403302`) |
| ACC-03 | Key สร้างอัตโนมัติ 1 ค่าต่อเจ้าของ: สุ่ม 32 byte ด้วย `crypto/rand` แสดงเป็น hex ตัวพิมพ์เล็ก 64 ตัว · **สร้างใหม่ไม่ได้** · ใช้ทั้งระบุตัวและคำนวณ `sign` |
| ACC-04 | Key ดูซ้ำได้ (ปุ่มคัดลอก) → เก็บแบบเข้ารหัส AES-256-GCM ที่ถอดกลับได้ (ไม่ใช่ hash) + `sha256(Key)` ไว้ค้นตอนลูกค้าเรียก API · response ใส่ `Cache-Control: no-store` · ห้าม log Key |
| ACC-05 | สร้าง Key ตอนสร้างบัญชีเจ้าของ (module ②) · บัญชีที่ยังไม่มี Key สร้างตอนเปิดหน้าครั้งแรก · เรียกพร้อมกันได้ Key เดียวเสมอ (unique ที่ `agent_id`) |
| ACC-06 | ลิงก์ตอบกลับ: ต้องเป็น URL `https://` ที่มี host · ยาวไม่เกิน 500 ตัว · เว้นว่างได้ (= ยังไม่ตั้ง) |
| ACC-07 | IP ที่อนุญาต: IPv4 หรือช่วง CIDR ของ IPv4 (IP เดี่ยวเก็บเป็น `/32`) · ไม่เกิน **50** รายการ · ห้ามซ้ำ · **ไม่มีเลย = ลูกค้าเรียก API ของเราไม่ได้** |
| ACC-08 | บันทึก = แทนทั้งชุด (ลิงก์ + รายการ IP) · ต้องส่ง `passcode` ของผู้กด (`RequirePasscode`) · tx + `SELECT ... FOR UPDATE` แถว credential |
| ACC-09 | ทุกการบันทึกเก็บประวัติ `api_credential_logs` (ผู้แก้, ลิงก์เก่า → ใหม่, IP เก่า → ใหม่, ip, request_id, เวลา) ใน tx เดียวกัน · ห้ามเก็บ Key ในประวัติ |
| ACC-10 | ชื่อผู้ใช้ในหน้านี้ = username ของเจ้าของ (อ่านอย่างเดียว) |

## 4. สิ่งที่พบในโค้ดเก่า และการตัดสินใจ

ไม่มี — ไม่ได้ extract จากโค้ดเก่า

## 5. Endpoints

ใต้ `/bo/pr` ผ่าน `Authenticated` + `PassedGates()` · error ร่วม: `401202`, `401203`, `401301`, `401302`, `401304`, `401306`, `401307`
· เส้นที่ต้องมีสิทธิ์ใช้ middleware สิทธิ์ของ module ② บรรทัดเดียวกับ route

| Method | Path | middleware เพิ่ม |
|---|---|---|
| GET | `/api/v1/bo/pr/account/dashboard` | สิทธิ์ Dashboard · `effective_status = ACTIVE` |
| GET | `/api/v1/bo/pr/account/profile` | — (เปิดได้ตอน SUSPENDED) |
| GET | `/api/v1/bo/pr/account/api-credential` | สิทธิ์ดู 1.3 (sub) |
| POST | `/api/v1/bo/pr/account/api-credential` | สิทธิ์แก้ไข 1.3 (sub) · `RequirePasscode` |

### GET /api/v1/bo/pr/account/dashboard

Query: `game` (รหัสเกม · ไม่ส่ง = ทุกเกม) · `month` (1–12) · `year` · `currency` (บังคับ)

Response `data`:
```json
{
  "currency": "THB",
  "period": { "from": "2026-10-01", "to": "2026-10-31" },
  "updated_at": "2026-10-05T00:47:00+07:00",
  "cards": {
    "active_members":   { "value": 12 },
    "new_members":      { "value": 3, "change_pct": "50.00" },
    "total_members":    { "value": 120, "change_pct": "2.56" },
    "bet_count":        { "value": 5400, "change_pct": "-3.10" },
    "win_count":        { "value": 2100, "change_pct": "1.20" },
    "bet_amount":       { "value": 98765400, "change_pct": "4.00" },
    "bet_amount_pt":    { "value": 9876540, "change_pct": "4.00" },
    "win_amount_pt":    { "value": 8000000, "change_pct": null },
    "win_loss_pt":      { "value": 1876540, "change_pct": "12.50" }
  },
  "top_bet":  [ { "rank": 1, "username": "m001", "amount": 1500000 } ],
  "top_win":  [ { "rank": 1, "username": "m007", "amount": 900000 } ],
  "top_loss": [ { "rank": 1, "username": "m013", "amount": 700000 } ]
}
```
จำนวนเงินเป็นหน่วยย่อยที่สุดของสกุล (`int64`) · `change_pct` เป็นข้อความทศนิยม 2 ตำแหน่ง หรือ `null`

### GET /api/v1/bo/pr/account/profile

Response `data` (ตัวอย่าง sub ของ Share B2C):
```json
{
  "username": "share01@staff",
  "role": "SHAREHOLDER",
  "user_type": "SHARE_B2C",
  "status": "ACTIVE",
  "effective_status": "SUSPENDED",
  "is_subaccount": true,
  "owner_username": "share01",
  "passcode_set": true,
  "last_login_at": "2026-10-05T10:00:00+07:00",
  "last_login_ip": "203.0.113.10",
  "created_at": "2026-10-01T09:00:00+07:00",
  "currencies": ["THB"],
  "pt": {
    "hold_bp": 2000, "release_bp": 7000, "remain_bp": 0, "force_bp": 0, "commission_bp": 50
  },
  "permissions": ["dashboard.view", "report.view"]
}
```
- `user_type`: `SUPERADMIN` · `ADMIN` · `COMPANY_TRANSFER` · `COMPANY_SEAMLESS_RESELLER` · `COMPANY_SEAMLESS_MASTER` ·
  `COMPANY_SEAMLESS_1TO1` · `SHARE_B2B` · `SHARE_B2C` · `SHARE_RESELLER` · `SHARE_MASTER` · `AGENT`
- `owner_username` = `null` เมื่อไม่ใช่ sub · `last_login_*` = `null` ถ้ายังไม่เคยบันทึก
- หน่วยของค่าหุ้นส่วน, รายการสกุล และชื่อสิทธิ์ ใช้ตามที่ module ② กำหนด (ค่าในตัวอย่างเป็นแค่รูปแบบ)

Error codes: error ร่วมของ `/bo/pr` เท่านั้น

### GET /api/v1/bo/pr/account/api-credential

Response `data` (header `Cache-Control: no-store`):
```json
{
  "username": "shareresel01",
  "key": "cb886b34f28b4e7964ca039335a86c74d04a3dea605e1b2c9d0f7a8b6c5d4e3f",
  "callback_url": "https://api.customer.example/minigame",
  "allowed_ips": ["203.0.113.10/32", "198.51.100.0/24"]
}
```
`callback_url` = `null` เมื่อยังไม่ตั้ง · `allowed_ips` = `[]` เมื่อยังไม่มี

Error codes: `403301` · error สิทธิ์ (sub)

### POST /api/v1/bo/pr/account/api-credential

Request:
```json
{
  "callback_url": "https://api.customer.example/minigame",
  "allowed_ips": ["203.0.113.10", "198.51.100.0/24"],
  "passcode": "123456"
}
```
Response: `{"code":200,"msg":"สำเร็จ"}` (ไม่มี `data`)

Error codes: `422` (ลิงก์ไม่ใช่ https / ยาวเกิน · IP ผิดรูปแบบ · เกิน 50 รายการ · IP ซ้ำ — msg บอก field และรายการ), `403301`,
`403302`, `401204`, `401205` · error สิทธิ์ (sub)

## 6. Schema

migration ไฟล์ใหม่ (1.3) · Dashboard: ตารางสรุปรายวันออกแบบตอนทำ module เดิมพัน (ต้องมี index ตาม เจ้าของสาย + วันที่ + เกม + สกุล)

```sql
CREATE TABLE api_credentials (
    agent_id       BIGINT       PRIMARY KEY REFERENCES user_agents(id),  -- เจ้าของ Key
    key_ciphertext BYTEA        NOT NULL,                               -- AES-256-GCM (nonce + ciphertext)
    key_hash       CHAR(64)     NOT NULL,                               -- sha256(Key) hex — ค้นตอนลูกค้าเรียก API
    callback_url   VARCHAR(500),
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_api_credentials_key_hash UNIQUE (key_hash)
);

CREATE TABLE api_allowed_ips (
    id         BIGSERIAL   PRIMARY KEY,
    agent_id   BIGINT      NOT NULL REFERENCES api_credentials(agent_id) ON DELETE CASCADE,
    cidr       CIDR        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_api_allowed_ips UNIQUE (agent_id, cidr),
    CONSTRAINT ck_api_allowed_ips_v4 CHECK (family(cidr) = 4)
);

CREATE TABLE api_credential_logs (
    id               BIGSERIAL    PRIMARY KEY,
    agent_id         BIGINT       NOT NULL REFERENCES user_agents(id),
    actor_type       VARCHAR(10)  NOT NULL,  -- AGENT / SUB
    actor_id         BIGINT       NOT NULL,
    actor_username   VARCHAR(71)  NOT NULL,
    old_callback_url VARCHAR(500),
    new_callback_url VARCHAR(500),
    old_ips          TEXT[]       NOT NULL,
    new_ips          TEXT[]       NOT NULL,
    ip               VARCHAR(45),
    request_id       VARCHAR(64),
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_api_credential_logs_agent ON api_credential_logs(agent_id, created_at);
```

- `api_credentials` ใช้ได้ทั้ง Company และ Share เพราะทุกเจ้าของอยู่ใน `user_agents` (PK = `agent_id` ของเจ้าของ)
- `idx_api_allowed_ips` ไม่ต้องเพิ่ม — `uq_api_allowed_ips` เริ่มด้วย `agent_id` ใช้แทน index ได้
- config ใหม่ `API_KEY_ENCRYPTION_KEY` (32 byte เข้ารหัส base64 · required) ใน `pkg/configs` และ `.env.example`

## 7. Test cases

| Rule | Input | Expected |
|---|---|---|
| ACC-31 | `effective_status = SUSPENDED` เรียก profile | สำเร็จ |
| ACC-31 | `effective_status = SUSPENDED` เรียก dashboard / POST api-credential / route อื่นนอก Profile, Report | ถูกปฏิเสธ (AUTH-54) |
| ACC-21 | Company A ดู dashboard | นับเฉพาะ Member ในสาย A |
| ACC-22 | `currency` ที่บัญชีใช้ไม่ได้ / ไม่ส่ง | `422` |
| ACC-24 | ช่วงก่อนหน้ามียอด 0 | `change_pct = null` |
| ACC-25 | Member แพ้ 1,000 · คนที่ login ถือ 20% | `win_loss_pt` บวก 200 (ไม่รวม Commission) |
| ACC-26 | Member ในสาย 15 คน | Top 10 แต่ละตารางมี 10 แถว เรียงจากมากไปน้อย |
| ACC-11 | Agent / SUPERADMIN / ADMIN เรียก profile | ข้อมูลของตัวเอง · `is_subaccount = false` · `owner_username = null` |
| ACC-11 | ยังไม่ตั้ง passcode เรียก profile | `401304` (ด่านหลัง login) |
| ACC-12 | ผู้สร้างของ sub เป็น SUSPENDED | `status = ACTIVE` · `effective_status = SUSPENDED` |
| ACC-12 | Company Seamless Master เรียก profile | `user_type = COMPANY_SEAMLESS_MASTER` · `pt.hold_bp = 0` · `pt.release_bp = 10000` |
| ACC-15 | sub ของ Share B2C เรียก profile | `user_type`, `currencies`, `pt` เท่าของผู้สร้าง · `permissions` = ที่ผู้สร้างให้ |
| ACC-16 | SUPERADMIN เรียก profile | `pt = null` |
| ACC-14 | login แล้วเรียก profile | `last_login_at` / `last_login_ip` ตรงกับ login นี้ |
| ACC-01 | Company Seamless 1 to 1 / Share Master / Share Reseller เรียก GET | สำเร็จ · ได้ Key ของตัวเอง |
| ACC-01 | Share Reseller 2 บัญชีใต้ Company เดียวกัน | ได้ Key คนละค่า |
| ACC-01 | Company Seamless Reseller / Master, Company Transfer, Share B2B / B2C, Agent, Superadmin เรียก GET / POST | `403301` |
| ACC-02 | sub ที่ได้สิทธิ์ดูอย่างเดียว | GET สำเร็จ · POST ถูกปฏิเสธ (error สิทธิ์) |
| ACC-02 | sub ที่ไม่ได้รับสิทธิ์ | GET / POST ถูกปฏิเสธ |
| ACC-02 | เจ้าของ SUSPENDED กดบันทึก | ถูกปฏิเสธ (ACC-31) |
| ACC-03 | เจ้าของ GET ครั้งแรก | Key hex ตัวพิมพ์เล็ก 64 ตัว |
| ACC-04 | GET ซ้ำ | Key เดิม · header `Cache-Control: no-store` · DB ไม่มี Key แบบ plain |
| ACC-05 | GET พร้อมกัน 2 คำขอครั้งแรก | ได้ Key เดียวกันทั้งคู่ · มี credential 1 แถว |
| ACC-06 | `http://...` / `ftp://...` / `https://` ไม่มี host / ยาว 501 | `422` |
| ACC-06 | `callback_url` เว้นว่าง | สำเร็จ · GET ได้ `null` |
| ACC-07 | `2001:db8::1` / `300.1.1.1` / `1.2.3.4/33` | `422` |
| ACC-07 | 51 รายการ / IP ซ้ำ (`1.2.3.4` กับ `1.2.3.4/32`) | `422` |
| ACC-07 | `1.2.3.4` | GET ได้ `1.2.3.4/32` |
| ACC-08 | ไม่ส่ง `passcode` / ผิด | `422` / `401204` |
| ACC-08 | บันทึกรายการใหม่ | IP เดิมที่ไม่อยู่ในรายการใหม่ถูกลบ |
| ACC-09 | บันทึกสำเร็จ | มีแถว `api_credential_logs` ค่าเก่า / ใหม่ถูกต้อง · ไม่มี Key |

## 8. Contract changes (แจ้ง frontend)

- เส้นใหม่ 4 เส้นตามหัวข้อ 5
- ตัดสินสิทธิ์การเข้าใช้งานจาก `effective_status` (ACC-31) · `status` ใช้แสดงว่าบัญชีนี้ถูกตั้งสถานะเองหรือไม่
- Profile มี `user_type`, `currencies`, `pt` (object ชุดเดียว ใช้กับทุกเกม), `permissions` — ใช้แสดงผล / เช็คเบื้องต้นในฟอร์มได้ หลังบ้านเช็คซ้ำเสมอ
- หน้า 1.3 แสดงเฉพาะ Company Seamless 1 to 1 / Share Master / Share Reseller และ sub ที่ได้รับสิทธิ์
- เครดิตแยกสกุลยังไม่มีใน Profile — เพิ่มตอนทำ module การชำระเงิน

## 9. การตัดสินใจ (2026-10-05)

1. Key ดูซ้ำได้ (ตามภาพตัวอย่าง) แทนการแสดงครั้งเดียว → เก็บแบบเข้ารหัส ถอดกลับได้
2. Key ค่าเดียว (ไม่แยก key / secret) · สร้างใหม่ไม่ได้
3. ไม่มี IP = ปิดการเรียก API · รองรับ IPv4 และ CIDR · สูงสุด 50 รายการ
4. **review 2026-10-05:**
   - เจ้าของ Key เปลี่ยนจาก "1 Key ต่อ Company Seamless" เป็น Company Seamless 1 to 1 / Share Master / Share Reseller บัญชีละ Key
   - คง `status` และ `effective_status` แยก 2 field (เหตุผล: แยก "ถูกระงับเอง" กับ "ได้รับผลจากหัวสาย") · การเข้าใช้งานตัดสินจาก `effective_status`
   - Profile ส่งประเภท, สกุลเงิน, ค่าหุ้นส่วน (key `pt` เดียว ชุดเดียว — หน้าบ้านตั้งที่เดียว ระบบใช้กับทุกเกม) และสิทธิ์
   - ระบบสิทธิ์กำหนดใน module ② ครั้งเดียว · Profile และ 1.3 ใช้ตามนั้น
   - Dashboard กลับมาอยู่ใน module นี้ · spec ตอนนี้ implement หลังมี bet
5. **Error code** (module `account` = `bb=03`) — business error ตอบ HTTP 200

| Code | HTTP | ความหมาย |
|---|---|---|
| 403301 | 200 | บัญชีนี้ไม่มีข้อมูลรับรอง API |
| 403302 | 200 | บัญชีถูกระงับ ทำรายการไม่ได้ |
