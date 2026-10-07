# สถานะ — `boiledegg/bo/account`

## 1. ภาพรวม

| | |
|---|---|
| branch | `boiledegg/bo/account` (แตกจาก `dev` @ `ff12296`) |
| spec | `docs/modules/account.md` — **APPROVED** 2026-10-07 · AUTH-54 (`agent_auth_phase2.md`) APPROVED |
| merge | ยังไม่ merge เข้า `dev` · AUTH-54 merge เข้า `boiledegg/test-infra` แล้ว (`9629ff2`) |

## 2. งาน

| ส่วน | สถานะ | หมายเหตุ |
|---|---|---|
| AUTH-54 — SUSPENDED เข้าได้แค่ Profile / Report | ✅ เสร็จ · test ผ่าน | `PassedGates` ปฏิเสธ `401311` · `PassedGatesAllowSuspended` |
| Profile `GET /account/profile` | 🟡 ใช้ได้ · บาง field ค่าชั่วคราว | `user_type` (Company / Share), `currencies`, `balances`, `pt`, `status_game`, `permissions` รอตารางของ module ② — รายละเอียดใน spec หัวข้อ 1 |
| 1.3 ข้อมูลรับรอง API | 🟡 บางส่วน | เสร็จ: config `API_KEY_ENCRYPTION_KEY`, migration, เข้ารหัส Key, validation, repository, DTO · **รอ**: service / route (ต้องใช้ `agent_type` ของ module ②) |
| Dashboard | ⏳ ยังไม่เริ่ม | รอ module เดิมพัน · ใช้เรทจาก `currency_rate` |
| เอกสารหน้าบ้าน `docs/frontend/` | ✅ | `auth.md`, `account.md` (มี curl) |

**ติด:** module ② (`boiledegg/bo/management`) ยังไม่อนุมัติ — Profile / 1.3 เสร็จไม่ได้จนกว่าจะมีตารางของ module ②

## 3. รอ lead

- (ไม่มีข้อที่เป็นของ branch นี้โดยตรง — ดู `bo-management.md`)

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
