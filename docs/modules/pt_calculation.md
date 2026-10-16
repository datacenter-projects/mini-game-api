# การคิด PT ต่อ Member (`pt_calculation`) — Spec

- สถานะ: **APPROVED**
- อนุมัติโดย: zerph · วันที่: 2026-10-15 ([docs/reviews/pt-calculation.md](../reviews/pt-calculation.md) — Q1 – Q5 · แก้ตาม Q2 / Q3 แล้ว)
- ผู้ดูแล: maofoy · branch `maofoy/utils/pt-calculate` · PR review: boiledegg
- เป็น input ของ bet core / settle (lead ทำเอง)
- ที่มาของ rule: [docs/references/pt-force-remain-commission.md](../references/pt-force-remain-commission.md) (reference ยังไม่ใช่ spec) · agent_management MGMT-16 – MGMT-25 · member_management MGMT-21 · ไม่ได้ extract จากโค้ดเก่า
- ชื่อ module ในโค้ด: `pt_calculation` (`core/pt_calculation` · `service/pt_calculation` · `controllers/pt_calculation` · `dto/pt_calculation`)

> เอกสารนี้คือ source of truth ของ module — โค้ดและ test ต้องตรงกับเอกสารนี้ ถ้าไม่ตรงให้แก้โค้ด
> หรือแก้เอกสารผ่านการอนุมัติใหม่ ห้ามอ้างเอกสารของโปรเจกต์เก่า

## 1. หน้าที่ของ module

ใส่ `member_id` + `game_code` แล้วได้ **PT สุดท้าย** (หลังคิด Force → Remain → คืน Superadmin) และ **Commission สุทธิ** (`commission_net` — % ของยอดแทงที่ได้ / เสีย) ของทุกชั้นในสายของ Member คนนั้น

- **service** `CalculateMemberPT` (รับ `db` / tx ของผู้เรียก) — ไว้ให้ module เดิมพันเรียกตอนรับ bet (เก็บผลไว้กับ bet — MGMT-23 "มีผลกับ bet ใหม่")
- **เส้น dev** `POST /api/v1/dev/pt/calculate` (context `developer` · ไม่ต้อง login) — ให้ dev / QA เช็คผลเอง · เปิดเฉพาะ env ที่ตั้ง `DEV_ROUTES_ENABLED=true`

**ไม่อยู่ใน module นี้:** คิดเป็นเงิน (ได้ / เสียตาม PT · ยอด Commission ที่ชั้นบนจ่าย) · กรณี Member ชนะ · เช็คสถานะบัญชี / เปิด-ปิดเกม / `win_limit_per_day` ก่อนรับ bet — ทั้งหมดอยู่ module เดิมพัน / Settle

## 2. คำศัพท์

| คำ | ความหมาย |
|---|---|
| สาย | ชั้นฝั่ง agent จาก Superadmin ลงมาถึงผู้สร้าง Member = `user_members.cnf.parent` (เรียงบน → ล่าง · MGMT-61) · ยาวไม่เท่ากัน (Agent ซ้อนได้ไม่จำกัด · Company 1 to 1 สร้าง Member ตรง — MGMT-02) |
| `L0 … Lk` | ชั้นในสาย · `L0` = Superadmin · `Lk` = ผู้สร้าง Member |
| `P[i]` | `agent_game_settings.pt_from_parent` ของ `Li` (ได้รับจากผู้สร้าง) · `P[0]` = 100 |
| `hold[i]` | PT ที่ `Li` ถือตามที่ตั้งไว้ · `i < k`: `P[i] − P[i+1]` · `i = k`: `pt` ของ Member (`user_member_game_settings.pt`) · `Li` ไม่รับ PT = `0` (PTC-02A) |
| ไม่รับ PT | `agent_game_settings.status = false` ของ `Li` ในเกมนี้ (`pt` → ระบบ → `status` — MGMT-20) · ไม่ใช้กับ `L0` |
| CSM | `Li` ที่ `role = COMPANY` และ `agent_type = SEAMLESS_MASTER` (MGMT-19) |
| `Force[i]` / `Remain[i]` | `force` / `remain` ในแถว `agent_game_settings` ของ `Li` (ผู้สร้างของ `Li` ตั้งให้) · `0` = ไม่ตั้ง · ไม่ใช้ของ `L0` |
| `left` | PT ที่ยังไม่มีใครถือ = `100 − Σ hold` |
| PT สุดท้าย | PT ที่ชั้นนั้นถือจริงหลังคิด Force / Remain / คืน Superadmin · รวมทุกชั้น = 100 |

## 3. Business rules

### การคิด PT

| ID | Rule |
|---|---|
| PTC-01 | อ่านค่าของเกม `game_code` ที่ส่งมา: แถว `agent_game_settings` ของทุกชั้นในสาย + แถว `user_member_game_settings` ของ Member · ใช้ค่าล่าสุด ณ ตอนคิด |
| PTC-02 | `hold` ตั้งต้นตามคำศัพท์ · `left = 100 − Σ hold` (เท่ากับ `P[k] − pt` ของ Member) · **ไม่ใช้คอลัมน์ `user_member_game_settings.remain`** (เป็นค่าที่ระบบคิดเก็บไว้แสดงผล ไม่ใช่กฎ Remain) |
| PTC-02A | **ชั้นที่ไม่รับ PT** (lead Q2 · ปิด O1 / MGMT-20): `hold[i] = 0` · ส่วนที่ชั้นนั้นจะถือรวมอยู่ใน `left` แล้วไหลตามกฎปกติ (Force / Remain ของชั้นอื่นรับก่อน ไม่มีก็คืน Superadmin) · **ไม่โดน Force ของตัวเอง** (ข้าม PTC-04 ของชั้นนั้น) · **Remain ที่ชั้นนั้นตั้งให้ลูกยังทำงาน** (PTC-05 ของ `L(i+1)` ยังเพิ่มให้ `Li` ได้ — ชั้นที่ไม่รับ PT จึงอาจมี PT สุดท้าย > 0 ได้ ดูชุด E) · Remain ที่ผู้สร้างตั้งให้ชั้นนั้นทำงานตามปกติ |
| PTC-03 | คิด**จากชั้นล่างสุดขึ้นไป** `i = k … 1` · ในชั้นเดียวกัน**คิด Force ก่อน Remain เสมอ** · ใช้ค่าหลังสเต็ปก่อนหน้า |
| PTC-04 | **Force[i]:** `s = Σ eff[i..k]` · ถ้า `s < Force[i]` → **`Li` ถือเพิ่ม** `min(Force[i] − s, left)` · ไม่งั้นไม่ทำงาน · ข้ามเมื่อ `Li` ไม่รับ PT (PTC-02A) หรือ `Li` เป็น CSM (PTC-04A) |
| PTC-04A | **ข้าม Force ของ CSM เสมอ** (lead Q3 — กันพลาด · CSM ถือสู้ไม่ได้ MGMT-19 · agent_management ห้ามตั้งแล้ว MGMT-18A) · เจอ `force ≠ 0` ในแถวของ CSM = log `warn` (ไม่ error · คิดต่อ) |
| PTC-05 | **Remain[i]:** `s = Σ eff[i..k]` (หลัง Force ของชั้นเดียวกัน) · ถ้า `s < Remain[i]` → **`L(i−1)` ถือเพิ่ม** `min(Remain[i] − s, left)` · ไม่งั้นไม่ทำงาน |
| PTC-06 | จบทุกชั้นแล้ว `left` ที่เหลือ**คืน Superadmin** (`eff[0] += left` — MGMT-22) |
| PTC-07 | ปัด 4 ตำแหน่ง (`utils.Round4`) หลังบวก / ลบทุกครั้ง · เทียบ `s < Force` / `s < Remain` ด้วยค่าที่ปัดแล้ว (กฎข้อ 9) |
| PTC-08 | ตรวจผล: `Round4(Σ PT สุดท้าย) = 100` · ทุกค่า ≥ 0 · ไม่ผ่าน = `508601` + log ทั้งสาย (ห้ามคืนผลที่ผิด) |
| PTC-09 | ข้อมูลในสายผิดรูป = `508601` + log: `cnf` ว่าง หรือชั้นแรกไม่ใช่ Superadmin · ขาดแถว settings ของชั้นใดชั้นหนึ่ง / ของ Member สำหรับเกมนี้ · `P[i+1] > P[i]` · `pt` ของ Member > `P[k]` · `parent_id` ของแถวไม่ตรงกับลำดับใน `cnf` (รวมแถวของ Member ต้องเป็น `Lk`) |
| PTC-10 | ข้อสังเกต (ไม่ต้องเขียนโค้ดเพิ่ม): ถ้าข้อมูลตรง MGMT-18 / MGMT-25 (`Force` / `Remain` ≤ ค่าที่ให้ลูก) ส่วนที่ขาดจะไม่เกิน `left` เสมอ (ยังจริงเมื่อมีชั้นไม่รับ PT เพราะ `hold` ที่ลดไปย้ายไปอยู่ใน `left` เท่ากัน) — `min(…, left)` เป็นแค่ตัวกัน · ถ้าโดนตัดจริงให้ log `warn` ไว้ตรวจข้อมูล |

ตัวอย่างอ่านได้ใน [docs/references/pt-force-remain-commission.md](../references/pt-force-remain-commission.md) ข้อ 3 — ตรงกับ PTC-03 – PTC-06 เมื่อ `k = 3` (ไม่มีชั้นไม่รับ PT / CSM)

```
eff  = hold                         // ชั้นไม่รับ PT: hold = 0 (PTC-02A)
left = 100 − Σ hold
for i = k down to 1:
    if Force[i] > 0 และ Li รับ PT และ Li ไม่ใช่ CSM:      // PTC-04 / 04A
        s = Σ eff[i..k] ; if s < Force[i]:  add = min(Force[i] − s, left) ; eff[i]   += add ; left −= add
    if Remain[i] > 0:                                      // PTC-05
        s = Σ eff[i..k] ; if s < Remain[i]: add = min(Remain[i] − s, left) ; eff[i−1] += add ; left −= add
eff[0] += left                                             // PTC-06
```

### Commission

| ID | Rule |
|---|---|
| PTC-11 | % Commission ที่ใช้คิด (input ของ PTC-13): ชั้นฝั่ง agent = `agent_game_settings.commission` ของชั้นนั้น (ผู้สร้างตั้งให้) · Superadmin = `0` เสมอ (ไม่มีผู้สร้าง) · Member = `user_member_game_settings.commission` |
| PTC-12 | **ไม่คืน % ที่ตั้งไว้** (แก้ 2026-10-16 — คืนแค่ `commission_net`) · ไม่คิดเป็นเงิน (ไม่มียอดแทงใน input — คูณยอดแทงที่ module เดิมพัน / Settle) |
| PTC-13 | **`commission_net`** = % ของยอดแทงที่ชั้นนั้นได้ / เสีย **สุทธิ** (แบบ ก — ชั้นบนตรงจ่าย commission ของลูกทั้งก้อน · ตาราง "จ่ายจริง" ใน reference ข้อ 4 กลับเครื่องหมาย · lead อนุมัติ 2026-10-16 ผ่าน maofoy): |
| | • `t[j] = Round4(%Com[j] × PTเหนือ[j] ÷ 100)` · `PTเหนือ[j]` = PT สุดท้ายของทุกชั้นเหนือ `Lj` รวมกัน · Member: `PTเหนือ` = 100 |
| | • Superadmin = `−t[1]` · `Li` (1 … k) = `t[i] − t[i+1]` (ได้จากผู้สร้าง − จ่ายลูก · ลูกของ `Lk` = Member) · Member = `t[Member]` · ทุกค่า Round4 · ไม่มี `-0` |
| | • ตรวจ: Σ `commission_net` ทุกแถว = 0 (ได้เท่ากับเสีย) |
| | • ตัวอย่าง: Agent ไม่ได้ commission จากชั้นบน (0) แต่ให้ Member 0.1% → Agent `−0.1` · Member `+0.1` · ชั้นที่ได้ commission ได้จริงเฉพาะส่วนที่ชั้นบนถือ PT (เช่น ตั้ง 0.3% · PT เหนือ 95% → ได้ 0.285%) |
| | • ยอดเงิน = ยอดแทง × `commission_net` ÷ 100 (คิดที่ module เดิมพัน) |

### Input / ผลลัพธ์

| ID | Rule |
|---|---|
| PTC-20 | `member_id` บังคับ > 0 · `game_code` บังคับ ต้องเป็นเกมที่มีในระบบ (`app/core/game`) — ไม่ผ่าน = `422` (msg บอก field) |
| PTC-21 | ไม่พบ Member = `408401` |
| PTC-22 | ผลเป็น array เรียงจาก Superadmin ลงไปถึง Member (ตามลำดับใน `cnf` + Member ท้ายสุด) · Agent ซ้อนหลายชั้น = มี `position: "agent"` หลายแถว · แถว Member ไม่มี `pt` |
| PTC-23 | `position` = role ตัวเล็กตาม `cnf` (`superadmin` · `company` · `shareholder` · `agent`) · Member = `member` · `id` ของ Member เป็น id ใน `user_members` (คนละตารางกับฝั่ง agent — อาจซ้ำกับ id ของ agent ได้ ให้ดูคู่กับ `position`) |
| PTC-24 | ค่า % ส่งตามที่เก็บ (ปัด 4 ตำแหน่งแล้ว) หน้าบ้านปัดเองตอนแสดง (กฎข้อ 9) |

### อ่านข้อมูล

| ID | Rule |
|---|---|
| PTC-30 | อ่านอย่างเดียว ไม่ lock · ใช้ `db` ที่ผู้เรียกส่งมา (module เดิมพันส่ง tx ของ bet ได้) · query ไม่เกิน 3 ครั้งต่อการคิด: Member (`cnf`) · settings ของ Member · settings ของทุกชั้นในสาย JOIN `user_agents` เอา `agent_type` (หา CSM) (`agent_id IN (…) AND game_code = ?` — ห้ามวนทีละชั้น · กฎข้อ 23) · `position` ของทุกชั้นจาก `cnf` |
| PTC-31 | ใช้ index ที่มีอยู่: `agent_game_settings` PK `(agent_id, game_code)` · `user_member_game_settings` PK `(user_member_id, game_code)` — ไม่ต้องเพิ่ม migration |

### เส้น dev

| ID | Rule |
|---|---|
แก้ตาม lead รอบ 2 (2026-10-15 · แทน Q5 เดิม ADMIN + `/bo/pr/dev`)

| ID | Rule |
|---|---|
| PTC-40 | เส้น dev อยู่ **context ใหม่ `app/internals/developer/`** (`routes.go` + `controllers/pt_calculation` + `dto/pt_calculation`) · path `/api/v1/dev/pt/calculate` · `pkg/routes` เรียก `developer.RegisterRoutes(api)` **เฉพาะเมื่อ** `configs.Cfg.DevRoutesEnabled = true` · service / core / repository อยู่ที่เดิม · เครื่องมือ dev อื่นในอนาคตอยู่ context นี้ |
| PTC-41 | config `DEV_ROUTES_ENABLED` (bool · ค่าเริ่มต้น `false`) · `config.go` + `.env.example` (กฎข้อ 25) · **กันแค่ prod** — `APP_ENV=prod` และ `DEV_ROUTES_ENABLED=true` = boot ไม่ผ่าน |
| PTC-42 | **ไม่ต้อง login** (public เฉพาะ env ที่เปิด flag — CLAUDE.md กฎข้อ 28 ข้อยกเว้น `developer`) · ไม่ใช้ middleware ของ backoffice · เห็นทุกสาย (ข้อยกเว้นกฎข้อ 22 เฉพาะเส้น dev) |
| PTC-43 | ปิด flag แล้วเส้นไม่มีอยู่ = `404001` (ไม่บอกว่ามีเส้นนี้) |
| PTC-44 | `508601` service log เอง (`pt chain invalid`) พร้อม `member_id` · `game_code` · `cnf` · settings ของ Member · settings ทุกชั้นในสาย · สาเหตุ — ผู้เรียก (module เดิมพัน) ห่อ error ต่อแล้ว log ไม่หาย (lead รอบ 2 S1) |

## 4. สิ่งที่พบในโค้ดเก่า และการตัดสินใจ

ไม่ได้ extract จากโค้ดเก่า (rule มาจาก reference + spec ที่อนุมัติแล้ว)

## 5. Endpoints

| Method | Path | สิทธิ์ที่ต้องมี | Contract เทียบของเก่า |
|---|---|---|---|
| POST | `/api/v1/dev/pt/calculate` | ไม่ต้อง login · มีเฉพาะเมื่อ `DEV_ROUTES_ENABLED=true` (prod เปิดไม่ได้) | ใหม่ (เส้น dev ไม่แจ้ง frontend) |

### POST /api/v1/dev/pt/calculate

Request:
```json
{ "member_id": 123, "game_code": "coin_toss" }
```

Response `data` (ตัวอย่างจาก reference ข้อ 3 / 4 — ดูข้อมูลตั้งต้นในหัวข้อ 7.1 · แทง 1,000 → `commission_net` × 10 = คอลัมน์ "จ่ายจริง" กลับเครื่องหมาย):
```json
[
  { "id": 1,   "position": "superadmin",  "pt": 20, "commission_net": -0.2 },
  { "id": 2,   "position": "company",     "pt": 25, "commission_net": -0.115 },
  { "id": 3,   "position": "shareholder", "pt": 25, "commission_net": 0.105 },
  { "id": 4,   "position": "agent",       "pt": 30, "commission_net": -0.79 },
  { "id": 123, "position": "member",                "commission_net": 1 }
]
```

Error codes: `422` · `408401` · `508601` · `404001` (flag ปิด)

### Service (ให้ module เดิมพันเรียก)

```go
// app/service/pt_calculation
func CalculateMemberPT(ctx context.Context, db *gorm.DB, memberID uint, gameCode string) ([]Layer, error) // module เดิมพันส่ง tx ของ bet
func CalculateMemberPTService(ctx context.Context, memberID uint, gameCode string) ([]Layer, error)        // เส้น dev (ใช้ connection หลัก)

// app/core/pt_calculation — pure function + table test (กฎข้อ 6) · คืน warning ให้ service log (core ห้ามใช้ logger)
func Calculate(in Input) (Result, error) // in: ต่อชั้น P · Force · Remain · รับ PT ไหม · เป็น CSM ไหม + pt ของ Member → PT สุดท้าย[0..k]
```

## 6. Schema

ไม่มีตารางใหม่ · ไม่มี migration · อ่าน `user_members` (`cnf`) · `user_member_game_settings` · `agent_game_settings`

## 7. Test cases

### 7.1 ข้อมูลตั้งต้น

| ชุด | สาย (`P` = ค่าที่ได้รับ · F / R = Force / Remain ในแถวของชั้นนั้น) | Member | PT สุดท้าย |
|---|---|---|---|
| A (reference ข้อ 3) | SA → Company `P` 95 → Share `P` 85 F 20 R 70 → Agent `P` 70 F 30 R 40 | `pt` 20 | SA 20 · Co 25 · Sh 25 · Ag 30 |
| B (Agent ซ้อน) | SA → Company `P` 90 → Share B2B `P` 80 → Agent a1 `P` 70 F 0 R 60 → Agent a2 `P` 50 F 40 R 45 | `pt` 30 | SA 15 · Co 10 · Sh 10 · a1 25 · a2 40 |
| C (Company 1 to 1 สร้าง Member) | SA → Company 1to1 `P` 90 F 80 | `pt` 60 | SA 20 · Co 80 |
| D (ไม่มี Force / Remain) | SA → Company `P` 90 → Share `P` 70 → Agent `P` 60 | `pt` 40 | SA 30 · Co 20 · Sh 10 · Ag 40 (เหลือ 20 คืน SA — ตรงตัวอย่าง m2 ใน agent_management) |

| E (ชั้นกลางไม่รับ PT — lead Q2) | ชุด A แต่ Share `status = false` | `pt` 20 | SA 20 · Co 40 · Sh 10 · Ag 30 |
| E2 (ผู้สร้าง Member ไม่รับ PT) | ชุด A แต่ Agent `status = false` | `pt` 20 | SA 20 · Co 25 · Sh 55 · Ag 0 |
| F (CSM มี force ค้าง — กันพลาด) | SA → CSM `P` 80 F 60 → Share Master `P` 80 | `pt` 50 | SA 50 · CSM 0 · SM 50 (ข้าม Force ของ CSM + log warn) |

ไล่ชุด E: hold 5 / 10 / **0** / 20 · left 65 → Ag Force 30: 20 < 30 → Ag +10 = 30 (left 55) → Ag Remain 40: 30 < 40 → **Sh +10 = 10** (Remain ที่ Share ตั้งให้ลูกยังทำงาน · left 45) →
Sh Force 20: **ข้าม** (Share ไม่รับ PT) → Sh Remain 70: 10 + 30 = 40 < 70 → Co +30 = 40 (left 15) → คืน SA 15 → 20 · รวม 20 + 40 + 10 + 30 = 100

ไล่ชุด E2: hold 5 / 10 / 15 / **0** · left 70 → Ag Force: **ข้าม** → Ag Remain 40: 0 < 40 → Sh +40 = 55 (left 30) → Sh Force 20: 55 ≥ 20 ไม่ทำงาน →
Sh Remain 70: 55 < 70 → Co +15 = 25 (left 15) → คืน SA 15 → 20 · รวม 20 + 25 + 55 + 0 = 100

ไล่ชุด B: hold 10 / 10 / 10 / 20 / 30 · left 20 → a2 Force 40: 30 < 40 → a2 +10 = 40 (left 10) → a2 Remain 45: 40 < 45 → a1 +5 = 25 (left 5) →
a1 Remain 60: 25 + 40 = 65 ≥ 60 ไม่ทำงาน → คืน SA 5 → 15 · รวม 15 + 10 + 10 + 25 + 40 = 100

### 7.2 Test

| Rule ID | Input | Expected |
|---|---|---|
| PTC-03 – 06 | ชุด A · B · C · D (core table test + integration) | PT สุดท้ายตาม 7.1 · รวม 100 |
| PTC-02A | ชุด E · E2 (core + integration — ชั้นกลาง `status = false` ตาม lead Q2) | PT สุดท้ายตาม 7.1 |
| PTC-04A | ชุด F (core · integration ตั้ง `force` ของ CSM ผ่าน SQL ตรง เพราะ API ห้ามแล้ว) | ตาม 7.1 · มี warning CSM force |
| PTC-04 | Force = `s` พอดี | ไม่ทำงาน (`<` ไม่ใช่ `≤`) |
| PTC-05 | Force แล้ว `s` ≥ Remain | Remain ไม่ทำงาน |
| PTC-07 | (core table test เท่านั้น — ค่าจริงตั้งได้ทีละ 0.5% ตาม MGMT-18) ค่าที่มีเศษ เช่น `P` 33.3333 / `pt` 11.1111 | ทุกค่าปัด 4 ตำแหน่ง · รวม 100 |
| PTC-10 | ข้อมูลจงใจผิด MGMT-18 (Force > ค่าที่ได้รับ) | ถือเพิ่มไม่เกิน `left` · log warn · รวม 100 |
| PTC-09 | ลบแถว settings ของชั้นกลาง · `P` ลูก > `P` ผู้สร้าง | `508601` |
| PTC-11 | ชุด A | ใช้ commission ที่ตั้งไว้ (SA = 0) คิด `commission_net` · response ไม่มี field `commission` |
| PTC-13 | ชุด A + commission Co 1 · Sh 0.7 · Ag 0.3 · Member 1 (reference ข้อ 4) | `commission_net` SA −0.2 · Co −0.115 · Sh 0.105 · Ag −0.79 · Member 1 · รวม 0 |
| PTC-13 | PT 77 / 14 / 4 / 5 · commission 0 / 0 / 0 · Member 0.1 (Agent ให้ส่วนลด Member อย่างเดียว) | Agent −0.1 · Member 0.1 · ชั้นอื่น 0 (ไม่มี `-0`) |
| PTC-13 | PT 77 / 14 / 4 / 5 · SA→Co 0.5 · Co→Sh 0.4 · Sh→Ag 0.3 · Ag→Member 0.1 | SA −0.385 · Co 0.021 · Sh 0.079 · Ag 0.185 · Member 0.1 |
| PTC-20 | ไม่ส่ง `member_id` · `game_code` ไม่มีจริง | `422` msg บอก field |
| PTC-21 | `member_id` ไม่มีจริง | `408401` |
| PTC-22 | ชุด B | 6 แถว · `agent` 2 แถวเรียงบน → ล่าง |
| PTC-30 | ชุด D (4 ชั้น · integration ใช้ gorm logger นับ SQL) | query = 3 ครั้งพอดี (0 = counter ไม่ทำงาน → fail) |
| PTC-41 – 43 | flag ปิด · flag เปิด ไม่ส่ง token · `APP_ENV=prod` + flag เปิด (unit test ของ `configs.Load`) | `404001` · `200` · boot ไม่ผ่าน |
| PTC-44 | ข้อมูลในสายผิด | `508601` + log `pt chain invalid` ที่ service |

## 8. Contract changes

ไม่มี — เส้น dev ไม่แจ้ง frontend

## 9. Error codes (`bb=08`)

| Code | HTTP | ความหมาย |
|---|---|---|
| 408401 | 200 | ไม่พบ Member |
| 508601 | 500 | ข้อมูล PT ในสายไม่ถูกต้อง คิด PT ไม่ได้ (log รายละเอียด · msg ไม่บอกรายละเอียดภายใน) |

## 10. การตัดสินของ lead (zerph · 2026-10-15)

| # | เรื่อง | ผล |
|---|---|---|
| Q1 | ใช้กฎใน reference เป็น rule จริง (PTC-03 – PTC-06) และขยายเป็นสายยาว N ชั้น | **อนุมัติ** |
| Q2 | ชั้นที่ `pt.status = false` (ปิด O1 ของ agent_management · lead Q-R3) | **ถือ 0 · ไม่โดน Force ของตัวเอง · ส่วนนั้นไหลตามกฎปกติ · Remain ที่ชั้นนั้นตั้งให้ลูกยังทำงาน** (PTC-02A) · ไม่ใช้ `408301` (ตัดออก) |
| Q3 | CSM ถือสู้ไม่ได้ แต่เดิมตั้ง `force` ได้ | **ห้ามตั้ง `force` ให้ CSM** — agent_management MGMT-18A (`422`) + migration ตั้ง `force` ของ CSM เดิมเป็น 0 · module นี้ข้าม Force ของ CSM กันพลาด (PTC-04A) |
| Q4 | ไม่มีกฎ Force ≤ Remain | **รับทราบ — ไม่ validate** |
| Q5 | เส้น dev ADMIN · `DEV_ROUTES_ENABLED` · prod เปิดไม่ได้ | ~~อนุมัติ~~ → **แก้ในรอบ 2:** ย้ายไป context `developer` `/api/v1/dev` ไม่ต้อง login · flag กันแค่ prod (PTC-40 – PTC-43) |

### รอบ 2 (ตรวจโค้ด · lead zerph 2026-10-15)

| # | ผล |
|---|---|
| M1 | integration test (pt_calculation · CSM force · migration `20261015150000`) ต้องรันผ่านและแนบผลก่อน lead เริ่ม bet core |
| S1 | service log `508601` + ค่าทั้งสายเอง → PTC-44 |
| S2 | เพิ่ม integration ชุด C / D / E2 + นับ query ≤ 3 (หัวข้อ 7.2) |
| L1 | Member `pt = 0` แต่ผู้สร้างมี Force → ผู้สร้างถือตาม Force — **ตั้งใจ** (Force = บังคับถืออย่างน้อยเท่านี้) |
| L2 | ปิด `pt.status` ของ CSM → sync ไป Share Master → ทั้งสายถือ 0 คืน Superadmin — **ตั้งใจ** (ตรง Q2) |

### Commission สุทธิ (lead อนุมัติ 2026-10-16 ผ่าน maofoy)

| # | เรื่อง | ผล |
|---|---|---|
| C1 | ใครแบก commission ที่ให้ลูก: ก) ชั้นบนตรงจ่ายทั้งก้อน · ข) ทุกชั้นเหนือลูกจ่ายตามสัดส่วน PT | **ก** (ตาราง "จ่ายจริง" ใน reference ข้อ 4) → PTC-13 `commission_net` |
| C2 | คิดที่ไหน | service กลาง `CalculateMemberPT` (module เดิมพันใช้ต่อ) + เส้น dev แสดงผล |
