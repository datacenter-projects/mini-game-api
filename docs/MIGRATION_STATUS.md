# Migration Status — askmelotto_minigame-api → mini-game-api

## ขั้นตอนต่อ module (ต้องครบทุกขั้นตามลำดับ)

1. **Extract** — อ่าน *โค้ด* เก่า (ห้ามอ้างเอกสารเก่า รวมถึงการแบ่ง module/เลข M ของเอกสารเก่า) สรุปว่า endpoint
   ทำอะไรจริง คำนวณอย่างไร และจุดที่น่าสงสัย
2. **Spec** — lead ยืนยัน/แก้ business rule → เขียนลง `docs/modules/{module}.md` (ใช้ `_TEMPLATE.md`)
3. **Implement** — ตาม `docs/ARCHITECTURE.md`
4. **Test** — table test ใน `app/core` + integration test ตามตาราง test case ใน spec
5. **Review** — PR ผ่าน `make check` + CI + review อย่างน้อย 1 คน

สถานะ: `—` ยังไม่เริ่ม · `EX` extract แล้ว · `SP` spec อนุมัติ · `IM` implement แล้ว · `✅` test+review ผ่าน

การแบ่ง module ทำใหม่ทีละส่วนตามที่ lead กำหนด — ยังไม่มีแผนรวมทั้งระบบ

## Phase 0 — Foundation ✅

โครงโฟลเดอร์, config, logger, DB, Redis, goose, apperr, response, pagination, ParseBody + Validate(), health, middleware,
Makefile, docker-compose, Dockerfile, CI, check_structure.sh, CLAUDE.md, ARCHITECTURE.md

## Backoffice

| Module | Spec | สถานะ | ขอบเขตที่ทำแล้ว | ยังไม่ทำ |
|---|---|---|---|---|
| `agent_auth` | [agent_auth.md](modules/agent_auth.md) | IM | login, logout, middleware `Authenticated`, ตาราง `user_agents` (เฉพาะคอลัมน์ auth), `scripts/create_superadmin` | — |
| `agent_auth` phase 2 | [agent_auth_phase2.md](modules/agent_auth_phase2.md) | IM | subaccount login, passcode setup/change, `RequirePasscode`, เปลี่ยนรหัสผ่าน, admin reset passcode/password, ด่านหลัง login, `auth_audit_logs`, `scripts/create_admin` | `scripts/reset_credentials` (กู้รหัส SUPERADMIN / ADMIN — รอ lead เรื่อง actor ใน audit log) · AUTH-54 (`401311`) ทำใน branch account เข้า dev พร้อม account |
