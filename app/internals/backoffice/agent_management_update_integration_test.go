//go:build integration

// API test ของ agent_management phase 4 (แก้บัญชี) ตาม docs/modules/agent_management.md หัวข้อ 7
package backoffice_test

import (
	"encoding/json"
	"strings"
	"testing"

	"app/app/models"
	agentAuthPostgres "app/app/repository/postgres/agent_auth"
	"app/pkg/utils"
	"app/platform/database"

	"github.com/gofiber/fiber/v2"
)

const (
	agentInfoPath      = "/api/v1/bo/pr/manage/agents/update-info"
	memberInfoPath     = "/api/v1/bo/pr/manage/members/update-info"
	agentStatusPath    = "/api/v1/bo/pr/manage/agents/update-status"
	memberStatusPath   = "/api/v1/bo/pr/manage/members/update-status"
	updatePTPath       = "/api/v1/bo/pr/manage/agents/update-pt"
	updateMemberPTPath = "/api/v1/bo/pr/manage/members/update-pt"
	updateGamesPath    = "/api/v1/bo/pr/manage/agents/update-games"
)

func ptBody(id uint, give, force, remain, commission float64) map[string]any {
	return map[string]any{"id": id, "pt": childPT(give, force, remain, commission)}
}

// createMemberWithPT — Member ใต้บัญชีของ tok ที่ผู้สร้างถือสู้ pt (member_management MGMT-21)
func createMemberWithPT(t *testing.T, app *fiber.App, tok, username string, pt float64) created {
	t.Helper()
	b := memberBody(username, 0)
	b["pt"].(map[string]any)["minigame"].(map[string]any)["pt"] = pt
	r := call(t, app, "POST", createMemberPath, b, tok)
	expect(t, r, 200, 200)
	var d created
	_ = json.Unmarshal(r.Data, &d)
	return d
}

func memberPTBody(id uint, pt, commission float64) map[string]any {
	return map[string]any{"id": id, "pt": map[string]any{"minigame": map[string]any{"pt": pt, "commission_percent": commission}}}
}

func memberSetting(t *testing.T, memberID uint) models.UserMemberGameSetting {
	t.Helper()
	var s models.UserMemberGameSetting
	database.DBConn.Where("user_member_id = ? AND game_code = ?", memberID, "coin_toss").Take(&s)
	return s
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

	// MGMT-24 ตัวอย่าง spec: share01 ให้ agent01 60 · ถือสู้ Member ของตัวเอง 30 → ลดค่าที่ให้ share01 ได้ต่ำสุด 60
	createMemberWithPT(t, app, c.shareTok, "memshare30", 30)
	r := call(t, app, "POST", updatePTPath, ptBody(c.share.ID, 55, 0, 0, 0.5), c.comTok)
	expect(t, r, 200, 402306)
	if !strings.Contains(r.Msg, "60") {
		t.Fatalf("msg ต้องบอกค่าต่ำสุด 60: %s", r.Msg)
	}
	if s := gameSetting(t, c.share.ID); s.PTFromParent != 70 {
		t.Fatalf("ถูกปฏิเสธต้องไม่เปลี่ยน ได้ %v", s.PTFromParent)
	}
	expect(t, call(t, app, "POST", updatePTPath, ptBody(c.share.ID, 60, 5, 5, 0.3), c.comTok), 200, 200)
	s := gameSetting(t, c.share.ID)
	if s.PTFromParent != 60 || s.Force != 5 || s.Remain != 5 || s.Commission != 0.3 {
		t.Fatalf("after update %+v", s)
	}
	if g := gameSetting(t, c.agent.ID); g.PTFromParent != 60 {
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

}

func TestUpdateHoldRemoved(t *testing.T) { // ลบ 2026-10-09 — ถือสู้กับ Member ตั้งต่อ Member
	app := setup2(t)
	c := buildChain(t, app)
	r := call(t, app, "POST", "/api/v1/bo/pr/manage/agents/update-hold", map[string]any{"pt": map[string]any{"minigame": map[string]any{"pt": 40}}}, c.agentTok)
	if r.Status != 404 {
		t.Fatalf("update-hold ต้องไม่มีแล้ว ได้ HTTP %d code %d", r.Status, r.Code)
	}
	r = call(t, app, "POST", agentDetailPath, map[string]any{"id": c.agent.ID}, c.shareTok)
	expect(t, r, 200, 200)
	var d struct {
		PT map[string]map[string]any `json:"pt"`
	}
	_ = json.Unmarshal(r.Data, &d)
	if _, has := d.PT["minigame"]["pt"]; has {
		t.Fatalf("detail ต้องไม่มี pt ของบัญชีแล้ว %+v", d.PT)
	}
}

func TestUpdatePTMemberPT(t *testing.T) { // MGMT-24 R1 – R3 (member_management หัวข้อ 7)
	app := setup2(t)
	c := buildChain(t, app) // agent01 ได้รับ 60
	ma := createMemberWithPT(t, app, c.agentTok, "memhold50", 50)
	mb := createMemberWithPT(t, app, c.agentTok, "memhold20", 20)

	// R1: ลดต่ำกว่า pt ของ Member = 402306 บอกค่าต่ำสุด 50
	r := call(t, app, "POST", updatePTPath, ptBody(c.agent.ID, 45, 0, 0, 0), c.shareTok)
	expect(t, r, 200, 402306)
	expectMsgHas(t, r.Msg, "50")

	// R2: สำเร็จแล้ว remain = ค่าใหม่ − pt · pt ไม่เปลี่ยน
	for _, tc := range []struct{ give, wantA, wantB float64 }{{55, 5, 35}, {70, 20, 50}} {
		expect(t, call(t, app, "POST", updatePTPath, ptBody(c.agent.ID, tc.give, 0, 0, 0), c.shareTok), 200, 200)
		sa, sb := memberSetting(t, ma.ID), memberSetting(t, mb.ID)
		if sa.PT != 50 || sb.PT != 20 || sa.Remain != tc.wantA || sb.Remain != tc.wantB {
			t.Fatalf("give %v: member a %+v b %+v", tc.give, sa, sb)
		}
	}
	var rows []models.UserMemberGameSetting
	database.DBConn.Where("user_member_id = ?", ma.ID).Find(&rows)
	for _, s := range rows {
		if s.Remain != 20 {
			t.Fatalf("ต้อง sync ทุกเกมในกลุ่ม %s %+v", s.GameCode, s)
		}
	}
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
	var got models.UserMember
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
	expect(t, call(t, app, "POST", updateMemberPTPath, memberPTBody(m.ID, 0, 0.5), c.agentTok), 200, 200)
	var ms models.UserMemberGameSetting
	database.DBConn.Where("user_member_id = ?", m.ID).Take(&ms)
	if ms.Commission != 0.5 {
		t.Fatalf("commission %v", ms.Commission)
	}
	expect(t, call(t, app, "POST", updateMemberPTPath, memberPTBody(m.ID, 0, 1.1), c.agentTok), 200, 402309)
	b := memberPTBody(m.ID, 0, 0.5)
	b["pt"].(map[string]any)["minigame"].(map[string]any)["pt_from_parent"] = 10
	expect(t, call(t, app, "POST", updateMemberPTPath, b, c.agentTok), 200, 422)

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

func TestMemberPT(t *testing.T) { // MGMT-21 แก้ 2026-10-09 (agent ถือสู้ Member แต่ละคนแยกกัน)
	app := setup2(t)
	c := buildChain(t, app) // agent01 ได้รับ 60

	// สร้าง: pt 0 ถึงค่าที่ผู้สร้างได้รับ ทีละ 0.5 · remain = ได้รับ − pt · ส่ง remain_quota / ไม่ส่ง pt = 422
	withPT := func(username string, pt any, extra map[string]any) map[string]any {
		b := memberBody(username, 0.3)
		g := b["pt"].(map[string]any)["minigame"].(map[string]any)
		if pt == nil {
			delete(g, "pt")
		} else {
			g["pt"] = pt
		}
		for k, v := range extra {
			g[k] = v
		}
		return b
	}
	expect(t, call(t, app, "POST", createMemberPath, withPT("memover", 60.5, nil), c.agentTok), 200, 402305)
	expect(t, call(t, app, "POST", createMemberPath, withPT("memstep", 30.25, nil), c.agentTok), 200, 422)
	expect(t, call(t, app, "POST", createMemberPath, withPT("memnopt", nil, nil), c.agentTok), 200, 422)
	expect(t, call(t, app, "POST", createMemberPath, withPT("memremain", 30, map[string]any{"remain_quota": 10}), c.agentTok), 200, 422)
	expect(t, call(t, app, "POST", createMemberPath, withPT("memforce", 30, map[string]any{"force": 0}), c.agentTok), 200, 422)

	r := call(t, app, "POST", createMemberPath, withPT("mema", 50, nil), c.agentTok)
	expect(t, r, 200, 200)
	var ma created
	_ = json.Unmarshal(r.Data, &ma)
	r = call(t, app, "POST", createMemberPath, withPT("memb", 20, nil), c.agentTok)
	expect(t, r, 200, 200)
	var mb created
	_ = json.Unmarshal(r.Data, &mb)
	if s := memberSetting(t, mb.ID); s.PT != 20 || s.Remain != 40 {
		t.Fatalf("memb %+v", s)
	}
	var rows []models.UserMemberGameSetting
	database.DBConn.Where("user_member_id = ?", ma.ID).Find(&rows)
	if len(rows) != 3 {
		t.Fatalf("ต้องกระจายครบ 3 เกม ได้ %d", len(rows))
	}
	for _, s := range rows { // parent_id = agent ผู้สร้าง
		if s.ParentID != c.agent.ID {
			t.Fatalf("parent_id %s = %d, want %d", s.GameCode, s.ParentID, c.agent.ID)
		}
	}
	for _, s := range rows {
		if s.PT != 50 || s.Remain != 10 || s.Commission != 0.3 {
			t.Fatalf("mema %s %+v", s.GameCode, s)
		}
	}

	// update-pt: เฉพาะผู้สร้างโดยตรง · pt ไม่เกินที่ได้รับ (60) · remain คิดใหม่ · ต้องครบ pt + commission_percent
	expect(t, call(t, app, "POST", updateMemberPTPath, memberPTBody(ma.ID, 60, 0.2), c.agentTok), 200, 200)
	if s := memberSetting(t, ma.ID); s.PT != 60 || s.Remain != 0 || s.Commission != 0.2 || s.UpdatedBy != "agent01" {
		t.Fatalf("update-pt %+v", s)
	}
	expect(t, call(t, app, "POST", updateMemberPTPath, memberPTBody(ma.ID, 60.5, 0.2), c.agentTok), 200, 402305)
	expect(t, call(t, app, "POST", updateMemberPTPath, memberPTBody(ma.ID, 30, 0.2), c.shareTok), 200, 402304)
	b := memberPTBody(ma.ID, 30, 0.2)
	b["pt"].(map[string]any)["minigame"].(map[string]any)["remain_quota"] = 40
	expect(t, call(t, app, "POST", updateMemberPTPath, b, c.agentTok), 200, 422)
	expect(t, call(t, app, "POST", updateMemberPTPath, map[string]any{"id": ma.ID, "pt": map[string]any{"minigame": map[string]any{"pt": 30}}}, c.agentTok), 200, 422)

	// detail แสดง pt · remain_quota · commission_percent
	r = call(t, app, "POST", memberDetailPath, map[string]any{"id": ma.ID}, c.agentTok)
	expect(t, r, 200, 200)
	var d struct {
		PT map[string]map[string]any `json:"pt"`
	}
	_ = json.Unmarshal(r.Data, &d)
	if mg := d.PT["minigame"]; mg["pt"] != float64(60) || mg["remain_quota"] != float64(0) || mg["commission_percent"] != 0.2 {
		t.Fatalf("detail pt %+v", d.PT)
	}
}

func TestUpdateSubPermission(t *testing.T) { // MGMT-51
	app := setup2(t)
	c := buildChain(t, app)
	hash, _ := utils.HashPassword(mgPassword)
	sub := models.Subaccount{AgentID: c.com.ID, Username: "comp01@staff", PasswordHash: hash, Status: models.AgentStatusActive,
		Permissions: `{"member":"edit"}`}
	if err := agentAuthPostgres.CreateSubaccountRepository(database.DBConn, &sub); err != nil {
		t.Fatal(err)
	}
	tok := readyToken(t, app, models.AccountTypeSub, sub.ID, sub.Username)
	expect(t, call(t, app, "POST", agentInfoPath, map[string]any{"id": c.share.ID, "name": "share01", "phone": ""}, tok), 200, 200)
	expect(t, call(t, app, "POST", updatePTPath, ptBody(c.share.ID, 80, 0, 0, 0), tok), 200, 402303)

	setCols(t, models.AccountTypeSub, sub.ID, map[string]any{"permissions": `{"pt":"edit"}`})
	expect(t, call(t, app, "POST", updatePTPath, ptBody(c.share.ID, 80, 0, 0, 0), tok), 200, 200)
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
