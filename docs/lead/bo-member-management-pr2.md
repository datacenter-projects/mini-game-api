# Member_Management — ทำตาม lead review (C1–C9) · 2026-10-11

จาก boiledegg (ทำแทน maofoy) · review: [docs/reviews/member-management.md](../reviews/member-management.md)
branch `boiledegg/bo/member_management` · merge เข้า `dev` แล้ว (`6d75062`) · **ยังไม่ deploy** — รอ lead ดูก่อน

- PR 1 `82fcde9` — spec เขียนใหม่ตาม S1–S12 ([docs/modules/member_management.md](../modules/member_management.md) · เอกสารหน้าบ้าน [docs/frontend/member_management.md](../frontend/member_management.md))
- PR 2 `430fe9a` — โค้ด C1–C9 (สรุปด้านล่าง)

## 1. ผลแต่ละข้อ

| ข้อ | ผล |
|---|---|
| C1 | เส้นใหม่ M1–M5: `members/create` · `members/detail/get` · `members/detail/update` · `members/status/update` · `members/list` · ทุกเส้นใช้สิทธิ์ `member` (view / edit) · path เดิม (`members/detail` · `update-info` · `update-status` · `update-pt`) ได้ `404` |
| C2 | `members/detail/update` · section `info` / `pt` ไม่บังคับ แต่ไม่ส่งเลย = `422` "ต้องส่งอย่างน้อย 1 section" · tx เดียว (pt ผิด = info ไม่บันทึก) · lock Member → `agent_game_settings` SHARE → `user_member_game_settings` UPDATE · log `UPDATE_INFO` / `UPDATE_PT` แยกแถวเฉพาะที่เปลี่ยนจริง · ผู้สร้างโดยตรงเท่านั้น (`402304` / `402402`) · ไม่มีแถวค่าตั้ง = ข้ามเงียบ ไม่เขียน log (MQ4) |
| C3 | `members/status/update` ส่งสถานะเดิม = `200` ไม่เขียน log (MQ5) |
| C4 | `members/list` ใหม่ — Member ลูกตรง · `keyword` 4–32 · response `id` `username` `name` `status` · A→Z · สูงสุด 500 · ไม่มี = `[]` |
| C5 | `pt` ใน `members/detail/get` แสดงเสมอ · ลบ `CanSeePT` (ทั้งที่เรียกและ shim) |
| C6 | ลำดับเช็ค `members/create` ตาม U3: ประเภทผู้สร้าง `402301` → username `402401` → เบอร์ `402403` → balance (Seamless `422` → ยอดไม่พอ `402312`) → `pt` `402305` → commission `402309` · log `INITIAL_BALANCE` ใช้ key `amounts` |
| C7 | เบอร์ 2 field `phone_country_code` + `phone` ทุก request / response (รวมแถว Member ใน `downlines/list`) · unique `(agent_id, phone_country_code, phone)` · migration `20261011120000` แปลงเบอร์เดิมแบบ B1 (ซ้ำใต้ผู้สร้างเดียวกันเก็บ id น้อยสุด) · ที่ถูกล้างบันทึกใน `migration_phone_cleared` (`table_name = user_members` · คอลัมน์ใหม่ `kept_member_id`) · ไม่แก้ไฟล์ migration ของฝั่ง agent |
| C8 | แก้ comment ที่ชี้ spec ผิดครบ 6 จุด |
| C9 | test ใหม่ `member_management_integration_test.go` (เบอร์ · detail/update · สถานะ · list · สิทธิ์ sub · ลำดับเช็ค / `request_id` / Seamless / log · path เดิม 404) + test migration เบอร์ Member · test เดิมเปลี่ยนไปใช้ path ใหม่ |

ลบ helper เบอร์ field เดียวที่ค้างใน `agent_management` (ใช้แค่ฝั่ง Member): `ValidateAccountFields` · `UpdateInfoRequest` · `CheckPhoneFree` · `LockAndCheckIdentity` · `IsValidPhone` · section `info` ใช้ type เดียวกับ `agents/detail/update`

## 2. ผล test

- lint 0 issues · structure OK · build / vet ผ่าน · unit + integration ผ่านทุก package (Windows test env)
- `-race` ไม่ได้รันในเครื่องนี้ (ไม่มี cgo) — CI รันให้ตอนออก tag

## 3. ขอ lead ก่อน deploy

1. ดูผลตามข้อ 1 · ถ้าผ่านจะ deploy dev: migrate RDS `20261011120000` → ออก tag
2. **migration นี้ล้างเบอร์ Member** ที่แปลงไม่ได้ (ไม่ขึ้นต้น `0` / `66` 11 หลัก) และที่ซ้ำกันใต้ผู้สร้างเดียวกัน — รายการเก็บใน `migration_phone_cleared` · จะรายงานจำนวนหลัง migrate
3. **frontend ต้องเปลี่ยนพร้อม deploy**: path ใหม่ (path เดิมได้ `404`) · เบอร์ 2 field · body ของ `members/detail/update` — รายละเอียด [docs/frontend/member_management.md](../frontend/member_management.md) หัวข้อ 4

## 4. ถึง maofoy

งาน C1–C9 ทำแล้วบน `dev` · branch `maofoy/bo/member-management` เก่ากว่า `dev` — ถ้าจะทำต่อให้แตก branch ใหม่จาก `dev`
open item ที่ยังเหลือของ module: MQ7 (ผล `LOCKED` / `SUSPENDED` ของ Member → Player Auth) · ความหมาย Remain ตอน settle (module Settle)
