# API หลังบ้าน — การจัดการสมาชิก (สำหรับหน้าบ้าน) — 9 ต.ค. 2026 (โครงเส้นใหม่)

เอกสารนี้ตรงกับโค้ดใน branch `boiledegg/bo/agent_management` (PR 2 — **เส้นใหม่ตาม lead review 9 ต.ค.** ใช้บน server หลัง merge + deploy) ·
spec เต็ม: `docs/modules/agent_management.md` · พื้นฐานเรื่อง login / session / ด่านหลัง login ดู `docs/frontend/auth.md`

> **เปลี่ยนจากเดิม (9 ต.ค.)** — path ใหม่ทั้งหมด เส้นเดิมได้ `404` (หัวข้อ 5) · `downlines/search` รวมเข้า `downlines/list` ด้วย `keyword` ·
> แก้บัญชีเหลือเส้นเดียว `agents/detail/update` · `copy-sources` → `agents/list` · ไม่มีเมนูสิทธิ์ `pt` แล้ว (`pt` แสดงเสมอ) ·
> Share Reseller / Share Master สร้าง Agent ไม่ได้ · **ยังไม่มีในรอบนี้ (PR 3):** เบอร์ 2 field `phone_country_code` + `phone` · ฟอร์ม Share Master ส่งแค่ Commission

เส้นของ Member อยู่ที่ [member_management.md](member_management.md) · เส้นค้นหาบัญชีของ ADMIN อยู่ที่ [admin_management.md](admin_management.md)

## 1. พื้นฐาน

### 1.1 Base URL และ header

| เรื่อง | ค่า |
|---|---|
| prefix | `/api/v1/bo/pr/manage/...` (ต้อง login ทุกเส้น) |
| header | `Authorization: Bearer <token>` · `Content-Type: application/json` |
| ภาษาของ `msg` | header `X-Lang: en` → อังกฤษ · ไม่ส่ง = ไทย |

### 1.2 รูปแบบ response

```json
{ "code": 200, "msg": "สำเร็จ", "data": { } }
```

- `code = 200` = สำเร็จ · อย่างอื่น = error (หัวข้อ 4) · เส้นที่ไม่มีข้อมูลส่งกลับ **ไม่มี key `data`**
- error ของ module นี้ตอบ HTTP 200 และดูที่ `code` · `422` = ข้อมูลไม่ผ่าน `msg` บอกว่าผิดที่ field ไหน

### 1.3 กติกาของ module นี้

| เรื่อง | กติกา |
|---|---|
| ใครเรียก | มาจาก token เสมอ |
| บัญชีที่จะดู / แก้ | ส่ง `id` ใน body · หลังบ้านเช็คว่าอยู่ในสายของผู้เรียก |
| method | `POST` ทุกเส้น |
| `null` | ห้ามส่ง (`422`) · ไม่มีค่า: ข้อความ `""` · ตัวเลข `0` · รายการ `[]` · object `{}` |
| % | JSON number ทศนิยมไม่เกิน 4 ตำแหน่ง · PT / Force / Remain ทีละ `0.5` · Commission ทีละ `0.1` ไม่เกิน `1` |
| เงิน | JSON number ทศนิยมไม่เกิน 4 ตำแหน่ง · response ส่งตามที่เก็บ (ตัด 0 ท้าย) **หน้าบ้านปัด 3 ตำแหน่งตอนแสดงเอง** |
| `status` ใน response | สถานะที่ใช้งานจริง — หัวสายถูกระงับ ลูกทุกชั้นแสดง `SUSPENDED` |
| sub | ทุกเส้นใช้เมนู `member`: ดู = `view` · สร้าง / แก้ / สถานะ = `edit` · ส่ง `balance` ต้องมี `payment` = `edit` เพิ่ม · ไม่มี = `402303` |

### 1.4 ค่าหุ้นส่วน `pt`

key ของระบบคือ `minigame` · **แสดงเสมอ** (ไม่มีเมนูสิทธิ์ `pt` แล้ว)

**ตอนส่ง (สร้าง / `detail/update`)** — ชุดเดียวต่อระบบ หลังบ้านกระจายลงทุกเกมเอง:
```json
"pt": { "minigame": { "pt_from_parent": 70, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true } },
"status_game": { "scratch_card": false }
```

**ตอนแสดง** — ชุดเดียวต่อระบบ + วันที่ / คนสร้าง / คนแก้ล่าสุด · `status_game` แยกต่อเกม:
```json
"pt": {
  "minigame": { "pt_from_parent": 70, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "comp01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "comp01@staff" }
},
"status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": false }
```

| field | ความหมาย |
|---|---|
| `pt_from_parent` | ค่าที่ได้รับจากผู้สร้าง |
| `force` · `remain_quota` | บังคับถือสู้ · เอาส่วนที่เหลือ (ไม่เกินค่าที่ให้) |
| `commission_percent` | Commission 0–1 |
| `status` | รับ PT ในระบบนี้ไหม · `false` = เกมยังเปิด แต่บัญชีนี้ไม่รับ PT |
| `status_game` | เปิด / ปิดทีละเกมที่ตั้งกับบัญชีเอง (อยู่นอก `pt`) |
| `created_at` · `created_by` | เวลา / username ของคนกดสร้าง (sub = `comp01@staff`) |
| `updated_at` · `updated_by` | เวลา / username ของคนที่แก้ล่าสุด |

แถว Member ในรายชื่อ: `pt.minigame` = `pt` (ผู้สร้างถือสู้กับ Member คนนี้) · `remain_quota` · `commission_percent` + `created_*` / `updated_*`

### 1.5 ข้อมูลตัวอย่าง

`comp01` (id 10, Company Transfer) → `share01` (id 12, Share B2C · THB) → `agent01` (id 15) → `mem01` (Member id 501) · sub ของ comp01 = `comp01@staff` (id 30)

## 2. ใครสร้างอะไรได้

| ผู้สร้าง | สร้างได้ (`user_type`) | `currencies` ที่ต้องส่ง |
|---|---|---|
| Superadmin | `COMPANY_TRANSFER` · `COMPANY_SEAMLESS_RESELLER` · `COMPANY_SEAMLESS_MASTER` | ไม่ต้องส่ง (ครบทุกสกุล) |
| Superadmin | `COMPANY_SEAMLESS_1TO1` | 1 สกุล |
| Company Transfer | `SHARE_B2B` | หลายสกุล อย่างน้อย 1 |
| Company Transfer | `SHARE_B2C` | 1 สกุล |
| Company Seamless Reseller / Master | `SHARE_B2C` (เก็บเป็น `SHARE_RESELLER` / `SHARE_MASTER`) | 1 สกุล |
| Company Seamless 1 to 1 | Member เท่านั้น | — |
| Share B2B | `AGENT` เท่านั้น | 1 สกุล (ในสกุลของผู้สร้าง) |
| Share B2C | `AGENT` หรือ Member | ไม่ต้องส่ง |
| **Share Reseller / Share Master** | **Member เท่านั้น** (แก้ 9 ต.ค.) | — |
| Agent | `AGENT` (ซ้อนได้ไม่จำกัดชั้น) หรือ Member | ไม่ต้องส่ง |

Member สร้างที่ `/manage/members/create` ([member_management.md](member_management.md)) · บัญชีฝั่ง Seamless ไม่มียอดเงิน — ห้ามส่ง `balance`

## 3. Endpoints

| # | Path | ใช้เมื่อ | สิทธิ์ของ sub |
|---|---|---|---|
| 3.1 | `/api/v1/bo/pr/manage/agents/create` | สร้าง Company / Share / Agent | `member` edit (+ `payment` edit ถ้าส่ง `balance`) |
| 3.2 | `/api/v1/bo/pr/manage/downlines/list` | รายชื่อลูกตรง · ค้นทั้งสายด้วย `keyword` | `member` view |
| 3.3 | `/api/v1/bo/pr/manage/agents/detail/get` | รายละเอียดบัญชีฝั่ง agent | `member` view |
| 3.4 | `/api/v1/bo/pr/manage/agents/detail/update` | แก้ชื่อ / เบอร์ · ค่าหุ้นส่วน · เปิดปิดเกม (เส้นเดียว) | `member` edit |
| 3.5 | `/api/v1/bo/pr/manage/agents/status/update` | ระงับ / ล็อก / ปลด | `member` edit |
| 3.6 | `/api/v1/bo/pr/manage/agents/list` | dropdown ลูกตรงฝั่ง agent ("คัดลอกการตั้งค่าจาก") | `member` view |
| 3.7 | `/api/v1/bo/pr/manage/subaccounts/list` | รายชื่อ sub | บัญชีหลักเท่านั้น |
| 3.8 | `/api/v1/bo/pr/manage/subaccounts/detail/get` | รายละเอียด sub | บัญชีหลักเท่านั้น |
| 3.9 | `/api/v1/bo/pr/manage/subaccounts/create` | สร้าง sub | บัญชีหลักเท่านั้น |
| 3.10 | `/api/v1/bo/pr/manage/subaccounts/detail/update` | แก้ชื่อเล่น · เบอร์ · สิทธิ์ของ sub | เจ้าของ sub |
| 3.11 | `/api/v1/bo/pr/manage/subaccounts/status/update` | `ACTIVE` / `INACTIVE` ของ sub | เจ้าของ sub |

**error ที่เกิดได้กับทุกเส้น:** `401202`, `401203`, `401301`, `401302` → หน้า login · `401304`, `401306`, `401307` → ด่านหลัง login · `401311` → บัญชีถูกระงับ

### ตั้งค่า Postman

`MG_URL` = `https://dev-mini-api.pirate168.com` หรือ `http://localhost:8282` · `TOKEN` ตั้งอัตโนมัติจากเส้น Login (script ใน `docs/frontend/auth.md`)

### 3.1 สร้าง — `agents/create`

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/agents/create" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" \
  -d '{"request_id":"{{$guid}}","user_type":"SHARE_B2C","username":"Share01","password":"aA4b4c4d4e4f","name":"share01","phone":"0812345678","currencies":["THB"],"balance":{"THB":10000},"pt":{"minigame":{"pt_from_parent":70,"force":0,"remain_quota":0,"commission_percent":0.5,"status":true}},"status_game":{"scratch_card":false}}'
```
Response: `{ "code": 200, "msg": "สำเร็จ", "data": { "id": 12, "username": "share01", "user_type": "SHARE_B2C" } }`

- `request_id` UUID ใหม่ทุกครั้งที่กดสร้าง (หน้าบ้านสร้างเอง เช่น `crypto.randomUUID()`) · ส่งซ้ำด้วยค่าเดิม = ได้บัญชีเดิม ไม่สร้าง / ไม่โอนซ้ำ · ค่าที่คนอื่นใช้แล้ว = `422`
- `currencies` ตามหัวข้อ 2 · `balance` ไม่บังคับ (โอนจากยอดของผู้สร้าง) · `phone` ไม่กรอกส่ง `""`
- `pt` ครบ 5 ค่า · `status_game` ไม่บังคับ (เกมที่ไม่ส่ง = เปิด)

Error: `422`, `402301`, `402303`, `402305`, `402307`, `402308`, `402309`, `402310`, `402312`, `402401`, `402403`

### 3.2 รายชื่อ / ค้นหา — `downlines/list`

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/downlines/list" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" \
  -d '{"parent_id":12,"keyword":"","page":1,"limit":20}'
```

| field | กติกา |
|---|---|
| `parent_id` | ไม่ส่ง / `0` = ตัวเอง · ต้องเป็นตัวเองหรือบัญชีฝั่ง agent ในสายล่าง (`402402`) |
| `keyword` | ไม่ส่ง / `""` = **ลูกตรง**ของ `parent_id` · ส่ง 4–32 ตัว = **ค้น username ทุกชั้น**ใต้ `parent_id` (บางส่วน ไม่สนตัวพิมพ์) · สั้น / ยาวเกิน = `422` |
| `page` · `limit` | ค่าเริ่มต้น 1 · 20 · `limit` สูงสุด 100 |

Response `data` (แบ่งหน้า):
```json
{
  "current_page": 1, "total_page": 1, "total_count": 2, "limit": 20, "has_next": false, "has_prev": false,
  "data": [
    {
      "id": 15, "role": "AGENT", "user_type": "AGENT", "username": "agent01", "name": "agent01", "phone": "0898765432",
      "status": "ACTIVE", "parent_username": "share01",
      "last_login_at": "2026-10-08T21:00:00+07:00", "last_login_ip": "203.0.113.10", "created_at": "2026-10-01T09:00:00+07:00",
      "pt": { "minigame": { "pt_from_parent": 60, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "share01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "share01" } },
      "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": true },
      "balances": [ { "currency": "THB", "amount": 5000 } ]
    },
    {
      "id": 501, "role": "MEMBER", "user_type": "MEMBER", "username": "mem01", "name": "ใจดี", "phone": "",
      "status": "ACTIVE", "parent_username": "share01",
      "last_login_at": "", "last_login_ip": "", "created_at": "2026-10-01T09:00:00+07:00",
      "pt": { "minigame": { "pt": 30, "remain_quota": 40, "commission_percent": 0.3, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "share01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "share01" } },
      "balances": [ { "currency": "THB", "amount": 100 } ]
    }
  ]
}
```
- ทุกแถวมี `parent_username` (ผู้สร้างตรง — ใช้ตอนค้นเพื่อรู้ว่าแถวอยู่ใต้ใคร) · แถว agent มี `status_game` · แถว Member ไม่มี
- เรียง username A→Z · ยังไม่เคย login = `""`

Error: `422`, `402303`, `402402`

### 3.3 รายละเอียด — `agents/detail/get`

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/agents/detail/get" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"id":12}'
```
Response `data`:
```json
{
  "id": 12, "role": "SHAREHOLDER", "user_type": "SHARE_B2C", "username": "share01", "name": "share01", "phone": "0812345678",
  "status": "ACTIVE", "parent_username": "comp01", "currencies": ["THB"], "balances": [ { "currency": "THB", "amount": 10000 } ],
  "pt": { "minigame": { "pt_from_parent": 70, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "comp01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "comp01" } },
  "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": true },
  "status_game_effective": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": false },
  "passcode_set": true, "last_login_at": "2026-10-05T10:00:00+07:00", "last_login_ip": "203.0.113.10", "created_at": "2026-10-01T09:00:00+07:00"
}
```
- **`status_game`** = ค่าที่ตั้งกับบัญชีเอง → ใช้เติมฟอร์มแก้ · **`status_game_effective`** = เปิดจริงหลังรวมหัวสาย → ใช้แสดงผล (ตัวอย่าง: หัวสายปิด `scratch_card`)
- `id` ต้องเป็นบัญชีฝั่ง agent ในสายล่าง (ตัวเอง / สายบน / นอกสาย = `402402` · ของตัวเองดูที่ account Profile)

Error: `422`, `402303`, `402402`

### 3.4 แก้บัญชี — `agents/detail/update`

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/agents/detail/update" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" \
  -d '{"id":12,"info":{"name":"share01","phone":"0812345678"},"pt":{"minigame":{"pt_from_parent":60,"force":0,"remain_quota":0,"commission_percent":0.5,"status":true}},"status_game":{"scratch_card":false}}'
```
Response: `{ "code": 200, "msg": "สำเร็จ" }`

| section | กติกา |
|---|---|
| `info` | `name` · `phone` — ส่งบาง field ได้ (ไม่ส่ง = คงเดิม) · ล้างเบอร์ส่ง `"phone": ""` |
| `pt` | ส่งเฉพาะกลุ่มที่แก้ · กลุ่มละครบ 5 ค่า · ลดได้ไม่ต่ำกว่าที่ลูกใช้อยู่ (`402306` บอกค่าต่ำสุด) |
| `status_game` | เกมที่จะเปลี่ยน → `true` / `false` · อย่างน้อย 1 เกม |

- **ส่งเฉพาะ section ที่แก้** · ไม่ส่งเลยสัก section = `422`
- แก้ได้เฉพาะ**ลูกตรง** (ตัวเอง / หลาน = `402304` · นอกสาย = `402402`) · ไม่ต้อง passcode
- **ทั้งคำขอสำเร็จหรือไม่บันทึกเลย** — section ไหนผิด section อื่นก็ไม่ถูกบันทึก
- ค่าเหมือนเดิมทั้งหมด = 200 (ไม่มีประวัติเพิ่ม) · แก้ PT แล้วระบบคำนวณ `remain_quota` ของ Member ใต้ลูกให้เอง

Error: `422`, `402303`, `402304`, `402305`, `402306`, `402307`, `402308`, `402309`, `402402`, `402403`

### 3.5 สถานะ — `agents/status/update`

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/agents/status/update" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"id":12,"status":"SUSPENDED"}'
```
Response: `{ "code": 200, "msg": "สำเร็จ" }`
- `ACTIVE` / `SUSPENDED` / `LOCKED` · ลูกตรงเท่านั้น (ตัวเอง / หลาน = `402304`) · ไม่ต้อง passcode
- `LOCKED` มีผลตั้งแต่ request ถัดไป · ปลดลูกแล้วถ้าหัวสายที่สูงกว่ายังถูกระงับ ลูกยังแสดง `SUSPENDED`

Error: `422`, `402303`, `402304`, `402402`

### 3.6 dropdown ลูกตรง — `agents/list`

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/agents/list" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"keyword":""}'
```
Response `data`:
```json
[
  { "id": 12, "username": "share01", "name": "share01", "user_type": "SHARE_B2C", "status": "ACTIVE" },
  { "id": 13, "username": "share02", "name": "share02", "user_type": "SHARE_B2B", "status": "ACTIVE" }
]
```
- ลูกตรงฝั่ง agent ของตัวเอง (ไม่มี Member) · `keyword` ไม่บังคับ 4–32 ตัว · A→Z · ไม่แบ่งหน้า สูงสุด 500 · ไม่มีลูก = `[]`
- "คัดลอกการตั้งค่าจาก": เลือกแล้วเรียก `agents/detail/get` เอา `pt` / `status_game` มาเติมฟอร์มสร้าง

Error: `422`, `402303`

### 3.7 รายชื่อ sub — `subaccounts/list`

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/subaccounts/list" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"owner_id":10,"q":"","page":1,"limit":20}'
```
Response `data` (แบ่งหน้า) แต่ละแถว:
```json
{
  "id": 30, "username": "comp01@staff", "name": "staff01", "phone": "", "status": "ACTIVE",
  "permissions": { "dashboard": "view", "member": "edit", "report": "view", "bet_cancel": "off", "payment": "off", "asset": "off", "announcement": "off", "api_credential": "view" },
  "created_at": "2026-10-06T12:26:43+07:00", "last_login_at": "", "last_login_ip": ""
}
```
- `owner_id` ไม่ส่ง = sub ของตัวเอง · ชั้นบนดู sub ของสายล่างได้ · `q` ค้น username บางส่วน
- `permissions` ครบทุกเมนูของประเภทเจ้าของ (Superadmin: `dashboard` `member` `report` `bet_cancel` `payment` `asset` `rate` · Company / Share / Agent: `dashboard` `member` `report` `bet_cancel` `payment` `asset` `announcement` `api_credential`)

Error: `402311`, `402402`

### 3.8 รายละเอียด sub — `subaccounts/detail/get`

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/subaccounts/detail/get" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"id":30}'
```
Response `data` = แถวเดียวกับ 3.7 · ดูได้: เจ้าของ และชั้นบนในสาย · อื่น = `402404`

Error: `422`, `402311`, `402404`

### 3.9 สร้าง sub — `subaccounts/create`

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/subaccounts/create" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" \
  -d '{"name_suffix":"Staff","password":"aA4b4c4d4e4f","name":"staff01","phone":"","permissions":{"dashboard":"view","member":"edit","report":"view"}}'
```
Response: `{ "code": 200, "msg": "สำเร็จ", "data": { "id": 30, "username": "comp01@staff" } }`
- เมนูที่ไม่ส่ง = `off` · `dashboard` / `report` สูงสุด `view` · เมนูที่ประเภทเจ้าของไม่มี (รวม `pt`) = `422` · เบอร์ของ sub ซ้ำได้

Error: `422`, `402301`, `402311`, `402401`

### 3.10 แก้ sub — `subaccounts/detail/update`

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/subaccounts/detail/update" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" \
  -d '{"id":30,"name":"staff01","phone":"0811111111","permissions":{"member":"view","report":"view"}}'
```
Response: `{ "code": 200, "msg": "สำเร็จ" }` · แทนทั้งชุด (ต้องส่งครบ) · แก้ได้เฉพาะเจ้าของ (ชั้นบน = `402404`)

Error: `422`, `402311`, `402404`

### 3.11 สถานะ sub — `subaccounts/status/update`

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/subaccounts/status/update" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" -d '{"id":30,"status":"INACTIVE"}'
```
Response: `{ "code": 200, "msg": "สำเร็จ" }` · `ACTIVE` / `INACTIVE` · `INACTIVE` = login ได้แต่ทำรายการไม่ได้ · ไม่มีการลบ sub

Error: `422`, `402311`, `402404`

## 4. Error codes ของ module นี้

ตอบ HTTP 200 ทั้งหมด · ดูที่ `code` · ได้ error ของ field แรกที่ผิด (ไล่บนลงล่างตามลำดับ field ใน body) · `msg` บอกชื่อ field และค่าที่ตั้งได้ — แสดงข้างช่องนั้นได้เลย

| Code | ความหมาย | หน้าบ้านควรทำ |
|---|---|---|
| `422` | ข้อมูลไม่ผ่าน (`msg` บอก field) | แสดง `msg` ที่ช่องนั้น |
| `402301` | สร้างบัญชีประเภทนี้ไม่ได้ (`msg` บอกเหตุผลและประเภทที่สร้างได้) | ซ่อนประเภทที่สร้างไม่ได้ (หัวข้อ 2) |
| `402303` | ไม่มีสิทธิ์ใช้งานเมนูนี้ | ซ่อนเมนูตาม `permissions` ใน Profile |
| `402304` | แก้ไขได้เฉพาะผู้สร้างของบัญชีนี้ | ซ่อนปุ่มแก้ของบัญชีที่ไม่ใช่ลูกตรง |
| `402305` | ค่าที่ให้ลูกเกินกว่าที่ได้รับ | แสดงค่าสูงสุดจาก `pt_from_parent` ของตัวเอง |
| `402306` | ค่าที่ให้ลูกต่ำกว่าที่ลูกใช้อยู่ | แสดง `msg` (บอกค่าต่ำสุด) |
| `402307` | Share Master ใช้ค่า PT ตาม Company Seamless Master | ล็อกช่องในฟอร์ม |
| `402308` | Force / Remain เกินที่กำหนด | — |
| `402309` | Commission เกินที่กำหนด | — |
| `402310` | สกุลเงินไม่อยู่ในสกุลของผู้สร้าง | ให้เลือกจาก `currencies` ใน Profile |
| `402311` | บัญชีย่อยทำรายการนี้ไม่ได้ | ซ่อนเมนู sub เมื่อ `is_subaccount = true` |
| `402312` | ยอดเงินไม่พอสำหรับยอดเงินตั้งต้น | — |
| `402401` | username นี้ถูกใช้แล้ว | — |
| `402402` | ไม่พบบัญชีในสายของคุณ | — |
| `402403` | เบอร์โทรนี้ถูกใช้แล้ว | — |
| `402404` | ไม่พบบัญชีย่อยของคุณ | — |

## 5. เส้นเดิมที่ลบ (9 ต.ค.) → ใช้แทนด้วย

| เส้นเดิม | ใช้แทน |
|---|---|
| `downlines/search` | `downlines/list` + `keyword` |
| `agents/detail` | `agents/detail/get` |
| `agents/update-info` · `agents/update-pt` · `agents/update-games` | `agents/detail/update` (section `info` / `pt` / `status_game`) |
| `agents/update-status` | `agents/status/update` |
| `agents/copy-sources` (GET) | `agents/list` (POST) + `agents/detail/get` |
| `agents/update-hold` | ถือสู้กับ Member ตั้งต่อ Member (`members/update-pt`) |
| `subaccounts/detail` · `update-info` · `update-status` | `subaccounts/detail/get` · `detail/update` · `status/update` |
