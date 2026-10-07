# สถานะ — `boiledegg/bo/management`

## 1. ภาพรวม

| | |
|---|---|
| branch | `boiledegg/bo/management` (แตกจาก `dev` @ `ff12296`) |
| spec | `docs/modules/agent_management.md` — **DRAFT** รออนุมัติ |
| merge | ยังไม่มีโค้ด |

## 2. งาน

| ส่วน | สถานะ |
|---|---|
| spec module ② (Company / Share / Agent / Member / sub / สิทธิ์ / PT / ยอดเงินตั้งต้น) | 📝 เขียนครบ แก้ตาม review หลายรอบ · รออนุมัติ |
| โค้ด | ⏳ ยังไม่เริ่ม (รอ spec อนุมัติ) |

สรุป spec ล่าสุด:
- path ทุกเส้น `/api/v1/bo/pr/manage/...` · ใช้แค่ `GET` / `POST` · ไม่มี `null` ใน API
- ผู้สร้างตั้งค่าที่ให้ลูกแต่ละคนแยกกัน (`pt_from_parent`) · บัญชีตั้งค่าถือของตัวเอง (`/manage/agents/me/pt`) · ส่วนที่เหลือคืน Superadmin
- sub: `ACTIVE` / `INACTIVE` · ไม่มีการลบ · สิทธิ์ `off` / `view` / `edit` 9 เมนู (Superadmin มี `rate` · ฝั่ง agent มี `announcement`)
- ยอดเงินตั้งต้นตอนสร้าง (ledger + `request_id`) · Superadmin วงเงินไม่จำกัด

**ติด:** spec ยังไม่อนุมัติ — module ② เป็นตัวที่กั้น Profile และข้อมูลรับรอง API ของ account

## 3. รอ lead

1. อนุมัติ `agent_management.md`
2. `status_game` (เปิด / ปิดทีละเกม) — จะย้ายไป module ไหน และให้ใครใช้

## 4. ประวัติอัปเดต

| วันที่ | commit | อัปเดต |
|---|---|---|
| 2026-10-05 | `647d682` | ร่าง spec ช่วง 1 (Company / Share / Agent) |
| 2026-10-06 | `3d61389` | เขียนใหม่ทั้งฉบับตาม review ของ lead (13 ข้อ) |
| 2026-10-06 | `537c82d` → `7a15dd2` | PT เก็บต่อเกม · `pt_from_parent` ในกลุ่ม · PT ต่อลูก + ค่าถือของตัวเอง |
| 2026-10-07 | `fc6dc6a` | GET / POST เท่านั้น · sub status / ไม่ลบ · สิทธิ์ object · ยอดเงินตั้งต้น |
| 2026-10-07 | `a8031d5` · `b7a8320` · `ef3998e` | path `/manage` · ไม่มี null · เมนู `rate` · Superadmin วงเงินไม่จำกัด |
