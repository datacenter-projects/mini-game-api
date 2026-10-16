# Player Auth (ผู้เล่นเข้าระบบ · สาย Transfer) — verify ผ่าน (รอบ 6) · แก้ S2 / S3 / nit แล้ว · **รอ merge ส่วนต่าง + S1 เป็นงานถัดไป**

จาก boiledegg · review: [docs/reviews/player-auth.md](../reviews/player-auth.md) (รอบ 1–4) · spec: [docs/modules/player_auth.md](../modules/player_auth.md) (**APPROVED**) · เอกสารหน้าบ้าน: [docs/frontend/player_auth.md](../frontend/player_auth.md)
branch `boiledegg/player/auth` + `boiledegg/fix/client-ip` **merge เข้า dev แล้ว** (`3c42948`) · ไม่เปิด PR · โค้ดอยู่ใน image `dev-202610160626` (16 ต.ค.) · migration `20261015140000` / `20261015150000` รันบน RDS dev แล้ว · **ใช้จริงบน dev ได้เมื่อตั้ง `TRUSTED_PROXIES` + `PLAYER_LOBBY_URL` (รอ ingress CIDR จาก lead)** · แก้ต่อเนื่องที่ `boiledegg/fix/player-auth-config`

## 1. เสร็จแล้ว

- spec แก้ตามรอบ 2–3 ครบ → lead อนุมัติรอบ 4 (`f50a8b8`) · รับ `407308` · แก้เงื่อนไข R4-2 ใน spec แล้ว (PA-09 win limit)
- **implement ครบ 8 เส้น** (PA1 `player/pb/login` · PA3 `logout` · PA4 `password/update` · PA5 `me/get` · PA6 `balance/get` · PA7 `games/list` · L2 `transfer/login` · TR `transfer/register`) + middleware ผู้เล่น (`typ pl` · `pl:sess:*` · 1 session ต่อคน · เช็คสถานะตัวเอง + สายบนทุก request) + middleware operator (key + IP whitelist · ประเภทผู้สร้าง · สถานะ) + `members/status/update` LOCKED ลบ session หลัง commit
- commit แรกตามที่สั่ง: ย้าย `BearerToken` ไป `pkg/middleware` (`de52ac2` · แยก commit · ไม่เปลี่ยน behavior)
- migration `20261015140000_player_auth_username_length` — `user_members.username` เป็น `VARCHAR(65)` **และแก้ CHECK `ck_user_members_username`** ให้รับ `{a-z0-9 3–32}@{a-z0-9 3–32}` (spec หัวข้อ 6 ไม่ได้เขียนไว้ — เจอตอนเทส: CHECK เดิมรับแค่ `^[a-z0-9]{3,32}$` ผู้เล่น `agent@player` insert ไม่ได้) · up/down/up ผ่านบน DB เปล่า · **ยังไม่ได้รันบน RDS dev**
- error module `07` (`pkg/apperr/player_auth_errors.go` + `docs/ERROR_CODES.md`) · env ใหม่ `PLAYER_SESSION_IDLE_TIMEOUT` / `PLAYER_SESSION_ABSOLUTE_TIMEOUT` / `OPERATOR_IP_LIMIT_PER_MINUTE` / `PLAYER_LOBBY_URL` (`.env.example` + `.env.test` + `docs/DEPLOYMENT.md`)
- ชื่อเกม `name_th` / `name_en` ใน `app/core/game` (มี test กันลืม)
- refactor เล็ก (ไม่เปลี่ยน behavior): `core/agent_auth.PasswordViolationMessage` (ข้อความ AUTH-36 ใช้ร่วม BO + ผู้เล่น) · `agent_management.LoadCreatorByAgentID` · `asset.GameCoversOfAgentService`
- ปิด MGMT-30A / MQ7 ใน `member_management.md`
- test: unit ผ่านทุก package · integration (`player_auth_integration_test.go` + `player_auth_transfer_integration_test.go`) ครอบ PA-03/04/05/07/08/08A/08B/10/11/13/14/20–22/26/30–35/40–42/50/51/54/55/56/60 ผ่านบน test env (Postgres 55432 / Redis 56379) · `go vet` · `gofmt` · `check_structure.sh` ผ่าน

## 2. ติด / ต้องประสาน

- **PA-09 win limit (R4-2) — ครบแล้ว:** lead ออก member-payment รอบ 2.1 (2026-10-15) ว่า `win_limit_per_day` เป็นหน่วยบาทเสมอ ไม่ต้องแปลงสกุล / ไม่ใช้ `currency_rate` (การแปลงอยู่ที่ module เดิมพัน) และให้ Player Auth ใช้ `DefaultWinLimitPerDay` ตัวเดียวกับ `members/create` → ผู้เล่นที่ระบบสร้างได้ 10,000,000 บาท ส่งชัดเจนตอน insert ไม่พึ่ง DEFAULT ของ DB (ฟังก์ชัน `defaultWinLimit()` ใน `operator_service.go`) · ไม่ต้องรอ maofoy แล้ว (โค้ดเขาคงเดิม) · spec PA-09 อัปเดตแล้ว
- ชื่อเกมไทย (สแครชการ์ด / โยนเหรียญ / เป่ายิ้งฉุบ) — lead ยืนยันแล้ว
- **ก่อน deploy dev:** ต้อง migrate RDS (เช็ค `goose_db_version` + เลข `20261015140000` ไม่ชนกับ branch อื่น) และตั้ง `PLAYER_LOBBY_URL` ใน env ของ dev (ไม่ตั้ง = `transfer/login` ตอบ `503`)

## 2.1 ตามเงื่อนไข merge รอบ 6 (verify `0ce5c78` + `a529bcd` ผ่าน · must-fix 0)

- **S2 แก้แล้ว:** ผู้เล่นที่สร้างผ่าน `transfer/register` / สร้างอัตโนมัติใน `transfer/login` เขียน `account_change_logs` CREATE ใน tx เดียวกัน (ผู้ทำ = agent เจ้าของ key · ip = IP ของ operator · ไม่มีรหัสผ่าน / hash) — spec PA-09B + `TestTransferCreateWritesChangeLog`
- **S3 แก้แล้ว:** `TestOperatorSecretsNotLogged` เก็บ log ทุกทาง (zap debug · SQL ของ gorm info พร้อมค่า · request log ของ fiber) แล้วตรวจว่าไม่มี key / password — **ตรวจแล้วว่าถ้าใส่โค้ด log key test ตก** (mutation check)
- **ผล integration test:** `go test -tags=integration ./...` ทุก package ผ่านบน test env (Postgres 55432 / Redis 56379) หลังแก้ · `go vet` · `gofmt` · `check_structure.sh` ผ่าน · ยังไม่ได้รัน `golangci-lint` (รอ CI)
- **nit 7 ข้อ:** ทำแล้ว 3 ข้อที่ระบุในรอบ 6 (error list ของ L2 เพิ่ม `407301` / `503` · PA4 รหัสเดิมผิดครบ → `407303` + ตัด session เขียนใน spec PA-41 / PA-50 · `last_login_ip` ของ L2 = IP ของ operator เขียนใน PA-25) · **อีก 4 ข้อยังไม่ได้ทำ — ไฟล์ `reviews/mini-game-api/verify-player-auth.md` ไม่อยู่ใน repo นี้ ขอรายการ**
- **merge เข้า dev ไปแล้วก่อนได้รับผล verify** (77ba1b7 · ตามคำสั่งผู้ใช้) + migrate RDS dev `20261015140000` แล้ว · **ยังไม่ deploy / ไม่ได้ออก tag** · ส่วนต่างรอบนี้อยู่ใน branch ยังไม่เข้า dev
- **S1 (merge เข้า dev แล้ว `3c42948` · lead ตรวจผ่านรอบ 7):** `middleware.ClientIP` ไล่ `X-Forwarded-For` จากขวาข้าม proxy ใน `TRUSTED_PROXIES` (ไม่ใช้ `EnableIPValidation`) · ใช้แทน `c.IP()` ทุกจุด (operator whitelist / rate limit · player login · BO login · last_login_ip · request log) · `FiberConfig` ไม่ตั้ง `ProxyHeader` แล้ว · test: unit ตาราง 19 กรณี + integration (whitelist หลาย proxy · ปลอม IP ที่ whitelist ไม่ผ่าน · สลับ XFF ไม่หลบ rate limit ทั้ง operator / player / BO) — ตรวจแล้วว่า test ตกเมื่อใช้พฤติกรรมเดิม · **ยังไม่ได้ยิงผ่าน ingress จริงบน dev** (ต้อง deploy ก่อน · ต้องรู้ CIDR ของ ingress ทุกชั้นเพื่อตั้ง `TRUSTED_PROXIES`)

## 2.2 ตามรอบ 7 (verify dev `3c42948` · lead 2026-10-15)

- **S1-a แก้แล้ว** (branch `boiledegg/fix/player-auth-config`): `TRUSTED_PROXIES` validate + parse ครั้งเดียวใน `configs.Load` (ไม่ใช่ IP / CIDR = boot ไม่ขึ้น) · `ClientIP` อ่านจาก `Cfg.TrustedProxyPrefixes` · test `TestParseTrustedProxies` + `TestLoadRejectsBadTrustedProxiesAndLobbyURL`
- **N5 แก้แล้ว:** `PLAYER_LOBBY_URL` ตรวจตอน `configs.Load` (ต้องเป็น absolute http(s) + host) และตรวจซ้ำก่อนสร้างผู้เล่น / ออก session ใน `transfer/login` · `LobbyLink` รับเฉพาะ http(s) · test `TestTransferLobbyURLInvalid` (ผู้เล่นเดิมไม่ถูกเตะ · ผู้เล่นใหม่ไม่ถูกสร้าง)
- **nit doc แก้แล้ว:** spec PA-10 (ClientIP) · สถานะ merge ในไฟล์นี้ · N3: spec เขียน "2 query" ให้ตรงโค้ด
- **ยังไม่ทำ (ตอนว่าง):** N4 (chainOf ซ้ำ / helper `open` ตัวเดียว) · N7 (ย้าย test ไป package กลาง)
- **S1-b:** ยังไม่ได้ยิงผ่าน ingress จริง — รอ deploy dev + CIDR ของ ingress จาก lead
- **ก่อน deploy dev:** migrate `20261015150000` (Maofoy) · ตั้ง `PLAYER_LOBBY_URL` + `TRUSTED_PROXIES` · ออก tag `dev-*` ให้ CI (รวม golangci-lint) เขียว

## 3. คำถามที่รอ lead

| # | เรื่อง | ข้อเสนอ |
|---|---|---|
| 1 | nit ที่เหลือ 4 จาก 7 ข้อ — ไฟล์ verify report ไม่อยู่ใน repo | ขอรายการ (วางในแชท / commit ไฟล์เข้า `docs/reviews/`) |

## 4. ประวัติอัปเดต

- 2026-10-15 · ร่าง spec `player_auth.md` + ส่งคำถาม 13 ข้อให้ lead
- 2026-10-15 · แก้ spec ตาม review รอบ 2–3 (PA6 · name_th/en · register password · snake_case · SUSPENDED/LOCKED ฯลฯ) · สถานะ → รอ approve
- 2026-10-15 · lead อนุมัติ spec (รอบ 4) · แก้ PA-09 win limit ตาม R4-2 · สถานะ → APPROVED · พร้อม implement
- 2026-10-15 · implement 8 เส้น + middleware + migration + test (branch `boiledegg/player/auth`) · win limit ปิดตาม member-payment รอบ 2.1 (บาทเสมอ ไม่แปลงสกุล) · รอสั่งเปิด PR
- 2026-10-15 · ปิดเรื่อง win limit: ตามรอบ 2.1 ใช้ `DefaultWinLimitPerDay` (บาท ไม่แปลงสกุล) · ตัด TODO / พารามิเตอร์สกุลออกจาก `defaultWinLimit()` · merge dev ล่าสุด
- 2026-10-15 · lead อนุมัติ migration แก้ CHECK + ยืนยันชื่อเกม · เปิด verify ที่ `0ce5c78` · รอผล verify ก่อน merge / deploy
- 2026-10-15 · verify ผ่าน (รอบ 6) · แก้ S2 (change log) + S3 (test ไม่ log key / password) + nit 3 ข้อ · integration ทั้งชุดผ่าน · ขอรายการ nit ที่เหลือ 4 ข้อ
- 2026-10-15 · S1: เพิ่ม `middleware.ClientIP` (ไล่ XFF จากขวาข้าม proxy ที่เชื่อถือ) แทน `c.IP()` ทุกจุด · branch `boiledegg/fix/client-ip` · ยังไม่ merge / ยังไม่ได้ทดสอบผ่าน ingress จริง
- 2026-10-15 · รอบ 7: แก้ S1-a (validate TRUSTED_PROXIES ตอน boot) + N5 (PLAYER_LOBBY_URL) + nit doc (PA-10 · สถานะ merge · N3) · เหลือ N4 / N7 / S1-b
- 2026-10-15 · รอบ 4 ผ่าน (20722ab) · เพิ่มตามที่ lead ฝาก: log warn ตอน boot ถ้าไม่ตั้ง TRUSTED_PROXIES นอก local (`Config.ProxyWarning`) + ข้อความเตือนชัดใน DEPLOYMENT.md · ยังไม่ deploy dev จนกว่าจะได้ CIDR ของ ingress
