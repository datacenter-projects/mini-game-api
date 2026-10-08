# API หลังบ้าน — Auth (สำหรับหน้าบ้าน) — 8 ต.ค. 2026

เอกสารนี้สรุปจากโค้ดใน branch `boiledegg/bo/account` (`dev` + AUTH-54 บัญชีถูกระงับ — ยังไม่ merge เข้า `dev`) ·
spec เต็ม: `docs/modules/agent_auth.md`, `docs/modules/agent_auth_phase2.md` · เส้นอื่นดู `account.md`, `agent_management.md`

## 1. พื้นฐาน

### 1.1 Base URL และ header

| เรื่อง | ค่า |
|---|---|
| prefix | `/api/v1/bo` |
| เส้นไม่ต้อง login | `/api/v1/bo/pb/...` |
| เส้นต้อง login | `/api/v1/bo/pr/...` + header `Authorization: Bearer <token>` |
| body | JSON (`Content-Type: application/json`) |
| ภาษาของ `msg` | header `X-Lang: en` หรือ `Accept-Language: en` → อังกฤษ · ไม่ส่ง = ไทย |

### 1.2 รูปแบบ response

ทุกเส้นตอบรูปแบบเดียวกัน:

```json
{ "code": 200, "msg": "สำเร็จ", "data": { } }
```

- `code = 200` = สำเร็จ · อย่างอื่น = error (ดูหัวข้อ 4)
- เส้นที่ไม่มีข้อมูลส่งกลับ **ไม่มี key `data`**
- error ส่วนใหญ่ตอบ HTTP 200 และดูที่ `code` · ยกเว้น error ของการ login / session / สิทธิ์ ที่ตอบ HTTP 401 / 403 (ดูคอลัมน์ HTTP ในหัวข้อ 4)
- `422` = ข้อมูลไม่ผ่าน · `msg` บอกว่าผิดที่ field ไหน

### 1.3 ประเภทบัญชีที่ login หลังบ้านได้

| บัญชี | username | `role` |
|---|---|---|
| Superadmin | `admin01` | `SUPERADMIN` |
| ADMIN (ดูแลระบบ — รีเซ็ตรหัสให้ผู้อื่น) | `support01` | `ADMIN` |
| Company / Share / Agent | `agent01` | `COMPANY` / `SHAREHOLDER` / `AGENT` |
| บัญชีย่อย (sub) | `agent01@staff` (`{ผู้สร้าง}@{ชื่อ}`) | role ของผู้สร้าง · `is_subaccount = true` |

username ไม่สนตัวพิมพ์ใหญ่ / เล็ก และตัดช่องว่างหัวท้ายให้

## 2. Flow หลัง login (สำคัญ)

login แล้วใช้งานได้ทันที แต่ถ้าบัญชีมีเงื่อนไขค้าง ต้องทำให้เสร็จ**ตามลำดับ**ก่อนเรียกเส้นอื่น
หน้าบ้านดูจาก response ของ login แล้วพาไปหน้าที่ถูก:

| ลำดับ | เงื่อนไข (จาก login) | หน้าที่ต้องไป | เรียกได้แค่ | เรียกเส้นอื่น |
|---|---|---|---|---|
| 1 | `must_change_password = true` | เปลี่ยนรหัสผ่าน | `POST /pr/auth/password/change` (ไม่ต้องส่ง `passcode`) | `401306` |
| 2 | `must_change_passcode = true` | เปลี่ยน passcode | `POST /pr/auth/passcode/change` | `401307` |
| 3 | `passcode_set = false` | ตั้ง passcode ครั้งแรก | `POST /pr/auth/passcode/setup` | `401304` |
| — | ผ่านครบ | ใช้งานปกติ | ทุกเส้นตามสิทธิ์ | — |

- `logout` เรียกได้เสมอ
- ทำขั้นหนึ่งเสร็จแล้วใช้ **token เดิม** ทำขั้นต่อไปได้เลย ไม่ต้อง login ใหม่
- ถูก admin รีเซ็ตทั้งรหัสผ่านและ passcode → เปลี่ยนรหัสผ่านก่อน แล้วค่อยเปลี่ยน passcode
- บัญชีถูกระงับ (หัวข้อ 2.1) ทำด่านข้างบนไม่ได้ (`401311`) และเส้นอื่นยังติดด่านเหมือนเดิม — ต้องรอชั้นบนยกเลิกระงับก่อน

```
login → must_change_password? → password/change
      → must_change_passcode? → passcode/change
      → passcode_set = false? → passcode/setup
      → ใช้งานปกติ
```

### 2.1 บัญชีถูกระงับ (`SUSPENDED`)

บัญชีถูกระงับ หรือหัวสายด้านบนถูกระงับ (sub = เจ้าของถูกระงับ) **login ได้** แต่ใช้งานได้จำกัด:

| เรียกได้ | เรียกไม่ได้ (`401311`) |
|---|---|
| `logout` · `GET /pr/account/profile` · รายงาน (ดูอย่างเดียว — ยังไม่มีเส้น) | ทุกเส้นอื่น **รวม** ตั้ง / เปลี่ยน passcode และเปลี่ยนรหัสผ่านของตัวเอง |

- ได้ `401311` → แสดง msg แล้วพาไปหน้า Profile
- ถูกระงับระหว่างใช้งาน → request ถัดไปได้ `401311` (ไม่ถูกตัด session) · ถูก**ล็อก**ระหว่างใช้งาน → `401301` / `401302` session หลุด
- ค่า `status` ใน Profile บอกว่าบัญชีอยู่ในสถานะไหน (`ACTIVE` / `SUSPENDED`)

## 3. Endpoints

| # | Method | Path | ใช้เมื่อ |
|---|---|---|---|
| 3.1 | POST | `/api/v1/bo/pb/auth/login` | เข้าสู่ระบบ |
| 3.2 | POST | `/api/v1/bo/pr/auth/logout` | ออกจากระบบ |
| 3.3 | POST | `/api/v1/bo/pr/auth/passcode/setup` | ตั้ง passcode ครั้งแรก |
| 3.4 | POST | `/api/v1/bo/pr/auth/passcode/change` | เปลี่ยน passcode |
| 3.5 | POST | `/api/v1/bo/pr/auth/password/change` | เปลี่ยนรหัสผ่าน |
| 3.6 | POST | `/api/v1/bo/pr/admin/passcode/reset` | ADMIN รีเซ็ต passcode ให้บัญชีอื่น |
| 3.7 | POST | `/api/v1/bo/pr/admin/password/reset` | ADMIN รีเซ็ตรหัสผ่านให้บัญชีอื่น |

**error ที่เกิดได้กับทุกเส้นใต้ `/pr`** (ไม่เขียนซ้ำในแต่ละเส้น): `401202`, `401203`, `401301`, `401302` → พาไปหน้า login ·
`401311` บัญชีถูกระงับ (ทุกเส้นยกเว้น logout และ Profile — หัวข้อ 2.1)

### ตั้งค่า Postman (ตัวแปร `{{MG_URL}}` และ `{{TOKEN}}`)

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

ขอข้อความ error เป็นภาษาอังกฤษ: เพิ่ม `-H "X-Lang: en"`

### 3.1 POST /api/v1/bo/pb/auth/login

```
curl -X POST "{{MG_URL}}/api/v1/bo/pb/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"agent01","password":"Zx9!kq2m"}'
```

Request:
```json
{ "username": "agent01", "password": "Zx9!kq2m" }
```

Response `data`:
```json
{
  "token": "<jwt>",
  "username": "agent01",
  "role": "AGENT",
  "is_subaccount": false,
  "passcode_set": true,
  "must_change_password": false,
  "must_change_passcode": false,
  "expires_at": "2026-10-06T22:00:00+07:00"
}
```

| field | ความหมาย |
|---|---|
| `token` | ใส่ใน header `Authorization: Bearer <token>` ทุกเส้นใต้ `/pr` |
| `role` | role ของบัญชี (sub = role ของผู้สร้าง) |
| `is_subaccount` | เป็นบัญชีย่อยไหม |
| `passcode_set` / `must_change_password` / `must_change_passcode` | ใช้เลือกหน้าถัดไป (หัวข้อ 2) |
| `expires_at` | เวลาที่ session หมดแน่นอน (12 ชม. นับจาก login) |

Error: `422`, `401201`, `401301`, `401302`, `401303`, `401309`, `401310`, `429`, `500`

### 3.2 POST /api/v1/bo/pr/auth/logout

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/auth/logout" \
  -H "Authorization: Bearer {{TOKEN}}"
```

Header `Authorization: Bearer <token>` · ไม่มี body · Response: `{ "code": 200, "msg": "สำเร็จ" }`

- เรียกซ้ำ หรือ session หลุดไปแล้ว ก็ตอบสำเร็จ
- Error: `401202` (token ปลอม / หมดอายุเกิน 12 ชม.)

### 3.3 POST /api/v1/bo/pr/auth/passcode/setup

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/auth/passcode/setup" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"passcode":"123456","confirm_passcode":"123456"}'
```

Request:
```json
{ "passcode": "123456", "confirm_passcode": "123456" }
```
Response: `{ "code": 200, "msg": "สำเร็จ" }`

Error: `422`, `401401` (ตั้งไว้แล้ว), `401306`, `401307`, `401311`

### 3.4 POST /api/v1/bo/pr/auth/passcode/change

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/auth/passcode/change" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"old_passcode":"123456","new_passcode":"654321","confirm_passcode":"654321"}'
```

Request:
```json
{ "old_passcode": "123456", "new_passcode": "654321", "confirm_passcode": "654321" }
```
- ถูก admin รีเซ็ต: `old_passcode` = passcode ชั่วคราวที่ได้จาก admin

Response: `{ "code": 200, "msg": "สำเร็จ" }` · token เดิมใช้ต่อได้

Error: `422`, `401204`, `401205`, `401403`, `401310`, `401304`, `401306`, `401311`

### 3.5 POST /api/v1/bo/pr/auth/password/change

```
# ปกติ
curl -X POST "{{MG_URL}}/api/v1/bo/pr/auth/password/change" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"old_password":"Zx9!kq2m","new_password":"Nw8@pq3r","confirm_password":"Nw8@pq3r","passcode":"123456"}'

# ถูกบังคับเปลี่ยน (ไม่ต้องส่ง passcode)
curl -X POST "{{MG_URL}}/api/v1/bo/pr/auth/password/change" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"old_password":"Xk7mPq4RtW9z","new_password":"Nw8@pq3r","confirm_password":"Nw8@pq3r"}'
```

Request (ปกติ — ต้องส่ง `passcode` ของตัวเอง):
```json
{
  "old_password": "Zx9!kq2m",
  "new_password": "Nw8@pq3r",
  "confirm_password": "Nw8@pq3r",
  "passcode": "123456"
}
```
Request (ถูกบังคับเปลี่ยน `must_change_password = true` — ไม่ต้องส่ง `passcode`):
```json
{ "old_password": "<รหัสชั่วคราว>", "new_password": "Nw8@pq3r", "confirm_password": "Nw8@pq3r" }
```
Response: `{ "code": 200, "msg": "สำเร็จ" }` · token เดิมใช้ต่อได้

Error: `422`, `401204`, `401205`, `401206`, `401303`, `401402`, `401310`, `401304`, `401307`, `401311`

### 3.6 POST /api/v1/bo/pr/admin/passcode/reset (เฉพาะ ADMIN)

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/admin/passcode/reset" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"username":"agent01@staff","passcode":"123456"}'
```

Request (`passcode` = passcode ของ **ADMIN เอง**):
```json
{ "username": "agent01@staff", "passcode": "123456" }
```
Response `data` (**แสดงครั้งเดียว** — ปิดหน้าแล้วดูซ้ำไม่ได้):
```json
{ "username": "agent01@staff", "temp_passcode": "482913", "temp_expires_at": "2026-10-07T10:00:00+07:00" }
```
Error: `422`, `401308`, `401404`, `401405`, `401406`, `401407`, `401204`, `401205`, `401304`, `401306`, `401307`, `401311`

### 3.7 POST /api/v1/bo/pr/admin/password/reset (เฉพาะ ADMIN)

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/admin/password/reset" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"username":"agent01","passcode":"123456"}'
```

Request (`passcode` = passcode ของ **ADMIN เอง**):
```json
{ "username": "agent01", "passcode": "123456" }
```
Response `data` (**แสดงครั้งเดียว**):
```json
{ "username": "agent01", "temp_password": "Xk7mPq4RtW9z", "temp_expires_at": "2026-10-07T10:00:00+07:00" }
```
Error: `422`, `401308`, `401404`, `401406`, `401407`, `401204`, `401205`, `401304`, `401306`, `401307`, `401311`

- รีเซ็ตได้เฉพาะ Company / Share / Agent และ sub · Superadmin, ADMIN และตัวเองรีเซ็ตไม่ได้ (`401406`)
- ค่าชั่วคราวหมดอายุใน 24 ชม. · ผู้ถูกรีเซ็ต login แล้วจะได้ `must_change_password` / `must_change_passcode = true`

## 4. Error codes

| Code | HTTP | ความหมาย | หน้าบ้านควรทำ |
|---|---|---|---|
| 422 | 200 | ข้อมูลไม่ผ่าน (msg บอก field) | แสดง msg ที่ช่องนั้น |
| 429 | 429 | เรียก login ถี่เกิน (เกิน 20 ครั้ง/นาที ต่อ IP) | ให้รอแล้วลองใหม่ |
| 500 | 500 | ระบบขัดข้อง | แสดงข้อความทั่วไป |
| 401201 | 401 | username หรือรหัสผ่านไม่ถูกต้อง | แสดง msg |
| 401202 | 401 | ไม่พบ token / token ไม่ถูกต้องหรือหมดอายุ | ล้าง token → หน้า login |
| 401203 | 401 | session สิ้นสุดแล้ว (logout, idle เกิน 60 นาที, ครบ 12 ชม., login ที่อื่น) | ล้าง token → หน้า login |
| 401204 | 200 | passcode ไม่ถูกต้อง (msg บอกจำนวนครั้งที่เหลือ) | แสดง msg ให้กรอกใหม่ |
| 401205 | 401 | passcode ผิดครบ 5 ครั้ง session ถูกตัด และระงับ 1 ชม. | ล้าง token → หน้า login พร้อม msg |
| 401206 | 200 | รหัสผ่านเดิมไม่ถูกต้อง | แสดง msg |
| 401301 | 403 | บัญชีถูกล็อก | หน้า login พร้อม msg |
| 401302 | 403 | สายบัญชีต้นสังกัดถูกล็อก | หน้า login พร้อม msg |
| 401303 | 403 | login ผิดหลายครั้ง ระงับชั่วคราว 15 นาที | แสดง msg |
| 401304 | 403 | ต้องตั้ง passcode ก่อน | พาไปหน้าตั้ง passcode |
| 401306 | 403 | ต้องเปลี่ยนรหัสผ่านก่อน | พาไปหน้าเปลี่ยนรหัสผ่าน |
| 401307 | 403 | ต้องเปลี่ยน passcode ก่อน | พาไปหน้าเปลี่ยน passcode |
| 401308 | 403 | ไม่มีสิทธิ์ใช้งานส่วนของ admin | แสดง msg |
| 401309 | 403 | ระงับ login เพราะ passcode ผิดหลายครั้ง (1 ชม.) | แสดง msg |
| 401310 | 403 | ค่าชั่วคราวหมดอายุ กรุณาติดต่อ admin | แสดง msg |
| 401311 | 403 | บัญชีถูกระงับ (ตัวเองหรือหัวสาย) ใช้งานได้เฉพาะหน้าประวัติของฉันและรายงาน | แสดง msg · พาไปหน้า Profile |
| 401401 | 200 | ตั้ง passcode ไว้แล้ว | พาไปหน้าหลัก |
| 401402 | 200 | รหัสผ่านใหม่ซ้ำกับรหัสที่เคยใช้ | แสดง msg |
| 401403 | 200 | passcode ใหม่ซ้ำกับตัวเดิม | แสดง msg |
| 401404 | 200 | ไม่พบบัญชีที่ต้องการรีเซ็ต | แสดง msg |
| 401405 | 200 | บัญชีนี้ยังไม่ได้ตั้ง passcode | แสดง msg |
| 401406 | 200 | ไม่สามารถรีเซ็ตบัญชีนี้ได้ | แสดง msg |
| 401407 | 200 | บัญชีเป้าหมายถูกล็อก | แสดง msg |

## 5. กติกาที่หน้าบ้านควรเช็คในฟอร์ม (หลังบ้านเช็คซ้ำเสมอ)

### 5.1 รหัสผ่าน

- ยาว **8–64** ตัว
- ใช้ได้เฉพาะ `a-z` `A-Z` `0-9` และเครื่องหมาย `` !"#$%&'()*+,-./:;<=>?@[\]^_`{|}~ ``
- ห้ามมีช่องว่าง · ห้ามอักษรไทย
- ต้องมีตัวอักษรอย่างน้อย 1 ตัว **และ** ตัวเลขอย่างน้อย 1 ตัว
- ห้ามตัวเดียวกันซ้ำติดกัน 4 ตัว (`aaaa`, `1111`)
- ห้ามซ้ำกับรหัสปัจจุบันและรหัสก่อนหน้า 1 รหัส (`401402`)
- `confirm_password` ต้องตรงกับ `new_password`

ตัวอย่าง: ผ่าน `Zx9!kq2m` · `abcd1234` — ไม่ผ่าน `Zx9!kq2` (7 ตัว) · `Zxkqmwpt` (ไม่มีตัวเลข) · `Zx9aaaam` (ซ้ำ 4)

### 5.2 passcode

- ตัวเลข **6 หลัก** · ส่งเป็น **JSON string** เสมอ (`"012345"`) — ส่งเป็น number (`12345`) ได้ `422` เพราะเลข 0 นำหน้าจะหาย
- ห้ามซ้ำกับตัวปัจจุบัน (`401403`)
- ส่งใน body เท่านั้น (ไม่ส่งทาง header / query)

### 5.3 การนับผิด

| เรื่อง | กติกา |
|---|---|
| login / `old_password` ผิด | 5 ครั้งใน 15 นาที → login ไม่ได้ 15 นาที (`401303`) |
| passcode ผิด (ทุกที่รวมกัน) | ครั้งที่ 1–4 → `401204` (msg บอกจำนวนที่เหลือ) · ครั้งที่ 5 → `401205` session ถูกตัด login ไม่ได้ 1 ชม. (`401309`) |
| ทำสำเร็จ | ล้างตัวนับ |

## 6. Session

- 1 บัญชีมีได้ 1 session — login ที่เครื่องใหม่ เครื่องเดิมได้ `401203`
- หลุดเมื่อไม่มี request เกิน **60 นาที** หรือครบ **12 ชม.** นับจาก login (`expires_at`)
- ทุก request ที่สำเร็จต่ออายุ idle อีก 60 นาที
- บัญชีหรือหัวสายถูกล็อกระหว่างใช้งาน → request ถัดไปได้ `401301` / `401302`
- บัญชีหรือหัวสายถูกระงับระหว่างใช้งาน → session ไม่หลุด แต่เส้นที่ไม่ได้รับอนุญาตได้ `401311` (หัวข้อ 2.1)

## 7. ยังไม่มีในรอบนี้

- รายงาน (Report) — `401311` ยกเว้นให้บัญชีถูกระงับดูได้ เมื่อมีเส้นรายงาน
- การรีเซ็ตรหัสผ่าน / passcode ด้วย script ของทีมหลังบ้าน (ไม่ใช่ API — รอ lead)
