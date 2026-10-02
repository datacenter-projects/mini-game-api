# Migration Status — askmelotto_minigame-api → mini-game-api

## ขั้นตอนต่อ module (ต้องครบทุกขั้นตามลำดับ)

1. **Extract** — อ่าน *โค้ด* เก่า (ห้ามอ้างเอกสารเก่า) สรุปว่า endpoint ทำอะไรจริง คำนวณอย่างไร และจุดที่น่าสงสัย
2. **Spec** — lead ยืนยัน/แก้ business rule → เขียนลง `docs/modules/{module}.md` (ใช้ `_TEMPLATE.md`)
3. **Implement** — ตาม `docs/ARCHITECTURE.md`
4. **Test** — table test ใน `app/core` + integration test ของ service ตาม spec
5. **Review** — PR ผ่าน `make check` + review อย่างน้อย 1 คน

สถานะ: `—` ยังไม่เริ่ม · `EX` extract แล้ว · `SP` spec อนุมัติ · `IM` implement แล้ว · `✅` test+review ผ่าน

> รายการ module และจำนวน endpoint ด้านล่างเป็นการประมาณจากโค้ดเก่า ต้องยืนยันใหม่ในขั้น Extract

## Phase 0 — Foundation

| งาน | สถานะ |
|---|---|
| โครงโฟลเดอร์, config, logger, DB, Redis, goose | ✅ |
| apperr, response envelope, pagination, validator | ✅ |
| health, global middleware, routes mount | ✅ |
| Makefile, docker-compose, Dockerfile, CI, check_structure.sh | ✅ |
| CLAUDE.md, ARCHITECTURE.md | ✅ |
| helper สำหรับ integration test (DB ต่อ test) | — (ทำพร้อม module แรกที่มีตาราง) |

## Module

| ลำดับ | Module | Context | Endpoints (ประมาณ) | สถานะ | Spec | หมายเหตุ |
|---|---|---|---|---|---|---|
| 1 | Auth / Passcode | backoffice | 10 | — | | JWT, session, middleware auth ของ backoffice |
| 2 | Agent hierarchy | backoffice | 9 | — | | |
| 3 | Subaccount + permission | backoffice | 6 | — | | |
| 4 | PT commission / Force-Remain | backoffice + core | 5 | — | | กฎคำนวณต้องยืนยันใหม่ทั้งหมด |
| 5 | Agent float | backoffice | 3 | — | | |
| 6 | Bet / Wallet | backoffice + service | 5 | — | | |
| 7 | Player management | backoffice | 13 | — | | |
| 8 | Scratch card | game | 8 | — | | |
| 9 | Coin toss | game | 6 | — | | |
| 10 | Rock-paper-scissors | game | 6 | — | | |
| 11 | Game rate | game | 2 | — | | |
| 12 | Provider (inbound) | externals | 2 | — | | |
| 13 | Operator callback (outbound + config) | service + backoffice | 8 | — | | |
| 14 | Player frontend | frontend | 7 | — | | |
| 15 | Reports / dashboard | backoffice | 7 | — | | |
| 16 | Asset / game theme | backoffice | 11 | — | | |
| 17 | Site content / branding | backoffice | 11 | — | | |
| 18 | Currency rate | backoffice | 2 | — | | |
| 19 | Maintenance | backoffice | 2 | — | | |
