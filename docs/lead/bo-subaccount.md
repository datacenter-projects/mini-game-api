# SubAccount_Management — ทำตาม lead review (S1–S7 · C1–C5) · 2026-10-11

จาก boiledegg · review: [docs/reviews/subaccount.md](../reviews/subaccount.md)
branch `boiledegg/bo/subaccount` (แตกจาก `dev` @ `ffec25f`) · **ยังไม่ merge / deploy** — รอ lead review

## 1. error code ที่เลือก (C3 — แจ้ง lead)

**`401312` (HTTP 403) "บัญชีย่อยนี้ถูกปิดใช้งาน กรุณาติดต่อเจ้าของบัญชี"** — code ใหม่ ไม่ใช้ `401301`
- เหตุผล: `401301` = บัญชีถูกล็อก (เกิดจาก admin / ระบบ) แต่ `INACTIVE` คือเจ้าของปิด sub เอง หน้าบ้านต้องบอกให้ติดต่อเจ้าของ ไม่ใช่ admin · แยก code แล้วหน้าบ้านแสดงข้อความถูก
- จองใน `docs/ERROR_CODES.md` แล้ว (ช่วง 4013xx ของ agent_auth)

## 2. ผลแต่ละข้อ

| ข้อ | ผล |
|---|---|
| C1 | `SubUpdateRequest` แบบ section: `info` (ใช้ type เดียวกับ section `info` ของ `agents/detail/update` — ชื่อ · เบอร์คู่ · ส่งบาง field ได้) · `permissions` (`*map` — ส่งมา = แทนทั้งชุด · `{}` = ปิดทุกเมนู · `null` = `422`) · ไม่ส่งเลย = `422` "ต้องส่งอย่างน้อย 1 section" · ลำดับเช็ค `id` → `info` → `permissions` |
| C2 | `UpdateSubaccountService` tx เดียว · section ละ repository (`UpdateSubaccountInfoRepository` / `UpdateSubaccountPermissionsRepository`) · log `UPDATE_INFO` / `SUB_PERMISSIONS` (action ใหม่ใน models) แยกแถวเฉพาะที่เปลี่ยน · สิทธิ์เทียบแบบครบทุกเมนู (ไม่มี key = `off`) · ค่าเดิม = ไม่เขียน log |
| C3 | login: sub ที่สถานะตัวเอง `SUSPENDED` (= `INACTIVE`) → `401312` หลังรหัสผ่านถูก (ลำดับ 401301 → 401302 → 401312 → 401309 → 401310) · middleware `Authenticated` ปฏิเสธ `401312` + ลบ session · `status/update` → `INACTIVE` ลบ session ทุกอันของ sub นั้นหลัง commit (request ถัดไป `401203`) · lock แถว sub `FOR NO KEY UPDATE` · sub `INACTIVE` ไม่เข้าข่าย "SUSPENDED ดูได้" (Profile / `api-credential` ก็ไม่ได้) · ผู้สร้าง / หัวสายถูกระงับ → sub ยังดูได้ตาม AUTH-54 เหมือนเดิม |
| C4 | `CreateSubaccountService` ตั้ง `MustChangePassword: true` ไม่ตั้งวันหมดอายุ · login ตอบ `must_change_password: true` · เส้นอื่น `401306` จนกว่าเปลี่ยนรหัส |
| C5 | test ใหม่ / แก้: `TestSubUpdate` (section เดียว · 2 section · ไม่ส่ง · `{}` / `null` · ค่าเดิมไม่เขียน log · `SUB_PERMISSIONS` · info ถูก + permissions ผิด ไม่บันทึก · แก้สิทธิ์แล้ว request ถัดไปใช้สิทธิ์ใหม่) · `TestSubInactive` (session ถูกตัด รวม Profile / `api-credential` · login `401312` · ตั้งกลับ ACTIVE · สถานะเดิมไม่เขียน log · INACTIVE ตรงใน DB → middleware `401312`) · `TestSubMustChangePassword` (`401306` · ADMIN สร้าง sub `402301`) · แก้ `TestEffectiveStatus` / `TestSuspendedAccess` ที่ทดสอบพฤติกรรมเดิม (sub SUSPENDED ดูได้) |
| S1–S5 | `docs/modules/agent_management.md`: หัวเอกสาร (SA · SN · SQ) · MGMT-41 / 42 / 43 · endpoint `detail/update` / `status/update` / `create` · schema / action `SUB_PERMISSIONS` · test case |
| S6 | `docs/frontend/agent_management.md` 3.10 / 3.11 + หัวข้อ sub ต้องเปลี่ยนรหัสผ่าน · `docs/frontend/auth.md` / `account.md` (`401312`) |
| S7 | `agent_auth_phase2.md` AUTH-53 / 54 + test case + error · `account.md` ACC-31 |

ไม่ได้แตะ: รีเซ็ตรหัส sub (คง AUTH-52 — SQ2) · sub ของ Superadmin (SQ4) · จำนวน sub (SQ7) · `keyword` (SQ8)

## 3. ผล test

- lint 0 issues · structure OK · build / vet ผ่าน · unit + integration ผ่านทุก package (Windows test env · `-race` รันไม่ได้ในเครื่องนี้ — CI รัน)
- ไม่มี migration ใหม่

## 4. แจ้ง frontend (หลัง merge)

1. `subaccounts/detail/update` body ใหม่ `{ "id", "info": {…}, "permissions": {…} }` ส่งเฉพาะ section ที่แก้
2. `INACTIVE` = sub ใช้งานไม่ได้เลย · session ถูกตัด (`401203`) · login ได้ `401312` → แสดง msg ให้ติดต่อเจ้าของ
3. sub ใหม่ต้องเปลี่ยนรหัสผ่านตอน login ครั้งแรก (`must_change_password` · `401306`)
