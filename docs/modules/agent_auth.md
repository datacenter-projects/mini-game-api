# Backoffice Auth (`agent_auth`) — Spec

- สถานะ: **APPROVED** (หัวข้อ 9 ปิดแล้ว)
- อนุมัติโดย: lead (Zerph) · วันที่: 2026-10-02
- ชื่อ module ในโค้ด: `agent_auth` (`controllers/agent_auth`, `dto/agent_auth`, `service/agent_auth`, `core/agent_auth`)
- โค้ดเก่าที่อ้างอิง (extract พฤติกรรมเท่านั้น): `askmelotto_minigame-api/app/services/agent/login_*.go`,
  `app/internals/backoffice/middleware/jwt_middleware.go`, `pkg/auth/*`

## 1. หน้าที่ของ module

ยืนยันตัวตนของผู้ใช้หลังบ้าน (agent) สร้าง/ยกเลิก session และมี middleware ให้ route หลังบ้านใช้ตรวจว่า
request มาจาก session ที่ยังใช้งานได้

**อยู่ใน scope รอบนี้:** login, logout, middleware `Authenticated` · **ยังไม่อยู่:** subaccount login,
passcode, เปลี่ยนรหัสผ่าน, quick-start, การเข้าถึง agent ในสาย (`/agents/:agent_id`), role/permission guard

## 2. คำศัพท์

| คำ | ความหมาย |
|---|---|
| agent | บัญชีหลังบ้าน 1 แถวใน `user_agents` (ทุก role) |
| session | การ login 1 ครั้ง เก็บใน Redis ระบุด้วย `sid` |
| token | JWT ที่ client ส่งมาใน `Authorization: Bearer <token>` — มี `sid` อยู่ข้างใน |
| idle timeout | ไม่มี request เกินเวลานี้ session หลุด |
| absolute timeout | นับจาก login ครบเวลานี้ session หลุดเสมอ แม้ยังใช้งานอยู่ |

## 3. Business rules

| ID | Rule |
|---|---|
| AUTH-01 | username ถูก trim + แปลงเป็นตัวพิมพ์เล็กก่อนค้นหาเสมอ |
| AUTH-02 | ไม่พบ username หรือรหัสผ่านผิด → ตอบ error เดียวกัน (`401201`) และใช้เวลาตอบใกล้เคียงกัน (เทียบกับ dummy hash) — ไม่บอกว่าผิดที่ username หรือ password |
| AUTH-03 | รหัสผ่านเก็บเป็น bcrypt hash เท่านั้น |
| AUTH-04 | agent ที่ `status = LOCKED` login ไม่ได้ (`401301`) — เช็คหลังรหัสผ่านถูกแล้วเท่านั้น |
| AUTH-05 | ถ้ามี upline คนใดในสายเป็น `LOCKED` → login ไม่ได้ (`401302`) |
| AUTH-06 | 1 agent มี session ที่ใช้งานได้ได้ 1 session — login ใหม่ทำให้ session เดิมใช้ไม่ได้ทันที |
| AUTH-07 | session หลุดเมื่อไม่มี request เกิน **60 นาที** (idle) หรือครบ **12 ชั่วโมง** นับจาก login (absolute) — ทั้งสองค่าตั้งผ่าน env |
| AUTH-08 | ทุก request ที่ผ่าน middleware สำเร็จ ต่ออายุ idle ออกไปอีก 60 นาที (ไม่เกิน absolute) |
| AUTH-09 | logout ยกเลิก session ปัจจุบันทันที — เรียกซ้ำ/เรียกด้วย session ที่หลุดไปแล้ว (logout แล้ว, ถูกเตะ, idle หมด) ตอบสำเร็จเหมือนกัน · token ปลอม/เลยเวลา absolute ตอบ `401202` · logout ด้วย token เก่าไม่กระทบ session ปัจจุบัน |
| AUTH-10 | กันเดารหัสผ่าน: login ผิด **5 ครั้งภายใน 15 นาที** ต่อ username → username นั้น login ไม่ได้ **15 นาที** (`401303`) แม้รหัสถูก · login สำเร็จล้างตัวนับ · ไม่เปลี่ยน status ของบัญชี |
| AUTH-11 | กันยิงถี่: เรียก login เกิน **20 ครั้ง/นาที ต่อ IP** → `429` |
| AUTH-12 | login สำเร็จบันทึก `last_login_at`, `last_login_ip` — ถ้าบันทึกไม่ได้ ไม่ทำให้ login ล้ม (log warn) |
| AUTH-13 | ถ้าสร้าง session ใน Redis ไม่สำเร็จ → login ล้ม (`500`) ห้ามออก token ที่ไม่มี session รองรับ |
| AUTH-14 | middleware: token ไม่มี/ลายเซ็นผิด/หมดอายุ → `401202` · session ไม่มีแล้ว (logout/ถูกเตะ/หมดเวลา) → `401203` |
| AUTH-15 | middleware: agent ถูกเปลี่ยนเป็น `LOCKED` ระหว่างใช้งาน → request ถัดไปถูกปฏิเสธ (`401301`) และ session ถูกลบ |
| AUTH-16 | IP ตอนใช้งานไม่ตรงกับ IP ตอน login → **log warn เท่านั้น ไม่บล็อก** |

## 4. สิ่งที่พบในโค้ดเก่า และการตัดสินใจ

| # | โค้ดเก่าทำอะไร | การตัดสินใจ |
|---|---|---|
| 1 | JWT ล้วน + `token_epoch` ใน DB, ไม่มี logout | เปลี่ยนเป็น JWT + session ใน Redis มี logout จริง (AUTH-06/09) |
| 2 | bump epoch ล้มเหลว → ยังออก token ให้ (session เก่ายังใช้ได้) | ห้าม — AUTH-13 |
| 3 | token อายุคงที่ 1 ชม. ไม่มี refresh (`JWT_REFRESH_TIME` ประกาศไว้แต่ไม่ถูกใช้) | idle 60 นาที + absolute 12 ชม. (AUTH-07/08) |
| 4 | ไม่มีกันเดารหัสผ่าน | AUTH-10/11 |
| 5 | agent `SUSPENDED` login ได้ | ดูหัวข้อ 9 ข้อ 1 |
| 6 | เช็ค upline LOCKED ตอน login (ไล่สายทุกครั้ง) | คงไว้ (AUTH-05) — ระหว่างใช้งาน ดูหัวข้อ 9 ข้อ 2 |
| 7 | middleware โหลด agent ทั้งแถว + set Locals ~12 ค่า | โหลดเฉพาะที่ auth ใช้ set Locals เป็น struct เดียว (`Actor`) |
| 8 | ตอบ HTTP 401/403 จริงสำหรับ error auth | ดูหัวข้อ 9 ข้อ 3 |
| 9 | IP mismatch log อย่างเดียว | คงไว้ (AUTH-16) |
| 10 | login เส้นเดียวรับทั้ง agent และ subaccount (`agent@sub`) | คงแนวนี้ไว้ — รอบนี้ username ที่มี `@` ตอบ `401201` จนกว่าจะทำ module subaccount |

## 5. Endpoints

| Method | Path | ต้อง login | Contract เทียบของเก่า |
|---|---|---|---|
| POST | `/api/v1/bo/pb/auth/login` | ไม่ | path/body/field เหมือนเดิม |
| POST | `/api/v1/bo/pr/auth/logout` | ใช่ | **ใหม่** |

### POST /api/v1/bo/pb/auth/login

Request:
```json
{ "username": "agent01", "password": "••••••" }
```
Response `data`:
```json
{
  "token": "<jwt>",
  "username": "agent01",
  "role": "AGENT",
  "is_subaccount": false,
  "passcode_set": false,
  "expires_at": "2026-10-02T22:00:00+07:00"
}
```
Error codes: `422` (ไม่กรอก username/password หรือยาวเกิน — msg บอกว่าผิดที่ field ไหน), `401201`, `401301`, `401302`, `401303`, `429`, `500`

### POST /api/v1/bo/pr/auth/logout

Header `Authorization: Bearer <token>` · Response: `{"code":200,"msg":"สำเร็จ"}` (ไม่มี `data`) · Error codes: `401202`

### Middleware `Authenticated` (ใช้กับทุก route ใต้ `/bo/pr`)

1. อ่าน Bearer token → ตรวจลายเซ็น/หมดอายุ → อ่าน `sid`, `agent_id`
2. หา session ใน Redis ด้วย `sid` — ต้องมีอยู่และเป็นของ `agent_id` เดียวกัน
3. โหลด agent (`id, username, role, status, parent_id`) ด้วย primary key → เช็ค AUTH-15
4. ต่ออายุ idle (AUTH-08) แล้ว set `c.Locals("actor", Actor{...})`

## 6. Schema

```sql
CREATE TABLE user_agents (
    id              BIGSERIAL PRIMARY KEY,
    parent_id       BIGINT REFERENCES user_agents(id),
    username        VARCHAR(50)  NOT NULL,
    password_hash   VARCHAR(100) NOT NULL,
    role            VARCHAR(20)  NOT NULL,
    status          VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE',
    last_login_at   TIMESTAMPTZ,
    last_login_ip   VARCHAR(45),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_user_agents_username UNIQUE (username),
    CONSTRAINT ck_user_agents_status CHECK (status IN ('ACTIVE','SUSPENDED','LOCKED')),
    CONSTRAINT ck_user_agents_role CHECK (role IN ('SUPERADMIN','COMPANY','SHAREHOLDER','AGENT'))
);
CREATE INDEX idx_user_agents_parent_id ON user_agents(parent_id);
```

คอลัมน์อื่นของ agent (สกุลเงิน, integration, PT ฯลฯ) จะเพิ่มด้วย migration ใหม่ตอนทำ module สายงาน
role/status ใช้ค่าตามโค้ดเก่าไปก่อน — ความหมายของแต่ละ role จะกำหนดใน spec module สายงาน

Redis:

| Key | ค่า | TTL |
|---|---|---|
| `bo:sess:{sid}` | JSON `{agent_id, ip, created_at}` | idle 60 นาที (ต่ออายุทุก request, ไม่เกิน absolute) |
| `bo:sess:agent:{agent_id}` | `sid` ปัจจุบัน | absolute 12 ชม. |
| `bo:login:fail:{username}` | จำนวนครั้งที่ผิด | 15 นาที |
| `bo:login:ip:{ip}` | จำนวนครั้งที่เรียก | 1 นาที |

บัญชีแรก (SUPERADMIN) สร้างด้วย `go run ./scripts/create_superadmin` (รหัสผ่านอ่านจาก stdin ไม่รับผ่าน argument)

## 7. Test cases

| Rule | Input | Expected |
|---|---|---|
| AUTH-01 | login `"  Agent01 "` (บัญชี `agent01`) | สำเร็จ |
| AUTH-02 | username ไม่มีในระบบ | `401201` |
| AUTH-02 | รหัสผิด | `401201` |
| AUTH-04 | บัญชี LOCKED + รหัสถูก | `401301` |
| AUTH-04 | บัญชี LOCKED + รหัสผิด | `401201` (ไม่บอกว่าบัญชีถูกล็อก) |
| AUTH-05 | ปู่ LOCKED, ตัวเอง ACTIVE | `401302` |
| AUTH-06 | login ที่เครื่อง A แล้วเครื่อง B → ใช้ token A | `401203` |
| AUTH-07 | ไม่มี request 61 นาที | `401203` |
| AUTH-07 | ใช้งานต่อเนื่องจนครบ 12 ชม. | `401203` |
| AUTH-09 | logout แล้วใช้ token เดิม | `401203` |
| AUTH-09 | logout ซ้ำ 2 ครั้ง | สำเร็จทั้งคู่ |
| AUTH-10 | ผิด 5 ครั้ง แล้วใส่รหัสถูก | `401303` |
| AUTH-10 | ผิด 4 ครั้ง แล้วถูก แล้วผิดอีก 4 ครั้ง | ยัง login ได้ (ตัวนับถูกล้าง) |
| AUTH-11 | IP เดียวยิง 21 ครั้งใน 1 นาที | ครั้งที่ 21 ได้ `429` |
| AUTH-14 | ไม่ส่ง token / token ปลอมลายเซ็น | `401202` |
| AUTH-15 | ถูก LOCK ระหว่างใช้งาน | request ถัดไป `401301` และ session ถูกลบ |

## 8. Contract changes (แจ้ง frontend)

- เพิ่ม `POST /api/v1/bo/pr/auth/logout`
- response login: เพิ่ม `expires_at` (เวลาหลุดแบบ absolute) · `passcode_set` ส่ง `false` เสมอจนกว่าจะทำ module passcode
- token หลุดได้จาก idle 60 นาที (เดิมหลุดที่ 1 ชม. คงที่) — frontend ต้องพา user ไปหน้า login เมื่อได้ `401202` / `401203`
- **error code ชุดใหม่ทั้งหมด** (ไม่ใช้เลข 2600xx ของระบบเก่า) — ดูหัวข้อ 9
- เพิ่ม `401303` (ล็อกชั่วคราวจาก login ผิดหลายครั้ง) และ `429`

## 9. การตัดสินใจ (ปิดแล้ว 2026-10-02)

1. **SUSPENDED login ได้** ตามเดิม — ความหมายของ SUSPENDED (เช่น ดูได้แต่ทำรายการไม่ได้) กำหนดใน module สายงาน
2. **upline ถูก LOCK ระหว่างใช้งาน** — คำสั่ง LOCK (module สายงาน) ต้องลบ session ของทั้งสายใต้คนนั้น
   middleware จึงเช็คแค่สถานะของตัวเอง (AUTH-15) ไม่ไล่สายทุก request
3. **HTTP status**: `401201/401202/401203` → 401 · `401301/401302/401303` → 403 · error business อื่นของระบบยังตอบ 200
4. **error code นับใหม่** ตามรูปแบบ `abbcdd` (ดู `docs/ERROR_CODES.md`) module `agent_auth` = `bb=01`

| Code | HTTP | ความหมาย | เลขเดิม (อ้างอิงเท่านั้น) |
|---|---|---|---|
| 401201 | 401 | username หรือรหัสผ่านไม่ถูกต้อง | 260011 |
| 401202 | 401 | ไม่พบ token / token ไม่ถูกต้องหรือหมดอายุ | 260012 |
| 401203 | 401 | session สิ้นสุดแล้ว (logout / login ที่อื่น / ไม่ได้ใช้งานนาน) | 260042 |
| 401301 | 403 | บัญชีถูกล็อก | 260045 |
| 401302 | 403 | สายบัญชีต้นสังกัดถูกล็อก | 260047 |
| 401303 | 403 | login ผิดหลายครั้ง ระงับชั่วคราว | — |

5. **`passcode_set` ส่ง `false`** ไว้ก่อน จะเปลี่ยนเป็นค่าจริงตอนทำ module passcode
