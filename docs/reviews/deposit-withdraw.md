# Review Deposit / Withdraw (ฝาก / ถอนเงินให้ลูกตรง) — lead: zerph · 2026-10-13 · ✅ ตัดสินแล้ว

อ้างอิง: `docs/modules/deposit_withdraw.md` (DRAFT · dev `11a777b`) · เจ้าของ: **maofoy** · ไฟล์นี้คือการตัดสินของ lead — แก้ spec ตามรายการด้านล่างก่อน แล้ว implement ใน branch ใหม่จาก dev และ**เปิด PR ให้ boiledegg review** (ไม่ commit ตรงเข้า dev)

## ภาพรวม (ยืนยัน)
- ย้ายเครดิตภายในสายกับ**ลูกตรงชั้นเดียว** (agent หรือ Member) · ฝาก = แม่ลด ลูกเพิ่ม · ถอน = ลูกลด แม่เพิ่ม · **สาย Transfer เท่านั้น** (Seamless ทั้งหมด รวม CSM / Share Master / Seamless Reseller ใช้ไม่ได้) · Superadmin เป็นแหล่งเงินเดียวของระบบ (วงเงินไม่จำกัด)
- เส้น: DW1 `deposit-withdraw/downlines/list` (`payment` view) · DW2 `deposit-withdraw/create` (`payment` edit + passcode) · DW3 `deposit-withdraw/logs/list` (`payment` view)

## การตัดสินของ lead
| # | เรื่อง | ตัดสิน |
|---|---|---|
| DQ1 | ลูกที่ SUSPENDED / LOCKED | **ฝากไม่ได้ (`404301`) แต่ถอนได้** — ล็อกคนโกงแล้วต้องดึงเงินคืนได้ · หน้าบ้านเปิดปุ่มถอนเมื่อลูกไม่ ACTIVE |
| DQ2 | กดซ้ำ / ยิงพร้อมกัน | **lock รายการบน Redis ด้วย key ของ user ที่ถูกทำรายการ (ลูก) TTL 5–10 วินาที** (lead) — ดูรายละเอียดด้านล่าง |
| DQ3 | ลูก agent ที่ยังไม่มีแถวยอด | upsert แถวที่ขาดก่อน → lock `agent_balances` `FOR UPDATE` เรียง `agent_id` → `user_members` `FOR NO KEY UPDATE` · ย้ายกฎลำดับ lock ไป `docs/ARCHITECTURE.md` |
| DQ4 | ถอนจาก Member ที่รอบเกมค้าง | module เดิมพันต้องหักเงินตอนแทง (ข้อบังคับของ Bet) → ถอนไม่กระทบรอบค้าง · open item ของ module เดิมพัน |
| DQ5 | ledger ของ Superadmin | แยก reason `DEPOSIT_FROM_SUPERADMIN` / `WITHDRAW_TO_SUPERADMIN` |
| DQ6 | เพดานต่อรายการ | 1,000,000,000 |
| DQ7 | `null` ใน response | ใช้ `parent_unlimited` แทน (ACC-32) |
| DQ8 | ชื่อ field | `credit` → `amount` |
| DQ9 | error module id | `04` (จองใน `ERROR_CODES.md`) |
| DQ10 | `request_id` | เช็คข้ามเส้นทั้ง 2 ทางกับ `agents/create` / `members/create` (ใช้ซ้ำข้ามเส้น = `422`) |
| DQ11 | DW1 ค้นหา | เพิ่ม `keyword` (4–32 · username แบบเดียวกับ E2) |
| DQ12 | note + ประวัติ | `note` ทำเลย (ไม่บังคับ ≤ 200 ตัว · `""`) · **ทำ DW3 `logs/list` รอบนี้** (+1.5 pd) |
| DQ13 | path / สิทธิ์ | ตาม spec (ตรง N1 / A5) |

### DQ2 รายละเอียด (Redis lock ที่ user ที่ถูกทำรายการ)
- key เช่น `dw:lock:{owner_type}:{owner_id}` (ลูกที่ถูกฝาก/ถอน) · `SET NX` + TTL **10 วินาที** + ค่า token ของ request · ปลด lock ด้วย token (ไม่ลบ lock ของคนอื่น) · ได้ lock ไม่ได้ = ตอบทันที "มีรายการของบัญชีนี้กำลังทำอยู่ กรุณาลองใหม่" (code ใหม่ใน `04xxxx`) ไม่รอ
- tx ต้องจบเร็วกว่า TTL (ตั้ง timeout ของ tx / statement < 10 วินาที) · Redis ใช้ไม่ได้ = ปฏิเสธ (fail closed) ไม่ทำรายการโดยไม่มี lock
- **ยังต้องเก็บ `request_id` ไว้ด้วย** — เช็ค `request_id` เป็นอย่างแรกหลังได้ lock: เคยทำสำเร็จแล้ว = คืนผลเดิม (กรณีผู้ใช้กดซ้ำหลังรายการแรกจบ) · DB lock ตาม DQ3 ยังคงไว้เป็นชั้นสุดท้าย
- ใช้ Redis ตัวเดียวกับ lock อื่นของ mini-game-api (ระบุใน spec)

## รายการแก้ spec (maofoy)
- S1 หัวเอกสาร: บันทึกการตัดสิน DQ1–DQ13 (lead zerph 2026-10-13) · สถานะ APPROVED หลังแก้
- S2 DW-05: ลูกไม่ ACTIVE ฝากไม่ได้ (`404301`) · ถอนได้ (DQ1) + test ถอนจากลูก LOCKED
- S3 DW-21 / 22: ลำดับ lock ตาม DQ3 + ย้ายกฎไป `docs/ARCHITECTURE.md`
- S4 DW-24 / 25: Redis lock ตาม DQ2 + เช็ค `request_id` หลังได้ lock ก่อนอย่างอื่น · เช็คข้ามเส้น (DQ10) · error code ใหม่ "กำลังทำรายการ"
- S5 DW-10: เพดาน 1,000,000,000 (DQ6)
- S6 DW-23: reason ของ Superadmin (DQ5) · map CHECK violation → `404304` / `404306`
- S7 ข้อสมมติ A1 / A2 → open item ของ module เดิมพัน (DQ4)
- S8 DW2 request / response: `amount` (DQ8) · `parent_unlimited` (DQ7) · `note` (DQ12) · `request_id` ใน response
- S9 schema `deposit_withdraw_logs`: `ip` · `parent_balance_*` NOT NULL DEFAULT · `parent_unlimited` · `note VARCHAR(200) NOT NULL DEFAULT ''`
- S10 DW1 `keyword` (DQ11) · หมายเหตุหน้าบ้าน: ปุ่มถอนเปิดเมื่อลูกไม่ ACTIVE
- S11 DW3 `logs/list`: ตัวอย่าง request/response เต็ม · กรองตามช่วงวัน / ลูก / ประเภท · แบ่งหน้า
- S12 `payment_type` ไม่มีผลกับ DW (P1) · สาย Seamless ทั้งหมดใช้ไม่ได้
- S13 `ERROR_CODES.md` จอง `04` + ตาราง code (HTTP 200)
- S14 test case: ยิงพร้อมกันที่ลูกเดียวกัน (ได้ lock 1 · อีกอันได้ "กำลังทำรายการ") · กดซ้ำหลังจบ = ผลเดิม · retry หลังลูกถูกระงับ = ผลเดิม · ฝากครั้งแรกให้ agent ที่ไม่มีแถวยอด · ถอนจากลูก LOCKED · sub ของ Superadmin ฝาก · `amount` 1,000,000,001 · `request_id` ซ้ำข้ามเส้น `422` · Redis ล่ม = ปฏิเสธ · `null` ใน `note` = `422`
- S15 `docs/MIGRATION_STATUS.md` + `docs/frontend/deposit_withdraw.md`
