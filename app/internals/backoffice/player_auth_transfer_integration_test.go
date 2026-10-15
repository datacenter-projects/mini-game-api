//go:build integration

// API test ของ player_auth สาย operator (transfer/login · transfer/register) ตามตาราง test case ใน docs/modules/player_auth.md หัวข้อ 7
package backoffice_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"app/app/models"
	"app/pkg/configs"
	"app/pkg/testutil"
	"app/platform/database"

	"github.com/gofiber/fiber/v2"
)

const (
	trLoginPath    = "/api/v1/transfer/login"
	trRegisterPath = "/api/v1/transfer/register"
	trTestIP       = "0.0.0.0/32" // IP ของ request ใน app.Test
)

// opKeyOf — Key ของ agent ใน test (hex 64 ตัว · ไม่ซ้ำต่อ agent)
func opKeyOf(agentID uint) string { return fmt.Sprintf("%064x", 1000+agentID) }

// opSeed — ให้ agent มี Key + IP whitelist (ไม่ส่ง cidrs = ปล่อย IP ของ test)
func opSeed(t *testing.T, agentID uint, cidrs ...string) {
	t.Helper()
	if err := database.DBConn.Save(&models.APICredential{AgentID: agentID, APIKey: opKeyOf(agentID)}).Error; err != nil { // agent ทุกคนมีแถว Key อยู่แล้ว (ACC-05) → แทนค่า
		t.Fatal(err)
	}
	if cidrs == nil {
		cidrs = []string{trTestIP}
	}
	for _, c := range cidrs {
		if err := database.DBConn.Create(&models.APIAllowedIP{AgentID: agentID, CIDR: c}).Error; err != nil {
			t.Fatal(err)
		}
	}
}

// trEnv — plEnv + Key / whitelist ของ agent01 (AGENT) และ share01 (SHARE_B2C) · operator หลักคือ agent01
type trEnv struct {
	plEnv
}

func trSetup(t *testing.T) trEnv {
	t.Helper()
	e := plSetup(t)
	opSeed(t, e.c.agent.ID)
	opSeed(t, e.c.share.ID)
	return trEnv{e}
}

func (e trEnv) body(player string, extra map[string]any) map[string]any {
	b := map[string]any{"agent_username": "agent01", "key": opKeyOf(e.c.agent.ID), "player_username": player}
	for k, v := range extra {
		b[k] = v
	}
	return b
}

func (e trEnv) login(t *testing.T, player string, extra map[string]any) testutil.Response {
	t.Helper()
	return call(t, e.app, "POST", trLoginPath, e.body(player, extra), "")
}

func (e trEnv) register(t *testing.T, player string, extra map[string]any) testutil.Response {
	t.Helper()
	return call(t, e.app, "POST", trRegisterPath, e.body(player, extra), "")
}

type trLoginData struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
	Link      string `json:"link"`
}

func trLoginOK(t *testing.T, e trEnv, player string, extra map[string]any) trLoginData {
	t.Helper()
	r := e.login(t, player, extra)
	expect(t, r, 200, 200)
	var d trLoginData
	if err := json.Unmarshal(r.Data, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

func memberByUsername(t *testing.T, username string) (models.UserMember, bool) {
	t.Helper()
	var m models.UserMember
	err := database.DBConn.Where("username = ?", username).Take(&m).Error
	return m, err == nil
}

func countMembers(t *testing.T, username string) int64 {
	t.Helper()
	return countRows(t, &models.UserMember{}, "username = ?", username)
}

// callQuiet — เหมือน testutil.Call แต่ไม่เรียก t.Fatal (ใช้ใน goroutine) คืน HTTP status + code
func callQuiet(app *fiber.App, path string, body any) (int, int) {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		return -1, -1
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var out struct {
		Code int `json:"code"`
	}
	_ = json.Unmarshal(raw, &out)
	return resp.StatusCode, out.Code
}

func TestTransferLoginCreatesPlayer(t *testing.T) { // PA-07, PA-09, LQ-c
	e := trSetup(t)
	d := trLoginOK(t, e, "p001", nil)
	if d.Token == "" || d.ExpiresAt == "" {
		t.Fatalf("ต้องได้ token + expires_at: %+v", d)
	}
	if want := "https://lobby.test/?token=" + d.Token + "&lang=th"; d.Link != want { // token ใน URL · lang default th
		t.Errorf("link = %q want %q", d.Link, want)
	}

	m, ok := memberByUsername(t, "agent01@p001") // PA-05: {agent}@{player}
	if !ok {
		t.Fatal("ต้องสร้างผู้เล่น agent01@p001")
	}
	if m.AgentID != e.c.agent.ID || m.Name != "p001" || m.Currency != "THB" || m.Status != models.AgentStatusActive { // PA-09
		t.Errorf("member = %+v", m)
	}
	if m.PasswordHash != "" || m.Credit != 0 || m.PaymentType != models.PaymentTypeCash || m.WinLimitPerDay != 10_000_000 {
		t.Errorf("ค่าตั้งต้นผิด: hash %q credit %v payment %v win %v", m.PasswordHash, m.Credit, m.PaymentType, m.WinLimitPerDay)
	}
	if m.LastLoginAt == nil {
		t.Error("ต้องบันทึก last_login_at")
	}

	var settings []models.UserMemberGameSetting // pt = ค่าเต็มที่ agent ได้รับ (60) · remain 0 · commission 0
	database.DBConn.Where("user_member_id = ?", m.ID).Find(&settings)
	if len(settings) != 3 {
		t.Fatalf("game settings = %d want 3", len(settings))
	}
	for _, s := range settings {
		if s.PT != 60 || s.Remain != 0 || s.Commission != 0 || s.ParentID != e.c.agent.ID {
			t.Errorf("%s: pt %v remain %v commission %v parent %d", s.GameCode, s.PT, s.Remain, s.Commission, s.ParentID)
		}
	}

	r := call(t, e.app, "POST", plMePath, nil, d.Token) // token ใช้กับเส้นผู้เล่นได้
	expect(t, r, 200, 200)
	if me := plData(t, r); me["username"] != "agent01@p001" || me["status"] != "ACTIVE" || me["can_bet"] != true {
		t.Errorf("me = %v", me)
	}
	if _, has := plData(t, e.login(t, "p001", nil))["created"]; has { // ไม่บอกว่าสร้างใหม่หรือเดิม
		t.Error("response ต้องไม่มี created")
	}
}

func TestTransferLoginExistingPlayer(t *testing.T) { // PA-07, PA-22
	e := trSetup(t)
	a := trLoginOK(t, e, "p001", nil)
	b := trLoginOK(t, e, "P001", nil) // รับตัวใหญ่ เก็บตัวเล็ก
	if n := countMembers(t, "agent01@p001"); n != 1 {
		t.Fatalf("ต้องไม่สร้างซ้ำ: %d แถว", n)
	}
	expect(t, call(t, e.app, "POST", plMePath, nil, a.Token), 401, 407204) // ครั้งที่ 2 เตะ session แรก
	expect(t, call(t, e.app, "POST", plMePath, nil, b.Token), 200, 200)
}

func TestTransferLoginValidation(t *testing.T) { // PA-05, L2 language
	e := trSetup(t)
	for _, bad := range []string{"AB", "abc!", strings.Repeat("a", 33), "ab cd", "a@bc", ""} {
		expect(t, e.login(t, bad, nil), 200, 422)
	}
	if _, ok := memberByUsername(t, "agent01@abc"); ok {
		t.Fatal("ยังไม่ควรมี agent01@abc")
	}
	trLoginOK(t, e, "ABC", nil)
	if _, ok := memberByUsername(t, "agent01@abc"); !ok {
		t.Error("ABC ต้องเก็บเป็น agent01@abc")
	}

	if d := trLoginOK(t, e, "p001", map[string]any{"language": "en"}); !strings.HasSuffix(d.Link, "&lang=en") {
		t.Errorf("language en: %q", d.Link)
	}
	if d := trLoginOK(t, e, "p001", map[string]any{"language": "TH"}); !strings.HasSuffix(d.Link, "&lang=th") {
		t.Errorf("language TH: %q", d.Link)
	}
	expect(t, e.login(t, "p001", map[string]any{"language": "fr"}), 200, 422)
	expect(t, e.login(t, "p001", map[string]any{"currency": nil}), 200, 422) // ห้าม null
}

func TestOperatorAuthRejects(t *testing.T) { // PA-10, PA-11, PA-14
	e := trSetup(t)
	post := func(b map[string]any) testutil.Response { return call(t, e.app, "POST", trLoginPath, b, "") }
	good := e.body("p001", nil)

	bad := func(k string, v any) map[string]any {
		b := map[string]any{}
		for kk, vv := range good {
			b[kk] = vv
		}
		b[k] = v
		return b
	}
	expect(t, post(bad("key", strings.Repeat("0", 64))), 401, 407206) // key ผิด
	expect(t, post(bad("agent_username", "ghost")), 401, 407206)      // ไม่มี agent
	expect(t, post(bad("agent_username", "Agent01 ")), 200, 200)      // agent_username trim + ตัวเล็ก
	expect(t, post(bad("agent_username", "")), 200, 422)
	expect(t, post(bad("key", "")), 200, 422)

	r := post(bad("key", "secret-key-should-not-echo"))
	if strings.Contains(r.Msg, "secret-key-should-not-echo") || strings.Contains(string(r.Data), "secret-key-should-not-echo") { // PA-12
		t.Error("error ต้องไม่สะท้อน key กลับ")
	}
	// ข้อความเดียวกันทั้ง agent ไม่มี / key ผิด (ไม่บอกว่าผิดที่ไหน)
	if a, b := post(bad("agent_username", "ghost")), post(bad("key", strings.Repeat("0", 64))); a.Msg != b.Msg {
		t.Errorf("ข้อความต่างกัน: %q vs %q", a.Msg, b.Msg)
	}

	// agent ที่ไม่มี Key เลย
	other, _ := mustCreate(t, e.app, e.c.shareTok, agentBody("AGENT", "agent02", nil, childPT(60, 0, 0, 0.5)))
	expect(t, post(map[string]any{"agent_username": "agent02", "key": opKeyOf(other.ID), "player_username": "p001"}), 401, 407206)

	// PA-14: Company Transfer / Share B2B ใช้ไม่ได้ · Share B2C ใช้ได้
	opSeed(t, e.c.com.ID)
	expect(t, post(map[string]any{"agent_username": "comp01", "key": opKeyOf(e.c.com.ID), "player_username": "p001"}), 403, 407306)
	b2b, _ := mustCreate(t, e.app, e.c.comTok, agentBody("SHARE_B2B", "shareb2b", []string{"THB"}, childPT(70, 0, 0, 0.5)))
	opSeed(t, b2b.ID)
	expect(t, post(map[string]any{"agent_username": "shareb2b", "key": opKeyOf(b2b.ID), "player_username": "p001"}), 403, 407306)
	expect(t, post(map[string]any{"agent_username": "share01", "key": opKeyOf(e.c.share.ID), "player_username": "p002"}), 200, 200)
	if _, ok := memberByUsername(t, "share01@p002"); !ok {
		t.Error("Share B2C ต้องสร้างผู้เล่น share01@p002 ได้")
	}
}

func TestOperatorIPWhitelist(t *testing.T) { // PA-10, PA-11 · ACC-07
	e := plSetup(t)
	opSeed(t, e.c.agent.ID, "10.0.0.1/32", "198.51.100.0/24") // ไม่รวม IP ของ test
	body := map[string]any{"agent_username": "agent01", "key": opKeyOf(e.c.agent.ID), "player_username": "p001"}
	expect(t, call(t, e.app, "POST", trLoginPath, body, ""), 403, 407305)
	expect(t, call(t, e.app, "POST", trRegisterPath, body, ""), 403, 407305)
	if _, ok := memberByUsername(t, "agent01@p001"); ok {
		t.Error("IP ไม่ผ่านต้องไม่สร้างผู้เล่น")
	}

	// whitelist ว่าง = ปฏิเสธทุก IP
	if err := database.DBConn.Where("agent_id = ?", e.c.agent.ID).Delete(&models.APIAllowedIP{}).Error; err != nil {
		t.Fatal(err)
	}
	expect(t, call(t, e.app, "POST", trLoginPath, body, ""), 403, 407305)

	// ใส่ IP ของ test แล้วผ่าน · key ผิดต้องได้ 407206 ก่อน (ไม่เปิดเผย whitelist ให้คนไม่มี key)
	if err := database.DBConn.Create(&models.APIAllowedIP{AgentID: e.c.agent.ID, CIDR: trTestIP}).Error; err != nil {
		t.Fatal(err)
	}
	expect(t, call(t, e.app, "POST", trLoginPath, body, ""), 200, 200)
}

func TestTransferCurrency(t *testing.T) { // PA-08A
	e := trSetup(t)
	expect(t, e.login(t, "p001", map[string]any{"currency": "USD"}), 200, 422) // ไม่ใช่สกุลของ agent
	if _, ok := memberByUsername(t, "agent01@p001"); ok {
		t.Fatal("currency ผิดต้องไม่สร้างผู้เล่น")
	}
	trLoginOK(t, e, "p001", map[string]any{"currency": "thb"}) // ตัวเล็กได้
	if m, _ := memberByUsername(t, "agent01@p001"); m.Currency != "THB" {
		t.Errorf("currency = %q", m.Currency)
	}
	r := e.login(t, "p001", map[string]any{"currency": "USD"}) // ผู้เล่นเดิม สกุลไม่ตรง
	expect(t, r, 200, 422)
	if !strings.Contains(r.Msg, "สกุลเงินไม่ตรงกับผู้เล่น") {
		t.Errorf("msg = %q", r.Msg)
	}
	trLoginOK(t, e, "p001", map[string]any{"currency": "THB"}) // ตรง = ผ่าน
	trLoginOK(t, e, "p001", nil)                               // ไม่ส่ง = ผ่าน
}

func TestTransferOperatorStatus(t *testing.T) { // PA-13
	e := trSetup(t)
	trLoginOK(t, e, "p001", nil) // ผู้เล่นเดิมไว้ทดสอบตอน SUSPENDED

	t.Run("agent SUSPENDED", func(t *testing.T) {
		plSetAgentStatus(t, e.c.agent.ID, models.AgentStatusSuspended)
		d := trLoginOK(t, e, "p001", nil) // ผู้เล่นเดิม login ได้ · can_bet = false
		if me := plData(t, call(t, e.app, "POST", plMePath, nil, d.Token)); me["status"] != "SUSPENDED" || me["can_bet"] != false {
			t.Errorf("me = %v", me)
		}
		expect(t, e.login(t, "newbie", nil), 403, 407308)    // ผู้เล่นใหม่ใน login สร้างไม่ได้
		expect(t, e.register(t, "newbie", nil), 403, 407308) // register ผู้เล่นใหม่สร้างไม่ได้
		if _, ok := memberByUsername(t, "agent01@newbie"); ok {
			t.Error("ต้องไม่สร้างผู้เล่น")
		}
		expect(t, e.register(t, "p001", nil), 200, 407401) // register ผู้เล่นที่มีอยู่ = มีแล้ว (ไม่ใช่ 407308)
		plSetAgentStatus(t, e.c.agent.ID, models.AgentStatusActive)
	})

	t.Run("agent LOCKED", func(t *testing.T) {
		plSetAgentStatus(t, e.c.agent.ID, models.AgentStatusLocked)
		expect(t, e.login(t, "p001", nil), 403, 407302)
		expect(t, e.register(t, "p009", nil), 403, 407302)
		plSetAgentStatus(t, e.c.agent.ID, models.AgentStatusActive)
	})

	t.Run("สายบน LOCKED", func(t *testing.T) {
		plSetAgentStatus(t, e.c.com.ID, models.AgentStatusLocked)
		expect(t, e.login(t, "p001", nil), 403, 407302)
		expect(t, e.register(t, "p009", nil), 403, 407302)
		plSetAgentStatus(t, e.c.com.ID, models.AgentStatusActive)
	})

	t.Run("ผู้เล่น LOCKED", func(t *testing.T) {
		m, _ := memberByUsername(t, "agent01@p001")
		plSetMemberStatus(t, m.ID, models.AgentStatusLocked)
		expect(t, e.login(t, "p001", nil), 403, 407301)
		plSetMemberStatus(t, m.ID, models.AgentStatusActive)
		trLoginOK(t, e, "p001", nil)
	})
}

func TestTransferRegister(t *testing.T) { // PA-08
	e := trSetup(t)

	r := e.register(t, "p001", nil) // ไม่ส่ง password
	expect(t, r, 200, 200)
	if d := plData(t, r); d["username"] != "agent01@p001" || d["currency"] != "THB" || len(d) != 2 {
		t.Errorf("register data = %v (ต้องมีแค่ username + currency · ไม่ออก token)", d)
	}
	m, _ := memberByUsername(t, "agent01@p001")
	if m.PasswordHash != "" {
		t.Error("ไม่ส่ง password ต้องไม่มี hash")
	}
	expect(t, plLogin(t, e.app, "agent01@p001", mgPassword), 401, 407201) // เข้าทาง operator อย่างเดียว

	expect(t, e.register(t, "p001", nil), 200, 407401)                                    // ซ้ำ
	expect(t, e.register(t, "P001", map[string]any{"password": mgPassword}), 200, 407401) // ซ้ำแม้ตัวใหญ่ · ไม่ตั้งรหัสให้ผู้เล่นเดิม
	if m2, _ := memberByUsername(t, "agent01@p001"); m2.PasswordHash != "" {
		t.Error("register ซ้ำต้องไม่แก้ผู้เล่นเดิม")
	}

	// ส่ง password → login ตรงที่ PA1 ได้ด้วยชื่อเต็ม · ไม่คืนรหัส
	r = e.register(t, "p002", map[string]any{"password": mgPassword})
	expect(t, r, 200, 200)
	if strings.Contains(string(r.Data), mgPassword) || strings.Contains(r.Msg, mgPassword) {
		t.Error("ห้ามคืนรหัส")
	}
	m2, _ := memberByUsername(t, "agent01@p002")
	if m2.PasswordHash == "" || m2.PasswordHash == mgPassword {
		t.Errorf("hash ผิด: %q", m2.PasswordHash)
	}
	tok := plToken(t, e.app, "agent01@p002", mgPassword)
	expect(t, call(t, e.app, "POST", plMePath, nil, tok), 200, 200)

	// PA-51: ผู้เล่นที่ตั้งรหัสไว้เปลี่ยนรหัสได้ · ผู้เล่นที่ไม่มีรหัสเปลี่ยนไม่ได้
	expect(t, call(t, e.app, "POST", plPasswordPath, map[string]any{"old_password": mgPassword, "new_password": "Zx9!kq2m"}, tok), 200, 200)
	noPw := trLoginOK(t, e, "p001", nil)
	expect(t, call(t, e.app, "POST", plPasswordPath, map[string]any{"old_password": mgPassword, "new_password": "Zx9!kq2m"}, noPw.Token), 200, 407307)

	// รหัสไม่ผ่าน AUTH-36 / ยาวเกิน → 422 และไม่สร้างผู้เล่น
	expect(t, e.register(t, "p003", map[string]any{"password": "short1"}), 200, 422)
	expect(t, e.register(t, "p003", map[string]any{"password": strings.Repeat("aA1", 30)}), 200, 422)
	if _, ok := memberByUsername(t, "agent01@p003"); ok {
		t.Error("รหัสไม่ผ่านต้องไม่สร้างผู้เล่น")
	}
	expect(t, e.register(t, "p004", map[string]any{"password": ""}), 200, 200) // "" = ไม่ตั้งรหัส
}

func TestTransferConcurrentCreate(t *testing.T) { // PA-08B
	e := trSetup(t)
	const n = 8

	t.Run("login พร้อมกัน ไม่ error", func(t *testing.T) {
		var wg sync.WaitGroup
		codes := make([][2]int, n)
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				s, c := callQuiet(e.app, trLoginPath, e.body("racer", nil))
				codes[i] = [2]int{s, c}
			}(i)
		}
		wg.Wait()
		for i, c := range codes {
			if c != [2]int{200, 200} {
				t.Errorf("คำขอที่ %d ได้ HTTP %d code %d (ต้อง 200/200 ไม่มี 500)", i, c[0], c[1])
			}
		}
		if got := countMembers(t, "agent01@racer"); got != 1 {
			t.Errorf("แถวผู้เล่น = %d want 1", got)
		}
	})

	t.Run("register พร้อมกัน 1 สำเร็จ ที่เหลือ 407401", func(t *testing.T) {
		var wg sync.WaitGroup
		codes := make([][2]int, n)
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				s, c := callQuiet(e.app, trRegisterPath, e.body("racer2", nil))
				codes[i] = [2]int{s, c}
			}(i)
		}
		wg.Wait()
		ok, dup := 0, 0
		for i, c := range codes {
			switch c {
			case [2]int{200, 200}:
				ok++
			case [2]int{200, 407401}:
				dup++
			default:
				t.Errorf("คำขอที่ %d ได้ HTTP %d code %d", i, c[0], c[1])
			}
		}
		if ok != 1 || dup != n-1 {
			t.Errorf("สำเร็จ %d ซ้ำ %d want 1 / %d", ok, dup, n-1)
		}
		if got := countMembers(t, "agent01@racer2"); got != 1 {
			t.Errorf("แถวผู้เล่น = %d want 1", got)
		}
	})
}

func TestOperatorIPRateLimit(t *testing.T) { // PA-42
	e := trSetup(t)
	configs.Cfg.Auth.OperatorIPLimit = 3
	for i := 1; i <= 3; i++ {
		expect(t, e.login(t, "p001", nil), 200, 200)
	}
	expect(t, e.login(t, "p001", nil), 429, 429)
	expect(t, e.register(t, "p002", nil), 429, 429) // นับรวมทุกเส้นของ transfer/*
	// key ผิดก็นับ — ยิงเดา key ไม่ได้ไม่จำกัด
}

func TestTransferLobbyNotConfigured(t *testing.T) { // L2: PLAYER_LOBBY_URL ไม่ตั้ง → 503 ก่อนสร้างอะไร
	e := trSetup(t)
	configs.Cfg.PlayerLobbyURL = ""
	expect(t, e.login(t, "p001", nil), 503, 503)
	if _, ok := memberByUsername(t, "agent01@p001"); ok {
		t.Error("ระบบไม่พร้อมออก link ต้องไม่สร้างผู้เล่น")
	}
	expect(t, e.register(t, "p002", nil), 200, 200) // register ไม่ต้องใช้ lobby
}
