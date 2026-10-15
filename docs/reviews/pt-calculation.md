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
