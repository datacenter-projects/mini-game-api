//go:build integration

// API test ของ game_rate (อัตราแพ้ชนะ · เมนู ⑥) ตามตาราง test case ใน docs/modules/game_rate.md หัวข้อ 7
package backoffice_test

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"app/app/models"
	agentAuthPostgres "app/app/repository/postgres/agent_auth"
	gameRateService "app/app/service/game_rate"
	"app/pkg/utils"
	"app/platform/database"

	"github.com/gofiber/fiber/v2"
)

const (
	rateListPath     = "/api/v1/bo/pr/rate/list"
	rateDetailPath   = "/api/v1/bo/pr/rate/detail/get"
	rateCheckPath    = "/api/v1/bo/pr/rate/detail/check"
	rateUpdatePath   = "/api/v1/bo/pr/rate/detail/update"
	rateVersionsPath = "/api/v1/bo/pr/rate/versions/list"
	rateVersionPath  = "/api/v1/bo/pr/rate/versions/detail/get"
	rateRollbackPath = "/api/v1/bo/pr/rate/versions/rollback"
)

// rateSteps — ชุดตัวอย่างของรูปแบบ steps (spec 5.2) · edit แก้ก่อนส่ง
func rateSteps(edit func(s [][2]float64)) []map[string]any {
	s := [][2]float64{{1.9, 50}, {3.61, 48}, {6.859, 46}, {13.0321, 44}, {24.761, 42}, {47.0459, 40}, {89.3872, 38}, {169.8356, 36}, {322.6877, 34}, {613.1066, 32}}
	if edit != nil {
		edit(s)
	}
	out := make([]map[string]any, len(s))
	for i, v := range s {
		out[i] = map[string]any{"multiplier": v[0], "win_percent": v[1]}
	}
	return out
}

// ratePrizes — ตารางรางวัลตัวอย่างของ Scratch (diagram)
func ratePrizes() []map[string]any {
	return []map[string]any{{"multiplier": 0.5, "share_percent": 35}, {"multiplier": 1, "share_percent": 35}, {"multiplier": 2, "share_percent": 18},
		{"multiplier": 5, "share_percent": 10}, {"multiplier": 8, "share_percent": 2}}
}

func coinBody(base int, steps []map[string]any) map[string]any {
	return map[string]any{"game_code": "coin_toss", "base_version": base, "steps": steps, "passcode": passcode}
}

func scratchBody(base int, win float64, prizes []map[string]any) map[string]any {
	return map[string]any{"game_code": "scratch_card", "base_version": base, "win_percent": win, "prizes": prizes, "passcode": passcode}
}

type rateSaved struct {
	Version      int  `json:"version"`
	Created      bool `json:"created"`
	RollbackFrom int  `json:"rollback_from"`
}

func saveRate(t *testing.T, app *fiber.App, tok, path string, body map[string]any) rateSaved {
	t.Helper()
	r := call(t, app, "POST", path, body, tok)
	expect(t, r, 200, 200)
	var d rateSaved
	_ = json.Unmarshal(r.Data, &d)
	return d
}

func rateVersionCount(t *testing.T, game string) int64 {
	t.Helper()
	return countRows(t, &models.GameRateVersion{}, "game_code = ?", game)
}

// superadminSub — sub ของ Superadmin ที่ได้สิทธิ์ตาม perms พร้อม token ที่ผ่านด่าน
func superadminSub(t *testing.T, app *fiber.App, name, perms string) string {
	t.Helper()
	var sa models.UserAgent
	database.DBConn.Where("username = ?", "superadmin").Take(&sa)
	hash, _ := utils.HashPassword(mgPassword)
	s := models.Subaccount{AgentID: sa.ID, Username: "superadmin@" + name, PasswordHash: hash, Status: models.AgentStatusActive, Permissions: perms}
	if err := agentAuthPostgres.CreateSubaccountRepository(database.DBConn, &s); err != nil {
		t.Fatal(err)
	}
	return readyToken(t, app, models.AccountTypeSub, s.ID, s.Username)
}

func TestRatePermissions(t *testing.T) { // RATE-02 · lead RS1: Superadmin + sub ที่มีสิทธิ์ rate เท่านั้น
	app := setup2(t)
	c := buildChain(t, app)
	admin := createAccount(t, "admin01", models.AgentRoleAdmin, nil)
	adminTok := loginReady(t, app, models.AccountTypeAgent, admin.ID, "admin01")

	for name, tok := range map[string]string{"Company": c.comTok, "Share": c.shareTok, "Agent": c.agentTok, "ADMIN": adminTok} {
		if r := call(t, app, "POST", rateListPath, map[string]any{}, tok); r.Code != 402303 {
			t.Fatalf("%s list ได้ %d", name, r.Code)
		}
		// ไม่มีสิทธิ์ + ไม่ส่ง passcode = 402303 (สิทธิ์ก่อน passcode)
		if r := call(t, app, "POST", rateUpdatePath, map[string]any{"game_code": "coin_toss"}, tok); r.Code != 402303 {
			t.Fatalf("%s update ได้ %d", name, r.Code)
		}
	}

	noneTok := superadminSub(t, app, "none", `{}`)
	viewTok := superadminSub(t, app, "viewer", `{"rate":"view"}`)
	editTok := superadminSub(t, app, "editor", `{"rate":"edit"}`)
	expect(t, call(t, app, "POST", rateListPath, map[string]any{}, noneTok), 200, 402303)
	expect(t, call(t, app, "POST", rateListPath, map[string]any{}, viewTok), 200, 200)
	expect(t, call(t, app, "POST", rateCheckPath, map[string]any{"game_code": "coin_toss", "steps": rateSteps(nil)}, viewTok), 200, 200)
	expect(t, call(t, app, "POST", rateUpdatePath, coinBody(0, rateSteps(nil)), viewTok), 200, 402303)

	// sub ที่มี edit บันทึกได้ · log คนแก้ (RATE-08)
	expect(t, call(t, app, "POST", rateUpdatePath, map[string]any{"game_code": "coin_toss", "base_version": 0, "steps": rateSteps(nil)}, editTok), 200, 422) // ไม่ส่ง passcode
	saveRate(t, app, editTok, rateUpdatePath, coinBody(0, rateSteps(nil)))
	var v models.GameRateVersion
	database.DBConn.Where("game_code = ? AND version = 1", "coin_toss").Take(&v)
	if v.ActorType != models.AccountTypeSub || v.CreatedBy != "superadmin@editor" || v.IP == "" || v.RequestID == "" || v.RollbackFrom != 0 || v.WinPercent != nil {
		t.Fatalf("version log %+v", v)
	}
	// Superadmin บันทึกได้ · actor_type AGENT
	saveRate(t, app, c.saTok, rateUpdatePath, scratchBody(0, 50, ratePrizes()))
	v = models.GameRateVersion{}
	database.DBConn.Where("game_code = ? AND version = 1", "scratch_card").Take(&v)
	if v.ActorType != models.AccountTypeAgent || v.CreatedBy != "superadmin" || v.WinPercent == nil || *v.WinPercent != 50 {
		t.Fatalf("scratch version %+v", v)
	}
}

func TestRateFormat(t *testing.T) { // RATE-01 / 03 / 04 / 10 / 11 / 20 / 21 / 30 / 31 · 422 ไม่สร้างเวอร์ชัน
	app := setup2(t)
	c := buildChain(t, app)
	tests := []struct {
		name  string
		body  map[string]any
		code  int
		field string
	}{
		{"ไม่ส่ง game_code", map[string]any{"base_version": 0, "steps": rateSteps(nil), "passcode": passcode}, 422, "game_code"},
		{"เกมไม่มี", map[string]any{"game_code": "slot", "base_version": 0, "passcode": passcode}, 406401, ""},
		{"ไม่ส่ง base_version", map[string]any{"game_code": "coin_toss", "steps": rateSteps(nil), "passcode": passcode}, 422, "base_version"},
		{"ทศนิยม 5 ตำแหน่ง", coinBody(0, rateSteps(func(s [][2]float64) { s[0][0] = 1.90005 })), 422, "steps[0].multiplier"},
		{"win 5 ตำแหน่ง", coinBody(0, rateSteps(func(s [][2]float64) { s[1][1] = 48.12345 })), 422, "steps[1].win_percent"},
		{"9 ขั้น", coinBody(0, rateSteps(nil)[:9]), 422, "steps"},
		{"ตัวคูณ 1,000.0001", coinBody(0, rateSteps(func(s [][2]float64) { s[9][0] = 1000.0001 })), 422, "steps[9].multiplier"},
		{"ตัวคูณขั้น 4 ≤ ขั้น 3", coinBody(0, rateSteps(func(s [][2]float64) { s[3][0] = 6.859 })), 422, "steps[3].multiplier"},
		{"win 0", coinBody(0, rateSteps(func(s [][2]float64) { s[2][1] = 0 })), 422, "steps[2].win_percent"},
		{"win 50.0001 (เพดาน 50)", coinBody(0, rateSteps(func(s [][2]float64) { s[0][1] = 50.0001 })), 422, "steps[0].win_percent"},
		{"prizes ให้ coin", func() map[string]any { b := coinBody(0, rateSteps(nil)); b["prizes"] = ratePrizes(); return b }(), 422, "prizes"},
		{"win_percent ให้ coin", func() map[string]any { b := coinBody(0, rateSteps(nil)); b["win_percent"] = 50; return b }(), 422, "win_percent"},
		{"RPS ส่ง draw_percent", func() map[string]any {
			s := rateSteps(nil)
			s[0]["draw_percent"] = 30
			return map[string]any{"game_code": "rock_paper_scissors", "base_version": 0, "steps": s, "passcode": passcode}
		}(), 422, "steps[0].draw_percent"},
		{"steps ให้ scratch", func() map[string]any { b := scratchBody(0, 50, ratePrizes()); b["steps"] = rateSteps(nil); return b }(), 422, "steps"},
		{"scratch win 50.0001", scratchBody(0, 50.0001, ratePrizes()), 422, "win_percent"},
		{"scratch 0 แถว", scratchBody(0, 50, []map[string]any{}), 422, "prizes"},
		{"scratch ตัวคูณซ้ำ", scratchBody(0, 50, []map[string]any{{"multiplier": 1, "share_percent": 50}, {"multiplier": 1, "share_percent": 50}}), 422, "prizes[1].multiplier"},
		{"scratch รางวัล 0 เท่า", scratchBody(0, 50, []map[string]any{{"multiplier": 0, "share_percent": 100}}), 422, "prizes[0].multiplier"},
		{"ตัวเลขเป็นข้อความ", map[string]any{"game_code": "scratch_card", "base_version": 0, "win_percent": "50", "prizes": ratePrizes(), "passcode": passcode}, 400, ""},
	}
	many := make([]map[string]any, 11)
	for i := range many {
		many[i] = map[string]any{"multiplier": i + 1, "share_percent": 1}
	}
	tests = append(tests, struct {
		name  string
		body  map[string]any
		code  int
		field string
	}{"scratch 11 แถว", scratchBody(0, 50, many), 422, "prizes"})

	for _, tt := range tests {
		r := call(t, app, "POST", rateUpdatePath, tt.body, c.saTok)
		if r.Code != tt.code {
			t.Fatalf("%s: code %d (%s) want %d", tt.name, r.Code, r.Msg, tt.code)
		}
		if tt.field != "" && !strings.HasPrefix(r.Msg, tt.field+" ") {
			t.Fatalf("%s: msg %q ต้องขึ้นต้นด้วย field %s", tt.name, r.Msg, tt.field)
		}
	}
	if n := countRows(t, &models.GameRateVersion{}, "1 = 1"); n != 0 {
		t.Fatalf("รูปแบบผิดต้องไม่สร้างเวอร์ชัน ได้ %d", n)
	}
	// RPS ไม่มีเสมอ: รูปแบบเดียวกับ Coin บันทึกได้ (lead Q7)
	saveRate(t, app, c.saTok, rateUpdatePath, map[string]any{"game_code": "rock_paper_scissors", "base_version": 0, "steps": rateSteps(nil), "passcode": passcode})
}

func TestRateRules(t *testing.T) { // RATE-07 · RATE-32 · RATE-33 · RN2 (detail/check)
	app := setup2(t)
	c := buildChain(t, app)

	// house edge ต่ำกว่า 1%: msg บอก field + ค่าสูงสุด (ปัดลง) · ไม่สร้างเวอร์ชัน
	r := call(t, app, "POST", rateUpdatePath, coinBody(0, rateSteps(func(s [][2]float64) { s[0] = [2]float64{1.99, 50} })), c.saTok)
	expect(t, r, 200, 406101)
	expectMsgHas(t, r.Msg, "steps[0].win_percent", "49.7487")
	if rateVersionCount(t, "coin_toss") != 0 {
		t.Fatal("ผิดกฎต้องไม่สร้างเวอร์ชัน")
	}
	saveRate(t, app, c.saTok, rateUpdatePath, coinBody(0, rateSteps(func(s [][2]float64) { s[0] = [2]float64{1.99, 49.7487} }))) // ค่าที่บอกตั้งได้จริง

	r = call(t, app, "POST", rateUpdatePath, scratchBody(0, 50, []map[string]any{{"multiplier": 1, "share_percent": 64.5}, {"multiplier": 2, "share_percent": 35}}), c.saTok)
	expect(t, r, 200, 406102)
	expectMsgHas(t, r.Msg, "99.5")
	expect(t, call(t, app, "POST", rateUpdatePath, scratchBody(0, 50, []map[string]any{{"multiplier": 2, "share_percent": 100}}), c.saTok), 200, 406101)

	// detail/check: ผิดกฎยังตอบ 200 พร้อมค่าคำนวณ + issues ทุกจุด · ไม่บันทึก
	r = call(t, app, "POST", rateCheckPath, map[string]any{"game_code": "scratch_card", "win_percent": 50,
		"prizes": []map[string]any{{"multiplier": 2, "share_percent": 99.9}}}, c.saTok) // share ไม่ครบ + จ่ายคืน 99.9%
	expect(t, r, 200, 200)
	var chk struct {
		Valid  bool `json:"valid"`
		Issues []struct {
			Field string `json:"field"`
			Code  int    `json:"code"`
		} `json:"issues"`
		HouseEdge float64 `json:"house_edge_percent"`
	}
	_ = json.Unmarshal(r.Data, &chk)
	if chk.Valid || len(chk.Issues) != 2 || chk.Issues[0].Code != 406102 || chk.Issues[1].Code != 406101 {
		t.Fatalf("check %s", r.Data)
	}
	r = call(t, app, "POST", rateCheckPath, map[string]any{"game_code": "scratch_card", "win_percent": 50, "prizes": ratePrizes()}, c.saTok)
	_ = json.Unmarshal(r.Data, &chk)
	if !chk.Valid || len(chk.Issues) != 0 || chk.HouseEdge != 22.75 {
		t.Fatalf("check ผ่าน %s", r.Data)
	}
	if rateVersionCount(t, "scratch_card") != 0 {
		t.Fatal("detail/check ต้องไม่บันทึก")
	}
	expect(t, call(t, app, "POST", rateCheckPath, map[string]any{"game_code": "coin_toss", "steps": rateSteps(nil)[:9]}, c.saTok), 200, 422)
}

func TestRateVersions(t *testing.T) { // RATE-05 / 05A / 05B / 05C · RE1 / RE2 / RE5 / RE6
	app := setup2(t)
	c := buildChain(t, app)

	// ยังไม่เคยตั้ง: version 0 · steps / prizes []
	r := call(t, app, "POST", rateDetailPath, map[string]any{"game_code": "scratch_card"}, c.saTok)
	expect(t, r, 200, 200)
	if !strings.Contains(string(r.Data), `"version":0`) || !strings.Contains(string(r.Data), `"prizes":[]`) {
		t.Fatalf("ยังไม่ตั้ง %s", r.Data)
	}

	s1 := saveRate(t, app, c.saTok, rateUpdatePath, scratchBody(0, 35, []map[string]any{{"multiplier": 0.5, "share_percent": 20}, {"multiplier": 1, "share_percent": 50}, {"multiplier": 2, "share_percent": 30}}))
	s2 := saveRate(t, app, c.saTok, rateUpdatePath, scratchBody(1, 50, []map[string]any{{"multiplier": 1, "share_percent": 60}, {"multiplier": 2, "share_percent": 30}, {"multiplier": 5, "share_percent": 10}}))
	if !s1.Created || s1.Version != 1 || !s2.Created || s2.Version != 2 {
		t.Fatalf("v1 %+v v2 %+v", s1, s2)
	}
	// ค่าเดิมทั้งชุด = created false ไม่มีเวอร์ชันใหม่ (RATE-05C)
	if s := saveRate(t, app, c.saTok, rateUpdatePath, scratchBody(2, 50, []map[string]any{{"multiplier": 1, "share_percent": 60}, {"multiplier": 2, "share_percent": 30}, {"multiplier": 5, "share_percent": 10}})); s.Created || s.Version != 2 {
		t.Fatalf("ค่าเดิม %+v", s)
	}
	// base_version เก่า = 406301 บอกเวอร์ชันล่าสุดและคนแก้ (RATE-05B · Q5)
	r = call(t, app, "POST", rateUpdatePath, scratchBody(1, 45, ratePrizes()), c.saTok)
	expect(t, r, 200, 406301)
	expectMsgHas(t, r.Msg, "v2", "superadmin")

	// ย้อนไป v1 = v3 ค่าเท่า v1 · rollback_from 1 · v2 ไม่เปลี่ยน
	rb := saveRate(t, app, c.saTok, rateRollbackPath, map[string]any{"game_code": "scratch_card", "version": 1, "base_version": 2, "passcode": passcode})
	if !rb.Created || rb.Version != 3 || rb.RollbackFrom != 1 {
		t.Fatalf("rollback %+v", rb)
	}
	// ย้อนไปเวอร์ชันที่ค่าเท่าปัจจุบัน = created false · ไม่มีเวอร์ชัน · rollback_from 0
	if rb := saveRate(t, app, c.saTok, rateRollbackPath, map[string]any{"game_code": "scratch_card", "version": 1, "base_version": 3, "passcode": passcode}); rb.Created || rb.Version != 3 || rb.RollbackFrom != 0 {
		t.Fatalf("rollback ค่าเดิม %+v", rb)
	}
	expect(t, call(t, app, "POST", rateRollbackPath, map[string]any{"game_code": "scratch_card", "version": 9, "base_version": 3, "passcode": passcode}, c.saTok), 200, 406402)
	expect(t, call(t, app, "POST", rateRollbackPath, map[string]any{"game_code": "scratch_card", "version": 2, "base_version": 1, "passcode": passcode}, c.saTok), 200, 406301)
	expect(t, call(t, app, "POST", rateRollbackPath, map[string]any{"game_code": "scratch_card", "version": 2, "base_version": 3}, c.saTok), 200, 422) // ไม่ส่ง passcode

	// RE6 ค่าของเวอร์ชันเก่า
	r = call(t, app, "POST", rateVersionPath, map[string]any{"game_code": "scratch_card", "version": 2}, c.saTok)
	expect(t, r, 200, 200)
	if !strings.Contains(string(r.Data), `"version":2`) || !strings.Contains(string(r.Data), `"win_percent":50`) {
		t.Fatalf("v2 %s", r.Data)
	}
	expect(t, call(t, app, "POST", rateVersionPath, map[string]any{"game_code": "scratch_card", "version": 7}, c.saTok), 200, 406402)

	// RE5: ใหม่ → เก่า · changes เทียบกับเวอร์ชันก่อน · same_as_current · win_percent (Scratch)
	r = call(t, app, "POST", rateVersionsPath, map[string]any{"game_code": "scratch_card", "page": 1, "limit": 2}, c.saTok)
	expect(t, r, 200, 200)
	var page struct {
		Total int64 `json:"total_count"`
		Data  []struct {
			Version       int      `json:"version"`
			WinPercent    *float64 `json:"win_percent"`
			RollbackFrom  int      `json:"rollback_from"`
			SameAsCurrent bool     `json:"same_as_current"`
			Changes       []struct {
				Type       string   `json:"type"`
				Multiplier float64  `json:"multiplier"`
				From       *float64 `json:"from"`
				To         *float64 `json:"to"`
			} `json:"changes"`
		} `json:"data"`
	}
	_ = json.Unmarshal(r.Data, &page)
	if page.Total != 3 || len(page.Data) != 2 || page.Data[0].Version != 3 || page.Data[1].Version != 2 {
		t.Fatalf("versions %s", r.Data)
	}
	v3, v2 := page.Data[0], page.Data[1]
	if !v3.SameAsCurrent || v3.RollbackFrom != 1 || v3.WinPercent == nil || *v3.WinPercent != 35 || len(v3.Changes) != 4 { // v2 → v3: win · 0.5× เพิ่ม · 1× share · 5× หาย (2× คงเดิม)
		t.Fatalf("v3 %+v", v3)
	}
	if v2.SameAsCurrent || len(v2.Changes) != 4 || v2.Changes[0].Type != "win_percent" { // v1 → v2 (เวอร์ชันก่อนแถวสุดท้ายโหลดแยก)
		t.Fatalf("v2 %+v", v2)
	}
	r = call(t, app, "POST", rateVersionsPath, map[string]any{"game_code": "scratch_card", "page": 2, "limit": 2}, c.saTok)
	_ = json.Unmarshal(r.Data, &page)
	if len(page.Data) != 1 || page.Data[0].Version != 1 || len(page.Data[0].Changes) != 0 || !page.Data[0].SameAsCurrent { // เวอร์ชันแรก changes [] · ค่าเท่า v3
		t.Fatalf("หน้า 2 %s", r.Data)
	}
	if !strings.Contains(string(r.Data), `"changes":[]`) {
		t.Fatalf("changes ต้องเป็น [] ไม่ใช่ null %s", r.Data)
	}

	// RE1: ลำดับเกม · เวอร์ชัน · house edge ต่ำสุด
	r = call(t, app, "POST", rateListPath, map[string]any{}, c.saTok)
	var list []struct {
		GameCode  string  `json:"game_code"`
		Version   int     `json:"version"`
		MinEdge   float64 `json:"min_house_edge_percent"`
		CreatedBy string  `json:"created_by"`
	}
	_ = json.Unmarshal(r.Data, &list)
	if len(list) != 3 || list[0].GameCode != "scratch_card" || list[0].Version != 3 || list[0].CreatedBy != "superadmin" ||
		list[1].GameCode != "coin_toss" || list[1].Version != 0 || list[2].GameCode != "rock_paper_scissors" {
		t.Fatalf("list %s", r.Data)
	}
}

func TestRateRollbackChecksCurrentRules(t *testing.T) { // RATE-05A: เวอร์ชันเก่าที่ไม่ผ่านกฎปัจจุบันย้อนไม่ได้
	app := setup2(t)
	c := buildChain(t, app)
	saveRate(t, app, c.saTok, rateUpdatePath, coinBody(0, rateSteps(nil)))
	// เวอร์ชันที่ house edge 0.5% (จำลองข้อมูลก่อนกฎ 1% — ใส่ตรงใน DB)
	old := models.GameRateVersion{GameCode: "coin_toss", Version: 2, ActorType: models.AccountTypeAgent, ActorID: 1, CreatedBy: "legacy"}
	if err := database.DBConn.Create(&old).Error; err != nil {
		t.Fatal(err)
	}
	var rows []models.GameRateStep
	for i, s := range rateSteps(func(s [][2]float64) { s[0] = [2]float64{1.99, 50} }) {
		rows = append(rows, models.GameRateStep{VersionID: old.ID, Step: i + 1, Multiplier: s["multiplier"].(float64), WinPercent: s["win_percent"].(float64)})
	}
	if err := database.DBConn.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	r := call(t, app, "POST", rateRollbackPath, map[string]any{"game_code": "coin_toss", "version": 2, "base_version": 2, "passcode": passcode}, c.saTok)
	expect(t, r, 200, 406101)
	if rateVersionCount(t, "coin_toss") != 2 {
		t.Fatal("ย้อนไม่ผ่านต้องไม่สร้างเวอร์ชัน")
	}
}

func TestRateConcurrent(t *testing.T) { // RATE-05B: base_version เดียวกันยิงพร้อมกัน = สำเร็จ 1 อีกอัน 406301
	app := setup2(t)
	c := buildChain(t, app)
	saveRate(t, app, c.saTok, rateUpdatePath, coinBody(0, rateSteps(nil)))
	bodies := []map[string]any{
		coinBody(1, rateSteps(func(s [][2]float64) { s[0][1] = 49 })),
		coinBody(1, rateSteps(func(s [][2]float64) { s[0][1] = 48 })),
		coinBody(1, rateSteps(func(s [][2]float64) { s[0][1] = 47 })),
	}
	codes := make([]int, len(bodies))
	var wg sync.WaitGroup
	for i, b := range bodies {
		wg.Add(1)
		go func(i int, b map[string]any) {
			defer wg.Done()
			codes[i] = callCode(app, rateUpdatePath, b, c.saTok)
		}(i, b)
	}
	wg.Wait()
	ok, stale := 0, 0
	for _, code := range codes {
		switch code {
		case 200:
			ok++
		case 406301:
			stale++
		default:
			t.Fatalf("codes %v", codes)
		}
	}
	if ok != 1 || stale != 2 || rateVersionCount(t, "coin_toss") != 2 {
		t.Fatalf("codes %v versions %d", codes, rateVersionCount(t, "coin_toss"))
	}
}

func TestRateEngineFunctions(t *testing.T) { // หัวข้อ 5.8 · RATE-09A · RH1
	app := setup2(t)
	c := buildChain(t, app)
	ctx := t.Context()
	if _, err := gameRateService.CurrentVersionService(ctx, database.DBConn, "coin_toss"); !errors.Is(err, gameRateService.ErrRateNotConfigured) {
		t.Fatalf("ยังไม่ตั้งต้องได้ ErrRateNotConfigured ได้ %v", err)
	}
	saveRate(t, app, c.saTok, rateUpdatePath, coinBody(0, rateSteps(nil)))
	cur, err := gameRateService.CurrentVersionService(ctx, database.DBConn, "coin_toss")
	if err != nil || cur.Version != 1 || cur.VersionID == 0 || len(cur.Steps) != 10 || cur.Steps[1].Multiplier != 3.61 {
		t.Fatalf("current %+v %v", cur, err)
	}
	saveRate(t, app, c.saTok, rateUpdatePath, coinBody(1, rateSteps(func(s [][2]float64) { s[0][1] = 45 })))
	old, err := gameRateService.ByVersionIDService(ctx, cur.VersionID) // เดิมพันเก่ายังได้ค่าเดิม (RATE-06)
	if err != nil || old.Version != 1 || old.Steps[0].WinPercent != 50 {
		t.Fatalf("by id %+v %v", old, err)
	}
	if _, err := gameRateService.ByVersionIDService(ctx, 999999); !errors.Is(err, gameRateService.ErrRateVersionMissing) {
		t.Fatalf("id ไม่มี %v", err)
	}
}
