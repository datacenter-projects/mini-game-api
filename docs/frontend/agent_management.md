# API หลังบ้าน — การจัดการสมาชิก (สำหรับหน้าบ้าน) — 9 ต.ค. 2026

เอกสารนี้สรุปจากโค้ดใน branch `dev` (merge แล้ว 8 ต.ค. 2026 — มีบน server dev หลัง deploy รอบถัดไป) ·
spec เต็ม: `docs/modules/agent_management.md` · พื้นฐานเรื่อง login / session / ด่านหลัง login ดู `docs/frontend/auth.md`

เส้นของ Member (สร้าง · รายละเอียด · แก้ข้อมูล · แก้สถานะ · แก้ Commission) ย้ายไป [member_management.md](member_management.md) (แยกเอกสาร 2026-10-09 · path ไม่เปลี่ยน)
เส้นค้นหาบัญชีของ ADMIN (`/admin/accounts/search`) ย้ายไป [admin_management.md](admin_management.md) (2026-10-09 · path ไม่เปลี่ยน)

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

- `code = 200` = สำเร็จ · อย่างอื่น = error (หัวข้อ 4)
- เส้นที่ไม่มีข้อมูลส่งกลับ **ไม่มี key `data`**
- error ของ module นี้ตอบ HTTP 200 และดูที่ `code` · `422` = ข้อมูลไม่ผ่าน `msg` บอกว่าผิดที่ field ไหน

### 1.3 กติกาของ module นี้

| เรื่อง | กติกา |
|---|---|
| ใครเรียก | มาจาก token เสมอ — ไม่ต้องส่งข้อมูลของตัวเอง |
| บัญชีที่จะดู / แก้ | ส่ง `id` ใน body (ไม่อยู่ใน URL) · หลังบ้านเช็คว่าอยู่ในสายของผู้เรียก |
| method | รายชื่อ / อ่านที่ต้องระบุบัญชี / สร้าง / แก้ = `POST` · `GET` เฉพาะเส้นที่ไม่ต้องส่งอะไร |
| `null` | ห้ามส่ง (`422`) · ไม่มีค่า: ข้อความ `""` · ตัวเลข `0` · รายการ `[]` · object `{}` |
| % | JSON number ทศนิยมไม่เกิน 4 ตำแหน่ง เช่น `70`, `0.5` · PT / Force / Remain ทีละ `0.5` · Commission ทีละ `0.1` ไม่เกิน `1` |
| เงิน | JSON number ทศนิยมไม่เกิน 4 ตำแหน่ง เช่น `10000.5` · response ส่งตามที่เก็บ (ตัด 0 ท้าย) **หน้าบ้านปัด 3 ตำแหน่งตอนแสดงเอง** (แก้ 9 ต.ค.) |
| `status` ใน response | สถานะที่ใช้งานจริง — หัวสายถูกระงับ ลูกทุกชั้นแสดง `SUSPENDED` |
| sub | ทำงานแทนเจ้าของตามสิทธิ์ที่ได้รับ · ไม่มีสิทธิ์ = `402303` |

### 1.4 ค่าหุ้นส่วน `pt`

key ของระบบคือ `minigame` (ทุกเกม minigame · ระบบอื่น เช่น `askmelotto` จะเพิ่มข้างๆ ภายหลัง)

**ตอนส่ง (สร้าง / `update-pt`)** — ชุดเดียวต่อระบบ หลังบ้านกระจายลงทุกเกมเอง:
```json
"pt": { "minigame": { "pt_from_parent": 70, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true } },
"status_game": { "scratch_card": false }
```

**ตอนแสดง (ทุกเส้น)** — ชุดเดียวต่อระบบ + วันที่ / คนสร้าง / คนแก้ล่าสุด · `status_game` แยกต่อเกม:
```json
"pt": {
  "minigame": { "pt_from_parent": 70, "pt": 30, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true,  "created_at": "2026-10-01T09:00:00+07:00", "created_by": "comp01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "comp01" }
},
"status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": false }
```

| field | ความหมาย | ใครตั้ง |
|---|---|---|
| `pt_from_parent` | ค่าที่ได้รับจากผู้สร้าง | ผู้สร้าง (สร้าง / `update-pt`) |
| `pt` | ค่าถือจาก Member ใต้ตัวเอง (ไม่เกิน `pt_from_parent`) | บัญชีนั้นเอง (`update-hold`) |
| `force` · `remain_quota` | บังคับถือสู้ · เอาส่วนที่เหลือ (ไม่เกินค่าที่ให้) | ผู้สร้าง |
| `commission_percent` | Commission 0–1 | ผู้สร้าง |
| `status` | รับ PT ในระบบนี้ไหม · `false` = เกมยังเปิด แต่บัญชีนี้ไม่รับ PT (ไม่ใช่สถานะเกม) | ผู้สร้าง |
| `status_game` | เปิด / ปิดทีละเกม (อยู่นอก `pt`) | ผู้สร้าง (ตอนสร้าง) |
| `created_at` · `created_by` | เวลา / username ของผู้สร้างค่า PT (ไม่เปลี่ยนหลังสร้าง) | ระบบตั้งให้ |
| `updated_at` · `updated_by` | เวลา / username ของคนที่แก้ค่า PT ล่าสุด (รวม sub) | ระบบตั้งให้ |

Member ตอนส่ง `{ "minigame": { "commission_percent": 0.3 } }` · ตอนแสดงมีแค่ `commission_percent` · `created_at` · `created_by` · `updated_at` · `updated_by` ·
ผู้เรียกไม่มีสิทธิ์ `pt` อย่างน้อย `view` = response ไม่มี field `pt`

### 1.5 ข้อมูลตัวอย่างในเอกสารนี้

`comp01` (id 10, Company Transfer) → `share01` (id 12, Share B2C · THB) → `agent01` (id 15) → `mem01` (Member id 501) ·
sub ของ comp01 = `comp01@staff` (id 30)

## 2. ใครสร้างอะไรได้

| ผู้สร้าง | สร้างได้ (`user_type`) | `currencies` ที่ต้องส่ง |
|---|---|---|
| Superadmin | `COMPANY_TRANSFER` · `COMPANY_SEAMLESS_RESELLER` · `COMPANY_SEAMLESS_MASTER` | ไม่ต้องส่ง (ครบทุกสกุล) |
| Superadmin | `COMPANY_SEAMLESS_1TO1` | 1 สกุล |
| Company Transfer | `SHARE_B2B` | หลายสกุล อย่างน้อย 1 |
| Company Transfer | `SHARE_B2C` | 1 สกุล |
| Company Seamless Reseller / Master | `SHARE_B2C` (เก็บเป็น `SHARE_RESELLER` / `SHARE_MASTER`) | 1 สกุล |
| Company Seamless 1 to 1 | Member เท่านั้น (`/members/create` — ดู [member_management.md](member_management.md)) | — |
| Share B2B | `AGENT` เท่านั้น | 1 สกุล (ในสกุลของผู้สร้าง) |
| Share B2C | `AGENT` หรือ Member | ไม่ต้องส่ง (ใช้สกุลของผู้สร้าง) |
| Share Reseller / Share Master | `AGENT` หรือ Member | ไม่ต้องส่ง (ใช้สกุลของผู้สร้าง) |
| Agent | `AGENT` (ซ้อนได้ไม่จำกัดชั้น) หรือ Member | ไม่ต้องส่ง (ใช้สกุลของผู้สร้าง) |
| Member | สร้างบัญชีไม่ได้ (ชั้นล่างสุด) | — |
| ADMIN | สร้างบัญชีไม่ได้ | — |

บัญชีฝั่ง Seamless (Company Seamless ทุกประเภทและสายล่าง) ไม่มียอดเงิน — ห้ามส่ง `balance`

## 3. Endpoints

| # | Method | Path | ใช้เมื่อ | สิทธิ์ของ sub |
|---|---|---|---|---|
| 3.1 | POST | `/api/v1/bo/pr/manage/agents/create` | สร้าง Company / Share / Agent | `member` edit · `pt` edit (+ `payment` edit ถ้าส่ง `balance`) |
| 3.3 | POST | `/api/v1/bo/pr/manage/downlines/list` | รายชื่อลูกตรง (กรองด้วย `q` ได้) | `member` view |
| 3.3A | POST | `/api/v1/bo/pr/manage/downlines/search` | ค้นหาทั้งสายใต้ตัวเอง (username) | `member` view |
| 3.4 | POST | `/api/v1/bo/pr/manage/agents/detail` | รายละเอียดบัญชีฝั่ง agent | `member` view |
| 3.6 | GET | `/api/v1/bo/pr/manage/agents/copy-sources` | ลูกตรง + ค่า PT สำหรับ "คัดลอกการตั้งค่าจาก" | `pt` view |
| 3.7 | POST | `/api/v1/bo/pr/manage/agents/update-info` | แก้ชื่อ · เบอร์ (ฝั่ง agent) | `member` edit |
| 3.9 | POST | `/api/v1/bo/pr/manage/agents/update-status` | แก้สถานะ (ฝั่ง agent) | `member` edit |
| 3.11 | POST | `/api/v1/bo/pr/manage/agents/update-pt` | ผู้สร้างแก้ค่าที่ให้ลูก | `pt` edit |
| 3.13 | POST | `/api/v1/bo/pr/manage/agents/update-hold` | ตั้งค่าถือของตัวเอง | `pt` edit |
| 3.13A | POST | `/api/v1/bo/pr/manage/agents/update-games` | เปิด / ปิดเกมให้ลูกตรง | `pt` edit · ผู้สร้างโดยตรง |
| 3.14 | POST | `/api/v1/bo/pr/manage/subaccounts/list` | รายชื่อ sub (กรองด้วย `q` ได้) | บัญชีหลักเท่านั้น |
| 3.15 | POST | `/api/v1/bo/pr/manage/subaccounts/detail` | รายละเอียด sub | บัญชีหลักเท่านั้น |
| 3.16 | POST | `/api/v1/bo/pr/manage/subaccounts/create` | สร้าง sub | บัญชีหลักเท่านั้น |
| 3.17 | POST | `/api/v1/bo/pr/manage/subaccounts/update-info` | แก้ชื่อเล่น · เบอร์ · สิทธิ์ของ sub | เจ้าของ sub |
| 3.18 | POST | `/api/v1/bo/pr/manage/subaccounts/update-status` | `ACTIVE` / `INACTIVE` ของ sub | เจ้าของ sub |

**error ที่เกิดได้กับทุกเส้น** (ไม่เขียนซ้ำในแต่ละเส้น): `401202`, `401203`, `401301`, `401302` → หน้า login ·
`401304`, `401306`, `401307` → ด่านหลัง login (`docs/frontend/auth.md` หัวข้อ 2)

### ตั้งค่า Postman (ตัวแปร `{{MG_URL}}` และ `{{TOKEN}}`)

คัดลอก curl ไป **Import** ใน Postman ได้เลย · `MG_URL` = `https://dev-mini-api.pirate168.com` (server dev หลัง deploy) หรือ `http://localhost:8282` (test env) · `TOKEN` ตั้งอัตโนมัติจากเส้น Login
(script ใน `docs/frontend/auth.md`) · ขอข้อความ error เป็นภาษาอังกฤษ: เพิ่ม `-H "X-Lang: en"`

### 3.1 POST /api/v1/bo/pr/manage/agents/create

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/agents/create" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"request_id":"6f1c2a1e-3b7d-4c55-9a40-0f7f3c2d8e11","user_type":"SHARE_B2C","username":"Share01","password":"aA4b4c4d4e4f","name":"share01","phone":"0812345678","currencies":["THB"],"balance":{"THB":10000.00},"pt":{"minigame":{"pt_from_parent":70,"force":0,"remain_quota":0,"commission_percent":0.5,"status":true}},"status_game":{"scratch_card":false}}'
```

Request:
```json
{
  "request_id": "6f1c2a1e-3b7d-4c55-9a40-0f7f3c2d8e11",
  "user_type": "SHARE_B2C",
  "username": "Share01",
  "password": "aA4b4c4d4e4f",
  "name": "share01",
  "phone": "0812345678",
  "currencies": ["THB"],
  "balance": { "THB": 10000.00 },
  "pt": {
    "minigame": { "pt_from_parent": 70, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true }
  },
  "status_game": { "scratch_card": false }
}
```

Response `data`:
```json
{ "id": 12, "username": "share01", "user_type": "SHARE_B2C" }
```

| field | กติกา |
|---|---|
| `request_id` | บังคับ · UUID ใหม่ทุกครั้งที่กดสร้าง · กดซ้ำด้วยค่าเดิม = ได้ response เดิม ไม่สร้าง / ไม่โอนซ้ำ |
| `user_type` | ตามหัวข้อ 2 · response ตอบประเภทที่เก็บจริง (เช่น `SHARE_RESELLER`) |
| `username` | 3–32 ตัว `a-z` `0-9` · ส่งตัวพิมพ์ใหญ่ได้ เก็บเป็นตัวเล็ก · ห้ามซ้ำทั้งระบบ |
| `password` | 8–64 ตัว มีตัวอักษรและตัวเลข · ห้ามตัวเดียวกันซ้ำติดกัน 4 ตัวขึ้นไป |
| `name` | 3–32 ตัวอักษร ไทย / อังกฤษ / ตัวเลข ไม่มีช่องว่าง |
| `phone` | `""` (ไม่ตั้ง) หรือตัวเลข 8–15 ตัว · ห้ามซ้ำ |
| `currencies` | ตามหัวข้อ 2 (ประเภทที่ไม่ต้องส่ง ห้ามส่ง) |
| `balance` | ไม่บังคับ · ยอดเงินตั้งต้นต่อสกุล > 0 · โอนจากยอดของผู้สร้าง (Superadmin ไม่จำกัด) · Seamless ห้ามส่ง |
| `pt` | ต้องส่งครบทุกกลุ่ม ครบ 5 ค่า |
| `status_game` | ไม่บังคับ · ส่งเฉพาะเกมที่จะปิด · เกมที่ไม่ส่ง = `true` (เปิด) · ไม่ส่งทั้ง field = เปิดทุกเกม |

Error: `422`, `402301`, `402303`, `402305`, `402307`, `402308`, `402309`, `402310`, `402312`, `402401`, `402403`

### 3.3 POST /api/v1/bo/pr/manage/downlines/list

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/downlines/list" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"parent_id":12,"q":"","page":1,"limit":20}'
```

Request (ทุกค่าไม่บังคับ · ไม่กรองให้ส่ง `{}`):
```json
{ "parent_id": 12, "q": "", "page": 1, "limit": 20 }
```

Response `data` (ลูกตรงของ share01 · หัวสาย comp01 ถูกระงับ):
```json
{
  "current_page": 1,
  "total_page": 1,
  "total_count": 2,
  "limit": 20,
  "has_next": false,
  "has_prev": false,
  "data": [
    {
      "id": 15,
      "role": "AGENT",
      "user_type": "AGENT",
      "username": "agent01",
      "name": "agent01",
      "phone": "0898765432",
      "status": "SUSPENDED",
      "pt": {
        "minigame": { "pt_from_parent": 60, "pt": 40, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "share01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "share01" }
      },
      "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": true },
      "balances": [ { "currency": "THB", "amount": 5000.00 } ]
    },
    {
      "id": 501,
      "role": "MEMBER",
      "user_type": "MEMBER",
      "username": "mem01",
      "name": "ใจดี",
      "phone": "",
      "status": "SUSPENDED",
      "pt": {
        "minigame": { "commission_percent": 0.3, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "agent01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "agent01" }
      },
      "balances": [ { "currency": "THB", "amount": 100.00 } ]
    }
  ]
}
```

| field | ความหมาย |
|---|---|
| `parent_id` | ไม่ส่ง = ลูกตรงของตัวเอง · ต้องเป็นบัญชีฝั่ง agent ในสายล่าง |
| `q` | ค้น username บางส่วน ไม่สนตัวพิมพ์ |
| `page` / `limit` | ค่าเริ่มต้น 1 / 20 · `limit` สูงสุด 100 |
| `role` | `MEMBER` = แถว Member (ไม่มีลูก กดลงต่อไม่ได้) · อื่น = ฝั่ง agent |
| `balances` | 1 รายการต่อสกุลของบัญชี · ยังไม่มียอด / บัญชี Seamless = `0.00` |

เรียง username A→Z · Error: `402303`, `402402`

### 3.3A POST /api/v1/bo/pr/manage/downlines/search

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/downlines/search" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"q":"sh","page":1,"limit":20}'
```

Request:
```json
{ "q": "sh", "page": 1, "limit": 20 }
```

Response `data` (comp01 ค้น `sh`):
```json
{
  "current_page": 1,
  "total_page": 1,
  "total_count": 3,
  "limit": 20,
  "has_next": false,
  "has_prev": false,
  "data": [
    {
      "id": 12,
      "role": "SHAREHOLDER",
      "user_type": "SHARE_B2C",
      "username": "sh1",
      "name": "sh1",
      "phone": "",
      "status": "ACTIVE",
      "parent_username": "comp01",
      "pt": {
        "minigame": { "pt_from_parent": 70, "pt": 30, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "comp01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "comp01" }
      },
      "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": true },
      "balances": [ { "currency": "THB", "amount": 10000.00 } ]
    },
    {
      "id": 13,
      "role": "SHAREHOLDER",
      "user_type": "SHARE_B2B",
      "username": "sh2",
      "name": "sh2",
      "phone": "",
      "status": "ACTIVE",
      "parent_username": "comp01",
      "pt": {
        "minigame": { "pt_from_parent": 50, "pt": 50, "force": 0, "remain_quota": 0, "commission_percent": 0.1, "status": true, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "comp01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "comp01" }
      },
      "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": true },
      "balances": [ { "currency": "THB", "amount": 0.00 }, { "currency": "USD", "amount": 0.00 } ]
    },
    {
      "id": 21,
      "role": "AGENT",
      "user_type": "AGENT",
      "username": "sh3agent",
      "name": "sh3agent",
      "phone": "",
      "status": "SUSPENDED",
      "parent_username": "sh1",
      "pt": {
        "minigame": { "pt_from_parent": 40, "pt": 40, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "sh1", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "sh1" }
      },
      "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": true },
      "balances": [ { "currency": "THB", "amount": 500.00 } ]
    }
  ]
}
```

| field | ความหมาย |
|---|---|
| `q` | บังคับ · username บางส่วน 2–32 ตัว ไม่สนตัวพิมพ์ |
| ขอบเขต | **ทุกชั้นใต้ตัวเอง** (ทั้งฝั่ง agent และ Member) · ไม่เห็นตัวเอง ชั้นบน และสายข้างเคียง |
| `parent_username` | ผู้สร้างตรงของแถวนั้น (บอกว่าอยู่ใต้ใคร) |
| อื่นๆ | เหมือนแถวของ 3.3 |

ต่างจาก 3.3: 3.3 ดูทีละชั้น (ลูกตรงของ `parent_id`) · 3.3A ค้นรวดเดียวทุกชั้น · Error: `422`, `402303`

### 3.4 POST /api/v1/bo/pr/manage/agents/detail

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/agents/detail" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"id":12}'
```

Request:
```json
{ "id": 12 }
```

Response `data`:
```json
{
  "id": 12,
  "role": "SHAREHOLDER",
  "user_type": "SHARE_B2C",
  "username": "share01",
  "name": "share01",
  "phone": "0812345678",
  "status": "ACTIVE",
  "parent_username": "comp01",
  "currencies": ["THB"],
  "balances": [ { "currency": "THB", "amount": 10000.00 } ],
  "pt": {
    "minigame": { "pt_from_parent": 70, "pt": 30, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "comp01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "comp01" }
  },
  "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": false },
  "passcode_set": true,
  "last_login_at": "2026-10-05T10:00:00+07:00",
  "last_login_ip": "203.0.113.10",
  "created_at": "2026-10-01T09:00:00+07:00"
}
```

- `id` = บัญชีในสายล่าง (ตัวเอง / สายบน / นอกสาย = `402402`) · ข้อมูลของตัวเองดูที่ `GET /account/profile`
- ยังไม่เคย login = `last_login_at` / `last_login_ip` เป็น `""`

Error: `422`, `402303`, `402402`

### 3.6 GET /api/v1/bo/pr/manage/agents/copy-sources

```
curl "{{MG_URL}}/api/v1/bo/pr/manage/agents/copy-sources" \
  -H "Authorization: Bearer {{TOKEN}}"
```

ไม่มี body (ลูกตรงฝั่ง agent ของบัญชีใน token)

Response `data` (ของ comp01):
```json
[
  {
    "id": 12,
    "username": "share01",
    "user_type": "SHARE_B2C",
    "pt": {
      "minigame": { "pt_from_parent": 70, "pt": 30, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "comp01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "comp01" }
    },
    "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": false }
  },
  {
    "id": 13,
    "username": "share02",
    "user_type": "SHARE_B2B",
    "pt": {
      "minigame": { "pt_from_parent": 50, "pt": 50, "force": 5, "remain_quota": 5, "commission_percent": 0.1, "status": true, "created_at": "2026-10-01T09:00:00+07:00", "created_by": "comp01", "updated_at": "2026-10-08T10:00:00+07:00", "updated_by": "comp01" }
    },
    "status_game": { "coin_toss": true, "rock_paper_scissors": true, "scratch_card": true }
  }
]
```

- ลูกตรงทั้งหมด เรียง A→Z (ไม่แบ่งหน้า) · ไม่มีลูก = `[]`
- หน้าบ้านเติม `pt_from_parent` / `force` / `remain_quota` / `commission_percent` / `status` / `status_game` ลงฟอร์มสร้าง แล้วกดสร้างตามปกติ

Error: `402303`

### 3.7 POST /api/v1/bo/pr/manage/agents/update-info

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/agents/update-info" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"id":12,"name":"share01","phone":""}'
```

Request (แทนทั้งชุด — ต้องส่งครบ):
```json
{ "id": 12, "name": "share01", "phone": "" }
```
Response: `{ "code": 200, "msg": "สำเร็จ" }`

- แก้ได้เฉพาะ**ผู้สร้างโดยตรง** · `phone` `""` = ลบเบอร์ · ไม่ส่ง `phone` = `422`

Error: `422`, `402303`, `402304` (ไม่ใช่ลูกตรง), `402402` (นอกสาย), `402403`

### 3.9 POST /api/v1/bo/pr/manage/agents/update-status

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/agents/update-status" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"id":12,"status":"SUSPENDED"}'
```

Request:
```json
{ "id": 12, "status": "SUSPENDED" }
```
Response: `{ "code": 200, "msg": "สำเร็จ" }`

| `status` | ผล |
|---|---|
| `ACTIVE` | ใช้งานปกติ (ปลดล็อก / ยกเลิกระงับได้) |
| `SUSPENDED` | บัญชีนี้และสายล่างเข้าได้แค่ Profile / รายงาน |
| `LOCKED` | บัญชีนี้และสายล่างเข้าไม่ได้ |

- แก้ได้เฉพาะผู้สร้างโดยตรง · ไม่ต้อง passcode · ถ้าหัวสายยังถูกระงับ ตั้ง `ACTIVE` แล้วบัญชียังแสดง `SUSPENDED`

Error: `422`, `402303`, `402304`, `402402`

### 3.11 POST /api/v1/bo/pr/manage/agents/update-pt

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/agents/update-pt" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"id":12,"pt":{"minigame":{"pt_from_parent":60,"force":0,"remain_quota":0,"commission_percent":0.5,"status":true}}}'
```

Request (ส่งเฉพาะกลุ่มที่จะแก้ · ในกลุ่มต้องครบ 5 ค่า):
```json
{
  "id": 12,
  "pt": {
    "minigame": { "pt_from_parent": 60, "force": 0, "remain_quota": 0, "commission_percent": 0.5, "status": true }
  }
}
```
Response: `{ "code": 200, "msg": "สำเร็จ" }`

- แก้ได้เฉพาะผู้สร้างโดยตรง · `pt_from_parent` ไม่เกินค่าที่ผู้สร้างได้รับ (`402305`)
- **ลดได้ไม่ต่ำกว่าที่ลูกใช้อยู่** = ค่าที่มากที่สุดระหว่าง ค่าถือของลูก และค่าที่ลูกให้ลูกของมันแต่ละคน
  — ตัวอย่าง: share01 ถือ 30 และให้ agent01 60 → ตั้งต่ำสุด 60 · ตั้ง 55 ได้ error:
  ```json
  { "code": 402306, "msg": "pt.minigame.pt_from_parent ต่ำกว่าที่ลูกใช้อยู่ ตั้งได้ต่ำสุด 60" }
  ```
- ห้ามส่ง `pt` (ค่าถือที่ลูกตั้งเอง) = `422`

Error: `422`, `402303`, `402304`, `402305`, `402306`, `402307`, `402308`, `402309`, `402402`

### 3.13 POST /api/v1/bo/pr/manage/agents/update-hold

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/agents/update-hold" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"pt":{"minigame":{"pt":40}}}'
```

Request (ไม่ส่ง `id` — แก้บัญชีใน token):
```json
{ "pt": { "minigame": { "pt": 40 } } }
```
Response: `{ "code": 200, "msg": "สำเร็จ" }`

- ค่าถือจาก Member ใต้ตัวเอง · ไม่เกินค่าที่ได้รับ (`402305`) · ส่วนที่เหลือผ่าน Force / Remain แล้วคืน Superadmin
- Company Seamless Master ตั้งได้แค่ `0` (`402307`) · ส่ง `pt_from_parent` / `force` ฯลฯ มา = `422`

Error: `422`, `402303`, `402305`, `402307`

### 3.13A POST /api/v1/bo/pr/manage/agents/update-games

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/agents/update-games" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"id":12,"status_game":{"scratch_card":false}}'
```

Request (ส่งเฉพาะเกมที่จะเปลี่ยน · อย่างน้อย 1 เกม):
```json
{ "id": 12, "status_game": { "scratch_card": false } }
```
Response: `{ "code": 200, "msg": "สำเร็จ" }`

- `false` = ปิดเกมนั้นให้บัญชีนี้ · Member ในสายเล่นเกมนั้นไม่ได้ · `true` = เปิดกลับ
- แก้ได้เฉพาะผู้สร้างโดยตรง (`402304`) · บัญชีนอกสาย `402402`
- ไม่เปลี่ยนค่าของสายล่าง — ชั้นบนปิดแล้วสายล่างเปิดเองไม่ได้ (ระบบเช็คทั้งสายตอนเล่น)
- ไม่เปลี่ยนค่า PT และ `updated_by` ของ `pt` · ดูค่าปัจจุบันที่ `status_game` ของ `agents/detail`

Error: `422`, `402303`, `402304`, `402402`

### 3.14 POST /api/v1/bo/pr/manage/subaccounts/list

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/subaccounts/list" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"owner_id":10,"q":"","page":1,"limit":20}'
```

Request (ทุกค่าไม่บังคับ · ไม่กรองให้ส่ง `{}`):
```json
{ "owner_id": 10, "q": "", "page": 1, "limit": 20 }
```

Response `data`:
```json
{
  "current_page": 1,
  "total_page": 1,
  "total_count": 1,
  "limit": 20,
  "has_next": false,
  "has_prev": false,
  "data": [
    {
      "id": 30,
      "username": "comp01@staff",
      "name": "staff01",
      "phone": "",
      "status": "ACTIVE",
      "permissions": {
        "dashboard": "view", "member": "edit", "pt": "view", "report": "view",
        "bet_cancel": "off", "payment": "off", "asset": "off", "announcement": "off"
      },
      "created_at": "2026-10-06T12:26:43+07:00",
      "last_login_at": "",
      "last_login_ip": ""
    }
  ]
}
```

| field | ความหมาย |
|---|---|
| `owner_id` | ไม่ส่ง = sub ของตัวเอง · ชั้นบนดู sub ของบัญชีในสายล่างได้ · นอกสาย = `402402` |
| `q` / `page` / `limit` | เหมือน 3.3 |
| `status` | `ACTIVE` / `INACTIVE` · เจ้าของหรือหัวสายถูกระงับ / ล็อก = แสดงสถานะนั้น |
| `permissions` | ครบทุกเมนูของประเภทเจ้าของ (Superadmin มี `rate` แทน `announcement`) |

Error: `402311` (sub เรียก), `402402`

### 3.15 POST /api/v1/bo/pr/manage/subaccounts/detail

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/subaccounts/detail" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"id":30}'
```

Request:
```json
{ "id": 30 }
```

Response `data`: รูปแบบเดียวกับ 1 แถวของ 3.14
```json
{
  "id": 30,
  "username": "comp01@staff",
  "name": "staff01",
  "phone": "",
  "status": "ACTIVE",
  "permissions": {
    "dashboard": "view", "member": "edit", "pt": "view", "report": "view",
    "bet_cancel": "off", "payment": "off", "asset": "off", "announcement": "off"
  },
  "created_at": "2026-10-06T12:26:43+07:00",
  "last_login_at": "2026-10-07T09:15:00+07:00",
  "last_login_ip": "203.0.113.20"
}
```

ดูได้: เจ้าของ และชั้นบนในสาย · Error: `422`, `402311`, `402404`

### 3.16 POST /api/v1/bo/pr/manage/subaccounts/create

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/subaccounts/create" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"name_suffix":"Staff","password":"aA4b4c4d4e4f","name":"staff01","phone":"","permissions":{"dashboard":"view","member":"edit","pt":"view","report":"view"}}'
```

Request:
```json
{
  "name_suffix": "Staff",
  "password": "aA4b4c4d4e4f",
  "name": "staff01",
  "phone": "",
  "permissions": { "dashboard": "view", "member": "edit", "pt": "view", "report": "view" }
}
```

Response `data`:
```json
{ "id": 30, "username": "comp01@staff" }
```

| field | กติกา |
|---|---|
| `name_suffix` | ส่วนหลัง `@` · 3–20 ตัว `a-z` `0-9` · ส่งตัวพิมพ์ใหญ่ได้ เก็บเป็นตัวเล็ก · username = `{เจ้าของ}@{name_suffix}` |
| `name` | ชื่อเล่น · กติกาเดียวกับ `name` ใน 3.1 |
| `phone` | `""` หรือตัวเลข 8–15 ตัว · **ซ้ำได้** |
| `permissions` | เมนูที่ไม่ส่ง = `off` · ระดับ `off` / `view` / `edit` · `dashboard` / `report` สูงสุด `view` · เมนูที่ประเภทเจ้าของไม่มี = `422` |

Error: `422`, `402301` (ADMIN สร้าง), `402311`, `402401` (ชื่อซ้ำ)

### 3.17 POST /api/v1/bo/pr/manage/subaccounts/update-info

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/subaccounts/update-info" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"id":30,"name":"staff01","phone":"0811111111","permissions":{"member":"view","report":"view"}}'
```

Request (แทนทั้งชุด — ต้องส่งครบ · ไม่ให้สิทธิ์ใดส่ง `"permissions": {}`):
```json
{
  "id": 30,
  "name": "staff01",
  "phone": "0811111111",
  "permissions": { "member": "view", "report": "view" }
}
```
Response: `{ "code": 200, "msg": "สำเร็จ" }`

แก้ได้เฉพาะเจ้าของ (ชั้นบน = `402404`) · Error: `422`, `402311`, `402404`

### 3.18 POST /api/v1/bo/pr/manage/subaccounts/update-status

```
curl -X POST "{{MG_URL}}/api/v1/bo/pr/manage/subaccounts/update-status" \
  -H "Authorization: Bearer {{TOKEN}}" \
  -H "Content-Type: application/json" \
  -d '{"id":30,"status":"INACTIVE"}'
```

Request:
```json
{ "id": 30, "status": "INACTIVE" }
```
Response: `{ "code": 200, "msg": "สำเร็จ" }`

- `ACTIVE` / `INACTIVE` · `INACTIVE` = login ได้แต่ทำรายการไม่ได้ (นอกจาก Profile / รายงาน) · ตั้งกลับ `ACTIVE` ได้
- ไม่มีการลบ sub — เลิกใช้ให้ตั้ง `INACTIVE` · แก้ได้เฉพาะเจ้าของ

Error: `422`, `402311`, `402404`

## 4. Error codes ของ module นี้

ตอบ HTTP 200 ทั้งหมด · ดูที่ `code`

**ลำดับ (แก้ 9 ต.ค.):** ได้ error ของ field แรกที่ผิด ไล่บนลงล่างตามลำดับ field ใน body — รูปแบบทุก field ก่อน (`422`) แล้วค่อยกฎที่ต้องดูข้อมูล
(ประเภท → username / เบอร์ซ้ำ → สกุล → ยอดเงิน → `pt` ทีละ field) · `msg` บอกชื่อ field และค่าที่ตั้งได้ เช่น
`pt.minigame.pt_from_parent ตั้งได้ไม่เกิน 90 (ค่าที่คุณได้รับ)` · `currencies: COMPANY_TRANSFER ได้ครบทุกสกุล ห้ามส่ง currencies` — แสดง `msg` ข้างช่องนั้นได้เลย

| Code | ความหมาย | หน้าบ้านควรทำ |
|---|---|---|
| `422` | ข้อมูลไม่ผ่าน (`msg` บอก field) | แสดง `msg` ที่ช่องนั้น |
| `402301` | สร้างบัญชีประเภทนี้ไม่ได้ | ซ่อนประเภทที่สร้างไม่ได้ (หัวข้อ 2) |
| `402303` | ไม่มีสิทธิ์ใช้งานเมนูนี้ | ซ่อนเมนูตาม `permissions` ใน Profile |
| `402304` | แก้ไขได้เฉพาะผู้สร้างของบัญชีนี้ | ซ่อนปุ่มแก้ของบัญชีที่ไม่ใช่ลูกตรง |
| `402305` | ค่าเกินกว่าที่ได้รับ | แสดงค่าสูงสุดจาก `pt_from_parent` ของตัวเอง |
| `402306` | ค่าที่ให้ลูกต่ำกว่าที่ลูกใช้อยู่ | แสดง `msg` (บอกค่าต่ำสุด) |
| `402307` | Company Seamless Master ต้องถือ 0 และให้ทั้งหมด | ล็อกช่องในฟอร์ม |
| `402308` | Force / Remain เกินที่กำหนด | — |
| `402309` | Commission เกินที่กำหนด | — |
| `402310` | สกุลเงินไม่อยู่ในสกุลของผู้สร้าง | ให้เลือกจาก `currencies` ใน Profile |
| `402311` | บัญชีย่อยทำรายการนี้ไม่ได้ | ซ่อนเมนู sub เมื่อ `is_subaccount = true` |
| `402312` | ยอดเงินไม่พอสำหรับยอดเงินตั้งต้น | — |
| `402401` | username นี้ถูกใช้แล้ว | — |
| `402402` | ไม่พบบัญชีในสายของคุณ | — |
| `402403` | เบอร์โทรนี้ถูกใช้แล้ว | — |
| `402404` | ไม่พบบัญชีย่อยของคุณ | — |
