# Review Game win/loss rate settings (`game_rate` · เมนู ⑥) — lead: zerph · 2026-10-12 · ✅ ตัดสินแล้ว

อ้างอิง: `docs/modules/game_rate.md` (อนุมัติ 2026-10-08 · branch `boiledegg/bo/game-rate` `ca85b8c`) · spec + โค้ด: **boiledegg** · ไฟล์นี้คือการตัดสินของ lead — แก้ spec ก่อน แล้วค่อย implement ใน branch ใหม่จาก dev และเปิด PR ให้ lead review
หมายเหตุ branch: `boiledegg/bo/game-rate` มี commit test-infra เก่า `3d5aef8` ที่ไม่อยู่ใน dev → แตก branch ใหม่จาก dev แล้ว cherry-pick เฉพาะ 3 commit เอกสาร (`dd9a247` `ea92f09` `ca85b8c`)

## การตัดสินของ lead (ตามแนะนำทั้งหมด)
| # | เรื่อง | ตัดสิน |
|---|---|---|
| RS1 | ใครตั้ง | **ค่ากลางชุดเดียว Superadmin (+ sub ที่มีเมนู `rate`) ตั้ง** · ไม่ไล่สาย ไม่มีค่าต่อบัญชี · H2/H3 ไม่เกี่ยว |
| RR1 | ทศนิยม | **float (DOUBLE) ไม่เกิน 4 ตำแหน่ง** ตามกฎข้อ 9 (แทน INT ×100 / bp) · RNG `0–999,999` · ตรวจสูตรด้วยเลขแม่นยำ (`big.Rat`) ไม่ใช่ float |
| RR2 | house edge ขั้นต่ำ | **≥ 1% ทุกขั้น** (ทุกเกม) |
| RR3 | เพดาน | **ตัวคูณ ≤ 1,000** · `win_percent > 0` |
| RR4 | แก้ทับกัน | request ส่ง **`base_version`** · ไม่ตรงเวอร์ชันปัจจุบัน = error ใหม่ **`406301`** "มีคนแก้ไปแล้ว กรุณาโหลดใหม่" (ทั้ง update และ rollback) |
| RR5 | บันทึกค่าเดิม / rollback ไปเวอร์ชันที่ค่าเท่าปัจจุบัน | **200 ไม่สร้างเวอร์ชันใหม่** (`created: false`) แทน `422` |
| RR6 | หยุดเกมด่วน | ยืนยันว่า Superadmin ปิด `status_game` ของตัวเองแล้วทุกสายปิดตาม → ใช้เป็นสวิตช์ฉุกเฉิน · เขียนใน runbook |
| RN1 | path | ทุกเส้น `POST` ใต้ `/bo/pr/rate/`: RE1 `list` · RE2 `detail/get` · RE3 `detail/check` · RE4 `detail/update` (+ passcode) · RE5 `versions/list` · RE6 `versions/detail/get` · RE7 `versions/rollback` (+ passcode) |
| RN2 | เส้นตรวจค่า | **เพิ่ม RE3 `detail/check`** — ตรวจ + คำนวณ RTP / house edge โดยไม่บันทึก (หน้าบ้านไม่ต้องเขียนสูตรเอง) · สิทธิ์ `rate` view |
| RP1 | ใครทำ | **boiledegg ทำทั้ง spec และโค้ด (~6.5 pd)** (lead ตัดสิน — ไม่ใช่ maofoy) · **ต้องเสร็จก่อนเริ่ม engine / module เดิมพัน** (Bet ต้องใช้ `rate_version_id`) |

## รายการแก้

### 7.1 Spec `docs/modules/game_rate.md` (boiledegg · branch ใหม่จาก dev)
- S1 หัวเอกสาร: บันทึกการตัดสินของ lead 2026-10-1x (RS · RR · RN) + ระบุ path ใหม่
- S2 RATE-02: สิทธิ์ `rate` ทุกเส้น (A5) · ตัด `402311` · ลำดับ middleware สิทธิ์ก่อน passcode · ค่ากลางชุดเดียว ไม่ไล่สาย ไม่มีค่าต่อบัญชี (RS1 · H2 ไม่เกี่ยว)
- S3 RATE-01A: อ้าง `status_game` (ตัด `pt`) · รายชื่อเกมอยู่ `app/core/game`
- S4 RATE-03 / RATE-04 / หัวข้อ 4 แถว 1: float ≤ 4 ตำแหน่งตามกฎข้อ 9 · RNG `0–999,999` · การตรวจใช้เลขแม่นยำ (RR1)
- S5 RATE-07: ขั้นต่ำ house edge (RR2) · RATE-11 / 21 / 30 / 31: เพดานตัวคูณ + `win_percent > 0` (RR3)
- S6 RATE-05 / 05A: `base_version` + `406301` (RR4) · ค่าเท่าปัจจุบัน = 200 `created: false` (RR5)
- S7 หัวข้อ 5: path RE1–RE7 · ทุกเส้น POST · ตัวอย่าง JSON เต็ม (10 ขั้น) · ตาราง field · RE3 (ถ้าอนุมัติ RN2) · `created_*` แทน `updated_*` · ค่าคำนวณ `Round4`
- S8 หัวข้อ 6 schema: `DOUBLE PRECISION` + CHECK ช่วง · `game_rate_prizes` PK `(version_id, row_no)` · `ip` / `request_id` `NOT NULL DEFAULT ''` · เขียนว่าไม่ใช้ MN2
- S9 หัวข้อ 7 test: ทศนิยม 5 ตำแหน่ง `422` · house edge < ขั้นต่ำ · ตัวคูณ 1,001 · win 0 · `base_version` เก่า `406301` · บันทึกค่าเดิม `created: false` · rollback ไปเวอร์ชันที่ไม่ผ่านกฎใหม่ `406101` · Company / ADMIN / sub ไม่มีสิทธิ์ `402303`
- S10 หัวข้อ 8: เพิ่ม `406301` · หัวข้อ 9: ย้าย RATE-09B (บาทเต็ม) + RH1–RH4 ไปเป็น open item ของ module เดิมพัน

### 7.2 งานโค้ด (boiledegg · branch `feature/game_rate-settings` จาก dev)
- C1 `app/core/game`: รายชื่อเกม + ย้ายจาก `pt_rule.go:36` (commit แรก แจ้ง boiledegg)
- C2 `app/core/game_rate`: `ValidateCoin/RPS/Scratch` + `Compute…` (reach / house edge / max win / avg) ด้วย `big.Rat` · ค่าคงที่ขั้นต่ำ/เพดาน · table test ครอบตัวอย่าง diagram ทุกตัว (5% · 13.36% · 48.57% · 22.75% · 64.72%)
- C3 migration `game_rate_versions` / `game_rate_steps` / `game_rate_prizes` (timestamp หลังไฟล์ล่าสุดบน dev — MA6) · models · repository (`db` param แรก · ไม่มี tx ใน repo)
- C4 service: tx เดียว · `base_version` · ค่าเดิม → `created: false` · insert batch · audit (`actor_type/id`, `created_by`, `ip`, `request_id`) · rollback
- C5 DTO `Validate()` ด้วย `if` · ≤ 4 ตำแหน่ง · ไม่รับ field ของเกมอื่น (ยกเว้น `passcode`) · `ParseBodyNoNull` · controller + routes RE1–RE7 · `pkg/apperr/game_rate_errors.go` (`06`) + `ERROR_CODES.md`
- C6 ฟังก์ชันสำหรับ module เดิมพัน: `CurrentVersionService(ctx, tx, game)` (version 0 = error ให้ engine ปฏิเสธ) · `ByVersionIDService(ctx, id)` + cache ใน memory ตาม `version_id`
- C7 integration test ตาม S9 + ยิง update พร้อมกัน 2 request (คนหนึ่งได้ `406301`) · แนบผล `make test-integration`
- C8 เอกสาร: `docs/frontend/game_rate.md` (ตัวอย่างค่าเริ่มต้นจาก diagram ให้ tester ตั้งบน dev/uat) · `MIGRATION_STATUS.md` · แจ้ง frontend

### 7.3 ส่งต่อ module เดิมพัน / engine (open item — ไม่ต้องตัดสินตอนนี้)
- RH1 Bet เก็บ `rate_version_id` · Streak ทุกขั้นอ่านค่าจาก id นั้น · version 0 → ปฏิเสธ Bet ("เกมนี้ไม่เปิดให้บริการ")
- RH2 ยอดแทงบาทเต็ม ขั้นต่ำ 1 (RATE-09B 8 ต.ค.) vs กฎข้อ 9 float · ช่วงยอดแทง 10–500 ใน diagram · เพดานยอดจ่ายต่อบิล (เกี่ยวกับ RR3)
- RH3 หน้าเกมของ Member ต้องได้ตารางตัวคูณ (Coin/RPS 10 ขั้น · รางวัล Scratch) — **ควรให้เห็นแค่ตัวคูณ ไม่ส่ง `win_percent` / `share_percent`** · เป็นเส้นฝั่งผู้เล่นของ module เดิมพัน
- RH4 Scratch ตรง 2 ช่อง แบ่งครึ่ง "ช่องแรกปัดลง" (RATE-35) → ปัดที่ 4 ตำแหน่ง (`Round4`) ไม่ใช่สตางค์ · Streak ค้างเกิน 1 วัน CASH OUT อัตโนมัติ (RATE-09C) — job ของ engine

## การตัดสินที่ผ่าน boiledegg (spec หัวข้อ 9 Q1–Q8) — lead ทบทวน 2026-10-13
| # | เรื่อง | lead |
|---|---|---|
| Q1 | หยุดเกมด่วน (RR6) — spec เดิม: ไม่เพิ่มเส้น ให้ปิดที่ Company ทีละตัว | **กลับมาทำสวิตช์ปิดเกม** ให้ Superadmin ปิด / เปิดเกมทั้งระบบได้ในที่เดียว (ใช้ตอนปิดปรับปรุงเกม) — ข้อเสนอ: เส้นใหม่ใต้ `/bo/pr/rate/` เช่น `status/update` `{ game_code, status_game }` (สิทธิ์ `rate` edit + passcode) ตั้ง `status_game` ของ Superadmin แล้วผลไหลลงทั้งสายผ่านการเช็คทั้งสาย (`IsGameOpen`) · `rate/list` แสดงสถานะ · log การปิด/เปิด · เอนจิ้น / Player ต้องเช็คก่อนรับเดิมพัน (RH) — boiledegg เสนอรายละเอียดก่อน implement |
| Q2 | `detail/check` ตอบ `valid` + `issues` | รับ |
| Q3 | `max_win_percent` = ค่าสูงสุดที่ house edge ≥ 1% | รับ |
| Q4 | `versions/list` เพิ่ม `same_as_current` · `win_percent` (Scratch) · `changes` | รับ |
| Q5 | error ไม่มี `data` — ข้อมูลอยู่ใน `msg` | รับ |
| Q6 | `win_percent` ≤ 50 ทุกเกม | **lead ตัดสินเอง (ยืนยัน 2026-10-13)** |
| Q7 | RPS ไม่มีผลเสมอ (แพ้ = 100 − ชนะ · `draw_percent` = 422) | **lead ตัดสินเอง (ยืนยัน 2026-10-13)** · เอนจิ้นต้องไม่มีผลเสมอ |
| Q8 | ไม่มีอัตราชนะรวม — กล่องสรุปใช้ `min_house_edge_*` / `max_payout_percent` | รับ |

## รอบ 3 — ข้อความ error (lead zerph 2026-10-14 · branch `4815765`)
| # | เรื่อง | ผล |
|---|---|---|
| Q9 | ชื่อ field ในข้อความ error | **`field` คง path เดิม** (เช่น `steps[0].win_percent` — หน้าบ้านใช้ชี้ช่อง) · **`message` / `msg` ใช้ชื่อที่คนอ่านเข้าใจ** แบบ `utils.FieldLabel` เหมือน agent / member management · array มีเลขแถว: `steps[i]` → "ขั้นที่ i+1" · `prizes[i]` → "รางวัลที่ i+1" (เพิ่มในตาราง label ที่เดียว) · ตัวอย่าง: `"ขั้นที่ 1 · โอกาสชนะ ตั้งได้ไม่เกิน 49.7487% (ร้านต้องได้อย่างน้อย 1%)"` · แก้ตัวอย่างใน spec + `docs/frontend/game_rate.md` |

## รอบ 4 — ผลตรวจโค้ด `4815765` (lead zerph 2026-10-14 · รายงาน verify-game-rate)
ยังไม่ merge — แก้ 3 ข้อนี้ใน branch เดิมแล้วส่งตรวจอีกรอบ:

| # | เรื่อง | ผล |
|---|---|---|
| M1 | Q9 ยังไม่ทำ — msg `422` / `406101` / `406102` และ RE3 `issues[].message` ขึ้นต้นด้วย path ดิบ | **ต้องแก้** ตาม Q9 (`utils.FieldLabel` + "ขั้นที่ i+1" / "รางวัลที่ i+1") + แก้ตัวอย่างใน spec และ `docs/frontend/game_rate.md` |
| G-S1 | RE8: Superadmin ไม่มีแถว `agent_game_settings` ของเกมนั้น → ตอบ `changed: false` แต่เกมไม่ได้ปิด (สวิตช์ฉุกเฉินล้มเงียบ) | **upsert แถวที่ขาดใน tx เดียวกันแล้วปิด/เปิดตามที่สั่ง** · log `UPDATE_GAMES` ตามปกติ · เพิ่ม test เกมที่ไม่มีแถว |
| G-S2 | ผลรวม `prizes[].payout_percent` อาจต่างจาก `payout_percent` ของใบ ±0.0001 (ปัดทีละแถว) | **แก้เอกสาร** (RATE-09 / RATE-34 + frontend doc) ว่าอาจต่างได้ไม่เกิน 0.0001 จากการปัด — ใช้แสดงผลอย่างเดียว · ไม่ทำ largest-remainder |

nit 4 ข้อใน verify report ทำหรือไม่ก็ได้ · ก่อนส่งตรวจ: merge dev ล่าสุด + integration test ผ่าน
