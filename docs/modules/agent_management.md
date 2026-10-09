# การจัดการสมาชิก (`agent_management`) — Spec

- สถานะ: **APPROVED** · อนุมัติโดย: lead (zerph) · วันที่: 2026-10-07
- **แก้ใหญ่ตาม lead review 2026-10-09** ([docs/reviews/agent-management.md](../reviews/agent-management.md) — U1–U4 · H1–H5 · Q-R1–R4 · Q-P1–P2 · Q-C1–C3 · E1–E6 · N1 · P3–P5 · B1–B3):
  โครงเส้นใหม่ E1–E6 (หัวข้อ 5) · Share Reseller / Share Master สร้างได้แค่ Member · Company Seamless Master ถือสู้ไม่ได้ ระบบปรับ Share Master ตาม ·
  เบอร์โทรแยก `phone_country_code` + `phone` · ลบเมนูสิทธิ์ `pt` (ทุกเส้นใช้ `member`) · เมนู `api_credential` · `status_game_effective` ·
  lead อนุมัติงานก่อนหน้า: ลบค่าถือ `pt` ค่าเดียวของ agent + `update-hold` (U1) · MGMT-24 R1–R3 (U2) · ลำดับเช็ค 7.1 + ข้อความ error (U3) · เปลี่ยนชื่อตาราง Member (U4)
- การแก้ก่อนหน้า (สรุป): 2026-10-08 ชื่อเป็นภาษาไทยได้ · เบอร์ของ sub ซ้ำได้ · ยอดเงินตั้งต้นอยู่ใน module นี้ · `pt` ชุดเดียวต่อระบบ (`minigame`) · `pt.status` = รับ PT ไหม ·
  2026-10-09 เงินและ % เป็น float ปัด 4 ตำแหน่ง (CLAUDE.md กฎข้อ 9 — lead อนุมัติ) · `user_agents.cnf` + `agent_game_settings.parent_id` (MGMT-61 / 62 — lead อนุมัติ) · แถวรายชื่อมี `last_login_*` + `created_at`
- **โค้ด:** ทำครบตาม review แล้ว — PR 2 (เส้น E1–E6 · สิทธิ์ · Share Reseller / Master) · PR 3 (CSM → Share Master · เบอร์ 2 field + migration) · ส่วน Member (เบอร์ · path `members/*` · `amounts`) อยู่ใน member_management
- ชื่อ module ในโค้ด: `agent_management` (`controllers/` `dto/` `service/` `core/agent_management`) · เส้น `/manage/members/*` อยู่ module `member_management` ([member_management.md](member_management.md) · ผู้ดูแล maofoy) · กฎ business ร่วมอยู่ที่ `core/agent_management` และ spec ฉบับนี้
- เมนู: 2 การจัดการสมาชิก — เพิ่มบัญชี · รายชื่อดาวน์ไลน์ (ไล่ลงได้ถึง Member · ค้นทั้งสาย) · แก้ไข · บัญชีย่อย
- ที่มาของ rule: เอกสารของ lead (Company Hierarchy, PT, PT Force, PT Remain, PT Commission, System Overview) · review ของ lead · คำตอบของ boiledegg · ไม่ได้ extract จากโค้ดเก่า
- ทีมเราทำฝั่ง API — การแสดงผล (dropdown, ฟอร์ม, รูปแบบตัวเลขบนจอ) เป็นของหน้าบ้าน

> เอกสารนี้คือ source of truth ของ module — โค้ดและ test ต้องตรงกับเอกสารนี้ ถ้าไม่ตรงให้แก้โค้ด
> หรือแก้เอกสารผ่านการอนุมัติใหม่ ห้ามอ้างเอกสารของโปรเจกต์เก่า

## 1. หน้าที่ของ module

สร้างและจัดการบัญชีทุกชั้นในสาย: Company · Share · Agent · บัญชีย่อย (sub) — สร้าง, ดูรายชื่อไล่ลงทีละชั้น / ค้นทั้งสาย, ดูรายละเอียด,
แก้ข้อมูล + ค่าหุ้นส่วน (PT · Force · Remain · Commission) + เปิด / ปิดเกม ในเส้นเดียว, แก้สถานะ และกำหนดสิทธิ์ของ sub (ระบบสิทธิ์ของทั้งหลังบ้านกำหนดที่นี่ครั้งเดียว)

**ไม่อยู่ใน module นี้:** เส้นของ Member (member_management) · การเติม / ถอน / โอนเงินหลังสร้างบัญชี (module การชำระเงิน — ใช้ตาราง `agent_balances` ·
`user_members.credit` · `balance_ledger` ที่สร้างใน module นี้ต่อ) · **ยอดเงินตั้งต้นตอนสร้างบัญชีอยู่ใน module นี้** (MGMT-15A) ·
การคิด PT / Force / Remain / Commission ตอน settle (module เดิมพัน) · Key / ลิงก์ตอบกลับ / IP (account 1.3 — module นี้แค่สร้าง Key ตอนสร้างเจ้าของ ACC-05)

## 2. คำศัพท์

| คำ | ความหมาย |
|---|---|
| ฝั่ง agent | Superadmin · Company · Share · Agent (ตาราง `user_agents`) |
| Member | ผู้เล่น (ตาราง `user_members` แยกจากฝั่ง agent) |
| ผู้สร้าง | บัญชีที่สร้างบัญชีนั้น (`parent_id` / `agent_id` ของ Member) — ลูกตรง = บัญชีที่ตัวเองสร้าง |
| สายล่าง | ทุกบัญชีใต้ตัวเองทุกชั้น ไม่นับตัวเอง |
| `user_type` | ประเภทบัญชี (MGMT-01) — ตรงกับ account ACC-12 |
| CSM | Company Seamless Master · ลูกของ CSM คือ Share Master (`SHARE_MASTER`) |
| กลุ่ม PT | ระบบที่มีเรื่อง PT ใช้เป็น key ของ `pt` (`minigame` = ทุกเกม minigame · ระบบอื่นเพิ่มได้โดยไม่เปลี่ยนโครงสร้าง) · รับค่าชุดเดียวต่อระบบ · เก็บต่อเกม |
| ได้รับ / ให้ลูก | ได้รับ (`pt_from_parent`) = PT ที่ผู้สร้างให้ · ให้ลูก = PT ที่แบ่งให้ลูกแต่ละคน (ไม่เกินที่ได้รับ) · ผู้สร้างถือในสายของลูก = ได้รับ − ให้ลูกคนนั้น · ถือสู้กับ Member ตั้งต่อ Member (member_management MGMT-21) |
| ทศนิยม 4 ตำแหน่ง | เงินและ % เก็บเป็น `float64` / `DOUBLE PRECISION` ปัด 4 ตำแหน่ง (CLAUDE.md กฎข้อ 9) · API ส่งค่าตามที่เก็บ หน้าบ้านปัด 3 ตำแหน่งเอง |
| `status` | สถานะที่ใช้งานจริง key เดียว (account ACC-30) = เข้มที่สุดของสถานะที่ตั้งกับบัญชีเอง, ผู้สร้าง (sub) และหัวสายทุกชั้น (AUTH-53) · DB เก็บสถานะที่ตั้งกับบัญชีเองแยกไว้ |
| `status_game` / `status_game_effective` | เปิด / ปิดเกมที่ตั้งกับบัญชีเอง · เปิดจริงเมื่อบัญชีและหัวสายทุกชั้นเปิด (MGMT-20) |

## 3. Business rules

### ประเภทและผู้สร้าง

| ID | Rule |
|---|---|
| MGMT-01 | `user_type`: `SUPERADMIN` · `ADMIN` · `COMPANY_TRANSFER` · `COMPANY_SEAMLESS_RESELLER` · `COMPANY_SEAMLESS_MASTER` · `COMPANY_SEAMLESS_1TO1` · `SHARE_B2B` · `SHARE_B2C` · `SHARE_RESELLER` · `SHARE_MASTER` · `AGENT` · `MEMBER` · เก็บเป็น `role` + `agent_type` ใน `user_agents` (Member อยู่ตาราง `user_members`) |
| MGMT-02 | ใครสร้างอะไรได้ (แก้ 2026-10-09 — lead H1): |
| | • Superadmin → Company ทั้ง 4 ประเภท |
| | • Company Transfer → Share B2B · Share B2C |
| | • Company Seamless Reseller → Share B2C (เก็บเป็น `SHARE_RESELLER`) · CSM → Share B2C (เก็บเป็น `SHARE_MASTER`) |
| | • Company Seamless 1 to 1 → Member |
| | • Share B2C → Agent หรือ Member |
| | • **Share Reseller · Share Master → Member เท่านั้น** (สร้าง Agent = `402301`) |
| | • Share B2B → Agent เท่านั้น (สร้าง Member ตรงไม่ได้ ต้องผ่าน Agent) |
| | • Agent → Agent (ซ้อนได้ไม่จำกัดชั้น) หรือ Member |
| | ADMIN สร้างไม่ได้ (AUTH-43) · ประเภทการเดิมพัน (Transfer / Seamless) ตามผู้สร้าง เลือกไม่ได้ |
| MGMT-03 | ผู้สร้างต้อง `status = ACTIVE` (AUTH-54 บังคับที่ middleware) · บัญชีใหม่ `status = ACTIVE` |
| MGMT-04 | สร้างบัญชีฝั่ง agent ทุกประเภท (Company / Share / Agent) → สร้าง Key ของ account 1.3 ใน tx เดียวกัน (ACC-05) |

### ข้อมูลบัญชี

| ID | Rule |
|---|---|
| MGMT-05 | username (ฝั่ง agent และ Member): 3–32 ตัว `a-z` `0-9` (รับตัวพิมพ์ใหญ่แล้วแปลงเป็นเล็ก — AUTH-01) · **ห้ามซ้ำทั้งระบบ** ข้ามตาราง `user_agents` และ `user_members` (เช็คใน tx พร้อม `pg_advisory_xact_lock` ของ username) · แก้ไม่ได้ (AUTH-19) |
| MGMT-06 | รหัสผ่านตาม AUTH-36 · ฝั่ง agent ตั้ง passcode ตอน login ครั้งแรก (AUTH-29) |
| MGMT-07 | ชื่อ: บังคับ · 3–32 ตัวอักษร (นับเป็นตัวอักษร) · ภาษาไทย `A-Z` `a-z` `0-9` · ไม่มีช่องว่าง |
| MGMT-08 | **เบอร์โทร 2 field** (แก้ 2026-10-09 — lead E1-5): `phone_country_code` = รหัสโทรออกของประเทศ ตัวเลข 1–3 หลัก ไม่มี `+` (เช่น `66`) ต้องเป็นรหัสที่มีจริง (รายการใน `app/core`) · `phone` = เบอร์ในประเทศ ตัวเลขล้วน **ไม่มี 0 นำหน้า** (เช่น `812345678`) · รวมสองค่าไม่เกิน 15 หลัก (E.164) · ไม่กรอก = `""` ทั้งคู่ · ส่งค่าเดียว = `422` · ห้าม `null` · **ห้ามซ้ำที่คู่** (code + phone) ภายในตารางเดียวกัน · sub ซ้ำได้ (MGMT-41) · ใช้รูปแบบนี้ทุกที่: agent · sub · Member |
| MGMT-08A | migration เบอร์เดิม (lead B1): ขึ้นต้น `0` → `66` + ตัด 0 · ขึ้นต้น `66` และยาว 11 หลัก → `66` + ส่วนที่เหลือ · **แบบอื่นทั้งหมดล้างเป็น `""`** (ไม่เดาประเทศ) · ฝั่ง agent ถ้าแปลงแล้วคู่ซ้ำกัน (เช่น `0812345678` กับ `66812345678`) เก็บแถวที่ `id` น้อยสุด แถวอื่นล้าง · ทุกรายการที่ถูกล้างบันทึกในตาราง `migration_phone_cleared` (`table_name` · `row_id` · `old_phone` · `reason` = `FORMAT` / `DUPLICATE`) |
| MGMT-09 | แก้ได้ (ผู้สร้างโดยตรง): ชื่อ · เบอร์โทร · ค่าหุ้นส่วน · เปิด / ปิดเกม (เส้น `agents/detail/update`) · สถานะ (`agents/status/update`) · แก้ไม่ได้: username · ประเภท · สกุลเงิน · ไม่มีการลบบัญชีฝั่ง agent และ Member |
| MGMT-61 | **สายชั้นบน** `user_agents.cnf` (JSONB): `{"parent": [{"id": 1, "position": "superadmin"}, {"id": 4754, "position": "company"}, …]}` · เรียงจาก Superadmin ลงมาถึงผู้สร้างตรง · **ไม่รวมตัวเอง** · `position` = role ตัวเล็ก · Superadmin / ADMIN = `{"parent": []}` · ตั้งตอนสร้าง = `parent` ของผู้สร้าง + ผู้สร้าง · ไม่เปลี่ยนหลังสร้าง · **ไม่ส่งใน API** · ใช้ไล่สายขึ้นและค้นทั้งสายลง (`cnf @> '{"parent":[{"id":X}]}'` + GIN index) · Member ใช้ `user_members.cnf` (member_management) |
| MGMT-62 | `agent_game_settings.parent_id` = id ผู้สร้างของบัญชีเจ้าของแถว · Superadmin = `NULL` · ตั้งตอนสร้าง |

### สกุลเงิน

ระบบมี 27 สกุล: ARS AUD BDT BOB BRL CLP CNY EUR GBP HKD IDR INR JPY KHR KRW LAK MMK MXN MYR NGN PHP PKR THB TWD USD USDT VND · ทุกสกุลทศนิยมไม่เกิน 4 ตำแหน่ง (ACC-18)

| ID | Rule |
|---|---|
| MGMT-10 | Company Transfer · Company Seamless Reseller / Master: มีครบทุกสกุล (ไม่ต้องส่ง) |
| MGMT-11 | Company Seamless 1 to 1: 1 สกุล ที่ Superadmin เลือก |
| MGMT-12 | Share B2B: เลือกได้หลายสกุล อย่างน้อย 1 · Share B2C / Reseller / Master: 1 สกุล |
| MGMT-13 | Agent ที่ Share B2B สร้าง: 1 สกุล · Agent ที่ Share B2C หรือ Agent สร้าง และ Member: ใช้สกุลของผู้สร้าง (ไม่ต้องส่ง) |
| MGMT-14 | สกุลที่เลือกต้องอยู่ในสกุลของผู้สร้าง (`402310`) · เปลี่ยนหลังสร้างไม่ได้ |
| MGMT-15 | บัญชีฝั่ง Seamless ทุกชั้น (Company Seamless ทุกประเภท และสายล่าง) ไม่มียอดเงิน · ส่ง `balance` ตอนสร้าง = `422` |
| MGMT-15A | **ยอดเงินตั้งต้นตอนสร้าง** (`balance` · ไม่บังคับ · ฝั่ง Transfer เท่านั้น): ต่อสกุล · ต้องเป็นสกุลของบัญชีใหม่ · มากกว่า 0 · ทศนิยมไม่เกิน 4 ตำแหน่ง · **โอนจากยอดของผู้สร้าง** (ผู้สร้างลด · บัญชีใหม่เพิ่ม) · ยอดผู้สร้างไม่พอ = `402312` |
| | • สร้างบัญชี + โอนใน **transaction เดียวกัน** · lock แถวยอดของผู้สร้างก่อนอ่าน (`FOR UPDATE` เรียงสกุล) · ledger ทั้งสองฝั่งใน tx เดียวกัน (กฎข้อ 10–11) |
| | • กันยิงซ้ำ: `request_id` (UUID · บังคับ) จากหน้าบ้าน · unique ใน DB · ส่งซ้ำด้วยค่าเดิม = ได้บัญชีเดิม ไม่สร้าง / ไม่โอนซ้ำ (กฎข้อ 12) · **`request_id` ที่ผู้อื่นหรือเส้นอื่นใช้แล้ว = `422` "request_id นี้ถูกใช้แล้ว"** (lead Q-R4) |
| | • ไม่ต้องยืนยัน passcode · ส่ง `balance` ต้องมีสิทธิ์ `payment` = edit ด้วย (MGMT-51 · เช็คใน service หลังตรวจรูปแบบ — 7.1) |
| | • **Superadmin วงเงินไม่จำกัด**: ไม่มียอดของตัวเองให้หัก · ledger บันทึกเฉพาะฝั่งที่ได้รับ (`reason = INITIAL_FROM_SUPERADMIN`) |
| | • ตารางยอดเงินและ ledger เป็นของ module การชำระเงิน — ออกแบบขั้นต่ำไว้ในหัวข้อ 6 |

### ค่าหุ้นส่วน (PT · Force · Remain · Commission)

| ID | Rule |
|---|---|
| MGMT-16 | **รับค่าชุดเดียวต่อกลุ่ม · เก็บต่อเกม**: |
| | • ส่งเป็น object `pt` แยกตาม**กลุ่ม PT** · ระบบกระจายค่าลงทุกเกมในกลุ่ม แล้วเก็บทีละเกม (`agent_game_settings`) · ค่าของเกมในกลุ่มเท่ากันเสมอ |
| | • ตอนนี้มีกลุ่ม `minigame` กลุ่มเดียว = `coin_toss` · `rock_paper_scissors` · `scratch_card` · การจับคู่กลุ่ม → เกม กำหนดใน `app/core` |
| | • ผู้สร้างตั้งให้ลูก: กลุ่มละ 5 ค่า `pt_from_parent` · `force` · `remain_quota` · `commission_percent` · `status` (รับ PT ไหม — MGMT-20) · ยกเว้น Share Master (MGMT-19) |
| | • แสดง (response ทุกเส้น): `pt` → ระบบ → `pt_from_parent` · `force` · `remain_quota` · `commission_percent` · `status` · `created_at` · `created_by` · `updated_at` · `updated_by` · `status_game` → รหัสเกม → bool แยกด้านนอก `pt` · **แสดงเสมอ** (ไม่มีเมนูสิทธิ์ `pt` แล้ว — P4) |
| | • `created_by` / `updated_by` = username ของ**คนที่กดจริง** (sub = `comp01@staff` — lead Q-R2) · `created_*` ตั้งตอนสร้างบัญชีไม่เปลี่ยนอีก · `updated_*` = คนที่แก้ล่าสุด (ระบบปรับ Share Master ตาม CSM = username ของคนที่แก้ CSM) |
| | • เกมใหม่ที่เพิ่มเข้ากลุ่มภายหลัง: ทุกบัญชีได้ค่าของกลุ่มนั้น และ `status_game = true` อัตโนมัติ |
| MGMT-17 | ค่า % ใน API เป็น JSON number ทศนิยมไม่เกิน 4 ตำแหน่ง · ภายในเป็น `float64` ปัด 4 ตำแหน่ง (กฎข้อ 9) · **ห้าม `null`** · ไม่ตั้ง Force / Remain = `0` |
| MGMT-18 | ค่าที่ให้ลูก (`pt_from_parent`): 0 ถึง**ค่าที่ตัวเองได้รับ** ทีละ 0.5% (เกิน = `402305`) · Force / Remain: 0 ถึงค่าที่ให้ลูกคนนั้น ทีละ 0.5% (`402308`) · Commission: 0–1% ทีละ 0.1% (`402309`) · Commission ของลูกตั้ง**เกินของผู้สร้างได้** แต่ไม่เกิน 1% |
| MGMT-19 | **Company Seamless Master (CSM) — lead H2 / H3 / Q-R1 / E1-1 / B2:** |
| | • CSM **ถือสู้ไม่ได้** · ได้รับจาก Superadmin เท่าไรต้องส่งให้ Share Master ทั้งหมด · Share Master มีได้ไม่จำกัด |
| | • Share Master ใช้ค่าตาม CSM อัตโนมัติ: `pt_from_parent` = ค่าที่ CSM ได้รับ · `force` = `remain_quota` = 0 · `status` = ของ CSM · `status_game` = ของ CSM |
| | • **Commission** CSM ตั้งให้ Share Master แต่ละคนแยกกันได้ |
| | • สร้าง Share Master: `pt.minigame` ส่งแค่ `commission_percent` · `status_game` ห้ามส่ง · ส่ง field ที่ระบบตั้งเอง = `422` "Share Master ใช้ค่าตาม Company Seamless Master" |
| | • CSM แก้ Share Master (`agents/detail/update`): `pt.minigame` ส่งได้แค่ `commission_percent` · ส่ง `status_game` = `422` · `info` (ชื่อ / เบอร์) แก้ได้ปกติ |
| | • Superadmin แก้ PT / `pt.status` / `status_game` ของ CSM → **ระบบตั้งค่าของ Share Master ทุกคนใต้ CSM ให้ตามใน tx เดียว** (ข้อยกเว้นของ MGMT-24 "ระบบไม่แก้ค่าลูกให้เอง" — เฉพาะคู่นี้) · ลด: ค่าใหม่ต้อง ≥ `pt` ที่ Share Master ทุกคนถือสู้กับ Member ของตัวเอง ไม่งั้น `402306` · เพิ่ม: ตามทันที · `remain_quota` ของ Member ใต้ Share Master คำนวณใหม่ (R2) · บันทึก `account_change_logs` ของ Share Master แต่ละคน (`action = SYNC_FROM_CSM`) |
| | • `pt` ที่ Share Master ถือสู้กับ Member: Share Master ตั้งเองต่อ Member (member_management MGMT-21 — lead H4) |
| MGMT-19A | Company Seamless Reseller **ถือสู้ได้** และตั้งค่าให้ Share Reseller แต่ละคนแยกกันได้ตามกฎทั่วไป (MGMT-22 — lead H5) |
| MGMT-20 | **`pt.status` กับ `status_game` แยกหน้าที่กัน**: |
| | • `pt` → ระบบ → `status` (bool) = รับ PT ในระบบนั้นไหม · `false` = เกมยังเปิด แต่บัญชีนี้ไม่รับ PT · PT ส่วนนั้นไปอยู่กับใคร: **open item ต้องตัดสินก่อนเริ่ม module settle** (lead Q-R3) |
| | • `status_game` → รหัสเกม (bool) = เปิด / ปิดเกมรายบัญชี · ผู้สร้างตั้งให้ · ชั้นบนปิดแล้วสายล่างเปิดเองไม่ได้ — เกมเปิดจริง (`status_game_effective`) เมื่อบัญชีและหัวสายทุกชั้นเปิด · `false` = Member ในสายเล่นเกมนั้นไม่ได้ (บังคับใน module เดิมพัน) |
| MGMT-21 | สร้างบัญชีฝั่ง agent ต้องส่งครบทุกกลุ่มที่มี · Member: ผู้สร้างตั้ง `pt` (ถือสู้กับ Member คนนี้) + `commission_percent` · `remain_quota` ระบบคิด — รายละเอียดใน member_management MGMT-21 |
| MGMT-22 | **ผู้สร้างกำหนดค่าที่ให้ลูกแต่ละคนแยกกัน** (ทุกชั้น รวม Superadmin → Company ยกเว้น CSM → Share Master): ลูกแต่ละคนได้รับไม่เท่ากันได้ · ห้ามเกินค่าที่ผู้สร้างได้รับ · Superadmin ได้รับ 100% · ผู้สร้างถือในสายของลูกคนนั้น = ค่าที่ได้รับ − ค่าที่ให้ลูก (ระบบคำนวณ) · ถือสู้กับ Member ตั้งต่อ Member · ส่วนที่ไม่มีใครถือผ่าน Force / Remain ตอนคิดเงินก่อน ถ้าไม่มีก็**คืน Superadmin** |
| MGMT-23 | ค่าที่ให้ลูก, Force, Remain, Commission, เปิด / ปิดเกม แก้ได้เฉพาะ**ผู้สร้างโดยตรง** (หลาน = `402304` · นอกสาย = `402402` · ตัวเอง = `402304`) · sub ตามสิทธิ์ `member` = edit · แก้เฉพาะกลุ่มที่ส่ง · ในกลุ่มที่ส่งต้องครบ 5 ค่า (ยกเว้น Share Master — MGMT-19) · มีผลกับ bet ใหม่ |
| MGMT-24 | **ลดค่าที่ให้ลูก**: ค่าใหม่ต้อง ≥ ค่าที่มากที่สุดระหว่าง (ค่าที่ลูกให้ลูกของมันแต่ละคน, `pt` ที่ลูกถือสู้กับ Member แต่ละคนที่ลูกสร้าง — R1) ไม่งั้น**ปฏิเสธทั้งคำสั่ง** (`402306` msg บอกค่าต่ำสุด) · ระบบไม่แก้ค่าของลูก หลาน หรือ `pt` ของ Member ให้เอง (ยกเว้น CSM → Share Master — MGMT-19) · **เพิ่ม**: ได้ถึงค่าที่ผู้สร้างได้รับ · **สำเร็จแล้ว** `remain_quota` ของ Member ทุกคนที่ลูกสร้าง = ค่าใหม่ − `pt` ของ Member คนนั้น ใน tx เดียวกัน (R2) · lock `agent_game_settings` ของลูกและลูกของลูก เรียงตาม id **ก่อน** `user_member_game_settings` (R3) |
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
 └─ Member m1 pt = 30 → share1 ถือสู้ m1 (remain_quota 40)
agent1 (ได้รับ 60)
 └─ Member m2 pt = 40 → agent1 ถือสู้ m2 · remain_quota 20 ผ่าน Force / Remain ไม่งั้นคืน Superadmin

CSM (ได้รับ 80 · ถือสู้ไม่ได้)
 ├─ Share Master sm1: pt_from_parent 80 · force 0 · remain 0 · commission 0.3 (CSM ตั้ง)
 └─ Share Master sm2: pt_from_parent 80 · force 0 · remain 0 · commission 0.5 (CSM ตั้ง)
```

m2 แพ้ 1,000: Superadmin 10% (100) · comA 20% (200) · share1 10% (100) · agent1 40% (400) · เหลือ 20% (200) คืน Superadmin

MGMT-24: comA ลดค่าที่ให้ share1 ได้ต่ำสุด 60 (ค่าที่มากที่สุดของ share1 = ให้ agent1 60 · `pt` ของ m1 30) · ตั้ง 55 → `402306`

### รายชื่อดาวน์ไลน์และรายละเอียด

| ID | Rule |
|---|---|
| MGMT-26 | **เส้นรายชื่อเส้นเดียว** `downlines/list` (รวมการค้นหา — lead E2): `parent_id` ไม่ส่ง / `0` = ตัวเอง · ต้องเป็นตัวเองหรือบัญชีฝั่ง agent ในสายล่าง (`402402`) · **`keyword` ไม่ส่ง / `""`** = ลูกตรงของ `parent_id` (agent + Member ปนกัน · ไล่ลงทีละชั้นได้ถึง Member) · **ส่ง `keyword`** = ค้น username บางส่วน (ไม่สนตัวพิมพ์) ในบัญชี**ทุกชั้นใต้ `parent_id`** ทั้งฝั่ง agent และ Member · ไม่รวม `parent_id` เอง ชั้นบน และสายข้างเคียง · ADMIN ไม่อยู่ในผล (AUTH-43) |
| MGMT-27 | `keyword` 4–32 ตัว (`422`) · ค้นเฉพาะ username · เรียง username A→Z · `page` / `limit` ใน body · `limit` ค่าเริ่มต้น 20 สูงสุด 100 (`utils.NewPage`) |
| MGMT-27B | ค้นหาบัญชีของ ADMIN — module `admin_management` ([admin_management.md](admin_management.md)) |
| MGMT-28 | แต่ละแถว: `id` · `role` · `user_type` · `username` · `name` · `phone_country_code` · `phone` · `status` (ACC-30) · `parent_username` (ผู้สร้างตรง · ทุกแถว) · `pt` · `balances` (account ACC-19) · `last_login_at` · `last_login_ip` · `created_at` (RFC 3339 · ยังไม่เคย login = `""`) · **แถว agent** มี `status_game` · **แถว Member** `pt.minigame` = `pt` · `remain_quota` · `commission_percent` + `created_*` / `updated_*` (lead Q-C2) |
| MGMT-29 | `agents/detail/get`: ทุก field ของบัญชี **ยกเว้น** password, passcode, hash, token · สกุลเงิน · `balances` · `pt` · `status_game` (ค่าที่ตั้งกับบัญชีเอง — ใช้เติมฟอร์มแก้) · **`status_game_effective`** (รวมผลหัวสาย — ใช้แสดงผล · lead E3-1) · login ล่าสุด · วันที่สร้าง · ระบุได้ทุกบัญชีฝั่ง agent ในสายล่าง (`402402`) |

### แก้บัญชี (`agents/detail/update` — lead E4)

| ID | Rule |
|---|---|
| MGMT-32 | 3 section **ไม่บังคับทุก section** ส่งเฉพาะที่แก้ (ไม่ส่ง = ไม่เปลี่ยน): `info` (`name` · `phone_country_code` + `phone` — ส่งบาง field ได้ · เบอร์ต้องส่งเป็นคู่) · `pt` (กลุ่มละ 5 ค่า — MGMT-23) · `status_game` (รหัสเกม → bool · อย่างน้อย 1 เกม) · ไม่ส่งเลยสัก section = `422` |
| MGMT-33 | แก้ได้เฉพาะ**ลูกตรง**ของตัวเอง (หลาน / ตัวเอง = `402304` · นอกสาย = `402402`) · **tx เดียว** — section ไหนผิด ไม่บันทึกเลยสักส่วน · ไม่ต้อง passcode · ลูกไม่มีแถวค่าตั้งของกลุ่มนั้น = ข้ามเงียบ |
| MGMT-34 | `account_change_logs` แยกแถวตาม section ที่เปลี่ยนจริง (`UPDATE_INFO` · `UPDATE_PT` · `UPDATE_GAMES`) · ค่าเหมือนเดิมทั้งหมด = ตอบ 200 ไม่เขียน log ของ section นั้น · ระบบปรับชั้นล่างเอง (R2 · CSM → Share Master) ยังทำตามปกติ |
| MGMT-35 | `agents/list` (แทน `copy-sources` — lead E6): ลูกตรงฝั่ง agent ของตัวเอง ไว้ทำ dropdown → เลือกแล้วเรียก `agents/detail/get` เอาค่ามาตั้งต้นฟอร์มสร้าง · `keyword` ไม่บังคับ 4–32 ตัว ค้น username · แต่ละแถว `id` · `username` · `name` · `user_type` · `status` · A→Z · ไม่มี pagination · สูงสุด 500 รายการ · ไม่มีลูก = `[]` · ตอนสร้างตรวจตามกฎปกติ |

### สถานะ

| ID | Rule |
|---|---|
| MGMT-30 | `agents/status/update`: ค่า `ACTIVE` / `SUSPENDED` / `LOCKED` · แก้ได้เฉพาะ**ผู้สร้างโดยตรง** · **แก้ของตัวเอง = `402304`** · เปลี่ยนกลับเป็น `ACTIVE` ได้ · ไม่ต้อง passcode · แยกเส้นจาก `detail/update` เพราะมีผลทั้งสายล่าง · ค่าที่แก้ = สถานะที่ตั้งกับบัญชีนั้นเอง · response แสดง `status` ที่ใช้งานจริง — ปลดลูกแล้ว ถ้าหัวสายที่สูงกว่ายังถูกระงับ ลูกยังเป็น `SUSPENDED` |
| MGMT-31 | ผลต่อสายล่างผ่าน `status` ที่ใช้งานจริง (ACC-30 · AUTH-53) ไม่แก้แถวของชั้นล่าง · `LOCKED` = เข้าใช้ไม่ได้ **ตั้งแต่ request ถัดไป** (AUTH-27 ลบ session) · `SUSPENDED` = เข้าได้เฉพาะ Profile และ Report ดูอย่างเดียว (AUTH-54) · Member ใต้บัญชีที่ไม่ ACTIVE เดิมพันไม่ได้ |

### บัญชีย่อย (sub)

| ID | Rule |
|---|---|
| MGMT-40 | สร้างได้: บัญชีหลักฝั่ง agent (ADMIN ไม่ได้ — AUTH-43) · sub สร้าง / แก้ / เปลี่ยนสถานะ / ดูรายชื่อ sub ไม่ได้ (`402311`) |
| MGMT-41 | username = `{username เจ้าของ}@{ชื่อ}` · ส่วนหลัง `@` 3–20 ตัว `a-z` `0-9` (รับตัวพิมพ์ใหญ่ เก็บตัวเล็ก) · รหัสผ่านตาม AUTH-36 · ชื่อเล่นตาม MGMT-07 · เบอร์ตาม MGMT-08 แต่**ซ้ำได้** |
| MGMT-42 | แก้ได้: ชื่อเล่น · เบอร์โทร · สิทธิ์ (`subaccounts/detail/update`) · สถานะ (`subaccounts/status/update`) · username แก้ไม่ได้ · รหัสผ่านใช้ระบบ reset ของ admin (AUTH-52) |
| MGMT-43 | สถานะของ sub: `ACTIVE` / `INACTIVE` · เจ้าของเป็นคนเปลี่ยน · `INACTIVE` = login ได้แต่ทำรายการไม่ได้ (เก็บเป็น `SUSPENDED` · API แสดง `INACTIVE`) · ไม่ต้อง passcode · เจ้าของหรือหัวสายถูกระงับ / ล็อก → sub ได้ผลตาม `status` ที่ใช้งานจริง |
| MGMT-44 | ไม่มีการลบ sub — เลิกใช้ให้เปลี่ยนเป็น `INACTIVE` |
| MGMT-45 | แก้ / เปลี่ยนสถานะได้เฉพาะเจ้าของ · ชั้นบนดูรายชื่อและรายละเอียด sub ของบัญชีในสายล่างได้อย่างเดียว (`402404`) |
| MGMT-46 | รายชื่อ sub: `owner_id` (ไม่ส่ง = ตัวเอง · ต้องเป็นตัวเองหรือสายล่าง) · ค้น username บางส่วน · A→Z · `page` / `limit` · แต่ละแถว `id` · `username` · `name` · `phone_country_code` · `phone` · `status` · `permissions` · `created_at` · `last_login_at` · `last_login_ip` · `subaccounts/detail/get` ส่ง field ชุดเดียวกัน |

### สิทธิ์ (ใช้ทั้งหลังบ้าน)

| ID | Rule |
|---|---|
| MGMT-50 | สิทธิ์ต่อเมนู `off` / `view` / `edit` (edit รวม view) · ไม่ส่ง = `off` · บัญชีหลักได้ `edit` ทุกเมนูที่ประเภทนั้นมี · sub ได้ตามที่เจ้าของให้ · รูปแบบ API `{ "{menu}": "off" \| "view" \| "edit" }` · **sub ต้องมีสิทธิ์ทุกเส้นของ BO** ยกเว้น Profile และเส้น auth ของบัญชีตัวเอง (`passcode/setup` · `passcode/change` · `password/change` · `logout`) — lead A5 |
| MGMT-51 | เช็คด้วย middleware บรรทัดเดียวกับ route (กฎข้อ 28) · **ทุกเส้นของ module นี้ใช้เมนู `member`** (lead P3–P5): ดู (`downlines/list` · `agents/detail/get` · `agents/list`) = `member` view · สร้าง / แก้ / สถานะ = `member` edit · ส่ง `balance` ตอนสร้าง ต้องมี `payment` edit เพิ่ม · เมนูอื่น: `dashboard` (off / view) · `report` (off / view) · `bet_cancel` · `payment` · `asset` · `announcement` · `rate` (เฉพาะ Superadmin) · `api_credential` (account 1.3 — `RequireSubPermission` เช็คเฉพาะ sub) · **ไม่มีเมนู `pt` แล้ว** (ลบ 2026-10-09 — lead P4) |
| MGMT-52 | เมนูที่แต่ละประเภทมี (สิทธิ์ของ sub ให้ได้เฉพาะเมนูเหล่านี้ · ส่งเมนูอื่น = `422`): **Superadmin** = 7 เมนู `dashboard` · `member` · `report` · `bet_cancel` · `payment` · `asset` · `rate` · **Company · Share · Agent** = 8 เมนู `dashboard` · `member` · `report` · `bet_cancel` · `payment` · `asset` · `announcement` · `api_credential` · Superadmin / ADMIN ไม่มี `api_credential` (ไม่มี Key — account ACC-01 · lead A10) · migration ล้าง key `pt` ออกจากสิทธิ์ของ sub เดิม |
| MGMT-53 | Profile (account ACC-12) ส่ง `permissions` รูปแบบเดียวกับ MGMT-50 · บัญชีหลักได้ `edit` ทุกเมนูของประเภทตัวเอง |

### ประวัติ

| ID | Rule |
|---|---|
| MGMT-60 | ทุกการสร้าง / แก้ข้อมูล / แก้สถานะ / แก้ PT / เปิดปิดเกม / ระบบปรับ Share Master ตาม CSM / สร้าง-แก้-เปลี่ยนสถานะ sub / ยอดเงินตั้งต้น เก็บ `account_change_logs` ใน tx เดียวกัน (ผู้ทำ, เป้าหมาย, action, ค่าเก่า → ใหม่, ip, request_id) · ยอดเงินตั้งต้นใช้ key `amounts` (lead E1-4) · ห้ามเก็บรหัสผ่าน |

## 4. สิ่งที่พบในโค้ดเก่า และการตัดสินใจ

ไม่มี — ไม่ได้ extract จากโค้ดเก่า

## 5. Endpoints

ใช้แค่ method `POST` · ระบุบัญชีด้วย `id` ใน body (ผู้เรียกมาจาก token) · ใต้ `/bo/pr` ผ่าน `Authenticated` + `PassedGates()` (รวม AUTH-54) ·
รูปแบบชื่อเส้น `…/detail/get` · `…/detail/update` · `…/status/update` (lead N1 — ใช้กับ sub ด้วย · Member ตาม member_management) ·
error ร่วม: `401202`, `401203`, `401301`, `401302`, `401304`, `401306`, `401307`, `401311`, `402303`

| Method | Path | สิทธิ์ (sub) | หน้าที่ |
|---|---|---|---|
| POST | `/api/v1/bo/pr/manage/agents/create` | `member` edit (+ `payment` edit ถ้าส่ง `balance`) | E1 สร้าง Company / Share / Agent |
| POST | `/api/v1/bo/pr/manage/downlines/list` | `member` view | E2 ลูกตรง หรือค้นทั้งสายด้วย `keyword` |
| POST | `/api/v1/bo/pr/manage/agents/detail/get` | `member` view | E3 รายละเอียดฝั่ง agent |
| POST | `/api/v1/bo/pr/manage/agents/detail/update` | `member` edit | E4 แก้ `info` / `pt` / `status_game` |
| POST | `/api/v1/bo/pr/manage/agents/status/update` | `member` edit | E5 ระงับ / ล็อก / ปลด |
| POST | `/api/v1/bo/pr/manage/agents/list` | `member` view | E6 ลูกตรงฝั่ง agent สำหรับ dropdown |
| POST | `/api/v1/bo/pr/manage/subaccounts/list` | บัญชีหลัก (`RequireMainAccount`) | รายชื่อ sub |
| POST | `/api/v1/bo/pr/manage/subaccounts/detail/get` | บัญชีหลัก | รายละเอียด sub |
| POST | `/api/v1/bo/pr/manage/subaccounts/create` | บัญชีหลัก | สร้าง sub |
| POST | `/api/v1/bo/pr/manage/subaccounts/detail/update` | บัญชีหลัก (เจ้าของ) | แก้ชื่อเล่น · เบอร์ · สิทธิ์ |
| POST | `/api/v1/bo/pr/manage/subaccounts/status/update` | บัญชีหลัก (เจ้าของ) | `ACTIVE` / `INACTIVE` |

เส้นที่ลบ (ไม่คง alias — ยังไม่มี prod): `downlines/search` · `agents/detail` · `agents/update-info` · `agents/update-pt` · `agents/update-games` ·
`agents/update-status` · `agents/copy-sources` · `agents/update-hold` · `subaccounts/detail` · `subaccounts/update-info` · `subaccounts/update-status`

ทุกตัวอย่างใช้ข้อมูลชุดเดียวกัน: `comp01` (id 10, Company Transfer) → `share01` (id 12, Share B2C · THB) → `agent01` (id 15) → `mem01` (Member id 501) · sub ของ comp01 = `comp01@staff` (id 30)

ทุก response รูปแบบ `{ "code", "msg", "data" }` · สำเร็จ `code = 200` · เส้นที่ไม่มีข้อมูลตอบกลับไม่มี key `data` · business error ตอบ HTTP 200 พร้อม `code`:
```json
{ "code": 402306, "msg": "pt.minigame.pt_from_parent ต่ำกว่าที่ลูกใช้อยู่ ตั้งได้ต่ำสุด 60" }
```

### E1 · POST /api/v1/bo/pr/manage/agents/create

Request:
```json
{
  "request_id": "6f1c2a1e-3b7d-4c55-9a40-0f7f3c2d8e11",
  "user_type": "SHARE_B2C",
  "username": "Share01",
  "password": "aA4b4c4d4e4f",
  "name": "share01",
  "phone_country_code": "66",
  "phone": "812345678",
  "currencies": ["THB"],
  "balance": { "THB": 10000 },
  "pt": {
    "minigame": { "pt_from_parent": 70, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true }
  },
  "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": false }
}
```
Response:
```json
{ "code": 200, "msg": "สำเร็จ", "data": { "id": 12, "username": "share01", "user_type": "SHARE_B2C" } }
```
- `request_id` บังคับ (UUID ใหม่ทุกครั้งที่กดสร้าง) · ส่งซ้ำด้วยค่าเดิม = ได้บัญชีเดิม · ของคนอื่น / เส้นอื่น = `422` (MGMT-15A)
- `user_type` ตาม MGMT-02 · ใต้ Company Seamless Reseller / Master ส่ง `SHARE_B2C` ระบบเก็บเป็น `SHARE_RESELLER` / `SHARE_MASTER` (response ตอบประเภทที่เก็บจริง)
- `currencies` ตาม MGMT-10–13 · `balance` ไม่บังคับ (MGMT-15A) · เบอร์ตาม MGMT-08 (ไม่กรอก = `""` ทั้งคู่)
- `pt` ครบทุกกลุ่ม ครบ 5 ค่า · `status_game` ไม่บังคับ (เกมที่ไม่ส่ง = เปิด)
- **CSM สร้าง Share Master:** `"pt": { "minigame": { "commission_percent": 0.3 } }` อย่างเดียว · ไม่ส่ง `status_game` (MGMT-19)

Error codes: `422`, `402301`, `402303`, `402305`, `402307`, `402308`, `402309`, `402310`, `402312`, `402401`, `402403`

### E2 · POST /api/v1/bo/pr/manage/downlines/list

Request (ทุกค่าไม่บังคับ):
```json
{ "parent_id": 12, "keyword": "", "page": 1, "limit": 20 }
```
Response (ลูกตรงของ share01 · หัวสาย comp01 ถูกระงับ):
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "current_page": 1, "total_page": 1, "total_count": 2, "limit": 20, "has_next": false, "has_prev": false,
    "data": [
      {
        "id": 15,
        "role": "AGENT",
        "user_type": "AGENT",
        "username": "agent01",
        "name": "agent01",
        "phone_country_code": "66",
        "phone": "898765432",
        "status": "SUSPENDED",
        "parent_username": "share01",
        "last_login_at": "2026-10-08T21:00:00+07:00",
        "last_login_ip": "203.0.113.10",
        "created_at": "2026-10-01T09:00:00+07:00",
        "pt": {
          "minigame": { "pt_from_parent": 60, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "share01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "share01" }
        },
        "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": true },
        "balances": [ { "currency": "THB", "amount": 5000 } ]
      },
      {
        "id": 501,
        "role": "MEMBER",
        "user_type": "MEMBER",
        "username": "mem01",
        "name": "ใจดี",
        "phone_country_code": "",
        "phone": "",
        "status": "SUSPENDED",
        "parent_username": "share01",
        "last_login_at": "",
        "last_login_ip": "",
        "created_at": "2026-10-01T09:00:00+07:00",
        "pt": {
          "minigame": { "pt": 30, "remain_quota": 40, "commission_percent": 0.3, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "share01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "share01" }
        },
        "balances": [ { "currency": "THB", "amount": 100 } ]
      }
    ]
  }
}
```
- `parent_id` ไม่ส่ง / `0` = ตัวเอง · ต้องเป็นตัวเองหรือบัญชีฝั่ง agent ในสายล่าง (`402402`)
- `keyword` ไม่ส่ง / `""` = ลูกตรงของ `parent_id` · ส่ง (4–32 ตัว) = ค้น username บางส่วน ทุกชั้นใต้ `parent_id` (MGMT-26 / 27)
- `status` = สถานะที่ใช้งานจริง (ตัวอย่างนี้หัวสายถูกระงับ ทุกแถวจึง `SUSPENDED`) · แถว agent มี `status_game` · แถว Member ไม่มี
- `balances` 1 รายการต่อสกุลของบัญชี · บัญชี Seamless = `0`

Error codes: `422`, `402303`, `402402`

### E3 · POST /api/v1/bo/pr/manage/agents/detail/get

Request: `{ "id": 12 }`

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
    "phone_country_code": "66",
    "phone": "812345678",
    "status": "ACTIVE",
    "parent_username": "comp01",
    "currencies": ["THB"],
    "balances": [ { "currency": "THB", "amount": 10000 } ],
    "pt": {
      "minigame": { "pt_from_parent": 70, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "comp01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "comp01@staff" }
    },
    "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": true },
    "status_game_effective": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": false },
    "passcode_set": true,
    "last_login_at": "2026-10-05T10:00:00+07:00",
    "last_login_ip": "203.0.113.10",
    "created_at": "2026-10-01T09:00:00+07:00"
  }
}
```
- `id` = บัญชีฝั่ง agent ในสายล่าง (ตัวเอง / สายบน / นอกสาย = `402402` · ของตัวเองดูที่ account Profile)
- `status_game` = ค่าที่ตั้งกับบัญชีเอง (ใช้เติมฟอร์มแก้) · `status_game_effective` = เปิดจริงหลังรวมหัวสาย (ตัวอย่าง: comp01 ปิด `scratch_card` จึงเป็น `false`)
- ยังไม่เคย login = `last_login_at` / `last_login_ip` เป็น `""`

Error codes: `422`, `402303`, `402402`

### E4 · POST /api/v1/bo/pr/manage/agents/detail/update

Request (section ไม่บังคับ ส่งเฉพาะที่แก้ · อย่างน้อย 1 section):
```json
{
  "id": 12,
  "info": { "name": "share01", "phone_country_code": "66", "phone": "812345678" },
  "pt": {
    "minigame": { "pt_from_parent": 60, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true }
  },
  "status_game": { "scratch_card": false }
}
```
Response: `{ "code": 200, "msg": "สำเร็จ" }`
- แก้ได้เฉพาะลูกตรง (MGMT-33) · tx เดียว — section ไหนผิดไม่บันทึกเลย · ไม่ต้อง passcode
- `info`: ส่งบาง field ได้ · เบอร์ต้องส่งเป็นคู่ · ล้างเบอร์ส่ง `""` ทั้งคู่
- `pt`: ส่งเฉพาะกลุ่มที่แก้ · กลุ่มละครบ 5 ค่า · ลดได้ไม่ต่ำกว่าที่ลูกใช้อยู่ (MGMT-24) · สำเร็จแล้วระบบคำนวณ `remain_quota` ของ Member ใต้ลูก (R2)
- `status_game`: เกมที่จะเปลี่ยน → bool · ไม่ส่งต่อลงสายล่าง (สายล่างเช็คทั้งสายตอนเล่น)
- **CSM แก้ Share Master:** `pt.minigame` ส่งได้แค่ `commission_percent` · ส่ง `status_game` = `422` · `info` ได้ปกติ (MGMT-19)
- **Superadmin แก้ CSM:** PT / `pt.status` / `status_game` ของ Share Master ทุกคนใต้ CSM ถูกปรับตามใน tx เดียว (MGMT-19)
- ค่าเหมือนเดิมทั้งหมด = 200 ไม่เขียน log ของ section นั้น (MGMT-34)

Error codes: `422`, `402303`, `402304`, `402305`, `402306`, `402307`, `402308`, `402309`, `402402`, `402403`

### E5 · POST /api/v1/bo/pr/manage/agents/status/update

Request: `{ "id": 12, "status": "SUSPENDED" }` · Response: `{ "code": 200, "msg": "สำเร็จ" }`
- `ACTIVE` / `SUSPENDED` / `LOCKED` · ผู้สร้างโดยตรงเท่านั้น · ตัวเอง / หลาน = `402304` · นอกสาย = `402402` · ไม่ต้อง passcode (MGMT-30)
- `LOCKED` มีผลตั้งแต่ request ถัดไปของบัญชีนั้นและสายล่าง · ปลดลูกแล้ว ถ้าหัวสายที่สูงกว่ายังถูกระงับ ลูกยังเป็น `SUSPENDED`

Error codes: `422`, `402303`, `402304`, `402402`

### E6 · POST /api/v1/bo/pr/manage/agents/list

Request: `{}` หรือ `{ "keyword": "shar" }`

Response:
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": [
    { "id": 12, "username": "share01", "name": "share01", "user_type": "SHARE_B2C", "status": "ACTIVE" },
    { "id": 13, "username": "share02", "name": "share02", "user_type": "SHARE_B2B", "status": "ACTIVE" }
  ]
}
```
- ลูกตรงฝั่ง agent ของบัญชีใน token · `keyword` ไม่บังคับ 4–32 ตัว ค้น username บางส่วน · A→Z · ไม่มี pagination · สูงสุด 500 · ไม่มีลูก = `[]` (MGMT-35)
- เลือกแล้วเรียก `agents/detail/get` เอา `pt` / `status_game` มาตั้งต้นฟอร์มสร้าง

Error codes: `422`, `402303`

### POST /api/v1/bo/pr/manage/subaccounts/list

Request (ทุกค่าไม่บังคับ): `{ "owner_id": 10, "q": "", "page": 1, "limit": 20 }`

Response:
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "current_page": 1, "total_page": 1, "total_count": 1, "limit": 20, "has_next": false, "has_prev": false,
    "data": [
      {
        "id": 30,
        "username": "comp01@staff",
        "name": "staff01",
        "phone_country_code": "",
        "phone": "",
        "status": "ACTIVE",
        "permissions": {
          "dashboard": "view", "member": "edit", "report": "view", "bet_cancel": "off",
          "payment": "off", "asset": "off", "announcement": "off", "api_credential": "view"
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
- `permissions` ครบทุกเมนูของประเภทเจ้าของ (MGMT-52)

Error codes: `402311`, `402402`

### POST /api/v1/bo/pr/manage/subaccounts/detail/get

Request: `{ "id": 30 }` · Response `data` = แถวเดียวกับ `subaccounts/list` · ดูได้: เจ้าของ และชั้นบนในสาย · อื่น = `402404`

Error codes: `422`, `402311`, `402404`

### POST /api/v1/bo/pr/manage/subaccounts/create

Request:
```json
{
  "name_suffix": "Staff",
  "password": "aA4b4c4d4e4f",
  "name": "staff01",
  "phone_country_code": "",
  "phone": "",
  "permissions": { "dashboard": "view", "member": "edit", "report": "view" }
}
```
Response: `{ "code": 200, "msg": "สำเร็จ", "data": { "id": 30, "username": "comp01@staff" } }`
- username = `{username เจ้าของ}@{name_suffix}` · เมนูที่ไม่ส่ง = `off` · `dashboard` / `report` สูงสุด `view` · เมนูที่ประเภทเจ้าของไม่มี (รวม `pt`) = `422` · เบอร์ซ้ำได้

Error codes: `422`, `402301`, `402311`, `402401`

### POST /api/v1/bo/pr/manage/subaccounts/detail/update

Request (แทนทั้งชุด — ต้องส่งครบ · ไม่ให้สิทธิ์ใดส่ง `"permissions": {}`):
```json
{ "id": 30, "name": "staff01", "phone_country_code": "66", "phone": "811111111", "permissions": { "member": "view", "report": "view" } }
```
Response: `{ "code": 200, "msg": "สำเร็จ" }` · แก้ได้เฉพาะเจ้าของ (ชั้นบน = `402404`)

Error codes: `422`, `402311`, `402404`

### POST /api/v1/bo/pr/manage/subaccounts/status/update

Request: `{ "id": 30, "status": "INACTIVE" }` · Response: `{ "code": 200, "msg": "สำเร็จ" }`
- `ACTIVE` / `INACTIVE` (เก็บเป็น `SUSPENDED`) · แก้ได้เฉพาะเจ้าของ (ชั้นบน = `402404`) · ไม่มีการลบ sub

Error codes: `422`, `402311`, `402404`

## 6. Schema

สถานะปัจจุบัน (หลัง migration ถึง `20261009140100`) + migration ที่ต้องเพิ่มใน PR 2 / 3

```sql
-- user_agents (ส่วนของ module นี้)
--   name VARCHAR(32) · phone VARCHAR(15) · agent_type VARCHAR(20) (CHECK ตาม role) · cnf JSONB NOT NULL DEFAULT '{"parent": []}' (GIN · MGMT-61)
--   uq_user_agents_phone (phone) · idx_user_agents_parent_username (parent_id, username)

CREATE TABLE agent_currencies (agent_id BIGINT REFERENCES user_agents(id), currency VARCHAR(4), PRIMARY KEY (agent_id, currency));

CREATE TABLE agent_game_settings (                -- ค่าหุ้นส่วน + เปิด/ปิด ต่อเกม (MGMT-16, MGMT-20)
    agent_id       BIGINT           NOT NULL REFERENCES user_agents(id),
    category       VARCHAR(30)      NOT NULL,
    game_code      VARCHAR(50)      NOT NULL,
    parent_id      BIGINT           REFERENCES user_agents(id),          -- MGMT-62
    pt_from_parent DOUBLE PRECISION NOT NULL CHECK (pt_from_parent BETWEEN 0 AND 100),
    force          DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (force BETWEEN 0 AND 100),
    remain         DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (remain BETWEEN 0 AND 100),
    commission     DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (commission BETWEEN 0 AND 1),
    status         BOOLEAN          NOT NULL DEFAULT true,               -- รับ PT ไหม (MGMT-20)
    status_game    BOOLEAN          NOT NULL DEFAULT true,               -- เปิด / ปิดทีละเกม
    created_by     VARCHAR(71)      NOT NULL DEFAULT '',                 -- คนกดจริง (MGMT-16)
    created_at     TIMESTAMPTZ      NOT NULL DEFAULT now(),
    updated_by     VARCHAR(71)      NOT NULL DEFAULT '',
    updated_at     TIMESTAMPTZ      NOT NULL DEFAULT now(),
    PRIMARY KEY (agent_id, game_code)
);
CREATE INDEX idx_agent_game_settings_parent ON agent_game_settings(parent_id, game_code);

-- subaccounts: name VARCHAR(32) · phone VARCHAR(15) · permissions JSONB NOT NULL DEFAULT '{}' (ไม่มี key = off) · INACTIVE เก็บเป็น 'SUSPENDED'

CREATE TABLE agent_balances (agent_id BIGINT, currency VARCHAR(4), amount DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (amount >= 0), updated_at TIMESTAMPTZ, PRIMARY KEY (agent_id, currency));
CREATE TABLE balance_ledger (                     -- ทุกการเปลี่ยนยอด (กฎข้อ 11) · amount / balance_after เป็น DOUBLE PRECISION
    id BIGSERIAL PRIMARY KEY, owner_type VARCHAR(10), owner_id BIGINT, currency VARCHAR(4),
    amount DOUBLE PRECISION, balance_after DOUBLE PRECISION, reason VARCHAR(30), ref_type VARCHAR(10), ref_id BIGINT,
    request_id VARCHAR(64), actor_type VARCHAR(10), actor_id BIGINT, created_at TIMESTAMPTZ,
    CONSTRAINT uq_balance_ledger_request UNIQUE (request_id, owner_type, owner_id, currency)
);
CREATE TABLE create_requests (request_id VARCHAR(64) PRIMARY KEY, creator_type VARCHAR(10), creator_id BIGINT, target_type VARCHAR(10), target_id BIGINT, created_at TIMESTAMPTZ);
CREATE TABLE account_change_logs (                -- MGMT-60 · action: CREATE / UPDATE_INFO / UPDATE_STATUS / UPDATE_PT / ADJUST_PT / UPDATE_GAMES / SYNC_FROM_CSM / SUB_STATUS / INITIAL_BALANCE
    id BIGSERIAL PRIMARY KEY, actor_type VARCHAR(10), actor_id BIGINT, actor_username VARCHAR(71), target_type VARCHAR(10), target_id BIGINT,
    target_username VARCHAR(71), action VARCHAR(30), old_value JSONB, new_value JSONB, ip VARCHAR(45), request_id VARCHAR(64), created_at TIMESTAMPTZ
);

-- migration ใหม่ PR 2: ล้าง key "pt" ออกจาก subaccounts.permissions (MGMT-52)
--   UPDATE subaccounts SET permissions = permissions - 'pt' WHERE permissions ? 'pt';
-- migration ใหม่ PR 3: เบอร์ (MGMT-08 / 08A) — user_agents และ subaccounts (user_members ทำใน member_management)
--   ADD COLUMN phone_country_code VARCHAR(3) · แปลงเบอร์เดิมตาม MGMT-08A · CHECK phone ~ '^[1-9][0-9]*$' และความยาวรวม ≤ 15
--   user_agents: unique (phone_country_code, phone) แทน uq_user_agents_phone · subaccounts ไม่ unique
--   เบอร์ที่ถูกล้าง → migration_phone_cleared(id, table_name, row_id, old_phone, reason FORMAT / DUPLICATE, cleared_at)
-- migration ใหม่ PR 3: Share Master ที่มีอยู่ปรับตาม CSM ครั้งเดียว (MGMT-19) — pt_from_parent · force / remain = 0 · status · status_game · commission ไม่แตะ
--   + remain ของ Member ใต้ Share Master คำนวณใหม่ (R2) · down ไม่คืนค่า
```

- Superadmin มีแถว `agent_game_settings` ของตัวเอง (`pt_from_parent = 100`) · ADMIN ไม่มี · ผู้สร้างถือในสายของลูก = `pt_from_parent` ของผู้สร้าง − ของลูก (คำนวณตอนคิดเงิน ไม่เก็บ)
- รายการสกุล · กลุ่ม PT · รหัสเกม · เมนูสิทธิ์ · รหัสโทรออกของประเทศ เป็นค่าคงที่ใน `app/core`

## 7. Test cases

| Rule | Input | Expected |
|---|---|---|
| MGMT-02 | Company Transfer สร้าง `SHARE_B2B` / `SHARE_B2C` · Company Seamless Reseller สร้าง `SHARE_B2C` | สำเร็จ · `user_type = SHARE_RESELLER` |
| MGMT-02 | Share Reseller / Share Master สร้าง Agent · Company Seamless 1 to 1 สร้าง Share · Share B2B สร้าง Member | `402301` |
| MGMT-02 | Share Reseller / Share Master / Share B2C / Agent สร้าง Member · Agent สร้าง Agent 3 ชั้น | สำเร็จ |
| MGMT-04 | สร้าง Company · Share · Agent | ทุกบัญชีมีแถว `api_credentials` |
| MGMT-05 | username `ab` / 33 ตัว / มี `_` · ซ้ำ · ชื่อเดียวกับ Member | `422` · `402401` · `402401` |
| MGMT-07 | ชื่อ `สมชาย01` · `สม` · `สมชาย ใจดี` | สำเร็จ · `422` · `422` |
| MGMT-08 | `"66"` + `"812345678"` · ส่งค่าเดียว · `phone` ขึ้นต้น 0 · code `999` (ไม่มีจริง) · รวมเกิน 15 หลัก · คู่ซ้ำ | สำเร็จ · `422` · `422` · `422` · `422` · `402403` |
| MGMT-08A | migration: `0812345678` · `66812345678` · `12345` · `+6681…` | `66`/`812345678` · `66`/`812345678` · `""` (log) · `""` (log) |
| MGMT-41 | sub 2 บัญชีเบอร์เดียวกัน | สำเร็จทั้งคู่ |
| MGMT-12 | Share B2B ไม่ส่งสกุล · Share B2C ส่ง 2 สกุล | `422` |
| MGMT-14 | Agent ใต้ Share B2B (THB, USD) เลือก JPY | `402310` |
| MGMT-15A | Company THB 50,000 สร้าง Share พร้อม THB 10,000 · ยอดไม่พอ · Superadmin ให้ 1,000,000 | Company 40,000 · Share 10,000 · ledger 2 แถว · `402312` · สำเร็จ ไม่มีแถวยอด Superadmin |
| MGMT-15A | `request_id` เดิมของตัวเอง · ของคนอื่น · ที่เคยใช้สร้าง Member | ได้ `id` เดิม ไม่โอนซ้ำ · `422` · `422` |
| MGMT-15A | สร้างพร้อมกัน 2 คำขอ ยอดพอแค่คำขอเดียว | สำเร็จ 1 · อีกอัน `402312` |
| MGMT-15A | sub มี `member` edit ไม่มี `payment` ส่ง `balance` | `402303` |
| MGMT-17 / 18 | `"force": null` · `pt_from_parent` `30.25` · commission `1.01` · `30.12345` · ให้ลูกเกินที่ได้รับ | `422` · `422` · `422` · `422` · `402305` ("ค่าที่ให้ลูกเกินกว่าที่ได้รับ") |
| MGMT-19 | CSM สร้าง Share Master ส่งแค่ `commission_percent` | สำเร็จ · `pt_from_parent` = ค่าที่ CSM ได้รับ · force / remain 0 · `status_game` = ของ CSM |
| MGMT-19 | CSM สร้าง Share Master ส่ง `pt_from_parent` · ส่ง `status_game` | `422` "Share Master ใช้ค่าตาม Company Seamless Master" |
| MGMT-19 | CSM `detail/update` Share Master: `commission_percent` · `pt_from_parent` · `status_game` · `info` | สำเร็จ · `422` · `422` · สำเร็จ |
| MGMT-19 | Superadmin ลด PT ของ CSM จาก 80 เหลือ 70 · Member ของ sm1 `pt` 75 | `402306` ต่ำสุด 75 · ไม่มีแถวไหนเปลี่ยน |
| MGMT-19 | Superadmin เพิ่ม PT ของ CSM 80 → 85 · ปิด `scratch_card` ของ CSM | sm1 / sm2 `pt_from_parent = 85` · `remain_quota` ของ Member ใต้ sm คำนวณใหม่ · `status_game.scratch_card = false` · log `SYNC_FROM_CSM` ต่อ Share Master |
| MGMT-19A | Company Seamless Reseller ให้ Share Reseller 2 คนต่างกัน | สำเร็จ |
| MGMT-24 | comA ลดค่าที่ให้ share1 เหลือ 55 (share1 ให้ agent1 60 · Member ของ share1 `pt` 30) | `402306` ต่ำสุด 60 |
| MGMT-24 R1 / R2 | agent01 ได้รับ 60 · Member `pt` 50 / 20 · ลดเหลือ 45 · ตั้ง 55 · ตั้ง 70 | `402306` ต่ำสุด 50 · `remain_quota` 5 / 35 · 20 / 50 |
| MGMT-25 | ให้ลูก 60 · `force` 70 | `402308` |
| MGMT-16 | sub `comp01@staff` สร้างบัญชี · แก้ PT | `created_by = updated_by = "comp01@staff"` · แก้แล้ว `updated_by` เปลี่ยน `created_by` คงเดิม |
| MGMT-20 | `pt.minigame.status = false` | บันทึก · `status_game` ไม่เปลี่ยน |
| MGMT-20 | comp01 ปิด `scratch_card` ของ share01 · ดู detail ของ agent01 (ลูก share01) | agent01 `status_game.scratch_card = true` · `status_game_effective.scratch_card = false` |
| MGMT-26 | ไม่ส่งอะไร · `parent_id` = agent01 · `parent_id` สายอื่น | ลูกตรงของตัวเอง · ลูกตรงของ agent01 (Agent + Member ปน A→Z) · `402402` |
| MGMT-26 / 27 | comp01 `keyword: "shar"` (share01 ลูกตรง · share9 อยู่ใต้ agent01) · `keyword` 3 ตัว / 33 ตัว | ได้ทั้งสองพร้อม `parent_username` ไม่เห็นตัวเอง / ชั้นบน / สายข้าง · `422` |
| MGMT-28 | แถว agent · แถว Member | agent มี `status_game` · Member มี `pt.minigame.pt` · `remain_quota` · `commission_percent` · ทุกแถวมี `parent_username` |
| MGMT-29 | detail ของบัญชีในสายล่าง · ของตัวเอง · ไม่มี field รหัสผ่าน | สำเร็จ · `402402` |
| MGMT-32 | `detail/update` ส่ง `{ "id": 12 }` อย่างเดียว | `422` |
| MGMT-33 | `detail/update` หลาน · ตัวเอง · นอกสาย | `402304` · `402304` · `402402` |
| MGMT-33 | ส่ง `info` ถูก + `pt` ผิด | `422` / `402xxx` · ชื่อไม่เปลี่ยน |
| MGMT-34 | ส่ง `info` + `status_game` ค่าเดิมทั้งหมด · ส่ง `info` เปลี่ยนชื่อ + `pt` เปลี่ยน | 200 ไม่มี log · log 2 แถว `UPDATE_INFO` + `UPDATE_PT` |
| MGMT-35 | `agents/list` · `keyword` 3 ตัว · ไม่มีลูก | ลูกตรงฝั่ง agent A→Z (ไม่มี Member) · `422` · `[]` |
| MGMT-30 | แก้สถานะตัวเอง · หลาน · LOCKED → ACTIVE | `402304` · `402304` · สำเร็จ |
| MGMT-31 | Company ระงับ Share · Share ล็อก Agent | Share และสายล่าง `SUSPENDED` Company ปกติ · Agent request ถัดไป `401301` |
| MGMT-43 / 45 | sub `INACTIVE` · ชั้นบนแก้ sub ของสายล่าง | login ได้ เรียกเส้นอื่น `401311` · `402404` |
| MGMT-40 | sub เรียก `subaccounts/create` / `list` | `402311` |
| MGMT-51 | sub มี `member` view เรียก `downlines/list` · `detail/get` · `agents/list` · `detail/update` | สำเร็จ · สำเร็จ · สำเร็จ · `402303` |
| MGMT-51 | sub มี `member` edit สร้าง Agent (ไม่ส่ง `balance`) | สำเร็จ (ไม่ต้องมี `pt` แล้ว) |
| MGMT-52 | sub ส่งสิทธิ์ `pt` · sub ของ Superadmin ได้ `announcement` / `api_credential` · Company ให้ `rate` | `422` |
| MGMT-52 | migration: sub เดิมที่มี `"pt": "edit"` | key `pt` หายจาก `permissions` |
| MGMT-60 | สร้างพร้อม `balance` | log `INITIAL_BALANCE` key `amounts` |
| — | เรียกเส้นที่ลบ (`downlines/search` · `agents/update-pt` · `agents/copy-sources` ฯลฯ) | `404` |

### 7.1 ลำดับเช็ค

ไล่ **บนลงล่างตามลำดับ field ใน body** 2 รอบ · ตอบ error ของ field แรกที่ผิด:

1. **รูปแบบ** (DTO — ไม่ต้องดู DB) ทุก field บนลงล่าง → `422`
2. **กฎที่ต้องดู DB** บนลงล่าง — เส้นสร้าง: `user_type` (`402301`) → `username` ซ้ำ (`402401`) → เบอร์ซ้ำ (`402403`) → `currencies` (`422` / `402310`) →
   `balance` (`422` / `402312`) → `pt` ทีละ field: `pt_from_parent` (`402305` / `402307`) → `force` → `remain_quota` (`402308`) → `commission_percent` (`402309`)
   · `detail/update`: `info` (เบอร์ซ้ำ `402403`) → `pt` (`pt_from_parent` เพดาน `402305` → ต่ำสุดที่ลูกใช้ `402306` → `force` → `remain_quota` → `commission_percent`) → `status_game`
3. สิทธิ์ (`401308` / `402303` / `402311`) เช็คที่ middleware ก่อนทั้งหมด · **ข้อยกเว้น:** สิทธิ์ `payment` ตอนส่ง `balance` เช็คใน service หลังตรวจรูปแบบ (lead U3)
4. Share Master (MGMT-19): ส่ง field ที่ระบบตั้งเอง = `422` ในรอบรูปแบบไม่ได้ (ต้องรู้ว่าลูกเป็น Share Master ไหม) → เช็คเป็น**ข้อแรกของรอบ DB** (สร้าง: ต่อจากโหลดผู้สร้าง · แก้: ก่อนแก้ section ใดๆ) · รวมถึง `pt.{กลุ่ม}.{field} ต้องส่ง` ของบัญชีอื่น (รูปแบบตัวเลขยังเช็คในรอบรูปแบบ)

ค่าที่เป็น object (`pt`, `balance`, `status_game`, `permissions`) ไล่ key ตามลำดับตัวอักษร

**ข้อความ error บอก field และค่าที่ตั้งได้**: `402301` → `user_type: ส่ง SHARE_B2C ไม่ได้ · SHARE_B2C สร้างได้โดย COMPANY_TRANSFER, … เท่านั้น แต่คุณเป็น SUPERADMIN · SUPERADMIN สร้างได้เฉพาะ …` ·
`422` currencies → `currencies: COMPANY_TRANSFER ได้ครบทุกสกุล ห้ามส่ง currencies` · `402305` → `pt.minigame.pt_from_parent ตั้งได้ไม่เกิน 90 (ค่าที่คุณได้รับ)` ·
`402308` → `pt.minigame.force ตั้งได้ไม่เกิน 50 (ค่าที่ให้ลูก)` · `402312` → `balance.THB ยอดของคุณไม่พอ (มี 0)` · สิทธิ์ sub → `permissions.rate ไม่มีเมนูนี้ · เมนูที่ให้ได้: …`

## 8. Contract changes (แจ้ง frontend)

**เปลี่ยน 2026-10-09 (lead review):**
- path ใหม่ทั้งหมดตามหัวข้อ 5 · เส้นเดิมถูกลบ (ได้ `404`) · `downlines/search` รวมเข้า `downlines/list` ด้วย `keyword` (แทน `q`)
- แก้บัญชีเหลือเส้นเดียว `agents/detail/update` (`info` / `pt` / `status_game`) · สถานะแยกที่ `agents/status/update` · dropdown ใช้ `agents/list` แทน `copy-sources`
- เบอร์โทร 2 field `phone_country_code` + `phone` ทุก request / response (agent · sub · Member) · เบอร์เดิมแปลงให้แล้ว บางเบอร์ถูกล้างเป็น `""`
- `permissions` ไม่มีเมนู `pt` แล้ว · มี `api_credential` (Company / Share / Agent) · sub ใช้แค่ `member` view / edit กับ module นี้
- `pt` แสดงเสมอ · แถวรายชื่อมี `parent_username` ทุกแถว · แถว agent มี `status_game` · detail มี `status_game_effective`
- Share Reseller / Share Master สร้าง Agent ไม่ได้ · ฟอร์มสร้าง / แก้ Share Master ส่งแค่ `commission_percent`
- ข้อความ `402305` = "ค่าที่ให้ลูกเกินกว่าที่ได้รับ" · `402307` = "Share Master ใช้ค่า PT ตาม Company Seamless Master"

**คงเดิม:** ค่า % / เงินเป็น JSON number ทศนิยมไม่เกิน 4 ตำแหน่ง · ไม่มี `null` ใน API (ACC-32) · เส้นสร้างรับ `request_id` (บังคับ) และ `balance` (ไม่บังคับ)

## 9. Error codes (`bb=02`) — business error ตอบ HTTP 200

| Code | ความหมาย |
|---|---|
| 402301 | สร้างบัญชีประเภทนี้ไม่ได้ |
| 402303 | ไม่มีสิทธิ์ใช้งานเมนูนี้ |
| 402304 | แก้ไขได้เฉพาะผู้สร้างของบัญชีนี้ |
| 402305 | ค่าที่ให้ลูกเกินกว่าที่ได้รับ (lead E1-2) |
| 402306 | ค่าที่ให้ลูกต่ำกว่าที่ลูกใช้อยู่ (msg บอกค่าต่ำสุดที่ตั้งได้) |
| 402307 | Share Master ใช้ค่า PT ตาม Company Seamless Master (lead E1-2) |
| 402308 | Force / Remain เกินที่กำหนด |
| 402309 | Commission เกินที่กำหนด |
| 402310 | สกุลเงินไม่อยู่ในสกุลของผู้สร้าง |
| 402311 | บัญชีย่อยทำรายการนี้ไม่ได้ |
| 402312 | ยอดเงินของคุณไม่พอสำหรับยอดเงินตั้งต้น (MGMT-15A) |
| 402401 | username นี้ถูกใช้แล้ว |
| 402402 | ไม่พบบัญชีในสายของคุณ |
| 402403 | เบอร์โทรนี้ถูกใช้แล้ว |
| 402404 | ไม่พบบัญชีย่อยของคุณ |

## 10. Open items

| # | เรื่อง | ตัดสินเมื่อ |
|---|---|---|
| O1 | PT ของบัญชีที่ `pt.status = false` ไปอยู่กับใคร (lead Q-R3) | ก่อนเริ่ม module Bet / Settle |
