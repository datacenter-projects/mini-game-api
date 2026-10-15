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
| L2 | `POST /api/v1/transfer/login` | server operator สาย Transfer | token + link lobby (ติดตั๋ว) |
| L3 | `POST /api/v1/transfer/launch` | server operator สาย Transfer · + `game_code` | token + link เข้าเกมนั้น (ติดตั๋ว) |
| L4 | `POST /api/v1/seamless/login` | server operator สาย Seamless | ⏸ พักไว้ |
| L5 | `POST /api/v1/seamless/launch` | server operator สาย Seamless | ⏸ พักไว้ |

- `redirectLogin` แบบ sportbook = launcher → ใช้ชื่อ `launch`
- launch **ไม่ต้อง login ก่อน** — operator ส่งครั้งเดียวจบ
- **link ติดตั๋วใช้ครั้งเดียว อายุ 60 วินาที ไม่ใช่ token** · หน้าเกมแลกตั๋วเป็น session ที่ `POST /api/v1/player/pb/launch/exchange` · token ยังคืนให้ operator ใน response
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
- **1 session ต่อ Member** — login / launch / exchange ใหม่เตะอันเก่า · session เก่าได้ code "ถูก login จากที่อื่น"

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
- rate limit แบบ BO (IP 20/นาที · ผิด 5 ครั้ง / 15 นาที บล็อก 15 นาที · key `pl:login:*`) · ตั๋วใช้ครั้งเดียว + IP limit เดียวกัน
- path ผู้เล่น `/api/v1/player/pb/*` (public) · `/api/v1/player/pr/*` (ต้อง login) · POST ทั้งหมด (N1) · `agent_id` / `member_id` อ่านจาก session เท่านั้น ห้ามรับจาก body
- รายชื่อเกมใน lobby ต้อง login (GQ9) · bundle ของ asset = Schneider ทำหลัง middleware ผู้เล่น merge (GE22-2)
- เกมปิด: เช็คตอนเดิมพัน + launch · lobby แสดงเป็นเกมปิด · maintenance ทั้งระบบ = spec แยก
- error module **`07` = player_auth** ลง `docs/ERROR_CODES.md`
- บันทึก IP ตอน login อย่างเดียว ไม่ผูก session กับ IP / device · ไม่มี captcha

## ขอบเขตรอบนี้ (Transfer)
| # | เส้น | หน้าที่ |
|---|---|---|
| PA1 | `POST /api/v1/player/pb/login` | login ตรง → `{token, expires_at, member{…}}` |
| PA2 | `POST /api/v1/player/pb/launch/exchange` | `{ticket}` → token |
| PA3 | `POST /api/v1/player/pr/auth/logout` | ลบ session |
| PA4 | `POST /api/v1/player/pr/auth/password/update` | เปลี่ยนรหัสตัวเอง |
| PA5 | `POST /api/v1/player/pr/me/get` | โปรไฟล์ + ยอด + สถานะ (`can_bet`) |
| PA7 | `POST /api/v1/player/pr/games/list` | รายชื่อเกมของ lobby (เปิด / ปิด) |
| L2 | `POST /api/v1/transfer/login` | token + link lobby |
| L3 | `POST /api/v1/transfer/launch` | token + link เข้าเกม |
| TR | `POST /api/v1/transfer/register` | สร้างผู้เล่น |
| — | middleware ผู้เล่น + middleware operator (key + IP) | |
| — | `members/status/update` LOCKED ลบ session ผู้เล่น | แก้ member_management |

นอกขอบเขต: `seamless/*` · GetMember · balance / deposit / withdraw ของ operator (spec "Transfer API" แยก) · BO reset รหัส · เปลี่ยน username ของ Member จาก BO
