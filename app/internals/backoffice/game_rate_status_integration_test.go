//go:build integration

// API test ของสวิตช์ปิดเกมทั้งระบบ RE8 (docs/modules/game_rate.md หัวข้อ 5.9 · RATE-01B · lead Q1 / QS1–QS3)
package backoffice_test

import (
	"encoding/json"
	"testing"

	agentManagementCore "app/app/core/agent_management"
	"app/app/models"
	agentAuthPostgres "app/app/repository/postgres/agent_auth"
	"app/platform/database"

	"github.com/gofiber/fiber/v2"
)

const rateStatusPath = "/api/v1/bo/pr/rate/status/update"

// seedSuperadmin2 — Superadmin บัญชีที่ 2 (ระบบมีได้หลายบัญชี — QS1) พร้อมแถว agent_game_settings เปิดทุกเกม
func seedSuperadmin2(t *testing.T) models.UserAgent {
	t.Helper()
	sa := models.UserAgent{Username: "superadmin2", PasswordHash: "x", Role: models.AgentRoleSuperAdmin, Status: models.AgentStatusActive}
	if err := agentAuthPostgres.CreateUserAgentRepository(database.DBConn, &sa); err != nil {
		t.Fatal(err)
	}
	var games []models.AgentGameSetting
	for _, g := range agentManagementCore.GamesOf(agentManagementCore.PTGroupMinigame) {
		games = append(games, models.AgentGameSetting{AgentID: sa.ID, GameCode: g.GameCode, Category: g.Category,
			PTFromParent: 100, Status: true, StatusGame: true})
	}
	if err := database.DBConn.Create(&games).Error; err != nil {
		t.Fatal(err)
	}
	return sa
}

func setGameStatus(t *testing.T, app *fiber.App, tok, game string, on bool) bool {
	t.Helper()
	r := call(t, app, "POST", rateStatusPath, map[string]any{"game_code": game, "status_game": on, "passcode": passcode}, tok)
	expect(t, r, 200, 200)
	var d struct {
		GameCode   string `json:"game_code"`
		StatusGame bool   `json:"status_game"`
		Changed    bool   `json:"changed"`
	}
	_ = json.Unmarshal(r.Data, &d)
	if d.GameCode != game || d.StatusGame != on {
		t.Fatalf("status response %+v", d)
	}
	return d.Changed
}

// listStatus — status_game ต่อเกมจาก RE1
func listStatus(t *testing.T, app *fiber.App, tok string) map[string]bool {
	t.Helper()
	r := call(t, app, "POST", rateListPath, map[string]any{}, tok)
	expect(t, r, 200, 200)
	var rows []struct {
		GameCode   string `json:"game_code"`
		StatusGame bool   `json:"status_game"`
	}
	_ = json.Unmarshal(r.Data, &rows)
	out := map[string]bool{}
	for _, row := range rows {
		out[row.GameCode] = row.StatusGame
	}
	return out
}

// effectiveGame — status_game_effective ของบัญชี id (agents/detail/get)
func effectiveGame(t *testing.T, app *fiber.App, tok string, id uint, game string) bool {
	t.Helper()
	r := call(t, app, "POST", agentDetailPath, map[string]any{"id": id}, tok)
	expect(t, r, 200, 200)
	var d struct {
		StatusGameEffective map[string]bool `json:"status_game_effective"`
	}
	_ = json.Unmarshal(r.Data, &d)
	return d.StatusGameEffective[game]
}

func superadminStatus(t *testing.T, id uint, game string) bool {
	t.Helper()
	var s models.AgentGameSetting
	if err := database.DBConn.Where("agent_id = ? AND game_code = ?", id, game).Take(&s).Error; err != nil {
		t.Fatal(err)
	}
	return s.StatusGame
}

func TestRateStatusValidation(t *testing.T) { // RE8 สิทธิ์ / รูปแบบ
	app := setup2(t)
	c := buildChain(t, app)
	viewTok := superadminSub(t, app, "viewer", `{"rate":"view"}`)
	ok := map[string]any{"game_code": "coin_toss", "status_game": false, "passcode": passcode}
	expect(t, call(t, app, "POST", rateStatusPath, ok, c.comTok), 200, 402303)
	expect(t, call(t, app, "POST", rateStatusPath, ok, viewTok), 200, 402303)
	expect(t, call(t, app, "POST", rateStatusPath, map[string]any{"game_code": "coin_toss", "status_game": false}, c.saTok), 200, 422)
	expect(t, call(t, app, "POST", rateStatusPath, map[string]any{"game_code": "coin_toss", "status_game": false, "passcode": "654321"}, c.saTok), 200, 401204)
	expect(t, call(t, app, "POST", rateStatusPath, map[string]any{"game_code": "baccarat", "status_game": false, "passcode": passcode}, c.saTok), 200, 406401)
	expect(t, call(t, app, "POST", rateStatusPath, map[string]any{"game_code": "coin_toss", "passcode": passcode}, c.saTok), 200, 422)
	expect(t, call(t, app, "POST", rateStatusPath, map[string]any{"game_code": "coin_toss", "status_game": nil, "passcode": passcode}, c.saTok), 200, 422)
	if n := countRows(t, &models.AccountChangeLog{}, "action = ?", models.ChangeUpdateGames); n != 0 {
		t.Fatalf("log %d แถว", n)
	}
}

func TestRateStatusSwitch(t *testing.T) { // RE8 · QS1 ทุกบัญชี Superadmin · QS3 log UPDATE_GAMES · ไหลลงทั้งสาย
	app := setup2(t)
	c := buildChain(t, app)
	var sa models.UserAgent
	database.DBConn.Where("username = ?", "superadmin").Take(&sa)
	sa2 := seedSuperadmin2(t)
	// share01 ปิด coin_toss เองไว้ก่อน (ปิดโดย comp01) — เปิดสวิตช์คืนแล้วต้องยังปิด
	expect(t, call(t, app, "POST", agentDetailUpdatePath, map[string]any{"id": c.share.ID, "status_game": map[string]bool{"coin_toss": false}}, c.comTok), 200, 200)
	logsBefore := countRows(t, &models.AccountChangeLog{}, "action = ?", models.ChangeUpdateGames)

	if st := listStatus(t, app, c.saTok); !st["coin_toss"] || !st["scratch_card"] || len(st) != 3 {
		t.Fatalf("list ก่อนปิด %v", st)
	}
	if !effectiveGame(t, app, c.saTok, c.com.ID, "coin_toss") {
		t.Fatal("comp01 ต้องเปิด coin_toss ก่อนปิดสวิตช์")
	}

	// ปิด: Superadmin ทุกบัญชี · log 1 แถวต่อบัญชี
	if !setGameStatus(t, app, c.saTok, "coin_toss", false) {
		t.Fatal("ปิดครั้งแรกต้อง changed")
	}
	if superadminStatus(t, sa.ID, "coin_toss") || superadminStatus(t, sa2.ID, "coin_toss") || !superadminStatus(t, sa.ID, "scratch_card") {
		t.Fatal("status_game ของ Superadmin ไม่ตรง")
	}
	var logs []models.AccountChangeLog
	database.DBConn.Where("action = ? AND target_id IN ?", models.ChangeUpdateGames, []uint{sa.ID, sa2.ID}).Order("target_id").Find(&logs)
	if len(logs) != 2 || logs[0].ActorUsername != "superadmin" || logs[0].TargetType != "AGENT" || logs[1].TargetUsername != "superadmin2" ||
		!logGame(logs[0].OldValue, "coin_toss", true) || !logGame(logs[0].NewValue, "coin_toss", false) || !logGame(logs[1].NewValue, "coin_toss", false) ||
		logs[0].RequestID == nil || *logs[0].RequestID == "" || *logs[0].RequestID != *logs[1].RequestID {
		t.Fatalf("log %+v", logs)
	}
	if st := listStatus(t, app, c.saTok); st["coin_toss"] || !st["scratch_card"] || !st["rock_paper_scissors"] {
		t.Fatalf("list หลังปิด %v", st)
	}
	if effectiveGame(t, app, c.saTok, c.com.ID, "coin_toss") || !effectiveGame(t, app, c.saTok, c.com.ID, "scratch_card") {
		t.Fatal("comp01 ต้องปิด coin_toss ตามสวิตช์")
	}

	// ส่งซ้ำ: changed false · ไม่มี log เพิ่ม
	if setGameStatus(t, app, c.saTok, "coin_toss", false) {
		t.Fatal("ค่าเดิมต้อง changed false")
	}
	if n := countRows(t, &models.AccountChangeLog{}, "action = ?", models.ChangeUpdateGames); n != logsBefore+2 {
		t.Fatalf("log %d แถว ต้อง %d", n, logsBefore+2)
	}

	// Superadmin บัญชีเดียวเปิดอยู่ = RE1 ยังเป็น false · sub ที่มี rate edit เปิดคืน → แก้แค่บัญชีที่ต่าง
	database.DBConn.Model(&models.AgentGameSetting{}).Where("agent_id = ? AND game_code = ?", sa2.ID, "coin_toss").Update("status_game", true)
	if st := listStatus(t, app, c.saTok); st["coin_toss"] {
		t.Fatalf("list ต้อง false เมื่อ Superadmin บางบัญชีปิด %v", st)
	}
	editTok := superadminSub(t, app, "editor", `{"rate":"edit"}`)
	if !setGameStatus(t, app, editTok, "coin_toss", true) {
		t.Fatal("เปิดคืนต้อง changed")
	}
	var last models.AccountChangeLog
	database.DBConn.Where("action = ?", models.ChangeUpdateGames).Order("id DESC").Take(&last)
	if last.ActorType != models.AccountTypeSub || last.ActorUsername != "superadmin@editor" || last.TargetID != sa.ID {
		t.Fatalf("log sub %+v", last)
	}
	if n := countRows(t, &models.AccountChangeLog{}, "action = ?", models.ChangeUpdateGames); n != logsBefore+3 {
		t.Fatalf("log %d แถว ต้อง %d", n, logsBefore+3)
	}
	if st := listStatus(t, app, c.saTok); !st["coin_toss"] {
		t.Fatalf("list หลังเปิด %v", st)
	}
	// เปิดคืน: comp01 กลับมาเปิด · agent01 ยังปิดเพราะ share01 ปิดเอง (ไม่แก้ค่าสายล่าง)
	if !effectiveGame(t, app, c.saTok, c.com.ID, "coin_toss") || effectiveGame(t, app, c.comTok, c.agent.ID, "coin_toss") {
		t.Fatal("effective หลังเปิดคืนไม่ตรง")
	}
}

// logGame — old_value / new_value ของ log UPDATE_GAMES มี status_game[game] = want
func logGame(v *string, game string, want bool) bool {
	if v == nil {
		return false
	}
	var d struct {
		StatusGame map[string]bool `json:"status_game"`
	}
	if err := json.Unmarshal([]byte(*v), &d); err != nil {
		return false
	}
	got, ok := d.StatusGame[game]
	return ok && got == want && len(d.StatusGame) == 1
}

func TestRateStatusMissingRow(t *testing.T) { // lead G-S1: Superadmin ไม่มีแถวของเกม → เพิ่มแถวแล้วปิด / เปิดตามสั่ง (ไม่ล้มเงียบ)
	app := setup2(t)
	c := buildChain(t, app)
	sa2 := seedSuperadmin2(t)
	database.DBConn.Where("agent_id = ? AND game_code = ?", sa2.ID, "coin_toss").Delete(&models.AgentGameSetting{})

	// เปิดอยู่แล้วทุกบัญชี (ไม่มีแถว = เปิด) → ส่งเปิด: เพิ่มแถว · changed false · ไม่มี log
	if setGameStatus(t, app, c.saTok, "coin_toss", true) {
		t.Fatal("เปิดซ้ำต้อง changed false")
	}
	var row models.AgentGameSetting
	if err := database.DBConn.Where("agent_id = ? AND game_code = ?", sa2.ID, "coin_toss").Take(&row).Error; err != nil {
		t.Fatalf("ต้องเพิ่มแถวที่ขาด: %v", err)
	}
	if !row.StatusGame || row.PTFromParent != 100 || !row.Status || row.Category != "minigame" || row.CreatedBy != "superadmin2" {
		t.Fatalf("แถวที่เพิ่ม %+v", row)
	}
	if n := countRows(t, &models.AccountChangeLog{}, "action = ?", models.ChangeUpdateGames); n != 0 {
		t.Fatalf("log %d แถว", n)
	}

	// ลบอีกครั้งแล้วปิด: เพิ่มแถวเป็น false · log ของบัญชีนั้น (เดิม true)
	database.DBConn.Where("agent_id = ? AND game_code = ?", sa2.ID, "coin_toss").Delete(&models.AgentGameSetting{})
	if !setGameStatus(t, app, c.saTok, "coin_toss", false) {
		t.Fatal("ปิดต้อง changed")
	}
	if superadminStatus(t, sa2.ID, "coin_toss") {
		t.Fatal("แถวที่เพิ่มต้องปิด")
	}
	var logs []models.AccountChangeLog
	database.DBConn.Where("action = ? AND target_id = ?", models.ChangeUpdateGames, sa2.ID).Find(&logs)
	if len(logs) != 1 || !logGame(logs[0].OldValue, "coin_toss", true) || !logGame(logs[0].NewValue, "coin_toss", false) {
		t.Fatalf("log %+v", logs)
	}
	if st := listStatus(t, app, c.saTok); st["coin_toss"] {
		t.Fatalf("list %v", st)
	}
}

// statusRow — status_* ของเกมจาก RE1
type statusRow struct {
	GameCode        string `json:"game_code"`
	StatusGame      bool   `json:"status_game"`
	StatusUpdatedAt string `json:"status_updated_at"`
	StatusUpdatedBy string `json:"status_updated_by"`
}

func listStatusRows(t *testing.T, app *fiber.App, tok string) map[string]statusRow {
	t.Helper()
	r := call(t, app, "POST", rateListPath, map[string]any{}, tok)
	expect(t, r, 200, 200)
	var rows []statusRow
	_ = json.Unmarshal(r.Data, &rows)
	out := map[string]statusRow{}
	for _, row := range rows {
		out[row.GameCode] = row
	}
	return out
}

func TestRateStatusLastAction(t *testing.T) { // RE1 / RE8 status_updated_at / status_updated_by (2026-10-14)
	app := setup2(t)
	c := buildChain(t, app)

	// ยังไม่เคยเปิด / ปิด = "" ทุกเกม
	for code, row := range listStatusRows(t, app, c.saTok) {
		if row.StatusUpdatedAt != "" || row.StatusUpdatedBy != "" {
			t.Fatalf("%s ยังไม่เคยเปลี่ยน %+v", code, row)
		}
	}

	// Superadmin ปิด coin_toss → response + RE1 บอกเวลา + คนทำ · เกมอื่นยังว่าง
	r := call(t, app, "POST", rateStatusPath, map[string]any{"game_code": "coin_toss", "status_game": false, "passcode": passcode}, c.saTok)
	expect(t, r, 200, 200)
	var res struct {
		Changed         bool   `json:"changed"`
		StatusUpdatedAt string `json:"status_updated_at"`
		StatusUpdatedBy string `json:"status_updated_by"`
	}
	_ = json.Unmarshal(r.Data, &res)
	if !res.Changed || res.StatusUpdatedBy != "superadmin" || res.StatusUpdatedAt == "" {
		t.Fatalf("response %+v", res)
	}
	rows := listStatusRows(t, app, c.saTok)
	if rows["coin_toss"].StatusUpdatedBy != "superadmin" || rows["coin_toss"].StatusUpdatedAt != res.StatusUpdatedAt || rows["scratch_card"].StatusUpdatedBy != "" {
		t.Fatalf("list %+v", rows)
	}
	first := res.StatusUpdatedAt

	// ส่งซ้ำ (changed false) = คืนครั้งล่าสุดเดิม ไม่เปลี่ยน
	r = call(t, app, "POST", rateStatusPath, map[string]any{"game_code": "coin_toss", "status_game": false, "passcode": passcode}, c.saTok)
	_ = json.Unmarshal(r.Data, &res)
	if res.Changed || res.StatusUpdatedAt != first || res.StatusUpdatedBy != "superadmin" {
		t.Fatalf("ส่งซ้ำ %+v", res)
	}

	// sub ที่มี rate edit เปิดคืน → คนทำ = owner@name · เวลาใหม่
	editTok := superadminSub(t, app, "editor", `{"rate":"edit"}`)
	r = call(t, app, "POST", rateStatusPath, map[string]any{"game_code": "coin_toss", "status_game": true, "passcode": passcode}, editTok)
	expect(t, r, 200, 200)
	_ = json.Unmarshal(r.Data, &res)
	if !res.Changed || res.StatusUpdatedBy != "superadmin@editor" || res.StatusUpdatedAt == first {
		t.Fatalf("sub เปิดคืน %+v", res)
	}
	if row := listStatusRows(t, app, c.saTok)["coin_toss"]; !row.StatusGame || row.StatusUpdatedBy != "superadmin@editor" || row.StatusUpdatedAt != res.StatusUpdatedAt {
		t.Fatalf("list หลังเปิดคืน %+v", row)
	}

	// แก้ status_game ของ Company ผ่าน agents/detail/update (UPDATE_GAMES target ไม่ใช่ Superadmin) ไม่นับ
	expect(t, call(t, app, "POST", agentDetailUpdatePath, map[string]any{"id": c.com.ID, "status_game": map[string]bool{"scratch_card": false}}, c.saTok), 200, 200)
	if row := listStatusRows(t, app, c.saTok)["scratch_card"]; row.StatusUpdatedBy != "" {
		t.Fatalf("log ของ Company ต้องไม่นับ %+v", row)
	}
}
