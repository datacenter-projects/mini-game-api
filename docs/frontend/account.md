# API หลังบ้าน — บัญชี (Account) สำหรับหน้าบ้าน — 8 ต.ค. 2026

เอกสารนี้สรุปจาก spec `docs/modules/account.md` (APPROVED) และโค้ดใน branch `dev` (merge 8 ต.ค. 2026 — มีบน server dev หลัง deploy รอบถัดไป)
· พื้นฐาน (base URL, header, รูปแบบ response, login, session) ดู [auth.md](auth.md)

## 0. สถานะของแต่ละเส้น

| เส้น | สถานะ | หมายเหตุ |
|---|---|---|
| `GET /api/v1/bo/pr/account/profile` | ✅ ใช้ได้ | ค่าจริงทุก field (หัวข้อ 2.2) |
| `GET /api/v1/bo/pr/account/api-credential` | ✅ ใช้ได้ | |
| `POST /api/v1/bo/pr/account/update-credential` | ✅ ใช้ได้ | |
| `GET /api/v1/bo/pr/account/dashboard` | ⏸ ยังไม่ทำ | lead ตัดออกจาก scope รอบนี้ (9 ต.ค.) |

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

- เงินและ % = JSON number ทศนิยมไม่เกิน 4 ตำแหน่งตามที่เก็บ (ตัด 0 ท้าย) เช่น `962056.5` · **หน้าบ้านปัดเป็น 3 ตำแหน่งตอนแสดงเอง** (แก้ 9 ต.ค.)
- % = JSON number เช่น `20`, `0.5`

### 1.3 สถานะ `status`

key เดียว = สถานะที่ใช้งานจริง (รวมผลจากหัวสายแล้ว)

| `status` | ใช้งานได้ |
|---|---|
| `ACTIVE` | ปกติ |
| `SUSPENDED` | เข้าได้เฉพาะหน้า **Profile**, **ข้อมูลรับรอง API** (ดู — บันทึกไม่ได้) และ **รายงาน** แบบดูอย่างเดียว · เส้นอื่นได้ `401311` |
| `LOCKED` | เข้าไม่ได้ (`401301` / `401302`) |

ตัวอย่าง: หัวสายถูกระงับ → บัญชีนี้ได้ `status = SUSPENDED` แม้บัญชีตัวเองไม่ได้ถูกระงับ

### 1.4 ตั้งค่า Postman (ตัวแปร `{{MG_URL}}` และ `{{TOKEN}}`)

ทุกตัวอย่างใช้ตัวแปร Postman 2 ตัว — คัดลอก curl ไป **Import** ใน Postman ได้เลย (Postman แทนค่า `{{...}}` ให้)

| ตัวแปร | ค่า | ตั้งที่ |
|---|---|---|
| `MG_URL` | `https://dev-mini-api.pirate168.com` (server dev) · `http://localhost:8282` (test env ในเครื่อง) | ตัวแปรของ collection หรือ environment |
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
  "currencies": ["THB"],
  "balances": [
    { "currency": "THB", "amount": 5000.00 }
  ],
  "pt": {
    "minigame": { "pt_from_parent": 60, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true,  "created_at": "2026-10-01T09:00:00+07:00", "created_by": "share01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "agent01" }
  },
  "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": false },
  "permissions": {
    "dashboard": "edit", "member": "edit", "pt": "edit", "report": "edit",
    "bet_cancel": "edit", "payment": "edit", "asset": "edit", "announcement": "edit", "api_credential": "edit"
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
| `pt` | ค่าหุ้นส่วนชุดเดียวต่อระบบ (หัวข้อ 2.3) |
| `status_game` | เปิด / ปิดทีละเกม (หัวข้อ 2.3) |
| `permissions` | สิทธิ์ต่อเมนู `off` / `view` / `edit` (หัวข้อ 2.4) |

### 2.2 ที่มาของค่า (8 ต.ค. 2026 — เลิกใช้ค่าชั่วคราวแล้ว)

| field | ค่า |
|---|---|
| `user_type` | ประเภทจริง เช่น `COMPANY_TRANSFER`, `SHARE_RESELLER` · sub = ของผู้สร้าง |
| `currencies` | เฉพาะสกุลของบัญชี (Superadmin, Company Transfer, Company Seamless Reseller / Master = ครบ 27) |
| `balances` | 1 รายการต่อสกุลใน `currencies` · ยอดจริง · ยังไม่มียอด และบัญชี Seamless = `0.00` |
| `pt` | ค่าจริงจากตอนสร้าง / แก้ในการจัดการสมาชิก แสดงชุดเดียวต่อระบบ (หัวข้อ 2.3) |
| `permissions` | บัญชีหลัก = `edit` ทุกเมนูของประเภท · sub = ตามที่เจ้าของให้ (ไม่ได้ให้ = `off`) |
| ADMIN | `currencies` / `balances` = `[]` · `pt` / `permissions` = `{}` |

### 2.3 รูปแบบ `pt` และ `status_game` (แก้ 8 ต.ค. 2026 — `pt` ชุดเดียวต่อระบบ)

```json
"pt": {
  "minigame": { "pt_from_parent": 90, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true,  "created_at": "2026-10-01T09:00:00+07:00", "created_by": "comp01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "comp01" }
},
"status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": false }
```

| field | ความหมาย |
|---|---|
| `minigame` | ระบบ (ระบบอื่น เช่น `askmelotto` จะเพิ่มข้างๆ ภายหลัง) |
| `pt_from_parent` | PT ที่ได้รับจากผู้สร้าง |
| `force` · `remain_quota` · `commission_percent` | บังคับถือสู้ · เอาส่วนที่เหลือ · Commission |
| `status` | `true` = บัญชีนี้รับ PT ในระบบนั้น · `false` = ไม่รับ (ไม่ใช่สถานะเกม — เปิด/ปิดเกมดูที่ `status_game`) |
| `created_at` · `created_by` | เวลา / username ของผู้สร้างค่า PT (ไม่เปลี่ยนหลังสร้าง) |
| `updated_at` · `updated_by` | เวลา / username ของคนที่แก้ค่า PT ล่าสุด (รวม sub) |
| `status_game` | อยู่นอก `pt` · รหัสเกม `coin_toss` · `rock_paper_scissors` · `scratch_card` (หน้าบ้านแปลงเป็นชื่อแสดงเอง) → `true` เปิด / `false` ปิด (Member ในสายเล่นเกมนั้นไม่ได้) |

หลังบ้านเก็บค่า PT แยกต่อเกม แต่ส่ง / รับชุดเดียวต่อระบบ (ทุกเกมในระบบใช้ค่าเดียวกัน)

### 2.4 เมนูสิทธิ์ (`permissions`)

| key | เมนู | ระดับ |
|---|---|---|
| `dashboard` | หน้าแรก (1.1 ภาพรวม) | off / view |
| `member` | การจัดการสมาชิก | off / view / edit |
| `pt` | ถือสู้ (PT) | off / view / edit |
| `report` | รายงาน | off / view |
| `bet_cancel` | ยกเลิกเดิมพัน | off / view / edit |
| `payment` | ฝาก-ถอน | off / view / edit |
| `asset` | การจัดการ Asset | off / view / edit |
| `announcement` | ประกาศ / เนื้อหา (Company / Share / Agent) | off / view / edit |
| `rate` | อัตราแพ้ชนะ (Superadmin เท่านั้น) | off / view / edit |
| `api_credential` | 1.3 ข้อมูลรับรอง API (ทุกบัญชีหลัก · **ใหม่ 9 ต.ค.** · Superadmin มีเมนูแต่ไม่มี Key → `403301`) · view = ดู Key · edit = บันทึก | off / view / edit |

ระดับในตาราง = ระดับสูงสุดที่เจ้าของให้ sub ได้ · บัญชีหลักได้ `edit` ทุกเมนู · ใช้ซ่อน / แสดงเมนูในหน้าบ้านได้ · หลังบ้านเช็คซ้ำทุกเส้น (ไม่มีสิทธิ์ = `402303`)

### 2.5 Error

`401202` · `401203` · `401301` · `401302` → หน้า login · `401304` / `401306` / `401307` → พาไปทำด่านหลัง login ([auth.md](auth.md) หัวข้อ 2)

## 3. ข้อมูลรับรอง API (✅ ใช้ได้)

หน้า 1.3 · ทุกบัญชี **Company / Share / Agent** เปิดได้ · **sub ต้องมีสิทธิ์ `api_credential`** (แก้ 9 ต.ค.): `view` = ดู · `edit` = บันทึก · ไม่มี = `402303`
· Superadmin / ADMIN ได้ `403301`
· Key สร้างให้อัตโนมัติตอนสร้างบัญชี (บัญชีที่สร้างก่อนหน้านั้นได้ Key แล้วจาก migration) หน้าบ้านไม่ต้องส่ง · Key เปลี่ยนไม่ได้

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
- sub ที่มีสิทธิ์ `api_credential` ≥ `view` เห็น Key เต็มของเจ้าของ
- บัญชีถูกระงับ (ตัวเอง / หัวสาย / เจ้าของของ sub) ดูได้ แต่บันทึกได้ `401311`

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
- **ใหม่ 9 ต.ค.:** ห้ามช่วงกว้างกว่า `/16` (`0.0.0.0/0`, `8.0.0.0/8` → `422`) · ห้าม IP ภายใน / พิเศษ `0.0.0.0/8` · `10.0.0.0/8` · `100.64.0.0/10` · `127.0.0.0/8` · `169.254.0.0/16` · `172.16.0.0/12` · `192.168.0.0/16` → `422` · msg บอกรายการที่ผิด เช่น `allowed_ips[0] เป็น IP ภายใน / พิเศษ ใส่ไม่ได้ (…)`
- Response: `{ "code": 200, "msg": "สำเร็จ" }`
- sub บันทึกได้เมื่อมีสิทธิ์ `api_credential` = `edit` (ใช้ passcode ของ sub เอง)

Error: `422` (msg บอก field เช่น `allowed_ips[1] ซ้ำกับรายการก่อนหน้า`) · `402303` (sub ไม่มีสิทธิ์ `api_credential`) · `403301` (Superadmin / ADMIN) · `401204` (passcode ผิด) · `401205` · `401311` (บัญชีถูกระงับ)

## 4. Dashboard (⏸ ยังไม่ทำ)

lead ตัดออกจาก scope รอบนี้ (9 ต.ค.) · contract ร่างเก็บไว้ที่ `docs/modules/account.md` หัวข้อ 10

## 5. Error codes ที่เพิ่มจากเอกสาร Auth

| Code | HTTP | ความหมาย | หน้าบ้านควรทำ |
|---|---|---|---|
| 401311 | 403 | บัญชีถูกระงับ ใช้งานได้เฉพาะหน้าประวัติของฉัน ข้อมูลรับรอง API (ดูอย่างเดียว) และรายงาน | แสดง msg · พาไปหน้า Profile |
| 402303 | 200 | ไม่มีสิทธิ์ใช้งานเมนูนี้ (sub) | แสดง msg / ซ่อนเมนู |
| 403301 | 200 | บัญชีนี้ไม่มีข้อมูลรับรอง API (Superadmin / ADMIN) | ซ่อนเมนู 1.3 |
