# บัญชี (`account`) — Dashboard · Profile · ข้อมูลรับรอง API — Spec

- สถานะ: **DRAFT** (แก้ตาม review module ② 2026-10-07: รูปแบบสิทธิ์ และยอดเงินจริง — รออนุมัติใหม่)
- อนุมัติโดย: lead (zerph) · วันที่: 2026-10-06 · ฉบับก่อนหน้าอนุมัติ 2026-10-05 และ 2026-10-06 (ก่อน review รอบ 2)
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
| Profile — ยอดเงิน | ยอดเงินแยกสกุล (`balances`) | ตาราง `agent_balances` ของ module ② (MGMT-15A) — module การชำระเงินใช้ต่อ |
| 1.3 | ประเภทบัญชี (หาเจ้าของ Key) · ระบบสิทธิ์ (สิทธิ์ดู / แก้ไขของ sub) | module ② User Management |
| Dashboard | bet, ผลได้เสียตาม PT, Commission | module เดิมพัน / Settle |

**ไม่อยู่ใน module นี้:** การเก็บและย้ายยอดเงิน (module การชำระเงิน — Profile แค่อ่านยอดมาแสดง ACC-19) ·
การใช้ Key / IP / ลิงก์ตอนลูกค้าเรียก API (module provider `app/externals/provider`)

## 2. คำศัพท์

| คำ | ความหมาย |
|---|---|
| ประเภทบัญชี (`user_type`) | role + ประเภทย่อยตาม Company Hierarchy เช่น Company Transfer, Share B2C, Agent |
| ค่าหุ้นส่วน | PT ที่ได้รับ (ผู้สร้างตั้งให้ลูกแต่ละคน) · PT ถือจาก Member ใต้ตัวเอง (ตั้งเอง) · Remain (เอาส่วนที่เหลือ) · Force (บังคับถือสู้) · Commission — ระบบเก็บต่อเกม แต่หน้าบ้านตั้ง**ชุดเดียว**แล้วใช้กับทุกเกม (module ②) |
| สิทธิ์ | รายการสิ่งที่บัญชีทำได้ (กำหนดใน module ② ครั้งเดียว) · บัญชีหลักได้ตามประเภท · sub ได้ตามที่ผู้สร้างให้ |
| เจ้าของ Key | Company Seamless 1 to 1 · Share Master · Share Reseller — บัญชีละ 1 Key (ACC-01) |
| Key | ค่าลับที่ลูกค้าใช้ระบุตัวและคำนวณ `sign` ทุกคำขอ (Seamless API Flow) |
| ลิงก์ตอบกลับ | URL ของระบบลูกค้าที่ระบบเราเรียก (GetMember, GetBalance, Bet, Settle, Cancel) |
| IP ที่อนุญาต | IP ของลูกค้าที่เรียก API ของเราได้ |

## 3. Business rules

### สถานะ (ใช้กับทุก module)

| ID | Rule |
|---|---|
| ACC-30 | **API ส่งสถานะ key เดียว `status`** = สถานะที่ใช้งานจริง (เข้มที่สุดของสถานะที่ตั้งกับบัญชีนี้เอง, ผู้สร้างกรณี sub และ upline ทั้งสาย — AUTH-53 · `LOCKED` > `SUSPENDED` > `ACTIVE`) · ทุก module ใช้ความหมายนี้ · DB ยังเก็บสถานะที่ตั้งกับบัญชีเองแยกไว้ (ใช้คำนวณ และเป็นค่าที่ผู้สร้างแก้) |
| | • บัญชีตัวเอง ACTIVE · หัวสาย LOCKED → `status = LOCKED` เข้าใช้งานไม่ได้ |
| | • บัญชีตัวเองถูกล็อก · หัวสาย ACTIVE → ลูก `status = LOCKED` เข้าใช้งานไม่ได้ · **หัวสายใช้งานได้ปกติ** (สถานะของลูกไม่ส่งผลขึ้นไปข้างบน) |
| | • Company ระงับ Share → Share และ Agent / Member ใต้ Share `status = SUSPENDED` · Company ยัง `ACTIVE` |
| ACC-31 | การเข้าใช้งานตัดสินจาก `status` (ACC-30): `ACTIVE` ใช้งานได้ปกติ · `SUSPENDED` เข้าได้**เฉพาะหน้า Profile และ Report** และดูได้อย่างเดียว · `LOCKED` เข้าใช้งานไม่ได้เลย (AUTH-27) — บังคับทุก module ที่ middleware กลาง (AUTH-54) |

### Dashboard (1.1)

| ID | Rule |
|---|---|
| ACC-20 | เปิดได้: บัญชีหลังบ้านที่มีสิทธิ์ `dashboard` = `view` ขึ้นไป (module ② MGMT-51) · `status` ต้องเป็น `ACTIVE` (ACC-31) |
| ACC-21 | ข้อมูลเฉพาะสายของตัวเอง (ตัวเอง + สายล่าง) · sub เห็นของผู้สร้าง (AUTH-26) · ไม่เห็นข้ามสาย ไม่เห็นชั้นบน (System Overview) |
| ACC-22 | ตัวกรอง: เกม (ทั้งหมด หรือเกมเดียว) · ปี (บังคับ) + เดือน (ไม่บังคับ — **ไม่ส่ง = ทั้งปี** ใช้กับทางลัด "ปีนี้") · ทางลัด ก่อนหน้า / เดือนนี้ / ปีนี้ / ถัดไป หน้าบ้านแปลงเป็น `month` / `year` เอง · สกุลเงิน 1 สกุลต่อครั้ง (บังคับ · ต้องเป็นสกุลที่บัญชีมีตาม module ② MGMT-10 ถึง 13) |
| ACC-23 | ข้อมูลเป็นยอดสรุปรายวัน อัปเดตวันละครั้ง ช่วง 00:15–01:00 น. (ไม่ใช่ real-time) · ช่วงเวลาที่ยังไม่สรุปไม่นับ |
| ACC-24 | การ์ด (ชนิดของค่าในวงเล็บ — **จำนวน** = จำนวนเต็ม · **เงิน** = จำนวนเงินในสกุลที่เลือก): สมาชิกที่กำลังใช้งาน (จำนวนคน) · สมาชิกที่สมัครใหม่ (จำนวนคน) · สมาชิกทั้งหมด (จำนวนคน) · ยอดเล่นทั้งหมด (จำนวนครั้ง) · ยอดเล่นที่ถูกรางวัล (จำนวนครั้ง) · ยอดเล่นจริง (เงิน) · ยอดเล่นทั้งหมดตาม PT (เงิน) · ยอดถูกรางวัลตาม PT (เงิน) · ยอดแพ้ / ชนะตาม PT (เงิน) · Commission ที่ได้ (เงิน) · Commission ที่จ่าย (เงิน) · รายได้ (เงิน) · ทุกการ์ด (ยกเว้นสมาชิกที่กำลังใช้งาน) แสดง % เปลี่ยนแปลงเทียบช่วงก่อนหน้าที่ยาวเท่ากัน |
| ACC-25 | "ตาม PT" = ยอดคูณ PT ที่**คนที่ login ถือ** (ค่า ณ ตอน bet) · **รายได้** = ยอดแพ้ / ชนะตาม PT + Commission ที่ได้ − Commission ที่จ่าย (สูตร Commission ตามเอกสาร PT Commission ของ lead) |
| ACC-26 | ตาราง Top 10 สมาชิกในสาย: ยอดเล่นมากสุด · ยอดถูกมากสุด · ยอดเสียมากสุด (ลำดับ, username, จำนวนเงิน) |
| ACC-27 | ตัวเลขตาม ACC-18: เงินและ % ใน JSON เป็น number (เงินทศนิยม 2 ตำแหน่ง · % ทศนิยม 2 ตำแหน่ง) ภายในเป็นจำนวนเต็ม · ช่วงก่อนหน้าเป็น 0 → `change_pct = null` |

### Profile (1.2)

| ID | Rule |
|---|---|
| ACC-11 | ทุกบัญชีหลังบ้านเรียกได้ (SUPERADMIN, ADMIN, Company, Share, Agent และ sub) · ต้องผ่านด่านหลัง login (`PassedGates()`) · เปิดได้แม้ `status = SUSPENDED` (ACC-31) · แสดงข้อมูลของ**ตัวเอง** |
| ACC-12 | ข้อมูลที่แสดง: username · role · **ประเภทบัญชี** · `status` (key เดียว — ACC-30) · เป็น sub ไหม · username ของผู้สร้าง (เฉพาะ sub) · ตั้ง passcode แล้วหรือยัง · login ล่าสุด (เวลา, IP) · วันที่สร้างบัญชี · **สกุลเงิน** · **ยอดเงินแยกสกุล** (ACC-19) · **ค่าหุ้นส่วน** (ACC-16) · **สิทธิ์** (object `{ menu: off | view | edit }` — module ② MGMT-50) |
| ACC-15 | sub: role, ประเภทบัญชี, สกุลเงิน, ยอดเงิน และค่าหุ้นส่วน = ของผู้สร้าง (AUTH-25) · สิทธิ์ = ที่ผู้สร้างให้ sub นั้น |
| ACC-16 | ค่าหุ้นส่วนส่ง**ชุดเดียวต่อกลุ่ม PT** (รูปแบบเดียวกับตอนสร้าง / แก้ใน module ② MGMT-16): `pt` → กลุ่ม (ตอนนี้ `game`) → `pt_from_parent` (ได้รับ — ผู้สร้างตั้งให้บัญชีนี้) · `pt` (ถือจาก Member ใต้ตัวเองตรงๆ — บัญชีนี้ตั้งเอง ผ่าน module ② `PUT /agents/me/pt`) · `force` · `remain_quota` · `commission_percent` · `status` (เปิด / ปิดทั้งกลุ่ม) · `status_game` → รหัสเกม → bool (เปิด / ปิดทีละเกม · แยกจาก `pt`) · ผู้สร้างตั้งค่าที่ให้ลูกแต่ละคนแยกกัน (module ② MGMT-22) · ระบบเก็บแยกต่อเกม แต่ค่าในกลุ่มเท่ากันทุกเกม API จึงส่งชุดเดียว · SUPERADMIN = ค่าตั้งของตัวเอง (`pt_from_parent` = 100 — module ② MGMT-22) · ADMIN ไม่มีค่าหุ้นส่วน (`pt = {}`, `status_game = {}`) |
| ACC-19 | ยอดเงินแยกสกุล `balances`: 1 รายการต่อสกุลที่บัญชีมี (`currencies`) · `amount` เป็น JSON number ทศนิยม 2 ตำแหน่ง (ACC-18) · อ่านจาก `agent_balances` (module ② MGMT-15A) · สกุลที่ยังไม่มียอดส่ง `0` · บัญชีฝั่ง Seamless ไม่มียอดเงิน ส่ง `0` (module ② MGMT-15) · cache ห้ามเป็นที่มาของยอด (กฎข้อ 14) |
| ACC-17 | Profile มีไว้ให้หน้าบ้านแสดงผล / เช็คเบื้องต้นในฟอร์ม (เช่น สร้างบัญชีชั้นถัดไปที่ PT เกินของตัวเองไม่ได้) · **หลังบ้านต้องเช็คซ้ำทุกครั้ง** ใน module ที่ทำรายการ |
| ACC-18 | **ตัวเลขใน API vs ภายใน (กฎข้อ 9)**: JSON ใช้ number (เงินเช่น `962056.00` · % เช่น `95.5`) ตามรูปแบบที่หน้าบ้านใช้ · **ภายใน DB และ Go ใช้จำนวนเต็มเสมอ** (เงิน = `int64` หน่วยย่อยที่สุดของสกุล — **ทุกสกุลทศนิยม 2 ตำแหน่ง** หน่วยย่อย = 1/100 · % = bp ×100) · แปลงที่ขอบระบบเท่านั้น · ตอนรับค่าอ่าน JSON number เป็นข้อความแล้วแปลงเป็นจำนวนเต็มตรง (ไม่ผ่าน float) · ห้ามคำนวณเงิน / % ด้วย float |
| ACC-13 | อ่านอย่างเดียว · เปลี่ยนรหัสผ่าน / passcode ของตัวเองใช้เส้นเดิม `POST /bo/pr/auth/password/change` และ `/auth/passcode/change` |
| ACC-14 | login ล่าสุด = ค่าที่บันทึกตอน login (AUTH-12) จึงเป็นเวลาของ session ปัจจุบัน |

### 1.3 ข้อมูลรับรอง API

| ID | Rule |
|---|---|
| ACC-01 | เจ้าของ Key = **Company Seamless 1 to 1** · **Share Master** · **Share Reseller** · แต่ละบัญชีมี Key, ลิงก์ตอบกลับ และรายการ IP **ของตัวเอง** · บัญชีอื่นทั้งหมด (Superadmin, ADMIN, Company Transfer, Company Seamless Reseller / Master, Share B2B / B2C, Agent) เปิดหน้านี้ไม่ได้ (`403301`) |
| ACC-02 | เจ้าของเปิดดู / บันทึกได้ · sub ของเจ้าของ: สิทธิ์ `account` = `view` → GET ได้ · `account` = `edit` → POST ได้ · ไม่ได้รับสิทธิ์ → `402303` (module ② MGMT-51) · บัญชีที่ `status` ไม่ใช่ `ACTIVE` ถูกกันที่ middleware กลาง (ACC-31 / AUTH-54) |
| ACC-03 | Key สร้างอัตโนมัติ 1 ค่าต่อเจ้าของ: สุ่ม 32 byte ด้วย `crypto/rand` แสดงเป็น hex ตัวพิมพ์เล็ก 64 ตัว · **สร้างใหม่ไม่ได้** · ใช้ทั้งระบุตัวและคำนวณ `sign` |
| ACC-04 | Key ดูซ้ำได้ (ปุ่มคัดลอก) → เก็บแบบเข้ารหัส AES-256-GCM ที่ถอดกลับได้ (ไม่ใช่ hash) + `sha256(Key)` ไว้ค้นตอนลูกค้าเรียก API · response ใส่ `Cache-Control: no-store` · ห้าม log Key |
| ACC-05 | สร้าง Key ตอนสร้างบัญชีเจ้าของ (module ②) · บัญชีที่ยังไม่มี Key สร้างตอนเปิดหน้าครั้งแรก · เรียกพร้อมกันได้ Key เดียวเสมอ (unique ที่ `agent_id`) |
| ACC-06 | ลิงก์ตอบกลับ: ต้องเป็น URL `https://` ที่มี host · ยาวไม่เกิน 500 ตัว · เว้นว่างได้ (= ยังไม่ตั้ง): ส่ง `null` หรือ `""` (ตัดช่องว่างแล้วว่าง) → เก็บเป็น `NULL` · response ส่ง `null` · ไม่ส่ง field = `422` (บันทึกแทนทั้งชุด — ACC-08) |
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
| GET | `/api/v1/bo/pr/account/dashboard` | `RequirePermission(dashboard, view)` |
| GET | `/api/v1/bo/pr/account/profile` | — (เปิดได้ตอน SUSPENDED) |
| GET | `/api/v1/bo/pr/account/api-credential` | `RequirePermission(account, view)` |
| POST | `/api/v1/bo/pr/account/api-credential` | `RequirePermission(account, edit)` · `RequirePasscode` |

### GET /api/v1/bo/pr/account/dashboard

Query: `game` (รหัสเกม · ไม่ส่ง = ทุกเกม) · `year` (บังคับ) · `month` (1–12 · ไม่ส่ง = ทั้งปี) · `currency` (บังคับ)

Response `data`:
```json
{
  "currency": "THB",
  "period": { "from": "2026-10-01", "to": "2026-10-31" },
  "updated_at": "2026-10-05T00:47:00+07:00",
  "cards": {
    "active_members":   { "value": 12 },
    "new_members":      { "value": 3, "change_pct": 50.0 },
    "total_members":    { "value": 120, "change_pct": 2.56 },
    "bet_count":        { "value": 5400, "change_pct": -3.1 },
    "win_count":        { "value": 2100, "change_pct": 1.2 },
    "bet_amount":       { "value": 987654.00, "change_pct": 4.0 },
    "bet_amount_pt":    { "value": 98765.40, "change_pct": 4.0 },
    "win_amount_pt":    { "value": 80000.00, "change_pct": null },
    "win_loss_pt":      { "value": 18765.40, "change_pct": 12.5 },
    "commission_received": { "value": 1200.00, "change_pct": 3.0 },
    "commission_paid":     { "value": 800.00, "change_pct": -1.0 },
    "income":              { "value": 19165.40, "change_pct": 11.8 }
  },
  "top_bet":  [ { "rank": 1, "username": "m001", "amount": 15000.00 } ],
  "top_win":  [ { "rank": 1, "username": "m007", "amount": 9000.00 } ],
  "top_loss": [ { "rank": 1, "username": "m013", "amount": 7000.00 } ]
}
```
| field | การ์ด | ชนิด |
|---|---|---|
| `active_members` · `new_members` · `total_members` | สมาชิกที่กำลังใช้งาน · สมัครใหม่ · ทั้งหมด | จำนวนคน |
| `bet_count` · `win_count` | ยอดเล่นทั้งหมด · ยอดเล่นที่ถูกรางวัล | จำนวนครั้ง |
| `bet_amount` | ยอดเล่นจริง | เงิน |
| `bet_amount_pt` · `win_amount_pt` · `win_loss_pt` | ยอดเล่นทั้งหมด / ถูกรางวัล / แพ้ชนะ ตาม PT | เงิน |
| `commission_received` · `commission_paid` · `income` | Commission ที่ได้ · ที่จ่าย · รายได้ (ACC-25) | เงิน |

- จำนวนเงินและ `change_pct` เป็น JSON number ตาม ACC-18 (ภายในเป็นจำนวนเต็ม) · `change_pct` = `null` เมื่อช่วงก่อนหน้าเป็น 0
- `period`: ส่ง `month` = เดือนนั้น · ไม่ส่ง = 1 ม.ค. – 31 ธ.ค. ของ `year`

Error codes: `422` (`year` / `month` ผิดรูปแบบ · ไม่ส่ง `currency` หรือเป็นสกุลที่บัญชีไม่มี · `game` ไม่มีในระบบ — msg บอก field) · `402303` (sub ไม่มีสิทธิ์)

### GET /api/v1/bo/pr/account/profile

Response `data` (ตัวอย่าง sub ของ Share B2C):
```json
{
  "username": "share01@staff",
  "role": "SHAREHOLDER",
  "user_type": "SHARE_B2C",
  "status": "SUSPENDED",
  "is_subaccount": true,
  "owner_username": "share01",
  "passcode_set": true,
  "last_login_at": "2026-10-05T10:00:00+07:00",
  "last_login_ip": "203.0.113.10",
  "created_at": "2026-10-01T09:00:00+07:00",
  "currencies": ["THB"],
  "balances": [
    { "currency": "THB", "amount": 962056.00 }
  ],
  "pt": {
    "game": { "pt_from_parent": 90, "pt": 20, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true }
  },
  "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": false },
  "permissions": {
    "dashboard": "view", "account": "view", "member": "off", "pt": "off", "report": "view",
    "bet_cancel": "off", "payment": "off", "asset": "off", "announcement": "off"
  }
}
```
- `status` = สถานะที่ใช้งานจริง (ACC-30) — ตัวอย่างนี้ผู้สร้าง (`share01`) ถูกระงับ sub จึงได้ `SUSPENDED`
- `pt.game`: ได้รับ 90 (`pt_from_parent` — ผู้สร้างตั้ง) · ถือจาก Member ของตัวเอง 20 (`pt` — ตั้งเอง) · `status = true` เปิดทั้งกลุ่ม · `status_game` เปิด / ปิดทีละเกม
- `user_type`: `SUPERADMIN` · `ADMIN` · `COMPANY_TRANSFER` · `COMPANY_SEAMLESS_RESELLER` · `COMPANY_SEAMLESS_MASTER` ·
  `COMPANY_SEAMLESS_1TO1` · `SHARE_B2B` · `SHARE_B2C` · `SHARE_RESELLER` · `SHARE_MASTER` · `AGENT`
- `owner_username` = `null` เมื่อไม่ใช่ sub · `last_login_*` = `null` ถ้ายังไม่เคยบันทึก
- ค่า % (`pt`, `pt_from_parent`, `force`, `remain_quota`, `commission_percent`) เป็น JSON number ทศนิยมไม่เกิน 2 ตำแหน่ง — ภายในเก็บเป็นจำนวนเต็ม (ACC-18)
- กลุ่ม PT, รหัสเกม, รายการสกุล (27 สกุล) และชื่อสิทธิ์ ใช้ตามที่ module ② กำหนด (ค่าในตัวอย่างเป็นแค่รูปแบบ)
- `balances` ตาม ACC-19 (ยอดจริงจาก `agent_balances`)
- `permissions` = object `{ "{menu}": "off" | "view" | "edit" }` ครบทุกเมนูของประเภทบัญชี (module ② MGMT-50 · MGMT-53) · บัญชีหลักได้ `edit` ทุกเมนู · sub ได้ตามที่เจ้าของให้

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

Error codes: `403301` · `402303` (sub ไม่มีสิทธิ์)

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

- `callback_url`: `null` หรือ `""` = ไม่ตั้ง (ACC-06)

Error codes: `422` (ไม่ส่ง field · ลิงก์ไม่ใช่ https / ยาวเกิน · IP ผิดรูปแบบ · เกิน 50 รายการ · IP ซ้ำ — msg บอก field และรายการ),
`403301`, `401204`, `401205` · `402303` (sub ไม่มีสิทธิ์)

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
| ACC-30 | Agent ACTIVE · Share ที่เป็นหัวสายถูกล็อก | Agent `status = LOCKED` |
| ACC-30 | Agent ถูกล็อก · Share ที่เป็นหัวสาย ACTIVE | Agent `status = LOCKED` · Share `status = ACTIVE` ใช้งานได้ปกติ |
| ACC-31 | `status = SUSPENDED` เรียก profile | สำเร็จ |
| ACC-31 | `status = SUSPENDED` เรียก dashboard / POST api-credential / route อื่นนอก Profile, Report | ถูกปฏิเสธ (AUTH-54) |
| ACC-21 | Company A ดู dashboard | นับเฉพาะ Member ในสาย A |
| ACC-22 | `currency` ที่บัญชีไม่มี / ไม่ส่ง · `month = 13` · ไม่ส่ง `year` | `422` |
| ACC-22 | ไม่ส่ง `month` | `period` = ทั้งปีของ `year` |
| ACC-24 | ช่วงก่อนหน้ามียอด 0 | `change_pct = null` |
| ACC-25 | Member แพ้ 1,000 · คนที่ login ถือ 20% | `win_loss_pt` บวก 200 · `income` = `win_loss_pt` + `commission_received` − `commission_paid` |
| ACC-26 | Member ในสาย 15 คน | Top 10 แต่ละตารางมี 10 แถว เรียงจากมากไปน้อย |
| ACC-11 | Agent / SUPERADMIN / ADMIN เรียก profile | ข้อมูลของตัวเอง · `is_subaccount = false` · `owner_username = null` |
| ACC-11 | ยังไม่ตั้ง passcode เรียก profile | `401304` (ด่านหลัง login) |
| ACC-12 | ผู้สร้างของ sub เป็น SUSPENDED | sub ได้ `status = SUSPENDED` · ไม่มี field `effective_status` |
| ACC-12 | Company Seamless Master เรียก profile | `user_type = COMPANY_SEAMLESS_MASTER` · `pt.game.pt = 0` (ล็อก — module ② MGMT-19) · `pt.game.pt_from_parent` = ค่าที่ Superadmin ให้ |
| ACC-15 | sub ของ Share B2C เรียก profile | `user_type`, `currencies`, `balances`, `pt` เท่าของผู้สร้าง · `permissions` = ที่ผู้สร้างให้ |
| ACC-16 | ADMIN เรียก profile | `pt = {}` · `status_game = {}` |
| ACC-16 | SUPERADMIN เรียก profile | `pt.game.pt_from_parent = 100` · `pt.game.pt = 0` (ไม่มี Member ตรง) |
| ACC-16 | ชั้นบนปิดเกม `scratch_card` ให้บัญชีนี้ | `status_game.scratch_card = false` · เกมอื่น `true` · `pt` ยังเป็นชุดเดียว |
| ACC-19 | Share B2B มี THB และ USD | `balances` 2 รายการ (THB, USD) |
| ACC-19 | Company Seamless 1 to 1 | `balances` ทุกรายการ `amount = 0` |
| ACC-19 | Share สร้างพร้อมยอดเงินตั้งต้น THB 10,000 (module ② MGMT-15A) | `balances` THB `amount = 10000.00` |
| ACC-12 | Agent (บัญชีหลัก) เรียก profile | `permissions` ครบ 9 เมนู ค่า `edit` ทั้งหมด |
| ACC-15 | sub ที่ได้แค่ `report` = `view` | `permissions.report = "view"` · เมนูอื่น `"off"` |
| ACC-18 | Commission 0.5% | profile ได้ `0.5` · DB เก็บจำนวนเต็ม |
| ACC-14 | login แล้วเรียก profile | `last_login_at` / `last_login_ip` ตรงกับ login นี้ |
| ACC-01 | Company Seamless 1 to 1 / Share Master / Share Reseller เรียก GET | สำเร็จ · ได้ Key ของตัวเอง |
| ACC-01 | Share Reseller 2 บัญชีใต้ Company เดียวกัน | ได้ Key คนละค่า |
| ACC-01 | Company Seamless Reseller / Master, Company Transfer, Share B2B / B2C, Agent, Superadmin เรียก GET / POST | `403301` |
| ACC-02 | sub ที่ได้สิทธิ์ `account` = `view` | GET สำเร็จ · POST `402303` |
| ACC-02 | sub ที่ได้ `account` = `off` | GET / POST `402303` |
| ACC-02 | เจ้าของ SUSPENDED กดบันทึก | ถูกปฏิเสธที่ middleware (ACC-31 / AUTH-54) |
| ACC-03 | เจ้าของ GET ครั้งแรก | Key hex ตัวพิมพ์เล็ก 64 ตัว |
| ACC-04 | GET ซ้ำ | Key เดิม · header `Cache-Control: no-store` · DB ไม่มี Key แบบ plain |
| ACC-05 | GET พร้อมกัน 2 คำขอครั้งแรก | ได้ Key เดียวกันทั้งคู่ · มี credential 1 แถว |
| ACC-06 | `http://...` / `ftp://...` / `https://` ไม่มี host / ยาว 501 | `422` |
| ACC-06 | `callback_url` = `null` / `""` / `"   "` | สำเร็จ · GET ได้ `null` |
| ACC-06 | ไม่ส่ง field `callback_url` | `422` |
| ACC-07 | `2001:db8::1` / `300.1.1.1` / `1.2.3.4/33` | `422` |
| ACC-07 | 51 รายการ / IP ซ้ำ (`1.2.3.4` กับ `1.2.3.4/32`) | `422` |
| ACC-07 | `1.2.3.4` | GET ได้ `1.2.3.4/32` |
| ACC-08 | ไม่ส่ง `passcode` / ผิด | `422` / `401204` |
| ACC-08 | บันทึกรายการใหม่ | IP เดิมที่ไม่อยู่ในรายการใหม่ถูกลบ |
| ACC-09 | บันทึกสำเร็จ | มีแถว `api_credential_logs` ค่าเก่า / ใหม่ถูกต้อง · ไม่มี Key |

## 8. Contract changes (แจ้ง frontend)

- เส้นใหม่ 4 เส้นตามหัวข้อ 5
- สถานะส่ง key เดียว `status` = สถานะที่ใช้งานจริง (ACC-30) · ไม่มี `effective_status`
- Profile มี `user_type`, `currencies`, `balances`, `pt` (ชุดเดียวต่อกลุ่ม — รวม `pt_from_parent` ในกลุ่ม), `status_game` (ต่อเกม แยกจาก `pt`), `permissions` — ใช้แสดงผล / เช็คเบื้องต้นในฟอร์มได้ หลังบ้านเช็คซ้ำเสมอ
- Dashboard: `month` ไม่บังคับ (ไม่ส่ง = ทั้งปี) · เพิ่มการ์ด `commission_received`, `commission_paid`, `income`
- หน้า 1.3 แสดงเฉพาะ Company Seamless 1 to 1 / Share Master / Share Reseller และ sub ที่ได้รับสิทธิ์ · `callback_url` ว่างส่ง `null` หรือ `""`
- `balances` ส่งยอดจริงจาก `agent_balances` · `permissions` เป็น object `{ menu: off | view | edit }`

## 9. การตัดสินใจ (2026-10-05)

1. Key ดูซ้ำได้ (ตามภาพตัวอย่าง) แทนการแสดงครั้งเดียว → เก็บแบบเข้ารหัส ถอดกลับได้
2. Key ค่าเดียว (ไม่แยก key / secret) · สร้างใหม่ไม่ได้
3. ไม่มี IP = ปิดการเรียก API · รองรับ IPv4 และ CIDR · สูงสุด 50 รายการ
4. **review 2026-10-05:**
   - เจ้าของ Key เปลี่ยนจาก "1 Key ต่อ Company Seamless" เป็น Company Seamless 1 to 1 / Share Master / Share Reseller บัญชีละ Key
   - คง `status` และ `effective_status` แยก 2 field (เหตุผล: แยก "ถูกระงับเอง" กับ "ได้รับผลจากหัวสาย") · การเข้าใช้งานตัดสินจาก `effective_status`
   - Profile ส่งประเภท, สกุลเงิน, ค่าหุ้นส่วน `pt_by_game` (อ่านแยกต่อเกม · ตั้งค่าชุดเดียวใช้กับทุกเกม) และสิทธิ์
   - ระบบสิทธิ์กำหนดใน module ② ครั้งเดียว · Profile และ 1.3 ใช้ตามนั้น
   - Dashboard กลับมาอยู่ใน module นี้ · spec ตอนนี้ implement หลังมี bet
   - ตัวเลขใน API เป็น JSON number ตามรูปแบบที่หน้าบ้านใช้ แต่ DB / Go เก็บและคำนวณเป็นจำนวนเต็ม (ACC-18) — ไม่แก้กฎข้อ 9
   - Profile ใช้ `pt_by_game` (หมวด → เกม) และ `status_game` รายบัญชี · `balances` 27 สกุลเพิ่มตอนทำ module การเงิน
5. **review รอบ 2 (2026-10-06)** — แทนข้อที่ขัดกันในข้อ 4:
   - Profile ส่งค่าหุ้นส่วนชุดเดียวต่อกลุ่ม (`pt` รวม `pt_from_parent` ในกลุ่ม) + `status_game` ต่อเกม แยกจาก `pt` แทน `pt_by_game` · ระบบยังเก็บต่อเกม (module ② MGMT-16)
   - Profile มี `balances` ยอดเงินแยกสกุล (ACC-19) · ค่าจริงมาจาก module การชำระเงิน
   - รวม `status` / `effective_status` เป็น key เดียว `status` = สถานะที่ใช้งานจริง (ACC-30) · ใช้ทั้งระบบรวม module ②
   - Dashboard: เพิ่มการ์ดรายได้และ Commission · `month` ไม่บังคับ · ระบุ error และชนิดของการ์ด · `currency` ต้องเป็นสกุลที่บัญชีมี
   - ตัด `403302` — middleware กลาง (AUTH-54) กันก่อนถึง logic เสมอ · error สิทธิ์ของ sub ใช้ `402303`
   - `callback_url` ว่าง: รับ `null` / `""` เก็บเป็น `NULL`
   - (review module ② 2026-10-07) `permissions` เป็น object `{ menu: off | view | edit }` · `balances` ส่งยอดจริงจาก `agent_balances` (module ② MGMT-15A)
   - (คุยกับ lead 2026-10-06) `pt_from_parent` อยู่ในกลุ่ม `pt` · เพิ่ม `status` ของกลุ่ม · ผู้สร้างตั้งค่าที่ให้ลูกแต่ละคนแยกกัน และบัญชีตั้งค่าถือ `pt` ของตัวเอง (module ② MGMT-22)
6. **Error code** (module `account` = `bb=03`) — business error ตอบ HTTP 200

| Code | HTTP | ความหมาย |
|---|---|---|
| 403301 | 200 | บัญชีนี้ไม่มีข้อมูลรับรอง API |

`403302` (เดิม: บัญชีถูกระงับ) ตัดออกก่อนปล่อยใช้ — จองถาวร
