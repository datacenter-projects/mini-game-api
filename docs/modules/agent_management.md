# การจัดการสมาชิก (`agent_management`) — Spec

- สถานะ: **DRAFT** (เขียนใหม่ทั้งฉบับ 2026-10-06 ตาม review ของ lead + กฎที่เก็บจากภาพหน้าจอ)
- อนุมัติโดย: — · วันที่: —
- ชื่อ module ในโค้ด: `agent_management` (`controllers/agent_management`, `dto/agent_management`, `service/agent_management`, `core/agent_management`)
- เมนู: 2 การจัดการสมาชิก — เพิ่มบัญชี · รายชื่อดาวน์ไลน์ (ไล่ลงได้ถึง Member) · แก้ไข · บัญชีย่อย (เพิ่ม · รายชื่อ · แก้ · ลบ)
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

**ไม่อยู่ใน module นี้:** ยอดเงิน / ยอดเงินตั้งต้น / โอนเงิน (module การชำระเงิน — `balances` เพิ่มเข้า list และ detail ตอนทำ module นั้น) ·
การคิด PT / Force / Remain / Commission ตอน settle (module เดิมพัน) · login ของ Member (module หน้าบ้าน) ·
Key / ลิงก์ตอบกลับ / IP (account 1.3 — module นี้แค่สร้าง Key ตอนสร้างเจ้าของ ACC-05)

## 2. คำศัพท์

| คำ | ความหมาย |
|---|---|
| ฝั่ง agent | Superadmin · Company · Share · Agent (ตาราง `user_agents`) |
| Member | ผู้เล่น (ตาราง `members` แยกจากฝั่ง agent) |
| ผู้สร้าง | บัญชีที่สร้างบัญชีนั้น (`parent_id` / `agent_id` ของ member) — ลูกตรง = บัญชีที่ตัวเองสร้าง |
| สายล่าง | ทุกบัญชีใต้ตัวเองทุกชั้น ไม่นับตัวเอง |
| `user_type` | ประเภทบัญชี (MGMT-01) — ตรงกับ account ACC-12 |
| กลุ่ม PT | ชุดค่าหุ้นส่วนที่รับตอนสร้าง / แก้ แยกตามระบบที่มีเรื่อง PT (`game` = ทุกเกมในหมวด `minigame` · เพิ่มกลุ่มได้โดยไม่เปลี่ยนโครงสร้าง) · ระบบเก็บต่อเกม |
| ได้รับ / ถือ / ปล่อย | ได้รับ = PT ที่ผู้สร้างปล่อยให้ · ถือ = ส่วนที่เก็บเอง · ปล่อย = ได้รับ − ถือ (ให้ลูกตรง) |
| bp | % × 100 เป็นจำนวนเต็ม เช่น 95.50% = `9550` — ใช้ภายใน DB / Go เท่านั้น · API ส่ง % เป็น JSON number (ACC-18) |
| `status` | สถานะที่ใช้งานจริง key เดียว (account ACC-30) = เข้มที่สุดของสถานะที่ตั้งกับบัญชีเอง, ผู้สร้าง (sub) และหัวสายทุกชั้น (AUTH-53) · DB เก็บสถานะที่ตั้งกับบัญชีเองแยกไว้ |

## 3. Business rules

### ประเภทและผู้สร้าง

| ID | Rule |
|---|---|
| MGMT-01 | `user_type`: `SUPERADMIN` · `ADMIN` · `COMPANY_TRANSFER` · `COMPANY_SEAMLESS_RESELLER` · `COMPANY_SEAMLESS_MASTER` · `COMPANY_SEAMLESS_1TO1` · `SHARE_B2B` · `SHARE_B2C` · `SHARE_RESELLER` · `SHARE_MASTER` · `AGENT` · `MEMBER` · เก็บเป็น `role` + `agent_type` ใน `user_agents` (Member อยู่ตาราง `members`) |
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
| MGMT-04 | สร้าง Company Seamless 1 to 1 · Share Reseller · Share Master → สร้าง Key ของ account 1.3 ใน tx เดียวกัน (ACC-05) |

### ข้อมูลบัญชี

| ID | Rule |
|---|---|
| MGMT-05 | username (ฝั่ง agent และ Member): 3–32 ตัว `a-z` `0-9` (รับตัวพิมพ์ใหญ่แล้วแปลงเป็นเล็ก — AUTH-01) · **ห้ามซ้ำทั้งระบบ** ข้ามตาราง `user_agents` และ `members` (เช็คใน service ภายใน tx พร้อม `pg_advisory_xact_lock` ของ username กันสร้างพร้อมกัน) · แก้ไม่ได้ (AUTH-19) |
| MGMT-06 | รหัสผ่านตาม AUTH-36 · ฝั่ง agent ตั้ง passcode ตอน login ครั้งแรก (AUTH-29) |
| MGMT-07 | ชื่อ: บังคับ · 3–32 ตัว `A-Z` `a-z` `0-9` |
| MGMT-08 | เบอร์โทร: ไม่บังคับ · 8–15 ตัว ตัวเลขเท่านั้น (ห้าม `+`) · ห้ามซ้ำภายในตารางเดียวกัน (unique index ต่อตาราง) |
| MGMT-09 | แก้ได้: ชื่อ · เบอร์โทร · สถานะ (MGMT-30) · ค่าหุ้นส่วนและเปิด / ปิดเกม (MGMT-20) · แก้ไม่ได้: username · ประเภท · สกุลเงิน · ไม่มีการลบบัญชีฝั่ง agent และ Member |
| MGMT-09A | แก้ไข Member ได้: ชื่อ · เบอร์โทร · สถานะ · Commission (แต่ละอย่างใช้เส้นของตัวเอง — หัวข้อ 5) |

### สกุลเงิน

ระบบมี 27 สกุล: ARS AUD BDT BOB BRL CLP CNY EUR GBP HKD IDR INR JPY KHR KRW LAK MMK MXN MYR NGN PHP PKR THB TWD USD USDT VND ·
ทุกสกุลทศนิยม 2 ตำแหน่ง (ACC-18)

| ID | Rule |
|---|---|
| MGMT-10 | Company Transfer · Company Seamless Reseller / Master: มีครบทุกสกุล (ไม่ต้องส่ง) |
| MGMT-11 | Company Seamless 1 to 1: 1 สกุล ที่ Superadmin เลือก |
| MGMT-12 | Share B2B: เลือกได้หลายสกุล อย่างน้อย 1 · Share B2C / Reseller / Master: 1 สกุล |
| MGMT-13 | Agent ที่ Share B2B สร้าง: 1 สกุล · Agent ที่ Share B2C / Reseller / Master หรือ Agent สร้าง และ Member: ใช้สกุลของผู้สร้าง (ไม่ต้องส่ง) |
| MGMT-14 | สกุลที่เลือกต้องอยู่ในสกุลของผู้สร้าง · เปลี่ยนหลังสร้างไม่ได้ |
| MGMT-15 | บัญชีฝั่ง Seamless ทุกชั้น (Company Seamless ทุกประเภท และสายล่าง) ไม่มียอดเงิน · ยอดเงินตั้งต้นของบัญชี Transfer ทำใน module การชำระเงิน |

### ค่าหุ้นส่วน (PT · Force · Remain · Commission)

| ID | Rule |
|---|---|
| MGMT-16 | **รับค่าชุดเดียวต่อกลุ่ม · เก็บและแสดงต่อเกม**: |
| | • สร้าง / แก้: ส่งเป็น object `pt` แยกตาม**กลุ่ม PT** · แต่ละกลุ่มมี 4 ค่า: `pt` (ถือ) · `force` · `remain_quota` · `commission_percent` · ระบบ**กระจายค่าชุดนั้นลงทุกเกมในกลุ่ม** แล้วเก็บทีละเกม (`agent_game_settings`) |
| | • ตอนนี้มีกลุ่ม `game` กลุ่มเดียว = ทุกเกมในหมวด `minigame` (`coin_toss`, `rock_paper_scissors`, `scratch_card`) · `provider` ฯลฯ เพิ่มเมื่อมีระบบนั้น · การจับคู่กลุ่ม → หมวด → เกม กำหนดใน `app/core` เพิ่มได้โดยไม่เปลี่ยน API / ตาราง |
| | • แสดง: ทุกเส้นส่งเป็น object `pt` ชุดเดียวต่อกลุ่ม (ค่าของเกมในกลุ่มเท่ากันเสมอ) · รายละเอียดบัญชี (MGMT-29) และ Profile (account ACC-16) ส่ง `pt_from_parent` ต่อกลุ่ม และ `status_game` ต่อเกม เพิ่ม |
| | • เกมใหม่ที่เพิ่มเข้าหมวดภายหลัง: ทุกบัญชีได้ค่าของกลุ่มนั้น และ `status_game = true` อัตโนมัติ |
| MGMT-17 | ค่า % ใน API เป็น JSON number ทศนิยมไม่เกิน 2 ตำแหน่ง (เช่น `30`, `0.5`) · ภายใน DB / Go เป็นจำนวนเต็ม bp (ACC-18) · **ห้าม `null`** · ไม่ตั้ง Force / Remain = `0` |
| MGMT-18 | ถือ: 0 ถึงค่าที่ได้รับ · ทีละ 0.5% · Commission: 0–1% ทีละ 0.1% · Force / Remain: 0 ถึงค่าที่ได้รับ ทีละ 0.5% · Commission ของลูกตั้ง**เกินของผู้สร้างได้** (เช่น ผู้สร้างได้ 0.5% ตั้งให้ลูก 0.6% ได้) แต่ไม่เกิน 1% |
| MGMT-19 | Company Seamless Master: ถือ 0 · ปล่อยทั้งหมด · Force และ Remain = 0 · แก้ได้แค่ Commission |
| MGMT-20 | เปิด / ปิดเกมรายบัญชี: `status_game` ต่อเกม (bool) · ผู้สร้างตั้งให้ · ปิดแล้ว Member ในสายเล่นเกมนั้นไม่ได้ (บังคับใน module เดิมพัน) · ชั้นบนปิดแล้ว**สายล่างเปิดเองไม่ได้** — เกมใช้ได้จริงเมื่อบัญชีตัวเองและหัวสายทุกชั้นเปิดอยู่ (แบบเดียวกับ `status` ที่ใช้งานจริง — ACC-30) |
| MGMT-21 | สร้างบัญชีฝั่ง agent ต้องส่งครบทุกกลุ่มที่มี · Member มีแค่ `commission_percent` ต่อกลุ่ม (Member ไม่ถือ PT — เอกสาร PT Commission) กระจายและเก็บต่อเกมแบบเดียวกัน (`member_game_settings`) |
| MGMT-22 | ได้รับของลูก = ค่าปล่อยของผู้สร้าง (ลูกตรงทุกคนได้รับเท่ากัน) · Superadmin ได้รับ 100% · Superadmin ถือได้ มีค่าตั้งของตัวเองชุดเดียว แก้ผ่านเส้นแก้ PT ของตัวเอง · ทุก Company ได้รับ = ค่าปล่อยของ Superadmin · Agent ที่มี Agent ลูกปล่อยลงได้เหมือนชั้นอื่น |
| MGMT-23 | แก้ได้เฉพาะ**ผู้สร้างโดยตรง** (sub ตามสิทธิ์ — MGMT-50) · แก้เฉพาะกลุ่มที่ส่ง · ในกลุ่มที่ส่งต้องครบ 4 ค่า · มีผลกับ bet ใหม่ (bet เก็บค่า ณ ตอน bet — module เดิมพัน) |
| MGMT-24 | **ค่าปล่อยของบัญชี X เปลี่ยน** (ผู้สร้างของ X แก้ค่าถือของ X — ต่อกลุ่ม): |
| | • ปล่อยลด: ลูกตรงทุกคนของ X ลดค่าถือลงเท่ากัน · ค่าปล่อยของลูกและชั้นล่างไม่เปลี่ยน · ค่าปล่อยใหม่ของ X ต้อง ≥ ค่าปล่อยที่มากที่สุดของลูกตรง ไม่งั้นปฏิเสธทั้งคำสั่ง (`402306` msg บอกค่าปล่อยต่ำสุดที่ตั้งได้) |
| | • ปล่อยเพิ่ม: ลูกตรงทุกคนของ X ถือเพิ่มเท่ากัน · ค่าปล่อยของลูกไม่เปลี่ยน |
| | • ปรับค่าของลูกใน tx เดียวกัน · lock แถว X และลูกตรงเรียงตาม id · ปรับทุกเกมในกลุ่มพร้อมกัน · ทุกบัญชีที่เปลี่ยนเก็บประวัติ (MGMT-60) |
| MGMT-25 | หลัง MGMT-24 ถ้า Force / Remain ของลูกเกินค่าที่ได้รับใหม่ → ลดลงให้เท่าค่าที่ได้รับใหม่อัตโนมัติใน tx เดียวกัน และเก็บประวัติ (MGMT-60) |

ตัวอย่าง MGMT-24 (กลุ่ม `game`): Superadmin แก้ Company จากถือ 10 ปล่อย 80 เป็นถือ 14 ปล่อย 76

| ชั้น | ก่อน (ได้รับ / ถือ / ปล่อย) | หลัง |
|---|---|---|
| Company | 90 / 10 / 80 | 90 / 14 / 76 |
| share1 | 80 / 30 / 50 | 76 / 26 / 50 |
| share2 | 80 / 5 / 75 | 76 / 1 / 75 |
| Agent ใต้ share1 / share2 | 50 / 50 / 0 · 75 / 75 / 0 | ไม่เปลี่ยน |

ค่าปล่อยต่ำสุดที่ Company ตั้งได้ = 75 (ค่าปล่อยของ share2)

### รายชื่อดาวน์ไลน์และรายละเอียด

| ID | Rule |
|---|---|
| MGMT-26 | เส้นรายชื่อเส้นเดียว: แสดง**ลูกตรง**ของบัญชีที่ระบุ (ไม่ระบุ = ตัวเอง) ทั้งฝั่ง agent และ Member ปนกัน · ไล่ลงทีละชั้นได้ถึง Member · ระบุได้เฉพาะตัวเองหรือบัญชีในสายล่าง (`402402`) · ADMIN ไม่อยู่ในรายการใด (AUTH-43) |
| MGMT-27 | ตัวกรองมีแค่ `page` / `limit` และค้นหา username บางส่วน (ไม่สนตัวพิมพ์) · เรียง username A→Z · `limit` ค่าเริ่มต้น 20 สูงสุด 100 (`utils.ParsePage`) |
| MGMT-28 | แต่ละแถว: `id` · `role` (บอกว่าเป็นฝั่ง agent หรือ Member) · `user_type` · username · ชื่อ · เบอร์โทร · `status` (ACC-30) · `pt` (ตาม MGMT-51) · `balances` (ยอดแยกสกุลตาม account ACC-19 — ระหว่างยังไม่มี module การชำระเงินส่ง `0`) |
| MGMT-29 | เส้นรายละเอียดแยก agent / Member: ทุก field ของบัญชี **ยกเว้น** password, passcode, hash และ token · รวม สกุลเงิน · `balances` · `pt` ชุดเดียวต่อกลุ่ม + `pt_from_parent` ต่อกลุ่ม + `status_game` ต่อเกม (รูปแบบเดียวกับ account ACC-16 · Member มีแค่ `pt` → `commission_percent`) · login ล่าสุด · วันที่สร้าง · ระบุได้เฉพาะบัญชีในสายล่าง |

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
| MGMT-40 | สร้างได้: บัญชีหลักฝั่ง agent (ADMIN ไม่ได้ — AUTH-43 · Superadmin สร้างได้ — Company Hierarchy: ทุกประเภทสร้าง sub ได้) · sub สร้าง / แก้ / ลบ / ดูรายชื่อ sub ไม่ได้ (`402311`) |
| MGMT-41 | username = `{username เจ้าของ}@{ชื่อ}` · ส่วนหลัง `@` ตาม AUTH-18 (`^[a-z0-9]{3,20}$`) · รหัสผ่านตาม AUTH-36 · ชื่อเล่น: บังคับ 3–32 ตัว `A-Z` `a-z` `0-9` · เบอร์โทรตาม MGMT-08 |
| MGMT-42 | แก้ได้: ชื่อเล่น · เบอร์โทร · สิทธิ์ · แก้ username ไม่ได้ · รหัสผ่านใช้ระบบ reset ของ agent_auth (AUTH-52) |
| MGMT-43 | `status` ของ sub = `ACTIVE` เสมอ ไม่มีเส้นเปลี่ยน · ได้รับผลจากเจ้าของและหัวสายผ่าน `status` ที่ใช้งานจริง (ACC-30 · AUTH-53) |
| MGMT-44 | ลบ = **ลบแถวจริง** · username นำกลับมาใช้ได้ · session ของ sub นั้นถูกลบทันที (request ถัดไป `401203`) · ประวัติใน `auth_audit_logs` ยังอ่านได้ (เก็บ username ณ เวลานั้น · ไม่มี FK) · ไม่ต้อง passcode |
| MGMT-45 | แก้ / ลบได้เฉพาะเจ้าของ · ชั้นบนดูรายชื่อ sub ของบัญชีในสายล่างได้อย่างเดียว |
| MGMT-46 | รายชื่อ sub: ระบุเจ้าของ (ไม่ระบุ = ตัวเอง · ต้องเป็นตัวเองหรือสายล่าง) · ค้นหา username บางส่วน · เรียง A→Z · `page` / `limit` · แต่ละแถว: `id` · username · ชื่อเล่น · เบอร์โทร · สิทธิ์ · วันที่สร้าง · login ล่าสุด (เวลา, IP) |

### สิทธิ์ (ใช้ทั้งหลังบ้าน)

| ID | Rule |
|---|---|
| MGMT-50 | สิทธิ์ต่อเมนู ระดับ `NONE` / `VIEW` / `EDIT` (`EDIT` รวม `VIEW`) · บัญชีหลักได้ `EDIT` ทุกเมนูที่ประเภทนั้นมี · sub ได้ตามที่เจ้าของให้ (ค่าเริ่มต้น `NONE`) · เปลี่ยนรหัสผ่าน / passcode ของตัวเองทำได้เสมอ |
| MGMT-51 | เมนูและการเช็ค (middleware `RequirePermission(menu, level)` บรรทัดเดียวกับ route — กฎข้อ 28): |
| | `dashboard` (NONE / VIEW) — account 1.1 · `account` — account 1.2 Profile, 1.3 API · `member` — ดูรายชื่อ / รายละเอียด (VIEW) · แก้ข้อมูล / สถานะ (EDIT) · `pt` — เห็นค่า `pt` ใน response (VIEW) · แก้ PT / เปิดปิดเกม (EDIT) · `report` (NONE / VIEW) · `bet_cancel` · `payment` (ฝาก-ถอน) · `asset` · `announcement` |
| | ไม่มี `pt` ≥ VIEW → response ไม่มี field `pt`, `pt_from_parent`, `status_game` · **สร้างบัญชี ต้องมี `member` = EDIT และ `pt` = EDIT** |
| MGMT-52 | เมนูที่แต่ละประเภทมี (สิทธิ์ของ sub ให้ได้เฉพาะเมนูเหล่านี้ · ส่งเมนูอื่นมา = `422`): Superadmin = 8 เมนู (ไม่มี `announcement`) · Company · Share · Agent = ครบ 9 เมนู |
| MGMT-53 | Profile (account ACC-12) ส่งสิทธิ์เป็น `["{menu}.view", "{menu}.edit"]` · `EDIT` ส่งทั้งสองค่า |

### ประวัติ

| ID | Rule |
|---|---|
| MGMT-60 | ทุกการสร้าง / แก้ข้อมูล / แก้สถานะ / แก้ PT (รวมแถวลูกที่ปรับตาม MGMT-24) / เปิดปิดเกม / สร้าง-แก้-ลบ sub เก็บ `account_change_logs` ใน tx เดียวกัน (ผู้ทำ, เป้าหมาย, action, ค่าเก่า → ใหม่, ip, request_id) · ห้ามเก็บรหัสผ่าน |

## 4. สิ่งที่พบในโค้ดเก่า และการตัดสินใจ

ไม่มี — ไม่ได้ extract จากโค้ดเก่า

## 5. Endpoints

ใต้ `/bo/pr` ผ่าน `Authenticated` + `PassedGates()` + AUTH-54 · สิทธิ์ใส่บรรทัดเดียวกับ route ·
error ร่วม: `401202`, `401203`, `401301`, `401302`, `401304`, `401306`, `401307`, `402303`

| Method | Path | สิทธิ์ | หมายเหตุ |
|---|---|---|---|
| POST | `/api/v1/bo/pr/agents` | `member` EDIT · `pt` EDIT | สร้าง Company / Share / Agent |
| POST | `/api/v1/bo/pr/members` | `member` EDIT · `pt` EDIT | สร้าง Member |
| GET | `/api/v1/bo/pr/downlines` | `member` VIEW | รายชื่อลูกตรง (agent + Member) |
| GET | `/api/v1/bo/pr/agents/:id` | `member` VIEW | รายละเอียดฝั่ง agent |
| GET | `/api/v1/bo/pr/members/:id` | `member` VIEW | รายละเอียด Member |
| PUT | `/api/v1/bo/pr/agents/:id/info` · `/members/:id/info` | `member` EDIT | ชื่อ · เบอร์โทร |
| PUT | `/api/v1/bo/pr/agents/:id/status` · `/members/:id/status` | `member` EDIT | สถานะ |
| PUT | `/api/v1/bo/pr/agents/:id/pt` · `/members/:id/pt` | `pt` EDIT | ค่าหุ้นส่วน (เฉพาะกลุ่มที่ส่ง) |
| PUT | `/api/v1/bo/pr/agents/:id/games` | `pt` EDIT | เปิด / ปิดเกม |
| GET | `/api/v1/bo/pr/agents/copy-sources` | `pt` VIEW | ลูกตรงฝั่ง agent ของตัวเอง + `pt` + `status_game` (MGMT-35) |
| GET | `/api/v1/bo/pr/subaccounts` | บัญชีหลัก | รายชื่อ sub |
| POST | `/api/v1/bo/pr/subaccounts` | บัญชีหลัก | สร้าง sub |
| PUT | `/api/v1/bo/pr/subaccounts/:id` | บัญชีหลัก | แก้ sub |
| DELETE | `/api/v1/bo/pr/subaccounts/:id` | บัญชีหลัก | ลบ sub |

### POST /api/v1/bo/pr/agents

Request:
```json
{
  "user_type": "SHARE_B2C",
  "username": "share01",
  "password": "••••••••",
  "name": "share01",
  "phone": "0812345678",
  "currencies": ["THB"],
  "pt": {
    "game": { "pt": 30, "force": 0, "remain_quota": 0, "commission_percent": 0.5 }
  },
  "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": false }
}
```
- `user_type` = ประเภทของบัญชีใหม่ (MGMT-02) · ใต้ Company Seamless Reseller / Master ส่ง `SHARE_B2C` ระบบเก็บเป็น `SHARE_RESELLER` / `SHARE_MASTER`
- `currencies` ตาม MGMT-10 ถึง 13 (ไม่ต้องส่ง = ใช้ตามกฎ · ส่งเกิน / ผิด = `422` หรือ `402310`)
- `phone` ไม่บังคับ · `status_game` ไม่ส่ง = เปิดทุกเกม

Response `data`: `{ "id": 12, "username": "share01", "user_type": "SHARE_B2C" }`

Error codes: `422`, `402301`, `402305`, `402307`, `402308`, `402309`, `402310`, `402401`, `402403`

### POST /api/v1/bo/pr/members

Request: `username`, `password`, `name`, `phone`, `pt` (กลุ่มละ `commission_percent` เท่านั้น — MGMT-21) · สกุลเงินตามผู้สร้าง
Response `data`: `{ "id": 501, "username": "mem01" }`
Error codes: `422`, `402301`, `402309`, `402401`, `402403`

### GET /api/v1/bo/pr/downlines

Query: `parent_id` (ไม่ส่ง = ตัวเอง · ต้องเป็นบัญชีฝั่ง agent ในสายล่าง) · `q` · `page` · `limit`
Response `data` (`response.Page`) แต่ละแถว:
```json
{
  "id": 12,
  "role": "SHAREHOLDER",
  "user_type": "SHARE_B2C",
  "username": "share01",
  "name": "share01",
  "phone": "0812345678",
  "status": "SUSPENDED",
  "pt": {
    "game": { "pt": 30, "force": 0, "remain_quota": 0, "commission_percent": 0.5 }
  },
  "balances": [ { "currency": "THB", "amount": 0.00 } ]
}
```
- `status` = สถานะที่ใช้งานจริง (ACC-30) — ตัวอย่างนี้หัวสายของ share01 ถูกระงับ
- แถว Member: `role = "MEMBER"` · `pt` มีแค่ `commission_percent` · `phone` = `null` เมื่อไม่ได้ตั้ง
- `balances` ตาม account ACC-19 (`amount = 0` จนกว่าจะมี module การชำระเงิน · บัญชี Seamless = `0`)

Error codes: `402402`

### GET /api/v1/bo/pr/agents/:id · /members/:id

Response `data` (ตัวอย่างฝั่ง agent — หลังสร้างด้วย `pt.game.pt = 20` และปิด `scratch_card`):
```json
{
  "id": 12,
  "role": "SHAREHOLDER",
  "user_type": "SHARE_B2C",
  "username": "share01",
  "name": "share01",
  "phone": "0812345678",
  "status": "ACTIVE",
  "parent_username": "comp01",
  "currencies": ["THB"],
  "balances": [ { "currency": "THB", "amount": 0.00 } ],
  "pt": {
    "game": { "pt": 20, "force": 0, "remain_quota": 0, "commission_percent": 0.5 }
  },
  "pt_from_parent": { "game": 90 },
  "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": false },
  "passcode_set": true,
  "last_login_at": "2026-10-05T10:00:00+07:00",
  "last_login_ip": "203.0.113.10",
  "created_at": "2026-10-01T09:00:00+07:00"
}
```
- รูปแบบ `pt`, `pt_from_parent`, `status_game` เหมือน Profile (account ACC-16) — ไม่มี `pt_by_game`
- Member: ไม่มี `pt_from_parent`, `status_game`, `passcode_set` · `pt.game` มีแค่ `commission_percent`

Error codes: `402402`

### PUT /api/v1/bo/pr/agents/:id/info · /members/:id/info

Request: `{ "name": "share01", "phone": null }` · Response: ไม่มี `data` · Error codes: `422`, `402304`, `402402`, `402403`

### PUT /api/v1/bo/pr/agents/:id/status · /members/:id/status

Request: `{ "status": "SUSPENDED" }` · Response: ไม่มี `data` · Error codes: `422`, `402304`, `402402`

### PUT /api/v1/bo/pr/agents/:id/pt · /members/:id/pt

Request (ส่งเฉพาะกลุ่มที่จะแก้ · ในกลุ่มต้องครบ 4 ค่า · ห้าม `null`):
```json
{ "pt": { "game": { "pt": 14, "force": 0, "remain_quota": 0, "commission_percent": 0.5 } } }
```
Response: ไม่มี `data` · Error codes: `422`, `402304`, `402305`, `402306`, `402307`, `402308`, `402309`, `402402`

### PUT /api/v1/bo/pr/agents/:id/games

Request: `{ "status_game": { "scratch_card": false } }` (ส่งเฉพาะเกมที่จะเปลี่ยน) · Response: ไม่มี `data` ·
Error codes: `422`, `402304`, `402402`

### GET /api/v1/bo/pr/agents/copy-sources

Response `data`: `[{ "id": 12, "username": "share01", "user_type": "SHARE_B2C", "pt": {...}, "status_game": {...} }]` (ลูกตรงฝั่ง agent ทั้งหมด เรียง A→Z)

### Sub

- `GET /subaccounts?owner_id=&q=&page=&limit=` → แถว `{ id, username, name, phone, permissions, created_at, last_login_at, last_login_ip }`
- `POST /subaccounts` → `{ "name_suffix": "staff", "password": "••••", "name": "staff01", "phone": null, "permissions": { "dashboard": "VIEW", "member": "EDIT", "pt": "NONE", ... } }` → `data`: `{ "id": 30, "username": "comp01@staff" }`
- `PUT /subaccounts/:id` → `{ "name", "phone", "permissions" }` (แทนทั้งชุด)
- `DELETE /subaccounts/:id`
- Error codes: `422`, `402311`, `402401`, `402403`, `402404`

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
    status_game       BOOLEAN     NOT NULL DEFAULT true,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (agent_id, game_code),
    CHECK (pt_bp <= pt_from_parent_bp)
);

CREATE TABLE members (
    id            BIGSERIAL    PRIMARY KEY,
    agent_id      BIGINT       NOT NULL REFERENCES user_agents(id),  -- ผู้สร้าง
    username      VARCHAR(32)  NOT NULL,
    password_hash VARCHAR(100) NOT NULL,
    name          VARCHAR(32)  NOT NULL,
    phone         VARCHAR(15),
    currency      VARCHAR(4)   NOT NULL,
    status        VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','SUSPENDED','LOCKED')),
    last_login_at TIMESTAMPTZ,
    last_login_ip VARCHAR(45),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_members_username UNIQUE (username),
    CONSTRAINT ck_members_username CHECK (username = lower(username) AND username ~ '^[a-z0-9]{3,32}$'),
    CONSTRAINT ck_members_phone CHECK (phone IS NULL OR phone ~ '^[0-9]{8,15}$')
);
CREATE INDEX idx_members_agent_username ON members(agent_id, username);
CREATE UNIQUE INDEX uq_members_phone ON members(phone) WHERE phone IS NOT NULL;

CREATE TABLE member_game_settings (             -- Member มีแค่ Commission ต่อเกม (MGMT-21)
    member_id     BIGINT      NOT NULL REFERENCES members(id),
    category      VARCHAR(30) NOT NULL,
    game_code     VARCHAR(50) NOT NULL,
    commission_bp INT         NOT NULL DEFAULT 0 CHECK (commission_bp BETWEEN 0 AND 100),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (member_id, game_code)
);

ALTER TABLE subaccounts
    ADD COLUMN name        VARCHAR(32),
    ADD COLUMN phone       VARCHAR(15),
    ADD COLUMN permissions JSONB NOT NULL DEFAULT '{}';   -- {"member":"EDIT", ...} อ่านพร้อมแถว sub ทุก request (AUTH-27)
CREATE INDEX idx_subaccounts_agent_username ON subaccounts(agent_id, username);

CREATE TABLE account_change_logs (               -- MGMT-60
    id              BIGSERIAL   PRIMARY KEY,
    actor_type      VARCHAR(10) NOT NULL,        -- AGENT / SUB
    actor_id        BIGINT      NOT NULL,
    actor_username  VARCHAR(71) NOT NULL,
    target_type     VARCHAR(10) NOT NULL,        -- AGENT / MEMBER / SUB
    target_id       BIGINT      NOT NULL,
    target_username VARCHAR(71) NOT NULL,
    action          VARCHAR(30) NOT NULL,        -- CREATE / UPDATE_INFO / UPDATE_STATUS / UPDATE_PT / ADJUST_PT / UPDATE_GAMES / DELETE
    old_value       JSONB,
    new_value       JSONB,
    ip              VARCHAR(45),
    request_id      VARCHAR(64),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_account_change_logs_target ON account_change_logs(target_type, target_id, created_at);
```

- Superadmin มีแถว `agent_game_settings` ของตัวเอง (`pt_from_parent_bp = 10000` — MGMT-22) · ADMIN ไม่มี
- สายล่างใช้ recursive CTE ตาม `parent_id` (แบบ `ListAncestorStatusesRepository`) · `idx_user_agents_parent_id` มีแล้ว
- รายการสกุล 27 สกุล · กลุ่ม PT · รหัสเกม (`coin_toss`, `rock_paper_scissors`, `scratch_card`) · เมนูสิทธิ์ เป็นค่าคงที่ใน `app/core`

## 7. Test cases

| Rule | Input | Expected |
|---|---|---|
| MGMT-02 | Company Transfer สร้าง `SHARE_B2B` / `SHARE_B2C` | สำเร็จ |
| MGMT-02 | Company Seamless Reseller สร้าง `SHARE_B2C` | สำเร็จ · `user_type = SHARE_RESELLER` |
| MGMT-02 | Company Seamless 1 to 1 สร้าง Share · Share B2B สร้าง Member | `402301` |
| MGMT-02 | Share B2C สร้าง Member · Agent สร้าง Member · Agent สร้าง Agent 3 ชั้น | สำเร็จ |
| MGMT-04 | สร้าง Share Master | มีแถว `api_credentials` ของบัญชีใหม่ |
| MGMT-05 | username `ab` / 33 ตัว / มี `_` | `422` · `Share01` เก็บเป็น `share01` |
| MGMT-05 | username ซ้ำ | `402401` |
| MGMT-08 | เบอร์ `+66812345678` / 7 หลัก | `422` · เบอร์ซ้ำ `402403` |
| MGMT-12 | Share B2B ไม่ส่งสกุล · Share B2C ส่ง 2 สกุล | `422` |
| MGMT-14 | Agent ใต้ Share B2B (THB, USD) เลือก JPY | `402310` |
| MGMT-17 | ส่ง `"force": null` | `422` |
| MGMT-18 | ถือ `30.25` (ไม่ลง 0.5%) · commission `1.01` · ถือ `30.123` (ทศนิยมเกิน 2) | `422` |
| MGMT-18 | ได้รับ 80% ส่งถือ `80.5` | `402305` |
| MGMT-18 | ผู้สร้างมี commission 0.5% ตั้งให้ลูก 0.6% | สำเร็จ |
| MGMT-19 | Company Seamless Master ถือ `1` | `402307` |
| MGMT-23 | แก้ PT ของหลาน (ไม่ใช่ผู้สร้างโดยตรง) | `402304` |
| MGMT-23 | ส่งแค่กลุ่ม `game` | กลุ่มอื่นไม่เปลี่ยน |
| MGMT-24 | ตัวอย่างตาราง (Company ถือ 10 → 14) | share1 ถือ 26 · share2 ถือ 1 · Agent ไม่เปลี่ยน · มี log ทุกแถว |
| MGMT-24 | Company ถือ 10 → 16 (ปล่อย 74 < 75) | `402306` · ไม่มีแถวไหนเปลี่ยน |
| MGMT-24 | Company ถือ 10 → 5 (ปล่อย 85) | share1 ถือ 35 · share2 ถือ 10 |
| MGMT-26 | `parent_id` เป็นบัญชีสายอื่น | `402402` |
| MGMT-26 | Agent ที่มีทั้ง Agent และ Member เป็นลูก | ได้ทั้งสองแบบ `role` ถูกต้อง เรียง A→Z |
| MGMT-27 | `q=ab` | ได้เฉพาะ username ที่มี `ab` |
| MGMT-29 | ดูรายละเอียด | ไม่มี field รหัสผ่าน / passcode / hash |
| MGMT-16 | สร้างด้วย `pt.game.pt = 20` | `agent_game_settings` มี 3 แถว (ทุกเกมใน `minigame`) ค่า `pt_bp = 2000` · รายละเอียดแสดง `pt.game.pt = 20` และ `status_game` ครบ 3 เกม · รายชื่อแสดง `pt.game.pt = 20` |
| MGMT-30 | Share แก้สถานะ Agent ของ Agent ลูก (ไม่ใช่ผู้สร้างโดยตรง) | `402304` |
| MGMT-30 | LOCKED → ACTIVE โดยผู้สร้าง | สำเร็จ |
| MGMT-31 | Company ระงับ Share | Share และ Agent ใต้ Share `status = SUSPENDED` · Company `status = ACTIVE` |
| MGMT-30 | Share ล็อก Agent ลูก (Share ACTIVE) | Agent `status = LOCKED` · Share ยังใช้งานได้ |
| MGMT-30 | หัวสายของ Agent ถูกระงับ · ผู้สร้างตั้ง Agent เป็น ACTIVE | สำเร็จ · Agent ยังแสดง `status = SUSPENDED` |
| MGMT-44 | ลบ sub ที่ login อยู่ | request ถัดไปของ sub `401203` · สร้าง sub ชื่อเดิมได้ |
| MGMT-45 | ชั้นบนลบ sub ของบัญชีในสายล่าง | `402404` |
| MGMT-40 | sub เรียก `POST /subaccounts` | `402311` |
| MGMT-51 | sub มี `member` EDIT · `pt` NONE สร้างบัญชี | `402303` |
| MGMT-51 | sub มี `member` VIEW · `pt` NONE ดูรายชื่อ | สำเร็จ · ไม่มี field `pt` |
| MGMT-51 | sub มี `pt` EDIT · `member` NONE แก้ PT | สำเร็จ |
| MGMT-60 | แก้ PT สำเร็จ | มี log ค่าเก่า / ใหม่ ไม่มีรหัสผ่าน |

| MGMT-05 | สร้าง Member ชื่อเดียวกับ Agent ที่มีอยู่ | `402401` |
| MGMT-20 | Company ปิด `scratch_card` แล้ว Share ใต้ Company ตั้งเปิดให้ Agent | Agent ยังเล่น `scratch_card` ไม่ได้ (เกมปิดจากหัวสาย) |
| MGMT-25 | ลูกได้รับ 80 · Force 70 · ผู้สร้างลดค่าปล่อยจน ลูกได้รับ 60 | Force ของลูกเหลือ 60 · มี log |
| MGMT-52 | sub ของ Superadmin ได้สิทธิ์ `announcement` | `422` |

### 7.1 ลำดับเช็ค (เส้นสร้าง)

`422` (รูปแบบ) → `402303` (สิทธิ์ sub) → `402301` (สร้างประเภทนี้ไม่ได้) → `402310` (สกุล) → `402401` / `402403` (ซ้ำ) →
`402305` / `402307` / `402308` / `402309` (ค่าหุ้นส่วน)

## 8. Contract changes (แจ้ง frontend)

- เส้นทั้งหมดในหัวข้อ 5 เป็นเส้นใหม่
- ค่า % ส่งเป็น JSON number (ACC-18) อยู่ใน object `pt` แยกกลุ่ม · ห้าม `null` · ไม่ตั้ง = `0`
- สิทธิ์ของ sub ส่งเป็น object `{ "menu": "NONE" | "VIEW" | "EDIT" }`
- account (review รอบ 2): Profile ส่ง `pt` ชุดเดียว + `pt_from_parent` + `status_game` และ `status` key เดียว — ตรงกับ module นี้ · ข้อมูลอ่านจาก `agent_game_settings`

## 9. Error codes (`bb=02`) — business error ตอบ HTTP 200

| Code | ความหมาย |
|---|---|
| 402301 | สร้างบัญชีประเภทนี้ไม่ได้ |
| 402303 | ไม่มีสิทธิ์ใช้งานเมนูนี้ |
| 402304 | แก้ไขได้เฉพาะผู้สร้างของบัญชีนี้ |
| 402305 | ค่าถือเกินกว่าที่ได้รับ |
| 402306 | ค่าปล่อยต่ำกว่าที่ชั้นล่างปล่อยอยู่ (msg บอกค่าต่ำสุดที่ตั้งได้) |
| 402307 | Company Seamless Master ต้องถือ 0 และปล่อยทั้งหมด |
| 402308 | Force / Remain เกินที่กำหนด |
| 402309 | Commission เกินที่กำหนด |
| 402310 | สกุลเงินไม่อยู่ในสกุลของผู้สร้าง |
| 402311 | บัญชีย่อยทำรายการนี้ไม่ได้ |
| 402401 | username นี้ถูกใช้แล้ว |
| 402402 | ไม่พบบัญชีในสายของคุณ |
| 402403 | เบอร์โทรนี้ถูกใช้แล้ว |
| 402404 | ไม่พบบัญชีย่อยของคุณ |

`402302` (เดิม: บัญชีถูกระงับ) ตัดออก — AUTH-54 บังคับที่ middleware แทน
