# PR review — module `asset` (`carl/asset` → `dev`)

ผู้เปิด PR: schneider · spec: [docs/modules/asset.md](../modules/asset.md) (**APPROVED** lead zerph 2026-10-14) ·
ผลตรวจโค้ดของ lead: [docs/reviews/game-asset.md](../reviews/game-asset.md) หัวข้อ "ผลตรวจโค้ด" · สถานะ branch: [carl-asset.md](carl-asset.md)

## 1. สรุป

module `asset` ฝั่งหลังบ้าน — ให้แต่ละบัญชีตั้ง**หน้าตาเกม**ที่ผู้เล่นของตัวเองเห็น: รูปต่อช่อง · ธีม (ฟอนต์ / สี) · branding (ชื่อเว็บ + โลโก้)
16 เส้นใต้ `/api/v1/bo/pr/asset/` (GN1–14, GN16, GN22) · เส้นผู้เล่น (GN23) รอ Player Auth

| กลุ่ม | เส้น | ทำอะไร |
|---|---|---|
| catalog | `games/list` · `slots/list` | รายชื่อเกม + ปก · ช่องรูป + ขนาดที่รับ |
| รูป | `files/upload` · `files/list` · `files/delete` | อัป (multipart) · คลังรูป (แบ่งหน้า) · ลบ (soft delete) |
| รูปที่ใช้ | `used/list` · `used/update` · `used/reset` | รูปที่ใช้จริงทุกช่อง · เลือกจากคลัง · กลับไปใช้ของชั้นบน |
| ธีม | `theme/detail/get` · `theme/detail/update` · `theme/reset` | ต่อเกม · 5 ค่า + shades 19 ระดับ |
| branding | `branding/detail/get` · `branding/detail/update` · `branding/logo/upload` · `branding/reset` | ชื่อเว็บ + โลโก้ (ต่อบัญชี) |
| preview | `bundle/detail/get` | asset + ธีม + branding ที่ผู้เล่นของบัญชีนี้จะเห็น |

**กฎหลักที่ควรเข้าใจก่อน review**
- **ไล่สาย** (`cnf`): ค่าที่ใช้จริง = ตั้งเอง → ชั้นบนใกล้สุด → ค่ากลาง (GLOBAL) · รูปไล่ทีละช่อง · ธีม / branding ไล่ทั้งชุด · branding ไม่มี GLOBAL
- **สิทธิ์ต่อเป้าหมาย** (`scope` + `agent_id` ใน body): GLOBAL เขียนได้เฉพาะ Superadmin · ตัวเองแก้ได้ · สายล่างดูได้ แก้ไม่ได้ (`405301`) · นอกสาย `402402` · **Share Master แก้ไม่ได้ ใช้ของ CSM**
- **ลำดับเช็ค (7.1):** สิทธิ์ต่อเป้าหมายตรวจใน middleware **ก่อน** body (lead รับแล้ว — ต่างจาก agent_management)
- **ไฟล์รูป:** ≤ 5 MB · png / jpeg / gif / webp ตรวจจากเนื้อไฟล์ · อ่าน header ไม่ได้ = `405102` · ≤ 4096 px · ตรวจครบ**ก่อน**อัปขึ้น S3

## 2. แนะนำลำดับการอ่าน

| # | ไฟล์ / โฟลเดอร์ | ดูอะไร |
|---|---|---|
| 1 | `app/internals/backoffice/routes.go` (ส่วน asset) | 16 เส้น · middleware ต่อเส้น (กฎข้อ 28) |
| 2 | `app/internals/backoffice/middleware/asset_middleware.go` | อ่าน `scope` / `agent_id` · ตรวจสิทธิ์ต่อเป้าหมาย · ส่งสาย (`Chain`) ต่อให้ service |
| 3 | `app/core/asset/` | กฎล้วน + table test: สิทธิ์ (`scope_rule`) · ไล่สาย (`inherit_rule`) · ตรวจไฟล์ (`file_rule`) · ธีม / shades · storage key |
| 4 | `app/service/asset/` | transaction · ลำดับตรวจ upload → S3 → DB · ลบ object เมื่อ DB ล้ม · lock config ก่อนลบ / เลือกใช้ |
| 5 | `app/repository/postgres/asset/` | query scope ด้วยเจ้าของ · ไล่สายใน query เดียว |
| 6 | `database/migrations/20261012160000_*` · `20261012170000_*` | 5 ตาราง + index + seed catalog / ธีมกลาง (**รันบน dev DB แล้ว — ห้ามแก้ ตาม lead A3**) |
| 7 | `platform/storage/` | client S3 (AWS SDK v2) · boot ได้แม้ storage ไม่พร้อม (`505801`) |
| 8 | `app/internals/backoffice/asset_*_integration_test.go` | test ต่อกฎ (อ้าง rule id `AST-xx` ในชื่อ / comment) |

## 3. จุดที่อยากให้ช่วยดูเป็นพิเศษ

1. **สิทธิ์ต่อเป้าหมาย** — `core.Authorize` + `CheckAssetTargetService`: มีทางไหนที่ทำให้แก้ของบัญชีอื่น / GLOBAL ได้ไหม · เป้าหมายที่ resolve ไม่ได้ต้องถูกปฏิเสธเสมอ (fail-closed — AST-09)
2. **ความเป็นเจ้าของ config** — `used/update` / `files/delete` scope ด้วยเจ้าของใน query + เช็คซ้ำใน service · ตอบ `405402` เหมือนกันทุกกรณี (กันไล่เดา id)
3. **แข่งกัน `files/delete` vs `used/update`** — lock แถว config (`FOR UPDATE`) ทั้งสองฝั่ง · มี test แข่งกัน
4. **upload → S3 → DB** — S3 สำเร็จแต่ DB ล้ม: ลบ object ด้วย ctx ที่ไม่ถูกยกเลิก · ลบไม่ได้ = log key ไว้
5. **ตรวจไฟล์** — `core.DetectImage` อ่านแค่ header (ไม่ถอดรหัสทั้งภาพ) · เพดาน 4096 px · fail-closed
6. **การไล่สาย** — `core.ResolveUsed` / `ResolveTheme` / `ResolveBranding` · Share Master ได้ค่าจาก CSM
7. **ชื่อ / โครงตาม CLAUDE.md** — layer · ไม่มี business logic ใน controller · error ประกาศใน `apperr` · ไม่มี interface เผื่อ mock

## 4. ส่วนกลางที่ PR นี้แตะ

| ไฟล์ | เปลี่ยนอะไร |
|---|---|
| `pkg/configs/config.go` · `.env.example` | `S3_REGION` · `S3_BUCKET` · `S3_CDN_BASE_URL` · `S3_UPLOAD_TIMEOUT` · `S3_ENDPOINT` |
| `go.mod` / `go.sum` | AWS SDK v2 (s3) |
| `pkg/utils/multipart.go` | `ReadFormFile` (อ่านไฟล์จาก multipart จำกัดขนาด) |
| `pkg/utils/json_null.go` | แยก `CheckNoNull` ให้ `ParseBodyNoNull` และ middleware asset ใช้ร่วมกัน (ข้อความเดิม `FieldLabel`) |
| `pkg/apperr/asset_errors.go` · `docs/ERROR_CODES.md` | error module `05` |
| `pkg/testutil/` | `NewFakeS3` (S3 จำลองใน integration test) · seed ธีมกลางหลังล้างตาราง |
| `application.go` | `storage.Init` ตอน boot |
| `docker-compose*.yml` · `docs/TESTING.md` | เอา MinIO ออก (เลิกแจก image) → `scripts/fakes3` (lead รับ A4) |
| `scripts/seed_asset_global` · `scripts/fakes3` | อัปรูป GLOBAL จากโฟลเดอร์ · S3 จำลองในเครื่อง |

## 5. ลองรันเอง

```bash
make check                                    # structure + build + vet + unit
make test-integration                         # ต้องเปิด test env ก่อน (docs/TESTING.md)
go test -count=1 -tags=integration -run 'TestAsset|TestSeedGlobal' ./app/internals/backoffice/
```

ลองยิงด้วยมือ: `go run ./scripts/fakes3` + ตั้ง `S3_*` ตาม `docs/TESTING.md` · ตัวอย่าง request / response / error ทุกเส้นอยู่ใน
[docs/frontend/asset.md](../frontend/asset.md) (ยิงจากโค้ดจริง)

ผลในเครื่อง (หลัง merge `dev` ล่าสุด `db5bfa1`): `make check` ผ่าน · integration ทั้ง suite ผ่าน (`-race`, Postgres 18 + miniredis) · ผลที่ยืนยันจริงคือ CI (Postgres 17 + Redis 7)

## 6. ที่ยังไม่อยู่ใน PR นี้

- เส้นผู้เล่น `/player/asset/bundle` + lobby — รอ Player Auth
- banner · favicon · `site_theme` · change log — เลื่อน (lead GS1)
- คำถามที่ยังเปิด (ไม่ขวาง merge): `inherited_from` ใน bundle · รูปแบบโฟลเดอร์ของ script อัปรูป · รายชื่อช่อง / ขนาด (รอทีมเกม)

## 7. ก่อน deploy (ไม่ใช่เงื่อนไข merge)

- infra: S3 bucket · CloudFront · IAM (IRSA) · ค่า `S3_*` ใน ConfigMap
- migrate ก่อน rollout (Job ตาม `docs/DEPLOYMENT.md`) — dev DB รันแล้ว
- อัปรูป GLOBAL ให้ครบทุกช่องก่อน UAT (`scripts/seed_asset_global` — ขอ lead ก่อนรันบน env ร่วม)
