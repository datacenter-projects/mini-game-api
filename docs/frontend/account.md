# API หลังบ้าน — บัญชี (Account) สำหรับหน้าบ้าน — 8 ต.ค. 2026

เอกสารนี้สรุปจาก spec `docs/modules/account.md` (APPROVED) และโค้ดใน branch `boiledegg/bo/account` (merge การจัดการสมาชิกแล้ว)
· พื้นฐาน (base URL, header, รูปแบบ response, login, session) ดู [auth.md](auth.md)

## 0. สถานะของแต่ละเส้น

| เส้น | สถานะ | หมายเหตุ |
|---|---|---|
| `GET /api/v1/bo/pr/account/profile` | ✅ ใช้ได้ (ยังไม่ merge เข้า `dev`) | ค่าจริงทุก field (หัวข้อ 2.2) |
| `GET /api/v1/bo/pr/account/api-credential` | ✅ ใช้ได้ (ยังไม่ merge เข้า `dev`) | |
| `POST /api/v1/bo/pr/account/update-credential` | ✅ ใช้ได้ (ยังไม่ merge เข้า `dev`) | |
| `GET /api/v1/bo/pr/account/dashboard` | ⏳ contract ล่วงหน้า | รอ module เดิมพัน |

เส้น ⏳ ใช้ทำหน้าจอรอได้ แต่ยังเรียกไม่ได้ (ได้ `404001`)

## 1. กติกาที่ใช้ทุกเส้น

### 1.1 ไม่มี `null`

| ชนิด | ไม่มีค่า ส่งเป็น |
|---|---|
| ข้อความ | `""` |
| ตัวเลข | `0` |
| รายการ | `[]` |
| object | `{}` |

หน้าบ้านส่ง `null` มา → `422` (msg บอก field)

### 1.2 ตัวเลข

- เงิน = JSON number ทศนิยม 2 ตำแหน่ง เช่น `962056.00` (ทุกสกุล 2 ตำแหน่ง)
- % = JSON number เช่น `20`, `0.5`

### 1.3 สถานะ `status`

key เดียว = สถานะที่ใช้งานจริง (รวมผลจากหัวสายแล้ว)

| `status` | ใช้งานได้ |
|---|---|
| `ACTIVE` | ปกติ |
| `SUSPENDED` | เข้าได้เฉพาะหน้า **Profile** และ **รายงาน** แบบดูอย่างเดียว · เส้นอื่นได้ `401311` |
| `LOCKED` | เข้าไม่ได้ (`401301` / `401302`) |

ตัวอย่าง: หัวสายถูกระงับ → บัญชีนี้ได้ `status = SUSPENDED` แม้บัญชีตัวเองไม่ได้ถูกระงับ

### 1.4 ตั้งค่า Postman (ตัวแปร `{{MG_URL}}` และ `{{TOKEN}}`)

ทุกตัวอย่างใช้ตัวแปร Postman 2 ตัว — คัดลอก curl ไป **Import** ใน Postman ได้เลย (Postman แทนค่า `{{...}}` ให้)

| ตัวแปร | ค่า | ตั้งที่ |
|---|---|---|
| `MG_URL` | `http://localhost:8181` (dev ในเครื่อง) · `http://localhost:8282` (test env) | ตัวแปรของ collection หรือ environment |
| `TOKEN` | token จาก login (ตั้งอัตโนมัติด้วย script ด้านล่าง) | ตัวแปรของ collection |

ทุก path ต้องมี `/api/v1/bo` ต่อจาก `{{MG_URL}}` เช่น `{{MG_URL}}/api/v1/bo/pr/account/profile`

1. เส้น **Login** → แท็บ **Scripts** → **Post-response**:
   ```javascript
   const data = pm.response.json().data;
   if (data && data.token) pm.collectionVariables.set("TOKEN", data.token);
   ```
2. เส้นอื่น → แท็บ **Authorization** → **Bearer Token** → `{{TOKEN}}`
   (ตั้งที่ folder ครั้งเดียวแล้วให้เส้นข้างในเลือก **Inherit auth from parent** · เส้น Login ตั้ง **No Auth**)
3. กด Login ครั้งเดียว แล้วเรียกเส้นอื่นได้เลย · ได้ `401202` = ยังไม่มี token หรือหมดอายุ → Login ใหม่

## 2. Profile — `GET /api/v1/bo/pr/account/profile` (✅ ใช้ได้)

ข้อมูลบัญชีของคนที่ login (หน้า 1.2 ประวัติของฉัน) · เปิดได้แม้ `status = SUSPENDED`

```
curl "{{MG_URL}}/api/v1/bo/pr/account/profile" -H "Authorization: Bearer {{TOKEN}}"
```

### 2.1 Response `data`

```json
{
  "username": "agent01",
  "role": "AGENT",
  "user_type": "AGENT",
  "status": "ACTIVE",
  "is_subaccount": false,
  "owner_username": "",
  "passcode_set": true,
  "last_login_at": "2026-10-07T21:22:39.123+07:00",
  "last_login_ip": "203.0.113.10",
  "created_at": "2026-10-01T09:00:00+07:00",
  "currencies": ["ARS", "AUD", "...", "THB", "USD", "USDT", "VND"],
  "balances": [
    { "currency": "ARS", "amount": 0.00 },
    { "currency": "THB", "amount": 0.00 }
  ],
  "pt": {},
  "status_game": {},
  "permissions": {
    "dashboard": "off", "account": "off", "member": "off", "pt": "off", "report": "off",
    "bet_cancel": "off", "payment": "off", "asset": "off", "announcement": "off"
  }
}
```

| field | ความหมาย |
|---|---|
| `username` | username (sub = `ผู้สร้าง@ชื่อ`) |
| `role` | `SUPERADMIN` / `ADMIN` / `COMPANY` / `SHAREHOLDER` / `AGENT` · sub = role ของผู้สร้าง |
| `user_type` | ประเภทบัญชี: `SUPERADMIN` · `ADMIN` · `COMPANY_TRANSFER` · `COMPANY_SEAMLESS_RESELLER` · `COMPANY_SEAMLESS_MASTER` · `COMPANY_SEAMLESS_1TO1` · `SHARE_B2B` · `SHARE_B2C` · `SHARE_RESELLER` · `SHARE_MASTER` · `AGENT` |
| `status` | สถานะที่ใช้งานจริง (หัวข้อ 1.3) |
| `is_subaccount` | เป็นบัญชีย่อยไหม |
| `owner_username` | username ของผู้สร้าง (เฉพาะ sub · ไม่ใช่ sub = `""`) |
| `passcode_set` | ตั้ง passcode แล้วหรือยัง |
| `last_login_at` · `last_login_ip` | login ล่าสุด (= session ปัจจุบัน) · ยังไม่เคย = `""` |
| `created_at` | วันที่สร้างบัญชี |
| `currencies` | สกุลที่บัญชีใช้ได้ |
| `balances` | ยอดเงินแยกสกุล |
| `pt` | ค่าหุ้นส่วนต่อกลุ่ม (หัวข้อ 2.3) |
| `status_game` | เปิด / ปิดทีละเกม |
| `permissions` | สิทธิ์ต่อเมนู `off` / `view` / `edit` (หัวข้อ 2.4) |

### 2.2 ที่มาของค่า (8 ต.ค. 2026 — เลิกใช้ค่าชั่วคราวแล้ว)

| field | ค่า |
|---|---|
| `user_type` | ประเภทจริง เช่น `COMPANY_TRANSFER`, `SHARE_RESELLER` · sub = ของผู้สร้าง |
| `currencies` | เฉพาะสกุลของบัญชี (Superadmin, Company Transfer, Company Seamless Reseller / Master = ครบ 27) |
| `balances` | 1 รายการต่อสกุลใน `currencies` · ยอดจริง · ยังไม่มียอด และบัญชี Seamless = `0.00` |
| `pt` · `status_game` | ค่าจริงจากตอนสร้าง / แก้ในการจัดการสมาชิก (หัวข้อ 2.3) |
| `permissions` | บัญชีหลัก = `edit` ทุกเมนูของประเภท · sub = ตามที่เจ้าของให้ (ไม่ได้ให้ = `off`) |
| ADMIN | `currencies` / `balances` = `[]` · `pt` / `status_game` / `permissions` = `{}` |

### 2.3 รูปแบบ `pt` และ `status_game`

```json
"pt": {
  "game": {
    "pt_from_parent": 90,
    "pt": 20,
    "force": 0,
    "remain_quota": 0,
    "commission_percent": 0.5,
    "status": true
  }
},
"status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": false }
```

- `pt_from_parent` = PT ที่ได้รับจากผู้สร้าง · `pt` = PT ที่ถือจาก Member ใต้ตัวเอง
- `status` = เปิด / ปิดทั้งกลุ่ม · `status_game` = เปิด / ปิดทีละเกม — เกมเล่นได้เมื่อทั้งสองเป็น `true`

### 2.4 เมนูสิทธิ์ (`permissions`)

| key | เมนู | ระดับ |
|---|---|---|
| `dashboard` | หน้าแรก (1.1 ภาพรวม) | off / view |
| `account` | บัญชี (1.2 ประวัติของฉัน · 1.3 ข้อมูลรับรอง API) | off / view / edit |
| `member` | การจัดการสมาชิก | off / view / edit |
| `pt` | ถือสู้ (PT) | off / view / edit |
| `report` | รายงาน | off / view |
| `bet_cancel` | ยกเลิกเดิมพัน | off / view / edit |
| `payment` | ฝาก-ถอน | off / view / edit |
| `asset` | การจัดการ Asset | off / view / edit |
| `announcement` | ประกาศ / เนื้อหา (Company / Share / Agent) | off / view / edit |
| `rate` | อัตราแพ้ชนะ (Superadmin เท่านั้น) | off / view / edit |

ระดับในตาราง = ระดับสูงสุดที่เจ้าของให้ sub ได้ · บัญชีหลักได้ `edit` ทุกเมนู · ใช้ซ่อน / แสดงเมนูในหน้าบ้านได้ · หลังบ้านเช็คซ้ำทุกเส้น (ไม่มีสิทธิ์ = `402303`)

### 2.5 Error

`401202` · `401203` · `401301` · `401302` → หน้า login · `401304` / `401306` / `401307` → พาไปทำด่านหลัง login ([auth.md](auth.md) หัวข้อ 2)

## 3. ข้อมูลรับรอง API (✅ ใช้ได้)

หน้า 1.3 · เข้าได้เฉพาะ **Company Seamless 1 to 1 · Share Master · Share Reseller** และ sub ที่ได้สิทธิ์ `account`
· บัญชีอื่นได้ `403301`
· Key สร้างให้อัตโนมัติตอนเปิดหน้านี้ครั้งแรก หน้าบ้านไม่ต้องส่ง (สร้างพร้อมบัญชีเจ้าของจะตามมาภายหลัง) · Key เปลี่ยนไม่ได้

### 3.1 ดู — `GET /api/v1/bo/pr/account/api-credential`

```
curl "{{MG_URL}}/api/v1/bo/pr/account/api-credential" -H "Authorization: Bearer {{TOKEN}}"
```

```json
{
  "username": "shareresel01",
  "key": "cb886b34f28b4e7964ca039335a86c74d04a3dea605e1b2c9d0f7a8b6c5d4e3f",
  "callback_url": "https://api.customer.example/minigame",
  "allowed_ips": ["203.0.113.10/32", "198.51.100.0/24"]
}
```

- `key` = hex 64 ตัว · สร้างให้อัตโนมัติ · สร้างใหม่ไม่ได้ · ดูซ้ำได้ (ปุ่มคัดลอก)
- ยังไม่ตั้งลิงก์ = `""` · ไม่มี IP = `[]`
- response มี header `Cache-Control: no-store`
- sub ต้องมีสิทธิ์ `account` = `view` ขึ้นไป

### 3.2 อัปเดต — `POST /api/v1/bo/pr/account/update-credential`

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/account/update-credential" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"callback_url":"https://api.customer.example/minigame","allowed_ips":["203.0.113.10","198.51.100.0/24"],"passcode":"123456"}'
```

| field | กติกา |
|---|---|
| `callback_url` | บังคับส่ง · `https://` ที่มี host · ไม่เกิน 500 ตัว · ไม่ตั้ง = `""` |
| `allowed_ips` | บังคับส่ง · IPv4 หรือ CIDR ของ IPv4 · ไม่เกิน 50 รายการ · ห้ามซ้ำ · ไม่มี = `[]` (= ลูกค้าเรียก API ไม่ได้) |
| `passcode` | passcode ของคนที่กดบันทึก |

- **แทนทั้งชุด** — IP เดิมที่ไม่อยู่ในรายการใหม่ถูกลบ
- IP เดี่ยว `1.2.3.4` เก็บเป็น `1.2.3.4/32` · CIDR ต้องเป็นที่อยู่เครือข่าย (`1.2.3.0/24` ได้ · `1.2.3.4/24` ได้ `422`)
- Response: `{ "code": 200, "msg": "สำเร็จ" }`
- sub ต้องมีสิทธิ์ `account` = `edit`

Error: `422` (msg บอก field เช่น `allowed_ips[1] ซ้ำกับรายการก่อนหน้า`) · `403301` · `401204` (passcode ผิด) · `401205` · `402303`

## 4. Dashboard (⏳ contract ล่วงหน้า)

`GET /api/v1/bo/pr/account/dashboard?year=2026&month=10&currency=THB&game=coin_toss`

| query | กติกา |
|---|---|
| `year` | บังคับ |
| `month` | 1–12 · ไม่ส่ง = ทั้งปี |
| `currency` | บังคับ · ต้องเป็นสกุลที่บัญชีมี |
| `game` | ไม่ส่ง = ทุกเกม |

- ข้อมูลสรุปรายวัน อัปเดตวันละครั้ง 00:15–01:00 น.
- การ์ด: สมาชิกที่กำลังใช้งาน / สมัครใหม่ / ทั้งหมด (จำนวนคน) · ยอดเล่นทั้งหมด / ถูกรางวัล (จำนวนครั้ง) · ยอดเล่นจริง · ยอดตาม PT · Commission ที่ได้ / ที่จ่าย · รายได้ (เงิน) · แต่ละการ์ดมี `change_pct` (ไม่มีช่วงก่อนหน้า = `0`)
- Top 10: ยอดเล่น / ยอดถูก / ยอดเสีย สูงสุด
- sub ต้องมีสิทธิ์ `dashboard` = `view`

รายละเอียด response ดู `docs/modules/account.md` หัวข้อ 5 · จะส่งเอกสารเต็มเมื่อเส้นพร้อม

## 5. Error codes ที่เพิ่มจากเอกสาร Auth

| Code | HTTP | ความหมาย | หน้าบ้านควรทำ |
|---|---|---|---|
| 401311 | 403 | บัญชีถูกระงับ ใช้งานได้เฉพาะหน้าประวัติของฉันและรายงาน | แสดง msg · พาไปหน้า Profile |
| 402303 | 200 | ไม่มีสิทธิ์ใช้งานเมนูนี้ (sub) | แสดง msg / ซ่อนเมนู |
| 403301 | 200 | บัญชีนี้ไม่มีข้อมูลรับรอง API | ซ่อนเมนู 1.3 |
