# Agent_Management — ตอบตรวจงานรอบที่ 1 (V1–V7) · 2026-10-10

จาก boiledegg · review: [docs/reviews/agent-management.md](../reviews/agent-management.md) หัวข้อ "ตรวจงานรอบที่ 1"
branch `boiledegg/bo/agent_management` · merge เข้า `dev` แล้ว (`4e46f5d`, `f961925`) · deploy dev tag `dev-202610091753`

## 1. ถึง lead (zerph)

### ผลแต่ละข้อ

| ข้อ | ผล |
|---|---|
| V1 | `agents/status/update` ส่งสถานะเดิม = `200` ไม่เขียน log + test · ใช้กับ `subaccounts/status/update` ด้วย (ตัดสินฝั่งเรา) · ฝั่ง Member maofoy ทำ |
| V2 | migration ปรับ Share Master ตาม CSM: Member ที่คำนวณ remain ไม่ได้ บันทึกลงตาราง `migration_member_remain_skipped` · บน dev ไม่มีแถวแบบนี้ |
| V3 | เจอ Agent ใต้ Share Reseller / Share Master บน dev 2 บัญชี (`mockagentreseller`, `mockagentmaster` · ข้อมูล mock) · ลบแล้วพร้อม Member ใต้มัน (`mockmemreseller`, `mockmemmaster`) และ sub · เก็บ `account_change_logs` ไว้ · ไม่มีเงิน / ledger |
| V4 | sub ใช้เบอร์ซ้ำได้ · ใช้รูปแบบ `phone_country_code` + `phone` · มี test |
| V5 | เพิ่ม test: Share Master สร้าง Agent ได้ `402301` · migration เบอร์ · migration ปรับ Share Master ตาม CSM · ยิงพร้อมกัน · `request_id` / `csm_id` ใน log |
| V6 | log `SYNC_FROM_CSM` มี `csm_id` ใน `new_value` (ทั้ง PT และเกม) · `request_id` เดียวกับตอนแก้ CSM · มี test |
| V7 | เพิ่มคอลัมน์ `kept_agent_id` ใน `migration_phone_cleared` · RDS dev ยังไม่เคยรันไฟล์เดิม เลยไม่ต้อง reset · แจ้งทีมให้ reset DB ในเครื่องแล้ว |

### บั๊กที่ test ยิงพร้อมกันเจอ (แก้แล้ว)

- อาการ: Superadmin แก้ CSM พร้อมกับที่ CSM สร้าง Share Master → deadlock ได้ `500`
- สาเหตุ: ตอนแก้บัญชี lock แถว `user_agents` ด้วย `FOR UPDATE` ซึ่งขวาง lock ที่ Postgres ใช้เช็ค FK ตอนสร้างบัญชีลูก
- แก้: เปลี่ยนเป็น `FOR NO KEY UPDATE` (`LockUserAgentRowRepository`) · รัน test ซ้ำ 10 รอบผ่านทุกรอบ

### ผล test

- lint 0 issues · structure OK · unit + integration ผ่านทุก package
- test ยิงพร้อมกันกับ test migration 2 ตัว รันซ้ำ 10 รอบ ผ่านทั้ง 30 ครั้ง

### Deploy dev

- migrate RDS `20261009150000` / `20261009160000` / `20261009170000` ผ่าน
- เบอร์ agent 5 เบอร์แปลงเป็น `66` ครบ ไม่มีเบอร์ถูกล้าง · ไม่มีแถว Member ที่ migration ข้าม · ไม่เหลือสิทธิ์ `pt` ของ sub
- tag `dev-202610091753`

### เรื่องอื่นในรอบนี้

- `subaccounts/list` ใช้ `keyword` 4–32 ตัวแทน `q` ให้เหมือน `downlines/list`
- เจอ test ของ `downlines/list` ที่ยังส่ง `q` (ค้างจาก PR 2) · แก้แล้ว

### ส่วนที่ maofoy ทำ (งาน Member)

เบอร์ 2 field + SQL เบอร์ Member ใน `downlines/list` · ส่งสถานะเดิมไม่เขียน log · ลบ `CanSeePT` · `amounts_minor` → `amounts` · path `members/*`

## 2. ถึง maofoy

### ที่เปลี่ยนแล้วและกระทบโค้ดฝั่งคุณ

1. RDS dev รัน migration แล้ว: `user_agents` / `subaccounts` มี `phone_country_code` + `phone` (ไม่มี 0 นำหน้า) · มีตาราง `migration_phone_cleared` (`table_name` · `row_id` · `old_phone` · `reason` · `kept_agent_id`) ใช้กับ `user_members` ได้
2. `LockUserMemberGameSettingsRepository` ที่เดิม lock ทีละ agent เปลี่ยนเป็น `LockUserMemberGameSettingsByAgentsRepository(db, agentIDs, codes)` แล้ว
3. CSM → Share Master: ตอน Superadmin แก้ CSM ระบบคำนวณ `remain_quota` ของ Member ใต้ Share Master ใหม่ให้ใน tx เดียว (MA2)
4. RDS dev ลบ `mockmemreseller` / `mockmemmaster` แล้ว (lead V3)

### งานที่ยังค้างฝั่งคุณ (จาก review)

1. เบอร์ Member 2 field + migration แบบ B1 · unique `(parent, phone_country_code, phone)` (MQ1) · ฝาก**ใช้ timestamp ใหม่ ไม่แก้ไฟล์ migration ของผม**
2. SQL ของ `downlines/list` ตอนนี้ส่ง `phone_country_code` ของแถว Member เป็น `NULL` → `""` (`agent_management_repository.go` ส่วน member rows) · เปลี่ยนเป็นคอลัมน์จริงตอนคุณเพิ่ม column
3. `members/status/update` ส่งสถานะเดิม = `200` ไม่เขียน log (MQ5)
4. ลบ `CanSeePT` ที่เรียกอยู่ · ผมจะลบ shim ทีหลัง
5. `amounts_minor` → `amounts` ใน `create_member_service.go`
6. path `members/detail/get` · `detail/update` · `status/update` (N1)

### ฝากระวัง

`LockUserMemberRowRepository` ยังใช้ `FOR UPDATE` · ฝั่ง agent ผมเจอ deadlock เพราะ `FOR UPDATE` ขวาง lock ที่ Postgres ใช้เช็ค FK
ตอน insert ตารางลูก (ตอนสร้าง Member / settings) · แก้เป็น `FOR NO KEY UPDATE` แล้วหาย · ถ้า Member มีตารางลูกที่ insert
ระหว่างที่แถวถูก lock ควรเปลี่ยนเหมือนกัน

### ใช้ร่วมกัน

`UpdateInfoRequest` / `CheckPhoneFree` ใน DTO / service ของ agent_management · ตอนนี้ Member ยังใช้เบอร์แบบเดิม (ฟิลด์เดียว)
เปลี่ยนแล้วบอกผม จะได้ลบของเก่าออก

## 3. แจ้งทีม — reset DB ในเครื่อง (lead V7)

ใครที่ pull `dev` ตั้งแต่ commit `7bd675c` แล้วรัน migrate ใน DB เครื่องตัวเอง: ไฟล์
`20261009160000_agent_management_phone_country_code.sql` และ `20261009170000_agent_management_share_master_follow_csm.sql`
ถูกแก้ (lead อนุญาต · ยังไม่มี prod) · ให้รัน `go run ./scripts/migrate down` 2 ครั้ง แล้ว `go run ./scripts/migrate up`
(หรือ reset DB ทั้งก้อน) · RDS dev รันไฟล์ใหม่แล้ว ไม่ต้องทำอะไร
