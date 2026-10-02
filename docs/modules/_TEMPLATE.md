# {Module name} — Spec

- สถานะ: DRAFT | APPROVED
- อนุมัติโดย: {ชื่อ} · วันที่: {YYYY-MM-DD}
- โค้ดเก่าที่อ้างอิง (เพื่อ extract พฤติกรรมเท่านั้น): `askmelotto_minigame-api/...`

> เอกสารนี้คือ source of truth ของ module — โค้ดและ test ต้องตรงกับเอกสารนี้ ถ้าไม่ตรงให้แก้โค้ด
> หรือแก้เอกสารผ่านการอนุมัติใหม่ ห้ามอ้างเอกสารของโปรเจกต์เก่า

## 1. หน้าที่ของ module

{อธิบายสั้นๆ ว่า module นี้มีไว้ทำอะไร ใครใช้}

## 2. คำศัพท์

| คำ | ความหมาย |
|---|---|

## 3. Business rules

แต่ละข้อมี id เพื่ออ้างใน test และ code (`// rule: AUTH-03`)

| ID | Rule | ตัวอย่างตัวเลข |
|---|---|---|
| XXX-01 | | |

## 4. สิ่งที่พบในโค้ดเก่า และการตัดสินใจ

| # | โค้ดเก่าทำอะไร | ถูก/ผิด | การตัดสินใจ |
|---|---|---|---|

## 5. Endpoints

| Method | Path | สิทธิ์ที่ต้องมี | Contract เทียบของเก่า |
|---|---|---|---|
| | | | เหมือนเดิม / เปลี่ยน (อธิบาย) |

### {METHOD} {path}

Request:
```json
```
Response `data`:
```json
```
Error codes: `...`

## 6. Schema

{ตารางที่ module นี้เป็นเจ้าของ + เหตุผลของ index/constraint สำคัญ}

## 7. Test cases

| Rule ID | Input | Expected |
|---|---|---|

## 8. Contract changes (สำหรับแจ้ง frontend/provider)

{รายการ breaking change ถ้ามี}
