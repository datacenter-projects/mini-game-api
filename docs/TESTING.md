# Testing — mini-game-api

ชนิดของ test และที่อยู่ดู `docs/ARCHITECTURE.md` หัวข้อ 5 · เอกสารนี้คือวิธีเตรียม env และรัน

| ชนิด | ต้องมี | คำสั่ง |
|---|---|---|
| unit | Go | `go test -count=1 ./...` |
| integration | test env (Postgres + Redis) | `go test -count=1 -tags=integration ./...` |
| load | test env + server + k6 | `k6 run tests/load/<script>.js` |

## 1. Test env

แยกจาก env ของ dev: port **55432** (Postgres) / **56379** (Redis) · database `minigame_test` · ค่าทั้งหมดอยู่ใน `.env.test`
(commit ได้ ไม่มีความลับ)

> integration test **ล้างทุกตารางและ FLUSHDB Redis** ทุก test — `pkg/testutil` ไม่ยอมรันถ้า `APP_ENV` เป็น `dev` / `uat` / `prod`
> ห้ามชี้ `.env.test` ไปที่ DB / Redis ที่มีข้อมูลจริงหรือของโปรแกรมอื่น

### เครื่องที่มี Docker

```bash
docker compose -f docker-compose.test.yml up -d     # Postgres 17 + Redis 7 เก็บข้อมูลใน RAM
docker compose -f docker-compose.test.yml down      # ข้อมูลหายหมด
```

หรือ `make test-env-up` / `make test-env-down`

### Windows ที่ใช้ Docker ไม่ได้ (เช่น Windows Server)

ใช้ `scripts/testenv.ps1` — เปิด Postgres แยก instance (ข้อมูลใน `.testenv/`) และ **miniredis** (Redis จำลองที่รองรับคำสั่ง Redis 7)
ต้องมี PostgreSQL binaries ใน PATH หรือตั้ง `$env:PG_BIN` (เช่น `D:\pglocal\pgsql\bin`)

```powershell
powershell -ExecutionPolicy Bypass -File scripts\testenv.ps1 up        # เปิด (สร้างข้อมูลครั้งแรกเอง)
powershell -ExecutionPolicy Bypass -File scripts\testenv.ps1 status
powershell -ExecutionPolicy Bypass -File scripts\testenv.ps1 down      # ปิด ข้อมูลยังอยู่
powershell -ExecutionPolicy Bypass -File scripts\testenv.ps1 reset     # ปิด + ลบข้อมูลทั้งหมด
```

- miniredis ใช้แทน Redis 7 เพื่อทดสอบ logic เท่านั้น — ผลที่ยืนยันได้จริงคือ CI (Postgres 17 + Redis 7 จริง)
- Postgres ของ script เป็นเวอร์ชันที่ติดตั้งในเครื่อง (CI ใช้ 17)

## 2. Integration test

```bash
go test -count=1 -tags=integration ./...
```

- `pkg/testutil` โหลด `.env.test` ที่ root ของ repo เอง — ไม่ต้อง export env · ถ้า environment มี `DB_HOST` อยู่แล้ว (เช่น CI) ไม่โหลดไฟล์เลย · ค่าที่ตั้งไว้แล้วไม่ถูกทับ
- ถ้าต่อ DB / Redis ไม่ได้ ทุก test ได้ error เดียวกัน `testutil: postgres localhost:55432/minigame_test: ...`

## 3. Load test (k6)

สคริปต์อยู่ใน `tests/load/` · ค่าร่วมใน `tests/load/lib/config.js` (override ด้วย `-e KEY=value`)

ขั้นตอน (Windows ไม่มี Docker):

```powershell
powershell -ExecutionPolicy Bypass -File scripts\testenv.ps1 up
powershell -ExecutionPolicy Bypass -File scripts\testenv.ps1 seed -count 100   # บัญชี load0001–load0100
powershell -ExecutionPolicy Bypass -File scripts\testenv.ps1 serve             # server ที่ :8282 (หน้าต่างแยก)
k6 run tests/load/auth_login_logout.js
```

เครื่องที่มี Docker (k6 ไม่ต้องติดตั้ง):

```bash
make test-env-up
set -a && . ./.env.test && set +a && go run ./scripts/seed_loadtest -count 100
set -a && . ./.env.test && set +a && go run .        # server ที่ :8282
docker run --rm -i --network host -v "$PWD/tests/load:/scripts" grafana/k6 run /scripts/auth_login_logout.js
```

(Docker Desktop บน Mac / Windows ใช้ `-e BASE_URL=http://host.docker.internal:8282` แทน `--network host`)

| สคริปต์ | ทดสอบ | หมายเหตุ |
|---|---|---|
| `auth_login_logout.js` | login → logout ไล่ 0 → 50 VU | VU ละบัญชี (1 บัญชี 1 session) · seed บัญชี ≥ จำนวน VU · login ช้าโดยตั้งใจ (bcrypt cost 12) |

- k6 บน Windows: ดาวน์โหลด `k6-vX.Y.Z-windows-amd64.zip` จาก release ทางการ `github.com/grafana/k6` ตรวจ checksum กับ
  `k6-vX.Y.Z-checksums.txt` แล้วแตกไว้ที่ `.testenv/tools/` (ไม่ commit)
- threshold ในสคริปต์เป็นค่าตั้งต้น ปรับหลังได้ผลรอบแรกบนเครื่องที่ใช้วัดจริง

### ผลอ้างอิง (ไม่ใช่ตัวเลข production)

| วันที่ | สคริปต์ | สภาพแวดล้อม | ผล |
|---|---|---|---|
| 2026-10-05 | `auth_login_logout.js` 0→50 VU 2 นาที | EC2 m8i-flex.2xlarge (8 vCPU) · Windows · Postgres 16 · miniredis · k6 v2.3.0 เครื่องเดียวกับ server | 3,774 req ล้มเหลว 0% · login p95 1.31s (med 443ms) · logout p95 124ms (med 1ms) · ~15.7 login/s — คอขวดคือ bcrypt cost 12 (CPU) |
- `.env.test` เปิด `LOGIN_IP_LIMIT_PER_MINUTE` กว้างเพราะทุก VU ยิงจาก IP เดียว — ห้ามใช้ค่านี้นอก test env
- `scripts/seed_loadtest` ไม่ยอมรันถ้า `APP_ENV` เป็น `dev` / `uat` / `prod`

### บัญชี mock ทุกประเภท (`scripts/seed_mock_users`)

สร้างสายทุกประเภทใต้ superadmin ที่มีอยู่ (Company 4 แบบ → Share → Agent → Member + sub `{username}@staff` สิทธิ์ view) ผ่าน service ตัวเดียวกับ API ·
รหัสผ่านพิมพ์ครั้งเดียวใช้ทุกบัญชี · จบแล้วพิมพ์ตาราง username · รันซ้ำได้ (บัญชีที่มีแล้วข้าม) · ไม่รันบน `prod` · รายละเอียดสายดูหัวไฟล์ `main.go`

```bash
APP_ENV=dev go run ./scripts/seed_mock_users -root superadmin1          # mockcomtransfer, mockshareb2b, …
```

บน dev ใช้ `APP_ENV=dev` (ไม่พิมพ์ SQL ที่มี hash ออกจอ) และ `DB_READ_HOST` ต้องชี้ที่เดียวกับ `DB_HOST`

## 4. Index และ query plan

ตรวจว่า query ใช้ index จริงด้วย `EXPLAIN ANALYZE` บน test env หลัง seed ข้อมูลให้มากพอ (ตารางเล็ก Postgres อาจเลือก seq scan)
ผลบน test env บอกได้แค่ว่า index ถูกเลือกไหม ไม่ได้บอกความเร็วของ production
