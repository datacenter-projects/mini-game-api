# ADMIN (`admin_management`) — Spec

- สถานะ: **APPROVED** — กฎอนุมัติพร้อม spec `agent_auth` phase 2 (AUTH-43 – AUTH-52 · lead (zerph) 2026-10-05 / 2026-10-08) และ `agent_management` (MGMT-27B · 2026-10-08) · แยก module และเอกสาร 2026-10-09 (boiledegg)
- ผู้ดูแล: boiledegg
- ชื่อ module ในโค้ด: `admin_management` (`controllers/` `dto/` `middleware/` `service/admin_management` · `repository/postgres/admin_management`)
- rule ID ใช้เลขเดิม (`AUTH-43` – `AUTH-52` จาก [agent_auth_phase2.md](agent_auth_phase2.md) · `MGMT-27B` จาก [agent_management.md](agent_management.md)) — โค้ดและ test อ้างเลขเดิม
- error ใช้ `4013xx` / `4014xx` ของ module `agent_auth` (bb=01) · ไม่มี module id ใหม่
- path ของ API ไม่เปลี่ยน (`/api/v1/bo/pr/admin/...`)

> เอกสารนี้คือ source of truth ของเส้น ADMIN — กฎเต็มของการรีเซ็ตรหัสอยู่ใน agent_auth_phase2.md และอ้างด้วย ID

## 1. หน้าที่ของ module

งานของบัญชี role `ADMIN` (ทีม support): รีเซ็ต passcode / รหัสผ่านให้บัญชีอื่น · ค้นหาบัญชีทั้งระบบด้วย username ·
และ script บนเซิร์ฟเวอร์สำหรับกู้รหัสของ SUPERADMIN / ADMIN

**แบ่งหน้าที่กับ `agent_auth`:** module นี้ตัดสินว่า **ใครรีเซ็ตใครได้** (AUTH-44, AUTH-45, AUTH-51) ·
การตั้งค่าชั่วคราว, บังคับเปลี่ยน, audit log และตัด session เป็นของ `agent_auth` (`agentAuthService.ResetCredentialsService` — AUTH-46 – AUTH-49)

**ไม่อยู่ใน module นี้:** login / ด่านหลัง login / เปลี่ยนรหัสของตัวเอง (agent_auth) · การสร้าง ADMIN (`scripts/create_admin` — AUTH-43)

## 2. Business rules

| ID | สรุป |
|---|---|
| AUTH-43 | ADMIN อยู่ใน `user_agents` role `ADMIN` ไม่มี parent · สร้าง sub ไม่ได้ · ไม่ขึ้นในสายงาน / รายชื่อ / รายงาน · สร้างด้วย `scripts/create_admin` เท่านั้น |
| AUTH-44 | เส้น `/admin/*` เฉพาะบัญชีหลัก role `ADMIN` (`adminMw.RequireAdmin()`) · role อื่นรวม SUPERADMIN และ sub = `401308` · เส้นรีเซ็ตต้องส่ง `passcode` ของ ADMIN เอง (`RequirePasscode`) |
| AUTH-45 | ระบุเป้าหมายด้วย `username` · ไม่พบ `401404` · รีเซ็ตได้เฉพาะ Company / Share / Agent และ sub · SUPERADMIN, ADMIN และตัวเอง `401406` · เป้าหมาย `LOCKED` (รวมหัวสายถูกล็อก) `401407` |
| AUTH-46 | ค่าชั่วคราวสุ่มด้วย `crypto/rand` แสดงครั้งเดียว · หมดอายุ 24 ชม. · response `Cache-Control: no-store` · ห้าม log body · รหัสผ่าน 12 ตัว (ไม่มี `0 O o 1 l I`) · passcode 6 หลัก |
| AUTH-47 | รีเซ็ต passcode: เป้าหมายยังไม่เคยตั้ง `401405` · `must_change_passcode = true` · ล้างบล็อก passcode · ตัด session |
| AUTH-48 | รีเซ็ตรหัสผ่าน: `must_change_password = true` · ล้างบล็อก login · ตัด session |
| AUTH-49 | บันทึก `auth_audit_logs` `RESET_PASSCODE` / `RESET_PASSWORD` ใน transaction เดียวกับการรีเซ็ต (actor = ADMIN หรือ `SCRIPT`) |
| AUTH-50 | ถูกรีเซ็ตทั้งสองอย่าง → เปลี่ยนรหัสผ่านก่อน แล้วเปลี่ยน passcode (token เดิม) |
| AUTH-51 | SUPERADMIN / ADMIN รีเซ็ตผ่าน API ไม่ได้ → `scripts/reset_credentials` (เฉพาะ SUPERADMIN / ADMIN · `-password` / `-passcode` · กู้ได้แม้ถูกล็อก · actor = `SCRIPT`) |
| AUTH-52 | ผู้สร้างรีเซ็ตให้ sub ของตัวเองไม่ได้ |
| MGMT-27B | ADMIN ค้นบัญชี**ทั้งระบบ** ทุกประเภท (รวม SUPERADMIN, ADMIN, sub, Member) ด้วย username **ตรงทั้งคำ** ไม่สนตัวพิมพ์ · ผล 0–2 แถว ไม่มี page · ข้อมูลระบุตัวตนเท่านั้น (ไม่มียอดเงิน / PT / id) · `status` = สถานะที่ใช้งานจริง |

## 3. Endpoints

ใต้ `/api/v1/bo/pr` · ทุกเส้น `Authenticated` + `PassedGates()` + `adminMw.RequireAdmin()`
error ร่วม: `401202`, `401203`, `401301`, `401302`, `401304`, `401306`, `401307`, `401308`, `401311`

| Method | Path | middleware เพิ่ม | ใช้ทำอะไร |
|---|---|---|---|
| POST | `/api/v1/bo/pr/admin/passcode/reset` | `RequirePasscode` | รีเซ็ต passcode ให้บัญชีอื่น |
| POST | `/api/v1/bo/pr/admin/password/reset` | `RequirePasscode` | รีเซ็ตรหัสผ่านให้บัญชีอื่น |
| POST | `/api/v1/bo/pr/admin/accounts/search` | — | ค้นหาบัญชีทั้งระบบ (MGMT-27B) |

### POST /api/v1/bo/pr/admin/passcode/reset (รีเซ็ต passcode)

Header `Authorization: Bearer <token ของ admin>`

Request:
```json
{ "username": "agent01@staff", "passcode": "123456" }
```
`passcode` = passcode ของ admin เอง

Response `data` (แสดงครั้งเดียว · header `Cache-Control: no-store`):
```json
{
  "username": "agent01@staff",
  "temp_passcode": "482913",
  "temp_expires_at": "2026-10-06T10:00:00+07:00"
}
```

Error codes: `422`, `401308`, `401404`, `401405`, `401406`, `401407`, `401204`, `401205`, `401304`, `401306`,
`401307`

### POST /api/v1/bo/pr/admin/password/reset (รีเซ็ตรหัสผ่าน)

Header `Authorization: Bearer <token ของ admin>`

Request:
```json
{ "username": "agent01", "passcode": "123456" }
```
`passcode` = passcode ของ admin เอง

Response `data` (แสดงครั้งเดียว · header `Cache-Control: no-store`):
```json
{
  "username": "agent01",
  "temp_password": "Xk7mPq4RtW9z",
  "temp_expires_at": "2026-10-06T10:00:00+07:00"
}
```

Error codes: `422`, `401308`, `401404`, `401406`, `401407`, `401204`, `401205`, `401304`, `401306`, `401307`

### POST /api/v1/bo/pr/admin/accounts/search (ADMIN ค้นหาบัญชีทั้งระบบ)

สิทธิ์: ADMIN เท่านั้น (ไม่ใช่ = `401308`) · MGMT-27B

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

Error codes: `422`, `401308`

## 4. Scripts (ไม่ใช่ API)

| script | ใช้ทำอะไร | กฎ |
|---|---|---|
| `scripts/create_admin -username <ชื่อ>` | สร้างบัญชี ADMIN (รหัสผ่านอ่านจาก stdin) | AUTH-43 |
| `scripts/reset_credentials -username <ชื่อ> [-password] [-passcode]` | กู้รหัสของ SUPERADMIN / ADMIN · แสดงค่าชั่วคราวบนจอครั้งเดียว | AUTH-51 |

รันกับ DB บน server ให้ตั้ง `APP_ENV` ที่ไม่ใช่ `local` (ไม่ให้ log SQL พร้อมค่า)

## 5. Schema

ไม่มีตารางของตัวเอง · ใช้ `user_agents` · `subaccounts` · `user_members` (อ่าน) และ `auth_audit_logs` (ผ่าน agent_auth)

## 6. Test cases

test อยู่ใน `admin_management_reset_integration_test.go` · `admin_management_search_integration_test.go` · `admin_management_script_reset_integration_test.go`
ตาราง test ของ AUTH-44 – AUTH-51 อยู่ใน agent_auth_phase2.md หัวข้อ 7 · MGMT-27B อยู่ใน agent_management.md หัวข้อ 7

## 7. Error codes

| Code | HTTP | ความหมาย |
|---|---|---|
| 422 | 200 | ข้อมูลไม่ผ่าน (msg บอก field) |
| 401204 | 200 | passcode ของ ADMIN ไม่ถูกต้อง |
| 401205 | 401 | passcode ผิดครบ 5 ครั้ง session ถูกตัด |
| 401308 | 403 | ไม่มีสิทธิ์ใช้งานส่วนของ admin |
| 401404 | 200 | ไม่พบบัญชีที่ต้องการรีเซ็ต |
| 401405 | 200 | บัญชีนี้ยังไม่ได้ตั้ง passcode |
| 401406 | 200 | ไม่สามารถรีเซ็ตบัญชีนี้ได้ |
| 401407 | 200 | บัญชีเป้าหมายถูกล็อก |
