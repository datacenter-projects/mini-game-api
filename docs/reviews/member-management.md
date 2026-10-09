# Review BO Member_Management — lead: zerph · 2026-10-09 · ✅ review ครบแล้ว

อ้างอิง: `docs/modules/member_management.md` (dev `7937231`) · เจ้าของ module: maofoy · ไฟล์นี้คือการตัดสินของ lead — ทำตามรายการแก้ใน branch ใหม่แล้วเปิด PR ให้ lead review

## การตัดสินของ lead
| # | เรื่อง | ตัดสิน |
|---|---|---|
| MA1 | agent ถือ PT สู้กับ Member แต่ละคน (`pt` ต่อ Member, 0 ถึงค่าที่ผู้สร้างได้รับ ทีละ 0.5) | **อนุมัติ** (คู่กับ U1 ของ agent_management) |
| MA2 | `remain_quota` = ได้รับ − `pt` ระบบคิดและเก็บ + sync | **อนุมัติ** + เงื่อนไข: งาน C6 ของ agent_management (CSM → Share Master) ต้องคำนวณ `remain_quota` ของ Member ใต้ Share Master ใหม่ใน tx เดียว · ความหมาย Remain ตอน settle ตัดสินพร้อม module Settle |
| MA3 | `update-commission` → `update-pt` (`pt` + `commission_percent`) | **อนุมัติเนื้อหา** — path ถูกรวมเข้า `members/detail/update` (ขั้นที่ 2) |
| MA4 | `user_members.cnf` สายชั้นบน | **อนุมัติ** |
| MA5 | ยอด Member → `user_members.credit` + DROP `user_member_balances` | **อนุมัติ** + กฎ: ทุกการแก้ `credit` ต้อง lock แถว `user_members` FOR UPDATE ก่อน |
| MA6 | เปลี่ยนเลข migration ที่ merge แล้ว | **รับทราบ** + กติกา: ก่อน merge ให้ rebase แล้วสร้างเลข migration ใหม่ถ้าเลขต่ำกว่าเลขล่าสุดบน dev |
| M5 | **เส้นใหม่ `POST /manage/members/list`** (แทน MN3) | แบบเดียวกับ `agents/list` (E6/E6-1): list **Member ที่เป็นลูกตรงของตัวเอง** ไว้ทำ dropdown → เลือกแล้วเรียก `members/detail/get` เอาค่ามาตั้งต้นฟอร์มสร้าง Member · request `{}` + `keyword` ไม่บังคับ 4–32 ตัว (username) · response `id` `username` `name` `status` · A→Z · ไม่มี pagination · สูงสุด 500 · สิทธิ์ sub `member` view · error `402303` |
| MN1 | โครง endpoint ใหม่ | **M1 `members/create` · M2 `members/detail/get` · M3 `members/detail/update` (รวม `info` + `pt`) · M4 `members/status/update` · M5 `members/list`** · ไม่คง path เดิม · รายชื่อ/ค้นหา Member ทั้งสายใช้ `downlines/list` (E2) |
| MN2 | `detail/update` ไม่ส่ง section เลย | **`422` "ต้องส่งอย่างน้อย 1 section"** — ใช้กับ `agents/detail/update` ด้วย |
| MN4 | สิทธิ์ sub | ทุกเส้น `member` view/edit (+ `payment` edit เมื่อส่ง `balance`) · `pt` ใน detail แสดงเสมอ |
| M3-1 | `members/detail/update` | section `info` (ชื่อ · เบอร์คู่ E1-5 · ส่งบาง field ได้) · section `pt.minigame` ส่งได้แค่ `pt` + `commission_percent` (ครบทั้ง 2) · ส่ง `remain_quota`/`pt_from_parent`/`force`/`status` = `422` · ผู้สร้างโดยตรงเท่านั้น (หลาน `402304` · นอกสาย `402402`) · tx เดียว · log `UPDATE_INFO`/`UPDATE_PT` แยกแถว · ค่าเดิม = ไม่เขียน log · ไม่ต้อง passcode |
| MQ1 | เบอร์ Member | เบอร์แบบ E1-5 + migration แบบ B1 · **ห้ามซ้ำเฉพาะภายใต้ parent (ผู้สร้าง) เดียวกัน** — Member คนเดียวกันอาจเล่นหลายเว็บ/หลายสาย ใช้เบอร์เดียวกันได้ถ้าคนละ parent · unique index `(parent, phone_country_code, phone)` เฉพาะแถวที่มีเบอร์ · ซ้ำ = `402403` · Member กับ agent ใช้เบอร์เดียวกันได้ |
| MQ2 | ลำดับเช็ค `members/create` | แก้โค้ดให้ตาม U3 (เหมือนฝั่ง agent) |
| MQ3 | error ไม่ลงขั้น `pt` (0.5) / commission (0.1) | คงโค้ด: ไม่ลงขั้น = `422` · เกินเพดาน = `402305` / `402309` · เขียนใน spec |
| MQ4 | แก้ `pt` ของ Member ที่ไม่มีแถวค่าตั้ง (บั๊ก: 200 + log แต่ไม่บันทึก) | ข้ามเงียบ **ไม่เขียน log** (เหมือน E4-1) |
| MQ5 | ส่งสถานะเดิม | 200 ไม่เขียน log — ใช้กับ `agents/status/update` ด้วย |
| MQ6 | Member ฝั่ง Seamless `balances` | คง `amount: 0` (ACC-19) · spec เขียนว่า Seamless = 0 เสมอ · หน้าบ้านซ่อน |
| MQ7 | ผล LOCKED / SUSPENDED ของ Member ต่อ login / เดิมพัน | เลื่อนไปกำหนดตอน review Player Auth (ร่าง: LOCKED = login ไม่ได้ + ตัด session · SUSPENDED = login/ดูได้ เดิมพันไม่ได้) |

## รายการแก้ (ส่งให้ maofoy — ทำใน branch ใหม่แล้วเปิด PR ให้ lead review)

### Spec `docs/modules/member_management.md`
- S1 หัวเอกสาร: บันทึกการอนุมัติ MA1–MA6 + การตัดสินทั้งหมดในไฟล์นี้ (lead zerph 2026-10-09)
- S2 หัวข้อ Endpoints เขียนใหม่ตาม **M1–M5** (path N1 · ตัวอย่าง request/response · ตาราง field · error) · ลบ `detail` / `update-info` / `update-status` / `update-pt`
- S3 MGMT-51 สิทธิ์ = `member` view/edit (+ `payment` edit เมื่อส่ง `balance`) · ลบ `pt` (P4) · MGMT-29 `pt` แสดงเสมอ
- S4 MGMT-08 + ตัวอย่าง + schema: เบอร์ E1-5 · unique ภายใต้ parent เดียวกัน (MQ1)
- S5 MGMT-09A: แก้ข้อมูล + PT เส้นเดียว (M3) · section ไม่บังคับ แต่ต้องมีอย่างน้อย 1 (MN2) · ค่าเดิมไม่เขียน log (MQ4, MQ5)
- S6 MGMT-15A: `request_id` ของคนอื่น/เส้นอื่น = `422` (Q-R4) · ข้อยกเว้นสิทธิ์ `payment` เช็คหลัง DTO (U3) · แก้ `credit` ต้อง lock แถว `user_members` FOR UPDATE (MA5) · Seamless `balances` = 0 (MQ6)
- S7 MGMT-21: error `422` ไม่ลงขั้น (MQ3) · sync `remain_quota` ทุกทาง รวม CSM → Share Master (MA2) · MGMT-16 `created_by` = คนกดจริง (Q-R2) · H1: Share Reseller / Share Master สร้าง Member ได้ · Share Master ตั้ง `pt` ของ Member เอง (H4)
- S8 test case: H1 · Seamless + `balance` → `422` · replay `request_id` · log `target_type = MEMBER` · detail/update / status/update ค่าเดิม · เบอร์ซ้ำภายใต้ parent เดียวกัน `402403` และคนละ parent ผ่าน
- S9 ล้างข้อความค้าง/ขัด (spec: หัวเอกสาร APPROVED vs "รอ lead" · comment คอลัมน์ `remain` "ยังไม่ทำ" · บรรทัด 265 "boiledegg ตัดสิน" → lead อนุมัติ U1 · MGMT-09A "คนละเส้น")
- S10 MQ7 ใส่เป็น open item → Player Auth

### เอกสารอื่น
- S11 `docs/frontend/member_management.md`: path ใหม่ · เบอร์ · สิทธิ์ · body ของ detail/update · ลบบรรทัด 6 "ยังไม่อยู่ใน dev"
- S12 `docs/MIGRATION_STATUS.md:28` อัปเดตสถานะ sync · `docs/lead/bo-management-member.md` ปิดรายการข้อ 3
- S13 (boiledegg) agent_management C6: CSM → Share Master ต้อง sync `remain_quota` ของ Member ใต้ Share Master ใน tx เดียว (MA2)

### โค้ด
- C1 `routes.go`: path ใหม่ M1–M5 · สิทธิ์ `member` (B3 — boiledegg เปลี่ยนสิทธิ์ให้แล้ว/กำลังทำ ระวัง conflict)
- C2 `members/detail/update`: DTO section `info` / `pt` ไม่บังคับ (อย่างน้อย 1 → `422`) · tx เดียว · lock ตามลำดับเดิม (Member → `agent_game_settings` SHARE → `user_member_game_settings` UPDATE) · log แยกต่อ section ที่เปลี่ยนจริง · ลูกตรงเท่านั้น · ไม่มีแถวค่าตั้ง = ข้ามเงียบไม่เขียน log (MQ4)
- C3 `members/status/update`: ค่าเดิม = ไม่เขียน log (MQ5)
- C4 `members/list` ใหม่ (M5): ลูกตรงที่เป็น Member · `keyword` 4–32 · `id` `username` `name` `status` · A→Z · สูงสุด 500
- C5 `read_service.go` + `view_dto.go`: `pt` แสดงเสมอ (ลบ `CanSeePT`)
- C6 `create_member_service.go`: ลำดับเช็คตาม U3 (MQ2) · `amounts_minor` → `amounts` (E1-4)
- C7 เบอร์ E1-5 ฝั่ง Member · **unique index `(parent, phone_country_code, phone)` WHERE phone <> ''** แทน `uq_user_members_phone` · migration แปลงเบอร์ตาม B1 (ทำพร้อม C7 ของ agent_management — ประสานกับ boiledegg ไม่ให้ทับไฟล์กัน)
- C8 แก้ comment ที่ชี้ spec ผิด (`create_dto.go:1`, `update_dto.go:7`, `create_member_service.go:1`, `update_service.go:20`, `user_member_game_setting_repository.go:13`, `user_member_repository.go:93`)
- C9 test ตาม S8 · **รัน integration test ของ Member ให้ผ่าน** (`scripts/testenv.ps1 up` → `make test-integration`) แนบผลใน PR · แจ้ง frontend (path, เบอร์, สิทธิ์, body)
