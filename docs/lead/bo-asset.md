# Asset (หน้าตาเกม / หน้าเว็บ) — เตรียม merge เข้า dev · 2026-10-14

จาก boiledegg · งานของ carl (`carl/asset`) · review: [docs/reviews/game-asset.md](../reviews/game-asset.md) · สถานะของ carl: [carl-asset.md](carl-asset.md)
branch `boiledegg/bo/asset` (แตกจาก `carl/asset` @ `9b2956e` + merge dev ล่าสุด) · **ยังไม่ merge / deploy — รอ lead approve**

## 1. เงื่อนไข merge จากผลตรวจโค้ด `b29a7ff` (lead 2026-10-14)

| เงื่อนไข | ผล |
|---|---|
| A1 เพดาน 4096 px ทุกเส้น upload | ✅ carl แก้แล้ว (`1e5eb36`) — `core/asset` `MaxImageDimension = 4096` + test |
| A2 อ่าน header ภาพไม่ได้ = `405102` (fail-closed) | ✅ carl แก้แล้ว (`1e5eb36`) — `image.DecodeConfig` ไม่ผ่าน = `405102` + test · spec APPROVED |
| merge dev ล่าสุด | ✅ `824827f` (dev @ `3a80c7b` — รวม game_rate · MGMT-18A · deposit_withdraw) |
| CI (integration) เขียว | ✅ รันในเครื่อง: unit + integration ผ่านทุก package · lint 0 issues |
| A3 ห้ามแก้ `20261012160000` / `20261012170000` | ✅ ไม่ได้แตะ · 2 ไฟล์นี้รันบน RDS dev แล้ว (ตอน deploy ไม่มี migration ของ asset ให้รันเพิ่ม) |

## 2. ที่ boiledegg ทำเพิ่มใน branch นี้

| commit | งาน |
|---|---|
| `824827f` | merge dev → conflict 2 ไฟล์ (ทั้งสองฝั่งเพิ่มบรรทัดตรงจุดเดียวกัน — เก็บทั้งคู่): `routes.go` import asset + deposit_withdraw · `docs/ERROR_CODES.md` module 04 deposit_withdraw + 05 asset เรียงตามเลข · `pkg/testutil` รายการตารางที่ล้างครบ (game_rate · deposit_withdraw · asset) ไม่ซ้ำ |
| `8492825` | แก้ lint 10 ข้อใน `scripts/fakes3/main.go` (ไม่แก้ CI จะไม่ build image): สิทธิ์โฟลเดอร์ `0750` / ไฟล์ `0600` · `http.Server` มี timeout · log ค่าจาก request ผ่าน `strconv.Quote` · `Close` เช็ค error · `//nolint:gosec` G304 พร้อมเหตุผล (path ผ่าน `resolve` อยู่ใต้ root เท่านั้น) · ลองรันจริง PUT / GET / DELETE ผ่าน |

## 3. smoke test ในเครื่อง (API + `scripts/fakes3` + DB test) — รอบ 1 ผ่าน 20 / 20 · รอบ 2 ผ่าน 51 / 51

| ส่วน | ผล |
|---|---|
| GLOBAL (Superadmin) | อัป `icon_button` 32×32 · `main_play` 186×76 + ใช้ทันที · เปิด URL รูปได้ (200 `image/png`) |
| ไฟล์ผิด | ขนาดไม่ตรง → `405101` · ไม่ใช่รูป → `405102` · ช่องที่ไม่มี → `405401` |
| สิทธิ์ | Superadmin แก้ของ Company → `405301` · Company ลบรูป GLOBAL → `405301` (ตรง 1.3) |
| Company ของตัวเอง | อัป `icon_button` ขนาดที่ 2 (40×40) · `used/list` · `files/list` · `used/reset` · ลบรูปที่ไม่ใช้ |
| bundle | `source: "mixed"` — `icon_button` จาก Company · `main_play` จาก GLOBAL |
| theme / branding | ตั้งแล้ว get ได้ `source: "self"` ค่าตรง |

### smoke รอบ 2 — ผ่าน 51 / 51

| ส่วน | ผล |
|---|---|
| เกมอื่น | `coin_toss` (11 ช่อง) / `rock_paper_scissors` (12 ช่อง) — ทุกช่องไม่กำหนดขนาด (`allowed_sizes: []`) · อัป GLOBAL + ใช้ทันที · เปิด URL ได้ · `used/list` / theme GLOBAL ได้ |
| ไล่สาย Company → Share → Agent | Company ตั้งรูป / ธีม / ชื่อเว็บ → Agent ได้ค่าของ Company (`upline` · `inherited_from` = Company) · Share ตั้ง `main_play` → Agent ได้ `main_play` จาก Share + `icon_button` จาก Company (`source: upline` — ที่มาประเภทเดียวกัน) · Agent ตั้งเองอีกช่อง → `source: mixed` |
| สิทธิ์ในสาย | Company ดูของ Agent ได้ · Company แก้ของ Share → `405301` · Agent ดูของ Company → `402402` |
| โลโก้ | อัปได้ เปิด URL ได้ · Agent ได้โลโก้ของ Company · กว้าง 4100 px → `405101` · ไม่ใช่รูป → `405102` · ไม่แนบไฟล์ → `422` · อัปซ้ำได้ URL ใหม่ (เขียนทับ) · `branding/reset` แล้ว Agent ไม่มี branding (ไม่มีค่ากลาง) |
| sub ของ Company สิทธิ์ `asset` = view | เปลี่ยนรหัสครั้งแรก + ตั้ง passcode · ดู `used/list` / bundle ได้ · อัปรูป / ตั้งธีม / อัปโลโก้ → `402303` |

ไม่เจอบั๊ก · ข้อที่ไม่ผ่านระหว่างเขียน script เป็นความเข้าใจผิดของ script เอง (ชื่อมีช่องว่าง / ใช้ token ผิดคน / ช่องไม่กำหนดขนาด / คาด `mixed` ทั้งที่ทุกช่องเป็น `upline`) — ระบบตอบตรง spec ทุกข้อ

## 4. ขอ lead approve

1. merge `boiledegg/bo/asset` เข้า dev แล้ว deploy (tag ใหม่) — **ไม่มี migration ต้องรัน**
2. infra ของ dev: ต้องตั้ง `S3_BUCKET` / `S3_REGION` / `S3_CDN_BASE_URL` + credential ของ AWS ใน env ของ dev ก่อน (ไม่ตั้ง = เส้นอัปโหลดตอบ `505801` แต่ระบบยัง boot ได้) — ตามรายการ "สิ่งที่ต้องมีจาก infra" ใน review
3. หลัง deploy: รัน `scripts/seed_asset_global` อัปรูปค่ากลางของทุกช่อง (AST-61 · ต้องว่างก่อน UAT) — รอไฟล์จากทีมเกม / ออกแบบ
4. nit 7 ข้อใน verify report ยังไม่ได้ทำ (ทำหรือไม่ก็ได้ตามผลตรวจ)
