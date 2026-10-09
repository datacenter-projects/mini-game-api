# Backoffice Auth (`agent_auth`) ระยะ 2 — Subaccount login · Passcode · Password · Admin reset — Spec

- สถานะ: **APPROVED** · `scripts/reset_credentials` อนุมัติ 2026-10-08 (lead — actor = `SCRIPT`) · AUTH-54 อนุมัติ 2026-10-06
- อนุมัติโดย: lead (zerph) · วันที่: 2026-10-05
- ชื่อ module ในโค้ด: `agent_auth` (ต่อจาก [agent_auth.md](agent_auth.md) — rule `AUTH-01`–`AUTH-16` ยังใช้ทั้งหมด)
- ที่มาของ rule: คำตอบของ boiledegg ในการวางแผน (2026-10-02 / 03) + feedback ของ lead (zerph) 2026-10-05 ·
  ไม่ได้ extract จากโค้ดเก่า

> เอกสารนี้คือ source of truth ของงานระยะ 2 ของ module — โค้ดและ test ต้องตรงกับเอกสารนี้ ถ้าไม่ตรงให้แก้โค้ด
> หรือแก้เอกสารผ่านการอนุมัติใหม่ ห้ามอ้างเอกสารของโปรเจกต์เก่า

## 1. หน้าที่ของ module (ส่วนที่เพิ่ม)

**อยู่ใน scope รอบนี้:**
- subaccount login (เส้น login เดิม)
- passcode: ตั้งครั้งแรก, เปลี่ยนเอง, middleware `RequirePasscode`, ตัวนับผิดระดับบัญชี
- เปลี่ยนรหัสผ่านเอง
- role `ADMIN`: รีเซ็ต passcode / รหัสผ่านให้ COMPANY / SHAREHOLDER / AGENT / sub ที่จำไม่ได้ (เส้นอยู่ module `admin_management` · middleware `adminMw.RequireAdmin` — แยก 2026-10-09)
- บังคับเปลี่ยนรหัสผ่าน / passcode หลังถูกรีเซ็ต (ค่าชั่วคราวหมดอายุ 24 ชม.)
- ตาราง `auth_audit_logs` (เหตุการณ์ที่เปลี่ยนข้อมูลบัญชี — AUTH-49)
- scripts: `create_admin`, `reset_credentials` (สำหรับ SUPERADMIN / ADMIN) และแก้ `create_superadmin` ให้ใช้กติการหัสผ่านกลาง
- ปรับของ Phase 1 ที่กระทบ: JWT claims, session JSON, logout, middleware `Authenticated`, `Actor`

**แยกออกไปเป็นงานอื่น:** quick-start, การเข้าถึง agent ในสาย (`/agents/:agent_id`), role/permission guard
(รอบนี้มีแค่ `adminMw.RequireAdmin()` (module admin_management)), สร้าง/จัดการ subaccount

## 2. คำศัพท์

| คำ | ความหมาย |
|---|---|
| subaccount (sub) | บัญชีย่อยที่ Company, Share หรือ Agent สร้าง ทำงานในนามของผู้สร้างตามสิทธิ์ที่ได้รับ |
| ผู้สร้าง | agent ที่เป็นเจ้าของ sub (`subaccounts.agent_id`) |
| passcode | ตัวเลข 6 หลัก (string) ใช้แค่ 2 ที่: ยืนยันตอนทำรายการสำคัญ (`RequirePasscode`) และตอนเปลี่ยน passcode |
| admin | บัญชี role `ADMIN` ใน `user_agents` (ทีม support) รีเซ็ต passcode / รหัสผ่านให้ผู้ใช้ที่จำไม่ได้ |
| ค่าชั่วคราว | passcode / รหัสผ่านที่ระบบสุ่มให้ตอน admin รีเซ็ต · หมดอายุใน 24 ชม. |
| ถูกบังคับเปลี่ยน | บัญชีที่ถูกรีเซ็ต (`must_change_password` / `must_change_passcode` = true) |
| ด่านหลัง login | เงื่อนไขที่ session ต้องผ่านก่อนเรียก route อื่นใต้ `/bo/pr` (AUTH-29) |

## 3. Business rules

### Subaccount และ login

| ID | Rule |
|---|---|
| AUTH-17 | sub เก็บในตาราง `subaccounts` แยกจาก `user_agents` และเก็บ `agent_id` ของผู้สร้าง |
| AUTH-18 | username ของ sub = `{username ผู้สร้าง}@{name}` · `{name}` ตรง `^[a-z0-9]{3,20}$` · username มี `@` → ค้นใน `subaccounts` · ไม่มี `@` → ค้นใน `user_agents` · username ของ agent ห้ามมี `@` · ใช้ AUTH-01/02/03 เหมือน agent · ไม่พบ sub (รวมกรณีไม่มีผู้สร้างนั้น เช่น `ghost@x`) → `401201` พร้อม dummy bcrypt compare |
| AUTH-19 | username **ห้ามเปลี่ยน** ทั้ง agent และ sub |
| AUTH-20 | sub มี status ของตัวเอง (`ACTIVE` / `SUSPENDED` / `LOCKED`) ผู้สร้างล็อก sub ทีละตัวได้ · sub ที่ `LOCKED` login ไม่ได้ (`401301`) — เช็คหลังรหัสผ่านถูกแล้วเท่านั้น |
| AUTH-21 | ผู้สร้างหรือ upline คนใดของผู้สร้างเป็น `LOCKED` → sub login ไม่ได้ (`401302`) · การไล่ upline (`HasLockedAncestor`) ไม่นับตัวเอง — กรณี sub ต้องเช็คผู้สร้างแยกด้วย |
| AUTH-22 | sub หรือผู้สร้างเป็น `SUSPENDED` → login ได้ แต่ทำอะไรไม่ได้เหมือนผู้สร้าง (ความหมายของ SUSPENDED กำหนดใน module สายงาน) |
| AUTH-23 | session แยกต่อคน — AUTH-06/07/08/09 ใช้กับ sub แต่ละตัว · sub login ไม่ทำให้ผู้สร้างหลุด และผู้สร้าง login ไม่ทำให้ sub หลุด |
| AUTH-24 | AUTH-10/11 ใช้กติกาเดียวกับ agent · ตัวนับคิดตาม username เต็มของ sub |
| AUTH-25 | response login เหมือน agent · `role` = role ของผู้สร้าง · `is_subaccount = true` · ไม่มี field ข้อมูลผู้สร้าง (frontend ตัดจาก `@` เอง) |
| AUTH-26 | sub ใช้งานในนามของผู้สร้าง — ขอบเขตข้อมูลเป็นของผู้สร้าง (การจำกัดสิทธิ์อยู่ในงาน permission) |
| AUTH-27 | middleware (ทุก request ทั้ง agent และ sub): บัญชีเป็น `LOCKED` → `401301` · ผู้สร้าง (กรณี sub) หรือ upline คนใดเป็น `LOCKED` → `401302` · ทั้งสองกรณีลบ session · **แทน Phase 1 หัวข้อ 9 ข้อ 2** (เดิม middleware เช็คแค่สถานะตัวเอง) |
| AUTH-28 | login (หลังเทียบรหัสผ่านถูกแล้วเท่านั้น): ถูกบล็อกเพราะ passcode ผิด (AUTH-35) → `401309` · รหัสผ่านชั่วคราว **หรือ** passcode ชั่วคราวหมดอายุ → `401310` · ลำดับเช็ค `401301` → `401302` → `401309` → `401310` |

### ด่านหลัง login

| ID | Rule |
|---|---|
| AUTH-29 | หลัง login ใช้งานได้ทันที **ไม่ต้องยืนยัน passcode** · session ต้องผ่านด่านตามลำดับนี้ก่อนเรียก route อื่นใต้ `/bo/pr` (logout เรียกได้เสมอ): |
| | 1. `must_change_password` → เรียกได้เฉพาะ `password/change` อื่นๆ ตอบ `401306` |
| | 2. `must_change_passcode` → เรียกได้เฉพาะ `passcode/change` อื่นๆ ตอบ `401307` |
| | 3. ยังไม่มี passcode (`passcode_hash IS NULL`) → เรียกได้เฉพาะ `passcode/setup` อื่นๆ ตอบ `401304` |

### Passcode

| ID | Rule |
|---|---|
| AUTH-30 | passcode เป็นตัวเลข 6 หลัก (`0`–`9`) ทุก field ต้องเป็น **JSON string** — ส่งเป็น number ตอบ `422` (กันเลข 0 นำหน้าหาย) · ไม่มีเลขต้องห้าม · เก็บเป็น bcrypt hash เท่านั้น |
| AUTH-31 | agent, sub และ admin มี passcode ของตัวเองแยกกัน |
| AUTH-32 | ตั้ง passcode ครั้งแรก: ส่ง `passcode` + `confirm_passcode` ตรงกัน ไม่ต้องกรอกรหัสผ่าน · ใช้ได้ครั้งเดียวตอน `passcode_hash IS NULL` · มี passcode แล้วตอบ `401401` **เสมอ** · เขียนด้วย `UPDATE ... WHERE passcode_hash IS NULL` ถ้าไม่มีแถวถูกอัปเดตตอบ `401401` (กันยิงพร้อมกัน) |
| AUTH-33 | `RequirePasscode`: รับ `passcode` จาก **body เท่านั้น** · route ที่ใช้ห้ามเป็น GET · แต่ละ module เลือกเองว่า route ไหนต้องใช้ |
| AUTH-34 | เปลี่ยน passcode เอง: ส่ง `old_passcode`, `new_passcode`, `confirm_passcode` (ไม่ต้องใช้รหัสผ่าน) · agent, sub และ admin เปลี่ยนของตัวเองได้ · passcode ใหม่ห้ามซ้ำกับตัวปัจจุบัน รวมถึงค่าชั่วคราว (`401403`) · สำเร็จ → **session เดิมใช้ต่อได้** · `must_change_passcode = false` และล้างวันหมดอายุค่าชั่วคราว · ถูกบังคับเปลี่ยน: `old_passcode` = ค่าชั่วคราว · หมดอายุแล้วตอบ `401310` |
| AUTH-35 | **ตัวนับ passcode ผิด** — ตัวนับเดียวระดับบัญชี (INCR แบบ atomic, TTL 24 ชม. นับจากครั้งแรกที่ผิด) · นับจาก `RequirePasscode` และ `old_passcode` · ผิดครั้งที่ 1–4 → `401204` msg บอกจำนวนครั้งที่เหลือ (เช่น "passcode ไม่ถูกต้อง (เหลือ 2 ครั้ง)") · ผิดครั้งที่ 5 → ลบ session, บล็อก 1 ชม., ตอบ `401205` · status บัญชีไม่เปลี่ยน · ตอนบล็อกตัวนับเริ่ม 0 ใหม่ · ใส่ถูก → ตัวนับเป็น 0 · ระหว่างถูกบล็อก login ได้ `401309` |

### รหัสผ่าน

| ID | Rule |
|---|---|
| AUTH-36 | กติการหัสผ่าน — ใช้ร่วมกันทุกที่ที่ตั้งรหัส (อยู่ใน `app/core`): ยาว **8–64** ตัว · ใช้ได้เฉพาะ `a-z`, `A-Z`, `0-9` และ ASCII punctuation `` !"#$%&'()*+,-./:;<=>?@[\]^_`{|}~ `` · ห้ามมีช่องว่าง · ต้องมีตัวอักษร (`a-z`/`A-Z`) อย่างน้อย 1 ตัว และตัวเลขอย่างน้อย 1 ตัว · ตัวพิมพ์เล็ก/ใหญ่ถือเป็นคนละตัว · ห้ามตัวเดียวกันซ้ำติดกัน 4 ตัว (`aaaa`, `1111`) · ไม่เช็ค username · ไม่เทียบรายการรหัสยอดนิยม · `create_superadmin`, `create_admin`, `reset_credentials` ใช้กติกาชุดนี้ (แก้ `create_superadmin` ของ Phase 1 ที่บังคับ ≥12) |
| AUTH-37 | เปลี่ยนรหัสผ่านเอง: ส่ง `old_password`, `new_password`, `confirm_password` และ `passcode` (ผ่าน `RequirePasscode`) · agent, sub และ admin เปลี่ยนของตัวเองได้ |
| AUTH-38 | รหัสใหม่ห้ามซ้ำกับรหัสปัจจุบันและรหัสก่อนหน้า 1 รหัส (`401402`) |
| AUTH-39 | `old_password` ผิด → `401206` และนับรวมกับตัวนับ AUTH-10 · ครบ 5 ครั้ง → login ไม่ได้ 15 นาที (`401303`) และ session ปัจจุบันถูกลบ |
| AUTH-40 | เปลี่ยนสำเร็จ → **session เดิมใช้ต่อได้** · `must_change_password = false` และล้างวันหมดอายุค่าชั่วคราว |
| AUTH-41 | ถูกบังคับเปลี่ยน: ไม่ต้องส่ง `passcode` · `old_password` = รหัสชั่วคราว · หมดอายุแล้วตอบ `401310` |

### Transaction

| ID | Rule |
|---|---|
| AUTH-42 | การตั้ง/เปลี่ยน/รีเซ็ตรหัสผ่านหรือ passcode ทุกแบบทำใน transaction และ `SELECT ... FOR UPDATE` แถวบัญชีก่อนอ่านค่า |

### Admin

| ID | Rule |
|---|---|
| AUTH-43 | admin เป็นบัญชีใน `user_agents` role `ADMIN` · **ไม่มี parent** · **สร้าง sub ไม่ได้** · **ถูกกรองออกจาก query สายงาน, รายงาน และ list agent ทุกที่** · สร้างด้วย `scripts/create_admin` เท่านั้น (รหัสผ่านอ่านจาก stdin ตรวจตาม AUTH-36) · login และด่าน AUTH-29 เหมือนบัญชีอื่น |
| AUTH-44 | route ของ admin ใช้ได้เฉพาะ role `ADMIN` (`adminMw.RequireAdmin()` — module admin_management) — role อื่นรวม SUPERADMIN ตอบ `401308` · ต้องส่ง `passcode` ของ admin เองใน body (`RequirePasscode`) |
| AUTH-45 | ระบุเป้าหมายด้วย `username` (ตาม AUTH-18 · ผ่าน AUTH-01) · ไม่พบ → `401404` · เป้าหมายได้เฉพาะ COMPANY / SHAREHOLDER / AGENT และ sub · SUPERADMIN, ADMIN และตัวเอง → `401406` · เป้าหมาย `LOCKED` (รวม sub ที่ผู้สร้างหรือ upline ล็อก) → `401407` |
| AUTH-46 | ระบบสุ่มค่าชั่วคราวด้วย `crypto/rand` (admin ไม่ได้กรอกเอง) · แสดงใน response ครั้งเดียว เก็บแค่ hash · หมดอายุใน 24 ชม. · response ใส่ `Cache-Control: no-store` · ห้าม log response body · รหัสผ่านชั่วคราว: 12 ตัว ตัวอักษรกับตัวเลขเท่านั้น ไม่ใช้ `0 O o 1 l I` และต้องผ่าน AUTH-36 · passcode ชั่วคราว: ตัวเลข 6 หลัก |
| AUTH-47 | รีเซ็ต passcode: เป้าหมายที่ยังไม่เคยตั้ง passcode → `401405` · บันทึกค่าชั่วคราว (ถ้าสุ่มได้ซ้ำกับ passcode ปัจจุบันให้สุ่มใหม่ภายใน) · `must_change_passcode = true` · ล้างตัวนับและบล็อก passcode (AUTH-35) ของเป้าหมาย · session ของเป้าหมายถูกลบทันที |
| AUTH-48 | รีเซ็ตรหัสผ่าน: บันทึกค่าชั่วคราว (ถ้าสุ่มได้ซ้ำกับรหัสปัจจุบันหรือก่อนหน้าให้สุ่มใหม่ภายใน ไม่ตอบ `401402`) · `must_change_password = true` · ล้างบล็อก login (AUTH-10) ของเป้าหมาย · session ของเป้าหมายถูกลบทันที |
| AUTH-49 | `auth_audit_logs` บันทึก**เฉพาะเหตุการณ์ที่เปลี่ยนข้อมูลบัญชี** (ไม่บันทึก login / logout / login ผิดทีละครั้ง — ตารางจะบวม): |
| | • ผู้ทำคือบัญชีเอง: `PASSCODE_SETUP`, `PASSCODE_CHANGE`, `PASSWORD_CHANGE` — บันทึกใน transaction เดียวกับการเปลี่ยน |
| | • admin รีเซ็ต: `RESET_PASSCODE`, `RESET_PASSWORD` — ใน transaction เดียวกับการรีเซ็ต |
| | • ระบบบล็อก (actor = `SYSTEM`): `PASSCODE_BLOCKED` (AUTH-35), `LOGIN_BLOCKED` (AUTH-10/39 · username ที่ไม่มีจริงเก็บแค่ `target_username`) — บันทึกไม่ได้ไม่ทำให้ request ล้ม |
| | • เก็บ actor, target (type + id + username ณ เวลานั้น), ip, user agent, request_id · ห้ามเก็บรหัสผ่าน, passcode, token หรือค่าชั่วคราว |
| AUTH-50 | ถูกรีเซ็ตทั้งสองอย่าง → login → เปลี่ยนรหัสผ่าน → เปลี่ยน passcode → ใช้งานได้ (ไม่ต้อง login ใหม่ระหว่างทาง) |
| AUTH-51 | SUPERADMIN และ ADMIN รีเซ็ตผ่าน API ไม่ได้ ต้องใช้ `scripts/reset_credentials` บนเซิร์ฟเวอร์ (กู้บัญชีที่ลืมรหัส) · script ใช้ได้กับ SUPERADMIN / ADMIN **เท่านั้น** · `-password` / `-passcode` เลือกอย่างใดอย่างหนึ่งหรือทั้งคู่ (ทั้งคู่ = transaction เดียว) · สุ่มค่าชั่วคราว + บังคับเปลี่ยน + หมดอายุ 24 ชม. แบบเดียวกับ AUTH-46–48 · แสดงค่าบนจอครั้งเดียว · บันทึก `auth_audit_logs` actor = `SCRIPT` · `actor_username` = user ของเครื่องที่รัน · ไม่มี `actor_id` |
| AUTH-52 | ผู้สร้างรีเซ็ตให้ sub ของตัวเอง **ไม่ได้** (กำหนดทิศทางไว้สำหรับ module subaccount) |

### Effective status

| ID | Rule |
|---|---|
| AUTH-53 | upline ถูก LOCK หรือ SUSPEND คนข้างล่างโดนไปด้วย · effective status ของบัญชี = สถานะที่เข้มที่สุด (`LOCKED` > `SUSPENDED` > `ACTIVE`) ของตัวเอง, ผู้สร้าง (กรณี sub) และ upline ทั้งสาย · คำนวณใน middleware ทุก request ใส่ไว้ใน `Actor.EffectiveStatus` · module อื่นตัดสินว่าทำรายการได้ไหมจากค่านี้ (ความหมายของ SUSPENDED กำหนดใน module สายงาน) |
| AUTH-54 | **SUSPENDED เข้าได้เฉพาะหน้า Profile และ Report ดูได้อย่างเดียว** · **เพิ่ม 2026-10-08 (lead อนุมัติ): ดูข้อมูลรับรอง API (`GET /account/api-credential`) ได้ด้วย** (review account 2026-10-05 — อนุมัติ 2026-10-06): ตัดสินจาก `EffectiveStatus` (ตัวเอง, ผู้สร้างกรณี sub หรือ upline ถูกระงับ) · ทุก route ใต้ `/pr` ใช้ `mw.PassedGates(...)` ซึ่ง**ปฏิเสธ SUSPENDED** (`401311`) · route ดูข้อมูลของ Profile / Report ใช้ `mw.PassedGatesAllowSuspended(...)` บรรทัดเดียวกับ route · ด่านหลัง login (AUTH-29) ยังใช้กับทุก route · **เปลี่ยน passcode / รหัสผ่านของตัวเองก็ไม่ได้** (setup, change — ตัดสิน 2026-10-06) · logout ได้เสมอ · ACTIVE ใช้งานปกติ · LOCKED ถูกเตะตาม AUTH-27 |

## 4. สิ่งที่พบในโค้ดเก่า และการตัดสินใจ

ไม่มี — ไม่ได้ extract จากโค้ดเก่า

## 5. Endpoints

| Method | Path | ด่าน AUTH-29 | สิทธิ์ / middleware | Contract |
|---|---|---|---|---|
| POST | `/api/v1/bo/pb/auth/login` | — | — | เส้นเดิม รับ sub + field ใหม่ |
| POST | `/api/v1/bo/pr/auth/logout` | ไม่ต้องผ่าน | — | path/body เดิม **ข้างในเปลี่ยน** |
| POST | `/api/v1/bo/pr/auth/passcode/setup` | ข้อ 3 | — | **ใหม่** |
| POST | `/api/v1/bo/pr/auth/passcode/change` | ผ่านครบ หรือข้อ 2 | — | **ใหม่** |
| POST | `/api/v1/bo/pr/auth/password/change` | ผ่านครบ (+ `RequirePasscode`) หรือข้อ 1 | — | **ใหม่** |

ลำดับ middleware ใน `routes.go`: `Authenticated` → ด่าน AUTH-29 → `RequirePasscode` (ถ้ามี)

เส้นของ ADMIN (`/admin/passcode/reset` · `/admin/password/reset`) ย้ายไป [admin_management.md](admin_management.md) (แยก module 2026-10-09) — กฎ AUTH-43 – AUTH-52 ยังอยู่ในเอกสารนี้
— เขียนไว้บรรทัดเดียวกับ route ทุกเส้น

**error ของทุก route ใต้ `/bo/pr` ที่ผ่าน `Authenticated`** (ไม่เขียนซ้ำในแต่ละเส้นด้านล่าง):
`401202`, `401203`, `401301`, `401302` · route ที่ใช้ `PassedGates` เพิ่ม `401311` (AUTH-54) · logout ไม่ผ่าน `Authenticated` (AUTH-09) จึงตอบแค่ `401202`

### POST /api/v1/bo/pb/auth/login (เส้นเดิม — รับ sub + field ใหม่)

Request (sub):
```json
{ "username": "agent01@staff", "password": "••••••••" }
```
Response `data`:
```json
{
  "token": "<jwt>",
  "username": "agent01@staff",
  "role": "AGENT",
  "is_subaccount": true,
  "passcode_set": false,
  "must_change_password": false,
  "must_change_passcode": false,
  "expires_at": "2026-10-02T22:00:00+07:00"
}
```
- `role` = role ของผู้สร้าง (กรณี sub)
- `passcode_set` เป็นค่าจริงแล้ว (เดิมส่ง `false` เสมอ)
- `must_change_password` / `must_change_passcode` — **ใหม่** frontend ใช้เลือกหน้าถัดไปตาม AUTH-29

Error codes: `422`, `401201`, `401301`, `401302`, `401303`, `401309`, `401310`, `429`, `500`

### POST /api/v1/bo/pr/auth/logout (path/body เดิม — ข้างในเปลี่ยน)

- Phase 1 สร้าง pointer key จาก `claims.AgentID` ทำให้ sub logout ไปเล็ง key ของผู้สร้าง และ `bo:sess:sub:{id}` ค้าง
  → ต้องอ่าน `account_type` + `account_id` จาก token แทน
- logout หลังได้ `401205` (session ถูกลบแล้ว) ตอบ 200 ตาม AUTH-09

Response: `{"code":200,"msg":"สำเร็จ"}` (ไม่มี `data`) · Error codes: `401202`

### POST /api/v1/bo/pr/auth/passcode/setup (ใหม่ — ตั้ง passcode ครั้งแรก)

Header `Authorization: Bearer <token>`

Request:
```json
{ "passcode": "123456", "confirm_passcode": "123456" }
```
Response:
```json
{ "code": 200, "msg": "สำเร็จ" }
```
ไม่มี `data`

Error codes: `422` (ไม่ใช่ string ตัวเลข 6 หลัก / สองช่องไม่ตรงกัน — msg บอกว่าผิดที่ field ไหน), `401401`,
`401306`, `401307`

### POST /api/v1/bo/pr/auth/passcode/change (ใหม่ — เปลี่ยน passcode เอง)

Header `Authorization: Bearer <token>`

Request:
```json
{ "old_passcode": "123456", "new_passcode": "654321", "confirm_passcode": "654321" }
```
Response:
```json
{ "code": 200, "msg": "สำเร็จ" }
```
ไม่มี `data` · session เดิมใช้ต่อได้

Error codes: `422`, `401204`, `401205`, `401403`, `401310`, `401304`, `401306`

### POST /api/v1/bo/pr/auth/password/change (ใหม่ — เปลี่ยนรหัสผ่านเอง)

Header `Authorization: Bearer <token>`

Request (ปกติ):
```json
{
  "old_password": "••••••••",
  "new_password": "••••••••",
  "confirm_password": "••••••••",
  "passcode": "123456"
}
```
Request (ถูกบังคับเปลี่ยน — ไม่มี `passcode`):
```json
{ "old_password": "••••••••", "new_password": "••••••••", "confirm_password": "••••••••" }
```
Response:
```json
{ "code": 200, "msg": "สำเร็จ" }
```
ไม่มี `data` · session เดิมใช้ต่อได้

Error codes: `422` (ไม่กรอก / `confirm_password` ไม่ตรง / ผิดกติกา AUTH-36 — msg บอกว่าผิดกฎข้อไหน), `401204`,
`401205`, `401206`, `401303`, `401402`, `401310`, `401304`, `401307`

## 6. Schema

migration ไฟล์ใหม่ (ไม่แก้ `20261002000000_agent_auth_create_user_agents.sql`)

```sql
ALTER TABLE user_agents
    ADD COLUMN passcode_hash            VARCHAR(100),
    ADD COLUMN previous_password_hash   VARCHAR(100),
    ADD COLUMN must_change_password     BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN must_change_passcode     BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN temp_password_expires_at TIMESTAMPTZ,
    ADD COLUMN temp_passcode_expires_at TIMESTAMPTZ,
    DROP CONSTRAINT ck_user_agents_role,
    ADD CONSTRAINT ck_user_agents_role
        CHECK (role IN ('SUPERADMIN','ADMIN','COMPANY','SHAREHOLDER','AGENT'));

CREATE TABLE subaccounts (
    id                       BIGSERIAL PRIMARY KEY,
    agent_id                 BIGINT       NOT NULL REFERENCES user_agents(id),  -- ผู้สร้าง
    username                 VARCHAR(71)  NOT NULL,                             -- {owner 50}@{name 20}
    password_hash            VARCHAR(100) NOT NULL,
    previous_password_hash   VARCHAR(100),
    passcode_hash            VARCHAR(100),
    must_change_password     BOOLEAN      NOT NULL DEFAULT false,
    must_change_passcode     BOOLEAN      NOT NULL DEFAULT false,
    temp_password_expires_at TIMESTAMPTZ,
    temp_passcode_expires_at TIMESTAMPTZ,
    status                   VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE',
    last_login_at            TIMESTAMPTZ,
    last_login_ip            VARCHAR(45),
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_subaccounts_username UNIQUE (username),
    CONSTRAINT ck_subaccounts_username CHECK (username = lower(username) AND username ~ '^[^@]+@[a-z0-9]{3,20}$'),
    CONSTRAINT ck_subaccounts_status CHECK (status IN ('ACTIVE','SUSPENDED','LOCKED'))
);
CREATE INDEX idx_subaccounts_agent_id ON subaccounts(agent_id);

-- ห้ามเก็บรหัสผ่าน, passcode, token หรือค่าชั่วคราว
CREATE TABLE auth_audit_logs (
    id              BIGSERIAL    PRIMARY KEY,
    action          VARCHAR(30)  NOT NULL,
    actor_type      VARCHAR(10)  NOT NULL,  -- AGENT / SUB / SCRIPT / SYSTEM
    actor_id        BIGINT,                 -- มีเฉพาะ AGENT / SUB
    actor_username  VARCHAR(71),            -- ชื่อ ณ เวลานั้น
    target_type     VARCHAR(10),            -- AGENT / SUB · NULL = ไม่มีบัญชีนี้ในระบบ
    target_id       BIGINT,
    target_username VARCHAR(71)  NOT NULL,
    ip              VARCHAR(45),
    user_agent      VARCHAR(255),
    request_id      VARCHAR(64),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT ck_auth_audit_logs_action CHECK (action IN (
        'PASSCODE_SETUP','PASSCODE_CHANGE','PASSWORD_CHANGE',
        'RESET_PASSCODE','RESET_PASSWORD','PASSCODE_BLOCKED','LOGIN_BLOCKED')),
    CONSTRAINT ck_auth_audit_logs_actor_type CHECK (actor_type IN ('AGENT','SUB','SCRIPT','SYSTEM')),
    CONSTRAINT ck_auth_audit_logs_actor_id CHECK ((actor_type IN ('AGENT','SUB')) = (actor_id IS NOT NULL)),
    CONSTRAINT ck_auth_audit_logs_target_type CHECK (target_type IS NULL OR target_type IN ('AGENT','SUB')),
    CONSTRAINT ck_auth_audit_logs_target_id CHECK ((target_type IS NULL) = (target_id IS NULL))
);
CREATE INDEX idx_auth_audit_logs_target ON auth_audit_logs(target_type, target_id, created_at);
CREATE INDEX idx_auth_audit_logs_actor ON auth_audit_logs(actor_type, actor_id, created_at);
CREATE INDEX idx_auth_audit_logs_action ON auth_audit_logs(action, created_at);
CREATE INDEX idx_auth_audit_logs_created_at ON auth_audit_logs(created_at);
```

- `passcode_hash IS NULL` = `passcode_set = false`
- `previous_password_hash` เก็บรหัสก่อนหน้า 1 รหัสสำหรับ AUTH-38
- `subaccounts` มีเฉพาะคอลัมน์ที่ auth ใช้ — สิทธิ์และคอลัมน์อื่นเพิ่มตอนทำ module subaccount/permission
- การทดสอบ: integration test insert แถว sub ลง DB เอง ไม่มี script สร้าง sub

JWT claims (token ของ Phase 1 ที่ไม่มี `account_type` ถือว่า invalid → `401202`):

| Claim | ค่า |
|---|---|
| `sid` | session id |
| `account_type` | `AGENT` / `SUB` |
| `account_id` | id ของบัญชี (`user_agents.id` หรือ `subaccounts.id`) |
| `agent_id` | id ของผู้สร้าง — มีเฉพาะเมื่อ `account_type = SUB` |

`Actor` (middleware `Authenticated` สร้าง): ของเดิม + `AccountType`, `SubaccountID`, `EffectiveStatus` (AUTH-53),
`MustChangePassword`, `MustChangePasscode`

Redis:

| Key | ค่า | TTL |
|---|---|---|
| `bo:sess:{sid}` | JSON `{account_type, account_id, agent_id, ip, created_at, expires_at}` | idle 60 นาที (เหมือนเดิม) — `expires_at` ใช้คำนวณ absolute timeout (AUTH-07) |
| `bo:sess:agent:{agent_id}` | `sid` ปัจจุบันของ agent / admin | absolute 12 ชม. (เดิม) |
| `bo:sess:sub:{subaccount_id}` | `sid` ปัจจุบันของ sub | absolute 12 ชม. |
| `bo:login:fail:{username}` | จำนวนครั้งที่ผิด (login + `old_password` ผิด) | 15 นาที (เดิม) |
| `bo:login:ip:{ip}` | จำนวนครั้งที่เรียก | 1 นาที (เดิม) |
| `bo:passcode:fail:{type}:{id}` | จำนวนครั้งที่ passcode ผิด (INCR) | 24 ชม. นับจากครั้งแรกที่ผิด |
| `bo:passcode:block:{type}:{id}` | บล็อกเพราะ passcode ผิดครบ 5 ครั้ง | 1 ชม. |

## 7. Test cases

| Rule | Input | Expected |
|---|---|---|
| AUTH-18 | login `" Agent01@Staff "` (บัญชี `agent01@staff`) | สำเร็จ, `is_subaccount = true` |
| AUTH-18 | `agent01@nobody` | `401201` |
| AUTH-18 | `ghost@x` (ไม่มีผู้สร้างนี้) | `401201` (ผ่าน dummy bcrypt) |
| AUTH-20 | sub `LOCKED` + รหัสถูก | `401301` |
| AUTH-20 | sub `LOCKED` + รหัสผิด | `401201` |
| AUTH-21 | ผู้สร้าง `LOCKED` | `401302` |
| AUTH-21 | upline ของผู้สร้าง `LOCKED` | `401302` |
| AUTH-22 | ผู้สร้าง `SUSPENDED` | login สำเร็จ |
| AUTH-23 | sub login แล้วผู้สร้าง login | token ของทั้งสองยังใช้ได้ |
| AUTH-23 | sub logout | ผู้สร้างยังใช้งานได้ · `bo:sess:sub:{id}` ถูกลบ |
| AUTH-24 | sub ผิด 5 ครั้ง แล้วใส่ถูก | `401303` · ผู้สร้างยัง login ได้ |
| AUTH-25 | sub ของ Agent | `role = AGENT`, `is_subaccount = true` |
| AUTH-27 | sub ถูก LOCK ระหว่างใช้งาน | request ถัดไป `401301` และ session ถูกลบ |
| AUTH-27 | ผู้สร้างถูก LOCK ระหว่างที่ sub ใช้งาน | request ถัดไปของ sub `401302` และ session ถูกลบ |
| AUTH-27 | upline ของ agent (เช่น Company) ถูก LOCK ระหว่างใช้งาน | request ถัดไปของ agent `401302` และ session ถูกลบ |
| AUTH-53 | upline ของผู้สร้าง SUSPENDED | `EffectiveStatus` ของ agent และ sub = `SUSPENDED` |
| AUTH-53 | sub SUSPENDED · ผู้สร้าง ACTIVE | sub = `SUSPENDED` · ผู้สร้าง = `ACTIVE` |
| AUTH-54 | บัญชีที่ `EffectiveStatus = SUSPENDED` (upline ถูกระงับ หรือตัวเอง) เรียก route ที่ใช้ `PassedGatesAllowSuspended` | สำเร็จ |
| AUTH-54 | บัญชีที่ `EffectiveStatus = SUSPENDED` เรียก route อื่น · `RequirePasscode` · passcode setup / change · password change | `401311` (HTTP 403) |
| AUTH-54 | sub SUSPENDED · ผู้สร้าง ACTIVE | sub `401311` · ผู้สร้างใช้งานได้ปกติ |
| AUTH-54 | ยังไม่ตั้ง passcode และถูกระงับ | passcode setup `401311` · route ดูข้อมูลติดด่าน `401304` |
| AUTH-28 | ถูกบล็อก passcode แล้ว login + รหัสถูก | `401309` |
| AUTH-28 | ถูกบล็อก passcode แล้ว login + รหัสผิด | `401201` |
| AUTH-28 | รหัสชั่วคราวเกิน 24 ชม. แล้ว login ด้วยรหัสนั้น | `401310` |
| AUTH-29 | login แล้วเรียก route อื่นโดยยังไม่มี passcode | `401304` |
| AUTH-29 | มี passcode แล้ว login → เรียก route อื่นทันที | ผ่าน (ไม่ต้องยืนยัน) |
| AUTH-29 | `must_change_password` แล้วเรียก `passcode/setup` | `401306` |
| AUTH-29 | `must_change_passcode` แล้วเรียก `password/change` | `401307` |
| AUTH-29 | ยังไม่ผ่านด่านแล้ว logout | สำเร็จ |
| AUTH-30 | `"passcode": 123456` (number) | `422` |
| AUTH-30 | `confirm_passcode` ไม่ตรง / `"12345"` / `"12345a"` | `422` |
| AUTH-32 | ตั้งซ้ำครั้งที่ 2 | `401401` |
| AUTH-32 | ยิง setup พร้อมกัน 2 request | สำเร็จ 1 · อีกอันได้ `401401` |
| AUTH-33 | route ที่ใช้ `RequirePasscode` ส่ง passcode ทาง header / query | `422` |
| AUTH-34 | sub เปลี่ยน passcode ของตัวเอง | สำเร็จ · token เดิมใช้ต่อได้ |
| AUTH-34 | `new_passcode` = ตัวปัจจุบัน | `401403` |
| AUTH-34 | ถูกบังคับเปลี่ยน · `new_passcode` = ค่าชั่วคราว | `401403` |
| AUTH-34 | ถูกบังคับเปลี่ยน · ค่าชั่วคราวเกิน 24 ชม. | `401310` |
| AUTH-35 | ผิดครั้งที่ 3 | `401204` msg บอก "เหลือ 2 ครั้ง" |
| AUTH-35 | ผิด 4 ครั้ง แล้วถูก แล้วผิดอีก 4 ครั้ง | ยังไม่ถูกบล็อก (ตัวนับถูกล้าง) |
| AUTH-35 | ผิดใน `RequirePasscode` 3 ครั้ง + `old_passcode` ผิด 2 ครั้ง | ครั้งที่ 5 `401205` · token เดิม `401203` · status บัญชีไม่เปลี่ยน |
| AUTH-35 | ผิด 4 ครั้ง → logout → login ใหม่ → ผิด 1 ครั้ง | `401205` (ตัวนับอยู่ระดับบัญชี) |
| AUTH-35 | ระหว่างถูกบล็อก 1 ชม. login + รหัสถูก | `401309` |
| AUTH-36 | `Zx9!kq2` (7 ตัว) / 65 ตัว | `422` |
| AUTH-36 | `Zx9 kq2m` (ช่องว่าง) / `Zx9กkq2m` (อักษรไทย) | `422` |
| AUTH-36 | `Zxkqmwpt` (ไม่มีตัวเลข) / `92837465` (ไม่มีตัวอักษร) | `422` |
| AUTH-36 | `Zx9aaaam` (ซ้ำ 4) | `422` |
| AUTH-36 | `Zx9!kq2m` / `abcd1234` / `Zx9~"'{}` / 64 ตัว | ผ่าน |
| AUTH-38 | ใช้รหัสปัจจุบัน / รหัสก่อนหน้า | `401402` |
| AUTH-39 | `old_password` ผิด 1 ครั้ง | `401206` |
| AUTH-39 | `old_password` ผิด 5 ครั้ง | session ถูกลบ · login ไม่ได้ 15 นาที (`401303`) |
| AUTH-40 | เปลี่ยนสำเร็จ | token เดิมใช้ต่อได้ · login ด้วยรหัสใหม่ได้ · รหัสเดิมไม่ได้ |
| AUTH-41 | ถูกบังคับเปลี่ยน · ไม่ส่ง `passcode` | สำเร็จ |
| AUTH-41 | ถูกบังคับเปลี่ยน · รหัสชั่วคราวเกิน 24 ชม. | `401310` |
| AUTH-44 | AGENT / SUPERADMIN เรียก `admin/password/reset` | `401308` |
| AUTH-44 | admin ส่ง `passcode` ของตัวเองผิด | `401204` (นับรวมตัวนับของ admin) |
| AUTH-45 | `username` ไม่มีในระบบ | `401404` |
| AUTH-45 | เป้าหมายเป็น SUPERADMIN / ADMIN คนอื่น / ตัวเอง | `401406` |
| AUTH-45 | เป้าหมาย `LOCKED` / sub ที่ผู้สร้างถูก LOCK | `401407` |
| AUTH-46 | รีเซ็ตรหัสผ่าน | `temp_password` ยาว 12 ตัว ไม่มี `0 O o 1 l I` ผ่าน AUTH-36 · header `Cache-Control: no-store` |
| AUTH-47 | รีเซ็ต passcode ของบัญชีที่ยังไม่เคยตั้ง | `401405` |
| AUTH-47 | รีเซ็ต passcode ของ sub ที่ถูกบล็อก passcode และ login อยู่ | token ของ sub ได้ `401203` · login ได้ (บล็อกถูกล้าง) `must_change_passcode = true` |
| AUTH-48 | รีเซ็ตรหัสผ่านของบัญชีที่ถูกบล็อก login (AUTH-10) | login ด้วยรหัสชั่วคราวได้ทันที `must_change_password = true` |
| AUTH-49 | รีเซ็ตสำเร็จ | มีแถว `RESET_*` (actor, target, username, ip, request_id) |
| AUTH-49 | ตั้ง passcode / เปลี่ยนรหัสผ่าน | มีแถว `PASSCODE_SETUP` / `PASSWORD_CHANGE` · actor = target = ตัวเอง |
| AUTH-49 | passcode ผิดครบ 5 | แถว `PASSCODE_BLOCKED` actor = `SYSTEM` |
| AUTH-49 | login ผิดครบ 5 ด้วย username ที่ไม่มีจริง | แถว `LOGIN_BLOCKED` ไม่มี target_type / target_id · มี target_username |
| AUTH-49 | login สำเร็จ / logout / login ผิดครั้งเดียว | ไม่มีแถวเพิ่ม |
| AUTH-49 | รีเซ็ตไม่สำเร็จ | ไม่มีแถว |
| AUTH-50 | ถูกรีเซ็ตทั้งสองอย่าง → login → เรียก `passcode/change` ก่อน | `401306` |
| AUTH-50 | ถูกรีเซ็ตทั้งสองอย่าง → login → เปลี่ยนรหัสผ่าน → เปลี่ยน passcode → เรียก route อื่น | ผ่าน ด้วย token เดิม |
| AUTH-51 | `reset_credentials -username <superadmin> -password -passcode` | ค่าชั่วคราว 12 ตัว / 6 หลัก · session เดิมหลุด · login แล้ว `must_change_password` และ `must_change_passcode = true` · audit 2 แถว actor `SCRIPT` |
| AUTH-51 | script กับ Agent / sub · username ที่ไม่มี · ไม่เลือก `-password` หรือ `-passcode` | `401406` · `401404` · error |

## 8. Contract changes (แจ้ง frontend)

- login ไม่ต้องยืนยัน passcode · response เพิ่ม `must_change_password`, `must_change_passcode` · `passcode_set` เป็นค่าจริง
- หลัง login เลือกหน้าตามลำดับ AUTH-29: เปลี่ยนรหัสผ่าน → เปลี่ยน passcode → ตั้ง passcode
- route ใต้ `/bo/pr` ตอบ `401304` / `401306` / `401307` จนกว่าจะผ่านด่าน
- passcode ทุก field ต้องส่งเป็น string
- username ที่มี `@` login ได้จริงถ้าเป็น sub (เดิมตอบ `401201` เสมอ)
- token ของ Phase 1 ใช้ไม่ได้หลัง deploy (ต้อง login ใหม่)
- เปลี่ยนรหัสผ่าน / passcode สำเร็จแล้ว token เดิมใช้ต่อได้
- `401205` → พาไปหน้า login · `401204` แสดง msg (มีจำนวนครั้งที่เหลือ)
- admin reset ได้ค่าชั่วคราวใน response ครั้งเดียว ต้องแสดงให้ admin เห็นแล้วไม่เก็บไว้
- เพิ่ม 5 route ใหม่ (หัวข้อ 5) และ error code ใหม่ (หัวข้อ 9 ข้อ 3)
- route ที่ต้องยืนยันตอนทำรายการ ต้องส่ง `passcode` ใน body

## 9. การตัดสินใจ

1. **ไม่มีการยืนยัน passcode หลัง login** (lead 2026-10-05) — passcode ใช้แค่ `RequirePasscode` และตอนเปลี่ยน passcode
2. **ตัวนับ passcode ผิดอยู่ระดับบัญชี** (lead 2026-10-05) — แทนแบบนับต่อ session ของ draft แรก
   ตอนถูกบล็อกตัวนับเริ่ม 0 ใหม่
2.1 **middleware เช็ค upline ทุก request ทั้ง agent และ sub** (lead 2026-10-05) — แทน Phase 1 หัวข้อ 9 ข้อ 2
   ส่วน upline SUSPEND ไม่เตะออก แต่ทำให้ effective status ของคนข้างล่างเป็น SUSPENDED (AUTH-53)
3. **Error code** (module `agent_auth` = `bb=01`)

| Code | HTTP | ความหมาย |
|---|---|---|
| 401204 | 200 | passcode ไม่ถูกต้อง (msg บอกจำนวนครั้งที่เหลือ) |
| 401205 | 401 | passcode ผิดครบ 5 ครั้ง session ถูกตัดและระงับ 1 ชม. |
| 401206 | 200 | รหัสผ่านเดิมไม่ถูกต้อง |
| 401304 | 403 | ต้องตั้ง passcode ก่อน |
| 401306 | 403 | ต้องเปลี่ยนรหัสผ่านก่อน |
| 401307 | 403 | ต้องเปลี่ยน passcode ก่อน |
| 401308 | 403 | ไม่มีสิทธิ์ใช้งานส่วนของ admin |
| 401309 | 403 | ระงับ login เพราะ passcode ผิดหลายครั้ง |
| 401310 | 403 | ค่าชั่วคราวหมดอายุ กรุณาติดต่อ admin |
| 401311 | 403 | บัญชีถูกระงับ ใช้งานได้เฉพาะหน้าประวัติของฉันและรายงาน (AUTH-54) |
| 401401 | 200 | ตั้ง passcode ไว้แล้ว |
| 401402 | 200 | รหัสผ่านใหม่ซ้ำกับรหัสที่เคยใช้ |
| 401403 | 200 | passcode ใหม่ซ้ำกับตัวเดิม |
| 401404 | 200 | ไม่พบบัญชีที่ต้องการรีเซ็ต |
| 401405 | 200 | บัญชีนี้ยังไม่ได้ตั้ง passcode |
| 401406 | 200 | ไม่สามารถรีเซ็ตบัญชีนี้ได้ |
| 401407 | 200 | บัญชีเป้าหมายถูกล็อก |

`401305` ถูกตัดก่อนปล่อยใช้ (ไม่เคยประกาศในโค้ด) · `401204` / `401206` ตอบ HTTP 200 เพราะ frontend พาไปหน้า login
เมื่อได้ 401 — ใส่ผิดครั้งเดียวไม่ควรถูกเด้งออก

4. **บัญชีทดสอบ sub** — integration test insert ลง DB เอง ไม่ทำ script
5. **role/permission guard เต็มรูปแบบ** แยกเป็นงานอื่น — รอบนี้มีแค่ `adminMw.RequireAdmin()` (module admin_management) เพื่อให้ route admin
   ระบุสิทธิ์บรรทัดเดียวกับ route ได้ตามกฎข้อ 28
