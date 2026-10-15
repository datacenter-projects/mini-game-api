# Player Auth (ผู้เล่นเข้าระบบ · สาย Transfer) — spec **อนุมัติแล้ว** (lead zerph 2026-10-15) · พร้อม implement

จาก boiledegg · review: [docs/reviews/player-auth.md](../reviews/player-auth.md) (รอบ 1–4) · spec: [docs/modules/player_auth.md](../modules/player_auth.md) (**APPROVED**)
branch `boiledegg/player/auth` (merge dev ล่าสุดแล้ว) · ยังไม่มีโค้ด

## 1. เสร็จแล้ว

- spec แก้ตามรอบ 2–3 ครบ → lead **อนุมัติรอบ 4** (`f50a8b8`) · รับ `407308` (HTTP 403)
- แก้เงื่อนไข **R4-2**: PA-09 `win_limit_per_day` ของผู้เล่นที่ระบบสร้าง = ค่าตั้งต้นตาม MGMT-65 ล่าสุด (เพดาน 1,000,000 บาทแปลงเป็นสกุลผู้เล่น) ผ่านฟังก์ชันเดียวกับ `members/create` · ห้ามพึ่ง `DEFAULT` ของ DB · ใส่ "อนุมัติโดย: zerph · 2026-10-15" ในหัว spec · เอา [เสนอ] ออก

## 2. ต่อไป

- implement ผ่าน PR ตาม checklist หัวข้อ 10 ของ spec — **commit แรก: ย้าย `BearerToken` ไป `pkg/middleware`**
- ส่วน `win_limit_per_day` ของผู้เล่นที่ระบบสร้าง **ต้องประสาน maofoy** (กำลังแก้ MGMT-65 + migration ใหม่) — ใช้ฟังก์ชันกลางตัวเดียวกัน ห้ามเขียนซ้ำ
- เปิด PR แล้วแจ้ง lead เพื่อเปิด verify

## 3. คำถามที่รอ lead

ไม่มี

## 4. ประวัติอัปเดต

- 2026-10-15 · ร่าง spec `player_auth.md` + ส่งคำถาม 13 ข้อให้ lead
- 2026-10-15 · แก้ spec ตาม review รอบ 2–3 (PA6 · name_th/en · register password · snake_case · SUSPENDED/LOCKED ฯลฯ) · สถานะ → รอ approve
- 2026-10-15 · lead อนุมัติ spec (รอบ 4) · แก้ PA-09 win limit ตาม R4-2 · สถานะ → APPROVED · พร้อม implement
