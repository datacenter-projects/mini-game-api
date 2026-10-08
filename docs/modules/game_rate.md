# game_rate (อัตราแพ้ชนะ) — Spec

- สถานะ: **APPROVED** — boiledegg รีวิว 2026-10-08
- อนุมัติโดย: lead (zerph) · วันที่: 2026-10-08
- ที่มาของ rule: diagram ของ lead (`Coin Flip Flow.html`, `Rock Paper Scissors Flow.html`, `Scratch Card Flow.html`) +
  คำตอบของ boiledegg 2026-10-08 · ไม่ได้อ้างโค้ดหรือเอกสารของโปรเจกต์เก่า
- ชื่อ module ในโค้ด: `game_rate` · path `/api/v1/bo/pr/rate/...` · error module id `06` (ตรงกับเมนู ⑥)

> เอกสารนี้คือ source of truth ของ module — โค้ดและ test ต้องตรงกับเอกสารนี้ ถ้าไม่ตรงให้แก้โค้ด
> หรือแก้เอกสารผ่านการอนุมัติใหม่ ห้ามอ้างเอกสารของโปรเจกต์เก่า

## 1. หน้าที่ของ module

เก็บและแก้ "อัตราแพ้ชนะ" ของแต่ละเกม (ตัวคูณ + โอกาสชนะ) ที่เอนจิ้นเกมใช้ตัดสินผล · เมนู ⑥ ของหลังบ้าน

- ผู้ตั้งค่า: **Superadmin เท่านั้น** (และ sub ของ Superadmin ที่ได้สิทธิ์ `rate`) — Company / Share / Agent / Member ตั้งหรือดูไม่ได้
- module นี้**ไม่ได้**สุ่มผลหรือจ่ายเงิน — แค่เก็บค่า · เอนจิ้นเกมและ module เดิมพันอ่านค่าไปใช้

## 2. คำศัพท์

| คำ | ความหมาย |
|---|---|
| ขั้น (step) | รอบที่ Member ทายต่อใน Streak (Coin Flip / Rock Paper Scissors) · มี 10 ขั้นตายตัว |
| Streak | Member ทายต่อเพื่อสะสมเงิน · ปิด Streak = เล่นครั้งเดียว ใช้ค่าของขั้น 1 |
| ตัวคูณ (`multiplier`) | เงินที่ได้ = ยอดแทง × ตัวคูณ (ไม่รวมทบจากขั้นก่อน) |
| `win_percent` | โอกาสชนะของขั้นนั้น (Coin Flip / RPS) หรือของบัตร 1 ใบ (Scratch Card) |
| `draw_percent` | โอกาสเสมอของขั้นนั้น (RPS เท่านั้น) · เสมอ = เล่นขั้นเดิมใหม่ ไม่หักเงินเพิ่ม |
| `reach_percent` (R) | โอกาสที่ Member ไปถึงและชนะขั้นนั้น = ผลคูณของโอกาสชนะจริงตั้งแต่ขั้น 1 ถึงขั้นนั้น |
| `house_edge_percent` | ร้านได้เฉลี่ย (% ของยอดแทง) = 100% − จ่ายคืนคาดหวัง |
| เวอร์ชัน (`version`) | ทุกการบันทึกสร้างชุดค่าใหม่ 1 เวอร์ชัน ห้ามแก้เวอร์ชันเดิม |
| ตารางรางวัล (Scratch Card) | รายการรางวัลของบัตรที่ชนะ: ตัวคูณ + สัดส่วนในบัตรที่ชนะ |

## 3. Business rules

### ทั่วไป

| ID | Rule |
|---|---|
| RATE-01 | ตั้งค่าแยกต่อเกม (`coin_toss`, `rock_paper_scissors`, `scratch_card`) · ชุดเดียวใช้กับ**ทุกยอดแทง** (ค่าเป็น "เท่า" ของยอดแทง ไม่แยกตามยอด) · **ชุดเดียวใช้กับทุกสกุลเงิน** (ตัวคูณ / % ไม่ขึ้นกับสกุล — รีวิว 2026-10-08) |
| RATE-01A | **รายชื่อเกมอยู่ในโค้ด** (`app/core` — รายการเดียวกับ module ② `pt` / `status_game`) · ตอนนี้ 3 เกม · เพิ่มเกมใหม่ = แก้โค้ดพร้อมเอนจิ้นของเกมนั้น · ไม่มีเส้นสร้าง / เปิด-ปิดเกมทั้งระบบ (รีวิว 2026-10-08 — ระบบเก่าแยกเกมตามระดับราคาแทงจึงต้องมีเส้นสร้างเกม · ระบบใหม่ไม่แยก RATE-01) · ปิดเกมรายบัญชีใช้ module ② `update-games` |
| RATE-02 | ผู้ดู / ผู้แก้: Superadmin · sub ของ Superadmin ที่มีสิทธิ์ `rate` (ดู = `view` · แก้ = `edit`) · บัญชีอื่น `402303` / `402311` ตามระบบสิทธิ์ของ module ② |
| RATE-03 | ตัวคูณ: JSON number **ทศนิยมไม่เกิน 2 ตำแหน่ง** (คำตอบ 2026-10-08) · เก็บเป็นจำนวนเต็มหน่วย 0.01 (`x1.90` = `190`) |
| RATE-04 | %: JSON number **ทศนิยมไม่เกิน 2 ตำแหน่ง** · เก็บเป็นจำนวนเต็ม bp (`50%` = `5000`) — ตรงกับการสุ่ม 0–9,999 ของเอนจิ้น (1 ค่า = 0.01%) |
| RATE-05 | บันทึก = **แทนทั้งชุดของเกมนั้น** และสร้างเวอร์ชันใหม่ (เลขเวอร์ชันต่อเกมเพิ่มทีละ 1) · เวอร์ชันเดิมห้ามแก้ / ลบ |
| RATE-05A | **ย้อนเวอร์ชัน** (`/rate/rollback` — รีวิว 2026-10-08): ส่ง `game_code` + `version` ที่จะย้อนไป · ระบบคัดลอกค่าของเวอร์ชันนั้นเป็น**เวอร์ชันใหม่** (เช่นมี v1–v3 ย้อนไป v2 = สร้าง v4 ค่าเท่า v2) · เวอร์ชันเดิมไม่ถูกแก้ · ตรวจ RATE-07 ซ้ำ · ต้อง passcode และสิทธิ์ `rate` แก้ไขเหมือน `/rate/update` · log เก็บ `rollback_from` · ย้อนไปเวอร์ชันที่ใช้อยู่ = `422` |
| RATE-06 | ค่าใหม่มีผลกับ**เดิมพันใหม่เท่านั้น** · เดิมพันที่เริ่มแล้ว (รวม Streak ที่ค้างอยู่) ใช้เวอร์ชันเดิมจนจบ · module เดิมพันต้องเก็บ `rate_version_id` คู่กับเดิมพันทุกครั้ง |
| RATE-07 | **ร้านต้องได้กำไรทุกจุดที่ Member หยุดได้** — ถ้าคำนวณแล้ว `house_edge_percent ≤ 0` ที่จุดไหน บันทึกไม่ได้ทั้งชุด (`406101`) และ msg บอกขั้น / แถวที่ผิด |
| RATE-08 | **บันทึกต้องยืนยัน passcode** (`RequirePasscode` — คำตอบ 2026-10-08) · ทุกการบันทึกเก็บ log คนแก้ในแถวเวอร์ชัน: `created_by` (username ณ เวลานั้น · sub = `owner@name`), `actor_type` / `actor_id`, `ip`, `request_id`, `created_at` · ตารางเวอร์ชันคือ log (ห้ามแก้ / ลบ) ดูได้ที่ `/rate/history` |
| RATE-09A | **ไม่มีค่าเริ่มต้น** (คำตอบ 2026-10-08): เกมที่ยังไม่เคยตั้งค่า = `version = 0` · เอนจิ้นต้อง**ไม่รับเดิมพัน**ของเกมนั้นจนกว่า Superadmin จะบันทึกครั้งแรก (บังคับใน module เกม / เดิมพัน) |
| RATE-09B | ยอดแทง: **บาทเต็ม ขั้นต่ำ 1 บาท** (คำตอบ 2026-10-08 — หน้าบ้านมีจำนวนให้เลือก ไม่ได้พิมพ์เอง) · ส่งเศษสตางค์ = `422` — ตรวจใน module เดิมพัน ไม่มีค่าให้ตั้งใน module นี้ · เงินรางวัล = ยอดแทง × ตัวคูณของขั้นนั้น คิดครั้งเดียว · บาทเต็ม × ตัวคูณ 2 ตำแหน่ง = **ลงตัวที่สตางค์เสมอ ไม่ต้องปัด** · Scratch Card ตรง 2 ช่อง แบ่งครึ่งตาม RATE-35 |
| RATE-09C | เวลารอ Member **1 วัน** (คำตอบ 2026-10-08): Streak ที่ค้างอยู่เกิน 1 วัน = CASH OUT อัตโนมัติด้วยเงินของขั้นที่ชนะล่าสุด · Scratch Card ที่ยังไม่ขูดเกิน 1 วัน = เปิดบัตรอัตโนมัติ (ผลไม่เปลี่ยน) — กฎของเอนจิ้นเกม ไม่มีค่าให้ตั้งใน module นี้ |
| RATE-09 | response แสดงค่าที่คำนวณให้ด้วย (`reach_percent`, `house_edge_percent`, `max_win_percent`) — หลังบ้านคำนวณจากสูตรเดียวกับที่ใช้ตรวจตอนบันทึก |

### Coin Flip (`coin_toss`)

| ID | Rule |
|---|---|
| RATE-10 | 10 ขั้นตายตัว ต้องส่งครบ 10 ขั้น · แต่ละขั้น `multiplier` + `win_percent` |
| RATE-11 | `multiplier` ≥ 1.00 และ**มากกว่าขั้นก่อนหน้าเสมอ** (ยืนยัน 2026-10-08) · `win_percent` 0–100 |
| RATE-12 | RANDOM (ระบบเลือกฝั่งให้) ใช้ `win_percent` เดียวกับเลือกฝั่งเอง |
| RATE-13 | `reach_n = win_1 × … × win_n` · `house_edge_n = 100% − reach_n × multiplier_n` ต้อง > 0 **ทุกขั้น** (Member CASH OUT ขั้นไหนก็ได้) · `max_win_n = 100% ÷ (reach_{n−1} × multiplier_n)` |

ตัวอย่าง (ค่าใน diagram): ขั้น 2 `reach = 50% × 48% = 24%` · `house_edge = 100% − 24% × 3.61 = 13.36%`

### Rock Paper Scissors (`rock_paper_scissors`)

| ID | Rule |
|---|---|
| RATE-20 | 10 ขั้นตายตัว · แต่ละขั้น `multiplier` + `win_percent` + `draw_percent` · % แพ้ = 100 − ชนะ − เสมอ (ระบบคำนวณ ไม่ต้องส่ง) |
| RATE-21 | `multiplier` ≥ 1.00 และมากกว่าขั้นก่อนหน้า · `win_percent` 0–100 · `draw_percent` 0–99 · `win_percent + draw_percent < 100` |
| RATE-22 | เสมอไม่นับเป็นชนะหรือแพ้ → โอกาสชนะจริงของขั้น `q_n = win_n ÷ (100 − draw_n)` · `reach_n = q_1 × … × q_n` · `house_edge_n = 100% − reach_n × multiplier_n` > 0 ทุกขั้น · `max_win_n = (100 − draw_n) ÷ (reach_{n−1} × multiplier_n)` |

ตัวอย่าง (ค่าใน diagram): ขั้น 1 ชนะ 34 · เสมอ 30 → `q_1 = 34 ÷ 70 = 48.57%`

### Scratch Card (`scratch_card`)

| ID | Rule |
|---|---|
| RATE-30 | `win_percent` (W) = โอกาสที่บัตร 1 ใบชนะ · 0–100 |
| RATE-31 | `prizes` = ตารางรางวัลของบัตรที่ชนะ · แต่ละแถว `multiplier` (> 0 — ยืนยัน 2026-10-08) + `share_percent` (สัดส่วนในบัตรที่ชนะ 0–100) · **1–10 แถว** (คำตอบ 2026-10-08) · `multiplier` ห้ามซ้ำกัน |
| RATE-32 | `share_percent` ทุกแถวรวมกันต้อง **= 100 พอดี** (`406102`) |
| RATE-33 | `avg_multiplier (A) = Σ(multiplier × share_percent)` · จ่ายคืนคาดหวัง `P = W × A` · `house_edge = 100% − P` > 0 · `max_win_percent = 100% ÷ A` |
| RATE-34 | % ต่อใบของแต่ละแถว = `W × share_percent` (ระบบคำนวณให้ใน response) |
| RATE-35 | ตรง 1 หรือ 2 ช่อง เอนจิ้นสุ่มเองด้วยโอกาสเท่ากัน — **ไม่มีค่าให้ตั้ง** · ตรง 2 ช่อง = แบ่งรางวัลครึ่ง (ช่องแรกปัดลง ช่องที่สองได้ส่วนที่เหลือ รวมเท่ารางวัลพอดี) — เป็นกฎของเอนจิ้น ไม่ใช่ของ module นี้ |

ตัวอย่าง (ค่าใน diagram): W 50% · ตาราง 0.5×/35, 1×/35, 2×/18, 5×/10, 8×/2 → `A = 1.545` · `P = 77.25%` · `house_edge = 22.75%` · `max_win ≈ 64.72%`

## 4. สิ่งที่พบในโค้ดเก่า และการตัดสินใจ

ไม่ได้อ่านโค้ดเก่า — ใช้ diagram ของ lead และคำตอบของ boiledegg

| # | ที่มา | การตัดสินใจ |
|---|---|---|
| 1 | diagram เก็บตัวคูณทศนิยม 4 ตำแหน่ง (`1.90^n`) แต่ comment ใน struct บอกหน่วย 0.01 | ใช้ **2 ตำแหน่ง** ตามคำตอบ 2026-10-08 |
| 2 | diagram ตารางรางวัล Scratch Card ตัวคูณ `step 0.1, min 0` | ทศนิยม 2 ตำแหน่งตาม RATE-03 · ต้อง > 0 (รางวัล 0 เท่าไม่ใช่การชนะ) |
| 3 | diagram มี dropdown ยอดแทง 10 / 20 / 50 / 100 / 200 / 500 | ใช้แค่แสดงตัวอย่างเงิน ไม่ใช่ค่าที่ตั้ง — ตั้งชุดเดียวใช้ทุกยอด (RATE-01) |
| 4 | diagram "เปลี่ยนค่าแล้วมีผลกับ Bet ใหม่ · บันทึกค่าคู่กับ Bet" | เก็บเป็นเวอร์ชัน (RATE-05) ให้เดิมพันอ้าง `rate_version_id` แทนการคัดลอกค่าทั้งชุด |
| 5 | diagram มีค่าเริ่มต้นของทุกเกม | **ไม่ใส่ค่าเริ่มต้น** — Superadmin ตั้งเองก่อนเปิดเกม (RATE-09A) |

## 5. Endpoints

ใต้ `/api/v1/bo/pr` · `Authenticated` + `PassedGates()` ทุกเส้น · ตามแบบ module ②: อ่านที่ต้องระบุ = `POST` · ส่ง `game_code` ใน body

| Method | Path | สิทธิ์ | ใช้เมื่อ |
|---|---|---|---|
| GET | `/rate/list` | `rate` view | รายชื่อเกม + เวอร์ชันปัจจุบัน + `house_edge` สรุป |
| POST | `/rate/detail` | `rate` view | ค่าปัจจุบันของ 1 เกม พร้อมค่าที่คำนวณ |
| POST | `/rate/update` | `rate` edit | บันทึกค่าใหม่ของ 1 เกม (= เวอร์ชันใหม่) |
| POST | `/rate/history` | `rate` view | รายการเวอร์ชันของ 1 เกม (page / limit ใน body) |
| POST | `/rate/version` | `rate` view | ค่าของเวอร์ชันที่ระบุ (ใช้ "โหลดค่าเดิมกลับเข้าฟอร์ม" — หน้าบ้านกดบันทึกเองเป็นเวอร์ชันใหม่) |
| POST | `/rate/rollback` | `rate` edit + passcode | ย้อนไปใช้ค่าของเวอร์ชันที่ระบุ (= เวอร์ชันใหม่ — RATE-05A) |

### GET /api/v1/bo/pr/rate/list

Response `data`:
```json
[
  { "game_code": "coin_toss", "version": 3, "min_house_edge_percent": 5, "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "superadmin1" },
  { "game_code": "rock_paper_scissors", "version": 0, "min_house_edge_percent": 0, "updated_at": "", "updated_by": "" },
  { "game_code": "scratch_card", "version": 1, "min_house_edge_percent": 22.75, "updated_at": "2026-10-05T09:00:00+07:00", "updated_by": "superadmin1@staff" }
]
```
- `min_house_edge_percent` = ค่าที่ต่ำสุดในทุกขั้น (Scratch Card มีค่าเดียว)
- เกมที่ยังไม่เคยตั้งค่า (RATE-09A): `version = 0` · ตัวเลข `0` · ข้อความ `""` — เกมนั้นยังเปิดให้เล่นไม่ได้

### POST /api/v1/bo/pr/rate/detail

Request:
```json
{ "game_code": "coin_toss" }
```
Response `data` (Coin Flip):
```json
{
  "game_code": "coin_toss",
  "version": 3,
  "steps": [
    { "step": 1, "multiplier": 1.90, "win_percent": 50, "reach_percent": 50, "house_edge_percent": 5, "max_win_percent": 52.63 },
    { "step": 2, "multiplier": 3.61, "win_percent": 48, "reach_percent": 24, "house_edge_percent": 13.36, "max_win_percent": 55.4 }
  ],
  "created_at": "2026-10-08T10:00:00+07:00",
  "created_by": "superadmin1"
}
```
- `steps` มีครบ 10 ขั้น (ตัวอย่างตัดเหลือ 2)
- RPS: แต่ละขั้นเพิ่ม `draw_percent` และ `lose_percent` (คำนวณ)
- Scratch Card:
```json
{
  "game_code": "scratch_card",
  "version": 1,
  "win_percent": 50,
  "prizes": [
    { "multiplier": 0.5, "share_percent": 35, "per_ticket_percent": 17.5 },
    { "multiplier": 1, "share_percent": 35, "per_ticket_percent": 17.5 },
    { "multiplier": 2, "share_percent": 18, "per_ticket_percent": 9 },
    { "multiplier": 5, "share_percent": 10, "per_ticket_percent": 5 },
    { "multiplier": 8, "share_percent": 2, "per_ticket_percent": 1 }
  ],
  "avg_multiplier": 1.545,
  "payout_percent": 77.25,
  "house_edge_percent": 22.75,
  "max_win_percent": 64.72,
  "created_at": "2026-10-05T09:00:00+07:00",
  "created_by": "superadmin1@staff"
}
```
- ค่าที่คำนวณ (`reach_percent`, `house_edge_percent`, `max_win_percent`, `payout_percent`) ปัดแสดง 2 ตำแหน่ง · `avg_multiplier` แสดงสูงสุด 4 ตำแหน่ง (เช่น `1.545`) · การตรวจ RATE-07 ใช้ค่าเต็มก่อนปัด
- เกมที่ยังไม่เคยตั้งค่า: `version = 0` · `steps` / `prizes` = `[]` · ตัวเลข `0` · ข้อความ `""`

Error: `422`, `406401`

### POST /api/v1/bo/pr/rate/update

Request (Coin Flip — ส่งครบ 10 ขั้น):
```json
{
  "game_code": "coin_toss",
  "steps": [
    { "multiplier": 1.90, "win_percent": 50 },
    { "multiplier": 3.61, "win_percent": 48 }
  ],
  "passcode": "123456"
}
```
Request (RPS): แต่ละขั้น `{ "multiplier", "win_percent", "draw_percent" }` + `passcode`

Request (Scratch Card):
```json
{
  "game_code": "scratch_card",
  "win_percent": 50,
  "prizes": [ { "multiplier": 0.5, "share_percent": 35 }, { "multiplier": 8, "share_percent": 65 } ],
  "passcode": "123456"
}
```
Response `data`: `{ "game_code": "scratch_card", "version": 2 }`

- ลำดับขั้นตามลำดับใน array (ตัวแรก = ขั้น 1) · ไม่รับ field ที่ไม่ใช่ของเกมนั้น (`422`)
- ต้องส่ง `passcode` ของคนที่กดบันทึก (RATE-08 · `RequirePasscode` — ผิด / ครบ 5 ครั้งตามกติกา auth)

Error: `422`, `401204`, `401205`, `406101`, `406102`, `406401`

### POST /api/v1/bo/pr/rate/history

Request: `{ "game_code": "coin_toss", "page": 1, "limit": 20 }`

Response `data` (`response.Page`):
```json
{
  "data": [ { "version": 3, "min_house_edge_percent": 5, "created_at": "2026-10-08T10:00:00+07:00", "created_by": "superadmin1", "ip": "203.0.113.10", "rollback_from": 0 } ],
  "total_count": 3
}
```
- เรียงเวอร์ชันใหม่ → เก่า · ใช้เป็น log คนแก้ (RATE-08) · ไม่ส่ง `actor_id` / `request_id` ออกไป

### POST /api/v1/bo/pr/rate/version

Request: `{ "game_code": "coin_toss", "version": 2 }` · Response `data` = รูปแบบเดียวกับ `/rate/detail` · ไม่พบ `406402`

### POST /api/v1/bo/pr/rate/rollback

Request:
```json
{ "game_code": "coin_toss", "version": 2, "passcode": "123456" }
```
Response `data`: `{ "game_code": "coin_toss", "version": 4, "rollback_from": 2 }`

- สร้างเวอร์ชันใหม่ที่ค่าเท่าเวอร์ชันที่ระบุ · `/rate/history` แสดง `rollback_from` (ไม่ได้ย้อน = `0`)
- ระบุเวอร์ชันที่ใช้อยู่ = `422` · ไม่พบเวอร์ชัน `406402`

Error: `422`, `401204`, `401205`, `406101`, `406401`, `406402`

## 6. Schema (migration ใหม่)

```sql
CREATE TABLE game_rate_versions (            -- 1 แถวต่อการบันทึก 1 ครั้ง (RATE-05) · ห้าม UPDATE / DELETE
    id                 BIGSERIAL    PRIMARY KEY,
    game_code          VARCHAR(50)  NOT NULL,
    version            INT          NOT NULL,
    win_bp             INT          CHECK (win_bp BETWEEN 0 AND 10000),   -- Scratch Card เท่านั้น
    actor_type         VARCHAR(10)  NOT NULL CHECK (actor_type IN ('AGENT','SUB')),   -- log คนแก้ (RATE-08)
    actor_id           BIGINT       NOT NULL,
    created_by         VARCHAR(71)  NOT NULL,   -- username ณ เวลานั้น
    ip                 VARCHAR(45),
    request_id         VARCHAR(64),
    rollback_from      INT          NOT NULL DEFAULT 0,   -- ย้อนมาจากเวอร์ชันไหน (RATE-05A) · 0 = บันทึกปกติ
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_game_rate_versions UNIQUE (game_code, version)          -- กันบันทึกพร้อมกันได้เลขซ้ำ + index หาเวอร์ชันล่าสุด
);

CREATE TABLE game_rate_steps (               -- Coin Flip / RPS
    version_id     BIGINT  NOT NULL REFERENCES game_rate_versions(id),
    step           SMALLINT NOT NULL CHECK (step BETWEEN 1 AND 10),
    multiplier_x100 INT    NOT NULL CHECK (multiplier_x100 >= 100),
    win_bp         INT     NOT NULL CHECK (win_bp BETWEEN 0 AND 10000),
    draw_bp        INT     NOT NULL DEFAULT 0 CHECK (draw_bp BETWEEN 0 AND 9900),
    PRIMARY KEY (version_id, step)
);

CREATE TABLE game_rate_prizes (              -- Scratch Card
    version_id      BIGINT NOT NULL REFERENCES game_rate_versions(id),
    multiplier_x100 INT    NOT NULL CHECK (multiplier_x100 > 0),
    share_bp        INT    NOT NULL CHECK (share_bp BETWEEN 0 AND 10000),
    PRIMARY KEY (version_id, multiplier_x100)
);
```
- บันทึก: tx เดียว · หาเลขเวอร์ชันล่าสุด + 1 · ชนกัน (`uq_game_rate_versions`) = ลองใหม่ 1 ครั้งแล้วค่อยตอบ error
- เวอร์ชันปัจจุบัน = `MAX(version)` ของเกม (ใช้ index ของ unique)

## 7. Test cases

| Rule ID | Input | Expected |
|---|---|---|
| RATE-02 | Company / Share / Agent เรียกทุกเส้น | `402303` |
| RATE-02 | sub ของ Superadmin ไม่มีสิทธิ์ `rate` · มี `view` แล้ว update | `402303` ทั้งคู่ |
| RATE-03 | ตัวคูณ `1.905` | `422` |
| RATE-04 | `win_percent` `50.125` | `422` |
| RATE-05 | บันทึก 2 ครั้ง | เวอร์ชัน 2 และ 3 · เวอร์ชัน 2 ยังอ่านได้ค่าเดิม |
| RATE-05 | บันทึกพร้อมกัน 2 request | ได้เวอร์ชันไม่ซ้ำ ทั้งคู่สำเร็จ |
| RATE-05A | มี v1–v3 ย้อนไป v2 | ได้ v4 ค่าเท่า v2 · `rollback_from = 2` · v3 ไม่เปลี่ยน |
| RATE-05A | ย้อนไปเวอร์ชันที่ใช้อยู่ · เวอร์ชันไม่มี · ไม่ส่ง passcode | `422` · `406402` · `422` |
| RATE-07 / RATE-13 | Coin Flip ขั้น 3 ตั้ง % ชนะเกิน `max_win` | `406101` · msg บอก `steps[2]` · ไม่มีเวอร์ชันใหม่ |
| RATE-10 | ส่ง 9 ขั้น | `422` |
| RATE-11 | ตัวคูณขั้น 4 ≤ ขั้น 3 | `422` (msg บอก `steps[3].multiplier`) |
| RATE-21 | RPS `win 60 · draw 40` | `422` |
| RATE-22 | RPS ขั้น 1 ชนะ 34 · เสมอ 30 | `reach_1 = 48.57` |
| RATE-32 | Scratch Card share รวม 99.5 | `406102` |
| RATE-33 | Scratch Card W 50 กับตารางตัวอย่างหัวข้อ 3 | `house_edge = 22.75` · `max_win = 64.72` |
| RATE-33 | W 65 กับตารางตัวอย่าง | `406101` |
| RATE-31 | ตัวคูณซ้ำ 2 แถว · 0 แถว · 11 แถว | `422` |
| RATE-08 | บันทึกโดยไม่ส่ง `passcode` · passcode ผิด | `422` · `401204` |
| RATE-08 | sub `superadmin1@staff` บันทึก | แถวเวอร์ชันมี `actor_type = SUB` · `created_by = superadmin1@staff` · `ip` · `request_id` |
| RATE-09A | ระบบใหม่ ยังไม่เคยตั้ง | `/rate/list` ทุกเกม `version = 0` · `/rate/detail` `steps = []` |
| — | `game_code` ไม่มีในระบบ | `406401` |
| RATE-01 | ส่ง `prizes` ให้ `coin_toss` | `422` |

## 8. Error codes (`06`)

| Code | HTTP | ความหมาย |
|---|---|---|
| 406101 | 200 | ตั้งค่านี้แล้วร้านขาดทุน (msg บอกขั้น / แถว และค่าสูงสุดที่ตั้งได้) |
| 406102 | 200 | สัดส่วนรางวัลรวมต้องเท่ากับ 100% |
| 406401 | 200 | ไม่พบเกมนี้ |
| 406402 | 200 | ไม่พบเวอร์ชันนี้ |

## 9. ส่งต่อให้ module อื่น

- ยอดแทงสูงสุด และขั้นต่ำ 1 บาท / บาทเต็ม ใช้กับทุกสกุลไหม — ถามตอนทำ module เดิมพัน
