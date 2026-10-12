# API หลังบ้าน — อัตราแพ้ชนะ (`game_rate` · เมนู ⑥) — 12 ต.ค. 2026

spec เต็ม: `docs/modules/game_rate.md` · **ยังไม่ deploy** (กำลังทำใน branch `boiledegg/bo/game_rate-settings`) · ผู้ดูแล: boiledegg ·
พื้นฐาน (base URL · header · รูปแบบ response · ห้าม `null` · Postman) ดู [agent_management.md](agent_management.md) หัวข้อ 1 · login / ด่านหลัง login ดู [auth.md](auth.md)

## 1. ภาพรวม

- **ค่ากลางชุดเดียวทั้งระบบ** ต่อเกม · ใช้ได้เฉพาะ **Superadmin** และ sub ของ Superadmin ที่มีเมนู `rate` (ดู = view · บันทึก / ย้อน = edit) · บัญชีอื่น `402303`
- 3 เกม (ตามลำดับ): `scratch_card` · `coin_toss` · `rock_paper_scissors`
- **Coin / RPS** ตั้ง 10 ขั้น แต่ละขั้น `multiplier` + `win_percent` (RPS **ไม่มีเสมอ** — แบบเดียวกับ Coin)
- **Scratch** ตั้ง `win_percent` ของบัตร + ตารางรางวัล 1–10 แถว (`multiplier` + `share_percent` รวม 100)
- ตัวเลขทุกตัวทศนิยมไม่เกิน 4 ตำแหน่ง · ตัวคูณ ≤ 1,000 · **`win_percent` > 0 และ ≤ 50 ทุกเกม** · **ร้านต้องได้ ≥ 1% ทุกขั้น** (ไม่ผ่าน = `406101`)
- **ค่าที่คำนวณ backend ส่งให้ทั้งหมด** (ไม่ต้องเขียนสูตรเอง): `reach_percent` · `payout_percent` · `house_edge_percent` · `max_win_percent` (ต่อขั้น) · `min_house_edge_percent` · `min_house_edge_step` · `max_payout_percent` (สรุป) · Scratch `avg_multiplier` · `per_ticket_percent` · `payout_percent` · `house_edge_percent` · `max_win_percent`
- ทุกการบันทึกที่ค่าเปลี่ยน = **เวอร์ชันใหม่** (ค่าเดิมทั้งชุด = `created: false` ไม่สร้างเวอร์ชัน) · ค่าใหม่มีผลกับเดิมพันใหม่เท่านั้น · เกมที่ยังไม่เคยตั้ง = `version: 0` (เปิดเล่นไม่ได้)
- บันทึก / ย้อนต้องส่ง `passcode` และ `base_version` (เวอร์ชันที่หน้าจอโหลดมา) · มีคนบันทึกตัดหน้า = `406301` (msg บอกเวอร์ชันล่าสุดและคนแก้ → ให้โหลดใหม่)

| # | Path (`POST` ทุกเส้น) | ใช้เมื่อ | สิทธิ์ |
|---|---|---|---|
| RE1 | `/api/v1/bo/pr/rate/list` | หน้าเลือกเกม | `rate` view |
| RE2 | `/api/v1/bo/pr/rate/detail/get` | โหลดค่าปัจจุบันของเกม | `rate` view |
| RE3 | `/api/v1/bo/pr/rate/detail/check` | คำนวณ / ตรวจระหว่างแก้ (ไม่บันทึก) | `rate` view |
| RE4 | `/api/v1/bo/pr/rate/detail/update` | บันทึก (+ passcode) | `rate` edit |
| RE5 | `/api/v1/bo/pr/rate/versions/list` | การ์ดประวัติ | `rate` view |
| RE6 | `/api/v1/bo/pr/rate/versions/detail/get` | ดูค่าของเวอร์ชันเก่า | `rate` view |
| RE7 | `/api/v1/bo/pr/rate/versions/rollback` | ย้อนไปใช้ค่าเวอร์ชันเก่า (+ passcode) | `rate` edit |

**error ที่เกิดได้กับทุกเส้น:** `401202`, `401203`, `401301`, `401302`, `401312` → หน้า login · `401304`, `401306`, `401307` → ด่านหลัง login · `401311` บัญชีถูกระงับ · `402303` ไม่มีสิทธิ์

## 2. Endpoints

### RE1 · รายชื่อเกม
```bash
curl -X POST "{{MG_URL}}/api/v1/bo/pr/rate/list" -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{}'
```
`data`: `[ { "game_code", "version", "min_house_edge_percent", "created_at", "created_by" } ]` · ยังไม่เคยตั้ง `version: 0` · ข้อความ `""`

### RE2 · ค่าปัจจุบัน
```bash
curl -X POST "{{MG_URL}}/api/v1/bo/pr/rate/detail/get" -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"game_code":"coin_toss"}'
```
- Coin / RPS: `steps[]` (`step` · `multiplier` · `win_percent` · `reach_percent` · `payout_percent` · `house_edge_percent` · `max_win_percent`) + `min_house_edge_percent` · `min_house_edge_step` · `max_payout_percent` · `version` · `created_at` · `created_by`
- Scratch: `win_percent` · `prizes[]` (`row_no` · `multiplier` · `share_percent` · `per_ticket_percent`) · `avg_multiplier` · `payout_percent` · `house_edge_percent` · `max_win_percent`
- ยังไม่เคยตั้ง: `version: 0` · `steps` / `prizes` = `[]`

Error: `422`, `406401`

### RE3 · ตรวจ + คำนวณ (ไม่บันทึก)
body เหมือน RE4 แต่**ไม่ส่ง** `passcode` / `base_version`
```bash
curl -X POST "{{MG_URL}}/api/v1/bo/pr/rate/detail/check" -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"game_code":"scratch_card","win_percent":50,"prizes":[{"multiplier":0.5,"share_percent":35},{"multiplier":1,"share_percent":35},{"multiplier":2,"share_percent":18},{"multiplier":5,"share_percent":10},{"multiplier":8,"share_percent":2}]}'
```
- `data` = รูปแบบเดียวกับ RE2 + `valid` (true / false) + `issues[]` (`field` · `code` · `message`) — กฎร้านได้ < 1% / สัดส่วนไม่ครบ 100 ยังได้ `200` พร้อมค่าคำนวณ แสดง `issues` ข้างช่องได้เลย
- รูปแบบผิด (เช่น ทศนิยม 5 ตำแหน่ง · ขาดขั้น) = `422`

Error: `422`, `406401`

### RE4 · บันทึก
```bash
curl -X POST "{{MG_URL}}/api/v1/bo/pr/rate/detail/update" -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"game_code":"coin_toss","base_version":0,"steps":[{"multiplier":1.9,"win_percent":50},{"multiplier":3.61,"win_percent":48},{"multiplier":6.859,"win_percent":46},{"multiplier":13.0321,"win_percent":44},{"multiplier":24.761,"win_percent":42},{"multiplier":47.0459,"win_percent":40},{"multiplier":89.3872,"win_percent":38},{"multiplier":169.8356,"win_percent":36},{"multiplier":322.6877,"win_percent":34},{"multiplier":613.1066,"win_percent":32}],"passcode":"123456"}'
```
- `data`: `{ "game_code", "version", "created": true }` · ค่าเท่าปัจจุบัน = `created: false` (ไม่มีเวอร์ชันใหม่)
- RPS body เหมือน Coin (`steps` 10 ขั้น) · Scratch ส่ง `win_percent` + `prizes`
- ส่ง field ของเกมอื่น (เช่น `prizes` ให้ Coin · `draw_percent`) = `422`

Error: `422`, `401204` (passcode ผิด · msg บอกครั้งที่เหลือ), `401205`, `406101`, `406102`, `406301`, `406401`

### RE5 · ประวัติ
```bash
curl -X POST "{{MG_URL}}/api/v1/bo/pr/rate/versions/list" -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"game_code":"scratch_card","page":1,"limit":20}'
```
- แบ่งหน้า (`response.Page`) · เรียงใหม่ → เก่า · แต่ละแถว `version` · `min_house_edge_percent` · `win_percent` (Scratch) · `rollback_from` · `same_as_current` (ปิดปุ่มย้อน) · `changes[]` (เทียบกับเวอร์ชันก่อน · เวอร์ชันแรก `[]`) · `created_at` · `created_by` · `ip`
- `changes` Coin / RPS: `{ "type": "step", "step": 4, "field": "multiplier", "from": 13.0321, "to": 12 }` · Scratch: `win_percent` / `prize_added` / `prize_removed` / `prize_share` (จับคู่รางวัลด้วย `multiplier`)

Error: `422`, `406401`

### RE6 · ค่าของเวอร์ชันเก่า
```bash
curl -X POST "{{MG_URL}}/api/v1/bo/pr/rate/versions/detail/get" -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"game_code":"coin_toss","version":2}'
```
`data` = รูปแบบเดียวกับ RE2 · Error: `422`, `406401`, `406402`

### RE7 · ย้อนเวอร์ชัน
```bash
curl -X POST "{{MG_URL}}/api/v1/bo/pr/rate/versions/rollback" -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"game_code":"coin_toss","version":2,"base_version":3,"passcode":"123456"}'
```
- สร้างเวอร์ชันใหม่ที่ค่าเท่าเวอร์ชันที่เลือก · `data`: `{ "game_code", "version", "rollback_from", "created" }` · ค่าเท่าปัจจุบัน = `created: false`
- เวอร์ชันเก่าที่ไม่ผ่านกฎปัจจุบัน (ร้านได้ < 1%) = `406101`

Error: `422`, `401204`, `401205`, `406101`, `406102`, `406301`, `406401`, `406402`

## 3. Error codes

| Code | ความหมาย | หน้าบ้านควรทำ |
|---|---|---|
| `422` | ข้อมูลไม่ผ่าน (`msg` บอก field) | แสดงข้างช่อง |
| `402303` | ไม่มีสิทธิ์เมนู `rate` | ซ่อนเมนู ⑥ |
| `406101` | ร้านได้ต่ำกว่า 1% (msg บอกขั้น / แถว และค่าสูงสุดที่ตั้งได้) | แสดงข้างช่อง · ใช้ RE3 ก่อนบันทึก |
| `406102` | สัดส่วนรางวัลรวมไม่เท่ากับ 100% | แสดงที่ตารางรางวัล |
| `406301` | มีคนแก้ไปแล้ว (msg บอกเวอร์ชันล่าสุดและคนแก้) | popup แล้วโหลด RE2 ใหม่ |
| `406401` | ไม่พบเกมนี้ | — |
| `406402` | ไม่พบเวอร์ชันนี้ | — |

## 4. ค่าตัวอย่างสำหรับ tester (ตั้งบน dev / uat)

ระบบไม่มีค่าเริ่มต้น — Superadmin ต้องบันทึกครั้งแรกเองก่อนเปิดเกม
- **Coin / RPS:** body ของ RE4 ด้านบน (ขั้น 1 `1.9 / 50` · ขั้น 2 `3.61 / 48` · …) · ร้านได้ต่ำสุด 5% ที่ขั้น 1
- **Scratch:** `win_percent` 50 · รางวัล `0.5×/35` · `1×/35` · `2×/18` · `5×/10` · `8×/2` → ร้านได้ 22.75%

## 5. ล็อก / เกลี่ย % ของรางวัล (Scratch — ฝั่งหน้าบ้าน)

backend **ไม่เก็บสถานะล็อก** และไม่มีเส้นเกลี่ย · ตอนบันทึก / RE3 ตรวจแค่ว่า `share_percent` ทุกแถวรวมกัน = **100 พอดี** (ตรวจแบบเลขแม่นยำ ไม่ปัด · ไม่ตรง = `406102`) ดังนั้นตอนเกลี่ยให้:
- แต่ละแถวทศนิยม**ไม่เกิน 4 ตำแหน่ง** (เกิน = `422`) · แต่ละแถว > 0
- เศษที่เหลือจากการหาร (เช่น 100 ÷ 3 = 33.3333 × 3 = 99.9999) ให้ใส่เพิ่มที่แถวที่ไม่ล็อกแถวใดแถวหนึ่ง (เช่น แถวสุดท้าย = 33.3334) ให้ผลรวมเป็น 100 พอดี
- ถ้าล็อกจนแถวที่เหลือไม่มีที่ให้เกลี่ย (ล็อกรวมเกิน 100 หรือไม่เหลือแถวที่ไม่ล็อก) ให้แจ้งที่หน้าจอเอง
- ใช้ RE3 `detail/check` ตรวจผลรวมและ house edge ก่อนกดบันทึกได้
