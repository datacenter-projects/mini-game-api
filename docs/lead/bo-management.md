# สถานะ — `boiledegg/bo/management`

## 1. ภาพรวม

| | |
|---|---|
| branch | `boiledegg/bo/management` (แตกจาก `dev` @ `ff12296`) · ล่าสุด `860c007` |
| spec | `docs/modules/agent_management.md` — **APPROVED** 2026-10-07 · มีส่วนที่แก้หลังอนุมัติ (2026-10-08 — หัวข้อ 3 ข้อ 1) |
| merge | ยังไม่ merge เข้า `dev` · ต้องเข้า `dev` ก่อน `boiledegg/bo/account` (account ใช้ตาราง / สิทธิ์ของ module ②) |
| test | `make check` ผ่าน · integration test 44 ตัวผ่าน (agent_auth + module ②) |

## 2. งาน

| ส่วน | สถานะ | หมายเหตุ |
|---|---|---|
| phase 1 — migration, models, core rules, error codes, `RequirePermission` | ✅ | table test ครบ |
| phase 2 — สร้างบัญชี `POST /manage/agents/create` · `/manage/members/create` | ✅ | ยอดเงินตั้งต้น + ledger + `request_id` · Superadmin วงเงินไม่จำกัด |
| phase 3 — รายชื่อ / รายละเอียด / copy-sources | ✅ | `POST /manage/downlines/list` · `/manage/agents/detail` · `/manage/members/detail` · `GET /manage/agents/copy-sources` |
| phase 4 — แก้บัญชี | ✅ | `update-info` · `update-status` · `update-pt` (MGMT-24) · `update-commission` · `update-hold` |
| phase 5 — sub | ✅ | `/manage/subaccounts/list` (รายชื่อ) · `/detail` · `/create` · `/update-info` · `/update-status` · `RequireMainAccount` |
| เครื่องมือ test (`scripts/testenv.ps1`, `.env.test`, k6) | ✅ | อยู่ในทุก branch (เลิกใช้ `boiledegg/test-infra` แล้ว) |
| `POST /manage/agents/update-games` (เปิด / ปิดทีละเกม) | ⏳ | รอ lead (หัวข้อ 3 ข้อ 3) |
| MGMT-04 สร้าง Key ของ account 1.3 พร้อมบัญชีเจ้าของ | ⏳ | ทำหลัง module ② และ account เข้า `dev` ทั้งคู่ |
| test: sub `INACTIVE` เรียกเส้นอื่นได้ `401311` | ⏳ | ใช้ middleware AUTH-54 ที่อยู่ใน branch account — เทสหลัง account เข้า `dev` |

เอกสารหน้าบ้าน: `docs/frontend/agent_management.md`

## 3. รอ lead

1. **รับทราบสิ่งที่แก้หลังอนุมัติ (2026-10-08 · boiledegg ตัดสิน):**
   - MGMT-07 ชื่อ / ชื่อเล่นเป็นภาษาไทยได้ (ไทย อังกฤษ ตัวเลข ไม่มีช่องว่าง 3–32 ตัวอักษร)
   - MGMT-41 เบอร์โทรของ sub ซ้ำได้ (migration ใหม่ `20261007194632`) · ฝั่ง agent และ Member ยังห้ามซ้ำ
   - หัวข้อ 1: ยอดเงินตั้งต้นตอนสร้างอยู่ใน module นี้ (ให้ตรงกับ MGMT-15A)
   - ชื่อเส้น: สร้าง `/create` · แก้ `/update-info` `/update-status` `/update-pt` `/update-commission` `/update-hold` · sub `/detail` `/create` `/update-info` `/update-status`
   - id ของบัญชีส่งใน body ทุกเส้น (ผู้เรียกมาจาก token) — รายชื่อ / รายละเอียดจึงเป็น `POST` · `page` / `limit` ใน body
   - MGMT-53 บัญชีหลักได้ `edit` ทุกเมนู (รวม `dashboard` / `report`) — เพดาน `view` ใช้จำกัดสิทธิ์ที่ให้ sub
2. **แก้กฎข้อ 19 ใน `CLAUDE.md`:** list ที่รับ `page` / `limit` ใน body ใช้ `utils.NewPage` (ค่าเริ่มต้น / เพดานเดียวกับ `ParsePage`) — ร่าง: "list ต้องใช้ `response.Page` · page / limit ใน query ใช้ `utils.ParsePage` · ใน body (เส้น POST) ใช้ `utils.NewPage`"
3. **`status_game`** (เปิด / ปิดทีละเกม) — อยู่ module ไหน และใครตั้ง
4. **เส้นสร้าง** — แยก 2 เส้น (agent / Member) เหมือนเดิม หรือรวมเป็นเส้นเดียวโดยส่ง `"user_type": "MEMBER"`

## 4. ประวัติอัปเดต

| วันที่ | commit | อัปเดต |
|---|---|---|
| 2026-10-05 | `647d682` | ร่าง spec ช่วง 1 (Company / Share / Agent) |
| 2026-10-06 | `3d61389` | เขียนใหม่ทั้งฉบับตาม review ของ lead (13 ข้อ) |
| 2026-10-06 | `537c82d` → `7a15dd2` | PT เก็บต่อเกม · `pt_from_parent` ในกลุ่ม · PT ต่อลูก + ค่าถือของตัวเอง |
| 2026-10-07 | `fc6dc6a` | GET / POST เท่านั้น · sub status / ไม่ลบ · สิทธิ์ object · ยอดเงินตั้งต้น |
| 2026-10-07 | `a8031d5` · `b7a8320` · `ef3998e` | path `/manage` · ไม่มี null · เมนู `rate` · Superadmin วงเงินไม่จำกัด |
| 2026-10-07 | `2f72122` → `8f951a4` | spec อนุมัติ · phase 1 (migration, models, core, error codes, `RequirePermission`) |
| 2026-10-08 | `cefddae` · `e745335` | phase 2 สร้างบัญชี · แก้ CHECK `agent_type` ให้ปฏิเสธ NULL |
| 2026-10-08 | `ef93ea4` · `6db2ec5` | เครื่องมือ test · ชื่อไทย · เบอร์ sub ซ้ำได้ · `create_superadmin` เติม PT / สกุล |
| 2026-10-08 | `64d590e` | phase 3 รายชื่อ / รายละเอียด / copy-sources |
| 2026-10-08 | `bea3b26` | phase 4 แก้บัญชี · id ใน body · ชื่อเส้นใหม่ |
| 2026-10-08 | `860c007` | phase 5 sub |
