# Player Auth (ผู้เล่นเข้าระบบ · สาย Transfer) — ร่าง spec รอ lead review · 2026-10-15

จาก boiledegg · review ที่มาของงาน: [docs/reviews/player-auth.md](../reviews/player-auth.md) · spec: [docs/modules/player_auth.md](../modules/player_auth.md) (**DRAFT**)
branch `boiledegg/bo/player_auth` (แตกจาก dev @ `f483d43`) · ยังไม่มีโค้ด — ตามลำดับ: game_rate 3 ข้อ (M1 / G-S1 / G-S2) เสร็จแล้ว (`ba3d63f` อยู่บน dev) → ร่าง spec → lead review ทีละ endpoint → implement ผ่าน PR

## 1. เสร็จแล้ว

- ร่าง spec ครบ 7 เส้น (PA1 · PA3 · PA4 · PA5 · PA7 · L2 · TR) + middleware ผู้เล่น / operator + Redis `pl:*` + migration `user_members.username VARCHAR(65)` + error module `07` (จองเลขใน `docs/ERROR_CODES.md` · code จริงปิดตอนอนุมัติ)
- Response ทุกเส้นมีตาราง field (type · ความหมาย · ตัวอย่าง) · `games/list` มี `game_cover` ตาม asset AST-51

## 2. ยังไม่ทำ

- โค้ดทั้งหมด (รอ lead อนุมัติ spec) · `transfer/launch` · `seamless/*` · BO reset รหัส · เปลี่ยน username Member จาก BO (นอกขอบเขตตาม review)

## 3. คำถามที่รอ lead (review ไม่ได้ระบุ — ผู้ร่างไม่เดา · มีข้อเสนอให้เลือก / แก้)


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

## 4. ประวัติอัปเดต

- 2026-10-15 · ร่าง spec `player_auth.md` + ส่งคำถาม 13 ข้อให้ lead
