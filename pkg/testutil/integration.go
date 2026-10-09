//go:build integration

// Package testutil คือ helper ของ integration test (ต้องมี Postgres + Redis จริง — docs/TESTING.md)
//
//	func TestX(t *testing.T) {
//		app := testutil.Setup(t)   // ต่อ DB/Redis, migrate, ล้างข้อมูลหลังจบ test
//		...
//	}
//
// env มาจาก .env.test ที่ root ของ repo — ถ้า environment มี DB_HOST อยู่แล้ว (เช่น CI) ใช้ environment อย่างเดียว
package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"app/pkg/configs"
	"app/pkg/middleware"
	"app/pkg/routes"
	"app/pkg/utils"
	"app/platform/database"
	"app/platform/logger"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

var (
	once     sync.Once
	setupErr error
)

// ตารางที่ต้องล้างระหว่าง test — เพิ่มเมื่อมีตารางใหม่
var tables = []string{"account_change_logs", "auth_audit_logs", "balance_ledger", "create_requests",
	"user_member_game_settings", "user_members", "agent_balances", "agent_game_settings", "agent_currencies",
	"subaccounts", "user_agents"}

// env ที่ห้ามรัน integration test เด็ดขาด — test ล้างทุกตารางและ FLUSHDB Redis
var forbiddenEnvs = map[string]bool{"dev": true, "uat": true, "prod": true}

// Setup ต่อ DB/Redis ตาม env (ครั้งเดียวต่อ package), รัน migration, คืน fiber app ที่ประกอบ route จริง
// และล้างข้อมูลทั้งหมดหลัง test จบ — test ที่ใช้ helper นี้ห้ามรัน t.Parallel()
func Setup(t *testing.T) *fiber.App {
	t.Helper()
	once.Do(func() { setupErr = connect() })
	if setupErr != nil { // ทุก test ได้ error เดิม ไม่ panic ต่อด้วย nil connection
		t.Fatalf("testutil: %v", setupErr)
	}

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

func connect() error {
	if err := loadTestEnv(); err != nil {
		return err
	}
	if err := configs.Load(); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if forbiddenEnvs[configs.Cfg.AppEnv] {
		return fmt.Errorf("APP_ENV=%s: integration test ล้างข้อมูลทั้งหมด ห้ามรันกับ env นี้", configs.Cfg.AppEnv)
	}
	configs.Cfg.Auth.PasswordCost = bcrypt.MinCost // cost 12 + -race ทำ test ช้าจน timeout
	utils.SetPasswordCost(configs.Cfg.Auth.PasswordCost)
	logger.InitLogger(true)
	ctx := context.Background()
	if err := database.PostgreSQLConnection(configs.Cfg.DB); err != nil {
		return fmt.Errorf("postgres %s:%d/%s: %w", configs.Cfg.DB.Host, configs.Cfg.DB.Port, configs.Cfg.DB.Name, err)
	}
	if err := database.Migrate(ctx, "up"); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if err := database.RedisConnection(ctx, configs.Cfg.Redis); err != nil {
		return fmt.Errorf("redis %s: %w", configs.Cfg.Redis.Addr, err)
	}
	return nil
}

// loadTestEnv โหลด .env.test ที่ root ของ repo ตามกติกาของ configs.LoadTestEnvFile
// go test รันใน directory ของ package จึงต้องหา root เอง (ที่ที่มี go.mod)
func loadTestEnv() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return errors.New("หา go.mod ไม่เจอ")
		}
		dir = parent
	}
	return configs.LoadTestEnvFile(filepath.Join(dir, ".env.test"))
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
