# Player Auth (ผู้เล่นเข้าระบบ) — Spec

- สถานะ: **APPROVED**
- อนุมัติโดย: zerph · วันที่: 2026-10-15 ([review รอบ 4](../reviews/player-auth.md) — เงื่อนไข R4-2 แก้ใน PA-09 แล้ว)
- ผู้ทำ: Boiledegg · branch `boiledegg/player/auth`
- ชื่อ module ในโค้ด: `player_auth` (`app/internals/frontend/{controllers,dto,middleware}/player_auth`, `app/externals/transfer/{controllers,dto,middleware}`, `app/service/player_auth`, `app/core/player_auth`)
- ที่มาของการตัดสิน: [docs/reviews/player-auth.md](../reviews/player-auth.md) (lead zerph 2026-10-14 / 15 · PQ1–PQ14 · LQ-c · รอบ 2 Q1–Q13 · รอบ 3 ทีละ endpoint)
- โค้ดเก่าที่อ้างอิง (extract พฤติกรรมเท่านั้น): `askmelotto_minigame-api` — `/provider/launch` (สร้างผู้เล่น ไม่ให้ token ไม่มีรหัส **ห้ามลอกแบบ**) ·
  แบบที่ใกล้สุดคือ transferwallet `loginRequest` / sportbook `redirectLogin` ของ `askmelotto-api`

> เอกสารนี้คือ source of truth ของ module — โค้ดและ test ต้องตรงกับเอกสารนี้
> ทุกข้อในไฟล์นี้ผ่าน lead review และอนุมัติแล้ว (รอบ 2–4)

## 1. หน้าที่ของ module

ยืนยันตัวตนของ**ผู้เล่น (Member)** ที่เข้าหน้าเกม สร้าง / ยกเลิก session และมี middleware ให้ route ฝั่งผู้เล่นตรวจว่า request มาจาก session ที่ยังใช้ได้
มี 2 ทางเข้า:

1. ผู้เล่น login เองด้วย username + password (`player/pb/login`) — ใช้ได้กับ Member ที่สร้างจากหลังบ้าน (BO) และผู้เล่นจาก operator ที่ operator ตั้งรหัสไว้ตอน `transfer/register`
2. server ของ operator สาย Transfer เรียก `transfer/login` ส่ง `agent_username` + `key` + `player_username` ได้ token + link เข้า lobby กลับไป · ผู้เล่นที่ยังไม่มี = สร้างให้อัตโนมัติ

**รอบนี้ทำเฉพาะสาย Transfer** — Seamless (`seamless/*` · GetMember · mock operator) พักไว้ แต่ออกแบบ middleware / session ให้ใช้ร่วมกันได้

**อยู่ใน scope รอบนี้ (8 เส้น + งานประกอบ):** PA1 `player/pb/login` · PA3 `logout` · PA4 `password/update` · PA5 `me/get` · **PA6 `balance/get`** · PA7 `games/list` · L2 `transfer/login` · TR `transfer/register` · middleware ผู้เล่น · middleware operator · `members/status/update` → LOCKED ลบ session · ย้าย `BearerToken` ไป `pkg/middleware`

**ยังไม่อยู่:** `transfer/launch` (L3 — lead 2026-10-15) · `player/pb/launch/exchange` (ยกเลิก — LQ-c) · `seamless/*` · GetMember · balance / deposit / withdraw ของ operator (spec "Transfer API" แยก) · BO reset รหัสผู้เล่น · operator reset / เปลี่ยนรหัสให้ผู้เล่น · เปลี่ยน username ของ Member ที่สร้างจาก BO · bundle asset ของหน้าเกม (Schneider — GE22-2) · win limit / ยอดที่เหลือวันนี้ใน `me/get` (→ spec Bet) · maintenance ทั้งระบบ (spec แยก)

## 2. คำศัพท์

| คำ | ความหมาย |
|---|---|
| ผู้เล่น / Member | แถวใน `user_members` |
| ผู้สร้าง (`agent_id`) | บัญชีฝั่ง agent ที่เป็นแม่ตรงของ Member |
| operator | ลูกค้าที่มี server เชื่อมระบบเรา ถือ `agent_username` + Key ของ**ผู้สร้าง**ผู้เล่น (account.md ACC-01) |
| `agent_username` / `key` | username และ Key ของผู้สร้าง (`api_credentials`) ที่ operator ส่งมาใน body |
| `player_username` | ชื่อผู้เล่นที่ operator ส่งมา (ยังไม่มี prefix) |
| username ผู้เล่นจาก operator | `{agent_username}@{player_username}` — ระบบต่อ prefix ให้เอง |
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
| PA-03 | `player/pb/login` ใช้ได้ทุกสาย · ผู้เล่นที่**มีรหัสผ่าน**เท่านั้น: Member จาก BO · ผู้เล่นจาก operator ที่ operator ส่ง `password` ตอน `transfer/register` (login ด้วยชื่อเต็ม `agent01@p001`) · ผู้เล่นที่ `password_hash = ''` → `407201` ข้อความเดียวกับรหัสผิด (กันเดา username) **ไม่ใช้ `407307` ในเส้นนี้** |
| PA-04 | username ที่รับทางตรง (`player/pb/login`): trim + ตัวพิมพ์เล็กก่อนค้นหา · ไม่พบ หรือรหัสผิด → `407201` ตัวเดียว และเวลาตอบใกล้เคียงกัน (เทียบ dummy hash เหมือน AUTH-02) |
| PA-05 | username ผู้เล่นจาก operator = `{agent_username}@{player_username}` · `agent` ต้องเป็นเจ้าของ `agent_username` / `key` ที่เรียก (= แม่ตรงของผู้เล่น) · ส่วน player: `a-z0-9` ยาว 3–32 (รับตัวใหญ่ เก็บตัวเล็ก) · ไม่ผ่าน = `422` บอก field `player_username` |
| PA-06 | `user_members.username` ขยายเป็น `VARCHAR(65)` (32 + `@` + 32) · ห้ามซ้ำทั้งระบบเหมือนเดิม (MGMT-05) · `agent@player` ชนรูปแบบ `agent@sub` ของบัญชีย่อย **ไม่ต้องกัน** — คนละตาราง + login คนละเส้น (lead Q11) |
| PA-07 | `transfer/login`: ไม่มีผู้เล่น = **สร้างอัตโนมัติ** แล้ว login ต่อในคำขอเดียว · มีแล้ว = login เลย · ผู้เล่นที่สร้างให้ไม่มีรหัส (`password_hash = ''`) · ค่าตั้งต้นตาม PA-09 · `currency` ตาม PA-08A |
| PA-08 | `transfer/register`: สร้างผู้เล่นอย่างเดียว ไม่ออก token · ส่ง `password` (ไม่บังคับ) = ผู้เล่น login ตรงได้ (PA-03) · ไม่ส่ง = เข้าทาง operator อย่างเดียว · กติการหัส = AUTH-36 (ฟังก์ชันเดียวกับ BO ใน `app/core`) ไม่ผ่าน `422` · **ห้าม log รหัส · ไม่คืนรหัส** · มีผู้เล่นแล้ว = `407401` (operator ถือเป็น "สร้างไว้แล้ว" — เขียนในเอกสาร frontend) |
| PA-08A | **สกุลเงิน** (lead Q2 / รอบ 3): `currency` ใน L2 / TR **ไม่บังคับ** · agent มีสกุลเดียว → ใช้สกุลนั้น · agent มีหลายสกุลแล้วไม่ส่ง = `422` · ส่งมาต้องอยู่ในสกุลของ agent (`422`) · **ผู้เล่นมีอยู่แล้วแต่ `currency` ที่ส่งมาไม่ตรง → `422` "สกุลเงินไม่ตรงกับผู้เล่น"** (L2) |
| PA-08B | **สร้างพร้อมกัน**: สร้างผู้เล่นใหม่ 2 คำขอพร้อมกัน — ตัวที่แพ้ unique constraint: `register` ได้ `407401` · `login` อ่านผู้เล่นที่สร้างแล้วทำต่อ **ไม่ error** (ห้ามคืน 500) · มี integration test ยิงพร้อมกัน |
| PA-09 | ค่าตั้งต้นของผู้เล่นที่ระบบสร้างให้ (lead Q3–Q5): `name` = `player_username` · `credit = 0` · `payment_type = CASH` (ผู้สร้างฝากให้ทีหลัง — spec Transfer API) · `pt` = **ค่าเต็มที่ agent ได้รับ** (agent ถือสู้เองทั้งหมด · `remain_quota = 0`) · `commission_percent = 0` · **`win_limit_per_day` = ค่าตั้งต้นตาม MGMT-65** (lead R4-2 · แก้ความหมายโดย member-payment รอบ 2.1): **หน่วยบาท (THB) เสมอ ไม่มีสกุลเงิน** · ใช้ค่าตั้งต้น `DefaultWinLimitPerDay` (`app/core/member_management` = 10,000,000 บาท) ตัวเดียวกับที่ `members/create` ให้เมื่อไม่ส่งค่า · **ไม่ต้องแปลงสกุล ไม่ใช้ `currency_rate`** (การแปลงอยู่ที่ module เดิมพัน) · **ห้ามพึ่งค่า `DEFAULT` ของคอลัมน์ใน DB** (ส่งค่านี้ชัดเจนตอน insert) · `status = ACTIVE` |
| PA-09A | Member ที่สร้างจาก BO ยังใช้ username แบบเดิม (MGMT-05) — รอบนี้**ไม่แก้** `members/create` |
| PA-09B | **เขียน `account_change_logs` `CREATE`** (lead S2 รอบ 6) เมื่อสร้างผู้เล่นให้ operator (`transfer/register` และสร้างอัตโนมัติใน `transfer/login`) ใน tx เดียวกับการสร้าง — เหมือน `members/create` (MGMT-60): ผู้ทำ = **agent เจ้าของ key** · `ip` = IP ของ operator · `new_value` = username · name · agent_id · currency · pt ต่อกลุ่ม · payment_type · win_limit_per_day · `has_password` · `source = transfer_api` (**ห้ามมีรหัสผ่านหรือ hash**) · login ซ้ำ / `register` ซ้ำไม่เขียน log เพิ่ม · ชนกันตอนสร้างพร้อมกัน tx ที่แพ้ rollback ทั้งก้อน (ไม่มี log เกิน 1 แถวต่อผู้เล่น) |

### 3.2 operator ยืนยันตัว (PQ4)

| ID | Rule |
|---|---|
| PA-10 | middleware operator (ใช้กับ `transfer/*` ทุกเส้น และ `seamless/*` ภายหลัง): ตรวจ `agent_username` + `key` ใน body → หาเจ้าของ Key → ตรวจ IP ผู้เรียกอยู่ใน `api_allowed_ips` ของเจ้าของ (ACC-07) · ไม่มี `sign` · **whitelist ว่าง = ปฏิเสธทุก IP** (ACC-07) · **IP ของผู้เรียกอ่านด้วย `middleware.ClientIP`** (lead S1 รอบ 6): เชื่อ `X-Forwarded-For` เฉพาะเมื่อ connection มาจาก proxy ใน `TRUSTED_PROXIES` แล้วไล่จากขวาไปซ้าย ข้ามทุก IP ที่อยู่ใน `TRUSTED_PROXIES` ใช้ IP แรกที่ไม่ใช่ proxy · ค่าอ่านไม่ได้แทรกในสาย = fail-closed (ใช้ IP ของ proxy) · ไม่ตั้ง `TRUSTED_PROXIES` = IP ของ connection · **ไม่ใช้ `c.IP()`** · `TRUSTED_PROXIES` พิมพ์ผิด = boot ไม่ขึ้น (S1-a) |
| PA-11 | `agent_username` ไม่มี / `key` ไม่ตรงเจ้าของ → `407206` (ข้อความเดียวกันทั้งสองกรณี ไม่บอกว่าผิดที่ไหน) · IP ไม่อยู่ใน whitelist → `407305` |
| PA-12 | **mask `key` ใน log ทุกจุด** (กฎข้อ 26) รวม request log กลางและ audit · ห้ามใส่ `key` / `password` ใน error message หรือ log |
| PA-13 | สถานะของ agent เจ้าของ Key (lead รอบ 3): **agent หรือสายบนถูก `LOCKED` → ปฏิเสธทุก `transfer/*` ด้วย `407302`** · agent `SUSPENDED` → **ผู้เล่นเดิม login ได้** (`can_bet = false`) แต่ **สร้างผู้เล่นใหม่ไม่ได้** ทั้ง `register` และ auto-create ใน `login` → `407308` |
| PA-14 | ผู้สร้างที่เรียก `transfer/*` ต้องเป็น **`SHARE_B2C` หรือ `AGENT`** เท่านั้น (lead Q1) — ประเภทอื่น `407306` |

### 3.3 session (PQ5 / PQ6)

| ID | Rule |
|---|---|
| PA-20 | แยก middleware 3 ตัว (lead รอบ 3): BO `Authenticated` (typ `bo`) · ผู้เล่น `PlayerAuthenticated` (typ `pl` · `app/internals/frontend/middleware`) · operator `OperatorAuthenticated` (`app/externals/transfer`) · **ใช้ร่วมแค่ `pkg/utils/jwt.go` + secret** · middleware ผู้เล่นปฏิเสธ token `typ` อื่น และ middleware BO ปฏิเสธ `typ = "pl"` (กัน token ข้ามฝั่ง) |
| PA-20A | ย้าย `BearerToken` จาก backoffice middleware ไป `pkg/middleware` (frontend **ห้าม import backoffice** — CLAUDE.md ข้อ 29) · รายการ error ของ middleware ผู้เล่น (`407202` / `203` / `204` / `301` / `302`) เขียนที่เดียวในหัวข้อ 5 ทุกเส้น `pr/*` อ้างถึง |
| PA-21 | หมดอายุ: ไม่มี request เกิน **30 นาที** (idle) หรือครบ **4 ชั่วโมง** นับจาก login (absolute) · ค่าตั้งผ่าน env (เพิ่ม key ใน `config.go` + `.env.example` PR เดียวกัน) · ไม่มี refresh — ทุก request ที่ผ่าน middleware ต่ออายุ idle (ไม่เกิน absolute) · **หน้าเกมใช้ `balance/get` (PA6) เป็น ping กันหลุด** รีเฟรชยอดทุก 5–10 นาที ไม่มีเส้น ping แยก |
| PA-22 | **1 session ต่อ Member** — login (ทางไหนก็ตาม) ใหม่เตะอันเก่าทันที · token เก่าได้ `407204` "ถูก login จากที่อื่น" |
| PA-23 | ถ้าสร้าง session ใน Redis ไม่สำเร็จ → login ล้ม (`500`) ห้ามออก token ที่ไม่มี session รองรับ (เหมือน AUTH-13) |
| PA-24 | token ไม่มี / ลายเซ็นผิด / หมดอายุ → `407202` · session ไม่มีแล้ว (logout / idle / absolute) → `407203` |
| PA-25 | บันทึก `last_login_at` / `last_login_ip` ตอน login (ไม่ผูก session กับ IP / device · ไม่มี captcha) · บันทึกไม่ได้ไม่ทำให้ login ล้ม (log warn) · IP ตอนใช้งานไม่ตรงกับตอน login ไม่ block ไม่ต้อง log · **`transfer/login` (L2): `last_login_ip` = IP ของ server operator ที่เรียก (ไม่ใช่ IP ของผู้เล่น)** — ระบบไม่เห็น IP ผู้เล่นในเส้นนี้ |
| PA-26 | logout (`pr/auth/logout`) ยกเลิก session ปัจจุบัน · idempotent: เรียกซ้ำ / session หลุดไปแล้ว ตอบสำเร็จเหมือนกัน · token ปลอม / เลย absolute `407202` · logout ด้วย token เก่าไม่กระทบ session ปัจจุบัน |
| PA-27 | HTTP status ของ error ทั้ง module ตรงแบบ BO auth (module 01): `407201–204` → 401 · `407301–303` → 403 · error business อื่นตอบ 200 · `429` / `500` ตาม common |

### 3.4 สถานะ (PQ7) — สถานะแย่ที่สุดของ Member + ผู้สร้าง + สายบนทุกคน

| สถานะที่ใช้งานจริง | login / launch | ใช้งานหลัง login | เดิมพัน | session |
|---|---|---|---|---|
| `LOCKED` (ตัวเองหรือสายบน) | ❌ `407301` / `407302` | ❌ | ❌ | ลบทันที |
| `SUSPENDED` | ✅ | ✅ ดูข้อมูล / ประวัติ / lobby / `balance/get` | ❌ | คงไว้ |
| `ACTIVE` | ✅ | ✅ | ✅ (ถ้าเกมเปิด) | |

| ID | Rule |
|---|---|
| PA-30 | middleware ผู้เล่นเช็คสถานะตัวเอง + สายบน**ทุก request** (ไม่ไล่ลบ Member เมื่อ agent ถูก LOCKED — เช็คตอนใช้งานแทน) · **ไม่ cache** · 2 query (Member + สถานะสายบนทั้งสาย ไม่ N+1 · lead N3) · วัดตอน load test (k6) (lead Q13) |
| PA-31 | login เช็คสถานะ**หลัง**รหัสผ่านถูก (เหมือน AUTH-04) — รหัสผิดได้ `407201` ไม่บอกว่าบัญชีถูกล็อก · `transfer/login` ไม่มีรหัส จึงเช็คสถานะหลัง Key ผ่าน |
| PA-32 | LOCKED ระหว่างใช้งาน → request ถัดไป `407301` (หรือ `407302` ถ้าสายบนถูกล็อก) และลบ session |
| PA-33 | `SUSPENDED` ผ่าน middleware ได้ · `can_bet = false` ใน `me/get` · module เดิมพัน (Bet) ต้องเช็คเอง · SUSPENDED cash-out Streak ที่ค้างได้ (spec Bet) |
| PA-34 | `members/status/update` ตั้ง `LOCKED` → **ลบ session ผู้เล่นหลัง commit** · Redis ล่ม = log warn ไม่ทำให้ request ล้ม (middleware เช็คสถานะทุก request อยู่แล้ว จึงยังกันได้) · ปิดข้อ MGMT-30A (open item MQ7) |
| PA-35 | `can_bet` = สถานะที่ใช้งานจริงเป็น `ACTIVE` และ `win_limit_per_day > 0` (MGMT-65) · เกมปิดดูที่ `games/list` รายเกม (lead Q6) |

### 3.5 กันเดารหัส / ยิงถี่ (แบบ BO)

| ID | Rule |
|---|---|
| PA-40 | `player/pb/login`: เรียกเกิน **20 ครั้ง/นาที ต่อ IP** → `429` |
| PA-41 | ผิด **5 ครั้งภายใน 15 นาที** ต่อ username → username นั้น login ไม่ได้ **15 นาที** (`407303`) แม้รหัสถูก · ตัวนับเดียวกันนับ **login ผิด (PA1) และรหัสเดิมผิดใน `password/update` (PA4)** · login สำเร็จล้างตัวนับ · ไม่เปลี่ยน status ของบัญชี · **ถ้าครบ limit ที่ PA4 (รหัสเดิมผิดครั้งที่ 5) → ตอบ `407303` และ session ปัจจุบันของผู้เล่นถูกตัด** (token เดิมได้ `407203`) |
| PA-42 | `transfer/*`: เรียกเกิน **20 ครั้ง/นาที ต่อ IP** (key `pl:op:ip:*` · lead Q8) → `429` |

### 3.6 เปลี่ยนรหัสผ่าน / ข้อมูลผู้เล่น

| ID | Rule |
|---|---|
| PA-50 | ผู้เล่นเปลี่ยนรหัสเองได้ (`pr/auth/password/update`) · ส่งรหัสเดิม + รหัสใหม่ · รหัสเดิมผิด `407205` (และนับตาม PA-41 — ครั้งที่ครบ limit ได้ `407303` + ตัด session) · รหัสใหม่ตามกติกา AUTH-36 (ฟังก์ชันใน `app/core` ตัวเดียวกับ BO) · ต้องไม่ซ้ำรหัสเดิม · session ปัจจุบันใช้ต่อ (1 session ต่อคน ไม่มี session อื่นให้เตะ — lead Q10) |
| PA-51 | ผู้เล่นที่ไม่มีรหัส (`password_hash = ''`) เปลี่ยนรหัสไม่ได้ `407307` (ใช้ในเส้นนี้ได้) · ผู้เล่นจาก operator ที่ตั้งรหัสไว้ (TR) เปลี่ยนได้ |
| PA-53 | BO reset รหัสผู้เล่น · operator reset / เปลี่ยนรหัสให้ผู้เล่น **ยังไม่ทำ** |
| PA-54 | `me/get` คืนโปรไฟล์ + สถานะ: `username` · `name` · `currency` · `status` · `can_bet` · **ไม่มี `credit`** (ใช้ `balance/get`) · ยังไม่มี win limit / ยอดที่เหลือวันนี้ (→ spec Bet) |
| PA-56 | **`balance/get` (PA6)**: คืน `{currency, credit}` · อ่านจาก `user_members.credit` ตรง **ไม่ cache** (กฎข้อ 14) · SUSPENDED เรียกได้ · ใช้เป็น ping กันหลุด (PA-21) · Seamless (รอบหน้า) เปลี่ยนเฉพาะเส้นนี้ไปเรียก GetBalance โดย contract เดิม |
| PA-55 | `games/list` ต้อง login (GQ9) · ใช้ `GamesListService` ของ asset ซ้ำ (`app/service/asset/catalog_service.go` — AST-51 ห้ามเขียน query คู่ขนาน) · ส่ง `game_code` · **`name_th` / `name_en`** · `open` · `game_cover` · **ไม่ส่งค่า rate** · ชื่อเกมเพิ่มใน **catalog ฝั่ง backend** (catalog ของ asset ตอนนี้มีแค่ `game_code` + `game_cover`) · เรียงตาม catalog คงที่ในโค้ด · `open` ดูจาก agent + สายบนเท่านั้น (Member ไม่มีสวิตช์เกมรายคน) · เกมปิดแสดงเป็นเกมปิด ไม่ซ่อน · ไม่มีปก = `game_cover = ""` |

### 3.7 เกมปิด

| ID | Rule |
|---|---|
| PA-60 | เกมปิดเช็คตอนเดิมพัน (Bet) และตอน launch (นอก scope) · lobby แสดงเป็นเกมปิด |

## 4. สิ่งที่พบในโค้ดเก่า และการตัดสินใจ

| # | โค้ดเก่าทำอะไร | ถูก/ผิด | การตัดสินใจ |
|---|---|---|---|
| 1 | `/provider/launch` สร้างผู้เล่นอย่างเดียว ไม่ให้ token ไม่มีรหัส → ผู้เล่นที่มาจาก launch login ไม่ได้ | ผิด | ไม่ลอก — `transfer/login` ให้ token + link เลย (PA-07) |
| 2 | askmelotto-api ยังไม่เคยต่อ minigame | — | ใช้แบบ transferwallet `loginRequest` / sportbook `redirectLogin`: `agent_username` + `key` + `player_username` → link (PQ4) · ชื่อ field เป็น `snake_case` เหมือน API ทั้งระบบ (lead Q9) |
| 3 | `user_members.password_hash NOT NULL` · `username VARCHAR(32)` | — | ผู้เล่น operator เก็บ `''` (หรือ hash ถ้า operator ส่ง `password`) · ขยาย username เป็น 65 (PA-06 / PA-07) |
| 4 | BO auth ใช้ JWT `typ` + Redis session 1 ต่อบัญชี + rate limit + สถานะสายบน | ใช้ซ้ำได้ | ใช้โครงเดียวกัน แยก key `pl:*` และ `typ = "pl"` (PA-20 – PA-41) |
| 5 | ตั๋วใช้ครั้งเดียว 60 วินาที + `launch/exchange` (ร่างเดิม) | ยกเลิก | link ใส่ token ใน URL ตรงๆ (LQ-c) — มาตรการลดความเสี่ยงอยู่หัวข้อ 5 |

## 5. Endpoints

| ID | Method | Path | ผู้เรียก / สิทธิ์ | หมายเหตุ |
|---|---|---|---|---|
| PA1 | POST | `/api/v1/player/pb/login` | ผู้เล่น · public + rate limit | login ตรง |
| PA3 | POST | `/api/v1/player/pr/auth/logout` | ผู้เล่น · `PlayerAuthenticated` | idempotent |
| PA4 | POST | `/api/v1/player/pr/auth/password/update` | ผู้เล่น · `PlayerAuthenticated` | |
| PA5 | POST | `/api/v1/player/pr/me/get` | ผู้เล่น · `PlayerAuthenticated` (SUSPENDED ผ่าน) | |
| PA6 | POST | `/api/v1/player/pr/balance/get` | ผู้เล่น · `PlayerAuthenticated` (SUSPENDED ผ่าน) | ยอด + ping กันหลุด |
| PA7 | POST | `/api/v1/player/pr/games/list` | ผู้เล่น · `PlayerAuthenticated` (SUSPENDED ผ่าน) | |
| L2 | POST | `/api/v1/transfer/login` | operator Transfer · `agent_username` + `key` + IP whitelist | สร้างผู้เล่นให้ถ้ายังไม่มี |
| TR | POST | `/api/v1/transfer/register` | operator Transfer · เหมือน L2 | |

สิทธิ์ของแต่ละ route ต้องเขียนไว้บรรทัดเดียวกับ route ใน `routes.go` (กฎข้อ 28) · ทุกเส้นตอบผ่าน `pkg/response` · request ผ่าน `utils.ParseBodyNoNull` + `Validate()` (กฎข้อ 19)

**error ของ middleware ผู้เล่น (ทุกเส้น `pr/*` อ้างถึงที่นี่ ไม่เขียนซ้ำ):** `407202` (token) · `407203` (session สิ้นสุด) · `407204` (ถูก login จากที่อื่น) · `407301` (ล็อก) · `407302` (สายบนล็อก)

### PA1 · POST /api/v1/player/pb/login

Request:
```json
{ "username": "mem01", "password": "••••••" }
```
Response `data`:
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
| `member.username` | string | username ที่ใช้ login (Member จาก BO = ชื่อเดิม · ผู้เล่น operator = `agent@player`) | `mem01` |
| `member.name` | string | ชื่อที่แสดง | `mem01` |
| `member.currency` | string | สกุลเงินของผู้เล่น (ISO) | `THB` |
| `member.status` | string `ACTIVE` / `SUSPENDED` | สถานะที่ใช้งานจริง (หัวข้อ 3.4) · `LOCKED` ไม่มีในนี้เพราะ login ไม่ผ่าน | `ACTIVE` |
| `member.can_bet` | bool | เดิมพันได้ไหม (PA-35) · `false` เมื่อ `SUSPENDED` | `true` |

`member` ไม่มี `credit` / `id`
Error codes: `422` · `407201` (รวมผู้เล่นไม่มีรหัส) · `407301` · `407302` · `407303` · `429` · `500`

### PA3 · POST /api/v1/player/pr/auth/logout

Header `Authorization: Bearer <token>` · Response: `{"code":200,"msg":"สำเร็จ"}` (ไม่มี `data`) · Error codes: `407202`

### PA4 · POST /api/v1/player/pr/auth/password/update

Request: `{ "old_password": "••••••", "new_password": "••••••" }` · Response: `{"code":200,"msg":"สำเร็จ"}`
Error codes: `422` (รหัสใหม่ไม่ผ่าน AUTH-36 / ซ้ำรหัสเดิม — msg บอก field) · `407205` (รหัสเดิมผิด — นับ PA-41) · `407303` (ผิดครบ 5 ครั้ง / 15 นาที) · `407307` (ไม่มีรหัส) · error ของ middleware

### PA5 · POST /api/v1/player/pr/me/get

Request: `{}` · Response `data`:
```json
{ "username": "agent01@p001", "name": "p001", "currency": "THB", "status": "SUSPENDED", "can_bet": false }
```

| Field | Type | ความหมาย | ตัวอย่าง |
|---|---|---|---|
| `username` | string | username ผู้เล่น (ผู้เล่นจาก operator = `agent@player`) | `agent01@p001` |
| `name` | string | ชื่อที่แสดง | `p001` |
| `currency` | string | สกุลเงิน (ISO) | `THB` |
| `status` | string `ACTIVE` / `SUSPENDED` | สถานะที่ใช้งานจริง (หัวข้อ 3.4) | `SUSPENDED` |
| `can_bet` | bool | เดิมพันได้ไหม (PA-35) | `false` |

**ไม่มี `credit`** — ใช้ PA6 · ยังไม่ใส่ win limit / ยอดที่เหลือวันนี้ (→ spec Bet)
Error codes: error ของ middleware

### PA6 · POST /api/v1/player/pr/balance/get

Request: `{}` · Response `data`:
```json
{ "currency": "THB", "credit": 1250.5 }
```

| Field | Type | ความหมาย | ตัวอย่าง |
|---|---|---|---|
| `currency` | string | สกุลเงินของผู้เล่น (ISO) | `THB` |
| `credit` | number | ยอดเงินคงเหลือ หน่วยสกุล ส่งค่าที่เก็บตามจริง ไม่ปัดเพิ่ม (กฎข้อ 9) · หน้าบ้านปัด 3 ตำแหน่งตอนแสดงเอง · อ่าน `user_members.credit` ตรง ไม่ cache | `1250.5` |

- SUSPENDED เรียกได้ · **หน้าเกมเรียกทุก 5–10 นาทีเป็น ping กันหลุด idle** (PA-21) — ไม่มีเส้น ping แยก
- Seamless (รอบหน้า) เปลี่ยนเฉพาะเส้นนี้ไปเรียก GetBalance โดย contract เดิม (PA-56)

Error codes: error ของ middleware

### PA7 · POST /api/v1/player/pr/games/list

Request: `{}` · Response `data`:
```json
{
  "items": [
    { "game_code": "scratch_card", "name_th": "สแครชการ์ด", "name_en": "Scratch Card", "open": true,
      "game_cover": "https://cdn.example.com/assets/scratch_card/game_cover/3f9a….png" },
    { "game_code": "coin_toss", "name_th": "โยนเหรียญ", "name_en": "Coin Toss", "open": false, "game_cover": "" }
  ]
}
```

| Field | Type | ความหมาย | ตัวอย่าง |
|---|---|---|---|
| `items` | array | เกมทั้งหมดของ lobby เรียงตาม catalog คงที่ในโค้ด (เกมปิดไม่ซ่อน — PA-55) · ไม่มีเกม = `[]` | |
| `items[].game_code` | string | รหัสเกม ตรงกับ `game_code` ของ game_rate | `scratch_card` |
| `items[].name_th` | string | ชื่อเกมภาษาไทย (จาก catalog ฝั่ง backend — เพิ่มใหม่) | `สแครชการ์ด` |
| `items[].name_en` | string | ชื่อเกมภาษาอังกฤษ | `Scratch Card` |
| `items[].open` | bool | เปิดให้เล่นไหม (ตาม agent + สายบน · Member ไม่มีสวิตช์รายคน) · `false` = แสดงเป็นเกมปิด เดิมพันไม่ได้ (PA-60) | `true` |
| `items[].game_cover` | string (URL) | **รูปปกเกม** = ช่อง `game_cover` ไล่สายจากผู้สร้างตรงของผู้เล่นขึ้นไปทีละช่อง (ตัวเอง → ชั้นบนที่ใกล้สุด → GLOBAL — asset AST-25 / AST-51 / AST-57) · URL = `S3_CDN_BASE_URL` + storage key (AST-49) · ไม่มีปก / อ่านไม่สำเร็จ = `""` ตอบสำเร็จ ไม่ใช่ error (AST-46 / ACC-32) + log warn | `https://cdn.example.com/…/game_cover/3f9a….png` |

- **ไม่ส่งค่า rate** · รูปช่องอื่นเป็นของ bundle (`/player/asset/bundle` — GN23 · asset AST-48 · Schneider ทำหลัง middleware ผู้เล่น merge)
- ปก + `open` อ่านจาก service ตัวเดียวกับ `games/list` ของ BO / `used/list` (asset AST-51) · เป้าหมายมาจาก session ผู้เล่นเท่านั้น (PA-01 / asset AST-52) · ทุกเกมใน query เดียว ไม่ N+1 (กฎข้อ 23)
- รูปปกขนาดตาม catalog ของช่อง `game_cover` (เช่น scratch_card 608×1080) — ขนาดตัดสินที่ asset

Error codes: error ของ middleware

### L2 · POST /api/v1/transfer/login

Request:
```json
{ "agent_username": "agent01", "key": "<64 hex>", "player_username": "p001", "currency": "THB", "language": "th" }
```

| Field | บังคับ | ความหมาย |
|---|---|---|
| `agent_username` | ✅ | username ของผู้สร้าง (เจ้าของ Key) |
| `key` | ✅ | Key ของผู้สร้าง — mask ใน log ทุกจุด (PA-12) |
| `player_username` | ✅ | ชื่อผู้เล่น `a-z0-9` 3–32 (PA-05) |
| `currency` | ไม่ | ISO · ดู PA-08A (สกุลเดียว = ใช้สกุลนั้น · หลายสกุลต้องส่ง · ผู้เล่นเดิมต้องตรง) |
| `language` | ไม่ | `th` / `en` · default `th` → ต่อ `&lang=` ท้าย link |

Response `data`:
```json
{ "token": "<jwt>", "expires_at": "2026-10-15T16:00:00+07:00", "link": "https://<lobby>/?token=<jwt>&lang=th" }
```

| Field | Type | ความหมาย | ตัวอย่าง |
|---|---|---|---|
| `token` | string | JWT ของผู้เล่น (เหมือน PA1) · operator เก็บไว้เรียกต่อเองได้ แต่ผู้เล่นใช้ผ่าน `link` | `eyJhbGciOi…` |
| `expires_at` | string (RFC 3339 +07:00) | หมดอายุแบบ absolute (login + 4 ชม.) | `2026-10-15T16:00:00+07:00` |
| `link` | string (URL) | link เข้า lobby ที่ต่อ `?token=<token>&lang=<th/en>` (LQ-c) · operator redirect ผู้เล่นไป link นี้ตรงๆ · ฐาน URL = env `PLAYER_LOBBY_URL` (+ `.env.example` PR เดียวกัน · lead Q12) · query ชื่อ `token` · **ต้องเป็น absolute http(s) URL ที่มี host — ตรวจตอน `configs.Load` (ผิด = boot ไม่ขึ้น) และตรวจซ้ำใน `transfer/login` ก่อนสร้างผู้เล่น / ออก session (ค่าผิดต้องไม่ทำให้ผู้เล่นถูกเตะโดยไม่ได้ link — N5)** | `https://<lobby>/?token=eyJ…&lang=th` |
| (ไม่มี) `created` | — | **ไม่บอกว่าสร้างผู้เล่นใหม่หรือเดิม** — operator ไม่ต้องรู้ (PA-07) | |

- ผู้เล่นมีอยู่แล้วแต่ `currency` ที่ส่งมาไม่ตรง → `422` "สกุลเงินไม่ตรงกับผู้เล่น" (PA-08A)
- agent หรือสายบน LOCKED → `407302` ทุกกรณี · agent SUSPENDED → ผู้เล่นเดิม login ได้ (`can_bet = false`) · ผู้เล่นใหม่ `407308` (PA-13)
- login ผู้เล่นใหม่พร้อมกัน → ไม่ error (PA-08B) · ผู้เล่นที่สร้างอัตโนมัติไม่มีรหัส (PA-07)
- มาตรการลดความเสี่ยง token อยู่ใน URL (LQ-c): หน้าเกมลบ token ออกจาก URL ทันทีหลังอ่าน (`history.replaceState`) · หน้าเกมตั้ง `Referrer-Policy: no-referrer` · **ห้าม log query string ของ URL หน้าเกมฝั่งเรา** (รวม access log ของ proxy ที่เราคุมได้) · token อายุตาม session และ login ใหม่เตะอันเก่า (PA-21 / PA-22)

Error codes: `422` · `407206` · `407305` · `407301` · `407302` · `407306` · `407308` · `429` · `500` · `503` (ไม่ได้ตั้ง `PLAYER_LOBBY_URL`)

### TR · POST /api/v1/transfer/register

Request:
```json
{ "agent_username": "agent01", "key": "<64 hex>", "player_username": "p001", "currency": "THB", "password": "••••••" }
```
- `currency` / `password` ไม่บังคับ · `password` = AUTH-36 (PA-08) ไม่ผ่าน `422` · ไม่ส่ง = ผู้เล่นเข้าทาง operator อย่างเดียว

Response `data`: `{ "username": "agent01@p001", "currency": "THB" }`

| Field | Type | ความหมาย | ตัวอย่าง |
|---|---|---|---|
| `username` | string | username ที่ระบบเก็บ = `{agent_username}@{player_username}` (ตัวเล็กทั้งหมด) — ผู้เล่นที่ตั้ง `password` ใช้ชื่อนี้ login ที่ PA1 | `agent01@p001` |
| `currency` | string | สกุลเงินที่ระบบกำหนดให้ผู้เล่น (PA-08A) | `THB` |
| token / link / รหัส | — | **ไม่ออก / ไม่คืน** — ได้ token ต้องเรียก `transfer/login` | |

- register ซ้ำ = `407401` · operator ถือเป็น "สร้างไว้แล้ว" (เขียนในเอกสาร frontend)
- agent SUSPENDED สร้างผู้เล่นใหม่ไม่ได้ `407308` (PA-13) · agent หรือสายบน LOCKED `407302`

Error codes: `422` · `407206` · `407305` · `407302` · `407306` · `407308` · `407401` · `429`

### Middleware ผู้เล่น `PlayerAuthenticated` (ใช้กับทุก route ใต้ `/player/pr`)

อยู่ที่ `app/internals/frontend/middleware` · ใช้ `BearerToken` จาก `pkg/middleware` (PA-20A)

1. อ่าน Bearer token → ตรวจลายเซ็น / หมดอายุ / `typ = "pl"` → อ่าน `sid`, `member_id`
2. หา session ใน Redis ด้วย `sid` — ต้องมีและเป็นของ `member_id` เดียวกัน · ไม่ใช่ session ปัจจุบันของ Member (ถูกเตะ) = `407204`
3. โหลด Member ด้วย primary key (`id, username, agent_id, status, cnf`) + สถานะสายบนจาก `cnf.parent` ด้วย query แยก 2 ครั้ง (Member 1 + สถานะสายบนทั้งสาย 1 query — ไม่ใช่ N+1 · lead N3 · รวมเป็น JOIN `cnf->'parent'` ถ้า k6 ชี้ว่าเป็นคอขวด) → คำนวณสถานะที่ใช้งานจริง (PA-30 / PA-32)
4. ต่ออายุ idle (PA-21) แล้ว set `c.Locals("player", Player{...})` (struct เดียว มี `can_bet`)

### Middleware operator `OperatorAuthenticated` (ใช้กับ `transfer/*`)

อยู่ที่ `app/externals/transfer/middleware` · อ่าน body → PA-10 / PA-11 / PA-13 / PA-14 → set `c.Locals("operator", Operator{agent_id, username})` · ห้าม log body ดิบ (มี `key` / `password`)

## 6. Schema

**migration** (`make migration name=player_auth_username_length`):

```sql
-- +goose Up
ALTER TABLE user_members ALTER COLUMN username TYPE VARCHAR(65);
ALTER TABLE user_members DROP CONSTRAINT ck_user_members_username;
ALTER TABLE user_members ADD CONSTRAINT ck_user_members_username
    CHECK (username = lower(username) AND username ~ '^[a-z0-9]{3,32}(@[a-z0-9]{3,32})?$');
-- +goose Down
-- ย้อนกลับไม่ได้ถ้ามีผู้เล่นแบบ agent@player แล้ว — ลบแถวผู้เล่น operator ก่อน
ALTER TABLE user_members DROP CONSTRAINT ck_user_members_username;
ALTER TABLE user_members ADD CONSTRAINT ck_user_members_username CHECK (username = lower(username) AND username ~ '^[a-z0-9]{3,32}$');
ALTER TABLE user_members ALTER COLUMN username TYPE VARCHAR(32);
```
- **แก้เพิ่มตอน implement (2026-10-15 · lead อนุมัติแล้วที่ 0ce5c78):** `user_members` มี CHECK `ck_user_members_username` รับแค่ `^[a-z0-9]{3,32}$` — ผู้เล่น `agent@player` จะ insert ไม่ได้ถ้าไม่แก้ CHECK ด้วย (เจอตอนรัน integration test) · แถวเดิมทุกแถว (ไม่มี `@`) ผ่าน CHECK ใหม่เหมือนเดิม
- `uq_user_members_username` (unique เดิม) รองรับ PA-06 / PA-08 / PA-08B · ไม่ต้องมี index เพิ่ม · ตารางเล็ก ไม่ต้องแยก 3 ขั้น (docs/DEPLOYMENT.md)
- ไม่มีตารางใหม่ (ใช้ `user_members` · `user_member_game_settings` · `api_credentials` · `api_allowed_ips` · `user_agents`)
- `core/player_auth` มี validator `player_username` (3–32 `a-z0-9`) และประกอบ username ความยาวไม่เกิน 65 (รวม `@`) แยกจาก MGMT-05
- ชื่อเกม `name_th` / `name_en`: เพิ่มใน catalog เกมฝั่ง backend (`app/core/game` — ชื่อเกมทุกเกมต้องมีทั้ง 2 ภาษา · ครอบด้วย table test) ไม่เพิ่มตาราง

Redis:

| Key | ค่า | TTL |
|---|---|---|
| `pl:sess:{sid}` | JSON `{member_id, ip, created_at}` | idle 30 นาที (ต่อทุก request ไม่เกิน absolute) |
| `pl:sess:member:{member_id}` | `sid` ปัจจุบัน | absolute 4 ชม. |
| `pl:login:fail:{username}` | จำนวนครั้งที่ผิด (PA1 + PA4) | 15 นาที |
| `pl:login:ip:{ip}` | จำนวนครั้งที่เรียก PA1 | 1 นาที |
| `pl:op:ip:{ip}` | จำนวนครั้งที่เรียก `transfer/*` | 1 นาที |

## 7. Test cases

| Rule | Input | Expected |
|---|---|---|
| PA-04 | login `"  Mem01 "` (บัญชี `mem01`) | สำเร็จ |
| PA-03 | login ตรงด้วยผู้เล่น operator ที่ไม่มีรหัส | `407201` (ข้อความเดียวกับรหัสผิด · ไม่ใช่ `407307`) |
| PA-03 / PA-08 | `register` ส่ง `password` แล้ว login ตรง `agent01@p001` | สำเร็จ · token |
| PA-04 | ไม่พบ username · รหัสผิด | `407201` ทั้งคู่ |
| PA-05 | `player_username` `"AB"` · `"abc!"` · 33 ตัว · `"ABC"` | `422` · `422` · `422` · สำเร็จ เก็บ `agent@abc` |
| PA-07 / PA-09 | `transfer/login` ผู้เล่นใหม่ | สร้าง + token + link · `name = player_username` · `credit = 0` · `payment_type = CASH` · `pt` = ค่าเต็มของ agent · `commission = 0` · `win_limit_per_day` = `DefaultWinLimitPerDay` 10,000,000 บาท (เท่ากับที่ `members/create` ให้เมื่อไม่ส่งค่า · ส่งชัดเจนตอน insert ไม่ใช่ default ของ DB · ไม่แปลงสกุล) |
| PA-07 | `transfer/login` ผู้เล่นเดิม 2 ครั้ง | ไม่สร้างซ้ำ · ครั้งที่ 2 เตะ session แรก |
| PA-07 | `language` = `en` · ไม่ส่ง · `fr` | link ลงท้าย `&lang=en` · `&lang=th` · `422` |
| PA-08 | `register` ซ้ำ | `407401` |
| PA-08 | `register` `password` ไม่ผ่าน AUTH-36 | `422` (ไม่พบรหัสใน log / response) |
| PA-08A | agent สกุลเดียวไม่ส่ง `currency` · agent หลายสกุลไม่ส่ง · ส่งสกุลนอกสกุล agent | ใช้สกุลนั้น · `422` · `422` |
| PA-08A | ผู้เล่นเดิม สกุล THB แต่ส่ง `currency` USD | `422` "สกุลเงินไม่ตรงกับผู้เล่น" |
| PA-08B | ยิง `transfer/login` ผู้เล่นใหม่ชื่อเดียวกันพร้อมกัน N คำขอ | ทุกคำขอสำเร็จ · มีแถวเดียว · ไม่มี 500 |
| PA-08B | ยิง `register` ชื่อเดียวกันพร้อมกัน | 1 สำเร็จ · ที่เหลือ `407401` · ไม่มี 500 |
| PA-10/11 | key ผิด · agent ไม่มี · IP นอก whitelist · whitelist ว่าง | `407206` · `407206` · `407305` · `407305` |
| PA-12 | ยิง `transfer/login` / `register` ด้วย key จริง · key ผิด · agent ไม่มี · password ผู้เล่น · password ไม่ผ่านกติกา · IP นอก whitelist แล้วเก็บ log ทุกทาง (zap ระดับ debug · SQL ของ gorm ระดับ info พร้อมค่า · request log ของ fiber) | ไม่พบ key / password ในทุกบรรทัด (test `TestOperatorSecretsNotLogged` — ตรวจแล้วว่า log key แล้ว test ตก) |
| PA-09B | `transfer/login` ผู้เล่นใหม่ · `register` (มี / ไม่มี password) · login ซ้ำ · register ซ้ำ · สร้างพร้อมกัน | `account_change_logs` CREATE 1 แถวต่อผู้เล่น · ผู้ทำ = agent เจ้าของ key · ip ของ operator · ไม่มีรหัสผ่าน / hash ใน `new_value` (`TestTransferCreateWritesChangeLog`) |
| PA-13 | agent LOCKED · สายบนของ agent LOCKED · เรียก `login` / `register` | `407302` ทุกเส้น ทุกกรณี |
| PA-13 | agent SUSPENDED: ผู้เล่นเดิม login · ผู้เล่นใหม่ login · register | สำเร็จ `can_bet = false` · `407308` · `407308` |
| PA-14 | เรียกด้วย `COMPANY_TRANSFER` / `SHARE_B2B` · `SHARE_B2C` · `AGENT` | `407306` · สำเร็จ · สำเร็จ |
| PA-22 | login เครื่อง A แล้วเครื่อง B → ใช้ token A | `407204` |
| PA-21 | ไม่มี request 31 นาที · ใช้ต่อเนื่องครบ 4 ชม. · `balance/get` ทุก 10 นาที ต่อเนื่อง | `407203` · `407203` · session ไม่หลุดก่อน 4 ชม. |
| PA-20 | token BO ยิง `/player/pr` · token ผู้เล่น (`typ = "pl"`) ยิง `/bo/pr` | `407202` · `401202` |
| PA-26 | logout ซ้ำ 2 ครั้ง · logout แล้วใช้ token เดิม | สำเร็จทั้งคู่ · `407203` |
| PA-30/32 | Member ACTIVE สายบนถูก LOCKED ระหว่างใช้งาน | request ถัดไป `407302` และ session ถูกลบ |
| PA-33 | Member SUSPENDED เรียก `me/get` · `balance/get` · `games/list` | สำเร็จ · `can_bet = false` |
| PA-34 | `members/status/update` → LOCKED | session ผู้เล่นถูกลบหลัง commit · token เดิม `407203` · Redis ล่ม: status เปลี่ยนสำเร็จ + log warn |
| PA-35 | `win_limit_per_day = 0` · ACTIVE | `can_bet = false` |
| PA-41 | ผิด 5 ครั้งแล้วใส่รหัสถูก · ผิด 4 ครั้ง ถูก แล้วผิดอีก 4 | `407303` · ยัง login ได้ |
| PA-41 | `password/update` รหัสเดิมผิด 5 ครั้ง / 15 นาที · ผสม login ผิด + password/update ผิด รวม 5 | `407303` 15 นาที ทั้งสอง |
| PA-40 | IP เดียวยิง `player/pb/login` 21 ครั้ง/นาที | ครั้งที่ 21 `429` |
| PA-42 | IP เดียวยิง `transfer/*` 21 ครั้ง/นาที | ครั้งที่ 21 `429` |
| PA-50 | รหัสเดิมผิด · รหัสใหม่ผิดกติกา · รหัสใหม่เหมือนเดิม | `407205` · `422` · `422` |
| PA-50 | เปลี่ยนรหัสสำเร็จ แล้วใช้ token เดิม | ใช้ต่อได้ (session ปัจจุบันอยู่) |
| PA-51 | ผู้เล่น operator ที่ไม่มีรหัสเปลี่ยนรหัส · ที่ตั้งรหัสไว้ (TR) | `407307` · สำเร็จ |
| PA-54 | `me/get` | มี `username` `name` `currency` `status` `can_bet` · **ไม่มี `credit`** |
| PA-56 | `balance/get` หลังฝาก (DW) · SUSPENDED | ยอดล่าสุดทันที (ไม่ cache) · สำเร็จ |
| PA-55 | `games/list` | ทุกเกมมี `name_th` / `name_en` ไม่ว่าง · เรียงตาม catalog · เกมปิด `open = false` ไม่ถูกซ่อน · ไม่มี rate · ไม่มีปก → `game_cover = ""` |

ต้องมี table test ของ `core/player_auth` (validator `player_username`, ประกอบ username, สถานะที่ใช้งานจริง, `can_bet`) และของ catalog ชื่อเกม · integration test ทุกเส้น · rule id อ้างใน test ผ่าน comment `// rule: PA-xx`

## 8. Contract changes (แจ้ง frontend / operator)

- ฝั่งผู้เล่น (หน้าเกม): endpoint ใหม่ 6 เส้น (PA1 / PA3 / PA4 / PA5 / PA6 / PA7) · token ใส่ `Authorization: Bearer` · หน้าเกมอ่าน token จาก URL (L2) แล้วลบออกจาก URL ทันที · ใช้ `balance/get` รีเฟรชยอดทุก 5–10 นาทีเป็น ping กันหลุด · `407202` / `407203` / `407204` → พาไปหน้าเริ่มต้น (ผู้เล่นจาก operator กลับไปเว็บ operator)
- `me/get` ไม่มี `credit` (ย้ายไป `balance/get`) · `games/list` มี `name_th` / `name_en` / `game_cover`
- ฝั่ง operator: `transfer/login` และ `transfer/register` (ใหม่) · field เป็น `snake_case` (`agent_username` · `player_username` · `key` · `currency` · `language` · `password`) · ต้องตั้ง IP whitelist ที่เมนู API ก่อน (ACC-07) · `register` ซ้ำ = `407401` ถือว่า "สร้างไว้แล้ว"
- `members/status/update` (M4): LOCKED ลบ session ผู้เล่นหลัง commit — response ไม่เปลี่ยน
- `user_members.username` ยาวได้ถึง 65 ตัว — หน้า list / detail ของ Member (BO) ต้องรองรับ username ที่มี `@`
- ย้าย `BearerToken` ไป `pkg/middleware` — ไม่กระทบ API (refactor ภายใน)
- error module `07` = player_auth ลง `docs/ERROR_CODES.md` (ตารางด้านล่าง)

### error code ที่จะจอง (module `07`)

| Code | HTTP | ความหมาย |
|---|---|---|
| 407201 | 401 | username หรือรหัสผ่านไม่ถูกต้อง (รวมผู้เล่นไม่มีรหัส) |
| 407202 | 401 | ไม่พบ token / token ไม่ถูกต้องหรือหมดอายุ |
| 407203 | 401 | session สิ้นสุดแล้ว (logout / ไม่ได้ใช้งานนาน) |
| 407204 | 401 | บัญชีนี้ถูกเข้าสู่ระบบจากที่อื่น |
| 407205 | 200 | รหัสผ่านเดิมไม่ถูกต้อง |
| 407206 | 401 | `agent_username` หรือ `key` ไม่ถูกต้อง |
| 407301 | 403 | บัญชีถูกล็อก |
| 407302 | 403 | สายบัญชีต้นสังกัดถูกล็อก (รวม agent เจ้าของ Key ถูกล็อก) |
| 407303 | 403 | ใส่รหัสผิดหลายครั้ง ระงับชั่วคราว (PA1 + PA4) |
| 407305 | 403 | IP ไม่ได้รับอนุญาต |
| 407306 | 403 | บัญชีนี้ใช้ทางเข้านี้ไม่ได้ |
| 407307 | 200 | บัญชีนี้ไม่มีรหัสผ่าน |
| 407308 | 403 | บัญชีผู้สร้างถูกระงับ สร้างผู้เล่นใหม่ไม่ได้ |
| 407401 | 200 | มีผู้เล่นนี้แล้ว |

(`407304` เว้นไว้ — เดิมพันไม่ได้เพราะ SUSPENDED เป็นของ module Bet จะจองตอนทำ spec นั้น)

## 9. การตัดสินของ lead (ปิดแล้ว) และข้อที่เหลือ

คำถามรอบ 2 (Q1–Q13 · lead 2026-10-15) ตัดสินครบแล้ว — ใส่เป็นกฎข้างบนทุกข้อ:

| Q | เรื่อง | ผล | กฎ |
|---|---|---|---|
| Q1 | ใครเรียก `transfer/*` ได้ | `SHARE_B2C` และ `AGENT` | PA-14 |
| Q2 | สกุลเงินผู้เล่นที่ระบบสร้าง | `currency` ไม่บังคับ · สกุลเดียวใช้สกุลนั้น · หลายสกุลไม่ส่ง `422` · ต้องอยู่ในสกุล agent | PA-08A |
| Q3–Q5 | ค่าตั้งต้น | `name = player_username` · `credit = 0` · `CASH` · `pt` = ค่าเต็ม agent · commission 0 · win limit = `DefaultWinLimitPerDay` (บาท ไม่แปลงสกุล — R4-2 + member-payment รอบ 2.1) | PA-09 |
| Q6 | `can_bet` | ACTIVE + `win_limit_per_day > 0` | PA-35 |
| Q7 | `games/list` | `GamesListService` ซ้ำ · `game_code` / `name_th` / `name_en` / `open` / `game_cover` · ไม่มี rate | PA-55 |
| Q8 | rate limit `transfer/*` | IP 20/นาที · whitelist ว่าง = ปฏิเสธ | PA-42 · PA-10 |
| Q9 | ชื่อ field operator | `snake_case` | หัวข้อ 5 |
| Q10 | เปลี่ยนรหัสเตะ session | session ปัจจุบันใช้ต่อ | PA-50 |
| Q11 | `agent@player` ชน `agent@sub` | ไม่ต้องกัน | PA-06 |
| Q12 | URL lobby | env `PLAYER_LOBBY_URL` · query `token` | L2 |
| Q13 | เช็คสายบนทุก request | ไม่ cache · 2 query (Member + สถานะสายบน ไม่ N+1) · วัดด้วย k6 | PA-30 |

รอบ 3 (ทีละ endpoint): เพิ่ม PA6 · เอา `credit` ออกจาก `me/get` · `games/list` เพิ่มชื่อเกม · `register` รับ `password` · L2 รับ `language` และ `currency` ไม่ตรง = `422` · ยิงพร้อมกันไม่ error · agent SUSPENDED สร้างผู้เล่นใหม่ไม่ได้ · LOCKED ปฏิเสธทุก `transfer/*` · PA1 ไม่มีรหัส = `407201` · PA4 นับ PA-41 · field `snake_case` · ย้าย `BearerToken`

**รอบ 4 (lead zerph 2026-10-15) — อนุมัติ spec:** R4-1 รับ `407308` (HTTP 403) "บัญชีผู้สร้างถูกระงับ สร้างผู้เล่นใหม่ไม่ได้" · R4-2 แก้ PA-09 `win_limit_per_day` ของผู้เล่นที่ระบบสร้างเป็นค่าตั้งต้นตาม MGMT-65 ไม่พึ่ง default ของ DB (แก้แล้ว) · member-payment รอบ 2.1 (2026-10-15) กำหนดว่า win limit เป็นบาทเสมอ ไม่ต้องแปลงสกุล — PA-09 ใช้ `DefaultWinLimitPerDay` ตัวเดียวกับ `members/create`

## 10. Checklist ก่อนเปิด PR (ตามลำดับ review → implement)

1. ~~lead approve spec~~ — อนุมัติแล้ว 2026-10-15 (รอบ 4 · รับ `407308`)
2. จอง code module `07` ใน `docs/ERROR_CODES.md` (ตารางหัวข้อ 8) + `pkg/apperr/player_auth_errors.go`
3. ย้าย `BearerToken` → `pkg/middleware` (refactor ก่อน — แยก commit)
4. migration `username VARCHAR(65)` · env ใหม่ใน `pkg/configs/config.go` + `.env.example` (`PLAYER_LOBBY_URL` · idle / absolute ของ session ผู้เล่น)
5. ชื่อเกม `name_th` / `name_en` ใน catalog `app/core/game` + table test
6. `app/core/player_auth` (pure + table test) → service → repository → controllers → routes (บรรทัดเดียวกับสิทธิ์)
7. แก้ `members/status/update` (member_management) + ปิด MGMT-30A ใน `docs/modules/member_management.md`
8. อัปเดต `docs/MIGRATION_STATUS.md` · `docs/frontend/player_auth.md` · `make check` + integration test (รวม test ยิงพร้อมกัน PA-08B)
