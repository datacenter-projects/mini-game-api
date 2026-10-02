# Architecture — mini-game-api

เอกสารนี้คือ "แม่แบบ" ของโปรเจกต์ ใครจะเพิ่ม endpoint ให้ลอกตามนี้ ถ้าจะทำต่างจากนี้ต้องคุยในทีมและแก้เอกสารนี้ก่อน

> โครงสร้างนี้ตั้งใจให้เหมือน **askmelotto-api (โครงสร้างใหม่)** ที่ทีมคุ้นเคย โดยปรับ 4 จุดเรื่องความปลอดภัยของเงิน
> (ดู [ส่วนที่ต่างจาก askmelotto-api](#ส่วนที่ต่างจาก-askmelotto-api))

## 1. ภาพรวม

```
HTTP request
  → pkg/middleware (global: cors, requestid, recover, logger)
  → app/internals/{context}/routes.go  (group + middleware ต่อ route)
  → app/internals/{context}/controllers/{module}/xxx_controller.go
  → app/service/{module}/xxx_service.go        ← business logic + transaction
       ├→ app/core/{topic}                      ← pure rule (คำนวณ ไม่มี side effect)
       └→ app/repository/{postgres|redis}       ← query เท่านั้น
            → app/models                        ← GORM struct
```

**ไล่โค้ดได้ด้วย Go to Definition ตลอดทาง** — ไม่มี interface, ไม่มี DI container, ไม่มี wiring file

## 2. โฟลเดอร์

```
application.go                         # boot: config → logger → DB → migrate → redis → fiber → routes
app/
├── internals/                         # API ที่ client ของเราเองเรียก
│   ├── backoffice/                    # หลังบ้าน agent/subaccount   → /api/v1/bo/{pb,pr}
│   │   ├── controllers/{module}/
│   │   ├── dto/{module}/
│   │   ├── middleware/
│   │   └── routes.go
│   ├── frontend/                      # ผู้เล่น                     → /api/v1/player
│   └── game/                          # เอนจิ้นเกม                  → /api/v1/{scratch,cointoss,rockpaperscissors}
├── externals/                         # API ที่ partner ภายนอกเรียก (trust boundary แยก)
│   └── provider/                      #                             → /api/v1/provider
├── service/{module}/                  # business logic — ใช้ร่วมทุก context
├── core/{topic}/                      # pure business rule + table test
├── repository/postgres/               # query PostgreSQL
├── repository/redis/                  # cache / session / lock
├── models/                            # GORM struct
└── health/                            # /health/live, /health/ready
pkg/
├── apperr/                            # error + business code
├── response/                          # envelope {code,msg,data} + pagination
├── configs/                           # Config struct (ที่เดียวที่อ่าน env)
├── middleware/                        # global middleware
├── routes/                            # mount ทุก context
└── utils/                             # ParseBody, ParsePage, ...
platform/
├── database/                          # DBConn (postgres), DBRedis, Migrate
└── logger/                            # zap + logger.Ctx(ctx)
database/migrations/                   # goose .sql
scripts/                               # migrate, check_structure.sh, งาน ops
docs/                                  # เอกสารที่อนุมัติแล้วเท่านั้น
```

### กฎตั้งชื่อ

| สิ่ง | รูปแบบ | ตัวอย่าง |
|---|---|---|
| โฟลเดอร์ module | snake_case, **ชื่อเดียวกันทุก layer** | `controllers/agent_wallet`, `dto/agent_wallet`, `service/agent_wallet` |
| package | ชื่อโฟลเดอร์ตัด `_` | `package agentwallet` |
| ไฟล์ | `{operation}_{layer}.go` | `adjust_balance_controller.go`, `adjust_balance_service.go`, `adjust_balance_dto.go` |
| repository | `{table}_repository.go` | `agent_balance_repository.go` |
| model | `{table}_models.go` | `agent_balance_models.go` |
| ฟังก์ชัน | ลงท้ายด้วยชื่อ layer | `AdjustBalanceController`, `AdjustBalanceService`, `LockAgentBalanceRepository` |
| import alias | `{module}Service`, `{module}Dto` | `agentWalletService "app/app/service/agent_wallet"` |

`scripts/check_structure.sh` บังคับว่า `controllers/{x}` ต้องมี `dto/{x}` และ `app/service/{x}` คู่กันเสมอ

## 3. แม่แบบต่อ layer

ตัวอย่างใช้ module สมมติ `note` (ไม่ใช่ business จริง) เพื่อให้เห็น pattern ล้วนๆ

### 3.1 DTO — `app/internals/backoffice/dto/note/create_note_dto.go`

```go
package note

type CreateNoteRequest struct {
	Title  string `json:"title"  validate:"required,max=100"`
	Amount int64  `json:"amount" validate:"gte=0"` // เงิน = int64 หน่วยย่อยที่สุด
}

type NoteResponse struct {
	ID     uint   `json:"id"`
	Title  string `json:"title"`
	Amount int64  `json:"amount"`
}
```

### 3.2 Controller — `app/internals/backoffice/controllers/note/create_note_controller.go`

หน้าที่มีแค่: อ่าน request → เรียก service → ตอบ ห้ามมี business logic

```go
package note

import (
	noteDto "app/app/internals/backoffice/dto/note"
	noteService "app/app/service/note"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

func CreateNoteController(c *fiber.Ctx) error {
	var req noteDto.CreateNoteRequest
	if err := utils.ParseBody(c, &req); err != nil {
		return response.Error(c, err)
	}
	agentID := c.Locals("agent_id").(uint) // middleware auth เป็นคน set

	res, err := noteService.CreateNoteService(c.UserContext(), agentID, req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}
```

### 3.3 Service — `app/service/note/create_note_service.go`

เจ้าของ business logic และ transaction

```go
package note

func CreateNoteService(ctx context.Context, agentID uint, req noteDto.CreateNoteRequest) (noteDto.NoteResponse, error) {
	var res noteDto.NoteResponse

	err := database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		agent, err := postgres.LockAgentRepository(tx, agentID)       // SELECT ... FOR UPDATE
		if err != nil {
			return err
		}
		if !notecore.CanCreate(agent.Status) {                          // rule อยู่ใน core
			return apperr.ErrNoteNotAllowed
		}
		n := models.Note{AgentID: agentID, Title: req.Title, Amount: req.Amount}
		if err := postgres.CreateNoteRepository(tx, &n); err != nil {
			return err
		}
		res = noteDto.NoteResponse{ID: n.ID, Title: n.Title, Amount: n.Amount}
		return nil
	})
	if err != nil {
		logger.Ctx(ctx).Warnw("create note failed", "agent_id", agentID, "error", err)
		return res, err
	}
	return res, nil
}
```

- อ่านอย่างเดียว ไม่ต้องมี tx: `postgres.GetNoteRepository(database.DBConn.WithContext(ctx), id)`
- service ของ module อื่นเรียกกันได้ ถ้าต้องอยู่ใน tx เดียวกัน ให้ทำฟังก์ชันที่รับ `tx *gorm.DB` เพิ่ม (เช่น `DebitWalletTx(tx, ...)`)

### 3.4 Repository — `app/repository/postgres/note_repository.go`

```go
package postgres

func CreateNoteRepository(db *gorm.DB, n *models.Note) error {
	return db.Create(n).Error
}

func GetNoteRepository(db *gorm.DB, agentID, id uint) (models.Note, error) {
	var n models.Note
	err := db.Select("id", "agent_id", "title", "amount").
		Where("id = ? AND agent_id = ?", id, agentID). // scope ด้วยเจ้าของเสมอ
		First(&n).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return n, apperr.ErrNotFound
	}
	return n, err
}

func LockAgentRepository(db *gorm.DB, id uint) (models.Agent, error) {
	var a models.Agent
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&a, id).Error
	return a, err
}
```

### 3.5 Core — `app/core/note/rule.go` + `rule_test.go`

```go
func CanCreate(status string) bool { return status == "ACTIVE" }

func TestCanCreate(t *testing.T) {
	tests := []struct{ status string; want bool }{
		{"ACTIVE", true},
		{"SUSPENDED", false},
	}
	for _, tt := range tests { /* ... */ }
}
```

### 3.6 Error — `pkg/apperr/note_errors.go`

```go
var ErrNoteNotAllowed = New(230101, "ไม่สามารถสร้างโน้ตได้", "Note creation not allowed")
```

ช่วง code ของแต่ละ module ต้องจองใน `docs/ERROR_CODES.md` ก่อนใช้

### 3.7 Route — `app/internals/backoffice/routes.go`

```go
pr.Post("/notes", mw.Perm(perm.NoteEdit), noteController.CreateNoteController)
```

middleware ทุกตัวที่ endpoint ต้องผ่านให้อยู่บรรทัดเดียวกับ route เปิดไฟล์นี้ไฟล์เดียวต้องรู้สิทธิ์ครบ

### 3.8 Migration — `make migration name=note_create_table`

```sql
-- +goose Up
CREATE TABLE notes (
    id         BIGSERIAL PRIMARY KEY,
    agent_id   BIGINT       NOT NULL REFERENCES agents(id),
    title      VARCHAR(100) NOT NULL,
    amount     BIGINT       NOT NULL DEFAULT 0 CHECK (amount >= 0),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_notes_agent_id ON notes(agent_id);

-- +goose Down
DROP TABLE notes;
```

## 4. Response contract

ทุก endpoint ตอบ `{"code": int, "msg": string, "data": any}`

- `code = 200` คือสำเร็จ อย่างอื่นดู `docs/ERROR_CODES.md`
- `msg` เลือกภาษาจาก header `X-Lang` (หรือ `Accept-Language`) — `en` ได้อังกฤษ อื่นๆ ได้ไทย
- `data` มีเสมอ (error = `null`)
- business error ตอบ HTTP 200 (client อ่าน `code`) ยกเว้น error ระดับ infra (429/500/503) และ route ที่ไม่มีอยู่ (404)
- error ที่ไม่ได้ประกาศใน apperr → log เต็ม แล้วตอบ `500` โดยไม่หลุดรายละเอียดภายใน
- list ใช้ `response.Page(...)` เท่านั้น:
  `{current_page, total_page, total_count, limit, has_next, has_prev, data: []}` — `limit` default 20 สูงสุด 100

## 5. Test

| ชนิด | ที่อยู่ | แตะ DB | รัน |
|---|---|---|---|
| unit | `app/core/**`, `pkg/**` | ไม่ | `make test` |
| integration | `app/service/**`, `app/repository/**` ใส่ `//go:build integration` | Postgres/Redis จริงจาก docker-compose | `make dev-up && make test-integration` |

- ไม่ใช้ mock — logic ที่อยากเทสแบบไม่มี DB ให้ย้ายไป `app/core`
- test case ของ business ต้องมาจาก spec ที่อนุมัติแล้ว (`docs/modules/{module}.md`) ไม่ใช่จากพฤติกรรมของโค้ดเก่า

## 6. ส่วนที่ต่างจาก askmelotto-api

| # | askmelotto-api | mini-game-api | เหตุผล |
|---|---|---|---|
| 1 | `internals/controllers/{context}/{module}` | `internals/{context}/controllers/{module}` | เปิดโฟลเดอร์ context เดียวเห็นครบ (controller, dto, middleware, routes) |
| 2 | repository ใช้ `database.DBConn` และเปิด tx เอง | repository รับ `db *gorm.DB`, service เปิด tx | เงินต้อง lock + เขียนหลายตารางใน tx เดียว |
| 3 | service คืน `(data, code, error)` | service คืน `(data, error)` โดย error มี code (`apperr`) | ไม่ต้องส่ง code คู่ขนาน, ไม่ต้องเทียบข้อความ error |
| 4 | `os.Getenv` กระจาย | `configs.Cfg` โหลดครั้งเดียว | key ขาด = boot ไม่ขึ้น |
| 5 | request_id ส่งเป็น param แรก | อยู่ใน `ctx` แล้วใช้ `logger.Ctx(ctx)` | ctx ต้องส่งอยู่แล้วเพื่อ timeout/cancel ของ DB |
| 6 | calc อยู่ปนใน service/models | `app/core` เป็น pure function | เทสกฎเงินได้โดยไม่ต้องมี DB |
| 7 | AutoMigrate | goose `.sql` | schema review ได้, มี version, up/down |
