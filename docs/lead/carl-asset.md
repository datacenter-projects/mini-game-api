# สถานะ — `carl/asset`

## 1. ภาพรวม

| | |
|---|---|
| branch | `carl/asset` (แตกจาก `dev` @ `65300f1`) · merge `dev` แล้ว (ล่าสุด `0c9f595`) |
| spec | `docs/modules/asset.md` — **DRAFT** · lead ตรวจรอบ 1 แล้ว (ฉบับ `f19a511` → **เริ่มโค้ดได้**) · **แก้ตาม AV1–AV4 + ตาราง GE แล้ว** (rule `AST-01`–`AST-61`) · รอ lead รับทราบฉบับแก้ |
| code | **implement แล้ว (MVP)** — 16 เส้น BO (GN1–14, GN16, GN22) · commit แล้ว (`5669728` · push `origin/carl/asset`) · เส้นผู้เล่น (GN23) รอ Player Auth |
| merge | ยังไม่ merge เข้า `dev` · `carl/asset` คือ branch ของ module asset (S13) · ก่อนเปิด PR ต้อง merge `dev` ล่าสุด — ชน `ParseBodyNoNull` (dev ใช้ `FieldLabel` → ย้ายไปไว้ใน `utils.CheckNoNull`) · เช็คเลข migration ไม่ซ้ำ |
| test | `make check` ผ่าน · integration test ของ asset ผ่านบน test env ในเครื่อง (Postgres 18 + miniredis · `-race` ผ่าน) · ทั้ง suite ผ่าน · ผลที่ยืนยันจริงคือ CI (Postgres 17 + Redis 7) |

## 2. งาน

| ส่วน | สถานะ | หมายเหตุ |
|---|---|---|
| Extract โค้ดเก่า (asset, theme, branding, banner, bundle) | ✅ | ผลอยู่ในหัวข้อ 4 ของ spec (31 ข้อ) |
| spec `docs/modules/asset.md` — แก้ตามรีวิว S1–S11 | ✅ | MVP (GS1) · ไล่สาย `cnf` (GR4) · `use_now` (GR1) · soft delete (GR2) · ≤ 5 MB (GR3) · reset 3 เส้น (GR5) · path GN1–GN23 · error `05` · ลำดับเช็ค 7.1 |
| spec — แก้ตามรอบตรวจ 1 (AV1–AV4 · GE) | ✅ | Share Master ใช้ของ CSM (AV1) · `inherited_from` (AV3) · สคริปต์อัปรูป GLOBAL (AV4 · AST-61) · `games/list` ไล่สาย + ไม่กรอง `status_game` (GE1) · ตัด `description` (GE2-1) · `used/list` ส่งทุกช่อง (GE6-1) · theme ค่าเดิม = no-op (GE10-1) · `site_name` trim + ห้ามอักขระควบคุม (GE13) |
| implement (MVP) | ✅ โค้ดเสร็จ รอ review | migration `20261012160000_asset_create_tables` · `app/core/asset` (ไล่สาย · shades · ธีม · ตรวจไฟล์) · repository / service / controller / middleware · `platform/storage` (AWS SDK v2) + `S3_*` · `utils.ReadFormFile` · test: core table test + dto + integration 13 กลุ่ม |
| `scripts/seed_asset_global` (AST-61) | ✅ โค้ดเสร็จ | อัปรูป GLOBAL จากโฟลเดอร์ `{game_code}/{asset_type}.{ext}` ผ่าน service เดียวกับ API · รอไฟล์รูปจากทีมเกม / ออกแบบ |
| MinIO ใน docker-compose (GQ12) | ✅ เอาออกแล้ว (เปลี่ยนจากที่ lead อนุมัติ) | MinIO เลิกแจกไบนารีและ image (`dl.min.io` 410 · Docker Hub ไม่พบ `minio/minio`, `minio/mc`) — ถ้าปล่อยไว้ `docker compose up` ของทุกคนจะ pull ไม่ได้ · ใช้ `scripts/fakes3` แทน (docs/TESTING.md) · integration test ใช้ `testutil.NewFakeS3` เหมือนเดิม · ขอ lead รับทราบ |
| `scripts/fakes3` (S3 จำลองในเครื่อง) | ✅ ลองกับ `platform/storage` จริงแล้ว | `go run ./scripts/fakes3` → `http://127.0.0.1:9000` · bucket `minigame-asset-local` · เก็บที่ `.testenv/fakes3` · ตั้ง `S3_ENDPOINT` / `S3_BUCKET` / `S3_CDN_BASE_URL` + `AWS_*` (ค่าอะไรก็ได้) · ใช้ทดสอบเท่านั้น (ไม่ตรวจลายเซ็น) · ยังไม่ได้ลองอัปผ่าน API จริง |
| เลขเวอร์ชัน migration | ✅ แก้แล้ว | `dev` มี `20261012120000_member_management_payment` → ของ asset เปลี่ยนเป็น `20261012160000` · ตาราง asset ยังไม่ถูกสร้างบน dev DB จนกว่าจะรัน `migrate up` (dev/prod: Job ก่อน rollout ตาม DEPLOYMENT.md — ต้องยืนยันกับ infra ว่ามี step นี้) · ตอน merge `dev` ให้เช็คเลขซ้ำอีกครั้ง |
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
7. **เอา MinIO ออกจาก compose** — เปลี่ยนจากที่ lead อนุมัติใน GQ12 เพราะ MinIO เลิกแจก image / ไบนารี · ใช้ `scripts/fakes3` แทน (ไม่ต้องใช้ Docker) · ถ้า lead อยาก S3 จำลองตัวอื่นใน compose (เช่น SeaweedFS / Garage) ค่อยเพิ่มภายหลัง
8. **ลำดับสิทธิ์ต่อเป้าหมาย กับ body** (พบตอนรีวิวโค้ด) — spec เดิมให้ตรวจ body (`422`) ก่อน แต่โค้ดตรวจสิทธิ์ต่อเป้าหมายใน middleware ก่อน controller · schneider เลือก**แก้ spec ให้ตรงโค้ด** (7.1 · หัวข้อ 8 · คำถามข้อ 6 ในหัวข้อ 9) · ต่างจาก agent_management เฉพาะกรณีที่ไม่มีสิทธิ์และ body ผิดพร้อมกัน (ได้ `402402` / `405301` แทน `422`) · กระทบ 12 เส้น · มี test ล็อกลำดับไว้ — ขอรับทราบ

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
