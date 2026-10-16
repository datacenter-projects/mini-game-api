//go:build integration

// API test ของเส้น dev คิด PT ต่อ Member — docs/modules/pt_calculation.md (PTC-02 – PTC-43 · ชุดข้อมูลหัวข้อ 7.1)
// เส้นอยู่ context developer (/api/v1/dev — ไม่ต้อง login) แต่ test อยู่ package นี้เพราะสร้างสายผ่าน API หลังบ้าน (helper ของ agent_management)
package backoffice_test

import (
	"context"
	"encoding/json"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	ptCalculationService "app/app/service/pt_calculation"
	"app/pkg/configs"
	"app/pkg/testutil"
	"app/platform/database"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const ptCalculatePath = "/api/v1/dev/pt/calculate"

type ptLayer struct {
	ID            uint     `json:"id"`
	Position      string   `json:"position"`
	PT            *float64 `json:"pt"`
	CommissionNet float64  `json:"commission_net"`
}

// setupDev — app ที่เปิด / ปิดเส้น dev (PTC-40) · testutil.Setup คืนค่า config ให้ test ถัดไปเอง
func setupDev(t *testing.T, enabled bool) *fiber.App {
	t.Helper()
	testutil.Setup(t)
	configs.Cfg.Auth.LoginIPLimit = 1000
	configs.Cfg.DevRoutesEnabled = enabled
	return testutil.NewApp(nil)
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

// calcPT — เรียกเส้น dev โดยไม่มี token (public — PTC-42)
func calcPT(t *testing.T, app *fiber.App, memberID uint) []ptLayer {
	t.Helper()
	r := call(t, app, "POST", ptCalculatePath, map[string]any{"member_id": memberID, "game_code": "coin_toss"}, "")
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

// queryCounter — gorm logger ที่นับจำนวน SQL ที่รันจริง (PTC-30)
type queryCounter struct{ n atomic.Int32 }

func (q *queryCounter) LogMode(gormlogger.LogLevel) gormlogger.Interface { return q }
func (q *queryCounter) Info(context.Context, string, ...any)             {}
func (q *queryCounter) Warn(context.Context, string, ...any)             {}
func (q *queryCounter) Error(context.Context, string, ...any)            {}
func (q *queryCounter) Trace(context.Context, time.Time, func() (string, int64), error) {
	q.n.Add(1)
}

func TestPTCalculateDisabled(t *testing.T) { // PTC-43: ปิด flag = ไม่มีเส้น
	app := setupDev(t, false)
	expect(t, call(t, app, "POST", ptCalculatePath, map[string]any{"member_id": 1, "game_code": "coin_toss"}, ""), 404, 404001)
}

func TestPTCalculate(t *testing.T) {
	app := setupDev(t, true)

	// ชุด A (reference ข้อ 3): SA → Company 95 → Share 85 F20 R70 → Agent 70 F30 R40 → Member pt 20
	sa, saTok := seedSuperadmin(t, app)
	co, coTok := mustCreate(t, app, saTok, agentBody("COMPANY_TRANSFER", "ptcco", nil, childPT(95, 0, 0, 1)))
	sh, shTok := mustCreate(t, app, coTok, agentBody("SHARE_B2C", "ptcsh", []string{"THB"}, childPT(85, 20, 70, 0.7)))
	ag, agTok := mustCreate(t, app, shTok, agentBody("AGENT", "ptcag", nil, childPT(70, 30, 40, 0.3)))
	mem := memberWith(t, app, agTok, "ptcmem", 20, 1)

	expectLayers(t, "ชุด A", calcPT(t, app, mem.ID), []ptLayer{
		{sa.ID, "superadmin", pt(20), -0.2}, {co.ID, "company", pt(25), -0.115}, {sh.ID, "shareholder", pt(25), 0.105},
		{ag.ID, "agent", pt(30), -0.79}, {mem.ID, "member", nil, 1}, // commission_net ตรง "จ่ายจริง" ของ reference ข้อ 4 (PTC-13)
	})

	// Agent ซ้อน (PTC-22): Agent → Agent2 50 F40 R45 → Member pt 30 → SA 10 · Co 10 · Sh 15 · Ag 25 · Ag2 40
	ag2, ag2Tok := mustCreate(t, app, agTok, agentBody("AGENT", "ptcag2", nil, childPT(50, 40, 45, 0.2)))
	mem2 := memberWith(t, app, ag2Tok, "ptcmem2", 30, 0.5)
	expectLayers(t, "Agent ซ้อน", calcPT(t, app, mem2.ID), []ptLayer{
		{sa.ID, "superadmin", pt(10), -0.1}, {co.ID, "company", pt(10), -0.04}, {sh.ID, "shareholder", pt(15), 0.035},
		{ag.ID, "agent", pt(25), -0.015}, {ag2.ID, "agent", pt(40), -0.38}, {mem2.ID, "member", nil, 0.5},
	})

	// ชุด E (lead Q2): Share ไม่รับ PT → SA 20 · Co 40 · Sh 10 · Ag 30
	execSQL(t, "UPDATE agent_game_settings SET status = false WHERE agent_id = ?", sh.ID)
	expectLayers(t, "ชุด E", calcPT(t, app, mem.ID), []ptLayer{
		{sa.ID, "superadmin", pt(20), -0.2}, {co.ID, "company", pt(40), -0.22}, {sh.ID, "shareholder", pt(10), 0.21},
		{ag.ID, "agent", pt(30), -0.79}, {mem.ID, "member", nil, 1},
	})

	// ชุด E2: ผู้สร้าง Member (Agent) ไม่รับ PT → SA 20 · Co 25 · Sh 55 · Ag 0
	execSQL(t, "UPDATE agent_game_settings SET status = true WHERE agent_id = ?", sh.ID)
	execSQL(t, "UPDATE agent_game_settings SET status = false WHERE agent_id = ?", ag.ID)
	expectLayers(t, "ชุด E2", calcPT(t, app, mem.ID), []ptLayer{
		{sa.ID, "superadmin", pt(20), -0.2}, {co.ID, "company", pt(25), -0.115}, {sh.ID, "shareholder", pt(55), 0.015},
		{ag.ID, "agent", pt(0), -0.7}, {mem.ID, "member", nil, 1},
	})

	// input (PTC-20 / PTC-21)
	for name, body := range map[string]map[string]any{
		"ไม่ส่ง member_id": {"game_code": "coin_toss"},
		"game_code ไม่มี":  {"member_id": mem.ID, "game_code": "poker"},
	} {
		if r := call(t, app, "POST", ptCalculatePath, body, ""); r.Code != 422 {
			t.Errorf("%s: code %d msg %s", name, r.Code, r.Msg)
		}
	}
	expect(t, call(t, app, "POST", ptCalculatePath, map[string]any{"member_id": 999999, "game_code": "coin_toss"}, ""), 200, 408401)

	// ข้อมูลในสายผิด (PTC-09): ขาดแถวของชั้นกลาง · ลูกได้รับเกินผู้สร้าง
	execSQL(t, "DELETE FROM agent_game_settings WHERE agent_id = ? AND game_code = 'coin_toss'", sh.ID)
	expect(t, call(t, app, "POST", ptCalculatePath, map[string]any{"member_id": mem.ID, "game_code": "coin_toss"}, ""), 500, 508601)
	execSQL(t, "UPDATE agent_game_settings SET pt_from_parent = 99 WHERE agent_id = ?", ag2.ID)
	expect(t, call(t, app, "POST", ptCalculatePath, map[string]any{"member_id": mem2.ID, "game_code": "scratch_card"}, ""), 500, 508601)
}

func TestPTCalculateChains(t *testing.T) { // ชุด C · D (lead รอบ 2 S2) + นับ query ≤ 3 (PTC-30)
	app := setupDev(t, true)
	sa, saTok := seedSuperadmin(t, app)

	// ชุด C: SA → Company 1 to 1 90 F80 → Member pt 60 → SA 20 · Co 80
	one, oneTok := mustCreate(t, app, saTok, agentBody("COMPANY_SEAMLESS_1TO1", "ptcone", []string{"THB"}, childPT(90, 80, 0, 0.5)))
	memC := memberWith(t, app, oneTok, "ptcmemc", 60, 0.3)
	expectLayers(t, "ชุด C", calcPT(t, app, memC.ID), []ptLayer{
		{sa.ID, "superadmin", pt(20), -0.1}, {one.ID, "company", pt(80), -0.2}, {memC.ID, "member", nil, 0.3},
	})

	// ชุด D: SA → Company 90 → Share 70 → Agent 60 → Member pt 40 (ไม่มี Force / Remain) → SA 30 · Co 20 · Sh 10 · Ag 40
	co, coTok := mustCreate(t, app, saTok, agentBody("COMPANY_TRANSFER", "ptcdco", nil, childPT(90, 0, 0, 0.5)))
	sh, shTok := mustCreate(t, app, coTok, agentBody("SHARE_B2C", "ptcdsh", []string{"THB"}, childPT(70, 0, 0, 0.4)))
	ag, agTok := mustCreate(t, app, shTok, agentBody("AGENT", "ptcdag", nil, childPT(60, 0, 0, 0.3)))
	memD := memberWith(t, app, agTok, "ptcmemd", 40, 0.2)
	expectLayers(t, "ชุด D", calcPT(t, app, memD.ID), []ptLayer{
		{sa.ID, "superadmin", pt(30), -0.15}, {co.ID, "company", pt(20), -0.05}, {sh.ID, "shareholder", pt(10), 0.02},
		{ag.ID, "agent", pt(40), -0.02}, {memD.ID, "member", nil, 0.2},
	})

	// PTC-30: สาย 4 ชั้น → 3 query (Member · settings ของ Member · settings ทั้งสาย) ไม่วนทีละชั้น
	qc := &queryCounter{}
	db := database.DBConn.Session(&gorm.Session{Logger: qc})
	if _, err := ptCalculationService.CalculateMemberPT(context.Background(), db, memD.ID, "coin_toss"); err != nil {
		t.Fatal(err)
	}
	if n := qc.n.Load(); n != 3 { // ต้องเท่ากัน — n = 0 แปลว่า counter ไม่ทำงาน (lead รอบ 2)
		t.Fatalf("query %d ครั้ง want 3 (Member · settings ของ Member · settings ทั้งสาย)", n)
	}
}

func TestPTCalculateCSM(t *testing.T) { // ชุด F · PTC-04A: CSM มี force ค้าง (ตั้งผ่าน SQL — API ห้ามแล้ว) → ข้าม · SA 50 · CSM 0 · SM 50
	// commission_net (PTC-13): CSM ได้ 0.5 × 50% = 0.25 · จ่าย Share Master 0.2 × 50% = 0.1 → SA −0.25 · CSM 0.15 · SM 0.1 · Member 0
	app := setupDev(t, true)
	k := buildCSMChain(t, app) // master01 (CSM 80 · commission 0.5) → sharem1 (0.2) → memsm1 pt 50
	execSQL(t, "UPDATE agent_game_settings SET force = 60 WHERE agent_id = ?", k.csm.ID)

	got := calcPT(t, app, k.mem.ID)
	if len(got) != 4 || got[0].Position != "superadmin" || *got[0].PT != 50 || got[1].ID != k.csm.ID || *got[1].PT != 0 ||
		got[2].ID != k.sm1.ID || *got[2].PT != 50 || got[3].ID != k.mem.ID || got[3].PT != nil ||
		got[0].CommissionNet != -0.25 || got[1].CommissionNet != 0.15 || got[2].CommissionNet != 0.1 || got[3].CommissionNet != 0 {
		gj, _ := json.Marshal(got)
		t.Fatalf("ชุด F: %s", gj)
	}
}
