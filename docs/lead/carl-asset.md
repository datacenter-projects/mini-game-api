# สถานะ — `carl/asset`

## 1. ภาพรวม

| | |
|---|---|
| branch | `carl/asset` (แตกจาก `dev` @ `65300f1`) · merge `dev` แล้ว (ล่าสุด `67ee41a`) |
| spec | `docs/modules/asset.md` — **APPROVED** (lead zerph 2026-10-14 หลังแก้ A2) · rule `AST-01`–`AST-61` · คำถามที่ยังเปิด 4 ข้อไม่ขวางการ merge |
| code | **implement แล้ว (MVP)** — 16 เส้น BO (GN1–14, GN16, GN22) · lead ตรวจโค้ด 2026-10-14 (`b29a7ff`): ไม่มี must-fix · **แก้ A1 (เพดาน 4096 px) + A2 (อ่าน header ไม่ได้ = `405102`) แล้ว** · **lead 2026-10-16:** BO ใช้บัญชีจาก token (ไม่รับ `agent_id` · ดูได้แค่ของตัวเอง) + เส้นผู้เล่น GN23 `POST /player/pr/asset/bundle` — implement แล้ว · ยังไม่ commit |
| merge | ยังไม่ merge เข้า `dev` · `carl/asset` คือ branch ของ module asset (S13) · ขั้นต่อไปตาม lead: merge `dev` ล่าสุด → เปิด PR → รอ CI (integration) เขียว → แจ้ง lead · **ห้ามแก้ migration `20261012160000` / `20261012170000` (A3)** |
| test | `make check` ผ่าน · integration test ของ asset ผ่านบน test env ในเครื่อง (Postgres 18 + miniredis · `-race` ผ่าน) · ทั้ง suite ผ่าน · ผลที่ยืนยันจริงคือ CI (Postgres 17 + Redis 7) |

## 2. งาน

| ส่วน | สถานะ | หมายเหตุ |
|---|---|---|
| Extract โค้ดเก่า (asset, theme, branding, banner, bundle) | ✅ | ผลอยู่ในหัวข้อ 4 ของ spec (31 ข้อ) |
| spec `docs/modules/asset.md` — แก้ตามรีวิว S1–S11 | ✅ | MVP (GS1) · ไล่สาย `cnf` (GR4) · `use_now` (GR1) · soft delete (GR2) · ≤ 5 MB (GR3) · reset 3 เส้น (GR5) · path GN1–GN23 · error `05` · ลำดับเช็ค 7.1 |
| spec — แก้ตามรอบตรวจ 1 (AV1–AV4 · GE) | ✅ | Share Master ใช้ของ CSM (AV1) · `inherited_from` (AV3) · สคริปต์อัปรูป GLOBAL (AV4 · AST-61) · `games/list` ไล่สาย + ไม่กรอง `status_game` (GE1) · ตัด `description` (GE2-1) · `used/list` ส่งทุกช่อง (GE6-1) · theme ค่าเดิม = no-op (GE10-1) · `site_name` trim + ห้ามอักขระควบคุม (GE13) |
| implement (MVP) | ✅ lead ตรวจแล้ว (A1 / A2 แก้แล้ว) | migration `20261012160000_asset_create_tables` · `app/core/asset` (ไล่สาย · shades · ธีม · ตรวจไฟล์) · repository / service / controller / middleware · `platform/storage` (AWS SDK v2) + `S3_*` · `utils.ReadFormFile` · test: core table test + dto + integration 13 กลุ่ม |
| `scripts/seed_asset_global` (AST-61) | ✅ โค้ดเสร็จ | อัปรูป GLOBAL จากโฟลเดอร์ `{game_code}/{asset_type}.{ext}` ผ่าน service เดียวกับ API · รันเองทีละ env (ไม่มี Job ตอน deploy · ไม่ใส่ image — lead 2026-10-15) · ขอ lead ก่อนรัน |
| รูป `scratch_card` 19 ช่องใน repo | ✅ PR `feature/asset-seed-images` (รอ lead merge) | `seed/global-assets/scratch_card/` (WebP 5.1 MB) · `TestSeedGlobalRepoImages` ตรวจทุกไฟล์ใน CI · `make seed-asset-global` · DEPLOYMENT.md เขียนวิธีรันเอง · ไม่แตะ Dockerfile |
| MinIO ใน docker-compose (GQ12) | ✅ เอาออกแล้ว · lead รับ (A4) | MinIO เลิกแจกไบนารีและ image (`dl.min.io` 410 · Docker Hub ไม่พบ `minio/minio`, `minio/mc`) — ถ้าปล่อยไว้ `docker compose up` ของทุกคนจะ pull ไม่ได้ · ใช้ `scripts/fakes3` แทน (docs/TESTING.md) · integration test ใช้ `testutil.NewFakeS3` เหมือนเดิม |
| `scripts/fakes3` (S3 จำลองในเครื่อง) | ✅ ลองกับ `platform/storage` จริงแล้ว | `go run ./scripts/fakes3` → `http://127.0.0.1:9000` · bucket `minigame-asset-local` · เก็บที่ `.testenv/fakes3` · ตั้ง `S3_ENDPOINT` / `S3_BUCKET` / `S3_CDN_BASE_URL` + `AWS_*` (ค่าอะไรก็ได้) · ใช้ทดสอบเท่านั้น (ไม่ตรวจลายเซ็น) · ยังไม่ได้ลองอัปผ่าน API จริง |
| เลขเวอร์ชัน migration | ✅ แก้แล้ว | `dev` มี `20261012120000_member_management_payment` → ของ asset เปลี่ยนเป็น `20261012160000` · ตาราง asset ยังไม่ถูกสร้างบน dev DB จนกว่าจะรัน `migrate up` (dev/prod: Job ก่อน rollout ตาม DEPLOYMENT.md — ต้องยืนยันกับ infra ว่ามี step นี้) · ตอน merge `dev` ให้เช็คเลขซ้ำอีกครั้ง |
| `docs/MIGRATION_STATUS.md` เพิ่มแถว `asset` (S12) | ✅ | สถานะ IM · spec APPROVED |
| เรื่อง 1 — BO ใช้บัญชีจาก token (lead 2026-10-16) | ✅ รอ PR | `ResolveTarget` ไม่รับ `agent_id` (`422`) · `RequireAssetTarget` เติมบัญชี + สายจาก token · `Authorize` ไม่มี relation · `402402` ไม่เกิดใน asset · spec AST-04/05/07/08/09 · 7.1 · หัวข้อ 9 · เอกสารหน้าบ้าน (breaking change) · integration test แก้ทุกกลุ่ม |
| เรื่อง 2 — เส้นผู้เล่น GN23 (lead 2026-10-16) | ✅ รอ PR | `POST /api/v1/player/pr/asset/bundle` · `buildBundle` ตัวเดียวกับ preview · เว็บ = `Player.AgentID` · เมิน `agent_id` (AST-52) · เกมปิดตอบปกติ · ไม่มี cache · `TestPlayerAssetBundle` · เอกสารหน้าบ้าน 2.17 |
| อัปรูป GLOBAL ให้ครบทุกช่อง | ⏳ | `scratch_card` มีไฟล์ครบใน repo (รอรัน — รอ S3 / CloudFront ของ dev + lead บอกว่าใครรัน) · `coin_toss` / `rock_paper_scissors` รอรูปจากทีมเกม · ต้องเสร็จก่อน UAT (AST-24 — ไม่มี fallback ไฟล์ล่าสุดแล้ว) · ต้องมี infra + ไฟล์รูปก่อน |
| infra (S3 / CloudFront / IAM / ConfigMap) | ⏳ | lead สั่งแล้ว 2026-10-12 · ต้องมีก่อน deploy |
| เอกสารหน้าบ้าน `docs/frontend/asset.md` | ✅ | `b29a7ff` · ตัวอย่าง / error ทุกตัวยิงจากโค้ดจริง · แก้ตาม A1 / A2 แล้ว |
| ผลตรวจโค้ด lead (A1–A4 · 2026-10-14) | ✅ | A1 เพดาน 4096 px ทุกเส้น upload (`405101`) · A2 อ่าน header ไม่ได้ = `405102` (แก้ AST-17 / AST-43) · test ใหม่ `TestAssetImageLimits` + core table test · A3 / A4 รับทราบ |

## 3. รอ lead

คำถามที่ยังเปิด (ไม่ขวางการ merge — รายละเอียดหัวข้อ 9 ของ spec):

1. **AST-61** — รูปแบบโฟลเดอร์ไฟล์ที่ทีมเกมส่ง (`{game_code}/{asset_type}.{ext}`) · `-force` · `created_by = script:<user>`
2. **รายชื่อช่อง / ขนาด / palette** (GQ7) — รอทีมเกม / หน้าบ้านยืนยัน

lead รับแล้ว (A4 · 2026-10-14): GE10-1 · ลำดับสิทธิ์ต่อเป้าหมายก่อน body · MinIO → fakes3

lead ตัดสินแล้ว (2026-10-16): BO ใช้บัญชีจาก token · เส้นผู้เล่น GN23 (`game_code` ใน body · ไม่มี `inherited_from` · ไม่มี cache)

ต้องแจ้ง lead ก่อน merge: BO ที่ยังส่ง `agent_id` ได้ `422` (**ไม่เมิน**) — กันหน้าจอเดิมแก้ของตัวเองโดยไม่รู้ตัว · ฝั่งผู้เล่นเมิน (AST-52)

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
| 2026-10-12 | `a2b94e6` | รีวิวโค้ดเอง + แก้: ลบไฟล์ S3 ด้วย ctx ที่ไม่ถูกยกเลิก · กู้แถวธีม GLOBAL ได้ · `created_at` จาก DB · reset timeout ของ storage · seed (โฟลเดอร์อ่านไม่ได้ / ไฟล์ซ้ำช่อง) · `utils.CheckNoNull` · กรอง config ที่ลบแล้ว · `games/list` query เดียว (+ migration `20261012170000` index) · ส่งสายเป้าหมายจาก middleware · ล็อก config scope เจ้าของ · fakes3 ไม่ใช้ goroutine · spec 7.1 สิทธิ์ต่อเป้าหมายก่อน body (schneider เลือก) · test ใหม่ |
| 2026-10-12 | `5669728` | แก้ spec ตาม AV1–AV4 + GE · **implement MVP** (migration `20261012160000` · core · repository · service · controller · middleware · `platform/storage` · `S3_*` · script อัปรูป GLOBAL · `scripts/fakes3` · test · MinIO compose · MIGRATION_STATUS · ERROR_CODES) |
| 2026-10-12 | `67ee41a` · `efb61f3` | merge `dev` (แก้ conflict `json_null.go` — `FieldLabel` อยู่ใน `utils.CheckNoNull`) · เอา MinIO ออกจาก compose → `scripts/fakes3` · เพิ่ม fakes3 ใน TESTING.md · `migrate up` บน dev DB (index `20261012170000`) |
| 2026-10-14 | `b29a7ff` | เอกสารหน้าบ้าน `docs/frontend/asset.md` (16 เส้น · request / response / error ไทย-อังกฤษ ยิงจากโค้ดจริง) |
| 2026-10-14 | `1e5eb36` | แก้ตามผลตรวจโค้ด lead: A1 เพดาน 4096 px · A2 อ่าน header ไม่ได้ = `405102` · AST-17 / AST-43 · test · เอกสารหน้าบ้าน · **spec APPROVED** · ปิดคำถามที่ lead รับแล้ว (A4) |
| 2026-10-15 | — (PR `feature/asset-seed-images`) | เก็บรูป GLOBAL ใน repo ตาม lead: `seed/global-assets/` · `TestSeedGlobalRepoImages` · `make seed-asset-global` · DEPLOYMENT.md วิธีรันเอง · AST-61 · ไม่ใส่ image / ไม่มี Job |
| 2026-10-16 | — (ยังไม่ commit) | lead ตัดสิน 2 เรื่อง: **BO ใช้บัญชีจาก token** (ไม่รับ `agent_id` · ดูได้แค่ของตัวเอง + GLOBAL · `402402` ไม่เกิด) · **เส้นผู้เล่น GN23** `POST /player/pr/asset/bundle` · spec · เอกสารหน้าบ้าน · test |
