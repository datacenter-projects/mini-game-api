# Asset · ธีมเกม · Branding (`asset`) — Spec

- สถานะ: **DRAFT** — lead review แล้ว 2026-10-12 ([docs/reviews/game-asset.md](../reviews/game-asset.md)) · แก้ตามรีวิวแล้ว รอ lead รับทราบฉบับแก้ (หัวข้อ 9) ก่อน implement
- อนุมัติโดย: — (รอ lead) · ร่าง 2026-10-08 · แก้ตามรีวิว 2026-10-12
- ผู้ดูแล: schneider
- ชื่อ module ในโค้ด: `asset` (`controllers/asset`, `dto/asset`, `service/asset`, `core/asset`)
- เมนูสิทธิ์: `asset` ใช้ทุกเส้นของ module (off / view / edit — `agent_management.md` MGMT-50–52 · lead GQ3)
- การตัดสินของ lead (รีวิว 2026-10-12): scope MVP **GS1** · อัปแล้วขึ้นจริงไหม **GR1** · ลบ **GR2** · ขนาดไฟล์ **GR3 + GQ11** · เจ้าของ + การสืบทอด **GR4 + GQ1 + GQ2** · reset **GR5** · lobby ผู้เล่น **GR6** · **GQ3–GQ13** · ชื่อเส้น **GN1–GN23** — สรุปว่าตัดสินอะไรไว้ที่ไหนในหัวข้อ 9 · **rule ในเอกสารนี้มาจากข้อเสนอที่ lead อนุมัติแล้ว** (`ASSET_SYSTEM_SPEC.md` เป็นข้อเสนอ · ตรวจกับโค้ดเก่าแล้วในหัวข้อ 4)
- โค้ดเก่าที่อ้างอิง (extract พฤติกรรมเท่านั้น): `askmelotto_minigame-api/app/services/asset/*.go`,
  `app/internals/backoffice/middleware/asset_target_middleware.go`, `app/internals/backoffice/routes/module_routes.go`,
  `app/repository/postgres/minigame_asset_*_repository.go`, `app/services/agent/*site*_service.go`,
  `database/postgres/seeddata/seed_minigame_asset_type.go`, `platform/s3client/s3.go`
- spec ที่อ้างถึง: [agent_auth.md](agent_auth.md), [agent_auth_phase2.md](agent_auth_phase2.md),
  [agent_management.md](agent_management.md), [member_management.md](member_management.md), [account.md](account.md) (APPROVED ทั้งหมด)

> เอกสารนี้คือ source of truth ของ module — โค้ดและ test ต้องตรงกับเอกสารนี้ ถ้าไม่ตรงให้แก้โค้ด
> หรือแก้เอกสารผ่านการอนุมัติใหม่ ห้ามอ้างเอกสารของโปรเจกต์เก่า

## 1. หน้าที่ของ module

จัดการหน้าตาของเกมและหน้าเว็บแบบ multi-tenant: ทุกบัญชีใช้ระบบเดียวกัน ตั้งหน้าตาของตัวเองได้ · บัญชีที่ไม่ได้ตั้งได้ค่าของ
ชั้นบนที่ใกล้ที่สุด แล้วตกไปที่ค่ากลาง (GLOBAL) — ทั้ง asset, theme และ branding (lead GR4)

**MVP รอบนี้ (lead GS1 · ~8.5 pd — ถ้าไม่ทันจริงลดเหลือ GLOBAL อย่างเดียว ~5 pd):**
- **catalog** ของช่อง asset + seed (migration)
- **Game Asset** — อัปรูปเข้าช่อง (`use_now`), คลังรูป, เลือกรูปที่ใช้งาน, ลบ (soft delete), reset กลับไปใช้ของชั้นบน
- **Game Theme** — ฟอนต์ + สีหลัก 2 สี + สี / ฟอนต์ของแถบสถิติ ต่อเกม + ชุดเฉดสี + reset
- **Branding** — ชื่อเว็บ + โลโก้ + reset
- **Bundle** — รวม asset + theme + branding ในคำขอเดียว: หลังบ้าน preview · **ผู้เล่น: กำหนด contract ไว้ implement หลัง Player Auth**
  (มี lobby ของเราเอง — lead GR6 · เส้นรายชื่อเกมของ lobby ตัดสินพร้อม Player Auth — GQ9)

**เลื่อนไปทำภายหลัง (ดูหัวข้อ "ภายหลัง" ท้ายหัวข้อ 3):** banner · favicon · site-themes (+ `site_theme` ของ branding) ·
change log (ระหว่างนี้ใช้ `created_by` / `updated_by` + คลังรูป)

**ไม่อยู่ใน module นี้:** CDN / การ purge cache · job กวาดไฟล์ขยะใน storage (job แยกนอก request) ·
การสร้าง / แก้ catalog ผ่าน API (migration เท่านั้น — AST-11) · ตัวเกมเอง (`app/internals/game`)

## 2. คำศัพท์

| คำ | ความหมาย |
|---|---|
| เจ้าของหน้าตา | บัญชีหลักฝั่ง agent ที่ตั้งหน้าตาของตัวเองได้ (AST-02) |
| GLOBAL | ค่ากลางของระบบ — ปลายทางสุดท้ายของการไล่สาย · เขียนได้เฉพาะ Superadmin (AST-03) |
| AGENT | ค่าที่ตั้งไว้ที่บัญชีใดบัญชีหนึ่ง (`agent_id` = id ของบัญชีนั้น) |
| สาย (chain) | ลำดับบัญชีที่ใช้ไล่หาค่า: ตัวเอง → ชั้นบนจากใกล้ → ไกล → GLOBAL (AST-57) |
| ชั้นบน (upline) | บัญชีใน `cnf` ของบัญชีนั้น (MGMT-61) |
| CSM | Company Seamless Master — ตั้งครั้งเดียว Share Master ทุกคนใต้ตัวเองได้ตามผ่านสาย (lead H2) |
| เป้าหมาย | scope + บัญชีที่คำขอนั้นทำงานด้วย (AST-08) |
| ช่อง (slot) | หน่วยที่ปรับแต่งได้ 1 หน่วย = (เป้าหมาย, `game_code`, `asset_type`) |
| catalog | รายการช่องที่มีได้ของแต่ละเกม (`asset_types`) พร้อมขนาดภาพที่รับ |
| config | ไฟล์ 1 เวอร์ชันที่เคยอัปโหลด (`asset_configs`) — เก็บทุกเวอร์ชัน = "คลังรูป" |
| used | config ที่ถูกเลือกใช้ของช่อง (`asset_used`) — 1 แถวต่อ (เป้าหมาย, ช่อง) |
| ใช้งานจริง | ค่าที่เกม render ของช่อง = `used` ที่ใกล้ที่สุดในสาย (AST-24, AST-25) |
| `source` | ที่มาของค่า: `self` (ตัวเองตั้ง) · `upline` (ได้มาจากชั้นบน) · `global` (ค่ากลาง) · `mixed` (ทั้งชุด — บางช่องมาจากหลายที่) |
| `use_now` | ตอนอัป: `true` (ค่าเริ่มต้น) = ตั้งเป็น used ทันที · `false` = เก็บไว้ในคลังเฉยๆ (AST-21) |
| reset | ลบค่าที่ตั้งเองของบัญชีนั้น เพื่อกลับไปใช้ของชั้นบน / ค่ากลาง (AST-54–56) |
| theme | ฟอนต์ + สี 5 ค่าของเกม (AST-33) |
| shades | ชุดเฉด 19 ระดับที่คำนวณจากสีหลัก 1 สี (AST-39) |
| branding | ชื่อเว็บ + โลโก้ (MVP) ของบัญชี |
| storage key | path ของไฟล์ใน object storage (S3) ที่เก็บใน DB — URL สร้างตอนตอบ (AST-49) |

## 3. Business rules

แต่ละข้อมี id เพื่ออ้างใน test และ code (`// rule: AST-03`) · เลขที่ตัด / เลื่อนไม่นำมาใช้ซ้ำ

### Scope และสิทธิ์

| ID | Rule |
|---|---|
| AST-01 | asset และ theme มี **2 scope**: `GLOBAL` · `AGENT` · เจ้าของค่าคือ**บัญชีเดี่ยว** (ไม่มีระดับ "กลุ่มเว็บ") · บัญชีที่ไม่ได้ตั้งเองได้ค่าจากชั้นบนตาม AST-57 · branding มีแค่ `AGENT` (ไม่มี GLOBAL) |
| AST-02 | **เจ้าของหน้าตา** (ตั้ง scope `AGENT` ได้) = บัญชีหลักฝั่ง agent: Company ทุกประเภท (รวม CSM) · Share B2B / B2C / Reseller · Agent · **Share Master ตั้งเองไม่ได้** — ได้ตามที่ CSM ตั้งผ่านสาย (lead H2) → เขียน = `405301` · Superadmin ตั้งได้เฉพาะ GLOBAL (AST-03) · ADMIN ไม่มีเมนู `asset` (AST-06) · **ไม่มีข้อจำกัดว่า "เป็นเว็บหรือไม่"** (ตัด `405302` — lead GR4) |
| AST-03 | **เขียน GLOBAL**: SUPERADMIN (บัญชีหลัก) · sub ของ SUPERADMIN ต้องมี `asset` = `edit` · บัญชีอื่น → `405301` |
| AST-04 | **เขียน AGENT**: เฉพาะบัญชีตัวเอง (บัญชีหลักที่เป็นเจ้าของหน้าตา AST-02) หรือ sub ของบัญชีนั้นที่มี `asset` = `edit` (ทำในนามเจ้าของ) · เป้าหมายเป็นบัญชีในสายล่าง = **ดูได้ แก้ไม่ได้** → `405301` · เป้าหมายนอกสาย → `402402` · CSM ตั้งของ CSM ครั้งเดียว Share Master ทุกคนใต้ได้ตามเอง (ไม่ต้องตั้งแทนทีละคน) |
| AST-05 | **อ่าน**: ต้องมี `asset` ≥ `view` ทุกเส้น · อ่าน GLOBAL ได้ทุกบัญชีที่มีสิทธิ์ · อ่าน AGENT ได้: ตัวเอง และบัญชีในสายล่าง (ชั้นบนดูของสายล่างได้ — ตรวจจาก `cnf` ของเป้าหมาย) · นอกสาย → `402402` · Share Master อ่านค่าที่ได้จาก CSM ของตัวเองได้ (อ่านอย่างเดียว) |
| AST-06 | sub ต้องมีสิทธิ์ `asset` ทุกเส้นของ module นี้ (AUTH-26 · lead A5) · ADMIN ไม่มีเมนู `asset` (AUTH-43) → `402303` · บัญชีที่ `status` ไม่ใช่ `ACTIVE` ถูกกันที่ `PassedGates()` (AUTH-54 → `401311` · AUTH-27) · ทุกเส้นไม่ต้องยืนยัน passcode |
| AST-07 | middleware ตอบแค่ "ผู้เรียกทำกับเป้าหมายนี้ได้ไหม" — **ไม่ได้พิสูจน์ว่า resource เป็นของเป้าหมาย** · service ต้องเช็คซ้ำทุกครั้งว่า config ที่ระบุเป็นของเป้าหมาย (หรือเป็นของ GLOBAL เมื่อกฎอนุญาต) |

### การระบุเป้าหมาย

| ID | Rule |
|---|---|
| AST-08 | เส้นที่เลือก scope ได้ส่ง `scope` = `"GLOBAL"` / `"AGENT"` **บังคับ** · `AGENT` ต้องส่ง `agent_id` > 0 · `GLOBAL` ต้องไม่ส่ง `agent_id` หรือส่ง `0` · เส้น branding ส่ง `agent_id` > 0 อย่างเดียว (ไม่มี scope) · ไม่ส่ง / ค่าอื่น / `null` / อ่านไม่ออก → `422` msg บอก field (JSON ใช้ `utils.ParseBodyNoNull`) · เส้น multipart ส่งเป็น **form field** (lead GQ4) · **ไม่มีค่าพิเศษ `me`** · ห้ามตีความค่าว่างหรือ 0 เป็น GLOBAL เงียบๆ — พิมพ์ผิดต้องไม่กลายเป็นการเขียนทับค่ากลาง |
| AST-09 | middleware `ResolveAssetTarget` เก็บผลเป็น struct ที่บอกชัดว่า resolve แล้วหรือยัง และได้ scope ไหน · guard ที่มีความหมายเฉพาะฝั่งเดียว (AST-02, AST-04) ต้องอยู่หลังตัวแยกฝั่ง · ถ้าอ่านผล resolve ไม่ได้ → ปฏิเสธ (fail-closed) ห้ามปล่อยผ่าน |

### Catalog

| ID | Rule |
|---|---|
| AST-10 | รหัสเกมใช้ชุดเดียวกับ MGMT-16: `coin_toss` · `rock_paper_scissors` · `scratch_card` (อยู่ใน `app/core`) · module นี้ไม่มีตารางเกม |
| AST-11 | ช่องต้องมาจาก `asset_types` เท่านั้น · catalog เพิ่ม / แก้ขนาด / เลิกใช้ ด้วย migration ใหม่เท่านั้น · **ห้ามสร้างช่องจากชื่อที่ส่งมาตอนอัปโหลด** |
| AST-12 | `allowed_sizes` = รายการ `{width, height}` (ได้หลายขนาดต่อช่อง) · `[]` = ไม่ตรวจขนาด · `slots/list` กับ `files/upload` อ่านจากตารางเดียวกัน ห้ามมีค่าขนาดซ้ำในโค้ด |
| AST-13 | เลิกใช้ช่อง = `is_retired = true` **ห้ามลบแถว** · ช่องที่เลิกใช้ไม่แสดงใน `slots/list` · อัปโหลด / เลือกใช้ใหม่ไม่ได้ (`405304`) · ไฟล์ที่ใช้งานอยู่แล้วยังอยู่ใน `used/list` / `bundle` (เกมที่ยังอ้างช่องนั้นต้องไม่พัง) |

### อัปโหลด

| ID | Rule |
|---|---|
| AST-14 | `game_code` และ `asset_type` ต้องตรงกับ catalog **ทุกตัวอักษร** (ไม่แปลงตัวพิมพ์ ไม่แทนช่องว่าง) · ไม่พบ → `405401` |
| AST-15 | ลำดับตรวจ: สิทธิ์ → catalog (AST-14) → เลิกใช้ (AST-13) → ขนาด / ชนิดไฟล์ (AST-16) → ขนาดภาพ (AST-17) → **แล้วค่อยอัปขึ้น storage** · ไม่ผ่านข้อใด = ไม่มีไฟล์ขึ้น bucket |
| AST-16 | ไฟล์ ≤ **5 MB** (`405103` — lead GR3 · `BodyLimit` 10 MB ของ request ไม่ต้องแก้) · ชนิด `png` · `jpeg` · `gif` · `webp` ตรวจจาก**เนื้อไฟล์** (magic bytes) ไม่เชื่อ `Content-Type` ที่ client ส่ง · **ห้าม SVG** (`405102`) · นามสกุลของ storage key มาจากชนิดที่ตรวจได้ · object ที่อัปตั้ง `Content-Type` ตามชนิดที่ตรวจได้ และ `Cache-Control: public, max-age=31536000, immutable` (key ไม่ซ้ำ จึง cache ยาวได้) |
| AST-17 | ขนาดภาพ: ช่องมีสเปก + อ่านขนาดได้ + ไม่ตรงขนาดใดเลย → `405101` msg บอกขนาดที่รับ · **ผ่านโดยตั้งใจ (fail-open) 2 กรณี**: ช่องไม่มีสเปก · อ่านขนาดจากไฟล์ไม่ได้ — ขนาดภาพเป็นเรื่องหน้าตา ไม่ใช่ความปลอดภัย |
| AST-18 | อ่าน catalog จาก DB ไม่สำเร็จ → `500` ปฏิเสธ (**fail-closed** — ห้ามข้ามไปบันทึกต่อ) |
| AST-19 | storage ไม่พร้อม → `505801` · อัปขึ้น storage ไม่สำเร็จ / timeout → `505802` (ไม่มีแถว DB) · อัปสำเร็จแต่บันทึก DB ไม่สำเร็จ → ลบ object ทิ้ง (ลบไม่ได้ log error พร้อม key) แล้วตอบ `500` |
| AST-20 | อัปโหลดทุกครั้งสร้าง config ใหม่ 1 แถว (เก็บทุกเวอร์ชัน) · `created_by` = username จริงของผู้กด (sub = username ของ sub) · **ไม่มีโควตา ไม่มีการตัดเวอร์ชันเก่าอัตโนมัติในรูปแบบใด** — ลบได้จาก `files/delete` เท่านั้น |
| AST-21 | `files/upload` มี `use_now` (ค่าเริ่มต้น `true` — lead GR1): `true` → สร้าง config + ตั้งเป็น used ของช่องนั้นของเป้าหมายใน **transaction เดียวกัน** · `false` → สร้าง config อย่างเดียว ไม่แตะ `asset_used` (อยู่ในคลัง เลือกใช้ทีหลังด้วย `used/update`) |

### ไฟล์ที่ใช้งานจริง (used) และการไล่สาย

| ID | Rule |
|---|---|
| AST-22 | `asset_used` มี 1 แถวต่อ (เป้าหมาย, ช่อง) · เลือกไฟล์ใหม่ = **แก้แถวเดิม** (upsert `ON CONFLICT`) ไม่ใช่เพิ่มแถว · unique ต้องเป็น (เป้าหมาย, ช่อง) **ห้ามรวม `asset_config_id`** · GLOBAL ใช้ partial unique index แยก (Postgres มอง `NULL` ไม่เท่ากัน) |
| AST-23 | `used/update`: config ต้อง (ก) มีอยู่และไม่ถูกลบ (ข) เป็นช่องและเกมเดียวกับ `game_code` ที่ส่ง (ค) เป้าหมาย `GLOBAL` เลือกได้เฉพาะ config `GLOBAL` · เป้าหมาย `AGENT` เลือกได้ config `GLOBAL` หรือของบัญชีนั้นเอง (ไม่เลือก config ของชั้นบน — ใช้การสืบทอดแทน) · ช่องเลิกใช้ → `405304` · ผิด (ก)–(ค) ข้อใดก็ตอบ **`405402` ข้อความเดียวกันหมด** (กันไล่เดา id) |
| AST-24 | **ใช้งานจริง = `used` เท่านั้น** (lead GR1): ช่องไหนไม่มีแถว used ในสายเลยและ GLOBAL ก็ไม่มี → ไม่มีรูปในช่องนั้น · **ไม่มีกฎ "ไม่เคยเลือก = ใช้ไฟล์ล่าสุด"** (ตัดแล้ว) · ก่อน go-live ต้องมี GLOBAL used ของทุกช่องที่เกมใช้ |
| AST-25 | **ไล่ทีละช่อง** ตาม AST-57: ช่องที่ตัวเองมี used → `self` · ไม่มี → used ของชั้นบนที่ใกล้ที่สุด → `upline` · ไม่มีเลย → GLOBAL → `global` · ทั้งชุด: ทุกช่องมาจากที่เดียว = ค่านั้น · ต่างกัน = `mixed` · ไม่มีรูปสักช่อง = `""` · ห้าม fallback ทั้งชุดจาก "ตัวเองมีสักช่องไหม" |
| AST-26 | repository ที่ตอบ "ของบัญชีนี้มีอะไร" กรองเข้ม `scope = AGENT` + `agent_id` ตรง ไม่ปน GLOBAL · การไล่สาย / รวมค่าทำใน service ด้วย pure function ใน `app/core/asset` (AST-57) |
| AST-27 | `files/list` (คลังรูปของช่อง): config ที่ยังไม่ถูกลบของเป้าหมาย + ของ GLOBAL (เมื่อเป้าหมายเป็น `AGENT`) เรียงใหม่ → เก่า · ทุกแถวมี `source` (`self` / `global`), `is_used`, `created_by`, `created_at` · แบ่งหน้า: `page` / `limit` ใน body (`utils.NewPage` — ค่าเริ่มต้น 20 สูงสุด 100) ตอบด้วย `response.Page` (กฎข้อ 19) |
| AST-54 | `used/reset`: ลบแถว used ของ (บัญชีตัวเอง, ช่อง) → กลับไปใช้ของชั้นบน / GLOBAL ตาม AST-25 (lead GR5) · ใช้ได้เฉพาะ scope `AGENT` ของบัญชีที่เขียนได้ตาม AST-04 · `scope = GLOBAL` → `422` · ไม่มีแถวอยู่แล้ว = สำเร็จ (idempotent) · ไม่ลบ config (ยังอยู่ในคลัง) |
| AST-57 | **สาย (chain) สำหรับไล่ค่า** ของบัญชี T: T → ชั้นบนจากใกล้ → ไกล (`user_agents.cnf.parent` กลับลำดับ — MGMT-61) → GLOBAL · sub ใช้สายของเจ้าของ · **Member ใช้สายของผู้สร้างตรง** (T = ผู้สร้าง — Member ไม่มีค่าของตัวเอง) · Superadmin อยู่ในสายแต่ไม่มีแถว `AGENT` (ค่าของ Superadmin คือ GLOBAL) · อ่านค่าของทั้งสายด้วย **query เดียว** (`agent_id` ในสาย หรือ `IS NULL`) แล้วเลือกใน `app/core/asset` ด้วย pure function (table test) · ห้ามวน query ทีละชั้น (กฎข้อ 23) · **กฎเดียวกันทั้ง BO preview และผู้เล่น** (lead GR4) |

### ลบ (soft delete ทั้ง 2 scope — lead GR2)

| ID | Rule |
|---|---|
| AST-28 | `files/delete`: **soft delete** (`is_deleted = true`, `deleted_at`, `deleted_by`) ทั้ง config GLOBAL และ AGENT · ใครลบได้ = ผู้มีสิทธิ์เขียนเป้าหมายนั้น (AST-03 / AST-04) **ไม่ผูกกับคนอัป** (ตัดกฎ "ลบได้เฉพาะคนอัป" และ `405303`) · config ไม่ใช่ของเป้าหมาย / ไม่พบ / ลบแล้ว → `405402` · ลำดับ: tx → `SELECT ... FOR UPDATE` แถว config → นับ used ที่ชี้มา > 0 → `405403` → ตั้ง `is_deleted` · `used/update` ล็อกแถว config เดียวกันแล้วเช็ค `is_deleted` ซ้ำ — ลบกับเลือกใช้พร้อมกันจึงชนะได้ฝั่งเดียว (ไม่มี `500`) |
| AST-29 | **ไม่ลบไฟล์ใน storage ระหว่าง request** — ไฟล์ค้างให้ job กวาดภายหลัง (นอก module / นอกรอบนี้) · soft delete ไม่แตะ storage เลย |
| AST-30 | storage ไม่พร้อม: ระบบต้อง **boot ได้** (lead GQ12) · กระทบเฉพาะ `files/upload` / `logo/upload` (`505801`) · เส้นอ่าน, ลบ, reset ทำงานปกติ (URL สร้างจาก `S3_CDN_BASE_URL` ไม่เรียก storage) |
| AST-31 | **เลือกใช้** กับ **ลบ** เป็นคนละสิทธิ์: บัญชี `AGENT` เลือกใช้ config GLOBAL ได้ (AST-23) แต่ลบ config GLOBAL ผ่าน scope `AGENT` ไม่ได้ · config ของ GLOBAL / ของบัญชีอื่น / ไม่พบ / ลบแล้ว → `405402` ข้อความเดียวกัน |
| AST-32 | `405403` = "ยังถูกใช้งานอยู่" **ข้อความเดียว** · "ใช้อยู่" = มีแถว `asset_used` ใดๆ ชี้มาที่ config (รวมกรณี config GLOBAL ที่บัญชีอื่นเลือกไว้ · และกรณีที่ลูกในสายสืบทอดมา — เพราะลูกที่สืบทอดได้ค่ามาจากแถว used ของชั้นบนที่ชี้ config นั้นอยู่แล้ว) · ลำดับที่ถูก: **เลือกรูปอื่น (`used/update`) หรือ `used/reset` ก่อน แล้วค่อยลบ** |

### ธีมเกม

| ID | Rule |
|---|---|
| AST-33 | theme มี 5 ค่าเสมอ: `font_family` · `main_color_1` · `main_color_2` · `stats_bar_color` · `stats_bar_font_family` · `theme/detail/update` ต้องส่ง**ครบ 5 ค่า** = **แทนทั้งชุด** (ไม่ใช่แบบ section / partial ของ MN2) · ขาดค่าใด → `422` |
| AST-34 | ฟอนต์ (ทั้ง 2 field) เป็น whitelist ตัวพิมพ์ตรงตัว: `noto` · `prompt` · `kanit` · `mitr` · `sarabun` · ไม่รับข้อความอิสระ (กัน CSS injection และฟอนต์ที่หน้าบ้านไม่ได้ preload) · ค่าอื่น → `422` |
| AST-35 | สีหลักตรง `^#[0-9A-Fa-f]{6}$` เท่านั้น (ไม่รับ `#RGB`) · เก็บและตอบเป็นตัวพิมพ์เล็ก · ไม่ตรง → `422` |
| AST-36 | `stats_bar_color` เก็บเป็น**ชื่อ** `"main_color_1"` / `"main_color_2"` ไม่ใช่ hex (เปลี่ยนสีหลักแล้วแถบสถิติตามไปเอง) · response เพิ่ม `stats_bar_color_hex` ที่ resolve แล้ว |
| AST-37 | **ไล่ทั้งชุด** ตาม AST-57: บัญชีใกล้สุดในสายที่มีแถว theme ของเกมนั้น → ใช้ทั้ง 5 ค่าของบัญชีนั้น (`self` / `upline`) · ไม่มีเลย → ค่ากลาง GLOBAL ทั้ง 5 ค่า (`global`) · **ห้ามผสมทีละค่า** (ฟอนต์กับสีออกแบบมาคู่กัน) · ค่ากลางเป็นแถว GLOBAL ใน `game_themes` (seed: `noto` · `#f7447e` · `#ffffff` · `main_color_1` · `prompt`) · ค่าของบัญชีเป็นแถว override แยก **ห้าม copy ค่ากลางเข้าไปตอนสร้างบัญชี** · `theme/reset` (AST-55) กลับไปใช้ของชั้นบน |
| AST-38 | validation ชุดเดียวกัน (AST-33–36) ทั้งเขียน GLOBAL และ AGENT — อยู่ใน `app/core/asset` ที่เดียว |
| AST-39 | **shades** ต่อสีหลัก 1 สี = 19 ระดับ ชื่อ `50` `100` `200` … `800` `900` `1000` … `1800` (index 0–18): |
| | • `900` (index 9) = สีหลักตรงตัว |
| | • index < 9: ผสมขาว `t = (9 − i) / 10` → `c' = c + (255 − c) × t` (`50` ผสมขาว 90%) |
| | • index > 9: ผสมดำ `k = (10 − (i − 9)) / 10` → `c' = c × k` (`1800` เหลือ 10%) |
| | • หาร **10 ไม่ใช่ 9** (ปลายทั้งสองข้างไม่ถึงขาว / ดำสนิท) · คำนวณบน **RGB ไม่ใช่ HSL** · ปัดแบบ round-half-up แล้ว clamp 0–255 · hex ตัวพิมพ์เล็ก |
| AST-40 | shades คำนวณ**สดทุกครั้ง**จากสีที่ resolve แล้ว (หลัง AST-37) ไม่เก็บลง DB · ห้ามคำนวณจากสีกลางเมื่อบัญชีตั้งสีเอง · ต้อง pin ค่าด้วย test fixture ที่เทียบกับไฟล์ palette จริงของหน้าบ้าน (lead GQ7 — ยังรอไฟล์) |
| AST-55 | `theme/reset`: ลบแถว theme ของ (บัญชีตัวเอง, เกม) → กลับไปใช้ของชั้นบน / GLOBAL (lead GR5) · ใช้ได้เฉพาะ scope `AGENT` ที่เขียนได้ตาม AST-04 · `scope = GLOBAL` → `422` (ค่ากลางห้ามหาย) · ไม่มีแถวอยู่แล้ว = สำเร็จ |

### Branding (MVP: ชื่อเว็บ + โลโก้)

| ID | Rule |
|---|---|
| AST-41 | branding **ไม่มี GLOBAL และไม่มี scope** — เก็บ 1 แถวต่อบัญชี (`site_brandings` — lead GQ8) · "ยังไม่ตั้ง" = ไม่มีแถว · สิทธิ์เขียน / อ่านตาม AST-02–AST-05 · **ห้ามใช้ query / fallback ชุดเดียวกับ asset / theme** (ไม่มี GLOBAL ให้ query) · เมนูสิทธิ์ `asset` (GQ3) |
| AST-42 | `branding/detail/update` = **แทนทั้งชุด** (ไม่ใช่ section) · MVP รับ `site_name` อย่างเดียว: บังคับ 1–50 ตัวอักษร (นับเป็นตัวอักษร) · ผิด → `422` msg บอก field · **`site_theme` เลื่อน** (หัวข้อ 9 ข้อ 2) · สร้างแถวให้ถ้ายังไม่มี (โลโก้เป็น `""`) |
| AST-43 | โลโก้ (`branding/logo/upload`): ชนิด / ขนาดไฟล์ตาม AST-16 · ไม่ตรวจขนาดภาพ · **เขียนทับ** key เดิมของบัญชี (ไม่เก็บประวัติ) · **ไม่ลบไฟล์เดิม** (lead GQ5) · สร้างแถวให้ถ้ายังไม่มี (`site_name` เป็น `""`) |
| AST-56 | `branding/reset`: ลบแถว branding ของบัญชีตัวเอง → กลับไปใช้ของชั้นบน (lead GR5) · ไม่มีแถวอยู่แล้ว = สำเร็จ · ไม่ลบไฟล์โลโก้ใน storage (AST-29) |

### Bundle, รูปปก และฝั่งผู้เล่น

| ID | Rule |
|---|---|
| AST-45 | bundle ของ (บัญชี, เกม) = `source` + `assets` (map `asset_type` → URL ตาม AST-25) + `theme` (AST-37 + shades) + `site` (branding) · **`assets` กับ `site` แยก key** — ชื่อช่องอะไรก็ไม่ชนกับ branding |
| AST-46 | ไม่มี asset เลย → `assets = {}` ตอบสำเร็จ (ไม่ใช่ error) · อ่าน branding ไม่สำเร็จ → `site` เป็นค่าว่างตาม ACC-32 + log warn — **asset คือส่วนที่จำเป็น branding ล้มต้องไม่ทำให้ทั้ง bundle ล้ม** |
| AST-47 | branding **ไล่ทั้งชุด** ตาม AST-57: บัญชีใกล้สุดในสายที่ "มี branding" = แถวที่มี `site_name` หรือโลโก้ไม่ว่าง → ใช้**ทั้งแถว**ของบัญชีนั้น ห้ามผสม field จากคนละชั้น (ชื่อของลูก + โลโก้ของแม่ = ไม่ใช่ branding ของใคร) · ไม่เจอเลย → ทุก field `""` (ไม่ใช่ error · ไม่ใช่ `null`) |
| AST-48 | bundle ผู้เล่น (`/player/asset/bundle` — GN23): implement ใน module หน้าบ้านหลัง Player Auth · ต้องเรียก **service เดียวกับ preview หลังบ้าน** (ห้ามเขียน logic รวมซ้ำ) · เว็บเป้าหมายมาจากผู้สร้างตรงของ Member ตาม AST-57 |
| AST-51 | `game_cover` เป็น**ชื่อช่องสงวน** ต้องมีใน catalog ของทุกเกม (seed) · `games/list` อ่านปกจากช่อง `game_cover` ของ **GLOBAL เท่านั้น** ผ่าน service ตัวเดียวกับ `used/list` (ห้ามเขียน query คู่ขนาน) · เกมที่ยังไม่มีปก / อ่านปกไม่สำเร็จ → `game_cover = ""` (ACC-32) + log warn · **ห้ามทำให้ทั้งเส้นล้มเพราะเกมเดียว** · เส้นรายชื่อเกมของ lobby ผู้เล่นใช้ service เดียวกัน (ตัดสินพร้อม Player Auth — GQ9) |
| AST-52 | ฝั่งผู้เล่น (Member) **อ่านอย่างเดียว** ไม่มีเส้นเขียนใดๆ · บัญชีเป้าหมายมาจาก token ของผู้เล่นเท่านั้น · **ไม่รับ `agent_id` จาก path / query / body** — ขอดูของบัญชีอื่นไม่ได้ทุกกรณี |
| AST-53 | `*/detail/get` ของ theme และ branding ตอบ**ค่าที่ไล่สายแล้ว** + `source` (`self` / `upline` / `global`) กฎเดียวกับ bundle (lead GR4: "ใช้กฎเดียวกันทั้ง BO preview และผู้เล่น") · หน้าตั้งค่ารู้ว่าเป็นของตัวเองหรือไม่จาก `source` · ปุ่ม reset แสดงเมื่อ `source = self` |

### Storage และประวัติ

| ID | Rule |
|---|---|
| AST-49 | DB เก็บ **storage key** เท่านั้น · URL = `S3_CDN_BASE_URL` (ตัด `/` ท้าย) + `/` + key (ตัด `/` หน้า) · key ใหม่ทุกครั้งที่อัป (ไม่เขียนทับ object เดิม) ตามรูปแบบในหัวข้อ 6 · credential ใช้ IRSA / default chain ห้ามเก็บ key ใน env |
| AST-50 | ทุกตารางเก็บ `created_by` / `updated_by` เป็น username จริงของผู้กด (ไม่ใช่ `"system"` — ยกเว้นแถว seed) + เวลา · **change log เลื่อน** (lead GQ5 — ระหว่างนี้ใช้ `created_by` / `updated_by` + คลังรูป) |

### ภายหลัง (เลื่อนตาม lead GS1 — ยังไม่ implement รอบนี้ · เก็บ rule ไว้ไม่ให้หาย)

| ID | Rule (ร่าง) |
|---|---|
| AST-44 | **banner** (GN18–21): สร้างใหม่ 1 แถวต่อครั้ง · รูปตาม AST-16 · `link_url` = `""` หรือ URL `https://` ที่มี host ยาวไม่เกิน 500 (แบบ ACC-06) · `status` `ACTIVE` (ค่าเริ่มต้น) / `INACTIVE` · ลบ = soft delete · ผู้มีสิทธิ์เขียนบัญชีนั้นลบ / เปลี่ยนสถานะได้ ไม่ผูกกับคนอัป · ไม่พบ / ของบัญชีอื่น / ลบแล้ว → ข้อความเดียวกัน (code ใหม่ตอนทำจริง) · รายชื่อแบ่งหน้า (AST-27) · เมนู `asset` |
| AST-58 | **favicon** (GN15): `ico` หรือ `png` ≤ 5 MB ตรวจจากเนื้อไฟล์ · ไม่ตรวจขนาดภาพ · เขียนทับ · เพิ่มคอลัมน์ `favicon_key` ใน `site_brandings` ด้วย migration ใหม่ · อยู่ในชุด branding ที่ไล่ทั้งชุด (AST-47) |
| AST-59 | **site-themes** (GN17) + `site_theme` ใน `branding/detail/update`: catalog `default` · `dark` · `light` · `neon` · `gold` (ตาราง `site_themes` + seed) · `site_theme` = `code` ที่ `is_active` หรือ `""` · เพิ่มคอลัมน์ `site_theme` ใน `site_brandings` ด้วย migration ใหม่ |
| AST-60 | **change log** แบบ MGMT-60 (ใคร / ทำอะไร / ค่าเก่า → ใหม่ / ip / request_id) ใน tx เดียวกับการเปลี่ยน |

## 4. สิ่งที่พบในโค้ดเก่า และการตัดสินใจ

extract จากโค้ดเท่านั้น (ไม่อ่าน doc / comment อธิบาย business / test ของโปรเจกต์เก่า) · "lead" = การตัดสินใน [docs/reviews/game-asset.md](../reviews/game-asset.md)

| # | โค้ดเก่าทำอะไร | ถูก/ผิด | การตัดสินใจ |
|---|---|---|---|
| 1 | สร้างทุกตารางด้วย GORM AutoMigrate | ผิดกฎข้อ 20 | goose migration (หัวข้อ 6) |
| 2 | unique `(agent_id, game_type_id, asset_type_id)` ของ used แต่**ไม่มี partial index ของ GLOBAL** — DB ไม่กันแถว GLOBAL ซ้ำ | ผิด | partial unique 2 ตัว (AST-22) |
| 3 | `DROP INDEX idx_asset_used_slot` ทุกครั้งที่ boot แล้วให้ AutoMigrate สร้างใหม่ | ผิด | ไม่มี — index อยู่ใน migration |
| 4 | catalog (`minigame_asset_types`) และตารางเกมไม่มี unique ที่ชื่อ | ผิด | unique `(game_code, asset_type)` |
| 5 | ช่องที่ไม่มีใน catalog ถูกปฏิเสธ **แต่ถ้า lookup error จะข้ามไป และ repository สร้างช่องใหม่ให้เอง** | ผิด (ต้นเหตุช่องสะกดผิดซ้ำใน DB) | fail-closed (AST-11, AST-18) · repository ไม่มีการสร้างช่อง |
| 6 | แปลงชื่อเกม / ช่อง (ตัวเล็ก, ช่องว่าง → `_`, ตัดอักขระอื่น, ว่าง → `unknown`) ก่อนค้น | เสี่ยง | เทียบตรงทุกตัวอักษร (AST-14) |
| 7 | ช่องเลิกใช้ปฏิเสธก่อนตรวจขนาด · ขนาดภาพ fail-open 2 กรณี · ตรวจก่อนขึ้น S3 · ≤ 10 MB | ถูก (ขนาดไฟล์เปลี่ยน) | คงไว้ (AST-13, AST-15, AST-17) · ไฟล์ ≤ 5 MB (lead GR3 → AST-16) |
| 8 | ชนิดไฟล์เชื่อ `Content-Type` ของ client · decoder รองรับ png/jpeg/gif/webp | เสี่ยง | ตรวจจากเนื้อไฟล์ · ห้าม SVG (AST-16) |
| 9 | ตอบ error ด้วย `err.Error()` ดิบ (รวม `500`) · ข้อความ error ของขนาดภาพ / ช่องเลิกใช้ไม่ถึง client | ผิด | `apperr` + msg ที่บอกขนาดที่รับ (AST-17) |
| 10 | `update-used` ไม่เช็คว่า config เป็นเกมเดียวกับที่ส่ง (เอา `game_type_id` จาก request แต่ `asset_type_id` จาก config) | ผิด | เช็คเกม (AST-23 ข้อ ข) |
| 11 | upsert used แบบ find แล้ว update / insert ไม่มี lock — GLOBAL แข่งกันได้แถวซ้ำ | ผิด | `ON CONFLICT` บน partial index (AST-22) |
| 12 | ไม่พบ / ผิด scope / ของเว็บอื่น / ถูกลบ ตอบ `"asset_config_id N not found"` เหมือนกัน | ถูก | คงไว้ (`405402`) |
| 13 | ลบ GLOBAL: hard delete เฉพาะคนอัป + `409` ถ้ายังใช้ + ลบไฟล์หลังลบแถว · ลบ AGENT: soft delete · `409` มี 2 ข้อความ | — | **lead GR2: soft delete ทั้งคู่ · ไม่ลบไฟล์ใน request · ไม่ผูกคนอัป · `405403` ข้อความเดียว** (AST-28–32) |
| 14 | GLOBAL เขียนได้ด้วย `requireSystemRole` = role SUPERADMIN · **token ของ sub มี role ของเจ้าของ → sub ของ SUPERADMIN เขียน GLOBAL ได้โดยไม่ต้องมีสิทธิ์ asset** | ผิด | sub ต้องมี `asset` = `edit` (AST-03, AST-06) |
| 15 | เป้าหมาย AGENT ผ่านด่าน "ตัวเองหรือสายล่าง" แต่ capability `asset.manage` เป็น SELF → ในทางปฏิบัติแก้ได้แค่ของตัวเอง · CSM ได้ scope `SUBTREE_LOCKED` ตั้งแทนสายล่าง (สายล่างของ CSM ได้ NONE) | — | **lead GR4: ทุกบัญชีหลักตั้งของตัวเอง · ชั้นบนดูได้ แก้ไม่ได้ · CSM ตั้งครั้งเดียว Share Master ได้ตามผ่านการสืบทอด** (AST-02, AST-04, AST-57) แทนการ "ตั้งแทน" |
| 16 | "ถือ member ได้" (`requireCanHoldMembers`) จำกัดประเภทบัญชีที่ตั้งได้ | — | ตัด — ไม่จำกัดตามนี้ (AST-02 · ตัด `405302`) |
| 17 | อ่าน GLOBAL (assets / library / theme) เปิดให้ทุกคนที่ login รวม sub ที่ไม่มีสิทธิ์ asset | ผิด | ต้องมี `asset` ≥ `view` (AST-05, AST-06) |
| 18 | เลือกเป้าหมายด้วย `?agent_id=` (ไม่ส่ง = GLOBAL · `me` = ตัวเอง · อ่านไม่ออกหรือ `0` → `400`) · เส้นลบเป็น `DELETE /assets/:config_id` | ถูกบางส่วน | ทุกเส้น `POST` · `scope` บังคับ · ไม่มี `me` · multipart ใช้ form field (AST-08 · lead GN, GQ4) |
| 19 | merge used ทับ "ไฟล์ล่าสุด" · ช่องที่ไม่มี used ใช้ไฟล์ล่าสุดทันที · fallback ทีละช่อง agent → GLOBAL (ไม่ไล่สาย) · `source` agent/global/mixed | — | **lead GR1: ใช้งานจริง = `used` เท่านั้น · ตัดกฎไฟล์ล่าสุด · `use_now`** (AST-21, AST-24) · **GR4: ไล่สายทีละช่อง** (AST-25, AST-57) |
| 20 | ไม่มี keep-N / cron / purge ใดๆ · soft delete อยู่ถาวร | ถูก | คงไว้ (AST-20) |
| 21 | theme: fallback ทั้งชุด, whitelist 5 ฟอนต์, regex `#RRGGBB`, `stats_bar_color` เป็นชื่อ, update ครบ 5 ค่า · ไม่ normalize ตัวพิมพ์ของสี (คอลัมน์ `varchar(9)`) · ค่ากลางอยู่ในตารางเกม `minigame_types` | ถูกบางส่วน | คงกฎไว้ · เก็บสีตัวเล็ก (AST-35) · ค่ากลางเป็นแถว GLOBAL ใน `game_themes` · ไล่สายทั้งชุด (AST-37) · เพิ่ม `theme/reset` (AST-55) |
| 22 | shades 19 ระดับ ฐาน `900` บน RGB หาร 10 · `math.Round` (= round-half-up สำหรับค่าบวก) · clamp · hex ตัวเล็ก · สีไม่ถูกต้องได้ shades `nil` เงียบๆ | ถูก | คงไว้ (AST-39) · สีไม่ถูกต้องเข้า DB ไม่ได้ (AST-35) |
| 23 | `allowed_sizes` เป็น text เก็บ JSON (`""` = ไม่มีสเปก, JSON พังก็ถือว่าไม่มีสเปก) | เสี่ยง | `JSONB NOT NULL DEFAULT '[]'` |
| 24 | seed catalog รันเมื่อ `IS_SEED=true` · **retire ทุกช่องที่ไม่อยู่ใน catalog** · มีขนาดเฉพาะ `scratch_card` | — | seed ใน migration (หัวข้อ 6) · แก้ภายหลังด้วย migration ใหม่ |
| 25 | ตาราง `minigame_asset_groups` สร้างไว้แต่ไม่มี service ใช้ | ตาย | ไม่สร้าง |
| 26 | used เก็บ `asset_path` ซ้ำจาก config | — | ไม่เก็บซ้ำ join `asset_configs` |
| 27 | banner: `created_by` = username ของ**เป้าหมาย** แต่ลบต้อง `created_by` = ผู้เรียก → ชั้นบน / sub ที่อัปลบไม่ได้ · `link_url` ไม่ตรวจ · อยู่ใต้ capability `announcement` | ผิดบางส่วน | **เลื่อน** (lead GS1) · ร่างกฎใน AST-44 · เมนู `asset` (GQ3) |
| 28 | branding เป็นคอลัมน์ใน `user_agents` · โลโก้ / favicon เขียนทับ path ไม่ลบไฟล์เดิม · catalog ธีม `default dark light neon gold` | — | ตาราง 1:1 `site_brandings` (lead GQ8) · ไม่ลบไฟล์เดิม (GQ5) · favicon / site-themes เลื่อน (AST-58, AST-59) |
| 29 | bundle ผู้เล่น: data เป็น map แบน `{asset_type: url, ..., "site": {...}}` + header `X-Asset-Source` · ช่องชื่อ `site` จะชน · branding ล้มได้ค่าว่างเงียบๆ · ไม่มี asset ตอบ code แยก | ผิดบางส่วน | แยก key (AST-45) · `source` อยู่ใน body · ไม่มี asset = `{}` (AST-46) |
| 30 | bundle หลังบ้านไม่มี `site` · ไม่มี asset ตอบ HTTP 404 | ไม่ตรงกับฝั่งผู้เล่น | preview ใช้ service เดียวกับผู้เล่น (AST-48) |
| 31 | cache Redis ของ config / used / theme กลาง (bump version ตอนเขียน) | — | ไม่มี cache รอบนี้ — เพิ่มทีหลังถ้าวัดแล้วจำเป็น |

## 5. Endpoints

ทุกเส้น BO เป็น `POST` (lead GN) ใต้ `/api/v1/bo/pr/asset/` · ระบุบัญชีด้วย `agent_id` ใน body / form · ผ่าน `Authenticated` + `PassedGates()` ·
สิทธิ์เมนูด้วย `RequirePermission(MenuAsset, LevelView / LevelEdit)` · สิทธิ์ต่อเป้าหมาย (AST-03 / 04 / 05) ด้วย `ResolveAssetTarget` + `RequireAssetTarget` ·
ลำดับใน `routes.go`: `Authenticated` → `PassedGates()` → `RequirePermission` → `ResolveAssetTarget` → `RequireAssetTarget` — เขียนบรรทัดเดียวกับ route (กฎข้อ 28) ·
sub ต้องมีสิทธิ์ `asset` (AST-06) · ลำดับเช็คและ error ดูหัวข้อ 7.1 · ชื่อ `config_id` ใช้เหมือนกันทุกเส้น ·
`games/list` · `slots/list` · `used/list` · `theme/detail/get` · `branding/detail/get` · `bundle/detail/get` ตอบ array / object ธรรมดา ·
`files/list` แบ่งหน้าตามกฎข้อ 19 (`response.Page`)

error ร่วมทุกเส้น: `401202`, `401203`, `401301`, `401302`, `401304`, `401306`, `401307`, `401311`, `402303` ·
เส้นที่ระบุเป้าหมายเพิ่ม: `422` (AST-08), `402402`, `405301`

| GN | Path (ต่อท้าย `/api/v1/bo/pr/asset/`) | สิทธิ์ `asset` | ทำอะไร | เทียบของเก่า |
|---|---|---|---|---|
| GN1 | `games/list` | view | รายชื่อเกมที่ตั้งหน้าตาได้ + รูปปก | เปลี่ยน (เดิม `GET /pb/games` public) |
| GN2 | `slots/list` | view | ช่องของเกมหนึ่ง + ขนาดภาพที่รับ | เปลี่ยน (เดิม `GET /pb/games/:game_type/asset-types`) |
| GN3 | `files/upload` | edit | อัปรูปเข้าช่อง (multipart · `use_now`) | เปลี่ยน (เดิม `POST /pr/assets/upload`) |
| GN4 | `files/list` | view | คลังรูปของช่อง (แบ่งหน้า) | เปลี่ยน (เดิม `GET /pr/assets/library`) |
| GN5 | `files/delete` | edit | ลบรูปจากคลัง (soft delete) | เปลี่ยน (เดิม `DELETE /pr/assets/:config_id`) |
| GN6 | `used/list` | view | แต่ละช่องใช้รูปไหนอยู่ + ที่มา | เปลี่ยน (เดิม `GET /pr/assets`) |
| GN7 | `used/update` | edit | เลือกรูปจากคลังมาใช้กับช่อง | เปลี่ยน (เดิม `POST /pr/assets/used`) |
| GN8 | `used/reset` | edit | ยกเลิกรูปที่ตั้งเอง กลับไปใช้ของชั้นบน | **ใหม่** |
| GN9 | `theme/detail/get` | view | ธีมที่ใช้อยู่ + shades + ที่มา | เปลี่ยน (เดิม `GET /pr/games/:game_type/theme`) |
| GN10 | `theme/detail/update` | edit | ตั้งฟอนต์ / สี (แทนทั้งชุด) | เปลี่ยน (เดิม `POST /pr/games/:game_type/theme`) |
| GN11 | `theme/reset` | edit | กลับไปใช้ธีมของชั้นบน | **ใหม่** |
| GN12 | `branding/detail/get` | view | ชื่อเว็บ + โลโก้ที่ใช้อยู่ + ที่มา | เปลี่ยน (เดิม `GET /pr/agents/:agent_id/site-settings`) |
| GN13 | `branding/detail/update` | edit | แก้ชื่อเว็บ (แทนทั้งชุด) | เปลี่ยน |
| GN14 | `branding/logo/upload` | edit | อัปโลโก้ (multipart · เขียนทับ) | เปลี่ยน |
| GN16 | `branding/reset` | edit | กลับไปใช้ branding ของชั้นบน | **ใหม่** |
| GN22 | `bundle/detail/get` | view | preview: asset + theme + branding ของบัญชีหนึ่ง | เปลี่ยน (เดิมไม่มี `site`) |
| GN23 | `/api/v1/player/asset/bundle` | token ผู้เล่น | ผู้เล่นโหลดรูป ธีม และ branding | **contract เท่านั้น** — implement หลัง Player Auth (AST-48) |

**เลื่อน (ไม่ implement รอบนี้):** GN15 `branding/favicon/upload` · GN17 `site-themes/list` · GN18 `banners/list` · GN19 `banners/create` · GN20 `banners/status/update` · GN21 `banners/delete` (ภายหลัง — AST-44, AST-58, AST-59)

**รอ Player Auth:** รายชื่อเกมของ lobby ฝั่งผู้เล่น (lead GR6 / GQ9) — ใช้ service เดียวกับ `games/list` (AST-51)

ตัวอย่างใช้ข้อมูลชุดเดียวกับ `agent_management.md`: `superadmin` · `comp01` (id 10, Company Transfer) → `share01` (id 12, Share B2C) →
`agent01` (id 15) → `mem01` (Member id 501) · `csm01` (id 20, Company Seamless Master) → `sm01` (id 22, Share Master) → `mem02` (Member id 502) ·
sub ของ share01 = `share01@design` · `S3_CDN_BASE_URL = https://cdn.example.com`

### GN1 · POST games/list

Request: `{}` · Response `data`:
```json
[
  { "game_code": "scratch_card", "game_cover": "https://cdn.example.com/games/scratch_card/global/game_cover/1759900000000000000-a1b2c3d4.png" },
  { "game_code": "coin_toss", "game_cover": "" },
  { "game_code": "rock_paper_scissors", "game_cover": "" }
]
```
`game_cover` = ช่อง `game_cover` ของ GLOBAL ผ่าน service เดียวกับ `used/list` (AST-51) · ไม่มี / อ่านไม่ได้ = `""` โดยเกมอื่นยังได้ปกตามปกติ ·
ชื่อเกมที่แสดงบนจอเป็นของหน้าบ้าน

### GN2 · POST slots/list

Request: `{ "game_code": "scratch_card" }` · Response `data`:
```json
[
  { "asset_type": "icon_button", "description": "ปุ่มไอคอน", "allowed_sizes": [ { "width": 32, "height": 32 }, { "width": 40, "height": 40 } ] },
  { "asset_type": "win_highlight", "description": "แสงไฮไลต์ตอนชนะ", "allowed_sizes": [ { "width": 425, "height": 425 } ] }
]
```
`allowed_sizes = []` = ไม่ตรวจขนาด · ไม่รวมช่องเลิกใช้ · Error codes: `422` (ไม่ส่ง `game_code`), `405401`

### GN3 · POST files/upload (multipart/form-data)

| field | ค่า |
|---|---|
| `scope` | `GLOBAL` / `AGENT` |
| `agent_id` | id ของบัญชีเป้าหมาย (เฉพาะ `AGENT`) |
| `game_code` | เช่น `scratch_card` |
| `asset_type` | ชื่อช่องตาม catalog |
| `use_now` | `true` / `false` (ไม่ส่ง = `true`) |
| `file` | ไฟล์รูป ≤ 5 MB |

Response `data`:
```json
{
  "config_id": 9001,
  "scope": "AGENT",
  "agent_id": 12,
  "game_code": "scratch_card",
  "asset_type": "icon_button",
  "url": "https://cdn.example.com/games/scratch_card/agents/12/icon_button/1759900000000000000-a1b2c3d4.png",
  "is_used": true,
  "created_by": "share01@design",
  "created_at": "2026-10-12T10:00:00+07:00"
}
```
เป้าหมาย `GLOBAL` → `agent_id = 0` · Error codes: `422`, `405101`, `405102`, `405103`, `405304`, `405401`, `505801`, `505802`, `500`

### GN4 · POST files/list (คลังรูปของช่อง)

Request:
```json
{ "scope": "AGENT", "agent_id": 12, "game_code": "scratch_card", "asset_type": "icon_button", "page": 1, "limit": 20 }
```
Response `data` (`response.Page`):
```json
{
  "current_page": 1,
  "total_page": 1,
  "total_count": 2,
  "limit": 20,
  "has_next": false,
  "has_prev": false,
  "data": [
    { "config_id": 9001, "url": "https://cdn.example.com/games/scratch_card/agents/12/icon_button/1759900000000000000-a1b2c3d4.png", "source": "self", "is_used": true, "created_by": "share01@design", "created_at": "2026-10-12T10:00:00+07:00" },
    { "config_id": 118, "url": "https://cdn.example.com/games/scratch_card/global/icon_button/1759700000000000000-c9d0e1f2.png", "source": "global", "is_used": false, "created_by": "superadmin", "created_at": "2026-10-01T09:00:00+07:00" }
  ]
}
```
`is_used` = เป็นไฟล์ที่เป้าหมายเลือกใช้ (แถว used ของตัวเอง) · Error codes: `422`, `405401`

### GN5 · POST files/delete

Request: `{ "scope": "AGENT", "agent_id": 12, "config_id": 9001 }` · Response: `{"code":200,"msg":"สำเร็จ"}` (ไม่มี `data`) ·
Error codes: `422`, `405402`, `405403`

### GN6 · POST used/list (ไฟล์ใช้งานจริงต่อช่อง ไล่สายแล้ว)

Request: `{ "scope": "AGENT", "agent_id": 15, "game_code": "scratch_card" }` · Response `data`:
```json
{
  "source": "mixed",
  "items": [
    { "asset_type": "icon_button", "config_id": 9001, "url": "https://cdn.example.com/games/scratch_card/agents/12/icon_button/1759900000000000000-a1b2c3d4.png", "source": "upline", "is_retired": false },
    { "asset_type": "win_highlight", "config_id": 120, "url": "https://cdn.example.com/games/scratch_card/global/win_highlight/1759800000000000000-e5f6a7b8.png", "source": "global", "is_retired": false }
  ]
}
```
1 รายการต่อช่องที่มีไฟล์ตาม AST-25 · ช่องที่ไม่มีไฟล์ในสายและ GLOBAL ไม่อยู่ใน `items` · `source` ระดับบนสุด = ถ้าทุกช่องมาจากที่เดียวคือค่านั้น ไม่เช่นนั้น `mixed`
(ไม่มีรูปเลย = `""`) · เป้าหมาย `GLOBAL` ทุกรายการเป็น `global` · Error codes: `422`, `405401`

### GN7 · POST used/update

Request: `{ "scope": "AGENT", "agent_id": 12, "game_code": "scratch_card", "config_id": 118 }` · Response: `{"code":200,"msg":"สำเร็จ"}` (ไม่มี `data`) ·
Error codes: `422`, `405304`, `405402`

### GN8 · POST used/reset

Request: `{ "scope": "AGENT", "agent_id": 12, "game_code": "scratch_card", "asset_type": "icon_button" }` · Response: `{"code":200,"msg":"สำเร็จ"}` (ไม่มี `data`) ·
Error codes: `422` (รวม `scope = GLOBAL`), `405401`

### GN9 · POST theme/detail/get

Request: `{ "scope": "AGENT", "agent_id": 15, "game_code": "scratch_card" }` · Response `data`:
```json
{
  "source": "upline",
  "font_family": "kanit",
  "main_color_1": "#1e90ff",
  "main_color_2": "#ffffff",
  "stats_bar_color": "main_color_2",
  "stats_bar_color_hex": "#ffffff",
  "stats_bar_font_family": "prompt",
  "main_color_1_shades": { "50": "#e9f4ff", "900": "#1e90ff", "1800": "#030e1a" },
  "main_color_2_shades": { "50": "#ffffff", "900": "#ffffff", "1800": "#1a1a1a" }
}
```
`source` = `self` / `upline` / `global` · shades ในตัวอย่างย่อไว้ — ของจริงมีครบ 19 key · Error codes: `422`, `405401`

### GN10 · POST theme/detail/update

Request:
```json
{
  "scope": "AGENT",
  "agent_id": 12,
  "game_code": "scratch_card",
  "font_family": "kanit",
  "main_color_1": "#1E90FF",
  "main_color_2": "#FFFFFF",
  "stats_bar_color": "main_color_2",
  "stats_bar_font_family": "prompt"
}
```
Response: `{"code":200,"msg":"สำเร็จ"}` (ไม่มี `data`) · Error codes: `422` (ขาดค่า / ฟอนต์ไม่อยู่ในรายการ / สีผิดรูปแบบ — msg บอก field), `405401`

### GN11 · POST theme/reset

Request: `{ "scope": "AGENT", "agent_id": 12, "game_code": "scratch_card" }` · Response: `{"code":200,"msg":"สำเร็จ"}` (ไม่มี `data`) ·
Error codes: `422` (รวม `scope = GLOBAL`), `405401`

### GN12 · POST branding/detail/get

Request: `{ "agent_id": 15 }` · Response `data`:
```json
{ "source": "upline", "site_name": "Share One", "logo": "https://cdn.example.com/sites/12/logo/1759900000000000000-0a1b2c3d.png" }
```
`source` = `self` / `upline` / `""` (ไม่มีใครตั้ง — ทุก field `""`) · ไล่สายแล้ว (AST-47, AST-53) · Error codes: `422`

### GN13 · POST branding/detail/update

Request: `{ "agent_id": 12, "site_name": "Share One" }` · Response: `{"code":200,"msg":"สำเร็จ"}` (ไม่มี `data`) · Error codes: `422`

### GN14 · POST branding/logo/upload (multipart/form-data)

field: `agent_id`, `file` · Response `data`:
```json
{ "url": "https://cdn.example.com/sites/12/logo/1759900000000000000-0a1b2c3d.png" }
```
Error codes: `422`, `405102`, `405103`, `505801`, `505802`

### GN16 · POST branding/reset

Request: `{ "agent_id": 12 }` · Response: `{"code":200,"msg":"สำเร็จ"}` (ไม่มี `data`) · Error codes: `422`

### GN22 · POST bundle/detail/get (preview — ผู้เล่นของบัญชีนี้จะเห็นอะไร)

Request: `{ "agent_id": 15, "game_code": "scratch_card" }` · Response `data`:
```json
{
  "source": "mixed",
  "assets": {
    "icon_button": "https://cdn.example.com/games/scratch_card/agents/12/icon_button/1759900000000000000-a1b2c3d4.png",
    "win_highlight": "https://cdn.example.com/games/scratch_card/global/win_highlight/1759800000000000000-e5f6a7b8.png"
  },
  "theme": {
    "source": "upline",
    "font_family": "kanit",
    "main_color_1": "#1e90ff",
    "main_color_2": "#ffffff",
    "stats_bar_color": "main_color_2",
    "stats_bar_color_hex": "#ffffff",
    "stats_bar_font_family": "prompt",
    "main_color_1_shades": { "50": "#e9f4ff", "900": "#1e90ff", "1800": "#030e1a" },
    "main_color_2_shades": { "50": "#ffffff", "900": "#ffffff", "1800": "#1a1a1a" }
  },
  "site": { "site_name": "Share One", "logo": "https://cdn.example.com/sites/12/logo/1759900000000000000-0a1b2c3d.png" }
}
```
Error codes: `422`, `405401`

### GN23 · POST /api/v1/player/asset/bundle (contract — implement หลัง Player Auth)

Request: `{ "game_code": "scratch_card" }` (เกมเดียวต่อคำขอ — รอ review Player Auth ยืนยัน P1) · `data` รูปแบบเดียวกับ GN22 ·
ใช้ token ของ Member · อ่านอย่างเดียว · บัญชีเป้าหมายมาจากผู้สร้างตรงของ Member เท่านั้น ไม่รับ `agent_id` ใดๆ (AST-52) ·
การเช็คเกมเปิด / ปิด (`status_game` — MGMT-20) เป็นของ module หน้าบ้าน

## 6. Schema

migration ใหม่ (ไม่แก้ migration เดิม · ไม่ ALTER `user_agents` · timestamp ต้องอยู่หลังไฟล์ล่าสุดบน `dev` — lead MA6) · เกมอ้างด้วย
`game_code` string (MGMT-16 — ไม่มีตารางเกม) · เฉพาะ MVP (ตารางของงานที่เลื่อนจะเพิ่มด้วย migration ใหม่)

```sql
CREATE TABLE asset_types (                          -- catalog (AST-11)
    id            BIGSERIAL    PRIMARY KEY,
    game_code     VARCHAR(50)  NOT NULL,
    asset_type    VARCHAR(50)  NOT NULL,
    description   VARCHAR(255) NOT NULL DEFAULT '',
    allowed_sizes JSONB        NOT NULL DEFAULT '[]', -- [{"width":544,"height":851}] · [] = ไม่ตรวจ (AST-12)
    is_retired    BOOLEAN      NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_asset_types_slot UNIQUE (game_code, asset_type),
    CONSTRAINT ck_asset_types_asset_type CHECK (asset_type ~ '^[a-z0-9_]{1,50}$')
);

CREATE TABLE asset_configs (                        -- ทุกเวอร์ชันที่อัป = คลังรูป (AST-20)
    id            BIGSERIAL    PRIMARY KEY,
    scope         VARCHAR(10)  NOT NULL,
    agent_id      BIGINT       REFERENCES user_agents(id), -- NULL = GLOBAL
    asset_type_id BIGINT       NOT NULL REFERENCES asset_types(id),
    storage_key   VARCHAR(500) NOT NULL,
    created_by    VARCHAR(71)  NOT NULL,              -- username จริง
    is_deleted    BOOLEAN      NOT NULL DEFAULT false, -- soft delete ทั้ง 2 scope (AST-28)
    deleted_at    TIMESTAMPTZ,
    deleted_by    VARCHAR(71),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT ck_asset_configs_scope CHECK (
        (scope = 'GLOBAL' AND agent_id IS NULL) OR (scope = 'AGENT' AND agent_id IS NOT NULL)),
    CONSTRAINT ck_asset_configs_deleted_at CHECK (is_deleted = (deleted_at IS NOT NULL))
);
CREATE INDEX idx_asset_configs_slot ON asset_configs(asset_type_id, agent_id, created_at DESC) WHERE is_deleted = false;
CREATE INDEX idx_asset_configs_agent ON asset_configs(agent_id) WHERE agent_id IS NOT NULL;

CREATE TABLE asset_used (                           -- 1 แถวต่อ (เป้าหมาย, ช่อง) (AST-22)
    id              BIGSERIAL   PRIMARY KEY,
    scope           VARCHAR(10) NOT NULL,
    agent_id        BIGINT      REFERENCES user_agents(id),
    asset_type_id   BIGINT      NOT NULL REFERENCES asset_types(id),
    asset_config_id BIGINT      NOT NULL REFERENCES asset_configs(id) ON DELETE RESTRICT,
    created_by      VARCHAR(71) NOT NULL,
    updated_by      VARCHAR(71) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ck_asset_used_scope CHECK (
        (scope = 'GLOBAL' AND agent_id IS NULL) OR (scope = 'AGENT' AND agent_id IS NOT NULL))
);
-- Postgres มอง NULL ไม่เท่ากัน → unique ธรรมดากัน GLOBAL ซ้ำไม่ได้ ต้องแยก 2 ตัว (ตัวแรกรองรับ agent_id = ANY(สาย) ด้วย)
CREATE UNIQUE INDEX uq_asset_used_agent_slot  ON asset_used(agent_id, asset_type_id) WHERE agent_id IS NOT NULL;
CREATE UNIQUE INDEX uq_asset_used_global_slot ON asset_used(asset_type_id)           WHERE agent_id IS NULL;
CREATE INDEX idx_asset_used_config ON asset_used(asset_config_id);               -- นับผู้ใช้ตอนลบ (AST-28, AST-32)

CREATE TABLE game_themes (                          -- GLOBAL = ค่ากลาง · AGENT = override (AST-37)
    id                    BIGSERIAL   PRIMARY KEY,
    scope                 VARCHAR(10) NOT NULL,
    agent_id              BIGINT      REFERENCES user_agents(id),
    game_code             VARCHAR(50) NOT NULL,
    font_family           VARCHAR(20) NOT NULL,       -- whitelist อยู่ใน app/core (AST-34)
    main_color_1          CHAR(7)     NOT NULL,
    main_color_2          CHAR(7)     NOT NULL,
    stats_bar_color       VARCHAR(20) NOT NULL,
    stats_bar_font_family VARCHAR(20) NOT NULL,
    created_by            VARCHAR(71) NOT NULL,
    updated_by            VARCHAR(71) NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ck_game_themes_scope CHECK (
        (scope = 'GLOBAL' AND agent_id IS NULL) OR (scope = 'AGENT' AND agent_id IS NOT NULL)),
    CONSTRAINT ck_game_themes_colors CHECK (main_color_1 ~ '^#[0-9a-f]{6}$' AND main_color_2 ~ '^#[0-9a-f]{6}$'),
    CONSTRAINT ck_game_themes_stats_bar_color CHECK (stats_bar_color IN ('main_color_1','main_color_2'))
);
CREATE UNIQUE INDEX uq_game_themes_agent  ON game_themes(agent_id, game_code) WHERE agent_id IS NOT NULL;
CREATE UNIQUE INDEX uq_game_themes_global ON game_themes(game_code)           WHERE agent_id IS NULL;

CREATE TABLE site_brandings (                       -- 1:1 กับบัญชี (AST-41) · ไม่มี scope / GLOBAL / ประวัติ (lead GQ8)
    agent_id    BIGINT       PRIMARY KEY REFERENCES user_agents(id),
    site_name   VARCHAR(50)  NOT NULL DEFAULT '',
    logo_key    VARCHAR(500) NOT NULL DEFAULT '',
    created_by  VARCHAR(71)  NOT NULL,
    updated_by  VARCHAR(71)  NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
```

**เลื่อน (เพิ่มด้วย migration ใหม่ตอนทำ):** `site_brandings.favicon_key` · `site_brandings.site_theme` + ตาราง `site_themes` (seed `default` · `dark` · `light` · `neon` · `gold`) · ตาราง `site_banners` · ตาราง change log

**Seed (อยู่ใน migration เดียวกับที่สร้างตาราง — ไม่มี flag `IS_SEED`):**

| ตาราง | ข้อมูล |
|---|---|
| `asset_types` · `scratch_card` (19 ช่อง มีขนาด) | `background` 1920×1080 · `game_cover` 608×1080 · `bg_loading` 449×52 · `frame_loading` 449×52 · `background_main` 608×1080 · `main_scratch_area` 544×851 · `outer_frame` 544×851 · `top_scratch` 544×851 · `main_play` 186×76 · `icon_button` 32×32 หรือ 40×40 · `auto_scratchers` 520×347 · `selection_button` 115×76 · `bet_limit` 322×322 · `bet_history` 572×929 · `bar_bet_history` 540×74 · `how_to_scratchers` 572×929 · `winner` 502×514 · `you_lose` 544×287 · `win_highlight` 425×425 |
| `asset_types` · `coin_toss` (11 ช่อง ไม่มีขนาด) | `background` · `coin_heads` · `coin_tails` · `selection_button` · `outer_frame` · `bet_limit` · `bet_history` · `winner` · `game_cover` · `loading` · `win_highlight` |
| `asset_types` · `rock_paper_scissors` (12 ช่อง ไม่มีขนาด) | `background` · `hand_rock` · `hand_paper` · `hand_scissors` · `selection_button` · `outer_frame` · `bet_limit` · `bet_history` · `winner` · `game_cover` · `loading` · `win_highlight` |
| `game_themes` | แถว GLOBAL ทั้ง 3 เกม: `noto` · `#f7447e` · `#ffffff` · `main_color_1` · `prompt` · `created_by = 'migration'` |

ขนาดและรายชื่อช่องมาจาก seed ของโค้ดเก่า — ส่งทีมเกมยืนยัน · ไม่ได้คำตอบใน ~3 วัน seed ตามร่างนี้ไปก่อน แก้ด้วย migration ใหม่ภายหลัง (lead GQ7)
· **seed ไม่มีรูป** — ก่อนใช้งานต้องอัป GLOBAL (`use_now = true`) ให้ครบทุกช่อง (AST-24)

**Storage key** (AST-49) · `{ts}` = unix nano · `{rand}` = 8 hex จาก `crypto/rand` · `{ext}` จากชนิดที่ตรวจได้ (AST-16):

| ของ | key |
|---|---|
| asset GLOBAL | `games/{game_code}/global/{asset_type}/{ts}-{rand}.{ext}` |
| asset AGENT | `games/{game_code}/agents/{agent_id}/{asset_type}/{ts}-{rand}.{ext}` |
| โลโก้ | `sites/{agent_id}/logo/{ts}-{rand}.{ext}` |

## 7. Test cases

ตัวอย่างใช้ข้อมูลหัวข้อ 5 · `sa@ops` = sub ของ superadmin · chain ของ `agent01` = agent01 → share01 → comp01 → (superadmin) → GLOBAL ·
chain ของ `mem01` = chain ของผู้สร้างตรง (agent01) · chain ของ `sm01` = sm01 → csm01 → (superadmin) → GLOBAL

| Rule | Input | Expected |
|---|---|---|
| AST-01 | share01 และ comp01 ตั้ง theme ช่อง / เกมเดียวกันต่างกัน | แต่ละบัญชีเป็นเจ้าของค่าของตัวเอง ไม่กระทบกัน |
| AST-01 | `scope=GLOBAL` เรียก branding | `422` (branding ไม่มี GLOBAL) |
| AST-02 | share01 · agent01 · comp01 · csm01 ตั้ง `scope=AGENT` ของตัวเอง | สำเร็จ |
| AST-02 | sm01 (Share Master) `theme/detail/update` ของตัวเอง | `405301` |
| AST-02 | superadmin `scope=AGENT, agent_id=<id ตัวเอง>` | `405301` |
| AST-03 | superadmin อัป `scope=GLOBAL` | สำเร็จ · `created_by = superadmin` |
| AST-03 | `sa@ops` ที่ `asset = view` อัป `scope=GLOBAL` | `402303` |
| AST-03 | `sa@ops` ที่ `asset = edit` อัป `scope=GLOBAL` | สำเร็จ · `created_by = sa@ops` |
| AST-03 | share01 อัป `scope=GLOBAL` | `405301` |
| AST-04 | share01 อัป / ตั้ง theme ให้ `agent_id=15` (ลูกในสายล่าง) | `405301` · ข้อมูลของ agent01 ไม่เปลี่ยน |
| AST-04 | share01 ตั้งให้ `agent_id=20` (csm01 — นอกสาย) | `402402` |
| AST-04 | `share01@design` ที่ `asset = edit` อัปให้ `agent_id=12` | สำเร็จ · เป็นของ share01 |
| AST-04 | csm01 ตั้ง theme ของตัวเอง | `sm01` และ `sm02` ได้ตาม (`source = upline`) โดยไม่ต้องตั้งต่อคน |
| AST-05 | `share01@design` ที่ `asset = off` เรียก `used/list scope=GLOBAL` | `402303` |
| AST-05 | agent01 เรียก `used/list scope=GLOBAL` | สำเร็จ |
| AST-05 | share01 เรียก `used/list scope=AGENT, agent_id=15` (ดูสายล่าง) | สำเร็จ |
| AST-05 | share01 เรียก `used/list scope=AGENT, agent_id=20` (นอกสาย) | `402402` |
| AST-05 | sm01 เรียก `theme/detail/get` ของตัวเอง | สำเร็จ · ได้ค่าของ csm01 · `source = upline` |
| AST-06 | ADMIN เรียก `games/list` | `402303` |
| AST-06 | share01 ถูก SUSPEND เรียก `used/list` | `401311` |
| AST-06 | `share01@design` ที่ไม่มีสิทธิ์ `asset` เรียกทุกเส้น | `402303` |
| AST-07 | share01 `used/update` / `files/delete` ด้วย `config_id` ของ agent01 | `405402` (ข้อความเดียวกับไม่พบ) |
| AST-08 | ไม่ส่ง `scope` / `scope="global"` / `scope=null` | `422` |
| AST-08 | `scope=AGENT` ไม่ส่ง `agent_id` / `agent_id=0` / `agent_id="abc"` | `422` (ไม่ตกไปเป็น GLOBAL) |
| AST-08 | `scope=GLOBAL, agent_id=12` | `422` |
| AST-08 | multipart `files/upload` ส่ง `agent_id` เป็น query แทน form field | `422` (ไม่ส่ง field) |
| AST-09 | เส้น AGENT-only (branding) ตอนไม่มีผล resolve (bug จำลอง) | ปฏิเสธ ไม่ผ่าน |
| AST-10 | `slots/list` `game_code=plinko` | `405401` |
| AST-11 | อัป `asset_type = "win_highlite"` (สะกดผิด) | `405401` · `asset_types` ไม่มีแถวเพิ่ม |
| AST-12 | migration แก้ขนาดของ `win_highlight` | `slots/list` และการตรวจตอน `files/upload` ใช้ขนาดใหม่พร้อมกัน |
| AST-13 | ช่องถูก retire แล้วอัป / `used/update` | `405304` |
| AST-13 | ช่องถูก retire ระหว่างที่บัญชีใช้อยู่ | `used/list` / `bundle` ยังมีช่องนั้น · `slots/list` ไม่มี |
| AST-14 | `asset_type = "Icon_Button"` | `405401` |
| AST-15 | ช่อง retired + ไฟล์ขนาดผิด | `405304` (ไม่ถึงตรวจขนาด) · ไม่มี object ใน bucket |
| AST-16 | ไฟล์ 5 MB + 1 byte | `405103` |
| AST-16 | ไฟล์ SVG · ไฟล์ PDF ที่ส่ง `Content-Type: image/png` | `405102` |
| AST-16 | ไฟล์ 5 MB พอดี | ไม่โดน 413 ของ `BodyLimit` · สำเร็จ |
| AST-16 | อัปสำเร็จ | object มี `Content-Type` ตามชนิดจริง · `Cache-Control: public, max-age=31536000, immutable` |
| AST-17 | `win_highlight` ไฟล์ 400×400 | `405101` msg มี `425×425` · ไม่มี object ใน bucket |
| AST-17 | `icon_button` ไฟล์ 40×40 | สำเร็จ (ขนาดที่ 2) |
| AST-17 | ช่องของ `coin_toss` (ไม่มีสเปก) ไฟล์ขนาดใดก็ได้ | สำเร็จ |
| AST-17 | webp ที่ decoder อ่านขนาดไม่ได้ | สำเร็จ (fail-open) |
| AST-18 | DB ล่มระหว่างอ่าน catalog | `500` · ไม่มี config · ไม่มีช่องใหม่ |
| AST-19 | storage ไม่พร้อม แล้วอัป | `505801` |
| AST-19 | อัปขึ้น S3 สำเร็จ แต่ insert config ล้ม | `500` · object ถูกลบ |
| AST-20 | อัปช่องเดิม 30 ครั้ง | config 30 แถว ไม่มีแถวถูกลบ |
| AST-21 | อัปโดยไม่ส่ง `use_now` | config 1 แถว + used ของช่องชี้ config นั้น (tx เดียว) |
| AST-21 | อัป `use_now=false` ขณะช่องมี used ชี้ไฟล์เก่า | ใช้งานจริงยังเป็นไฟล์เก่า · `asset_used` ไม่เปลี่ยน · `files/list` ได้ `is_used=false` |
| AST-21 | อัป `use_now=true` แล้วบันทึก used ล้ม | ไม่มี config ค้าง (tx ย้อน) · object ถูกลบ (AST-19) |
| AST-22 | `used/update` ช่องเดิม 2 ครั้งต่างไฟล์ | `asset_used` ของช่องนั้นมี 1 แถว ชี้ไฟล์ล่าสุดที่เลือก |
| AST-22 | `used/update scope=GLOBAL` ช่องเดียวกันพร้อมกัน 2 คำขอ | สำเร็จทั้งคู่ · GLOBAL ของช่องนั้นมี 1 แถว |
| AST-23 | `game_code=coin_toss` + `config_id` ของ `scratch_card` | `405402` |
| AST-23 | `scope=GLOBAL` เลือก config ของ share01 | `405402` |
| AST-23 | share01 เลือก config GLOBAL | สำเร็จ |
| AST-23 | agent01 เลือก config ของ share01 (ชั้นบน) | `405402` |
| AST-23 | เลือก config ที่ถูก soft delete | `405402` |
| AST-24 | ช่อง `winner` ไม่มี used ทั้งในสายและ GLOBAL แต่มี config ที่อัปด้วย `use_now=false` | `used/list` / `bundle` ไม่มีช่อง `winner` (ไม่ใช้ไฟล์ล่าสุด) |
| AST-24 | share01 เลือก config GLOBAL ตัวเก่า (ไม่ใช่ตัวล่าสุด) ให้ช่อง `winner` | `used/list` ได้ตัวที่เลือก · `source = self` |
| AST-25 | share01 ตั้ง used ช่อง A · agent01 ตั้ง used ช่อง B | agent01: A = `upline` · B = `self` · ที่เหลือ `global` · `source = mixed` |
| AST-25 | ทุกช่องมาจาก GLOBAL | `source = global` |
| AST-25 | ไม่มีรูปสักช่อง | `source = ""` · `assets = {}` |
| AST-26 | repository "config ของ share01" ขณะ GLOBAL มีช่องเดียวกัน | คืนเฉพาะแถว `AGENT` ของ share01 (integration test ของ repository) |
| AST-27 | `files/list` ของ share01 ช่อง `icon_button` | ของ share01 + ของ GLOBAL ใหม่ → เก่า · ไม่มีของบัญชีอื่น · ไม่มีแถวที่ลบแล้ว |
| AST-27 | ช่องมี 30 เวอร์ชัน · ไม่ส่ง `page` / `limit` · `limit = 500` | หน้า 1 ได้ 20 แถว `total_count = 30` · `limit` ถูกปรับเป็น 100 |
| AST-28 | ลบ config GLOBAL ที่ไม่มีใครเลือก โดย superadmin ที่ไม่ใช่คนอัป | สำเร็จ · `is_deleted = true` · ไม่หายจากตาราง |
| AST-28 | share01 ลบ config ของตัวเองที่ไม่ได้ใช้ · `share01@design` ลบ config ที่ share01 อัป | สำเร็จทั้งคู่ (ไม่ผูกคนอัป) |
| AST-28 | **ลบ config กับ `used/update` เลือก config เดียวกันพร้อมกัน** | ผลอย่างใดอย่างหนึ่ง: ลบสำเร็จ + `used/update` ได้ `405402` · หรือ `used/update` สำเร็จ + ลบได้ `405403` · ไม่มี `500` · ไม่มี used ชี้ config ที่ถูกลบ |
| AST-29 | ลบสำเร็จ | ไม่มีการเรียก storage · object ยังอยู่ใน bucket |
| AST-30 | boot ขณะ storage ไม่พร้อม | ระบบขึ้นได้ · `files/upload` ได้ `505801` · `files/delete` · `used/list` · `theme/detail/get` สำเร็จ |
| AST-31 | share01 ลบ config GLOBAL (`scope=AGENT`) | `405402` |
| AST-31 | share01 เลือก config GLOBAL ด้วย `used/update` | สำเร็จ |
| AST-32 | ลบ config ที่ used ของตัวเองชี้อยู่ | `405403` |
| AST-32 | ลบ config GLOBAL ที่ share01 เลือกใช้อยู่ | `405403` (ข้อความเดียวกัน) |
| AST-32 | share01 ตั้ง used ช่อง A แล้ว agent01 สืบทอด · share01 ลบ config ช่อง A | `405403` · agent01 ยังเห็นรูปเดิม |
| AST-32 | share01 `used/reset` ช่อง A แล้วลบ config | สำเร็จ · agent01 ได้ค่าของชั้นถัดไป (comp01 → GLOBAL) |
| AST-33 | `theme/detail/update` ไม่ส่ง `stats_bar_font_family` | `422` · ค่าเดิมไม่เปลี่ยน |
| AST-34 | `font_family = "Kanit"` / `"Comic Sans"` / `"noto; color:red"` | `422` |
| AST-35 | `#FFF` / `FF0000` / `#GG0000` / `#ff00001` | `422` |
| AST-35 | `#1E90FF` | สำเร็จ · `theme/detail/get` ได้ `#1e90ff` |
| AST-36 | `stats_bar_color = main_color_2` แล้วแก้ `main_color_2` | `stats_bar_color_hex` เปลี่ยนตาม |
| AST-37 | share01 ไม่เคยตั้ง theme · ไม่มีใครในสายตั้ง | ได้ 5 ค่ากลาง · `source = global` |
| AST-37 | **3 ชั้น:** share01 ตั้ง theme · agent01 ไม่ตั้ง · `bundle` ของ agent01 / ผู้สร้างของ mem01 | ได้ theme ของ share01 ทั้ง 5 ค่า · `source = upline` |
| AST-37 | agent01 ตั้ง theme ของตัวเองด้วย | agent01 / mem01 ได้ของ agent01 ทั้ง 5 ค่า (ไม่ผสมกับ share01) · `source = self` |
| AST-37 | superadmin แก้ theme GLOBAL หลังบัญชีถูกสร้าง | บัญชีที่ไม่มีใครในสายตั้ง ได้ค่าใหม่ทันที |
| AST-38 | ค่าเดียวกันที่ผ่าน / ไม่ผ่านฝั่ง AGENT | ได้ผลเดียวกันฝั่ง GLOBAL |
| AST-39 | shades ของ `#f7447e` | `900 = #f7447e` · ครบ 19 key · ตรงกับ fixture palette ของหน้าบ้านทุกค่า (table test ใน `app/core/asset`) |
| AST-39 | shades ของ `#000000` / `#ffffff` | `50 = #e6e6e6` · `1800 = #000000` / `50 = #ffffff` · `1800 = #1a1a1a` |
| AST-40 | share01 ตั้ง `main_color_1 = #1e90ff` | shades ใน `theme/detail/get` / `bundle` คำนวณจาก `#1e90ff` |
| AST-41 | ไม่มีใครในสายตั้ง branding | `branding/detail/get` ได้ `source = ""` · ทุก field `""` |
| AST-42 | `site_name = ""` / 51 ตัวอักษร | `422` |
| AST-42 | `site_name = "ร้านหนึ่ง"` ส่ง `site_theme` ด้วย | `422` (MVP ไม่รับ field นอกสเปก — หัวข้อ 9 ข้อ 2) |
| AST-43 | อัปโลโก้ 2 ครั้ง | แถวเดียว · key ใหม่ · ไฟล์เดิมยังอยู่ใน bucket |
| AST-43 | ยังไม่มีแถว · อัปโลโก้ก่อนตั้งชื่อ | สร้างแถว `site_name = ""` · `logo_key` ตามไฟล์ |
| AST-45 | ช่อง asset ชื่อ `site` (สมมุติ) | อยู่ใน `assets.site` ไม่ชนกับ `site` ของ branding |
| AST-46 | เกมที่ไม่มีไฟล์เลยทั้งในสายและ GLOBAL | สำเร็จ · `assets = {}` |
| AST-46 | อ่าน `site_brandings` error | สำเร็จ · `site` ทุก field `""` · log warn |
| AST-47 | **3 ชั้น:** agent01 ไม่มี branding · share01 มี | `bundle` ของ agent01 ได้ `site` ทั้งแถวของ share01 |
| AST-47 | agent01 ตั้งแค่โลโก้ · share01 ตั้งครบ | `bundle` ของ agent01 ได้ของ agent01 (`site_name = ""`) — ไม่เอาชื่อของ share01 มาผสม |
| AST-47 | ทั้งสายไม่มีใครตั้ง | `site` ทุก field `""` · สำเร็จ |
| AST-48 | Member `mem01` เรียก bundle ฝั่งผู้เล่น (เมื่อมี module หน้าบ้าน) | `data` เท่ากับ `bundle/detail/get` ของ agent01 ทุกส่วน (BO preview = ผู้เล่น) |
| AST-49 | `S3_CDN_BASE_URL = https://cdn.example.com/` | URL ไม่มี `//` ซ้อน |
| AST-50 | `share01@design` อัป แล้ว share01 แก้ theme | config `created_by = share01@design` · theme `updated_by = share01` |
| AST-51 | `scratch_card` มีปก · `coin_toss` ยังไม่มีปก | `games/list` สำเร็จ · `scratch_card` ได้ URL · `coin_toss` ได้ `""` |
| AST-51 | GLOBAL มีปก 2 ไฟล์ เลือกใช้ตัวเก่าไว้ | `games/list` ได้ตัวที่เลือก (ตรงกับ `used/list scope=GLOBAL`) |
| AST-51 | catalog ของทุกเกม | มีช่อง `game_cover` |
| AST-52 | ผู้เล่นใต้ share01 ส่ง `agent_id=15` ใน body / query | ถูกเมิน · ได้ของบัญชีตัวเองเสมอ |
| AST-53 | agent01 ไม่ตั้ง branding · share01 ตั้งไว้ · agent01 เรียก `branding/detail/get` | ได้ของ share01 · `source = upline` (ค่าเดียวกับ `bundle`) |
| AST-53 | agent01 ตั้งเองแล้วเรียก `theme/detail/get` | `source = self` |
| AST-54 | agent01 `used/reset` ช่อง `icon_button` ที่ตั้งเองไว้ | แถว used ของ agent01 หาย · `used/list` ได้ของ share01 (`upline`) · config ยังอยู่ในคลัง |
| AST-54 | `used/reset` ช่องที่ไม่มีแถว | สำเร็จ |
| AST-54 | `used/reset scope=GLOBAL` · sm01 `used/reset` · share01 `used/reset` ให้ agent01 | `422` · `405301` · `405301` |
| AST-55 | agent01 `theme/reset` | แถว theme ของ agent01 หาย · ได้ theme ของ share01 (`upline`) |
| AST-55 | `theme/reset scope=GLOBAL` | `422` · ค่ากลางไม่หาย |
| AST-56 | agent01 `branding/reset` | แถว branding หาย · bundle ได้ของ share01 · ไฟล์โลโก้ยังอยู่ใน bucket |
| AST-57 | สายลึก 6 ชั้น | อ่านค่าของทั้งสายด้วย query เดียว · pure function เลือกใกล้สุด (table test: ว่าง · ชั้นเดียว · หลายชั้น · GLOBAL เท่านั้น) |
| AST-57 | Member `mem02` (ผู้สร้าง sm01) | chain = sm01 → csm01 → (superadmin) → GLOBAL · ได้ค่าของ csm01 |
| AST-57 | สาย Superadmin ไม่มีแถว `AGENT` | ข้ามไป GLOBAL ไม่ error |
| AST-44 | (เลื่อน) banner | ยังไม่ทดสอบรอบนี้ |
| AST-58 | (เลื่อน) favicon | ยังไม่ทดสอบรอบนี้ |
| AST-59 | (เลื่อน) site-themes / `site_theme` | ยังไม่ทดสอบรอบนี้ |
| AST-60 | (เลื่อน) change log | ยังไม่ทดสอบรอบนี้ |

### 7.1 ลำดับเช็ค (แบบเดียวกับ agent_management 7.1 — lead U3)

เช็คตามลำดับนี้ทุกเส้น · ผิดข้อแรกที่เจอตอบ error นั้นทันที · msg บอก field และค่าที่รับได้

| ลำดับ | ที่ | error |
|---|---|---|
| 1 | `Authenticated` → `PassedGates()` | `401202` · `401203` · `401301` · `401302` · `401304` · `401306` · `401307` · `401311` |
| 2 | `RequirePermission(MenuAsset, view / edit)` | `402303` |
| 3 | อ่าน body / form + `Validate()` (`scope` · `agent_id` · ค่า field · ไม่มี `null`) | `400` · `422` |
| 4 | `ResolveAssetTarget` + `RequireAssetTarget` (AST-03 / AST-04 / AST-05) | `402402` → `405301` |
| 5 | service — `files/upload`: catalog → เลิกใช้ → ขนาดไฟล์ → ชนิดไฟล์ → ขนาดภาพ → storage (AST-15) | `405401` → `405304` → `405103` → `405102` → `405101` → `505801` / `505802` |
| 5 | service — `used/update`: ช่อง → เลิกใช้ → config (AST-23) | `405401` → `405304` → `405402` |
| 5 | service — `files/delete`: config → ยังถูกใช้ (AST-28) | `405402` → `405403` |
| 5 | service — `used/reset` · `theme/reset` · `branding/reset` | `422` (scope GLOBAL) |

## 8. Contract changes (แจ้ง frontend)

- **path / method ใหม่ทั้งหมด** (หัวข้อ 5 · ตาม lead GN) · ทุกเส้น `POST` · ไม่มี `DELETE` · ไม่มี `?agent_id=` และ `me` — ส่ง `scope` + `agent_id` ใน body / form field (multipart)
- จัดกลุ่มใหม่: ไฟล์รูปอยู่ใต้ `files/*` · รูปที่ใช้งานใต้ `used/*` · ธีมใต้ `theme/*` · branding ใต้ `branding/*` · รูปแบบชื่อ `…/detail/get` · `…/detail/update` (lead N1)
- **เส้นใหม่:** `used/reset` · `theme/reset` · `branding/reset` (กลับไปใช้ของชั้นบน)
- **อัปแล้วขึ้นจริงไหม:** ใช้งานจริง = `used` เท่านั้น · `files/upload` ส่ง `use_now` (ไม่ส่ง = `true` = ใช้ทันที) · ไม่มีกฎ "ใช้ไฟล์ล่าสุดอัตโนมัติ" แล้ว
- **ลบ = soft delete ทั้งค่ากลางและของบัญชี** · ลบรูปที่ยังถูกเลือกใช้ไม่ได้ (`405403`) ต้อง `used/update` หรือ `used/reset` ก่อน · ลบได้ไม่ผูกกับคนอัป
- **ไล่สาย:** บัญชีที่ไม่ได้ตั้งได้ค่าของชั้นบนที่ใกล้ที่สุด แล้วค่ากลาง (asset ทีละช่อง · theme / branding ทั้งชุด) · `source` เป็น `self` / `upline` / `global` / `mixed` (เดิม `agent` / `global` / `mixed`)
- ชั้นบนดูของสายล่างได้ แก้ไม่ได้ (`405301`) · Share Master ตั้งเองไม่ได้ ได้ตาม CSM
- `theme/detail/get` · `branding/detail/get` ตอบค่าที่ไล่สายแล้ว + `source` (เดิม branding ไม่ไล่สายในหน้าตั้งค่า)
- ไฟล์ ≤ **5 MB** (เดิม 10 MB) · ห้าม SVG · ชนิดไฟล์ตรวจจากเนื้อไฟล์ · ชื่อช่องต้องตรง catalog ทุกตัวอักษร
- `games/list` · `slots/list` ต้อง login และมีสิทธิ์ `asset` (เดิม public) · `game_type` → `game_code`
- อ่านค่ากลาง (GLOBAL) ต้องมีสิทธิ์ `asset` ≥ `view` (เดิมใครที่ login ก็อ่านได้)
- bundle: `data` เป็น `{ source, assets, theme, site }` (เดิม map แบน + `site` ปน) · ไม่ใช้ header `X-Asset-Source` · ไม่มีไฟล์ = `assets: {}` (เดิม code แยก / 404) · ผู้เล่นเรียก `POST /player/asset/bundle` (contract — หลัง Player Auth)
- สีใน response เป็นตัวพิมพ์เล็กเสมอ
- **เลื่อน:** banner · favicon · site-themes (`site_theme`) — frontend ยังไม่ต้องทำ
- **error code ชุดใหม่ทั้งหมด** (ไม่ใช้ `1004xx`, `2000xx`, `2600xx` ของระบบเก่า) — ดูหัวข้อ 9

## 9. การตัดสินใจ

### ✅ lead ตัดสินแล้ว ([docs/reviews/game-asset.md](../reviews/game-asset.md) · 2026-10-12) → อยู่ที่ rule ไหน

| lead | เรื่อง | ตัดสิน → rule |
|---|---|---|
| GS1 | scope | MVP: catalog + asset + theme + bundle + branding ชื่อ / โลโก้ · เลื่อน banner / favicon / site-themes / change log → หัวข้อ 1 · AST-44, 58–60 |
| GR4 · GQ1 · GQ2 | เจ้าของ + สืบทอด | ไล่สายทั้ง asset / theme / branding · ชั้นบนดู แก้ไม่ได้ · นอกสาย `402402` · CSM ตั้งครั้งเดียว Share Master ได้ตาม · ตัด `405302` → AST-01, 02, 04, 05, 25, 37, 47, 57 |
| GR1 | อัปแล้วขึ้นจริงไหม | ใช้งานจริง = `used` เท่านั้น · `use_now` (ค่าเริ่มต้น `true`) · ตัดกฎไฟล์ล่าสุด → AST-21, 24 |
| GR2 | ลบ | soft delete ทั้ง 2 scope · ไม่ลบไฟล์ S3 ใน request · ตัดกฎคนอัป · ตัด `405303` → AST-28–32 |
| GR3 · GQ11 | ขนาดไฟล์ | ≤ 5 MB · multipart · ไม่แตะ `BodyLimit` · ห้าม SVG · `Content-Type` / `Cache-Control` ของ object → AST-16 |
| GR5 | กลับไปใช้ค่าชั้นบน | reset 3 เส้น → AST-54, 55, 56 |
| GR6 · GQ9 | ผู้เล่น / lobby | มี lobby ของเรา → branding จำเป็น · เส้นรายชื่อเกมของ lobby ตัดสินพร้อม Player Auth · ตอนนี้ทำ service ให้ใช้ซ้ำ → AST-48, 51 |
| GN | ชื่อเส้น | รูปแบบ N1 · ทุกเส้น `POST` · ตาราง GN1–GN23 → หัวข้อ 5 |
| GQ3 | เมนูสิทธิ์ | ใช้เมนู `asset` ทั้งหมด → AST-41 |
| GQ4 | upload ระบุเป้าหมาย | form field → AST-08 |
| GQ5 | ลบโลโก้เก่า / change log | ไม่ลบไฟล์ · change log เลื่อน → AST-43, 50, 60 |
| GQ6 | error module id | `05` · game_rate ใช้ `06` (ไม่ใช่ `04`) · ตัด code ที่ไม่ใช้ → ตาราง error ด้านล่าง |
| GQ7 | ช่อง / ขนาด / palette | ส่งทีมเกมยืนยัน · ไม่ได้คำตอบ ~3 วัน seed ตามร่าง → หัวข้อ 6 · AST-40 |
| GQ8 | ที่เก็บ branding | ตาราง 1:1 `site_brandings` → AST-41 · หัวข้อ 6 |
| GQ10 | theme ฝั่งผู้เล่น | รวมใน bundle → AST-45, 48 |
| GQ12 | ส่วนกลาง | อนุมัติ (ตารางด้านล่าง) · credential ใช้ IRSA / default chain · boot ได้แม้ storage ไม่พร้อม · MinIO ใน test env → AST-30, 49 |
| GQ13 | บั๊กเก่าที่ spec ปิด | รับทราบ → หัวข้อ 4 |

**error code** (module `bb=05` · business error ตอบ HTTP 200 · ตัด `405302`, `405303` และ `405404` (banner เลื่อน) — เลขที่ตัดไม่ใช้ซ้ำ):

| Code | HTTP | ความหมาย |
|---|---|---|
| 405101 | 200 | ขนาดภาพไม่ตรงกับที่ช่องนี้รับ (msg บอกขนาดที่รับ) |
| 405102 | 200 | ชนิดไฟล์ไม่รองรับ (รับ png / jpeg / gif / webp) |
| 405103 | 200 | ไฟล์ใหญ่เกิน 5 MB |
| 405301 | 200 | ไม่มีสิทธิ์แก้หน้าตาของบัญชีนี้ (ค่ากลางที่ไม่ใช่ Superadmin · บัญชีในสายล่าง · Share Master) |
| 405304 | 200 | ช่องนี้เลิกใช้แล้ว |
| 405401 | 200 | ไม่พบเกมหรือช่อง asset |
| 405402 | 200 | ไม่พบ asset (ตอบเหมือนกันทุกกรณี — AST-23, AST-31) |
| 405403 | 200 | asset ยังถูกใช้งานอยู่ (ข้อความเดียว — AST-32) |
| 505801 | 503 | ที่เก็บไฟล์ยังไม่พร้อม |
| 505802 | 502 | อัปโหลดไปที่เก็บไฟล์ไม่สำเร็จ |

validation ทั่วไปใช้ `422` + msg บอก field · สิทธิ์ใช้ code ของ module ② (`402303`, `402402`)

### ✅ ตีความ / ตัดสินเองในเอกสารนี้ (ขอให้ lead รับทราบ)

1. **Share Master ตั้งเองไม่ได้** (schneider 2026-10-12 — ตาม lead H2): เขียน = `405301` · CSM ตั้งของ CSM แล้ว Share Master ทุกคนได้ตามผ่านสาย (ไม่มีการ "ตั้งแทน" ทีละคน — แทน capability `SUBTREE_LOCKED` ของระบบเก่า)
2. **ใช้ `user_agents.cnf` (MGMT-61) ไล่สาย** แทน recursive CTE ที่ lead เขียนไว้ (GR4) — ผลลัพธ์เหมือนกัน (query เดียว ไม่ N+1) แต่มี index (GIN) และทีมใช้อยู่แล้ว
3. **`*/detail/get` ตอบค่าที่ไล่สายแล้ว + `source`** (AST-53) แทนที่จะแยก "อ่านของตัวเอง" กับ "ประกอบ bundle" ตามเอกสารสรุป — ตาม GR4 "ใช้กฎเดียวกัน" และ GN6 ("มาจากตัวเอง ชั้นบน หรือค่ากลาง")
4. **ลบไฟล์ที่มีคนใช้อยู่ = `405403`** รวมกรณีลูกสืบทอด (AST-32) · ต้อง `used/reset` / `used/update` ก่อนลบ
5. **Member ใช้สายของผู้สร้างตรง** (AST-57) — เท่ากับ `user_members.cnf` + ผู้สร้าง

### ❓ รอ lead

1. **รายการ GE1-1, GE2-1, GE6-1, GE12-1, GE13-1 (S10) ไม่พบ** — รีวิวอ้างถึงที่ S10 แต่ไม่มีตารางนิยามใน `game-asset.md` (ตารางมีแค่ GN ที่ส่งมาเป็นภาพ) · ขอ lead ส่งตาราง GE เพื่อเขียน spec ตามที่ตัดสินไว้ · ระหว่างนี้ path / response ตาม GN
2. **`site_theme` ใน MVP** — GN13 เขียนว่า "แก้ชื่อเว็บและธีมหน้าเว็บ" แต่ GN17 (`site-themes/list`) เลื่อน และ GS1 บอก branding MVP = ชื่อ / โลโก้ · เสนอ: MVP ตัด `site_theme` ออกจาก GN13 (AST-42 / AST-59) เพื่อไม่ต้องมี catalog ธีมไว้ตรวจค่า
3. **`/player/asset/bundle`** (GN23) — `game_code` ใน body ตามที่ร่างไว้ใช่ไหม (เส้นผู้เล่นรอ review Player Auth — P1) · เส้นรายชื่อเกมของ lobby (GQ9) ตัดสินพร้อม Player Auth
4. **ก่อน go-live** ต้องอัป GLOBAL ให้ครบทุกช่องที่เกมใช้ (seed ไม่มีรูป และไม่มี fallback ไฟล์ล่าสุดแล้ว — AST-24) — ใครเป็นคนอัป / เมื่อไร
5. **รายชื่อช่อง / ขนาด / palette** (GQ7) — รอทีมเกม / หน้าบ้านยืนยัน (ไม่ตอบใน ~3 วัน seed ตามร่าง)

### ส่วนกลางที่ lead อนุมัติแล้ว (GQ12) — ทำตอน implement ใน branch ใหม่หลัง merge `dev` ล่าสุด (S13)

| เรื่อง | ที่ต้องแก้ |
|---|---|
| dependency S3 | `go.mod` เพิ่ม AWS SDK v2 (s3) |
| config | `pkg/configs/config.go` + `.env.example`: `S3_REGION` · `S3_BUCKET` · `S3_CDN_BASE_URL` · `S3_UPLOAD_TIMEOUT` · `S3_ENDPOINT` (MinIO ใน local / test) · credential จาก IRSA / default chain ห้าม key ใน env · ขึ้นได้แม้ storage ไม่พร้อม (AST-30) |
| storage client | package ใหม่ `platform/storage` |
| ขนาด body | **ไม่แตะ** `BodyLimit` (10 MB ทั้ง request ครอบไฟล์ 5 MB ได้) |
| multipart | helper อ่าน multipart + `Validate()` ใน `pkg/utils` |
| middleware | `ResolveAssetTarget` / `RequireAssetTarget` ใน `app/internals/backoffice/middleware` · ใช้ `RequirePermission` ของ module ② |
| test env | MinIO ใน `docker-compose` / `scripts/testenv.ps1` |
| เอกสารกลาง | จอง module id `05` ใน `docs/ERROR_CODES.md` (game_rate = `06`) · เพิ่มแถว `asset` ใน `docs/MIGRATION_STATUS.md` · หลัง spec อนุมัติ: `docs/frontend/asset.md` + แถวใน `docs/frontend/README.md` |
| migration | ไฟล์ใหม่ต้องมี timestamp หลัง `20261011120000_member_management_phone_country_code.sql` (ล่าสุดบน `dev` ตอนตรวจ 2026-10-12) |

**infra ที่ต้องมีต่อ env (dev / uat / prod)** — ดูรายละเอียดใน [docs/reviews/game-asset.md](../reviews/game-asset.md) หัวข้อ 7.3: S3 bucket (Block Public Access · SSE-S3) · CloudFront + OAC + โดเมน CDN · IAM (IRSA) `s3:PutObject` / `GetObject` / `DeleteObject` เฉพาะ `games/*` และ `sites/*` · ConfigMap + ServiceAccount annotation · body limit ที่ ingress / ALB ≥ 6 MB สำหรับเส้น upload
