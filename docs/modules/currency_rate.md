# อัตราแลกเปลี่ยน (`currency_rate`) — Spec

- สถานะ: **DRAFT**
- อนุมัติโดย: — · วันที่: —
- ชื่อ module ในโค้ด: `currency_rate` (`service/currency_rate`, `core/currency_rate`, `repository/redis`)
- ที่มาของ rule: คำตอบของ boiledegg (2026-10-07) + ผลการยิงเส้นจริงบน `https://dev-api.amblotto.net` (2026-10-07) ·
  ไม่ได้ extract จากโค้ดเก่า

> เอกสารนี้คือ source of truth ของ module — โค้ดและ test ต้องตรงกับเอกสารนี้ ถ้าไม่ตรงให้แก้โค้ด
> หรือแก้เอกสารผ่านการอนุมัติใหม่ ห้ามอ้างเอกสารของโปรเจกต์เก่า

## 1. หน้าที่ของ module

ดึงอัตราแลกเปลี่ยนเทียบ USDT ของ 27 สกุลจากระบบ askme ทุก 15 นาที เก็บค่าล่าสุดไว้ใน Redis
แล้วให้ module อื่น (Dashboard ของ account, รายงาน) ใช้**แปลงยอดข้ามสกุลเพื่อแสดงผล**

**ไม่อยู่ใน module นี้:** การย้ายเงินหรือแลกเงินจริงระหว่างสกุล (เรทนี้ใช้แสดงผลเท่านั้น — ยอดเงินจริงยังแยกสกุลตาม ACC-19 / MGMT-15A) ·
การแสดงยอดที่แปลงแล้วในแต่ละหน้า (กำหนดใน spec ของหน้านั้น)

## 2. คำศัพท์

| คำ | ความหมาย |
|---|---|
| เรท (rate) | 1 USDT = กี่หน่วยของสกุลนั้น เช่น THB `33.62` = 1 USDT แลกได้ 33.62 บาท |
| askme | ระบบภายนอกที่เป็นแหล่งเรท (`POST {CURRENCY_RATE_API_BASE_URL}/api/ex/rate/usdt`) |
| รอบ sync | การดึงเรท 1 ครั้ง |

## 3. Business rules

### การดึงเรท

| ID | Rule |
|---|---|
| CR-01 | ดึงเรท**ทุก 15 นาที** ด้วย worker ใน server (goroutine เดียว มีขอบเขต รับ ctx และหยุดตอน server ปิด — กฎข้อ 27) · ดึง 1 รอบทันทีตอน server เริ่ม |
| CR-02 | **เปิดเป็นค่าเริ่มต้น** · ปิดด้วย `IS_CURRENCY_RATE_SYNC=false` (ไม่เริ่ม worker เลย) |
| CR-03 | **Redis lock กันหลาย instance ยิงพร้อมกัน**: ก่อนยิงทำ `SET currency_rate:sync:lock {token} NX PX 300000` (5 นาที) · ได้ lock = ยิง · ไม่ได้ = ข้ามรอบนี้ · จบแล้วปล่อย lock เฉพาะถ้า token ยังเป็นของตัวเอง (Lua compare-and-delete) |
| CR-04 | request: `POST {CURRENCY_RATE_API_BASE_URL}/api/ex/rate/usdt` · header `Content-Type: application/json` · body `{"date":"YYYY-MM-DD"}` = **วันปัจจุบันตามเวลาไทย (Asia/Bangkok)** · timeout 10 วินาที · ไม่ต้องยืนยันตัวตน |
| CR-05 | ถือว่าสำเร็จเมื่อ: HTTP 200 **และ** `code = 200` **และ** `data.base_currency = "USDT"` **และ** `data.json_rate` เป็น object · error ของ askme ตอบ HTTP 200 แต่ `code` ไม่ใช่ 200 (เช่น `430502` รูปแบบวันที่ผิด · `430503` ยังไม่มีเรทของวันนั้น) |
| CR-06 | เรทแต่ละสกุลต้องเป็น number มากกว่า 0 · อ่าน JSON number เป็นข้อความแล้วแปลงเป็นจำนวนเต็ม**คูณ 10^8** (ไม่ผ่าน float — กฎข้อ 9) · ทศนิยมเกิน 8 ตำแหน่ง = ไม่รับสกุลนั้น · รับเฉพาะ 27 สกุลของระบบ (สกุลอื่นใน response ไม่สนใจ) |
| CR-07 | **ไม่สำเร็จ** (เชื่อมต่อไม่ได้, timeout, HTTP ไม่ใช่ 200, `code` ไม่ใช่ 200 รวม `430503`, รูปแบบผิด) → **คงเรทล่าสุดที่มีไว้** · log warn (มี `code` / `msg` ของ askme) · ไม่ retry ในรอบเดียวกัน รอรอบถัดไป |
| CR-08 | สกุลที่ response ขาดหรือค่าผิด (CR-06) → คงเรทเดิมของสกุลนั้น · log warn บอกสกุล · สกุลอื่นอัปเดตตามปกติ |

### การเก็บและการใช้

| ID | Rule |
|---|---|
| CR-10 | เก็บค่าล่าสุดใน **Redis เท่านั้น** (ไม่มีตาราง DB) · key `currency_rate:usdt` (hash: สกุล → เรท ×10^8 เป็นข้อความ) · key `currency_rate:usdt:meta` (hash: `date` ของเรท, `synced_at`) · ไม่มี TTL (เก็บค่าล่าสุดไว้จนกว่ารอบใหม่สำเร็จ) |
| CR-11 | เขียนทั้งชุดในคำสั่งเดียว (`MULTI` / `HSET` ทั้ง 2 key / `EXEC`) — ผู้อ่านไม่เห็นเรทครึ่งชุด |
| CR-12 | Redis หายหรือถูกล้าง → ยังไม่มีเรทจนกว่ารอบถัดไปสำเร็จ (ไม่เกิน 15 นาที หรือตอน server เริ่ม) · ผู้ใช้เรทต้องรับมือกรณีไม่มีเรท (CR-14) |
| CR-13 | แปลงยอด: `ยอดในสกุล B = ยอดในสกุล A × เรท B ÷ เรท A` · คำนวณด้วยจำนวนเต็ม (big int) ทั้งหมด · ปัดผลเป็นหน่วยย่อย (1/100) แบบ**ปัดครึ่งขึ้น** · ใช้**แสดงผลเท่านั้น** ห้ามใช้ย้ายเงิน |
| CR-14 | ไม่มีเรทของสกุลใดสกุลหนึ่งที่ต้องใช้ → ฟังก์ชันแปลงคืน error `ErrRateUnavailable` · หน้าที่เรียกตัดสินเองว่าจะแสดงอย่างไร (กำหนดใน spec ของหน้านั้น) |
| CR-15 | Redis ไม่ใช่ source of truth ของยอดเงิน (กฎข้อ 14) — เรทเป็นข้อมูลอ้างอิงสำหรับแสดงผล ไม่ใช่ยอดเงิน |

## 4. สิ่งที่พบในโค้ดเก่า และการตัดสินใจ

ไม่มี — ไม่ได้ extract จากโค้ดเก่า · ข้อมูลของเส้น askme มาจากการยิงจริงบน dev (หัวข้อ 5)

## 5. เส้นภายนอกที่เรียก (askme)

`POST {CURRENCY_RATE_API_BASE_URL}/api/ex/rate/usdt`

Request:
```json
{ "date": "2026-10-07" }
```

Response สำเร็จ (ยิงจริง 2026-10-07 · 0.8–1.8 วินาที):
```json
{
  "code": 200,
  "data": {
    "date": "2026-10-07",
    "base_currency": "USDT",
    "json_rate": {
      "THB": 33.62, "MMK": 2099.53, "CNY": 6.7, "EUR": 0.88, "GBP": 0.75, "IDR": 17849.27, "INR": 96.37,
      "JPY": 158.15, "KHR": 4046.24, "KRW": 1337.9, "LAK": 22394.92, "PHP": 62.77, "USD": 0.99, "VND": 25988.07,
      "AUD": 1.43, "TWD": 31.78, "MYR": 4.08, "HKD": 7.84, "MXN": 17.97, "USDT": 1, "BRL": 4.98, "BDT": 122.75,
      "PKR": 276.6, "CLP": 973.67, "ARS": 1520.54, "BOB": 11.5, "NGN": 1324.42
    }
  },
  "msg": { "en": "Success", "th": "สำเร็จ" }
}
```

Response error (HTTP 200 ทุกกรณี):

| `code` | เมื่อ |
|---|---|
| 430502 | ไม่ส่ง `date` หรือรูปแบบผิด (ต้องเป็น `YYYY-MM-DD`) |
| 430503 | ยังไม่มีเรทของวันที่ระบุ (เช่น วันในอนาคต) |

ไม่มี endpoint ของเราเองใน module นี้ (worker ภายใน)

## 6. Config และ Schema

| env | ค่าเริ่มต้น | กฎ |
|---|---|---|
| `IS_CURRENCY_RATE_SYNC` | `true` | `false` = ไม่เริ่ม worker (CR-02) |
| `CURRENCY_RATE_API_BASE_URL` | `https://dev-api.amblotto.net` (host ฝั่ง dev) | ต้องเป็น **origin เปล่า** (`scheme://host[:port]`) ไม่มี path, query หรือ `/` ต่อท้าย ไม่งั้น server ไม่เริ่ม · **บังคับตั้งใน uat / prod** (กันชี้ไป dev โดยไม่ตั้งใจ) · local / dev ไม่ตั้ง = ใช้ค่าเริ่มต้น |

- เพิ่มใน `pkg/configs` และ `.env.example` ใน PR เดียวกัน (กฎข้อ 25)
- ไม่มีตาราง DB · Redis key ตาม CR-03, CR-10

## 7. Test cases

| Rule | Input | Expected |
|---|---|---|
| CR-02 | `IS_CURRENCY_RATE_SYNC=false` | ไม่เริ่ม worker · ไม่มีการยิง |
| CR-03 | 2 instance เริ่มรอบพร้อมกัน | ยิงแค่ 1 ครั้ง |
| CR-03 | instance ที่ถือ lock ค้างจน lock หมดอายุ แล้วอีก instance ได้ lock | instance แรกไม่ลบ lock ของอีก instance |
| CR-04 | เวลา 2026-10-07 23:30 UTC | ส่ง `date = 2026-10-08` (เวลาไทย) |
| CR-05 | `code = 430503` | ไม่สำเร็จ · เรทเดิมไม่เปลี่ยน · log warn |
| CR-05 | HTTP 500 / timeout / body ไม่ใช่ JSON / `base_currency = "USD"` | ไม่สำเร็จ · เรทเดิมไม่เปลี่ยน |
| CR-06 | `"THB": 33.62` | เก็บ `3362000000` |
| CR-06 | `"THB": 0` / `-1` / `"33.62"` (string) / ทศนิยม 9 ตำแหน่ง | ไม่รับ THB · คง THB เดิม (CR-08) |
| CR-06 | response มีสกุล `XYZ` | ไม่สนใจ |
| CR-08 | response ไม่มี `KRW` | KRW คงค่าเดิม · สกุลอื่นอัปเดต · log warn |
| CR-11 | อ่านระหว่างกำลังเขียน | ได้ชุดเก่าทั้งชุดหรือชุดใหม่ทั้งชุด |
| CR-13 | 100.00 THB → USD (THB 33.62 · USD 0.99) | 2.94 USD (100 × 0.99 ÷ 33.62 = 2.9446… ปัดครึ่งขึ้น) |
| CR-13 | 1,000.00 USDT → VND (VND 25988.07) | 25,988,070.00 VND |
| CR-14 | ยังไม่มีเรทใน Redis แล้วเรียกแปลง | `ErrRateUnavailable` |
| config | `CURRENCY_RATE_API_BASE_URL=https://x.example/api` / `https://x.example/` | server ไม่เริ่ม (ต้องเป็น origin เปล่า) |
| config | `APP_ENV=prod` ไม่ตั้ง `CURRENCY_RATE_API_BASE_URL` | server ไม่เริ่ม |

## 8. Contract changes (แจ้ง frontend)

ไม่มี endpoint ใหม่ · หน้าที่แสดงยอดแปลงสกุล (Dashboard / รายงาน) จะระบุใน spec ของหน้านั้น
