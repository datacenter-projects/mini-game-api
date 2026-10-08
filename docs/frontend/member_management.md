# API หลังบ้าน — การจัดการ Member (สำหรับหน้าบ้าน) — 9 ต.ค. 2026

เอกสารนี้สรุปจากโค้ดใน branch `dev` · spec เต็ม: `docs/modules/member_management.md` · ผู้ดูแล module: maofoy
แยกจาก [agent_management.md](agent_management.md) 2026-10-09 — **path, request และ response ไม่เปลี่ยน**

พื้นฐานใช้ร่วมกับเอกสารการจัดการสมาชิก: base URL / header / รูปแบบ response / กติกา (`id` ใน body · ห้าม `null` · % / เงิน) และตั้งค่า Postman ดู
[agent_management.md](agent_management.md) หัวข้อ 1 · login / session / ด่านหลัง login ดู [auth.md](auth.md)

## 1. ภาพรวม

| # | Method | Path | ใช้เมื่อ | สิทธิ์ (sub) |
|---|---|---|---|---|
| 2.1 | POST | `/api/v1/bo/pr/manage/members/create` | สร้าง Member | `member` edit · `pt` edit (+ `payment` edit ถ้าส่ง `balance`) |
| 2.2 | POST | `/api/v1/bo/pr/manage/members/detail` | รายละเอียด Member | `member` view |
| 2.3 | POST | `/api/v1/bo/pr/manage/members/update-info` | แก้ชื่อ · เบอร์ | `member` edit |
| 2.4 | POST | `/api/v1/bo/pr/manage/members/update-status` | แก้สถานะ | `member` edit |
| 2.5 | POST | `/api/v1/bo/pr/manage/members/update-commission` | แก้ Commission | `pt` edit |

- **ใครสร้าง Member ได้:** Company Seamless 1 to 1 · Share B2C / Reseller / Master · Agent (Share B2B สร้างไม่ได้)
- **สกุลเงิน:** ใช้สกุลของผู้สร้าง ไม่ต้องส่ง · **`pt`** มีแค่ `commission_percent` (0–1 ทีละ 0.1) ใต้ key `minigame`
- **รายชื่อ / ค้นหา Member:** ใช้ `downlines/list` · `downlines/search` ของ agent_management (Member ปนกับฝั่ง agent ดูจาก `role = MEMBER`)

**error ที่เกิดได้กับทุกเส้น:** `401202`, `401203`, `401301`, `401302` → หน้า login · `401304`, `401306`, `401307` → ด่านหลัง login · `401311` บัญชีถูกระงับ · `402303` sub ไม่มีสิทธิ์

## 2. Endpoints

### 2.1 POST /api/v1/bo/pr/manage/members/create

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/members/create" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"request_id":"0b9c5a77-2d1e-4f3a-8c61-5e4d3c2b1a09","username":"Mem01","password":"aA4b4c4d4e4f","name":"ใจดี","phone":"","balance":{"THB":100.00},"pt":{"minigame":{"commission_percent":0.3}}}'
```

Request:
```json
{
  "request_id": "0b9c5a77-2d1e-4f3a-8c61-5e4d3c2b1a09",
  "username": "Mem01",
  "password": "aA4b4c4d4e4f",
  "name": "ใจดี",
  "phone": "",
  "balance": { "THB": 100.00 },
  "pt": { "minigame": { "commission_percent": 0.3 } }
}
```

Response `data`:
```json
{ "id": 501, "username": "mem01" }
```

- กติกา `request_id` / `username` / `password` / `name` / `phone` / `balance` เหมือนเส้นสร้างฝั่ง agent ([agent_management.md](agent_management.md) หัวข้อ 3.1)
- ไม่มี `user_type` · `currencies` · `status_game` · สกุล = สกุลของผู้สร้าง · `pt` มีแค่ `commission_percent`

Error: `422`, `402301`, `402303`, `402309`, `402312`, `402401`, `402403`

### 2.2 POST /api/v1/bo/pr/manage/members/detail

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/members/detail" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"id":501}'
```

Request:
```json
{ "id": 501 }
```

Response `data`:
```json
{
  "id": 501,
  "role": "MEMBER",
  "user_type": "MEMBER",
  "username": "mem01",
  "name": "ใจดี",
  "phone": "",
  "status": "ACTIVE",
  "parent_username": "agent01",
  "currencies": ["THB"],
  "balances": [ { "currency": "THB", "amount": 100.00 } ],
  "pt": {
    "minigame": { "commission_percent": 0.3, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "agent01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "agent01" }
  },
  "last_login_at": "",
  "last_login_ip": "",
  "created_at": "2026-10-02T14:30:00+07:00"
}
```

- ดูได้: ผู้สร้าง Member และชั้นบนในสาย · ไม่มี `status_game` / `passcode_set`

Error: `422`, `402303`, `402402`

### 2.3 POST /api/v1/bo/pr/manage/members/update-info

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/members/update-info" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"id":501,"name":"ใจดี","phone":"0822222222"}'
```

Request:
```json
{ "id": 501, "name": "ใจดี", "phone": "0822222222" }
```
Response: `{ "code": 200, "msg": "สำเร็จ" }`

- แก้ได้เฉพาะผู้สร้าง Member โดยตรง (`402304`) · `phone` `""` = ไม่ตั้ง · ส่ง `null` หรือไม่ส่ง field = `422`

Error: `422`, `402303`, `402304`, `402402`, `402403`

### 2.4 POST /api/v1/bo/pr/manage/members/update-status

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/members/update-status" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"id":501,"status":"LOCKED"}'
```

Request:
```json
{ "id": 501, "status": "LOCKED" }
```
Response: `{ "code": 200, "msg": "สำเร็จ" }`

- `status`: `ACTIVE` / `SUSPENDED` / `LOCKED` · ตั้งกลับ `ACTIVE` ได้ · แก้ได้เฉพาะผู้สร้างโดยตรง (`402304`) · ไม่ต้อง passcode

Error: `422`, `402303`, `402304`, `402402`

### 2.5 POST /api/v1/bo/pr/manage/members/update-commission

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/members/update-commission" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"id":501,"pt":{"minigame":{"commission_percent":0.3}}}'
```

Request:
```json
{ "id": 501, "pt": { "minigame": { "commission_percent": 0.3 } } }
```
Response: `{ "code": 200, "msg": "สำเร็จ" }`

- มีแค่ `commission_percent` · ส่ง field อื่นในกลุ่ม = `422`

Error: `422`, `402303`, `402304`, `402309`, `402402`

## 3. Error codes

| Code | HTTP | ความหมาย |
|---|---|---|
| 422 | 200 | ข้อมูลไม่ผ่าน (`msg` บอก field) |
| 402301 | 200 | สร้างบัญชีประเภทนี้ไม่ได้ |
| 402303 | 200 | ไม่มีสิทธิ์ใช้งานเมนูนี้ (sub) |
| 402304 | 200 | แก้ไขได้เฉพาะผู้สร้างของบัญชีนี้ |
| 402309 | 200 | Commission เกินที่กำหนด |
| 402312 | 200 | ยอดเงินของคุณไม่พอสำหรับยอดเงินตั้งต้น |
| 402401 | 200 | username นี้ถูกใช้แล้ว |
| 402402 | 200 | ไม่พบบัญชีในสายของคุณ |
| 402403 | 200 | เบอร์โทรนี้ถูกใช้แล้ว |
