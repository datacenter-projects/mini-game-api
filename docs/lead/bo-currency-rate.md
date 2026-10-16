# สถานะ — `boiledegg/bo/currency-rate`

## 1. ภาพรวม

| | |
|---|---|
| branch | `boiledegg/bo/currency-rate` (แตกจาก `dev` @ `ff12296`) |
| spec | `docs/modules/currency_rate.md` — **APPROVED** 2026-10-07 |
| merge | ยังไม่ merge เข้า `boiledegg/test-infra` / `dev` |

## 2. งาน

| ส่วน | สถานะ |
|---|---|
| worker ดึงเรท USDT ทุก 15 นาที (ดึงทันทีตอนเริ่ม) + Redis lock | ✅ |
| config `IS_CURRENCY_RATE_SYNC` · `CURRENCY_RATE_API_BASE_URL` (origin เปล่า · บังคับใน uat / prod) | ✅ |
| เก็บเรทใน Redis เป็นจำนวนเต็ม ×10^8 · ดึงไม่สำเร็จคงเรทเดิม | ✅ |
| `ConvertAmountService` แปลงยอดข้ามสกุลเพื่อแสดงผล | ✅ |
| unit test (httptest + miniredis) | ✅ ผ่าน |
| ทดสอบกับ askme dev จริง | ✅ ครบ 27 สกุล · 1.16 วินาที · 100.00 THB → 2.94 USD |

**ข้อควรรู้:** worker เปิดเป็นค่าเริ่มต้น — เครื่อง dev ที่ start server จะยิง askme dev ทุก 15 นาที (ปิดด้วย `IS_CURRENCY_RATE_SYNC=false`)
· list 27 สกุลซ้ำกับ `app/core/account` ใน branch account — รวมให้เหลือที่เดียวตอน merge

## 3. รอ lead

- (ไม่มี)

## 4. ประวัติอัปเดต

| วันที่ | commit | อัปเดต |
|---|---|---|
| 2026-10-07 | `3abaf04` | ร่าง spec (ยิงเส้น askme จริงเพื่อดู response) |
| 2026-10-07 | `574a188` | spec อนุมัติ · โค้ดครบตาม spec + test |
