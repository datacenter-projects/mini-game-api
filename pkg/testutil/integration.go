//go:build integration

// Package testutil คือ helper ของ integration test (ต้องมี Postgres + Redis จริง — make dev-up)
//
//	func TestX(t *testing.T) {
//		app := testutil.Setup(t)   // ต่อ DB/Redis, migrate, ล้างข้อมูลหลังจบ test
//		...
//	}
package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"app/pkg/configs"
	"app/pkg/middleware"
	"app/pkg/routes"
	"app/platform/database"
	"app/platform/logger"

	_ "github.com/joho/godotenv/autoload"

	"github.com/gofiber/fiber/v2"
)

var once sync.Once

// ตารางที่ต้องล้างระหว่าง test — เพิ่มเมื่อมีตารางใหม่
var tables = []string{"auth_audit_logs", "subaccounts", "user_agents"}

// Setup ต่อ DB/Redis ตาม env (ครั้งเดียวต่อ package), รัน migration, คืน fiber app ที่ประกอบ route จริง
// และล้างข้อมูลทั้งหมดหลัง test จบ — test ที่ใช้ helper นี้ห้ามรัน t.Parallel()
func Setup(t *testing.T) *fiber.App {
	t.Helper()
	once.Do(func() {
		if err := configs.Load(); err != nil {
			t.Fatalf("config: %v", err)
		}
		logger.InitLogger(true)
		ctx := context.Background()
		if err := database.PostgreSQLConnection(configs.Cfg.DB); err != nil {
			t.Fatalf("postgres: %v", err)
		}
		if err := database.Migrate(ctx, "up"); err != nil {
			t.Fatalf("migrate: %v", err)
		}
		if err := database.RedisConnection(ctx, configs.Cfg.Redis); err != nil {
			t.Fatalf("redis: %v", err)
		}
	})

	original := *configs.Cfg
	reset(t)
	t.Cleanup(func() {
		reset(t)
		*configs.Cfg = original // test แก้ config ได้ แล้วคืนค่าให้ test ถัดไป
	})

	return NewApp(nil)
}

// NewApp ประกอบ app แบบเดียวกับ application.go — extra (ถ้ามี) ถูกเรียกก่อน SetupRoutes
// ใช้เพิ่ม route ทดสอบ เช่น a.Get("/x", mw.Authenticated(), handler) — route ที่เพิ่มตรงนี้ match ก่อน
// route จริง จึงต้องใส่ middleware ที่ต้องการทดสอบเองใน route นั้น
func NewApp(extra func(a *fiber.App)) *fiber.App {
	app := fiber.New(configs.FiberConfig())
	middleware.FiberMiddleware(app)
	if extra != nil {
		extra(app)
	}
	routes.SetupRoutes(app)
	return app
}

func reset(t *testing.T) {
	t.Helper()
	for _, tbl := range tables {
		if err := database.DBConn.Exec("TRUNCATE TABLE " + tbl + " RESTART IDENTITY CASCADE").Error; err != nil {
			t.Fatalf("truncate %s: %v", tbl, err)
		}
	}
	if err := database.DBRedis.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("flush redis: %v", err)
	}
}

// Response คือผลของ Call
type Response struct {
	Status int
	Header http.Header     `json:"-"`
	Code   int             `json:"code"`
	Msg    string          `json:"msg"`
	Data   json.RawMessage `json:"data"`
}

// Call ยิง request เข้า app — body เป็น struct/map (nil = ไม่มี body), token ว่าง = ไม่ส่ง Authorization
func Call(t *testing.T, app *fiber.App, method, path string, body any, token string) Response {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	out := Response{Status: resp.StatusCode, Header: resp.Header}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s %s: invalid json %q", method, path, raw)
	}
	return out
}
