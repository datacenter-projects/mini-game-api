//go:build integration

// API test ของ agent_management phase 4 (แก้บัญชี) ตาม docs/modules/agent_management.md หัวข้อ 7
package backoffice_test

import (
	"encoding/json"
	"strings"
	"testing"

	"app/app/models"
	"app/app/repository/postgres"
	"app/pkg/utils"
	"app/platform/database"
)

const (
	agentInfoPath        = "/api/v1/bo/pr/manage/agents/update-info"
	memberInfoPath       = "/api/v1/bo/pr/manage/members/update-info"
	agentStatusPath      = "/api/v1/bo/pr/manage/agents/update-status"
	memberStatusPath     = "/api/v1/bo/pr/manage/members/update-status"
	updatePTPath         = "/api/v1/bo/pr/manage/agents/update-pt"
	updateCommissionPath = "/api/v1/bo/pr/manage/members/update-commission"
	updateHoldPath       = "/api/v1/bo/pr/manage/agents/update-hold"
	updateGamesPath      = "/api/v1/bo/pr/manage/agents/update-games"
)

func ptBody(id uint, give, force, remain, commission float64) map[string]any {
	return map[string]any{"id": id, "pt": childPT(give, force, remain, commission)}
}

func holdBody(pt float64) map[string]any {
	return map[string]any{"pt": map[string]any{"minigame": map[string]any{"pt": pt}}}
}

func gameSetting(t *testing.T, agentID uint) models.AgentGameSetting {
	t.Helper()
	var s models.AgentGameSetting
	database.DBConn.Where("agent_id = ? AND game_code = ?", agentID, "coin_toss").Take(&s)
	return s
}

func TestUpdatePT(t *testing.T) { // MGMT-18 – MGMT-25
	app := setup2(t)
	c := buildChain(t, app) // comp01 ได้รับ 90 → share01 70 → agent01 60

	// MGMT-24 ตัวอย่าง spec: share01 ถือ 30 · ให้ agent01 60 → ลดค่าที่ให้ share01 ได้ต่ำสุด 60
	expect(t, call(t, app, "POST", updateHoldPath, holdBody(30), c.shareTok), 200, 200)
	r := call(t, app, "POST", updatePTPath, ptBody(c.share.ID, 55, 0, 0, 0.5), c.comTok)
	expect(t, r, 200, 402306)
	if !strings.Contains(r.Msg, "60") {
		t.Fatalf("msg ต้องบอกค่าต่ำสุด 60: %s", r.Msg)
	}
	if s := gameSetting(t, c.share.ID); s.PTFromParentBP != 7000 {
		t.Fatalf("ถูกปฏิเสธต้องไม่เปลี่ยน ได้ %d", s.PTFromParentBP)
	}
	expect(t, call(t, app, "POST", updatePTPath, ptBody(c.share.ID, 60, 5, 5, 0.3), c.comTok), 200, 200)
	s := gameSetting(t, c.share.ID)
	if s.PTFromParentBP != 6000 || s.PTBP != 3000 || s.ForceBP != 500 || s.RemainBP != 500 || s.CommissionBP != 30 {
		t.Fatalf("after update %+v", s)
	}
	if g := gameSetting(t, c.agent.ID); g.PTFromParentBP != 6000 {
		t.Fatal("ลูกของลูกต้องไม่เปลี่ยน")
	}

	// เพิ่มได้ถึงค่าที่ผู้สร้างได้รับ (90) · เกิน = 402305
	expect(t, call(t, app, "POST", updatePTPath, ptBody(c.share.ID, 90, 0, 0, 0.5), c.comTok), 200, 200)
	expect(t, call(t, app, "POST", updatePTPath, ptBody(c.share.ID, 90.5, 0, 0, 0.5), c.comTok), 200, 402305)
	expect(t, call(t, app, "POST", updatePTPath, ptBody(c.share.ID, 60, 70, 0, 0.5), c.comTok), 200, 402308) // MGMT-25
	expect(t, call(t, app, "POST", updatePTPath, ptBody(c.share.ID, 60, 0, 0, 1.1), c.comTok), 200, 402309)

	// แก้ได้เฉพาะผู้สร้างโดยตรง (MGMT-23) · นอกสาย 402402
	expect(t, call(t, app, "POST", updatePTPath, ptBody(c.agent.ID, 50, 0, 0, 0), c.comTok), 200, 402304)
	_, otherTok := mustCreate(t, app, c.saTok, agentBody("COMPANY_TRANSFER", "othercom", nil, childPT(50, 0, 0, 0)))
	expect(t, call(t, app, "POST", updatePTPath, ptBody(c.share.ID, 50, 0, 0, 0), otherTok), 200, 402402)

	// field ไม่อยู่ในเส้นนั้น = 422 · ไม่ส่ง id = 422
	b := ptBody(c.share.ID, 60, 0, 0, 0)
	b["pt"].(map[string]any)["minigame"].(map[string]any)["pt"] = 10
	expect(t, call(t, app, "POST", updatePTPath, b, c.comTok), 200, 422)
	expect(t, call(t, app, "POST", updatePTPath, map[string]any{"pt": childPT(60, 0, 0, 0)}, c.comTok), 200, 422)
	b = holdBody(10)
	b["pt"].(map[string]any)["minigame"].(map[string]any)["pt_from_parent"] = 10
	expect(t, call(t, app, "POST", updateHoldPath, b, c.agentTok), 200, 422)

	// MGMT-16: ผู้สร้างค่า PT ไม่เปลี่ยน · ผู้แก้ล่าสุด = username ของคนที่แก้ (sub = owner@name) ทุกเกมในระบบ
	sub := createSubAPI(t, app, c.comTok, "staff", map[string]string{"member": "edit", "pt": "edit"})
	subTok := readyToken(t, app, models.AccountTypeSub, sub.ID, "comp01@staff")
	before := gameSetting(t, c.share.ID)
	expect(t, call(t, app, "POST", updatePTPath, ptBody(c.share.ID, 80, 0, 0, 0.5), subTok), 200, 200)
	var rows []models.AgentGameSetting
	database.DBConn.Where("agent_id = ?", c.share.ID).Find(&rows)
	for _, s := range rows {
		if s.CreatedBy != "comp01" || s.UpdatedBy != "comp01@staff" || !s.CreatedAt.Equal(before.CreatedAt) || !s.UpdatedAt.After(before.UpdatedAt) {
			t.Fatalf("audit %s %+v", s.GameCode, s)
		}
	}
	r = call(t, app, "POST", agentDetailPath, map[string]any{"id": c.share.ID}, c.comTok)
	expect(t, r, 200, 200)
	var d struct {
		PT map[string]map[string]any `json:"pt"`
	}
	_ = json.Unmarshal(r.Data, &d)
	if mg := d.PT["minigame"]; mg["created_by"] != "comp01" || mg["updated_by"] != "comp01@staff" || mg["pt_from_parent"] != float64(80) {
		t.Fatalf("detail pt %+v", d.PT)
	}

	// แก้ค่าถือของตัวเอง → updated_by = ตัวเอง
	expect(t, call(t, app, "POST", updateHoldPath, holdBody(20), c.shareTok), 200, 200)
	if s := gameSetting(t, c.share.ID); s.UpdatedBy != "share01" || s.CreatedBy != "comp01" {
		t.Fatalf("hold audit %+v", s)
	}
}

func TestUpdateHold(t *testing.T) { // MGMT-19, MGMT-22
	app := setup2(t)
	c := buildChain(t, app)
	expect(t, call(t, app, "POST", updateHoldPath, holdBody(40), c.agentTok), 200, 200)
	if s := gameSetting(t, c.agent.ID); s.PTBP != 4000 {
		t.Fatalf("pt_bp %d", s.PTBP)
	}
	expect(t, call(t, app, "POST", updateHoldPath, holdBody(61), c.agentTok), 200, 402305)
	expect(t, call(t, app, "POST", updateHoldPath, holdBody(40.25), c.agentTok), 200, 422)

	_, masterTok := mustCreate(t, app, c.saTok, agentBody("COMPANY_SEAMLESS_MASTER", "master01", nil, childPT(80, 0, 0, 0)))
	expect(t, call(t, app, "POST", updateHoldPath, holdBody(1), masterTok), 200, 402307)
	expect(t, call(t, app, "POST", updateHoldPath, holdBody(0), masterTok), 200, 200)
}

func TestUpdateStatus(t *testing.T) { // MGMT-30, MGMT-31
	app := setup2(t)
	c := buildChain(t, app)
	status := func(tok string, id uint) string {
		t.Helper()
		r := call(t, app, "POST", agentDetailPath, map[string]any{"id": id}, tok)
		expect(t, r, 200, 200)
		var d struct {
			Status string `json:"status"`
		}
		_ = json.Unmarshal(r.Data, &d)
		return d.Status
	}

	// Company ระงับ Share → Share และ Agent ใต้ Share เป็น SUSPENDED · Company ไม่กระทบ
	expect(t, call(t, app, "POST", agentStatusPath, map[string]any{"id": c.share.ID, "status": "SUSPENDED"}, c.comTok), 200, 200)
	if status(c.comTok, c.share.ID) != "SUSPENDED" || status(c.comTok, c.agent.ID) != "SUSPENDED" {
		t.Fatal("share / agent ต้อง SUSPENDED")
	}

	// หัวสายถูกระงับ · ผู้สร้างตั้ง Agent เป็น ACTIVE → ยังแสดง SUSPENDED (ใช้ token ของ company ตั้งแทนไม่ได้ — ไม่ใช่ผู้สร้างโดยตรง)
	expect(t, call(t, app, "POST", agentStatusPath, map[string]any{"id": c.agent.ID, "status": "LOCKED"}, c.comTok), 200, 402304)
	setStatus(t, c.share.ID, models.AgentStatusActive)
	expect(t, call(t, app, "POST", agentStatusPath, map[string]any{"id": c.agent.ID, "status": "LOCKED"}, c.shareTok), 200, 200)
	if status(c.comTok, c.agent.ID) != "LOCKED" || status(c.comTok, c.share.ID) != "ACTIVE" {
		t.Fatal("ล็อก agent ต้องไม่กระทบ share")
	}
	expect(t, call(t, app, "POST", agentStatusPath, map[string]any{"id": c.agent.ID, "status": "ACTIVE"}, c.shareTok), 200, 200)
	setStatus(t, c.share.ID, models.AgentStatusSuspended)
	if status(c.comTok, c.agent.ID) != "SUSPENDED" {
		t.Fatal("หัวสายถูกระงับ ลูกต้องแสดง SUSPENDED")
	}
	expect(t, call(t, app, "POST", agentStatusPath, map[string]any{"id": c.share.ID, "status": "PAUSED"}, c.comTok), 200, 422)

	// Member (MGMT-09A)
	setStatus(t, c.share.ID, models.AgentStatusActive)
	r := call(t, app, "POST", createMemberPath, memberBody("mem02", 0.3), c.agentTok)
	expect(t, r, 200, 200)
	var m created
	_ = json.Unmarshal(r.Data, &m)
	expect(t, call(t, app, "POST", memberStatusPath, map[string]any{"id": m.ID, "status": "LOCKED"}, c.agentTok), 200, 200)
	expect(t, call(t, app, "POST", memberStatusPath, map[string]any{"id": m.ID, "status": "ACTIVE"}, c.shareTok), 200, 402304)
	var got models.Member
	database.DBConn.Where("id = ?", m.ID).Take(&got)
	if got.Status != models.AgentStatusLocked {
		t.Fatalf("member status %s", got.Status)
	}
}

func TestUpdateInfoAndCommission(t *testing.T) { // MGMT-08, MGMT-09, MGMT-09A, MGMT-21, MGMT-60
	app := setup2(t)
	c := buildChain(t, app)
	expect(t, call(t, app, "POST", agentInfoPath, map[string]any{"id": c.agent.ID, "name": "สมชาย01", "phone": "0811111111"}, c.shareTok), 200, 200)
	var a models.UserAgent
	database.DBConn.Where("id = ?", c.agent.ID).Take(&a)
	if a.Name == nil || *a.Name != "สมชาย01" || a.Phone == nil || *a.Phone != "0811111111" {
		t.Fatalf("info %+v", a)
	}
	expect(t, call(t, app, "POST", agentInfoPath, map[string]any{"id": c.share.ID, "name": "share01", "phone": "0811111111"}, c.comTok), 200, 402403)
	expect(t, call(t, app, "POST", agentInfoPath, map[string]any{"id": c.agent.ID, "name": "agent01", "phone": "0811111111"}, c.shareTok), 200, 200) // เบอร์เดิมของตัวเอง
	expect(t, call(t, app, "POST", agentInfoPath, map[string]any{"id": c.agent.ID, "name": "agent01", "phone": nil}, c.shareTok), 200, 422)
	expect(t, call(t, app, "POST", agentInfoPath, map[string]any{"id": c.agent.ID, "name": "agent01"}, c.shareTok), 200, 422)
	expect(t, call(t, app, "POST", agentInfoPath, map[string]any{"id": c.agent.ID, "name": "agent01", "phone": ""}, c.shareTok), 200, 200)
	database.DBConn.Where("id = ?", c.agent.ID).Take(&a)
	if a.Phone != nil {
		t.Fatal(`phone "" ต้องเก็บเป็น NULL`)
	}

	r := call(t, app, "POST", createMemberPath, memberBody("mem01", 0.3), c.agentTok)
	expect(t, r, 200, 200)
	var m created
	_ = json.Unmarshal(r.Data, &m)
	expect(t, call(t, app, "POST", memberInfoPath, map[string]any{"id": m.ID, "name": "ใจดี", "phone": ""}, c.agentTok), 200, 200)
	expect(t, call(t, app, "POST", updateCommissionPath, map[string]any{"id": m.ID, "pt": map[string]any{"minigame": map[string]any{"commission_percent": 0.5}}}, c.agentTok), 200, 200)
	var ms models.MemberGameSetting
	database.DBConn.Where("member_id = ?", m.ID).Take(&ms)
	if ms.CommissionBP != 50 {
		t.Fatalf("commission %d", ms.CommissionBP)
	}
	expect(t, call(t, app, "POST", updateCommissionPath, map[string]any{"id": m.ID, "pt": map[string]any{"minigame": map[string]any{"commission_percent": 1.1}}}, c.agentTok), 200, 402309)
	expect(t, call(t, app, "POST", updateCommissionPath, map[string]any{"id": m.ID, "pt": map[string]any{"minigame": map[string]any{"commission_percent": 0.5, "pt_from_parent": 10}}}, c.agentTok), 200, 422)

	// MGMT-60: ทุกการแก้มี log ค่าเก่า / ใหม่ · ไม่มีรหัสผ่าน
	var logs []models.AccountChangeLog
	database.DBConn.Where("action IN ?", []models.AccountChangeAction{models.ChangeUpdateInfo, models.ChangeUpdatePT}).Order("id").Find(&logs)
	if len(logs) != 5 {
		t.Fatalf("logs = %d, want 5", len(logs))
	}
	if logs[0].OldValue == nil || !strings.Contains(*logs[0].NewValue, "สมชาย01") || logs[0].ActorUsername != "share01" {
		t.Fatalf("log %+v", logs[0])
	}
}

func TestUpdateSubPermission(t *testing.T) { // MGMT-51
	app := setup2(t)
	c := buildChain(t, app)
	hash, _ := utils.HashPassword(mgPassword)
	sub := models.Subaccount{AgentID: c.com.ID, Username: "comp01@staff", PasswordHash: hash, Status: models.AgentStatusActive,
		Permissions: `{"member":"edit"}`}
	if err := postgres.CreateSubaccountRepository(database.DBConn, &sub); err != nil {
		t.Fatal(err)
	}
	tok := readyToken(t, app, models.AccountTypeSub, sub.ID, sub.Username)
	expect(t, call(t, app, "POST", agentInfoPath, map[string]any{"id": c.share.ID, "name": "share01", "phone": ""}, tok), 200, 200)
	expect(t, call(t, app, "POST", updatePTPath, ptBody(c.share.ID, 80, 0, 0, 0), tok), 200, 402303)
	expect(t, call(t, app, "POST", updateHoldPath, holdBody(50), tok), 200, 402303)

	setCols(t, models.AccountTypeSub, sub.ID, map[string]any{"permissions": `{"pt":"edit"}`})
	expect(t, call(t, app, "POST", updatePTPath, ptBody(c.share.ID, 80, 0, 0, 0), tok), 200, 200)
	expect(t, call(t, app, "POST", updateHoldPath, holdBody(50), tok), 200, 200) // ค่าถือของเจ้าของ
	if s := gameSetting(t, c.com.ID); s.PTBP != 5000 {
		t.Fatalf("owner pt_bp %d", s.PTBP)
	}
	expect(t, call(t, app, "POST", agentInfoPath, map[string]any{"id": c.share.ID, "name": "share01", "phone": ""}, tok), 200, 402303)
}

func TestUpdateGames(t *testing.T) { // MGMT-20
	app := setup2(t)
	c := buildChain(t, app)
	statusGame := func(agentID uint) map[string]bool {
		t.Helper()
		var rows []models.AgentGameSetting
		database.DBConn.Where("agent_id = ?", agentID).Find(&rows)
		out := map[string]bool{}
		for _, s := range rows {
			out[s.GameCode] = s.StatusGame
		}
		return out
	}
	before := gameSetting(t, c.share.ID)

	// ผู้สร้างปิดเกมให้ลูกตรง · ส่งเฉพาะเกมที่เปลี่ยน · ค่า PT และผู้แก้ PT ไม่เปลี่ยน
	body := map[string]any{"id": c.share.ID, "status_game": map[string]any{"scratch_card": false}}
	expect(t, call(t, app, "POST", updateGamesPath, body, c.comTok), 200, 200)
	if sg := statusGame(c.share.ID); sg["scratch_card"] || !sg["coin_toss"] || !sg["rock_paper_scissors"] {
		t.Fatalf("status_game %v", sg)
	}
	if after := gameSetting(t, c.share.ID); after.UpdatedBy != before.UpdatedBy || !after.UpdatedAt.Equal(before.UpdatedAt) || !after.Status {
		t.Fatalf("ค่า PT ต้องไม่เปลี่ยน %+v", after)
	}
	if sg := statusGame(c.agent.ID); !sg["scratch_card"] {
		t.Fatal("ไม่ส่งต่อลงสายล่าง — สายล่างเช็คตอนเล่น")
	}
	r := call(t, app, "POST", agentDetailPath, map[string]any{"id": c.share.ID}, c.comTok)
	var d struct {
		StatusGame map[string]bool `json:"status_game"`
	}
	_ = json.Unmarshal(r.Data, &d)
	if d.StatusGame["scratch_card"] || !d.StatusGame["coin_toss"] {
		t.Fatalf("detail status_game %v", d.StatusGame)
	}

	// เปิดกลับได้ · log ค่าเก่า / ใหม่
	body["status_game"] = map[string]any{"scratch_card": true}
	expect(t, call(t, app, "POST", updateGamesPath, body, c.comTok), 200, 200)
	var logs []models.AccountChangeLog
	database.DBConn.Where("action = ?", models.ChangeUpdateGames).Order("id").Find(&logs)
	var oldV, newV struct {
		StatusGame map[string]bool `json:"status_game"`
	}
	if len(logs) != 2 || logs[1].OldValue == nil || logs[1].NewValue == nil || logs[1].ActorUsername != "comp01" {
		t.Fatalf("logs %+v", logs)
	}
	_ = json.Unmarshal([]byte(*logs[1].OldValue), &oldV)
	_ = json.Unmarshal([]byte(*logs[1].NewValue), &newV)
	if v, ok := oldV.StatusGame["scratch_card"]; !ok || v || !newV.StatusGame["scratch_card"] || len(newV.StatusGame) != 1 {
		t.Fatalf("log values %s → %s", *logs[1].OldValue, *logs[1].NewValue)
	}

	// ไม่ใช่ผู้สร้างโดยตรง 402304 · นอกสาย 402402 · validation 422
	expect(t, call(t, app, "POST", updateGamesPath, map[string]any{"id": c.agent.ID, "status_game": map[string]any{"coin_toss": false}}, c.comTok), 200, 402304)
	_, otherTok := mustCreate(t, app, c.saTok, agentBody("COMPANY_TRANSFER", "othercom", nil, childPT(50, 0, 0, 0)))
	expect(t, call(t, app, "POST", updateGamesPath, map[string]any{"id": c.share.ID, "status_game": map[string]any{"coin_toss": false}}, otherTok), 200, 402402)
	for _, b := range []map[string]any{
		{"id": c.share.ID, "status_game": map[string]any{}},
		{"id": c.share.ID, "status_game": map[string]any{"poker": false}},
		{"id": c.share.ID, "status_game": map[string]any{"coin_toss": nil}},
		{"status_game": map[string]any{"coin_toss": false}},
	} {
		expect(t, call(t, app, "POST", updateGamesPath, b, c.comTok), 200, 422)
	}
}
