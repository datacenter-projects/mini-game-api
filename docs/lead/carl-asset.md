# สถานะ — `carl/asset`

## 1. ภาพรวม

| | |
|---|---|
| branch | `carl/asset` (แตกจาก `dev` @ `65300f1`) · merge `dev` แล้ว (ล่าสุด `ceb3ac9` = `dev` @ `113abeb`) |
| spec | `docs/modules/asset.md` — **DRAFT** รอ lead review (rule `AST-01`–`AST-50`) · การแก้รอบตรวจซ้ำหลัง merge `dev` ยังไม่ commit |
| merge | ยังไม่ merge เข้า `dev` · มีแค่เอกสาร ยังไม่มีโค้ด |
| test | `make check` ผ่าน (ไม่มีโค้ดใหม่) |

## 2. งาน

| ส่วน | สถานะ | หมายเหตุ |
|---|---|---|
| Extract โค้ดเก่า (asset, theme, branding, banner, bundle) | ✅ | ผลอยู่ในหัวข้อ 4 ของ spec (31 ข้อ) — เอกสารสรุประบบเดิมไม่ตรงกับโค้ดหลายจุด |
| spec `docs/modules/asset.md` | ✅ ร่างแล้ว | Game Asset · Game Theme + shades · Site Branding + banner · bundle (หลังบ้าน preview · ผู้เล่นกำหนดแค่ contract) |
| ตรวจ spec กับ `dev` ล่าสุด | ✅ | กฎข้อ 19 (`library`, `banners/list` แบ่งหน้า) · `ParseBodyNoNull` · ลิงก์ spec module ② / account |
| implement | ⏳ | รอ spec อนุมัติ + อนุมัติการแก้ส่วนกลาง (หัวข้อ 3 ข้อ 4) |
| เอกสารหน้าบ้าน `docs/frontend/asset.md` | ⏳ | ทำหลัง spec อนุมัติ |

## 3. รอ lead

1. **review spec `docs/modules/asset.md`** — โดยเฉพาะหัวข้อ 4 (สิ่งที่พบในโค้ดเก่า) และบั๊กเก่าที่ spec ปิด: sub ของ SUPERADMIN
   เขียนค่ากลางได้โดยไม่มีสิทธิ์ · อัปโหลดสร้างช่องใหม่เองเมื่ออ่าน catalog ไม่ได้ · เลือกไฟล์ใช้งานไม่เช็คเกม · ไม่มี unique กันแถว GLOBAL ซ้ำ
2. **ตัดสินคำถามในหัวข้อ 9 ของ spec:**
   1. "เว็บ" (ตั้งหน้าตาเกมของตัวเองได้) = ประเภทที่สร้าง Member ได้ตาม MGMT-02? ชั้นบนดู / แก้ของสายล่างได้ไหม (เสนอ: ดูได้ แก้ไม่ได้)
   2. Member ใต้ Agent ซ้อนชั้นใช้หน้าตาของใคร (เสนอ: ไล่จากผู้สร้างตรงขึ้นไปทีละช่อง → GLOBAL)
   3. branding / banner ใช้เมนูสิทธิ์ `asset` หรือ `announcement` (Superadmin ไม่มี `announcement`)
   4. upload ระบุเป้าหมายทาง form field หรือ query (เสนอ: form field ตาม convention)
   5. ลบไฟล์โลโก้ / favicon เก่าตอนเขียนทับไหม · ต้องมี `asset_change_logs` แบบ MGMT-60 ไหม
   6. error module id `05` + ตาราง code ที่เสนอ
   7. รายชื่อช่อง / ขนาดใน seed และไฟล์ palette ของหน้าบ้าน (ต้องให้ทีมเกม / หน้าบ้านยืนยัน)
3. **`BodyLimit` 10 MB ทั้ง request** ชนกับเพดานไฟล์ 10 MB — เพิ่ม `BodyLimit` เป็น 11 MB (กระทบทุกเส้น) หรือลดเพดานไฟล์
4. **อนุมัติการแก้ส่วนกลางตอน implement:** AWS SDK v2 (S3) ใน `go.mod` · config `S3_*` + `.env.example` · `platform/storage` ·
   helper multipart ใน `pkg/utils` · middleware `ResolveAssetTarget` / `RequireAssetTarget` · จอง module id ใน `ERROR_CODES.md` ·
   แถว `asset` ใน `MIGRATION_STATUS.md`

## 4. ประวัติอัปเดต

| วันที่ | commit | อัปเดต |
|---|---|---|
| 2026-10-08 | `9e95aa8` | ร่าง spec `asset.md` (DRAFT) จากเอกสารสรุประบบเดิม + extract โค้ดเก่า |
| 2026-10-08 | `91bacde` · `ceb3ac9` | merge `dev` (module ② · account · reset script) |
| 2026-10-08 | — (ยังไม่ commit) | ตรวจ spec กับ `dev` ล่าสุด: แบ่งหน้า `library` / `banners/list` · แก้ข้อมูล `BodyLimit` · ลิงก์ spec · เพิ่มไฟล์สถานะนี้ |
