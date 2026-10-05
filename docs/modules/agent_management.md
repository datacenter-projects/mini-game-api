# การจัดการสมาชิก (`agent_management`) ช่วง 1 — บริษัท · ตัวแทน · เอเย่นต์ — Spec

- สถานะ: **DRAFT**
- อนุมัติโดย: — · วันที่: —
- ชื่อ module ในโค้ด: `agent_management` (`controllers/agent_management`, `dto/agent_management`, `service/agent_management`, `core/agent_management`)
- เมนูในช่วงนี้: 2.1 เพิ่มบริษัท · 2.2 รายชื่อบริษัท · 2.3 เพิ่มตัวแทน · 2.4 รายชื่อตัวแทน · 2.5 เพิ่มเอเย่นต์ ·
  2.7 รายชื่อสมาชิกและเอเย่นต์ (ส่วนเอเย่นต์)
- ที่มาของ rule: เอกสารของ lead (Company Hierarchy, PT, PT Force, PT Remain, PT Commission, System Overview) +
  คำตอบของ boiledegg ในการวางแผน (2026-10-05) · ไม่ได้ extract จากโค้ดเก่า

> เอกสารนี้คือ source of truth ของช่วง 1 — โค้ดและ test ต้องตรงกับเอกสารนี้ ถ้าไม่ตรงให้แก้โค้ด
> หรือแก้เอกสารผ่านการอนุมัติใหม่ ห้ามอ้างเอกสารของโปรเจกต์เก่า

## 1. หน้าที่ของ module

สร้างและจัดการบัญชีในสายงาน (Company, Share, Agent): สร้างบัญชี, ดูรายชื่อในสายของตัวเอง, เปลี่ยนสถานะ,
ตั้ง/แก้ PT · Force · Remain · Commission ให้ลูก

**อยู่ในช่วงนี้:** สร้าง Company / Share / Agent · รายชื่อ + drill-down · เปลี่ยนสถานะ · ตั้ง/แก้ค่า PT ฯลฯ + ประวัติการแก้

**ไม่อยู่ในช่วงนี้:** สมาชิก (ช่วง 2) · บัญชีย่อยและสิทธิ์ (ช่วง 3) · คัดลอกสมาชิก (ช่วง 4) ·
ข้อมูลเชื่อม Seamless (module 1 เมนู 1.3) · การคำนวณ PT / Force / Remain / Commission ตอน settle (module เดิมพัน)

## 2. คำศัพท์

| คำ | ความหมาย |
|---|---|
| บริษัท / ตัวแทน / เอเย่นต์ / สมาชิก | Company / Share / Agent / Member (ตาม Company Hierarchy) |
| ผู้สร้าง | บัญชีที่สร้างบัญชีนั้น = `parent_id` |
| สายล่าง | ทุกบัญชีที่อยู่ใต้ตัวเองทุกชั้น (ลูก หลาน ...) ไม่นับตัวเอง |
| PT ที่ได้รับ | % ที่ผู้สร้างปล่อยให้ (`pt_bp`) · Superadmin = 100% |
| bp | ค่า % × 100 เก็บเป็นจำนวนเต็ม เช่น 95.50% = `9550` |

## 3. Business rules

### ประเภทและการสร้าง

| ID | Rule |
|---|---|
| MGMT-01 | role เดิม `COMPANY` / `SHAREHOLDER` / `AGENT` + คอลัมน์ `agent_type`: Company = `TRANSFER` / `SEAMLESS_RESELLER` / `SEAMLESS_MASTER` / `SEAMLESS_1TO1` · Share = `B2B` / `B2C` / `RESELLER` / `MASTER` · Agent ไม่มีประเภท |
| MGMT-02 | กฎการสร้าง (Company Hierarchy) · ผู้สร้าง → สร้างได้: |
| | • SUPERADMIN → Company ทุกประเภท (เลือกตอนสร้าง) |
| | • Company `TRANSFER` → Share `B2B` / `B2C` · `SEAMLESS_RESELLER` → Share `RESELLER` · `SEAMLESS_MASTER` → Share `MASTER` · `SEAMLESS_1TO1` → ไม่มี Share (สร้าง Member — ช่วง 2) |
| | • Share `B2B` / `B2C` → Agent · Share `RESELLER` / `MASTER` → ไม่มี Agent (สร้าง Member — ช่วง 2) |
| | • Agent → Agent ซ้อนได้**ไม่จำกัดชั้น** |
| | ประเภทของ Share ถูกกำหนดจากประเภทของ Company ตามตารางนี้ (Company Transfer เลือก B2B หรือ B2C) |
| MGMT-03 | สร้าง Company ได้เฉพาะ `SUPERADMIN` · `ADMIN` สร้างไม่ได้ |
| MGMT-04 | ผู้สร้างต้องมี `EffectiveStatus = ACTIVE` (AUTH-53) — SUSPENDED / LOCKED สร้างไม่ได้ |
| MGMT-05 | บัญชีที่สร้าง: `parent_id` = ผู้สร้าง · `status = ACTIVE` · ชั้นบนสร้างลูกได้หลายคน |
| MGMT-06 | username: `a-z` `0-9` ยาว 4–20 ตัว (ผ่าน AUTH-01) · ไม่ซ้ำทั้งระบบ · แก้ไม่ได้ (AUTH-19) |
| MGMT-07 | password ตาม AUTH-36 · บัญชีใหม่ตั้ง passcode ตอน login ครั้งแรก (AUTH-29) |
| MGMT-08 | ชื่อแสดง (`display_name`): ไม่บังคับ · ยาวไม่เกิน 100 ตัวอักษร · ตัดช่องว่างหัวท้าย |

### สกุลเงิน

| ID | Rule |
|---|---|
| MGMT-10 | Company และ Share ไม่ผูกสกุลเงิน (ใช้ได้ทุกสกุล) |
| MGMT-11 | Agent ตัวแรกใต้ Share: ผู้สร้างเลือกสกุลเงิน 1 สกุล (บังคับ) · Agent ที่ Agent สร้าง: ใช้สกุลของผู้สร้าง (ไม่ได้เลือก) · แก้ไม่ได้ |

### PT / Force / Remain / Commission

ผู้สร้างตั้งให้ลูกตอนสร้าง และแก้ภายหลังได้ (มีผลกับ bet ใหม่ — bet เก็บค่า ณ ตอน bet ใน module เดิมพัน)

| ID | Rule |
|---|---|
| MGMT-13 | ทุก % ละเอียด 2 ตำแหน่ง · API และ DB ใช้หน่วย bp (จำนวนเต็ม) |
| MGMT-14 | PT: ผู้สร้างปล่อยให้ลูก 0 – (PT ที่ตัวเองได้รับ) · Superadmin ได้รับ 100% (`10000`) · ลูกแต่ละคนตั้งแยกกัน |
| MGMT-15 | Company `SEAMLESS_MASTER`: PT ที่ปล่อยให้ Share Master ต้อง**เท่ากับ** PT ที่ Company ได้รับ (ถือ 0%) · ตั้ง Force ให้ Share Master ไม่ได้ |
| MGMT-16 | Force / Remain: ไม่ตั้ง = ไม่มี (`NULL`) · ตั้งได้ 0 – (PT ที่ปล่อยให้ลูกคนนั้น) |
| MGMT-17 | Commission: 0 – 1.00% (`0`–`100`) · บังคับตั้ง (0 ได้) |
| MGMT-18 | แก้ได้เฉพาะ**ผู้สร้าง**ของบัญชีนั้น |
| MGMT-19 | แก้ลด PT ของลูก จนต่ำกว่า PT ที่ลูกปล่อยต่อให้ลูกของมันคนใดคนหนึ่ง → error |
| MGMT-20 | แก้ PT จนต่ำกว่า Force / Remain ที่ตั้งไว้ → error (ต้องแก้ Force / Remain ในคำขอเดียวกัน) |
| MGMT-21 | ทุกการแก้บันทึก `agent_setting_logs` (ใครแก้, ค่าเก่า → ค่าใหม่, เมื่อไร) ใน tx เดียวกัน · tx `SELECT ... FOR UPDATE` แถวลูก และอ่านค่าของหลาน (MGMT-19) ใน tx เดียวกัน |

### รายชื่อ

| ID | Rule |
|---|---|
| MGMT-22 | เห็นเฉพาะสายล่างของตัวเอง ไม่เห็นข้ามสาย ไม่เห็นชั้นบน · ADMIN ไม่อยู่ในรายชื่อใด (AUTH-43) · Superadmin เห็นทุกสาย |
| MGMT-23 | รายชื่อบริษัท (SUPERADMIN) / รายชื่อตัวแทน / รายชื่อเอเย่นต์ (ทุก Agent ในสายล่าง ทุกชั้น) |
| MGMT-24 | ค้น username / ชื่อแสดง (ส่วนหนึ่งของคำ ไม่สนตัวพิมพ์) · กรอง `status` · กรอง `agent_type` · drill-down ดูลูกตรงของบัญชีในสายล่าง · pagination มาตรฐาน (`utils.ParsePage` + `response.Page`) · เรียงตามวันที่สร้างล่าสุดก่อน |
| MGMT-25 | แต่ละแถวแสดง: username, ชื่อแสดง, role, ประเภท, สถานะ, PT, Force, Remain, Commission, สกุลเงิน, username ผู้สร้าง, จำนวนลูกตรง, วันที่สร้าง, login ล่าสุด |

### สถานะ

| ID | Rule |
|---|---|
| MGMT-26 | เปลี่ยน `ACTIVE` / `SUSPENDED` / `LOCKED` ของใครก็ได้ในสายล่าง (ข้ามชั้นได้) · เปลี่ยนของตัวเองไม่ได้ |
| MGMT-27 | ผลกับชั้นล่างเป็นไปตาม AUTH-27 / AUTH-53 (ไม่แก้แถวของชั้นล่าง) · LOCK แล้ว session ของเป้าหมายและสายล่างหลุดใน request ถัดไป |
| MGMT-28 | SUSPENDED = login และดูได้ แต่สร้าง / แก้ / ทำรายการไม่ได้ · Member ทั้งสายเดิมพันไม่ได้ (บังคับใน module เดิมพันจาก EffectiveStatus) |
| MGMT-29 | ผู้เปลี่ยนต้องมี `EffectiveStatus = ACTIVE` |

### อื่นๆ

| ID | Rule |
|---|---|
| MGMT-30 | แก้อย่างอื่นไม่ได้ (username, ชื่อแสดง, ประเภท, สกุลเงิน) · ไม่มีการลบบัญชี |
| MGMT-31 | sub account ทำรายการในช่วงนี้ไม่ได้ทุกเส้น (ทั้งดูและแก้) จนกว่าช่วง 3 จะกำหนดสิทธิ์ |

## 4. สิ่งที่พบในโค้ดเก่า และการตัดสินใจ

ไม่มี — ไม่ได้ extract จากโค้ดเก่า

## 5. Endpoints

ทุกเส้นใต้ `/bo/pr` ผ่าน `Authenticated` + `PassedGates()` · error ร่วม: `401202`, `401203`, `401301`, `401302`, `401304`,
`401306`, `401307`, `402303` (sub)

| Method | Path | ใครเรียกได้ | เมนู |
|---|---|---|---|
| POST | `/api/v1/bo/pr/companies` | SUPERADMIN | 2.1 |
| GET | `/api/v1/bo/pr/companies` | SUPERADMIN | 2.2 |
| POST | `/api/v1/bo/pr/shares` | Company `TRANSFER` / `SEAMLESS_RESELLER` / `SEAMLESS_MASTER` | 2.3 |
| GET | `/api/v1/bo/pr/shares` | SUPERADMIN, Company | 2.4 |
| POST | `/api/v1/bo/pr/agents` | Share `B2B` / `B2C`, Agent | 2.5 |
| GET | `/api/v1/bo/pr/agents` | SUPERADMIN, Company, Share, Agent | 2.7 |
| GET | `/api/v1/bo/pr/accounts/:id/children` | ชั้นบนของ `:id` | drill-down |
| POST | `/api/v1/bo/pr/accounts/:id/status` | ชั้นบนของ `:id` | เปลี่ยนสถานะ |
| POST | `/api/v1/bo/pr/accounts/:id/settings` | ผู้สร้างของ `:id` | แก้ PT ฯลฯ |

### POST /api/v1/bo/pr/companies

Request:
```json
{
  "username": "comp01",
  "password": "••••••••",
  "display_name": "บริษัท เอบีซี",
  "agent_type": "TRANSFER",
  "pt_bp": 9500,
  "force_bp": null,
  "remain_bp": null,
  "commission_bp": 100
}
```
Response `data`:
```json
{ "id": 12, "username": "comp01" }
```
Error codes: `422`, `402301`, `402302`, `402305`, `402307`, `402308`, `402309`, `402401`

### POST /api/v1/bo/pr/shares

Request: เหมือน companies · `agent_type` = ประเภท Share ตาม MGMT-02
Error codes: เหมือน companies

### POST /api/v1/bo/pr/agents

Request: เหมือน companies แต่ไม่มี `agent_type` · มี `currency` (บังคับเมื่อผู้สร้างเป็น Share · ห้ามส่งเมื่อผู้สร้างเป็น Agent)
Error codes: เหมือน companies

### GET /api/v1/bo/pr/{companies,shares,agents}

Query: `page`, `limit`, `q` (ค้น username / ชื่อแสดง), `status`, `agent_type`
Response `data` (`response.Page`) แต่ละแถว:
```json
{
  "id": 12,
  "username": "comp01",
  "display_name": "บริษัท เอบีซี",
  "role": "COMPANY",
  "agent_type": "TRANSFER",
  "status": "ACTIVE",
  "pt_bp": 9500,
  "force_bp": null,
  "remain_bp": null,
  "commission_bp": 100,
  "currency": null,
  "parent_username": "root",
  "children_count": 3,
  "created_at": "2026-10-05T10:00:00+07:00",
  "last_login_at": null
}
```

### GET /api/v1/bo/pr/accounts/:id/children

Query และ Response เหมือนรายชื่อ · `:id` ต้องอยู่ในสายล่าง (หรือเป็นตัวเอง) · Error codes: `402402`

### POST /api/v1/bo/pr/accounts/:id/status

Request: `{ "status": "LOCKED" }` · Response: ไม่มี `data` · Error codes: `422`, `402302`, `402402`, `402304` (เปลี่ยนของตัวเอง)

### POST /api/v1/bo/pr/accounts/:id/settings

Request (ส่งครบทั้ง 4 ค่า — แทนค่าเดิมทั้งชุด):
```json
{ "pt_bp": 9000, "force_bp": 2000, "remain_bp": null, "commission_bp": 50 }
```
Response: ไม่มี `data` · Error codes: `422`, `402302`, `402304`, `402305`, `402306`, `402307`, `402308`, `402309`, `402402`

## 6. Schema

migration ไฟล์ใหม่

```sql
ALTER TABLE user_agents
    ADD COLUMN display_name  VARCHAR(100),
    ADD COLUMN agent_type    VARCHAR(20),
    ADD COLUMN currency      VARCHAR(10),
    ADD COLUMN pt_bp         INT,
    ADD COLUMN force_bp      INT,
    ADD COLUMN remain_bp     INT,
    ADD COLUMN commission_bp INT,
    ADD CONSTRAINT ck_user_agents_agent_type CHECK (
        (role = 'COMPANY' AND agent_type IN ('TRANSFER','SEAMLESS_RESELLER','SEAMLESS_MASTER','SEAMLESS_1TO1')) OR
        (role = 'SHAREHOLDER' AND agent_type IN ('B2B','B2C','RESELLER','MASTER')) OR
        (role NOT IN ('COMPANY','SHAREHOLDER') AND agent_type IS NULL)),
    ADD CONSTRAINT ck_user_agents_pt CHECK (pt_bp IS NULL OR pt_bp BETWEEN 0 AND 10000),
    ADD CONSTRAINT ck_user_agents_force CHECK (force_bp IS NULL OR force_bp BETWEEN 0 AND 10000),
    ADD CONSTRAINT ck_user_agents_remain CHECK (remain_bp IS NULL OR remain_bp BETWEEN 0 AND 10000),
    ADD CONSTRAINT ck_user_agents_commission CHECK (commission_bp IS NULL OR commission_bp BETWEEN 0 AND 100);

CREATE INDEX idx_user_agents_role_created ON user_agents(role, created_at DESC);

CREATE TABLE agent_setting_logs (
    id                BIGSERIAL   PRIMARY KEY,
    agent_id          BIGINT      NOT NULL REFERENCES user_agents(id),
    actor_type        VARCHAR(10) NOT NULL,  -- AGENT / SUB
    actor_id          BIGINT      NOT NULL,
    actor_username    VARCHAR(71) NOT NULL,
    old_pt_bp         INT, new_pt_bp         INT,
    old_force_bp      INT, new_force_bp      INT,
    old_remain_bp     INT, new_remain_bp     INT,
    old_commission_bp INT, new_commission_bp INT,
    ip                VARCHAR(45),
    request_id        VARCHAR(64),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_agent_setting_logs_agent ON agent_setting_logs(agent_id, created_at);
```

- SUPERADMIN / ADMIN: `pt_bp` และค่าอื่นเป็น `NULL` (Superadmin ได้รับ 100% โดยนิยาม)
- Company / Share / Agent ที่สร้างผ่าน module นี้ต้องมี `pt_bp` และ `commission_bp` เสมอ (บังคับใน service)
- สายล่างใช้ recursive CTE ตาม `parent_id` (แบบเดียวกับ `ListAncestorStatusesRepository`) · `idx_user_agents_parent_id` มีแล้ว

## 7. Test cases

| Rule | Input | Expected |
|---|---|---|
| MGMT-02 | Company `TRANSFER` สร้าง Share `B2C` | สำเร็จ |
| MGMT-02 | Company `TRANSFER` สร้าง Share `RESELLER` | `402301` |
| MGMT-02 | Company `SEAMLESS_1TO1` สร้าง Share | `402301` |
| MGMT-02 | Share `RESELLER` สร้าง Agent | `402301` |
| MGMT-02 | Agent สร้าง Agent ซ้อน 3 ชั้น | สำเร็จทุกชั้น |
| MGMT-03 | ADMIN / Company สร้าง Company | `402301` |
| MGMT-04 | ผู้สร้าง SUSPENDED หรือ upline SUSPENDED | `402302` |
| MGMT-06 | username `ab1` / 21 ตัว / มีตัวพิมพ์ใหญ่หลัง normalize ไม่ได้ / มี `_` | `422` |
| MGMT-06 | username ซ้ำกับบัญชีที่มีอยู่ | `402401` |
| MGMT-07 | password ผิดกติกา AUTH-36 | `422` |
| MGMT-08 | ชื่อแสดง 101 ตัว | `422` · ไม่ส่ง = สำเร็จ |
| MGMT-11 | Share สร้าง Agent ไม่ส่ง `currency` | `422` |
| MGMT-11 | Agent สร้าง Agent | สกุลเท่ากับผู้สร้าง |
| MGMT-14 | Company ได้ 95% สร้าง Share 96% | `402305` |
| MGMT-15 | Company Master ได้ 95% สร้าง Share Master 90% | `402307` · 95% = สำเร็จ |
| MGMT-15 | Company Master ตั้ง Force ให้ Share Master | `402308` |
| MGMT-16 | ปล่อย PT 50% ตั้ง Force 60% | `402309` |
| MGMT-17 | commission 1.01% (`101`) | `422` |
| MGMT-18 | Company แก้ settings ของ Agent (หลาน) | `402304` |
| MGMT-19 | Company ลด PT ของ Share จาก 80% เหลือ 50% ขณะที่ Share ปล่อยให้ Agent 60% | `402306` |
| MGMT-20 | ลด PT เหลือ 50% แต่ Force เดิม 60% และไม่ได้แก้ Force | `402309` |
| MGMT-21 | แก้สำเร็จ | มีแถวใน `agent_setting_logs` ค่าเก่า/ใหม่ถูกต้อง |
| MGMT-22 | Company A ดูรายชื่อเอเย่นต์ | เห็นเฉพาะ Agent ในสาย A |
| MGMT-22 | drill-down บัญชีที่อยู่สายอื่น | `402402` |
| MGMT-24 | ค้น `q=abc` / กรอง `status=LOCKED` / `agent_type=B2C` | ได้เฉพาะที่ตรง |
| MGMT-26 | Company ล็อก Agent (หลาน) | สำเร็จ · Share ยังใช้งานได้ |
| MGMT-26 | เปลี่ยนสถานะของตัวเอง | `402304` |
| MGMT-27 | ล็อก Share ขณะที่ Agent ใต้ Share login อยู่ | request ถัดไปของ Agent `401302` |
| MGMT-31 | sub ของ Company เรียก `POST /shares` / `GET /agents` | `402303` |

## 7.1 ลำดับเช็ค (เส้นสร้าง)

`422` (รูปแบบ) → `402303` (sub) → `402302` (สถานะผู้สร้าง) → `402301` (สร้างประเภทนี้ไม่ได้) → `402401` (username ซ้ำ) →
`402305` / `402307` / `402308` / `402309` (ค่า PT ฯลฯ)

## 8. Contract changes (แจ้ง frontend)

- เส้นใหม่ทั้งหมดตามหัวข้อ 5
- ค่า % ทุกตัวส่ง/รับเป็นจำนวนเต็มหน่วย bp (95.50% = `9550`) — frontend แปลงเองตอนแสดง

## 9. การตัดสินใจ (2026-10-05)

1. role เดิม 3 ค่า + `agent_type` (ไม่แตก role ตามประเภท)
2. Agent ซ้อนได้ไม่จำกัดชั้น (System Overview)
3. แก้ PT ฯลฯ ได้เฉพาะผู้สร้าง · เปลี่ยนสถานะได้ทุกชั้นบนในสาย
4. sub ทำรายการในช่วงนี้ไม่ได้ (สิทธิ์ของ sub กำหนดตอนสร้าง sub — ช่วง 3)
5. **Error code** (module `agent_management` = `bb=02`) — business error ตอบ HTTP 200

| Code | HTTP | ความหมาย |
|---|---|---|
| 402301 | 200 | สร้างบัญชีประเภทนี้ไม่ได้ |
| 402302 | 200 | บัญชีของคุณถูกระงับ ทำรายการไม่ได้ |
| 402303 | 200 | บัญชีย่อยยังทำรายการนี้ไม่ได้ |
| 402304 | 200 | ไม่มีสิทธิ์แก้ไขบัญชีนี้ |
| 402305 | 200 | PT เกินกว่าที่ได้รับ |
| 402306 | 200 | PT ต่ำกว่าที่บัญชีนี้ปล่อยให้ชั้นล่างไปแล้ว |
| 402307 | 200 | Company Seamless Master ต้องปล่อย PT ทั้งหมด |
| 402308 | 200 | ตั้ง Force ให้ Share Master ไม่ได้ |
| 402309 | 200 | Force / Remain เกิน PT ที่ปล่อย |
| 402401 | 200 | username นี้ถูกใช้แล้ว |
| 402402 | 200 | ไม่พบบัญชีในสายของคุณ |
