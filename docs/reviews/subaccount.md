# Review BO SubAccount_Management — lead: zerph · 2026-10-11 · ✅ review ครบแล้ว

อ้างอิง: เส้น `subaccounts/*` ใน `docs/modules/agent_management.md` (dev `e481e40`) · เจ้าของ module: boiledegg · ไฟล์นี้คือการตัดสินของ lead — ทำตามรายการแก้ใน branch ใหม่แล้วเปิด PR ให้ lead review

## การตัดสินของ lead (ตามแนะนำทั้งหมด)
| # | เรื่อง | ตัดสิน |
|---|---|---|
| SA1 | `subaccounts/status/update` ส่งสถานะเดิม = 200 ไม่เขียน log (boiledegg ทำเอง `0fb5d0b`) | **อนุมัติ** · spec อ้าง "lead MQ5 / V1" |
| SA2 | `subaccounts/list` ใช้ `keyword` 4–32 แทน `q` (`03e4b57`) | **อนุมัติ** |
| SN | path 5 เส้น `create` · `list` · `detail/get` · `detail/update` · `status/update` (ตาม N1 แล้ว) | **ยืนยัน** · `list` คงเป็นตารางแบ่งหน้า (ไม่ทำ dropdown) |
| SN4 | `subaccounts/detail/update` | **แบบ section** เหมือน E4 / M3: `info` (ชื่อ · เบอร์คู่ — ส่งบาง field ได้) และ `permissions` (ส่งมา = แทนทั้งชุด · เมนูที่ไม่ส่ง = `off` · `{}` = ปิดทุกเมนู) ไม่บังคับทั้งคู่ แต่ต้องมีอย่างน้อย 1 section (`422` — MN2) · tx เดียว · ลำดับเช็คตามลำดับ field (U3) · เจ้าของเท่านั้น (`402404`) |
| SQ1 | sub `INACTIVE` | **login ไม่ได้ + ลบ session ทันที** ตอนตั้ง `INACTIVE` (ใช้แทนการลบ / เลิกจ้าง) · แสดงเป็น `INACTIVE` · ไม่เข้าข่าย "SUSPENDED ดูได้" (AUTH-54 / ACC-31) |
| SQ2 | รีเซ็ตรหัสผ่าน sub | **คง AUTH-52 รอบนี้** — รีเซ็ตผ่าน ADMIN เท่านั้น (เส้นให้เจ้าของรีเซ็ตเองค่อยทำภายหลังถ้าต้องการ) |
| SQ3 | สร้าง sub | **บังคับเปลี่ยนรหัสผ่านตอน login ครั้งแรก** (`must_change_password = true` · ไม่ตั้งวันหมดอายุ) — เจ้าของไม่รู้รหัสของพนักงาน · log เชื่อได้ |
| SQ4 | sub ของ Superadmin | **คงไว้** (Superadmin ตั้งสิทธิ์เอง) |
| SQ5 | `detail/update` ค่าเดิม | **200 ไม่เขียน log** ของ section ที่ไม่เปลี่ยน |
| SQ6 | log การแก้สิทธิ์ | **แยก action `SUB_PERMISSIONS`** จาก `UPDATE_INFO` |
| SQ7 | จำนวน sub สูงสุดต่อเจ้าของ | **ไม่จำกัด** (ตามเดิม) |
| SQ8 | `keyword` ค้นทั้ง username รวมส่วนชื่อเจ้าของ | **คงไว้** (เหมือน `downlines/list`) |

## รายการแก้ (ส่งให้ boiledegg — branch ใหม่ + PR ให้ lead review)

### Spec `docs/modules/agent_management.md`
- S1 หัวเอกสาร: บันทึกการตัดสิน SA · SN · SQ (lead zerph 2026-10-11)
- S2 MGMT-42 + endpoint `detail/update`: body แบบ section (SN4) · ตาราง field · `422` เมื่อไม่ส่ง section · ค่าเดิมไม่เขียน log (SQ5) · log `SUB_PERMISSIONS` (SQ6)
- S3 MGMT-43: ความหมาย `INACTIVE` ตาม SQ1 · อ้าง MQ5 / V1 (SA1)
- S4 MGMT-41: `must_change_password = true` ตอนสร้าง (SQ3) · ไม่จำกัดจำนวน (SQ7)
- S5 schema / action / test case ที่เกี่ยว

### เอกสารอื่น
- S6 `docs/frontend/agent_management.md` (หัวข้อ sub): body ใหม่ของ `detail/update` · ความหมาย `INACTIVE` · sub ต้องเปลี่ยนรหัสผ่านครั้งแรก
- S7 `agent_auth_phase2.md` AUTH-53 / 54 + `account.md` ACC-31: sub `INACTIVE` login ไม่ได้ ไม่เข้าข่าย "SUSPENDED ดูได้" (SQ1)

### โค้ด
- C1 `sub_dto.go`: `SubUpdateRequest` แบบ section (`Info *…` · `Permissions *map`) · อย่างน้อย 1 section · ลำดับเช็คตาม field
- C2 `sub_service.go` `UpdateSubaccountService`: เทียบค่าเดิมต่อ section · log แยก `UPDATE_INFO` / `SUB_PERMISSIONS` · ค่าเดิม = ไม่เขียน · เพิ่ม action ใน models
- C3 SQ1: login + middleware `Authenticated` ปฏิเสธ sub ที่สถานะของตัวเองเป็น `INACTIVE` (เลือก error code ที่เหมาะ เช่นใช้ `401301` หรือ code ใหม่ — แจ้ง lead) · `status/update` → `INACTIVE` ลบ session ของ sub นั้นทันที (หลัง commit) · ใช้ lock `FOR NO KEY UPDATE` ตามบทเรียน deadlock
- C4 SQ3: `CreateSubaccountService` ตั้ง `MustChangePassword: true`
- C5 test: section เดียว / สองส่วน / ไม่ส่งเลย 422 / ค่าเดิมไม่เขียน log / `SUB_PERMISSIONS` · sub INACTIVE login ไม่ได้ + session เดิมถูกตัด + เรียก `/account/api-credential` ไม่ได้ · sub ใหม่ต้องเปลี่ยนรหัสผ่านก่อนเรียกเส้นอื่น (`401306`) · แก้สิทธิ์แล้ว request ถัดไปใช้สิทธิ์ใหม่ · ADMIN สร้าง sub = `402301` · รัน integration ของ sub แนบผล · แจ้ง frontend
