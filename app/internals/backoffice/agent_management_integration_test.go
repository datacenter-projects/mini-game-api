//go:build integration

// API test ของ agent_management (phase 2: เส้นสร้าง) ตามตาราง test case ใน docs/modules/agent_management.md หัวข้อ 7
package backoffice_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	agentManagementCore "app/app/core/agent_management"
	"app/app/models"
	agentAuthPostgres "app/app/repository/postgres/agent_auth"
	"app/pkg/testutil"
	"app/pkg/utils"
	"app/platform/database"

	"github.com/gofiber/fiber/v2"
)

const (
	createAgentPath  = "/api/v1/bo/pr/manage/agents/create"
	createMemberPath = "/api/v1/bo/pr/manage/members/create"
	mgPassword       = "aA4b4c4d4e4f" // ผ่าน AUTH-36
)

var reqSeq int

// newRequestID — UUID ไม่ซ้ำต่อ test run
func newRequestID() string {
	reqSeq++
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", reqSeq)
}

// seedSuperadmin — Superadmin แบบเดียวกับ backfill ของ migration (ได้รับ 100% · ครบ 27 สกุล) พร้อม token
func seedSuperadmin(t *testing.T, app *fiber.App) (models.UserAgent, string) {
	t.Helper()
	hash, err := utils.HashPassword(mgPassword)
	if err != nil {
		t.Fatal(err)
	}
	sa := models.UserAgent{Username: "superadmin", PasswordHash: hash, Role: models.AgentRoleSuperAdmin, Status: models.AgentStatusActive}
	if err := agentAuthPostgres.CreateUserAgentRepository(database.DBConn, &sa); err != nil {
		t.Fatal(err)
	}
	var curs []models.AgentCurrency
	for _, c := range agentManagementCore.Currencies {
		curs = append(curs, models.AgentCurrency{AgentID: sa.ID, Currency: c})
	}
	var games []models.AgentGameSetting
	for _, g := range agentManagementCore.GamesOf(agentManagementCore.PTGroupMinigame) {
		games = append(games, models.AgentGameSetting{AgentID: sa.ID, GameCode: g.GameCode, Category: g.Category,
			PTFromParent: 100, Status: true, StatusGame: true})
	}
	if err := database.DBConn.Create(&curs).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.DBConn.Create(&games).Error; err != nil {
		t.Fatal(err)
	}
	return sa, readyToken(t, app, models.AccountTypeAgent, sa.ID, "superadmin")
}

func readyToken(t *testing.T, app *fiber.App, at models.AccountType, id uint, username string) string {
	t.Helper()
	setPasscode(t, at, id, passcode)
	return loginToken2(t, app, username, mgPassword)
}

func childPT(give, force, remain, commission float64) map[string]any {
	return map[string]any{"minigame": map[string]any{"pt_from_parent": give, "force": force, "remain_quota": remain,
		"commission_percent": commission, "status": true}}
}

func agentBody(userType, username string, currencies []string, pt map[string]any) map[string]any {
	b := map[string]any{"request_id": newRequestID(), "user_type": userType, "username": username, "password": mgPassword,
		"name": "Name" + username, "phone": "", "pt": pt}
	if currencies != nil {
		b["currencies"] = currencies
	}
	return b
}

func memberBody(username string, commission float64) map[string]any {
	return map[string]any{"request_id": newRequestID(), "username": username, "password": mgPassword, "name": "Name" + username,
		"phone": "", "pt": map[string]any{"minigame": map[string]any{"pt": 0, "commission_percent": commission}}}
}

type created struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	UserType string `json:"user_type"`
}

// mustCreate — สร้างผ่าน API แล้วคืน id + token ของบัญชีใหม่ (ตั้ง passcode ให้ผ่านด่าน)
func mustCreate(t *testing.T, app *fiber.App, tok string, body map[string]any) (created, string) {
	t.Helper()
	r := call(t, app, "POST", createAgentPath, body, tok)
	expect(t, r, 200, 200)
	var d created
	if err := json.Unmarshal(r.Data, &d); err != nil {
		t.Fatal(err)
	}
	return d, readyToken(t, app, models.AccountTypeAgent, d.ID, d.Username)
}

func setBalance(t *testing.T, agentID uint, currency string, amount float64) {
	t.Helper()
	if err := database.DBConn.Save(&models.AgentBalance{AgentID: agentID, Currency: currency, Amount: amount}).Error; err != nil {
		t.Fatal(err)
	}
}

func agentBalance(t *testing.T, agentID uint, currency string) float64 {
	t.Helper()
	var b models.AgentBalance
	database.DBConn.Where("agent_id = ? AND currency = ?", agentID, currency).Take(&b)
	return b.Amount
}

func countRows(t *testing.T, model any, where string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := database.DBConn.Model(model).Where(where, args...).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

// chain — Superadmin → Company Transfer (ได้รับ 90) → Share B2C (70) → Agent (60) ทุกตัวสกุล THB
type chain struct {
	saTok, comTok, shareTok, agentTok string
	com, share, agent                 created
}

func buildChain(t *testing.T, app *fiber.App) chain {
	t.Helper()
	var c chain
	_, c.saTok = seedSuperadmin(t, app)
	c.com, c.comTok = mustCreate(t, app, c.saTok, agentBody("COMPANY_TRANSFER", "comp01", nil, childPT(90, 0, 0, 0.5)))
	c.share, c.shareTok = mustCreate(t, app, c.comTok, agentBody("SHARE_B2C", "share01", []string{"THB"}, childPT(70, 0, 0, 0.5)))
	c.agent, c.agentTok = mustCreate(t, app, c.shareTok, agentBody("AGENT", "agent01", nil, childPT(60, 0, 0, 0.5)))
	return c
}

func TestCreateMatrix(t *testing.T) { // MGMT-02
	app := setup2(t)
	c := buildChain(t, app)
	_, b2bTok := mustCreate(t, app, c.comTok, agentBody("SHARE_B2B", "shareb2b", []string{"THB", "USD"}, childPT(50, 0, 0, 0)))
	reseller, resellerTok := mustCreate(t, app, c.saTok, agentBody("COMPANY_SEAMLESS_RESELLER", "reseller01", nil, childPT(80, 0, 0, 0)))
	_ = reseller
	_, oneTok := mustCreate(t, app, c.saTok, agentBody("COMPANY_SEAMLESS_1TO1", "one2one", []string{"JPY"}, childPT(80, 0, 0, 0)))

	r := call(t, app, "POST", createAgentPath, agentBody("SHARE_B2C", "shareres", []string{"THB"}, childPT(80, 0, 0, 0)), resellerTok)
	expect(t, r, 200, 200)
	var d created
	_ = json.Unmarshal(r.Data, &d)
	if d.UserType != "SHARE_RESELLER" {
		t.Fatalf("user_type = %q, want SHARE_RESELLER", d.UserType)
	}

	expect(t, call(t, app, "POST", createAgentPath, agentBody("SHARE_B2C", "share1to1", []string{"JPY"}, childPT(10, 0, 0, 0)), oneTok), 200, 402301)
	expect(t, call(t, app, "POST", createMemberPath, memberBody("memb2b", 0), b2bTok), 200, 402301)
	expect(t, call(t, app, "POST", createAgentPath, agentBody("SHARE_B2B", "compshare", []string{"THB"}, childPT(10, 0, 0, 0)), c.saTok), 200, 402301)

	expect(t, call(t, app, "POST", createMemberPath, memberBody("memshare", 0.3), c.shareTok), 200, 200)
	expect(t, call(t, app, "POST", createMemberPath, memberBody("memagent", 0.3), c.agentTok), 200, 200)
	expect(t, call(t, app, "POST", createMemberPath, memberBody("mem1to1", 0), oneTok), 200, 200)

	// Agent ซ้อน 3 ชั้น
	tok := c.agentTok
	for i := 1; i <= 3; i++ {
		_, tok = mustCreate(t, app, tok, agentBody("AGENT", fmt.Sprintf("sub%dagent", i), nil, childPT(60, 0, 0, 0)))
	}
	var m models.UserMember
	database.DBConn.Where("username = ?", "memagent").Take(&m)
	if m.Currency != "THB" || m.AgentID != c.agent.ID {
		t.Fatalf("member currency %q agent %d", m.Currency, m.AgentID)
	}
}

func TestCreateValidation(t *testing.T) { // MGMT-05, MGMT-08, MGMT-12, MGMT-14, MGMT-17, MGMT-18
	app := setup2(t)
	c := buildChain(t, app)
	_, b2bTok := mustCreate(t, app, c.comTok, agentBody("SHARE_B2B", "shareb2b", []string{"THB", "USD"}, childPT(50, 0, 0, 0)))

	var a models.UserAgent
	database.DBConn.Where("id = ?", c.share.ID).Take(&a)
	if a.Username != "share01" || c.share.Username != "share01" {
		t.Fatalf("username = %q", a.Username)
	}

	share := func(mod func(b map[string]any)) map[string]any {
		b := agentBody("SHARE_B2C", "newshare", []string{"THB"}, childPT(50, 0, 0, 0))
		mod(b)
		return b
	}
	tests := []struct {
		name string
		tok  string
		body map[string]any
		code int
	}{
		{"username สั้น", c.comTok, share(func(b map[string]any) { b["username"] = "ab" }), 422},
		{"username 33 ตัว", c.comTok, share(func(b map[string]any) { b["username"] = strings.Repeat("a", 33) }), 422},
		{"username มี _", c.comTok, share(func(b map[string]any) { b["username"] = "new_share" }), 422},
		{"username ซ้ำ", c.comTok, share(func(b map[string]any) { b["username"] = "Agent01" }), 402401},
		{"เบอร์มี +", c.comTok, share(func(b map[string]any) { b["phone"] = "+66812345678" }), 422},
		{"เบอร์ 7 หลัก", c.comTok, share(func(b map[string]any) { b["phone"] = "0812345" }), 422},
		{"ชื่อมีช่องว่าง", c.comTok, share(func(b map[string]any) { b["name"] = "สมชาย ใจดี" }), 422},
		{"ชื่อ 2 ตัว", c.comTok, share(func(b map[string]any) { b["name"] = "สม" }), 422},
		{"รหัสผ่านผิดกฎ", c.comTok, share(func(b map[string]any) { b["password"] = "aaaa1111" }), 422},
		{"ไม่มี request_id", c.comTok, share(func(b map[string]any) { delete(b, "request_id") }), 422},
		{"B2B ไม่ส่งสกุล", c.comTok, share(func(b map[string]any) { b["user_type"] = "SHARE_B2B"; delete(b, "currencies") }), 422},
		{"B2C 2 สกุล", c.comTok, share(func(b map[string]any) { b["currencies"] = []string{"THB", "USD"} }), 422},
		{"Agent ใต้ B2B เลือก JPY", b2bTok, agentBody("AGENT", "agjpy", []string{"JPY"}, childPT(10, 0, 0, 0)), 402310},
		{"force null", c.comTok, share(func(b map[string]any) { b["pt"].(map[string]any)["minigame"].(map[string]any)["force"] = nil }), 422},
		{"phone null", c.comTok, share(func(b map[string]any) { b["phone"] = nil }), 422},
		{"ไม่ส่งกลุ่ม game", c.comTok, share(func(b map[string]any) { b["pt"] = map[string]any{} }), 422},
		{"ถือ 30.25", c.comTok, share(func(b map[string]any) { b["pt"] = childPT(30.25, 0, 0, 0) }), 422},
		{"ถือ 30.12345 (ทศนิยมเกิน 4)", c.comTok, share(func(b map[string]any) { b["pt"] = childPT(30.12345, 0, 0, 0) }), 422},
		{"commission 1.01", c.comTok, share(func(b map[string]any) { b["pt"] = childPT(30, 0, 0, 1.01) }), 422},
		{"commission 1.1", c.comTok, share(func(b map[string]any) { b["pt"] = childPT(30, 0, 0, 1.1) }), 402309},
		{"ได้รับ 90 ให้ 90.5", c.comTok, share(func(b map[string]any) { b["pt"] = childPT(90.5, 0, 0, 0) }), 402305},
		{"ให้ 60 force 70", c.comTok, share(func(b map[string]any) { b["pt"] = childPT(60, 70, 0, 0) }), 402308},
		{"เกมที่ไม่มี", c.comTok, share(func(b map[string]any) { b["status_game"] = map[string]bool{"baccarat": true} }), 422},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expect(t, call(t, app, "POST", createAgentPath, tt.body, tt.tok), 200, tt.code)
		})
	}
	if n := countRows(t, &models.UserAgent{}, "username = ?", "newshare"); n != 0 {
		t.Fatal("ต้องไม่สร้างบัญชีเมื่อถูกปฏิเสธ")
	}

	// เบอร์ซ้ำ · Member ชื่อซ้ำกับ Agent · commission ของลูกเกินผู้สร้างได้
	expect(t, call(t, app, "POST", createAgentPath, share(func(b map[string]any) { b["phone"] = "0812345678" }), c.comTok), 200, 200)
	expect(t, call(t, app, "POST", createAgentPath, share(func(b map[string]any) { b["username"] = "other"; b["phone"] = "0812345678" }), c.comTok), 200, 402403)
	expect(t, call(t, app, "POST", createMemberPath, memberBody("Agent01", 0), c.agentTok), 200, 402401)
	expect(t, call(t, app, "POST", createAgentPath, agentBody("AGENT", "agcomm", nil, childPT(10, 0, 0, 0.6)), c.agentTok), 200, 200)

	// ชื่อภาษาไทย (MGMT-07)
	thai := agentBody("AGENT", "agthai", nil, childPT(10, 0, 0, 0))
	thai["name"] = "สมชาย01"
	expect(t, call(t, app, "POST", createAgentPath, thai, c.agentTok), 200, 200)
	mb := memberBody("memthai", 0)
	mb["name"] = "ใจดี"
	expect(t, call(t, app, "POST", createMemberPath, mb, c.agentTok), 200, 200)

	// เบอร์ของ sub ซ้ำได้ (MGMT-41)
	phone := "0899999999"
	for _, n := range []string{"staff1", "staff2"} {
		s := models.Subaccount{AgentID: c.agent.ID, Username: "agent01@" + n, PasswordHash: "x", Status: models.AgentStatusActive, Phone: &phone}
		if err := agentAuthPostgres.CreateSubaccountRepository(database.DBConn, &s); err != nil {
			t.Fatalf("sub เบอร์ซ้ำต้องสร้างได้: %v", err)
		}
	}
}

func TestCreatePTSettings(t *testing.T) { // MGMT-16, MGMT-19, MGMT-20, MGMT-22
	app := setup2(t)
	c := buildChain(t, app)

	var rows []models.AgentGameSetting
	database.DBConn.Where("agent_id = ?", c.share.ID).Find(&rows)
	if len(rows) != 3 {
		t.Fatalf("game settings = %d rows, want 3", len(rows))
	}
	for _, r := range rows {
		if r.PTFromParent != 70 || r.PT != 70 || r.Commission != 0.5 || !r.Status || !r.StatusGame || r.Category != "minigame" {
			t.Fatalf("row %+v", r)
		}
	}

	b := agentBody("AGENT", "agoff", nil, childPT(10, 0, 0, 0))
	b["status_game"] = map[string]bool{"scratch_card": false}
	ag, _ := mustCreate(t, app, c.agentTok, b)
	var off models.AgentGameSetting
	database.DBConn.Where("agent_id = ? AND game_code = ?", ag.ID, "scratch_card").Take(&off)
	if off.StatusGame || !off.Status {
		t.Fatalf("scratch_card %+v", off)
	}

	// Company Seamless Master: ถือ 0 · ให้ Share Master เท่าที่ได้รับเท่านั้น
	master, masterTok := mustCreate(t, app, c.saTok, agentBody("COMPANY_SEAMLESS_MASTER", "master01", nil, childPT(80, 0, 0, 0)))
	var ms models.AgentGameSetting
	database.DBConn.Where("agent_id = ?", master.ID).Take(&ms)
	if ms.PTFromParent != 80 || ms.PT != 0 {
		t.Fatalf("master %+v", ms)
	}
	expect(t, call(t, app, "POST", createAgentPath, agentBody("SHARE_B2C", "sharemas", []string{"THB"}, childPT(75, 0, 0, 0)), masterTok), 200, 402307)
	expect(t, call(t, app, "POST", createAgentPath, agentBody("SHARE_B2C", "sharemas", []string{"THB"}, childPT(80, 5, 0, 0)), masterTok), 200, 402307)
	r := call(t, app, "POST", createAgentPath, agentBody("SHARE_B2C", "sharemas", []string{"THB"}, childPT(80, 0, 0, 0.2)), masterTok)
	expect(t, r, 200, 200)
}

func TestCreateInitialBalance(t *testing.T) { // MGMT-15, MGMT-15A
	app := setup2(t)
	c := buildChain(t, app)
	setBalance(t, c.com.ID, "THB", 50000) // 50,000.00

	b := agentBody("SHARE_B2C", "sharebal", []string{"THB"}, childPT(50, 0, 0, 0))
	b["balance"] = map[string]any{"THB": 10000}
	r := call(t, app, "POST", createAgentPath, b, c.comTok)
	expect(t, r, 200, 200)
	var d created
	_ = json.Unmarshal(r.Data, &d)
	if got := agentBalance(t, c.com.ID, "THB"); got != 40000 {
		t.Fatalf("company THB = %v, want 40000", got)
	}
	if got := agentBalance(t, d.ID, "THB"); got != 10000 {
		t.Fatalf("share THB = %v, want 10000", got)
	}
	if n := countRows(t, &models.BalanceLedger{}, "request_id = ?", b["request_id"]); n != 2 {
		t.Fatalf("ledger rows = %d, want 2", n)
	}

	// ส่งซ้ำด้วย request_id เดิม = ได้ id เดิม ไม่โอนซ้ำ
	r2 := call(t, app, "POST", createAgentPath, b, c.comTok)
	expect(t, r2, 200, 200)
	var d2 created
	_ = json.Unmarshal(r2.Data, &d2)
	if d2.ID != d.ID || agentBalance(t, c.com.ID, "THB") != 40000 {
		t.Fatalf("replay id %v (want %v) company %v", d2.ID, d.ID, agentBalance(t, c.com.ID, "THB"))
	}

	// ยอดไม่พอ
	b = agentBody("SHARE_B2C", "sharepoor", []string{"THB"}, childPT(50, 0, 0, 0))
	b["balance"] = map[string]any{"THB": 40000.01}
	expect(t, call(t, app, "POST", createAgentPath, b, c.comTok), 200, 402312)
	if n := countRows(t, &models.UserAgent{}, "username = ?", "sharepoor"); n != 0 || agentBalance(t, c.com.ID, "THB") != 40000 {
		t.Fatal("ยอดไม่พอต้องไม่สร้างบัญชีและไม่แตะยอด")
	}

	// Superadmin วงเงินไม่จำกัด · ledger ฝั่งที่ได้รับแถวเดียว
	b = agentBody("COMPANY_TRANSFER", "comrich", nil, childPT(90, 0, 0, 0))
	b["balance"] = map[string]any{"THB": 1000000}
	r = call(t, app, "POST", createAgentPath, b, c.saTok)
	expect(t, r, 200, 200)
	_ = json.Unmarshal(r.Data, &d)
	if agentBalance(t, d.ID, "THB") != 1000000 || countRows(t, &models.BalanceLedger{}, "request_id = ?", b["request_id"]) != 1 {
		t.Fatal("superadmin initial balance")
	}
	if countRows(t, &models.BalanceLedger{}, "request_id = ? AND reason = ?", b["request_id"], models.LedgerInitialFromSuperadmin) != 1 {
		t.Fatal("reason ต้องเป็น INITIAL_FROM_SUPERADMIN")
	}

	// 422: Seamless · สกุลที่บัญชีใหม่ไม่มี · 0 · ติดลบ
	bad := []map[string]any{{"THB": 0}, {"THB": -5}, {"USD": 10}, {"THB": 1.00001}}
	for _, bal := range bad {
		b = agentBody("SHARE_B2C", "sharebad", []string{"THB"}, childPT(50, 0, 0, 0))
		b["balance"] = bal
		expect(t, call(t, app, "POST", createAgentPath, b, c.comTok), 200, 422)
	}
	b = agentBody("COMPANY_SEAMLESS_RESELLER", "resbal", nil, childPT(50, 0, 0, 0))
	b["balance"] = map[string]any{"THB": 10}
	expect(t, call(t, app, "POST", createAgentPath, b, c.saTok), 200, 422)

	// Member ได้ยอดเงินตั้งต้นจาก Agent
	setBalance(t, c.agent.ID, "THB", 100)
	mb := memberBody("memrich", 0)
	mb["balance"] = map[string]any{"THB": 25.5}
	expect(t, call(t, app, "POST", createMemberPath, mb, c.agentTok), 200, 200)
	var mbal models.UserMember // ยอดของ Member = user_members.credit
	database.DBConn.Select("id", "credit", "cnf").Where("username = ?", "memrich").Take(&mbal)
	if mbal.Credit != 25.5 || agentBalance(t, c.agent.ID, "THB") != 74.5 {
		t.Fatalf("member %v agent %v", mbal.Credit, agentBalance(t, c.agent.ID, "THB"))
	}
	// cnf ของ Member = สายของผู้สร้าง + ผู้สร้าง: superadmin → comp01 → share01 → agent01
	var chain struct {
		Parent []struct {
			ID       uint   `json:"id"`
			Position string `json:"position"`
		} `json:"parent"`
	}
	_ = json.Unmarshal([]byte(mbal.Cnf), &chain)
	if n := len(chain.Parent); n != 4 || chain.Parent[0].Position != "superadmin" || chain.Parent[1].ID != c.com.ID ||
		chain.Parent[2].ID != c.share.ID || chain.Parent[3].ID != c.agent.ID || chain.Parent[3].Position != "agent" {
		t.Fatalf("member cnf %s", mbal.Cnf)
	}
}

func TestCreateConcurrentBalance(t *testing.T) { // MGMT-15A: ยอดพอแค่คำขอเดียว
	app := setup2(t)
	c := buildChain(t, app)
	setBalance(t, c.com.ID, "THB", 10000)

	codes := make([]int, 2)
	var wg sync.WaitGroup
	for i := range codes {
		b := agentBody("SHARE_B2C", fmt.Sprintf("race%d", i), []string{"THB"}, childPT(50, 0, 0, 0))
		b["balance"] = map[string]any{"THB": 8000}
		raw, _ := json.Marshal(b)
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := httptest.NewRequest("POST", createAgentPath, bytes.NewReader(raw))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+c.comTok)
			resp, err := app.Test(req, -1)
			if err != nil {
				return
			}
			defer func() { _ = resp.Body.Close() }()
			var out testutil.Response
			_ = json.NewDecoder(resp.Body).Decode(&out)
			codes[i] = out.Code
		}(i)
	}
	wg.Wait()
	ok, poor := 0, 0
	for _, code := range codes {
		switch code {
		case 200:
			ok++
		case 402312:
			poor++
		}
	}
	if ok != 1 || poor != 1 || agentBalance(t, c.com.ID, "THB") != 2000 {
		t.Fatalf("codes %v balance %v", codes, agentBalance(t, c.com.ID, "THB"))
	}
}

func TestCreateSubPermission(t *testing.T) { // MGMT-51
	app := setup2(t)
	c := buildChain(t, app)
	var com models.UserAgent
	database.DBConn.Where("id = ?", c.com.ID).Take(&com)
	hash, _ := utils.HashPassword(mgPassword)
	sub := models.Subaccount{AgentID: com.ID, Username: com.Username + "@staff", PasswordHash: hash, Status: models.AgentStatusActive,
		Permissions: `{"member":"edit"}`}
	if err := agentAuthPostgres.CreateSubaccountRepository(database.DBConn, &sub); err != nil {
		t.Fatal(err)
	}
	tok := readyToken(t, app, models.AccountTypeSub, sub.ID, sub.Username)
	body := agentBody("SHARE_B2C", "subshare", []string{"THB"}, childPT(50, 0, 0, 0))
	expect(t, call(t, app, "POST", createAgentPath, body, tok), 200, 402303)

	setCols(t, models.AccountTypeSub, sub.ID, map[string]any{"permissions": `{"member":"edit","pt":"edit"}`})
	r := call(t, app, "POST", createAgentPath, body, tok)
	expect(t, r, 200, 200)
	var d created
	_ = json.Unmarshal(r.Data, &d)
	var a models.UserAgent
	database.DBConn.Where("id = ?", d.ID).Take(&a)
	if a.ParentID == nil || *a.ParentID != com.ID {
		t.Fatal("บัญชีที่ sub สร้างต้องอยู่ใต้เจ้าของ")
	}

	// ส่ง balance ต้องมี payment = edit
	setBalance(t, com.ID, "THB", 1000)
	b := agentBody("SHARE_B2C", "subbal", []string{"THB"}, childPT(50, 0, 0, 0))
	b["balance"] = map[string]any{"THB": 10}
	expect(t, call(t, app, "POST", createAgentPath, b, tok), 200, 402303)
}

func TestCreateChangeLog(t *testing.T) { // MGMT-60
	app := setup2(t)
	c := buildChain(t, app)
	var logs []models.AccountChangeLog
	database.DBConn.Where("target_id = ? AND target_type = ?", c.share.ID, "AGENT").Find(&logs)
	if len(logs) != 1 || logs[0].Action != models.ChangeCreate || logs[0].ActorUsername != "comp01" || logs[0].NewValue == nil {
		t.Fatalf("logs %+v", logs)
	}
	if strings.Contains(*logs[0].NewValue, mgPassword) || strings.Contains(*logs[0].NewValue, "password") {
		t.Fatal("log ห้ามมีรหัสผ่าน")
	}
}
