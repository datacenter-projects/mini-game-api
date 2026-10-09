# Review BO Account (lead: zerph) — 2026-10-09 · ✅ review ครบแล้ว

อ้างอิง: `docs/modules/account.md` (dev `ba9cd5f`) · เจ้าของ module: boiledegg · ไฟล์นี้คือการตัดสินของ lead — ทำตามรายการแก้ใน branch ใหม่แล้วเปิด PR ให้ lead review

## การตัดสินของ lead
| # | เรื่อง | ตัดสิน |
|---|---|---|
| A1 | Dashboard (1.1, `GET /account/dashboard`) | **ยังไม่ต้องทำ** (ออกจาก scope รอบนี้) |
| A2 | `GET /account/profile` | เจ้าของ token ดูได้เสมอ ไม่ขึ้นกับ permission · SUSPENDED ดูได้ · **LOCKED ดูไม่ได้** (คง AUTH-27) · **ต้องผ่านด่าน AUTH-29 ก่อน (ตั้ง passcode ครั้งแรกก่อน)** → ตรงกับโค้ดปัจจุบัน ไม่ต้องแก้ |
| A3 | ~~sub เห็น Key แบบปิดบางส่วน~~ | **ยกเลิก** — แทนด้วย A5/A6 |
| A4 | Key ของทุกประเภทบัญชี (รวม Transfer / Agent) | คงไว้ตาม spec · เรื่องโหมดกระเป๋าไปมีผลตอน bet / payout — ยังไม่โฟกัส |
| A5 | **กฎทั้งระบบ BO: sub ต้องมี permission เท่านั้นถึงใช้เส้น BO ได้ ยกเว้น Profile และ auth ของบัญชีตัวเอง** (`passcode/setup`, `passcode/change`, `password/change`, `logout`) | lead ตัดสิน (ตามแนะนำ) |
| A6 | สิทธิ์หน้า 1.3 | **เมนูใหม่ `api_credential`**: `view` = `GET /account/api-credential` · `edit` = `POST /account/update-credential` (ยังต้อง passcode) · บัญชีหลักได้ `edit` อัตโนมัติ · **sub ที่มีสิทธิ์เห็น Key เต็ม** |
| A7 | `allowed_ips` ช่วงกว้างเกิน | **ห้าม** prefix กว้างกว่า `/16` (เช่น `0.0.0.0/0`, `/8`) และห้าม IP ภายใน/พิเศษ (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `127.0.0.0/8`, `0.0.0.0/8`, `169.254.0.0/16`, `100.64.0.0/10`) → `422` |
| A8 | CIDR ที่มี host bits (`198.51.100.5/24`) | คงพฤติกรรมโค้ด (`422` "ต้องเป็นที่อยู่เครือข่าย") แล้ว**เขียนเพิ่มใน spec ACC-07** |
| A9 | sub บันทึก credential | ได้ ถ้ามีสิทธิ์ `api_credential` = `edit` (ตาม A6) |

## รายการแก้ (ส่งให้ทีม)

### Spec `docs/modules/account.md`
- S1 ACC-02 → เปลี่ยนเป็นมีสิทธิ์เมนู `api_credential` (A6) · ตาราง Endpoints หัวข้อ 5: GET = `RequirePermission(api_credential, view)` · POST = `RequirePermission(api_credential, edit)` + `RequirePasscode`
- S2 ACC-07 เพิ่มกฎ A7 + A8
- S3 ย้าย Dashboard (ACC-20–27 + endpoint) ไปหัวข้อ "ภายหลัง / ไม่อยู่ใน scope รอบนี้" (A1) และแก้ "ภายในเป็นจำนวนเต็ม" ให้ตรง ACC-18
- S4 ตัวอย่าง Profile: `permissions.dashboard` → `"off"` (SUSPENDED ตาม ACC-12) · เพิ่ม `api_credential` ในตัวอย่าง permissions
- S5 ลบประโยค "ถือจาก Member ของตัวเอง 20 (`pt` — ตั้งเอง)" + แก้ `\n-` ที่หลุด
- S6 หัวเอกสาร: บันทึกการแก้หลังอนุมัติ 2026-10-09 (lead)

### Spec module อื่น
- S7 `agent_management.md` MGMT-50/53: เพิ่มเมนู `api_credential` (view/edit) ในรายการสิทธิ์ · ระบุกฎ A5 (sub ต้องมี permission ทุกเส้น ยกเว้น Profile + auth ตัวเอง)
- S8 `agent_auth_phase2.md`: ระบุข้อยกเว้น A5 สำหรับเส้น auth ของตัวเอง

### โค้ด
- C1 `routes.go:54-55` เพิ่ม `RequirePermission(api_credential, view|edit)` (sub เท่านั้นที่ถูกเช็ค — บัญชีหลักผ่านเสมอ)
- C2 เพิ่มเมนู `api_credential` ในรายการสิทธิ์ (core agent_management) + migration/seed ถ้าสิทธิ์เก็บใน DB · sub เดิมได้ `off`
- C3 `app/core/account/api_credential_rule.go` `normalizeIPv4`: ปฏิเสธ prefix < 16 และช่วงภายใน/พิเศษตาม A7 → error ใหม่ใน `IPViolation` + msg ใน DTO
- C4 Profile `permissions` แสดงเมนู `api_credential` ด้วย
- C5 test: sub ไม่มีสิทธิ์ → `402303` ทั้ง GET/POST · sub `view` GET ได้ POST ไม่ได้ · sub `edit` POST ได้ · IP `/8`, `0.0.0.0/0`, `10.1.2.3` → `422`
- C6 `make`/lint/integration ผ่าน · แจ้ง frontend (เมนูสิทธิ์ใหม่ + error ใหม่ของ allowed_ips)

## ตรวจงานรอบที่ 1 (2026-10-09 · commit 57055a2 + 7430394)
ผลตรวจ: A1–A9 / S1–S8 / C1–C5 ✅ ครบ · C6 🟡 (แจ้ง frontend + รัน integration test)

| # | เรื่อง | lead ตัดสิน |
|---|---|---|
| A10 | 7430394 ให้เมนู `api_credential` กับ Superadmin (Superadmin ไม่มี Key → `403301` ทุกครั้ง) | **revert 7430394** — Superadmin / ADMIN ไม่มีเมนู `api_credential` (ตรงกับ ACC-01) |
| A11 | `allowed_ips` รับ multicast / broadcast | **ห้ามเพิ่ม:** `224.0.0.0/4` (multicast) · `240.0.0.0/4` (reserved) · `255.255.255.255` (broadcast) → `422` + test |

งานที่เหลือ: revert 7430394 (A10) · A11 · C6 (แจ้ง frontend + รัน integration test ของ account)
