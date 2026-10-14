# Review Game_Asset (BO + Player) — lead: zerph · 2026-10-12 · ✅ ตัดสินแล้ว

อ้างอิง: `docs/modules/asset.md` (DRAFT · branch `carl/asset` `823f88d`) · เจ้าของ module: schneider · ไฟล์นี้คือการตัดสินของ lead — แก้ spec ตามรายการด้านล่างก่อน แล้วค่อย implement ใน branch ใหม่ (merge dev ล่าสุดก่อน) และเปิด PR ให้ lead review

## การตัดสินของ lead (ตามแนะนำทั้งหมด + GR6)
| # | เรื่อง | ตัดสิน |
|---|---|---|
| GS1 | scope ต.ค. | **ทำ MVP (~8.5 pd):** catalog + asset ของเกม + theme + bundle (BO preview + ผู้เล่น) + branding ชื่อ/โลโก้ · **เลื่อน:** banner, favicon, site-themes, change log · ถ้าไม่ทันจริงลดเหลือ GLOBAL อย่างเดียว (~5 pd) |
| GR4 + GQ1 + GQ2 | เจ้าของ + การสืบทอด | **ไล่สายทั้ง asset / theme / branding:** บัญชีหลักฝั่ง agent ทุกประเภท (Company / Share / Agent) ตั้งของตัวเองได้ · สายล่างที่ไม่ได้ตั้งได้ค่าของชั้นบนที่ใกล้ที่สุด → GLOBAL · asset ไล่ทีละช่อง · theme / branding ไล่ทั้งชุด · ชั้นบนดูของสายล่างได้ แก้ไม่ได้ · นอกสาย `402402` · **CSM ตั้งครั้งเดียว Share Master ทุกคนได้ตาม (H2)** · ตัด `405302` · ใช้กฎเดียวกันทั้ง BO preview และผู้เล่น (recursive CTE query เดียว) |
| GR1 | อัปแล้วขึ้นจริงไหม | ใช้งานจริง = `used` เท่านั้น · upload มี `use_now` (ค่าเริ่มต้น `true`) · ตัดกฎ "ไม่เคยเลือก = ใช้ไฟล์ล่าสุด" |
| GR2 | ลบ | soft delete ทั้ง GLOBAL และ AGENT · ไม่ลบไฟล์ S3 ระหว่าง request (job กวาดทีหลัง) · ตัดกฎ "ลบได้เฉพาะคนอัป" · ตัด `405303` |
| GR3 + GQ11 | ขนาดไฟล์ | **≤ 5 MB** · คง multipart ผ่าน API (ไม่ใช้ presigned) · ไม่แตะ `BodyLimit` · ห้าม SVG |
| GR5 | กลับไปใช้ค่าชั้นบน | เพิ่มเส้น reset 3 เส้น: theme · branding · used |
| **GR6** | ผู้เล่นเข้าทาง lobby ของเราไหม | **มี lobby ของเรา** เพื่อแสดงเกมทั้งหมดที่เรามี → branding (ชื่อ/โลโก้) จำเป็น · ต้องมีเส้นรายชื่อเกมของ lobby ฝั่งผู้เล่น (GQ9 — ตัดสินพร้อม Player Auth) · banner ยังเลื่อน (GS1) |
| GN | ชื่อเส้น | รูปแบบ N1 · ทุกเส้น `POST` · เส้นผู้เล่นรอ review Player Auth |
| GQ3 | เมนูสิทธิ์ branding / banner | ใช้เมนู `asset` ทั้งหมด |
| GQ4 | upload ระบุเป้าหมาย | form field |
| GQ5 | ลบไฟล์โลโก้เก่า / change log | ไม่ลบไฟล์รอบนี้ · change log เลื่อนหลัง MVP (ระหว่างนี้ใช้ `created_by/updated_by` + library) |
| GQ6 | error module id | **`05`** · แก้ข้อมูลใน spec "game-rate จอง 04" → `06` · ตัด code ที่ไม่ใช้แล้ว |
| GQ7 | รายชื่อช่อง / ขนาด / palette | ส่งให้ทีมเกมยืนยัน · ไม่ได้คำตอบใน ~3 วัน seed ตามร่างไปก่อน แก้ด้วย migration ใหม่ภายหลัง |
| GQ8 | branding เก็บที่ไหน | ตาราง 1:1 `site_brandings` |
| GQ9 | รายชื่อเกมของ lobby (public / ต้อง login) | ตัดสินพร้อม Player Auth · ตอนนี้ทำ service ให้ใช้ซ้ำได้ |
| GQ10 | theme ฝั่งผู้เล่น | รวมใน bundle |
| GQ12 | ส่วนกลาง (AWS SDK v2 · `S3_*` · `platform/storage` · multipart helper · middleware · จอง id) | **อนุมัติ** + credential ใช้ IRSA/default chain (ห้าม key ใน env) · ระบบต้อง boot ได้แม้ storage ไม่พร้อม · MinIO ใน test env |
| GQ13 | บั๊กเก่าที่ spec ปิดแล้ว | รับทราบ ปิดตาม spec |

## รายการแก้ + สิ่งที่ต้องมีจาก infra

### 7.1 Spec `docs/modules/asset.md` (ยังไม่มีโค้ด ให้แก้ก่อนแล้วค่อย implement)
- S1 หัวเอกสาร: บันทึกการตัดสินของ lead (GQ · GS · GN · GR · GE) + ระบุว่า rule มาจากข้อเสนอที่ lead อนุมัติแล้ว
- S2 หัวข้อ 1: ขอบเขต MVP / เลื่อน ตาม GS1 · ย้าย banner, favicon, site-themes, change log ไปหัวข้อ "ภายหลัง"
- S3 AST-01/02/04/05 + GQ1/GQ2: เจ้าของ + การไล่สายตาม GR4 · ชั้นบนดูได้ แก้ไม่ได้ · กรณี CSM / Share Master (H2) · ตัด/คง `405302`
- S4 AST-21/24/25 → ใช้งานจริง = `used` เท่านั้น · `use_now` (GR1) · fallback ไล่สายทีละช่อง
- S5 AST-28–32 → soft delete ทั้ง 2 scope (GR2) · ตัด `405303` · ข้อความ `405403` แบบเดียว
- S6 AST-16 → ≤ 5 MB (GR3) · `Content-Type` / `Cache-Control` ของ object · ห้าม SVG
- S7 เพิ่มเส้น reset 3 เส้น (GR5) · AST-37 / AST-47 อ้างถึง reset
- S8 หัวข้อ 5: path ตาม GN1–GN23 · ทุกเส้น `POST` (ยกเว้นผู้เล่นรอ P1) · `config_id` ชื่อเดียวกันทุกเส้น · ตัวอย่าง JSON เต็มของทุกเส้น (รวม `logo/upload`, `banners/*` ที่ตอนนี้ย่อ)
- S9 เมนูสิทธิ์ (GQ3) · error module `05` + ตารางที่ตัดแล้ว (GQ6) · แก้ข้อมูล "game-rate จอง 04" → `06`
- S10 GE1-1, GE2-1, GE6-1, GE12-1, GE13-1 ตามที่ lead ตัดสิน · theme/branding `detail/update` = แทนทั้งชุด (เขียนว่าไม่ใช้ MN2)
- S11 test case: ไล่สาย 3 ชั้น (Member ใต้ Agent ใต้ Share B2C) · CSM → Share Master · reset · `use_now=false` · BO preview = ผู้เล่น
- S12 `docs/lead/carl-asset.md`: ลบหมายเหตุ "ยังไม่ commit" · `MIGRATION_STATUS.md` เพิ่มแถว asset · หลังอนุมัติแล้วทำ `docs/frontend/asset.md`
- S13 merge `dev` ล่าสุดเข้า branch ก่อนเริ่มโค้ด (ตอนนี้ตามหลังหลาย commit · timestamp ของ migration ต้องอยู่หลังไฟล์ล่าสุดบน dev ตาม MA6)

### 7.2 ส่วนกลางที่ต้องอนุมัติ (GQ12)
`go.mod` AWS SDK v2 (s3) · `pkg/configs`: `S3_REGION` `S3_BUCKET` `S3_CDN_BASE_URL` `S3_UPLOAD_TIMEOUT` (+ `S3_ENDPOINT` สำหรับ MinIO ใน local/test) + `.env.example` · `platform/storage` · `pkg/utils` helper multipart + `Validate()` · middleware `ResolveAssetTarget` / `RequireAssetTarget` · `docker-compose` / `scripts/testenv.ps1` เพิ่ม MinIO · `ERROR_CODES.md` จอง `05`

### 7.3 สิ่งที่ต้องมีจาก infra (ต่อ env: dev / uat / prod)
| ของ | รายละเอียด | ใคร |
|---|---|---|
| S3 bucket | 1 bucket ต่อ env (เช่น `minigame-asset-{env}`) · Block Public Access เปิดทั้งหมด · SSE-S3 · versioning ไม่จำเป็น (key ไม่ซ้ำอยู่แล้ว) · lifecycle: ไม่ตั้ง (ลบด้วย job ทีหลัง) · CORS ไม่ต้องตั้ง (ไม่ใช้ presigned) | infra (terraform) |
| CDN | CloudFront + OAC ไปที่ bucket · โดเมนแยก เช่น `cdn-{env}.<domain>` ผ่าน Cloudflare DNS · cache policy ยึด `Cache-Control` ของ origin · CORS header `Access-Control-Allow-Origin` ให้โดเมนเกม (ถ้าเกมวาดรูปลง canvas) | infra |
| IAM | IRSA role ให้ service account ของ mini-game-api: `s3:PutObject` `s3:DeleteObject` `s3:GetObject` เฉพาะ `arn:…:{bucket}/games/*` และ `/sites/*` · ห้ามใช้ access key ใน env | infra |
| config ใน k8s | `S3_REGION` `S3_BUCKET` `S3_CDN_BASE_URL` `S3_UPLOAD_TIMEOUT` ใน ConfigMap ของ infra-gitops · ServiceAccount annotation ของ IRSA | zerph / infra |
| ingress | ถ้ามี body size limit ที่ ingress/ALB ต้อง ≥ 6 MB สำหรับเส้น upload | infra |
| local / test | MinIO ใน `docker-compose` + test env | schneider |
| ค่าใช้จ่าย | น้อยมาก (รูปไม่กี่ร้อย MB · ค่า egress ของ CDN ขึ้นกับจำนวนผู้เล่น) | — |

**ถ้าจะทำใน ต.ค. ต้องสั่งงาน infra ภายในสัปดาห์นี้** เพราะ IRSA + CloudFront + DNS ต้องทำให้ครบทั้ง 3 env

---


## ตรวจ spec รอบที่ 1 (2026-10-12 · `carl/asset` f19a511) — lead ตัดสิน
ผลตรวจ: GS1 · GR1–GR3 · GR5 · GR6 · GN · GQ3–GQ13 · error 05/06 · ส่วนกลาง GQ12 · test S11 · timestamp migration ✅ · **เริ่มโค้ดได้** หลังแก้ข้อด้านล่าง (ส่วนที่ไม่ติด เช่น `platform/storage`, `S3_*`, MinIO, helper multipart, middleware เป้าหมาย, core ไล่สาย/shades, theme, reset เริ่มได้ทันที) · infra lead สั่งแล้ว 2026-10-12

| # | เรื่อง | lead ตัดสิน |
|---|---|---|
| AV1 | Share Master ตั้งหน้าตาเองได้ไหม (spec ตอบ `405301` — ต่างจาก GR4) | **Share Master ตั้งเองไม่ได้ ใช้ของ Company Seamless Master** (asset · theme · branding ทั้งหมด) — ทุกเว็บใต้ CSM เดียวกันหน้าตาเหมือนกัน · คง `405301` ของ spec · เป็นข้อยกเว้นของ GR4 เฉพาะ Share Master (สอดคล้อง H2/H3) · บัญชีประเภทอื่นตั้งเองได้ตาม GR4 |
| AV2 | ตาราง GE ที่ S10 อ้าง (schneider ไม่มีไฟล์) | ใช้ตารางด้านล่าง |
| AV3 | สิ่งที่ schneider ตัดสินเอง | **รับทั้งหมด:** ไล่สายด้วย `cnf` (MGMT-61) แทน recursive CTE · `detail/get` ตอบค่าที่ไล่สายแล้ว + `source` + **`inherited_from`** (username ของชั้นบนเมื่อ `source = upline`) · Member ใช้สายของผู้สร้าง · ลบไฟล์ที่สายล่างสืบทอดอยู่ได้ `405403` |
| AV4 | รูปค่ากลาง (GLOBAL) ทุกช่อง | ต้องอัปให้ครบก่อน UAT (ไม่มี fallback แล้ว) · **schneider ทำ seed / script อัปรูป GLOBAL** จากไฟล์ที่ทีมเกม/ออกแบบส่งมา เป็นส่วนหนึ่งของงาน |

### ตาราง GE (รายละเอียดทีละเส้น — ตัดสินตามแนะนำ)
| GE | สิ่งที่ต้องเป็น |
|---|---|
| GE1-1 | ปกเกมใน `games/list` ใช้ค่าที่**ไล่สายจากผู้เรียก** (เห็นแบบเดียวกับผู้เล่นของตัวเอง) ไม่ใช่ GLOBAL อย่างเดียว |
| GE1-2 | `games/list` ไม่กรองเกมที่ `status_game` ปิด (หน้าจัดการหน้าตายังต้องเห็นทุกเกม) |
| GE2-1 | ตัด `description` ออกจากช่อง (response + คอลัมน์ `asset_types`) — หน้าบ้านแปลชื่อช่องจาก `asset_type` เอง |
| GE6-1 | `used/list` ส่ง**ทุกช่อง**ของเกม ช่องที่ไม่มีไฟล์ได้ `url: ""` (ACC-32) |
| GE6 / GE9 / GE12 | เพิ่ม `inherited_from` เมื่อ `source = upline` (AV3) |
| GE10-1 | theme ส่งค่าเดิมทั้งชุด → 200 ไม่แตะ `updated_*` ไม่สร้างแถวใหม่ (แนว MQ5) |
| GE13 | `site_name` 1–50 ตัว ตัดช่องว่างหน้าหลัง ห้ามอักขระควบคุม |
| GE13-1 | `site_theme` ไม่อยู่ใน MVP (ตรงกับ spec แล้ว) |
| GE22-2 | เส้นฝั่งผู้เล่น (bundle + รายชื่อเกมของ lobby) ใครทำ → ตัดสินพร้อม review Player Auth |

เก็บเล็กน้อย: AST-41 ขัดกับ AST-47 · `docs/lead/carl-asset.md` ยังเขียน "รอ commit" · ตัวอย่าง shades ย่อ · ตาราง field ของ request (ใส่ใน frontend doc ได้) — รายละเอียดใน verify report

## ผลตรวจโค้ด `carl/asset` @ `b29a7ff` (lead zerph 2026-10-14 · รายงาน verify-asset-code)
ไม่มี must-fix · การตัดสินของ lead ครบ · 16 เส้นตรง spec → **merge เข้า dev ได้หลังทำ A1–A2 + merge dev ล่าสุด + CI (integration) เขียว** · เปิด PR

| # | เรื่อง | ผล |
|---|---|---|
| A1 | ไม่มีเพดาน pixel (โลโก้ไม่ตรวจขนาด · ช่องที่ไม่มี `allowed_sizes`) — PNG 5 MB ขนาด 30000² ทำ browser ผู้เล่นค้างได้ | **เพดานกลางใน `checkImage` กว้างและสูง ≤ 4096 px ทุกเส้น upload** · เกิน = `405101` · เพิ่ม test |
| A2 | อ่าน header ภาพไม่ได้แล้วปล่อยผ่าน (fail-open ตาม AST-17) | **เปลี่ยนเป็น fail-closed**: `DecodeConfig` ไม่ผ่าน = `405102` (รวม WebP) · แก้ AST-17 ใน spec · เพิ่ม test ไฟล์ที่มีแค่ magic byte |
| A3 | รัน `migrate up` บน dev DB จาก branch ที่ยังไม่ merge โดยไม่ขอก่อน | lead รับทราบ · **ห้ามแก้ไฟล์ `20261012160000` / `20261012170000` อีก** (ต้องเปลี่ยน = เพิ่ม migration ใหม่) · ครั้งต่อไปขอก่อนเขียน DB / รัน migration บน env ร่วม |
| A4 | เอา MinIO ออก ใช้ `scripts/fakes3` · ตรวจสิทธิ์ก่อนอ่าน body · GE10-1 เทียบค่าที่ไล่สายแล้ว | รับทั้งหมด |

nit 7 ข้อใน verify report ทำหรือไม่ก็ได้ · spec เปลี่ยนสถานะจาก DRAFT เป็น APPROVED ได้หลังแก้ A2
