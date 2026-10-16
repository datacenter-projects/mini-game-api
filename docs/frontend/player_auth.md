# API ผู้เล่น + operator สาย Transfer (`player_auth`) — 15 ต.ค. 2026

spec เต็ม: `docs/modules/player_auth.md` (**APPROVED** · zerph 2026-10-15) · สถานะ: **merge เข้า dev แล้ว · อยู่ใน image `dev-202610160626` · migration รันบน RDS dev แล้ว · ใช้จริงบน dev ได้เมื่อตั้ง `TRUSTED_PROXIES` + `PLAYER_LOBBY_URL` ครบ (รอ ingress CIDR จาก lead) — ก่อนนั้นยังไม่ควรให้หน้าบ้านเรียก** · ผู้ทำ: Boiledegg
ขอบเขตรอบนี้ = **สาย Transfer เท่านั้น** (Seamless พักไว้ · `transfer/launch` ยังไม่ทำ)

> ตัวอย่างด้านล่างเขียนตามโครงที่ integration test ตรวจ (`app/internals/backoffice/player_auth_*_integration_test.go`) — ยังไม่ได้ยิงบน dev จริง
> ถ้า field ไหนไม่ตรงกับที่ยิงได้ ให้ถือ spec เป็นหลักแล้วแจ้ง

## 1. ภาพรวม

- ทุกเส้น `POST` · body JSON · **ห้าม `null`** (ได้ `422`) · ชื่อ field `snake_case`
- response รูปแบบเดียวกับทั้งระบบ: `{"code": 200, "msg": "สำเร็จ", "data": {...}}` · error อ่านจาก `code` (ดูหัวข้อ 5) · HTTP status ของ error ตามหัวข้อ 5
- **ผู้เล่น 2 ทางเข้า:**
  1. **operator ส่งผู้เล่นเข้ามา** — server ของ operator เรียก `transfer/login` ได้ `link` แล้ว redirect ผู้เล่นไปที่ link นั้น (หัวข้อ 3)
  2. **ผู้เล่น login เอง** ด้วย `player/pb/login` — ใช้ได้กับ Member ที่สร้างจากหลังบ้าน และผู้เล่นจาก operator ที่ operator ตั้งรหัสไว้ตอน `transfer/register`
- token ใส่ header `Authorization: Bearer <token>` ทุกเส้น `/player/pr/*`
- **1 session ต่อผู้เล่น** — login (ทางไหนก็ตาม) ใหม่เตะอันเก่า (token เก่าได้ `407204`)
- session หมดอายุ: ไม่มี request เกิน **30 นาที** (idle) หรือครบ **4 ชั่วโมง**นับจาก login (absolute) · `expires_at` ใน response คือเวลา absolute
- **ping กันหลุด:** หน้าเกมเรียก `balance/get` ทุก 5–10 นาที (ทุก request ที่ผ่านต่ออายุ idle) · ไม่มีเส้น ping แยก
- ผู้เล่นจาก operator ใช้ username `{agent_username}@{player_username}` (ตัวเล็ก) เช่น `agent01@p001`

## 2. เส้นฝั่งผู้เล่น (`/api/v1/player/...`)

| # | Path | Header token | ใช้เมื่อ |
|---|---|---|---|
| PA1 | `/api/v1/player/pb/login` | ไม่ต้อง | ผู้เล่น login ตรง (username + password) |
| PA3 | `/api/v1/player/pr/auth/logout` | ต้อง (ไม่เช็ค session — logout ซ้ำได้) | ออกจากระบบ |
| PA4 | `/api/v1/player/pr/auth/password/update` | ต้อง | เปลี่ยนรหัสผ่านตัวเอง |
| PA5 | `/api/v1/player/pr/me/get` | ต้อง | โปรไฟล์ + สถานะ + `can_bet` |
| PA6 | `/api/v1/player/pr/balance/get` | ต้อง | ยอดเงิน (และใช้เป็น ping) |
| PA7 | `/api/v1/player/pr/games/list` | ต้อง | รายชื่อเกมของ lobby |

### PA1 · login

```bash
curl -s -X POST $BASE/api/v1/player/pb/login -H 'Content-Type: application/json' \
  -d '{"username":"mem01","password":"aA4b4c4d4e4f"}'
```
```json
{ "code": 200, "msg": "สำเร็จ", "data": {
    "token": "eyJhbGciOi…", "expires_at": "2026-10-15T20:30:00+07:00",
    "member": { "username": "mem01", "name": "mem01", "currency": "THB", "status": "ACTIVE", "can_bet": true } } }
```
- `username` ไม่สนตัวพิมพ์ / ตัดช่องว่างหัวท้าย · `member` ไม่มี `credit` / `id`
- `member.status` = `ACTIVE` หรือ `SUSPENDED` (สถานะที่ใช้งานจริง — แย่สุดของผู้เล่น + สายบน) · `LOCKED` login ไม่ผ่าน
- ไม่พบ username · รหัสผิด · **ผู้เล่นที่ไม่มีรหัส** → `407201` ข้อความเดียวกันหมด
- ใส่รหัสผิด 5 ครั้งใน 15 นาที → `407303` นาน 15 นาที (แม้ครั้งต่อไปรหัสถูก) · เรียก login เกิน 20 ครั้ง/นาทีต่อ IP → `429`

### PA3 · logout

`POST /api/v1/player/pr/auth/logout` · ไม่มี body · สำเร็จ `{"code":200,"msg":"สำเร็จ"}` (ไม่มี `data`)
เรียกซ้ำ / session หมดไปแล้ว ก็สำเร็จ · token ปลอมหรือหมดอายุ → `407202`

### PA4 · เปลี่ยนรหัสผ่าน

```json
{ "old_password": "aA4b4c4d4e4f", "new_password": "Zx9!kq2m" }
```
- รหัสใหม่ตามกติกาเดียวกับหลังบ้าน (8–64 ตัว · a-z A-Z 0-9 และสัญลักษณ์ ASCII · ห้ามช่องว่าง · ต้องมีตัวอักษรและตัวเลข · ห้ามตัวเดิมซ้ำติดกัน 4 ตัว) · ไม่ผ่าน = `422` ข้อความบอก `new_password`
- ซ้ำรหัสเดิม = `422` · รหัสเดิมผิด = `407205` (**นับรวมกับ login ผิด** — ครบ 5 ครั้ง = `407303` และ session ถูกตัด)
- ผู้เล่นที่ไม่มีรหัส (มาจาก operator และไม่ได้ตั้งรหัส) = `407307`
- สำเร็จแล้ว **session เดิมใช้ต่อได้** (ไม่ต้อง login ใหม่)

### PA5 · me/get

body `{}` (ไม่รับ field)
```json
{ "code": 200, "msg": "สำเร็จ", "data": { "username": "agent01@p001", "name": "p001", "currency": "THB", "status": "SUSPENDED", "can_bet": false } }
```
- **ไม่มี `credit`** — ใช้ PA6 · `can_bet` = สถานะ ACTIVE และลิมิตชนะต่อวัน > 0 · `SUSPENDED` เรียกได้ปกติ (ดูข้อมูลได้ แทงไม่ได้)

### PA6 · balance/get

```json
{ "code": 200, "msg": "สำเร็จ", "data": { "currency": "THB", "credit": 1250.5 } }
```
- `credit` ส่งค่าที่เก็บตามจริง ไม่ปัดเพิ่ม (หน้าบ้านปัด 3 ตำแหน่งตอนแสดงเอง) · อ่านจาก DB ตรง ไม่ cache · เรียกได้แม้ SUSPENDED

### PA7 · games/list

```json
{ "code": 200, "msg": "สำเร็จ", "data": { "items": [
    { "game_code": "scratch_card", "name_th": "สแครชการ์ด", "name_en": "Scratch Card", "open": true,
      "game_cover": "https://cdn.example.com/assets/scratch_card/game_cover/3f9a….png" },
    { "game_code": "coin_toss", "name_th": "โยนเหรียญ", "name_en": "Coin Toss", "open": false, "game_cover": "" },
    { "game_code": "rock_paper_scissors", "name_th": "เป่ายิ้งฉุบ", "name_en": "Rock Paper Scissors", "open": true, "game_cover": "" } ] } }
```
- เรียงตามลำดับคงที่ (scratch_card → coin_toss → rock_paper_scissors) · **เกมปิดไม่ถูกซ่อน** (`open: false` — แสดงเป็นเกมปิด)
- `open` ตามการเปิด / ปิดเกมของ agent ผู้สร้างและสายบนทั้งหมด (ผู้เล่นไม่มีสวิตช์รายคน) · `game_cover = ""` เมื่อยังไม่มีปก
- ไม่มีค่า rate · ชื่อเกมไทย / อังกฤษมาจาก catalog ฝั่ง backend

## 3. เส้นฝั่ง operator สาย Transfer (`/api/v1/transfer/...`)

| # | Path | ใช้เมื่อ |
|---|---|---|
| L2 | `/api/v1/transfer/login` | ได้ token + link เข้า lobby (ไม่มีผู้เล่น = สร้างให้) |
| TR | `/api/v1/transfer/register` | สร้างผู้เล่น (ไม่ออก token) |

**ยืนยันตัว operator ทุกเส้น:** ส่ง `agent_username` + `key` ใน body (ไม่มี `sign`) · key คือ API Key ของ agent ที่เมนู API ของหลังบ้าน
- **IP ของ server operator ต้องอยู่ใน whitelist** ที่เมนู API ก่อน · whitelist ว่าง = ปฏิเสธทุก IP (`407305`)
- เรียกได้เฉพาะ agent ประเภท **Share B2C** และ **Agent** (ประเภทอื่น `407306`)
- เรียกเกิน 20 ครั้ง/นาทีต่อ IP (นับทุกเส้นของ `transfer/*` รวมที่ key ผิด) → `429`
- **ห้าม log `key` / `password`** ฝั่ง operator ด้วย

### L2 · transfer/login

```bash
curl -s -X POST $BASE/api/v1/transfer/login -H 'Content-Type: application/json' \
  -d '{"agent_username":"agent01","key":"<64 hex>","player_username":"p001","currency":"THB","language":"th"}'
```
```json
{ "code": 200, "msg": "สำเร็จ", "data": {
    "token": "eyJhbGciOi…", "expires_at": "2026-10-15T20:30:00+07:00",
    "link": "https://lobby.example.com/?token=eyJhbGciOi…&lang=th" } }
```

| Field | บังคับ | ความหมาย |
|---|---|---|
| `agent_username` | ✅ | username ของ agent เจ้าของ key (ไม่สนตัวพิมพ์) |
| `key` | ✅ | API Key |
| `player_username` | ✅ | `a-z 0-9` ยาว 3–32 (ส่งตัวใหญ่ได้ ระบบเก็บตัวเล็ก) · ผิดรูปแบบ `422` |
| `currency` | ไม่ | ไม่ส่ง = ใช้สกุลของ agent (ถ้า agent มีสกุลเดียว) · หลายสกุลต้องส่ง · ต้องเป็นสกุลของ agent (`422`) · **ผู้เล่นที่มีอยู่แล้วแต่สกุลไม่ตรง = `422` "สกุลเงินไม่ตรงกับผู้เล่น"** |
| `language` | ไม่ | `th` / `en` (default `th`) → ต่อ `&lang=` ท้าย link · ค่าอื่น `422` |

- **ไม่มีผู้เล่น = สร้างให้อัตโนมัติ** (ไม่มีรหัสผ่าน · ยอดเริ่มต้น 0 · เล่นผ่าน operator อย่างเดียว) แล้ว login ต่อในคำขอเดียว · response **ไม่บอก**ว่าสร้างใหม่หรือเดิม
- ยิง login ผู้เล่นใหม่พร้อมกันหลายคำขอ — ทุกคำขอสำเร็จ ไม่ error (สร้างแถวเดียว)
- **redirect ผู้เล่นไป `link` ตรงๆ** · หน้าเกมอ่าน `token` จาก URL → เก็บไว้ใช้เป็น `Authorization: Bearer` → **ลบ token ออกจาก URL ทันที** (`history.replaceState`) · ตั้ง `Referrer-Policy: no-referrer` · ห้าม log query string ของหน้าเกม
- ฐาน URL ของ link มาจากการตั้งค่า `PLAYER_LOBBY_URL` ของเซิร์ฟเวอร์ — ถ้าไม่ได้ตั้ง `transfer/login` ตอบ `503`

### TR · transfer/register

```json
{ "agent_username": "agent01", "key": "<64 hex>", "player_username": "p001", "currency": "THB", "password": "aA4b4c4d4e4f" }
```
```json
{ "code": 200, "msg": "สำเร็จ", "data": { "username": "agent01@p001", "currency": "THB" } }
```
- `currency` / `password` ไม่บังคับ · **ส่ง `password`** (AUTH-36 — ไม่ผ่าน `422`) = ผู้เล่น login ตรงที่ PA1 ได้ด้วยชื่อเต็ม `agent01@p001` · **ไม่ส่ง** (หรือส่ง `""`) = เข้าทาง operator อย่างเดียว
- ไม่ออก token / link · ไม่คืนรหัส · **มีผู้เล่นนี้แล้ว = `407401`** — operator ถือเป็น "สร้างไว้แล้ว" (ไม่แก้ผู้เล่นเดิม ไม่ตั้งรหัสให้)

### สถานะของ agent ที่มีผลกับ operator

| สถานะ agent (หรือสายบน) | ผู้เล่นเดิม login | สร้างผู้เล่นใหม่ (`register` / auto-create ใน `login`) |
|---|---|---|
| ACTIVE | ✅ | ✅ |
| SUSPENDED | ✅ (`can_bet = false`) | ❌ `407308` |
| LOCKED | ❌ `407302` ทุกเส้น | ❌ `407302` |

ผู้เล่นเองถูกล็อก → `407301`

## 4. สถานะผู้เล่น (สถานะที่ใช้งานจริง = แย่ที่สุดของผู้เล่น + สายบนทุกคน)

| สถานะ | login | ใช้งานหลัง login | เดิมพัน | session |
|---|---|---|---|---|
| LOCKED | ❌ `407301` (ตัวเอง) / `407302` (สายบน) | ❌ | ❌ | ลบทันที |
| SUSPENDED | ✅ | ✅ ดูข้อมูล / ยอด / lobby | ❌ (`can_bet = false`) | คงไว้ |
| ACTIVE | ✅ | ✅ | ✅ (ถ้าเกมเปิด) | |

- เช็คทุก request — ถ้าถูกล็อกระหว่างใช้งาน request ถัดไปได้ `407301` / `407302` และ session ถูกลบ
- หลังบ้านตั้งผู้เล่นเป็น `LOCKED` (`members/status/update`) → session ของผู้เล่นถูกลบทันทีหลังบันทึก → token เดิมได้ `407203`

## 5. Error code (module `07`)

| code | HTTP | ความหมาย | ฝั่งหน้าเกมควร |
|---|---|---|---|
| 407201 | 401 | username หรือรหัสผ่านไม่ถูกต้อง (รวมผู้เล่นไม่มีรหัส) | แจ้งผู้เล่น |
| 407202 | 401 | ไม่พบ token / ไม่ถูกต้อง / หมดอายุ | กลับหน้าเริ่มต้น |
| 407203 | 401 | session สิ้นสุด (logout / ไม่ได้ใช้งานนาน / ถูกล็อก) | กลับหน้าเริ่มต้น |
| 407204 | 401 | ถูก login จากที่อื่น | แจ้งผู้เล่น แล้วกลับหน้าเริ่มต้น |
| 407205 | 200 | รหัสผ่านเดิมไม่ถูกต้อง | แจ้งผู้เล่น |
| 407206 | 401 | `agent_username` หรือ `key` ไม่ถูกต้อง (operator) | — |
| 407301 | 403 | บัญชีถูกล็อก | แจ้งผู้เล่น |
| 407302 | 403 | สายบัญชีต้นสังกัดถูกล็อก (รวม agent เจ้าของ key) | แจ้งผู้เล่น |
| 407303 | 403 | ใส่รหัสผิดหลายครั้ง ระงับชั่วคราว 15 นาที | แจ้งผู้เล่น |
| 407305 | 403 | IP ของ operator ไม่อยู่ใน whitelist | operator ต้องตั้ง whitelist |
| 407306 | 403 | agent ประเภทนี้ใช้ `transfer/*` ไม่ได้ | — |
| 407307 | 200 | บัญชีนี้ไม่มีรหัสผ่าน | ซ่อนเมนูเปลี่ยนรหัส |
| 407308 | 403 | agent ถูกระงับ สร้างผู้เล่นใหม่ไม่ได้ | — |
| 407401 | 200 | มีผู้เล่นนี้แล้ว (`transfer/register`) | operator ถือว่าสร้างแล้ว |
| 422 | 200 | ข้อมูลไม่ผ่าน (ข้อความบอก field) | — |
| 429 | 429 | เรียกบ่อยเกินไป | รอแล้วลองใหม่ |
| 503 | 503 | ระบบยังไม่พร้อม (ไม่ได้ตั้ง `PLAYER_LOBBY_URL`) | — |

## 6. ทดสอบ

- สร้างผู้เล่นทดสอบ 2 แบบ: **จากหลังบ้าน** (`members/create` — มีรหัส login ที่ PA1 ด้วย username ปกติ) และ **จาก operator** (`transfer/register` ใส่ `password` หรือไม่ใส่)
- ก่อนเรียก `transfer/*`: ที่เมนู API ของ agent (Share B2C / Agent) ต้องมี **API Key** (สร้างอัตโนมัติ) และ **IP whitelist** ของเครื่องที่ยิง
- ทดสอบสถานะ: ตั้ง Member `SUSPENDED` / `LOCKED` ที่ `members/status/update` แล้วยิง PA5 ด้วย token เดิม

## 7. ตัวอย่าง request / response ละเอียด (ต่อเส้น)

> ตัวอย่างในหัวข้อนี้ได้จากการยิง API จริงใน test env (16 ต.ค.) — ตัดเฉพาะ `token` ให้สั้น (`eyJhbGciOi…`) และเวลาเป็นค่าตัวอย่าง · ยังไม่ได้ยิงบน dev
> ทุก response มีรูปแบบ `{"code", "msg", "data"}` · เส้นที่ไม่มีข้อมูลคืน (logout · password/update) **ไม่มี `data`** · `HTTP` ในแต่ละตัวอย่างคือ status จริงที่ได้ (error ส่วนใหญ่ของฝั่งผู้เล่นเป็น `401` / `403` ยกเว้น `422` / `407205` / `407307` / `407401` ที่เป็น `200`)
> ตัวแปรในตัวอย่าง: `{{MG_URL}}` = base URL · `{{TOKEN}}` = token จาก PA1 หรือ L2

### 7.1 PA1 · `player/pb/login` — ผู้เล่น login ตรง

**Request** (ไม่ต้องมี header `Authorization`)
```bash
curl -X POST "{{MG_URL}}/api/v1/player/pb/login" -H "Content-Type: application/json" \
  -d '{"username":"mem01","password":"aA4b4c4d4e4f"}'
```

**Response — สำเร็จ** `HTTP 200`
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "token": "eyJhbGciOi…",
    "expires_at": "2026-10-16T20:12:43.897+07:00",
    "member": { "username": "mem01", "name": "Namemem01", "currency": "THB", "status": "ACTIVE", "can_bet": true }
  }
}
```
- `member.status` = `SUSPENDED` และ `can_bet: false` ได้ (login ผ่าน แต่แทงไม่ได้) · `member` ไม่มี `credit` / `id`

**Response — error**

| กรณี | HTTP | code | msg |
|---|---|---|---|
| username ไม่พบ / รหัสผิด / ผู้เล่นไม่มีรหัส (ข้อความเดียวกันหมด) | 401 | `407201` | `username หรือรหัสผ่านไม่ถูกต้อง` |
| ผู้เล่นถูกล็อก | 403 | `407301` | `บัญชีนี้ถูกล็อก กรุณาติดต่อผู้ดูแลระบบ` |
| สายบัญชีต้นสังกัดถูกล็อก | 403 | `407302` | `สายบัญชีต้นสังกัดถูกล็อก กรุณาติดต่อผู้ดูแลระบบ` |
| รหัสผิดครบ 5 ครั้งใน 15 นาที | 403 | `407303` | ระงับ 15 นาที (แม้ครั้งต่อไปรหัสถูก) |
| เรียก login เกิน 20 ครั้ง / นาที ต่อ IP | 429 | `429` | เรียกบ่อยเกินไป |
```json
{ "code": 407201, "msg": "username หรือรหัสผ่านไม่ถูกต้อง" }
```

### 7.2 PA5 · `player/pr/me/get` — โปรไฟล์ + สถานะ

**Request** body `{}`
```bash
curl -X POST "{{MG_URL}}/api/v1/player/pr/me/get" -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{}'
```

**Response — ACTIVE** `HTTP 200`
```json
{ "code": 200, "msg": "สำเร็จ", "data": { "username": "mem01", "name": "Namemem01", "currency": "THB", "status": "ACTIVE", "can_bet": true } }
```

**Response — SUSPENDED** (เรียกได้ปกติ · แทงไม่ได้)
```json
{ "code": 200, "msg": "สำเร็จ", "data": { "username": "mem01", "name": "Namemem01", "currency": "THB", "status": "SUSPENDED", "can_bet": false } }
```

**Response — ไม่ส่ง token** `HTTP 401`
```json
{ "code": 407202, "msg": "กรุณาเข้าสู่ระบบ" }
```

### 7.3 PA6 · `player/pr/balance/get` — ยอดเงิน (และใช้เป็น ping)

**Request** body `{}`
```bash
curl -X POST "{{MG_URL}}/api/v1/player/pr/balance/get" -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{}'
```
**Response** `HTTP 200`
```json
{ "code": 200, "msg": "สำเร็จ", "data": { "currency": "THB", "credit": 1250.5 } }
```
- `credit` ส่งค่าที่เก็บตามจริง (ทศนิยมได้ถึง 4 ตำแหน่ง) · อ่านจาก DB ตรง ไม่ cache

### 7.4 PA7 · `player/pr/games/list` — รายชื่อเกมของ lobby

**Request** body `{}`
```bash
curl -X POST "{{MG_URL}}/api/v1/player/pr/games/list" -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{}'
```
**Response** `HTTP 200`
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "items": [
      { "game_code": "scratch_card",        "name_th": "สแครชการ์ด",  "name_en": "Scratch Card",        "open": true, "game_cover": "" },
      { "game_code": "coin_toss",           "name_th": "โยนเหรียญ",   "name_en": "Coin Toss",           "open": true, "game_cover": "" },
      { "game_code": "rock_paper_scissors", "name_th": "เป่ายิ้งฉุบ", "name_en": "Rock Paper Scissors", "open": true, "game_cover": "" }
    ]
  }
}
```
- `open: false` = เกมปิด (ยังอยู่ในรายการ) · `game_cover` เป็น URL เมื่อมีปก ไม่มี = `""`

### 7.5 PA4 · `player/pr/auth/password/update` — เปลี่ยนรหัสผ่าน

**Request**
```bash
curl -X POST "{{MG_URL}}/api/v1/player/pr/auth/password/update" -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" \
  -d '{"old_password":"aA4b4c4d4e4f","new_password":"Zx9!kq2m"}'
```
**Response — สำเร็จ** `HTTP 200` (ไม่มี `data` · session เดิมใช้ต่อได้)
```json
{ "code": 200, "msg": "สำเร็จ" }
```

**Response — error** (ทั้งหมด `HTTP 200`)

| กรณี | code | msg |
|---|---|---|
| รหัสเดิมผิด (นับรวมกับ login ผิด · ครบ 5 ครั้ง = `407303` และ session ถูกตัด) | `407205` | `รหัสผ่านเดิมไม่ถูกต้อง` |
| รหัสใหม่ไม่ผ่านกติกา (เช่น สั้นเกินไป) | `422` | `new_password ต้องยาวอย่างน้อย 8 ตัว` |
| รหัสใหม่ซ้ำรหัสเดิม | `422` | `new_password ต้องไม่ซ้ำกับรหัสผ่านเดิม` |
| ผู้เล่นไม่มีรหัส (มาจาก operator และไม่ได้ตั้ง) | `407307` | `บัญชีนี้ไม่มีรหัสผ่าน` |
```json
{ "code": 407205, "msg": "รหัสผ่านเดิมไม่ถูกต้อง" }
```

### 7.6 PA3 · `player/pr/auth/logout` — ออกจากระบบ

**Request** ไม่มี body
```bash
curl -X POST "{{MG_URL}}/api/v1/player/pr/auth/logout" -H "Authorization: Bearer {{TOKEN}}"
```
**Response** `HTTP 200` (ไม่มี `data` · เรียกซ้ำ / session หมดไปแล้วก็สำเร็จ)
```json
{ "code": 200, "msg": "สำเร็จ" }
```
- token ปลอมหรือหมดอายุ → `HTTP 401` `407202`

### 7.7 ผล token หลังเหตุการณ์ต่างๆ (ทุกเส้น `pr/*` ที่ไม่ใช่ logout)

| เหตุการณ์ | HTTP | code | msg |
|---|---|---|---|
| ไม่ส่ง / ส่ง token ผิด | 401 | `407202` | `กรุณาเข้าสู่ระบบ` |
| หลัง logout / ไม่ใช้งานเกิน 30 นาที / ครบ 4 ชม. / ถูกล็อกระหว่างใช้งาน | 401 | `407203` | `เซสชันสิ้นสุดแล้ว กรุณาเข้าสู่ระบบใหม่` |
| มี login ใหม่จากที่อื่น (token เก่า) | 401 | `407204` | `บัญชีนี้ถูกเข้าสู่ระบบจากที่อื่น` |
| ผู้เล่นถูกล็อก (request ถัดไป) | 403 | `407301` | `บัญชีนี้ถูกล็อก กรุณาติดต่อผู้ดูแลระบบ` |

### 7.8 L2 · `transfer/login` — operator ส่งผู้เล่นเข้า lobby

**Request** (operator ฝั่ง server เป็นคนยิง · IP ต้องอยู่ใน whitelist)
```bash
curl -X POST "{{MG_URL}}/api/v1/transfer/login" -H "Content-Type: application/json" \
  -d '{"agent_username":"agent01","key":"<64 hex>","player_username":"p001","currency":"THB","language":"th"}'
```

**Response — สำเร็จ** `HTTP 200` (ผู้เล่นใหม่และเดิมได้รูปแบบเดียวกัน)
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "token": "eyJhbGciOi…",
    "expires_at": "2026-10-16T20:12:43.957+07:00",
    "link": "https://lobby.example.com/?token=eyJhbGciOi…&lang=th"
  }
}
```
- `language: "en"` → `…&lang=en` · ไม่ส่ง `language` = `th`

**Response — error**

| กรณี | HTTP | code | msg |
|---|---|---|---|
| `agent_username` หรือ `key` ผิด (ข้อความเดียวกัน ไม่บอกว่าผิดที่ไหน · ไม่สะท้อน key กลับ) | 401 | `407206` | `agent_username หรือ key ไม่ถูกต้อง` |
| IP ไม่อยู่ใน whitelist (whitelist ว่าง = ปฏิเสธทุก IP) | 403 | `407305` | IP ไม่ได้รับอนุญาต |
| agent ประเภทนี้ใช้ไม่ได้ (เช่น Company Transfer / Share B2B) | 403 | `407306` | `บัญชีนี้ใช้ทางเข้านี้ไม่ได้` |
| agent ถูกระงับ + ผู้เล่นใหม่ | 403 | `407308` | `บัญชีผู้สร้างถูกระงับ สร้างผู้เล่นใหม่ไม่ได้` |
| agent หรือสายบนถูกล็อก | 403 | `407302` | `สายบัญชีต้นสังกัดถูกล็อก กรุณาติดต่อผู้ดูแลระบบ` |
| ผู้เล่นถูกล็อก | 403 | `407301` | `บัญชีนี้ถูกล็อก กรุณาติดต่อผู้ดูแลระบบ` |
| `language` ไม่ใช่ `th` / `en` | 200 | `422` | `language ต้องเป็น th หรือ en` |
| สกุลเงินไม่ตรงกับผู้เล่นที่มีอยู่ (หรือไม่ใช่สกุลของ agent) | 200 | `422` | `สกุลเงินไม่ตรงกับผู้เล่น` |
| `player_username` ผิดรูปแบบ | 200 | `422` | `player_username ต้องเป็น a-z, 0-9 ยาว 3–32 ตัว` |
| ไม่ได้ตั้ง `PLAYER_LOBBY_URL` ฝั่ง server | 503 | `503` | ระบบยังไม่พร้อม |
| เรียกเกิน 20 ครั้ง / นาที ต่อ IP | 429 | `429` | เรียกบ่อยเกินไป |
```json
{ "code": 407206, "msg": "agent_username หรือ key ไม่ถูกต้อง" }
```

### 7.9 TR · `transfer/register` — operator สร้างผู้เล่น

**Request — ไม่ส่งรหัส** (ผู้เล่นเข้าทาง operator อย่างเดียว)
```bash
curl -X POST "{{MG_URL}}/api/v1/transfer/register" -H "Content-Type: application/json" \
  -d '{"agent_username":"agent01","key":"<64 hex>","player_username":"p002"}'
```
**Response** `HTTP 200`
```json
{ "code": 200, "msg": "สำเร็จ", "data": { "username": "agent01@p002", "currency": "THB" } }
```

**Request — ส่งรหัส** (ผู้เล่น login ตรงที่ PA1 ได้ด้วย `agent01@p003`)
```bash
curl -X POST "{{MG_URL}}/api/v1/transfer/register" -H "Content-Type: application/json" \
  -d '{"agent_username":"agent01","key":"<64 hex>","player_username":"p003","currency":"THB","password":"aA4b4c4d4e4f"}'
```
**Response** `HTTP 200`
```json
{ "code": 200, "msg": "สำเร็จ", "data": { "username": "agent01@p003", "currency": "THB" } }
```

**Response — error** (ทั้งหมด `HTTP 200` ยกเว้น operator ผิดตาม 7.8)

| กรณี | code | msg |
|---|---|---|
| มีผู้เล่นนี้แล้ว (operator ถือว่า "สร้างไว้แล้ว" · ไม่แก้ผู้เล่นเดิม) | `407401` | `มีผู้เล่นนี้แล้ว` |
| `password` ไม่ผ่านกติกา | `422` | `password ต้องยาวอย่างน้อย 8 ตัว` |

### 7.10 ลำดับเรียกที่ใช้จริง (ผู้เล่นเข้าทาง operator)
1. server ของ operator เรียก **L2** `transfer/login` → ได้ `link`
2. redirect ผู้เล่นไป `link` → หน้าเกมอ่าน `token` จาก URL แล้วลบออกจาก URL ทันที
3. หน้าเกมเรียก **PA5** `me/get` (ดู `status` / `can_bet`) และ **PA7** `games/list`
4. ระหว่างเล่นเรียก **PA6** `balance/get` ทุก 5–10 นาที (ต่อ session)
5. ถ้าได้ `407202` / `407203` / `407204` → กลับหน้าเริ่มต้น (ให้ operator ส่งเข้าใหม่) · ถ้า `407301` / `407302` → แจ้งผู้เล่น
