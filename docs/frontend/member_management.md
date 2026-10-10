# API หลังบ้าน — การจัดการ Member (สำหรับหน้าบ้าน) — 11 ต.ค. 2026 (โครงเส้นใหม่)

เอกสารนี้ตรงกับโค้ดใน branch `boiledegg/bo/member_management` (**เส้นใหม่ตาม lead review** — ใช้บน server หลัง merge + deploy) ·
spec เต็ม: `docs/modules/member_management.md` · ผู้ดูแล module: maofoy

> **เปลี่ยนจากเดิม (11 ต.ค.)** — path ใหม่ เส้นเดิมได้ `404` (หัวข้อ 4) · `update-info` + `update-pt` รวมเป็น `members/detail/update` ·
> เส้นใหม่ `members/list` (dropdown) · **เบอร์ 2 field** `phone_country_code` + `phone` · สิทธิ์ sub ใช้แค่ `member` · `pt` แสดงเสมอ · ส่งสถานะเดิม = 200 ไม่มีประวัติเพิ่ม

พื้นฐานใช้ร่วมกับเอกสารการจัดการสมาชิก: base URL / header / รูปแบบ response / กติกา (`id` ใน body · ห้าม `null` · % / เงิน) / เบอร์โทร / ตั้งค่า Postman ดู
[agent_management.md](agent_management.md) หัวข้อ 1 · login / session / ด่านหลัง login ดู [auth.md](auth.md)

## 1. ภาพรวม

| # | Path | ใช้เมื่อ | สิทธิ์ของ sub |
|---|---|---|---|
| 2.1 | `/api/v1/bo/pr/manage/members/create` | สร้าง Member | `member` edit (+ `payment` edit ถ้าส่ง `balance`) |
| 2.2 | `/api/v1/bo/pr/manage/members/detail/get` | รายละเอียด Member | `member` view |
| 2.3 | `/api/v1/bo/pr/manage/members/detail/update` | แก้ชื่อ / เบอร์ · PT / Commission (เส้นเดียว) | `member` edit |
| 2.4 | `/api/v1/bo/pr/manage/members/status/update` | แก้สถานะ | `member` edit |
| 2.5 | `/api/v1/bo/pr/manage/members/list` | dropdown Member ลูกตรง ("คัดลอกการตั้งค่าจาก") | `member` view |

ทุกเส้นเป็น `POST`

- **ใครสร้าง Member ได้:** Company Seamless 1 to 1 · Share B2C / Reseller / Master · Agent (Share B2B สร้างไม่ได้)
- **สกุลเงิน:** ใช้สกุลของผู้สร้าง ไม่ต้องส่ง · ฝั่ง Seamless ไม่มียอด (`amount` = 0 เสมอ — ซ่อนได้) และห้ามส่ง `balance`
- **`pt`** ใต้ key `minigame` ส่ง 2 ค่า: `pt` (ผู้สร้างถือสู้กับ Member คนนี้ · 0 ถึงค่าที่ผู้สร้างได้รับ ทีละ 0.5) · `commission_percent` (0–1 ทีละ 0.1) · `remain_quota` ระบบคิด = ค่าที่ผู้สร้างได้รับ − `pt`
- **เบอร์โทร:** กติกาเดียวกับฝั่ง agent ([agent_management.md](agent_management.md) หัวข้อ 1.5) ยกเว้นเรื่องซ้ำ: **ห้ามซ้ำเฉพาะ Member ของผู้สร้างเดียวกัน** (`402403`) · คนละผู้สร้าง / ซ้ำกับ agent ได้
- **รายชื่อ / ค้นหา Member ทั้งสาย:** ใช้ `downlines/list` ของ agent_management (Member ปนกับฝั่ง agent ดูจาก `role = MEMBER`)

**error ที่เกิดได้กับทุกเส้น:** `401202`, `401203`, `401301`, `401302` → หน้า login · `401304`, `401306`, `401307` → ด่านหลัง login · `401311` บัญชีถูกระงับ · `402303` sub ไม่มีสิทธิ์

## 2. Endpoints

### 2.1 สร้าง — `members/create`

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/members/create" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" \
  -d '{"request_id":"{{$guid}}","username":"Mem01","password":"aA4b4c4d4e4f","name":"ใจดี","phone_country_code":"66","phone":"812345678","balance":{"THB":100},"pt":{"minigame":{"pt":50,"commission_percent":0.3}}}'
```
Response: `{ "code": 200, "msg": "สำเร็จ", "data": { "id": 501, "username": "mem01" } }`

- กติกา `request_id` / `username` / `password` / `name` / `balance` เหมือนเส้นสร้างฝั่ง agent ([agent_management.md](agent_management.md) หัวข้อ 3.1)
- เบอร์ไม่กรอกส่ง `"phone_country_code": "", "phone": ""`
- ไม่มี `user_type` · `currencies` · `status_game` · `pt` ต้องส่งทั้ง `pt` และ `commission_percent` · ส่ง `remain_quota` / `pt_from_parent` / `force` / `status` = `422`

Error: `422`, `402301`, `402303`, `402305`, `402309`, `402312`, `402401`, `402403`

### 2.2 รายละเอียด — `members/detail/get`

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/members/detail/get" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"id":501}'
```
Response `data`:
```json
{
  "id": 501, "role": "MEMBER", "user_type": "MEMBER", "username": "mem01", "name": "ใจดี",
  "phone_country_code": "66", "phone": "812345678",
  "status": "ACTIVE", "parent_username": "agent01", "currencies": ["THB"],
  "balances": [ { "currency": "THB", "amount": 100 } ],
  "pt": {
    "minigame": { "pt": 50, "remain_quota": 10, "commission_percent": 0.3, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "agent01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "agent01" }
  },
  "last_login_at": "", "last_login_ip": "", "created_at": "2026-10-02T14:30:00+07:00"
}
```
- ดูได้: ผู้สร้าง Member และชั้นบนในสาย · `pt` แสดงเสมอ · ไม่มี `status_game` / `passcode_set`

Error: `422`, `402303`, `402402`

### 2.3 แก้ Member — `members/detail/update`

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/members/detail/update" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" \
  -d '{"id":501,"info":{"name":"ใจดี","phone_country_code":"66","phone":"822222222"},"pt":{"minigame":{"pt":40,"commission_percent":0.3}}}'
```
Response: `{ "code": 200, "msg": "สำเร็จ" }`

| section | กติกา |
|---|---|
| `info` | `name` · `phone_country_code` + `phone` — ส่งบาง field ได้ (ไม่ส่ง = คงเดิม) · เบอร์ต้องส่ง 2 field คู่กัน · ล้างเบอร์ส่ง `""` ทั้งคู่ |
| `pt` | ส่งเฉพาะกลุ่มที่แก้ · ในกลุ่มต้องครบ `pt` + `commission_percent` · `pt` ไม่เกินค่าที่ผู้สร้างได้รับ (`402305`) · `remain_quota` ระบบคิดใหม่ |

- **ส่งเฉพาะ section ที่แก้** · ไม่ส่งเลยสัก section = `422`
- แก้ได้เฉพาะ**ผู้สร้างโดยตรง** (ชั้นบนอื่น = `402304` · นอกสาย = `402402`) · ไม่ต้อง passcode
- **ทั้งคำขอสำเร็จหรือไม่บันทึกเลย** · ค่าเหมือนเดิมทั้งหมด = 200 (ไม่มีประวัติเพิ่ม)

Error: `422`, `402303`, `402304`, `402305`, `402309`, `402402`, `402403`

### 2.4 สถานะ — `members/status/update`

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/members/status/update" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"id":501,"status":"LOCKED"}'
```
Response: `{ "code": 200, "msg": "สำเร็จ" }`
- `ACTIVE` / `SUSPENDED` / `LOCKED` · ตั้งกลับ `ACTIVE` ได้ · ผู้สร้างโดยตรงเท่านั้น (`402304`) · ไม่ต้อง passcode · ส่งสถานะเดิม = 200 (ไม่มีประวัติเพิ่ม)

Error: `422`, `402303`, `402304`, `402402`

### 2.5 dropdown Member ลูกตรง — `members/list`

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/members/list" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"keyword":""}'
```
Response `data`:
```json
[
  { "id": 501, "username": "mem01", "name": "ใจดี", "status": "ACTIVE" },
  { "id": 502, "username": "mem02", "name": "mem02", "status": "LOCKED" }
]
```
- Member ที่ตัวเองสร้าง (ลูกตรง) · `keyword` ไม่บังคับ 4–32 ตัว ค้น username บางส่วน · A→Z · ไม่แบ่งหน้า สูงสุด 500 · ไม่มี = `[]`
- "คัดลอกการตั้งค่าจาก": เลือกแล้วเรียก `members/detail/get` เอา `pt` มาเติมฟอร์มสร้าง

Error: `422`, `402303`

## 3. Error codes

ตอบ HTTP 200 ทั้งหมด · ดูที่ `code` · `msg` บอกชื่อ field

| Code | ความหมาย |
|---|---|
| `422` | ข้อมูลไม่ผ่าน (`msg` บอก field) |
| `402301` | สร้างบัญชีประเภทนี้ไม่ได้ |
| `402303` | ไม่มีสิทธิ์ใช้งานเมนูนี้ (sub) |
| `402304` | แก้ไขได้เฉพาะผู้สร้างของบัญชีนี้ |
| `402305` | `pt` ของ Member เกินค่าที่ผู้สร้างได้รับ |
| `402309` | Commission เกินที่กำหนด |
| `402312` | ยอดเงินของคุณไม่พอสำหรับยอดเงินตั้งต้น |
| `402401` | username นี้ถูกใช้แล้ว |
| `402402` | ไม่พบบัญชีในสายของคุณ |
| `402403` | เบอร์โทรนี้ถูกใช้แล้ว (Member ของผู้สร้างเดียวกัน) |

## 4. เส้นเดิมที่ลบ (11 ต.ค.) → ใช้แทนด้วย

| เส้นเดิม | ใช้แทน |
|---|---|
| `members/detail` | `members/detail/get` |
| `members/update-info` · `members/update-pt` | `members/detail/update` (section `info` / `pt`) |
| `members/update-status` | `members/status/update` |
