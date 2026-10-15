# API หลังบ้าน — หน้าตาเกม / หน้าเว็บ (asset · ธีม · branding) (สำหรับหน้าบ้าน) — 14 ต.ค. 2026 (ฉบับแรก · แก้ตาม lead A1 / A2)

เอกสารนี้ตรงกับโค้ดบน branch `carl/asset` (**ยังไม่ merge เข้า `dev`** — ใช้บน server หลัง merge + deploy) ·
ตัวอย่าง response และข้อความ error ทุกตัว**ยิงจากโค้ดจริง** (14 ต.ค.) · spec เต็ม: `docs/modules/asset.md` (DRAFT — รอ lead รับทราบฉบับแก้ · ผู้ดูแล module: schneider) ·
login / session / ด่านหลัง login / ตั้งค่า Postman (`{{MG_URL}}` · `{{TOKEN}}`) ดู [auth.md](auth.md) และ [agent_management.md](agent_management.md) หัวข้อ 1

> **สถานะ:** MVP ฝั่งหลังบ้าน 16 เส้น · **ยังไม่มี**เส้นฝั่งผู้เล่น (`/player/asset/bundle` — รอ Player Auth) · banner / favicon / `site_theme` เลื่อนไปรอบหน้า (ยังไม่ต้องทำหน้าจอ)
>
> **ก่อนใช้งานจริง:** รูปค่ากลาง (GLOBAL) ของทุกช่องต้องถูกอัปก่อน (ทีมหลังบ้านรัน script) · ช่อง / ขนาดภาพในหัวข้อ 1.7 เป็น**ค่าร่าง รอทีมเกมยืนยัน** — ดึงจาก `slots/list` เสมอ อย่าฝังในหน้าบ้าน

ตัวอย่างทั้งเอกสารใช้บัญชีชุดเดียวกับ spec: `comp01` (Company Transfer) → `share01` (**id 12**, Share B2C) → `agent01` (**id 15**, Agent) ·
`csm01` (Company Seamless Master) → `sm01` (Share Master) · sub ของ share01 ชื่อ `share01@design` (สิทธิ์ `asset` = edit) · โดเมนรูปตัวอย่าง `https://cdn.example.com`

## 1. พื้นฐาน

### 1.1 ภาพรวมเส้น

| # | Path (ต่อท้าย `{{MG_URL}}`) | ใช้เมื่อ | สิทธิ์ `asset` | ระบุเป้าหมายด้วย |
|---|---|---|---|---|
| 2.1 | `/api/v1/bo/pr/asset/games/list` | รายชื่อเกม + รูปปก | view | — (บัญชีที่ login) |
| 2.2 | `/api/v1/bo/pr/asset/slots/list` | ช่องรูปของเกม + ขนาดภาพที่รับ | view | — |
| 2.3 | `/api/v1/bo/pr/asset/files/upload` | อัปรูปเข้าช่อง (multipart) | edit | `scope` + `agent_id` |
| 2.4 | `/api/v1/bo/pr/asset/files/list` | คลังรูปของช่อง (แบ่งหน้า) | view | `scope` + `agent_id` |
| 2.5 | `/api/v1/bo/pr/asset/files/delete` | ลบรูปจากคลัง | edit | `scope` + `agent_id` |
| 2.6 | `/api/v1/bo/pr/asset/used/list` | รูปที่ใช้อยู่จริงทุกช่องของเกม | view | `scope` + `agent_id` |
| 2.7 | `/api/v1/bo/pr/asset/used/update` | เลือกรูปจากคลังมาใช้กับช่อง | edit | `scope` + `agent_id` |
| 2.8 | `/api/v1/bo/pr/asset/used/reset` | ยกเลิกรูปที่ตั้งเอง → กลับไปใช้ของชั้นบน | edit | `scope` + `agent_id` (AGENT เท่านั้น) |
| 2.9 | `/api/v1/bo/pr/asset/theme/detail/get` | ธีมที่ใช้อยู่ + shades | view | `scope` + `agent_id` |
| 2.10 | `/api/v1/bo/pr/asset/theme/detail/update` | ตั้งฟอนต์ / สี (แทนทั้งชุด) | edit | `scope` + `agent_id` |
| 2.11 | `/api/v1/bo/pr/asset/theme/reset` | กลับไปใช้ธีมของชั้นบน | edit | `scope` + `agent_id` (AGENT เท่านั้น) |
| 2.12 | `/api/v1/bo/pr/asset/branding/detail/get` | ชื่อเว็บ + โลโก้ที่ใช้อยู่ | view | `agent_id` |
| 2.13 | `/api/v1/bo/pr/asset/branding/detail/update` | แก้ชื่อเว็บ | edit | `agent_id` |
| 2.14 | `/api/v1/bo/pr/asset/branding/logo/upload` | อัปโลโก้ (multipart) | edit | `agent_id` |
| 2.15 | `/api/v1/bo/pr/asset/branding/reset` | กลับไปใช้ branding ของชั้นบน | edit | `agent_id` |
| 2.16 | `/api/v1/bo/pr/asset/bundle/detail/get` | preview: สิ่งที่ผู้เล่นของบัญชีนี้จะเห็น | view | `agent_id` |

- ทุกเส้นเป็น `POST` · ไม่มี `GET` / `DELETE` · ไม่มี `?agent_id=` หรือ `me`
- header: `Authorization: Bearer <token>` · JSON ใช้ `Content-Type: application/json` · multipart ไม่ต้องตั้ง `Content-Type` เอง (หัวข้อ 1.6) · `X-Lang: en` = `msg` อังกฤษ (ไม่ส่ง = ไทย)

### 1.2 `scope` + `agent_id` — "กำลังดู / แก้ของใคร"

| `scope` | ความหมาย | `agent_id` |
|---|---|---|
| `"AGENT"` | ค่าที่ตั้งไว้ที่บัญชีหนึ่ง | **ต้องส่ง** เลขมากกว่า 0 |
| `"GLOBAL"` | ค่ากลางของระบบ (ปลายทางสุดท้ายของการไล่สาย) | **ห้ามส่ง** (ส่ง `0` ได้) |

- `scope` ต้องเป็น `"GLOBAL"` / `"AGENT"` **ตัวพิมพ์ใหญ่ตรงตัว** · ไม่ส่ง / ค่าอื่น / ไม่ใช่ข้อความ = `422` · **ห้ามเดาว่าไม่ส่ง = GLOBAL**
- `agent_id` เป็น JSON number (ใน multipart เป็นข้อความตัวเลข เช่น `"12"`) · ส่งเป็นข้อความใน JSON (`"12"`) = `422`
- **เส้น branding / bundle (2.12 – 2.16) ไม่มี `scope`** ส่งแค่ `agent_id` · ส่ง `"scope": "AGENT"` มาด้วยได้ (ไม่มีผล) · ส่ง `scope` ค่าอื่น = `422`
- **หน้าของตัวเอง:** ใช้ `id` ของบัญชีที่ login (ได้จาก Profile — [account.md](account.md)) · **sub ใช้ `id` ของเจ้าของ** (sub ทำในนามเจ้าของ)

### 1.3 สิทธิ์ — ใครดู / แก้อะไรได้

| เป้าหมาย | ดู (เส้น view) | แก้ (เส้น edit) |
|---|---|---|
| **GLOBAL** | ทุกบัญชีที่มีสิทธิ์ `asset` | **Superadmin เท่านั้น** (หรือ sub ของ Superadmin ที่มี `asset` = edit) · บัญชีอื่น = `405301` |
| **บัญชีตัวเอง** (`AGENT`) | ได้ | ได้ เมื่อเป็น Company ทุกประเภท · Share B2B / B2C / Reseller · Agent |
| **Share Master ตัวเอง** | ได้ (ได้ค่าตาม Company Seamless Master) | **ไม่ได้** `405301` — ใช้ของ CSM · CSM ตั้งครั้งเดียว ทุกเว็บใต้ CSM หน้าตาเหมือนกัน |
| **Superadmin ตั้ง `AGENT` ของตัวเอง** | — | **ไม่ได้** `405301` — Superadmin ตั้งได้เฉพาะ GLOBAL |
| **บัญชีในสายล่าง** | ได้ (ชั้นบนดูของสายล่างได้) | **ไม่ได้** `405301` |
| **นอกสาย / ไม่มีบัญชีนี้** | `402402` | `402402` |

- ทุกเส้นต้องมีสิทธิ์เมนู **`asset`** (`view` = เส้นอ่าน · `edit` = เส้นเขียน) · ไม่มี / sub ที่มีแค่ `view` เรียกเส้นเขียน = `402303` · **ADMIN ไม่มีเมนูนี้**
- **ซ่อน / ปิดปุ่มแก้ล่วงหน้า** เมื่อ: ดู GLOBAL แต่ไม่ใช่ Superadmin · ดูของสายล่าง · เป็น Share Master · สิทธิ์ `asset` เป็น `view`
- **ลำดับ error:** ไม่มีสิทธิ์กับเป้าหมาย (`402402` / `405301`) ตอบ**ก่อน** `422` ของ field อื่น · ยกเว้น `scope` / `agent_id` / `null` ผิด ได้ `422` ก่อนเสมอ ·
  ตัวอย่าง: share01 แก้ของ agent01 แล้วลืม `config_id` → ได้ `405301` (ไม่ใช่ `422`) — ถ้าได้ `402402` / `405301` ให้แสดงว่าไม่มีสิทธิ์ ไม่ต้องชี้ช่องที่กรอก

### 1.4 การไล่สาย (ค่าที่ใช้จริงมาจากไหน)

บัญชีที่ไม่ได้ตั้งเอง ได้ค่าของ**ชั้นบนที่ใกล้ที่สุดที่ตั้งไว้** · ไม่มีใครตั้งเลย = **ค่ากลาง (GLOBAL)**

| | ไล่อย่างไร | มี GLOBAL ไหม |
|---|---|---|
| รูป | ทีละช่อง (ช่องหนึ่งมาจากชั้นบน อีกช่องมาจาก GLOBAL ได้) | มี |
| ธีม | ทั้งชุด 5 ค่าจากแถวเดียว (ไม่ผสมทีละค่า) · แยกต่อเกม | มี (ค่าเริ่มต้นของระบบ) |
| branding | ทั้งชุด (ชื่อ + โลโก้จากบัญชีเดียว) · ต่อบัญชี ไม่แยกเกม | **ไม่มี** |

- `source`: `self` (ตั้งเอง) · `upline` (ได้จากชั้นบน) · `global` (ค่ากลาง) · `mixed` (มาจากหลายที่ — เฉพาะ `source` ระดับบนสุดของ `used/list` / bundle) · `""` = ไม่มีค่า
- `inherited_from` = username ของชั้นบนที่ให้ค่ามา เมื่อ `source = "upline"` ไม่เช่นนั้น `""`
- กลับไปใช้ของชั้นบน = เรียก `reset` (2.8 / 2.11 / 2.15) — **ไม่ใช่**การลบรูป

### 1.5 รูปแบบ request / response

| เรื่อง | กติกา |
|---|---|
| สำเร็จ | `{ "code": 200, "msg": "สำเร็จ", "data": ... }` (อังกฤษ `"msg": "Success"`) · เส้น `delete` / `update` / `reset` **ไม่มี key `data`** |
| error | `{ "code": 422, "msg": "..." }` · ไม่มี key `data` · ดูที่ `code` เสมอ |
| HTTP status | error ของ module นี้ + `400` / `422` ตอบ **HTTP 200** · ยกเว้น `505801` (**503**) · `505802` (**502**) · `500` (**500**) · error ของ login / ด่าน ใช้ HTTP ตามหัวข้อ 1.8 |
| `null` | ห้ามส่งที่ใดใน body (`422` "`<field>` ห้ามเป็น null") · response ไม่มี `null`: ไม่มีค่า = `""` · `0` · `[]` · `{}` |
| ชนิดข้อมูลผิด | เช่น `"game_code": 123` / `"site_name": 123` = `400` "คำขอไม่ถูกต้อง" **ไม่บอก field** · JSON ผิดรูป = `400` เช่นกัน |
| `game_code` / `asset_type` | ต้องตรงทุกตัวอักษร (ตัวเล็ก เช่น `scratch_card`, `icon_button`) · ผิด / ตัวใหญ่ = `405401` |
| URL รูป | URL เต็มเปิดได้เลย · `""` = ยังไม่มีรูป (แสดง placeholder) · URL เปลี่ยนเมื่ออัปใหม่ |
| สี | response เป็น `#rrggbb` ตัวพิมพ์เล็กเสมอ (request ส่งตัวใหญ่ / เล็กก็ได้) |
| เวลา | RFC 3339 เวลาไทย (`+07:00`) อาจมีเศษวินาที เช่น `"2026-10-14T10:40:59.180517+07:00"` |

### 1.6 ไฟล์รูป (เส้นอัปโหลด 2.3 / 2.14)

- ชนิด: **png / jpeg / gif / webp** · ตรวจจาก**เนื้อไฟล์** (ไม่ดูนามสกุล / `Content-Type`) · **ห้าม SVG** (`405102`)
- ขนาดไฟล์ไม่เกิน **5 MB** (5,242,880 ไบต์ · เกิน = `405103`) — ควรตรวจก่อนส่ง
- **ต้องอ่านขนาดภาพจากไฟล์ได้** — ไฟล์เสีย / มีแค่หัวไฟล์ (magic byte) / อ่าน header ไม่ได้ = `405102` (รวม WebP)
- **กว้างและสูงไม่เกิน 4096 px ทุกเส้น** (รวมโลโก้ และช่องที่ไม่มี `allowed_sizes`) · เกิน = `405101` "ขนาดภาพต้องไม่เกิน 4096×4096 px" — ควรตรวจก่อนส่ง
- ขนาดตามช่อง: ช่องที่มี `allowed_sizes` ต้องตรงขนาดใดขนาดหนึ่ง (`405101` `msg` บอกขนาดที่รับ) · `allowed_sizes = []` = ไม่ตรวจขนาดตามช่อง · โลโก้ไม่ตรวจขนาดตามช่อง
- ส่ง `multipart/form-data` (เช่น `FormData` ของ browser) · ชื่อ field ไฟล์ **`file`** · ไม่แนบ = `422` · field ข้อความอ่านจาก form เท่านั้น (ไม่อ่าน query string)
- **อย่าตั้ง `Content-Type` เอง** ให้ client ใส่ boundary ให้ · multipart ที่เสีย (boundary ผิด) อาจถูกปฏิเสธที่ server ก่อนถึง API — **ไม่ได้ JSON** รูปแบบปกติ
- อัปไม่สำเร็จ = **ไม่มีไฟล์เข้าคลัง** (ไม่ต้องล้างอะไร)

### 1.7 เกมและช่องรูป (ค่าร่าง — ดึงจาก `slots/list` เสมอ)

เกม (`game_code`) เรียงตามระบบ: `scratch_card` · `coin_toss` · `rock_paper_scissors` · ทุกเกมมีช่อง **`game_cover`** (รูปปกเกมในหน้า lobby)

| เกม | ช่อง (เรียงตามระบบ) | ตรวจขนาดภาพ |
|---|---|---|
| `scratch_card` (19) | ตามตารางด้านล่าง | ตรวจ |
| `coin_toss` (11) | `background` · `coin_heads` · `coin_tails` · `selection_button` · `outer_frame` · `bet_limit` · `bet_history` · `winner` · `game_cover` · `loading` · `win_highlight` | ไม่ตรวจ |
| `rock_paper_scissors` (12) | `background` · `hand_rock` · `hand_paper` · `hand_scissors` · `selection_button` · `outer_frame` · `bet_limit` · `bet_history` · `winner` · `game_cover` · `loading` · `win_highlight` | ไม่ตรวจ |

`scratch_card` (กว้าง × สูง · เรียงตามระบบ):

| ช่อง | ขนาด | ช่อง | ขนาด |
|---|---|---|---|
| `background` | 1920 × 1080 | `selection_button` | 115 × 76 |
| `game_cover` | 608 × 1080 | `bet_limit` | 322 × 322 |
| `bg_loading` | 449 × 52 | `bet_history` | 572 × 929 |
| `frame_loading` | 449 × 52 | `bar_bet_history` | 540 × 74 |
| `background_main` | 608 × 1080 | `how_to_scratchers` | 572 × 929 |
| `main_scratch_area` | 544 × 851 | `winner` | 502 × 514 |
| `outer_frame` | 544 × 851 | `you_lose` | 544 × 287 |
| `top_scratch` | 544 × 851 | `win_highlight` | 425 × 425 |
| `main_play` | 186 × 76 | `auto_scratchers` | 520 × 347 |
| `icon_button` | 32 × 32 **หรือ** 40 × 40 | | |

- ชื่อช่องที่แสดงบนจอ **หน้าบ้านแปลเอง** จาก `asset_type` (หลังบ้านไม่ส่งชื่อ / คำอธิบาย)
- ช่องที่เลิกใช้ (`is_retired = true`): ไม่อยู่ใน `slots/list` · อัปเข้าไม่ได้ (`405304`) · `used/list` ยังแสดงถ้ามีรูปค้าง

### 1.8 error ร่วม

**ทุกเส้น** (ไม่เขียนซ้ำในแต่ละเส้น):

| Code | HTTP | เมื่อไหร่ | msg (ไทย) | msg (EN) | หน้าบ้านควรทำ |
|---|---|---|---|---|---|
| `401202` | 401 | ไม่มี / token ผิด / หมดอายุ | กรุณาเข้าสู่ระบบ | Missing or invalid token | หน้า login |
| `401203` | 401 | session ถูกตัด | เซสชันสิ้นสุดแล้ว กรุณาเข้าสู่ระบบใหม่ | Session has ended, please log in again | หน้า login |
| `401301` | 403 | บัญชีถูกล็อก | บัญชีนี้ถูกล็อก กรุณาติดต่อผู้ดูแลระบบ | This account is locked | หน้า login |
| `401302` | 403 | สายต้นสังกัดถูกล็อก | สายบัญชีต้นสังกัดถูกล็อก กรุณาติดต่อผู้ดูแลระบบ | An upline account is locked | หน้า login |
| `401304` | 403 | ยังไม่ตั้ง passcode | กรุณาตั้ง passcode ก่อนใช้งาน | Please set your passcode first | ด่านหลัง login ([auth.md](auth.md)) |
| `401306` | 403 | ต้องเปลี่ยนรหัสผ่าน | กรุณาเปลี่ยนรหัสผ่านก่อนใช้งาน | Please change your password first | ด่านหลัง login |
| `401307` | 403 | ต้องเปลี่ยน passcode | กรุณาเปลี่ยน passcode ก่อนใช้งาน | Please change your passcode first | ด่านหลัง login |
| `401311` | 403 | บัญชีถูกระงับ | บัญชีถูกระงับ ใช้งานได้เฉพาะหน้าประวัติของฉันและรายงาน | This account is suspended, only Profile and Report are available | พาไปหน้า Profile |
| `401312` | 403 | sub ถูกเจ้าของปิด | บัญชีย่อยนี้ถูกปิดใช้งาน กรุณาติดต่อเจ้าของบัญชี | This subaccount is inactive, please contact the account owner | หน้า login |
| `402303` | 200 | ไม่มีสิทธิ์เมนู `asset` / sub มีแค่ view แต่เรียกเส้นเขียน | ไม่มีสิทธิ์ใช้งานเมนูนี้ | You do not have permission for this menu | ซ่อนเมนู / ปุ่ม |
| `400` | 200 | JSON ผิดรูป · ชนิดข้อมูลผิด (เช่นตัวเลขแทนข้อความ) | คำขอไม่ถูกต้อง | Bad request | แสดง msg (ไม่บอก field) |
| `422` | 200 | มี `null` ใน body | `<field>` ห้ามเป็น null (เช่น `game_code ห้ามเป็น null`) | `<field>` must not be null | แก้ request (ห้ามส่ง null) |
| `500` | 500 | ระบบผิดพลาด | เกิดข้อผิดพลาดภายในระบบ | Internal server error | แสดง msg ให้ลองใหม่ |

**เส้นที่ระบุเป้าหมาย** (2.3 – 2.16) เพิ่ม:

| Code | เมื่อไหร่ | msg (ไทย) | msg (EN) |
|---|---|---|---|
| `422` | ไม่ส่ง `scope` / ไม่ใช่ข้อความ | ต้องส่ง scope เป็น "GLOBAL" หรือ "AGENT" | scope must be "GLOBAL" or "AGENT" |
| `422` | `scope` เป็นค่าอื่น (เช่น `"agent"`) | scope ต้องเป็น "GLOBAL" หรือ "AGENT" | scope must be "GLOBAL" or "AGENT" |
| `422` | `scope = AGENT` แต่ไม่ส่ง `agent_id` / ส่ง `0` | scope = AGENT ต้องส่ง agent_id มากกว่า 0 | agent_id is required and must be greater than 0 when scope is AGENT |
| `422` | `scope = GLOBAL` แต่ส่ง `agent_id` (ไม่ใช่ 0) | scope = GLOBAL ห้ามส่ง agent_id | agent_id must not be sent when scope is GLOBAL |
| `422` | `agent_id` ไม่ใช่ตัวเลข (เช่น `"12"` ใน JSON) | agent_id ต้องเป็นตัวเลขจำนวนเต็ม | agent_id must be an integer |
| `422` | เส้น branding / bundle: ไม่ส่ง `agent_id` / ส่ง `0` | ต้องส่ง agent_id มากกว่า 0 | agent_id is required and must be greater than 0 |
| `422` | เส้น branding / bundle: ส่ง `scope` ที่ไม่ใช่ `AGENT` | ไม่ต้องส่ง scope (เส้นนี้ใช้ได้เฉพาะของบัญชี) | scope is not accepted here (this endpoint works on an account only) |
| `402402` | เป้าหมายอยู่นอกสาย / ไม่มีบัญชีนี้ (ตอบเหมือนกัน) | ไม่พบบัญชีในสายของคุณ | Account not found in your downline |
| `405301` | เส้นเขียน: ไม่มีสิทธิ์แก้เป้าหมายนี้ (หัวข้อ 1.3) | ไม่มีสิทธิ์แก้หน้าตาของบัญชีนี้ | You cannot change the look of this account |

ตาราง error ของแต่ละเส้นในหัวข้อ 2 แสดง**เฉพาะ error ของเส้นนั้น** — ให้รวมกับ 2 ตารางนี้เสมอ

## 2. Endpoints

### 2.1 รายชื่อเกม + ปก — `games/list`

**Request:** ไม่มี field (ส่ง `{}` หรือ body ว่าง)

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/asset/games/list" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{}'
```
**Response สำเร็จ** (agent01 เรียก · share01 ตั้งปก `scratch_card` · GLOBAL มีปก `coin_toss`):
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": [
    { "game_code": "scratch_card", "game_cover": "https://cdn.example.com/games/scratch_card/agents/12/game_cover/1791949259206671900-e668a769.png" },
    { "game_code": "coin_toss", "game_cover": "https://cdn.example.com/games/coin_toss/global/game_cover/1791949259194689800-9476d20d.png" },
    { "game_code": "rock_paper_scissors", "game_cover": "" }
  ]
}
```
- ส่ง**ทุกเกม**เสมอ เรียงตามระบบ · **ไม่กรองเกมที่ปิดอยู่** · ไม่แบ่งหน้า · ชื่อเกมที่แสดงเป็นของหน้าบ้าน
- `game_cover` = ปกที่**ไล่สายจากบัญชีที่ login** (เห็นแบบเดียวกับผู้เล่นของตัวเอง) · ไม่มี = `""`
- ดูปกของบัญชีอื่น: ใช้ `used/list` (2.6) แล้วหา item ที่ `asset_type = "game_cover"`

**Error:** มีแค่ error ร่วมทุกเส้น (หัวข้อ 1.8)

### 2.2 ช่องรูปของเกม — `slots/list`

**Request**

| field | ชนิด | บังคับ | กติกา |
|---|---|---|---|
| `game_code` | string | ✓ | หัวข้อ 1.7 |

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/asset/slots/list" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"game_code":"scratch_card"}'
```
**Response สำเร็จ** (`scratch_card` ครบ 19 ช่อง):
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": [
    { "asset_type": "background", "allowed_sizes": [ { "width": 1920, "height": 1080 } ] },
    { "asset_type": "game_cover", "allowed_sizes": [ { "width": 608, "height": 1080 } ] },
    { "asset_type": "bg_loading", "allowed_sizes": [ { "width": 449, "height": 52 } ] },
    { "asset_type": "frame_loading", "allowed_sizes": [ { "width": 449, "height": 52 } ] },
    { "asset_type": "background_main", "allowed_sizes": [ { "width": 608, "height": 1080 } ] },
    { "asset_type": "main_scratch_area", "allowed_sizes": [ { "width": 544, "height": 851 } ] },
    { "asset_type": "outer_frame", "allowed_sizes": [ { "width": 544, "height": 851 } ] },
    { "asset_type": "top_scratch", "allowed_sizes": [ { "width": 544, "height": 851 } ] },
    { "asset_type": "main_play", "allowed_sizes": [ { "width": 186, "height": 76 } ] },
    { "asset_type": "icon_button", "allowed_sizes": [ { "width": 32, "height": 32 }, { "width": 40, "height": 40 } ] },
    { "asset_type": "auto_scratchers", "allowed_sizes": [ { "width": 520, "height": 347 } ] },
    { "asset_type": "selection_button", "allowed_sizes": [ { "width": 115, "height": 76 } ] },
    { "asset_type": "bet_limit", "allowed_sizes": [ { "width": 322, "height": 322 } ] },
    { "asset_type": "bet_history", "allowed_sizes": [ { "width": 572, "height": 929 } ] },
    { "asset_type": "bar_bet_history", "allowed_sizes": [ { "width": 540, "height": 74 } ] },
    { "asset_type": "how_to_scratchers", "allowed_sizes": [ { "width": 572, "height": 929 } ] },
    { "asset_type": "winner", "allowed_sizes": [ { "width": 502, "height": 514 } ] },
    { "asset_type": "you_lose", "allowed_sizes": [ { "width": 544, "height": 287 } ] },
    { "asset_type": "win_highlight", "allowed_sizes": [ { "width": 425, "height": 425 } ] }
  ]
}
```
**Response สำเร็จ** (`coin_toss` — ไม่ตรวจขนาดทุกช่อง):
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": [
    { "asset_type": "background", "allowed_sizes": [] },
    { "asset_type": "coin_heads", "allowed_sizes": [] },
    { "asset_type": "coin_tails", "allowed_sizes": [] },
    { "asset_type": "selection_button", "allowed_sizes": [] },
    { "asset_type": "outer_frame", "allowed_sizes": [] },
    { "asset_type": "bet_limit", "allowed_sizes": [] },
    { "asset_type": "bet_history", "allowed_sizes": [] },
    { "asset_type": "winner", "allowed_sizes": [] },
    { "asset_type": "game_cover", "allowed_sizes": [] },
    { "asset_type": "loading", "allowed_sizes": [] },
    { "asset_type": "win_highlight", "allowed_sizes": [] }
  ]
}
```
- `allowed_sizes = []` = ไม่ตรวจขนาด · ไม่รวมช่องที่เลิกใช้ · ใช้สร้างฟอร์มอัป + ตรวจขนาดภาพก่อนส่ง (หลังบ้านตรวจซ้ำเสมอ)

**Error**

| Code | เมื่อไหร่ | msg (ไทย) | msg (EN) |
|---|---|---|---|
| `422` | ไม่ส่ง / ส่ง `game_code` ว่าง | ต้องส่ง game_code | game_code is required |
| `405401` | ไม่มีเกมนี้ (รวมตัวพิมพ์ใหญ่ เช่น `SCRATCH_CARD`) | ไม่พบเกมหรือช่อง asset | Game or asset slot not found |

### 2.3 อัปรูป — `files/upload` (multipart/form-data)

**Request** (form field ทั้งหมดเป็นข้อความ)

| field | บังคับ | กติกา |
|---|---|---|
| `scope` | ✓ | `GLOBAL` / `AGENT` (หัวข้อ 1.2) |
| `agent_id` | เฉพาะ `AGENT` | id ของบัญชีเป้าหมาย · `GLOBAL` ไม่ส่ง |
| `game_code` | ✓ | หัวข้อ 1.7 |
| `asset_type` | ✓ | ชื่อช่องจาก `slots/list` |
| `use_now` | | `"true"` / `"false"` · **ไม่ส่ง = `"true"`** (ตั้งเป็นรูปที่ใช้ทันที) · ค่าอื่น = `422` |
| `file` | ✓ | ไฟล์รูป (หัวข้อ 1.6) |

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/asset/files/upload" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -F "scope=AGENT" -F "agent_id=12" -F "game_code=scratch_card" -F "asset_type=icon_button" \
  -F "use_now=true" -F "file=@./icon_button.png"
```
GLOBAL (Superadmin):
```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/asset/files/upload" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -F "scope=GLOBAL" -F "game_code=scratch_card" -F "asset_type=win_highlight" -F "file=@./win_highlight.png"
```
**Response สำเร็จ** (sub `share01@design` อัปให้ share01 · `use_now` ไม่ส่ง):
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "config_id": 9001,
    "scope": "AGENT",
    "agent_id": 12,
    "game_code": "scratch_card",
    "asset_type": "icon_button",
    "url": "https://cdn.example.com/games/scratch_card/agents/12/icon_button/1791949259177670200-c402c6d5.png",
    "is_used": true,
    "created_by": "share01@design",
    "created_at": "2026-10-14T10:40:59.180517+07:00"
  }
}
```
**Response สำเร็จ** (GLOBAL):
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "config_id": 120,
    "scope": "GLOBAL",
    "agent_id": 0,
    "game_code": "scratch_card",
    "asset_type": "win_highlight",
    "url": "https://cdn.example.com/games/scratch_card/global/win_highlight/1791949259190673200-e47134b5.png",
    "is_used": true,
    "created_by": "superadmin",
    "created_at": "2026-10-14T10:40:59.191258+07:00"
  }
}
```
- `is_used` = รูปนี้ถูกตั้งเป็นรูปที่ใช้อยู่แล้ว (ตาม `use_now`) · `use_now = "false"` → `is_used: false` เก็บเข้าคลังเฉยๆ แล้วไปเลือกด้วย `used/update`
- เป้าหมาย `GLOBAL` → `agent_id: 0` · `created_by` = username ของคนกด (sub แสดง `owner@sub`)
- **อัปทุกครั้งได้รูปใหม่เข้าคลัง** (ไม่ทับของเก่า · ไม่มีโควตา) · `config_id` ใช้ต่อกับ `used/update` / `files/delete`

**Error** (ลำดับการตรวจ: field → ช่อง → ช่องเลิกใช้ → ขนาดไฟล์ → ชนิดไฟล์ + อ่าน header → เพดาน 4096 px → ขนาดตามช่อง → ที่เก็บไฟล์)

| Code | HTTP | เมื่อไหร่ | msg (ไทย) | msg (EN) |
|---|---|---|---|---|
| `422` | 200 | ไม่ส่ง `game_code` | ต้องส่ง game_code | game_code is required |
| `422` | 200 | ไม่ส่ง `asset_type` | ต้องส่ง asset_type | asset_type is required |
| `422` | 200 | `use_now` ไม่ใช่ `"true"` / `"false"` | use_now ต้องเป็น "true" หรือ "false" | use_now must be "true" or "false" |
| `422` | 200 | ไม่แนบไฟล์ใน field `file` | ต้องแนบไฟล์ใน field file | a file is required in field file |
| `405401` | 200 | ไม่มีเกม / ช่องนี้ | ไม่พบเกมหรือช่อง asset | Game or asset slot not found |
| `405304` | 200 | ช่องเลิกใช้แล้ว | ช่องนี้เลิกใช้แล้ว | This slot is retired |
| `405103` | 200 | ไฟล์ใหญ่เกิน 5 MB | ไฟล์ใหญ่เกิน 5 MB | File is larger than 5 MB |
| `405102` | 200 | ไม่ใช่ png / jpeg / gif / webp (รวม SVG) · หรืออ่านขนาดภาพจาก header ไม่ได้ (ไฟล์เสีย / มีแค่ magic byte) | ชนิดไฟล์ไม่รองรับ (รับ png / jpeg / gif / webp) | Unsupported file type (png, jpeg, gif and webp only) |
| `405101` | 200 | กว้างหรือสูงเกิน 4096 px (ทุกช่อง) | ขนาดภาพต้องไม่เกิน 4096×4096 px | Image must not exceed 4096×4096 px |
| `405101` | 200 | ขนาดภาพไม่ตรงกับ `allowed_sizes` ของช่อง (`msg` บอกขนาดที่รับ) | ขนาดภาพไม่ตรงกับที่ช่องนี้รับ (รับ 32×32, 40×40) | Image size is not accepted by this slot (accepts 32×32, 40×40) |
| `505801` | **503** | ที่เก็บไฟล์ยังไม่พร้อม | ที่เก็บไฟล์ยังไม่พร้อม | File storage is not available |
| `505802` | **502** | อัปขึ้นที่เก็บไฟล์ไม่สำเร็จ (ให้กดอัปใหม่ได้) | อัปโหลดไปที่เก็บไฟล์ไม่สำเร็จ | Uploading to file storage failed |

### 2.4 คลังรูปของช่อง — `files/list`

**Request**

| field | ชนิด | บังคับ | กติกา |
|---|---|---|---|
| `scope` · `agent_id` | | ✓ | หัวข้อ 1.2 |
| `game_code` | string | ✓ | |
| `asset_type` | string | ✓ | |
| `page` | number | | หน้าที่ต้องการ · ไม่ส่ง / น้อยกว่า 1 = 1 |
| `limit` | number | | ต่อหน้า · ไม่ส่ง / น้อยกว่า 1 = 20 · สูงสุด 100 (เกินถูกปรับเป็น 100) |

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/asset/files/list" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" \
  -d '{"scope":"AGENT","agent_id":12,"game_code":"scratch_card","asset_type":"icon_button","page":1,"limit":20}'
```
**Response สำเร็จ:**
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "current_page": 1,
    "total_page": 1,
    "total_count": 3,
    "limit": 20,
    "has_next": false,
    "has_prev": false,
    "data": [
      { "config_id": 9002, "url": "https://cdn.example.com/games/scratch_card/agents/12/icon_button/1791949259185670500-24aebae9.png", "source": "self", "is_used": false, "created_by": "share01", "created_at": "2026-10-14T10:40:59.186261+07:00" },
      { "config_id": 9001, "url": "https://cdn.example.com/games/scratch_card/agents/12/icon_button/1791949259177670200-c402c6d5.png", "source": "self", "is_used": true, "created_by": "share01@design", "created_at": "2026-10-14T10:40:59.180517+07:00" },
      { "config_id": 118, "url": "https://cdn.example.com/games/scratch_card/global/icon_button/1791700000000000000-c9d0e1f2.png", "source": "global", "is_used": false, "created_by": "superadmin", "created_at": "2026-10-01T09:00:00+07:00" }
    ]
  }
}
```
- คลัง = รูปของบัญชีนั้น (`source: "self"`) **รวมรูป GLOBAL** (`source: "global"`) ที่ยังไม่ถูกลบ · เรียง**ใหม่ → เก่า** · เป้าหมาย GLOBAL เห็นเฉพาะรูป GLOBAL
- `is_used` = เป็นรูปที่**เป้าหมายนี้เลือกใช้เอง**อยู่ (รูปที่ได้จากชั้นบนดูที่ `used/list`)
- รูปของชั้นบนที่ไม่ใช่ GLOBAL **ไม่อยู่ในคลัง** (อยากได้ของชั้นบนใช้การไล่สาย — ไม่ต้องเลือก)
- ไม่มีรูปเลย = `"data": []` · `total_count: 0`

**Error**

| Code | เมื่อไหร่ | msg (ไทย) | msg (EN) |
|---|---|---|---|
| `422` | ไม่ส่ง `game_code` | ต้องส่ง game_code | game_code is required |
| `422` | ไม่ส่ง `asset_type` | ต้องส่ง asset_type | asset_type is required |
| `405401` | ไม่มีเกม / ช่องนี้ | ไม่พบเกมหรือช่อง asset | Game or asset slot not found |

### 2.5 ลบรูปจากคลัง — `files/delete`

**Request**

| field | ชนิด | บังคับ | กติกา |
|---|---|---|---|
| `scope` · `agent_id` | | ✓ | หัวข้อ 1.2 |
| `config_id` | number | ✓ | มากกว่า 0 · จาก `files/list` |

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/asset/files/delete" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" \
  -d '{"scope":"AGENT","agent_id":12,"config_id":9002}'
```
**Response สำเร็จ:** `{ "code": 200, "msg": "สำเร็จ" }`

- ลบได้**เฉพาะรูปของเป้าหมายเอง**: GLOBAL ลบรูป GLOBAL (Superadmin) · AGENT ลบรูปของบัญชีนั้น (ลบรูป GLOBAL ด้วย `scope = AGENT` = `405402`)
- ลบแล้วหายจากคลังและเรียกคืนจากหน้าจอไม่ได้
- **ลบรูปที่ถูกเลือกใช้อยู่ไม่ได้** (`405403`) → เลือกรูปอื่น (`used/update`) หรือ `used/reset` ก่อน · รวมรูป GLOBAL ที่มีบัญชีใดเลือกใช้อยู่
- รูปของตัวเองที่สายล่าง**ได้ผ่านการไล่สาย** (ไม่ได้เลือกเอง) **ลบได้** — สายล่างไปใช้ของชั้นบนถัดไป / GLOBAL แทน

**Error**

| Code | เมื่อไหร่ | msg (ไทย) | msg (EN) |
|---|---|---|---|
| `422` | ไม่ส่ง `config_id` / ส่ง `0` | ต้องส่ง config_id มากกว่า 0 | config_id is required and must be greater than 0 |
| `405402` | ไม่พบ / ลบไปแล้ว / ไม่ใช่รูปของเป้าหมาย (ตอบเหมือนกันทุกกรณี) | ไม่พบ asset | Asset not found |
| `405403` | ยังถูกเลือกใช้อยู่ | asset นี้ยังถูกใช้งานอยู่ เลือกรูปอื่นหรือยกเลิกการเลือก (reset) ก่อนลบ | This asset is still in use. Pick another one or reset the selection before deleting |

### 2.6 รูปที่ใช้อยู่จริง — `used/list`

**Request**

| field | ชนิด | บังคับ | กติกา |
|---|---|---|---|
| `scope` · `agent_id` | | ✓ | หัวข้อ 1.2 |
| `game_code` | string | ✓ | |

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/asset/used/list" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" \
  -d '{"scope":"AGENT","agent_id":15,"game_code":"scratch_card"}'
```
**Response สำเร็จ** (agent01 ไม่ได้ตั้งเอง · ได้ปก + `icon_button` จาก share01 · `win_highlight` จาก GLOBAL · ครบ 19 ช่อง):
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "source": "mixed",
    "items": [
      { "asset_type": "background", "config_id": 0, "url": "", "source": "", "inherited_from": "", "is_retired": false },
      { "asset_type": "game_cover", "config_id": 9005, "url": "https://cdn.example.com/games/scratch_card/agents/12/game_cover/1791949259206671900-e668a769.png", "source": "upline", "inherited_from": "share01", "is_retired": false },
      { "asset_type": "bg_loading", "config_id": 0, "url": "", "source": "", "inherited_from": "", "is_retired": false },
      { "asset_type": "frame_loading", "config_id": 0, "url": "", "source": "", "inherited_from": "", "is_retired": false },
      { "asset_type": "background_main", "config_id": 0, "url": "", "source": "", "inherited_from": "", "is_retired": false },
      { "asset_type": "main_scratch_area", "config_id": 0, "url": "", "source": "", "inherited_from": "", "is_retired": false },
      { "asset_type": "outer_frame", "config_id": 0, "url": "", "source": "", "inherited_from": "", "is_retired": false },
      { "asset_type": "top_scratch", "config_id": 0, "url": "", "source": "", "inherited_from": "", "is_retired": false },
      { "asset_type": "main_play", "config_id": 0, "url": "", "source": "", "inherited_from": "", "is_retired": false },
      { "asset_type": "icon_button", "config_id": 9001, "url": "https://cdn.example.com/games/scratch_card/agents/12/icon_button/1791949259177670200-c402c6d5.png", "source": "upline", "inherited_from": "share01", "is_retired": false },
      { "asset_type": "auto_scratchers", "config_id": 0, "url": "", "source": "", "inherited_from": "", "is_retired": false },
      { "asset_type": "selection_button", "config_id": 0, "url": "", "source": "", "inherited_from": "", "is_retired": false },
      { "asset_type": "bet_limit", "config_id": 0, "url": "", "source": "", "inherited_from": "", "is_retired": false },
      { "asset_type": "bet_history", "config_id": 0, "url": "", "source": "", "inherited_from": "", "is_retired": false },
      { "asset_type": "bar_bet_history", "config_id": 0, "url": "", "source": "", "inherited_from": "", "is_retired": false },
      { "asset_type": "how_to_scratchers", "config_id": 0, "url": "", "source": "", "inherited_from": "", "is_retired": false },
      { "asset_type": "winner", "config_id": 0, "url": "", "source": "", "inherited_from": "", "is_retired": false },
      { "asset_type": "you_lose", "config_id": 0, "url": "", "source": "", "inherited_from": "", "is_retired": false },
      { "asset_type": "win_highlight", "config_id": 120, "url": "https://cdn.example.com/games/scratch_card/global/win_highlight/1791949259190673200-e47134b5.png", "source": "global", "inherited_from": "", "is_retired": false }
    ]
  }
}
```
- **ส่งทุกช่องของเกม** เรียงตามระบบ · ช่องที่ไม่มีรูปในสายและ GLOBAL: `config_id: 0` · `url: ""` · `source: ""`
- `source` ต่อช่อง: `self` / `upline` / `global` / `""` · `source` ระดับบนสุด: ทุกช่องที่มีรูปมาจากที่เดียวกัน = ค่านั้น · ไม่เช่นนั้น `mixed` · ไม่มีรูปเลย `""`
- `scope = GLOBAL`: ทุกช่องที่มีรูปเป็น `global` · ไม่มีรูปเลย = `"source": ""`
- ช่องที่ `source = "upline"`: `config_id` เป็นรูปของชั้นบน — ใช้กับ `used/update` / `files/delete` ของบัญชีนี้ไม่ได้ (`405402`) · ช่องที่ `source = "global"` เลือกรูปนั้นมาเป็นของตัวเองได้ (`used/update`)
- หน้าจอแนะนำ: รูปที่ใช้อยู่ + ป้าย "ตั้งเอง / ได้จาก {inherited_from} / ค่ากลาง / ยังไม่มีรูป" + ปุ่ม "เลือกจากคลัง" (`files/list` → `used/update`) และ "กลับไปใช้ของชั้นบน" (`used/reset` — แสดงเมื่อ `source = "self"`)

**Error**

| Code | เมื่อไหร่ | msg (ไทย) | msg (EN) |
|---|---|---|---|
| `422` | ไม่ส่ง `game_code` | ต้องส่ง game_code | game_code is required |
| `405401` | ไม่มีเกมนี้ | ไม่พบเกมหรือช่อง asset | Game or asset slot not found |

### 2.7 เลือกรูปจากคลัง — `used/update`

**Request**

| field | ชนิด | บังคับ | กติกา |
|---|---|---|---|
| `scope` · `agent_id` | | ✓ | หัวข้อ 1.2 |
| `game_code` | string | ✓ | เกมของรูป |
| `config_id` | number | ✓ | มากกว่า 0 · จาก `files/list` (ช่องถูกระบุจากรูปเอง) |

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/asset/used/update" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" \
  -d '{"scope":"AGENT","agent_id":12,"game_code":"scratch_card","config_id":120}'
```
**Response สำเร็จ:** `{ "code": 200, "msg": "สำเร็จ" }`

- เลือกได้**รูปของตัวเอง หรือรูป GLOBAL** · เป้าหมาย GLOBAL เลือกได้เฉพาะรูป GLOBAL · เลือกรูปใหม่ = แทนรูปเดิมของช่องนั้น (รูปเดิมยังอยู่ในคลัง)

**Error** (ลำดับ: field → เกม → รูป → ช่องเลิกใช้)

| Code | เมื่อไหร่ | msg (ไทย) | msg (EN) |
|---|---|---|---|
| `422` | ไม่ส่ง `game_code` | ต้องส่ง game_code | game_code is required |
| `422` | ไม่ส่ง `config_id` / ส่ง `0` | ต้องส่ง config_id มากกว่า 0 | config_id is required and must be greater than 0 |
| `405401` | ไม่มีเกมนี้ | ไม่พบเกมหรือช่อง asset | Game or asset slot not found |
| `405402` | รูปไม่พบ / ลบแล้ว / ไม่ใช่รูปที่เป้าหมายเลือกได้ / เป็นของคนละเกม (ตอบเหมือนกันทุกกรณี) | ไม่พบ asset | Asset not found |
| `405304` | ช่องของรูปเลิกใช้แล้ว | ช่องนี้เลิกใช้แล้ว | This slot is retired |

### 2.8 กลับไปใช้รูปของชั้นบน — `used/reset`

**Request**

| field | ชนิด | บังคับ | กติกา |
|---|---|---|---|
| `scope` · `agent_id` | | ✓ | **`scope` ต้องเป็น `AGENT`** |
| `game_code` | string | ✓ | |
| `asset_type` | string | ✓ | |

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/asset/used/reset" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" \
  -d '{"scope":"AGENT","agent_id":12,"game_code":"scratch_card","asset_type":"icon_button"}'
```
**Response สำเร็จ:** `{ "code": 200, "msg": "สำเร็จ" }`

- ยกเลิกรูปที่เลือกไว้ของช่องนี้ → ใช้ของชั้นบน / GLOBAL · **ไม่ลบรูปจากคลัง** · ยังไม่ได้เลือกไว้ = 200 (ไม่เปลี่ยนอะไร)
- ซ่อนปุ่มนี้เมื่อดูค่ากลาง (GLOBAL)

**Error**

| Code | เมื่อไหร่ | msg (ไทย) | msg (EN) |
|---|---|---|---|
| `422` | `scope = GLOBAL` | reset ใช้ได้เฉพาะ scope AGENT (ค่ากลางไม่มีชั้นบนให้กลับไปใช้) | reset only works on scope AGENT (the global value has nothing above it) |
| `422` | ไม่ส่ง `game_code` | ต้องส่ง game_code | game_code is required |
| `422` | ไม่ส่ง `asset_type` | ต้องส่ง asset_type | asset_type is required |
| `405401` | ไม่มีเกม / ช่องนี้ | ไม่พบเกมหรือช่อง asset | Game or asset slot not found |

### 2.9 ธีม — `theme/detail/get`

**Request**

| field | ชนิด | บังคับ |
|---|---|---|
| `scope` · `agent_id` | | ✓ |
| `game_code` | string | ✓ |

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/asset/theme/detail/get" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" \
  -d '{"scope":"AGENT","agent_id":15,"game_code":"scratch_card"}'
```
**Response สำเร็จ** (agent01 ได้ธีมจาก share01):
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "source": "upline",
    "inherited_from": "share01",
    "font_family": "kanit",
    "main_color_1": "#1e90ff",
    "main_color_2": "#ffffff",
    "stats_bar_color": "main_color_2",
    "stats_bar_color_hex": "#ffffff",
    "stats_bar_font_family": "prompt",
    "main_color_1_shades": {
      "50": "#e9f4ff", "100": "#d2e9ff", "200": "#bcdeff", "300": "#a5d3ff", "400": "#8fc8ff",
      "500": "#78bcff", "600": "#62b1ff", "700": "#4ba6ff", "800": "#359bff", "900": "#1e90ff",
      "1000": "#1b82e6", "1100": "#1873cc", "1200": "#1565b3", "1300": "#125699", "1400": "#0f4880",
      "1500": "#0c3a66", "1600": "#092b4d", "1700": "#061d33", "1800": "#030e1a"
    },
    "main_color_2_shades": {
      "50": "#ffffff", "100": "#ffffff", "200": "#ffffff", "300": "#ffffff", "400": "#ffffff",
      "500": "#ffffff", "600": "#ffffff", "700": "#ffffff", "800": "#ffffff", "900": "#ffffff",
      "1000": "#e6e6e6", "1100": "#cccccc", "1200": "#b3b3b3", "1300": "#999999", "1400": "#808080",
      "1500": "#666666", "1600": "#4d4d4d", "1700": "#333333", "1800": "#1a1a1a"
    }
  }
}
```
**Response สำเร็จ** (ยังไม่มีใครตั้ง — ค่าเริ่มต้นของระบบ):
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "source": "global",
    "inherited_from": "",
    "font_family": "noto",
    "main_color_1": "#f7447e",
    "main_color_2": "#ffffff",
    "stats_bar_color": "main_color_1",
    "stats_bar_color_hex": "#f7447e",
    "stats_bar_font_family": "prompt",
    "main_color_1_shades": {
      "50": "#feecf2", "100": "#fddae5", "200": "#fdc7d8", "300": "#fcb4cb", "400": "#fba2bf",
      "500": "#fa8fb2", "600": "#f97ca5", "700": "#f96998", "800": "#f8578b", "900": "#f7447e",
      "1000": "#de3d71", "1100": "#c63665", "1200": "#ad3058", "1300": "#94294c", "1400": "#7c223f",
      "1500": "#631b32", "1600": "#4a1426", "1700": "#310e19", "1800": "#19070d"
    },
    "main_color_2_shades": {
      "50": "#ffffff", "100": "#ffffff", "200": "#ffffff", "300": "#ffffff", "400": "#ffffff",
      "500": "#ffffff", "600": "#ffffff", "700": "#ffffff", "800": "#ffffff", "900": "#ffffff",
      "1000": "#e6e6e6", "1100": "#cccccc", "1200": "#b3b3b3", "1300": "#999999", "1400": "#808080",
      "1500": "#666666", "1600": "#4d4d4d", "1700": "#333333", "1800": "#1a1a1a"
    }
  }
}
```
- ธีมเป็นของ**ต่อเกม** · ไล่สายแล้ว · `source`: `self` / `upline` / `global`
- `stats_bar_color` เป็น**ชื่อสี** (`main_color_1` / `main_color_2`) ไม่ใช่ hex — hex ที่ใช้จริงคือ `stats_bar_color_hex`
- shades 19 ระดับ (`50`, `100` … `1800`) คำนวณจากสีหลักให้แล้ว · `900` = สีหลักเอง · **ลำดับ key ใน JSON ไม่แน่นอน** ให้อ่านตาม key

**Error**

| Code | เมื่อไหร่ | msg (ไทย) | msg (EN) |
|---|---|---|---|
| `422` | ไม่ส่ง `game_code` | ต้องส่ง game_code | game_code is required |
| `405401` | ไม่มีเกมนี้ | ไม่พบเกมหรือช่อง asset | Game or asset slot not found |

### 2.10 ตั้งธีม — `theme/detail/update`

**Request** (ส่งครบทุก field ทุกครั้ง = แทนทั้งชุด)

| field | ชนิด | บังคับ | ค่าที่รับ |
|---|---|---|---|
| `scope` · `agent_id` | | ✓ | หัวข้อ 1.2 |
| `game_code` | string | ✓ | |
| `font_family` | string | ✓ | `noto` · `prompt` · `kanit` · `mitr` · `sarabun` (ตัวเล็กตรงตัว) |
| `main_color_1` | string | ✓ | `#RRGGBB` (ไม่รับ `#RGB`) |
| `main_color_2` | string | ✓ | `#RRGGBB` |
| `stats_bar_color` | string | ✓ | `main_color_1` / `main_color_2` |
| `stats_bar_font_family` | string | ✓ | เหมือน `font_family` |

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/asset/theme/detail/update" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" \
  -d '{"scope":"AGENT","agent_id":12,"game_code":"scratch_card","font_family":"kanit","main_color_1":"#1E90FF","main_color_2":"#FFFFFF","stats_bar_color":"main_color_2","stats_bar_font_family":"prompt"}'
```
Request:
```json
{
  "scope": "AGENT",
  "agent_id": 12,
  "game_code": "scratch_card",
  "font_family": "kanit",
  "main_color_1": "#1E90FF",
  "main_color_2": "#FFFFFF",
  "stats_bar_color": "main_color_2",
  "stats_bar_font_family": "prompt"
}
```
**Response สำเร็จ:** `{ "code": 200, "msg": "สำเร็จ" }`

- ไม่มีการแก้ทีละค่า — ฟอร์มต้องส่งค่าปัจจุบัน (จาก `theme/detail/get`) ของ field ที่ไม่ได้แก้มาด้วย
- ส่งค่าเท่ากับที่ใช้อยู่ทั้งชุด = `200` แต่**ไม่บันทึกอะไร** → บัญชีที่ยังไม่เคยตั้ง กดบันทึกโดยไม่แก้ = ยังได้ค่าของชั้นบนต่อไป (อยาก "ปักค่า" ต้องแก้อย่างน้อย 1 ค่า)
- ตรวจ field ตามลำดับ `font_family` → `main_color_1` → `main_color_2` → `stats_bar_color` → `stats_bar_font_family` — แจ้งทีละ field

**Error**

| Code | เมื่อไหร่ | msg (ไทย) | msg (EN) |
|---|---|---|---|
| `422` | ไม่ส่ง `game_code` | ต้องส่ง game_code | game_code is required |
| `422` | ไม่ส่ง / ส่ง `font_family` ว่าง | ต้องส่ง font_family (ธีมต้องส่งครบทั้ง 5 ค่า) | font_family is required (send all 5 theme values) |
| `422` | ไม่ส่ง / ส่ง `main_color_1` ว่าง | ต้องส่ง main_color_1 (ธีมต้องส่งครบทั้ง 5 ค่า) | main_color_1 is required (send all 5 theme values) |
| `422` | ไม่ส่ง / ส่ง `main_color_2` ว่าง | ต้องส่ง main_color_2 (ธีมต้องส่งครบทั้ง 5 ค่า) | main_color_2 is required (send all 5 theme values) |
| `422` | ไม่ส่ง / ส่ง `stats_bar_color` ว่าง | ต้องส่ง stats_bar_color (ธีมต้องส่งครบทั้ง 5 ค่า) | stats_bar_color is required (send all 5 theme values) |
| `422` | ไม่ส่ง / ส่ง `stats_bar_font_family` ว่าง | ต้องส่ง stats_bar_font_family (ธีมต้องส่งครบทั้ง 5 ค่า) | stats_bar_font_family is required (send all 5 theme values) |
| `422` | ฟอนต์ไม่อยู่ในรายการ (รวมตัวใหญ่ เช่น `Kanit`) | font_family ต้องเป็นหนึ่งใน noto, prompt, kanit, mitr, sarabun | font_family must be one of noto, prompt, kanit, mitr, sarabun |
| `422` | ฟอนต์แถบสถิติไม่อยู่ในรายการ | stats_bar_font_family ต้องเป็นหนึ่งใน noto, prompt, kanit, mitr, sarabun | stats_bar_font_family must be one of noto, prompt, kanit, mitr, sarabun |
| `422` | `main_color_1` ผิดรูปแบบ (เช่น `#FFF`, `red`) | main_color_1 ต้องเป็นสีรูปแบบ #RRGGBB | main_color_1 must be a color in #RRGGBB format |
| `422` | `main_color_2` ผิดรูปแบบ | main_color_2 ต้องเป็นสีรูปแบบ #RRGGBB | main_color_2 must be a color in #RRGGBB format |
| `422` | `stats_bar_color` ไม่ใช่ชื่อสี (เช่นส่ง hex) | stats_bar_color ต้องเป็น main_color_1 หรือ main_color_2 | stats_bar_color must be main_color_1 or main_color_2 |
| `405401` | ไม่มีเกมนี้ | ไม่พบเกมหรือช่อง asset | Game or asset slot not found |

### 2.11 กลับไปใช้ธีมของชั้นบน — `theme/reset`

**Request**

| field | ชนิด | บังคับ | กติกา |
|---|---|---|---|
| `scope` · `agent_id` | | ✓ | **`scope` ต้องเป็น `AGENT`** |
| `game_code` | string | ✓ | |

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/asset/theme/reset" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" \
  -d '{"scope":"AGENT","agent_id":12,"game_code":"scratch_card"}'
```
**Response สำเร็จ:** `{ "code": 200, "msg": "สำเร็จ" }` · ลบธีมที่ตั้งเองของเกมนั้น → ใช้ของชั้นบน / GLOBAL · ยังไม่ได้ตั้ง = 200

**Error**

| Code | เมื่อไหร่ | msg (ไทย) | msg (EN) |
|---|---|---|---|
| `422` | `scope = GLOBAL` | reset ใช้ได้เฉพาะ scope AGENT (ค่ากลางไม่มีชั้นบนให้กลับไปใช้) | reset only works on scope AGENT (the global value has nothing above it) |
| `422` | ไม่ส่ง `game_code` | ต้องส่ง game_code | game_code is required |
| `405401` | ไม่มีเกมนี้ | ไม่พบเกมหรือช่อง asset | Game or asset slot not found |

### 2.12 Branding — `branding/detail/get`

**Request**

| field | ชนิด | บังคับ |
|---|---|---|
| `agent_id` | number | ✓ |

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/asset/branding/detail/get" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"agent_id":15}'
```
**Response สำเร็จ** (agent01 ได้จาก share01):
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "source": "upline",
    "inherited_from": "share01",
    "site_name": "Share One",
    "logo": "https://cdn.example.com/sites/12/logo/1791949267225017200-f68db11d.png"
  }
}
```
**Response สำเร็จ** (ไม่มีใครในสายตั้ง):
```json
{ "code": 200, "msg": "สำเร็จ", "data": { "source": "", "inherited_from": "", "site_name": "", "logo": "" } }
```
- ไล่สายแล้ว · `source`: `self` / `upline` / `""` · **ไม่มีค่ากลาง (GLOBAL)** · branding เป็นของบัญชี ไม่แยกเกม
- **ไล่ทั้งชุด ไม่ผสมคนละชั้น:** บัญชีที่ตั้งชื่ออย่างเดียว (ยังไม่อัปโลโก้) ได้ `"logo": ""` — **ไม่**ดึงโลโก้ของชั้นบนมาเติม → ฟอร์มควรให้ตั้งชื่อและโลโก้คู่กัน

**Error:** มีแค่ error ร่วม (หัวข้อ 1.8)

### 2.13 แก้ชื่อเว็บ — `branding/detail/update`

**Request**

| field | ชนิด | บังคับ | กติกา |
|---|---|---|---|
| `agent_id` | number | ✓ | |
| `site_name` | string | ✓ | ตัดช่องว่างหน้า-หลังแล้วต้องยาว 1–50 ตัวอักษร · ห้ามอักขระควบคุม (ขึ้นบรรทัดใหม่ / tab) |
| `site_theme` | | **ห้ามส่ง** | ยังไม่เปิดใช้ |

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/asset/branding/detail/update" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"agent_id":12,"site_name":"Share One"}'
```
**Response สำเร็จ:** `{ "code": 200, "msg": "สำเร็จ" }`

- ระบบตัดช่องว่างหน้า-หลังให้ (`"  Share One  "` เก็บเป็น `"Share One"`)
- บัญชีที่ยังไม่เคยตั้ง: แก้ชื่อแล้วระบบสร้าง branding ให้ (โลโก้ว่างจนกว่าจะอัป) · อยากล้างชื่อใช้ `branding/reset`

**Error**

| Code | เมื่อไหร่ | msg (ไทย) | msg (EN) |
|---|---|---|---|
| `422` | ไม่ส่ง / ว่าง / ช่องว่างล้วน | ต้องส่ง site_name (1–50 ตัวอักษร) | site_name is required (1–50 characters) |
| `422` | ยาวเกิน 50 ตัวอักษร | site_name ยาวเกิน 50 ตัวอักษร | site_name must not exceed 50 characters |
| `422` | มีอักขระควบคุม | site_name ห้ามมีอักขระควบคุม (เช่น ขึ้นบรรทัดใหม่ / tab) | site_name must not contain control characters |
| `422` | ส่ง `site_theme` | site_theme ยังไม่เปิดใช้งาน | site_theme is not supported yet |
| `400` | `site_name` ไม่ใช่ข้อความ (เช่น `123`) | คำขอไม่ถูกต้อง | Bad request |

### 2.14 อัปโลโก้ — `branding/logo/upload` (multipart/form-data)

**Request**

| field | บังคับ | กติกา |
|---|---|---|
| `agent_id` | ✓ | ข้อความตัวเลข |
| `file` | ✓ | ไฟล์รูป (หัวข้อ 1.6) · ไม่ตรวจขนาดตามช่อง แต่ต้องอ่านขนาดได้และ**ไม่เกิน 4096×4096 px** |

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/asset/branding/logo/upload" \
  -H "Authorization: Bearer {{TOKEN}}" -F "agent_id=12" -F "file=@./logo.png"
```
**Response สำเร็จ:**
```json
{ "code": 200, "msg": "สำเร็จ", "data": { "url": "https://cdn.example.com/sites/12/logo/1791949267225017200-f68db11d.png" } }
```
- **เขียนทับ**โลโก้เดิมของบัญชีนั้น (ไม่มีคลัง / ไม่มี `use_now`) · `url` เปลี่ยนทุกครั้งที่อัป — ใช้ค่าใหม่แสดงทันที

**Error** (ลำดับ: ขนาดไฟล์ → ชนิดไฟล์ + อ่าน header → เพดาน 4096 px → ที่เก็บไฟล์)

| Code | HTTP | เมื่อไหร่ | msg (ไทย) | msg (EN) |
|---|---|---|---|---|
| `422` | 200 | ไม่แนบไฟล์ใน field `file` | ต้องแนบไฟล์ใน field file | a file is required in field file |
| `405103` | 200 | ไฟล์ใหญ่เกิน 5 MB | ไฟล์ใหญ่เกิน 5 MB | File is larger than 5 MB |
| `405102` | 200 | ไม่ใช่ png / jpeg / gif / webp · หรืออ่านขนาดภาพจาก header ไม่ได้ | ชนิดไฟล์ไม่รองรับ (รับ png / jpeg / gif / webp) | Unsupported file type (png, jpeg, gif and webp only) |
| `405101` | 200 | กว้างหรือสูงเกิน 4096 px | ขนาดภาพต้องไม่เกิน 4096×4096 px | Image must not exceed 4096×4096 px |
| `505801` | **503** | ที่เก็บไฟล์ยังไม่พร้อม | ที่เก็บไฟล์ยังไม่พร้อม | File storage is not available |
| `505802` | **502** | อัปขึ้นที่เก็บไฟล์ไม่สำเร็จ | อัปโหลดไปที่เก็บไฟล์ไม่สำเร็จ | Uploading to file storage failed |

### 2.15 กลับไปใช้ branding ของชั้นบน — `branding/reset`

**Request**

| field | ชนิด | บังคับ |
|---|---|---|
| `agent_id` | number | ✓ |

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/asset/branding/reset" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"agent_id":12}'
```
**Response สำเร็จ:** `{ "code": 200, "msg": "สำเร็จ" }` · ล้างทั้งชื่อเว็บและโลโก้ที่ตั้งเอง → ใช้ของชั้นบน (ไม่มี = ค่าว่างทุก field) · ยังไม่ได้ตั้ง = 200

**Error:** มีแค่ error ร่วม (หัวข้อ 1.8)

### 2.16 Preview — `bundle/detail/get`

**Request**

| field | ชนิด | บังคับ |
|---|---|---|
| `agent_id` | number | ✓ |
| `game_code` | string | ✓ (เกมเดียวต่อคำขอ) |

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/asset/bundle/detail/get" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"agent_id":15,"game_code":"scratch_card"}'
```
**Response สำเร็จ** (agent01 · ธีมจาก share01 · มีชื่อเว็บ + โลโก้จาก share01):
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "source": "mixed",
    "assets": {
      "game_cover": "https://cdn.example.com/games/scratch_card/agents/12/game_cover/1791949259206671900-e668a769.png",
      "icon_button": "https://cdn.example.com/games/scratch_card/agents/12/icon_button/1791949259177670200-c402c6d5.png",
      "win_highlight": "https://cdn.example.com/games/scratch_card/global/win_highlight/1791949259190673200-e47134b5.png"
    },
    "theme": {
      "source": "upline",
      "font_family": "kanit",
      "main_color_1": "#1e90ff",
      "main_color_2": "#ffffff",
      "stats_bar_color": "main_color_2",
      "stats_bar_color_hex": "#ffffff",
      "stats_bar_font_family": "prompt",
      "main_color_1_shades": {
        "50": "#e9f4ff", "100": "#d2e9ff", "200": "#bcdeff", "300": "#a5d3ff", "400": "#8fc8ff",
        "500": "#78bcff", "600": "#62b1ff", "700": "#4ba6ff", "800": "#359bff", "900": "#1e90ff",
        "1000": "#1b82e6", "1100": "#1873cc", "1200": "#1565b3", "1300": "#125699", "1400": "#0f4880",
        "1500": "#0c3a66", "1600": "#092b4d", "1700": "#061d33", "1800": "#030e1a"
      },
      "main_color_2_shades": {
        "50": "#ffffff", "100": "#ffffff", "200": "#ffffff", "300": "#ffffff", "400": "#ffffff",
        "500": "#ffffff", "600": "#ffffff", "700": "#ffffff", "800": "#ffffff", "900": "#ffffff",
        "1000": "#e6e6e6", "1100": "#cccccc", "1200": "#b3b3b3", "1300": "#999999", "1400": "#808080",
        "1500": "#666666", "1600": "#4d4d4d", "1700": "#333333", "1800": "#1a1a1a"
      }
    },
    "site": {
      "site_name": "Share One",
      "logo": "https://cdn.example.com/sites/12/logo/1791949267225017200-f68db11d.png"
    }
  }
}
```
**Response สำเร็จ** (เกมที่ไม่มีรูปเลย · ไม่มี branding · ธีมค่ากลาง — shades ตามตัวอย่างค่าเริ่มต้นใน 2.9):
```json
{
  "code": 200,
  "msg": "สำเร็จ",
  "data": {
    "source": "",
    "assets": {},
    "theme": {
      "source": "global",
      "font_family": "noto",
      "main_color_1": "#f7447e",
      "main_color_2": "#ffffff",
      "stats_bar_color": "main_color_1",
      "stats_bar_color_hex": "#f7447e",
      "stats_bar_font_family": "prompt",
      "main_color_1_shades": {
        "50": "#feecf2", "100": "#fddae5", "200": "#fdc7d8", "300": "#fcb4cb", "400": "#fba2bf",
        "500": "#fa8fb2", "600": "#f97ca5", "700": "#f96998", "800": "#f8578b", "900": "#f7447e",
        "1000": "#de3d71", "1100": "#c63665", "1200": "#ad3058", "1300": "#94294c", "1400": "#7c223f",
        "1500": "#631b32", "1600": "#4a1426", "1700": "#310e19", "1800": "#19070d"
      },
      "main_color_2_shades": {
        "50": "#ffffff", "100": "#ffffff", "200": "#ffffff", "300": "#ffffff", "400": "#ffffff",
        "500": "#ffffff", "600": "#ffffff", "700": "#ffffff", "800": "#ffffff", "900": "#ffffff",
        "1000": "#e6e6e6", "1100": "#cccccc", "1200": "#b3b3b3", "1300": "#999999", "1400": "#808080",
        "1500": "#666666", "1600": "#4d4d4d", "1700": "#333333", "1800": "#1a1a1a"
      }
    },
    "site": { "site_name": "", "logo": "" }
  }
}
```
- เป็นชุดเดียวกับที่**ผู้เล่นของบัญชีนี้จะได้รับ** (asset + ธีม + branding ไล่สายแล้ว) — ใช้ทำหน้า preview
- `assets` = map `asset_type → URL` **เฉพาะช่องที่มีรูป** · ไม่มีรูปเลย = `{}` · ลำดับ key ไม่แน่นอน
- `theme` เหมือน 2.9 แต่**ไม่มี `inherited_from`** · `site` ไม่มี `source` / `inherited_from`
- เส้นผู้เล่นในอนาคตใช้รูปแบบ `data` เดียวกันนี้

**Error**

| Code | เมื่อไหร่ | msg (ไทย) | msg (EN) |
|---|---|---|---|
| `422` | ไม่ส่ง `game_code` | ต้องส่ง game_code | game_code is required |
| `405401` | ไม่มีเกมนี้ | ไม่พบเกมหรือช่อง asset | Game or asset slot not found |

## 3. สรุป error code ของ module นี้

ข้อความเต็มแต่ละกรณีอยู่ในตารางของแต่ละเส้น (หัวข้อ 2) และ error ร่วม (หัวข้อ 1.8)

| Code | HTTP | ความหมาย | เส้นที่เกิด | หน้าบ้านควรทำ |
|---|---|---|---|---|
| `400` | 200 | คำขอไม่ถูกต้อง (JSON ผิดรูป / ชนิดข้อมูลผิด) — ไม่บอก field | ทุกเส้นที่มี body | แสดง msg |
| `422` | 200 | ข้อมูลไม่ผ่าน (`msg` บอก field) | ทุกเส้นยกเว้น 2.1 | แสดง msg ที่ field |
| `402303` | 200 | ไม่มีสิทธิ์เมนู `asset` | ทุกเส้น | ซ่อนเมนู / ปุ่ม |
| `402402` | 200 | ไม่พบบัญชีในสายของคุณ | 2.3 – 2.16 | แสดง msg |
| `405101` | 200 | ขนาดภาพไม่ตรงกับที่ช่องนี้รับ (`msg` บอกขนาดที่รับ) · หรือเกิน 4096×4096 px | 2.3 · 2.14 | แสดง msg ให้เลือกรูปใหม่ |
| `405102` | 200 | ชนิดไฟล์ไม่รองรับ (รวมไฟล์ที่อ่าน header ภาพไม่ได้) | 2.3 · 2.14 | แสดง msg |
| `405103` | 200 | ไฟล์ใหญ่เกิน 5 MB | 2.3 · 2.14 | แสดง msg (ควรกันก่อนส่ง) |
| `405301` | 200 | ไม่มีสิทธิ์แก้หน้าตาของบัญชีนี้ | เส้นเขียน 2.3 · 2.5 · 2.7 · 2.8 · 2.10 · 2.11 · 2.13 – 2.15 | แสดง msg · ซ่อนปุ่มแก้ล่วงหน้า (1.3) |
| `405304` | 200 | ช่องนี้เลิกใช้แล้ว | 2.3 · 2.7 | โหลด `slots/list` / `used/list` ใหม่ |
| `405401` | 200 | ไม่พบเกมหรือช่อง asset | ทุกเส้นที่ส่ง `game_code` / `asset_type` | แสดง msg |
| `405402` | 200 | ไม่พบ asset | 2.5 · 2.7 | โหลดคลังใหม่ |
| `405403` | 200 | asset ยังถูกใช้งานอยู่ | 2.5 | แสดง msg · ชี้ให้เลือกรูปอื่น / reset ก่อน |
| `505801` | **503** | ที่เก็บไฟล์ยังไม่พร้อม | 2.3 · 2.14 | แสดง msg · ลองใหม่ภายหลัง |
| `505802` | **502** | อัปโหลดไปที่เก็บไฟล์ไม่สำเร็จ | 2.3 · 2.14 | ให้กดอัปใหม่ได้ |
| `500` | **500** | เกิดข้อผิดพลาดภายในระบบ | ทุกเส้น | แสดง msg |

## 4. สิ่งที่ยังไม่มี / จะเปลี่ยน

- **เส้นผู้เล่น** `POST /api/v1/player/asset/bundle` (ผู้เล่นโหลดรูป ธีม branding) — รอ Player Auth · `data` รูปแบบเดียวกับ 2.16
- **เส้นรายชื่อเกมของ lobby ฝั่งผู้เล่น:** ตัดสินพร้อม review Player Auth
- **เลื่อน (ไม่ต้องทำหน้าจอ):** banner · favicon · `site_theme` · ประวัติการแก้ (change log)
- ชื่อช่อง / ขนาดภาพ: รอทีมเกมยืนยัน — ถ้าเปลี่ยน หลังบ้านแก้ catalog ไม่ต้องแก้ path (หน้าบ้านดึงจาก `slots/list`)
- เส้นทั้งหมดเป็นเส้นใหม่ (ทุกเส้น `POST` · `game_type` → `game_code` · ไม่มี `?agent_id=`) — ถ้าเคยเรียก path ของระบบเก่า ให้เปลี่ยนตามหัวข้อ 1.1
