# Player Auth (ผู้เล่นเข้าระบบ · สาย Transfer) — spec แก้ตาม review รอบ 2–3 แล้ว **รอ approve** · 2026-10-15

จาก boiledegg · review: [docs/reviews/player-auth.md](../reviews/player-auth.md) (รอบ 1–3) · spec: [docs/modules/player_auth.md](../modules/player_auth.md) (**รอ approve**)
branch `boiledegg/player/auth` (แตกจาก dev @ `f483d43` · merge dev ล่าสุดแล้ว) · ยังไม่มีโค้ด — ลำดับ: game_rate 3 ข้อ (`ba3d63f` อยู่บน dev) → spec → lead review ทีละ endpoint → **แก้ spec (เสร็จ)** → lead approve → implement ผ่าน PR

## 1. เสร็จแล้ว

- แก้ spec ตาม **รอบ 3** ครบ: เพิ่ม PA6 `balance/get` `{currency, credit}` (เป็น ping กันหลุด · SUSPENDED เรียกได้) · เอา `credit` ออกจาก `me/get` · `games/list` เพิ่ม `name_th` / `name_en` (ชื่อเกมเพิ่มใน catalog backend) · `register` รับ `password` (AUTH-36) response `{username, currency}` · L2 รับ `language` (`&lang=`) · `currency` ไม่ตรงผู้เล่น = `422` · สร้างพร้อมกันไม่ error · agent SUSPENDED สร้างผู้เล่นใหม่ไม่ได้ (`407308`) · LOCKED ปฏิเสธทุก `transfer/*` (`407302`) · PA1 ไม่มีรหัส = `407201` · PA4 รหัสเดิมผิดนับ PA-41 · field operator `snake_case` · ย้าย `BearerToken` ไป `pkg/middleware` (แยก 3 middleware)
- ใส่คำตอบรอบ 2 (Q1–Q13) เป็นกฎใน spec ทุกข้อ (หัวข้อ 9 ของ spec มีตารางสรุป)
- Response ทุกเส้นมีตาราง field · ตาราง test case ครบ (รวมยิงพร้อมกัน) · เปลี่ยนสถานะ spec เป็น **รอ approve**

## 2. ยังไม่ทำ

- โค้ดทั้งหมด (รอ lead approve spec) · `transfer/launch` · `seamless/*` · BO reset รหัส · operator reset รหัสผู้เล่น · เปลี่ยน username Member จาก BO (นอกขอบเขตตาม review)

## 3. คำถามที่รอ lead

| # | เรื่อง | ข้อเสนอ |
|---|---|---|
| 1 | review รอบ 3 ไม่ได้ระบุเลข error ของ "agent SUSPENDED สร้างผู้เล่นใหม่ไม่ได้" (`register` + auto-create ใน `login`) | จอง **`407308`** (HTTP 403) "บัญชีผู้สร้างถูกระงับ สร้างผู้เล่นใหม่ไม่ได้" — BO ใช้ `401311` ที่ middleware ของ BO ซึ่ง operator ไม่ผ่านทางนั้น |

ที่เหลือไม่มีคำถามค้าง · approve แล้วจะเริ่ม implement ตาม checklist หัวข้อ 10 ของ spec (แยก commit refactor `BearerToken` ก่อน)

## 4. ประวัติอัปเดต

- 2026-10-15 · ร่าง spec `player_auth.md` + ส่งคำถาม 13 ข้อให้ lead
- 2026-10-15 · แก้ spec ตาม review รอบ 2–3 (PA6 · name_th/en · register password · snake_case · SUSPENDED/LOCKED ฯลฯ) · สถานะ → รอ approve · เหลือถามเลข `407308`
