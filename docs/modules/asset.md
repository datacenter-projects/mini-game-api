# Asset · ธีมเกม · Branding ของเว็บ (`asset`) — Spec

- สถานะ: **DRAFT** — ห้าม implement จนกว่า lead อนุมัติ (ข้อที่รอตัดสินอยู่ในหัวข้อ 9)
- อนุมัติโดย: — (รอ lead) · วันที่ร่าง: 2026-10-08
- ชื่อ module ในโค้ด: `asset` (`controllers/asset`, `dto/asset`, `service/asset`, `core/asset`)
- เมนูสิทธิ์: `asset` (off / view / edit — `agent_management.md` MGMT-50–52)
- ที่มาของ rule: `ASSET_SYSTEM_SPEC.md` (เอกสารสรุประบบเดิมที่ทีมส่งมา 2026-10-08 — **ใช้เป็นข้อเสนอเท่านั้น**) ·
  ตรวจเทียบกับ*โค้ด*เก่าตามขั้น Extract (ผลอยู่ในหัวข้อ 4)
- โค้ดเก่าที่อ้างอิง (extract พฤติกรรมเท่านั้น): `askmelotto_minigame-api/app/services/asset/*.go`,
  `app/internals/backoffice/middleware/asset_target_middleware.go`, `app/internals/backoffice/routes/module_routes.go`,
  `app/repository/postgres/minigame_asset_*_repository.go`, `app/services/agent/*site*_service.go`,
  `database/postgres/seeddata/seed_minigame_asset_type.go`, `platform/s3client/s3.go`
- spec ที่อ้างถึง: [agent_auth.md](agent_auth.md), [agent_auth_phase2.md](agent_auth_phase2.md),
  [agent_management.md](agent_management.md), [account.md](account.md) (APPROVED ทั้งหมด)

> เอกสารนี้คือ source of truth ของ module — โค้ดและ test ต้องตรงกับเอกสารนี้ ถ้าไม่ตรงให้แก้โค้ด
> หรือแก้เอกสารผ่านการอนุมัติใหม่ ห้ามอ้างเอกสารของโปรเจกต์เก่า

## 1. หน้าที่ของ module

จัดการหน้าตาของเกมและหน้าเว็บ แบบ multi-tenant: ทุกเว็บใช้ระบบเดียวกัน แต่ปรับหน้าตาเกมของตัวเองได้

**อยู่ใน scope รอบนี้:**
- **Game Asset** — ไฟล์รูปต่อ "ช่อง" ของแต่ละเกม (ปุ่ม, พื้นหลัง, win highlight ฯลฯ): อัปโหลด, เลือกไฟล์ที่ใช้งาน,
  ดูประวัติไฟล์ (library), ลบ
- **Game Theme** — ฟอนต์ + สีหลัก 2 สี + สี/ฟอนต์ของแถบสถิติ ต่อเกม และชุดเฉดสีที่คำนวณจากสีหลัก
- **Site Branding** — ชื่อเว็บ, โลโก้, favicon, ธีมหน้าเว็บ และ banner ของแต่ละเว็บ
- **Bundle** — รวม asset + theme + branding ของ (เว็บ, เกม) ในคำขอเดียว: หลังบ้านใช้ preview
- ตาราง catalog ของช่อง asset และ seed เริ่มต้น (ผ่าน migration)

**กำหนด contract ไว้ แต่ implement ใน module หน้าบ้าน:** bundle ฝั่งผู้เล่น (`/api/v1/player/...`) — ยังไม่มี login ของ
Member

**ไม่อยู่ใน module นี้:** CDN และการ purge cache · การกวาด object ขยะใน storage (job แยกนอก request) ·
การสร้าง/แก้ catalog ของช่องผ่าน API (ทำด้วย migration เท่านั้น — AST-11) · ตัวเกมเอง (`app/internals/game`)

## 2. คำศัพท์

| คำ | ความหมาย |
|---|---|
| เว็บ | บัญชีฝั่ง agent ที่มีหน้าตาเกมของตัวเองได้ (AST-02) |
| GLOBAL | ค่ากลางของระบบ — เว็บที่ยังไม่ปรับแต่งเห็นค่านี้ |
| AGENT | ของเว็บใดเว็บหนึ่ง (`agent_id` = id ของเว็บ) |
| เป้าหมาย | scope + เว็บ ที่คำขอนั้นทำงานด้วย (AST-08) |
| ช่อง (slot) | หน่วยที่ปรับแต่งได้ 1 หน่วย = (เป้าหมาย, `game_code`, `asset_type`) เช่น "`icon_button` ของ `scratch_card` ของเว็บ share01" |
| catalog | รายการช่องที่มีได้ของแต่ละเกม (`asset_types`) พร้อมขนาดภาพที่รับ |
| config | ไฟล์ 1 เวอร์ชันที่เคยอัปโหลด (`asset_configs`) — เก็บทุกเวอร์ชัน |
| used | config ที่ถูก "เลือกใช้" ของช่อง (`asset_used`) — มีได้ 1 แถวต่อช่อง |
| ใช้งานจริง | config ที่เกม render ของช่องนั้น (AST-24) |
| `source` | ที่มาของค่าที่ตอบ: `agent` (ของเว็บเอง) · `global` (ยืมค่ากลาง) · `mixed` (asset บางช่องของเว็บ บางช่องค่ากลาง) |
| theme | ฟอนต์ + สี 5 ค่าของเกม (AST-33) |
| shades | ชุดเฉด 19 ระดับที่คำนวณจากสีหลัก 1 สี (AST-39) |
| branding | ชื่อเว็บ, โลโก้, favicon, ธีมหน้าเว็บ ของเว็บ |
| storage key | path ของไฟล์ใน object storage (S3) ที่เก็บใน DB — URL สร้างตอนตอบ (AST-49) |

## 3. Business rules

แต่ละข้อมี id เพื่ออ้างใน test และ code (`// rule: AST-03`)

### Scope และสิทธิ์

| ID | Rule |
|---|---|
| AST-01 | asset และ theme มี **2 scope เท่านั้น**: `GLOBAL` · `AGENT` · ความเป็นเจ้าของอยู่ที่**ระดับบัญชีเดี่ยว** ไม่มีระดับ "กลุ่มเว็บ" — เว็บใต้หัวสายเดียวกันปรับของตัวเองอิสระจากกัน · branding และ banner มีแค่ `AGENT` |
| AST-02 | **เว็บ** = บัญชีฝั่ง agent ประเภทที่สร้าง Member ได้ (MGMT-02): `COMPANY_SEAMLESS_1TO1` · `SHARE_B2C` · `SHARE_RESELLER` · `SHARE_MASTER` · `AGENT` · เป้าหมาย `AGENT` ที่เป็นประเภทอื่น → `405302` (❓ หัวข้อ 9 ข้อ 1) |
| AST-03 | **เขียน GLOBAL**: SUPERADMIN (บัญชีหลัก) · sub ของ SUPERADMIN ต้องมี `asset` = `edit` · บัญชีอื่น → `405301` |
| AST-04 | **เขียน AGENT**: เฉพาะเว็บนั้นเอง หรือ sub ของเว็บนั้นที่มี `asset` = `edit` · ชั้นบนแก้ของสายล่างไม่ได้ · ระบุบัญชีที่ไม่ใช่ตัวเอง → `402402` (❓ หัวข้อ 9 ข้อ 1) |
| AST-05 | **อ่าน**: ต้องมี `asset` ≥ `view` ทุกเส้น · อ่าน GLOBAL ได้ทุกบัญชีที่มีสิทธิ์ (ต้องเห็นค่ากลางที่ตัวเอง fallback อยู่) · อ่าน AGENT ได้เฉพาะเว็บตัวเอง (❓ ชั้นบนดูของสายล่าง — หัวข้อ 9 ข้อ 1) |
| AST-06 | ADMIN ไม่มีเมนู `asset` (AUTH-43) → `402303` · บัญชีที่ `status` ไม่ใช่ `ACTIVE` ถูกกันที่ `PassedGates()` (AUTH-54 → `401311` · AUTH-27) · ทุกเส้นไม่ต้องยืนยัน passcode |
| AST-07 | middleware ตอบแค่ "ผู้เรียกทำกับเป้าหมายนี้ได้ไหม" — **ไม่ได้พิสูจน์ว่า resource เป็นของเป้าหมาย** · service ต้องเช็คซ้ำทุกครั้งว่า config / banner ที่ระบุเป็นของเป้าหมาย (เช่น `config.agent_id = เป้าหมาย`) |

### การระบุเป้าหมาย

| ID | Rule |
|---|---|
| AST-08 | เส้นที่เลือก scope ได้ส่ง `scope` = `"GLOBAL"` / `"AGENT"` **บังคับ** · `AGENT` ต้องส่ง `agent_id` > 0 · `GLOBAL` ต้องไม่ส่ง `agent_id` หรือส่ง `0` · เส้น branding / banner ส่ง `agent_id` > 0 อย่างเดียว · ไม่ส่ง / ค่าอื่น / `null` / อ่านไม่ออก → `422` msg บอก field (body JSON อ่านด้วย `utils.ParseBodyNoNull`) · **ไม่มีค่าพิเศษ `me`** (หน้าบ้านใช้ id จาก Profile) · **ไม่มีการตีความค่าว่างหรือ 0 เป็น GLOBAL เงียบๆ** — พิมพ์ผิดต้องไม่กลายเป็นการเขียนทับค่ากลาง |
| AST-09 | middleware `ResolveAssetTarget` เก็บผลเป็น struct ที่บอกชัดว่า resolve แล้วหรือยัง และได้ scope ไหน · guard ที่มีความหมายเฉพาะฝั่งเดียว (เช่น AST-02, AST-04) ต้องอยู่หลังตัวแยกฝั่ง · ถ้าอ่านผล resolve ไม่ได้ → ปฏิเสธ (fail-closed) ห้ามปล่อยผ่าน |

### Catalog

| ID | Rule |
|---|---|
| AST-10 | รหัสเกมใช้ชุดเดียวกับ MGMT-16: `coin_toss` · `rock_paper_scissors` · `scratch_card` (อยู่ใน `app/core`) · module นี้ไม่มีตารางเกม |
| AST-11 | ช่องต้องมาจาก `asset_types` เท่านั้น · catalog เพิ่ม / แก้ขนาด / เลิกใช้ ด้วย migration ใหม่เท่านั้น · **ห้ามสร้างช่องจากชื่อที่ส่งมาตอนอัปโหลด** |
| AST-12 | `allowed_sizes` = รายการ `{width, height}` (ได้หลายขนาดต่อช่อง) · `[]` = ไม่ตรวจขนาด · เส้น `slots` (ให้หน้าบ้านเช็คก่อนอัป) กับเส้น `upload` อ่านจากตารางเดียวกัน ห้ามมีค่าขนาดซ้ำในโค้ด |
| AST-13 | เลิกใช้ช่อง = `is_retired = true` **ห้ามลบแถว** · ช่องที่เลิกใช้ไม่แสดงใน `slots` · อัปโหลด / เลือกใช้ใหม่ไม่ได้ (`405304`) · ไฟล์ที่ใช้งานอยู่แล้วยังอยู่ใน `list` / `bundle` (เกมที่ยังอ้างช่องนั้นต้องไม่พัง) |

### อัปโหลด

| ID | Rule |
|---|---|
| AST-14 | `game_code` และ `asset_type` ต้องตรงกับ catalog **ทุกตัวอักษร** (ไม่แปลงตัวพิมพ์ ไม่แทนช่องว่าง) · ไม่พบ → `405401` |
| AST-15 | ลำดับตรวจ: สิทธิ์ → catalog (AST-14) → เลิกใช้ (AST-13) → ชนิด / ขนาดไฟล์ (AST-16) → ขนาดภาพ (AST-17) → **แล้วค่อยอัปขึ้น storage** · ไม่ผ่านข้อใด = ไม่มีไฟล์ขึ้น bucket |
| AST-16 | ไฟล์ ≤ **10 MB** (`405103` · ชนกับ `BodyLimit` 10 MB ของทั้ง request — ❓ หัวข้อ 9 ตาราง "ต้องขออนุมัติ") · ชนิด `png` · `jpeg` · `gif` · `webp` ตรวจจาก**เนื้อไฟล์** (magic bytes) ไม่เชื่อ `Content-Type` ที่ client ส่ง (`405102`) · นามสกุลของ storage key มาจากชนิดที่ตรวจได้ |
| AST-17 | ขนาดภาพ: ช่องมีสเปก + อ่านขนาดได้ + ไม่ตรงขนาดใดเลย → `405101` msg บอกขนาดที่รับ · **ผ่านโดยตั้งใจ (fail-open) 2 กรณี**: ช่องไม่มีสเปก · อ่านขนาดจากไฟล์ไม่ได้ — ขนาดภาพเป็นเรื่องหน้าตา ไม่ใช่ความปลอดภัย |
| AST-18 | อ่าน catalog จาก DB ไม่สำเร็จ → `500` ปฏิเสธ (**fail-closed** — ห้ามข้ามไปบันทึกต่อ) |
| AST-19 | storage ไม่พร้อม → `505801` · อัปขึ้น storage ไม่สำเร็จ / timeout → `505802` (ไม่มีแถว DB) · อัปสำเร็จแต่บันทึก DB ไม่สำเร็จ → ลบ object ทิ้ง (ลบไม่ได้ log error พร้อม key) แล้วตอบ `500` |
| AST-20 | อัปโหลดทุกครั้งสร้าง config ใหม่ 1 แถว (เก็บทุกเวอร์ชัน) · `created_by` = username จริงของผู้กด (sub = username ของ sub) · **ไม่มีโควตา ไม่มีการตัดเวอร์ชันเก่าอัตโนมัติในรูปแบบใด** — ลบได้จากเส้น `delete` เท่านั้น |
| AST-21 | อัปโหลดไม่แตะ `asset_used` · ผลต่อช่องที่ใช้งานจริงเป็นไปตาม AST-24 |

### ไฟล์ที่ใช้งานจริง (used)

| ID | Rule |
|---|---|
| AST-22 | `asset_used` มี 1 แถวต่อช่อง · เลือกไฟล์ใหม่ = **แก้แถวเดิม** (upsert `ON CONFLICT`) ไม่ใช่เพิ่มแถว · unique ต้องเป็น (เป้าหมาย, ช่อง) **ห้ามรวม `asset_config_id`** · GLOBAL ใช้ partial unique index แยก (Postgres มอง `NULL` ไม่เท่ากัน) |
| AST-23 | `update-used`: config ต้อง (ก) มีอยู่และไม่ถูกลบ (ข) เป็นเกมเดียวกับ `game_code` ที่ส่ง (ค) เป้าหมาย `GLOBAL` เลือกได้เฉพาะ config `GLOBAL` · เป้าหมาย `AGENT` เลือกได้ config `GLOBAL` หรือของเว็บนั้นเอง · ช่องเลิกใช้ → `405304` · ผิด (ก)–(ค) ข้อใดก็ตอบ **`405402` ข้อความเดียวกันหมด** (กันไล่เดา id) · ช่องของ used = ช่องของ config |
| AST-24 | ใช้งานจริงของช่องในขอบเขตเดียว (เว็บ หรือ GLOBAL): มี used → config ของ used · ไม่มี used → config ล่าสุดที่ยังไม่ถูกลบของขอบเขตนั้น (อัปแล้วมีผลทันทีถ้ายังไม่เคยเลือก) · การรวม used กับล่าสุดต้อง**เพิ่ม**ช่องที่มีแค่ used (เว็บเลือก config GLOBAL ไว้ แต่ไม่เคยอัปของตัวเอง) — ไม่ใช่แค่ทับช่องที่มีอยู่ ไม่งั้นหน้าตั้งค่าเห็นรูปหนึ่ง เกม render อีกรูป |
| AST-25 | **fallback ทีละช่อง** (เป้าหมาย AGENT): ช่องที่เว็บมีตาม AST-24 → ใช้ของเว็บ (`source = agent` รวมกรณีเว็บเลือก config GLOBAL) · ไม่มี → ใช้ GLOBAL ของช่องนั้นตาม AST-24 (`source = global`) · ทั้งชุด: `agent` / `global` / `mixed` · ห้าม fallback ทั้งชุดจาก "เว็บมีสักช่องไหม" |
| AST-26 | repository ที่ตอบ "ของเว็บนี้มีอะไร" กรองเข้ม `scope = AGENT` + `agent_id` ตรง ไม่ปน GLOBAL · การรวม / fallback ทำใน service (ตรรกะรวมเป็น pure function ใน `app/core/asset`) |
| AST-27 | `library` ของช่อง: config ที่ยังไม่ถูกลบของเป้าหมาย (+ ของ GLOBAL เมื่อเป้าหมายเป็น AGENT) เรียงใหม่ → เก่า · ทุกแถวมี `source`, `created_by`, `created_at`, `is_used` · แบ่งหน้า: `page` / `limit` ใน body (`utils.NewPage` — ค่าเริ่มต้น 20 สูงสุด 100) ตอบด้วย `response.Page` (กฎข้อ 19 — config ไม่มีการตัดเวอร์ชันเก่า จึงยาวได้ไม่จำกัด) |

### ลบ — อสมมาตรโดยตั้งใจ (ห้ามทำให้เหมือนกัน)

| | GLOBAL | AGENT |
|---|---|---|
| วิธีลบ | **ลบจริง**: ลบแถว DB ก่อน แล้วค่อยลบไฟล์ใน storage | **soft delete**: `is_deleted = true` ไม่แตะ storage |
| ใครลบได้ | ผู้มีสิทธิ์ AST-03 **ที่เป็นคนอัปเอง** (`created_by` = ผู้เรียก) | ผู้มีสิทธิ์ AST-04 ใครก็ได้ ไม่ผูกกับคนอัป |
| ยังถูกเลือกใช้อยู่ | `405403` | `405403` |

| ID | Rule |
|---|---|
| AST-28 | GLOBAL: config ไม่ใช่ GLOBAL / ไม่พบ → `405402` · ไม่ใช่คนอัป → `405303` · ลำดับตายตัว: ตรวจเจ้าของ → tx: `SELECT ... FOR UPDATE` แถว config → นับ used ที่ชี้มา > 0 → `405403` → ลบแถว → commit → ลบไฟล์ · ชน FK จากคำขอ `update-used` ที่แทรกพร้อมกัน → `405403` (ไม่ใช่ `500`) · **ลบแถวก่อนไฟล์เสมอ** (ลบแถวไม่ได้ ไฟล์ยังอยู่ retry ได้ · ลบไฟล์ก่อนแล้วลบแถวไม่ได้ = แถวชี้ไฟล์ที่ไม่มี กู้ไม่ได้) |
| AST-29 | ลบไฟล์ไม่สำเร็จ**หลัง**ลบแถวแล้ว → ตอบสำเร็จ + log error พร้อม storage key (ไฟล์ค้างให้ job แยกกวาด · ถ้าตอบ error ผู้เรียกจะ retry แล้วเจอไม่พบ) |
| AST-30 | storage ไม่พร้อม → ลบ GLOBAL ตอบ `505801` และไม่ลบอะไรเลย · ลบ AGENT และเส้นอ่านทั้งหมดทำงานปกติ |
| AST-31 | AGENT: config ของ GLOBAL / ของเว็บอื่น / ไม่พบ / ถูกลบแล้ว → `405402` ข้อความเดียวกัน · เว็บ**เลือกใช้** GLOBAL ได้ (AST-23) แต่**ลบ** GLOBAL ไม่ได้ — สองสิทธิ์นี้แยกกัน |
| AST-32 | `405403` เลือกข้อความตามสิ่งที่ผู้เรียกทำได้จริง: ผู้ถือเป็นเว็บอื่น (config GLOBAL ที่เว็บเลือกใช้) → "asset นี้ถูก N เว็บเลือกใช้อยู่" · ผู้ถือเป็นช่องของเป้าหมายเอง → "เลือกไฟล์อื่นให้ช่องนี้ก่อน" · ลำดับที่ถูก = **อัปของใหม่ + เลือกใช้ก่อน แล้วค่อยลบของเก่า** |

### ธีมเกม

| ID | Rule |
|---|---|
| AST-33 | theme มี 5 ค่าเสมอ: `font_family` · `main_color_1` · `main_color_2` · `stats_bar_color` · `stats_bar_font_family` · `theme/update` ต้องส่ง**ครบ 5 ค่า** = แทนทั้งชุด · ขาดค่าใด → `422` (ไม่มี partial update) |
| AST-34 | ฟอนต์ (ทั้ง 2 field) เป็น whitelist ตัวพิมพ์ตรงตัว: `noto` · `prompt` · `kanit` · `mitr` · `sarabun` · ไม่รับข้อความอิสระ (กัน CSS injection และฟอนต์ที่หน้าบ้านไม่ได้ preload) · ค่าอื่น → `422` |
| AST-35 | สีหลักตรง `^#[0-9A-Fa-f]{6}$` เท่านั้น (ไม่รับ `#RGB`) · เก็บและตอบเป็นตัวพิมพ์เล็ก · ไม่ตรง → `422` |
| AST-36 | `stats_bar_color` เก็บเป็น**ชื่อ** `"main_color_1"` / `"main_color_2"` ไม่ใช่ hex (เปลี่ยนสีหลักแล้วแถบสถิติตามไปเอง) · response เพิ่ม `stats_bar_color_hex` ที่ resolve แล้ว |
| AST-37 | **fallback ทั้งชุด**: เว็บมีแถว theme ของเกมนั้น → ใช้ทั้ง 5 ค่าของเว็บ (`source = agent`) · ไม่มี → ใช้ทั้ง 5 ค่ากลาง (`source = global`) · **ห้ามผสมทีละค่า** (ฟอนต์กับสีออกแบบมาคู่กัน) · ค่ากลางเป็นแถว GLOBAL ใน `game_themes` (seed: `noto` · `#f7447e` · `#ffffff` · `main_color_1` · `prompt`) · ค่าของเว็บเป็นแถว override แยก **ห้าม copy ค่ากลางเข้าไปตอนสร้างบัญชี** (แก้ค่ากลางต้องมีผลกับทุกเว็บที่ไม่เคยตั้ง) |
| AST-38 | validation ชุดเดียวกัน (AST-33–36) ทั้งเขียน GLOBAL และ AGENT — อยู่ใน `app/core/asset` ที่เดียว |
| AST-39 | **shades** ต่อสีหลัก 1 สี = 19 ระดับ ชื่อ `50` `100` `200` … `800` `900` `1000` … `1800` (index 0–18): |
| | • `900` (index 9) = สีหลักตรงตัว |
| | • index < 9: ผสมขาว `t = (9 − i) / 10` → `c' = c + (255 − c) × t` (`50` ผสมขาว 90%) |
| | • index > 9: ผสมดำ `k = (10 − (i − 9)) / 10` → `c' = c × k` (`1800` เหลือ 10%) |
| | • หาร **10 ไม่ใช่ 9** (ปลายทั้งสองข้างไม่ถึงขาว / ดำสนิท) · คำนวณบน **RGB ไม่ใช่ HSL** · ปัดแบบ round-half-up แล้ว clamp 0–255 · hex ตัวพิมพ์เล็ก |
| AST-40 | shades คำนวณ**สดทุกครั้ง**จากสีที่ resolve แล้ว (หลัง AST-37) ไม่เก็บลง DB · ห้ามคำนวณจากสีกลางเมื่อเว็บตั้งสีเอง · ต้อง pin ค่าด้วย test fixture ที่เทียบกับไฟล์ palette จริงของหน้าบ้าน (❓ หัวข้อ 9 ข้อ 7) |

### Branding ของเว็บ

| ID | Rule |
|---|---|
| AST-41 | branding เป็นของเว็บ (AST-02) เท่านั้น ไม่มี GLOBAL · สิทธิ์อ่าน / เขียนตาม AST-04 / AST-05 (❓ เมนู — หัวข้อ 9 ข้อ 3) |
| AST-42 | `branding/update` = แทนชุด `site_name` + `site_theme` · `site_name` บังคับ 1–50 ตัวอักษร · `site_theme` = `code` ที่ `is_active` ใน `site_themes` หรือ `""` (= ไม่ใช้ธีม) · ผิด → `422` msg บอก field |
| AST-43 | โลโก้: ชนิด / ขนาดไฟล์ตาม AST-16 · favicon: `ico` หรือ `png` ≤ 10 MB ตรวจจากเนื้อไฟล์ · ไม่ตรวจขนาดภาพ · **เขียนทับ** (ไม่เก็บประวัติ) · ไฟล์เดิมใน storage ❓ หัวข้อ 9 ข้อ 5 |
| AST-44 | banner: สร้างใหม่ 1 แถวต่อครั้ง · รูปตาม AST-16 · `link_url` = `""` หรือ URL `https://` ที่มี host ยาวไม่เกิน 500 (แบบ ACC-06) · `status` `ACTIVE` (ค่าเริ่มต้น) / `INACTIVE` · ลบ = soft delete · ลบ / เปลี่ยนสถานะได้โดยผู้มีสิทธิ์เขียนเว็บนั้น **ไม่ผูกกับคนอัป** · banner ไม่พบ / ของเว็บอื่น / ถูกลบ → `405404` ข้อความเดียวกัน · เรียงใหม่ → เก่า (ไม่มีการจัดลำดับรอบนี้) · รายชื่อแบ่งหน้าแบบเดียวกับ AST-27 |

### Bundle

| ID | Rule |
|---|---|
| AST-45 | bundle ของ (เว็บ, เกม) = `source` + `assets` (map `asset_type` → URL ตาม AST-25) + `theme` (AST-37 + shades) + `site` (branding) · **`assets` กับ `site` แยก key** — ช่อง asset ชื่ออะไรก็ไม่ชนกับ branding |
| AST-46 | ไม่มี asset ทั้งของเว็บและ GLOBAL → `assets = {}` ตอบสำเร็จ (ไม่ใช่ error) · อ่าน branding ไม่สำเร็จ → `site` เป็นค่าว่างตาม ACC-32 + log warn — **asset คือส่วนที่จำเป็น branding ล้มต้องไม่ทำให้ทั้ง bundle ล้ม** |
| AST-47 | branding ของ bundle: ใช้แถวของเว็บ · ถ้าเว็บไม่มีแถว → ไล่ขึ้นสายทีละชั้น ใช้**ทั้งแถว**ของชั้นแรกที่มี (ไม่ผสมทีละ field) · ไม่มีเลย → ค่าว่าง (❓ หัวข้อ 9 ข้อ 2) |
| AST-48 | bundle ฝั่งผู้เล่น: เว็บของ Member ตามหัวข้อ 9 ข้อ 2 · implement ใน module หน้าบ้าน แต่ต้องเรียก service เดียวกับ preview หลังบ้าน (ห้ามเขียน logic รวมซ้ำ) |

### Storage และประวัติ

| ID | Rule |
|---|---|
| AST-49 | DB เก็บ **storage key** เท่านั้น · URL = `S3_CDN_BASE_URL` (ตัด `/` ท้าย) + `/` + key (ตัด `/` หน้า) · key ใหม่ทุกครั้งที่อัป (ไม่เขียนทับ object เดิม) ตามรูปแบบในหัวข้อ 6 |
| AST-50 | ทุกตารางเก็บ `created_by` / `updated_by` เป็น username จริงของผู้กด (ไม่ใช่ `"system"` — ยกเว้นแถว seed) + เวลา · ประวัติแยกแบบ MGMT-60 ❓ หัวข้อ 9 ข้อ 5 |

## 4. สิ่งที่พบในโค้ดเก่า และการตัดสินใจ

extract จากโค้ดเท่านั้น (ไม่อ่าน doc / comment อธิบาย business / test ของโปรเจกต์เก่า)

| # | โค้ดเก่าทำอะไร | ถูก/ผิด | การตัดสินใจ |
|---|---|---|---|
| 1 | สร้างทุกตารางด้วย GORM AutoMigrate | ผิดกฎข้อ 20 | goose migration (หัวข้อ 6) |
| 2 | unique `(agent_id, game_type_id, asset_type_id)` ของ used แต่**ไม่มี partial index ของ GLOBAL** — DB ไม่กันแถว GLOBAL ซ้ำ | ผิด | partial unique 2 ตัว (AST-22) |
| 3 | `DROP INDEX idx_asset_used_slot` ทุกครั้งที่ boot แล้วให้ AutoMigrate สร้างใหม่ | ผิด | ไม่มี — index อยู่ใน migration |
| 4 | catalog (`minigame_asset_types`) และตารางเกมไม่มี unique ที่ชื่อ | ผิด | unique `(game_code, asset_type)` |
| 5 | ช่องที่ไม่มีใน catalog ถูกปฏิเสธ **แต่ถ้า lookup error จะข้ามไป และ repository สร้างช่องใหม่ให้เอง** | ผิด (ต้นเหตุช่องสะกดผิดซ้ำใน DB) | fail-closed (AST-11, AST-18) · repository ไม่มีการสร้างช่อง |
| 6 | แปลงชื่อเกม / ช่อง (ตัวเล็ก, ช่องว่าง → `_`, ตัดอักขระอื่น, ว่าง → `unknown`) ก่อนค้น | เสี่ยง | เทียบตรงทุกตัวอักษร (AST-14) |
| 7 | ช่องเลิกใช้ปฏิเสธก่อนตรวจขนาด · ขนาดภาพ fail-open 2 กรณี · ตรวจก่อนขึ้น S3 · ≤ 10 MB | ถูก | คงไว้ (AST-13, AST-15–17) |
| 8 | ชนิดไฟล์เชื่อ `Content-Type` ของ client · decoder รองรับ png/jpeg/gif/webp | เสี่ยง | ตรวจจากเนื้อไฟล์ (AST-16) |
| 9 | ตอบ error ด้วย `err.Error()` ดิบ (รวม `500`) · ข้อความ error ของขนาดภาพ / ช่องเลิกใช้ไม่ถึง client | ผิด | `apperr` + msg ที่บอกขนาดที่รับ (AST-17) |
| 10 | `update-used` ไม่เช็คว่า config เป็นเกมเดียวกับที่ส่ง (เอา `game_type_id` จาก request แต่ `asset_type_id` จาก config) | ผิด | เช็คเกม (AST-23 ข้อ ข) |
| 11 | upsert used แบบ find แล้ว update / insert ไม่มี lock — GLOBAL แข่งกันได้แถวซ้ำ | ผิด | `ON CONFLICT` บน partial index (AST-22) |
| 12 | ไม่พบ / ผิด scope / ของเว็บอื่น / ถูกลบ ตอบ `"asset_config_id N not found"` เหมือนกัน | ถูก | คงไว้ (`405402`) |
| 13 | ลบ GLOBAL: เฉพาะคนอัป, `409` ถ้ายังใช้, tx + `FOR UPDATE` + นับซ้ำ + จับ FK เป็น `409`, ลบไฟล์หลังลบแถว, ลบไฟล์ล้มยังตอบสำเร็จ, S3 ไม่พร้อม `503` เฉพาะ GLOBAL · ลบ AGENT: soft delete · ข้อความ `409` 2 แบบ | ถูก | คงไว้ (AST-28–32) |
| 14 | GLOBAL เขียนได้ด้วย `requireSystemRole` = role SUPERADMIN · **token ของ sub มี role ของเจ้าของ → sub ของ SUPERADMIN เขียน GLOBAL ได้โดยไม่ต้องมีสิทธิ์ asset** | ผิด | sub ต้องมี `asset` = `edit` (AST-03) |
| 15 | เป้าหมาย AGENT ผ่านด่าน "ตัวเองหรือสายล่าง" แต่ capability `asset.manage` ของ COMPANY / SHAREHOLDER / AGENT เป็น SELF → ในทางปฏิบัติแก้ได้แค่ของตัวเอง (เอกสารสรุปบอกว่า "บรรพบุรุษ" แก้ได้ — ไม่ตรงโค้ด) | — | แก้ได้เฉพาะตัวเอง (AST-04) · ❓ ข้อ 1 |
| 16 | "ถือ member ได้" (`requireCanHoldMembers`) จากตาราง `minigame_ownership_rules`: COMPANY 1to1 · SHAREHOLDER TRANSFER/B2C · SEAMLESS_MASTER/B2C · SEAMLESS_RESELLER/B2C · AGENT TRANSFER · ไม่ใช้กับเส้นลบ | — | ใช้ประเภทที่สร้าง Member ได้ตาม MGMT-02 (AST-02) · ❓ ข้อ 1 |
| 17 | อ่าน GLOBAL (assets / library / theme) เปิดให้ทุกคนที่ login รวม sub ที่ไม่มีสิทธิ์ asset | ผิด | ต้องมี `asset` ≥ `view` (AST-05) |
| 18 | เลือกเป้าหมายด้วย `?agent_id=` (ไม่ส่ง = GLOBAL · `me` = ตัวเอง · อ่านไม่ออกหรือ `0` → `400`) · เส้นลบเป็น `DELETE /assets/:config_id` | ถูกบางส่วน | ใช้ convention ของโปรเจกต์: GET/POST, id ใน body, `scope` บังคับ ไม่มี `me` (AST-08) · ❓ ข้อ 4 |
| 19 | merge used ทับล่าสุดแบบ "เพิ่มแถว" · ช่องที่ไม่มี used ใช้ไฟล์ล่าสุดทันที · fallback ทีละช่อง agent → GLOBAL (ไม่ไล่สาย) · `source` agent/global/mixed | ถูก | คงไว้ (AST-24, AST-25) · ❓ ข้อ 2 |
| 20 | ไม่มี keep-N / cron / purge ใดๆ · soft delete อยู่ถาวร | ถูก | คงไว้ (AST-20) |
| 21 | theme: fallback ทั้งชุด, whitelist 5 ฟอนต์, regex `#RRGGBB`, `stats_bar_color` เป็นชื่อ, update ครบ 5 ค่า · ไม่ normalize ตัวพิมพ์ของสี (คอลัมน์ `varchar(9)`) · ค่ากลางอยู่ในตารางเกม `minigame_types` | ถูกบางส่วน | คงกฎไว้ · เก็บสีตัวเล็ก (AST-35) · ค่ากลางเป็นแถว GLOBAL ใน `game_themes` (โปรเจกต์นี้ไม่มีตารางเกม) |
| 22 | shades 19 ระดับ ฐาน `900` บน RGB หาร 10 · `math.Round` (= round-half-up สำหรับค่าบวก) · clamp · hex ตัวเล็ก · สีไม่ถูกต้องได้ shades `nil` เงียบๆ | ถูก | คงไว้ (AST-39) · สีไม่ถูกต้องเข้า DB ไม่ได้ (AST-35) |
| 23 | `allowed_sizes` เป็น text เก็บ JSON (`""` = ไม่มีสเปก, JSON พังก็ถือว่าไม่มีสเปก) | เสี่ยง | `JSONB NOT NULL DEFAULT '[]'` (seed เองจึงไม่ต้องรองรับ `""`) |
| 24 | seed catalog รันเมื่อ `IS_SEED=true` · ตั้งขนาดใหม่ / ปลด retired ให้ช่องใน catalog · **retire ทุกช่องที่ไม่อยู่ใน catalog** · มีขนาดเฉพาะ `scratch_card` | — | seed ใน migration (หัวข้อ 6) · เปลี่ยนภายหลังด้วย migration ใหม่ |
| 25 | ตาราง `minigame_asset_groups` สร้างไว้แต่ไม่มี service ใช้ | ตาย | ไม่สร้าง |
| 26 | used เก็บ `asset_path` ซ้ำจาก config | — | ไม่เก็บซ้ำ join `asset_configs` (กันค่าไม่ตรงกัน) |
| 27 | banner: `created_by` = username ของ**เว็บเป้าหมาย** แต่ลบต้อง `created_by` = ผู้เรียก → ชั้นบน / sub ที่อัปลบไม่ได้ · `link_url` ไม่ตรวจ · `sort_order` ไม่เคยตั้ง · ไม่มีเส้นแก้ / เส้นฝั่งผู้เล่น · อยู่ใต้ capability `announcement` | ผิดบางส่วน | ลบตามสิทธิ์เว็บ (AST-44) · ตรวจ `link_url` · ไม่มี sort · ❓ ข้อ 3 |
| 28 | branding เป็นคอลัมน์ใน `user_agents` (`site_name`, `site_logo_path`, `site_favicon_path`, `site_theme`) · โลโก้ / favicon เขียนทับ path ไม่ลบไฟล์เดิม · favicon รับ `x-icon` / `png` · catalog ธีม `default dark light neon gold` | — | ตาราง `site_brandings` แยก (ไม่แก้ `user_agents`) · ❓ ข้อ 5 |
| 29 | bundle ผู้เล่น: data เป็น map แบน `{asset_type: url, ..., "site": {...}}` + header `X-Asset-Source` · ช่องชื่อ `site` จะชน · branding ไล่สายจากตัวเอง แถวแรกที่มีค่าใดๆ ชนะทั้งแถว · branding ล้มได้ค่าว่างเงียบๆ · ไม่มี asset ตอบ code แยก | ผิดบางส่วน | แยก key (AST-45) · `source` อยู่ใน body · ไม่มี asset = `{}` (AST-46) |
| 30 | bundle หลังบ้านไม่มี `site` · ไม่มี asset ตอบ HTTP 404 | ไม่ตรงกับฝั่งผู้เล่น | preview ใช้ service เดียวกับผู้เล่น (AST-48) |
| 31 | cache Redis ของ config / used / theme กลาง (bump version ตอนเขียน) | — | ไม่มี cache รอบนี้ — เพิ่มทีหลังถ้าวัดแล้วจำเป็น |

### ข้อเสนอใน `ASSET_SYSTEM_SPEC.md` หมวด 5 → rule ในเอกสารนี้

| ข้อเสนอ | rule | | ข้อเสนอ | rule | | ข้อเสนอ | rule |
|---|---|---|---|---|---|---|---|
| 1 | AST-01 | | 10 | AST-28 | | 19 | AST-33 |
| 2 | AST-26 | | 11 | AST-29 | | 20 | AST-38 |
| 3 | AST-23, AST-31 | | 12 | AST-30 | | 21 | AST-23, AST-31, AST-44 |
| 4 | AST-23 | | 13 | AST-32 | | 22 | AST-07 |
| 5 | AST-11, AST-18 | | 14 | AST-32 | | 23 | AST-08 · ❓ ข้อ 4 (เปลี่ยน) |
| 6 | AST-13 | | 15 | AST-24 | | 24 | AST-08 |
| 7 | AST-17 | | 16 | AST-34 | | 25 | AST-09 |
| 8 | AST-15 | | 17 | AST-35 | | 26 | AST-09 |
| 9 | AST-20 | | 18 | AST-36 | | 27 | AST-39, AST-40 |

anti-pattern ทั้ง 9 ข้อของหมวด 6 ถูกห้ามไว้ที่ AST-11 · AST-20 · AST-25 · ตารางลบ · AST-28 · AST-23 · AST-24 · AST-08 · AST-40 ตามลำดับ

## 5. Endpoints

ใช้แค่ `GET` / `POST` · ระบุบัญชีด้วย `agent_id` ใน body (ผู้เรียกมาจาก token) — เส้นอ่านที่ต้องระบุเป้าหมายจึงเป็น `POST` ·
ใต้ `/bo/pr` ผ่าน `Authenticated` + `PassedGates()` · สิทธิ์เมนูด้วย `RequirePermission(MenuAsset, view|edit)` ·
สิทธิ์ต่อเป้าหมาย (AST-03/04/05/09) ด้วย `ResolveAssetTarget` + `RequireAssetTarget(view|edit)` · ทุกอย่างเขียนบรรทัดเดียวกับ route (กฎข้อ 28) ·
รายชื่อที่ยาวได้ไม่จำกัด (`library`, `banners/list`) แบ่งหน้าตามกฎข้อ 19 (`page` / `limit` ใน body · `response.Page`) ·
รายการขนาดคงที่ (`games`, `slots`, `site-themes`, `list` ≤ จำนวนช่องของเกม) ตอบเป็น array ธรรมดาแบบ `copy-sources` ของ module ②

error ร่วมทุกเส้น: `401202`, `401203`, `401301`, `401302`, `401304`, `401306`, `401307`, `401311`, `402303` ·
เส้นที่ระบุเป้าหมายเพิ่ม: `422` (AST-08), `402402`, `405301`, `405302`

| Method | Path | สิทธิ์ (`asset`) | หมายเหตุ | Contract เทียบของเก่า |
|---|---|---|---|---|
| GET | `/api/v1/bo/pr/asset/games` | view | รายชื่อเกม + รูปปก GLOBAL | เปลี่ยน (เดิม `GET /pb/games` ไม่ต้อง login) |
| GET | `/api/v1/bo/pr/asset/slots?game_code=` | view | catalog ของเกม (ไม่รวมช่องเลิกใช้) | เปลี่ยน (เดิม `GET /pb/games/:game_type/asset-types`) |
| POST | `/api/v1/bo/pr/asset/upload` | edit + เป้าหมาย edit | multipart | เปลี่ยน (เดิม `POST /pr/assets/upload?agent_id=`) |
| POST | `/api/v1/bo/pr/asset/list` | view + เป้าหมาย view | ใช้งานจริงต่อช่อง | เปลี่ยน (เดิม `GET /pr/assets`) |
| POST | `/api/v1/bo/pr/asset/library` | view + เป้าหมาย view | ประวัติไฟล์ของช่อง | เปลี่ยน (เดิม `GET /pr/assets/library`) |
| POST | `/api/v1/bo/pr/asset/update-used` | edit + เป้าหมาย edit | เลือกไฟล์ให้ช่อง | เปลี่ยน (เดิม `POST /pr/assets/used`) |
| POST | `/api/v1/bo/pr/asset/delete` | edit + เป้าหมาย edit | ลบ config | เปลี่ยน (เดิม `DELETE /pr/assets/:config_id`) |
| POST | `/api/v1/bo/pr/asset/theme/detail` | view + เป้าหมาย view | theme + shades | เปลี่ยน (เดิม `GET /pr/games/:game_type/theme`) |
| POST | `/api/v1/bo/pr/asset/theme/update` | edit + เป้าหมาย edit | แทนทั้งชุด | เปลี่ยน (เดิม `POST /pr/games/:game_type/theme`) |
| GET | `/api/v1/bo/pr/asset/site-themes` | view | catalog ธีมหน้าเว็บ | เปลี่ยน path |
| POST | `/api/v1/bo/pr/asset/branding/detail` | view + เว็บ view | | เปลี่ยน (เดิม `GET /pr/agents/:agent_id/site-settings`) |
| POST | `/api/v1/bo/pr/asset/branding/update` | edit + เว็บ edit | ชื่อเว็บ + ธีมหน้าเว็บ | เปลี่ยน |
| POST | `/api/v1/bo/pr/asset/branding/upload-logo` | edit + เว็บ edit | multipart | เปลี่ยน |
| POST | `/api/v1/bo/pr/asset/branding/upload-favicon` | edit + เว็บ edit | multipart | เปลี่ยน |
| POST | `/api/v1/bo/pr/asset/banners/list` | view + เว็บ view | | เปลี่ยน |
| POST | `/api/v1/bo/pr/asset/banners/create` | edit + เว็บ edit | multipart | เปลี่ยน |
| POST | `/api/v1/bo/pr/asset/banners/update-status` | edit + เว็บ edit | `ACTIVE` / `INACTIVE` | **ใหม่** |
| POST | `/api/v1/bo/pr/asset/banners/delete` | edit + เว็บ edit | soft delete | เปลี่ยน (เดิม `POST .../banners/:id/delete`) |
| POST | `/api/v1/bo/pr/asset/bundle` | view + เว็บ view | preview ของเว็บ | เปลี่ยน (เดิมไม่มี `site`) |
| GET | `/api/v1/player/games/{game_code}/bundle` | token ผู้เล่น | **contract เท่านั้น** — implement ใน module หน้าบ้าน | `data` เปลี่ยนรูป (AST-45) |

ตัวอย่างใช้ข้อมูลชุดเดียวกับ `agent_management.md`: `superadmin` · `comp01` (id 10, Company Transfer — **ไม่ใช่เว็บ**) →
`share01` (id 12, Share B2C — เว็บ) → `agent01` (id 15 — เว็บ) · sub ของ share01 = `share01@design` · URL ตัวอย่างใช้
`S3_CDN_BASE_URL = https://cdn.example.com`

### GET /api/v1/bo/pr/asset/games

Response `data`:
```json
[
  { "game_code": "scratch_card", "game_cover": "https://cdn.example.com/games/scratch_card/global/game_cover/1759900000000000000-a1b2c3d4.png" },
  { "game_code": "coin_toss", "game_cover": "" },
  { "game_code": "rock_paper_scissors", "game_cover": "" }
]
```
`game_cover` = ไฟล์ใช้งานจริงของช่อง `game_cover` ฝั่ง GLOBAL (AST-24) · ไม่มี = `""` · ชื่อเกมที่แสดงบนจอเป็นของหน้าบ้าน

### GET /api/v1/bo/pr/asset/slots?game_code=scratch_card

Response `data`:
```json
[
  { "asset_type": "icon_button", "description": "ปุ่มไอคอน", "allowed_sizes": [ { "width": 32, "height": 32 }, { "width": 40, "height": 40 } ] },
  { "asset_type": "win_highlight", "description": "แสงไฮไลต์ตอนชนะ", "allowed_sizes": [ { "width": 425, "height": 425 } ] }
]
```
`allowed_sizes = []` = ไม่ตรวจขนาด · Error codes: `422` (ไม่ส่ง `game_code`), `405401`

### POST /api/v1/bo/pr/asset/upload (multipart/form-data)

| field | ค่า |
|---|---|
| `scope` | `GLOBAL` / `AGENT` |
| `agent_id` | id ของเว็บ (เฉพาะ `AGENT`) |
| `game_code` | เช่น `scratch_card` |
| `asset_type` | ชื่อช่องตาม catalog |
| `file` | ไฟล์รูป |

Response `data`:
```json
{
  "id": 9001,
  "scope": "AGENT",
  "agent_id": 12,
  "game_code": "scratch_card",
  "asset_type": "icon_button",
  "url": "https://cdn.example.com/games/scratch_card/agents/12/icon_button/1759900000000000000-a1b2c3d4.png",
  "created_by": "share01@design",
  "created_at": "2026-10-08T10:00:00+07:00"
}
```
Error codes: `422`, `405101`, `405102`, `405103`, `405304`, `405401`, `505801`, `505802`, `500`

### POST /api/v1/bo/pr/asset/list (ไฟล์ใช้งานจริงต่อช่อง)

Request:
```json
{ "scope": "AGENT", "agent_id": 12, "game_code": "scratch_card" }
```
Response `data`:
```json
{
  "source": "mixed",
  "items": [
    { "asset_type": "icon_button", "config_id": 9001, "url": "https://cdn.example.com/games/scratch_card/agents/12/icon_button/1759900000000000000-a1b2c3d4.png", "source": "agent", "is_retired": false },
    { "asset_type": "win_highlight", "config_id": 120, "url": "https://cdn.example.com/games/scratch_card/global/win_highlight/1759800000000000000-e5f6a7b8.png", "source": "global", "is_retired": false }
  ]
}
```
1 รายการต่อช่องที่มีไฟล์ (ตาม AST-25) · ช่องที่ไม่มีไฟล์ทั้งของเว็บและ GLOBAL ไม่อยู่ใน `items` · เป้าหมาย `GLOBAL` ทุกรายการเป็น `global` ·
Error codes: `422`, `405401`

### POST /api/v1/bo/pr/asset/library

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
    { "config_id": 9001, "url": "https://cdn.example.com/games/scratch_card/agents/12/icon_button/1759900000000000000-a1b2c3d4.png", "source": "agent", "is_used": true, "created_by": "share01@design", "created_at": "2026-10-08T10:00:00+07:00" },
    { "config_id": 118, "url": "https://cdn.example.com/games/scratch_card/global/icon_button/1759700000000000000-c9d0e1f2.png", "source": "global", "is_used": false, "created_by": "superadmin", "created_at": "2026-10-01T09:00:00+07:00" }
  ]
}
```
`is_used` = เป็นไฟล์ใช้งานจริงของช่องนี้ของเป้าหมาย · Error codes: `422`, `405401`

### POST /api/v1/bo/pr/asset/update-used

Request:
```json
{ "scope": "AGENT", "agent_id": 12, "game_code": "scratch_card", "config_id": 118 }
```
Response: `{"code":200,"msg":"สำเร็จ"}` (ไม่มี `data`) · Error codes: `422`, `405304`, `405402`

### POST /api/v1/bo/pr/asset/delete

Request:
```json
{ "scope": "GLOBAL", "config_id": 118 }
```
Response: `{"code":200,"msg":"สำเร็จ"}` (ไม่มี `data`) · Error codes: `422`, `405303`, `405402`, `405403`, `505801`

### POST /api/v1/bo/pr/asset/theme/detail

Request:
```json
{ "scope": "AGENT", "agent_id": 12, "game_code": "scratch_card" }
```
Response `data`:
```json
{
  "source": "global",
  "font_family": "noto",
  "main_color_1": "#f7447e",
  "main_color_2": "#ffffff",
  "stats_bar_color": "main_color_1",
  "stats_bar_color_hex": "#f7447e",
  "stats_bar_font_family": "prompt",
  "main_color_1_shades": { "50": "#feecf2", "900": "#f7447e", "1800": "#19070d" },
  "main_color_2_shades": { "50": "#ffffff", "900": "#ffffff", "1800": "#1a1a1a" }
}
```
shades ในตัวอย่างย่อไว้ — ของจริงมีครบ 19 key · Error codes: `422`

### POST /api/v1/bo/pr/asset/theme/update

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
Response: `{"code":200,"msg":"สำเร็จ"}` (ไม่มี `data`) · Error codes: `422` (ขาดค่า / ฟอนต์ไม่อยู่ในรายการ / สีผิดรูปแบบ — msg บอก field)

### GET /api/v1/bo/pr/asset/site-themes

Response `data`:
```json
[ { "code": "default", "name": "Default", "preview": "" }, { "code": "dark", "name": "Dark", "preview": "" } ]
```
เฉพาะ `is_active` เรียงตาม `sort_order`

### POST /api/v1/bo/pr/asset/branding/detail

Request:
```json
{ "agent_id": 12 }
```
Response `data`:
```json
{ "site_name": "Share One", "site_theme": "dark", "logo": "https://cdn.example.com/sites/12/logo/1759900000000000000-0a1b2c3d.png", "favicon": "" }
```
ไม่เคยตั้ง → ทุก field `""` (ไม่ไล่สาย — การไล่สายมีเฉพาะ bundle AST-47) · Error codes: `422`

### POST /api/v1/bo/pr/asset/branding/update

Request:
```json
{ "agent_id": 12, "site_name": "Share One", "site_theme": "dark" }
```
Response: `{"code":200,"msg":"สำเร็จ"}` · Error codes: `422`

### POST /api/v1/bo/pr/asset/branding/upload-logo · /branding/upload-favicon (multipart/form-data)

field: `agent_id`, `file` · Response `data`: `{ "url": "https://cdn.example.com/sites/12/logo/..." }` ·
Error codes: `422`, `405102`, `405103`, `505801`, `505802`

### POST /api/v1/bo/pr/asset/banners/list

Request: `{ "agent_id": 12, "page": 1, "limit": 20 }` · Response `data` (`response.Page`):
```json
{
  "current_page": 1,
  "total_page": 1,
  "total_count": 1,
  "limit": 20,
  "has_next": false,
  "has_prev": false,
  "data": [
    { "id": 55, "image": "https://cdn.example.com/sites/12/banners/1759900000000000000-11223344.png", "link_url": "https://share01.example.com/promo", "status": "ACTIVE", "created_by": "share01", "created_at": "2026-10-08T10:00:00+07:00" }
  ]
}
```

### POST /api/v1/bo/pr/asset/banners/create (multipart/form-data)

field: `agent_id`, `file`, `link_url` (`""` ได้), `status` (ไม่ส่ง = `ACTIVE`) · Response `data`: แถวเดียวรูปแบบเดียวกับ list ·
Error codes: `422`, `405102`, `405103`, `505801`, `505802`

### POST /api/v1/bo/pr/asset/banners/update-status · /banners/delete

Request: `{ "agent_id": 12, "id": 55, "status": "INACTIVE" }` (delete ไม่มี `status`) · Response: `{"code":200,"msg":"สำเร็จ"}` ·
Error codes: `422`, `405404`

### POST /api/v1/bo/pr/asset/bundle (preview ของเว็บ)

Request:
```json
{ "agent_id": 12, "game_code": "scratch_card" }
```
Response `data`:
```json
{
  "source": "mixed",
  "assets": {
    "icon_button": "https://cdn.example.com/games/scratch_card/agents/12/icon_button/1759900000000000000-a1b2c3d4.png",
    "win_highlight": "https://cdn.example.com/games/scratch_card/global/win_highlight/1759800000000000000-e5f6a7b8.png"
  },
  "theme": { "source": "global", "font_family": "noto", "main_color_1": "#f7447e", "...": "รูปแบบเดียวกับ theme/detail" },
  "site": { "site_name": "Share One", "site_theme": "dark", "logo": "https://cdn.example.com/sites/12/logo/1759900000000000000-0a1b2c3d.png", "favicon": "" }
}
```
Error codes: `422`

### GET /api/v1/player/games/{game_code}/bundle (contract — implement ใน module หน้าบ้าน)

ผู้เล่นใช้ token ของ Member · เว็บ = ตามหัวข้อ 9 ข้อ 2 · `data` รูปแบบเดียวกับ `POST /bo/pr/asset/bundle` ·
การเช็คเกมเปิด / ปิด (`status_game` — MGMT-20) เป็นของ module หน้าบ้าน

## 6. Schema

migration ใหม่ (ไม่แก้ migration เดิม · ไม่ ALTER `user_agents`) · เกมอ้างด้วย `game_code` string (MGMT-16 — ไม่มีตารางเกม)

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

CREATE TABLE asset_configs (                        -- ทุกเวอร์ชันที่อัป (AST-20)
    id            BIGSERIAL    PRIMARY KEY,
    scope         VARCHAR(10)  NOT NULL,
    agent_id      BIGINT       REFERENCES user_agents(id), -- NULL = GLOBAL
    asset_type_id BIGINT       NOT NULL REFERENCES asset_types(id),
    storage_key   VARCHAR(500) NOT NULL,
    created_by    VARCHAR(71)  NOT NULL,              -- username จริง (AST-28 เทียบคนอัป)
    is_deleted    BOOLEAN      NOT NULL DEFAULT false, -- ใช้กับ AGENT เท่านั้น (AST-31)
    deleted_at    TIMESTAMPTZ,
    deleted_by    VARCHAR(71),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT ck_asset_configs_scope CHECK (
        (scope = 'GLOBAL' AND agent_id IS NULL) OR (scope = 'AGENT' AND agent_id IS NOT NULL)),
    CONSTRAINT ck_asset_configs_soft_delete CHECK (scope = 'AGENT' OR is_deleted = false),
    CONSTRAINT ck_asset_configs_deleted_at CHECK (is_deleted = (deleted_at IS NOT NULL))
);
CREATE INDEX idx_asset_configs_slot ON asset_configs(asset_type_id, agent_id, created_at DESC) WHERE is_deleted = false;
CREATE INDEX idx_asset_configs_agent ON asset_configs(agent_id) WHERE agent_id IS NOT NULL;

CREATE TABLE asset_used (                           -- 1 แถวต่อช่อง (AST-22)
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
-- Postgres มอง NULL ไม่เท่ากัน → unique ธรรมดากัน GLOBAL ซ้ำไม่ได้ ต้องแยก 2 ตัว
CREATE UNIQUE INDEX uq_asset_used_agent_slot  ON asset_used(agent_id, asset_type_id) WHERE agent_id IS NOT NULL;
CREATE UNIQUE INDEX uq_asset_used_global_slot ON asset_used(asset_type_id)           WHERE agent_id IS NULL;
CREATE INDEX idx_asset_used_config ON asset_used(asset_config_id);               -- นับผู้ถือตอนลบ (AST-28, AST-32)

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

CREATE TABLE site_themes (                          -- catalog ธีมหน้าเว็บ (seed)
    code        VARCHAR(50)  PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    preview_key VARCHAR(500) NOT NULL DEFAULT '',
    sort_order  INT          NOT NULL DEFAULT 0,
    is_active   BOOLEAN      NOT NULL DEFAULT true
);

CREATE TABLE site_brandings (                       -- 1 แถวต่อเว็บ (AST-41) — แยกจาก user_agents
    agent_id    BIGINT       PRIMARY KEY REFERENCES user_agents(id),
    site_name   VARCHAR(50)  NOT NULL DEFAULT '',
    site_theme  VARCHAR(50)  NOT NULL DEFAULT '',     -- '' หรือ site_themes.code (ตรวจใน service)
    logo_key    VARCHAR(500) NOT NULL DEFAULT '',
    favicon_key VARCHAR(500) NOT NULL DEFAULT '',
    created_by  VARCHAR(71)  NOT NULL,
    updated_by  VARCHAR(71)  NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE site_banners (                         -- AST-44
    id          BIGSERIAL    PRIMARY KEY,
    agent_id    BIGINT       NOT NULL REFERENCES user_agents(id),
    image_key   VARCHAR(500) NOT NULL,
    link_url    VARCHAR(500) NOT NULL DEFAULT '',
    status      VARCHAR(10)  NOT NULL DEFAULT 'ACTIVE',
    is_deleted  BOOLEAN      NOT NULL DEFAULT false,
    deleted_at  TIMESTAMPTZ,
    created_by  VARCHAR(71)  NOT NULL,
    updated_by  VARCHAR(71)  NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT ck_site_banners_status CHECK (status IN ('ACTIVE','INACTIVE')),
    CONSTRAINT ck_site_banners_deleted_at CHECK (is_deleted = (deleted_at IS NOT NULL))
);
CREATE INDEX idx_site_banners_agent ON site_banners(agent_id, created_at DESC) WHERE is_deleted = false;
```

**Seed (อยู่ใน migration เดียวกับที่สร้างตาราง — ไม่มี flag `IS_SEED`):**

| ตาราง | ข้อมูล |
|---|---|
| `asset_types` · `scratch_card` (19 ช่อง มีขนาด) | `background` 1920×1080 · `game_cover` 608×1080 · `bg_loading` 449×52 · `frame_loading` 449×52 · `background_main` 608×1080 · `main_scratch_area` 544×851 · `outer_frame` 544×851 · `top_scratch` 544×851 · `main_play` 186×76 · `icon_button` 32×32 หรือ 40×40 · `auto_scratchers` 520×347 · `selection_button` 115×76 · `bet_limit` 322×322 · `bet_history` 572×929 · `bar_bet_history` 540×74 · `how_to_scratchers` 572×929 · `winner` 502×514 · `you_lose` 544×287 · `win_highlight` 425×425 |
| `asset_types` · `coin_toss` (11 ช่อง ไม่มีขนาด) | `background` · `coin_heads` · `coin_tails` · `selection_button` · `outer_frame` · `bet_limit` · `bet_history` · `winner` · `game_cover` · `loading` · `win_highlight` |
| `asset_types` · `rock_paper_scissors` (12 ช่อง ไม่มีขนาด) | `background` · `hand_rock` · `hand_paper` · `hand_scissors` · `selection_button` · `outer_frame` · `bet_limit` · `bet_history` · `winner` · `game_cover` · `loading` · `win_highlight` |
| `game_themes` | แถว GLOBAL ทั้ง 3 เกม: `noto` · `#f7447e` · `#ffffff` · `main_color_1` · `prompt` · `created_by = 'migration'` |
| `site_themes` | `default` · `dark` · `light` · `neon` · `gold` |

ขนาดและรายชื่อช่องมาจาก seed ของโค้ดเก่า — ❓ หัวข้อ 9 ข้อ 7 ให้ทีมเกมยืนยัน

**Storage key** (AST-49) · `{ts}` = unix nano · `{rand}` = 8 hex จาก `crypto/rand` · `{ext}` จากชนิดที่ตรวจได้ (AST-16):

| ของ | key |
|---|---|
| asset GLOBAL | `games/{game_code}/global/{asset_type}/{ts}-{rand}.{ext}` |
| asset AGENT | `games/{game_code}/agents/{agent_id}/{asset_type}/{ts}-{rand}.{ext}` |
| โลโก้ / favicon | `sites/{agent_id}/logo/{ts}-{rand}.{ext}` · `sites/{agent_id}/favicon/{ts}-{rand}.{ext}` |
| banner | `sites/{agent_id}/banners/{ts}-{rand}.{ext}` |

## 7. Test cases

ตัวอย่างใช้ข้อมูลหัวข้อ 5 · `sa@ops` = sub ของ superadmin

| Rule | Input | Expected |
|---|---|---|
| AST-01 | share01 และ agent01 (ใต้สายเดียวกัน) ตั้ง theme / asset ช่องเดียวกันต่างกัน | แต่ละเว็บได้ของตัวเอง ไม่กระทบกัน |
| AST-02 | `scope=AGENT, agent_id=10` (comp01 · Company Transfer) | `405302` |
| AST-02 | `scope=AGENT, agent_id=15` (agent01) โดย agent01 | สำเร็จ |
| AST-03 | superadmin อัป `scope=GLOBAL` | สำเร็จ · `created_by = superadmin` |
| AST-03 | `sa@ops` ที่ `asset = view` อัป `scope=GLOBAL` | `402303` |
| AST-03 | `sa@ops` ที่ `asset = edit` อัป `scope=GLOBAL` | สำเร็จ · `created_by = sa@ops` |
| AST-03 | share01 อัป `scope=GLOBAL` | `405301` |
| AST-04 | share01 อัปให้ `agent_id=15` (ลูกของตัวเอง) | `402402` |
| AST-04 | `share01@design` ที่ `asset = edit` อัปให้ `agent_id=12` | สำเร็จ |
| AST-05 | `share01@design` ที่ `asset = off` เรียก `list scope=GLOBAL` | `402303` |
| AST-05 | agent01 เรียก `list scope=GLOBAL` | สำเร็จ |
| AST-06 | ADMIN เรียก `asset/games` | `402303` |
| AST-06 | share01 ถูก SUSPEND เรียก `asset/list` | `401311` |
| AST-07 | share01 `update-used` / `delete` ด้วย `config_id` ของ agent01 | `405402` (ข้อความเดียวกับไม่พบ) |
| AST-08 | ไม่ส่ง `scope` / `scope="global"` / `scope=null` | `422` |
| AST-08 | `scope=AGENT` ไม่ส่ง `agent_id` / `agent_id=0` / `agent_id="abc"` | `422` (ไม่ตกไปเป็น GLOBAL) |
| AST-08 | `scope=GLOBAL, agent_id=12` | `422` |
| AST-09 | เส้น AGENT-only (branding) ตอนไม่มีผล resolve (bug จำลอง) | ปฏิเสธ ไม่ผ่าน |
| AST-10 | `slots?game_code=plinko` | `405401` |
| AST-11 | อัป `asset_type = "win_highlite"` (สะกดผิด) | `405401` · `asset_types` ไม่มีแถวเพิ่ม |
| AST-12 | migration แก้ขนาดของ `win_highlight` | `slots` และการตรวจตอน `upload` ใช้ขนาดใหม่พร้อมกัน |
| AST-13 | ช่องถูก retire แล้วอัป / `update-used` | `405304` |
| AST-13 | ช่องถูก retire ระหว่างที่เว็บใช้อยู่ | `list` / `bundle` ยังมีช่องนั้น · `slots` ไม่มี |
| AST-14 | `asset_type = "Icon_Button"` | `405401` |
| AST-15 | ช่อง retired + ไฟล์ขนาดผิด | `405304` (ไม่ถึงตรวจขนาด) · ไม่มี object ใน bucket |
| AST-16 | ไฟล์ 10 MB + 1 byte | `405103` |
| AST-16 | ไฟล์ PDF ที่ส่ง `Content-Type: image/png` | `405102` |
| AST-17 | `win_highlight` ไฟล์ 400×400 | `405101` msg มี `425×425` · ไม่มี object ใน bucket |
| AST-17 | `icon_button` ไฟล์ 40×40 | สำเร็จ (ขนาดที่ 2) |
| AST-17 | ช่องของ `coin_toss` (ไม่มีสเปก) ไฟล์ขนาดใดก็ได้ | สำเร็จ |
| AST-17 | webp ที่ decoder อ่านขนาดไม่ได้ | สำเร็จ (fail-open) |
| AST-18 | DB ล่มระหว่างอ่าน catalog | `500` · ไม่มี config · ไม่มีช่องใหม่ |
| AST-19 | storage ไม่พร้อม แล้วอัป | `505801` |
| AST-19 | อัปขึ้น S3 สำเร็จ แต่ insert config ล้ม | `500` · object ถูกลบ |
| AST-20 | อัปช่องเดิม 30 ครั้ง | config 30 แถว ไม่มีแถวถูกลบ |
| AST-21 | อัปไฟล์ใหม่ขณะช่องมี used ชี้ไฟล์เก่า | ใช้งานจริงยังเป็นไฟล์เก่า · `asset_used` ไม่เปลี่ยน |
| AST-22 | `update-used` ช่องเดิม 2 ครั้งต่างไฟล์ | `asset_used` ของช่องนั้นมี 1 แถว ชี้ไฟล์ล่าสุดที่เลือก |
| AST-22 | `update-used scope=GLOBAL` ช่องเดียวกันพร้อมกัน 2 คำขอ | สำเร็จทั้งคู่ · GLOBAL ของช่องนั้นมี 1 แถว |
| AST-23 | `game_code=coin_toss` + `config_id` ของ `scratch_card` | `405402` |
| AST-23 | `scope=GLOBAL` เลือก config ของ share01 | `405402` |
| AST-23 | share01 เลือก config GLOBAL | สำเร็จ |
| AST-23 | เลือก config ที่ถูก soft delete | `405402` |
| AST-24 | เว็บไม่เคย `update-used` แล้วอัป 2 ไฟล์ | `list` ได้ไฟล์ที่อัปล่าสุด |
| AST-24 | **เว็บไม่เคยอัปช่อง `winner` แต่เลือก config GLOBAL ตัวเก่า (ไม่ใช่ตัวล่าสุด) ไว้** | `list` / `bundle` ช่อง `winner` = ตัวที่เลือก (ไม่ใช่ GLOBAL ล่าสุด) · `source = agent` |
| AST-25 | เว็บอัป 1 ใน 19 ช่องของ `scratch_card` · GLOBAL มีครบ | `bundle.assets` มี 19 ช่อง · `source = mixed` |
| AST-25 | เว็บไม่มีของตัวเองเลย | `source = global` |
| AST-26 | repository "config ล่าสุดของเว็บ share01" ขณะ GLOBAL มีช่องเดียวกัน | คืนเฉพาะแถว `AGENT` ของ share01 (integration test ของ repository) |
| AST-27 | `library` ของ share01 ช่อง `icon_button` | ของ share01 + ของ GLOBAL ใหม่ → เก่า · ไม่มีของเว็บอื่น · ไม่มีแถวที่ลบแล้ว |
| AST-27 | ช่องมี 30 เวอร์ชัน · ไม่ส่ง `page` / `limit` · `limit = 500` | หน้า 1 ได้ 20 แถว `total_count = 30` · `limit` ถูกปรับเป็น 100 |
| AST-28 | superadmin ลบ config GLOBAL ที่ `sa@ops` อัป | `405303` |
| AST-28 | ลบ config GLOBAL ที่ไม่มีใครเลือก | แถวหาย · object หาย |
| AST-28 | **ลบ GLOBAL ขณะเว็บอื่นยิง `update-used` เลือก config เดียวกันพร้อมกัน** | ผลอย่างใดอย่างหนึ่ง: ลบสำเร็จ + `update-used` ได้ `405402` · หรือ `update-used` สำเร็จ + ลบได้ `405403` · ไม่มี `500` · ไม่มี used ชี้ config ที่หายไป |
| AST-29 | ลบแถวสำเร็จ แต่ลบ object ไม่ได้ | ตอบสำเร็จ · log error มี storage key |
| AST-30 | storage ไม่พร้อม · ลบ GLOBAL | `505801` · แถวยังอยู่ |
| AST-30 | storage ไม่พร้อม · ลบ AGENT / `list` / `theme/detail` | สำเร็จ |
| AST-31 | share01 ลบ config ของตัวเองที่ไม่ได้ใช้ | `is_deleted = true` · object ยังอยู่ |
| AST-31 | share01 ลบ config GLOBAL | `405402` |
| AST-31 | `share01@design` ลบ config ที่ share01 อัป | สำเร็จ (ไม่ผูกคนอัป) |
| AST-32 | ลบ GLOBAL ที่ 3 เว็บเลือกใช้ | `405403` msg "ถูก 3 เว็บเลือกใช้อยู่" |
| AST-32 | ลบ GLOBAL ที่ช่อง GLOBAL เลือกใช้อยู่ | `405403` msg "เลือกไฟล์อื่นให้ช่องนี้ก่อน" |
| AST-33 | `theme/update` ไม่ส่ง `stats_bar_font_family` | `422` · ค่าเดิมไม่เปลี่ยน |
| AST-34 | `font_family = "Kanit"` / `"Comic Sans"` / `"noto; color:red"` | `422` |
| AST-35 | `#FFF` / `FF0000` / `#GG0000` / `#ff00001` | `422` |
| AST-35 | `#1E90FF` | สำเร็จ · `theme/detail` ได้ `#1e90ff` |
| AST-36 | `stats_bar_color = main_color_2` แล้วแก้ `main_color_2` | `stats_bar_color_hex` เปลี่ยนตาม |
| AST-37 | share01 ไม่เคยตั้ง theme | ได้ 5 ค่ากลาง · `source = global` |
| AST-37 | superadmin แก้ theme GLOBAL หลัง share01 ถูกสร้าง | share01 (ไม่เคยตั้ง) ได้ค่าใหม่ทันที |
| AST-37 | share01 ตั้ง theme แล้ว superadmin แก้ GLOBAL | share01 ได้ค่าของตัวเองทั้ง 5 ค่า |
| AST-38 | ค่าเดียวกันที่ผ่าน / ไม่ผ่านฝั่ง AGENT | ได้ผลเดียวกันฝั่ง GLOBAL |
| AST-39 | shades ของ `#f7447e` | `900 = #f7447e` · ครบ 19 key · ตรงกับ fixture palette ของหน้าบ้านทุกค่า (table test ใน `app/core/asset`) |
| AST-39 | shades ของ `#000000` / `#ffffff` | `50 = #e6e6e6` · `1800 = #000000` / `50 = #ffffff` · `1800 = #1a1a1a` |
| AST-40 | share01 ตั้ง `main_color_1 = #1e90ff` | shades ใน `theme/detail` / `bundle` คำนวณจาก `#1e90ff` |
| AST-41 | comp01 เรียก `branding/update` `agent_id=10` | `405302` |
| AST-42 | `site_theme = "purple"` / `site_name = ""` | `422` |
| AST-42 | `site_theme = ""` | สำเร็จ · ธีมถูกล้าง |
| AST-43 | favicon เป็น `.ico` / `.png` | สำเร็จ · `.jpg` → `405102` |
| AST-44 | `link_url = "http://x.com"` | `422` |
| AST-44 | `share01@design` สร้าง banner แล้ว share01 ลบ | สำเร็จ |
| AST-44 | share01 ลบ banner ของ agent01 | `405404` |
| AST-45 | ช่อง asset ชื่อ `site` (สมมุติ) | อยู่ใน `assets.site` ไม่ชนกับ `site` ของ branding |
| AST-46 | เกมที่ไม่มีไฟล์เลยทั้ง GLOBAL และเว็บ | สำเร็จ · `assets = {}` |
| AST-46 | อ่าน `site_brandings` error | สำเร็จ · `site` ทุก field `""` · log warn |
| AST-47 | agent01 ไม่มี branding · share01 มี | `bundle` ของ agent01 ได้ `site` ทั้งแถวของ share01 |
| AST-48 | Member ใต้ share01 เรียก bundle ฝั่งผู้เล่น (เมื่อมี module หน้าบ้าน) | `data` เท่ากับ preview `POST /bo/pr/asset/bundle` ของ share01 |
| AST-49 | `S3_CDN_BASE_URL = https://cdn.example.com/` | URL ไม่มี `//` ซ้อน |
| AST-50 | `share01@design` อัป แล้ว share01 แก้ theme | config `created_by = share01@design` · theme `updated_by = share01` |

## 8. Contract changes (แจ้ง frontend)

- **path / method ใหม่ทั้งหมด** (หัวข้อ 5) · ไม่มี `DELETE` · ไม่มี `?agent_id=` และ `me` — ส่ง `scope` + `agent_id` ใน body / form
- `GET /games` และ `/asset-types` ต้อง login และมีสิทธิ์ `asset` (เดิมเป็น public) · เปลี่ยนชื่อ `game_type` → `game_code`
- อ่านค่ากลาง (GLOBAL) ต้องมีสิทธิ์ `asset` ≥ `view` (เดิมใครที่ login ก็อ่านได้)
- ชื่อช่องต้องตรง catalog ทุกตัวอักษร (เดิมระบบแปลงตัวพิมพ์ / ช่องว่างให้)
- ชนิดไฟล์ตรวจจากเนื้อไฟล์ (ไฟล์นามสกุลปลอมถูกปฏิเสธ)
- bundle: `data` เปลี่ยนจาก map แบน (+ `site` ปนอยู่) เป็น `{ source, assets, theme, site }` · ไม่ใช้ header `X-Asset-Source` ·
  ไม่มีไฟล์ = `assets: {}` (เดิม code แยก / 404)
- สีใน response เป็นตัวพิมพ์เล็กเสมอ
- banner: เพิ่ม `update-status` · ลบได้โดยทุกคนที่มีสิทธิ์ของเว็บนั้น
- **error code ชุดใหม่ทั้งหมด** (ไม่ใช้ `1004xx`, `2000xx`, `2600xx` ของระบบเก่า) — ดูหัวข้อ 9 ข้อ 6

## 9. การตัดสินใจ

### ❓ รอ lead

1. **ใครคือ "เว็บ" และชั้นบนเข้าถึงได้แค่ไหน** — เสนอ: เว็บ = ประเภทที่สร้าง Member ได้ (AST-02) · ชั้นบน**ดู** asset /
   theme / branding ของเว็บในสายล่างได้ แต่**แก้ไม่ได้** (โค้ดเก่าแก้ได้เฉพาะตัวเอง ส่วนเอกสารสรุปบอกว่าบรรพบุรุษแก้ได้)
   → ถ้าอนุมัติ ต้องแก้ AST-05 ให้ชั้นบนอ่านได้ (`402402` เฉพาะนอกสาย)
2. **Member ใต้ Agent ซ้อนชั้นใช้หน้าตาของใคร** — โค้ดเก่า: เว็บของผู้สร้างตรงของ Member → GLOBAL (ไม่ไล่สาย) แต่
   branding ไล่สาย · ปัญหา: Share B2C ตั้งหน้าตาไว้ แต่ Member ใต้ Agent ของ Share นั้นได้ค่ากลาง ·
   เสนอ: asset ไล่จากผู้สร้างตรงขึ้นไป**ทีละช่อง** จนเจอเว็บแรกที่มี → GLOBAL · theme / branding ไล่หา**ทั้งชุด**ของชั้นแรกที่มี (AST-47)
3. **branding / banner ใช้เมนูสิทธิ์ไหน** — โค้ดเก่าใช้ `announcement` · MGMT-52: Superadmin ไม่มี `announcement` ·
   เสนอ: branding ใช้ `asset` · banner ใช้ `announcement`
4. **upload แบบ multipart ระบุเป้าหมายทาง form field หรือ query** — เอกสารสรุปเสนอ query (ไม่ต้อง parse body ใหญ่ก่อนรู้สิทธิ์) ·
   แต่ fasthttp อ่าน body ทั้งก้อน (ตาม `BodyLimit`) ก่อนถึง handler อยู่แล้ว ข้อได้เปรียบจึงน้อย ·
   เสนอ: form field เพื่อให้ตรง convention (id ใน body)
5. **โลโก้ / favicon ที่ถูกเขียนทับ** — โค้ดเก่าไม่ลบไฟล์เดิม · เสนอ: ลบไฟล์เดิมหลังบันทึกแถวสำเร็จ ลบไม่ได้แค่ log ·
   **ประวัติการแก้**: เสนอเพิ่ม `asset_change_logs` แบบ MGMT-60 (ใคร / ทำอะไร / ค่าเก่า → ใหม่ / ip / request_id) — ตอนนี้มีแค่
   `created_by` / `updated_by` (AST-50)
6. **error code** — เสนอ module id `05` (ตรวจ 2026-10-08: `dev` จอง 01–03 · branch `game-rate` จอง 04 ·
   branch `currency-rate` ยังไม่จอง — ต้องเช็คอีกครั้งตอนจองจริง) · ข้อความ th/en ตามกฎ `ERROR_CODES.md`

| Code | HTTP | ความหมาย |
|---|---|---|
| 405101 | 200 | ขนาดภาพไม่ตรงกับที่ช่องนี้รับ (msg บอกขนาดที่รับ) |
| 405102 | 200 | ชนิดไฟล์ไม่รองรับ |
| 405103 | 200 | ไฟล์ใหญ่เกิน 10 MB |
| 405301 | 200 | ไม่มีสิทธิ์จัดการค่ากลาง |
| 405302 | 200 | บัญชีนี้ตั้งหน้าตาของเว็บไม่ได้ |
| 405303 | 200 | ลบค่ากลางได้เฉพาะผู้อัปโหลด |
| 405304 | 200 | ช่องนี้เลิกใช้แล้ว |
| 405401 | 200 | ไม่พบเกมหรือช่อง asset |
| 405402 | 200 | ไม่พบ asset (ตอบเหมือนกันทุกกรณี — AST-23, AST-31) |
| 405403 | 200 | asset ยังถูกเลือกใช้อยู่ (msg 2 แบบ — AST-32) |
| 405404 | 200 | ไม่พบ banner |
| 505801 | 503 | ที่เก็บไฟล์ยังไม่พร้อม |
| 505802 | 502 | อัปโหลดไปที่เก็บไฟล์ไม่สำเร็จ |

   validation ทั่วไปใช้ `422` + msg บอก field · สิทธิ์ใช้ code ของ module ② (`402303`, `402402`)
7. **ข้อมูลที่ต้องให้ทีมเกม / หน้าบ้านยืนยัน** — รายชื่อช่องและขนาดใน seed (มาจากโค้ดเก่า — `coin_toss` /
   `rock_paper_scissors` ยังไม่มีสเปกขนาด) · ไฟล์ palette จริงของหน้าบ้านสำหรับ fixture ของ AST-39 · ชื่อ / รูปตัวอย่างของ `site_themes`

### ต้องขออนุมัติก่อน implement (แตะส่วนกลางของโปรเจกต์)

| เรื่อง | ที่ต้องแก้ |
|---|---|
| dependency S3 | `go.mod` เพิ่ม AWS SDK v2 (S3) |
| config | `pkg/configs/config.go` + `.env.example`: `S3_REGION` · `S3_BUCKET` · `S3_CDN_BASE_URL` · `S3_UPLOAD_TIMEOUT` (credential จาก AWS default chain — ไม่เก็บ key ใน env) · storage ไม่พร้อมต้อง boot ได้ (AST-30) |
| storage client | package ใหม่ `platform/storage` |
| ขนาด body | `pkg/configs/fiber_config.go` ตั้ง `BodyLimit` = 10 MB **ทั้ง request** อยู่แล้ว · multipart มี boundary + field อื่นด้วย ไฟล์ใกล้ 10 MB จะโดน fasthttp ตอบ 413 ก่อนถึง handler (ไม่ใช่ `405103`) · เสนอ: เพิ่ม `BodyLimit` เป็น 11 MB (กระทบทุกเส้น) หรือลดเพดานไฟล์ใน AST-16 ลงให้ request ทั้งก้อนไม่เกิน 10 MB |
| multipart | helper อ่าน multipart + `Validate()` ใน `pkg/utils` (ตอนนี้มีแค่ `ParseBody` / `ParseQuery` แบบ JSON) |
| middleware | `ResolveAssetTarget` / `RequireAssetTarget` ใน `app/internals/backoffice/middleware` · ใช้ `RequirePermission` ของ module ② |
| เอกสารกลาง | จอง module id ใน `docs/ERROR_CODES.md` · เพิ่มแถว `asset` ใน `docs/MIGRATION_STATUS.md` · ไฟล์สถานะของ branch `docs/lead/carl-asset.md` + แถวใน `docs/lead/README.md` (1 ไฟล์ต่อ branch) · หลังอนุมัติ: `docs/frontend/asset.md` + แถวใน `docs/frontend/README.md` (เอกสารหน้าบ้านทำจาก spec ที่อนุมัติแล้วเท่านั้น) |
| migration | ไฟล์ใหม่ต้องมี timestamp หลัง `20261008160000_account_api_key_plain.sql` (ล่าสุดตอนตรวจ) |
