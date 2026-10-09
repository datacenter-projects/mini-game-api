# สถานะ — `maofoy/bo/member-management`

## 1. ภาพรวม

| | |
|---|---|
| branch | `maofoy/bo/member-management` · ล่าสุด `dbb15b7` |
| spec | `docs/modules/member_management.md` — **APPROVED** (ร่วมกับ `agent_management` 2026-10-07) · ส่วนที่แก้ 2026-10-09 **รอ lead อนุมัติ** (หัวข้อ 3 ข้อ 1–5) |
| merge | แยก module + เปลี่ยนชื่อตารางเข้า `dev` แล้ว (`6659078` · `e4fbf76`) · PT ต่อ Member, `cnf`, `credit` (`9280153` · `a1467cd` · `dbb15b7`) **ยังไม่ merge** — รออนุมัติ |
| migration dev | `20261009120050` · `20261009120100` **ยังไม่รันบน `dev_minigame`** — รันหลัง merge เข้า `dev` (`120100` DROP `user_member_balances` ที่โค้ดใน `dev` ยังใช้อยู่) |
| test | `make check` ผ่าน · integration test **ยังไม่ได้รัน** (รอเครื่องที่มี test env) |

> ห้ามรัน branch นี้กับ `dev_minigame` โดยเปิด `MIGRATE_ON_START=true` จนกว่าจะ merge — migration จะ DROP `user_member_balances` ทำให้เส้น Member ใน `dev` พัง

## 2. งาน

| ส่วน | สถานะ | หมายเหตุ |
|---|---|---|
| แยก `/manage/members/*` ออกจาก `agent_management` เป็น `member_management` | ✅ | `create` · `detail` · `update-info` · `update-status` · ใช้ helper ร่วมจาก `service/agent_management` · เข้า `dev` แล้ว |
| เปลี่ยนชื่อตาราง `members*` → `user_members*` | ✅ | migration `20261009000000_member_tables_rename` · รันบน `dev_minigame` แล้ว |
| PT ต่อ Member: `pt` + `remain_quota` · `update-commission` → `update-pt` | ✅ รออนุมัติ | migration `20261009120050_member_management_member_pt` (ยังไม่รันกับ DB ใด) |
| `user_members.cnf` (สายชั้นบน) + `user_members.credit` (ยอดเงิน float) · DROP `user_member_balances` | ✅ รออนุมัติ | migration `20261009120100_member_management_member_cnf_credit` (ยังไม่รันกับ DB ใด) · แก้ `TransferInitialBalance` (กรณี Member) และยอด Member ในดาวน์ไลน์ใน `agent_management` |
| `agents/update-pt` กันลดต่ำกว่า `pt` ของ Member + sync `remain_quota` (R1–R3) | ⏳ | อยู่ใน `agent_management` — รอผู้ดูแล module นั้น (spec หัวข้อ 7) |
| integration test ของ Member | ⏳ | เขียนแล้ว (`TestMemberPT` · `credit` · `cnf` และแก้ test เดิม) ยังไม่ได้รัน |
| migrate `dev_minigame` | ⏳ | หลัง merge เข้า `dev` (`make migrate-up` หรือ migration job ตอน deploy) |

เอกสารหน้าบ้าน: `docs/frontend/member_management.md`

## 3. รอ lead

1. **agent ถือ PT สู้กับ Member แต่ละคนแยกกัน (MGMT-21)**
   - เดิม: Member ไม่ถือ PT มีแค่ commission · agent ถือสู้ Member ทุกคนด้วยค่าเดียว (`agent_game_settings.pt_bp`)
   - ใหม่: ผู้สร้างตั้ง `pt` ให้ Member แต่ละคน (0 ถึงค่าที่ตัวเองได้รับ ทีละ 0.5% · เกิน = `402305`) เก็บที่ `user_member_game_settings.pt_bp`
   - เหตุผล: agent ถือสู้กับ Member แต่ละคนได้ไม่เท่ากัน (แบบเดียวกับ `pt_our` ของ member ใน askmelotto)
2. **`remain_quota` ของ Member (MGMT-21 / MGMT-29)**
   - ระบบคิดเอง = ค่าที่ผู้สร้างได้รับ − `pt` แล้วเก็บไว้ใช้คิด Remain ตอน settle (client ส่งไม่ได้)
   - Member ไม่มี `force` · ส่ง `pt_from_parent` / `force` / `remain_quota` / `status` = `422`
   - response ของ `members/detail` และแถว Member ใน `downlines/list` / `search` มี `pt` และ `remain_quota` เพิ่ม (เพิ่ม field อย่างเดียว)
3. **เปลี่ยนเส้น `members/update-commission` → `members/update-pt`**
   - body: `{ id, pt: { minigame: { pt, commission_percent } } }` · สิทธิ์ `pt` edit เหมือนเดิม
   - FE ต้องเปลี่ยน path และส่ง `pt` ตอนสร้าง Member
4. **requirement ของ `agents/update-pt`** (`agent_management` · ยังไม่ทำ · spec หัวข้อ 7) — ขอให้ตัดสินร่วมกับผู้ดูแล `agent_management` ว่าใครทำ
   - R1: ค่าที่ให้ agent ลูกใหม่ต้อง ≥ `pt` ที่ลูกถือสู้กับ Member แต่ละคน (`402306`)
   - R2: สำเร็จแล้ว sync `remain_quota` ของ Member ทุกคนที่ลูกสร้าง ใน tx เดียวกัน
   - R3: lock `agent_game_settings` ก่อน `user_member_game_settings`
   - ถ้าไม่ทำ: แม่แก้ค่าที่ให้ agent แล้ว `remain_quota` ของ Member ค้างค่าเก่า (ตอนนี้ยังไม่มี Member ในระบบ)
5. **`user_members.cnf` + `user_members.credit`**
   - `cnf` = สายชั้นบนของ Member รูปแบบเดียวกับ `user_agents.cnf` (MGMT-61) = `cnf` ของผู้สร้าง + ผู้สร้าง
   - ยอดของ Member ย้ายจากตาราง `user_member_balances` ไป `user_members.credit` และ DROP ตารางเดิม (ยังไม่มีข้อมูล) · response ยังเป็น `balances: [{currency, amount}]`
   - **`credit` เป็น float64 (DOUBLE PRECISION) ตามที่ทีมตกลง — ขัด CLAUDE.md กฎข้อ 9 ("ห้ามใช้ float กับเงิน") ต้องแก้กฎผ่าน review ทีม**
     · `balance_ledger` ยังเป็น int64 หน่วยย่อย แปลงด้วย `utils.MinorToCredit` / `CreditToMinor`

แจ้งเพิ่ม (ไม่ต้องอนุมัติ)

- `pt_bp` ของ agent และ `agents/update-hold` ไม่มีผลกับ Member แล้ว — จะเลิกใช้ไหม ให้ผู้ดูแล `agent_management` ตัดสิน
- เปลี่ยนเลข migration ที่ merge แล้ว `20261008170000_member_tables_rename` → `20261009000000` เพราะเลขชนกับ
  `20261008170000_account_api_keys_for_all` จน goose panic — ขัดกฎข้อ 20 จึงแจ้งไว้
- `20261009040811_member_management_member_pt` → `20261009120050` และตั้ง `20261009120100` เอง (ไม่ใช้ `make migration`)
  เพราะ `dev_minigame` รัน `20261009120000_agent_management_chain` แล้ว เลขที่น้อยกว่าจะถูก goose ปฏิเสธ (missing migration)
  · ไฟล์ยังไม่เคย merge / รันที่ไหน จึงไม่ขัดกฎข้อ 20

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
| 2026-10-09 | `dbb15b7` | เปลี่ยนเลข migration member_pt เป็น `20261009120050` · ยังไม่ migrate dev (รอ merge) |
