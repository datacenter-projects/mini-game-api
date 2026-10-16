# Lead review: Player Auth (ผู้เล่นเข้าระบบ) — zerph · 2026-10-14 / 15

- ผู้ทำ: **Boiledegg** (ร่าง spec `docs/modules/player_auth.md` → lead review ทีละ endpoint → implement ผ่าน PR)
- ลำดับ: แก้ game_rate (M1 / G-S1 / G-S2) ให้เสร็จก่อน แล้วค่อยเริ่มงานนี้
- **รอบนี้ทำเฉพาะสาย Transfer** — Seamless (`seamless/*` · GetMember · mock operator) พักไว้ แต่ออกแบบ middleware / session ให้ใช้ร่วมได้

## ข้อเท็จจริงที่ใช้ออกแบบ
1. ระบบเดิม: `/provider/launch` แค่สร้างผู้เล่น ไม่ให้ token และไม่มีรหัส → ผู้เล่นที่มาจาก launch login ไม่ได้เลย — **ห้ามลอกแบบ**
2. askmelotto-api ยังไม่เคยต่อ minigame · แบบที่ใกล้สุด: transferwallet `loginRequest` / sportbook `redirectLogin` (operator ส่ง `agentUsername` + `key` + `playerUsername` ได้ link กลับ)
3. BO auth (JWT `typ` · Redis session 1 ต่อบัญชี · rate limit · `WorstStatus` / สายบน) ใช้ซ้ำได้เกือบทั้งชุด
4. `user_members.password_hash` NOT NULL · `username` `VARCHAR(32)` a-z0-9

## การตัดสิน

### ทางเข้า (PQ1 / PQ2) — route group `transfer` / `seamless` register แยก ไม่มี prefix `ext`
| # | เส้น | ผู้เรียก | ผล |
|---|---|---|---|
| L1 | `POST /api/v1/player/pb/login` | ผู้เล่นเอง **ทุกสาย** · `{username, password}` | token |
| L2 | `POST /api/v1/transfer/login` | server operator สาย Transfer | token + link lobby (**ใส่ token ใน URL** — LQ-c ใหม่) |
| L3 | `POST /api/v1/transfer/launch` | server operator สาย Transfer · + `game_code` | token + link เข้าเกมนั้น · ⏸ **ยังไม่ทำรอบนี้** (lead 2026-10-15) |
| L4 | `POST /api/v1/seamless/login` | server operator สาย Seamless | ⏸ พักไว้ |
| L5 | `POST /api/v1/seamless/launch` | server operator สาย Seamless | ⏸ พักไว้ |

- `redirectLogin` แบบ sportbook = launcher → ใช้ชื่อ `launch`
- launch **ไม่ต้อง login ก่อน** — operator ส่งครั้งเดียวจบ
- **LQ-c (lead แก้ 2026-10-15): link ใส่ token ใน URL ตรงๆ แบบ askmelotto** — ไม่มีตั๋ว ไม่มีเส้น `launch/exchange` · หน้าเกมอ่าน token จาก URL แล้วใช้เป็น `Authorization` ต่อ
  - (เดิม: ตั๋วใช้ครั้งเดียว 60 วินาที + `POST /api/v1/player/pb/launch/exchange` — ยกเลิก)
  - ลดความเสี่ยง token หลุด: หน้าเกมลบ token ออกจาก URL ทันทีหลังอ่าน (`history.replaceState`) · หน้าเกมตั้ง `Referrer-Policy: no-referrer` · ห้าม log query string ของ URL หน้าเกมฝั่งเรา · token มีอายุตาม session (idle 30 นาที / absolute 4 ชั่วโมง) และ login ใหม่เตะอันเก่า
- เกมที่ระบุใน launch ปิดอยู่ = error ตั้งแต่ตอน launch
- รองรับทั้ง 2 โหมดกระเป๋า · ทำ Transfer ก่อน (milestone 23 ต.ค.)

### ตัวตนผู้เล่นจาก operator (PQ3)
- username ที่เก็บ = **`{agentUsername}@{playerUsername}`** · operator ส่งแค่ `playerUsername` ระบบต่อ `agent@` ให้เอง · `agent` = เจ้าของ `agentUsername` / `key` ที่เรียก (แม่ตรงของผู้เล่น)
- ส่วน player: a-z0-9 3–32 ตัว (รับตัวใหญ่ เก็บตัวเล็ก) · ไม่ผ่าน = error · คอลัมน์ `user_members.username` ขยายเป็น `VARCHAR(65)`
- Transfer: **สร้างอัตโนมัติตอน login / launch ครั้งแรก** + มีเส้น `POST /api/v1/transfer/register` แยก (มีแล้ว = error "มีผู้เล่นนี้แล้ว")
- ผู้เล่นที่ระบบสร้างให้ใช้ค่าตั้งต้นของ agent (PT / commission / win limit แบบที่ BO สร้างโดยไม่กรอกค่า)
- ผู้เล่นจาก operator **ไม่มีรหัสผ่าน** (`password_hash = ''`) → เข้าได้ทาง operator เท่านั้น · L1 ใช้กับ Member ที่สร้างจาก BO
- ⏸ Member ที่สร้างจาก BO ยังใช้ username แบบเดิม (MGMT-05) — lead ยังไม่ตัดสินว่าจะเปลี่ยนเป็น `agent@player` ไหม · ไม่ต้องแก้ `members/create` รอบนี้

### operator ยืนยันตัว (PQ4)
- **`agentUsername` + `key` ใน body + IP whitelist** (แบบ askmelotto / sportbook) · ไม่มี `sign` · middleware เดียวใช้ทั้ง group `transfer` (และ `seamless` ภายหลัง)
- **mask `key` ใน log ทุกจุด** (CLAUDE.md กฎข้อ 26)
- controller / middleware ของ group `transfer` / `seamless` อยู่ใต้ `app/externals/{transfer|seamless}/` (trust boundary แยก — CLAUDE.md กฎข้อ 29) · ฝั่งผู้เล่นอยู่ `app/internals/frontend/`
- Seamless (ภายหลัง): เรียก GetMember กลับไปที่ operator ทุกครั้งที่ login / launch · timeout 5 วินาที · operator ล่ม = error "เชื่อมต่อเว็บต้นทางไม่ได้"

### session (PQ5 / PQ6)
- แบบ BO: JWT `typ = "pl"` (secret เดียวกับ BO) · Redis `pl:sess:*` · **idle 30 นาที · absolute 4 ชั่วโมง** · ไม่มี refresh (ต่ออายุทุก request) · หน้าเกม ping ระหว่างเล่น
- **1 session ต่อ Member** — login / launch ใหม่เตะอันเก่า · session เก่าได้ code "ถูก login จากที่อื่น"

### สถานะ (PQ7) — ใช้สถานะที่แย่ที่สุดของ Member + สายบนทุกคน
| สถานะ | login / launch | ใช้งานหลัง login | เดิมพัน | session |
|---|---|---|---|---|
| LOCKED (ตัวเองหรือสายบน) | ❌ | ❌ | ❌ | ลบทันที |
| SUSPENDED | ✅ | ✅ ดูข้อมูล / ประวัติ / lobby | ❌ | คงไว้ |
| ACTIVE | ✅ | ✅ | ✅ (ถ้าเกมเปิด) | |
- เช็คทุก request ใน middleware · login เช็คสถานะ**หลัง**รหัสถูก · SUSPENDED cash-out Streak ที่ค้างได้ (→ spec Bet)

### อื่นๆ (PQ8–PQ14)
- logout `POST /api/v1/player/pr/auth/logout` (idempotent)
- `members/status/update` → LOCKED: **ลบ session ผู้เล่นหลัง commit** · Redis ล่ม = log warn · agent ถูก LOCKED ไม่ต้องไล่ลบ Member (middleware เช็คสายบน)
- ผู้เล่นเปลี่ยนรหัสเองได้ `POST /api/v1/player/pr/auth/password/update` (เตะ session อื่น) · ⏸ BO reset รหัสยังไม่ทำ
- rate limit แบบ BO (IP 20/นาที · ผิด 5 ครั้ง / 15 นาที บล็อก 15 นาที · key `pl:login:*`)
- path ผู้เล่น `/api/v1/player/pb/*` (public) · `/api/v1/player/pr/*` (ต้อง login) · POST ทั้งหมด (N1) · `agent_id` / `member_id` อ่านจาก session เท่านั้น ห้ามรับจาก body
- รายชื่อเกมใน lobby ต้อง login (GQ9) · bundle ของ asset = Schneider ทำหลัง middleware ผู้เล่น merge (GE22-2)
- เกมปิด: เช็คตอนเดิมพัน + launch · lobby แสดงเป็นเกมปิด · maintenance ทั้งระบบ = spec แยก
- error module **`07` = player_auth** ลง `docs/ERROR_CODES.md`
- บันทึก IP ตอน login อย่างเดียว ไม่ผูก session กับ IP / device · ไม่มี captcha

## ขอบเขตรอบนี้ (Transfer)
| # | เส้น | หน้าที่ |
|---|---|---|
| PA1 | `POST /api/v1/player/pb/login` | login ตรง → `{token, expires_at, member{…}}` |
| PA3 | `POST /api/v1/player/pr/auth/logout` | ลบ session |
| PA4 | `POST /api/v1/player/pr/auth/password/update` | เปลี่ยนรหัสตัวเอง |
| PA5 | `POST /api/v1/player/pr/me/get` | โปรไฟล์ + ยอด + สถานะ (`can_bet`) |
| PA7 | `POST /api/v1/player/pr/games/list` | รายชื่อเกมของ lobby (เปิด / ปิด) |
| L2 | `POST /api/v1/transfer/login` | token + link lobby (token ใน URL) |
| TR | `POST /api/v1/transfer/register` | สร้างผู้เล่น |
| — | middleware ผู้เล่น + middleware operator (key + IP) | |
| — | `members/status/update` LOCKED ลบ session ผู้เล่น | แก้ member_management |

นอกขอบเขต: `transfer/launch` (L3 — lead 2026-10-15) · `player/pb/launch/exchange` (ยกเลิก — LQ-c ใหม่) · `seamless/*` · GetMember · balance / deposit / withdraw ของ operator (spec "Transfer API" แยก) · BO reset รหัส · เปลี่ยน username ของ Member จาก BO

## รอบ 2 — คำถามจากร่าง spec (Boiledegg `docs/lead/bo-player-auth.md` · lead 2026-10-15)
| # | เรื่อง | ผล |
|---|---|---|
| Q1 | ใครเรียก `transfer/*` ได้ | เฉพาะ `SHARE_B2C` และ `AGENT` · ประเภทอื่น = error |
| Q2 | สกุลเงินของผู้เล่นที่ระบบสร้าง | `currency` **ไม่บังคับ**: agent มีสกุลเดียว → ใช้สกุลนั้น · หลายสกุลแล้วไม่ส่ง = `422` · ส่งมาต้องอยู่ในสกุลของ agent |
| Q3 | `name` | = `player_username` |
| Q4 | ยอดตั้งต้น | `credit = 0` · `payment_type = CASH` · ผู้สร้างฝากให้ทีหลัง (spec Transfer API) |
| Q5 | PT / commission ของผู้เล่นที่ระบบสร้าง | **`pt` = ค่าเต็มที่ agent ได้รับ (agent ถือสู้เองทั้งหมด · `remain_quota` = 0) · `commission_percent` = 0** · `win_limit_per_day` = ค่า default ของ DB |
| Q6 | `can_bet` | ACTIVE + `win_limit_per_day > 0` · เกมปิดดูที่ `games/list` รายเกม |
| Q7 | `games/list` | ใช้ `GamesListService` ของ asset ซ้ำ (`app/service/asset/catalog_service.go`) · ส่ง `game_code` · `name` (จาก catalog) · `open` (เปิดจริงตามสายบน) · `game_cover` · **ไม่ส่งค่า rate** |
| Q8 | rate limit `transfer/*` | IP 20/นาที (`pl:op:ip:*`) · whitelist ว่าง = ปฏิเสธทุก IP (ACC-07) |
| Q9 | ชื่อ field ของ operator | **`snake_case`** (`agent_username` · `player_username` · `key`) เหมือน API ทั้งระบบ — แทนชื่อ camelCase ในหัวข้อการตัดสินด้านบน |
| Q10 | เปลี่ยนรหัส | 1 session ต่อคน → session ปัจจุบันใช้ต่อ ไม่ต้องทำอะไรเพิ่ม |
| Q11 | `agent@player` ชนรูปแบบ `agent@sub` | ไม่ต้องกัน — คนละตาราง + login คนละเส้น |
| Q12 | URL lobby | env `PLAYER_LOBBY_URL` (+ `.env.example`) · query ชื่อ `token` |
| Q13 | เช็คสายบนทุก request | ไม่ cache · query เดียว · วัดตอน load test (k6) |

## รอบ 3 — review ร่าง spec ทีละ endpoint (`boiledegg/player/auth` @ `013e748` · lead 2026-10-15)
**ขอบเขตรอบนี้ 8 เส้น:** PA1 `player/pb/login` · PA3 `pr/auth/logout` · PA4 `pr/auth/password/update` · PA5 `pr/me/get` · **PA6 `pr/balance/get` (ใหม่)** · PA7 `pr/games/list` · L2 `transfer/login` · TR `transfer/register`

### middleware
- แยก 3 ตัว: BO `Authenticated` (typ `bo`) · ผู้เล่น `PlayerAuthenticated` (typ `pl` · `app/internals/frontend/middleware`) · operator `OperatorAuthenticated` (`app/externals/transfer`) · ใช้ร่วมแค่ `pkg/utils/jwt.go` + secret
- ย้าย `BearerToken` จาก backoffice middleware ไป `pkg/middleware` (frontend ห้าม import backoffice)
- รายการ error ของ middleware ผู้เล่นเขียนที่เดียว (`407202` / `203` / `204` / `301` / `302`) ทุกเส้น `pr/*` อ้างถึง

### PA1 · login
- ผู้เล่นที่ไม่มีรหัส login เส้นนี้ → `407201` ข้อความเดียวกับรหัสผิด (กันเดา username) · ไม่ใช้ `407307` ในเส้นนี้
- username ไม่สนตัวพิมพ์ (แปลงตัวเล็กก่อนค้น)
- `member` = `username` · `name` · `currency` · `status` · `can_bet` (ไม่ใส่ credit / id)
- HTTP status ของ error ทั้ง module ให้ตรงแบบ BO auth (module 01)

### PA3 · logout — ตามร่าง

### PA4 · password/update
- `407307` (ไม่มีรหัส) ใช้ในเส้นนี้ได้
- รหัสเดิมผิดนับรวมตัวนับ PA-41 · ผิด 5 ครั้ง / 15 นาที → `407303` 15 นาที
- ใช้ได้กับผู้เล่นจาก operator ที่ตั้งรหัสไว้ (TR)

### PA5 · me/get
- **เอา `credit` ออก** → `username` · `name` · `currency` · `status` · `can_bet`
- ยังไม่ใส่ win limit / ยอดที่เหลือวันนี้ (→ spec Bet)

### PA6 · `POST /api/v1/player/pr/balance/get` (ใหม่)
- response `{currency, credit}` · อ่าน `user_members.credit` ตรง ไม่ cache · SUSPENDED เรียกได้
- **ใช้เป็น ping กันหลุด idle** (หน้าเกมรีเฟรชยอดทุก 5–10 นาที) ไม่มีเส้น ping แยก
- Seamless (รอบหน้า) เปลี่ยนเฉพาะเส้นนี้ไปเรียก GetBalance โดย contract เดิม

### PA7 · games/list
- **เพิ่มชื่อเกม `name_th` / `name_en` ใน catalog ฝั่ง backend** แล้วส่งใน `items[]` (catalog ของ asset ตอนนี้มีแค่ `game_code` + `game_cover`) — แทน Q7 รอบ 2 ที่เขียนว่าชื่อมาจาก catalog
- เรียงตาม catalog คงที่ในโค้ด · `open` ดูจาก agent + สายบนเท่านั้น (Member ไม่มีสวิตช์เกมรายคน)

### L2 · transfer/login — request `agent_username` · `key` · `player_username` · `currency` (ไม่บังคับ) · `language` (ไม่บังคับ)
- ผู้เล่นมีแล้วแต่ `currency` ไม่ตรง → `422` "สกุลเงินไม่ตรงกับผู้เล่น"
- agent (หรือสายบน) LOCKED → ปฏิเสธทุก `transfer/*` (`407302`) · SUSPENDED → ผู้เล่นเดิม login ได้ `can_bet = false`
- `language` `th` / `en` default `th` → ต่อ `&lang=` ท้าย link
- login ผู้เล่นใหม่พร้อมกัน → unique constraint กัน · insert ชน = อ่านผู้เล่นที่สร้างแล้วทำต่อ ไม่ error · มี test ยิงพร้อมกัน
- ผู้เล่นที่สร้างอัตโนมัติไม่มีรหัส

### TR · transfer/register — request + **`password` (ไม่บังคับ — lead เพิ่ม)**
- ส่ง `password` = ผู้เล่น login ตรง (PA1) ได้ด้วยชื่อเต็ม `agent01@p001` · ไม่ส่ง = เข้าทาง operator อย่างเดียว — **แทน PQ9-a เดิม**
- กติการหัส = AUTH-36 (ฟังก์ชันเดียวกับ BO ใน `app/core`) · ไม่ผ่าน `422` · ห้าม log · ไม่คืนรหัส
- response `{username, currency}`
- register ซ้ำ = `407401` · spec เขียนว่า operator ถือเป็น "สร้างไว้แล้ว"
- agent SUSPENDED **สร้างผู้เล่นใหม่ไม่ได้** (ทั้ง register และ auto-create ใน L2) · ผู้เล่นเดิม login ได้
- operator reset / เปลี่ยนรหัสให้ผู้เล่น — ยังไม่ทำ

## รอบ 4 — approve spec (`boiledegg/player/auth` @ `f50a8b8` · lead zerph 2026-10-15)
**✅ อนุมัติ spec `docs/modules/player_auth.md`** โดยมีเงื่อนไข 1 ข้อ — แก้แล้วเริ่ม implement ผ่าน PR ได้ทันที (commit แรก: ย้าย `BearerToken` ไป `pkg/middleware`)

| # | เรื่อง | ผล |
|---|---|---|
| R4-1 | error ตอน agent SUSPENDED สร้างผู้เล่นใหม่ (register / auto-create ใน `transfer/login`) | **รับ `407308` (HTTP 403) "บัญชีผู้สร้างถูกระงับ สร้างผู้เล่นใหม่ไม่ได้"** |
| R4-2 | PA-09 `win_limit_per_day` ของผู้เล่นที่ระบบสร้าง = "ค่า default ของ DB" | **แก้**: ใช้ค่าตั้งต้นตาม MGMT-65 ล่าสุด (เพดาน 1,000,000 บาท แปลงเป็นสกุลของผู้เล่น — member-payment รอบ 2) ผ่านฟังก์ชันเดียวกับ `members/create` · ห้ามพึ่งค่า default ของคอลัมน์ใน DB |

Boiledegg: แก้ PA-09 + ใส่ "อนุมัติโดย: zerph · 2026-10-15" ในหัว spec · เอา **[เสนอ]** ออกจาก `407308`

## รอบ 5 — ระหว่าง implement (`boiledegg/player/auth` @ `0ce5c78` · lead 2026-10-15)
| # | เรื่อง | ผล |
|---|---|---|
| R5-1 | migration `20261015140000` แก้ CHECK `ck_user_members_username` เป็น `^[a-z0-9]{3,32}(@[a-z0-9]{3,32})?$` + `VARCHAR(65)` | **อนุมัติ** (CHECK เดิมรับ `agent@player` ไม่ได้) · down ย้อนไม่ได้เมื่อมีผู้เล่นแบบ `@` — ตามหมายเหตุใน migration |
| R5-2 | ชื่อเกมใน catalog (`app/core/game/game.go`) | **ยืนยันตามที่ตั้ง**: `scratch_card` สแครชการ์ด / Scratch Card · `coin_toss` โยนเหรียญ / Coin Toss · `rock_paper_scissors` เป่ายิ้งฉุบ / Rock Paper Scissors |
| R5-3 | verify โค้ด | เปิด verify ที่ `0ce5c78` ทันที ไม่รอ PR |

## รอบ 6 — ผล verify โค้ด (`0ce5c78` + `a529bcd` · lead 2026-10-15 · รายงาน verify-player-auth)
การตัดสินของ lead ตรงครบ · `key` ไม่อยู่ใน log ใด · must-fix 0 · should-fix 3 · nit 7 → **merge ได้เมื่อครบเงื่อนไขด้านล่าง**

| # | เรื่อง | ผล |
|---|---|---|
| S1 | `c.IP()` เมื่อตั้ง `TRUSTED_PROXIES` ได้ `X-Forwarded-For` ทั้งก้อน (`"a, b"`) → operator โดน `407305` เมื่อมี proxy หลายชั้น · client ปลอม XFF หลบ rate limit ได้ (ของเดิม — กระทบ BO login ด้วย) | **แยกเป็นงานแก้ ทำหลัง merge แต่ก่อนเปิดให้ operator ใช้จริง** · อ่าน IP จาก header ที่ ingress ของเราตั้ง หรือไล่ XFF จากขวาข้าม proxy ที่เชื่อถือ · **ห้ามแก้ด้วยเปิด `EnableIPValidation` อย่างเดียว** (ได้ IP ซ้ายสุดที่ client ปลอมได้ → หลบ whitelist) · ทดสอบยิงผ่าน ingress จริงบน dev |
| S2 | ผู้เล่นที่สร้างผ่าน `transfer/register` / สร้างอัตโนมัติ ไม่มี `account_change_logs` `CREATE` | **เขียน log** · ผู้ทำ = agent เจ้าของ `key` (เหมือน `members/create`) |
| S3 | test PA-12 เช็คแค่ response | **เพิ่ม test** ว่า `key` / `password` ไม่อยู่ใน log |

**เงื่อนไข merge:** แก้ S2 + S3 · แนบผล integration test (หรือ CI เขียว) · nit 7 ข้อใน `reviews/mini-game-api/verify-player-auth.md` (error list ของ L2 ขาด `407301` / `503` · PA4 ผิดครบแล้วตัด session ให้เขียนใน spec · `last_login_ip` ของ L2 = IP ของ operator ฯลฯ) ทำได้ใน PR เดียวกัน
**ก่อน deploy dev:** migrate `20261015140000` + ตั้ง `PLAYER_LOBBY_URL` (`TRUSTED_PROXIES` หลังแก้ S1)

## รอบ 7 — verify รอบ 2–3 + S1 (dev @ `3c42948` · lead 2026-10-15)
- S2 / S3 ✅ · **S1 client IP (`308f76d`) ✅ ออกแบบถูก ไม่มีทางปลอม IP** (peer ที่ไม่ใช่ proxy ที่เชื่อถือ = socket IP · ไล่ XFF จากขวา · fail-closed · ไม่อ่าน X-Real-IP / Forwarded · ไม่เหลือ `c.IP()` ใน request path)
- integration test: รับผลที่ Boiledegg รันเอง (`go test -count=1 -tags=integration ./...` 27 package ok · test งานนี้ 35 ตัว) · **ก่อน deploy dev ออก tag `dev-*` ให้ CI รันซ้ำ** (รวม golangci-lint)

| # | เรื่อง | ต้องทำ | เมื่อไร |
|---|---|---|---|
| S1-a | `TRUSTED_PROXIES` ที่พิมพ์ผิดถูกข้ามเงียบ → ทุกคนได้ IP ของ proxy → rate limit ใช้ร่วมกันทั้งระบบ (BO / player login ล่ม) + operator `407305` | validate ใน `configs.Load` (ผิด = boot ไม่ผ่าน) แล้ว parse ครั้งเดียว | ก่อน deploy dev |
| S1-b | ยังไม่ได้ยิงผ่าน ingress จริง | หลัง deploy dev: ยืนยันว่า ingress **ต่อท้าย** XFF (ไม่เขียนทับ) + ใส่ CIDR ทุกชั้น (ALB / nginx pod / CDN ถ้ามี) — CIDR ได้จาก lead / infra | ก่อนเปิด `transfer/*` ให้ operator |
| N5 | `operator_service.go:136-143` ออก session (เตะ session เดิม) ก่อนตรวจ `PLAYER_LOBBY_URL` → ค่าผิดรูปแบบ ผู้เล่นโดนเตะแต่ไม่ได้ link | ตรวจ `PLAYER_LOBBY_URL` ตอน `configs.Load()` (ถ้าตั้งต้องเป็น absolute URL) หรือเรียก `LobbyLink` ก่อน `issueSession` | ก่อนเปิดให้ใช้จริง |
| N3 | `session_service.go:226-237` middleware ใช้ 2 query (Member + สถานะสายบน) แต่ spec เขียน "query เดียว" (ไม่ใช่ N+1) | ปล่อยได้ (แก้ spec ให้ตรง) หรือรวมเป็น JOIN `cnf->'parent'` ถ้า k6 ชี้ว่าเป็นคอขวด | ตอนว่าง |
| N4 | `catalog_service.go` `GameCoversOfAgentService` · `player_service.go:323-346` — `chainOf` query สายซ้ำทั้งที่ `Player.UplineIDs` มีแล้ว · การคิด `open` เขียนซ้ำกับ `agent_management.effectiveStatusGame` | ส่ง chain เข้า wrapper / export helper `open` ตัวเดียว | ตอนว่าง |
| N7 | `app/internals/backoffice/player_auth*_integration_test.go` — test ของ frontend / transfer อยู่ใน package backoffice | ย้ายไป package กลาง (เช่น `app/integration`) | ตอนว่าง |
| nit | spec PA-10 ยังไม่ถูกแก้ทั้งที่ commit S1 บอกว่าแก้ · `docs/lead/bo-player-auth.md` ยังเขียน "ยังไม่ merge" | แก้ doc | ก่อน deploy dev |

**ก่อน deploy dev:** S1-a + nit doc · migrate RDS dev `20261015150000` (Maofoy) · ตั้ง `PLAYER_LOBBY_URL` + `TRUSTED_PROXIES` · tag `dev-*` ให้ CI เขียว

## รอบ 8 — S1-b ผลตรวจ network จริงของ dev (infra 2026-10-16 · lead zerph เลือกทาง B)
**เส้นทาง:** Client → Cloudflare (proxied) → AWS NLB (L4) → NodePort (kube-proxy · `externalTrafficPolicy: Cluster` → SNAT) → ingress-nginx → pod · host `dev-mini-api.pirate168.com` · cluster `thor-kube1`
**ปัญหา:** ingress-nginx **เขียนทับ** `X-Forwarded-For` ด้วย IP ของ node (`use-forwarded-headers` / `compute-full-forwarded-for` = default false) → app ได้ IP ของ node เสมอ (ทุกคนใช้ IP เดียวกัน) ไม่ว่าตั้ง `TRUSTED_PROXIES` อะไร · IP จริงอยู่ใน `CF-Connecting-IP` / `X-Original-Forwarded-For`

**ตัดสิน: ทาง B — แก้ที่ infra · ไม่แก้โค้ด `ClientIP`** (ไม่เลือกทาง A อ่าน `CF-Connecting-IP` เพราะ NLB เปิด public ยิงตรงแล้วปลอม header ได้ → หลบ whitelist ของ operator)
1. ingress-nginx (ทั้ง cluster): `use-forwarded-headers: "true"` · `compute-full-forwarded-for: "true"` · `proxy-real-ip-cidr` = CIDR Cloudflare (v4 + v6) · Service `externalTrafficPolicy: Local` · ถ้าทำได้จำกัด NLB / NodePort ให้รับเฉพาะ IP Cloudflare
   - มีผลทุก app ใน `thor-kube1` → infra แจ้งรายชื่อ app ที่ได้รับผล + นัดเวลา apply
2. หลัง apply ยิง `/__xff-test` ซ้ำ: คาด `X-Forwarded-For = <ค่าที่ client ใส่>, <IP จริง>, <IP Cloudflare>` → `ClientIP` ไล่จากขวาข้าม IP ที่เชื่อถือ ได้ IP จริง · ยิงตรงเข้า NLB ก็ปลอมไม่ได้ (nginx ต่อท้าย IP จริงของคนยิงเป็นตัวขวาสุด)
3. `TRUSTED_PROXIES=192.168.0.0/16,<CIDR Cloudflare v4 + v6 ทั้งหมด>` (configmap minigame dev)
4. **ระหว่างยังไม่ apply ข้อ 1:** `TRUSTED_PROXIES=""` · ทุกคนบน dev ได้ IP ของ nginx pod (rate limit login ใช้ร่วมกัน — รับได้ชั่วคราวบน dev) · **ห้ามเปิด `transfer/*` ให้ operator จริงจนกว่าข้อ 1–3 เสร็จและทดสอบผ่าน**

## รอบ 9 — S1-b เสร็จ (infra apply 2026-10-16 ~20:00)
- ingress-nginx ของ `thor-kube1`: XFF ต่อท้าย (`$full_x_forwarded_for`) · `externalTrafficPolicy: Local` · security group เปิด NodePort ให้เฉพาะ 15 CIDR ของ Cloudflare · ยิงตรงเข้า NLB = ถูกบล็อก
- ทดสอบจริง (IP ต้นทาง 52.76.142.142): ใส่ XFF ปลอม → `X-Forwarded-For = 1.2.3.4,52.76.142.142, 172.68.85.210` (ปลอม · จริง · Cloudflare) · ไม่ใส่ → `52.76.142.142, 104.23.176.10`
- `TRUSTED_PROXIES=192.168.0.0/16,<Cloudflare v4 + v6>` (configmap minigame-dev `91696867`) · app boot ผ่าน · log เห็น IP จริงของ client → `ClientIP` ไล่จากขวาข้าม Cloudflare / VPC ได้ IP จริง · ค่าปลอมทางซ้ายไม่มีผล
- **S1-b ปิด** · rate limit login (BO / player) แยกตามผู้ใช้แล้ว · IP whitelist ของ `transfer/*` ใช้งานได้จริง
- ยังค้างก่อนเปิด `transfer/*` ให้ operator จริง: `PLAYER_LOBBY_URL` (รอหน้าบ้าน) · spec Transfer API (ยอด / ฝาก / ถอน) ถ้าต้องให้ operator มีเงินเล่น
- ตรวจผลกระทบ app อื่นใน cluster (lead 2026-10-16): landingpage-api · sarahuay-api · rng_result-api · askmelotto_game-api (lucky48) · askmelotto-api ไม่พัง ไม่แก้โค้ด · askmeboard / aff-member / game-market-poc ไม่ได้ใช้แล้ว
- secret ดิบใน configmap minigame-dev (repo private): ย้ายเป็น Secret + ผูก IRSA **ก่อนตั้ง uat / prod**
