# game_rate (อัตราแพ้ชนะ) — Spec

- สถานะ: **APPROVED** (lead (zerph) 2026-10-08) · **แก้ตาม lead review 2026-10-12** ([docs/reviews/game-rate.md](../reviews/game-rate.md) — RS1 · RR1–RR6 · RN1–RN2 · RP1) — **ตัดสินครบ 2026-10-12 (Q2–Q8 · Q1 ทำทีหลัง) · implement C1–C8 เสร็จ รอ lead review** (branch `boiledegg/bo/game_rate-settings`)
  - RS1 ค่ากลางชุดเดียว Superadmin (+ sub ที่มีเมนู `rate`) ตั้ง · ไม่ไล่สาย ไม่มีค่าต่อบัญชี
  - RR1 ตัวคูณ / % เป็น float ทศนิยมไม่เกิน 4 ตำแหน่ง (กฎข้อ 9 · แทน INT ×100 / bp) · RNG `0–999,999` · ตรวจสูตรด้วยเลขแม่นยำ (`big.Rat`)
  - RR2 house edge ≥ 1% ทุกจุดที่หยุดได้ · RR3 ตัวคูณ ≤ 1,000 · `win_percent > 0` · **`win_percent` ≤ 50 ทุกเกม (2026-10-12)**
  - RR4 `base_version` กันแก้ทับ (`406301`) · RR5 ค่าเท่าปัจจุบัน = 200 `created: false` · RR6 หยุดเกมด่วนด้วย `status_game`
  - RN1 path ใหม่ RE1–RE7 ทุกเส้น `POST` · RN2 เส้นตรวจค่า `detail/check` · RP1 boiledegg ทำ spec + โค้ด · ต้องเสร็จก่อน engine / module เดิมพัน
  - **Q7 (lead ตัดสิน 2026-10-12): RPS ไม่มีเสมอ** — ต่อขั้นมีแค่ `multiplier` + `win_percent` · แพ้ = 100 − ชนะ · สูตรเดียวกับ Coin (แทนตัวอย่าง "ชนะ 34 · เสมอ 30" ใน diagram)
  - **ตัดสิน 2026-10-12 (ผ่าน boiledegg):** Q1 เส้นปิด `status_game` ของ Superadmin เอง ทำทีหลัง · Q2 `detail/check` ตอบ `valid` + `issues` · Q3 `max_win` = ค่าสูงสุดที่ house edge ≥ 1% · Q4 RE5 `changes` / `same_as_current` / `win_percent` · Q5 error ไม่มี `data` (ข้อมูลอยู่ใน `msg`) · Q6 **เพดาน `win_percent` 50 ทุกเกม** (เปลี่ยน 2026-10-12 · แทน ≤ 100) · Q8 ไม่มีอัตราชนะรวม — กล่องสรุปใช้ `min_house_edge_*` / `max_payout_percent`
- ที่มาของ rule: diagram ของ lead (`Coin Flip Flow.html`, `Rock Paper Scissors Flow.html`, `Scratch Card Flow.html`) + คำตอบของ boiledegg 2026-10-08 + lead review 2026-10-12 · ไม่ได้อ้างโค้ดหรือเอกสารของโปรเจกต์เก่า
- ชื่อ module ในโค้ด: `game_rate` (`controllers/` `dto/` `service/` `core/game_rate`) · รายชื่อเกม `core/game` · path `/api/v1/bo/pr/rate/...` · error module id `06` (ตรงกับเมนู ⑥)

> เอกสารนี้คือ source of truth ของ module — โค้ดและ test ต้องตรงกับเอกสารนี้ ถ้าไม่ตรงให้แก้โค้ด
> หรือแก้เอกสารผ่านการอนุมัติใหม่ ห้ามอ้างเอกสารของโปรเจกต์เก่า

## 1. หน้าที่ของ module

เก็บและแก้ "อัตราแพ้ชนะ" ของแต่ละเกม (ตัวคูณ + โอกาสชนะ) ที่เอนจิ้นเกมใช้ตัดสินผล · เมนู ⑥ ของหลังบ้าน

- **ค่ากลางชุดเดียวทั้งระบบ** (lead RS1): Superadmin และ sub ของ Superadmin ที่มีเมนู `rate` เป็นผู้ตั้ง · ไม่ไล่สาย ไม่มีค่าต่อบัญชี / ต่อ Company · กติกา PT / Share Master (H2 / H3 ของ agent_management) ไม่เกี่ยว
- module นี้**ไม่ได้**สุ่มผลหรือจ่ายเงิน — แค่เก็บค่า · เอนจิ้นเกมและ module เดิมพันอ่านค่าผ่านฟังก์ชันใน service (หัวข้อ 5.8)

## 2. คำศัพท์

| คำ | ความหมาย |
|---|---|
| ขั้น (step) | รอบที่ Member ทายต่อใน Streak (Coin Flip / Rock Paper Scissors) · มี 10 ขั้นตายตัว |
| Streak | Member ทายต่อเพื่อสะสมเงิน · ปิด Streak = เล่นครั้งเดียว ใช้ค่าของขั้น 1 |
| ตัวคูณ (`multiplier`) | เงินที่ได้ = ยอดแทง × ตัวคูณ (ไม่รวมทบจากขั้นก่อน) |
| `win_percent` | โอกาสชนะของขั้นนั้น (Coin Flip / RPS) หรือของบัตร 1 ใบ (Scratch Card) |
| `reach_percent` (R) | โอกาสที่ Member ไปถึงและชนะขั้นนั้น = ผลคูณของโอกาสชนะจริงตั้งแต่ขั้น 1 ถึงขั้นนั้น |
| `house_edge_percent` | ร้านได้เฉลี่ย (% ของยอดแทง) = 100% − จ่ายคืนคาดหวัง (RTP) |
| `max_win_percent` | `win_percent` สูงสุดของขั้น / บัตรที่ยังได้ house edge ≥ ขั้นต่ำ (RATE-07) |
| เวอร์ชัน (`version`) | ทุกการบันทึกที่ค่าเปลี่ยนสร้างชุดค่าใหม่ 1 เวอร์ชัน ห้ามแก้เวอร์ชันเดิม · ยังไม่เคยตั้ง = `0` |
| `base_version` | เวอร์ชันที่หน้าจอโหลดมาก่อนแก้ — ใช้กันคนสองคนแก้ทับกัน (RATE-05B) |
| ตารางรางวัล (Scratch Card) | รายการรางวัลของบัตรที่ชนะ: ตัวคูณ + สัดส่วนในบัตรที่ชนะ |

## 3. Business rules

### ทั่วไป

| ID | Rule |
|---|---|
| RATE-01 | ตั้งค่าแยกต่อเกม (ลำดับ `scratch_card`, `coin_toss`, `rock_paper_scissors` — 2026-10-12) · ชุดเดียวใช้กับ**ทุกยอดแทง** (ค่าเป็น "เท่า" ของยอดแทง) · **ชุดเดียวใช้กับทุกสกุลเงิน** (รีวิว 2026-10-08) · **ค่ากลางชุดเดียวทั้งระบบ** ไม่มีค่าต่อบัญชี (lead RS1) |
| RATE-01A | **รายชื่อเกมอยู่ในโค้ด** `app/core/game` (รายการเดียวกับที่ `pt` / `status_game` ของ agent_management ใช้ — ย้ายจาก `core/agent_management`) · ตอนนี้ 3 เกม · เพิ่มเกมใหม่ = แก้โค้ดพร้อมเอนจิ้นของเกมนั้น · ไม่มีเส้นสร้างเกม · ปิดเกมรายบัญชีใช้ `status_game` ของ agent_management (`agents/detail/update`) · **รองรับเกมใหม่:** แต่ละเกมผูกกับ**รูปแบบอัตรา** (`core/game_rate` `gameSpecs`) — `steps` (ทายต่อเป็นขั้น · มีจำนวนขั้นของเกม · Coin / RPS) หรือ `prizes` (โอกาสชนะของใบ + ตารางรางวัล · Scratch) · เกมใหม่ที่ใช้รูปแบบเดิม = เพิ่มรหัสใน `core/game` + 1 บรรทัดใน `gameSpecs` ไม่ต้องแก้ API / ตาราง / service · รูปแบบใหม่ = เพิ่ม Kind + กฎ / สูตร (และตารางลูกถ้าโครงต่างกัน) · test บังคับว่าทุกเกมใน `core/game` มีรูปแบบอัตรา |
| RATE-01B | **หยุดเกมด่วน** (lead RR6): ปิด `status_game` ของเกมนั้นที่ Superadmin → ทุกสายปิดตาม (MGMT-20 · เช็คทั้งสาย) · ใช้เป็นสวิตช์ฉุกเฉินแทนการแก้อัตรา · เขียนใน runbook (`docs/DEPLOYMENT.md`) · **ตอนนี้ยังไม่มีเส้นให้ Superadmin ปิด `status_game` ของตัวเอง — ทำทีหลัง (Q1) · ระหว่างนี้ปิดที่ Company ทีละตัว (`agents/detail/update`)** |
| RATE-02 | **สิทธิ์ `rate` ทุกเส้น** (lead A5 · RS1): ดู = `rate` view · แก้ / ย้อน = `rate` edit · บัญชีหลักที่ได้เมนู `rate` มีแค่ Superadmin · Company / Share / Agent / ADMIN / sub ที่ไม่มีสิทธิ์ = `402303` · sub เรียกได้ตามสิทธิ์ (ไม่มี `402311`) · ลำดับ middleware: สิทธิ์ (`RequirePermission`) **ก่อน** passcode (`RequirePasscode`) |
| RATE-03 | ตัวคูณ: JSON number **ทศนิยมไม่เกิน 4 ตำแหน่ง** (lead RR1 · กฎข้อ 9) · เก็บ `DOUBLE PRECISION` ค่าที่ปัด 4 ตำแหน่งแล้ว · **≥ 1 และ ≤ 1,000** (Coin / RPS) · **> 0 และ ≤ 1,000** (รางวัล Scratch) (lead RR3) |
| RATE-04 | %: JSON number **ทศนิยมไม่เกิน 4 ตำแหน่ง** · เก็บ `DOUBLE PRECISION` · `win_percent` **> 0** และ **≤ 50 ทุกเกม** (lead RR3 · เพดาน 50 ตัดสิน 2026-10-12 · `MaxWinPercent` ใน `core/game_rate`) · RNG ของเอนจิ้นสุ่ม `0–999,999` (1 ค่า = 0.0001%) · การตรวจสูตร (RATE-07, RATE-32) ใช้**เลขแม่นยำ** (`big.Rat` จากค่าทศนิยม 4 ตำแหน่ง) ไม่ใช่ float (lead RR1) |
| RATE-05 | บันทึก = **แทนทั้งชุดของเกมนั้น** และสร้างเวอร์ชันใหม่ (เลขเวอร์ชันต่อเกมเพิ่มทีละ 1) · เวอร์ชันเดิมห้ามแก้ / ลบ · ไม่ใช่แบบ section (ไม่ใช้กฎ "อย่างน้อย 1 section" — MN2) |
| RATE-05A | **ย้อนเวอร์ชัน** (`versions/rollback`): ส่ง `game_code` + `version` ที่จะย้อนไป + `base_version` · ระบบคัดลอกค่าของเวอร์ชันนั้นเป็น**เวอร์ชันใหม่** (มี v1–v3 ย้อนไป v2 = สร้าง v4 ค่าเท่า v2) · ตรวจ RATE-07 / RATE-32 **ด้วยกฎปัจจุบัน** — เวอร์ชันเก่าที่ไม่ผ่านกฎใหม่ = `406101` / `406102` · ต้อง passcode และ `rate` edit · log เก็บ `rollback_from` |
| RATE-05B | **กันแก้ทับ** (lead RR4): `detail/update` และ `versions/rollback` ต้องส่ง `base_version` = เวอร์ชันปัจจุบันที่หน้าจอโหลดมา (ยังไม่เคยตั้ง = `0`) · ไม่ตรงเวอร์ชันปัจจุบันตอนบันทึก = `406301` · `msg` บอกเวอร์ชันล่าสุดและคนแก้ เช่น "มีคนแก้ไปแล้ว (v59 โดย superadmin1@staff) กรุณาโหลดใหม่" (Q5 — error ไม่มี `data` · ต้องการวันที่ให้เรียก RE2) · ยิงพร้อมกัน 2 request ด้วย `base_version` เดียวกัน = สำเร็จ 1 อีกอัน `406301` |
| RATE-05C | **ค่าเท่าปัจจุบัน** (lead RR5): บันทึกค่าเดิมทั้งชุด หรือย้อนไปเวอร์ชันที่ค่าเท่าเวอร์ชันปัจจุบัน = `200` **ไม่สร้างเวอร์ชันใหม่** · response `created: false` + `version` ปัจจุบัน · เทียบค่าหลังปัด 4 ตำแหน่ง · Scratch เทียบตารางรางวัลตามลำดับแถว |
| RATE-06 | ค่าใหม่มีผลกับ**เดิมพันใหม่เท่านั้น** · เดิมพันที่เริ่มแล้ว (รวม Streak ที่ค้างอยู่) ใช้เวอร์ชันเดิมจนจบ · module เดิมพันเก็บ `rate_version_id` คู่กับเดิมพันทุกครั้ง (หัวข้อ 9 RH1) |
| RATE-07 | **ร้านต้องได้ house edge ≥ 1% ทุกจุดที่ Member หยุดได้** (lead RR2 · แทน "> 0" เดิม) — Coin / RPS ทุกขั้น · Scratch ทั้งใบ · ไม่ผ่านที่จุดไหน บันทึกไม่ได้ทั้งชุด (`406101`) และ msg บอกขั้น / field และ `win_percent` สูงสุดที่ตั้งได้ · ค่าคงที่ขั้นต่ำ / เพดานอยู่ใน `core/game_rate` |
| RATE-08 | **บันทึก / ย้อนต้องยืนยัน passcode** (`RequirePasscode`) · ทุกเวอร์ชันเก็บ log คนแก้: `created_by` (username ณ เวลานั้น · sub = `owner@name`), `actor_type` / `actor_id`, `ip`, `request_id`, `created_at` · ตารางเวอร์ชันคือ log (ห้ามแก้ / ลบ) ดูได้ที่ `versions/list` |
| RATE-09 | response แสดงค่าที่คำนวณ (`reach_percent`, `payout_percent`, `house_edge_percent`, `max_win_percent`, `avg_multiplier`, `per_ticket_percent` · สรุปทั้งเกม `min_house_edge_percent` + `min_house_edge_step` + `max_payout_percent` — Coin / RPS) **ปัด 4 ตำแหน่ง** (`utils.Round4`) · คำนวณด้วยเลขแม่นยำแล้วค่อยปัด · **ยกเว้น `max_win_percent` ปัดลง** (ค่าที่แสดงต้องตั้งได้จริง — ปัดขึ้นแล้วกรอกค่านั้น house edge จะต่ำกว่า 1%) · หน้าบ้านปัดแสดงเอง |
| RATE-09A | **ไม่มีค่าเริ่มต้น**: เกมที่ยังไม่เคยตั้งค่า = `version = 0` · เอนจิ้นต้อง**ไม่รับเดิมพัน**ของเกมนั้นจนกว่าจะบันทึกครั้งแรก (หัวข้อ 5.8 · RH1) · ค่าตัวอย่างสำหรับ tester อยู่ใน `docs/frontend/game_rate.md` |

### Coin Flip (`coin_toss`)

| ID | Rule |
|---|---|
| RATE-10 | 10 ขั้นตายตัว (จำนวนขั้นของเกมใน `gameSpecs`) ต้องส่งครบ 10 ขั้น · แต่ละขั้น `multiplier` + `win_percent` |
| RATE-11 | `multiplier` 1–1,000 และ**มากกว่าขั้นก่อนหน้าเสมอ** · `win_percent` > 0 และ ≤ 50 |
| RATE-12 | RANDOM (ระบบเลือกฝั่งให้) ใช้ `win_percent` เดียวกับเลือกฝั่งเอง |
| RATE-13 | `reach_n = win_1 × … × win_n` · `house_edge_n = 100% − reach_n × multiplier_n` ต้อง **≥ 1% ทุกขั้น** (Member CASH OUT ขั้นไหนก็ได้) · `max_win_n` = `win_n` สูงสุดที่ `house_edge_n` ยัง ≥ 1% = `(100% − 1%) ÷ (reach_{n−1} × multiplier_n)` (ขั้น 1: `reach_0 = 100%`) · **ไม่เกิน 50** (เพดาน `win_percent`) |

ตัวอย่าง (ค่าใน diagram): ขั้น 1 `1.90 / 50` → `house_edge = 5%` · ขั้น 2 `3.61 / 48` → `reach = 24%` · `house_edge = 100% − 24% × 3.61 = 13.36%`

### Rock Paper Scissors (`rock_paper_scissors`)

| ID | Rule |
|---|---|
| RATE-20 | 10 ขั้นตายตัว · แต่ละขั้น `multiplier` + `win_percent` · **ไม่มีเสมอ** (lead Q7 2026-10-12) · % แพ้ = 100 − ชนะ (ระบบไม่ส่งค่านี้) · ส่ง `draw_percent` = `422` |
| RATE-21 | `multiplier` 1–1,000 และมากกว่าขั้นก่อนหน้า · `win_percent` > 0 และ ≤ 50 (เหมือน RATE-11) |
| RATE-22 | สูตรเดียวกับ Coin (RATE-13): `reach_n = win_1 × … × win_n` · `house_edge_n = 100% − reach_n × multiplier_n` ≥ 1% ทุกขั้น · `max_win_n = 99% ÷ (reach_{n−1} × multiplier_n)` ไม่เกิน 50 |
| RATE-23 | เอนจิ้น RPS ต้อง**ไม่มีผลเสมอ** — ผลมีแค่ชนะ / แพ้ตาม `win_percent` (หัวข้อ 9 RH5) |

ตัวอย่าง: ใช้ชุดเดียวกับ Coin (หัวข้อ 5.2) · ขั้น 1 `1.9 / 50` → `house_edge = 5%`

### Scratch Card (`scratch_card`)

| ID | Rule |
|---|---|
| RATE-30 | `win_percent` (W) = โอกาสที่บัตร 1 ใบชนะ · > 0 และ ≤ 50 |
| RATE-31 | `prizes` = ตารางรางวัลของบัตรที่ชนะ · แต่ละแถว `multiplier` (> 0 · ≤ 1,000) + `share_percent` (> 0 · ≤ 100) · **1–10 แถว** · `multiplier` ห้ามซ้ำกัน · ลำดับแถวตามที่ส่ง (`row_no` 1–10) |
| RATE-32 | `share_percent` ทุกแถวรวมกันต้อง **= 100 พอดี** (ตรวจด้วยเลขแม่นยำ · `406102`) |
| RATE-33 | `avg_multiplier (A) = Σ(multiplier × share_percent ÷ 100)` · จ่ายคืนคาดหวัง `P = W × A` · `house_edge = 100% − P` ≥ 1% · `max_win_percent = 99% ÷ A` ไม่เกิน 50 |
| RATE-34 | % ต่อใบของแต่ละแถว `per_ticket_percent = W × share_percent ÷ 100` (ระบบคำนวณให้ใน response) |
| RATE-35 | ตรง 1 หรือ 2 ช่อง เอนจิ้นสุ่มเอง — **ไม่มีค่าให้ตั้ง** · การแบ่งรางวัลตรง 2 ช่องเป็นกฎของเอนจิ้น (หัวข้อ 9 RH4) |

ตัวอย่าง (ค่าใน diagram): W 50% · ตาราง 0.5×/35, 1×/35, 2×/18, 5×/10, 8×/2 → `A = 1.545` · `P = 77.25%` · `house_edge = 22.75%` · `max_win = 99% ÷ 1.545 = 64.0776%` → แสดง **50** (เพดาน)

## 4. สิ่งที่พบในโค้ดเก่า และการตัดสินใจ

ไม่ได้อ่านโค้ดเก่า — ใช้ diagram ของ lead และคำตอบของ boiledegg / lead

| # | ที่มา | การตัดสินใจ |
|---|---|---|
| 1 | diagram เก็บตัวคูณทศนิยม 4 ตำแหน่ง (`1.90^n`) | **float ทศนิยมไม่เกิน 4 ตำแหน่ง** (lead RR1 · แทน "2 ตำแหน่ง / INT ×100" ของ 2026-10-08) |
| 2 | diagram ตารางรางวัล Scratch Card ตัวคูณ `step 0.1, min 0` | ทศนิยม 4 ตำแหน่งตาม RATE-03 · ต้อง > 0 (รางวัล 0 เท่าไม่ใช่การชนะ) |
| 3 | diagram มี dropdown ยอดแทง 10 / 20 / 50 / 100 / 200 / 500 | ใช้แสดงตัวอย่างเงินเท่านั้น — ตั้งชุดเดียวใช้ทุกยอด (RATE-01) · ช่วงยอดแทงเป็นเรื่องของ module เดิมพัน (RH2) |
| 4 | diagram "เปลี่ยนค่าแล้วมีผลกับ Bet ใหม่ · บันทึกค่าคู่กับ Bet" | เก็บเป็นเวอร์ชัน (RATE-05) ให้เดิมพันอ้าง `rate_version_id` (RH1) |
| 5 | diagram มีค่าเริ่มต้นของทุกเกม | **ไม่ใส่ค่าเริ่มต้นในระบบ** (RATE-09A) · ใช้เป็นค่าตัวอย่างให้ tester ตั้งเองใน `docs/frontend/game_rate.md` |

## 5. Endpoints (lead RN1)

ใต้ `/api/v1/bo/pr/rate` · ทุกเส้น `POST` · `Authenticated` + `PassedGates()` + `RequirePermission(rate, …)` · เส้นที่บันทึกเพิ่ม `RequirePasscode()` หลังสิทธิ์ (RATE-02) ·
request ผ่าน `utils.ParseBodyNoNull` (ห้าม `null` · `422`) · ไม่รับ field ที่ไม่ใช่ของเกมนั้น (`422` — ยกเว้น `passcode`) · error ร่วมทุกเส้น: `401202`, `401203`, `401301`, `401302`, `401304`, `401306`, `401307`, `401311`, `402303`

| # | Path | สิทธิ์ | ใช้เมื่อ |
|---|---|---|---|
| RE1 | `/rate/list` | `rate` view | รายชื่อเกม + เวอร์ชันปัจจุบัน + house edge ต่ำสุด |
| RE2 | `/rate/detail/get` | `rate` view | ค่าปัจจุบันของ 1 เกม พร้อมค่าที่คำนวณ |
| RE3 | `/rate/detail/check` | `rate` view | ตรวจ + คำนวณค่าที่กำลังกรอก **โดยไม่บันทึก** (lead RN2) |
| RE4 | `/rate/detail/update` | `rate` edit + passcode | บันทึกค่าใหม่ของ 1 เกม (= เวอร์ชันใหม่) |
| RE5 | `/rate/versions/list` | `rate` view | รายการเวอร์ชันของ 1 เกม (log คนแก้) |
| RE6 | `/rate/versions/detail/get` | `rate` view | ค่าของเวอร์ชันที่ระบุ |
| RE7 | `/rate/versions/rollback` | `rate` edit + passcode | ย้อนไปใช้ค่าของเวอร์ชันที่ระบุ (= เวอร์ชันใหม่) |

### 5.1 RE1 · POST /api/v1/bo/pr/rate/list

Request: `{}` · Response `data`:
```json
[
  { "game_code": "scratch_card", "version": 1, "min_house_edge_percent": 22.75, "created_at": "2026-10-12T09:00:00+07:00", "created_by": "superadmin1@staff" }
  { "game_code": "coin_toss", "version": 3, "min_house_edge_percent": 5, "created_at": "2026-10-12T10:00:00+07:00", "created_by": "superadmin1" },
  { "game_code": "rock_paper_scissors", "version": 0, "min_house_edge_percent": 0, "created_at": "", "created_by": "" },
]
```
- เรียงตามรายชื่อเกมใน `core/game` · `min_house_edge_percent` = ค่าต่ำสุดในทุกขั้น (Scratch มีค่าเดียว) · `created_*` = ของเวอร์ชันปัจจุบัน
- ยังไม่เคยตั้ง (RATE-09A): `version = 0` · ตัวเลข `0` · ข้อความ `""` — เกมนั้นยังเปิดให้เล่นไม่ได้

Error: ร่วม

### 5.2 RE2 · POST /api/v1/bo/pr/rate/detail/get

Request: `{ "game_code": "coin_toss" }`

Response `data` (Coin Flip — ค่าตัวอย่างจาก diagram ครบ 10 ขั้น):
```json
{
  "game_code": "coin_toss",
  "version": 3,
  "steps": [
    { "step": 1,  "multiplier": 1.9,      "win_percent": 50, "reach_percent": 50,     "payout_percent": 95,      "house_edge_percent": 5,       "max_win_percent": 50 },
    { "step": 2,  "multiplier": 3.61,     "win_percent": 48, "reach_percent": 24,     "payout_percent": 86.64,   "house_edge_percent": 13.36,   "max_win_percent": 50 },
    { "step": 3,  "multiplier": 6.859,    "win_percent": 46, "reach_percent": 11.04,  "payout_percent": 75.7234, "house_edge_percent": 24.2766, "max_win_percent": 50 },
    { "step": 4,  "multiplier": 13.0321,  "win_percent": 44, "reach_percent": 4.8576, "payout_percent": 63.3047, "house_edge_percent": 36.6953, "max_win_percent": 50 },
    { "step": 5,  "multiplier": 24.761,   "win_percent": 42, "reach_percent": 2.0402, "payout_percent": 50.5172, "house_edge_percent": 49.4828, "max_win_percent": 50 },
    { "step": 6,  "multiplier": 47.0459,  "win_percent": 40, "reach_percent": 0.8161, "payout_percent": 38.3931, "house_edge_percent": 61.6069, "max_win_percent": 50 },
    { "step": 7,  "multiplier": 89.3872,  "win_percent": 38, "reach_percent": 0.3101, "payout_percent": 27.7198, "house_edge_percent": 72.2802, "max_win_percent": 50 },
    { "step": 8,  "multiplier": 169.8356, "win_percent": 36, "reach_percent": 0.1116, "payout_percent": 18.9603, "house_edge_percent": 81.0397, "max_win_percent": 50 },
    { "step": 9,  "multiplier": 322.6877, "win_percent": 34, "reach_percent": 0.038,  "payout_percent": 12.2484, "house_edge_percent": 87.7516, "max_win_percent": 50 },
    { "step": 10, "multiplier": 613.1066, "win_percent": 32, "reach_percent": 0.0121, "payout_percent": 7.447,   "house_edge_percent": 92.553,  "max_win_percent": 50 }
  ],
  "min_house_edge_percent": 5,
  "min_house_edge_step": 1,
  "max_payout_percent": 95,
  "created_at": "2026-10-12T10:00:00+07:00",
  "created_by": "superadmin1"
}
```
> ตัวอย่างขั้น 1–2 ตรงกับ diagram · ขั้น 3–10 เป็นชุดประกอบเอกสาร (ตัวคูณ `1.9^n` ปัด 4 ตำแหน่ง · ชนะลดขั้นละ 2%) คำนวณด้วย `big.Rat` แล้ว Round4 · ค่าที่ test ยึดคือตัวอย่าง diagram (5% · 13.36% · 22.75% · max_win ตัดที่ 50)
> `max_win_percent` ไม่เกิน 50 (เพดาน `win_percent` — ตัวอย่างนี้สูตรได้ 52.1052 · 54.8476 · 60.1399 … จึงแสดง 50 ทุกขั้น) · **ค่าที่คำนวณทุกตัว (`reach_*` · `payout_*` · `house_edge_*` · `max_win_*` · `min_house_edge_*` · `max_payout_*` · Scratch `avg_multiplier` / `per_ticket_percent`) ไม่ได้เก็บใน DB** — คำนวณจากค่าที่ตั้งทุกครั้งที่ตอบ (RE1 / RE2 / RE3 / RE5 / RE6) · ตัวเลขในตัวอย่างและในดีไซน์หน้าบ้านเป็นแค่ตัวอย่าง

- Coin / RPS: `payout_percent` ต่อขั้น = `reach × multiplier` (ผลตอบแทนผู้เล่นถ้าหยุดที่ขั้นนั้น · = 100 − `house_edge_percent`) · สรุปทั้งเกม `min_house_edge_percent` / `min_house_edge_step` (ขั้นที่ร้านได้น้อยสุด — ค่าเท่ากันเอาขั้นแรก) · `max_payout_percent` = 100 − ค่านั้น (กล่องสรุปของหน้าบ้าน — Q8)
- RPS: รูปแบบเดียวกับ Coin (`multiplier` · `win_percent` · ค่าคำนวณ) · ไม่มี `draw_percent` / `lose_percent` (RATE-20)
- Scratch Card:
```json
{
  "game_code": "scratch_card",
  "version": 1,
  "win_percent": 50,
  "prizes": [
    { "row_no": 1, "multiplier": 0.5, "share_percent": 35, "per_ticket_percent": 17.5 },
    { "row_no": 2, "multiplier": 1,   "share_percent": 35, "per_ticket_percent": 17.5 },
    { "row_no": 3, "multiplier": 2,   "share_percent": 18, "per_ticket_percent": 9 },
    { "row_no": 4, "multiplier": 5,   "share_percent": 10, "per_ticket_percent": 5 },
    { "row_no": 5, "multiplier": 8,   "share_percent": 2,  "per_ticket_percent": 1 }
  ],
  "avg_multiplier": 1.545,
  "payout_percent": 77.25,
  "house_edge_percent": 22.75,
  "max_win_percent": 50,
  "created_at": "2026-10-12T09:00:00+07:00",
  "created_by": "superadmin1@staff"
}
```
- ค่าที่คำนวณปัด 4 ตำแหน่ง (RATE-09) · ยังไม่เคยตั้ง: `version = 0` · `steps` / `prizes` = `[]` · ตัวเลข `0` · ข้อความ `""`

Error: `422`, `406401`

### 5.3 RE3 · POST /api/v1/bo/pr/rate/detail/check (lead RN2)

Request = body ของ RE4 **ไม่มี** `passcode` / `base_version` · ตรวจรูปแบบเหมือน RE4 (ผิด = `422`) แล้วคำนวณ **ไม่บันทึก ไม่เขียน log**

Response `data` = รูปแบบเดียวกับ RE2 (ไม่มี `version` / `created_*`) + ผลตรวจกฎ:
```json
{
  "game_code": "coin_toss",
  "valid": false,
  "issues": [ { "field": "steps[0].win_percent", "code": 406101, "message": "steps[0].win_percent ตั้งได้ไม่เกิน 49.7487 (ร้านต้องได้อย่างน้อย 1%)" } ],
  "steps": [ { "step": 1, "multiplier": 1.99, "win_percent": 50, "reach_percent": 50, "payout_percent": 99.5, "house_edge_percent": 0.5, "max_win_percent": 49.7487 } ],
  "min_house_edge_percent": 0.5, "min_house_edge_step": 1, "max_payout_percent": 99.5
}
```
- กฎ RATE-07 / RATE-32 ไม่ผ่าน = ยังตอบ `200` พร้อมค่าคำนวณ · `valid: false` · `issues` บอกทุกจุดที่ผิด (หน้าบ้านแสดงข้างช่องได้ ไม่ต้องเขียนสูตรเอง) · ผ่านทั้งหมด = `valid: true` · `issues: []`
- รูปแบบนี้ตัดสินแล้ว (Q2)

Error: `422`, `406401`

### 5.4 RE4 · POST /api/v1/bo/pr/rate/detail/update

Request (Coin Flip — ส่งครบ 10 ขั้น ตามลำดับขั้น):
```json
{
  "game_code": "coin_toss",
  "base_version": 3,
  "steps": [
    { "multiplier": 1.9, "win_percent": 50 }, { "multiplier": 3.61, "win_percent": 48 },
    { "multiplier": 6.859, "win_percent": 46 }, { "multiplier": 13.0321, "win_percent": 44 },
    { "multiplier": 24.761, "win_percent": 42 }, { "multiplier": 47.0459, "win_percent": 40 },
    { "multiplier": 89.3872, "win_percent": 38 }, { "multiplier": 169.8356, "win_percent": 36 },
    { "multiplier": 322.6877, "win_percent": 34 }, { "multiplier": 613.1066, "win_percent": 32 }
  ],
  "passcode": "123456"
}
```
Request (RPS): รูปแบบเดียวกับ Coin — แต่ละขั้น `{ "multiplier", "win_percent" }` · ส่ง `draw_percent` = `422`

Request (Scratch Card):
```json
{
  "game_code": "scratch_card",
  "base_version": 1,
  "win_percent": 50,
  "prizes": [ { "multiplier": 0.5, "share_percent": 35 }, { "multiplier": 1, "share_percent": 35 }, { "multiplier": 2, "share_percent": 18 },
              { "multiplier": 5, "share_percent": 10 }, { "multiplier": 8, "share_percent": 2 } ],
  "passcode": "123456"
}
```
Response `data`: `{ "game_code": "scratch_card", "version": 2, "created": true }` · ค่าเท่าปัจจุบัน = `{ "game_code": "scratch_card", "version": 1, "created": false }` (RATE-05C)

| field | กติกา |
|---|---|
| `game_code` | ต้องมีใน `core/game` (`406401`) |
| `base_version` | จำนวนเต็ม ≥ 0 · ไม่ตรงเวอร์ชันปัจจุบัน = `406301` (RATE-05B) |
| `steps` (Coin / RPS) | ครบ 10 ขั้น · RATE-11 / RATE-21 |
| `win_percent` + `prizes` (Scratch) | RATE-30 – RATE-32 |
| `passcode` | `RequirePasscode` (ผิด `401204` · ครบ 5 ครั้ง `401205`) |

- ลำดับเช็ค: รูปแบบ (DTO · บนลงล่างตาม field) → `game_code` (`406401`) → `base_version` (`406301`) → กฎ RATE-07 / 32 (`406101` / `406102`) → ค่าเท่าปัจจุบัน (`created: false`) → บันทึก

Error: `422`, `401204`, `401205`, `406101`, `406102`, `406301`, `406401`

### 5.5 RE5 · POST /api/v1/bo/pr/rate/versions/list

Request: `{ "game_code": "scratch_card", "page": 1, "limit": 20 }` (`utils.NewPage`) · Response `data` (`response.Page`):
```json
{
  "current_page": 1, "total_page": 1, "total_count": 3, "limit": 20, "has_next": false, "has_prev": false,
  "data": [
    {
      "version": 3, "min_house_edge_percent": 22.75, "win_percent": 50, "rollback_from": 0, "same_as_current": true,
      "changes": [
        { "type": "win_percent", "from": 35, "to": 50 },
        { "type": "prize_added", "multiplier": 5, "from": null, "to": 10 },
        { "type": "prize_removed", "multiplier": 0.5, "from": 20, "to": null },
        { "type": "prize_share", "multiplier": 1, "from": 50, "to": 60 }
      ],
      "created_at": "2026-10-12T10:00:00+07:00", "created_by": "superadmin1", "ip": "203.0.113.10"
    }
  ]
}
```
Coin / RPS — `changes` แยกต่อขั้นและ field:
```json
[ { "type": "step", "step": 1, "field": "win_percent", "from": 55, "to": 50 },
  { "type": "step", "step": 4, "field": "multiplier", "from": 13.0321, "to": 12 } ]
```

| field | ความหมาย |
|---|---|
| `min_house_edge_percent` | house edge ต่ำสุดของเวอร์ชันนั้น |
| `win_percent` | Scratch เท่านั้น (Coin / RPS มีหลายขั้น ไม่ส่ง key นี้) |
| `rollback_from` | ย้อนมาจากเวอร์ชันไหน · `0` = บันทึกปกติ |
| `same_as_current` | `true` = ค่าเท่าเวอร์ชันปัจจุบัน (หน้าบ้านปิดปุ่ม "ย้อนไปใช้ค่านี้" — RATE-05C) · แถวของเวอร์ชันปัจจุบันเอง = `true` |
| `changes` | สิ่งที่เปลี่ยนเทียบกับ**เวอร์ชันก่อนหน้า** (`version − 1`) · เวอร์ชันแรก = `[]` · `from` / `to` = ค่าที่เก็บ (ปัด 4 ตำแหน่ง) · ไม่มีค่า = `null` |
| `changes[].type` | `step` (Coin / RPS · มี `step` 1–10 + `field` = `multiplier` / `win_percent`) · `win_percent` (Scratch) · `prize_added` / `prize_removed` / `prize_share` (Scratch · จับคู่รางวัลด้วย `multiplier` ซึ่งห้ามซ้ำ — RATE-31 · `from` / `to` = `share_percent`) |

- เรียงเวอร์ชันใหม่ → เก่า · log คนแก้ (RATE-08) · ไม่ส่ง `actor_id` / `request_id` ออกไป
- โหลดค่าของทุกเวอร์ชันในหน้า + เวอร์ชันก่อนหน้าของแถวสุดท้าย และเวอร์ชันปัจจุบัน ด้วย query แบบ batch (ไม่ N+1 — กฎข้อ 23) แล้วเทียบใน `core/game_rate`
- `changes` / `same_as_current` / `win_percent` เพิ่มตามคำขอหน้าบ้าน 12 ต.ค. (Q4 — ตัดสินแล้ว)

Error: `422`, `406401`

### 5.6 RE6 · POST /api/v1/bo/pr/rate/versions/detail/get

Request: `{ "game_code": "coin_toss", "version": 2 }` · Response `data` = รูปแบบเดียวกับ RE2 · ไม่พบ `406402`

Error: `422`, `406401`, `406402`

### 5.7 RE7 · POST /api/v1/bo/pr/rate/versions/rollback

Request: `{ "game_code": "coin_toss", "version": 2, "base_version": 3, "passcode": "123456" }`
Response `data`: `{ "game_code": "coin_toss", "version": 4, "rollback_from": 2, "created": true }` · ค่าเท่าปัจจุบัน = `created: false` · `version` ปัจจุบัน · `rollback_from: 0`

- ลำดับเช็ค: รูปแบบ → `game_code` → `version` มีจริง (`406402`) → `base_version` (`406301`) → กฎปัจจุบัน (`406101` / `406102` — RATE-05A) → ค่าเท่าปัจจุบัน → บันทึก

Error: `422`, `401204`, `401205`, `406101`, `406102`, `406301`, `406401`, `406402`

### 5.8 ฟังก์ชันสำหรับ engine / module เดิมพัน (ไม่ใช่ endpoint — lead C6)

- `CurrentVersionService(ctx, tx, gameCode)` → ค่าของเวอร์ชันปัจจุบัน + `version_id` · ยังไม่เคยตั้ง (`version = 0`) = error ให้ engine ปฏิเสธ Bet ("เกมนี้ไม่เปิดให้บริการ")
- `ByVersionIDService(ctx, versionID)` → ค่าของเวอร์ชันนั้น · cache ใน memory ตาม `version_id` (เวอร์ชันแก้ไม่ได้ จึงไม่ต้องลบ cache — ไม่ใช่ source of truth ของเงิน · กฎข้อ 14)

## 6. Schema (migration ใหม่ — timestamp หลังไฟล์ล่าสุดบน dev · lead MA6)

```sql
CREATE TABLE game_rate_versions (            -- 1 แถวต่อการบันทึกที่ค่าเปลี่ยน (RATE-05) · ห้าม UPDATE / DELETE
    id             BIGSERIAL        PRIMARY KEY,
    game_code      VARCHAR(50)      NOT NULL,
    version        INT              NOT NULL CHECK (version >= 1),
    win_percent    DOUBLE PRECISION CHECK (win_percent > 0 AND win_percent <= 100),   -- เพดานจริง 50 ตรวจที่ core (MaxWinPercent) · CHECK ≤ 100 กันค่าหลุด · Scratch Card เท่านั้น · Coin / RPS = NULL
    actor_type     VARCHAR(10)      NOT NULL CHECK (actor_type IN ('AGENT','SUB')),     -- log คนแก้ (RATE-08)
    actor_id       BIGINT           NOT NULL,
    created_by     VARCHAR(71)      NOT NULL,                                          -- username ณ เวลานั้น
    ip             VARCHAR(45)      NOT NULL DEFAULT '',
    request_id     VARCHAR(64)      NOT NULL DEFAULT '',
    rollback_from  INT              NOT NULL DEFAULT 0,                                -- RATE-05A · 0 = บันทึกปกติ
    created_at     TIMESTAMPTZ      NOT NULL DEFAULT now(),
    CONSTRAINT uq_game_rate_versions UNIQUE (game_code, version)   -- เลขไม่ซ้ำ + index หาเวอร์ชันล่าสุด (ORDER BY version DESC LIMIT 1)
);

CREATE TABLE game_rate_steps (               -- Coin Flip / RPS
    version_id     BIGINT           NOT NULL REFERENCES game_rate_versions(id),
    step           SMALLINT         NOT NULL CHECK (step >= 1),         -- จำนวนขั้นตาม gameSpecs (เกมใหม่ไม่ต้องแก้ตาราง)
    multiplier     DOUBLE PRECISION NOT NULL CHECK (multiplier >= 1 AND multiplier <= 1000),
    win_percent    DOUBLE PRECISION NOT NULL CHECK (win_percent > 0 AND win_percent <= 100),
    PRIMARY KEY (version_id, step)
);

CREATE TABLE game_rate_prizes (              -- Scratch Card
    version_id     BIGINT           NOT NULL REFERENCES game_rate_versions(id),
    row_no         SMALLINT         NOT NULL CHECK (row_no BETWEEN 1 AND 10),
    multiplier     DOUBLE PRECISION NOT NULL CHECK (multiplier > 0 AND multiplier <= 1000),
    share_percent  DOUBLE PRECISION NOT NULL CHECK (share_percent > 0 AND share_percent <= 100),
    PRIMARY KEY (version_id, row_no)
);
```
- บันทึก: tx เดียวที่ service เปิด · advisory lock ต่อ `game_code` → อ่านเวอร์ชันล่าสุด → เทียบ `base_version` (`406301`) → เทียบค่าเท่าปัจจุบัน → insert เวอร์ชัน + แถวลูกแบบ batch · ชน `uq_game_rate_versions` (หลุด lock) = `406301`
- ค่าที่เก็บปัด 4 ตำแหน่งแล้ว (`utils.Round4`) · CHECK ในตารางกันค่าหลุดช่วง · กฎ RATE-07 / 32 ตรวจใน `core/game_rate` ไม่อยู่ใน DB

## 7. Test cases

| Rule ID | Input | Expected |
|---|---|---|
| RATE-02 | Company / Share / Agent / ADMIN เรียกทุกเส้น | `402303` |
| RATE-02 | sub ของ Superadmin ไม่มีสิทธิ์ `rate` · มี `view` แล้ว `detail/update` · มี `edit` | `402303` · `402303` · สำเร็จ (`actor_type = SUB` · `created_by = superadmin1@staff`) |
| RATE-02 | ไม่มีสิทธิ์ + ไม่ส่ง passcode | `402303` (สิทธิ์ก่อน passcode) |
| RATE-03 / 04 | ตัวคูณ `1.90005` · `win_percent` `50.12345` (5 ตำแหน่ง) | `422` |
| RATE-03 | ตัวคูณขั้น 10 `1000.0001` · รางวัล Scratch `0` | `422` |
| RATE-04 | `win_percent` `0` · `50.0001` (ทุกเกม) · `50` | `422` · `422` (เพดาน 50) · สำเร็จ |
| RATE-05 | บันทึก 2 ครั้งค่าต่างกัน | เวอร์ชัน 1 และ 2 · เวอร์ชัน 1 ยังอ่านได้ค่าเดิม |
| RATE-05B | `base_version` เก่า · ยิงพร้อมกัน 2 request ด้วย `base_version` เดียวกัน | `406301` · สำเร็จ 1 อีกอัน `406301` |
| RATE-05C | บันทึกค่าเดิมทั้งชุด · ย้อนไปเวอร์ชันที่ค่าเท่าปัจจุบัน | `200` `created: false` ไม่มีเวอร์ชันใหม่ |
| RATE-05A | มี v1–v3 ย้อนไป v2 · ย้อนไปเวอร์ชันที่ไม่ผ่านกฎใหม่ · เวอร์ชันไม่มี · ไม่ส่ง passcode | v4 ค่าเท่า v2 `rollback_from = 2` · `406101` · `406402` · `422` |
| RATE-07 / 13 | Coin Flip ขั้น 1 `1.99 / 50` (house edge 0.5%) · ขั้น 1 `1.99 / 49.7487` | `406101` msg บอก `steps[0].win_percent` ค่าสูงสุด 49.7487 · ไม่มีเวอร์ชันใหม่ · สำเร็จ |
| RATE-07 | house edge = 1% พอดี | สำเร็จ |
| RATE-10 | ส่ง 9 ขั้น | `422` |
| RATE-11 | ตัวคูณขั้น 4 ≤ ขั้น 3 | `422` (`steps[3].multiplier`) |
| RATE-20 | RPS ส่ง `draw_percent` · ส่ง 9 ขั้น | `422` · `422` |
| RATE-22 | RPS ใช้ชุดเดียวกับตัวอย่าง Coin · ขั้น 1 `1.99 / 50` | ค่าคำนวณเท่า Coin (5% · 13.36% ฯลฯ) · `406101` |
| RATE-31 | ตัวคูณซ้ำ 2 แถว · 0 แถว · 11 แถว | `422` |
| RATE-32 | share รวม 99.5 · รวม 100.0001 | `406102` |
| RATE-33 | W 50 กับตารางตัวอย่าง · W 50 กับตาราง `2×/100` (จ่ายคืน 100%) | `house_edge = 22.75` `max_win = 50` (สูตร 64.0776 เกินเพดาน) · `406101` |
| RATE-08 | ไม่ส่ง passcode · passcode ผิด | `422` · `401204` |
| RATE-09A | ระบบใหม่ ยังไม่เคยตั้ง | RE1 ทุกเกม `version = 0` · RE2 `steps = []` · `CurrentVersionService` = error |
| RN2 | RE3 ค่าผ่าน · ขั้น 3 เกิน · share รวม 99 | `valid: true` · `valid: false` + issue `406101` · issue `406102` · ไม่มีเวอร์ชันใหม่ทุกกรณี |
| — | `game_code` ไม่มีในระบบ | `406401` |
| RATE-01 | ส่ง `prizes` ให้ `coin_toss` · ส่ง `draw_percent` ให้ `coin_toss` / `rock_paper_scissors` | `422` |
| core | `core/game_rate` table test: ตัวอย่าง diagram 5% · 13.36% · 22.75% · `max_win` ถูกตัดที่ 50 · ขอบ house edge 1% · RPS ใช้สูตร Coin | ตรงทุกค่า (`big.Rat` แล้ว Round4) |

## 8. Error codes (`06`)

| Code | HTTP | ความหมาย |
|---|---|---|
| 406101 | 200 | ตั้งค่านี้แล้ว house edge ต่ำกว่าขั้นต่ำ (msg บอกขั้น / แถว และ `win_percent` สูงสุดที่ตั้งได้) |
| 406102 | 200 | สัดส่วนรางวัลรวมต้องเท่ากับ 100% |
| 406301 | 200 | มีคนแก้ไปแล้ว กรุณาโหลดใหม่ (`base_version` ไม่ตรง — RATE-05B) |
| 406401 | 200 | ไม่พบเกมนี้ |
| 406402 | 200 | ไม่พบเวอร์ชันนี้ |

## 9. Open items

### ขอ lead ยืนยัน (ก่อน implement)

| # | เรื่อง | ข้อเสนอ |
|---|---|---|
| Q1 | RR6 หยุดเกมด่วน: ตอนนี้**ไม่มีเส้นให้ Superadmin ปิด `status_game` ของตัวเอง** (`agents/detail/update` แก้ได้แค่ลูกตรง) | ⏸ **ทำทีหลัง** (2026-10-12) — ยังไม่มีเส้น · ระหว่างนี้ปิดที่ Company ทีละตัว |
| Q2 | RE3 `detail/check` รูปแบบผลตรวจ | ✅ ตัดสิน 2026-10-12: รับตามที่เสนอ |
| Q3 | `max_win_percent` นิยามใหม่ตาม RR2 = ค่าสูงสุดที่ยังได้ house edge ≥ 1% (เดิม = จุดคุ้มทุน 0%) | ✅ ตัดสิน 2026-10-12: รับตามที่เสนอ |
| Q4 | หน้าบ้านขอ (12 ต.ค.): RE5 `versions/list` เพิ่มต่อแถว `same_as_current` (ปิดปุ่มย้อน — RATE-05C) · `win_percent` (Scratch เท่านั้น) · `changes` = สิ่งที่เปลี่ยนเทียบกับเวอร์ชันก่อนหน้า (เวอร์ชันแรก `[]`) — Coin / RPS `{ "type": "step", "step", "field", "from", "to" }` · Scratch `win_percent` / `prize_added` / `prize_removed` / `prize_share` จับคู่รางวัลด้วย `multiplier` | ✅ ตัดสิน 2026-10-12: รับตามที่เสนอ |
| Q5 | หน้าบ้านขอ: error ส่ง `data` — `406301` ส่ง `{ version, created_by, created_at }` ของเวอร์ชันล่าสุด · `401204` ส่ง `remaining_attempts` | ✅ ตัดสิน 2026-10-12: ไม่เพิ่ม `data` · `406301` ใส่เวอร์ชันล่าสุด + คนแก้ใน `msg` (RATE-05B) · `401204` ใช้ msg "เหลือ N ครั้ง" เดิม |
| Q6 | หน้าบ้านขอ (ดีไซน์ "ตั้งได้สูงสุด 50%"): เพดาน `win_percent` ≤ 50% | ✅ เปลี่ยน 2026-10-12: **เพดาน `win_percent` 50 ทุกเกม** (ส่งเกิน = `422` · `max_win_percent` ไม่เกิน 50) |
| Q7 | RPS ไม่มีเสมอ (คำขอหน้าบ้าน) | ✅ **lead ตัดสิน 2026-10-12: ตัดเสมอออก** — แก้หัวข้อ 2, RATE-20 – 23, 5.2, 5.4, 5.5, 6 (ไม่มีคอลัมน์ `draw_percent`), 7 แล้ว · เอนจิ้นต้องไม่มีผลเสมอ (RH5) |
| Q8 | หน้าบ้านแจ้ง: ดีไซน์ Coin มีช่อง "อัตราชนะรวมของเกม" แต่ spec ตั้งแยกทีละขั้น | ✅ ตัดสิน 2026-10-12: ไม่มีอัตราชนะรวม (ตั้งทีละขั้น) · กล่องสรุปของหน้าบ้านใช้ค่าคำนวณ `min_house_edge_percent` + `min_house_edge_step` + `max_payout_percent` และ `payout_percent` ต่อขั้น (เพิ่มใน RE2 / RE3) |

### ส่งต่อ module เดิมพัน / engine (lead 7.3 — ไม่ต้องตัดสินตอนนี้)

| # | เรื่อง |
|---|---|
| RH1 | Bet เก็บ `rate_version_id` · Streak ทุกขั้นอ่านค่าจาก id นั้น · version 0 → ปฏิเสธ Bet ("เกมนี้ไม่เปิดให้บริการ") |
| RH2 | ยอดแทง: เดิม RATE-09B (8 ต.ค.) "บาทเต็ม ขั้นต่ำ 1 บาท" vs กฎข้อ 9 float · ช่วงยอดแทง 10–500 ใน diagram · เพดานยอดจ่ายต่อบิล (เกี่ยวกับ RR3) · ใช้กับทุกสกุลไหม |
| RH3 | หน้าเกมของ Member ต้องได้ตารางตัวคูณ (Coin / RPS 10 ขั้น · รางวัล Scratch) — ส่งแค่ตัวคูณ ไม่ส่ง `win_percent` / `share_percent` · เป็นเส้นฝั่งผู้เล่นของ module เดิมพัน |
| RH4 | Scratch ตรง 2 ช่อง แบ่งครึ่ง "ช่องแรกปัดลง" → ปัดที่ 4 ตำแหน่ง (`Round4`) · Streak ค้างเกิน 1 วัน CASH OUT อัตโนมัติ / Scratch ไม่ขูดเกิน 1 วันเปิดอัตโนมัติ (เดิม RATE-09C) — job ของ engine |
| RH5 | RPS ไม่มีผลเสมอ (lead Q7 2026-10-12) — เอนจิ้นสุ่มแค่ชนะ / แพ้ตาม `win_percent` ของขั้น · หน้าเกมไม่ต้องแสดงผลเสมอ |

## 10. แผน implement (lead C1–C8 · branch `boiledegg/bo/game_rate-settings`) — ✅ เสร็จครบ 2026-10-12

| ขั้น | งาน | ไฟล์ |
|---|---|---|
| C1 | รายชื่อเกมย้ายไป `app/core/game` (รหัส · หมวด · กลุ่ม `minigame`) · agent_management (`pt` / `status_game`) เรียกจาก `core/game` แทน `pt_rule.go` · commit แรก ไม่เปลี่ยนพฤติกรรม | `app/core/game/game.go` + test · `app/core/agent_management/pt_rule.go` |
| C2 | กฎและสูตรเป็น pure function ด้วย `big.Rat`: ตรวจ Coin / RPS (RATE-10 – 13, 20 – 22) · Scratch (RATE-30 – 33) · คำนวณ reach / payout / house edge / max win / สรุปทั้งเกม / avg / per ticket · ค่าคงที่ (ขั้นต่ำ house edge 1% · ตัวคูณ ≤ 1,000 · 10 ขั้น · 1–10 รางวัล) · เทียบค่าเท่ากัน (RATE-05C) · `changes` ของ RE5 · table test ตัวอย่าง diagram (5% · 13.36% · 22.75% · max_win ตัดที่ 50 · ขอบ 1%) | `app/core/game_rate/*.go` + `*_test.go` |
| C3 | migration 3 ตาราง (หัวข้อ 6 · timestamp หลัง `20261011120000`) · models · repository (`db` param แรก · ไม่มี tx · insert batch · โหลดหลายเวอร์ชันแบบ batch) | `database/migrations/*_game_rate.sql` · `app/models/game_rate_models.go` · `app/repository/postgres/game_rate/` |
| C4 | service: tx เดียว · advisory lock ต่อเกม → `base_version` (`406301` · msg บอกเวอร์ชัน + คนแก้) → กฎ → ค่าเท่าปัจจุบัน (`created: false`) → insert · rollback (RATE-05A) · list / detail / check / versions | `app/service/game_rate/` |
| C5 | DTO `Validate()` แบบ `if` (≤ 4 ตำแหน่ง · ไม่รับ field ของเกมอื่น ยกเว้น `passcode`) · controller · routes RE1 – RE7 (`RequirePermission(rate, …)` ก่อน `RequirePasscode()`) · `pkg/apperr/game_rate_errors.go` | `app/internals/backoffice/{dto,controllers}/game_rate/` · `routes.go` · `pkg/apperr/` |
| C6 | `CurrentVersionService` · `ByVersionIDService` + cache ใน memory ตาม `version_id` (หัวข้อ 5.8) | `app/service/game_rate/` |
| C7 | integration test ตามหัวข้อ 7 + ยิง update พร้อมกัน 2 request (`406301`) · รัน `make test-integration` แนบผล | `app/internals/backoffice/game_rate_*_integration_test.go` |
| C8 | เอกสารหน้าบ้าน (curl RE1 – RE7 · ค่าตัวอย่างให้ tester ตั้งบน dev / uat) · `MIGRATION_STATUS.md` · แจ้ง frontend | `docs/frontend/game_rate.md` |

ลำดับ: C1 → C2 → C3 → C4 + C6 → C5 → C7 → C8 · PR เดียว (หรือแยก C1 ถ้าต้องการ merge ก่อน) · ก่อน merge เอา `dev` เข้า branch และเช็คเลข migration (lead MA6)
