# การจัดการสมาชิก (`agent_management`) — Spec

- สถานะ: **APPROVED**
- อนุมัติโดย: lead (zerph) · วันที่: 2026-10-07
- แก้หลังอนุมัติ (2026-10-08 · boiledegg ตัดสิน · lead (zerph) รับทราบและอนุมัติ 2026-10-08): MGMT-07 ชื่อเป็นภาษาไทยได้ · MGMT-41 เบอร์โทรของ sub ซ้ำได้ · หัวข้อ 1 ยอดเงินตั้งต้นอยู่ใน module นี้ (ตรงกับ MGMT-15A) · ชื่อเส้น: สร้าง `/create` · แก้ `/update-info` `/update-status` `/update-pt` `/update-commission` `/update-hold` `/update-games` · id ของบัญชีส่งใน body ทุกเส้น (ไม่อยู่ใน path) · รายชื่อ (`/list`) และรายละเอียดเป็น `POST` · `page` / `limit` ใน body · เพิ่ม MGMT-27A ค้นหาทั้งสาย `/manage/downlines/search` · `pt` ใช้ key ระบบ `minigame` (ชุดเดียวต่อระบบ) + `created_at` `created_by` `updated_at` `updated_by` (key ตอนส่ง `game` → `minigame`) · sub: `/manage/subaccounts/list` `/detail` `/create` `/update-info` `/update-status` · MGMT-20 `pt.status` = รับ PT ไหม (`false` = เกมยังเปิด แต่ไม่รับ PT) · เปิด / ปิดเกมใช้ `status_game` อย่างเดียว · เส้นสร้างแยก 2 เส้นเหมือนเดิม (agents / members) · เปิด `update-games` ใน module นี้ · เพิ่ม MGMT-27B ค้นหาบัญชีของ ADMIN `/admin/accounts/search`
- แก้เพิ่มหลังอนุมัติรอบนั้น (2026-10-08 · boiledegg ตัดสิน · lead (zerph) รับทราบและอนุมัติ 2026-10-08): ไม่มีเมนูสิทธิ์ `account` แล้ว (8 เมนู — Profile / 1.3 เปิดได้เสมอ) · MGMT-04 ทุกบัญชีฝั่ง agent ได้ Key
- แก้ 2026-10-09 (boiledegg ตัดสิน): ลำดับเช็คบนลงล่างตาม field ใน body และข้อความ error บอก field + ค่าที่ตั้งได้ (หัวข้อ 7.1)
- แก้ 2026-10-09 (boiledegg · ทีมตกลง · CLAUDE.md กฎข้อ 9 lead (zerph) อนุมัติ 2026-10-09): **เงินและ % เก็บเป็นทศนิยม float ปัด 4 ตำแหน่ง** แทนจำนวนเต็ม (bp / หน่วยย่อย) · request รับไม่เกิน 4 ตำแหน่ง · API ส่งค่าตามที่เก็บ หน้าบ้านปัด 3 ตำแหน่งเอง · column ตัด `_bp` (MGMT-17, MGMT-15A, หัวข้อ 6)
- แก้ 2026-10-09 (boiledegg · lead (zerph) อนุมัติ 2026-10-09): เก็บสายชั้นบน `user_agents.cnf` (MGMT-61) และ `agent_game_settings.parent_id` (MGMT-62)
- แก้หลังอนุมัติ (2026-10-08 · maofoy · boiledegg ยืนยัน 2026-10-09): เปลี่ยนชื่อตารางของ Member ให้ตรงกับ `models.UserMember*` — `members` → `user_members` · `member_game_settings` → `user_member_game_settings` · `member_balances` → `user_member_balances` · คอลัมน์ `member_id` → `user_member_id` (หัวข้อ 6 · migration `20261009000000_member_tables_rename.sql` · เปลี่ยนชื่ออย่างเดียว ข้อมูลไม่เปลี่ยน)
- ชื่อ module ในโค้ด: `agent_management` (`controllers/agent_management`, `dto/agent_management`, `service/agent_management`, `core/agent_management`) · เส้น `/manage/members/*` แยกไปที่ `member_management` (`controllers/` `dto/` `service/member_management` — lead อนุมัติ 2026-10-08 · spec [member_management.md](member_management.md) แยกเอกสาร 2026-10-09) · กฎ business ยังอยู่ที่ `core/agent_management` และ spec ฉบับนี้
- เมนู: 2 การจัดการสมาชิก — เพิ่มบัญชี · รายชื่อดาวน์ไลน์ (ไล่ลงได้ถึง Member) · แก้ไข · บัญชีย่อย (เพิ่ม · รายชื่อ · รายละเอียด · แก้ · เปลี่ยนสถานะ)
- ที่มาของ rule: เอกสารของ lead (Company Hierarchy, PT, PT Force, PT Remain, PT Commission, System Overview) ·
  review ของ lead ต่อฉบับ 647d682 · ภาพหน้าจอตัวอย่าง + คำตอบของ boiledegg (2026-10-05 / 06) · ไม่ได้ extract จากโค้ดเก่า
- **ขอบเขต: ทั้ง module ในฉบับเดียว** (บริษัท · ตัวแทน · เอเย่นต์ · สมาชิก · บัญชีย่อย · สิทธิ์ · คัดลอกการตั้งค่า) — ไม่แบ่งช่วง
- ทีมเราทำฝั่ง API — การแสดงผล (dropdown, ฟอร์ม, รูปแบบตัวเลขบนจอ) เป็นของหน้าบ้าน

> เอกสารนี้คือ source of truth ของ module — โค้ดและ test ต้องตรงกับเอกสารนี้ ถ้าไม่ตรงให้แก้โค้ด
> หรือแก้เอกสารผ่านการอนุมัติใหม่ ห้ามอ้างเอกสารของโปรเจกต์เก่า

## 1. หน้าที่ของ module

สร้างและจัดการบัญชีทุกชั้นในสาย: Company · Share · Agent · Member · บัญชีย่อย (sub) — สร้าง, ดูรายชื่อไล่ลงทีละชั้น,
ดูรายละเอียด, แก้ข้อมูล, แก้สถานะ, ตั้ง / แก้ค่าหุ้นส่วน (PT · Force · Remain · Commission) และเปิด / ปิดเกม,
คัดลอกการตั้งค่า และกำหนดสิทธิ์ของ sub (ระบบสิทธิ์ของทั้งหลังบ้านกำหนดที่นี่ครั้งเดียว)

**ไม่อยู่ใน module นี้:** การเติม / ถอน / โอนเงินหลังสร้างบัญชี (module การชำระเงิน — ใช้ตาราง `agent_balances` · `user_members.credit` (ยอดของ Member — member_management.md · 2026-10-09 รอ lead อนุมัติ) · `balance_ledger` ที่สร้างใน module นี้ต่อ) · **ยอดเงินตั้งต้นตอนสร้างบัญชีอยู่ใน module นี้** (MGMT-15A — แก้ 2026-10-08) ·
การคิด PT / Force / Remain / Commission ตอน settle (module เดิมพัน) · login ของ Member (module หน้าบ้าน) ·
Key / ลิงก์ตอบกลับ / IP (account 1.3 — module นี้แค่สร้าง Key ตอนสร้างเจ้าของ ACC-05)

## 2. คำศัพท์

| คำ | ความหมาย |
|---|---|
| ฝั่ง agent | Superadmin · Company · Share · Agent (ตาราง `user_agents`) |
| Member | ผู้เล่น (ตาราง `user_members` แยกจากฝั่ง agent) |
| ผู้สร้าง | บัญชีที่สร้างบัญชีนั้น (`parent_id` / `agent_id` ของ member) — ลูกตรง = บัญชีที่ตัวเองสร้าง |
| สายล่าง | ทุกบัญชีใต้ตัวเองทุกชั้น ไม่นับตัวเอง |
| `user_type` | ประเภทบัญชี (MGMT-01) — ตรงกับ account ACC-12 |
| กลุ่ม PT | ระบบที่มีเรื่อง PT ใช้เป็น key ของ `pt` (`minigame` = ทุกเกม minigame · ระบบอื่น เช่น `askmelotto` เพิ่มได้โดยไม่เปลี่ยนโครงสร้าง) · รับค่าชุดเดียวต่อระบบ · เก็บและแสดงต่อเกม |
| ได้รับ / ถือ / ให้ลูก | ได้รับ (`pt_from_parent`) = PT ที่ผู้สร้างให้ · ให้ลูก = PT ที่แบ่งให้ลูกแต่ละคน (ไม่เกินที่ได้รับ) · ถือ (`pt`) = ส่วนที่ถือจาก Member ใต้ตัวเองตรงๆ (ตั้งเอง) · ผู้สร้างถือในสายของลูก = ได้รับ − ให้ลูกคนนั้น |
| ทศนิยม 4 ตำแหน่ง | เงินและ % เก็บเป็น `float64` / `DOUBLE PRECISION` ปัด 4 ตำแหน่ง เช่น 95.5% = `95.5` · 10,000.50 บาท = `10000.5` (กฎข้อ 9 — แก้ 2026-10-09 · เดิมเก็บ bp / หน่วยย่อย ×100) |
| `status` | สถานะที่ใช้งานจริง key เดียว (account ACC-30) = เข้มที่สุดของสถานะที่ตั้งกับบัญชีเอง, ผู้สร้าง (sub) และหัวสายทุกชั้น (AUTH-53) · DB เก็บสถานะที่ตั้งกับบัญชีเองแยกไว้ |

## 3. Business rules

### ประเภทและผู้สร้าง

| ID | Rule |
|---|---|
| MGMT-01 | `user_type`: `SUPERADMIN` · `ADMIN` · `COMPANY_TRANSFER` · `COMPANY_SEAMLESS_RESELLER` · `COMPANY_SEAMLESS_MASTER` · `COMPANY_SEAMLESS_1TO1` · `SHARE_B2B` · `SHARE_B2C` · `SHARE_RESELLER` · `SHARE_MASTER` · `AGENT` · `MEMBER` · เก็บเป็น `role` + `agent_type` ใน `user_agents` (Member อยู่ตาราง `user_members`) |
| MGMT-02 | ใครสร้างอะไรได้: |
| | • Superadmin → Company ทั้ง 4 ประเภท |
| | • Company Transfer → Share B2B · Share B2C |
| | • Company Seamless Reseller → Share B2C (เก็บเป็น `SHARE_RESELLER`) · Company Seamless Master → Share B2C (เก็บเป็น `SHARE_MASTER`) |
| | • Company Seamless 1 to 1 → Member |
| | • Share B2C · Share Reseller · Share Master → Agent หรือ Member |
| | • Share B2B → Agent |
| | • Agent → Agent (ซ้อนได้ไม่จำกัดชั้น) หรือ Member |
| | ADMIN สร้างไม่ได้ (AUTH-43) · ประเภทการเดิมพัน (Transfer / Seamless) ตามผู้สร้าง เลือกไม่ได้ |
| MGMT-03 | ผู้สร้างต้อง `status = ACTIVE` (AUTH-54 บังคับที่ middleware) · บัญชีใหม่ `status = ACTIVE` |
| MGMT-04 | สร้างบัญชีฝั่ง agent **ทุกประเภท** (Company / Share / Agent — แก้ 2026-10-08) → สร้าง Key ของ account 1.3 ใน tx เดียวกัน (ACC-05) |

### ข้อมูลบัญชี

| ID | Rule |
|---|---|
| MGMT-05 | username (ฝั่ง agent และ Member): 3–32 ตัว `a-z` `0-9` (รับตัวพิมพ์ใหญ่แล้วแปลงเป็นเล็ก — AUTH-01) · **ห้ามซ้ำทั้งระบบ** ข้ามตาราง `user_agents` และ `user_members` (เช็คใน service ภายใน tx พร้อม `pg_advisory_xact_lock` ของ username กันสร้างพร้อมกัน) · แก้ไม่ได้ (AUTH-19) |
| MGMT-06 | รหัสผ่านตาม AUTH-36 · ฝั่ง agent ตั้ง passcode ตอน login ครั้งแรก (AUTH-29) |
| MGMT-07 | ชื่อ: บังคับ · 3–32 ตัวอักษร (นับเป็นตัวอักษร ไม่ใช่ byte) · ใช้ได้ภาษาไทย `A-Z` `a-z` `0-9` · ไม่มีช่องว่าง (แก้ 2026-10-08 — เดิม `A-Z` `a-z` `0-9` เท่านั้น) |
| MGMT-08 | เบอร์โทร: ไม่บังคับ (ไม่กรอกส่ง `""` · ห้าม `null` — account ACC-32) · 8–15 ตัว ตัวเลขเท่านั้น (ห้าม `+`) · ห้ามซ้ำภายในตารางเดียวกัน (unique index ต่อตาราง) · ยกเว้น sub ซ้ำได้ (MGMT-41) |
| MGMT-09 | แก้ได้: ชื่อ · เบอร์โทร · สถานะ (MGMT-30) · ค่าหุ้นส่วนและเปิด / ปิดเกม (MGMT-20) · แก้ไม่ได้: username · ประเภท · สกุลเงิน · ไม่มีการลบบัญชีฝั่ง agent และ Member |
| MGMT-09A | แก้ไข Member ได้: ชื่อ · เบอร์โทร · สถานะ · Commission (แต่ละอย่างใช้เส้นของตัวเอง — หัวข้อ 5) |
| MGMT-61 | **สายชั้นบน** `user_agents.cnf` (JSONB — เพิ่ม 2026-10-09): `{"parent": [{"id": 1, "position": "superadmin"}, {"id": 4754, "position": "company"}, …]}` · เรียงจาก Superadmin ลงมาถึงผู้สร้างตรง · **ไม่รวมตัวเอง** (ตัวสุดท้าย = `parent_id`) · `position` = role ตัวเล็ก (`superadmin` · `company` · `shareholder` · `agent`) · Superadmin / ADMIN = `{"parent": []}` · ตั้งตอนสร้าง = `parent` ของผู้สร้าง + ผู้สร้าง · ไม่เปลี่ยนหลังสร้าง (ย้ายสายไม่ได้ — ถ้าวันหน้ามีต้องแก้ทั้งสายใต้บัญชีนั้น) · **ไม่ส่งใน API** · ใช้ไล่สายขึ้น (อ่านแถวตัวเอง) และค้นทั้งสายลง (`cnf @> '{"parent":[{"id":X}]}'` ใช้ GIN index) · บัญชีเดิม backfill ใน migration · Member (`user_members.cnf` — สายถึง agent ที่สร้าง) ทำใน module `member_management` |
| MGMT-62 | `agent_game_settings.parent_id` (เพิ่ม 2026-10-09) = id ผู้สร้างของบัญชีเจ้าของแถว (ค่าเดียวกับ `user_agents.parent_id`) · Superadmin = `NULL` · ตั้งตอนสร้าง · แถวเดิม backfill ใน migration · ใช้หาค่าตั้งของลูกตรงทุกคนของบัญชีหนึ่ง |

### สกุลเงิน

ระบบมี 27 สกุล: ARS AUD BDT BOB BRL CLP CNY EUR GBP HKD IDR INR JPY KHR KRW LAK MMK MXN MYR NGN PHP PKR THB TWD USD USDT VND ·
ทุกสกุลทศนิยมไม่เกิน 4 ตำแหน่ง (ACC-18 · แก้ 2026-10-09)

| ID | Rule |
|---|---|
| MGMT-10 | Company Transfer · Company Seamless Reseller / Master: มีครบทุกสกุล (ไม่ต้องส่ง) |
| MGMT-11 | Company Seamless 1 to 1: 1 สกุล ที่ Superadmin เลือก |
| MGMT-12 | Share B2B: เลือกได้หลายสกุล อย่างน้อย 1 · Share B2C / Reseller / Master: 1 สกุล |
| MGMT-13 | Agent ที่ Share B2B สร้าง: 1 สกุล · Agent ที่ Share B2C / Reseller / Master หรือ Agent สร้าง และ Member: ใช้สกุลของผู้สร้าง (ไม่ต้องส่ง) |
| MGMT-14 | สกุลที่เลือกต้องอยู่ในสกุลของผู้สร้าง · เปลี่ยนหลังสร้างไม่ได้ |
| MGMT-15 | บัญชีฝั่ง Seamless ทุกชั้น (Company Seamless ทุกประเภท และสายล่าง) ไม่มียอดเงิน · ส่ง `balance` ตอนสร้าง = `422` |
| MGMT-15A | **ยอดเงินตั้งต้นตอนสร้าง** (`balance` ในเส้นสร้าง · ไม่บังคับ · บัญชีฝั่ง Transfer เท่านั้น): ต่อสกุล · ต้องเป็นสกุลของบัญชีใหม่ · มากกว่า 0 · ทศนิยมไม่เกิน 4 ตำแหน่ง (แก้ 2026-10-09) · **โอนจากยอดของผู้สร้าง** ในสกุลเดียวกัน (ผู้สร้างลด · บัญชีใหม่เพิ่ม) · ยอดผู้สร้างไม่พอ = ปฏิเสธทั้งคำสั่ง (`402312`) |
| | • ทำตามกฎเงิน: สร้างบัญชี + โอนใน **transaction เดียวกัน** · lock แถวยอดของผู้สร้างก่อนอ่าน (`FOR UPDATE` เรียงตาม id) · ledger ทั้งสองฝั่งใน tx เดียวกัน (กฎข้อ 10–11) |
| | • กันยิงซ้ำ: ส่ง `request_id` จากหน้าบ้าน (unique ใน DB) · ส่งซ้ำด้วย `request_id` เดิม = ได้ผลเดิม ไม่สร้างซ้ำ ไม่โอนซ้ำ (กฎข้อ 12) |
| | • **ไม่ต้องยืนยัน passcode** แม้ส่ง `balance` (ตัดสิน 2026-10-07) |
| | • **Superadmin วงเงินไม่จำกัด** (ยืนยัน 2026-10-07): ให้ยอดเงินตั้งต้นกับ Company ได้เท่าไรก็ได้ · ไม่มียอดของตัวเองให้หัก ไม่มีเส้นเติมเงิน · ledger บันทึกเฉพาะฝั่งที่ได้รับ (`reason = INITIAL_FROM_SUPERADMIN`) · Profile ของ Superadmin `balances` ส่ง `0` (ไม่มีแถวยอด — account ACC-19) |
| | • ตารางยอดเงินและ ledger เป็นของ module การชำระเงิน — ออกแบบขั้นต่ำไว้ในหัวข้อ 6 ให้ module นั้นใช้ต่อ |

### ค่าหุ้นส่วน (PT · Force · Remain · Commission)

| ID | Rule |
|---|---|
| MGMT-16 | **รับค่าชุดเดียวต่อกลุ่ม · เก็บต่อเกม**: |
| | • ส่งเป็น object `pt` แยกตาม**กลุ่ม PT** · ระบบ**กระจายค่าลงทุกเกมในกลุ่ม** แล้วเก็บทีละเกม (`agent_game_settings`) · ค่าของเกมในกลุ่มเท่ากันเสมอ |
| | • ตอนนี้มีกลุ่ม `minigame` กลุ่มเดียว = `coin_toss`, `rock_paper_scissors`, `scratch_card` (แก้ชื่อจาก `game` 2026-10-08) · ระบบอื่น เช่น `askmelotto` เพิ่มเมื่อมีระบบนั้น · การจับคู่กลุ่ม → หมวด → เกม กำหนดใน `app/core` เพิ่มได้โดยไม่เปลี่ยน API / ตาราง |
| | • **ผู้สร้างตั้งให้ลูก** (สร้าง / แก้ลูก): กลุ่มละ 5 ค่า `pt_from_parent` (ให้ลูกคนนี้) · `force` · `remain_quota` · `commission_percent` · `status` (รับ PT ไหม — MGMT-20) |
| | • **บัญชีตั้งของตัวเอง** (MGMT-22): กลุ่มละ 1 ค่า `pt` (ถือจาก Member ใต้ตัวเองตรงๆ) |
| | • **แสดง (response ทุกเส้น — แก้ 2026-10-08): ชุดเดียวต่อระบบ** (ทุกเกมในระบบค่าเท่ากันเสมอ — หลังบ้านเก็บต่อเกม แล้วรวมตอนส่งออก) `pt` → ระบบ → `pt_from_parent` · `pt` · `force` · `remain_quota` · `commission_percent` · `status` · `created_at` · `created_by` · `updated_at` · `updated_by` · `status_game` → รหัสเกม → bool แยกด้านนอก `pt` (MGMT-20) · Member: ต่อระบบมีแค่ `commission_percent` · `created_at` · `created_by` · `updated_at` · `updated_by` |
| | • `status` = **รับ PT ในระบบนั้นไหม** (ไม่ใช่สถานะเกม — เกมใช้ `status_game`) · ความหมายตาม MGMT-20 |
| | • `created_at` / `created_by` = เวลา / username ของผู้สร้างค่า PT (ตอนสร้างบัญชี · ไม่เปลี่ยนอีก) · `updated_at` / `updated_by` = เวลา / username ของคนที่แก้ล่าสุด (รวม sub) · ตอนสร้าง = ผู้สร้าง |
| | • เกมใหม่ที่เพิ่มเข้าหมวดภายหลัง: ทุกบัญชีได้ค่าของกลุ่มนั้น และ `status_game = true` อัตโนมัติ |
| MGMT-17 | ค่า % ใน API เป็น JSON number ทศนิยมไม่เกิน 4 ตำแหน่ง (เช่น `30`, `0.5`) · ภายใน DB / Go เป็น `float64` ปัด 4 ตำแหน่ง (แก้ 2026-10-09 — กฎข้อ 9) · step เดิมไม่เปลี่ยน (MGMT-18) · **ห้าม `null`** · ไม่ตั้ง Force / Remain = `0` |
| MGMT-18 | ค่าที่ให้ลูก (`pt_from_parent`) และค่าถือ (`pt`): 0 ถึง**ค่าที่ตัวเองได้รับ** ทีละ 0.5% · Force / Remain: 0 ถึงค่าที่ให้ลูกคนนั้น ทีละ 0.5% · Commission: 0–1% ทีละ 0.1% · Commission ของลูกตั้ง**เกินของผู้สร้างได้** (เช่น ผู้สร้างได้ 0.5% ตั้งให้ลูก 0.6% ได้) แต่ไม่เกิน 1% |
| MGMT-19 | Company Seamless Master: `pt` ล็อกที่ 0 · ค่าที่ให้ Share Master ต้อง**เท่ากับค่าที่ตัวเองได้รับ** · Force และ Remain ที่ให้ Share Master = 0 · แก้ได้แค่ Commission (เอกสาร PT) |
| MGMT-20 | **`status` ใน `pt` กับ `status_game` แยกหน้าที่กัน** (boiledegg ตัดสิน 2026-10-08): |
| | • `pt` → ระบบ → `status` (bool) = รับ PT ในระบบนั้นไหม · `false` = **เกมยังเปิดให้เล่น แต่บัญชีนี้ไม่รับ PT** ในระบบนั้น · ผู้สร้างตั้งให้ (สร้าง / `update-pt`) · ส่วน PT ที่บัญชีนี้ไม่รับไปอยู่กับใคร กำหนดตอนทำ module คิด PT |
| | • `status_game` → รหัสเกม (bool) = **เปิด / ปิดเกมรายบัญชี** · `false` = Member ในสายเล่นเกมนั้นไม่ได้ (บังคับใน module เดิมพัน) · ผู้สร้างตั้งให้ · ชั้นบนปิดแล้ว**สายล่างเปิดเองไม่ได้** — เกมใช้ได้จริงเมื่อบัญชีตัวเองและหัวสายทุกชั้นเปิดอยู่ (แบบเดียวกับ `status` ที่ใช้งานจริง — ACC-30) |
| MGMT-21 | สร้างบัญชีฝั่ง agent ต้องส่งครบทุกกลุ่มที่มี · Member: ผู้สร้างตั้ง `pt` (ถือสู้กับ Member คนนี้) + `commission_percent` ต่อกลุ่ม · `remain_quota` ระบบคิด — **แก้ 2026-10-09 รอ lead อนุมัติ** รายละเอียดใน [member_management.md](member_management.md) MGMT-21 · กระจายและเก็บต่อเกมแบบเดียวกัน (`user_member_game_settings`) |
| MGMT-22 | **ผู้สร้างกำหนดค่าที่ให้ลูกแต่ละคนแยกกัน** (ทุกชั้น รวม Superadmin → Company): |
| | • ลูกแต่ละคนได้รับไม่เท่ากันได้ · ค่าที่ให้ลูกแต่ละคน**ห้ามเกินค่าที่ผู้สร้างได้รับ** · Superadmin ได้รับ 100% |
| | • ผู้สร้าง**ถือในสายของลูกคนนั้น** = ค่าที่ผู้สร้างได้รับ − ค่าที่ให้ลูกคนนั้น (ระบบคำนวณ ไม่ต้องส่ง) |
| | • ทุกบัญชี**ตั้งค่าถือ `pt` เอง**สำหรับ Member ที่อยู่ใต้ตัวเองตรงๆ (ห้ามเกินค่าที่ได้รับ) · บัญชีใหม่เริ่มที่ `pt` = ค่าที่ได้รับ (ถือทั้งหมด) จนกว่าจะแก้ |
| | • ส่วนที่ไม่มีใครถือ (เช่น ได้รับ 60 ถือ 40 → เหลือ 20) ผ่าน Force / Remain ตอนคิดเงินก่อน ถ้าไม่มีก็**คืน Superadmin** (เอกสาร PT Force / PT Remain) |
| MGMT-23 | ค่าที่ให้ลูก, Force, Remain, Commission แก้ได้เฉพาะ**ผู้สร้างโดยตรง** · `pt` ของบัญชีแก้ได้เฉพาะ**บัญชีนั้นเอง** · sub ตามสิทธิ์ (MGMT-50) · แก้เฉพาะกลุ่มที่ส่ง · ในกลุ่มที่ส่งต้องครบทุกค่าของเส้นนั้น · มีผลกับ bet ใหม่ (bet เก็บค่า ณ ตอน bet — module เดิมพัน) |
| MGMT-24 | **ลดค่าที่ให้ลูก**: ค่าใหม่ต้อง ≥ ค่าที่มากที่สุดระหว่าง (ค่าที่ลูกให้ลูกของมันแต่ละคน, `pt` ของลูก) ไม่งั้น**ปฏิเสธทั้งคำสั่ง** (`402306` msg บอกค่าต่ำสุดที่ตั้งได้) · ระบบไม่แก้ค่าของลูกหรือหลานให้เอง — ถ้าจะลดต่ำกว่านั้นลูกต้องลดของตัวเองก่อน · **เพิ่มค่าที่ให้ลูก**: ได้ถึงค่าที่ผู้สร้างได้รับ · ลูกคนอื่นไม่กระทบ · tx + lock แถวลูกและลูกของลูก (อ่านค่าที่ใช้อยู่) เรียงตาม id |
| MGMT-25 | Force / Remain ในคำสั่งเดียวกันต้องไม่เกินค่าที่ให้ลูกใหม่ (`402308`) |

ตัวอย่าง (กลุ่ม `minigame`):

```
Superadmin (ได้รับ 100)
 ├─ ให้ comA = 90  → Superadmin ถือในสาย comA = 10
 └─ ให้ comB = 80  → Superadmin ถือในสาย comB = 20
comA (ได้รับ 90)
 ├─ ให้ share1 = 70 → comA ถือในสาย share1 = 20
 └─ ให้ share2 = 50 → comA ถือในสาย share2 = 40
share1 (ได้รับ 70)
 ├─ ให้ agent1 = 60 → share1 ถือในสาย agent1 = 10
 └─ pt = 30         → ถือจาก Member ของ share1 เอง
agent1 (ได้รับ 60)
 └─ pt = 40         → ถือจาก Member ของ agent1 · เหลือ 20 ผ่าน Force / Remain ไม่งั้นคืน Superadmin
```

Member ใต้ agent1 แพ้ 1,000: Superadmin 10% (100) · comA 20% (200) · share1 10% (100) · agent1 40% (400) · เหลือ 20% (200) คืน Superadmin

MGMT-24: comA ลดค่าที่ให้ share1 ได้ต่ำสุด 60 (ค่าที่มากที่สุดของ share1 = ให้ agent1 60 · `pt` 30) · ตั้ง 55 → `402306`

### รายชื่อดาวน์ไลน์และรายละเอียด

| ID | Rule |
|---|---|
| MGMT-26 | เส้นรายชื่อเส้นเดียว: แสดง**ลูกตรง**ของบัญชีที่ระบุ (ไม่ระบุ = ตัวเอง) ทั้งฝั่ง agent และ Member ปนกัน · ไล่ลงทีละชั้นได้ถึง Member · ระบุได้เฉพาะตัวเองหรือบัญชีในสายล่าง (`402402`) · ADMIN ไม่อยู่ในรายการใด (AUTH-43) |
| MGMT-27 | ตัวกรองมีแค่ `page` / `limit` และค้นหา username บางส่วน (ไม่สนตัวพิมพ์) · เรียง username A→Z · `page` / `limit` อยู่ใน body (แก้ 2026-10-08) · `limit` ค่าเริ่มต้น 20 สูงสุด 100 (`utils.NewPage`) |
| MGMT-27A | **ค้นหาทั้งสาย** (`/manage/downlines/search` — เพิ่ม 2026-10-08): ค้น username บางส่วน (ไม่สนตัวพิมพ์) ในบัญชี**ทุกชั้นใต้ตัวเอง** ทั้งฝั่ง agent และ Member · ไม่รวมตัวเอง ชั้นบน และสายข้างเคียง (ขอบเขตจาก token — กฎข้อ 22) · `q` บังคับ 2–32 ตัว (`422`) · เรียง username A→Z · `page` / `limit` เหมือน MGMT-27 · แต่ละแถวเหมือน MGMT-28 + `parent_username` (ผู้สร้างตรง) · `status` = สถานะที่ใช้งานจริง · ADMIN ไม่อยู่ในผล |
| MGMT-27B | **ค้นหาบัญชีของ ADMIN** (`/admin/accounts/search` — เพิ่ม 2026-10-08 · boiledegg ตัดสิน) · **เส้นอยู่ module `admin_management`** ([admin_management.md](admin_management.md)): ADMIN เท่านั้น (`401308`) · ค้นบัญชี**ทั้งระบบ** (ADMIN ไม่มีสาย) ทุกประเภท รวม SUPERADMIN, ADMIN, sub และ Member · username **ตรงทั้งคำ** ไม่สนตัวพิมพ์ ตัดช่องว่าง (ไม่รับบางส่วน) · ผล 0–2 แถว (ฝั่ง agent / sub แล้วตามด้วย Member ที่ username ซ้ำได้) ไม่มี `page` · แต่ละแถวมีแค่ข้อมูลระบุตัวตน: `username` · `role` · `user_type` · `is_subaccount` · `status` (สถานะที่ใช้งานจริง · sub ตาม MGMT-43) · `parent_username` (ผู้สร้าง · sub = เจ้าของ · SUPERADMIN / ADMIN = `""`) · `created_at` · `last_login_at` · `last_login_ip` · **ไม่มี** ยอดเงิน, PT, id · sub ใช้ `role` / `user_type` ของเจ้าของ |
| MGMT-28 | แต่ละแถว: `id` · `role` (บอกว่าเป็นฝั่ง agent หรือ Member) · `user_type` · username · ชื่อ · เบอร์โทร · `status` (ACC-30) · `pt` (ตาม MGMT-51) · `balances` (ยอดแยกสกุลตาม account ACC-19 — ระหว่างยังไม่มี module การชำระเงินส่ง `0`) |
| MGMT-29 | เส้นรายละเอียดแยก agent / Member: ทุก field ของบัญชี **ยกเว้น** password, passcode, hash และ token · รวม สกุลเงิน · `balances` · `pt` ชุดเดียวต่อกลุ่ม (รวม `pt_from_parent` ในกลุ่ม — MGMT-16) ชุดเดียวต่อระบบ + `status_game` ต่อเกม (MGMT-16 · รูปแบบเดียวกับ account ACC-16) · login ล่าสุด · วันที่สร้าง · ระบุได้เฉพาะบัญชีในสายล่าง |

### สถานะ

| ID | Rule |
|---|---|
| MGMT-30 | ค่า `ACTIVE` / `SUSPENDED` / `LOCKED` · แก้ได้เฉพาะ**ผู้สร้างโดยตรง** · เปลี่ยนกลับเป็น `ACTIVE` ได้ · แก้ของตัวเองไม่ได้ · ไม่ต้อง passcode · เส้นแยก agent / Member · ค่าที่แก้ = สถานะที่ตั้งกับบัญชีนั้นเอง · response ทุกเส้นแสดง `status` ที่ใช้งานจริง (ACC-30) — ถ้าหัวสายถูกระงับ ปลดที่ลูกแล้วลูกยังเป็น `SUSPENDED` จนกว่าหัวสายจะปลด |
| MGMT-31 | ผลต่อสายล่างผ่าน `status` ที่ใช้งานจริง (ACC-30 · AUTH-53) ไม่แก้แถวของชั้นล่าง · สถานะของลูกไม่ส่งผลขึ้นไปที่หัวสาย · `LOCKED` = เข้าใช้ไม่ได้ (AUTH-27) · `SUSPENDED` = เข้าได้เฉพาะ Profile และ Report ดูอย่างเดียว (AUTH-54) · Member ใต้บัญชีที่ไม่ ACTIVE เดิมพันไม่ได้ (module เดิมพัน) |

### คัดลอกการตั้งค่า

| ID | Rule |
|---|---|
| MGMT-35 | หน้าบ้านดึงรายการลูกตรงของตัวเอง (ทุกประเภทฝั่ง agent) พร้อม `pt` และ `status_game` แล้วเติมลงฟอร์มเอง · ไม่มีเส้นคัดลอก · ตอนสร้างตรวจตามกฎปกติ |

### บัญชีย่อย (sub)

| ID | Rule |
|---|---|
| MGMT-40 | สร้างได้: บัญชีหลักฝั่ง agent (ADMIN ไม่ได้ — AUTH-43 · Superadmin สร้างได้ — Company Hierarchy: ทุกประเภทสร้าง sub ได้) · sub สร้าง / แก้ / เปลี่ยนสถานะ / ดูรายชื่อ sub ไม่ได้ (`402311`) |
| MGMT-41 | username = `{username เจ้าของ}@{ชื่อ}` · ส่วนหลัง `@` 3–20 ตัว `a-z` `0-9` (AUTH-18) · **รับตัวพิมพ์ใหญ่ได้ หลังบ้านแปลงเป็นตัวเล็กเอง** (ตอน login ก็แปลงก่อนเช็ค — AUTH-01) · รหัสผ่านตาม AUTH-36 · ชื่อเล่น: ตาม MGMT-07 · เบอร์โทร: รูปแบบตาม MGMT-08 แต่**ซ้ำได้** (แก้ 2026-10-08 — ฝั่ง agent และ Member ยังห้ามซ้ำ) |
| MGMT-42 | แก้ได้: ชื่อเล่น · เบอร์โทร · สิทธิ์ · สถานะ (MGMT-43) · แก้ username ไม่ได้ · รหัสผ่านใช้ระบบ reset ของ agent_auth (AUTH-52) |
| MGMT-43 | สถานะของ sub: `ACTIVE` / `INACTIVE` · **ผู้สร้าง sub เป็นคนเปลี่ยน** · `INACTIVE` = login ได้แต่ทำรายการไม่ได้ เหมือน `SUSPENDED` (AUTH-54 · ภายในเก็บเป็น `SUSPENDED` ในตาราง `subaccounts` · API แสดง `INACTIVE`) · เปลี่ยนกลับเป็น `ACTIVE` ได้ · ไม่ต้อง passcode · ถ้าเจ้าของหรือหัวสายถูกระงับ / ล็อก sub ได้ผลตาม `status` ที่ใช้งานจริง (ACC-30 · AUTH-53) |
| MGMT-44 | **ไม่มีการลบ sub** — เลิกใช้ให้เปลี่ยนเป็น `INACTIVE` |
| MGMT-45 | แก้ / เปลี่ยนสถานะได้เฉพาะเจ้าของ · ชั้นบนดูรายชื่อและรายละเอียด sub ของบัญชีในสายล่างได้อย่างเดียว |
| MGMT-46 | รายชื่อ sub: ระบุเจ้าของ (ไม่ระบุ = ตัวเอง · ต้องเป็นตัวเองหรือสายล่าง) · ค้นหา username บางส่วน · เรียง A→Z · `page` / `limit` · แต่ละแถว: `id` · username · ชื่อเล่น · เบอร์โทร · `status` · สิทธิ์ · วันที่สร้าง · login ล่าสุด (เวลา, IP) · รายละเอียด sub (`POST /manage/subaccounts/detail`) ส่ง field ชุดเดียวกัน |

### สิทธิ์ (ใช้ทั้งหลังบ้าน)

| ID | Rule |
|---|---|
| MGMT-50 | สิทธิ์ต่อเมนู ระดับ `off` (เข้าไม่ได้) / `view` (ดูอย่างเดียว) / `edit` (ดูและแก้) · **ไม่ติ๊ก = `off`** · บัญชีหลักได้ `edit` ทุกเมนูที่ประเภทนั้นมี · sub ได้ตามที่เจ้าของให้ · รูปแบบใน API เป็น object `{ "{menu}": "off" \| "view" \| "edit" }` ทั้ง request และ response (รวม Profile — account ACC-12) |
| MGMT-51 | เช็คด้วย middleware แยก `RequirePermission(menu, level)` บรรทัดเดียวกับ route (กฎข้อ 28) · route ของ admin ใช้ middleware แยก `adminMw.RequireAdmin()` ของ module admin_management (AUTH-44) · เมนู: |
| | `dashboard` (off / view) — account 1.1 · `member` — ดูรายชื่อ / รายละเอียด (view) · สร้าง / แก้ข้อมูล / สถานะ (edit) · `pt` — เห็นค่า `pt` ใน response (view) · แก้ PT (edit) · `report` (off / view) · `bet_cancel` · `payment` (ฝาก-ถอน) · `asset` · `announcement` · `rate` (อัตราแพ้ชนะ — เฉพาะ Superadmin) |
| | ไม่มี `pt` ≥ view → response ไม่มี field `pt` · **สร้างบัญชี ต้องมี `member` = edit และ `pt` = edit** (ถ้าส่ง `balance` ต้องมี `payment` = edit ด้วย — MGMT-15A) · เปลี่ยนรหัสผ่าน / passcode ของตัวเองไม่ต้องใช้สิทธิ์ (แต่ถูก AUTH-54 กันตอนถูกระงับ) |
| MGMT-52 | เมนูที่แต่ละประเภทมี (สิทธิ์ของ sub ให้ได้เฉพาะเมนูเหล่านี้ · ส่งเมนูอื่นมา = `422`): **Superadmin** = 8 เมนู — `dashboard` · `member` · `pt` · `report` · `bet_cancel` · `payment` · `asset` · `rate` (อัตราแพ้ชนะ · ไม่มี `announcement`) · **Company · Share · Agent** = 8 เมนูเดียวกัน — `dashboard` · `member` · `pt` · `report` · `bet_cancel` · `payment` · `asset` · `announcement` (ไม่มี `rate` — ยืนยัน 2026-10-07) · sub ของ Superadmin ได้สิทธิ์ `rate` เมื่อ Superadmin ให้ |
| MGMT-53 | Profile (account ACC-12) ส่ง `permissions` เป็น object รูปแบบเดียวกับ MGMT-50 · บัญชีหลักได้ `edit` ทุกเมนูของประเภทตัวเอง |

### ประวัติ

| ID | Rule |
|---|---|
| MGMT-60 | ทุกการสร้าง / แก้ข้อมูล / แก้สถานะ / แก้ PT (รวมแถวลูกที่ปรับตาม MGMT-24) / เปิดปิดเกม / สร้าง-แก้-เปลี่ยนสถานะ sub / ยอดเงินตั้งต้น เก็บ `account_change_logs` ใน tx เดียวกัน (ผู้ทำ, เป้าหมาย, action, ค่าเก่า → ใหม่, ip, request_id) · ห้ามเก็บรหัสผ่าน |

## 4. สิ่งที่พบในโค้ดเก่า และการตัดสินใจ

ไม่มี — ไม่ได้ extract จากโค้ดเก่า

## 5. Endpoints

ใช้แค่ method `GET` และ `POST` · **ระบุบัญชีด้วย `id` ใน body เสมอ** (ผู้เรียกมาจาก token) — เส้นอ่านที่ต้องระบุบัญชีจึงเป็น `POST` (แก้ 2026-10-08) · ใต้ `/bo/pr` ผ่าน `Authenticated` + `PassedGates()` (รวม AUTH-54) ·
สิทธิ์ของ sub ใช้ middleware `RequirePermission(menu, level)` บรรทัดเดียวกับ route ·
error ร่วม: `401202`, `401203`, `401301`, `401302`, `401304`, `401306`, `401307`, `401311`, `402303`

**เส้นของ Member** (`/manage/members/*`) ย้ายไป spec [member_management.md](member_management.md) (แยกเอกสาร 2026-10-09) · ตารางนี้เหลือเส้นของ module `agent_management`

| Method | Path | สิทธิ์ (sub) | หมายเหตุ |
|---|---|---|---|
| POST | `/api/v1/bo/pr/manage/agents/create` | `member` edit · `pt` edit (+ `payment` edit ถ้าส่ง `balance`) | สร้าง Company / Share / Agent |
| POST | `/api/v1/bo/pr/manage/downlines/list` | `member` view | รายชื่อลูกตรง (agent + Member) |
| POST | `/api/v1/bo/pr/manage/downlines/search` | `member` view | ค้นหาทั้งสายใต้ตัวเอง (MGMT-27A) |
| POST | `/api/v1/bo/pr/manage/agents/detail` | `member` view | รายละเอียดฝั่ง agent |
| POST | `/api/v1/bo/pr/manage/agents/update-info` | `member` edit | แก้ชื่อ · เบอร์โทร |
| POST | `/api/v1/bo/pr/manage/agents/update-status` | `member` edit | แก้สถานะ |
| POST | `/api/v1/bo/pr/manage/agents/update-pt` | `pt` edit | ค่าที่ให้ลูก + Force / Remain / Commission + เปิดปิดกลุ่ม (ผู้สร้างตั้งให้ลูก) |
| POST | `/api/v1/bo/pr/manage/agents/update-hold` | `pt` edit | ค่าถือ `pt` ของตัวเอง (MGMT-22) |
| POST | `/api/v1/bo/pr/manage/agents/update-games` | `pt` edit | เปิด / ปิดเกมรายบัญชีให้ลูกตรง (MGMT-20) |
| GET | `/api/v1/bo/pr/manage/agents/copy-sources` | `pt` view | ลูกตรงฝั่ง agent ของตัวเอง + `pt` (MGMT-35) |
| POST | `/api/v1/bo/pr/manage/subaccounts/list` | บัญชีหลัก (`RequireMainAccount`) | รายชื่อ sub |
| POST | `/api/v1/bo/pr/manage/subaccounts/detail` | บัญชีหลัก | รายละเอียด sub |
| POST | `/api/v1/bo/pr/manage/subaccounts/create` | บัญชีหลัก | สร้าง sub |
| POST | `/api/v1/bo/pr/manage/subaccounts/update-info` | บัญชีหลัก (เจ้าของ) | แก้ชื่อเล่น · เบอร์โทร · สิทธิ์ |
| POST | `/api/v1/bo/pr/manage/subaccounts/update-status` | บัญชีหลัก (เจ้าของ) | `ACTIVE` / `INACTIVE` (MGMT-43) |

ทุกตัวอย่างใช้ข้อมูลชุดเดียวกัน: `comp01` (id 10, Company Transfer) → `share01` (id 12, Share B2C · THB) → `agent01` (id 15) →
`mem01` (Member id 501) · sub ของ comp01 = `comp01@staff` (id 30)

ทุก response มีรูปแบบ `{ "code", "msg", "data" }` · สำเร็จ `code = 200` · เส้นที่ไม่มีข้อมูลตอบกลับไม่มี key `data` ·
business error ตอบ HTTP 200 พร้อม `code` ของ error · ตัวอย่าง error:
```json
{ "code": 402306, "msg": "pt.minigame.pt_from_parent ต่ำกว่าที่ลูกใช้อยู่ ตั้งได้ต่ำสุด 60" }
```
```json
{ "code": 422, "msg": "pt.minigame.force ห้ามเป็น null" }
```

### POST /api/v1/bo/pr/manage/agents/create

Request:
```json
{
  "request_id": "6f1c2a1e-3b7d-4c55-9a40-0f7f3c2d8e11",
  "user_type": "SHARE_B2C",
  "username": "Share01",
  "password": "aA4b4c4d4e4f",
  "name": "share01",
  "phone": "0812345678",
  "currencies": ["THB"],
  "balance": { "THB": 10000.00 },
  "pt": {
    "minigame": { "pt_from_parent": 70, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true }
  },
  "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": false }
}
```
Response:
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": { "id": 12, "username": "share01", "user_type": "SHARE_B2C" }
}
```
- `request_id` บังคับ (UUID ใหม่ทุกครั้งที่กดสร้าง — กันกดซ้ำ MGMT-15A) · ส่งซ้ำด้วยค่าเดิม = ได้ response เดิม ไม่สร้าง / ไม่โอนซ้ำ
- `user_type` = ประเภทของบัญชีใหม่ (MGMT-02) · ใต้ Company Seamless Reseller / Master ส่ง `SHARE_B2C` ระบบเก็บเป็น `SHARE_RESELLER` / `SHARE_MASTER` (response ตอบประเภทที่เก็บจริง)
- `username` รับตัวพิมพ์ใหญ่ได้ เก็บเป็นตัวเล็ก (`Share01` → `share01`)
- `currencies` ตาม MGMT-10 ถึง 13 (ประเภทที่ไม่ต้องส่ง ห้ามส่ง · ส่งผิดจำนวน = `422` · ไม่อยู่ในสกุลของผู้สร้าง = `402310`)
- `balance` ไม่บังคับ · ยอดเงินตั้งต้นต่อสกุล โอนจากยอดของผู้สร้าง (MGMT-15A) · บัญชีฝั่ง Seamless ห้ามส่ง
- `phone` ไม่ตั้งให้ส่ง `""` · `pt` ต้องส่งครบทุกกลุ่ม ครบ 5 ค่า · `status_game` ไม่บังคับ · เกมที่ไม่ส่ง = `true` (เปิด) · ไม่ส่งทั้ง field = เปิดทุกเกม · แก้ทีหลังที่ `update-games`

Error codes: `422`, `402301`, `402303`, `402305`, `402307`, `402308`, `402309`, `402310`, `402312`, `402401`, `402403`

### POST /api/v1/bo/pr/manage/downlines/list (รายชื่อลูกตรง)

Request (ทุกค่าไม่บังคับ · ไม่กรองให้ส่ง `{}`):
```json
{ "parent_id": 12, "q": "", "page": 1, "limit": 20 }
```
Response (ตัวอย่าง: ลูกตรงของ share01 · หัวสาย comp01 ถูกระงับ):
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "current_page": 1,
    "total_page": 1,
    "total_count": 2,
    "limit": 20,
    "has_next": false,
    "has_prev": false,
    "data": [
      {
        "id": 15,
        "role": "AGENT",
        "user_type": "AGENT",
        "username": "agent01",
        "name": "agent01",
        "phone": "0898765432",
        "status": "SUSPENDED",
        "pt": {
          "minigame": { "pt_from_parent": 60, "pt": 40, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "share01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "share01" }
        },
        "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": true },
        "balances": [ { "currency": "THB", "amount": 5000.00 } ]
      },
      {
        "id": 501,
        "role": "MEMBER",
        "user_type": "MEMBER",
        "username": "mem01",
        "name": "ใจดี",
        "phone": "",
        "status": "SUSPENDED",
        "pt": {
          "minigame": { "commission_percent": 0.3, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "agent01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "agent01" }
        },
        "balances": [ { "currency": "THB", "amount": 100.00 } ]
      }
    ]
  }
}
```
- `parent_id` ไม่ส่ง = ลูกตรงของตัวเอง · ต้องเป็นบัญชีฝั่ง agent ในสายล่าง (`402402`)
- `q` ค้น username บางส่วน ไม่สนตัวพิมพ์ · เรียง username A→Z · `limit` ค่าเริ่มต้น 20 สูงสุด 100
- `status` = สถานะที่ใช้งานจริง (ACC-30) — ตัวอย่างนี้หัวสายถูกระงับ ทุกแถวจึง `SUSPENDED`
- แถว `role = "MEMBER"` ไม่มีลูก · `pt` มีแค่ `commission_percent` · ไม่มีสิทธิ์ `pt` ≥ view = ไม่มี field `pt` (MGMT-51)
- `balances` 1 รายการต่อสกุลของบัญชี (MGMT-15A · account ACC-19) · ยังไม่มียอด และบัญชี Seamless = `0.00`

Error codes: `402303`, `402402`

### POST /api/v1/bo/pr/manage/downlines/search (ค้นหาทั้งสายใต้ตัวเอง)

Request:
```json
{ "q": "sh", "page": 1, "limit": 20 }
```
Response (comp01 ค้น `sh` — มี sh1 sh2 sh3 อยู่ใต้ในชั้นต่างกัน):
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "current_page": 1,
    "total_page": 1,
    "total_count": 3,
    "limit": 20,
    "has_next": false,
    "has_prev": false,
    "data": [
      {
        "id": 12,
        "role": "SHAREHOLDER",
        "user_type": "SHARE_B2C",
        "username": "sh1",
        "name": "sh1",
        "phone": "",
        "status": "ACTIVE",
        "parent_username": "comp01",
        "pt": {
          "minigame": { "pt_from_parent": 70, "pt": 30, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "comp01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "comp01" }
        },
        "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": true },
        "balances": [ { "currency": "THB", "amount": 10000.00 } ]
      },
      {
        "id": 13,
        "role": "SHAREHOLDER",
        "user_type": "SHARE_B2B",
        "username": "sh2",
        "name": "sh2",
        "phone": "",
        "status": "ACTIVE",
        "parent_username": "comp01",
        "pt": {
          "minigame": { "pt_from_parent": 50, "pt": 50, "force": 0, "remain_quota": 0, "commission_percent": 0.1, "status": true, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "comp01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "comp01" }
        },
        "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": true },
        "balances": [ { "currency": "THB", "amount": 0.00 }, { "currency": "USD", "amount": 0.00 } ]
      },
      {
        "id": 21,
        "role": "AGENT",
        "user_type": "AGENT",
        "username": "sh3agent",
        "name": "sh3agent",
        "phone": "",
        "status": "SUSPENDED",
        "parent_username": "sh1",
        "pt": {
          "minigame": { "pt_from_parent": 40, "pt": 40, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "sh1", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "sh1" }
        },
        "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": true },
        "balances": [ { "currency": "THB", "amount": 500.00 } ]
      }
    ]
  }
}
```
- ค้น username บางส่วน (ไม่สนตัวพิมพ์) ในบัญชี**ทุกชั้นใต้ตัวเอง** ทั้งฝั่ง agent และ Member (MGMT-27A) · ไม่รวมตัวเอง ชั้นบน และสายข้างเคียง
- `q` บังคับ อย่างน้อย 2 ตัว ไม่เกิน 32 ตัว (`422`) · เรียง username A→Z · `page` / `limit` เหมือน `/list`
- แต่ละแถวเหมือน `/list` + `parent_username` = username ของผู้สร้างตรง (บอกว่าแถวนั้นอยู่ใต้ใคร)
- `status` = สถานะที่ใช้งานจริง รวมผลจากทุกชั้นด้านบน (ตัวอย่าง: sh1 ปกติ แต่ sh3agent ถูกระงับเอง)

Error codes: `422`, `402303`

### POST /api/v1/bo/pr/manage/agents/detail

Request:
```json
{ "id": 12 }
```
Response:
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "id": 12,
    "role": "SHAREHOLDER",
    "user_type": "SHARE_B2C",
    "username": "share01",
    "name": "share01",
    "phone": "0812345678",
    "status": "ACTIVE",
    "parent_username": "comp01",
    "currencies": ["THB"],
    "balances": [ { "currency": "THB", "amount": 10000.00 } ],
    "pt": {
      "minigame": { "pt_from_parent": 70, "pt": 30, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "comp01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "comp01" }
    },
    "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": false },
    "passcode_set": true,
    "last_login_at": "2026-10-05T10:00:00+07:00",
    "last_login_ip": "203.0.113.10",
    "created_at": "2026-10-01T09:00:00+07:00"
  }
}
```
- `id` = บัญชีในสายล่างที่จะดู (ตัวเอง / สายบน / นอกสาย = `402402`) · ข้อมูลของตัวเองดูที่ account Profile
- รูปแบบ `pt` (ชุดเดียวต่อระบบ) และ `status_game` เหมือน Profile (account ACC-16) · ตัวอย่างนี้ ได้รับ 70 · ถือจาก Member ของตัวเอง 30 · `scratch_card` ถูกปิด
- ไม่มีสิทธิ์ `pt` ≥ view = ไม่มี field `pt` · ยังไม่เคย login = `last_login_at` / `last_login_ip` เป็น `""`

Error codes: `422`, `402303`, `402402`

### GET /api/v1/bo/pr/manage/agents/copy-sources

Request: ไม่มี body (ลูกตรงฝั่ง agent ของบัญชีใน token)

Response (ตัวอย่างของ comp01):
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": [
    {
      "id": 12,
      "username": "share01",
      "user_type": "SHARE_B2C",
      "pt": {
        "minigame": { "pt_from_parent": 70, "pt": 30, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "comp01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "comp01" }
      },
      "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": false }
    },
    {
      "id": 13,
      "username": "share02",
      "user_type": "SHARE_B2B",
      "pt": {
        "minigame": { "pt_from_parent": 50, "pt": 50, "force": 5, "remain_quota": 5, "commission_percent": 0.1, "status": true, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "comp01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "comp01" }
      },
      "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": true }
    }
  ]
}
```
- ลูกตรงฝั่ง agent ทั้งหมด เรียง A→Z (ไม่แบ่งหน้า) · ไม่มีลูก = `"data": []` · หน้าบ้านเติมค่าลงฟอร์มเอง (MGMT-35)

Error codes: `402303`

### POST /api/v1/bo/pr/manage/agents/update-info

Request (แทนทั้งชุด — ต้องส่งครบ):
```json
{ "id": 12, "name": "share01", "phone": "" }
```
Response:
```json
{ "code": 200, "msg": "สำเร็จ" }
```
- `id` = บัญชีที่จะแก้ (ลูกตรงเท่านั้น) · ผู้แก้มาจาก token · `phone` `""` = ไม่ตั้ง · ส่ง `null` หรือไม่ส่ง field = `422`

Error codes: `422`, `402303`, `402304`, `402402`, `402403`

### POST /api/v1/bo/pr/manage/agents/update-status

Request:
```json
{ "id": 12, "status": "SUSPENDED" }
```
Response:
```json
{ "code": 200, "msg": "สำเร็จ" }
```
- `status`: `ACTIVE` / `SUSPENDED` / `LOCKED` · ตั้งกลับ `ACTIVE` ได้ · ไม่ต้อง passcode (MGMT-30)

Error codes: `422`, `402303`, `402304`, `402402`

### POST /api/v1/bo/pr/manage/agents/update-pt (ผู้สร้างตั้งให้ลูก)

Request (ส่งเฉพาะกลุ่มที่จะแก้ · ในกลุ่มต้องครบ 5 ค่า · ห้าม `null`):
```json
{
  "id": 12,
  "pt": {
    "minigame": { "pt_from_parent": 60, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true }
  }
}
```
Response:
```json
{ "code": 200, "msg": "สำเร็จ" }
```
- `pt_from_parent` = ค่าที่ให้ลูกคนนี้ · ผู้สร้างถือในสายนี้ = ค่าที่ผู้สร้างได้รับ − ค่านี้ (MGMT-22) · `status` = รับ PT ไหม · `false` = เกมยังเปิด แต่ไม่รับ PT (MGMT-20)
- ลดได้ไม่ต่ำกว่าที่ลูกใช้อยู่ (MGMT-24) — ตัวอย่าง: share01 ถือ 30 และให้ agent01 60 → ตั้งต่ำสุด 60 · ตั้ง 55 = `402306`
- ส่ง `pt` (ค่าถือที่ลูกตั้งเอง) มา = `422`

Error codes: `422`, `402303`, `402304`, `402305`, `402306`, `402307`, `402308`, `402309`, `402402`

### POST /api/v1/bo/pr/manage/agents/update-hold (บัญชีตั้งค่าถือของตัวเอง)

Request (ไม่ส่ง `id` — แก้บัญชีใน token · ส่งเฉพาะกลุ่มที่จะแก้):
```json
{ "pt": { "minigame": { "pt": 40 } } }
```
Response:
```json
{ "code": 200, "msg": "สำเร็จ" }
```
- `pt` = ถือจาก Member ใต้ตัวเองตรงๆ · ไม่เกินค่าที่ตัวเองได้รับ (`402305`) · ส่วนที่เหลือผ่าน Force / Remain แล้วคืน Superadmin
- Company Seamless Master ตั้งได้แค่ `0` (`402307` — MGMT-19) · sub แก้ให้เจ้าของได้ตามสิทธิ์ `pt` = edit
- ส่งค่าที่ผู้สร้างเป็นคนตั้ง (`pt_from_parent` ฯลฯ) มา = `422`

Error codes: `422`, `402303`, `402305`, `402307`

### POST /api/v1/bo/pr/manage/agents/update-games

เปิด / ปิดเกมรายบัญชีให้ลูกตรง (MGMT-20 · boiledegg ตัดสิน 2026-10-08 — อยู่ใน module นี้)

Request (ส่งเฉพาะเกมที่จะเปลี่ยน · อย่างน้อย 1 เกม):
```json
{ "id": 12, "status_game": { "scratch_card": false } }
```
Response:
```json
{ "code": 200, "msg": "สำเร็จ" }
```

- แก้ได้เฉพาะ**ผู้สร้างโดยตรง** · ไม่ส่งต่อลงสายล่าง (สายล่างเช็คทั้งสายตอนเล่น — MGMT-20) · ไม่เปลี่ยนค่า PT และ `updated_by` ของ PT · บันทึก `account_change_logs` (`UPDATE_GAMES` ค่าเก่า / ใหม่)
- เกมไม่มีในระบบ · `status_game` ว่าง · `null` → `422`

Error codes: `422`, `402303`, `402304`, `402402`

### POST /api/v1/bo/pr/manage/subaccounts/list (รายชื่อ sub)

Request (ทุกค่าไม่บังคับ · ไม่กรองให้ส่ง `{}`):
```json
{ "owner_id": 10, "q": "", "page": 1, "limit": 20 }
```
Response:
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "current_page": 1,
    "total_page": 1,
    "total_count": 1,
    "limit": 20,
    "has_next": false,
    "has_prev": false,
    "data": [
      {
        "id": 30,
        "username": "comp01@staff",
        "name": "staff01",
        "phone": "",
        "status": "ACTIVE",
        "permissions": {
          "dashboard": "view", "member": "edit", "pt": "view", "report": "view",
          "bet_cancel": "off", "payment": "off", "asset": "off", "announcement": "off"
        },
        "created_at": "2026-10-06T12:26:43+07:00",
        "last_login_at": "",
        "last_login_ip": ""
      }
    ]
  }
}
```
- `owner_id` ไม่ส่ง = sub ของตัวเอง · ชั้นบนดู sub ของบัญชีในสายล่างได้ · นอกสาย = `402402`
- `status`: `ACTIVE` / `INACTIVE` · ถ้าเจ้าของ / หัวสายถูกระงับหรือล็อก แสดงสถานะนั้น (MGMT-43)
- `permissions` ครบทุกเมนูของประเภทเจ้าของ (Superadmin มี `rate` แทน `announcement`)

Error codes: `402311`, `402402`

### POST /api/v1/bo/pr/manage/subaccounts/detail

Request:
```json
{ "id": 30 }
```
Response:
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "id": 30,
    "username": "comp01@staff",
    "name": "staff01",
    "phone": "",
    "status": "ACTIVE",
    "permissions": {
      "dashboard": "view", "member": "edit", "pt": "view", "report": "view",
      "bet_cancel": "off", "payment": "off", "asset": "off", "announcement": "off"
    },
    "created_at": "2026-10-06T12:26:43+07:00",
    "last_login_at": "2026-10-07T09:15:00+07:00",
    "last_login_ip": "203.0.113.20"
  }
}
```
- ดูได้: เจ้าของ และชั้นบนในสาย · อื่น = `402404`

Error codes: `422`, `402311`, `402404`

### POST /api/v1/bo/pr/manage/subaccounts/create

Request:
```json
{
  "name_suffix": "Staff",
  "password": "aA4b4c4d4e4f",
  "name": "staff01",
  "phone": "",
  "permissions": { "dashboard": "view", "member": "edit", "pt": "view", "report": "view" }
}
```
Response:
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": { "id": 30, "username": "comp01@staff" }
}
```
- username = `{username เจ้าของ}@{name_suffix}` · `name_suffix` รับตัวพิมพ์ใหญ่ได้ เก็บเป็นตัวเล็ก
- เมนูที่ไม่ส่ง = `off` · ระดับ `off` / `view` / `edit` (`dashboard` / `report` สูงสุด `view`) · เมนูที่ประเภทเจ้าของไม่มี = `422`
- เบอร์โทรของ sub ซ้ำได้ (MGMT-41)

Error codes: `422`, `402301`, `402311`, `402401`

### POST /api/v1/bo/pr/manage/subaccounts/update-info

Request (แทนทั้งชุด — ต้องส่งครบ · ไม่ให้สิทธิ์ใดส่ง `"permissions": {}`):
```json
{
  "id": 30,
  "name": "staff01",
  "phone": "0811111111",
  "permissions": { "member": "view", "report": "view" }
}
```
Response:
```json
{ "code": 200, "msg": "สำเร็จ" }
```
- แก้ได้เฉพาะเจ้าของ (ชั้นบน = `402404`)

Error codes: `422`, `402311`, `402404`

### POST /api/v1/bo/pr/manage/subaccounts/update-status

Request:
```json
{ "id": 30, "status": "INACTIVE" }
```
Response:
```json
{ "code": 200, "msg": "สำเร็จ" }
```
- `status`: `ACTIVE` / `INACTIVE` · `INACTIVE` = login ได้แต่ทำรายการไม่ได้ (เก็บเป็น `SUSPENDED`) · ไม่มีการลบ sub (MGMT-44)
- แก้ได้เฉพาะเจ้าของ (ชั้นบน = `402404`)

Error codes: `422`, `402311`, `402404`

## 6. Schema

migration ไฟล์ใหม่

```sql
ALTER TABLE user_agents
    ADD COLUMN name       VARCHAR(32),
    ADD COLUMN phone      VARCHAR(15),
    ADD COLUMN agent_type VARCHAR(20),
    ADD CONSTRAINT ck_user_agents_agent_type CHECK (
        (role = 'COMPANY' AND agent_type IN ('TRANSFER','SEAMLESS_RESELLER','SEAMLESS_MASTER','SEAMLESS_1TO1')) OR
        (role = 'SHAREHOLDER' AND agent_type IN ('B2B','B2C','RESELLER','MASTER')) OR
        (role NOT IN ('COMPANY','SHAREHOLDER') AND agent_type IS NULL)),
    ADD CONSTRAINT ck_user_agents_phone CHECK (phone IS NULL OR phone ~ '^[0-9]{8,15}$');
CREATE UNIQUE INDEX uq_user_agents_phone ON user_agents(phone) WHERE phone IS NOT NULL;
CREATE INDEX idx_user_agents_parent_username ON user_agents(parent_id, username);   -- รายชื่อลูกตรง เรียง A→Z

CREATE TABLE agent_currencies (
    agent_id BIGINT     NOT NULL REFERENCES user_agents(id),
    currency VARCHAR(4) NOT NULL,
    PRIMARY KEY (agent_id, currency)
);

CREATE TABLE agent_game_settings (               -- ค่าหุ้นส่วน + เปิด/ปิด ต่อเกม (MGMT-16, MGMT-20)
    agent_id          BIGINT      NOT NULL REFERENCES user_agents(id),
    category          VARCHAR(30) NOT NULL,      -- 'minigame', ...
    game_code         VARCHAR(50) NOT NULL,      -- 'coin_toss', ...
    pt_from_parent_bp INT         NOT NULL CHECK (pt_from_parent_bp BETWEEN 0 AND 10000),
    pt_bp             INT         NOT NULL CHECK (pt_bp BETWEEN 0 AND 10000),
    force_bp          INT         NOT NULL DEFAULT 0 CHECK (force_bp BETWEEN 0 AND 10000),
    remain_bp         INT         NOT NULL DEFAULT 0 CHECK (remain_bp BETWEEN 0 AND 10000),
    commission_bp     INT         NOT NULL DEFAULT 0 CHECK (commission_bp BETWEEN 0 AND 100),
    status            BOOLEAN     NOT NULL DEFAULT true,   -- รับ PT ไหม (false = เกมยังเปิด แต่ไม่รับ PT — MGMT-20)
    status_game       BOOLEAN     NOT NULL DEFAULT true,   -- เปิด / ปิดทีละเกม
    created_by        VARCHAR(71) NOT NULL DEFAULT '',     -- username ผู้สร้าง (เพิ่ม 2026-10-08 — migration ใหม่ พร้อม created_at · user_member_game_settings ด้วย)
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by        VARCHAR(71) NOT NULL DEFAULT '',     -- username ของคนที่แก้ล่าสุด
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (agent_id, game_code),
    CHECK (pt_bp <= pt_from_parent_bp)
);

-- ตาราง Member (user_members · user_member_game_settings) อยู่ใน docs/modules/member_management.md หัวข้อ 4


ALTER TABLE subaccounts
    ADD COLUMN name        VARCHAR(32),
    ADD COLUMN phone       VARCHAR(15),
    ADD COLUMN permissions JSONB NOT NULL DEFAULT '{}';   -- {"member":"edit", ...} ไม่มี key = off · อ่านพร้อมแถว sub ทุก request (AUTH-27)
-- สถานะ INACTIVE ของ sub เก็บเป็น 'SUSPENDED' (ใช้ constraint เดิม ACTIVE / SUSPENDED / LOCKED) — MGMT-43
CREATE INDEX idx_subaccounts_agent_username ON subaccounts(agent_id, username);

-- ยอดเงินและ ledger (MGMT-15A) — เจ้าของจริงคือ module การชำระเงิน · ออกแบบขั้นต่ำที่นี่
CREATE TABLE agent_balances (
    agent_id   BIGINT      NOT NULL REFERENCES user_agents(id),
    currency   VARCHAR(4)  NOT NULL,
    amount     BIGINT      NOT NULL DEFAULT 0 CHECK (amount >= 0),   -- หน่วยย่อยที่สุด (1/100 — ACC-18)
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (agent_id, currency)
);

-- ยอดของ Member อยู่ที่ user_members.credit (เลิกใช้ user_member_balances 2026-10-09) — docs/modules/member_management.md หัวข้อ 4

CREATE TABLE balance_ledger (                    -- ทุกการเปลี่ยนยอดต้องมีแถวที่นี่ใน tx เดียวกัน (กฎข้อ 11)
    id            BIGSERIAL   PRIMARY KEY,
    owner_type    VARCHAR(10) NOT NULL,          -- AGENT / MEMBER
    owner_id      BIGINT      NOT NULL,
    currency      VARCHAR(4)  NOT NULL,
    amount        BIGINT      NOT NULL,          -- + เข้า / − ออก
    balance_after BIGINT      NOT NULL,
    reason        VARCHAR(30) NOT NULL,          -- INITIAL_TRANSFER_OUT / INITIAL_TRANSFER_IN / INITIAL_FROM_SUPERADMIN / ...
    ref_type      VARCHAR(10),                   -- อีกฝั่งของการโอน
    ref_id        BIGINT,
    request_id    VARCHAR(64) NOT NULL,
    actor_type    VARCHAR(10) NOT NULL,
    actor_id      BIGINT      NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_balance_ledger_request UNIQUE (request_id, owner_type, owner_id, currency)  -- กฎข้อ 12
);
CREATE INDEX idx_balance_ledger_owner ON balance_ledger(owner_type, owner_id, created_at);

CREATE TABLE create_requests (                   -- กันสร้างซ้ำด้วย request_id (MGMT-15A)
    request_id   VARCHAR(64) PRIMARY KEY,
    creator_type VARCHAR(10) NOT NULL,
    creator_id   BIGINT      NOT NULL,
    target_type  VARCHAR(10) NOT NULL,           -- AGENT / MEMBER
    target_id    BIGINT      NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE account_change_logs (               -- MGMT-60
    id              BIGSERIAL   PRIMARY KEY,
    actor_type      VARCHAR(10) NOT NULL,        -- AGENT / SUB
    actor_id        BIGINT      NOT NULL,
    actor_username  VARCHAR(71) NOT NULL,
    target_type     VARCHAR(10) NOT NULL,        -- AGENT / MEMBER / SUB
    target_id       BIGINT      NOT NULL,
    target_username VARCHAR(71) NOT NULL,
    action          VARCHAR(30) NOT NULL,        -- CREATE / UPDATE_INFO / UPDATE_STATUS / UPDATE_PT / ADJUST_PT / UPDATE_GAMES / SUB_STATUS / INITIAL_BALANCE
    old_value       JSONB,
    new_value       JSONB,
    ip              VARCHAR(45),
    request_id      VARCHAR(64),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_account_change_logs_target ON account_change_logs(target_type, target_id, created_at);

-- migration ใหม่ (2026-10-09) — เงินและ % เป็นทศนิยม (กฎข้อ 9 แก้ 2026-10-09) · แปลงค่าเดิม ÷100 ไปด้วย · CHECK ช่วงเดิมเป็นหน่วย %
--   agent_game_settings: pt_from_parent_bp → pt_from_parent · pt_bp → pt · force_bp → force · remain_bp → remain · commission_bp → commission
--                        ทุกตัว DOUBLE PRECISION · CHECK 0–100 (commission 0–1) · pt <= pt_from_parent
--   user_member_game_settings: pt_bp → pt · remain_bp → remain · commission_bp → commission (DOUBLE PRECISION)
--   agent_balances.amount · balance_ledger.amount / balance_after → DOUBLE PRECISION
-- migration ใหม่ (2026-10-09) — MGMT-61, MGMT-62
ALTER TABLE user_agents ADD COLUMN cnf JSONB NOT NULL DEFAULT '{"parent": []}';
CREATE INDEX idx_user_agents_cnf ON user_agents USING GIN (cnf jsonb_path_ops);          -- cnf @> '{"parent":[{"id":X}]}'
ALTER TABLE agent_game_settings ADD COLUMN parent_id BIGINT REFERENCES user_agents(id);
CREATE INDEX idx_agent_game_settings_parent ON agent_game_settings(parent_id, game_code);
-- backfill: cnf ด้วย recursive CTE ตาม parent_id · agent_game_settings.parent_id = user_agents.parent_id ของเจ้าของแถว
```

- Superadmin มีแถว `agent_game_settings` ของตัวเอง (`pt_from_parent = 100` · ไม่มี Member ตรง) · ADMIN ไม่มี · `pt` = ค่าถือจาก Member ใต้ตัวเอง (MGMT-22) · ผู้สร้างถือในสายของลูก = `pt_from_parent` ของผู้สร้าง − ของลูก (คำนวณตอนคิดเงิน ไม่เก็บ)
- สายล่างใช้ recursive CTE ตาม `parent_id` (แบบ `ListAncestorStatusesRepository`) · `idx_user_agents_parent_id` มีแล้ว · เส้นที่มีอยู่ยังใช้ CTE เดิม · query ใหม่ใช้ `cnf` ได้ (MGMT-61)
- รายการสกุล 27 สกุล · กลุ่ม PT · รหัสเกม (`coin_toss`, `rock_paper_scissors`, `scratch_card`) · เมนูสิทธิ์ เป็นค่าคงที่ใน `app/core`

## 7. Test cases

| Rule | Input | Expected |
|---|---|---|
| MGMT-02 | Company Transfer สร้าง `SHARE_B2B` / `SHARE_B2C` | สำเร็จ |
| MGMT-02 | Company Seamless Reseller สร้าง `SHARE_B2C` | สำเร็จ · `user_type = SHARE_RESELLER` |
| MGMT-02 | Company Seamless 1 to 1 สร้าง Share · Share B2B สร้าง Member | `402301` |
| MGMT-02 | Share B2C สร้าง Member · Agent สร้าง Member · Agent สร้าง Agent 3 ชั้น | สำเร็จ |
| MGMT-04 | สร้าง Company Transfer · Share B2C · Agent · Share Master | ทุกบัญชีมีแถว `api_credentials` ของตัวเอง |
| MGMT-05 | username `ab` / 33 ตัว / มี `_` | `422` · `Share01` เก็บเป็น `share01` |
| MGMT-05 | username ซ้ำ | `402401` |
| MGMT-08 | เบอร์ `+66812345678` / 7 หลัก | `422` · เบอร์ซ้ำ `402403` |
| MGMT-07 | ชื่อ `สมชาย01` · `ใจดี` | สำเร็จ · ชื่อ `สม` (2 ตัว) / `สมชาย ใจดี` (มีช่องว่าง) / `สมชาย_01` = `422` |
| MGMT-41 | สร้าง sub 2 บัญชีเบอร์เดียวกัน | สำเร็จทั้งคู่ |
| MGMT-12 | Share B2B ไม่ส่งสกุล · Share B2C ส่ง 2 สกุล | `422` |
| MGMT-14 | Agent ใต้ Share B2B (THB, USD) เลือก JPY | `402310` |
| MGMT-17 | ส่ง `"force": null` | `422` |
| ACC-32 | ส่ง `"phone": null` · รายละเอียดของบัญชีที่ไม่มีเบอร์ / ไม่เคย login | `422` · response `"phone": ""` · `"last_login_at": ""` |
| MGMT-18 | ถือ `30.25` (ไม่ลง 0.5%) · commission `1.01` · ถือ `30.12345` (ทศนิยมเกิน 4) · `balance` `10.12345` | `422` |
| กฎข้อ 9 | `balance.THB = 10000.1234` | สำเร็จ · `agent_balances.amount = 10000.1234` · detail ส่ง `10000.1234` |
| MGMT-18 | ได้รับ 80 · ให้ลูก `80.5` · ตั้ง `pt` ของตัวเอง `80.5` | `402305` |
| MGMT-18 | ผู้สร้างมี commission 0.5% ตั้งให้ลูก 0.6% | สำเร็จ |
| MGMT-19 | Company Seamless Master ตั้ง `pt` ของตัวเอง = 1 · ให้ Share Master น้อยกว่าที่ตัวเองได้รับ | `402307` |
| MGMT-23 | แก้ PT ของหลาน (ไม่ใช่ผู้สร้างโดยตรง) | `402304` |
| MGMT-23 | ส่งแค่กลุ่ม `minigame` | กลุ่มอื่นไม่เปลี่ยน |
| MGMT-22 | Superadmin ให้ comA 90 · comB 80 | comA `pt_from_parent = 90` · comB `= 80` · Superadmin ถือในสาย comA 10 · สาย comB 20
| MGMT-24 | comA ลดค่าที่ให้ share1 จาก 70 เหลือ 55 (share1 ให้ agent1 60 · `pt` 30) | `402306` msg บอกต่ำสุด 60 · ไม่มีแถวไหนเปลี่ยน |
| MGMT-24 | comA ลดค่าที่ให้ share1 เหลือ 60 · เพิ่มค่าที่ให้ share2 | สำเร็จ · share2 และลูกของ share1 ไม่เปลี่ยน |
| MGMT-26 | `parent_id` เป็นบัญชีสายอื่น | `402402` |
| MGMT-26 | Agent ที่มีทั้ง Agent และ Member เป็นลูก | ได้ทั้งสองแบบ `role` ถูกต้อง เรียง A→Z |
| MGMT-27 | `"q": "ab"` | ได้เฉพาะ username ที่มี `ab` |
| MGMT-27A | comp01 ค้น `"q": "sh"` (sh1 sh2 ลูกตรง · sh3agent อยู่ใต้ sh1) | ได้ทั้ง 3 แถวพร้อม `parent_username` · ไม่เห็นตัวเอง ชั้นบน และสายอื่น |
| MGMT-27A | share ค้นด้วยคำที่ตรงกับ Company ของตัวเอง หรือ share สายข้าง | ไม่พบ (`total_count = 0`) |
| MGMT-27A | `"q": "s"` (1 ตัว) · ไม่ส่ง `q` | `422` |
| MGMT-27B | ADMIN ค้น `"  COMP01 "` · `"comp"` · `"comp01@staff"` · `"mem01"` · `"superadmin"` | comp01 1 แถว · `[]` · sub (`role` ของเจ้าของ `parent_username = comp01`) · Member · SUPERADMIN `parent_username = ""` |
| MGMT-27B | หัวสายถูกระงับ · SUPERADMIN / Company เรียก · `username` ว่าง / `null` | ลูก `status = SUSPENDED` · `401308` · `422` |
| MGMT-29 | ดูรายละเอียด | ไม่มี field รหัสผ่าน / passcode / hash |
| MGMT-16 | comp01 สร้างด้วย `pt.minigame.pt_from_parent = 70` | `agent_game_settings` มี 3 แถว ค่า `pt_from_parent = 70` · `pt = 70` (เริ่มถือทั้งหมด — MGMT-22) · รายละเอียดแสดง `pt.minigame.pt_from_parent = 70` · `pt = 70` · `created_by` = `updated_by` = `"comp01"` · `status_game` ครบ 3 เกม |
| MGMT-16 | sub `comp01@staff` แก้ PT ให้ลูก | `pt.minigame.updated_by = "comp01@staff"` · `updated_at` เปลี่ยน · `created_by` ยังเป็น `"comp01"` |
| MGMT-30 | Share แก้สถานะ Agent ของ Agent ลูก (ไม่ใช่ผู้สร้างโดยตรง) | `402304` |
| MGMT-30 | LOCKED → ACTIVE โดยผู้สร้าง | สำเร็จ |
| MGMT-31 | Company ระงับ Share | Share และ Agent ใต้ Share `status = SUSPENDED` · Company `status = ACTIVE` |
| MGMT-30 | Share ล็อก Agent ลูก (Share ACTIVE) | Agent `status = LOCKED` · Share ยังใช้งานได้ |
| MGMT-30 | หัวสายของ Agent ถูกระงับ · ผู้สร้างตั้ง Agent เป็น ACTIVE | สำเร็จ · Agent ยังแสดง `status = SUSPENDED` |
| MGMT-43 | เจ้าของตั้ง sub เป็น `INACTIVE` | sub login ได้ · เรียก route อื่นนอก Profile / Report ได้ `401311` · รายชื่อ sub แสดง `INACTIVE` · ตั้งกลับ `ACTIVE` แล้วใช้งานได้ |
| MGMT-45 | ชั้นบนแก้หรือเปลี่ยนสถานะ sub ของบัญชีในสายล่าง | `402404` · ดูรายชื่อ / รายละเอียดได้ |
| MGMT-40 | sub เรียก `POST /manage/subaccounts/create` หรือ `/manage/subaccounts/list` | `402311` |
| MGMT-51 | sub มี `member` edit · `pt` off สร้างบัญชี | `402303` |
| MGMT-51 | sub มี `member` view · `pt` off ดูรายชื่อ | สำเร็จ · ไม่มี field `pt` |
| MGMT-51 | sub มี `pt` edit · `member` off แก้ PT | สำเร็จ |
| MGMT-50 | สร้าง sub ส่งแค่ `{ "member": "edit" }` | เมนูอื่นเป็น `off` ทั้งหมด |
| MGMT-50 | ส่งระดับ `"VIEW"` หรือ `"none"` | `422` |
| MGMT-41 | สร้าง sub `name_suffix = "Staff"` · login `Comp01@STAFF` | เก็บ `comp01@staff` · login สำเร็จ |
| MGMT-15A | Company Transfer มี THB 50,000 สร้าง Share พร้อม `balance` THB 10,000 | Company เหลือ 40,000 · Share มี 10,000 · ledger 2 แถว (ออก / เข้า) |
| MGMT-15A | ยอดของผู้สร้างไม่พอ | `402312` · ไม่สร้างบัญชี · ยอดไม่เปลี่ยน |
| MGMT-15A | Superadmin สร้าง Company Transfer พร้อม `balance` THB 1,000,000 | สำเร็จเสมอ · Company มี 1,000,000 · ledger ฝั่ง Company 1 แถว · ไม่มีแถวยอดของ Superadmin |
| MGMT-15A | ส่งซ้ำด้วย `request_id` เดิม | ได้ `id` เดิม · ไม่สร้างซ้ำ · ไม่โอนซ้ำ |
| MGMT-15A | ส่ง `balance` ให้บัญชีฝั่ง Seamless · สกุลที่บัญชีใหม่ไม่มี · ค่า 0 หรือติดลบ | `422` |
| MGMT-15A | สร้างพร้อมกัน 2 คำขอ ยอดพอแค่คำขอเดียว | สำเร็จ 1 · อีกอัน `402312` · ยอดไม่ติดลบ |
| MGMT-60 | แก้ PT สำเร็จ | มี log ค่าเก่า / ใหม่ ไม่มีรหัสผ่าน |

| MGMT-05 | สร้าง Member ชื่อเดียวกับ Agent ที่มีอยู่ | `402401` |
| MGMT-20 | Company ปิด `scratch_card` แล้ว Share ใต้ Company ตั้งเปิดให้ Agent | Agent ยังเล่น `scratch_card` ไม่ได้ (เกมปิดจากหัวสาย) |
| MGMT-25 | ให้ลูก 60 · `force` 70 ในคำสั่งเดียวกัน | `402308` |
| MGMT-22 | agent1 ได้รับ 60 ตั้ง `pt` 40 · Member แพ้ 1,000 | agent1 ได้ 400 · 200 คืน Superadmin (ไม่มี Force / Remain) |
| MGMT-22 | agent1 ตั้ง `pt` 61 | `402305` |
| MGMT-23 | ผู้สร้างแก้ `pt` ของลูก · ลูกแก้ `pt_from_parent` ของตัวเอง | `422` (field ไม่อยู่ในเส้นนั้น) |
| MGMT-20 | ผู้สร้างตั้ง `pt.minigame.status = false` | บันทึกสำเร็จ · รายละเอียดแสดง `false` · `status_game` ทุกเกมไม่เปลี่ยน (เกมยังเปิด) |
| MGMT-20 | comp01 `update-games` ปิด `scratch_card` ของ share01 แล้วเปิดกลับ | `status_game` ของ share01 เปลี่ยนเฉพาะเกมนั้น · agent ใต้ share01 ไม่เปลี่ยน · ค่า PT / `updated_by` ไม่เปลี่ยน · log `UPDATE_GAMES` 2 แถว |
| MGMT-20 | `update-games` กับลูกของลูก · นอกสาย · เกมไม่มีในระบบ / ว่าง / `null` | `402304` · `402402` · `422` |
| MGMT-52 | sub ของ Superadmin ได้สิทธิ์ `announcement` | `422` |
| MGMT-52 | Superadmin ให้ sub `rate` = `edit` · Company ให้ sub `rate` | สำเร็จ · `422` |

### 7.1 ลำดับเช็ค (แก้ 2026-10-09 — ทุกเส้นใน module นี้)

ไล่ **บนลงล่างตามลำดับ field ใน body** 2 รอบ · ตอบ error ของ field แรกที่ผิด:

1. **รูปแบบ** (DTO — ไม่ต้องดู DB) ทุก field บนลงล่าง → `422`
2. **กฎที่ต้องดู DB** บนลงล่าง — เส้นสร้างฝั่ง agent:
   `user_type` (`402301`) → `username` ซ้ำ (`402401`) → `phone` ซ้ำ (`402403`) → `currencies` (`422` / `402310`) →
   `balance` (`422` / `402312`) → `pt` ทีละ field: `pt_from_parent` (`402305` / `402307`) → `force` → `remain_quota` (`402308`) → `commission_percent` (`402309`)
   · `update-pt`: `pt_from_parent` (เพดาน `402305` → ต่ำสุดที่ลูกใช้ `402306`) → `force` → `remain_quota` → `commission_percent`
3. สิทธิ์ (`401308` / `402303` / `402311`) เช็คที่ middleware ก่อนทั้งหมด

ค่าที่เป็น object (`pt`, `balance`, `status_game`, `permissions`) ไล่ key ตามลำดับตัวอักษร — error ออกตัวเดียวกันทุกครั้ง

**ข้อความ error บอก field และค่าที่ตั้งได้** (code เดิม): เช่น `402301` → `user_type: ส่ง SHARE_B2C ไม่ได้ · SHARE_B2C สร้างได้โดย COMPANY_TRANSFER, … เท่านั้น แต่คุณเป็น SUPERADMIN · SUPERADMIN สร้างได้เฉพาะ COMPANY_TRANSFER, …` (บอกเหตุผล: ใครสร้างได้ · ไม่มีประเภทนี้ · MEMBER ใช้เส้น members/create · SHARE_RESELLER / SHARE_MASTER ให้ส่ง SHARE_B2C) ·
`422` currencies → `currencies: COMPANY_TRANSFER ได้ครบทุกสกุล ห้ามส่ง currencies` · `402305` → `pt.minigame.pt_from_parent ตั้งได้ไม่เกิน 90 (ค่าที่คุณได้รับ)` ·
`402308` → `pt.minigame.force ตั้งได้ไม่เกิน 50 (ค่าที่ให้ลูก)` · `402312` → `balance.THB ยอดของคุณไม่พอ (มี 0)` · สิทธิ์ sub → `permissions.rate ไม่มีเมนูนี้ · เมนูที่ให้ได้: …`

## 8. Contract changes (แจ้ง frontend)

- เส้นทั้งหมดในหัวข้อ 5 เป็นเส้นใหม่
- ค่า % ส่งเป็น JSON number (ACC-18) อยู่ใน object `pt` แยกกลุ่ม · ห้าม `null` · ไม่ตั้ง = `0`
- **ไม่มี `null` ใน API** (account ACC-32): ข้อความว่าง = `""` · ตัวเลขว่าง = `0` · request ส่ง `null` = `422`
- สิทธิ์ของ sub ส่งเป็น object `{ "{menu}": "off" | "view" | "edit" }` (รวม Profile — account ACC-12)
- ใช้แค่ method `GET` และ `POST` · ระบุบัญชีด้วย `id` ใน body (เส้นอ่านที่ต้องระบุบัญชีเป็น `POST`) · ไม่มีเส้นลบ sub (ใช้สถานะ `INACTIVE`)
- เส้นสร้างรับ `request_id` (บังคับ) และ `balance` (ไม่บังคับ — ยอดเงินตั้งต้น)
- `pt` ใน response ทุกเส้น (รวม account Profile) เป็นชุดเดียวใต้ระบบ `minigame` + `created_at` · `created_by` · `updated_at` · `updated_by` · `status_game` แยกต่อเกมด้านนอกเหมือนเดิม (แก้ 2026-10-08) · key ของระบบตอนส่งเปลี่ยนจาก `game` เป็น `minigame`

## 9. Error codes (`bb=02`) — business error ตอบ HTTP 200

| Code | ความหมาย |
|---|---|
| 402301 | สร้างบัญชีประเภทนี้ไม่ได้ |
| 402303 | ไม่มีสิทธิ์ใช้งานเมนูนี้ |
| 402304 | แก้ไขได้เฉพาะผู้สร้างของบัญชีนี้ |
| 402305 | ค่าถือเกินกว่าที่ได้รับ |
| 402306 | ค่าที่ให้ลูกต่ำกว่าที่ลูกใช้อยู่ (msg บอกค่าต่ำสุดที่ตั้งได้) |
| 402307 | Company Seamless Master ต้องถือ 0 และปล่อยทั้งหมด |
| 402308 | Force / Remain เกินที่กำหนด |
| 402309 | Commission เกินที่กำหนด |
| 402310 | สกุลเงินไม่อยู่ในสกุลของผู้สร้าง |
| 402311 | บัญชีย่อยทำรายการนี้ไม่ได้ |
| 402312 | ยอดเงินของคุณไม่พอสำหรับยอดเงินตั้งต้น (MGMT-15A) |
| 402401 | username นี้ถูกใช้แล้ว |
| 402402 | ไม่พบบัญชีในสายของคุณ |
| 402403 | เบอร์โทรนี้ถูกใช้แล้ว |
| 402404 | ไม่พบบัญชีย่อยของคุณ |

`402302` (เดิม: บัญชีถูกระงับ) ตัดออก — AUTH-54 บังคับที่ middleware แทน
