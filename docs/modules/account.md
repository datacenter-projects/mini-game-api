# บัญชี (`account`) — Profile · ข้อมูลรับรอง API — Spec

- สถานะ: **APPROVED** · implement ได้: Profile · 1.3 รอคอลัมน์ `agent_type` จาก module ② (หัวข้อ 1)
- อนุมัติโดย: lead · วันที่: 2026-10-05
- ชื่อ module ในโค้ด: `account` (`controllers/account`, `dto/account`, `service/account`, `core/account`)
- เมนู: **1.2 ประวัติของฉัน (Profile)** · **1.3 ข้อมูลรับรอง API** · (1.1 ภาพรวม ย้ายไป module Home — Dashboard)
- ที่มาของ rule: ภาพหน้าจอตัวอย่างที่ user ส่ง + เอกสาร Seamless API Flow ของ lead + คำตอบของ boiledegg (2026-10-05) ·
  ไม่ได้ extract จากโค้ดเก่า

> เอกสารนี้คือ source of truth ของ module — โค้ดและ test ต้องตรงกับเอกสารนี้ ถ้าไม่ตรงให้แก้โค้ด
> หรือแก้เอกสารผ่านการอนุมัติใหม่ ห้ามอ้างเอกสารของโปรเจกต์เก่า

## 1. หน้าที่ของ module

หน้าข้อมูลบัญชีของผู้ใช้หลังบ้าน · รอบนี้:
- **Profile (1.2)** — ข้อมูลบัญชีของตัวเอง + ปุ่มเปลี่ยนรหัสผ่าน / passcode (ใช้ endpoint ของ `agent_auth`)
- **1.3 ข้อมูลรับรอง API** — Key, ลิงก์ตอบกลับ และ IP ที่อนุญาต ที่ลูกค้าแบบ Seamless ใช้เชื่อมระบบกับเรา (Seamless API Flow)

**ไม่อยู่ในรอบนี้:**
- 1.1 ภาพรวม — ย้ายไป module Home (Dashboard) · ข้อมูลมาจาก bet
- ใน Profile: ชื่อแสดง / ประเภท / สกุลเงิน (คอลัมน์มาจาก module 2) · ค่า PT / Force / Remain / Commission ต่อเกม (module 2) ·
  เครดิตแยกสกุล (module 5 การชำระเงิน) — เพิ่มเข้า Profile ตอนทำ module นั้น
- การใช้ Key / IP / ลิงก์ตอนลูกค้าเรียก API (Login, GetBalance, Bet, Settle ฯลฯ) — module provider (`app/externals/provider`)

**ต้องมีก่อน implement 1.3:** ประเภทของ Company (`agent_type` — Transfer / Seamless Reseller / Master / 1 to 1) มาจาก
module ② User Management · ยังไม่มีในระบบ จึง implement 1.3 ได้หลังคอลัมน์นี้มีแล้ว (Profile ไม่ติด)

## 2. คำศัพท์

| คำ | ความหมาย |
|---|---|
| เจ้าของ Key | Company แบบ Seamless (Reseller · Master · 1 to 1) — 1 Key ต่อ Company (ACC-01) |
| Key | ค่าลับที่ลูกค้าใช้ระบุตัวและคำนวณ `sign` ทุกคำขอ (Seamless API Flow) |
| ลิงก์ตอบกลับ | URL ของระบบลูกค้าที่ระบบเราเรียก (GetMember, GetBalance, Bet, Settle, Cancel) |
| IP ที่อนุญาต | IP ของลูกค้าที่เรียก API ของเราได้ |

## 3. Business rules

| ID | Rule |
|---|---|
| ACC-01 | เจ้าของ Key = **Company แบบ Seamless** (`SEAMLESS_RESELLER` · `SEAMLESS_MASTER` · `SEAMLESS_1TO1`) · **1 Key ต่อ Company** ใช้ร่วมกันทุกเว็บ / Share ใต้ Company นั้น · บัญชีอื่นทั้งหมด (Superadmin, ADMIN, Company Transfer, Share, Agent) เปิดหน้านี้ไม่ได้ (`403301`) |
| ACC-02 | เปิดหน้า / บันทึกได้: เจ้าของ และ sub ของเจ้าของ (sub ทำงานในนามเจ้าของ — AUTH-26) · ต้องผ่านด่านหลัง login (`PassedGates()`) · บันทึกต้อง `EffectiveStatus = ACTIVE` (`403302`) |
| ACC-03 | Key สร้างอัตโนมัติ 1 ค่าต่อเจ้าของ: สุ่ม 32 byte ด้วย `crypto/rand` แสดงเป็น hex ตัวพิมพ์เล็ก 64 ตัว · **สร้างใหม่ไม่ได้** · ใช้ทั้งระบุตัวและคำนวณ `sign` |
| ACC-04 | Key ดูซ้ำได้ (ปุ่มคัดลอก) → เก็บแบบเข้ารหัส AES-256-GCM ที่ถอดกลับได้ (ไม่ใช่ hash) + `sha256(Key)` ไว้ค้นตอนลูกค้าเรียก API · response ใส่ `Cache-Control: no-store` · ห้าม log Key |
| ACC-05 | สร้าง Key ตอนเปิดหน้าครั้งแรกถ้ายังไม่มี · เรียกพร้อมกันได้ Key เดียวเสมอ (unique ที่ `agent_id`) · ตอนทำ module 2 ให้สร้างตอนสร้างบัญชีเจ้าของด้วย |
| ACC-06 | ลิงก์ตอบกลับ: ต้องเป็น URL `https://` ที่มี host · ยาวไม่เกิน 500 ตัว · เว้นว่างได้ (= ยังไม่ตั้ง) |
| ACC-07 | IP ที่อนุญาต: IPv4 หรือช่วง CIDR ของ IPv4 (IP เดี่ยวเก็บเป็น `/32`) · ไม่เกิน **50** รายการ · ห้ามซ้ำ · **ไม่มีเลย = ลูกค้าเรียก API ของเราไม่ได้** |
| ACC-08 | บันทึก = แทนทั้งชุด (ลิงก์ + รายการ IP) · ต้องส่ง `passcode` ของผู้กด (`RequirePasscode`) · tx + `SELECT ... FOR UPDATE` แถว credential |
| ACC-09 | ทุกการบันทึกเก็บประวัติ `api_credential_logs` (ผู้แก้, ลิงก์เก่า → ใหม่, IP เก่า → ใหม่, ip, request_id, เวลา) ใน tx เดียวกัน · ห้ามเก็บ Key ในประวัติ |
| ACC-10 | ชื่อผู้ใช้ในหน้านี้ = username ของเจ้าของ (อ่านอย่างเดียว) |

### Profile

| ID | Rule |
|---|---|
| ACC-11 | ทุกบัญชีหลังบ้านเรียกได้ (SUPERADMIN, ADMIN, Company, Share, Agent และ sub) · ต้องผ่านด่านหลัง login (`PassedGates()`) · แสดงข้อมูลของ**ตัวเอง**เท่านั้น (sub เห็นของ sub) |
| ACC-12 | ข้อมูลที่แสดง: username · role (sub = role ของผู้สร้าง — AUTH-25) · สถานะของตัวเอง · effective status (AUTH-53) · เป็น sub ไหม · username ของผู้สร้าง (เฉพาะ sub) · ตั้ง passcode แล้วหรือยัง · login ล่าสุด (เวลา, IP) · วันที่สร้างบัญชี |
| ACC-13 | อ่านอย่างเดียว · เปลี่ยนรหัสผ่าน / passcode ของตัวเองใช้เส้นเดิม `POST /bo/pr/auth/password/change` และ `/auth/passcode/change` (agent_auth) |
| ACC-14 | login ล่าสุด = ค่าที่บันทึกตอน login (AUTH-12) จึงเป็นเวลาของ session ปัจจุบัน |

## 4. สิ่งที่พบในโค้ดเก่า และการตัดสินใจ

ไม่มี — ไม่ได้ extract จากโค้ดเก่า

## 5. Endpoints

ใต้ `/bo/pr` ผ่าน `Authenticated` + `PassedGates()` · error ร่วม: `401202`, `401203`, `401301`, `401302`, `401304`, `401306`, `401307`

| Method | Path | middleware เพิ่ม |
|---|---|---|
| GET | `/api/v1/bo/pr/account/api-credential` | — |
| POST | `/api/v1/bo/pr/account/api-credential` | `RequirePasscode` |
| GET | `/api/v1/bo/pr/account/profile` | — |

### GET /api/v1/bo/pr/account/profile

Response `data`:
```json
{
  "username": "agent01@staff",
  "role": "AGENT",
  "status": "ACTIVE",
  "effective_status": "SUSPENDED",
  "is_subaccount": true,
  "owner_username": "agent01",
  "passcode_set": true,
  "last_login_at": "2026-10-05T10:00:00+07:00",
  "last_login_ip": "203.0.113.10",
  "created_at": "2026-10-01T09:00:00+07:00"
}
```
`owner_username` = `null` เมื่อไม่ใช่ sub · `last_login_*` = `null` ถ้ายังไม่เคยบันทึก

Error codes: error ร่วมของ `/bo/pr` เท่านั้น

### GET /api/v1/bo/pr/account/api-credential

Response `data` (header `Cache-Control: no-store`):
```json
{
  "username": "seamlessreseller01",
  "key": "cb886b34f28b4e7964ca039335a86c74d04a3dea605e1b2c9d0f7a8b6c5d4e3f",
  "callback_url": "https://api.customer.example/minigame",
  "allowed_ips": ["203.0.113.10/32", "198.51.100.0/24"]
}
```
`callback_url` = `null` เมื่อยังไม่ตั้ง · `allowed_ips` = `[]` เมื่อยังไม่มี

Error codes: `403301`

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
`403302`, `401204`, `401205`

## 6. Schema

migration ไฟล์ใหม่

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

- `idx_api_allowed_ips` ไม่ต้องเพิ่ม — `uq_api_allowed_ips` เริ่มด้วย `agent_id` ใช้แทน index ได้
- config ใหม่ `API_KEY_ENCRYPTION_KEY` (32 byte เข้ารหัส base64 · required) ใน `pkg/configs` และ `.env.example`

## 7. Test cases

| Rule | Input | Expected |
|---|---|---|
| ACC-01 | Company Seamless (Reseller / Master / 1 to 1) เรียก GET | สำเร็จ · ได้ Key ของ Company |
| ACC-01 | Company Transfer / Share Reseller / Share Master / Agent / Superadmin เรียก GET / POST | `403301` |
| ACC-02 | sub ของเจ้าของ GET / POST | สำเร็จ · ได้ Key ของเจ้าของ |
| ACC-02 | เจ้าของ SUSPENDED กดบันทึก | `403302` · GET ยังดูได้ |
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
| ACC-11 | Agent / SUPERADMIN / ADMIN เรียก profile | ได้ข้อมูลของตัวเอง · `is_subaccount = false` · `owner_username = null` |
| ACC-11 | sub เรียก profile | username ของ sub · `owner_username` = ผู้สร้าง |
| ACC-11 | ยังไม่ตั้ง passcode เรียก profile | `401304` (ด่านหลัง login) |
| ACC-12 | ผู้สร้างของ sub เป็น SUSPENDED | `status = ACTIVE` · `effective_status = SUSPENDED` |
| ACC-14 | login แล้วเรียก profile | `last_login_at` / `last_login_ip` ตรงกับ login นี้ |

## 8. Contract changes (แจ้ง frontend)

- เส้นใหม่ 3 เส้นตามหัวข้อ 5 · หน้า 1.3 แสดงเฉพาะเจ้าของ Key และ sub ของเจ้าของ
- Profile รอบนี้ยังไม่มีชื่อแสดง / ประเภท / สกุลเงิน / ค่า PT ต่อเกม / เครดิต — field จะเพิ่มตอนทำ module 2 และ 5

## 9. การตัดสินใจ (2026-10-05)

1. Key ดูซ้ำได้ (ตามภาพตัวอย่าง) แทนการแสดงครั้งเดียว → เก็บแบบเข้ารหัส ถอดกลับได้
2. Key ค่าเดียว (ไม่แยก key / secret) · สร้างใหม่ไม่ได้
3. ไม่มี IP = ปิดการเรียก API · รองรับ IPv4 และ CIDR · สูงสุด 50 รายการ
4. **Error code** (module `account` = `bb=03`) — business error ตอบ HTTP 200

| Code | HTTP | ความหมาย |
|---|---|---|
| 403301 | 200 | บัญชีนี้ไม่มีข้อมูลรับรอง API |
| 403302 | 200 | บัญชีถูกระงับ ทำรายการไม่ได้ |
