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
| `agent_auth` phase 2 | [agent_auth_phase2.md](modules/agent_auth_phase2.md) | IM | subaccount login, passcode setup/change, `RequirePasscode`, เปลี่ยนรหัสผ่าน, admin reset passcode/password, ด่านหลัง login, `auth_audit_logs`, `scripts/create_admin`, `scripts/reset_credentials`, AUTH-54 (`PassedGates` ปฏิเสธ SUSPENDED · `PassedGatesAllowSuspended`) | — |
| `agent_management` | [agent_management.md](modules/agent_management.md) | IM | สร้างบัญชี (agent / Member + ยอดเงินตั้งต้น + ledger), รายชื่อดาวน์ไลน์, รายละเอียด, copy-sources, แก้ข้อมูล / สถานะ / PT / Commission / ค่าถือ / เปิด-ปิดเกม (`update-games`), `pt` ชุดเดียวต่อระบบ (`minigame`) + ผู้สร้าง / ผู้แก้ล่าสุด, sub (รายชื่อ / สร้าง / แก้ / สถานะ), `RequirePermission`, `RequireMainAccount`, สายชั้นบน `user_agents.cnf` + `agent_game_settings.parent_id` (MGMT-61, MGMT-62), เงินและ % เป็น float ปัด 4 ตำแหน่ง (กฎข้อ 9 แก้ 2026-10-09 · migration `20261009130000`) | — (MGMT-04 และ test sub INACTIVE → 401311 ทำใน branch account) |
| `member_management` | [member_management.md](modules/member_management.md) | IM | แยกจาก `agent_management` (lead อนุมัติ 2026-10-08): สร้าง Member (`create`), รายละเอียด (`detail`), แก้ข้อมูล / สถานะ / PT + Commission (`update-info` `update-status` `update-pt`) · agent ถือ PT สู้ Member แต่ละคน (`pt` · `remain_quota` — 2026-10-09 รอ lead อนุมัติ · sync เมื่อ `agents/update-pt` รอ agent_management — member_management.md หัวข้อ 7) · ใช้ helper ร่วมจาก `service/agent_management` · `core` ยังเป็น `agent_management` | — |
| `admin_management` | [admin_management.md](modules/admin_management.md) | IM | แยกจาก `agent_auth` / `agent_management` (2026-10-09): `/admin/passcode/reset`, `/admin/password/reset`, `/admin/accounts/search`, middleware `adminMw.RequireAdmin`, `scripts/reset_credentials` | — |
| `account` | [account.md](modules/account.md) (APPROVED 2026-10-07) | IM | `GET /bo/pr/account/profile` ค่าจริงทุก field (ประเภท, สกุล, ยอด, PT, `status_game`, สิทธิ์ — บัญชีถูกระงับเห็นแค่ `report` = `view`) · 1.3 `GET /account/api-credential` + `POST /account/update-credential` (Key สร้างพร้อมบัญชีเจ้าของ — MGMT-04) · AUTH-54 `401311` | Dashboard (รอ module เดิมพัน) |
