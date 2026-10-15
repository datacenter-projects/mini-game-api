# รูปค่ากลาง (GLOBAL) ของ asset

โครง: `{game_code}/{asset_type}.{png|jpg|jpeg|gif|webp}` — ชื่อโฟลเดอร์ = รหัสเกม · ชื่อไฟล์ = ชื่อช่อง (ตัวพิมพ์เล็กตรงตัว) · ช่องละ 1 ไฟล์

- ตอนนี้มี `scratch_card` 19 ช่อง (WebP · ขนาดตรง catalog ใน `database/migrations/20261012160000_asset_create_tables.sql`)
- ไฟล์ละไม่เกิน 5 MB · กว้าง / สูงไม่เกิน 4096 px · ขนาดต้องตรง `allowed_sizes` ของช่อง (ดูจาก `slots/list`)
- เพิ่มรูปเกมอื่น: สร้างโฟลเดอร์ตามรหัสเกม (`coin_toss`, `rock_paper_scissors`) แล้วใส่ไฟล์ตามชื่อช่อง
- ทุกไฟล์ถูกตรวจใน CI ด้วย `TestSeedGlobalRepoImages` (`app/internals/backoffice/asset_seed_integration_test.go`)
- **ไม่ถูกใส่ใน image และไม่มี Job ตอน deploy** — คนรัน `make seed-asset-global` (หรือ `go run ./scripts/seed_asset_global -dir seed/global-assets`) เองทีละ env
  เขียน S3 + DB ตาม `.env` จริง → **ขอ lead ก่อนรันบน dev / uat / prod ทุกครั้ง** (ดู `docs/DEPLOYMENT.md` หัวข้อ "รัน seed รูปค่ากลางของ asset")
