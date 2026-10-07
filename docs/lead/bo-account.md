# สถานะ — `boiledegg/bo/account`

## 1. ภาพรวม

| | |
|---|---|
| branch | `boiledegg/bo/account` (แตกจาก `dev` @ `ff12296`) · merge `boiledegg/bo/management` แล้ว (ล่าสุด `f3721c0`) |
| spec | `docs/modules/account.md` — **APPROVED** 2026-10-07 · มีส่วนที่แก้หลังอนุมัติ (2026-10-08 — ACC-12 สิทธิ์ตอนถูกระงับ) · AUTH-54 APPROVED |
| merge | ยังไม่ merge เข้า `dev` · **ต้องเข้าหลัง `boiledegg/bo/management`** (branch นี้มีโค้ด module ② ติดมา) |
| test | `make check` ผ่าน · integration test 53 ตัวผ่าน (agent_auth + module ② + account) |

## 2. งาน

| ส่วน | สถานะ | หมายเหตุ |
|---|---|---|
| AUTH-54 — SUSPENDED เข้าได้แค่ Profile / Report | ✅ | `PassedGates` ปฏิเสธ `401311` · `PassedGatesAllowSuspended` |
| Profile `GET /account/profile` | ✅ ค่าจริงทุก field | ประเภท · สกุล · ยอด · PT · `status_game` · สิทธิ์จาก module ② · บัญชีถูกระงับเห็นสิทธิ์แค่ `report` = `view` |
| 1.3 `GET /account/api-credential` · `POST /account/update-credential` | ✅ | เจ้าของ Key: Seamless 1 to 1 / Share Reseller / Share Master · Key สร้างตอนเปิดหน้าครั้งแรก (ACC-05) |
| สร้าง Key พร้อมบัญชีเจ้าของ (module ② MGMT-04) | ⏳ | ทำหลัง module ② และ account เข้า `dev` ทั้งคู่ |
| Dashboard `GET /account/dashboard` | ⏳ | รอ module เดิมพัน · ใช้เรทจาก `currency_rate` |
| เอกสารหน้าบ้าน `docs/frontend/` | ✅ | `auth.md`, `account.md` (มี curl) |

## 3. รอ lead

1. รับทราบ ACC-12 ที่แก้หลังอนุมัติ: บัญชีที่ถูกระงับ (เอง หรือจากหัวสาย) `permissions` ใน Profile แสดงตามที่ใช้ได้จริง — `report` = `view` · เมนูอื่น `off`
2. ลำดับ merge เข้า `dev`: `boiledegg/bo/management` ก่อน แล้วค่อย `boiledegg/bo/account`

## 4. ประวัติอัปเดต

| วันที่ | commit | อัปเดต |
|---|---|---|
| 2026-10-05 | `e7f48a4` | spec แก้ตาม review รอบ 1 · revert โค้ด Profile รอบแรก |
| 2026-10-06 | `8c7040e` | spec + AUTH-54 อนุมัติ · ทศนิยม 2 ตำแหน่งทุกสกุล |
| 2026-10-06 | `286a302` → `e78c2f6` | แก้ตาม review รอบ 2 (`pt` ชุดเดียว, `balances`, `status` key เดียว, Dashboard) · กฎ PT ต่อลูก |
| 2026-10-07 | `9629ff2` | โค้ด AUTH-54 + test |
| 2026-10-07 | `4836144` → `6d83a61` | ACC-32 ไม่มี null · `permissions` object · spec อนุมัติ |
| 2026-10-07 | `f5a1ebe` · `285b307` | โค้ด Profile · 1.3 ส่วนที่ไม่ต้องรอ module ② · `utils.Money` |
| 2026-10-07 | `621f5df` | path เส้นอัปเดตเป็น `POST /account/update-credential` |
| 2026-10-07 | `8ed0cb0` · `3d41545` | เอกสารหน้าบ้าน `docs/frontend/` · หมายเหตุค่าชั่วคราวใน spec |
| 2026-10-08 | `664f607` | เครื่องมือ test (`scripts/testenv.ps1`, `.env.test`, k6) |
| 2026-10-08 | `d2e319f` | merge `boiledegg/bo/management` (module ② phase 1–5) |
| 2026-10-08 | `9aefb74` | Profile ค่าจริง · 1.3 service / route · สิทธิ์ตอนถูกระงับ · เอกสารหน้าบ้าน |
| 2026-10-08 | `f3721c0` | merge management อีกรอบ (เส้นรายชื่อ `/list` · เอกสาร module ②) |
