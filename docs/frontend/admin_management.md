# API หลังบ้าน — ADMIN (สำหรับหน้าบ้าน) — 9 ต.ค. 2026

เอกสารนี้สรุปจากโค้ดใน branch `boiledegg/bo/admin` (แยก module `admin_management` 2026-10-09 — **path, request และ response ไม่เปลี่ยน**) ·
spec เต็ม: `docs/modules/admin_management.md` · login / session / ด่านหลัง login / ตั้งค่า Postman ดู [auth.md](auth.md)

## 1. ภาพรวม

เส้นในเอกสารนี้ใช้ได้เฉพาะบัญชี role **`ADMIN`** (บัญชีหลัก) — role อื่นรวม Superadmin และบัญชีย่อยได้ `401308`

| # | Method | Path | ใช้เมื่อ |
|---|---|---|---|
| 2.1 | POST | `/api/v1/bo/pr/admin/passcode/reset` | รีเซ็ต passcode ให้บัญชีอื่น (ต้องใส่ passcode ของ ADMIN) |
| 2.2 | POST | `/api/v1/bo/pr/admin/password/reset` | รีเซ็ตรหัสผ่านให้บัญชีอื่น (ต้องใส่ passcode ของ ADMIN) |
| 2.3 | POST | `/api/v1/bo/pr/admin/accounts/search` | ค้นหาบัญชีทั้งระบบด้วย username ตรงทั้งคำ |

- รีเซ็ตได้เฉพาะ Company / Share / Agent และบัญชีย่อย · Superadmin, ADMIN และตัวเองรีเซ็ตไม่ได้ (`401406`)
- ใช้คู่กัน: ค้นหาบัญชีด้วย 2.3 → ส่ง `username` ที่ได้ไปรีเซ็ตด้วย 2.1 / 2.2

**error ที่เกิดได้กับทุกเส้น:** `401202`, `401203`, `401301`, `401302` → หน้า login · `401304`, `401306`, `401307` → ด่านหลัง login · `401308` ไม่ใช่ ADMIN · `401311` บัญชีถูกระงับ

## 2. Endpoints

### 2.1 POST /api/v1/bo/pr/admin/passcode/reset (เฉพาะ ADMIN)

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

### 2.2 POST /api/v1/bo/pr/admin/password/reset (เฉพาะ ADMIN)

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

### 2.3 POST /api/v1/bo/pr/admin/accounts/search (ADMIN เท่านั้น)

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/admin/accounts/search" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"username":"Share01"}'
```

Request:
```json
{ "username": "Share01" }
```
Response `data` (0–2 แถว · ไม่พบ = `[]`):
```json
[
  {
    "username": "share01",
    "role": "SHAREHOLDER",
    "user_type": "SHARE_B2C",
    "is_subaccount": false,
    "status": "ACTIVE",
    "parent_username": "comp01",
    "created_at": "2026-10-01T09:00:00+07:00",
    "last_login_at": "2026-10-08T10:00:00+07:00",
    "last_login_ip": "203.0.113.10"
  }
]
```

- ค้นได้ทุกบัญชีในระบบ: Superadmin, ADMIN, Company / Share / Agent, บัญชีย่อย, Member
- username ต้อง**ตรงทั้งคำ** (ไม่สนตัวพิมพ์) — พิมพ์บางส่วนได้ `[]` · ไม่มี `page` / `limit`
- ได้ 2 แถวเมื่อ username ตรงทั้งบัญชีฝั่ง agent และ Member · บัญชีย่อย `role` / `user_type` = ของเจ้าของ และ `parent_username` = เจ้าของ
- `status` = สถานะที่ใช้งานจริง (หัวสายถูกระงับ = `SUSPENDED`) · ยังไม่เคย login: `last_login_at` / `last_login_ip` = `""`
- ใช้คู่กับรีเซ็ตรหัส (หัวข้อ 2.1 / 2.2) — ส่ง `username` ที่ได้ไปรีเซ็ตต่อ

Error: `422`, `401308` (ไม่ใช่ ADMIN)

## 3. Error codes

| Code | HTTP | ความหมาย | หน้าบ้านควรทำ |
|---|---|---|---|
| 422 | 200 | ข้อมูลไม่ผ่าน (msg บอก field) | แสดง msg |
| 401204 | 200 | passcode ของ ADMIN ไม่ถูกต้อง (msg บอกจำนวนครั้งที่เหลือ) | ให้กรอกใหม่ |
| 401205 | 401 | passcode ผิดครบ 5 ครั้ง session ถูกตัด | หน้า login |
| 401308 | 403 | ไม่มีสิทธิ์ใช้งานส่วนของ admin | ซ่อนเมนู admin |
| 401404 | 200 | ไม่พบบัญชีที่ต้องการรีเซ็ต | แสดง msg |
| 401405 | 200 | บัญชีนี้ยังไม่ได้ตั้ง passcode | แสดง msg |
| 401406 | 200 | ไม่สามารถรีเซ็ตบัญชีนี้ได้ | แสดง msg |
| 401407 | 200 | บัญชีเป้าหมายถูกล็อก | แสดง msg |
