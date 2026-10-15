# Review BO Agent_Management (ฝั่ง agent) — lead: zerph · 2026-10-09 · ✅ review ครบแล้ว

อ้างอิง: `docs/modules/agent_management.md` (dev `bf501fc`) · เจ้าของ module: boiledegg · ไฟล์นี้คือการตัดสินของ lead — ทำตามรายการแก้ใน branch ใหม่แล้วเปิด PR ให้ lead review

## การตัดสินของ lead
| # | เรื่อง | ตัดสิน |
|---|---|---|
| U1 | ลบค่าถือ `pt` ค่าเดียวของ agent + ลบ `update-hold` (ถือสู้ตั้งต่อ Member — member_management MGMT-21) | **อนุมัติ** (คู่กับ member_management MGMT-21) |
| U2 | MGMT-24 R1–R3 (ค่าต่ำสุดนับ `pt` ของ Member · sync `remain_quota` · ลำดับ lock) | **อนุมัติ** |
| U3 | ลำดับเช็ค 7.1 + ข้อความ error บอก field/ค่าที่ตั้งได้ | **อนุมัติ** + เขียนข้อยกเว้น: สิทธิ์ `payment` (ส่ง `balance` ตอนสร้าง) เช็คใน service หลัง DTO |
| U4 | เปลี่ยนชื่อตาราง Member (`members` → `user_members` …) | **lead สั่งเอง** — รับทราบ |
| H1 | สายการสร้าง (MGMT-02) | **Share Reseller และ Share Master สร้างได้แค่ Member** (สร้าง Agent ไม่ได้ — spec/โค้ดเดิมให้สร้าง Agent ได้ → ต้องแก้) · Share B2B สร้าง Member ตรงไม่ได้ ต้องผ่าน Agent (ถูกต้องตามเดิม) · Share B2C คงเดิม (Agent หรือ Member) |
| H2 | Company Seamless Master (CSM) | **ถือสู้ PT ไม่ได้** · ได้จาก Superadmin เท่าไหร่ ต้องส่งให้ Share Master ทั้งหมด · Share Master มีได้ไม่จำกัด · เหตุผลธุรกิจ: CSM คือผู้ control คนเดียวของลูกค้าที่มีหลายเว็บ (Share Master = 1 เว็บ) ตั้งค่าทุกอย่างที่เดียว Share Master ไม่ต้องตั้งอะไร |
| Q-R1 | CSM ลด PT ไม่ได้ / เพิ่มแล้วผิด MGMT-19 | **ทางเลือก 1:** Superadmin แก้ PT ของ CSM → ระบบตั้ง PT ของ Share Master ทุกคนใต้ CSM ให้เท่ากันใน tx เดียว (ข้อยกเว้น MGMT-24 "ระบบไม่แก้ค่าลูกให้เอง" เฉพาะคู่นี้) · ลด: ค่าใหม่ต้อง ≥ `pt` ที่ Share Master ถือสู้กับ Member ทุกคน ไม่งั้น `402306` · เพิ่ม: ตามทันที + `remain_quota` ของ Member คำนวณใหม่ (R2) · CSM / Share Master แก้ PT ของ Share Master เองไม่ได้ · สร้าง Share Master = PT เท่า CSM อัตโนมัติ (ไม่ต้องกรอก) · บันทึก `account_change_logs` ของ Share Master ว่าระบบปรับตาม CSM |
| H3 | ค่าอื่นของ Share Master | **ตามแนะนำ:** PT · Force/Remain (= 0) · เปิด/ปิดเกม (`status_game`) **ตาม CSM อัตโนมัติ** · **Commission** CSM ตั้งให้แต่ละ Share Master แยกกันได้ (ตาม MGMT-19 เดิม) |
| H4 | `pt` ที่ Share Master ถือสู้กับ Member | **(ก) Share Master ตั้งเองต่อ Member** (ตามกฎปัจจุบัน member_management MGMT-21) |
| H5 | Company Seamless Reseller vs Master | **ยืนยัน:** Company Seamless Reseller ถือสู้ PT ได้ และตั้งค่าให้ Share Reseller แต่ละคนแยกกันได้ (กฎทั่วไป MGMT-22) |
| Q-R2 | `pt.created_by` เมื่อ sub สร้าง | **ชื่อคนกดจริง** (`comp01@staff`) แบบเดียวกับ `updated_by` (ตามโค้ด) · เขียนใน MGMT-16 |
| Q-R3 | PT ของบัญชีที่ `pt.status = false` ไปอยู่กับใคร | **เลื่อนไปตัดสินพร้อม module Bet/Settle** — open item ต้องตัดสินก่อนเริ่ม settle |
| Q-R4 | `request_id` ที่คนอื่น/เส้นอื่นใช้แล้ว → `422 "request_id นี้ถูกใช้แล้ว"` | **เขียนเพิ่มใน spec ตามโค้ด** |
| Q-P1 | เมนู `api_credential` | **เพิ่มเป็นเมนูที่ 9** ใน MGMT-50–53 + เขียนกฎ A5 (sub ต้องมีสิทธิ์ทุกเส้น ยกเว้น Profile + auth ตัวเอง) ใน MGMT-51 |
| Q-P2 | เมนู `dashboard` ในสิทธิ์ | **คงไว้** |
| Q-C1 | `status_game` ในแถวฝั่ง agent ของ `downlines/list` / `search` | **เพิ่มในโค้ด** + MGMT-28 |
| Q-C2 | แถว Member ในรายชื่อ (`pt` + `remain_quota` + `commission_percent`) | **ตามโค้ด** · แก้ตัวอย่างใน spec |
| Q-C3 | ข้อความค้างเรื่อง "ค่าถือ `pt` ของ agent" ใน spec | **boiledegg ลบทั้งหมดใน PR เดียวกัน** |

## โครง endpoint ใหม่ (lead ตัดสิน 2026-10-09) — แทน 9 เส้นเดิม
| # | เส้นใหม่ | แทนเส้นเดิม | หน้าที่ / กฎ |
|---|---|---|---|
| E1 | `POST /manage/agents/create` | `agents/create` | สร้าง Company / Share / Agent |
| E2 | `POST /manage/downlines/list` | `downlines/list` + **รวม** `downlines/search` | ไล่ดูสายทีละชั้น · **field ไม่บังคับ `keyword`** (username): ส่ง = ค้นทุกชั้นใต้ตัวเอง · ไม่ส่ง = ลูกตรงตามเดิม |
| E3 | `POST /manage/agents/detail/get` | `agents/detail` | ดูรายละเอียดบัญชี |
| E4 | `POST /manage/agents/detail/update` | **รวม** `update-info` + `update-pt` + `update-games` | section `info` / `pt` / `games` **ไม่บังคับทุก section** ส่งเฉพาะที่แก้ (ไม่ส่ง = ไม่เปลี่ยน) · **สิทธิ์ sub: `member` edit อย่างเดียว** (ทุก section) · tx เดียว — section ไหนผิด ไม่บันทึกเลยสักส่วน · `account_change_logs` แยกแถวตาม section ที่เปลี่ยน · **ไม่ต้อง passcode** |
| E5 | `POST /manage/agents/status/update` | `update-status` | ระงับ / ล็อก / ปลด — แยกเส้นเพราะมีผลทั้งสายล่าง |
| E6 | `POST /manage/agents/list` (ชื่อรอยืนยัน) | แทน `copy-sources` | list `user_agents` ที่เป็น**ลูกตรงของตัวเอง** ไว้ให้หน้าบ้านทำ dropdown → เลือกแล้วเรียก `agents/detail/get` เอาค่าคนนั้นมาตั้งต้นฟอร์มสร้าง agent ใหม่ |
| N1 | รูปแบบชื่อเส้น `…/detail/get` · `…/detail/update` · `…/status/update` | — | **ใช้กับ member_management และ subaccount ด้วย** |
| P3 | สิทธิ์ `agents/create` | sub ต้องมี **`member` edit อย่างเดียว** (+ `payment` edit เมื่อส่ง `balance`) — เดิม `member` edit + `pt` edit |
| P4 | เมนู `pt` ในรายการสิทธิ์ | **ลบออก** (ไม่มีเส้นใช้แล้ว — `member` คุมทั้งข้อมูลบัญชีและ PT) · เมนูเหลือ: dashboard · member · report · bet_cancel · payment · asset · announcement · api_credential · ต้องมี migration ล้างสิทธิ์ `pt` ของ sub เดิม |
| P5 | สิทธิ์ E2 / E3 / E6 | sub ต้องมี **`member` view** ทั้ง 3 เส้น |
| E6b | response ของ `agents/list` | `id` · `username` · `name` · `user_type` · `status` · ไม่มี pagination · สูงสุด 500 รายการ |
| E1-1 | `agents/create` โดย CSM (สร้าง Share Master) | ส่งแค่ `pt.minigame.commission_percent` · `pt_from_parent` / `force` / `remain_quota` / `status` ระบบตั้งตาม CSM · `status_game` ห้ามส่ง · ส่ง field ที่ระบบตั้งเอง = `422` "Share Master ใช้ค่าตาม Company Seamless Master" |
| E1-2 | ข้อความ error | `402305` → "ค่าที่ให้ลูกเกินกว่าที่ได้รับ" · `402307` → "Share Master ใช้ค่า PT ตาม Company Seamless Master" |
| E1-3 | 7.1 vs โค้ด (Force/Remain ใต้ CSM) | แก้ spec ตาม E1-1 |
| E1-4 | log `INITIAL_BALANCE` key `amounts_minor` | เปลี่ยนเป็น `amounts` |
| E1-5 | `phone` + รหัสประเทศ | **แบบ A:** แยก `phone_country_code` (ตัวเลข 1–3 หลัก รหัสโทรออก เช่น `66`) + `phone` (เบอร์ในประเทศ ไม่มี 0 นำหน้า ตัวเลขล้วน รวมกันไม่เกิน 15 หลัก — E.164) · บังคับ code เมื่อส่ง phone · ไม่กรอก = `""` ทั้งคู่ · ห้ามซ้ำที่คู่ (code + phone) · **ใช้ทุกที่: agent · sub · Member** · migration แปลงเบอร์เดิมที่ขึ้นต้น 0 → `66` + ตัด 0 · รับรหัสโทรออกที่มีจริงทุกประเทศ (รายการในโค้ด) |
| E2-1 | `downlines/list` request | `parent_id` ไม่ส่ง / 0 = ดึงภายใต้ตัวเอง · `keyword` (แทน `q`) ไม่ส่ง/`""` = ลูกตรงของ `parent_id` · ส่ง = ค้นทุกชั้นใต้ `parent_id` · **`keyword` 4–32 ตัว** · **ค้นเฉพาะ username** (บางส่วน ไม่สนตัวพิมพ์) |
| E2-2 | `downlines/list` response | ทุกแถวมี `parent_username` · แถว agent มี `status_game` (Q-C1) · เบอร์แบบ E1-5 · `pt` แสดงเสมอ (เมนู `pt` ถูกลบ — P4) · ลบข้อความเก่า "balances ส่ง 0 ระหว่างยังไม่มี payment" · error เพิ่ม `422` (keyword) · ลบเส้น `downlines/search` |
| E3-1 | `agents/detail/get` | path ใหม่ (N1) · เบอร์ E1-5 · `pt` แสดงเสมอ (P4) · ลบหมายเหตุ "ถือจาก Member ของตัวเอง 30" (U1) · **เพิ่ม `status_game_effective`** (รวมผลหัวสาย — ไว้แสดงผล) คง `status_game` (ค่าของบัญชีเอง — ไว้ฟอร์มแก้) · ดูได้ทุกบัญชีในสายล่างคงเดิม |
| E4-1 | `agents/detail/update` | **แก้ได้เฉพาะลูกตรงของตัวเอง** (หลาน = `402304` · นอกสาย = `402402` · ตัวเอง = `402304`) · section เกมชื่อ `status_game` · `info` ส่งบาง field ได้ (เบอร์ต้องคู่) · `pt.minigame` ต้องครบ 5 ค่า · ค่าเดิมทั้งหมด = 200 ไม่เขียนประวัติของ section ที่ไม่เปลี่ยน · ลูกไม่มีแถวค่าตั้ง = ข้ามเงียบคงเดิม · ใครปิดเกมดูจากประวัติ (คงเดิม) |
| E4-2 | ระบบปรับชั้นล่างเอง | **คงไว้**: Superadmin แก้ CSM → ระบบปรับ Share Master ตาม (Q-R1/H3) · แก้ PT ลูก → ระบบคำนวณ `remain_quota` ของ Member ใต้ลูก (R2) — request ยังส่งที่ลูกตรงเท่านั้น |
| E5-1 | `agents/status/update` | path ใหม่ (N1) · เขียนใน spec: แก้สถานะตัวเอง = `402304` · `LOCKED` มีผลตั้งแต่ request ถัดไป · ปลดลูกไม่ปลดสายล่างถ้าหัวสายสูงกว่ายังถูกระงับ |
| E6-1 | `agents/list` | **`POST`** · request `{}` + **`keyword` ไม่บังคับ 4–32 ตัว ค้น username** · response `id` `username` `name` `user_type` `status` · A→Z · สูงสุด 500 · ไม่มีลูก = `[]` · error `402303` |

## รายการแก้ (ส่งให้ทีม — ทำใน branch ใหม่แล้วเปิด PR ให้ lead review)

### Spec `docs/modules/agent_management.md`
- S1 หัวเอกสาร: บันทึกการอนุมัติ U1–U4 + การตัดสินทั้งหมดในไฟล์นี้ (lead zerph 2026-10-09)
- S2 **หัวข้อ 5 Endpoints เขียนใหม่ตามโครง E1–E6:** `agents/create` · `downlines/list` (รวม search, `keyword`) · `agents/detail/get` · `agents/detail/update` (รวม info/pt/status_game) · `agents/status/update` · `agents/list` · ลบ `downlines/search` `copy-sources` `update-info` `update-pt` `update-games`
- S3 MGMT-02: Share Reseller / Share Master สร้างได้แค่ Member (H1)
- S4 MGMT-19 + MGMT-24: CSM ถือสู้ไม่ได้ · ระบบปรับ Share Master ตาม CSM (PT · Force/Remain = 0 · status_game) ใน tx เดียว · Commission ตั้งแยกต่อ Share Master · ตอนสร้าง Share Master ส่งแค่ `commission_percent` (Q-R1, H2, H3, E1-1, E4-2)
- S5 MGMT-08 + ทุกตัวอย่าง: เบอร์ `phone_country_code` + `phone` (E1-5)
- S6 MGMT-50–53: เพิ่มเมนู `api_credential` · **ลบเมนู `pt`** · คง `dashboard` · กฎ A5 (sub ต้องมีสิทธิ์ทุกเส้น ยกเว้น Profile + auth ตัวเอง) · สิทธิ์ทุกเส้นของ module = `member` view/edit (+ `payment` edit เมื่อส่ง `balance`) (Q-P1, Q-P2, P3–P5)
- S7 MGMT-16 `created_by` = คนกดจริง (Q-R2) · MGMT-15A `request_id` ของคนอื่น/เส้นอื่น = `422` (Q-R4) · 7.1 ข้อยกเว้นสิทธิ์ `payment` เช็คหลัง DTO (U3) · MGMT-30 แก้ของตัวเอง = `402304` (E5-1)
- S8 MGMT-28 + ตัวอย่างรายชื่อ: `parent_username` ทุกแถว · `status_game` แถว agent · แถว Member (`pt` + `remain_quota` + `commission_percent`) (Q-C1, Q-C2, E2-2) · detail เพิ่ม `status_game_effective` (E3-1)
- S9 Open item: PT ของบัญชี `pt.status = false` ไปอยู่กับใคร — ตัดสินก่อนเริ่ม settle (Q-R3)
- S10 ล้างข้อความค้างเรื่อง "ค่าถือ `pt` ของ agent" + schema/ตัวอย่างเก่า (บรรทัด 42, 112, 444, 528, 833, 835, 717–729, 746, 155 และข้อความ error 402305/402307 ในหัวเอกสาร) (Q-C3)

### Spec module อื่น
- S11 `member_management.md`: path `members/detail/get` · `members/detail/update` · `members/status/update` (N1) · เบอร์ E1-5 · สิทธิ์ `member` (ลบ `pt`) · Share Reseller/Master สร้าง Member ได้ (H1)
- S12 subaccount (ใน agent_management): path รูปแบบเดียวกัน (N1) · เบอร์ E1-5 · รายการเมนูสิทธิ์ใหม่
- S13 `account.md`: Profile `permissions` ไม่มี `pt` แล้ว มี `api_credential`

### โค้ด
- C1 routes + controllers ตามโครง E1–E6 · ลบเส้นเดิม (ไม่ต้องคง alias — ยังไม่มี prod)
- C2 `agents/detail/update`: section ไม่บังคับ · tx เดียว · log แยกแถวต่อ section ที่เปลี่ยนจริง · ค่าเดิม = ไม่เขียน log · ลูกตรงเท่านั้น
- C3 `downlines/list`: `keyword` 4–32 ตัว ค้น username ทุกชั้นใต้ `parent_id` · `parent_username` · `status_game` (`DownlineRow`)
- C4 `agents/list` (ใหม่) + `keyword` · ลบ `copy-sources`
- C5 MGMT-02: Share Reseller / Share Master สร้าง Agent ไม่ได้ (`402301`)
- C6 CSM → Share Master: สร้าง (ส่งแค่ commission) · แก้ PT/status_game ของ CSM ปรับ Share Master ทุกคนใน tx (เช็คค่าต่ำสุดจาก `pt` ของ Member ใต้ Share Master) · log ระบบปรับ · ห้ามแก้ PT/เกมของ Share Master ตรง (`422`)
- C7 เบอร์: `phone_country_code` + `phone` ทุกตาราง (agent · sub · Member) · unique ที่คู่ · migration แปลงเบอร์เดิม (`0…` → `66` + ตัด 0) · รายการรหัสโทรออกในโค้ด
- C8 สิทธิ์: เพิ่ม `MenuAPICredential` · ลบเมนู `pt` (migration ล้างสิทธิ์ `pt` ของ sub เดิม) · ทุกเส้น module = `member`
- C9 E3 `status_game_effective` (ใช้ฟังก์ชันทั้งสายใน `pt_rule.go:174`)
- C10 `pkg/apperr/agent_management_errors.go`: ข้อความ `402305` / `402307` (E1-2) · `create_agent_service.go:154` `amounts_minor` → `amounts` (E1-4)
- C11 test ทุกข้อข้างบน + แก้ test เดิมที่อ้างเส้นที่ลบ · `go run scripts/…`/lint/integration ผ่าน · แจ้ง frontend (contract เปลี่ยนเยอะ — path, เบอร์, สิทธิ์)

## คำถามจาก boiledegg ก่อนเริ่มงาน (2026-10-09) — lead ตอบ
| # | คำถาม | lead ตัดสิน |
|---|---|---|
| B1 | migration เบอร์เดิม (E1-5) ที่ไม่ได้ขึ้นต้น 0 | ขึ้นต้น `0` → `66` + ตัด 0 · ขึ้นต้น `66` และยาว 11 หลัก → `66` + ส่วนที่เหลือ · **แบบอื่นทั้งหมด → ล้างเป็น `""`** (ไม่เดาประเทศ) · migration log จำนวน/รายการที่ถูกล้าง |
| B2 | CSM แก้ Share Master ผ่าน `agents/detail/update` | ใช่ — `pt.minigame` ส่งได้แค่ `commission_percent` · ส่ง `status_game` = `422` · section `info` (ชื่อ/เบอร์) แก้ได้ปกติ · Superadmin แก้ PT/`status_game` ของ CSM → ระบบปรับ Share Master ทุกคนตาม (Q-R1) |
| B3 | ลบเมนู `pt` กระทบเส้น member_management (`members/create`, `update-pt`) | ได้ — **เปลี่ยนแค่สิทธิ์ใน `routes.go` เป็น `member` edit** (ไม่งั้น build ไม่ผ่าน) · **ยังไม่เปลี่ยน path ของ member** (N1) — module ของ maofoy และยังไม่ได้ review · แจ้ง maofoy |

## คำถามจาก boiledegg ระหว่างทำ (2026-10-10) — lead ตอบ
| # | คำถาม | lead ตัดสิน |
|---|---|---|
| B4 | `agents/detail/update` ไม่ส่ง section เลย (`{ "id": 12 }`) | **`422` "ต้องส่งอย่างน้อย 1 section"** (= MN2 ของ member-management) |
| B5 | action ของ log ตอนระบบปรับ Share Master ตาม CSM | **`SYNC_FROM_CSM`** (แยกจากการแก้ที่คนกด · ผู้ทำ = คนที่แก้ CSM) · เก็บ id ของ CSM + `request_id` เดียวกับการแก้ CSM ใน log |
| B6 | เบอร์ agent ที่แปลงแล้วซ้ำกัน (เช่น `0812345678` กับ `66812345678`) | **บัญชี id น้อยสุดเก็บเบอร์ไว้** · บัญชีอื่นล้างเป็น `""` · บันทึกใน `migration_phone_cleared` พร้อมเหตุผล "ซ้ำกับบัญชี id X" |

## ตรวจงานรอบที่ 1 (2026-10-10 · commit 8325515 + c9cc1a5 + 7bd675c)
ผลตรวจเต็ม: E1–E6, N1, P3–P5, H1, Q-R1/H2/H3, B1–B4, C1–C10 ✅ · เส้นเดิมถูกลบหมด · CSM → Share Master ปรับใน tx เดียว + sync `remain_quota` + log `SYNC_FROM_CSM` ✅ · build / vet / unit / check-structure ผ่าน · **integration ยังไม่ได้รัน**

| # | เรื่อง | lead ตัดสิน |
|---|---|---|
| V1 | `agents/status/update` ส่งสถานะเดิมยังเขียน log (`update_service.go:158`) — MQ5 ตัดสินทีหลัง | **แก้: ค่าเดิม = 200 ไม่เขียน log** · ทำพร้อมงาน member ของ maofoy (DTO ร่วม) |
| V2 | migration `20261009170000_…_share_master_follow_csm.sql:100-105` ข้ามแถวที่ `pt_from_parent` ใหม่ < `pt` ของ Member แบบเงียบ | **บันทึกแถวพวกนี้ลงตาราง log** (แบบ `migration_phone_cleared`) ให้มีคนตาม · ไม่ต้องให้ migration ล้ม (ยังไม่มี prod) |
| V3 | Agent เก่าใต้ Share Reseller / Share Master (สร้างก่อน H1) — ตอนลด PT ของ CSM ไม่ได้เช็คชั้นนี้ | **รัน query ตรวจบน dev ก่อน** · ไม่มี = จบ · มี (ข้อมูลทดสอบ) = ลบหรือย้ายทิ้ง |
| V4 | เบอร์ของ sub ซ้ำได้ (MGMT-41) vs E1-5 "ใช้ทุกที่" | **sub ซ้ำได้ตามเดิม** · ใช้รูปแบบ `phone_country_code` + `phone` เหมือนกัน (E1-5 "ห้ามซ้ำที่คู่" ใช้กับ agent และ Member ใต้ parent เดียวกัน — MQ1) |
| V5 | test ที่ยังขาด | เพิ่ม: Share Master สร้าง Agent = `402301` (integration) · migration เบอร์ · ปรับ Share Master ตาม CSM · ยิงพร้อมกัน · B5 `request_id` + `csm_id` · **รัน integration ให้ผ่าน แนบผล** ก่อนส่ง tester |
| V6 | B5 log `SYNC_FROM_CSM` ยังไม่เก็บ id ของ CSM | ใส่ `csm_id` ใน `new_value` (`update_service.go:302-307`, `:478`) + test |
| V7 | B6 `migration_phone_cleared` reason แค่ `DUPLICATE` (VARCHAR(20)) ไม่บอกว่าซ้ำกับใคร | **อนุญาตให้แก้ไฟล์ migration เดิม** (ยังไม่มี prod — เหมือน MA6): เพิ่มคอลัมน์ `kept_agent_id` (บัญชีที่ได้เก็บเบอร์) · reset DB ของ dev แล้วรันใหม่ · **แจ้งทุกคนที่มี DB dev ในเครื่องให้ reset** |

**ประสานกับ maofoy** (งาน Member): SQL ของ `downlines/list` สำหรับเบอร์ Member 2 field (`agent_management_repository.go:178,274`) · `UpdateInfoRequest` / `CheckPhoneFree` ที่ใช้ร่วมกัน · `CanSeePT` shim · `amounts_minor` ฝั่ง Member — รายละเอียดใน verify report หัวข้อ 5
