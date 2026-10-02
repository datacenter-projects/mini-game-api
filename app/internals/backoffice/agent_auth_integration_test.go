//go:build integration

// API test ของ agent_auth ตามตาราง test case ใน docs/modules/agent_auth.md หัวข้อ 7
package backoffice_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	mw "app/app/internals/backoffice/middleware"
	"app/app/models"
	"app/app/repository/postgres"
	"app/pkg/configs"
	"app/pkg/response"
	"app/pkg/testutil"
	"app/pkg/utils"
	"app/platform/database"

	"github.com/gofiber/fiber/v2"
)

const (
	loginPath  = "/api/v1/bo/pb/auth/login"
	logoutPath = "/api/v1/bo/pr/auth/logout"
	mePath     = "/api/v1/bo/pr/_test/me" // route ทดสอบ middleware.Authenticated
	password   = "correct-password"
)

func setup(t *testing.T) *fiber.App {
	t.Helper()
	testutil.Setup(t)
	configs.Cfg.Auth.LoginIPLimit = 1000 // test ทุกตัวยิงจาก IP เดียวกัน — ทดสอบ AUTH-11 แยก
	return testutil.NewApp(func(a *fiber.App) {
		a.Get(mePath, mw.Authenticated(), func(c *fiber.Ctx) error {
			return response.OK(c, fiber.Map{"agent_id": mw.GetActor(c).AgentID})
		})
	})
}

func createAgent(t *testing.T, username string, status models.AgentStatus, parentID *uint) models.UserAgent {
	t.Helper()
	hash, err := utils.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	a := models.UserAgent{Username: username, PasswordHash: hash, Role: models.AgentRoleAgent, Status: status, ParentID: parentID}
	if err := postgres.CreateUserAgentRepository(database.DBConn, &a); err != nil {
		t.Fatal(err)
	}
	return a
}

func setStatus(t *testing.T, id uint, status models.AgentStatus) {
	t.Helper()
	if err := database.DBConn.Model(&models.UserAgent{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		t.Fatal(err)
	}
}

func login(t *testing.T, app *fiber.App, username, pw string) testutil.Response {
	t.Helper()
	return testutil.Call(t, app, "POST", loginPath, map[string]string{"username": username, "password": pw}, "")
}

func loginToken(t *testing.T, app *fiber.App, username string) string {
	t.Helper()
	r := login(t, app, username, password)
	if r.Code != 200 {
		t.Fatalf("login %s: code %d %s", username, r.Code, r.Msg)
	}
	var d struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(r.Data, &d)
	return d.Token
}

func expect(t *testing.T, r testutil.Response, status, code int) {
	t.Helper()
	if r.Status != status || r.Code != code {
		t.Fatalf("want HTTP %d code %d, got HTTP %d code %d (%s)", status, code, r.Status, r.Code, r.Msg)
	}
}

func TestLoginSuccess(t *testing.T) { // AUTH-01, AUTH-12
	app := setup(t)
	a := createAgent(t, "agent01", models.AgentStatusActive, nil)

	r := login(t, app, "  Agent01 ", password)
	expect(t, r, 200, 200)

	var d struct {
		Token        string    `json:"token"`
		Username     string    `json:"username"`
		Role         string    `json:"role"`
		IsSubaccount bool      `json:"is_subaccount"`
		PasscodeSet  bool      `json:"passcode_set"`
		ExpiresAt    time.Time `json:"expires_at"`
	}
	if err := json.Unmarshal(r.Data, &d); err != nil {
		t.Fatal(err)
	}
	if d.Token == "" || d.Username != "agent01" || d.Role != "AGENT" || d.IsSubaccount || d.PasscodeSet {
		t.Fatalf("unexpected data %+v", d)
	}
	if until := time.Until(d.ExpiresAt); until < 11*time.Hour || until > 12*time.Hour {
		t.Fatalf("expires_at ต้องประมาณ 12 ชม. ได้ %v", until)
	}

	expect(t, testutil.Call(t, app, "GET", mePath, nil, d.Token), 200, 200)

	var got models.UserAgent
	database.DBConn.First(&got, a.ID)
	if got.LastLoginAt == nil || got.LastLoginIP == nil {
		t.Fatal("ต้องบันทึก last_login_at / last_login_ip")
	}
}

func TestLoginRejects(t *testing.T) { // AUTH-02, AUTH-04, AUTH-05
	app := setup(t)
	grandpa := createAgent(t, "grandpa", models.AgentStatusLocked, nil)
	dad := createAgent(t, "dad", models.AgentStatusActive, &grandpa.ID)
	createAgent(t, "child", models.AgentStatusActive, &dad.ID)
	createAgent(t, "locked", models.AgentStatusLocked, nil)
	createAgent(t, "suspended", models.AgentStatusSuspended, nil)

	tests := []struct {
		name, username, pw string
		status, code       int
	}{
		{"ไม่มี username นี้", "ghost", password, 401, 401201},
		{"รหัสผิด", "dad", "wrong", 401, 401201},
		{"LOCKED + รหัสถูก", "locked", password, 403, 401301},
		{"LOCKED + รหัสผิด ไม่บอกว่าล็อก", "locked", "wrong", 401, 401201},
		{"ปู่ LOCKED", "child", password, 403, 401302},
		{"SUSPENDED login ได้", "suspended", password, 200, 200},
		{"subaccount ยังไม่เปิด", "dad@staff", password, 401, 401201},
		{"ไม่กรอกรหัส", "dad", "", 200, 422},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expect(t, login(t, app, tt.username, tt.pw), tt.status, tt.code)
		})
	}
}

func TestSingleSession(t *testing.T) { // AUTH-06
	app := setup(t)
	createAgent(t, "agent01", models.AgentStatusActive, nil)

	tokenA := loginToken(t, app, "agent01")
	tokenB := loginToken(t, app, "agent01")

	expect(t, testutil.Call(t, app, "GET", mePath, nil, tokenA), 401, 401203)
	expect(t, testutil.Call(t, app, "GET", mePath, nil, tokenB), 200, 200)
}

func TestSessionTimeouts(t *testing.T) { // AUTH-07, AUTH-08
	app := setup(t)
	createAgent(t, "agent01", models.AgentStatusActive, nil)

	t.Run("idle", func(t *testing.T) {
		configs.Cfg.Auth.SessionIdleTimeout = 2 * time.Second
		configs.Cfg.Auth.SessionAbsoluteTimeout = time.Hour
		tok := loginToken(t, app, "agent01")
		time.Sleep(1200 * time.Millisecond)
		expect(t, testutil.Call(t, app, "GET", mePath, nil, tok), 200, 200) // ต่ออายุ
		time.Sleep(1200 * time.Millisecond)
		expect(t, testutil.Call(t, app, "GET", mePath, nil, tok), 200, 200) // รวม 2.4s แต่ยังใช้งานต่อเนื่อง
		time.Sleep(2500 * time.Millisecond)
		expect(t, testutil.Call(t, app, "GET", mePath, nil, tok), 401, 401203)
	})

	t.Run("absolute", func(t *testing.T) {
		configs.Cfg.Auth.SessionIdleTimeout = 2 * time.Second
		configs.Cfg.Auth.SessionAbsoluteTimeout = 3 * time.Second
		tok := loginToken(t, app, "agent01")
		for i := 0; i < 2; i++ {
			time.Sleep(1200 * time.Millisecond)
			expect(t, testutil.Call(t, app, "GET", mePath, nil, tok), 200, 200)
		}
		time.Sleep(1200 * time.Millisecond) // รวม 3.6s เกิน absolute แม้ใช้งานต่อเนื่อง
		r := testutil.Call(t, app, "GET", mePath, nil, tok)
		if r.Status != 401 || (r.Code != 401203 && r.Code != 401202) { // token หมดอายุพร้อม absolute
			t.Fatalf("ต้องหลุดเมื่อครบ absolute ได้ HTTP %d code %d", r.Status, r.Code)
		}
	})
}

func TestLogout(t *testing.T) { // AUTH-09
	app := setup(t)
	createAgent(t, "agent01", models.AgentStatusActive, nil)
	tok := loginToken(t, app, "agent01")

	expect(t, testutil.Call(t, app, "POST", logoutPath, nil, tok), 200, 200)
	expect(t, testutil.Call(t, app, "GET", mePath, nil, tok), 401, 401203)
	expect(t, testutil.Call(t, app, "POST", logoutPath, nil, tok), 200, 200) // ซ้ำได้
	expect(t, testutil.Call(t, app, "POST", logoutPath, nil, ""), 401, 401202)
	expect(t, testutil.Call(t, app, "POST", logoutPath, nil, "garbage"), 401, 401202)

	// logout ด้วย token เก่าต้องไม่เตะ session ใหม่
	old := loginToken(t, app, "agent01")
	current := loginToken(t, app, "agent01")
	expect(t, testutil.Call(t, app, "POST", logoutPath, nil, old), 200, 200)
	expect(t, testutil.Call(t, app, "GET", mePath, nil, current), 200, 200)
}

func TestLoginBruteForce(t *testing.T) { // AUTH-10
	app := setup(t)
	createAgent(t, "agent01", models.AgentStatusActive, nil)
	createAgent(t, "agent02", models.AgentStatusActive, nil)

	t.Run("ผิด 4 ถูก 1 แล้วผิดอีก 4 ยัง login ได้", func(t *testing.T) {
		for i := 0; i < 4; i++ {
			expect(t, login(t, app, "agent02", "wrong"), 401, 401201)
		}
		expect(t, login(t, app, "agent02", password), 200, 200)
		for i := 0; i < 4; i++ {
			expect(t, login(t, app, "agent02", "wrong"), 401, 401201)
		}
		expect(t, login(t, app, "agent02", password), 200, 200)
	})

	t.Run("ผิด 5 ครั้งแล้วรหัสถูกก็เข้าไม่ได้", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			expect(t, login(t, app, "agent01", "wrong"), 401, 401201)
		}
		expect(t, login(t, app, "agent01", password), 403, 401303)
		expect(t, login(t, app, "agent02", password), 200, 200) // username อื่นไม่กระทบ
	})
}

func TestLoginIPRateLimit(t *testing.T) { // AUTH-11
	app := setup(t)
	configs.Cfg.Auth.LoginIPLimit = 20
	for i := 1; i <= 20; i++ { // username ต่างกันทุกครั้ง ไม่ให้ไปโดน AUTH-10 ก่อน
		expect(t, login(t, app, fmt.Sprintf("ghost%d", i), "x"), 401, 401201)
	}
	expect(t, login(t, app, "ghost21", "x"), 429, 429)
}

func TestMiddlewareRejects(t *testing.T) { // AUTH-14, AUTH-15
	app := setup(t)
	a := createAgent(t, "agent01", models.AgentStatusActive, nil)
	tok := loginToken(t, app, "agent01")

	expect(t, testutil.Call(t, app, "GET", mePath, nil, ""), 401, 401202)
	expect(t, testutil.Call(t, app, "GET", mePath, nil, "garbage"), 401, 401202)
	forged, _ := utils.SignSessionToken("another-secret-at-least-32-characters", utils.TokenTypeBackoffice, "sid", a.ID, time.Now(), time.Now().Add(time.Hour))
	expect(t, testutil.Call(t, app, "GET", mePath, nil, forged), 401, 401202)

	setStatus(t, a.ID, models.AgentStatusLocked)
	expect(t, testutil.Call(t, app, "GET", mePath, nil, tok), 403, 401301)
	setStatus(t, a.ID, models.AgentStatusActive)
	expect(t, testutil.Call(t, app, "GET", mePath, nil, tok), 401, 401203) // session ถูกลบไปแล้ว
}
