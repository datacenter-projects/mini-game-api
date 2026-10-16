# Lead review: การคิด PT ต่อ Member (`pt_calculation`) — zerph · 2026-10-15

- spec: `docs/modules/pt_calculation.md` (DRAFT) · branch `maofoy/utils/pt-calculate` @ `b49b8e4` · ผู้ทำ: Maofoy
- ผลรวม: **อนุมัติ spec ได้หลังแก้ตาม Q2 / Q3** → เริ่ม implement ผ่าน PR (Boiledegg review)
- module นี้คือ input ของ bet core / settle (lead ทำเอง) — ทำให้เสร็จก่อน lead เริ่ม settle

| # | เรื่อง | ผล |
|---|---|---|
| Q1 | ใช้กฎใน `docs/references/pt-force-remain-commission.md` เป็น rule จริง (ไล่ล่างขึ้นบน · Force ก่อน Remain · Force = ชั้นนั้นถือเพิ่ม · Remain = ผู้สร้างถือเพิ่ม · เหลือคืน Superadmin) และขยายเป็น N ชั้น | **อนุมัติ** (reference แปลงจาก diagram ของ lead) · N ชั้นจำเป็น (Agent ซ้อน · Company 1 to 1 สร้าง Member ตรง) |
| Q2 | ชั้นที่ `pt.status = false` — PT ส่วนนั้นไปอยู่กับใคร (ปิด open item MGMT-20 / lead Q-R3) | **ชั้นนั้น `hold = 0` และไม่โดน Force ของตัวเอง** · ส่วนที่ชั้นนั้นจะถือกลายเป็นส่วนที่ไม่มีใครถือ (`left`) → ไหลตามกฎปกติ: Force / Remain ของชั้นอื่นรับก่อน ไม่มีก็คืน Superadmin (MGMT-22) · Remain ที่ชั้นนั้นตั้งให้ลูกยังทำงาน · **ไม่ใช้ `408301`** (ตัดออก) · เพิ่ม test case ชั้นกลาง `pt.status = false` |
| Q3 | CSM ถือสู้ไม่ได้ (MGMT-19) แต่ Superadmin ตั้ง `force` ให้ CSM ได้ (MGMT-18A) | **ข) ห้ามตั้ง `force` ให้ CSM** — แนวเดียวกับ Remain ที่ Superadmin ตั้งไม่ได้ (MGMT-18A 2026-10-14) · แก้ agent_management: ส่ง `force` ≠ 0 ให้ CSM = `422` + migration ใหม่ตั้ง `force` ของ CSM เดิมเป็น `0` (ห้ามแก้ migration ที่ merge แล้ว) · `pt_calculation` ข้าม Force ของ CSM ไว้กันพลาดด้วย (log warn ถ้าเจอค่า ≠ 0) |
| Q4 | ไม่มีกฎ Force ≤ Remain (Force ≥ Remain แล้ว Remain ไม่มีผล) | **รับทราบ — ไม่ validate** (ตัวเลขไม่ผิด แค่ Remain ไม่มีผล) |
| Q5 | เส้น dev `POST /api/v1/bo/pr/dev/pt/calculate` ADMIN เท่านั้น · เปิดด้วย `DEV_ROUTES_ENABLED` · `APP_ENV=prod` + เปิด flag = boot ไม่ผ่าน | **อนุมัติ** |

Maofoy: แก้ spec ตาม Q2 / Q3 (รวมแก้ MGMT-18A / MGMT-20 ใน agent_management ให้ตรง) · ใส่ "อนุมัติโดย: zerph · 2026-10-15" · implement ผ่าน PR

## รอบ 2 — ผลตรวจโค้ด (`2e7d7bc` + `910bd38` บน dev · lead zerph 2026-10-15 · รายงาน verify-pt-calculation)
ตรรกะถูก: ชุด A–F + เคสคิดมือเพิ่ม 8 เคส ตรงกับโค้ดทุกเคส · Q1–Q5 ✅ · query 3 ครั้ง · CSM force `422` + migration `20261015150000` ปลอดภัย

| # | เรื่อง | ผล |
|---|---|---|
| M1 | **integration test (pt_calculation · CSM force · migration) ยังไม่เคยรัน** — merge เข้า dev ตรงไม่ผ่าน PR (CI รันแค่ PR / release) | **ต้องรันให้ผ่านและแนบผล** ก่อน lead เริ่ม bet core |
| S1 | service ไม่ log `508601` + ค่าทั้งสายเอง (ถ้า module เดิมพันห่อ error log จะหาย) | **แก้** — log ที่ service พร้อมค่าทั้งสาย |
| S2 | integration ขาดชุด C / D / E2 + test นับ query ≤ 3 | **เพิ่ม** |
| L1 | Member `pt = 0` แต่ผู้สร้างมี Force → ผู้สร้างถูกบังคับถือตาม Force | **ตั้งใจ** — Force = ชั้นบนบังคับให้ถืออย่างน้อยเท่านี้ ตั้ง pt 0 ให้ Member หลบไม่ได้ |
| L2 | Superadmin ปิด `pt.status` ของ CSM → sync ไป Share Master → ทั้งสายถือ 0 · PT คืน Superadmin ทั้งหมด | **ตั้งใจ** (ตรง Q2) |

### ย้ายเส้น dev ออกจาก backoffice (แทน Q5 / PTC-40 – PTC-43 ส่วนที่ขัด)
- **context ใหม่ `app/internals/developer/`** (`routes.go` + `controllers/{module}` + `dto/{module}`) · `pkg/routes` เรียกเฉพาะเมื่อ `DEV_ROUTES_ENABLED=true` · service / core / repository ของ `pt_calculation` อยู่ที่เดิม
- **path `/api/v1/dev/pt/calculate`** (ไม่อยู่ใต้ `/bo/...`)
- **ไม่ต้อง login** · ไม่ใช้ middleware ของ backoffice · ยังกันด้วย flag: **`APP_ENV=prod` + flag เปิด = boot ไม่ผ่าน (กันแค่ prod)** · ปิด flag = เส้นไม่มี (`404001`)
- เครื่องมือ dev อื่นในอนาคตอยู่ใน context นี้
- แก้ CLAUDE.md (โครงสร้าง `app/internals/{backoffice|frontend|game|developer}` + กฎข้อ 28 ระบุว่า context `developer` เป็น public เฉพาะ env ที่เปิด flag) · ARCHITECTURE.md · spec PTC-40 – PTC-43 · `.env.example` — **lead อนุมัติการเปลี่ยนโครงสร้างนี้** · ทำใน PR เดียวกับ M1 / S1 / S2

## รอบ 3 — `commission_net` (PTC-13 แบบ ก · `fd50811` · lead zerph 2026-10-16 · รายงาน verify-dev-1016)
- **บันทึกการตัดสิน PTC-13 แบบ ก (lead อนุมัติ 2026-10-16):** คืน `commission_net` (commission สุทธิทุกชั้น) แทน `commission` ที่ตั้ง · `t[j] = Round4(%Com[j] × PT รวมเหนือ j ÷ 100)` · Superadmin = −t[1] · ชั้นกลาง = t[i] − t[i+1] · Member = t[Member] · Σ = 0 · เงิน = ยอดแทง × `commission_net` ÷ 100
- ผลตรวจ: สูตรตรง reference ข้อ 4 + 9 เคสคิดมือ + สุ่ม 200,000 เคส · Σ = 0 · ไม่มี −0 · ตัด `commission` ไม่กระทบ (ใช้แค่เส้น dev)
- **ชั้นที่ถือ PT 0 (เช่น `pt.status = false` หรือ CSM) ยังได้ / เสีย commission — ตั้งใจ**: commission คือส่วนลดที่ชั้นบนให้ลูกแล้วชั้นนั้นส่งต่อให้ลูกของตัวเอง ไม่ขึ้นกับ PT ที่ชั้นนั้นถือ
- **A-S1 (ส่งต่อ module เดิมพัน / settle):** ห้ามปัดเงินทีละชั้น (ผลรวม ≠ 0 เช่น ยอดแทง 1.5 เกิน +0.01) → คิดแบบสะสม `T[j] = round(stake × t[j] ÷ 100)` แล้วเงินของแต่ละชั้น = ผลต่างของ T · ทำเป็นฟังก์ชันใน core ให้ settle เรียก
