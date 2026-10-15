# Player Auth (ผู้เล่นเข้าระบบ) — Spec

- สถานะ: **DRAFT** (รอ lead review ทีละ endpoint — หัวข้อ 9 คือคำถามที่ต้องตัดสินก่อน implement)
- อนุมัติโดย: — · วันที่: —
- ผู้ทำ: Boiledegg · branch `boiledegg/bo/player_auth`
- ชื่อ module ในโค้ด: `player_auth` (`app/internals/frontend/{controllers,dto}/player_auth`, `app/externals/transfer/{controllers,dto,middleware}`, `app/service/player_auth`, `app/core/player_auth`)
- ที่มาของการตัดสิน: [docs/reviews/player-auth.md](../reviews/player-auth.md) (lead zerph 2026-10-14 / 15 · PQ1–PQ14 · LQ-c)
- โค้ดเก่าที่อ้างอิง (extract พฤติกรรมเท่านั้น): `askmelotto_minigame-api` — `/provider/launch` (สร้างผู้เล่น ไม่ให้ token ไม่มีรหัส **ห้ามลอกแบบ**) ·
  แบบที่ใกล้สุดคือ transferwallet `loginRequest` / sportbook `redirectLogin` ของ `askmelotto-api`

> เอกสารนี้คือ source of truth ของ module — โค้ดและ test ต้องตรงกับเอกสารนี้
> ข้อความที่ขึ้นต้น **[ร่าง]** คือส่วนที่ review ยังไม่ได้ตัดสิน ผู้ร่างเสนอไว้เพื่อให้ lead ยืนยัน / แก้ — ห้าม implement จนกว่าจะปิดหัวข้อ 9

## 1. หน้าที่ของ module

ยืนยันตัวตนของ**ผู้เล่น (Member)** ที่เข้าหน้าเกม สร้าง / ยกเลิก session และมี middleware ให้ route ฝั่งผู้เล่นตรวจว่า request มาจาก session ที่ยังใช้ได้
มี 2 ทางเข้า:

1. ผู้เล่น login เองด้วย username + password (`player/pb/login`) — ใช้ได้กับ Member ที่สร้างจากหลังบ้าน (BO)
2. server ของ operator สาย Transfer เรียก `transfer/login` ส่ง `agentUsername` + `key` + `playerUsername` ได้ token + link เข้า lobby กลับไป · ผู้เล่นที่ยังไม่มี = สร้างให้อัตโนมัติ

**รอบนี้ทำเฉพาะสาย Transfer** — Seamless (`seamless/*` · GetMember · mock operator) พักไว้ แต่ออกแบบ middleware / session ให้ใช้ร่วมกันได้

**อยู่ใน scope รอบนี้ (7 เส้น + งานประกอบ):** PA1 `player/pb/login` · PA3 `logout` · PA4 `password/update` · PA5 `me/get` · PA7 `games/list` · L2 `transfer/login` · TR `transfer/register` · middleware ผู้เล่น · middleware operator · `members/status/update` → LOCKED ลบ session

**ยังไม่อยู่:** `transfer/launch` (L3 — lead 2026-10-15) · `player/pb/launch/exchange` (ยกเลิก — LQ-c) · `seamless/*` · GetMember · balance / deposit / withdraw ของ operator (spec "Transfer API" แยก) · BO reset รหัสผู้เล่น · เปลี่ยน username ของ Member ที่สร้างจาก BO · bundle asset ของหน้าเกม (Schneider — GE22-2) · maintenance ทั้งระบบ (spec แยก)

## 2. คำศัพท์

| คำ | ความหมาย |
|---|---|
| ผู้เล่น / Member | แถวใน `user_members` |
| ผู้สร้าง (`agent_id`) | บัญชีฝั่ง agent ที่เป็นแม่ตรงของ Member |
| operator | ลูกค้าที่มี server เชื่อมระบบเรา ถือ `agentUsername` + Key ของ**ผู้สร้าง**ผู้เล่น (account.md ACC-01) |
| `agentUsername` / `key` | username และ Key ของผู้สร้าง (`api_credentials`) ที่ operator ส่งมาใน body |
| `playerUsername` | ชื่อผู้เล่นที่ operator ส่งมา (ยังไม่มี prefix) |
| username ผู้เล่นจาก operator | `{agentUsername}@{playerUsername}` — ระบบต่อ prefix ให้เอง |
| token | JWT `typ = "pl"` ใน `Authorization: Bearer <token>` — มี `sid` อยู่ข้างใน |
| session | การเข้าระบบ 1 ครั้ง เก็บใน Redis `pl:sess:*` |
| สถานะที่ใช้งานจริง | สถานะที่แย่ที่สุดของ Member + ผู้สร้าง + สายบนทุกคน (`LOCKED` > `SUSPENDED` > `ACTIVE`) |
| link lobby | URL หน้าเกมที่มี token ต่อท้าย (LQ-c) |

## 3. Business rules

แต่ละข้อมี id เพื่ออ้างใน test และ code (`// rule: PA-03`)

### 3.1 ทางเข้า / ตัวตน

| ID | Rule |
|---|---|
| PA-01 | path ผู้เล่น `/api/v1/player/pb/*` (public) · `/api/v1/player/pr/*` (ต้อง login) · ทุกเส้นเป็น `POST` (N1) · `agent_id` / `member_id` อ่านจาก session เท่านั้น **ห้ามรับจาก body** |
| PA-02 | ทางเข้า operator อยู่ใต้ group `/api/v1/transfer/*` (register แยกจาก `seamless/*` — ไม่มี prefix `ext`) · controller / middleware อยู่ `app/externals/transfer/` (trust boundary แยก — CLAUDE.md ข้อ 29 ห้ามใช้ middleware auth ของ internals) |
| PA-03 | `player/pb/login` ใช้ได้ทุกสาย · ใช้กับ Member ที่มีรหัสผ่านเท่านั้น (สร้างจาก BO) · ผู้เล่นจาก operator `password_hash = ''` → login ตรงไม่ได้ ตอบ `407201` เหมือนรหัสผิด (ไม่บอกว่าไม่มีรหัส) |
| PA-04 | username ที่รับทางตรง (`player/pb/login`): trim + ตัวพิมพ์เล็กก่อนค้นหา · ไม่พบ หรือรหัสผิด → `407201` ตัวเดียว และเวลาตอบใกล้เคียงกัน (เทียบ dummy hash เหมือน AUTH-02) |
| PA-05 | username ผู้เล่นจาก operator = `{agentUsername}@{playerUsername}` · `agent` ต้องเป็นเจ้าของ `agentUsername` / `key` ที่เรียก (= แม่ตรงของผู้เล่น) · ส่วน player: `a-z0-9` ยาว 3–32 (รับตัวใหญ่ เก็บตัวเล็ก) · ไม่ผ่าน = `422` บอก field `player_username` |
| PA-06 | `user_members.username` ขยายเป็น `VARCHAR(65)` (32 + `@` + 32) · ห้ามซ้ำทั้งระบบเหมือนเดิม (MGMT-05) |
| PA-07 | `transfer/login`: ไม่มีผู้เล่น = **สร้างอัตโนมัติ** แล้ว login ต่อในคำขอเดียว · มีแล้ว = login เลย · ผู้เล่นที่สร้างให้ใช้ค่าตั้งต้นของ agent (PT / commission / win limit แบบที่ BO สร้าง Member โดยไม่กรอกค่า) · `password_hash = ''` |
| PA-08 | `transfer/register`: สร้างผู้เล่นอย่างเดียว ไม่ออก token · มีแล้ว = `407401` "มีผู้เล่นนี้แล้ว" (สร้างพร้อมกัน 2 คำขอ: ตัวที่แพ้ unique constraint ได้ `407401` · login ที่แพ้ให้ select ใหม่แล้วทำต่อ — ห้ามคืน 500) |
| PA-09 | Member ที่สร้างจาก BO ยังใช้ username แบบเดิม (MGMT-05) — รอบนี้**ไม่แก้** `members/create` (lead ยังไม่ตัดสินว่าจะเปลี่ยนเป็น `agent@player` ไหม) |

### 3.2 operator ยืนยันตัว (PQ4)

| ID | Rule |
|---|---|
| PA-10 | middleware operator (ใช้กับ `transfer/*` ทุกเส้น และ `seamless/*` ภายหลัง): ตรวจ `agentUsername` + `key` ใน body → หาเจ้าของ Key → ตรวจ IP ผู้เรียกอยู่ใน `api_allowed_ips` ของเจ้าของ (ACC-07) · ไม่มี `sign` |
| PA-11 | `agentUsername` ไม่มี / `key` ไม่ตรงเจ้าของ → `407206` (ข้อความเดียวกันทั้งสองกรณี ไม่บอกว่าผิดที่ไหน) · IP ไม่อยู่ใน whitelist → `407305` |
| PA-12 | **mask `key` ใน log ทุกจุด** (กฎข้อ 26) รวม request log กลางและ audit · ห้ามใส่ `key` ใน error message |
| PA-13 | เจ้าของ Key ต้อง `ACTIVE` และสายบนไม่ถูก `LOCKED` · ไม่ผ่านใช้ code เดียวกับสถานะผู้เล่น (`407301` / `407302`) |
| PA-14 | ผู้สร้างที่เรียก `transfer/*` ต้องอยู่สาย Transfer และเป็นประเภทที่สร้าง Member ได้ — ไม่ตรง `407306` · **[ร่าง]** ดูหัวข้อ 9 ข้อ 1 |

### 3.3 session (PQ5 / PQ6)

| ID | Rule |
|---|---|
| PA-20 | session แบบ BO: JWT `typ = "pl"` (secret เดียวกับ BO) · เก็บ Redis `pl:sess:*` · middleware ผู้เล่นปฏิเสธ token `typ` อื่น และ middleware BO ปฏิเสธ `typ = "pl"` (กัน token ข้ามฝั่ง) |
| PA-21 | หมดอายุ: ไม่มี request เกิน **30 นาที** (idle) หรือครบ **4 ชั่วโมง** นับจาก login (absolute) · ค่าตั้งผ่าน env (เพิ่ม key ใน `config.go` + `.env.example` PR เดียวกัน) · ไม่มี refresh — ทุก request ที่ผ่าน middleware ต่ออายุ idle (ไม่เกิน absolute) · หน้าเกม ping ระหว่างเล่น |
| PA-22 | **1 session ต่อ Member** — login (ทางไหนก็ตาม) ใหม่เตะอันเก่าทันที · token เก่าได้ `407204` "ถูก login จากที่อื่น" |
| PA-23 | ถ้าสร้าง session ใน Redis ไม่สำเร็จ → login ล้ม (`500`) ห้ามออก token ที่ไม่มี session รองรับ (เหมือน AUTH-13) |
| PA-24 | token ไม่มี / ลายเซ็นผิด / หมดอายุ → `407202` · session ไม่มีแล้ว (logout / idle / absolute) → `407203` |
| PA-25 | บันทึก `last_login_at` / `last_login_ip` ตอน login (ไม่ผูก session กับ IP / device) · บันทึกไม่ได้ไม่ทำให้ login ล้ม (log warn) · IP ตอนใช้งานไม่ตรงกับตอน login ไม่ block ไม่ต้อง log |
| PA-26 | logout (`pr/auth/logout`) ยกเลิก session ปัจจุบัน · idempotent: เรียกซ้ำ / session หลุดไปแล้ว ตอบสำเร็จเหมือนกัน · token ปลอม / เลย absolute `407202` · logout ด้วย token เก่าไม่กระทบ session ปัจจุบัน |

### 3.4 สถานะ (PQ7) — สถานะแย่ที่สุดของ Member + ผู้สร้าง + สายบนทุกคน

| สถานะที่ใช้งานจริง | login / launch | ใช้งานหลัง login | เดิมพัน | session |
|---|---|---|---|---|
| `LOCKED` (ตัวเองหรือสายบน) | ❌ `407301` / `407302` | ❌ | ❌ | ลบทันที |
| `SUSPENDED` | ✅ | ✅ ดูข้อมูล / ประวัติ / lobby | ❌ | คงไว้ |
| `ACTIVE` | ✅ | ✅ | ✅ (ถ้าเกมเปิด) | |

| ID | Rule |
|---|---|
| PA-30 | middleware ผู้เล่นเช็คสถานะตัวเอง + สายบน**ทุก request** (ไม่ไล่ลบ Member เมื่อ agent ถูก LOCKED — เช็คตอนใช้งานแทน) |
| PA-31 | login เช็คสถานะ**หลัง**รหัสผ่านถูก (เหมือน AUTH-04) — รหัสผิดได้ `407201` ไม่บอกว่าบัญชีถูกล็อก · `transfer/login` ไม่มีรหัส จึงเช็คสถานะหลัง Key ผ่าน |
| PA-32 | LOCKED ระหว่างใช้งาน → request ถัดไป `407301` (หรือ `407302` ถ้าสายบนถูกล็อก) และลบ session |
| PA-33 | `SUSPENDED` ผ่าน middleware ได้ · `can_bet = false` ใน `me/get` · module เดิมพัน (Bet) ต้องเช็คเอง · SUSPENDED cash-out Streak ที่ค้างได้ (spec Bet) |
| PA-34 | `members/status/update` ตั้ง `LOCKED` → **ลบ session ผู้เล่นหลัง commit** · Redis ล่ม = log warn ไม่ทำให้ request ล้ม (middleware เช็คสถานะทุก request อยู่แล้ว จึงยังกันได้) · ปิดข้อ MGMT-30A (open item MQ7) |
| PA-35 | `can_bet` = สถานะที่ใช้งานจริงเป็น `ACTIVE` และ `win_limit_per_day > 0` (MGMT-65) · **[ร่าง]** ดูหัวข้อ 9 ข้อ 6 |

### 3.5 กันเดารหัส / ยิงถี่ (แบบ BO)

| ID | Rule |
|---|---|
| PA-40 | `player/pb/login`: เรียกเกิน **20 ครั้ง/นาที ต่อ IP** → `429` |
| PA-41 | login ผิด **5 ครั้งภายใน 15 นาที** ต่อ username → username นั้น login ไม่ได้ **15 นาที** (`407303`) แม้รหัสถูก · login สำเร็จล้างตัวนับ · ไม่เปลี่ยน status ของบัญชี |
| PA-42 | `transfer/*`: **[ร่าง]** จำกัดต่อ IP แบบ PA-40 (key คนละตัว) — ดูหัวข้อ 9 ข้อ 8 |

### 3.6 เปลี่ยนรหัสผ่าน / ข้อมูลผู้เล่น

| ID | Rule |
|---|---|
| PA-50 | ผู้เล่นเปลี่ยนรหัสเองได้ (`pr/auth/password/update`) · ส่งรหัสเดิม + รหัสใหม่ · รหัสเดิมผิด `407205` · รหัสใหม่ตามกติกา AUTH-36 (ใช้ฟังก์ชันใน `app/core` ตัวเดียวกับ BO) · ต้องไม่ซ้ำรหัสเดิม |
| PA-51 | ผู้เล่นจาก operator ไม่มีรหัส → เปลี่ยนรหัสไม่ได้ `407307` · **[ร่าง]** |
| PA-52 | เปลี่ยนรหัสสำเร็จ "เตะ session อื่น" (PQ8–PQ14) — ดูหัวข้อ 9 ข้อ 10 เรื่องความหมายเมื่อมี 1 session ต่อคน |
| PA-53 | BO reset รหัสผู้เล่น **ยังไม่ทำ** |
| PA-54 | `me/get` คืนโปรไฟล์ + ยอด + สถานะ · ยอดอ่านจาก `user_members.credit` (ไม่ใช้ cache — กฎข้อ 14) |
| PA-55 | `games/list` ต้อง login (GQ9) · แสดงเกมทั้งหมดของ lobby พร้อมสถานะ เปิด / ปิด และ**รูปปก** (`game_cover` — อ่านจาก asset ตาม AST-51) · เกมปิดแสดงเป็นเกมปิด ไม่ซ่อน · ไม่มีปก = `game_cover = ""` |

### 3.7 เกมปิด

| ID | Rule |
|---|---|
| PA-60 | เกมปิดเช็คตอนเดิมพัน (Bet) และตอน launch (นอก scope) · lobby แสดงเป็นเกมปิด |

## 4. สิ่งที่พบในโค้ดเก่า และการตัดสินใจ

| # | โค้ดเก่าทำอะไร | ถูก/ผิด | การตัดสินใจ |
|---|---|---|---|
| 1 | `/provider/launch` สร้างผู้เล่นอย่างเดียว ไม่ให้ token ไม่มีรหัส → ผู้เล่นที่มาจาก launch login ไม่ได้ | ผิด | ไม่ลอก — `transfer/login` ให้ token + link เลย (PA-07) |
| 2 | askmelotto-api ยังไม่เคยต่อ minigame | — | ใช้แบบ transferwallet `loginRequest` / sportbook `redirectLogin`: `agentUsername` + `key` + `playerUsername` → link (PQ4) |
| 3 | `user_members.password_hash NOT NULL` · `username VARCHAR(32)` | — | ผู้เล่น operator เก็บ `''` · ขยาย username เป็น 65 (PA-06 / PA-07) |
| 4 | BO auth ใช้ JWT `typ` + Redis session 1 ต่อบัญชี + rate limit + สถานะสายบน | ใช้ซ้ำได้ | ใช้โครงเดียวกัน แยก key `pl:*` และ `typ = "pl"` (PA-20 – PA-41) |
| 5 | ตั๋วใช้ครั้งเดียว 60 วินาที + `launch/exchange` (ร่างเดิม) | ยกเลิก | link ใส่ token ใน URL ตรงๆ (LQ-c) — มาตรการลดความเสี่ยงอยู่หัวข้อ 5 |

## 5. Endpoints

| ID | Method | Path | ผู้เรียก / สิทธิ์ | หมายเหตุ |
|---|---|---|---|---|
| PA1 | POST | `/api/v1/player/pb/login` | ผู้เล่น · public + rate limit | login ตรง |
| PA3 | POST | `/api/v1/player/pr/auth/logout` | ผู้เล่น · middleware ผู้เล่น | idempotent |
| PA4 | POST | `/api/v1/player/pr/auth/password/update` | ผู้เล่น · middleware ผู้เล่น | |
| PA5 | POST | `/api/v1/player/pr/me/get` | ผู้เล่น · middleware ผู้เล่น (SUSPENDED ผ่าน) | |
| PA7 | POST | `/api/v1/player/pr/games/list` | ผู้เล่น · middleware ผู้เล่น (SUSPENDED ผ่าน) | |
| L2 | POST | `/api/v1/transfer/login` | operator Transfer · `agentUsername` + `key` + IP whitelist | สร้างผู้เล่นให้ถ้ายังไม่มี |
| TR | POST | `/api/v1/transfer/register` | operator Transfer · เหมือน L2 | |

สิทธิ์ของแต่ละ route ต้องเขียนไว้บรรทัดเดียวกับ route ใน `routes.go` (กฎข้อ 28) · ทุกเส้นตอบผ่าน `pkg/response` · request ผ่าน `utils.ParseBodyNoNull` + `Validate()` (กฎข้อ 19)

### PA1 · POST /api/v1/player/pb/login

Request:
```json
{ "username": "mem01", "password": "••••••" }
```
Response `data` **[ร่าง — field ของ `member` รอ lead]**:
```json
{
  "token": "<jwt>",
  "expires_at": "2026-10-15T16:00:00+07:00",
  "member": { "username": "mem01", "name": "mem01", "currency": "THB", "status": "ACTIVE", "can_bet": true }
}
```

| Field | Type | ความหมาย | ตัวอย่าง |
|---|---|---|---|
| `token` | string | JWT `typ = "pl"` ใส่ `Authorization: Bearer <token>` ทุกเส้น `pr/*` | `eyJhbGciOi…` |
| `expires_at` | string (RFC 3339 +07:00) | เวลาหมดอายุ**แบบ absolute** (login + 4 ชม. — PA-21) · idle 30 นาทีไม่รวมในค่านี้ ใช้งานแล้วต่ออายุเอง ไม่เกินค่านี้ | `2026-10-15T16:00:00+07:00` |
| `member.username` | string | username ที่ใช้ login (Member จาก BO = ชื่อเดิม) | `mem01` |
| `member.name` | string | ชื่อที่แสดง | `mem01` |
| `member.currency` | string | สกุลเงินของผู้เล่น (ISO) | `THB` |
| `member.status` | string `ACTIVE` / `SUSPENDED` | สถานะที่ใช้งานจริง (แย่สุดของ Member + สายบน — หัวข้อ 3.4) · `LOCKED` ไม่มีในนี้เพราะ login ไม่ผ่าน | `ACTIVE` |
| `member.can_bet` | bool | เดิมพันได้ไหม (PA-35) · `false` เมื่อ `SUSPENDED` | `true` |
Error codes: `422` · `407201` · `407301` · `407302` · `407303` · `429` · `500`

### PA3 · POST /api/v1/player/pr/auth/logout

Header `Authorization: Bearer <token>` · Response: `{"code":200,"msg":"สำเร็จ"}` (ไม่มี `data`) · Error codes: `407202`

### PA4 · POST /api/v1/player/pr/auth/password/update

Request: `{ "old_password": "••••••", "new_password": "••••••" }` · Response: `{"code":200,"msg":"สำเร็จ"}`
Error codes: `422` (รหัสใหม่ไม่ผ่าน AUTH-36 — msg บอก field) · `407202` · `407203` · `407205` · `407307` · `407301`

### PA5 · POST /api/v1/player/pr/me/get

Request: `{}` · Response `data` **[ร่าง]**:
```json
{ "username": "agent01@p001", "name": "p001", "currency": "THB", "credit": 1250.5, "status": "SUSPENDED", "can_bet": false }
```
| Field | Type | ความหมาย | ตัวอย่าง |
|---|---|---|---|
| `username` | string | username ผู้เล่น (ผู้เล่นจาก operator = `agent@player`) | `agent01@p001` |
| `name` | string | ชื่อที่แสดง | `p001` |
| `currency` | string | สกุลเงิน (ISO) | `THB` |
| `credit` | number | ยอดเงินคงเหลือ หน่วยสกุล ส่งค่าที่เก็บตามจริง ไม่ปัดเพิ่ม (กฎข้อ 9) · หน้าบ้านปัด 3 ตำแหน่งตอนแสดงเอง | `1250.5` |
| `status` | string `ACTIVE` / `SUSPENDED` | สถานะที่ใช้งานจริง (หัวข้อ 3.4) | `SUSPENDED` |
| `can_bet` | bool | เดิมพันได้ไหม (PA-35) | `false` |
Error codes: `407202` · `407203` · `407204` · `407301` · `407302`

### PA7 · POST /api/v1/player/pr/games/list

Request: `{}` · Response `data` **[ร่าง — แหล่งข้อมูลและ field รอ lead]**:
```json
{
  "items": [
    { "game_code": "scratch_card", "open": true,  "game_cover": "https://cdn.example.com/assets/scratch_card/game_cover/3f9a….png" },
    { "game_code": "coin_toss",    "open": false, "game_cover": "" }
  ]
}
```

| Field | Type | ความหมาย | ตัวอย่าง |
|---|---|---|---|
| `items` | array | เกมทั้งหมดของ lobby เรียงตามลำดับที่หน้าเกมแสดง (เกมปิดไม่ซ่อน — PA-55) · ไม่มีเกม = `[]` | |
| `items[].game_code` | string | รหัสเกม ตรงกับ `game_code` ของ game_rate | `scratch_card` |
| `items[].open` | bool | เปิดให้เล่นไหม (เปิดจริงตามสายบนของผู้สร้าง) · `false` = แสดงเป็นเกมปิด เดิมพันไม่ได้ (PA-60) | `true` |
| `items[].game_cover` | string (URL) | **รูปปกเกม** = ช่อง `game_cover` ของเกมนั้น ไล่สายจากผู้สร้างตรงของผู้เล่นขึ้นไปทีละช่อง (ตัวเอง → ชั้นบนที่ใกล้สุด → GLOBAL — asset AST-25 / AST-51 / AST-57) · ผู้เล่นเห็นแบบเดียวกับที่ BO preview เห็น · URL = `S3_CDN_BASE_URL` + storage key (AST-49) · เกมที่ยังไม่มีปก หรืออ่านปกไม่สำเร็จ = `""` ตอบสำเร็จ ไม่ใช่ error (AST-46 / ACC-32) + log warn | `https://cdn.example.com/…/game_cover/3f9a….png` |
| ชื่อเกม / อัตรา / รูปช่องอื่น | — | **ยังไม่อยู่ใน response** — รูปช่องอื่นเป็นของ bundle (`/player/asset/bundle` — GN23 · asset AST-48 · Schneider ทำหลัง middleware ผู้เล่น merge) |  |
- ปกอ่านจาก **service ตัวเดียวกับ `games/list` ของ BO / `used/list`** (asset AST-51 — ห้ามเขียน query คู่ขนาน) โดยเป้าหมายมาจาก session ของผู้เล่นเท่านั้น (PA-01 / asset AST-52) · ทุกเกมใน query เดียว ไม่ N+1 (กฎข้อ 23)
- รูปปกขนาดตาม catalog ของช่อง `game_cover` (เช่น scratch_card 608×1080) — ขนาดตัดสินที่ asset ไม่ใช่ที่นี่
Error codes: `407202` · `407203` · `407204` · `407301` · `407302`

### L2 · POST /api/v1/transfer/login

Request:
```json
{ "agentUsername": "agent01", "key": "<64 hex>", "playerUsername": "p001" }
```
Response `data` **[ร่าง]**:
```json
{ "token": "<jwt>", "expires_at": "2026-10-15T16:00:00+07:00", "link": "https://<lobby>/?token=<jwt>" }
```

| Field | Type | ความหมาย | ตัวอย่าง |
|---|---|---|---|
| `token` | string | JWT ของผู้เล่น (เหมือน PA1) · operator เก็บไว้เรียกต่อเองได้ แต่ผู้เล่นใช้ผ่าน `link` | `eyJhbGciOi…` |
| `expires_at` | string (RFC 3339 +07:00) | หมดอายุแบบ absolute (login + 4 ชม.) | `2026-10-15T16:00:00+07:00` |
| `link` | string (URL) | link เข้า lobby ที่ต่อ `?token=<token>` ท้าย URL (LQ-c) · operator redirect ผู้เล่นไป link นี้ตรงๆ · ฐาน URL มาจาก env (ข้อ 9 ข้อ 12) | `https://<lobby>/?token=eyJ…` |
| (ไม่มี) `created` | — | **ไม่บอกว่าสร้างผู้เล่นใหม่หรือเดิม** — operator ไม่ต้องรู้ (PA-07) | |
- ชื่อ field ของ request ใช้ camelCase ตามแบบ askmelotto / sportbook ที่ operator คุ้น (`agentUsername`, `key`, `playerUsername`) — **[ร่าง]** ยืนยันกับ lead (ข้อ 9 ข้อ 9) เพราะส่วนอื่นของระบบใช้ snake_case
- มาตรการลดความเสี่ยง token อยู่ใน URL (LQ-c): หน้าเกมลบ token ออกจาก URL ทันทีหลังอ่าน (`history.replaceState`) · หน้าเกมตั้ง `Referrer-Policy: no-referrer` · **ห้าม log query string ของ URL หน้าเกมฝั่งเรา** (รวม access log ของ proxy ที่เราคุมได้) · token อายุตาม session และ login ใหม่เตะอันเก่า (PA-21 / PA-22)
Error codes: `422` · `407206` · `407305` · `407301` · `407302` · `407306` · `429` · `500`

### TR · POST /api/v1/transfer/register

Request: `{ "agentUsername": "agent01", "key": "<64 hex>", "playerUsername": "p001" }` · Response **[ร่าง]**: `{"code":200,"msg":"สำเร็จ","data":{"username":"agent01@p001"}}`

| Field | Type | ความหมาย | ตัวอย่าง |
|---|---|---|---|
| `username` | string | username ที่ระบบเก็บ = `{agentUsername}@{playerUsername}` (ตัวเล็กทั้งหมด) — operator เก็บไว้เทียบกับผู้เล่นฝั่งตัวเอง | `agent01@p001` |
| token / link | — | **ไม่ออก** — ได้ token ต้องเรียก `transfer/login` |  |
Error codes: `422` · `407206` · `407305` · `407301` · `407302` · `407306` · `407401` · `429`

### Middleware ผู้เล่น `PlayerAuthenticated` (ใช้กับทุก route ใต้ `/player/pr`)

1. อ่าน Bearer token → ตรวจลายเซ็น / หมดอายุ / `typ = "pl"` → อ่าน `sid`, `member_id`
2. หา session ใน Redis ด้วย `sid` — ต้องมีและเป็นของ `member_id` เดียวกัน · ไม่ใช่ session ปัจจุบันของ Member (ถูกเตะ) = `407204`
3. โหลด Member ด้วย primary key (`id, username, agent_id, status, cnf`) + สถานะสายบนจาก `cnf.parent` ด้วย query เดียว (ไม่ N+1) → คำนวณสถานะที่ใช้งานจริง (PA-30 / PA-32)
4. ต่ออายุ idle (PA-21) แล้ว set `c.Locals("player", Player{...})` (struct เดียว มี `can_bet`)

### Middleware operator `OperatorAuthenticated` (ใช้กับ `transfer/*`)

อ่าน body → PA-10 / PA-11 / PA-13 / PA-14 → set `c.Locals("operator", Operator{agent_id, username})` · ห้าม log body ดิบ (มี `key`)

## 6. Schema

**migration** (`make migration name=player_auth_username_length`):

```sql
-- +goose Up
ALTER TABLE user_members ALTER COLUMN username TYPE VARCHAR(65);
-- +goose Down
-- ย่อกลับเป็น 32 ไม่ได้ถ้ามี username แบบ agent@player แล้ว — ลบแถวผู้เล่น operator ก่อนจึงย่อได้
ALTER TABLE user_members ALTER COLUMN username TYPE VARCHAR(32);
```
- `uq_user_members_username` (unique เดิม) รองรับ PA-06 / PA-08 · ไม่ต้องมี index เพิ่ม · ตารางเล็ก ไม่ต้องแยก 3 ขั้น (docs/DEPLOYMENT.md)
- ไม่มีตารางใหม่ (ใช้ `user_members` · `api_credentials` · `api_allowed_ips` · `user_agents`)
- **[ร่าง]** `core/` ต้องมี validator username แยกจาก MGMT-05 (ความยาว 65 รวม `@`)

Redis:

| Key | ค่า | TTL |
|---|---|---|
| `pl:sess:{sid}` | JSON `{member_id, ip, created_at}` | idle 30 นาที (ต่อทุก request ไม่เกิน absolute) |
| `pl:sess:member:{member_id}` | `sid` ปัจจุบัน | absolute 4 ชม. |
| `pl:login:fail:{username}` | จำนวนครั้งที่ผิด | 15 นาที |
| `pl:login:ip:{ip}` | จำนวนครั้งที่เรียก | 1 นาที |

## 7. Test cases

| Rule | Input | Expected |
|---|---|---|
| PA-04 | login `"  Mem01 "` (บัญชี `mem01`) | สำเร็จ |
| PA-03 | login ตรงด้วย username ผู้เล่น operator | `407201` |
| PA-04 | ไม่พบ username · รหัสผิด | `407201` ทั้งคู่ |
| PA-05 | `playerUsername` `"AB"` · `"abc!"` · 33 ตัว · `"ABC"` | `422` · `422` · `422` · สำเร็จ เก็บ `agent@abc` |
| PA-07 | `transfer/login` ผู้เล่นใหม่ | สร้าง + token + link · PT / commission / win limit = ค่าตั้งต้น |
| PA-07 | `transfer/login` ผู้เล่นเดิม 2 ครั้ง | ไม่สร้างซ้ำ · ครั้งที่ 2 เตะ session แรก |
| PA-08 | `register` ซ้ำ · 2 คำขอพร้อมกัน | `407401` · 1 สำเร็จ 1 `407401` ไม่มี 500 |
| PA-10/11 | key ผิด · agent ไม่มี · IP นอก whitelist | `407206` · `407206` · `407305` |
| PA-12 | ยิง `transfer/login` ด้วย key จริง แล้วอ่าน log | ไม่พบ key ในทุกบรรทัด |
| PA-22 | login เครื่อง A แล้วเครื่อง B → ใช้ token A | `407204` |
| PA-21 | ไม่มี request 31 นาที · ใช้ต่อเนื่องครบ 4 ชม. | `407203` · `407203` |
| PA-20 | token BO ยิง `/player/pr` · token ผู้เล่นยิง `/bo/pr` | `407202` · `401202` |
| PA-26 | logout ซ้ำ 2 ครั้ง · logout แล้วใช้ token เดิม | สำเร็จทั้งคู่ · `407203` |
| PA-30/32 | Member ACTIVE สายบนถูก LOCKED ระหว่างใช้งาน | request ถัดไป `407302` และ session ถูกลบ |
| PA-33 | Member SUSPENDED เรียก `me/get` · `games/list` | สำเร็จ · `can_bet = false` |
| PA-34 | `members/status/update` → LOCKED | session ผู้เล่นถูกลบหลัง commit · token เดิม `407203` · Redis ล่ม: status เปลี่ยนสำเร็จ + log warn |
| PA-41 | ผิด 5 ครั้งแล้วใส่รหัสถูก · ผิด 4 ครั้ง ถูก แล้วผิดอีก 4 | `407303` · ยัง login ได้ |
| PA-40 | IP เดียวยิง 21 ครั้ง/นาที | ครั้งที่ 21 `429` |
| PA-50 | รหัสเดิมผิด · รหัสใหม่ผิดกติกา · รหัสใหม่เหมือนเดิม | `407205` · `422` · `422` |
| PA-51 | ผู้เล่น operator เปลี่ยนรหัส | `407307` |

ต้องมี table test ของ `core/player_auth` (validator username, สถานะที่ใช้งานจริง, `can_bet`) และ integration test ทุกเส้น · rule id อ้างใน test ผ่าน comment `// rule: PA-xx`

## 8. Contract changes (แจ้ง frontend / operator)

- ฝั่งผู้เล่น (หน้าเกม): endpoint ใหม่ทั้ง 5 เส้น (PA1 / PA3 / PA4 / PA5 / PA7) · token ใส่ `Authorization: Bearer` · หน้าเกมอ่าน token จาก URL (L2) แล้วลบออกจาก URL ทันที · `407202` / `407203` / `407204` → พาไปหน้าเริ่มต้น (ผู้เล่นจาก operator กลับไปเว็บ operator)
- ฝั่ง operator: `transfer/login` และ `transfer/register` (ใหม่) · ต้องตั้ง IP whitelist ที่เมนู API ก่อน (ACC-07)
- `members/status/update` (M4): LOCKED ลบ session ผู้เล่นหลัง commit — response ไม่เปลี่ยน
- `user_members.username` ยาวได้ถึง 65 ตัว — หน้า list / detail ของ Member (BO) ต้องรองรับ username ที่มี `@`
- error module `07` = player_auth ลง `docs/ERROR_CODES.md` (ตารางด้านล่าง — จองก่อนประกาศ)

### error code ที่จะจอง (module `07`) **[ร่าง]**

| Code | HTTP | ความหมาย |
|---|---|---|
| 407201 | 401 | username หรือรหัสผ่านไม่ถูกต้อง |
| 407202 | 401 | ไม่พบ token / token ไม่ถูกต้องหรือหมดอายุ |
| 407203 | 401 | session สิ้นสุดแล้ว (logout / ไม่ได้ใช้งานนาน) |
| 407204 | 401 | บัญชีนี้ถูกเข้าสู่ระบบจากที่อื่น |
| 407205 | 200 | รหัสผ่านเดิมไม่ถูกต้อง |
| 407206 | 401 | `agentUsername` หรือ `key` ไม่ถูกต้อง |
| 407301 | 403 | บัญชีถูกล็อก |
| 407302 | 403 | สายบัญชีต้นสังกัดถูกล็อก |
| 407303 | 403 | login ผิดหลายครั้ง ระงับชั่วคราว |
| 407305 | 403 | IP ไม่ได้รับอนุญาต |
| 407306 | 403 | บัญชีนี้ใช้ทางเข้านี้ไม่ได้ |
| 407307 | 200 | บัญชีนี้ไม่มีรหัสผ่าน |
| 407401 | 200 | มีผู้เล่นนี้แล้ว |

(`407304` เว้นไว้ — เดิมพันไม่ได้เพราะ SUSPENDED เป็นของ module Bet จะจองตอนทำ spec นั้น)

## 9. คำถามที่ต้องให้ lead ตัดสินก่อน implement

> review ไม่ได้ระบุ — ผู้ร่างไม่เดา · ทุกข้อมีข้อเสนอของผู้ร่างไว้ให้เลือก / แก้

| # | คำถาม | ข้อเสนอผู้ร่าง |
|---|---|---|
| 1 | **ผู้สร้างประเภทไหนเรียก `transfer/*` ได้** (PA-14) — สร้าง Member ได้ในสาย Transfer คือ Share B2C และ Agent (MGMT-02) · Company Transfer / Share B2B สร้าง Member ไม่ได้ | อนุญาตเฉพาะ `SHARE_B2C` และ `AGENT` · ประเภทอื่น `407306` |
| 2 | **สกุลเงินของผู้เล่นที่ระบบสร้างให้** — `user_members.currency` ไม่เว้นว่างได้ แต่ body ของ L2 / TR ไม่มี `currency` และ Company Transfer มีหลายสกุล | operator ส่ง `currency` เพิ่ม (บังคับ · ต้องอยู่ในสกุลของผู้สร้าง — แบบ MGMT) หรือใช้สกุลเดียวของผู้สร้างถ้ามีสกุลเดียว |
| 3 | **`name` ของผู้เล่นที่ระบบสร้างให้** (NOT NULL · 3–32 ตัว) | ใช้ `playerUsername` (ผ่านกติกา 3–32 อยู่แล้ว) |
| 4 | **ยอดตั้งต้นและ `payment_type`** ของผู้เล่นที่ระบบสร้าง | `credit = 0` · `payment_type` ค่า default ของ DB (`CASH`) · ผู้สร้างฝากให้ทีหลังผ่าน spec Transfer API |
| 5 | **ค่าตั้งต้นของ PT / commission** "แบบที่ BO สร้างโดยไม่กรอกค่า" — `members/create` บังคับส่ง `pt` ทุกกลุ่ม (MGMT-21) ไม่มีโหมดไม่กรอก | ขอ lead ยืนยันค่า: `pt = 0` · `commission = 0` · `win_limit_per_day` = default DB 10,000,000 หรือคัดลอกจากผู้เล่นคนล่าสุดของ agent |
| 6 | **`can_bet`** (PA-35) — นับ `win_limit_per_day = 0` (เล่นไม่ได้ — MGMT-65) และเกมปิดด้วยไหม | `can_bet` = ACTIVE + `win_limit_per_day > 0` · เกมปิดดูที่ `games/list` รายเกม |
| 7 | **`games/list`**: แหล่งรายการเกม (`gameRateCore` specs / `agent_game_settings.status_game_effective` ของผู้สร้าง?) · field ที่ต้องส่งนอกจาก `game_code` / `open` / `game_cover` (ชื่อเกม? rate?) | รายการจาก catalog เกมในระบบ · `open` = เปิดจริงของ**ผู้สร้าง**ตามสายบน (`status_game_effective`) · `game_cover` ตาม asset AST-51 (ตัดสินแล้วในฝั่ง asset) · ชื่อเกมที่แสดง: ขอ lead ยืนยันว่าหน้าเกมใช้ชื่อจากไหน |
| 8 | **rate limit ของ `transfer/*`** (PA-42) | IP 20/นาทีต่อ IP (key `pl:op:ip:*`) · whitelist ว่าง = ปฏิเสธทุก IP (`407305`) ตาม ACC-07 ("ไม่มีเลย = ลูกค้าเรียก API ไม่ได้") ซึ่งไม่ต้องตัดสินใหม่ |
| 9 | **รูปแบบ field ของ operator**: camelCase (`agentUsername` / `playerUsername`) ตาม askmelotto หรือ snake_case ตามระบบนี้ | camelCase ตามที่ review เขียนไว้ (`agentUsername` / `playerUsername`) เพื่อให้ operator เก่าย้ายง่าย |
| 10 | **"เปลี่ยนรหัสเตะ session อื่น"** (PQ8–PQ14) — มี 1 session ต่อคน จึงไม่มี session อื่น | session ปัจจุบันอยู่ต่อ · ไม่ต้องทำอะไรเพิ่ม (หรือบังคับ login ใหม่ถ้า lead ต้องการ) |
| 11 | **ชน username ข้ามตาราง**: ผู้เล่น `agent@p001` ชนรูปแบบบัญชีย่อย `agent@sub` (BO login ใช้ `@` แยก sub) แม้อยู่คนละตาราง | ส่วน player (a-z0-9) ไม่รวม `@` แล้วเทียบกับ `subaccounts` ตอนสร้าง · ถ้า sub กับ player ชื่อซ้ำกันได้เพราะคนละ endpoint login ก็ไม่ต้องกัน — ขอ lead ยืนยัน |
| 12 | **URL ของ lobby** ที่ใส่ใน link — ต้องมี env ใหม่ (เช่น `PLAYER_LOBBY_URL`) และชื่อ query (`token`) | เพิ่ม env + `.env.example` ใน PR เดียวกัน · query ชื่อ `token` |
| 13 | **ตรวจสถานะสายบนทุก request** (PA-30) = query เพิ่มทุก request · lead ตัดสินแล้วว่าเช็คทุกครั้ง | ไม่ cache สถานะ (กันช้า LOCKED) · query เดียวด้วย `cnf.parent` ids · วัดซ้ำตอน load test (k6) |

## 10. Checklist ก่อนเปิด PR (ตามลำดับ review → implement)

1. lead review ทีละ endpoint → ปิดหัวข้อ 9 → เปลี่ยนสถานะเป็น APPROVED
2. จอง module `07` ใน `docs/ERROR_CODES.md` + `pkg/apperr/player_auth_errors.go`
3. migration `username VARCHAR(65)` · env ใหม่ใน `pkg/configs/config.go` + `.env.example`
4. `app/core/player_auth` (pure + table test) → service → repository → controllers → routes (บรรทัดเดียวกับสิทธิ์)
5. แก้ `members/status/update` (member_management) + ปิด MGMT-30A ใน `docs/modules/member_management.md`
6. อัปเดต `docs/MIGRATION_STATUS.md` · `docs/frontend/player_auth.md` · `make check` + integration test
