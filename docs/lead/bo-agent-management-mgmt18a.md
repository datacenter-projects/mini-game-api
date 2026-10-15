# Agent_Management — MGMT-18A Superadmin ไม่ตั้ง Remain ให้ลูก · 2026-10-14

จาก boiledegg · branch `boiledegg/bo/agent_management` (`19f1a9f` spec · `4bc5148` โค้ด) ·
**merge เข้า dev + deploy แล้ว** (`7407f0f` · tag `dev-202610140730`) — lead approve 2026-10-14

## 1. การตัดสิน (2026-10-14 · lead ดูแล้ว)

| # | เรื่อง | ผล |
|---|---|---|
| 1 | Superadmin ตั้ง `remain_quota` ให้ Company | **ไม่ได้** — ส่วนที่ไม่มีใครถือคืน Superadmin อยู่แล้ว (MGMT-22) · การคิดตอน settle เหมือนเดิม |
| 2 | ส่งค่าไม่ใช่ 0 | `422` (ไม่มี error code ใหม่) |
| 3 | `force` ของ Superadmin | ตามเดิม (ตั้งได้) |
| 4 | ข้อมูลเดิม | migration ปรับเป็น 0 |
| 5 | CSM | กฎเดียวกัน (Company ทุกประเภท) |

## 2. ที่ทำ

| ส่วน | ผล |
|---|---|
| spec | `docs/modules/agent_management.md` MGMT-18A + test case |
| core | `CheckChildPT` / `ValidateChildPT` รับ `creatorIsSuperadmin` · `PTSuperadminRemain` → `422` "เอาส่วนที่เหลือ Superadmin ตั้งไม่ได้ (ส่วนที่เหลือคืน Superadmin อยู่แล้ว) ต้องเป็น 0" · table test |
| service | สร้าง agent + `agents/detail/update` (section pt) ส่ง role ของผู้สร้าง (Superadmin + sub ของ Superadmin) |
| migration | `20261014120000_agent_management_superadmin_remain` — `remain = 0` ที่ `parent_id` เป็น Superadmin · ค่าอื่นไม่แตะ · prod เริ่มจาก DB ว่าง ไม่ต้องแยก 3 ขั้น |
| test | integration: สร้าง Company / CSM · แก้ Company · `force` ได้ · ชั้นอื่นตั้ง remain ได้ · migration test |
| เอกสารหน้าบ้าน | `docs/frontend/agent_management.md` หัวเอกสาร + ตาราง field |

lint 0 issues · unit + integration ผ่านทุก package · migrate RDS dev แล้ว (2026-10-14)
