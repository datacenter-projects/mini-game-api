# Game_Rate (อัตราแพ้ชนะ · เมนู ⑥) — ทำตาม lead review (S1–S10 · C1–C8) · 2026-10-12 · + C9 สวิตช์ปิดเกม RE8 · 2026-10-13

จาก boiledegg · review: [docs/reviews/game-rate.md](../reviews/game-rate.md)
branch `boiledegg/bo/game_rate-settings` (แตกจาก dev + cherry-pick 3 commit เอกสาร · ไม่เอา commit test-infra เก่า · ลบ branch `boiledegg/bo/game-rate` แล้ว) ·
**merge เข้า dev + deploy แล้ว** (`7407f0f` · migration `7702f1e` · tag `dev-202610140730` · 14 ต.ค.) — lead approve หลังรีวิวรอบ 4

## 1. การตัดสินระหว่างทำ (ผ่าน boiledegg · 2026-10-12)

| # | เรื่อง | ผล |
|---|---|---|
| Q1 | สวิตช์ปิดเกมทั้งระบบ (RR6) | **lead ทบทวน 2026-10-13: ทำ** → RE8 (C9) · QS1 ตั้งทุกบัญชี Superadmin · QS2 ปิดแค่เดิมพันใหม่ · QS3 log `account_change_logs` `UPDATE_GAMES` |
| Q2 | `detail/check` | `200` + `valid` + `issues[]` ทุกจุดที่ผิด |
| Q3 | `max_win_percent` | ค่าสูงสุดที่ยังได้ house edge ≥ 1% · **ปัดลง** 4 ตำแหน่ง (ค่าที่แสดงตั้งได้จริง) |
| Q4 | RE5 | เพิ่ม `changes` (เทียบกับเวอร์ชันก่อน) · `same_as_current` · `win_percent` (Scratch) |
| Q5 | error มี `data` | ไม่ทำ · `406301` ใส่เวอร์ชันล่าสุด + คนแก้ใน `msg` · `401204` ใช้ msg เดิม |
| Q6 | เพดาน `win_percent` | **≤ 50 ทุกเกม** · `max_win_percent` ไม่เกิน 50 |
| Q7 | RPS | **ไม่มีเสมอ** (lead) — สูตรเดียวกับ Coin · `draw_percent` = `422` |
| Q8 | อัตราชนะรวมของ Coin | ไม่มี · กล่องสรุปใช้ `min_house_edge_*` / `max_payout_percent` (เพิ่มใน RE2 / RE3) + `payout_percent` ต่อขั้น |
| — | ลำดับเกม | `scratch_card` → `coin_toss` → `rock_paper_scissors` |
| — | รองรับเกมในอนาคต | แต่ละเกมผูกกับ**รูปแบบอัตรา** (`steps` / `prizes`) ใน `core/game_rate` · เกมใหม่รูปแบบเดิม = 1 บรรทัด ไม่ต้องแก้ API / ตาราง / service · test บังคับว่าทุกเกมมีรูปแบบ |
| — | ล็อก / เกลี่ย % รางวัล | ฝั่งหน้าบ้าน · backend ตรวจผลรวม = 100 พอดี |

## 2. ผลแต่ละข้อ

| ข้อ | ผล |
|---|---|
| S1–S10 | spec `docs/modules/game_rate.md` แก้ครบ (หัวเอกสาร · สิทธิ์ · float 4 ตำแหน่ง · house edge ≥ 1% · เพดาน · `base_version` · RE1–RE7 + ตัวอย่างครบ 10 ขั้นที่คำนวณจริง · schema · test · error · open items) + หัวข้อ 10 แผน implement |
| C1 | `app/core/game` รายชื่อเกม · agent_management ใช้จากที่นี่ (ไม่เปลี่ยนพฤติกรรม) |
| C2 | `app/core/game_rate`: รูปแบบอัตรา · ตรวจรูปแบบ · สูตรด้วย `big.Rat` · ค่าคำนวณ · เทียบค่าเท่า · `changes` · table test (5% · 13.36% · 22.75% · ขอบ 1% · เพดาน 50) |
| C3 | migration `20261014110000_game_rate` (3 ตาราง · DOUBLE + CHECK · `step >= 1` ไม่ผูก 10 ขั้น) · models · repository (batch · advisory lock) |
| C4 | service: tx เดียว · lock ต่อเกม → `base_version` (`406301`) → กฎ → ค่าเท่า (`created: false`) → insert batch · rollback ตรวจกฎปัจจุบัน |
| C5 | routes RE1–RE7 `POST /bo/pr/rate/…` · `rate` view / edit ก่อน passcode · error `406101`–`406402` |
| C6 | `CurrentVersionService` (version 0 = `ErrRateNotConfigured`) · `ByVersionIDService` + cache ใน memory |
| C7 | integration test 7 ชุด: สิทธิ์ · รูปแบบ · กฎ · เวอร์ชัน / rollback / RE5 · rollback ตามกฎปัจจุบัน · ยิงพร้อมกัน (สำเร็จ 1 ที่เหลือ `406301` · รัน 5 รอบ) · ฟังก์ชันของ engine |
| C8 | `docs/frontend/game_rate.md` (curl RE1–RE7 · error · ค่าตัวอย่างให้ tester · วิธีเกลี่ย %) · `MIGRATION_STATUS.md` |
| C9 | RE8 `POST /bo/pr/rate/status/update` `{ game_code, status_game, passcode }` → `{ game_code, status_game, changed }` · `rate` edit + passcode · tx เดียว lock แถวของทุก Superadmin `FOR UPDATE` เรียง `agent_id` · update เฉพาะบัญชีที่ค่าต่าง · log `UPDATE_GAMES` 1 แถวต่อบัญชี · ค่าเดิม = `changed: false` ไม่มี log · ไม่แก้สายล่าง (ไหลลงผ่าน `IsGameOpen`) · RE1 เพิ่ม `status_game` · ไม่มี migration / error code ใหม่ · integration test 2 ชุด (สิทธิ์ / รูปแบบ · ปิด-ซ้ำ-เปิดคืน กับ Superadmin 2 บัญชี + sub) |

## 3. ผล test

- lint 0 issues · structure OK · build / vet ผ่าน · unit + integration ผ่านทุก package (Windows test env · `-race` รันไม่ได้ในเครื่องนี้ — CI รัน)

## 4. ก่อน deploy

1. migrate RDS dev `20261014110000_game_rate` (ตารางใหม่ล้วน ไม่แตะข้อมูลเดิม)
2. ระบบไม่มีค่าเริ่มต้น — Superadmin ต้องตั้งอัตราเองก่อนเปิดเกม (ค่าตัวอย่างใน `docs/frontend/game_rate.md` หัวข้อ 4)
3. แจ้งหน้าบ้าน: เส้นใหม่ RE1–RE8 (RE8 สวิตช์ปิดเกม · RE1 มี `status_game`) · เพดาน 50 · RPS ไม่มีเสมอ · `max_win_percent` ปัดลง · ล็อก / เกลี่ยเป็นฝั่งหน้าบ้าน

## 5. ส่งต่อ (ไม่บล็อก)

- RH1–RH7 ไป module เดิมพัน / engine: เก็บ `rate_version_id` · ยอดแทง · หน้าเกมเห็นแค่ตัวคูณ · Scratch แบ่งรางวัล / CASH OUT อัตโนมัติ · RPS ไม่มีผลเสมอ · Streak (RH6) · เช็คสวิตช์ปิดเกมทั้งสายก่อนรับเดิมพันใหม่ (RH7)
- RE3 `issues[].message` เป็นภาษาไทยอย่างเดียว

## 6. แก้ตามรีวิวรอบ 4 (ผลตรวจโค้ด `4815765` · 2026-10-14)

| # | ผล |
|---|---|
| M1 (Q9) | `msg` ของ `422` / `406101` / `406102` และ RE3 `issues[].message` ใช้ `utils.FieldLabel` · เพิ่ม label ของ game_rate ในตารางเดียวกัน (`rowLabels` `steps` → "ขั้นที่ i+1" · `prizes` → "รางวัลที่ i+1" + field ในแถว · `game_code` / `base_version` / `win_percent` / `steps` / `prizes` / `status_game`) · `issues[].field` คง path · ตัวอย่าง: "ขั้นที่ 1 · โอกาสชนะ ตั้งได้ไม่เกิน 49.7487% (ร้านต้องได้อย่างน้อย 1%)" · `406102`: "สัดส่วนรางวัลรวมต้องเท่ากับ 100% (ตอนนี้ 99.5%)" · แก้ตัวอย่างใน spec + frontend doc · test: `TestFieldLabel` + format table เทียบกับ label + RE3 field / message |
| G-S1 | RE8 lock `user_agents` ของทุก Superadmin (`FOR NO KEY UPDATE` เรียง id) แล้วแถวของเกม (`FOR UPDATE`) · บัญชีที่ไม่มีแถว = เพิ่มแถวใน tx เดียวกันตามค่าที่สั่ง (`pt_from_parent` 100 · รับ PT · ผู้สร้าง = ตัวเอง) · ไม่มีแถว = ถือว่าเปิดอยู่ (เท่ากับที่ `IsGameOpen` ไม่นับ) → ปิด = log `UPDATE_GAMES` (เดิม `true`) · เปิด = เพิ่มแถว `changed: false` ไม่มี log · test `TestRateStatusMissingRow` |
| G-S2 | เอกสาร RATE-09 / RATE-34 + frontend: ผลรวม `prizes[].payout_percent` ต่างจากของใบได้ไม่เกิน 0.0001 (ปัดทีละแถว · ใช้แสดงผล) |
| nit | ไม่ได้ทำ (verify report ไม่อยู่ใน repo) |

merge dev ล่าสุด (`532ae1e`) แล้ว · unit + integration ผ่านทุก package · lint 0 issues

## 7. deploy dev (2026-10-14)

1. lead approve → merge `boiledegg/bo/game_rate-settings` เข้า dev (`ccf8f00`) ตามด้วย `boiledegg/bo/agent_management` (MGMT-18A · `7407f0f`) — ไม่มี conflict
2. lint 0 issues · unit + integration ผ่านทุก package (test DB reset ใหม่)
3. ⚠ **เลข migration ชน:** `20261012160000` ของ game_rate ชนกับ `20261012160000_asset_create_tables` ของ `carl/asset`
   ที่รันบน RDS dev ไปแล้ว (branch ยังไม่ merge) → goose เห็นว่ารันแล้วแต่ไม่มีตาราง game_rate ·
   **เปลี่ยนเป็น `20261014110000`** (`7702f1e` — ไฟล์นี้ไม่เคยรันที่ไหน ไม่ขัด A3) · test ทั้งชุดผ่านอีกรอบ
4. migrate RDS dev: `20261014110000_game_rate` · `20261014120000_agent_management_superadmin_remain` — OK ทั้งคู่
5. tag `dev-202610140730` → GitHub Actions build image → infra rollout
6. ต่อจากนี้: แจ้งหน้าบ้าน (`docs/frontend/game_rate.md` หัวเอกสาร) · Superadmin บน dev ตั้งอัตรา 3 เกมให้ tester (ระบบไม่มีค่าเริ่มต้น)
   · ฝากแจ้ง carl: RDS dev มี `20261012160000` / `20261012170000` ของ `carl/asset` รันแล้ว — merge ต้องใช้เลขเดิม

## 8. เพิ่มเวลา + คนเปิด / ปิดเกมล่าสุด (RE1 / RE8 · 2026-10-14 · ตามคำขอ — ขอ lead รับทราบ)

- RE1 `rate/list` ทุกแถว + RE8 `status/update` response เพิ่ม `status_updated_at` / `status_updated_by` = เปิด / ปิดสวิตช์ครั้งล่าสุด (sub = `owner@name`) · ไม่เคย = `""` · ส่งค่าเดิมซ้ำ (`changed: false`) = ครั้งล่าสุดเดิม
- อ่านจาก log `UPDATE_GAMES` ที่ target เป็น Superadmin (status_game ของ Superadmin เปลี่ยนได้ทางเดียวคือ RE8) · query เดียว `DISTINCT ON` ต่อเกม · ใช้ index `idx_account_change_logs_target` · **ไม่มี migration / field ใหม่ใน DB**
- log ของบัญชีอื่น (เช่น `agents/detail/update` ปิดเกมของ Company) ไม่นับ
- test `TestRateStatusLastAction` (ยังไม่เคย = "" · ปิด · ส่งซ้ำ · sub เปิดคืน · log ของ Company ไม่นับ) · spec 5.1 / 5.9 + frontend RE1 / RE8

## 9. `status` ต่อขั้น / ต่อแถวรางวัล (RE3 / RE4 · 2026-10-16 · ตามคำขอหน้าบ้าน — **รอ lead ยืนยัน** · spec Q10 · RATE-36)

- `steps[].status` (Coin / RPS) และ `prizes[].status` (Scratch) boolean · ไม่ส่ง = `false` · `null` = `422` · ไม่มี endpoint ใหม่ · ไม่มี action
- backend **เก็บและคืนค่าเท่านั้น** (หน้าบ้านใช้ล็อกขั้น / แถวตอนเกลี่ย %) ไม่เข้าสูตร / กฎ RATE-07 / engine
- เป็นส่วนของค่า: เปลี่ยน `status` อย่างเดียว = เวอร์ชันใหม่ · ชุดเดิมรวม `status` = `created: false` · ย้อนเวอร์ชันคืน `status` · RE5 `changes` ไม่แสดง
- คืนใน RE2 / RE3 / RE6 ทุกขั้น / ทุกแถว
- migration `20261016110000_game_rate_status.sql` (`game_rate_steps.status` / `game_rate_prizes.status` `BOOLEAN NOT NULL DEFAULT false`) — **ยังไม่รันบน RDS · ยังไม่ merge เข้า dev**
- test `TestRateStatus` (Coin ขั้น 5/6/9/10 · ไม่ส่ง = false · ชุดเดิม created false · status อย่างเดียวเปลี่ยน = เวอร์ชันใหม่ · RE6 · ย้อน · RPS · Scratch) + `TestStatusIsPartOfValuesButNotFormula` (core)
- เอกสาร: spec Q10 / RATE-36 / schema / test case · `docs/frontend/game_rate.md` (ตัวอย่าง response + หมายเหตุ) · `docs/MIGRATION_STATUS.md`
- commit `194b60f` · `0045be1` · `6d452a6` บน `boiledegg/bo/game_rate-settings`
- **ขอ lead:** ยืนยันให้ merge (และให้รัน migration บน RDS dev) · ถ้าเห็นต่าง (เช่น ให้ `status` มีผลกับการเล่น หรือแสดงใน RE5 `changes`) ต้องออก spec เพิ่ม
