# การจัดการสมาชิก (`agent_management`) — Spec

- สถานะ: **DRAFT** (เขียนใหม่ทั้งฉบับ 2026-10-06 ตาม review ของ lead + กฎที่เก็บจากภาพหน้าจอ)
- อนุมัติโดย: — · วันที่: —
- ชื่อ module ในโค้ด: `agent_management` (`controllers/agent_management`, `dto/agent_management`, `service/agent_management`, `core/agent_management`)
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
| ได้รับ / ถือ / ให้ลูก | ได้รับ (`pt_from_parent`) = PT ที่ผู้สร้างให้ · ให้ลูก = PT ที่แบ่งให้ลูกแต่ละคน (ไม่เกินที่ได้รับ) · ถือ (`pt`) = ส่วนที่ถือจาก Member ใต้ตัวเองตรงๆ (ตั้งเอง) · ผู้สร้างถือในสายของลูก = ได้รับ − ให้ลูกคนนั้น |
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
| MGMT-08 | เบอร์โทร: ไม่บังคับ (ไม่กรอกส่ง `""` · ห้าม `null` — account ACC-32) · 8–15 ตัว ตัวเลขเท่านั้น (ห้าม `+`) · ห้ามซ้ำภายในตารางเดียวกัน (unique index ต่อตาราง) |
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
| MGMT-15 | บัญชีฝั่ง Seamless ทุกชั้น (Company Seamless ทุกประเภท และสายล่าง) ไม่มียอดเงิน · ส่ง `balance` ตอนสร้าง = `422` |
| MGMT-15A | **ยอดเงินตั้งต้นตอนสร้าง** (`balance` ในเส้นสร้าง · ไม่บังคับ · บัญชีฝั่ง Transfer เท่านั้น): ต่อสกุล · ต้องเป็นสกุลของบัญชีใหม่ · มากกว่า 0 · ทศนิยมไม่เกิน 2 ตำแหน่ง · **โอนจากยอดของผู้สร้าง** ในสกุลเดียวกัน (ผู้สร้างลด · บัญชีใหม่เพิ่ม) · ยอดผู้สร้างไม่พอ = ปฏิเสธทั้งคำสั่ง (`402312`) |
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
| | • ตอนนี้มีกลุ่ม `game` กลุ่มเดียว = ทุกเกมในหมวด `minigame` (`coin_toss`, `rock_paper_scissors`, `scratch_card`) · `provider` ฯลฯ เพิ่มเมื่อมีระบบนั้น · การจับคู่กลุ่ม → หมวด → เกม กำหนดใน `app/core` เพิ่มได้โดยไม่เปลี่ยน API / ตาราง |
| | • **ผู้สร้างตั้งให้ลูก** (สร้าง / แก้ลูก): กลุ่มละ 5 ค่า `pt_from_parent` (ให้ลูกคนนี้) · `force` · `remain_quota` · `commission_percent` · `status` (เปิด / ปิดทั้งกลุ่ม — MGMT-20) |
| | • **บัญชีตั้งของตัวเอง** (MGMT-22): กลุ่มละ 1 ค่า `pt` (ถือจาก Member ใต้ตัวเองตรงๆ) |
| | • แสดง (response ทุกเส้น): กลุ่มละ 6 ค่า `pt_from_parent` · `pt` · `force` · `remain_quota` · `commission_percent` · `status` · `status_game` แยกต่อเกม (MGMT-20) |
| | • เกมใหม่ที่เพิ่มเข้าหมวดภายหลัง: ทุกบัญชีได้ค่าของกลุ่มนั้น และ `status_game = true` อัตโนมัติ |
| MGMT-17 | ค่า % ใน API เป็น JSON number ทศนิยมไม่เกิน 2 ตำแหน่ง (เช่น `30`, `0.5`) · ภายใน DB / Go เป็นจำนวนเต็ม bp (ACC-18) · **ห้าม `null`** · ไม่ตั้ง Force / Remain = `0` |
| MGMT-18 | ค่าที่ให้ลูก (`pt_from_parent`) และค่าถือ (`pt`): 0 ถึง**ค่าที่ตัวเองได้รับ** ทีละ 0.5% · Force / Remain: 0 ถึงค่าที่ให้ลูกคนนั้น ทีละ 0.5% · Commission: 0–1% ทีละ 0.1% · Commission ของลูกตั้ง**เกินของผู้สร้างได้** (เช่น ผู้สร้างได้ 0.5% ตั้งให้ลูก 0.6% ได้) แต่ไม่เกิน 1% |
| MGMT-19 | Company Seamless Master: `pt` ล็อกที่ 0 · ค่าที่ให้ Share Master ต้อง**เท่ากับค่าที่ตัวเองได้รับ** · Force และ Remain ที่ให้ Share Master = 0 · แก้ได้แค่ Commission (เอกสาร PT) |
| MGMT-20 | เปิด / ปิดเกมรายบัญชี มี 2 ระดับ ผู้สร้างตั้งให้ทั้งคู่: **ทั้งกลุ่ม** `pt` → กลุ่ม → `status` (bool · `false` = ปิดทุกเกมในกลุ่ม) และ **ทีละเกม** `status_game` → รหัสเกม (bool · แยกจาก `pt`) · เกมเล่นได้เมื่อ `status` ของกลุ่มและ `status_game` ของเกมนั้นเป็น `true` ทั้งคู่ · ปิดแล้ว Member ในสายเล่นเกมนั้นไม่ได้ (บังคับใน module เดิมพัน) · ชั้นบนปิดแล้ว**สายล่างเปิดเองไม่ได้** — เกมใช้ได้จริงเมื่อบัญชีตัวเองและหัวสายทุกชั้นเปิดอยู่ (แบบเดียวกับ `status` ที่ใช้งานจริง — ACC-30) |
| MGMT-21 | สร้างบัญชีฝั่ง agent ต้องส่งครบทุกกลุ่มที่มี · Member มีแค่ `commission_percent` ต่อกลุ่ม (Member ไม่ถือ PT — เอกสาร PT Commission) กระจายและเก็บต่อเกมแบบเดียวกัน (`member_game_settings`) |
| MGMT-22 | **ผู้สร้างกำหนดค่าที่ให้ลูกแต่ละคนแยกกัน** (ทุกชั้น รวม Superadmin → Company): |
| | • ลูกแต่ละคนได้รับไม่เท่ากันได้ · ค่าที่ให้ลูกแต่ละคน**ห้ามเกินค่าที่ผู้สร้างได้รับ** · Superadmin ได้รับ 100% |
| | • ผู้สร้าง**ถือในสายของลูกคนนั้น** = ค่าที่ผู้สร้างได้รับ − ค่าที่ให้ลูกคนนั้น (ระบบคำนวณ ไม่ต้องส่ง) |
| | • ทุกบัญชี**ตั้งค่าถือ `pt` เอง**สำหรับ Member ที่อยู่ใต้ตัวเองตรงๆ (ห้ามเกินค่าที่ได้รับ) · บัญชีใหม่เริ่มที่ `pt` = ค่าที่ได้รับ (ถือทั้งหมด) จนกว่าจะแก้ |
| | • ส่วนที่ไม่มีใครถือ (เช่น ได้รับ 60 ถือ 40 → เหลือ 20) ผ่าน Force / Remain ตอนคิดเงินก่อน ถ้าไม่มีก็**คืน Superadmin** (เอกสาร PT Force / PT Remain) |
| MGMT-23 | ค่าที่ให้ลูก, Force, Remain, Commission แก้ได้เฉพาะ**ผู้สร้างโดยตรง** · `pt` ของบัญชีแก้ได้เฉพาะ**บัญชีนั้นเอง** · sub ตามสิทธิ์ (MGMT-50) · แก้เฉพาะกลุ่มที่ส่ง · ในกลุ่มที่ส่งต้องครบทุกค่าของเส้นนั้น · มีผลกับ bet ใหม่ (bet เก็บค่า ณ ตอน bet — module เดิมพัน) |
| MGMT-24 | **ลดค่าที่ให้ลูก**: ค่าใหม่ต้อง ≥ ค่าที่มากที่สุดระหว่าง (ค่าที่ลูกให้ลูกของมันแต่ละคน, `pt` ของลูก) ไม่งั้น**ปฏิเสธทั้งคำสั่ง** (`402306` msg บอกค่าต่ำสุดที่ตั้งได้) · ระบบไม่แก้ค่าของลูกหรือหลานให้เอง — ถ้าจะลดต่ำกว่านั้นลูกต้องลดของตัวเองก่อน · **เพิ่มค่าที่ให้ลูก**: ได้ถึงค่าที่ผู้สร้างได้รับ · ลูกคนอื่นไม่กระทบ · tx + lock แถวลูกและลูกของลูก (อ่านค่าที่ใช้อยู่) เรียงตาม id |
| MGMT-25 | Force / Remain ในคำสั่งเดียวกันต้องไม่เกินค่าที่ให้ลูกใหม่ (`402308`) |

ตัวอย่าง (กลุ่ม `game`):

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
| MGMT-27 | ตัวกรองมีแค่ `page` / `limit` และค้นหา username บางส่วน (ไม่สนตัวพิมพ์) · เรียง username A→Z · `limit` ค่าเริ่มต้น 20 สูงสุด 100 (`utils.ParsePage`) |
| MGMT-28 | แต่ละแถว: `id` · `role` (บอกว่าเป็นฝั่ง agent หรือ Member) · `user_type` · username · ชื่อ · เบอร์โทร · `status` (ACC-30) · `pt` (ตาม MGMT-51) · `balances` (ยอดแยกสกุลตาม account ACC-19 — ระหว่างยังไม่มี module การชำระเงินส่ง `0`) |
| MGMT-29 | เส้นรายละเอียดแยก agent / Member: ทุก field ของบัญชี **ยกเว้น** password, passcode, hash และ token · รวม สกุลเงิน · `balances` · `pt` ชุดเดียวต่อกลุ่ม (รวม `pt_from_parent` ในกลุ่ม — MGMT-16) + `status_game` ต่อเกม (รูปแบบเดียวกับ account ACC-16 · Member มีแค่ `pt` → `commission_percent`) · login ล่าสุด · วันที่สร้าง · ระบุได้เฉพาะบัญชีในสายล่าง |

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
| MGMT-41 | username = `{username เจ้าของ}@{ชื่อ}` · ส่วนหลัง `@` 3–20 ตัว `a-z` `0-9` (AUTH-18) · **รับตัวพิมพ์ใหญ่ได้ หลังบ้านแปลงเป็นตัวเล็กเอง** (ตอน login ก็แปลงก่อนเช็ค — AUTH-01) · รหัสผ่านตาม AUTH-36 · ชื่อเล่น: บังคับ 3–32 ตัว `A-Z` `a-z` `0-9` · เบอร์โทรตาม MGMT-08 |
| MGMT-42 | แก้ได้: ชื่อเล่น · เบอร์โทร · สิทธิ์ · สถานะ (MGMT-43) · แก้ username ไม่ได้ · รหัสผ่านใช้ระบบ reset ของ agent_auth (AUTH-52) |
| MGMT-43 | สถานะของ sub: `ACTIVE` / `INACTIVE` · **ผู้สร้าง sub เป็นคนเปลี่ยน** · `INACTIVE` = login ได้แต่ทำรายการไม่ได้ เหมือน `SUSPENDED` (AUTH-54 · ภายในเก็บเป็น `SUSPENDED` ในตาราง `subaccounts` · API แสดง `INACTIVE`) · เปลี่ยนกลับเป็น `ACTIVE` ได้ · ไม่ต้อง passcode · ถ้าเจ้าของหรือหัวสายถูกระงับ / ล็อก sub ได้ผลตาม `status` ที่ใช้งานจริง (ACC-30 · AUTH-53) |
| MGMT-44 | **ไม่มีการลบ sub** — เลิกใช้ให้เปลี่ยนเป็น `INACTIVE` |
| MGMT-45 | แก้ / เปลี่ยนสถานะได้เฉพาะเจ้าของ · ชั้นบนดูรายชื่อและรายละเอียด sub ของบัญชีในสายล่างได้อย่างเดียว |
| MGMT-46 | รายชื่อ sub: ระบุเจ้าของ (ไม่ระบุ = ตัวเอง · ต้องเป็นตัวเองหรือสายล่าง) · ค้นหา username บางส่วน · เรียง A→Z · `page` / `limit` · แต่ละแถว: `id` · username · ชื่อเล่น · เบอร์โทร · `status` · สิทธิ์ · วันที่สร้าง · login ล่าสุด (เวลา, IP) · รายละเอียด sub (`GET /manage/subaccounts/:id`) ส่ง field ชุดเดียวกัน |

### สิทธิ์ (ใช้ทั้งหลังบ้าน)

| ID | Rule |
|---|---|
| MGMT-50 | สิทธิ์ต่อเมนู ระดับ `off` (เข้าไม่ได้) / `view` (ดูอย่างเดียว) / `edit` (ดูและแก้) · **ไม่ติ๊ก = `off`** · บัญชีหลักได้ `edit` ทุกเมนูที่ประเภทนั้นมี · sub ได้ตามที่เจ้าของให้ · รูปแบบใน API เป็น object `{ "{menu}": "off" \| "view" \| "edit" }` ทั้ง request และ response (รวม Profile — account ACC-12) |
| MGMT-51 | เช็คด้วย middleware แยก `RequirePermission(menu, level)` บรรทัดเดียวกับ route (กฎข้อ 28) · route ของ admin ใช้ middleware แยก `RequireRole(ADMIN)` (AUTH-44) · เมนู: |
| | `dashboard` (off / view) — account 1.1 · `account` — account 1.2 Profile, 1.3 API · `member` — ดูรายชื่อ / รายละเอียด (view) · สร้าง / แก้ข้อมูล / สถานะ (edit) · `pt` — เห็นค่า `pt` ใน response (view) · แก้ PT (edit) · `report` (off / view) · `bet_cancel` · `payment` (ฝาก-ถอน) · `asset` · `announcement` · `rate` (อัตราแพ้ชนะ — เฉพาะ Superadmin) |
| | ไม่มี `pt` ≥ view → response ไม่มี field `pt` · **สร้างบัญชี ต้องมี `member` = edit และ `pt` = edit** (ถ้าส่ง `balance` ต้องมี `payment` = edit ด้วย — MGMT-15A) · เปลี่ยนรหัสผ่าน / passcode ของตัวเองไม่ต้องใช้สิทธิ์ (แต่ถูก AUTH-54 กันตอนถูกระงับ) |
| MGMT-52 | เมนูที่แต่ละประเภทมี (สิทธิ์ของ sub ให้ได้เฉพาะเมนูเหล่านี้ · ส่งเมนูอื่นมา = `422`): **Superadmin** = 9 เมนู — `dashboard` · `account` · `member` · `pt` · `report` · `bet_cancel` · `payment` · `asset` · `rate` (อัตราแพ้ชนะ · ไม่มี `announcement`) · **Company · Share · Agent** = 9 เมนูเดียวกัน — `dashboard` · `account` · `member` · `pt` · `report` · `bet_cancel` · `payment` · `asset` · `announcement` (ไม่มี `rate` — ยืนยัน 2026-10-07) · sub ของ Superadmin ได้สิทธิ์ `rate` เมื่อ Superadmin ให้ |
| MGMT-53 | Profile (account ACC-12) ส่ง `permissions` เป็น object รูปแบบเดียวกับ MGMT-50 · บัญชีหลักได้ `edit` ทุกเมนูของประเภทตัวเอง |

### ประวัติ

| ID | Rule |
|---|---|
| MGMT-60 | ทุกการสร้าง / แก้ข้อมูล / แก้สถานะ / แก้ PT (รวมแถวลูกที่ปรับตาม MGMT-24) / เปิดปิดเกม / สร้าง-แก้-เปลี่ยนสถานะ sub / ยอดเงินตั้งต้น เก็บ `account_change_logs` ใน tx เดียวกัน (ผู้ทำ, เป้าหมาย, action, ค่าเก่า → ใหม่, ip, request_id) · ห้ามเก็บรหัสผ่าน |

## 4. สิ่งที่พบในโค้ดเก่า และการตัดสินใจ

ไม่มี — ไม่ได้ extract จากโค้ดเก่า

## 5. Endpoints

ใช้แค่ method `GET` (อ่าน) และ `POST` (สร้าง / แก้) · ใต้ `/bo/pr` ผ่าน `Authenticated` + `PassedGates()` (รวม AUTH-54) ·
สิทธิ์ของ sub ใช้ middleware `RequirePermission(menu, level)` บรรทัดเดียวกับ route ·
error ร่วม: `401202`, `401203`, `401301`, `401302`, `401304`, `401306`, `401307`, `401311`, `402303`

| Method | Path | สิทธิ์ (sub) | หมายเหตุ |
|---|---|---|---|
| POST | `/api/v1/bo/pr/manage/agents` | `member` edit · `pt` edit (+ `payment` edit ถ้าส่ง `balance`) | สร้าง Company / Share / Agent |
| POST | `/api/v1/bo/pr/manage/members` | `member` edit · `pt` edit (+ `payment` edit ถ้าส่ง `balance`) | สร้าง Member |
| GET | `/api/v1/bo/pr/manage/downlines` | `member` view | รายชื่อลูกตรง (agent + Member) |
| GET | `/api/v1/bo/pr/manage/agents/:id` | `member` view | รายละเอียดฝั่ง agent |
| GET | `/api/v1/bo/pr/manage/members/:id` | `member` view | รายละเอียด Member |
| POST | `/api/v1/bo/pr/manage/agents/:id/info` · `/manage/members/:id/info` | `member` edit | แก้ชื่อ · เบอร์โทร |
| POST | `/api/v1/bo/pr/manage/agents/:id/status` · `/manage/members/:id/status` | `member` edit | แก้สถานะ |
| POST | `/api/v1/bo/pr/manage/agents/:id/pt` · `/manage/members/:id/pt` | `pt` edit | ค่าที่ให้ลูก + Force / Remain / Commission + เปิดปิดกลุ่ม (ผู้สร้างตั้งให้ลูก) |
| POST | `/api/v1/bo/pr/manage/agents/me/pt` | `pt` edit | ค่าถือ `pt` ของตัวเอง (MGMT-22) |
| POST | `/api/v1/bo/pr/manage/agents/:id/games` | `pt` edit | เปิด / ปิดทีละเกม — ❓ รอ lead: จะย้ายไป module อื่นและให้ใครใช้ |
| GET | `/api/v1/bo/pr/manage/agents/copy-sources` | `pt` view | ลูกตรงฝั่ง agent ของตัวเอง + `pt` (MGMT-35) |
| GET | `/api/v1/bo/pr/manage/subaccounts` | บัญชีหลัก | รายชื่อ sub |
| GET | `/api/v1/bo/pr/manage/subaccounts/:id` | บัญชีหลัก | รายละเอียด sub |
| POST | `/api/v1/bo/pr/manage/subaccounts` | บัญชีหลัก | สร้าง sub |
| POST | `/api/v1/bo/pr/manage/subaccounts/:id` | บัญชีหลัก (เจ้าของ) | แก้ชื่อเล่น · เบอร์โทร · สิทธิ์ |
| POST | `/api/v1/bo/pr/manage/subaccounts/:id/status` | บัญชีหลัก (เจ้าของ) | `ACTIVE` / `INACTIVE` (MGMT-43) |

### POST /api/v1/bo/pr/manage/agents

Request:
```json
{
  "request_id": "6f1c2a1e-3b7d-4c55-9a40-0f7f3c2d8e11",
  "user_type": "SHARE_B2C",
  "username": "Share01",
  "password": "••••••••",
  "name": "share01",
  "phone": "0812345678",
  "currencies": ["THB"],
  "balance": { "THB": 10000.00 },
  "pt": {
    "game": { "pt_from_parent": 70, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true }
  },
  "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": false }
}
```
- `request_id` บังคับ (UUID จากหน้าบ้าน — กันกดซ้ำ MGMT-15A) · ส่งซ้ำด้วยค่าเดิม = ได้ผลเดิม
- `user_type` = ประเภทของบัญชีใหม่ (MGMT-02) · ใต้ Company Seamless Reseller / Master ส่ง `SHARE_B2C` ระบบเก็บเป็น `SHARE_RESELLER` / `SHARE_MASTER`
- `username` รับตัวพิมพ์ใหญ่ได้ เก็บเป็นตัวเล็ก (`Share01` → `share01`)
- `currencies` ตาม MGMT-10 ถึง 13 (ไม่ต้องส่ง = ใช้ตามกฎ · ส่งเกิน / ผิด = `422` หรือ `402310`)
- `balance` ไม่บังคับ · ยอดเงินตั้งต้นต่อสกุล โอนจากยอดของผู้สร้าง (MGMT-15A) · บัญชีฝั่ง Seamless ห้ามส่ง
- `phone` ไม่บังคับ · `status_game` ไม่ส่ง = เปิดทุกเกม (❓ รอ lead — อาจย้ายไป module อื่น)

Response `data`: `{ "id": 12, "username": "share01", "user_type": "SHARE_B2C" }`

Error codes: `422`, `402301`, `402305`, `402307`, `402308`, `402309`, `402310`, `402312`, `402401`, `402403`

### POST /api/v1/bo/pr/manage/members

Request: `request_id`, `username` (รับตัวพิมพ์ใหญ่ได้ เก็บเป็นตัวเล็ก), `password`, `name`, `phone`, `balance` (ไม่บังคับ — MGMT-15A), `pt` (กลุ่มละ `commission_percent` เท่านั้น — MGMT-21) · สกุลเงินตามผู้สร้าง
Response `data`: `{ "id": 501, "username": "mem01" }`
Error codes: `422`, `402301`, `402309`, `402312`, `402401`, `402403`

### GET /api/v1/bo/pr/manage/downlines

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
    "game": { "pt_from_parent": 80, "pt": 30, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true }
  },
  "balances": [ { "currency": "THB", "amount": 0.00 } ]
}
```
- `status` = สถานะที่ใช้งานจริง (ACC-30) — ตัวอย่างนี้หัวสายของ share01 ถูกระงับ
- แถว Member: `role = "MEMBER"` · `pt` มีแค่ `commission_percent` · `phone` = `""` เมื่อไม่ได้ตั้ง (ไม่มี `null` — account ACC-32)
- `balances` อ่านจาก `agent_balances` / `member_balances` (MGMT-15A · account ACC-19) · สกุลที่ไม่มียอด และบัญชี Seamless = `0`

Error codes: `402402`

### GET /api/v1/bo/pr/manage/agents/:id · /manage/members/:id

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
    "game": { "pt_from_parent": 90, "pt": 20, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true }
  },
  "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": false },
  "passcode_set": true,
  "last_login_at": "2026-10-05T10:00:00+07:00",
  "last_login_ip": "203.0.113.10",
  "created_at": "2026-10-01T09:00:00+07:00"
}
```
- รูปแบบ `pt` (รวม `pt_from_parent` และ `status` ในกลุ่ม) และ `status_game` (แยกจาก `pt` · ต่อเกม) เหมือน Profile (account ACC-16) · ตัวอย่างนี้ ได้รับ 90 · ถือจาก Member ของตัวเอง 20
- Member: ไม่มี `status_game`, `passcode_set` · `pt.game` มีแค่ `commission_percent`

Error codes: `402402`

### POST /api/v1/bo/pr/manage/agents/:id/info · /manage/members/:id/info

Request: `{ "name": "share01", "phone": "" }` (`""` = ไม่ตั้ง · ส่ง `null` = `422`) · Response: ไม่มี `data` · Error codes: `422`, `402304`, `402402`, `402403`

### POST /api/v1/bo/pr/manage/agents/:id/status · /manage/members/:id/status

Request: `{ "status": "SUSPENDED" }` · Response: ไม่มี `data` · Error codes: `422`, `402304`, `402402`

### POST /api/v1/bo/pr/manage/agents/:id/pt · /manage/members/:id/pt (ผู้สร้างตั้งให้ลูก)

Request (ส่งเฉพาะกลุ่มที่จะแก้ · ในกลุ่มต้องครบ 5 ค่า · ห้าม `null`):
```json
{ "pt": { "game": { "pt_from_parent": 60, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true } } }
```
- `pt_from_parent` = ค่าที่ให้ลูกคนนี้ · ผู้สร้างถือในสายนี้ = ค่าที่ผู้สร้างได้รับ − ค่านี้ (MGMT-22)
- `status` = เปิด / ปิดทั้งกลุ่ม (MGMT-20)
- Member: กลุ่มละ `commission_percent` ค่าเดียว

Response: ไม่มี `data` · Error codes: `422`, `402304`, `402305`, `402306`, `402307`, `402308`, `402309`, `402402`

### POST /api/v1/bo/pr/manage/agents/me/pt (บัญชีตั้งค่าถือของตัวเอง)

Request (ส่งเฉพาะกลุ่มที่จะแก้):
```json
{ "pt": { "game": { "pt": 40 } } }
```
- `pt` = ถือจาก Member ใต้ตัวเองตรงๆ · ไม่เกินค่าที่ตัวเองได้รับ · ส่วนที่เหลือผ่าน Force / Remain แล้วคืน Superadmin
- Company Seamless Master แก้ไม่ได้ (ล็อก 0 — MGMT-19) · sub แก้ให้เจ้าของได้ตามสิทธิ์ `pt` = edit

Response: ไม่มี `data` · Error codes: `422`, `402305`, `402307`

### POST /api/v1/bo/pr/manage/agents/:id/games

Request: `{ "status_game": { "scratch_card": false } }` (ส่งเฉพาะเกมที่จะเปลี่ยน) · Response: ไม่มี `data` ·
Error codes: `422`, `402304`, `402402`

### GET /api/v1/bo/pr/manage/agents/copy-sources

Response `data`: `[{ "id": 12, "username": "share01", "user_type": "SHARE_B2C", "pt": {...}, "status_game": {...} }]` (ลูกตรงฝั่ง agent ทั้งหมด เรียง A→Z)

### Sub

`GET /manage/subaccounts?owner_id=&q=&page=&limit=` — แต่ละแถว และ `GET /manage/subaccounts/:id`:
```json
{
  "id": 30,
  "username": "comp01@staff",
  "name": "staff01",
  "phone": "",
  "status": "ACTIVE",
  "permissions": {
    "dashboard": "view", "account": "off", "member": "edit", "pt": "view", "report": "view",
    "bet_cancel": "off", "payment": "off", "asset": "off", "announcement": "off"
  },
  "created_at": "2026-10-06T12:26:43+07:00",
  "last_login_at": "",
  "last_login_ip": ""
}
```

`POST /manage/subaccounts` (สร้าง):
```json
{
  "name_suffix": "Staff",
  "password": "••••••••",
  "name": "staff01",
  "phone": "",
  "permissions": { "dashboard": "view", "member": "edit", "pt": "view", "report": "view" }
}
```
→ `data`: `{ "id": 30, "username": "comp01@staff" }` · `name_suffix` รับตัวพิมพ์ใหญ่ได้ เก็บเป็นตัวเล็ก · เมนูที่ไม่ส่ง = `off`

`POST /manage/subaccounts/:id` (แก้): `{ "name", "phone", "permissions" }` (แทนทั้งชุด)

`POST /manage/subaccounts/:id/status`: `{ "status": "INACTIVE" }` (`ACTIVE` / `INACTIVE`)

Error codes: `422`, `402311`, `402401`, `402403`, `402404`

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
    status            BOOLEAN     NOT NULL DEFAULT true,   -- เปิด / ปิดทั้งกลุ่ม (ทุกเกมในกลุ่มค่าเดียวกัน — MGMT-20)
    status_game       BOOLEAN     NOT NULL DEFAULT true,   -- เปิด / ปิดทีละเกม
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

CREATE TABLE member_balances (
    member_id  BIGINT      NOT NULL REFERENCES members(id),
    currency   VARCHAR(4)  NOT NULL,
    amount     BIGINT      NOT NULL DEFAULT 0 CHECK (amount >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (member_id, currency)
);

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
```

- Superadmin มีแถว `agent_game_settings` ของตัวเอง (`pt_from_parent_bp = 10000` · ไม่มี Member ตรง) · ADMIN ไม่มี · `pt_bp` = ค่าถือจาก Member ใต้ตัวเอง (MGMT-22) · ผู้สร้างถือในสายของลูก = `pt_from_parent_bp` ของผู้สร้าง − ของลูก (คำนวณตอนคิดเงิน ไม่เก็บ)
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
| ACC-32 | ส่ง `"phone": null` · รายละเอียดของบัญชีที่ไม่มีเบอร์ / ไม่เคย login | `422` · response `"phone": ""` · `"last_login_at": ""` |
| MGMT-18 | ถือ `30.25` (ไม่ลง 0.5%) · commission `1.01` · ถือ `30.123` (ทศนิยมเกิน 2) | `422` |
| MGMT-18 | ได้รับ 80 · ให้ลูก `80.5` · ตั้ง `pt` ของตัวเอง `80.5` | `402305` |
| MGMT-18 | ผู้สร้างมี commission 0.5% ตั้งให้ลูก 0.6% | สำเร็จ |
| MGMT-19 | Company Seamless Master ตั้ง `pt` ของตัวเอง = 1 · ให้ Share Master น้อยกว่าที่ตัวเองได้รับ | `402307` |
| MGMT-23 | แก้ PT ของหลาน (ไม่ใช่ผู้สร้างโดยตรง) | `402304` |
| MGMT-23 | ส่งแค่กลุ่ม `game` | กลุ่มอื่นไม่เปลี่ยน |
| MGMT-22 | Superadmin ให้ comA 90 · comB 80 | comA `pt_from_parent = 90` · comB `= 80` · Superadmin ถือในสาย comA 10 · สาย comB 20
| MGMT-24 | comA ลดค่าที่ให้ share1 จาก 70 เหลือ 55 (share1 ให้ agent1 60 · `pt` 30) | `402306` msg บอกต่ำสุด 60 · ไม่มีแถวไหนเปลี่ยน |
| MGMT-24 | comA ลดค่าที่ให้ share1 เหลือ 60 · เพิ่มค่าที่ให้ share2 | สำเร็จ · share2 และลูกของ share1 ไม่เปลี่ยน |
| MGMT-26 | `parent_id` เป็นบัญชีสายอื่น | `402402` |
| MGMT-26 | Agent ที่มีทั้ง Agent และ Member เป็นลูก | ได้ทั้งสองแบบ `role` ถูกต้อง เรียง A→Z |
| MGMT-27 | `q=ab` | ได้เฉพาะ username ที่มี `ab` |
| MGMT-29 | ดูรายละเอียด | ไม่มี field รหัสผ่าน / passcode / hash |
| MGMT-16 | สร้างด้วย `pt.game.pt_from_parent = 70` | `agent_game_settings` มี 3 แถว (ทุกเกมใน `minigame`) ค่า `pt_from_parent_bp = 7000` · `pt_bp = 7000` (เริ่มถือทั้งหมด — MGMT-22) · รายละเอียดแสดง `pt.game.pt_from_parent = 70` · `pt = 70` และ `status_game` ครบ 3 เกม · รายชื่อแสดง `pt.game.pt = 20` |
| MGMT-30 | Share แก้สถานะ Agent ของ Agent ลูก (ไม่ใช่ผู้สร้างโดยตรง) | `402304` |
| MGMT-30 | LOCKED → ACTIVE โดยผู้สร้าง | สำเร็จ |
| MGMT-31 | Company ระงับ Share | Share และ Agent ใต้ Share `status = SUSPENDED` · Company `status = ACTIVE` |
| MGMT-30 | Share ล็อก Agent ลูก (Share ACTIVE) | Agent `status = LOCKED` · Share ยังใช้งานได้ |
| MGMT-30 | หัวสายของ Agent ถูกระงับ · ผู้สร้างตั้ง Agent เป็น ACTIVE | สำเร็จ · Agent ยังแสดง `status = SUSPENDED` |
| MGMT-43 | เจ้าของตั้ง sub เป็น `INACTIVE` | sub login ได้ · เรียก route อื่นนอก Profile / Report ได้ `401311` · รายชื่อ sub แสดง `INACTIVE` · ตั้งกลับ `ACTIVE` แล้วใช้งานได้ |
| MGMT-45 | ชั้นบนแก้หรือเปลี่ยนสถานะ sub ของบัญชีในสายล่าง | `402404` · ดูรายชื่อ / รายละเอียดได้ |
| MGMT-40 | sub เรียก `POST /manage/subaccounts` | `402311` |
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
| MGMT-20 | ผู้สร้างตั้ง `pt.game.status = false` | ทุกเกมในกลุ่มเล่นไม่ได้ แม้ `status_game` ของเกมเป็น `true` |
| MGMT-52 | sub ของ Superadmin ได้สิทธิ์ `announcement` | `422` |
| MGMT-52 | Superadmin ให้ sub `rate` = `edit` · Company ให้ sub `rate` | สำเร็จ · `422` |

### 7.1 ลำดับเช็ค (เส้นสร้าง)

`422` (รูปแบบ) → `402303` (สิทธิ์ sub) → `402301` (สร้างประเภทนี้ไม่ได้) → `402310` (สกุล) → `402401` / `402403` (ซ้ำ) →
`402305` / `402307` / `402308` / `402309` (ค่าหุ้นส่วน)

## 8. Contract changes (แจ้ง frontend)

- เส้นทั้งหมดในหัวข้อ 5 เป็นเส้นใหม่
- ค่า % ส่งเป็น JSON number (ACC-18) อยู่ใน object `pt` แยกกลุ่ม · ห้าม `null` · ไม่ตั้ง = `0`
- **ไม่มี `null` ใน API** (account ACC-32): ข้อความว่าง = `""` · ตัวเลขว่าง = `0` · request ส่ง `null` = `422`
- สิทธิ์ของ sub ส่งเป็น object `{ "{menu}": "off" | "view" | "edit" }` (รวม Profile — account ACC-12)
- ใช้แค่ method `GET` และ `POST` · ไม่มีเส้นลบ sub (ใช้สถานะ `INACTIVE`)
- เส้นสร้างรับ `request_id` (บังคับ) และ `balance` (ไม่บังคับ — ยอดเงินตั้งต้น)
- account (review รอบ 2): Profile ส่ง `pt` ชุดเดียว (รวม `pt_from_parent` ในกลุ่ม) + `status_game` แยก และ `status` key เดียว — ตรงกับ module นี้ · ข้อมูลอ่านจาก `agent_game_settings`

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
