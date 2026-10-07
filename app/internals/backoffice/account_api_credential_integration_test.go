//go:build integration

// API test ของ 1.3 ข้อมูลรับรอง API ตามตาราง test case ใน docs/modules/account.md หัวข้อ 7 (ACC-01 – ACC-10, ACC-32)
// MGMT-04 (สร้าง Key พร้อมบัญชีเจ้าของ) ทำหลัง module ② และ account เข้า dev — ตอนนี้ Key สร้างตอนเปิดหน้าครั้งแรก (ACC-05)
package backoffice_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"app/app/models"
	"app/app/repository/postgres"
	"app/pkg/testutil"
	"app/pkg/utils"
	"app/platform/database"

	"github.com/gofiber/fiber/v2"
)

const (
	apiCredentialPath    = "/api/v1/bo/pr/account/api-credential"
	updateCredentialPath = "/api/v1/bo/pr/account/update-credential"
)

var keyRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

type apiCredentialData struct {
	Username    string   `json:"username"`
	Key         string   `json:"key"`
	CallbackURL string   `json:"callback_url"`
	AllowedIPs  []string `json:"allowed_ips"`
}

func getCredential(t *testing.T, app *fiber.App, tok string) (apiCredentialData, testutil.Response) {
	t.Helper()
	r := call(t, app, "GET", apiCredentialPath, nil, tok)
	expect(t, r, 200, 200)
	var d apiCredentialData
	if err := json.Unmarshal(r.Data, &d); err != nil {
		t.Fatal(err)
	}
	return d, r
}

func saveBody(callback any, ips any) map[string]any {
	b := map[string]any{"passcode": passcode}
	if callback != "-" {
		b["callback_url"] = callback
	}
	if ips != "-" {
		b["allowed_ips"] = ips
	}
	return b
}

// owners — Superadmin → 1 to 1 · Company Seamless Reseller → Share Reseller 2 บัญชี · Company Seamless Master → Share Master
type owners struct {
	chain
	oneTok, res1Tok, res2Tok, masterShareTok, resellerComTok string
	one, res1                                                created
}

func buildOwners(t *testing.T, app *fiber.App) owners {
	t.Helper()
	o := owners{chain: buildChain(t, app)}
	o.one, o.oneTok = mustCreate(t, app, o.saTok, agentBody("COMPANY_SEAMLESS_1TO1", "one2one", []string{"THB"}, childPT(80, 0, 0, 0)))
	_, o.resellerComTok = mustCreate(t, app, o.saTok, agentBody("COMPANY_SEAMLESS_RESELLER", "resellercom", nil, childPT(80, 0, 0, 0)))
	o.res1, o.res1Tok = mustCreate(t, app, o.resellerComTok, agentBody("SHARE_B2C", "shareres1", []string{"THB"}, childPT(50, 0, 0, 0)))
	_, o.res2Tok = mustCreate(t, app, o.resellerComTok, agentBody("SHARE_B2C", "shareres2", []string{"USD"}, childPT(50, 0, 0, 0)))
	_, masterTok := mustCreate(t, app, o.saTok, agentBody("COMPANY_SEAMLESS_MASTER", "mastercom", nil, childPT(80, 0, 0, 0)))
	_, o.masterShareTok = mustCreate(t, app, masterTok, agentBody("SHARE_B2C", "sharemas", []string{"THB"}, childPT(80, 0, 0, 0)))
	return o
}

func TestAPICredentialOwners(t *testing.T) { // ACC-01, ACC-03, ACC-04, ACC-10
	app := setup2(t)
	o := buildOwners(t, app)

	// ยังไม่มี MGMT-04: ก่อนเปิดหน้าครั้งแรกยังไม่มี Key (ACC-05)
	if n := countRows(t, &models.APICredential{}, "agent_id = ?", o.one.ID); n != 0 {
		t.Fatalf("ก่อนเปิดหน้าต้องยังไม่มี Key ได้ %d แถว", n)
	}

	d, r := getCredential(t, app, o.oneTok)
	if d.Username != "one2one" || !keyRe.MatchString(d.Key) || d.CallbackURL != "" || d.AllowedIPs == nil || len(d.AllowedIPs) != 0 {
		t.Fatalf("unexpected %+v", d)
	}
	if r.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", r.Header.Get("Cache-Control"))
	}
	again, _ := getCredential(t, app, o.oneTok)
	if again.Key != d.Key {
		t.Fatal("GET ซ้ำต้องได้ Key เดิม")
	}
	var cred models.APICredential
	database.DBConn.Where("agent_id = ?", o.one.ID).Take(&cred)
	if bytes.Contains(cred.KeyCiphertext, []byte(d.Key)) || cred.KeyHash != utils.HashAPIKey(d.Key) {
		t.Fatal("DB ต้องไม่มี Key แบบ plain และ key_hash = sha256(Key)")
	}

	k1, _ := getCredential(t, app, o.res1Tok)
	k2, _ := getCredential(t, app, o.res2Tok)
	km, _ := getCredential(t, app, o.masterShareTok)
	if k1.Key == k2.Key || k1.Key == d.Key || !keyRe.MatchString(km.Key) {
		t.Fatal("เจ้าของแต่ละบัญชีต้องได้ Key ของตัวเอง")
	}

	for name, tok := range map[string]string{"Superadmin": o.saTok, "Company Transfer": o.comTok, "Share B2C": o.shareTok,
		"Agent": o.agentTok, "Company Seamless Reseller": o.resellerComTok} {
		t.Run(name, func(t *testing.T) {
			expect(t, call(t, app, "GET", apiCredentialPath, nil, tok), 200, 403301)
			expect(t, call(t, app, "POST", updateCredentialPath, saveBody("", []string{}), tok), 200, 403301)
		})
	}
	if n := countRows(t, &models.APICredential{}, "1 = 1"); n != 4 {
		t.Fatalf("api_credentials = %d แถว, want 4 (เจ้าของเท่านั้น)", n)
	}
}

func TestAPICredentialFirstOpenConcurrent(t *testing.T) { // ACC-05: บัญชีที่ยังไม่มี Key · เปิดพร้อมกันได้ Key เดียว
	app := setup2(t)
	o := buildOwners(t, app)
	if err := database.DBConn.Where("agent_id = ?", o.one.ID).Delete(&models.APICredential{}).Error; err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 2)
	var wg sync.WaitGroup
	for i := range keys {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := httptest.NewRequest("GET", apiCredentialPath, nil)
			req.Header.Set("Authorization", "Bearer "+o.oneTok)
			resp, err := app.Test(req, -1)
			if err != nil {
				return
			}
			defer func() { _ = resp.Body.Close() }()
			var out struct {
				Data apiCredentialData `json:"data"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&out)
			keys[i] = out.Data.Key
		}(i)
	}
	wg.Wait()
	if keys[0] == "" || keys[0] != keys[1] || countRows(t, &models.APICredential{}, "agent_id = ?", o.one.ID) != 1 {
		t.Fatalf("keys %v", keys)
	}
}

func TestAPICredentialSave(t *testing.T) { // ACC-06 – ACC-09, ACC-32
	app := setup2(t)
	o := buildOwners(t, app)
	before, _ := getCredential(t, app, o.oneTok)

	tests := []struct {
		name string
		body map[string]any
		code int
	}{
		{"http", saveBody("http://a.example", []string{}), 422},
		{"ftp", saveBody("ftp://a.example", []string{}), 422},
		{"https ไม่มี host", saveBody("https://", []string{}), 422},
		{"ยาว 501", saveBody("https://a.example/"+strings.Repeat("x", 483), []string{}), 422},
		{"callback null", saveBody(nil, []string{}), 422},
		{"ไม่ส่ง callback", saveBody("-", []string{}), 422},
		{"ไม่ส่ง allowed_ips", saveBody("", "-"), 422},
		{"IPv6", saveBody("", []string{"2001:db8::1"}), 422},
		{"300.1.1.1", saveBody("", []string{"300.1.1.1"}), 422},
		{"/33", saveBody("", []string{"1.2.3.4/33"}), 422},
		{"IP ซ้ำ", saveBody("", []string{"1.2.3.4", "1.2.3.4/32"}), 422},
		{"ไม่ส่ง passcode", map[string]any{"callback_url": "", "allowed_ips": []string{}}, 422},
		{"passcode ผิด", map[string]any{"callback_url": "", "allowed_ips": []string{}, "passcode": "000000"}, 401204},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expect(t, call(t, app, "POST", updateCredentialPath, tt.body, o.oneTok), 200, tt.code)
		})
	}
	many := make([]string, 51)
	for i := range many {
		many[i] = fmt.Sprintf("10.0.0.%d", i+1)
	}
	expect(t, call(t, app, "POST", updateCredentialPath, saveBody("", many), o.oneTok), 200, 422)

	// บันทึกสำเร็จ · IP เดี่ยวเก็บเป็น /32
	expect(t, call(t, app, "POST", updateCredentialPath,
		saveBody(" https://api.customer.example/minigame ", []string{"203.0.113.10", "198.51.100.0/24"}), o.oneTok), 200, 200)
	d, _ := getCredential(t, app, o.oneTok)
	if d.Key != before.Key || d.CallbackURL != "https://api.customer.example/minigame" ||
		len(d.AllowedIPs) != 2 || d.AllowedIPs[0] != "203.0.113.10/32" || d.AllowedIPs[1] != "198.51.100.0/24" {
		t.Fatalf("after save %+v", d)
	}

	// แทนทั้งชุด: IP เดิมที่ไม่อยู่ในรายการใหม่ถูกลบ · ลิงก์ "   " = ไม่ตั้ง
	expect(t, call(t, app, "POST", updateCredentialPath, saveBody("   ", []string{"1.2.3.4"}), o.oneTok), 200, 200)
	d, _ = getCredential(t, app, o.oneTok)
	if d.CallbackURL != "" || len(d.AllowedIPs) != 1 || d.AllowedIPs[0] != "1.2.3.4/32" {
		t.Fatalf("after replace %+v", d)
	}

	// ประวัติ ACC-09: 2 แถว ค่าเก่า / ใหม่ถูก · ไม่มี Key
	var logs []models.APICredentialLog
	database.DBConn.Where("agent_id = ?", o.one.ID).Order("id").Find(&logs)
	if len(logs) != 2 {
		t.Fatalf("logs = %d, want 2", len(logs))
	}
	l := logs[1]
	if l.OldCallbackURL == nil || *l.OldCallbackURL != "https://api.customer.example/minigame" || l.NewCallbackURL != nil ||
		len(l.OldIPs) != 2 || len(l.NewIPs) != 1 || l.NewIPs[0] != "1.2.3.4/32" || l.ActorUsername != "one2one" || l.IP == nil {
		t.Fatalf("log %+v", l)
	}
	raw, _ := json.Marshal(logs)
	if strings.Contains(string(raw), before.Key) {
		t.Fatal("ประวัติห้ามมี Key")
	}
}

func TestAPICredentialSubPermission(t *testing.T) { // ACC-02
	app := setup2(t)
	o := buildOwners(t, app)
	hash, _ := utils.HashPassword(mgPassword)
	sub := models.Subaccount{AgentID: o.one.ID, Username: "one2one@staff", PasswordHash: hash, Status: models.AgentStatusActive,
		Permissions: `{"account":"view"}`}
	if err := postgres.CreateSubaccountRepository(database.DBConn, &sub); err != nil {
		t.Fatal(err)
	}
	tok := readyToken(t, app, models.AccountTypeSub, sub.ID, sub.Username)

	d, _ := getCredential(t, app, tok)
	if d.Username != "one2one" {
		t.Fatalf("sub ต้องเห็น username ของเจ้าของ ได้ %q", d.Username)
	}
	expect(t, call(t, app, "POST", updateCredentialPath, saveBody("", []string{}), tok), 200, 402303)

	setCols(t, models.AccountTypeSub, sub.ID, map[string]any{"permissions": `{"account":"edit"}`})
	expect(t, call(t, app, "POST", updateCredentialPath, saveBody("", []string{"1.2.3.4"}), tok), 200, 200)
	var l models.APICredentialLog
	database.DBConn.Where("agent_id = ?", o.one.ID).Take(&l)
	if l.ActorType != models.AccountTypeSub || l.ActorUsername != "one2one@staff" {
		t.Fatalf("log actor %+v", l)
	}

	setCols(t, models.AccountTypeSub, sub.ID, map[string]any{"permissions": `{}`})
	expect(t, call(t, app, "GET", apiCredentialPath, nil, tok), 200, 402303)
	expect(t, call(t, app, "POST", updateCredentialPath, saveBody("", []string{}), tok), 200, 402303)

	// เจ้าของ SUSPENDED กดบันทึก → ถูกกันที่ middleware (ACC-31 / AUTH-54)
	setStatus(t, o.one.ID, models.AgentStatusSuspended)
	expect(t, call(t, app, "POST", updateCredentialPath, saveBody("", []string{}), o.oneTok), 403, 401311)
}
