# สถานะ — `carl/asset`

## 1. ภาพรวม

| | |
|---|---|
| branch | `carl/asset` (แตกจาก `dev` @ `65300f1`) · merge `dev` แล้ว (ล่าสุด `0c9f595`) |
| spec | `docs/modules/asset.md` — **DRAFT** · lead ตรวจรอบ 1 แล้ว (ฉบับ `f19a511` → **เริ่มโค้ดได้**) · **แก้ตาม AV1–AV4 + ตาราง GE แล้ว** (rule `AST-01`–`AST-61`) · รอ lead รับทราบฉบับแก้ |
| code | **implement แล้ว (MVP)** — 16 เส้น BO (GN1–14, GN16, GN22) · ยังไม่ commit · เส้นผู้เล่น (GN23) รอ Player Auth |
| merge | ยังไม่ merge เข้า `dev` · ต้องแยก branch ใหม่จาก `dev` ล่าสุดตาม S13 (ผู้ใช้จัดการ git) |
| test | `make check` ผ่าน · integration test ของ asset ผ่านบน test env ในเครื่อง (Postgres 18 + miniredis · `-race` ผ่าน) · ทั้ง suite ผ่าน · ผลที่ยืนยันจริงคือ CI (Postgres 17 + Redis 7) |

## 2. งาน

| ส่วน | สถานะ | หมายเหตุ |
|---|---|---|
| Extract โค้ดเก่า (asset, theme, branding, banner, bundle) | ✅ | ผลอยู่ในหัวข้อ 4 ของ spec (31 ข้อ) |
| spec `docs/modules/asset.md` — แก้ตามรีวิว S1–S11 | ✅ | MVP (GS1) · ไล่สาย `cnf` (GR4) · `use_now` (GR1) · soft delete (GR2) · ≤ 5 MB (GR3) · reset 3 เส้น (GR5) · path GN1–GN23 · error `05` · ลำดับเช็ค 7.1 |
| spec — แก้ตามรอบตรวจ 1 (AV1–AV4 · GE) | ✅ | Share Master ใช้ของ CSM (AV1) · `inherited_from` (AV3) · สคริปต์อัปรูป GLOBAL (AV4 · AST-61) · `games/list` ไล่สาย + ไม่กรอง `status_game` (GE1) · ตัด `description` (GE2-1) · `used/list` ส่งทุกช่อง (GE6-1) · theme ค่าเดิม = no-op (GE10-1) · `site_name` trim + ห้ามอักขระควบคุม (GE13) |
| implement (MVP) | ✅ โค้ดเสร็จ รอ review | migration `20261012160000_asset_create_tables` · `app/core/asset` (ไล่สาย · shades · ธีม · ตรวจไฟล์) · repository / service / controller / middleware · `platform/storage` (AWS SDK v2) + `S3_*` · `utils.ReadFormFile` · test: core table test + dto + integration 13 กลุ่ม |
| `scripts/seed_asset_global` (AST-61) | ✅ โค้ดเสร็จ | อัปรูป GLOBAL จากโฟลเดอร์ `{game_code}/{asset_type}.{ext}` ผ่าน service เดียวกับ API · รอไฟล์รูปจากทีมเกม / ออกแบบ |
| MinIO ใน docker-compose (GQ12) | ⏳ เขียนแล้ว ยังไม่ได้ลองรัน | เครื่องที่ทำไม่มี Docker · integration test ใช้ S3 จำลองในตัว (`testutil.NewFakeS3`) |
| `docs/MIGRATION_STATUS.md` เพิ่มแถว `asset` (S12) | ✅ | สถานะ IM |
| อัปรูป GLOBAL ให้ครบทุกช่อง | ⏳ | ต้องเสร็จก่อน UAT (AST-24 — ไม่มี fallback ไฟล์ล่าสุดแล้ว) · ต้องมี infra + ไฟล์รูปก่อน |
| infra (S3 / CloudFront / IAM / ConfigMap) | ⏳ | lead สั่งแล้ว 2026-10-12 · ต้องมีก่อน deploy |
| เอกสารหน้าบ้าน `docs/frontend/asset.md` | ⏳ | ทำหลัง spec อนุมัติ |

## 3. รอ lead

คำถามเล็ก (ไม่ขวางการ review โค้ด — รายละเอียดหัวข้อ 9 ของ spec):

1. **GE10-1** — "ส่งค่าเดิมทั้งชุด" เทียบกับค่าที่ใช้อยู่จริง (ไล่สายแล้ว) หรือเทียบกับแถวของตัวเองเท่านั้น (โค้ดทำตามแบบแรก: บัญชีที่ยังไม่เคยตั้งกดบันทึกโดยไม่แก้ = ไม่สร้างแถว)
2. **`theme` ใน bundle ไม่มี `inherited_from`** (ไม่เปิด username ชั้นบนให้ผู้เล่น) — ยืนยัน
3. **AST-61** — รูปแบบโฟลเดอร์ไฟล์ที่ทีมเกมส่ง (`{game_code}/{asset_type}.{ext}`) · `-force` · `created_by = script:<user>`
4. **GN23 / lobby** — `game_code` ใน body · ใครทำเส้นผู้เล่น (GE22-2) ตัดสินพร้อม review Player Auth
5. **รายชื่อช่อง / ขนาด / palette** (GQ7) — รอทีมเกม / หน้าบ้านยืนยัน (ไม่ตอบ ~3 วัน seed ตามร่าง)
6. **ลำดับตรวจของ `used/update`** — โค้ดตรวจความเป็นเจ้าของ config ก่อนบอกว่าช่องเลิกใช้ (`405402` ก่อน `405304`) เพื่อไม่เปิดเผยสถานะของ config ที่ไม่ใช่ของตัวเอง — spec หัวข้อ 7.1 แก้ตามนี้แล้ว

## 4. ประวัติอัปเดต

| วันที่ | commit | อัปเดต |
|---|---|---|
| 2026-10-08 | `9e95aa8` | ร่าง spec `asset.md` (DRAFT) จากเอกสารสรุประบบเดิม + extract โค้ดเก่า |
| 2026-10-08 | `91bacde` · `ceb3ac9` | merge `dev` (module ② · account · reset script) |
| 2026-10-08 | `78da326` | ตรวจ spec กับ `dev` ล่าสุด: แบ่งหน้า · ลิงก์ spec · เพิ่มไฟล์สถานะนี้ |
| 2026-10-09 | `ea76f89` | ปรับตามเอกสารสรุปฉบับแก้: branding ไม่มี GLOBAL · รูปปก `game_cover` · ผู้เล่นอ่านอย่างเดียว · คำถามข้อ 8–10 |
| 2026-10-12 | `1dd674f` | lead review Game_Asset — MVP, ไล่สาย, lobby (GR6), GQ1–GQ13 + รายการแก้ + infra |
| 2026-10-12 | `f19a511` | แก้ spec ตามรีวิว S1–S11: MVP · ไล่สาย (`cnf`) · `use_now` · soft delete · ≤ 5 MB · reset 3 เส้น · path GN1–GN23 · error `05` · ลำดับเช็ค · test case ใหม่ |
| 2026-10-12 | `0ba21e8` | lead ตรวจ spec รอบ 1 — AV1–AV4 + ตาราง GE · เริ่มโค้ดได้ |
| 2026-10-12 | — (รอ commit) | แก้ spec ตาม AV1–AV4 + GE · **implement MVP** (migration · core · repository · service · controller · middleware · `platform/storage` · `S3_*` · script อัปรูป GLOBAL · test · MinIO compose · MIGRATION_STATUS · ERROR_CODES) |
