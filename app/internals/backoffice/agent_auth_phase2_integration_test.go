//go:build integration

// API test ของ agent_auth phase 2 ตามตาราง test case ใน docs/modules/agent_auth_phase2.md หัวข้อ 7
package backoffice_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	mw "app/app/internals/backoffice/middleware"
	"app/app/models"
	agentAuthPostgres "app/app/repository/postgres/agent_auth"
	"app/pkg/configs"
	"app/pkg/response"
	"app/pkg/testutil"
	"app/pkg/utils"
	"app/platform/database"

	"github.com/gofiber/fiber/v2"
)

//nolint:gosec // G101: path และค่าทดสอบ ไม่ใช่ credential จริง
const (
	gatedPath          = "/api/v1/bo/pr/_test/gated" // route ทั่วไป: Authenticated + PassedGates
	txPath             = "/api/v1/bo/pr/_test/tx"    // route ที่ต้องยืนยัน passcode
	viewPath           = "/api/v1/bo/pr/_test/view"  // route ที่บัญชีถูกระงับเข้าได้ (AUTH-54): PassedGatesAllowSuspended
	passcodeSetupPath  = "/api/v1/bo/pr/auth/passcode/setup"
	passcodeChangePath = "/api/v1/bo/pr/auth/passcode/change"
	passwordChangePath = "/api/v1/bo/pr/auth/password/change"
	resetPasscodePath  = "/api/v1/bo/pr/admin/passcode/reset"
	resetPasswordPath  = "/api/v1/bo/pr/admin/password/reset"
	passcode           = "123456"
	newPassword        = "Zx9!kq2m"
)

func setup2(t *testing.T) *fiber.App {
	t.Helper()
	testutil.Setup(t)
	configs.Cfg.Auth.LoginIPLimit = 1000
	return testutil.NewApp(func(a *fiber.App) {
		a.Get(gatedPath, mw.Authenticated(), mw.PassedGates(), func(c *fiber.Ctx) error {
			act := mw.GetActor(c)
			return response.OK(c, fiber.Map{"agent_id": act.AgentID, "subaccount_id": act.SubaccountID, "effective_status": act.EffectiveStatus})
		})
		a.Post(txPath, mw.Authenticated(), mw.PassedGates(), mw.RequirePasscode(), func(c *fiber.Ctx) error {
			return response.OK(c, nil)
		})
		a.Get(viewPath, mw.Authenticated(), mw.PassedGatesAllowSuspended(), func(c *fiber.Ctx) error {
			return response.OK(c, fiber.Map{"effective_status": mw.GetActor(c).EffectiveStatus})
		})
	})
}

func createAccount(t *testing.T, username string, role models.AgentRole, parentID *uint) models.UserAgent {
	t.Helper()
	a := createAgent(t, username, models.AgentStatusActive, parentID)
	if role != models.AgentRoleAgent {
		cols := map[string]any{"role": role}
		switch role { // Company / Share ต้องมี agent_type (agent_management MGMT-01)
		case models.AgentRoleCompany:
			cols["agent_type"] = models.AgentTypeTransfer
		case models.AgentRoleShareholder:
			cols["agent_type"] = models.AgentTypeShareB2C
		}
		if err := database.DBConn.Model(&models.UserAgent{}).Where("id = ?", a.ID).Updates(cols).Error; err != nil {
			t.Fatal(err)
		}
		a.Role = role
	}
	return a
}

func createSub(t *testing.T, owner models.UserAgent, name string, status models.AgentStatus) models.Subaccount {
	t.Helper()
	hash, err := utils.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	s := models.Subaccount{AgentID: owner.ID, Username: owner.Username + "@" + name, PasswordHash: hash, Status: status}
	if err := agentAuthPostgres.CreateSubaccountRepository(database.DBConn, &s); err != nil {
		t.Fatal(err)
	}
	return s
}

func table(t models.AccountType) string {
	if t == models.AccountTypeSub {
		return "subaccounts"
	}
	return "user_agents"
}

func setCols(t *testing.T, at models.AccountType, id uint, cols map[string]any) {
	t.Helper()
	if err := database.DBConn.Table(table(at)).Where("id = ?", id).Updates(cols).Error; err != nil {
		t.Fatal(err)
	}
}

func setPasscode(t *testing.T, at models.AccountType, id uint, plain string) {
	t.Helper()
	hash, err := utils.HashPassword(plain)
	if err != nil {
		t.Fatal(err)
	}
	setCols(t, at, id, map[string]any{"passcode_hash": hash})
}

// loginReady login แล้วคืน token ของบัญชีที่มี passcode แล้ว (ผ่านด่านครบ)
func loginReady(t *testing.T, app *fiber.App, at models.AccountType, id uint, username string) string {
	t.Helper()
	setPasscode(t, at, id, passcode)
	return loginToken(t, app, username)
}

type loginData struct {
	Token              string `json:"token"`
	Role               string `json:"role"`
	IsSubaccount       bool   `json:"is_subaccount"`
	PasscodeSet        bool   `json:"passcode_set"`
	MustChangePassword bool   `json:"must_change_password"`
	MustChangePasscode bool   `json:"must_change_passcode"`
}

func loginAs(t *testing.T, app *fiber.App, username, pw string) loginData {
	t.Helper()
	r := login(t, app, username, pw)
	expect(t, r, 200, 200)
	var d loginData
	if err := json.Unmarshal(r.Data, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

func call(t *testing.T, app *fiber.App, method, path string, body any, tok string) testutil.Response {
	t.Helper()
	return testutil.Call(t, app, method, path, body, tok)
}

func redisExists(t *testing.T, key string) bool {
	t.Helper()
	n, err := database.DBRedis.Exists(context.Background(), key).Result()
	if err != nil {
		t.Fatal(err)
	}
	return n > 0
}

func getStatus(t *testing.T, at models.AccountType, id uint) string {
	t.Helper()
	var s string
	database.DBConn.Table(table(at)).Select("status").Where("id = ?", id).Scan(&s)
	return s
}

// ---- Subaccount login (AUTH-17–AUTH-28) ----

func TestSubaccountLogin(t *testing.T) {
	app := setup2(t)
	grandpa := createAgent(t, "grandpa", models.AgentStatusActive, nil)
	owner := createAgent(t, "agent01", models.AgentStatusActive, &grandpa.ID)
	createSub(t, owner, "staff", models.AgentStatusActive)
	createSub(t, owner, "locked", models.AgentStatusLocked)

	d := loginAs(t, app, " Agent01@Staff ", password)          // AUTH-18
	if !d.IsSubaccount || d.Role != "AGENT" || d.PasscodeSet { // AUTH-25
		t.Fatalf("unexpected %+v", d)
	}

	tests := []struct {
		name, username, pw string
		status, code       int
	}{
		{"ไม่มี sub นี้", "agent01@nobody", password, 401, 401201},
		{"ไม่มีผู้สร้างนี้", "ghost@xyz", password, 401, 401201},
		{"รูปแบบ name ผิด", "agent01@ab", password, 401, 401201},
		{"sub LOCKED + รหัสถูก", "agent01@locked", password, 403, 401301},
		{"sub LOCKED + รหัสผิด", "agent01@locked", "wrong", 401, 401201},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expect(t, login(t, app, tt.username, tt.pw), tt.status, tt.code)
		})
	}

	t.Run("ผู้สร้าง LOCKED", func(t *testing.T) { // AUTH-21
		setStatus(t, owner.ID, models.AgentStatusLocked)
		defer setStatus(t, owner.ID, models.AgentStatusActive)
		expect(t, login(t, app, "agent01@staff", password), 403, 401302)
	})
	t.Run("upline ของผู้สร้าง LOCKED", func(t *testing.T) {
		setStatus(t, grandpa.ID, models.AgentStatusLocked)
		defer setStatus(t, grandpa.ID, models.AgentStatusActive)
		expect(t, login(t, app, "agent01@staff", password), 403, 401302)
	})
	t.Run("ผู้สร้าง SUSPENDED login ได้", func(t *testing.T) { // AUTH-22
		setStatus(t, owner.ID, models.AgentStatusSuspended)
		defer setStatus(t, owner.ID, models.AgentStatusActive)
		expect(t, login(t, app, "agent01@staff", password), 200, 200)
	})
}

func TestSubaccountSessions(t *testing.T) { // AUTH-23, AUTH-24, logout
	app := setup2(t)
	owner := createAgent(t, "agent01", models.AgentStatusActive, nil)
	sub := createSub(t, owner, "staff", models.AgentStatusActive)
	setPasscode(t, models.AccountTypeAgent, owner.ID, passcode)

	subTok := loginReady(t, app, models.AccountTypeSub, sub.ID, "agent01@staff")
	ownerTok := loginToken(t, app, "agent01")
	r := call(t, app, "GET", gatedPath, nil, subTok)
	expect(t, r, 200, 200)
	var d struct {
		AgentID      uint `json:"agent_id"`
		SubaccountID uint `json:"subaccount_id"`
	}
	_ = json.Unmarshal(r.Data, &d)
	if d.AgentID != owner.ID || d.SubaccountID != sub.ID { // AUTH-26
		t.Fatalf("actor ของ sub ต้องทำงานในนามผู้สร้าง ได้ %+v", d)
	}
	expect(t, call(t, app, "GET", gatedPath, nil, ownerTok), 200, 200)

	expect(t, call(t, app, "POST", logoutPath, nil, subTok), 200, 200)
	expect(t, call(t, app, "GET", gatedPath, nil, subTok), 401, 401203)
	expect(t, call(t, app, "GET", gatedPath, nil, ownerTok), 200, 200)
	if redisExists(t, fmt.Sprintf("bo:sess:sub:%d", sub.ID)) {
		t.Fatal("pointer ของ sub ต้องถูกลบ")
	}

	for i := 0; i < 5; i++ {
		expect(t, login(t, app, "agent01@staff", "wrong"), 401, 401201)
	}
	expect(t, login(t, app, "agent01@staff", password), 403, 401303)
	expect(t, login(t, app, "agent01", password), 200, 200)
}

func TestSubaccountMiddlewareLock(t *testing.T) { // AUTH-27
	app := setup2(t)
	owner := createAgent(t, "agent01", models.AgentStatusActive, nil)
	sub := createSub(t, owner, "staff", models.AgentStatusActive)

	tok := loginReady(t, app, models.AccountTypeSub, sub.ID, "agent01@staff")
	setCols(t, models.AccountTypeSub, sub.ID, map[string]any{"status": models.AgentStatusLocked})
	expect(t, call(t, app, "GET", gatedPath, nil, tok), 403, 401301)
	setCols(t, models.AccountTypeSub, sub.ID, map[string]any{"status": models.AgentStatusActive})
	expect(t, call(t, app, "GET", gatedPath, nil, tok), 401, 401203)

	tok = loginToken(t, app, "agent01@staff")
	setStatus(t, owner.ID, models.AgentStatusLocked)
	expect(t, call(t, app, "GET", gatedPath, nil, tok), 403, 401302)
	setStatus(t, owner.ID, models.AgentStatusActive)
	expect(t, call(t, app, "GET", gatedPath, nil, tok), 401, 401203)
}

func TestAgentUplineLockedMidSession(t *testing.T) { // AUTH-27 (ใช้กับ agent ด้วย)
	app := setup2(t)
	company := createAccount(t, "company01", models.AgentRoleCompany, nil)
	a := createAgent(t, "agent01", models.AgentStatusActive, &company.ID)
	tok := loginReady(t, app, models.AccountTypeAgent, a.ID, "agent01")

	setStatus(t, company.ID, models.AgentStatusLocked)
	expect(t, call(t, app, "GET", gatedPath, nil, tok), 403, 401302)
	setStatus(t, company.ID, models.AgentStatusActive)
	expect(t, call(t, app, "GET", gatedPath, nil, tok), 401, 401203) // session ถูกลบไปแล้ว
}

func TestEffectiveStatus(t *testing.T) { // AUTH-53
	app := setup2(t)
	company := createAccount(t, "company01", models.AgentRoleCompany, nil)
	a := createAgent(t, "agent01", models.AgentStatusActive, &company.ID)
	sub := createSub(t, a, "staff", models.AgentStatusActive)
	agentTok := loginReady(t, app, models.AccountTypeAgent, a.ID, "agent01")
	subTok := loginReady(t, app, models.AccountTypeSub, sub.ID, "agent01@staff")

	effective := func(tok string) string {
		t.Helper()
		r := call(t, app, "GET", viewPath, nil, tok)
		expect(t, r, 200, 200)
		var d struct {
			EffectiveStatus string `json:"effective_status"`
		}
		_ = json.Unmarshal(r.Data, &d)
		return d.EffectiveStatus
	}

	if got := effective(subTok); got != "ACTIVE" {
		t.Fatalf("ทั้งสาย ACTIVE ได้ %s", got)
	}
	setStatus(t, company.ID, models.AgentStatusSuspended) // upline SUSPENDED → ข้างล่างโดนด้วย
	if got := effective(agentTok); got != "SUSPENDED" {
		t.Fatalf("agent ที่ upline SUSPENDED ได้ %s", got)
	}
	if got := effective(subTok); got != "SUSPENDED" {
		t.Fatalf("sub ที่ upline ของผู้สร้าง SUSPENDED ได้ %s", got)
	}
	setStatus(t, company.ID, models.AgentStatusActive)
	setCols(t, models.AccountTypeSub, sub.ID, map[string]any{"status": models.AgentStatusSuspended})
	if got := effective(subTok); got != "SUSPENDED" {
		t.Fatalf("sub SUSPENDED ได้ %s", got)
	}
	if got := effective(agentTok); got != "ACTIVE" {
		t.Fatalf("sub SUSPENDED ต้องไม่กระทบผู้สร้าง ได้ %s", got)
	}
}

// ---- บัญชีถูกระงับ (AUTH-54) ----

func TestSuspendedAccess(t *testing.T) { // AUTH-54
	app := setup2(t)
	company := createAccount(t, "company01", models.AgentRoleCompany, nil)
	a := createAgent(t, "agent01", models.AgentStatusActive, &company.ID)
	sub := createSub(t, a, "staff", models.AgentStatusActive)
	agentTok := loginReady(t, app, models.AccountTypeAgent, a.ID, "agent01")
	subTok := loginReady(t, app, models.AccountTypeSub, sub.ID, "agent01@staff")

	// route ที่บัญชีถูกระงับใช้ไม่ได้: route ทั่วไป, route ที่ยืนยัน passcode, เปลี่ยน passcode / รหัสผ่านของตัวเอง
	denied := func(t *testing.T, tok string) {
		t.Helper()
		expect(t, call(t, app, "GET", gatedPath, nil, tok), 403, 401311)
		expect(t, call(t, app, "POST", txPath, map[string]any{"passcode": passcode}, tok), 403, 401311)
		expect(t, call(t, app, "POST", passcodeChangePath, map[string]any{
			"old_passcode": passcode, "new_passcode": "654321", "confirm_passcode": "654321",
		}, tok), 403, 401311)
		expect(t, call(t, app, "POST", passwordChangePath, map[string]any{
			"old_password": password, "new_password": newPassword, "confirm_password": newPassword, "passcode": passcode,
		}, tok), 403, 401311)
	}

	t.Run("upline SUSPENDED: agent และ sub ดูได้อย่างเดียว", func(t *testing.T) {
		setStatus(t, company.ID, models.AgentStatusSuspended)
		t.Cleanup(func() { setStatus(t, company.ID, models.AgentStatusActive) })
		for _, tok := range []string{agentTok, subTok} {
			expect(t, call(t, app, "GET", viewPath, nil, tok), 200, 200)
			denied(t, tok)
		}
	})

	t.Run("sub SUSPENDED: ผู้สร้างไม่กระทบ", func(t *testing.T) {
		setCols(t, models.AccountTypeSub, sub.ID, map[string]any{"status": models.AgentStatusSuspended})
		t.Cleanup(func() {
			setCols(t, models.AccountTypeSub, sub.ID, map[string]any{"status": models.AgentStatusActive})
		})
		expect(t, call(t, app, "GET", viewPath, nil, subTok), 200, 200)
		denied(t, subTok)
		expect(t, call(t, app, "GET", gatedPath, nil, agentTok), 200, 200)
	})

	t.Run("ปลดระงับแล้วใช้งานได้ปกติ", func(t *testing.T) {
		expect(t, call(t, app, "GET", gatedPath, nil, agentTok), 200, 200)
		expect(t, call(t, app, "GET", gatedPath, nil, subTok), 200, 200)
	})

	t.Run("ยังไม่ตั้ง passcode: ตั้งไม่ได้ และหน้าดูยังติดด่านหลัง login", func(t *testing.T) {
		createAgent(t, "agent02", models.AgentStatusSuspended, &company.ID)
		tok := loginAs(t, app, "agent02", password).Token
		expect(t, call(t, app, "POST", passcodeSetupPath, map[string]any{
			"passcode": passcode, "confirm_passcode": passcode,
		}, tok), 403, 401311)
		expect(t, call(t, app, "GET", viewPath, nil, tok), 403, 401304)
	})
}

// ---- ด่านหลัง login (AUTH-29) ----

func TestPostLoginGates(t *testing.T) {
	app := setup2(t)
	a := createAgent(t, "agent01", models.AgentStatusActive, nil)

	tok := loginToken(t, app, "agent01")
	expect(t, call(t, app, "GET", gatedPath, nil, tok), 403, 401304)
	expect(t, call(t, app, "POST", passwordChangePath, map[string]string{}, tok), 403, 401304)
	expect(t, call(t, app, "POST", passcodeSetupPath, map[string]string{"passcode": passcode, "confirm_passcode": passcode}, tok), 200, 200)
	expect(t, call(t, app, "GET", gatedPath, nil, tok), 200, 200) // AUTH-31 ไม่ต้องยืนยันซ้ำ

	tok = loginToken(t, app, "agent01") // มี passcode แล้ว ใช้งานได้ทันที
	expect(t, call(t, app, "GET", gatedPath, nil, tok), 200, 200)

	setCols(t, models.AccountTypeAgent, a.ID, map[string]any{"must_change_password": true})
	expect(t, call(t, app, "POST", passcodeSetupPath, map[string]string{"passcode": passcode, "confirm_passcode": passcode}, tok), 403, 401306)
	expect(t, call(t, app, "POST", logoutPath, nil, tok), 200, 200)

	setCols(t, models.AccountTypeAgent, a.ID, map[string]any{"must_change_password": false, "must_change_passcode": true})
	tok = loginToken(t, app, "agent01")
	expect(t, call(t, app, "POST", passwordChangePath, map[string]string{"old_password": password, "new_password": newPassword, "confirm_password": newPassword, "passcode": passcode}, tok), 403, 401307)
	expect(t, call(t, app, "GET", gatedPath, nil, tok), 403, 401307)
}

// ---- Passcode (AUTH-30–AUTH-35) ----

func TestPasscodeSetup(t *testing.T) {
	app := setup2(t)
	createAgent(t, "agent01", models.AgentStatusActive, nil)
	tok := loginToken(t, app, "agent01")

	bad := []map[string]any{
		{"passcode": 123456, "confirm_passcode": "123456"},
		{"passcode": "123456", "confirm_passcode": "654321"},
		{"passcode": "12345", "confirm_passcode": "12345"},
		{"passcode": "12345a", "confirm_passcode": "12345a"},
		{"confirm_passcode": "123456"},
	}
	for _, body := range bad {
		expect(t, call(t, app, "POST", passcodeSetupPath, body, tok), 200, 422)
	}

	body := map[string]string{"passcode": passcode, "confirm_passcode": passcode}
	var wg sync.WaitGroup
	codes := make([]int, 2)
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i] = testutil.Call(t, app, "POST", passcodeSetupPath, body, tok).Code
		}(i)
	}
	wg.Wait()
	oneWon := codes[0] == 200 && codes[1] == 401401 || codes[0] == 401401 && codes[1] == 200
	if !oneWon {
		t.Fatalf("ยิงพร้อมกันต้องสำเร็จ 1 ครั้ง ได้ %v", codes)
	}
	expect(t, call(t, app, "POST", passcodeSetupPath, body, tok), 200, 401401)
}

func TestRequirePasscode(t *testing.T) { // AUTH-33, AUTH-35
	app := setup2(t)
	a := createAgent(t, "agent01", models.AgentStatusActive, nil)
	tok := loginReady(t, app, models.AccountTypeAgent, a.ID, "agent01")
	wrong := map[string]string{"passcode": "000000"}

	expect(t, call(t, app, "POST", txPath, map[string]any{"passcode": 123456}, tok), 200, 422)
	expect(t, call(t, app, "POST", txPath, nil, tok), 200, 422)
	expect(t, call(t, app, "POST", txPath, map[string]string{"passcode": passcode}, tok), 200, 200)

	t.Run("ผิด 4 ถูก 1 ผิดอีก 4 ยังไม่ถูกบล็อก", func(t *testing.T) {
		for i := 1; i <= 4; i++ {
			r := call(t, app, "POST", txPath, wrong, tok)
			expect(t, r, 200, 401204)
			if want := fmt.Sprintf("เหลือ %d ครั้ง", 5-i); !strings.Contains(r.Msg, want) {
				t.Fatalf("msg ต้องบอก %q ได้ %q", want, r.Msg)
			}
		}
		expect(t, call(t, app, "POST", txPath, map[string]string{"passcode": passcode}, tok), 200, 200)
		for i := 0; i < 4; i++ {
			expect(t, call(t, app, "POST", txPath, wrong, tok), 200, 401204)
		}
	})

	t.Run("ตัวนับอยู่ระดับบัญชี ข้าม session", func(t *testing.T) {
		expect(t, call(t, app, "POST", logoutPath, nil, tok), 200, 200)
		tok = loginToken(t, app, "agent01")
		expect(t, call(t, app, "POST", txPath, wrong, tok), 401, 401205) // ครั้งที่ 5
		expect(t, call(t, app, "GET", gatedPath, nil, tok), 401, 401203)
		expect(t, call(t, app, "POST", logoutPath, nil, tok), 200, 200)
		if getStatus(t, models.AccountTypeAgent, a.ID) != string(models.AgentStatusActive) {
			t.Fatal("status บัญชีต้องไม่เปลี่ยน")
		}
		expect(t, login(t, app, "agent01", password), 403, 401309) // AUTH-28
		expect(t, login(t, app, "agent01", "wrong"), 401, 401201)
	})
}

func TestChangePasscode(t *testing.T) { // AUTH-34
	app := setup2(t)
	owner := createAgent(t, "agent01", models.AgentStatusActive, nil)
	sub := createSub(t, owner, "staff", models.AgentStatusActive)
	tok := loginReady(t, app, models.AccountTypeSub, sub.ID, "agent01@staff")

	expect(t, call(t, app, "POST", passcodeChangePath, map[string]string{"old_passcode": passcode, "new_passcode": passcode, "confirm_passcode": passcode}, tok), 200, 401403)
	expect(t, call(t, app, "POST", passcodeChangePath, map[string]string{"old_passcode": "000000", "new_passcode": "654321", "confirm_passcode": "654321"}, tok), 200, 401204)
	expect(t, call(t, app, "POST", passcodeChangePath, map[string]string{"old_passcode": passcode, "new_passcode": "654321", "confirm_passcode": "654321"}, tok), 200, 200)
	expect(t, call(t, app, "GET", gatedPath, nil, tok), 200, 200) // session เดิมใช้ต่อได้
	expect(t, call(t, app, "POST", txPath, map[string]string{"passcode": passcode}, tok), 200, 401204)
	expect(t, call(t, app, "POST", txPath, map[string]string{"passcode": "654321"}, tok), 200, 200)

	t.Run("old_passcode กับ RequirePasscode นับรวมกัน", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			expect(t, call(t, app, "POST", txPath, map[string]string{"passcode": "000000"}, tok), 200, 401204)
		}
		expect(t, call(t, app, "POST", passcodeChangePath, map[string]string{"old_passcode": "000000", "new_passcode": "111111", "confirm_passcode": "111111"}, tok), 200, 401204)
		expect(t, call(t, app, "POST", passcodeChangePath, map[string]string{"old_passcode": "000000", "new_passcode": "111111", "confirm_passcode": "111111"}, tok), 401, 401205)
	})
}

// ---- รหัสผ่าน (AUTH-36–AUTH-41) ----

func TestChangePassword(t *testing.T) {
	app := setup2(t)
	a := createAgent(t, "agent01", models.AgentStatusActive, nil)
	tok := loginReady(t, app, models.AccountTypeAgent, a.ID, "agent01")
	change := func(old, nw string) map[string]string {
		return map[string]string{"old_password": old, "new_password": nw, "confirm_password": nw, "passcode": passcode}
	}

	for _, p := range []string{"Zx9!kq2", "Zx9 kq2m", "Zxkqmwpt", "92837465", "Zx9aaaam"} { // AUTH-36
		expect(t, call(t, app, "POST", passwordChangePath, change(password, p), tok), 200, 422)
	}
	expect(t, call(t, app, "POST", passwordChangePath, map[string]string{"old_password": password, "new_password": newPassword, "confirm_password": newPassword}, tok), 200, 422) // ไม่ส่ง passcode
	expect(t, call(t, app, "POST", passwordChangePath, change("wrong", newPassword), tok), 200, 401206)

	expect(t, call(t, app, "POST", passwordChangePath, change(password, newPassword), tok), 200, 200)
	expect(t, call(t, app, "GET", gatedPath, nil, tok), 200, 200) // AUTH-40 session เดิมใช้ต่อได้
	expect(t, login(t, app, "agent01", password), 401, 401201)
	tok = loginToken2(t, app, "agent01", newPassword)

	expect(t, call(t, app, "POST", passwordChangePath, change(newPassword, newPassword), tok), 200, 401402) // ปัจจุบัน
	expect(t, call(t, app, "POST", passwordChangePath, change(newPassword, "abcd1234"), tok), 200, 200)
	expect(t, call(t, app, "POST", passwordChangePath, change("abcd1234", newPassword), tok), 200, 401402) // ก่อนหน้า

	t.Run("old_password ผิด 5 ครั้ง", func(t *testing.T) { // AUTH-39
		for i := 0; i < 4; i++ {
			expect(t, call(t, app, "POST", passwordChangePath, change("wrong", "Qw8@rt5y"), tok), 200, 401206)
		}
		expect(t, call(t, app, "POST", passwordChangePath, change("wrong", "Qw8@rt5y"), tok), 403, 401303)
		expect(t, call(t, app, "GET", gatedPath, nil, tok), 401, 401203)
		expect(t, login(t, app, "agent01", "abcd1234"), 403, 401303)
	})
}

func loginToken2(t *testing.T, app *fiber.App, username, pw string) string {
	t.Helper()
	return loginAs(t, app, username, pw).Token
}

// ---- Admin reset (AUTH-43–AUTH-50) ----

func auditRows(t *testing.T, action models.AuthAuditAction) []models.AuthAuditLog {
	t.Helper()
	var logs []models.AuthAuditLog
	if err := database.DBConn.Where("action = ?", action).Order("id").Find(&logs).Error; err != nil {
		t.Fatal(err)
	}
	return logs
}

func TestAuditLog(t *testing.T) { // AUTH-49: บันทึกเฉพาะเหตุการณ์ที่เปลี่ยนข้อมูลบัญชี
	app := setup2(t)
	a := createAgent(t, "agent01", models.AgentStatusActive, nil)
	tok := loginToken(t, app, "agent01")

	expect(t, call(t, app, "POST", passcodeSetupPath, map[string]string{"passcode": passcode, "confirm_passcode": passcode}, tok), 200, 200)
	expect(t, call(t, app, "POST", passwordChangePath, map[string]string{"old_password": password, "new_password": newPassword, "confirm_password": newPassword, "passcode": passcode}, tok), 200, 200)
	for _, action := range []models.AuthAuditAction{models.AuthAuditPasscodeSetup, models.AuthAuditPasswordChange} {
		logs := auditRows(t, action)
		if len(logs) != 1 || logs[0].ActorType != models.AuthAuditActorAgent || *logs[0].ActorID != a.ID || *logs[0].TargetID != a.ID {
			t.Fatalf("%s: audit ไม่ถูกต้อง %+v", action, logs)
		}
	}

	t.Run("passcode ผิดครบ → PASSCODE_BLOCKED โดย SYSTEM", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			call(t, app, "POST", txPath, map[string]string{"passcode": "000000"}, tok)
		}
		logs := auditRows(t, models.AuthAuditPasscodeBlocked)
		if len(logs) != 1 || logs[0].ActorType != models.AuthAuditActorSystem || logs[0].ActorID != nil || *logs[0].TargetID != a.ID {
			t.Fatalf("audit ไม่ถูกต้อง %+v", logs)
		}
	})

	t.Run("login ผิดครบ username ที่ไม่มีจริง → LOGIN_BLOCKED ไม่มี target", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			login(t, app, "ghost", "wrong")
		}
		logs := auditRows(t, models.AuthAuditLoginBlocked)
		if len(logs) != 1 || logs[0].TargetType != nil || logs[0].TargetUsername != "ghost" || logs[0].ActorType != models.AuthAuditActorSystem {
			t.Fatalf("audit ไม่ถูกต้อง %+v", logs)
		}
	})

	t.Run("login / logout / login ผิดทีละครั้งไม่บันทึก", func(t *testing.T) {
		var n int64
		database.DBConn.Model(&models.AuthAuditLog{}).Count(&n)
		createAgent(t, "agent02", models.AgentStatusActive, nil) // agent01 ถูกบล็อก passcode อยู่
		login(t, app, "agent02", "wrong")
		tok2 := loginToken(t, app, "agent02")
		expect(t, call(t, app, "POST", logoutPath, nil, tok2), 200, 200)
		var after int64
		database.DBConn.Model(&models.AuthAuditLog{}).Count(&after)
		if after != n {
			t.Fatalf("ต้องไม่มีแถวเพิ่ม ได้ %d → %d", n, after)
		}
	})
}

func TestForcedChangeBoth(t *testing.T) { // AUTH-50
	app := setup2(t)
	a := createAgent(t, "agent01", models.AgentStatusActive, nil)
	temp := "Ab3dEf6hJk7m"
	hash, _ := utils.HashPassword(temp)
	setPasscode(t, models.AccountTypeAgent, a.ID, "482913")
	exp := time.Now().Add(time.Hour)
	setCols(t, models.AccountTypeAgent, a.ID, map[string]any{"password_hash": hash, "must_change_password": true, "temp_password_expires_at": exp,
		"must_change_passcode": true, "temp_passcode_expires_at": exp})

	tok := loginToken2(t, app, "agent01", temp)
	expect(t, call(t, app, "POST", passcodeChangePath, map[string]string{"old_passcode": "482913", "new_passcode": "654321", "confirm_passcode": "654321"}, tok), 403, 401306)
	expect(t, call(t, app, "POST", passwordChangePath, map[string]string{"old_password": temp, "new_password": newPassword, "confirm_password": newPassword}, tok), 200, 200)
	expect(t, call(t, app, "POST", passcodeChangePath, map[string]string{"old_passcode": "482913", "new_passcode": "654321", "confirm_passcode": "654321"}, tok), 200, 200)
	expect(t, call(t, app, "GET", gatedPath, nil, tok), 200, 200)
}

func TestTempCredentialExpired(t *testing.T) { // AUTH-28, AUTH-34, AUTH-41
	app := setup2(t)
	a := createAgent(t, "agent01", models.AgentStatusActive, nil)
	setPasscode(t, models.AccountTypeAgent, a.ID, "482913")
	past := time.Now().Add(-time.Minute)

	setCols(t, models.AccountTypeAgent, a.ID, map[string]any{"must_change_password": true, "temp_password_expires_at": past})
	expect(t, login(t, app, "agent01", password), 403, 401310)
	expect(t, login(t, app, "agent01", "wrong"), 401, 401201) // เช็คหลังรหัสถูกเท่านั้น

	setCols(t, models.AccountTypeAgent, a.ID, map[string]any{"temp_password_expires_at": time.Now().Add(time.Hour)})
	tok := loginToken(t, app, "agent01")
	setCols(t, models.AccountTypeAgent, a.ID, map[string]any{"temp_password_expires_at": past})
	expect(t, call(t, app, "POST", passwordChangePath, map[string]string{"old_password": password, "new_password": newPassword, "confirm_password": newPassword}, tok), 403, 401310)

	setCols(t, models.AccountTypeAgent, a.ID, map[string]any{"must_change_password": false, "must_change_passcode": true, "temp_passcode_expires_at": time.Now().Add(time.Hour)})
	tok = loginToken(t, app, "agent01")
	setCols(t, models.AccountTypeAgent, a.ID, map[string]any{"temp_passcode_expires_at": past})
	expect(t, call(t, app, "POST", passcodeChangePath, map[string]string{"old_passcode": "482913", "new_passcode": "654321", "confirm_passcode": "654321"}, tok), 403, 401310)
}
