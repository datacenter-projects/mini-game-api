# API หลังบ้าน — ฝาก / ถอนเงินให้ลูกตรง (สำหรับหน้าบ้าน) — 16 ต.ค. 2026

เอกสารนี้ตรงกับโค้ดบน `dev` (deploy แล้ว `dev-202610160811` — รวม 2.4 `history/list` · `status` / `ip` · Superadmin มียอด · สาย Seamless มียอด) · spec เต็ม: `docs/modules/deposit_withdraw.md` · ผู้ดูแล module: maofoy

> **เปลี่ยนจากเดิม (16 ต.ค. — deploy แล้ว `dev-202610160701`)** — 2.3 / 2.4 เพิ่ม **`status`** (`SUCCESS` / `FAILED`) · 2.3 เพิ่ม **`ip`** (ip ของคนกด) ·
> รายการที่กดแล้วไม่สำเร็จ (ยอดไม่พอ ฯลฯ) มีแถว **`FAILED`** + เหตุผลท้าย `note` / `remark` · `request_id` เดียวกันมีหลายแถวได้ — ใช้ `id` เป็น key
>
> **เส้นใหม่ (15 ต.ค. — deploy แล้ว `dev-202610150558`)** — **2.4 `history/list`** ค้นประวัติฝาก / ถอนของบัญชีในสายด้วย `username` (หน้าจอ "ประวัติฝาก / ถอน")
>
> **เปลี่ยนจากเดิม (15 ต.ค. — deploy แล้ว `dev-202610150510`)** — **Superadmin มียอดของตัวเองแล้ว** (สกุลละ 100,000,000): ฝาก = หักยอด Superadmin · ฝากเกินยอดที่มี = `404304` ·
> ถอน = ยอด Superadmin เพิ่ม · `credit_detail` (2.1) ของ Superadmin เป็นยอดจริง · `parent_unlimited` (2.2 / 2.3) = `false` เสมอสำหรับรายการใหม่
>
> **เปลี่ยนจากเดิม (14 ต.ค. — มีผลหลัง deploy รอบถัดจาก `dev-202610140819`)** — `downlines/list` (2.1): แถวเปลี่ยน `balances` (array) เป็น
> **`currency` (map ครบ 27 สกุล · key ตัวใหญ่)** · เพิ่ม `last_login_ip` / `last_login_at` · `data` เพิ่ม **`credit_detail`** (ยอดของคนที่ login) และ
> **`total_sum`** (ยอดรวมของลูกตรงทั้งหมดแยกตาม role · key ตัวเล็ก เช่น `agent` / `member`) · field แบ่งหน้าอยู่ที่เดิม · 2.2 / 2.3 ไม่เปลี่ยน

พื้นฐานใช้ร่วมกับเอกสารการจัดการสมาชิก: base URL / header / รูปแบบ response / ห้ามส่ง `null` / ตัวเลขเป็น JSON number / ตั้งค่า Postman ดู
[agent_management.md](agent_management.md) หัวข้อ 1 · login / passcode / ด่านหลัง login ดู [auth.md](auth.md)

## 1. ภาพรวม

| # | Path | ใช้เมื่อ | สิทธิ์ของ sub |
|---|---|---|---|
| 2.1 | `/api/v1/bo/pr/deposit-withdraw/downlines/list` | รายชื่อลูกตรง + ยอด (ตารางหน้าฝากถอน) | `payment` view |
| 2.2 | `/api/v1/bo/pr/deposit-withdraw/create` | ฝาก / ถอน (**ต้องใส่ passcode**) | `payment` edit |
| 2.3 | `/api/v1/bo/pr/deposit-withdraw/logs/list` | ประวัติรายการที่บัญชีนี้ทำ | `payment` view |
| 2.4 | `/api/v1/bo/pr/deposit-withdraw/history/list` | ประวัติฝาก / ถอนที่ทำให้บัญชีหนึ่งในสาย (ค้นด้วย username) | `payment` view |

ทุกเส้นเป็น `POST`

- **ทำกับใครได้:** ฝาก / ถอน (2.2) เฉพาะลูกตรงชั้นเดียว (agent หรือ Member) · หลานทำไม่ได้ · ดูประวัติ (2.4) ค้นได้ทุกชั้นใต้คุณ · **ADMIN ใช้ไม่ได้ทุกเส้น** (`404305`)
- **ฝาก** = ยอดของคุณลด · ลูกเพิ่ม · **ถอน** = ลูกลด · ยอดของคุณเพิ่ม · **Superadmin มียอดของตัวเองเหมือนบัญชีอื่น** (ยอดตั้งต้นสกุลละ 100,000,000 · ฝากเกินยอดที่มี = `404304`)
- **ปุ่มฝาก:** เปิดเมื่อ `has_balance = true` **และ** `status = ACTIVE`
- **ปุ่มถอน:** เปิดเมื่อ `has_balance = true` — **ลูกที่ถูกระงับ / ล็อกยังถอนได้** (ดึงเงินคืนจากบัญชีที่ถูกล็อก)
- **`has_balance`** = `true` ทุกแถวตั้งแต่ 15 ต.ค. (สาย Seamless ฝาก / ถอนได้แล้ว · คง field ไว้ไม่ให้โค้ดเดิมพัง) · ยอดของสาย Seamless **ใช้แทงไม่ได้** (Member สาย Seamless แทงจาก wallet ของ operator)
- **เงิน:** JSON number · ทศนิยมไม่เกิน 4 ตำแหน่ง · ไม่เกิน 1,000,000,000 ต่อรายการ · API ส่งยอดตามที่เก็บ — หน้าบ้านปัด 3 ตำแหน่งตอนแสดงเอง
- **ภาษาของ `msg`:** ไทยเป็นค่าเริ่มต้น · ส่ง header `X-Lang: en` (หรือ `Accept-Language: en`) ได้ภาษาอังกฤษ

## 2. Endpoints

### 2.1 รายชื่อลูกตรง — `deposit-withdraw/downlines/list`

```bash
curl -X POST "{{MG_URL}}/api/v1/bo/pr/deposit-withdraw/downlines/list" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" \
  -d '{"keyword":"","page":1,"limit":20}'
```

| Field | บังคับ | กติกา |
|---|---|---|
| `keyword` | | ไม่ส่ง / `""` = ทั้งหมด · ส่ง 4–32 ตัว = ค้น username บางส่วน ไม่สนตัวพิมพ์ **เฉพาะลูกตรง** (ไม่ลงหลาน) |
| `page` / `limit` | | ค่าเริ่มต้น 1 / 20 · สูงสุด 100 |

Response `data` (ตัวอย่างย่อ map สกุลเหลือ 3 ตัว — **ของจริงมีครบ 27 สกุล** ทุก map):
```json
{
  "credit_detail": { "currency": { "THB": 900, "USD": 0, "CNY": 0 } },
  "total_sum": {
    "agent":  { "THB": 5000, "USD": 0, "CNY": 0 },
    "member": { "THB": 100,  "USD": 0, "CNY": 0 }
  },
  "current_page": 1, "total_page": 1, "total_count": 2, "limit": 20, "has_next": false, "has_prev": false,
  "data": [
    { "id": 15,  "role": "AGENT",  "username": "agent01", "name": "agent01", "status": "ACTIVE", "has_balance": true,
      "currency": { "THB": 5000, "USD": 0, "CNY": 0 },
      "last_login_ip": "203.0.113.10", "last_login_at": "2026-10-08T21:00:00+07:00" },
    { "id": 501, "role": "MEMBER", "username": "mem01",   "name": "ใจดี",    "status": "LOCKED", "has_balance": true,
      "currency": { "THB": 100, "USD": 0, "CNY": 0 },
      "last_login_ip": "", "last_login_at": "" }
  ]
}
```
- **`credit_detail.currency`** = ยอดของคุณ (คนที่ login · sub = ยอดของเจ้าของ) · Superadmin = ยอดจริงของ Superadmin
- **`total_sum`** = ยอดรวมของลูกตรง**ทั้งหมด** (ทุกหน้า · ตาม `keyword` ที่ส่ง) แยกตาม `role` — key เป็น**ตัวเล็ก** (`agent` · `member` · `shareholder` · `company`) ต่างจาก field `role` ในแถวที่เป็นตัวใหญ่ · มีเฉพาะ role ที่มีลูก · ไม่มีลูก = `{}`
- **`currency`** (ทุก map) = ครบ 27 สกุลที่ระบบรองรับ · key ตัวใหญ่ · สกุลที่บัญชีไม่มี / ยังไม่มียอด = `0` · บัญชี Seamless แสดงยอดจริง
- เรียง username A→Z · ไม่มีลูก = `"data": []` (`credit_detail` ยังมีเสมอ)
- `role`: `COMPANY` / `SHAREHOLDER` / `AGENT` / `MEMBER` — ส่งค่านี้กลับใน 2.2 ได้เลย
- `status`: `ACTIVE` / `SUSPENDED` / `LOCKED` (สถานะของลูกเอง)
- `last_login_ip` / `last_login_at`: ยังไม่เคย login = `""`
- ตัวอย่างนี้ `mem01` ถูกล็อก: ปุ่มฝากปิด · ปุ่มถอนเปิด

สกุลทั้ง 27 ตัว: `ARS` `AUD` `BDT` `BOB` `BRL` `CLP` `CNY` `EUR` `GBP` `HKD` `IDR` `INR` `JPY` `KHR` `KRW` `LAK` `MMK` `MXN` `MYR` `NGN` `PHP` `PKR` `THB` `TWD` `USD` `USDT` `VND`

### 2.2 ฝาก / ถอน — `deposit-withdraw/create`

```bash
curl -X POST "{{MG_URL}}/api/v1/bo/pr/deposit-withdraw/create" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" \
  -d '{"request_id":"{{$guid}}","role":"MEMBER","user_id":501,"currency":"THB","amount":100,"transaction_type":"DEPOSIT","note":"เติมเครดิตรอบแรก","passcode":"123456"}'
```

ถอน — เปลี่ยน `transaction_type` และใช้ `request_id` ใหม่:
```bash
curl -X POST "{{MG_URL}}/api/v1/bo/pr/deposit-withdraw/create" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" \
  -d '{"request_id":"{{$guid}}","role":"AGENT","user_id":15,"currency":"THB","amount":50,"transaction_type":"WITHDRAW","passcode":"123456"}'
```

| Field | บังคับ | กติกา |
|---|---|---|
| `request_id` | ✓ | UUID **สร้างใหม่ทุกครั้งที่กดปุ่ม** · ส่งซ้ำ (เน็ตหลุด / timeout / ได้ `404307`) ใช้ค่าเดิม = ได้ผลเดิม ไม่ย้ายเงินซ้ำ |
| `role` | ✓ | ค่า `role` จากแถวใน 2.1 (ตัวพิมพ์ใหญ่) |
| `user_id` | ✓ | ค่า `id` จากแถวใน 2.1 |
| `currency` | ✓ | สกุลที่ทั้งคุณและลูกมี (ไม่ใช่ = `404303`) · ⚠️ map `currency` ใน 2.1 มีครบ 27 สกุล — ค่า `0` แยกไม่ได้ว่า "มีสกุลนี้แต่ยอด 0" หรือ "ไม่มีสกุลนี้" |
| `amount` | ✓ | > 0 · ≤ 1,000,000,000 · ทศนิยม ≤ 4 ตำแหน่ง · ส่งเป็นข้อความ (`"100"`) = `400` |
| `transaction_type` | ✓ | `DEPOSIT` / `WITHDRAW` (ตัวพิมพ์ใหญ่เท่านั้น) |
| `note` | | หมายเหตุ ≤ 200 ตัว · ไม่มี = ไม่ส่ง หรือ `""` · ช่องว่างหัวท้ายถูกตัด |
| `passcode` | ✓ | passcode ของคนที่กด (ส่งเป็น string เช่น `"012345"`) |

Response `data`:
```json
{
  "id": 9001,
  "request_id": "6f1c2a9e-8b1d-4c1e-9a55-2f0d6b7c1a10",
  "transaction_type": "DEPOSIT",
  "role": "MEMBER",
  "user_id": 501,
  "username": "mem01",
  "currency": "THB",
  "amount": 100,
  "note": "เติมเครดิตรอบแรก",
  "parent_unlimited": false,
  "parent_balance_after": 4900,
  "target_balance_after": 200,
  "created_at": "2026-10-14T10:00:00.123456+07:00"
}
```
- `parent_balance_after` = ยอดของคุณหลังทำรายการ · `target_balance_after` = ยอดของลูกหลังทำรายการ
- `parent_unlimited` = `false` เสมอ (รวม Superadmin) · แสดง `parent_balance_after` ได้ทุกบัญชี
- ส่ง `request_id` เดิมพร้อมค่าเดิมทุก field = ได้ `data` เดิม (ไม่ใช่ error) · ค่าใดต่างจากเดิม = `422` "request_id นี้ถูกใช้แล้ว"
- **ได้ `404307`** = มีรายการของลูกคนนี้กำลังทำอยู่ (เช่น กดซ้ำเร็ว ๆ หรือมีคนอื่นทำพร้อมกัน) — รอสักครู่แล้วส่งใหม่ด้วย **`request_id` เดิม**

**ลำดับ error เมื่อผิดหลายอย่าง:** ADMIN `404305` → สิทธิ์ sub `402303` → passcode → รูปแบบ body (`400` / `422`) → `404307` / `504801` →
`request_id` `422` → ไม่ใช่ลูกตรง `404401` → ลูกถูกระงับ (เฉพาะฝาก) `404301` → สกุล `404303` → ยอดไม่พอ `404304` / `404306`

### 2.3 ประวัติ — `deposit-withdraw/logs/list`

```bash
curl -X POST "{{MG_URL}}/api/v1/bo/pr/deposit-withdraw/logs/list" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" \
  -d '{"date_from":"2026-10-01","date_to":"2026-10-14","transaction_type":"","role":"MEMBER","user_id":501,"page":1,"limit":20}'
```

| Field | บังคับ | กติกา |
|---|---|---|
| `date_from` / `date_to` | ✓ | `YYYY-MM-DD` เวลาไทย · รวมทั้ง 2 วัน · `date_from` ≤ `date_to` · ช่วงไม่เกิน 31 วัน |
| `transaction_type` | | `DEPOSIT` / `WITHDRAW` · ไม่ส่ง / `""` = ทั้งหมด |
| `role` + `user_id` | | ดูลูกคนเดียว · **ต้องส่งคู่กัน** · ไม่ส่ง / `""` + `0` = ทุกลูก |
| `page` / `limit` | | ค่าเริ่มต้น 1 / 20 · สูงสุด 100 |

Response `data` (เรียงใหม่ → เก่า):
```json
{
  "current_page": 1, "total_page": 1, "total_count": 3, "limit": 20, "has_next": false, "has_prev": false,
  "data": [
    {
      "id": 9003, "request_id": "7a1d0c4e-2b3f-4c5d-8e9f-0a1b2c3d4e5f", "transaction_type": "DEPOSIT", "status": "FAILED",
      "actor_username": "share01", "role": "MEMBER", "user_id": 501, "username": "mem01",
      "currency": "THB", "amount": 99999, "note": "เติมโบนัส · ล้มเหลว: ยอดเงินของคุณไม่พอ", "ip": "203.0.113.10", "parent_unlimited": false,
      "parent_balance_before": 4950, "parent_balance_after": 4950,
      "target_balance_before": 150, "target_balance_after": 150,
      "created_at": "2026-10-14T12:00:00.111111+07:00"
    },
    {
      "id": 9002, "request_id": "0b7e9d1c-3a2f-4e8b-9c11-5d6e7f8a9b0c", "transaction_type": "WITHDRAW", "status": "SUCCESS",
      "actor_username": "share01@staff", "role": "MEMBER", "user_id": 501, "username": "mem01",
      "currency": "THB", "amount": 50, "note": "", "ip": "203.0.113.25", "parent_unlimited": false,
      "parent_balance_before": 4900, "parent_balance_after": 4950,
      "target_balance_before": 200, "target_balance_after": 150,
      "created_at": "2026-10-14T11:00:00.654321+07:00"
    },
    {
      "id": 9001, "request_id": "6f1c2a9e-8b1d-4c1e-9a55-2f0d6b7c1a10", "transaction_type": "DEPOSIT", "status": "SUCCESS",
      "actor_username": "share01", "role": "MEMBER", "user_id": 501, "username": "mem01",
      "currency": "THB", "amount": 100, "note": "เติมเครดิตรอบแรก", "ip": "203.0.113.10", "parent_unlimited": false,
      "parent_balance_before": 5000, "parent_balance_after": 4900,
      "target_balance_before": 100, "target_balance_after": 200,
      "created_at": "2026-10-14T10:00:00.123456+07:00"
    }
  ]
}
```
- เห็นเฉพาะรายการที่บัญชีของคุณ (หรือ sub ของคุณ) เป็นคนทำ — ลูกไม่เห็นรายการที่แม่ทำกับตัวเองในเส้นนี้
- `actor_username` = คนที่กดจริง (sub เช่น `share01@staff`) · `ip` = ip ของคนที่กด · `username` / `role` = ค่าตอนทำรายการ
- `parent_unlimited = true` มีเฉพาะรายการเก่าก่อน 15 ต.ค. ที่ Superadmin ยังไม่มียอด → `parent_balance_before` / `after` = `0` ไม่ต้องแสดง

**`status` ของรายการ** (ใช้ทั้ง 2.3 และ 2.4):

| ค่า | ความหมาย | แสดงอย่างไร |
|---|---|---|
| `SUCCESS` | ย้ายเงินสำเร็จ | ยอดก่อน / หลังเปลี่ยนตาม `amount` |
| `FAILED` | กดแล้ว**ไม่สำเร็จ — เงินไม่ขยับ** | ยอดก่อน = ยอดหลัง (ยอด ณ ตอนบันทึก) · เหตุผลอยู่ท้าย `note` |

- **แถว `FAILED` เกิดเมื่อ** ระบบปฏิเสธหลังหาลูกเจอแล้ว — ลูกถูกระงับ / ล็อก (ฝาก) · สกุลไม่ตรง · ยอดของคุณไม่พอ · ยอดของลูกไม่พอ — หรือระบบขัดข้องระหว่างย้ายเงิน
- **ไม่มีแถวเลย** เมื่อ body ผิด · ไม่มีสิทธิ์ · passcode ผิด · ไม่ใช่ลูกตรง · ได้ `404307` (มีรายการอื่นกำลังทำ) — กรณีเหล่านี้ยังไม่ได้เริ่มย้ายเงิน
- **`note` ของแถว `FAILED`** = ข้อความที่คนกดพิมพ์ (ถ้ามี) + `" · ล้มเหลว: "` + เหตุผล เช่น `"เติมโบนัส · ล้มเหลว: ยอดเงินของคุณไม่พอ"` / ไม่ได้พิมพ์ = `"ล้มเหลว: ยอดเงินของคุณไม่พอ"` · ระบบขัดข้อง = `"ล้มเหลว: เกิดข้อผิดพลาดภายในระบบ"`
- **`request_id` เดียวกันอาจมีหลายแถว** (เช่น `FAILED` แล้วกดซ้ำด้วย `request_id` เดิมจนสำเร็จ = แถว `FAILED` + `SUCCESS`) — ใช้ `id` เป็น key ของแถว ไม่ใช่ `request_id`
- ไม่มีสถานะ "กำลังดำเนินการ" — การฝาก / ถอนจบใน request เดียว (สำเร็จหรือไม่สำเร็จทันที)

### 2.4 ประวัติของบัญชีในสาย — `deposit-withdraw/history/list`

ใช้กับหน้าจอ "ประวัติฝาก / ถอน": พิมพ์ชื่อผู้ใช้งาน + ช่วงเวลา แล้วดูว่า**ใครทำรายการให้บัญชีนั้นบ้าง** · ค้นได้ทุกบัญชีที่อยู่**ใต้คุณทุกชั้น** (agent และ Member)

```bash
curl -X POST "{{MG_URL}}/api/v1/bo/pr/deposit-withdraw/history/list" \
  -H "Authorization: Bearer {{TOKEN}}" -H "Content-Type: application/json" \
  -d '{"username":"share0041","starttime":"2026-10-15T00:00:00+07:00","endtime":"2026-10-15T23:59:59+07:00","page":1,"limit":100}'
```

| Field | บังคับ | กติกา |
|---|---|---|
| `username` | ✓ | ชื่อผู้ใช้งาน**ตรงตัว** (ไม่สนตัวพิมพ์ · ตัดช่องว่างหัวท้าย) · ไม่ส่ง / `""` = `422` "ต้องระบุชื่อผู้ใช้งาน" |
| `starttime` / `endtime` | ✓ | วันเวลารูปแบบ RFC3339 **ต้องมี timezone** (เช่น `2026-10-15T00:00:00+07:00`) · `starttime` ≤ `endtime` · รวมทั้ง 2 ขอบ · **ไม่จำกัดช่วง** |
| `page` / `limit` | | ค่าเริ่มต้น 1 / 20 · สูงสุด 100 (**ใช้ `limit` ไม่ใช่ `perpage`**) |

Response `data` (เรียงใหม่ → เก่า):
```json
{
  "current_page": 1, "total_page": 1, "total_count": 1, "limit": 100, "has_next": false, "has_prev": false,
  "data": [
    {
      "id": 13110,
      "agent_id": 7374,
      "agent_username": "share0041",
      "role": "SHAREHOLDER",
      "request_by": 7368,
      "request_type": "AGENT",
      "request_username": "com0004",
      "currency": "THB",
      "transaction_type": "DEPOSIT",
      "status": "SUCCESS",
      "amount": 1000,
      "remark": "",
      "created_at": "2026-10-15T11:58:24.123456+07:00"
    }
  ]
}
```

| คอลัมน์บนหน้าจอ | field |
|---|---|
| ผู้ทำรายการ | `request_username` — **คนที่กดจริง** (sub เช่น `com0004@staff`) |
| การจัดการ | `transaction_type` — `DEPOSIT` = ฝากเงิน · `WITHDRAW` = ถอนเงิน |
| สถานะ | `status` — `SUCCESS` = สำเร็จ · `FAILED` = ไม่สำเร็จ (เงินไม่ขยับ · เหตุผลท้าย `remark`) — ดูตาราง status ใน 2.3 |
| ทำรายการให้ | `agent_username` (บัญชีที่ค้น · Member ใช้ field นี้เหมือนกัน — ดูจาก `role`) |
| จำนวน / ค่าเงิน / หมายเหตุ / วันที่ | `amount` · `currency` (ตัวใหญ่ `THB`) · `remark` (`""` = ไม่มี) · `created_at` |

- **ค้นตัวเองไม่ได้** · บัญชีชั้นบน / สายอื่น / ไม่มีในระบบ / บัญชีย่อย (`xxx@yyy`) = `404402` "ไม่พบข้อมูลบัญชี" (ข้อความเดียวกันทุกกรณี)
- ไม่มีรายการในช่วงที่เลือก = `200` + `"data": []`
- `request_type`: `AGENT` / `SUB` — `request_by` เป็น id ของตารางตาม `request_type` (id ของ 2 ตารางซ้ำกันได้)
- `endtime` = `23:59:59` จะไม่รวมรายการในเสี้ยววินาทีสุดท้ายของวัน — ต้องการครบวันให้ส่ง `23:59:59.999999+07:00`
- `amount` เป็นค่าบวกเสมอ — ดูทิศจาก `transaction_type`
- มีแถว `FAILED` ด้วย (ความหมายเดียวกับ 2.3) · `remark` ของแถว `FAILED` = ข้อความที่พิมพ์ + `" · ล้มเหลว: เหตุผล"`

Error codes: `422`, `404402`

## 3. Error codes

| Code | HTTP | ความหมาย | เส้น |
|---|---|---|---|
| `400` | 200 | body อ่านไม่ได้ / ตัวเลขส่งเป็นข้อความ (ไม่บอก field) | ทุกเส้น |
| `422` | 200 | ข้อมูลไม่ถูกต้อง (msg บอก field เช่น "จำนวนเงิน ต้องไม่เกิน 1,000,000,000") · `request_id` ถูกใช้แล้ว | ทุกเส้น |
| `402303` | 200 | sub ไม่มีสิทธิ์เมนู `payment` (ดู = view · ฝากถอน = edit) | ทุกเส้น |
| `401204` / `401205` / `401304` | — | passcode ไม่ถูกต้อง / ผิดครบ 5 ครั้ง / ยังไม่ได้ตั้ง passcode — ดู [auth.md](auth.md) | 2.2 |
| `404301` | 200 | ฝากให้บัญชีที่ถูกระงับ / ล็อกไม่ได้ | 2.2 |
| `404302` | — | เลิกใช้ 15 ต.ค. (สาย Seamless ฝาก / ถอนได้แล้ว) — ไม่มีเส้นไหนตอบ code นี้แล้ว | — |
| `404303` | 200 | สกุลเงินนี้ไม่มีในบัญชีของคุณหรือบัญชีปลายทาง | 2.2 |
| `404304` | 200 | ยอดเงินของคุณไม่พอ | 2.2 |
| `404305` | 200 | ADMIN ทำรายการฝาก / ถอนไม่ได้ | ทุกเส้น |
| `404306` | 200 | ยอดเงินของบัญชีปลายทางไม่พอ | 2.2 |
| `404307` | 200 | มีรายการของบัญชีนี้กำลังทำอยู่ กรุณาลองใหม่ | 2.2 |
| `404401` | 200 | ไม่พบบัญชีลูกตรงของคุณ (หลาน / นอกสาย / ไม่มี id / `role` ไม่ตรง) | 2.2 |
| `404402` | 200 | ไม่พบข้อมูลบัญชี (username ไม่มี / ไม่อยู่ใต้คุณ / เป็นตัวคุณเอง) | 2.4 |
| `504801` | **503** | ระบบไม่พร้อมทำรายการ กรุณาลองใหม่ | 2.2 |

error ร่วมของหลังบ้าน (token / session / ด่านหลัง login / บัญชีถูกระงับ) ดู [auth.md](auth.md)
