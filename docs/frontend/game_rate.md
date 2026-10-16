# API หลังบ้าน — อัตราแพ้ชนะ (`game_rate` · เมนู ⑥) — 14 ต.ค. 2026

spec เต็ม: `docs/modules/game_rate.md` · **deploy บน dev แล้ว** (`dev-202610140730` · 14 ต.ค. · `status_updated_at` / `status_updated_by` ตั้งแต่ `dev-202610141129`) · ผู้ดูแล: boiledegg ·
พื้นฐาน (base URL · header · รูปแบบ response · ห้าม `null` · Postman) ดู [agent_management.md](agent_management.md) หัวข้อ 1 · login / ด่านหลัง login ดู [auth.md](auth.md)

> **ก่อนเทส:** ระบบไม่มีอัตราเริ่มต้น — Superadmin ต้องบันทึกอัตราของแต่ละเกมครั้งแรกเอง (RE4 · ค่าตัวอย่างหัวข้อ 4) · ยังไม่ตั้ง = `version: 0` เกมเปิดเล่นไม่ได้
>
> **เพิ่ม (16 ต.ค. — รอ lead ยืนยัน)** — `status` (boolean) ต่อขั้น Coin / RPS และต่อแถวรางวัล Scratch ใน RE3 / RE4 · คืนใน RE2 / RE3 / RE6 — ดูหมายเหตุใต้ตาราง RE4
> **เพิ่ม / เปลี่ยน (13–14 ต.ค.)** — RE8 `rate/status/update` สวิตช์เปิด / ปิดเกมทั้งระบบ · RE1 มี `status_game` + `status_updated_at` / `status_updated_by` (เปิด / ปิดล่าสุดเมื่อไหร่ โดยใคร) · Scratch มี `prizes[].payout_percent` ต่อแถว ·
> ข้อความ error (`msg` / `issues[].message`) ใช้ชื่ออ่านได้ เช่น "ขั้นที่ 1 · โอกาสชนะ …" — ชี้ช่องที่ผิดใช้ `issues[].field` (path เดิม)

> ตัวอย่าง response ส่วนใหญ่ได้จากการยิง API จริงใน test env (ตัดเฉพาะ `created_at` ให้อ่านง่าย) · field ที่เพิ่มทีหลัง (`status_game` · `status_updated_*` · `payout_percent` · ข้อความ error ใหม่) ใส่ตามรูปแบบที่ API ตอบจริง · ตัวเลขเป็นค่าตัวอย่าง

## 1. ภาพรวม

- **ค่ากลางชุดเดียวทั้งระบบ** ต่อเกม · ใช้ได้เฉพาะ **Superadmin** และ sub ของ Superadmin ที่มีเมนู `rate` (ดู = view · บันทึก / ย้อน = edit) · บัญชีอื่น `402303`
- **รหัสเกม (`game_code`) มี 3 เกม** — ใช้ส่งในทุกเส้นที่มี `game_code` · ดึงรายชื่อจาก RE1 `rate/list` ได้ (เรียงตามลำดับนี้) · รหัสผิด = `406401`

| `game_code` | เกม | รูปแบบอัตรา (body ของ RE3 / RE4) |
|---|---|---|
| `scratch_card` | เกมขูดบัตร (Scratch Card) | `win_percent` ของใบ + `prizes` 1–10 แถว (`multiplier` + `share_percent`) |
| `coin_toss` | โยนเหรียญ (Coin Flip) | `steps` 10 ขั้น แต่ละขั้น `multiplier` + `win_percent` |
| `rock_paper_scissors` | เป่ายิ้งฉุบ (Rock Paper Scissors · RPS) | `steps` 10 ขั้น **แบบเดียวกับโยนเหรียญ** (ไม่มีผลเสมอ · ส่ง `draw_percent` = `422`) |

- **Coin / RPS** ตั้ง 10 ขั้น แต่ละขั้น `multiplier` + `win_percent` (RPS **ไม่มีเสมอ** — รูปแบบเดียวกับ Coin)
- **Scratch** ตั้ง `win_percent` ของบัตร + ตารางรางวัล 1–10 แถว (`multiplier` + `share_percent` รวม 100)
- ตัวเลขทุกตัวทศนิยมไม่เกิน 4 ตำแหน่ง · ตัวคูณ ≤ 1,000 · **`win_percent` > 0 และ ≤ 50 ทุกเกม** · **ร้านต้องได้ ≥ 1% ทุกขั้น** (ไม่ผ่าน = `406101`)
- **ค่าที่คำนวณ backend ส่งให้ทั้งหมด** ไม่ต้องเขียนสูตรเอง และไม่ได้เก็บใน DB (คำนวณจากค่าที่ตั้งทุกครั้ง)
- ทุกการบันทึกที่ค่าเปลี่ยน = **เวอร์ชันใหม่** (ค่าเดิมทั้งชุด = `created: false` ไม่สร้างเวอร์ชัน) · ค่าใหม่มีผลกับเดิมพันใหม่เท่านั้น · เกมที่ยังไม่เคยตั้ง = `version: 0` (เปิดเล่นไม่ได้)
- บันทึก / ย้อนต้องส่ง `passcode` และ `base_version` (เวอร์ชันที่หน้าจอโหลดมา) · มีคนบันทึกตัดหน้า = `406301` (msg บอกเวอร์ชันล่าสุดและคนแก้ → ให้โหลดใหม่)
- ตัวเลขต้องเป็น JSON number (ส่งเป็นข้อความ เช่น `"50"` = code `400`)
- `max_win_percent` = ค่าสูงสุดที่ตั้งได้จริง (ปัดลง 4 ตำแหน่ง · ไม่เกิน 50) กรอกค่านั้นแล้วบันทึกผ่านแน่นอน
- **ข้อความ error (`msg` / RE3 `issues[].message`) ใช้ชื่อที่คนอ่านเข้าใจ** เช่น `steps[0].win_percent` → "ขั้นที่ 1 · โอกาสชนะ" · `prizes[1].share_percent` → "รางวัลที่ 2 · สัดส่วนรางวัล" · `win_percent` → "อัตราชนะ" (เปลี่ยน 14 ต.ค. — lead Q9) · **ชี้ช่องที่ผิดให้ใช้ `issues[].field`** (คง path เดิม) · `message` เป็นภาษาไทย · `changes[].from` / `to` เป็น `null` ได้ (รางวัลที่เพิ่ม / ลบ) — นอกนั้นไม่มี `null`

| # | Path (`POST` ทุกเส้น) | ใช้เมื่อ | สิทธิ์ |
|---|---|---|---|
| RE1 | `/api/v1/bo/pr/rate/list` | หน้าเลือกเกม | `rate` view |
| RE2 | `/api/v1/bo/pr/rate/detail/get` | โหลดค่าปัจจุบันของเกม | `rate` view |
| RE3 | `/api/v1/bo/pr/rate/detail/check` | คำนวณ / ตรวจระหว่างแก้ (ไม่บันทึก) | `rate` view |
| RE4 | `/api/v1/bo/pr/rate/detail/update` | บันทึก (+ passcode) | `rate` edit |
| RE5 | `/api/v1/bo/pr/rate/versions/list` | การ์ดประวัติ | `rate` view |
| RE6 | `/api/v1/bo/pr/rate/versions/detail/get` | ดูค่าของเวอร์ชันเก่า | `rate` view |
| RE7 | `/api/v1/bo/pr/rate/versions/rollback` | ย้อนไปใช้ค่าเวอร์ชันเก่า (+ passcode) | `rate` edit |
| RE8 | `/api/v1/bo/pr/rate/status/update` | เปิด / ปิดเกมทั้งระบบ (ปิดปรับปรุง · + passcode) | `rate` edit |

**error ที่เกิดได้กับทุกเส้น:** `401202`, `401203`, `401301`, `401302`, `401312` → หน้า login · `401304`, `401306`, `401307` → ด่านหลัง login · `401311` บัญชีถูกระงับ · `402303` ไม่มีสิทธิ์

## 2. Endpoints

### RE1 · รายชื่อเกม — `rate/list`

**Request**
```bash
curl -X POST "{{MG_URL}}/api/v1/bo/pr/rate/list" -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{}'
```

**Response**
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": [
    { "game_code": "scratch_card", "version": 2, "min_house_edge_percent": 22.75, "created_at": "2026-10-12T17:53:14+07:00", "created_by": "superadmin", "status_game": true, "status_updated_at": "", "status_updated_by": "" },
    { "game_code": "coin_toss", "version": 1, "min_house_edge_percent": 5, "created_at": "2026-10-12T17:53:14+07:00", "created_by": "superadmin", "status_game": false, "status_updated_at": "2026-10-14T16:20:00.123+07:00", "status_updated_by": "superadmin@staff" },
    { "game_code": "rock_paper_scissors", "version": 0, "min_house_edge_percent": 0, "created_at": "", "created_by": "", "status_game": true, "status_updated_at": "", "status_updated_by": "" }
  ]
}
```

| field | ความหมาย |
|---|---|
| `version` | เวอร์ชันปัจจุบัน · `0` = ยังไม่เคยตั้ง (เกมเปิดเล่นไม่ได้) |
| `min_house_edge_percent` | ร้านได้ต่ำสุดของเวอร์ชันปัจจุบัน (Coin / RPS = ขั้นที่น้อยสุด · Scratch = ทั้งใบ) |
| `created_at` / `created_by` | เวลา / คนบันทึกเวอร์ชันปัจจุบัน · ยังไม่เคยตั้ง = `""` |
| `status_game` | สวิตช์ของเกม (RE8) · `false` = ปิดปรับปรุงทั้งระบบ · เกมเล่นได้จริงเมื่อ `status_game = true` และ `version ≥ 1` |
| `status_updated_at` / `status_updated_by` | เปิด / ปิดสวิตช์ครั้งล่าสุด (RE8): เวลา + คนทำ (sub = `owner@name` เช่น `superadmin@staff`) · ไม่เคยเปิด / ปิด = `""` (เพิ่ม 14 ต.ค.) |

- ไม่แบ่งหน้า (1 แถวต่อเกม) · เรียงตามลำดับเกมด้านบน

### RE2 · ค่าปัจจุบันของเกม — `rate/detail/get`

**Request**
```bash
# เกมขูดบัตร
curl -X POST "{{MG_URL}}/api/v1/bo/pr/rate/detail/get" -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"game_code":"scratch_card"}'
# โยนเหรียญ
curl -X POST "{{MG_URL}}/api/v1/bo/pr/rate/detail/get" -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"game_code":"coin_toss"}'
# เป่ายิ้งฉุบ (response รูปแบบเดียวกับโยนเหรียญ)
curl -X POST "{{MG_URL}}/api/v1/bo/pr/rate/detail/get" -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"game_code":"rock_paper_scissors"}'
```

**Response — Coin / RPS**
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "game_code": "coin_toss",
    "version": 1,
    "steps": [
      { "step": 1,  "multiplier": 1.9,      "win_percent": 50, "reach_percent": 50,     "payout_percent": 95,      "house_edge_percent": 5,       "max_win_percent": 50, "status": false },
      { "step": 2,  "multiplier": 3.61,     "win_percent": 48, "reach_percent": 24,     "payout_percent": 86.64,   "house_edge_percent": 13.36,   "max_win_percent": 50, "status": false },
      { "step": 3,  "multiplier": 6.859,    "win_percent": 46, "reach_percent": 11.04,  "payout_percent": 75.7234, "house_edge_percent": 24.2766, "max_win_percent": 50, "status": false },
      { "step": 4,  "multiplier": 13.0321,  "win_percent": 44, "reach_percent": 4.8576, "payout_percent": 63.3047, "house_edge_percent": 36.6953, "max_win_percent": 50, "status": false },
      { "step": 5,  "multiplier": 24.761,   "win_percent": 42, "reach_percent": 2.0402, "payout_percent": 50.5172, "house_edge_percent": 49.4828, "max_win_percent": 50, "status": false },
      { "step": 6,  "multiplier": 47.0459,  "win_percent": 40, "reach_percent": 0.8161, "payout_percent": 38.3931, "house_edge_percent": 61.6069, "max_win_percent": 50, "status": false },
      { "step": 7,  "multiplier": 89.3872,  "win_percent": 38, "reach_percent": 0.3101, "payout_percent": 27.7198, "house_edge_percent": 72.2802, "max_win_percent": 50, "status": false },
      { "step": 8,  "multiplier": 169.8356, "win_percent": 36, "reach_percent": 0.1116, "payout_percent": 18.9603, "house_edge_percent": 81.0397, "max_win_percent": 50, "status": false },
      { "step": 9,  "multiplier": 322.6877, "win_percent": 34, "reach_percent": 0.038,  "payout_percent": 12.2484, "house_edge_percent": 87.7516, "max_win_percent": 50, "status": false },
      { "step": 10, "multiplier": 613.1066, "win_percent": 32, "reach_percent": 0.0121, "payout_percent": 7.447,   "house_edge_percent": 92.553,  "max_win_percent": 50, "status": false }
    ],
    "min_house_edge_percent": 5,
    "min_house_edge_step": 1,
    "max_payout_percent": 95,
    "created_at": "2026-10-12T17:53:14+07:00",
    "created_by": "superadmin"
  }
}
```

| field (ต่อขั้น) | ความหมาย (หน้าจอ) |
|---|---|
| `multiplier` / `win_percent` | ค่าที่ตั้ง (ตัวคูณรางวัล / โอกาสชนะ) |
| `reach_percent` | โอกาสถึงขั้นนี้ (ชนะติดกันตั้งแต่ขั้น 1) |
| `payout_percent` | ผลตอบแทนผู้เล่นถ้าหยุดที่ขั้นนี้ |
| `house_edge_percent` | ร้านได้ (= 100 − ผลตอบแทน) |
| `max_win_percent` | โอกาสชนะสูงสุดที่ตั้งได้ของขั้นนี้ |

| field (สรุป) | ความหมาย (กล่องด้านขวา) |
|---|---|
| `min_house_edge_percent` + `min_house_edge_step` | ร้านได้ต่ำสุด X% ที่ขั้น N |
| `max_payout_percent` | ผลตอบแทนผู้เล่นสูงสุด |

**Response — Scratch** (`{"game_code":"scratch_card"}`)
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "game_code": "scratch_card",
    "version": 2,
    "win_percent": 50,
    "prizes": [
      { "row_no": 1, "multiplier": 0.5, "share_percent": 35, "per_ticket_percent": 17.5, "payout_percent": 8.75, "status": false },
      { "row_no": 2, "multiplier": 1,   "share_percent": 35, "per_ticket_percent": 17.5, "payout_percent": 17.5, "status": false },
      { "row_no": 3, "multiplier": 2,   "share_percent": 18, "per_ticket_percent": 9, "payout_percent": 18, "status": false },
      { "row_no": 4, "multiplier": 5,   "share_percent": 10, "per_ticket_percent": 5, "payout_percent": 25, "status": false },
      { "row_no": 5, "multiplier": 8,   "share_percent": 2,  "per_ticket_percent": 1, "payout_percent": 8, "status": false }
    ],
    "avg_multiplier": 1.545,
    "payout_percent": 77.25,
    "house_edge_percent": 22.75,
    "max_win_percent": 50,
    "created_at": "2026-10-12T17:53:14+07:00",
    "created_by": "superadmin"
  }
}
```
- `per_ticket_percent` = โอกาสได้รางวัลแถวนั้นต่อบัตร 1 ใบ · `prizes[].payout_percent` = ผลตอบแทนที่แถวนั้นจ่ายคืนผู้เล่น (= `per_ticket_percent` × `multiplier` · ใช้กับคอลัมน์ "ผลตอบแทนผู้เล่น" · ทุกแถวรวมเท่ากับ `payout_percent` ของใบ — **อาจต่างได้ไม่เกิน 0.0001 จากการปัดทีละแถว** ใช้แสดงผลอย่างเดียว · เพิ่ม 14 ต.ค.) · `avg_multiplier` = ตัวคูณเฉลี่ยของบัตรที่ชนะ

**Response — ยังไม่เคยตั้ง**
```json
{ "code": 200, "msg": "สำเร็จ", "data": { "game_code": "coin_toss", "version": 0, "steps": [], "min_house_edge_percent": 0, "min_house_edge_step": 0, "max_payout_percent": 0, "created_at": "", "created_by": "" } }
```

Error: `422`, `406401`

### RE3 · ตรวจ + คำนวณ (ไม่บันทึก) — `rate/detail/check`

**Request** = body ของ RE4 แต่**ไม่ส่ง** `passcode` / `base_version`
```bash
curl -X POST "{{MG_URL}}/api/v1/bo/pr/rate/detail/check" -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" \
  -d '{"game_code":"coin_toss","steps":[{"multiplier":1.99,"win_percent":50,"status":false},{"multiplier":3.61,"win_percent":48,"status":false},{"multiplier":6.859,"win_percent":46,"status":false},{"multiplier":13.0321,"win_percent":44,"status":false},{"multiplier":24.761,"win_percent":42,"status":true},{"multiplier":47.0459,"win_percent":40,"status":true},{"multiplier":89.3872,"win_percent":38,"status":false},{"multiplier":169.8356,"win_percent":36,"status":false},{"multiplier":322.6877,"win_percent":34,"status":true},{"multiplier":613.1066,"win_percent":32,"status":true}]}'
```

**Response — ผิดกฎ (ยังได้ 200 พร้อมค่าคำนวณ)**
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "game_code": "coin_toss",
    "valid": false,
    "issues": [
      { "field": "steps[0].win_percent", "code": 406101, "message": "ขั้นที่ 1 · โอกาสชนะ ตั้งได้ไม่เกิน 49.7487% (ร้านต้องได้อย่างน้อย 1%)" }
    ],
    "steps": [
      { "step": 1, "multiplier": 1.99, "win_percent": 50, "reach_percent": 50, "payout_percent": 99.5, "house_edge_percent": 0.5, "max_win_percent": 49.7487, "status": false },
      { "step": 2, "multiplier": 3.61, "win_percent": 48, "reach_percent": 24, "payout_percent": 86.64, "house_edge_percent": 13.36, "max_win_percent": 50, "status": false }
    ],
    "min_house_edge_percent": 0.5,
    "min_house_edge_step": 1,
    "max_payout_percent": 99.5
  }
}
```
(`steps` ตัดเหลือ 2 ขั้น — ของจริงครบ 10)

**Response — ผ่าน (Scratch)**
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "game_code": "scratch_card",
    "valid": true,
    "issues": [],
    "win_percent": 50,
    "prizes": [ { "row_no": 1, "multiplier": 0.5, "share_percent": 35, "per_ticket_percent": 17.5, "payout_percent": 8.75, "status": false } ],
    "avg_multiplier": 1.545,
    "payout_percent": 77.25,
    "house_edge_percent": 22.75,
    "max_win_percent": 50
  }
}
```
- `issues[].code`: `406101` ร้านได้ < 1% · `406102` สัดส่วนรางวัลไม่ครบ 100 · ใช้แสดงข้างช่องที่ `field` ชี้
- รูปแบบผิด (ทศนิยม 5 ตำแหน่ง · ขาดขั้น · `win_percent` > 50 ฯลฯ) = `422` ไม่มีค่าคำนวณ

Error: `422`, `406401`

### RE4 · บันทึก — `rate/detail/update`

**curl** (`base_version` = `version` จาก RE2 · ยังไม่เคยตั้ง = `0` · `passcode` 6 หลักของคนที่กด)
```bash
# เกมขูดบัตร
curl -X POST "{{MG_URL}}/api/v1/bo/pr/rate/detail/update" -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json"   -d '{"game_code":"scratch_card","base_version":0,"win_percent":50,"prizes":[{"multiplier":0.5,"share_percent":35,"status":false},{"multiplier":1,"share_percent":35,"status":true},{"multiplier":2,"share_percent":18,"status":false},{"multiplier":5,"share_percent":10,"status":true},{"multiplier":8,"share_percent":2,"status":false}],"passcode":"123456"}'
# โยนเหรียญ
curl -X POST "{{MG_URL}}/api/v1/bo/pr/rate/detail/update" -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json"   -d '{"game_code":"coin_toss","base_version":0,"steps":[{"multiplier":1.9,"win_percent":50,"status":false},{"multiplier":3.61,"win_percent":48,"status":false},{"multiplier":6.859,"win_percent":46,"status":false},{"multiplier":13.0321,"win_percent":44,"status":false},{"multiplier":24.761,"win_percent":42,"status":true},{"multiplier":47.0459,"win_percent":40,"status":true},{"multiplier":89.3872,"win_percent":38,"status":false},{"multiplier":169.8356,"win_percent":36,"status":false},{"multiplier":322.6877,"win_percent":34,"status":true},{"multiplier":613.1066,"win_percent":32,"status":true}],"passcode":"123456"}'
# เป่ายิ้งฉุบ (body แบบเดียวกับโยนเหรียญ)
curl -X POST "{{MG_URL}}/api/v1/bo/pr/rate/detail/update" -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json"   -d '{"game_code":"rock_paper_scissors","base_version":0,"steps":[{"multiplier":1.9,"win_percent":50,"status":false},{"multiplier":3.61,"win_percent":48,"status":false},{"multiplier":6.859,"win_percent":46,"status":false},{"multiplier":13.0321,"win_percent":44,"status":false},{"multiplier":24.761,"win_percent":42,"status":true},{"multiplier":47.0459,"win_percent":40,"status":true},{"multiplier":89.3872,"win_percent":38,"status":false},{"multiplier":169.8356,"win_percent":36,"status":false},{"multiplier":322.6877,"win_percent":34,"status":true},{"multiplier":613.1066,"win_percent":32,"status":true}],"passcode":"123456"}'
```

**Request — Coin / RPS** (RPS ใช้ `"game_code":"rock_paper_scissors"` body เหมือนกัน)
```json
{
  "game_code": "coin_toss",
  "base_version": 0,
  "steps": [
    { "multiplier": 1.9, "win_percent": 50, "status": false }, { "multiplier": 3.61, "win_percent": 48, "status": false },
    { "multiplier": 6.859, "win_percent": 46, "status": false }, { "multiplier": 13.0321, "win_percent": 44, "status": false },
    { "multiplier": 24.761, "win_percent": 42, "status": true }, { "multiplier": 47.0459, "win_percent": 40, "status": true },
    { "multiplier": 89.3872, "win_percent": 38, "status": false }, { "multiplier": 169.8356, "win_percent": 36, "status": false },
    { "multiplier": 322.6877, "win_percent": 34, "status": true }, { "multiplier": 613.1066, "win_percent": 32, "status": true }
  ],
  "passcode": "123456"
}
```

**Request — Scratch**
```json
{
  "game_code": "scratch_card",
  "base_version": 1,
  "win_percent": 50,
  "prizes": [
    { "multiplier": 0.5, "share_percent": 35, "status": false }, { "multiplier": 1, "share_percent": 35, "status": true }, { "multiplier": 2, "share_percent": 18, "status": false },
    { "multiplier": 5, "share_percent": 10, "status": true }, { "multiplier": 8, "share_percent": 2, "status": false }
  ],
  "passcode": "123456"
}
```

| field | กติกา |
|---|---|
| `game_code` | `scratch_card` · `coin_toss` · `rock_paper_scissors` (ไม่มี = `406401`) |
| `base_version` | `version` จาก RE2 ที่หน้าจอโหลดมา · ยังไม่เคยตั้ง = `0` · ไม่ตรง = `406301` |
| `steps` (Coin / RPS) | ครบ 10 ขั้น · `multiplier` 1–1,000 และมากกว่าขั้นก่อน · `win_percent` > 0 และ ≤ 50 · ห้ามส่ง `draw_percent` · `status` (boolean · ไม่ส่ง = `false`) ต่อขั้น — ดูหมายเหตุ `status` ด้านล่าง |
| `win_percent` + `prizes` (Scratch) | `win_percent` > 0 และ ≤ 50 · รางวัล 1–10 แถว · `multiplier` > 0 ≤ 1,000 ห้ามซ้ำ · `share_percent` > 0 รวม = 100 พอดี · `status` (boolean · ไม่ส่ง = `false`) ต่อแถว |
| `passcode` | passcode ของคนที่กด |

> **`status` ต่อขั้น / ต่อแถวรางวัล (เพิ่ม 16 ต.ค. — รอ lead ยืนยัน · spec Q10)** — ใส่ได้ใน `steps[]` (Coin / RPS) และ `prizes[]` (Scratch) ของ RE3 / RE4 · boolean · **ไม่ส่ง = `false`** · ส่ง `null` = `422`
> backend **เก็บและคืนค่าเท่านั้น** (ใช้ล็อกขั้น / แถวตอนเกลี่ยฝั่งหน้าบ้าน) ไม่กระทบสูตร / กฎ / ผลคำนวณ · คืนใน RE2 · RE3 · RE6 ทุกขั้น / ทุกแถว (`"status": true|false`)
> เป็นส่วนของค่า: **เปลี่ยน `status` อย่างเดียว = เวอร์ชันใหม่** (`created: true`) · ส่งชุดเดิมรวม `status` = `created: false` · ย้อนเวอร์ชัน (RE7) คืน `status` ของเวอร์ชันนั้น · RE5 `changes` ไม่แสดงการเปลี่ยน `status`
> ตัวอย่าง request ด้านบนส่ง `status` ครบทุกขั้น / ทุกแถวให้เห็นค่า (`true` เฉพาะที่เลือก · ที่เหลือ `false`) · ไม่ส่ง = `false` เหมือนกัน · หน้าบ้านเลือกเองว่าจะส่ง / ไม่ส่ง

**Response**
```json
{ "code": 200, "msg": "สำเร็จ", "data": { "game_code": "scratch_card", "version": 2, "created": true } }
```
ค่าเท่าเวอร์ชันปัจจุบันทั้งชุด (ไม่สร้างเวอร์ชันใหม่):
```json
{ "code": 200, "msg": "สำเร็จ", "data": { "game_code": "scratch_card", "version": 2, "created": false } }
```

**Response — error (ตัวอย่างจริง)**
```json
{ "code": 406301, "msg": "มีคนแก้ไปแล้ว (v2 โดย superadmin) กรุณาโหลดใหม่" }
{ "code": 422, "msg": "ขั้นที่ 1 · โอกาสชนะ ต้องมากกว่า 0 และไม่เกิน 50" }
{ "code": 406101, "msg": "ขั้นที่ 1 · โอกาสชนะ ตั้งได้ไม่เกิน 49.7487% (ร้านต้องได้อย่างน้อย 1%)" }
{ "code": 406102, "msg": "สัดส่วนรางวัลรวมต้องเท่ากับ 100% (ตอนนี้ 99.5%)" }
```

Error: `400` (ตัวเลขเป็นข้อความ), `422`, `401204` (passcode ผิด · msg บอกครั้งที่เหลือ), `401205`, `406101`, `406102`, `406301`, `406401`

### RE5 · ประวัติ — `rate/versions/list`

**Request**
```bash
curl -X POST "{{MG_URL}}/api/v1/bo/pr/rate/versions/list" -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"game_code":"scratch_card","page":1,"limit":20}'
```

**Response**
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "current_page": 1, "total_page": 1, "total_count": 2, "limit": 20, "has_next": false, "has_prev": false,
    "data": [
      {
        "version": 2,
        "min_house_edge_percent": 22.75,
        "win_percent": 50,
        "rollback_from": 0,
        "same_as_current": true,
        "changes": [
          { "type": "win_percent", "from": 35, "to": 50 },
          { "type": "prize_share", "multiplier": 0.5, "from": 20, "to": 35 },
          { "type": "prize_share", "multiplier": 1, "from": 50, "to": 35 },
          { "type": "prize_share", "multiplier": 2, "from": 30, "to": 18 },
          { "type": "prize_added", "multiplier": 5, "from": null, "to": 10 },
          { "type": "prize_added", "multiplier": 8, "from": null, "to": 2 }
        ],
        "created_at": "2026-10-12T17:53:14+07:00",
        "created_by": "superadmin",
        "ip": "203.0.113.10"
      },
      {
        "version": 1,
        "min_house_edge_percent": 58,
        "win_percent": 35,
        "rollback_from": 0,
        "same_as_current": false,
        "changes": [],
        "created_at": "2026-10-12T17:53:14+07:00",
        "created_by": "superadmin",
        "ip": "203.0.113.10"
      }
    ]
  }
}
```

| field | ความหมาย |
|---|---|
| `win_percent` | Scratch เท่านั้น (Coin / RPS ไม่มี key นี้) |
| `rollback_from` | ย้อนมาจากเวอร์ชันไหน · `0` = บันทึกปกติ |
| `same_as_current` | `true` = ค่าเท่าเวอร์ชันปัจจุบัน → ปิดปุ่ม "ย้อนไปใช้ค่านี้" |
| `changes` | สิ่งที่เปลี่ยนเทียบกับเวอร์ชันก่อนหน้า · เวอร์ชันแรก `[]` |
| `changes[].type` | `win_percent` · `prize_added` (`from` null) · `prize_removed` (`to` null) · `prize_share` (Scratch · จับคู่รางวัลด้วย `multiplier`) · `step` (Coin / RPS) |

Coin / RPS `changes`:
```json
[ { "type": "step", "step": 1, "field": "win_percent", "from": 55, "to": 50 },
  { "type": "step", "step": 4, "field": "multiplier", "from": 13.0321, "to": 12 } ]
```

Error: `422`, `406401`

### RE6 · ค่าของเวอร์ชันเก่า — `rate/versions/detail/get`

**Request**
```bash
curl -X POST "{{MG_URL}}/api/v1/bo/pr/rate/versions/detail/get" -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"game_code":"scratch_card","version":1}'
```

**Response** (รูปแบบเดียวกับ RE2)
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "game_code": "scratch_card",
    "version": 1,
    "win_percent": 35,
    "prizes": [
      { "row_no": 1, "multiplier": 0.5, "share_percent": 20, "per_ticket_percent": 7, "payout_percent": 3.5, "status": false },
      { "row_no": 2, "multiplier": 1,   "share_percent": 50, "per_ticket_percent": 17.5, "payout_percent": 17.5, "status": false },
      { "row_no": 3, "multiplier": 2,   "share_percent": 30, "per_ticket_percent": 10.5, "payout_percent": 21, "status": false }
    ],
    "avg_multiplier": 1.2,
    "payout_percent": 42,
    "house_edge_percent": 58,
    "max_win_percent": 50,
    "created_at": "2026-10-12T17:53:14+07:00",
    "created_by": "superadmin"
  }
}
```

Error: `422`, `406401`, `406402` (ไม่พบเวอร์ชันนี้)

### RE7 · ย้อนเวอร์ชัน — `rate/versions/rollback`

**Request**
```json
{ "game_code": "scratch_card", "version": 1, "base_version": 2, "passcode": "123456" }
```
```bash
curl -X POST "{{MG_URL}}/api/v1/bo/pr/rate/versions/rollback" -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" \
  -d '{"game_code":"scratch_card","version":1,"base_version":2,"passcode":"123456"}'
```
- `version` = เวอร์ชันที่จะย้อนไป · `base_version` = เวอร์ชันปัจจุบันที่หน้าจอโหลดมา

**Response** — สร้างเวอร์ชันใหม่ที่ค่าเท่าเวอร์ชันที่เลือก
```json
{ "code": 200, "msg": "สำเร็จ", "data": { "game_code": "scratch_card", "version": 3, "rollback_from": 1, "created": true } }
```
ค่าเท่าเวอร์ชันปัจจุบันอยู่แล้ว (ไม่สร้างเวอร์ชันใหม่):
```json
{ "code": 200, "msg": "สำเร็จ", "data": { "game_code": "scratch_card", "version": 3, "rollback_from": 0, "created": false } }
```
- เวอร์ชันเก่าที่ไม่ผ่านกฎปัจจุบัน: ร้านได้ < 1% = `406101` · ผิดรูปแบบ (เช่น `win_percent` > 50) = `422`

Error: `422`, `401204`, `401205`, `406101`, `406102`, `406301`, `406401`, `406402`

### RE8 · เปิด / ปิดเกมทั้งระบบ — `rate/status/update` (เพิ่ม 13 ต.ค.)

สวิตช์ปิดปรับปรุงเกม · ปิดแล้ว**ทุกสายเล่นเกมนั้นไม่ได้** (เดิมพันใหม่ · เดิมพันที่ค้างอยู่เล่นต่อจนจบได้) · เปิดคืน = แต่ละบัญชีกลับไปตามค่าที่ตั้งไว้เอง (บัญชีที่ถูกปิดใน `agents/detail/update` ยังปิด)

**Request**
```bash
curl -X POST "{{MG_URL}}/api/v1/bo/pr/rate/status/update" -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" \
  -d '{"game_code":"coin_toss","status_game":false,"passcode":"123456"}'
```

| field | กติกา |
|---|---|
| `game_code` | ต้องส่ง · ไม่มีเกมนี้ = `406401` |
| `status_game` | ต้องส่ง · boolean (`true` เปิด · `false` ปิด) · ไม่ส่ง / `null` = `422` |
| `passcode` | ต้องส่ง (ไม่ส่ง `422` · ผิด `401204` · ครบ 5 ครั้ง `401205`) |

**Response**
```json
{ "code": 200, "msg": "สำเร็จ", "data": { "game_code": "coin_toss", "status_game": false, "changed": true, "status_updated_at": "2026-10-14T16:20:00.123+07:00", "status_updated_by": "superadmin@staff" } }
```
ส่งค่าเดิมซ้ำ (ปิดอยู่แล้ว):
```json
{ "code": 200, "msg": "สำเร็จ", "data": { "game_code": "coin_toss", "status_game": false, "changed": false, "status_updated_at": "2026-10-14T16:20:00.123+07:00", "status_updated_by": "superadmin@staff" } }
```
- `changed: false` = ไม่มีอะไรเปลี่ยน (ไม่บันทึกประวัติ) · `status_updated_at` / `status_updated_by` = ครั้งล่าสุดที่เปิด / ปิด (`changed: true` = ครั้งนี้เอง) ใช้แสดง "เปลี่ยนล่าสุดเมื่อ … โดย …" ได้เลย
- หลังสำเร็จให้โหลด RE1 ใหม่ (`status_game` ของเกมนั้นเปลี่ยน) · `agents/detail/get` ของทุกบัญชีจะเห็น `status_game_effective` ของเกมนั้นเป็น `false`
- ประวัติการเปิด / ปิดอยู่ในประวัติการแก้ไขของบัญชี Superadmin (`UPDATE_GAMES`)
- ตั้งให้ Superadmin ทุกบัญชี · บัญชีที่ยังไม่มีค่าของเกมนั้น ระบบสร้างให้แล้วปิด / เปิดตามที่ส่ง (ไม่ตอบ `changed: false` ทั้งที่เกมยังไม่ปิด)

Error: `422`, `401204`, `401205`, `402303`, `406401`

## 3. Error codes

| Code | ความหมาย | หน้าบ้านควรทำ |
|---|---|---|
| `400` | ตัวเลขส่งเป็นข้อความ (body ผิดชนิด) | ส่งเป็น JSON number |
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
