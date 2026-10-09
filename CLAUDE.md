# CLAUDE.md — mini-game-api

Go/Fiber backend ของระบบ mini-game (agent hierarchy, wallet/bet, PT commission, เอนจิ้นเกม, provider webhook)
เขียนใหม่จาก `askmelotto_minigame-api` โดยใช้โครงสร้างแบบ `askmelotto-api` ที่ทีมคุ้นเคย เป้าหมายคือ
**ทุกคนในทีมไล่โค้ด แก้ และเพิ่ม feature ได้** ไม่ใช่แค่คนเดียว

## Meta rules

1. ไฟล์นี้ต้องยาวไม่เกิน 300 บรรทัด และเก็บเฉพาะ index กับกฎบังคับ รายละเอียดกับตัวอย่างไปอยู่ใน `docs/`
2. การแก้กฎในไฟล์นี้เท่ากับเปลี่ยนข้อตกลงของทั้งทีม ต้องผ่าน review ก่อน merge

## ⛔ กฎเรื่องโปรเจกต์เก่า

1. **ห้ามเชื่อเอกสารใดๆ ของ `askmelotto_minigame-api`** ได้แก่ `doc/`, `CLAUDE.md`, `.claude/` (skill/agent), review note,
   change notice, comment ที่อธิบาย business และ test ของโปรเจกต์นั้น เพราะเขียนตาม business ที่เข้าใจผิด
2. โค้ดเก่าอ่านได้เพื่อสรุปว่า "ตอนนี้ระบบทำอะไร" เท่านั้น ไม่ใช่ว่า "ควรทำอะไร"
3. business rule ทุกข้อต้องมาจาก spec ที่ lead อนุมัติใน `docs/modules/{module}.md` ถ้ายังไม่มี spec ห้าม implement
4. ห้ามคัดลอกไฟล์ (โค้ด/test/doc) จากโปรเจกต์เก่ามาทั้งไฟล์ ให้เขียนใหม่ตามแม่แบบ

## Stack

Go 1.25 · Fiber v2 · GORM + PostgreSQL · go-redis v9 · zap · goose (migration)
go module ชื่อ `app` (import เป็น `app/app/...`, `app/pkg/...`, `app/platform/...`)

## โครงสร้าง (สรุป — รายละเอียดและแม่แบบโค้ดดู `docs/ARCHITECTURE.md`)

```
app/internals/{backoffice|frontend|game}/{controllers,dto,middleware}/{module}/ + routes.go
app/externals/{partner}/{controllers,dto,middleware}/ + routes.go
app/service/{module}/     business logic + transaction
app/core/{topic}/         pure business rule + table test
app/repository/{postgres|redis}/
app/models/
pkg/{apperr,response,configs,middleware,routes,utils}
platform/{database,logger}
database/migrations/      goose .sql
```

เริ่มอ่านโค้ดจาก `application.go` → `pkg/routes/routes.go` → `app/internals/{context}/routes.go`

## กฎบังคับ

### Layer

1. flow เป็น `controller → service → repository → models` ห้ามข้าม layer (controller ห้ามเรียก repository ตรง)
2. ชื่อ module ต้องเหมือนกันทุกตัวอักษรใน `controllers/{x}`, `dto/{x}` และ `app/service/{x}`
3. controller มีหน้าที่แค่ parse → เรียก service → `response.OK` / `response.Error` ห้ามมี business logic
4. service ห้าม import fiber หรือ package ใน `app/internals` / `app/externals`
5. repository
   - รับ `db *gorm.DB` เป็น param แรกเสมอ
   - ห้ามอ้าง `database.DBConn`
   - ห้ามเปิด Transaction
   - ห้ามมี business rule
6. `app/core` เป็น pure function เท่านั้น ห้ามแตะ DB/Redis/fiber/config/logger และต้องมี table test
7. `app/models` มีได้แค่ struct, `TableName()` และ enum constant ห้ามมี logic และห้ามมี global state
8. ห้ามใช้ interface หรือ DI เพื่อ "เผื่อ mock" ใช้ package function ทั้งหมด ยกเว้นกรณีมีหลาย implementation จริง
   ต้องคุยกันก่อน

### เงินและ transaction

9. เงินและเปอร์เซ็นต์เป็นทศนิยม `float64` (Go) / `DOUBLE PRECISION` (DB) — ทีมตกลง 2026-10-09 (แทนกฎ int64 หน่วยย่อยเดิม)
   - request รับทศนิยมไม่เกิน 4 ตำแหน่ง · ปัดเป็น 4 ตำแหน่ง (`utils.Round4`) ทุกครั้งก่อนบันทึกและหลังบวก / ลบ
   - เทียบยอด / เพดานด้วยค่าที่ปัด 4 ตำแหน่งแล้วเท่านั้น (ห้ามเทียบค่าที่ยังมีเศษ float เช่น 0.1 + 0.2)
   - API ส่งค่าที่เก็บไว้ตามจริง ไม่ปัดเพิ่ม · หน้าบ้านปัดเป็น 3 ตำแหน่งตอนแสดงเอง
10. การเขียนเงินทุกครั้งต้อง
    - อยู่ใน transaction ที่ service เปิด
    - lock row ก่อนอ่านยอด (`clause.Locking{Strength: "UPDATE"}`)
    - lock หลาย row ให้เรียงลำดับ id เสมอ เพื่อกัน deadlock
11. ทุกการเปลี่ยนยอดเงินต้องมีรายการ ledger ใน transaction เดียวกัน ห้ามแก้ยอดโดยไม่มี ledger
12. endpoint ที่ย้ายเงินต้อง idempotent ด้วย request id จาก client และต้องมี unique constraint ใน DB รองรับ
13. RNG ของเกมใช้ `crypto/rand` เท่านั้น ห้ามใช้ `math/rand`
14. cache ห้ามเป็น source of truth ของยอดเงิน

### Error และ response

15. error ที่ client เห็นต้องประกาศใน `pkg/apperr/{module}_errors.go` และจองช่วงใน `docs/ERROR_CODES.md` ก่อน
16. แยก error ด้วย `errors.Is(err, apperr.ErrX)` เท่านั้น **ห้ามเทียบข้อความ error** (`strings.Contains(err.Error(), ...)`)
17. service คืน `(result, error)` ห้ามคืน code เป็นค่าแยก
18. ทุก response ต้องผ่าน `pkg/response` ห้ามเรียก `c.JSON` เองใน controller
19. list ต้องตอบด้วย `response.Page` · page / limit มาจาก query (`utils.ParsePage`) หรือจาก body ของเส้น `POST` (`utils.NewPage`)
    - request body/query ต้องผ่าน `utils.ParseBody` / `utils.ParseBodyNoNull` / `utils.ParseQuery` และ DTO ต้องมี method `Validate()`
      ที่เขียนเป็น `if` ธรรมดา **ห้ามใช้ struct tag ของ validator**
    - ข้อความ error ต้องบอกว่าผิดที่ field ไหน (`apperr.ErrValidation.WithMessage(th, en)`)

### DB / query

20. schema มาจาก goose migration เท่านั้น (`make migration name=...`) ห้าม AutoMigrate
    - migration ที่ merge แล้วห้ามแก้ ต้องสร้างไฟล์ใหม่
21. query ต้องเป็น parameterized เสมอ ห้ามต่อ string ด้วยค่าจาก user
22. query ของหลังบ้านต้อง scope ด้วยเจ้าของข้อมูล (agent/สายงาน) ทุกครั้ง
23. Select เฉพาะคอลัมน์ที่ใช้ ห้าม N+1 และการเขียนหลายแถวใช้ batch
24. คอลัมน์ที่ใช้ใน WHERE/JOIN/ORDER ของ query หลักต้องมี index ใน migration เดียวกับที่สร้างตาราง

### Config / log

25. อ่าน env ที่ `pkg/configs` เท่านั้น
    - เพิ่ม key ต้องแก้ `config.go` และ `.env.example` ใน PR เดียวกัน
26. log ด้วย `logger.Ctx(ctx)` (มี request_id อัตโนมัติ) ห้ามใช้ `fmt.Print*`
    - ห้าม log password, token, key หรือข้อมูลบัตร
27. goroutine ต้องมีขอบเขต (worker pool/semaphore) และรับ ctx เสมอ

### Security

28. endpoint ใหม่ต้องระบุสิทธิ์ไว้บรรทัดเดียวกับ route ใน `routes.go` ห้ามซ่อนการเช็คสิทธิ์ไว้ใน service โดยไม่มีใน route
29. `app/externals/*` เป็น trust boundary แยก ห้ามใช้ middleware auth ของ internals

## Workflow

- ก่อนเปิด PR: `make check` (structure + build + vet + unit test) ต้องผ่าน
- PR หนึ่งอันต่อ module หรือต่อ endpoint กลุ่มเดียวกัน และต้องอัปเดต `docs/MIGRATION_STATUS.md`
- ขั้นตอนการ port ต่อ module (Extract → Spec → Implement → Test → Review) ดู `docs/MIGRATION_STATUS.md`
- branch: `feature/{module}-{what}` → PR เข้า `dev`

## Local dev

```bash
cp .env.example .env
make dev-up          # postgres + redis
make run             # MIGRATE_ON_START=true จะรัน migration ให้
make test            # unit
make test-integration   # ต้องเปิด test env ก่อน (make test-env-up หรือ scripts/testenv.ps1 up) — docs/TESTING.md
```

## Index เอกสาร

| เอกสาร | เนื้อหา |
|---|---|
| `docs/ARCHITECTURE.md` | โครงสร้าง, กฎตั้งชื่อ, แม่แบบโค้ดทุก layer, response contract, test, ส่วนที่ต่างจาก askmelotto-api |
| `docs/MIGRATION_STATUS.md` | ขั้นตอนการ port และสถานะของแต่ละ module |
| `docs/ERROR_CODES.md` | error code ทั้งหมดและช่วงที่จองไว้ |
| `docs/TESTING.md` | test env (Docker / Windows), integration test, load test (k6) |
| `docs/DEPLOYMENT.md` | ออก tag → image GHCR, สิ่งที่ k8s ต้องเตรียม (env, probe, migration job) |
| `docs/modules/_TEMPLATE.md` | แม่แบบ spec ของ module |
| `docs/modules/{module}.md` | spec ที่อนุมัติแล้ว (source of truth ของ business) |
