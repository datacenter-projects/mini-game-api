# สถานะ — `carl/asset`

## 1. ภาพรวม

| | |
|---|---|
| branch | `carl/asset` (แตกจาก `dev` @ `65300f1`) · merge `dev` แล้ว (ล่าสุด `1380078`) |
| spec | `docs/modules/asset.md` — **DRAFT** · lead review แล้ว 2026-10-12 (`docs/reviews/game-asset.md`) · **แก้ตามรีวิว S1–S11 แล้ว** (rule `AST-01`–`AST-60`) · รอ lead รับทราบฉบับแก้ |
| merge | ยังไม่ merge เข้า `dev` · มีแค่เอกสาร ยังไม่มีโค้ด (implement ใน branch ใหม่หลัง lead รับทราบ — S13) |
| test | `make check` ผ่าน (ไม่มีโค้ดใหม่) |

## 2. งาน

| ส่วน | สถานะ | หมายเหตุ |
|---|---|---|
| Extract โค้ดเก่า (asset, theme, branding, banner, bundle) | ✅ | ผลอยู่ในหัวข้อ 4 ของ spec (31 ข้อ) |
| spec `docs/modules/asset.md` | ✅ แก้ตามรีวิวแล้ว | MVP ตาม GS1 (catalog · asset · theme · bundle · branding ชื่อ / โลโก้) · ไล่สายทั้ง asset / theme / branding (GR4) · `use_now` (GR1) · soft delete (GR2) · ≤ 5 MB (GR3) · reset 3 เส้น (GR5) · path ตาม GN1–GN23 · error module `05` (GQ6) · ลำดับเช็ค 7.1 · test ไล่สาย 3 ชั้น / CSM / reset / `use_now=false` / BO = ผู้เล่น |
| `docs/MIGRATION_STATUS.md` เพิ่มแถว `asset` (S12) | ⏳ | ทำใน PR ของ implement (แก้ไฟล์กลาง — ไม่แตะตอนแก้ spec) |
| implement | ⏳ | รอ lead รับทราบฉบับแก้ + infra (S3 / CloudFront / IAM ต่อ env — รีวิวหัวข้อ 7.3) + merge `dev` ล่าสุด (S13) |
| เอกสารหน้าบ้าน `docs/frontend/asset.md` | ⏳ | ทำหลัง spec อนุมัติ |

## 3. รอ lead

1. **ตาราง GE ขาด** — รีวิว S10 อ้าง GE1-1, GE2-1, GE6-1, GE12-1, GE13-1 แต่ในไฟล์ `docs/reviews/game-asset.md` ไม่มีตารางนิยาม (มีแค่ GN ที่ส่งเป็นภาพ) · ขอส่งตาราง GE เพื่อเขียน spec ตามที่ตัดสิน
2. **`site_theme` ใน MVP** — GN13 เขียนว่าแก้ "ชื่อเว็บและธีมหน้าเว็บ" แต่ GN17 (`site-themes/list`) เลื่อน และ GS1 บอก branding MVP = ชื่อ / โลโก้ · เสนอ: ตัด `site_theme` ออกจาก GN13 ในรอบนี้
3. **รับทราบสิ่งที่ตัดสินเองใน spec (หัวข้อ 9):**
   - Share Master ตั้งเองไม่ได้ (`405301`) · CSM ตั้งครั้งเดียวทุกคนใต้ได้ตามผ่านสาย (ไม่มี "ตั้งแทน")
   - ไล่สายด้วย `user_agents.cnf` (MGMT-61) แทน recursive CTE — ผลเหมือนกัน query เดียว
   - `theme/detail/get` · `branding/detail/get` ตอบค่าที่ไล่สายแล้ว + `source` (ไม่แยก "อ่านของตัวเอง" กับ bundle)
   - ลบไฟล์ที่มีคนใช้ = `405403` รวมกรณีลูกสืบทอด · ต้อง `used/reset` / `used/update` ก่อน
4. **`/player/asset/bundle`** — `game_code` ใน body ใช่ไหม (รอ review Player Auth — P1) · เส้นรายชื่อเกมของ lobby (GQ9)
5. **ก่อน go-live** ต้องอัป GLOBAL ให้ครบทุกช่องที่เกมใช้ (seed ไม่มีรูป · ไม่มี fallback ไฟล์ล่าสุดแล้ว) — ใครเป็นคนอัป
6. **งาน infra** (ต้องสั่งภายในสัปดาห์นี้ถ้าจะทำใน ต.ค.): S3 bucket + CloudFront + DNS + IRSA ต่อ env · ConfigMap — เจ้าของ: zerph / infra (schneider ไปคุยเอง)

## 4. ประวัติอัปเดต

| วันที่ | commit | อัปเดต |
|---|---|---|
| 2026-10-08 | `9e95aa8` | ร่าง spec `asset.md` (DRAFT) จากเอกสารสรุประบบเดิม + extract โค้ดเก่า |
| 2026-10-08 | `91bacde` · `ceb3ac9` | merge `dev` (module ② · account · reset script) |
| 2026-10-08 | `78da326` | ตรวจ spec กับ `dev` ล่าสุด: แบ่งหน้า · ลิงก์ spec · เพิ่มไฟล์สถานะนี้ |
| 2026-10-09 | `ea76f89` | ปรับตามเอกสารสรุปฉบับแก้: branding ไม่มี GLOBAL · รูปปก `game_cover` · ผู้เล่นอ่านอย่างเดียว · คำถามข้อ 8–10 |
| 2026-10-12 | `1dd674f` | lead review Game_Asset — MVP, ไล่สาย, lobby (GR6), GQ1–GQ13 + รายการแก้ + infra |
| 2026-10-12 | — (รอ commit) | แก้ spec ตามรีวิว S1–S11: MVP · ไล่สาย (`cnf`) · `use_now` · soft delete · ≤ 5 MB · reset 3 เส้น · path GN1–GN23 · error `05` · ลำดับเช็ค · test case ใหม่ · คำถามใหม่ (GE ขาด · `site_theme`) |
