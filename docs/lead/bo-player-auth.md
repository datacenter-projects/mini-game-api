# Player Auth (ผู้เล่นเข้าระบบ · สาย Transfer) — spec อนุมัติแล้ว (lead zerph 2026-10-15) · **implement แล้ว รอเปิด PR / verify**

จาก boiledegg · review: [docs/reviews/player-auth.md](../reviews/player-auth.md) (รอบ 1–4) · spec: [docs/modules/player_auth.md](../modules/player_auth.md) (**APPROVED**) · เอกสารหน้าบ้าน: [docs/frontend/player_auth.md](../frontend/player_auth.md)
branch `boiledegg/player/auth` (merge dev ล่าสุดแล้ว · ยังไม่เปิด PR · ยังไม่ merge / deploy dev)

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

- **PA-09 win limit (R4-2) ยังไม่ครบ:** ตอนนี้ผู้เล่นที่ระบบสร้างใช้ค่าคงที่ `DefaultWinLimitPerDay` = 10,000,000 (ค่าเดียวกับที่ `members/create` ให้เมื่อไม่ส่งค่า · **ส่งชัดเจนตอน insert ไม่พึ่ง DEFAULT ของ DB**) ผ่านฟังก์ชันเดียว `defaultWinLimit(currency)` ใน `app/service/player_auth/operator_service.go` · เพดาน 1,000,000 บาทแปลงสกุล (member-payment รอบ 2) **ยังไม่มีบน dev** — `members/create` ยังใช้ค่าคงที่ · maofoy ยังไม่ push · `currency_rate` (`convert_service`) อยู่ใน branch `boiledegg/bo/currency-rate` ที่ยังไม่ merge → ต้องรอ maofoy merge แล้วสลับฟังก์ชันเดียวนี้ (มี TODO กำกับ)
- ชื่อเกมไทยที่ใส่ (สแครชการ์ด / โยนเหรียญ / เป่ายิ้งฉุบ) ผู้ร่างตั้งเอง — ขอให้ยืนยันคำที่ใช้บนหน้าเกม
- **ก่อน deploy dev:** ต้อง migrate RDS (เช็ค `goose_db_version` + เลข `20261015140000` ไม่ชนกับ branch อื่น) และตั้ง `PLAYER_LOBBY_URL` ใน env ของ dev (ไม่ตั้ง = `transfer/login` ตอบ `503`)

## 3. คำถามที่รอ lead

| # | เรื่อง | ข้อเสนอ |
|---|---|---|
| 1 | migration แก้ CHECK `ck_user_members_username` (เพิ่มเทียบ spec หัวข้อ 6 เดิม) | อัปเดต spec หัวข้อ 6 ให้ตรงแล้ว (มีหมายเหตุ "รอ lead ยืนยัน") — ขอยืนยันตอนรีวิว PR |

## 4. ประวัติอัปเดต

- 2026-10-15 · ร่าง spec `player_auth.md` + ส่งคำถาม 13 ข้อให้ lead
- 2026-10-15 · แก้ spec ตาม review รอบ 2–3 (PA6 · name_th/en · register password · snake_case · SUSPENDED/LOCKED ฯลฯ) · สถานะ → รอ approve
- 2026-10-15 · lead อนุมัติ spec (รอบ 4) · แก้ PA-09 win limit ตาม R4-2 · สถานะ → APPROVED · พร้อม implement
- 2026-10-15 · implement 8 เส้น + middleware + migration + test (branch `boiledegg/player/auth`) · เหลือ win limit ตาม MGMT-65 ใหม่ (รอ maofoy) · รอสั่งเปิด PR
