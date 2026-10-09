//go:build integration

// API test ของ Profile ตามตาราง test case ใน docs/modules/account.md หัวข้อ 7 (ACC-11 – ACC-19, ACC-30, ACC-32)
// บัญชีสร้างผ่านเส้นของ module ② (helper ใน agent_management_integration_test.go)
package backoffice_test

import (
	"encoding/json"
	"testing"
	"time"

	"app/app/models"
	agentAuthPostgres "app/app/repository/postgres/agent_auth"
	"app/pkg/utils"
	"app/platform/database"

	"github.com/gofiber/fiber/v2"
)

const profilePath = "/api/v1/bo/pr/account/profile"

type profilePT struct {
	PTFromParent      json.Number `json:"pt_from_parent"`
	PT                json.Number `json:"pt"` // ต้องไม่มี (ลบ 2026-10-09) — ถ้ามีจะไม่ว่าง
	Force             json.Number `json:"force"`
	RemainQuota       json.Number `json:"remain_quota"`
	CommissionPercent json.Number `json:"commission_percent"`
	Status            bool        `json:"status"`
	CreatedAt         string      `json:"created_at"`
	CreatedBy         string      `json:"created_by"`
	UpdatedAt         string      `json:"updated_at"`
	UpdatedBy         string      `json:"updated_by"`
}

type profileData struct {
	Username      string               `json:"username"`
	Role          string               `json:"role"`
	UserType      string               `json:"user_type"`
	Status        string               `json:"status"`
	IsSubaccount  bool                 `json:"is_subaccount"`
	OwnerUsername string               `json:"owner_username"`
	PasscodeSet   bool                 `json:"passcode_set"`
	LastLoginAt   string               `json:"last_login_at"`
	LastLoginIP   string               `json:"last_login_ip"`
	CreatedAt     string               `json:"created_at"`
	Currencies    []string             `json:"currencies"`
	Balances      []json.RawMessage    `json:"balances"`
	PT            map[string]profilePT `json:"pt"`
	StatusGame    map[string]bool      `json:"status_game"`
	Permissions   map[string]string    `json:"permissions"`
}

func getProfile(t *testing.T, app *fiber.App, tok string) (profileData, map[string]json.RawMessage) {
	t.Helper()
	r := call(t, app, "GET", profilePath, nil, tok)
	expect(t, r, 200, 200)
	var d profileData
	if err := json.Unmarshal(r.Data, &d); err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(r.Data, &raw); err != nil {
		t.Fatal(err)
	}
	for k, v := range raw { // ACC-32
		if string(v) == "null" {
			t.Fatalf("field %s เป็น null", k)
		}
	}
	if _, ok := raw["effective_status"]; ok {
		t.Fatal("ต้องไม่มี field effective_status (ACC-30)")
	}
	return d, raw
}

func allLevel(t *testing.T, perms map[string]string, want string) {
	t.Helper()
	if len(perms) != 8 {
		t.Fatalf("ต้องมีสิทธิ์ 8 เมนู ได้ %v", perms)
	}
	for m, lv := range perms {
		if lv != want {
			t.Fatalf("เมนู %s = %s, want %s", m, lv, want)
		}
	}
}

func TestProfileOwnAccount(t *testing.T) { // ACC-11, ACC-12, ACC-14, ACC-16, ACC-19
	app := setup2(t)
	c := buildChain(t, app) // Superadmin → comp01 (90) → share01 (70, THB) → agent01 (60)

	// SUPERADMIN: ได้รับ 100 · ไม่มี pt ของบัญชี (ลบ 2026-10-09) · ครบ 27 สกุล · มี rate
	d, _ := getProfile(t, app, c.saTok)
	if d.UserType != "SUPERADMIN" || len(d.Currencies) != 27 || len(d.Balances) != 27 ||
		d.PT["minigame"].PTFromParent != "100" || d.PT["minigame"].PT != "" || d.Permissions["rate"] != "edit" {
		t.Fatalf("superadmin %+v", d)
	}
	allLevel(t, d.Permissions, "edit")

	// Company Transfer: ประเภทย่อย · ค่าที่ Superadmin ให้ · ไม่มี pt ของบัญชี
	setBalance(t, c.com.ID, "THB", 962056)
	d, _ = getProfile(t, app, c.comTok)
	if d.Role != "COMPANY" || d.UserType != "COMPANY_TRANSFER" || len(d.Currencies) != 27 || d.IsSubaccount || d.OwnerUsername != "" {
		t.Fatalf("company %+v", d)
	}
	g := d.PT["minigame"]
	if g.PTFromParent != "90" || g.PT != "" || g.CommissionPercent != "0.5" || g.Force != "0" || !g.Status ||
		g.CreatedBy == "" || g.CreatedBy != g.UpdatedBy || g.CreatedAt == "" || g.CreatedAt != g.UpdatedAt {
		t.Fatalf("company pt %+v", g)
	}
	if len(d.StatusGame) != 3 || !d.StatusGame["scratch_card"] {
		t.Fatalf("status_game %v", d.StatusGame)
	}
	var thb string
	for _, b := range d.Balances {
		var x struct {
			Currency string          `json:"currency"`
			Amount   json.RawMessage `json:"amount"`
		}
		_ = json.Unmarshal(b, &x)
		if x.Currency == "THB" {
			thb = string(x.Amount)
		}
	}
	if thb != "962056" {
		t.Fatalf("THB amount = %s, want 962056", thb)
	}
	at, err := time.Parse(time.RFC3339Nano, d.LastLoginAt)
	if err != nil || time.Since(at) > time.Minute || d.LastLoginIP == "" || d.CreatedAt == "" {
		t.Fatalf("login ล่าสุด / วันที่สร้างต้องมีค่า %+v", d)
	}

	// Share B2C: 1 สกุล · ยอดสกุลที่ยังไม่มี = 0.00
	d, _ = getProfile(t, app, c.shareTok)
	if d.UserType != "SHARE_B2C" || len(d.Currencies) != 1 || d.Currencies[0] != "THB" ||
		len(d.Balances) != 1 || string(d.Balances[0]) != `{"currency":"THB","amount":0}` || d.PT["minigame"].PTFromParent != "70" {
		t.Fatalf("share %+v %s", d, d.Balances)
	}

	// Agent ที่ถูกปิด scratch_card (ACC-16)
	b := agentBody("AGENT", "agoff", nil, childPT(10, 0, 0, 0))
	b["status_game"] = map[string]bool{"scratch_card": false}
	_, offTok := mustCreate(t, app, c.agentTok, b)
	d, _ = getProfile(t, app, offTok)
	if d.UserType != "AGENT" || d.StatusGame["scratch_card"] || !d.StatusGame["coin_toss"] || len(d.PT) != 1 {
		t.Fatalf("agent off %+v", d)
	}
	allLevel(t, d.Permissions, "edit")

	// Company Seamless Master: ไม่มี pt · Seamless 1 to 1: ยอด 0 (ACC-12, ACC-19)
	_, masterTok := mustCreate(t, app, c.saTok, agentBody("COMPANY_SEAMLESS_MASTER", "master01", nil, childPT(80, 0, 0, 0)))
	d, _ = getProfile(t, app, masterTok)
	if d.UserType != "COMPANY_SEAMLESS_MASTER" || d.PT["minigame"].PT != "" || d.PT["minigame"].PTFromParent != "80" {
		t.Fatalf("master %+v", d)
	}
	_, oneTok := mustCreate(t, app, c.saTok, agentBody("COMPANY_SEAMLESS_1TO1", "one2one", []string{"JPY"}, childPT(80, 0, 0, 0)))
	d, _ = getProfile(t, app, oneTok)
	if d.UserType != "COMPANY_SEAMLESS_1TO1" || len(d.Balances) != 1 || string(d.Balances[0]) != `{"currency":"JPY","amount":0}` {
		t.Fatalf("1to1 %+v %s", d, d.Balances)
	}
}

func TestProfileAdmin(t *testing.T) { // ACC-12, ACC-16
	app := setup2(t)
	admin := createAccount(t, "support01", models.AgentRoleAdmin, nil)
	tok := loginReady(t, app, models.AccountTypeAgent, admin.ID, "support01")
	_, raw := getProfile(t, app, tok)
	for k, want := range map[string]string{"user_type": `"ADMIN"`, "currencies": "[]", "balances": "[]", "pt": "{}", "status_game": "{}", "permissions": "{}"} {
		if string(raw[k]) != want {
			t.Fatalf("ADMIN %s = %s, want %s", k, raw[k], want)
		}
	}
}

func TestProfileSubaccount(t *testing.T) { // ACC-11, ACC-12, ACC-15, ACC-30
	app := setup2(t)
	c := buildChain(t, app)
	hash, _ := utils.HashPassword(mgPassword)
	sub := models.Subaccount{AgentID: c.share.ID, Username: "share01@staff", PasswordHash: hash, Status: models.AgentStatusActive,
		Permissions: `{"report":"view"}`}
	if err := agentAuthPostgres.CreateSubaccountRepository(database.DBConn, &sub); err != nil {
		t.Fatal(err)
	}
	tok := readyToken(t, app, models.AccountTypeSub, sub.ID, sub.Username)

	d, _ := getProfile(t, app, tok)
	if d.Username != "share01@staff" || !d.IsSubaccount || d.OwnerUsername != "share01" || d.Role != "SHAREHOLDER" ||
		d.UserType != "SHARE_B2C" || len(d.Currencies) != 1 || d.PT["minigame"].PTFromParent != "70" {
		t.Fatalf("unexpected %+v", d)
	}
	if len(d.Permissions) != 8 || d.Permissions["report"] != "view" || d.Permissions["member"] != "off" || d.Permissions["account"] != "" {
		t.Fatalf("permissions %v", d.Permissions)
	}

	// ผู้สร้างถูกระงับ → status ของ sub = สถานะที่ใช้งานจริง และยังเปิด Profile ได้ (ACC-31 / AUTH-54)
	setStatus(t, c.share.ID, models.AgentStatusSuspended)
	d, _ = getProfile(t, app, tok)
	if d.Status != "SUSPENDED" {
		t.Fatalf("ผู้สร้าง SUSPENDED: status ของ sub ต้องเป็น SUSPENDED ได้ %+v", d)
	}
	// sub ได้ report = view ไว้ → ยังเป็น view · เมนูอื่น off · เจ้าของ (บัญชีหลัก) เหลือ report = view (ACC-12 · AUTH-54)
	for m, lv := range d.Permissions {
		if m != "report" && lv != "off" || m == "report" && lv != "view" {
			t.Fatalf("sub ตอนถูกระงับ: %s = %s, want off", m, lv)
		}
	}
	d, _ = getProfile(t, app, c.shareTok)
	if d.Status != "SUSPENDED" || d.Permissions["report"] != "view" || d.Permissions["member"] != "off" {
		t.Fatalf("บัญชีหลักถูกระงับ %+v", d.Permissions)
	}
	d, _ = getProfile(t, app, c.agentTok) // ถูกระงับจากหัวสาย
	if d.Status != "SUSPENDED" || d.Permissions["report"] != "view" || d.Permissions["pt"] != "off" {
		t.Fatalf("ลูกของบัญชีที่ถูกระงับ %+v", d.Permissions)
	}
}

func TestProfileRequiresGates(t *testing.T) { // ACC-11
	app := setup2(t)
	createAgent(t, "agent01", models.AgentStatusActive, nil)
	tok := loginToken(t, app, "agent01") // ยังไม่ตั้ง passcode
	expect(t, call(t, app, "GET", profilePath, nil, tok), 403, 401304)
	expect(t, call(t, app, "GET", profilePath, nil, ""), 401, 401202)
}
