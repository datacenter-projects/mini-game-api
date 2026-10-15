//go:build integration

// API test ของเส้น dev คิด PT ต่อ Member — docs/modules/pt_calculation.md (PTC-02 – PTC-43 · ชุดข้อมูลหัวข้อ 7.1)
package backoffice_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"app/app/models"
	"app/pkg/configs"
	"app/pkg/testutil"

	"github.com/gofiber/fiber/v2"
)

const ptCalculatePath = "/api/v1/bo/pr/dev/pt/calculate"

type ptLayer struct {
	ID         uint     `json:"id"`
	Position   string   `json:"position"`
	PT         *float64 `json:"pt"`
	Commission float64  `json:"commission"`
}

// setupDev — app ที่เปิด / ปิดเส้น dev (PTC-40) · testutil.Setup คืนค่า config ให้ test ถัดไปเอง
func setupDev(t *testing.T, enabled bool) *fiber.App {
	t.Helper()
	testutil.Setup(t)
	configs.Cfg.Auth.LoginIPLimit = 1000
	configs.Cfg.DevRoutesEnabled = enabled
	return testutil.NewApp(nil)
}

func adminTokenOf(t *testing.T, app *fiber.App) string {
	t.Helper()
	admin := createAccount(t, "support01", models.AgentRoleAdmin, nil)
	return loginReady(t, app, models.AccountTypeAgent, admin.ID, "support01")
}

// memberWith — Member ใต้บัญชีของ tok ที่ผู้สร้างถือสู้ pt + commission
func memberWith(t *testing.T, app *fiber.App, tok, username string, pt, commission float64) created {
	t.Helper()
	b := memberBody(username, commission)
	b["pt"].(map[string]any)["minigame"].(map[string]any)["pt"] = pt
	r := call(t, app, "POST", createMemberPath, b, tok)
	expect(t, r, 200, 200)
	var d created
	_ = json.Unmarshal(r.Data, &d)
	return d
}

func calcPT(t *testing.T, app *fiber.App, tok string, memberID uint) []ptLayer {
	t.Helper()
	r := call(t, app, "POST", ptCalculatePath, map[string]any{"member_id": memberID, "game_code": "coin_toss"}, tok)
	expect(t, r, 200, 200)
	var out []ptLayer
	if err := json.Unmarshal(r.Data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func pt(v float64) *float64 { return &v }

func expectLayers(t *testing.T, name string, got, want []ptLayer) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		gj, _ := json.Marshal(got)
		wj, _ := json.Marshal(want)
		t.Fatalf("%s:\n got  %s\n want %s", name, gj, wj)
	}
}

func TestPTCalculateDisabled(t *testing.T) { // PTC-43: ปิด flag = ไม่มีเส้น
	app := setupDev(t, false)
	tok := adminTokenOf(t, app)
	expect(t, call(t, app, "POST", ptCalculatePath, map[string]any{"member_id": 1, "game_code": "coin_toss"}, tok), 404, 404001)
}

func TestPTCalculate(t *testing.T) {
	app := setupDev(t, true)
	adminTok := adminTokenOf(t, app)

	// ชุด A (reference ข้อ 3): SA → Company 95 → Share 85 F20 R70 → Agent 70 F30 R40 → Member pt 20
	sa, saTok := seedSuperadmin(t, app)
	co, coTok := mustCreate(t, app, saTok, agentBody("COMPANY_TRANSFER", "ptcco", nil, childPT(95, 0, 0, 1)))
	sh, shTok := mustCreate(t, app, coTok, agentBody("SHARE_B2C", "ptcsh", []string{"THB"}, childPT(85, 20, 70, 0.7)))
	ag, agTok := mustCreate(t, app, shTok, agentBody("AGENT", "ptcag", nil, childPT(70, 30, 40, 0.3)))
	mem := memberWith(t, app, agTok, "ptcmem", 20, 1)

	// สิทธิ์: ADMIN เท่านั้น (PTC-42)
	expect(t, call(t, app, "POST", ptCalculatePath, map[string]any{"member_id": mem.ID, "game_code": "coin_toss"}, saTok), 403, 401308)

	expectLayers(t, "ชุด A", calcPT(t, app, adminTok, mem.ID), []ptLayer{
		{sa.ID, "superadmin", pt(20), 0}, {co.ID, "company", pt(25), 1}, {sh.ID, "shareholder", pt(25), 0.7},
		{ag.ID, "agent", pt(30), 0.3}, {mem.ID, "member", nil, 1},
	})

	// Agent ซ้อน (PTC-22): Agent → Agent2 50 F40 R45 → Member pt 30 → SA 10 · Co 10 · Sh 15 · Ag 25 · Ag2 40
	ag2, ag2Tok := mustCreate(t, app, agTok, agentBody("AGENT", "ptcag2", nil, childPT(50, 40, 45, 0.2)))
	mem2 := memberWith(t, app, ag2Tok, "ptcmem2", 30, 0.5)
	expectLayers(t, "Agent ซ้อน", calcPT(t, app, adminTok, mem2.ID), []ptLayer{
		{sa.ID, "superadmin", pt(10), 0}, {co.ID, "company", pt(10), 1}, {sh.ID, "shareholder", pt(15), 0.7},
		{ag.ID, "agent", pt(25), 0.3}, {ag2.ID, "agent", pt(40), 0.2}, {mem2.ID, "member", nil, 0.5},
	})

	// ชุด E (lead Q2): Share ไม่รับ PT → SA 20 · Co 40 · Sh 10 · Ag 30
	execSQL(t, "UPDATE agent_game_settings SET status = false WHERE agent_id = ?", sh.ID)
	expectLayers(t, "ชุด E", calcPT(t, app, adminTok, mem.ID), []ptLayer{
		{sa.ID, "superadmin", pt(20), 0}, {co.ID, "company", pt(40), 1}, {sh.ID, "shareholder", pt(10), 0.7},
		{ag.ID, "agent", pt(30), 0.3}, {mem.ID, "member", nil, 1},
	})

	// input (PTC-20 / PTC-21)
	for name, body := range map[string]map[string]any{
		"ไม่ส่ง member_id": {"game_code": "coin_toss"},
		"game_code ไม่มี":  {"member_id": mem.ID, "game_code": "poker"},
	} {
		if r := call(t, app, "POST", ptCalculatePath, body, adminTok); r.Code != 422 {
			t.Errorf("%s: code %d msg %s", name, r.Code, r.Msg)
		}
	}
	expect(t, call(t, app, "POST", ptCalculatePath, map[string]any{"member_id": 999999, "game_code": "coin_toss"}, adminTok), 200, 408401)

	// ข้อมูลในสายผิด (PTC-09): ขาดแถวของชั้นกลาง · ลูกได้รับเกินผู้สร้าง
	execSQL(t, "DELETE FROM agent_game_settings WHERE agent_id = ? AND game_code = 'coin_toss'", sh.ID)
	expect(t, call(t, app, "POST", ptCalculatePath, map[string]any{"member_id": mem.ID, "game_code": "coin_toss"}, adminTok), 500, 508601)
	execSQL(t, "UPDATE agent_game_settings SET pt_from_parent = 99 WHERE agent_id = ?", ag2.ID)
	expect(t, call(t, app, "POST", ptCalculatePath, map[string]any{"member_id": mem2.ID, "game_code": "scratch_card"}, adminTok), 500, 508601)
}

func TestPTCalculateCSM(t *testing.T) { // ชุด F · PTC-04A: CSM มี force ค้าง (ตั้งผ่าน SQL — API ห้ามแล้ว) → ข้าม · SA 50 · CSM 0 · SM 50
	app := setupDev(t, true)
	adminTok := adminTokenOf(t, app)
	k := buildCSMChain(t, app) // master01 (CSM 80 · commission 0.5) → sharem1 (0.2) → memsm1 pt 50
	execSQL(t, "UPDATE agent_game_settings SET force = 60 WHERE agent_id = ?", k.csm.ID)

	got := calcPT(t, app, adminTok, k.mem.ID)
	if len(got) != 4 || got[0].Position != "superadmin" || *got[0].PT != 50 || got[1].ID != k.csm.ID || *got[1].PT != 0 ||
		got[1].Commission != 0.5 || got[2].ID != k.sm1.ID || *got[2].PT != 50 || got[2].Commission != 0.2 || got[3].ID != k.mem.ID || got[3].PT != nil {
		gj, _ := json.Marshal(got)
		t.Fatalf("ชุด F: %s", gj)
	}
}
