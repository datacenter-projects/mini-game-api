# สถานะ — `maofoy/bo/member-management`

## 1. ภาพรวม

| | |
|---|---|
| branch | `maofoy/bo/member-management` · เท่ากับ `dev` @ `3603c50` |
| spec | `docs/modules/member_management.md` — **APPROVED** (ร่วมกับ `agent_management` 2026-10-07) · ส่วนที่แก้ 2026-10-09 **รอ lead อนุมัติ** (หัวข้อ 3 ข้อ 1–3, 5) · กฎข้อ 9 (float) lead อนุมัติแล้ว 2026-10-09 |
| merge | **เข้า `dev` แล้วทั้งหมด** (`625255a` · 2026-10-09) — เข้าก่อน lead อนุมัติ spec ส่วนที่แก้ |
| migration dev | `dev_minigame` รันครบถึง `20261009140000` แล้ว · `20261009140100` (parent_id) รอรันหลัง merge |
| test | `make check` ผ่าน · integration test **ยังไม่ได้รัน** (รอเครื่องที่มี test env) |

## 2. งาน

| ส่วน | สถานะ | หมายเหตุ |
|---|---|---|
| แยก `/manage/members/*` ออกจาก `agent_management` เป็น `member_management` | ✅ | `create` · `detail` · `update-info` · `update-status` · `update-pt` · ใช้ helper ร่วมจาก `service/agent_management` |
| เปลี่ยนชื่อตาราง `members*` → `user_members*` | ✅ | migration `20261009000000_member_tables_rename` |
| PT ต่อ Member: `pt` + `remain_quota` · `update-commission` → `update-pt` | ✅ รออนุมัติ | migration `20261009120050_member_management_member_pt` · คอลัมน์เป็น `pt` / `remain` / `commission` (float) หลัง `20261009130000` |
| `user_members.cnf` (สายชั้นบน) + `user_members.credit` (ยอดเงิน float) · DROP `user_member_balances` | ✅ รออนุมัติ | migration `20261009120100_member_management_member_cnf_credit` |
| `user_member_game_settings.parent_id` = agent ผู้สร้าง (MGMT-62 ของ Member) | ✅ lead อนุมัติ 2026-10-09 | migration `20261009140100_member_management_settings_parent` (ยังไม่รันบน dev — รันหลัง merge เพราะ NOT NULL) |
| `agents/update-pt` กันลดต่ำกว่า `pt` ของ Member + sync `remain_quota` (R1–R3) | ✅ | boiledegg ทำใน `agent_management` MGMT-24 (`3603c50`) |
| ลบ `pt` ค่าเดียวของ agent และ `agents/update-hold` | ✅ | boiledegg (`20261009140000_agent_management_drop_own_pt`) |
| เงินและ % เป็น float ปัด 4 ตำแหน่ง (กฎข้อ 9) | ✅ | boiledegg (`20261009130000_money_percent_float`) · lead อนุมัติ |
| integration test ของ Member | ⏳ | เขียนแล้ว (`TestMemberPT` · `credit` · `cnf`) ยังไม่ได้รัน |

เอกสารหน้าบ้าน: `docs/frontend/member_management.md`

## 3. รอ lead

1. **agent ถือ PT สู้กับ Member แต่ละคนแยกกัน (MGMT-21)**
   - เดิม: Member ไม่ถือ PT มีแค่ commission · agent ถือสู้ Member ทุกคนด้วยค่าเดียว (ลบแล้ว)
   - ใหม่: ผู้สร้างตั้ง `pt` ให้ Member แต่ละคน (0 ถึงค่าที่ตัวเองได้รับ ทีละ 0.5% · เกิน = `402305`) เก็บที่ `user_member_game_settings.pt`
   - เหตุผล: agent ถือสู้กับ Member แต่ละคนได้ไม่เท่ากัน (แบบเดียวกับ `pt_our` ของ member ใน askmelotto)
2. **`remain_quota` ของ Member (MGMT-21 / MGMT-29)**
   - ระบบคิดเอง = ค่าที่ผู้สร้างได้รับ − `pt` แล้วเก็บไว้ใช้คิด Remain ตอน settle (client ส่งไม่ได้) · sync เมื่อค่าที่ผู้สร้างได้รับเปลี่ยน
   - Member ไม่มี `force` · ส่ง `pt_from_parent` / `force` / `remain_quota` / `status` = `422`
   - response ของ `members/detail` และแถว Member ใน `downlines/list` / `search` มี `pt` และ `remain_quota` เพิ่ม
3. **เปลี่ยนเส้น `members/update-commission` → `members/update-pt`**
   - body: `{ id, pt: { minigame: { pt, commission_percent } } }` · สิทธิ์ `pt` edit เหมือนเดิม
   - FE ต้องเปลี่ยน path และส่ง `pt` ตอนสร้าง Member
4. ~~requirement R1–R3 ของ `agents/update-pt`~~ — **ทำแล้ว** (boiledegg · `agent_management` MGMT-24)
5. **`user_members.cnf` + `user_members.credit`**
   - `cnf` = สายชั้นบนของ Member รูปแบบเดียวกับ `user_agents.cnf` (MGMT-61) = `cnf` ของผู้สร้าง + ผู้สร้าง
   - ยอดของ Member ย้ายจากตาราง `user_member_balances` ไป `user_members.credit` (float ตามกฎข้อ 9 ที่อนุมัติแล้ว) และ DROP ตารางเดิม · response ยังเป็น `balances: [{currency, amount}]`

แจ้งเพิ่ม (ไม่ต้องอนุมัติ)

- งานนี้ merge เข้า `dev` และ migrate `dev_minigame` แล้วก่อนอนุมัติ — ถ้า lead ไม่อนุมัติข้อใด ต้องทำ migration ใหม่เพื่อย้อน
- เปลี่ยนเลข migration ที่ merge แล้ว `20261008170000_member_tables_rename` → `20261009000000` เพราะเลขชนกับ
  `20261008170000_account_api_keys_for_all` จน goose panic — ขัดกฎข้อ 20 จึงแจ้งไว้
- `20261009040811_member_management_member_pt` → `20261009120050` และตั้ง `20261009120100` เอง (ไม่ใช้ `make migration`)
  เพราะ `dev_minigame` รัน `20261009120000_agent_management_chain` แล้ว เลขที่น้อยกว่าจะถูก goose ปฏิเสธ · ไฟล์ยังไม่เคย merge / รันตอนเปลี่ยน
- log `INITIAL_BALANCE` ใช้ key `amounts_minor` แต่ค่าเป็นทศนิยมแล้ว (หลังกฎข้อ 9) — ใช้ทั้งฝั่ง agent และ Member จะเปลี่ยนชื่อพร้อมกันไหม ให้ผู้ดูแล `agent_management` ตัดสิน

## 4. ประวัติอัปเดต

| วันที่ | commit | อัปเดต |
|---|---|---|
| 2026-10-09 | `954bb9b` | ย้ายไฟล์ตามโครงสร้างใหม่ (repository แยกตาม module) · แยก `member_management` |
| 2026-10-09 | `6659078` | merge เข้า `dev` · เปลี่ยนชื่อตาราง `user_members*` |
| 2026-10-09 | `e4fbf76` | เปลี่ยนเลข migration rename เป็น `20261009000000` (เลขชน) · รันบน `dev_minigame` |
| 2026-10-09 | `9280153` | PT ต่อ Member (`pt` + `remain_quota`) · `update-commission` → `update-pt` · requirement R1–R3 ให้ `agent_management` |
| 2026-10-09 | `154da03` · `5368426` · `f588b38` | ไฟล์สถานะ lead (แยก agent / member) · แก้ถ้อยคำ spec |
| 2026-10-09 | `cb5347f` | merge `dev` เข้า branch (ได้ `user_agents.cnf` · `agent_game_settings.parent_id` จาก `agent_management`) |
| 2026-10-09 | `a1467cd` | `user_members.cnf` + `credit` (float) · DROP `user_member_balances` |
| 2026-10-09 | `dbb15b7` | เปลี่ยนเลข migration member_pt เป็น `20261009120050` |
| 2026-10-09 | `625255a` | merge เข้า `dev` (fast-forward) · migrate `dev_minigame` ถึง `20261009120100` |
| 2026-10-09 | `3603c50` | merge `dev` เข้า branch: กฎข้อ 9 float (lead อนุมัติ) · R1–R3 · ลบ `pt` ของ agent / `update-hold` (boiledegg) |
| 2026-10-09 | (ยังไม่ commit) | `user_member_game_settings.parent_id` (lead อนุมัติ) |
