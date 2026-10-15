# Review member_management หัวข้อ 9 (DRAFT 2026-10-12) — วิธีจ่ายเงิน · บัญชีธนาคาร · ลิมิตชนะต่อวัน — lead: zerph · 2026-10-12 · ✅ ตัดสินแล้ว

อ้างอิง: `docs/modules/member_management.md` หัวข้อ 9 (dev `8ee3f73` · maofoy `c734a2a` + `5388f72`) · เจ้าของ: **maofoy** · ไฟล์นี้คือการตัดสินของ lead — แก้ spec หัวข้อ 9 ให้เป็นกฎปกติ (ลบคำว่า DRAFT / ❓) แล้ว implement ใน branch ใหม่จาก dev และเปิด PR ให้ lead review (boiledegg review)
ข้อที่ maofoy ตัดสินแล้ว (✅ ใน DRAFT) **lead อนุมัติทั้งหมด**

## การตัดสินของ lead
| # | เรื่อง | ตัดสิน |
|---|---|---|
| P1 | ความหมาย `CREDIT` / `CASH` | **ข้อมูลแสดงผลอย่างเดียว** — ไม่มีผลกับยอดเงิน การเติม/ถอน หรือการแทง · ฝั่ง Seamless เก็บ `CASH` เสมอ (ตามที่ maofoy ตัดสิน) |
| P2 | `payment_type` ของ Member เดิม | `CASH` (migration default) |
| P3 | `bank_code` + `account_number` | ต้องส่งเป็นคู่ · ส่งค่าเดียว = `422` (แบบเดียวกับเบอร์) |
| P4 | นับเลขบัญชีซ้ำ | ที่คู่ (`bank_code` + `account_number`) ภายใต้ผู้สร้างเดียวกัน · ซ้ำ = `402405` |
| P5 | รายการธนาคาร | **ธนาคารไทยอย่างเดียว** · รายการคงที่ในโค้ด (เช่น KBANK · SCB · BBL · KTB · BAY · TTB · GSB · BAAC · GHB · KKP · CIMBT · UOBT · LHBANK · TISCO · ICBCT · ISBT ฯลฯ — ให้ maofoy ทำรายการเต็มใน spec) · รหัสไม่อยู่ในรายการ = `422` |
| P6 | รูปแบบ `account_number` | ตัวเลขล้วน 10–15 หลัก |
| P7 | ชื่อเจ้าของบัญชี 6 field | `_th` เฉพาะอักษรไทย · `_en` เฉพาะ A–Z a–z · มีช่องว่าง / `-` ได้ · ≤ 50 ตัว · ตัดช่องว่างหัวท้าย · ส่งชื่อโดยไม่มี `bank_code` + `account_number` = `422` |
| P8 | การแสดงเลขบัญชี | เต็มใน `members/detail/get` · ไม่อยู่ใน `downlines/list` · **`account_change_logs` ปิดบางส่วน** (เช่น `787•••6654` — 3 ตัวแรก + 4 ตัวท้าย) เพราะเป็นข้อมูลส่วนตัว |
| P9 | `win_limit_per_day` ไม่ส่ง | ไม่บังคับ · ไม่ส่ง = `0` (ไม่จำกัด) |
| P10 | วิธีนับ / ตัดรอบวัน / ผลเมื่อถึงลิมิต | รอบนี้แค่เก็บค่า → **open item ของ module เดิมพัน** |
| P11 | เพดานจากชั้นบน | ไม่มี |
| P12 | `members/detail/get` | เพิ่ม `payment_type` · `payment_account` (ไม่มี = object ที่ทุก field เป็น `""` — ACC-32) · `win_limit_per_day` |
| P13 | `members/detail/update` | section ใหม่ `payment` (`payment_type` + `payment_account`) และ `limit` (`win_limit_per_day`) · ผู้สร้างโดยตรงเท่านั้น · tx เดียว · log `UPDATE_PAYMENT` / `UPDATE_LIMIT` แยกแถว · ค่าเดิมไม่เขียน log |
| P14 | สิทธิ์ sub | section `payment` ต้องมี `payment` edit เพิ่ม · `limit` ใช้ `member` edit |
| P15 | `downlines/list` / คัดลอกค่าตั้งต้น | `downlines/list` ไม่เพิ่ม · หน้าบ้านคัดลอก `pt` + `win_limit_per_day` · **ไม่คัดลอก** `payment_account` |

## รายการงาน (maofoy)
- S1 spec หัวข้อ 9 → กฎปกติ (MGMT-63/64/65) ตาม P1–P15 · รายการธนาคารเต็ม · ตัวอย่าง request/response ของ `members/create` · `members/detail/get` · `members/detail/update` (section `payment` / `limit`) · ลำดับเช็ค (หัวข้อ 2.1) · ERROR_CODES จอง `402405` · `docs/frontend/member_management.md`
- C1 migration ใหม่ (timestamp หลังไฟล์ล่าสุดบน dev) ตาม 9.3 + unique ที่คู่ใต้ผู้สร้าง
- C2 DTO / service / repository: create + detail/get + detail/update (2 section ใหม่) · สิทธิ์ `payment` · lock `FOR NO KEY UPDATE` · unique violation → `402405`
- C3 log ปิดบางส่วนของเลขบัญชี (P8)
- C4 test: คู่ไม่ครบ 422 · ธนาคารไม่อยู่ในรายการ 422 · ซ้ำใต้ผู้สร้างเดียวกัน 402405 · คนละผู้สร้างได้ · Seamless = CASH · sub ไม่มี `payment` edit แก้ section payment ไม่ได้ · log ปิดบางส่วน · ค่าเดิมไม่เขียน log · รัน integration แนบผล · แจ้ง frontend

## ตรวจงานรอบที่ 1 (2026-10-12 · maofoy `faa5ea5` `0e23819` `0e6c1cc` `e982a65` `4527bb0` `ed0cd62` `12922d9` `6d6b95a`)
ผลตรวจ: P1–P15 ทำครบ ✅ · migration · unique → `402405` · เลขบัญชีใน log ปิดบางส่วน · สิทธิ์ `payment` · Seamless = CASH · tx / lock ✅ · build / vet / unit / structure ผ่าน · **integration ยังไม่มีใครรัน** · merge ตรงเข้า dev ไม่ผ่าน PR · **ยังไม่ deploy dev จนกว่าจะทำรายการด้านล่างครบ**

| # | เรื่อง | lead ตัดสิน |
|---|---|---|
| Q1 | `win_limit_per_day` ≤ 1,000,000 (maofoy เพิ่มเอง) | **คงเพดาน 1,000,000 บาท (THB)** — ระบบใช้ THB เป็นค่ากลาง · ตอนตรวจให้**แปลงค่าของ Member เป็นบาทก่อนแล้วค่อยเทียบ** (ไม่ใช่เทียบตัวเลขดิบในสกุลของ Member) · ต้องใช้อัตราแลกเปลี่ยน — module `currency_rate` ยังอยู่ใน branch `boiledegg/bo/currency-rate` ไม่อยู่บน dev → maofoy ประสาน boiledegg ว่าจะใช้อัตราจากไหน (ถ้ายังไม่มีอัตราของสกุลนั้น = ปฏิเสธพร้อม error ที่ชัดเจน ไม่ใช่ผ่านเงียบ) · Member สกุล THB เทียบตรงได้เลย |
| Q2 | request เงิน / % เป็น `float64` แทน `utils.Decimal` (ทีมตกลง) | **อนุมัติ** · ผลข้างเคียง: ตัวเลขที่ส่งเป็นข้อความ = 400 (เดิม 422) — แจ้งหน้าบ้าน |
| Q3 | sub ที่มีแค่ `member` edit ใส่บัญชีธนาคารตอน create ได้ | **ปล่อยตามเดิม** (การแก้ทีหลังยังต้องมี `payment` edit) |
| Q4 | รายการธนาคาร 17 แห่ง | **พอแล้ว** (เพิ่มทีหลังได้) |
| Q5 | เลขบัญชีเต็มเห็นได้ทั้งสายชั้นบน | **ตาม P8 เดิม** (เต็มใน `detail/get`) |
| Q6 | msg เป็น label แทน path ของ field | **รับ** · code ไม่เปลี่ยน · ถามหน้าบ้านก่อนว่าไม่ได้ parse path จาก msg |

**maofoy ต้องทำก่อน deploy dev:** 1) Q1 แปลงเป็นบาทก่อนเทียบ 2) `payment_type` ผิดตอน create ได้ `422` เหมือน update 3) รัน integration แนบผล 4) แจ้งหน้าบ้าน: `payment_type` บังคับ · ตัวเลขเป็นข้อความ = 400 · msg เปลี่ยนเป็น label · ต่อไปขอให้เปิด PR ให้ boiledegg review แทน commit ตรง dev

## รอบ 2 — `win_limit_per_day` ที่ maofoy เปลี่ยน (MGMT-65 · migration `20261014160000`) · lead zerph 2026-10-15
| เรื่อง | ผล |
|---|---|
| `0` = เล่นไม่ได้ | **รับ** (แทน P9 เดิม `0` = ไม่จำกัด) |
| ไม่ส่ง = ค่าเพดาน | **รับ** · Member เดิมที่เป็น `0` ปรับเป็นค่าเพดานตาม migration |
| เพดาน 10,000,000 ตัวเลขเดียวทุกสกุล | **ไม่รับ** → กลับเป็นตาม Q1 รอบ 1: **เพดาน 1,000,000 บาท · แปลงสกุลของ Member เป็นบาทก่อนเทียบ** (ใช้ currency_rate) · ค่า default เมื่อไม่ส่ง = เพดานนี้ในสกุลของ Member |

maofoy: แก้ MGMT-65 + validation + migration ให้ตรง (migration ที่ merge แล้วห้ามแก้ — เพิ่มไฟล์ใหม่) · แจ้งหน้าบ้าน · Player Auth (`can_bet` / ผู้เล่นที่ระบบสร้าง) ใช้ค่าเพดานนี้
