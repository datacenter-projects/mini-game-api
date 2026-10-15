# Lead review: อัตราแลกเปลี่ยน (`currency_rate`) — zerph · 2026-10-16

- โค้ด: merge เข้า dev แล้ว `72f48d5` (Boiledegg · ยังไม่ผ่าน review ก่อน merge) · spec `docs/modules/currency_rate.md` (เขียน APPROVED 2026-10-07)
- ผลตรวจโค้ด (hub verify): ใช้แสดงผลได้ แต่**ยังไม่ปลอดภัยสำหรับใช้กับเงิน** — must-fix 3 · should-fix 6 · nit 6
- การใช้งานเปลี่ยน: Dashboard ถูกตัด → **ผู้ใช้หลักคือ module เดิมพัน** แปลงยอดชนะของ Member สกุลอื่นเป็นบาทเพื่อเช็ค `win_limit_per_day` (หน่วยบาทเสมอ — member-payment รอบ 2.1) → spec ต้องออกรอบใหม่ (business logic ของ job รีวิวต่อในรอบ 2)

## รอบ 1 — โครงสร้าง + การใช้กับเงิน

### ใช้กับเงิน (เดิม spec เขียน "แสดงผลเท่านั้น")
| # | เรื่อง | ผล |
|---|---|---|
| C1 | ไม่มีเรท / เรทเก่าเกินกำหนด ตอน Member สกุลอื่นแทง | **ปฏิเสธการแทง (fail closed)** · Member THB ไม่ต้องแปลง ไม่โดน |
| C2 | อายุเรทสูงสุด | **24 ชั่วโมง** · เกิน = error (`ErrRateStale`) |
| C3 | เก็บเรทไว้กับรายการเดิมพัน | **เก็บ** เรทที่ใช้ + ยอดชนะที่แปลงเป็นบาทแล้ว (ออกแบบใน spec bet) |
| M1 | ฟังก์ชันแปลงใช้ int64 หน่วย 1/100 ปัด 2 ตำแหน่ง | **float64 + ปัด 4 ตำแหน่ง** ตามกฎข้อ 9 (คำนวณด้วย `big.Rat`) |

### โครงสร้าง cronjob (lead อนุมัติการเปลี่ยนโครงสร้าง)
| # | เรื่อง | ผล |
|---|---|---|
| J1 | งานตามเวลาอยู่ที่ไหน | **`app/cronjob/`** — `cronjob.go` (`Start(ctx)` จุดเดียวที่ `application.go` เรียก) + `{job}_job.go` (ตารางเวลา + Redis lock → เรียก service) · ไม่มี business logic · เรียกได้แค่ service · รันใน process เดียวกับ API |
| J2 | pod ไหนรัน | **flag `IS_CRONJOB` (ค่าเริ่มต้น `false`)** เปิดเฉพาะ deployment เดียว (แบบ askmelotto-api) · ยังคง Redis lock กันซ้ำ · แทน `IS_CURRENCY_RATE_SYNC` |
| J3 | ตัวยิง HTTP ไป askme | **client ใน `platform/askme`** · service เรียกผ่าน client · test ใช้ fake server |
| T1 | ความถี่ของ job | **อยู่ในโค้ด** (ไม่เก็บในตาราง) |
| T2 | เปิด / ปิด job | **ตาราง `cronjobs`**: `code` (PK) · `enabled` · `last_started_at` · `last_finished_at` · `last_status` (`SUCCESS` / `FAILED` / `SKIPPED`) · `last_error` (ตัดความยาว) · `updated_by` · `updated_at` · ทุกรอบอ่าน `enabled` ก่อน (ปิด = ข้ามรอบ) · ไม่มีแถว = สร้างตอน boot ตามค่าเริ่มต้นในโค้ด · **ตอนนี้เปิด / ปิดด้วย SQL (ขอ lead บน env ร่วม)** · เส้น BO ให้ Superadmin ทำทีหลัง |
| T3 | ประวัติทุกรอบ | **ไม่เก็บ** (`cronjob_runs` ยังไม่ทำ) · ดูรอบล่าสุดในตาราง · รอบเก่าดูจาก log / metrics |

แก้ CLAUDE.md (โครงสร้าง `app/cronjob` + กฎ cronjob เรียกได้แค่ service) · ARCHITECTURE.md · `.env.example` ใน PR เดียวกัน

### ควรแก้ (จากผลตรวจโค้ด)
- ลบ const ของ currency_rate ที่ค้างใน `app/repository/redis/agent_auth/keys.go` (จาก merge `1b763f6`)
- `synced_at` อัปเดตแม้ไม่มีสกุลไหนได้เรทใหม่ → แก้คู่กับ C2
- รายการ 27 สกุลซ้ำกับ agent_management → ใช้ที่เดียว
- รายละเอียดอื่น: `reviews/mini-game-api/verify-currency-rate.md` (lead ส่งให้)
