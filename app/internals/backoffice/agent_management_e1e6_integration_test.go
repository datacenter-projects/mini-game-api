//go:build integration

// โครงเส้นใหม่ E1–E6 (lead review agent_management 2026-10-09) — spec หัวข้อ 5 · MGMT-02 · MGMT-29 – MGMT-34
package backoffice_test

import (
	"encoding/json"
	"testing"

	"app/app/models"
	"app/platform/database"
)

func TestDetailUpdateSections(t *testing.T) { // MGMT-32 – MGMT-34
	app := setup2(t)
	c := buildChain(t, app) // comp01 → share01 → agent01
	path := agentDetailUpdatePath

	expect(t, call(t, app, "POST", path, map[string]any{"id": c.share.ID}, c.comTok), 200, 422)                                               // ไม่มี section
	expect(t, call(t, app, "POST", path, map[string]any{"id": c.share.ID, "info": map[string]any{}}, c.comTok), 200, 422)                     // info ว่าง
	expect(t, call(t, app, "POST", path, map[string]any{"id": c.com.ID, "info": map[string]any{"name": "comp01"}}, c.comTok), 200, 402304)    // ตัวเอง
	expect(t, call(t, app, "POST", path, map[string]any{"id": c.agent.ID, "info": map[string]any{"name": "agent01"}}, c.comTok), 200, 402304) // หลาน

	logs := func() int64 {
		return countRows(t, &models.AccountChangeLog{}, "target_type = ? AND target_id = ?", "AGENT", c.share.ID)
	}
	base := logs()

	// tx เดียว: info ถูก + pt เกินที่ได้รับ → ไม่บันทึกเลย
	b := ptBody(c.share.ID, 95, 0, 0, 0)
	b["info"] = map[string]any{"name": "newname01"}
	expect(t, call(t, app, "POST", path, b, c.comTok), 200, 402305)
	var a models.UserAgent
	database.DBConn.Where("id = ?", c.share.ID).Take(&a)
	if *a.Name != "Nameshare01" || logs() != base {
		t.Fatalf("ผิดแล้วต้องไม่บันทึก name=%s logs=%d", *a.Name, logs()-base)
	}

	// ค่าเหมือนเดิมทุก section = 200 ไม่มี log
	same := ptBody(c.share.ID, 70, 0, 0, 0.5)
	same["info"] = map[string]any{"name": "Nameshare01"}
	same["status_game"] = map[string]bool{"coin_toss": true}
	expect(t, call(t, app, "POST", path, same, c.comTok), 200, 200)
	if logs() != base {
		t.Fatalf("ค่าเดิมต้องไม่เขียน log ได้ %d แถว", logs()-base)
	}

	// เปลี่ยน info + pt = log 2 แถวแยก section · status_game เดิม = ไม่มีแถว
	chg := ptBody(c.share.ID, 75, 0, 0, 0.5)
	chg["info"] = map[string]any{"name": "newname01"}
	chg["status_game"] = map[string]bool{"coin_toss": true}
	expect(t, call(t, app, "POST", path, chg, c.comTok), 200, 200)
	var rows []models.AccountChangeLog
	database.DBConn.Where("target_type = ? AND target_id = ?", "AGENT", c.share.ID).Order("id").Find(&rows)
	added := rows[len(rows)-int(logs()-base):]
	if len(added) != 2 || added[0].Action != models.ChangeUpdateInfo || added[1].Action != models.ChangeUpdatePT {
		t.Fatalf("log ต่อ section %+v", added)
	}
	if s := gameSetting(t, c.share.ID); s.PTFromParent != 75 {
		t.Fatalf("pt %+v", s)
	}
}

func TestStatusUpdatePath(t *testing.T) { // MGMT-30 · lead E5
	app := setup2(t)
	c := buildChain(t, app)
	expect(t, call(t, app, "POST", agentStatusPath, map[string]any{"id": c.com.ID, "status": "SUSPENDED"}, c.comTok), 200, 402304)   // ตัวเอง
	expect(t, call(t, app, "POST", agentStatusPath, map[string]any{"id": c.agent.ID, "status": "SUSPENDED"}, c.comTok), 200, 402304) // หลาน
	expect(t, call(t, app, "POST", agentStatusPath, map[string]any{"id": c.share.ID, "status": "LOCKED"}, c.comTok), 200, 200)
	expect(t, call(t, app, "POST", agentStatusPath, map[string]any{"id": c.share.ID, "status": "ACTIVE"}, c.comTok), 200, 200)
	logs := func() int64 {
		return countRows(t, &models.AccountChangeLog{}, "action = ? AND target_id = ?", models.ChangeUpdateStatus, c.share.ID)
	}
	if n := logs(); n != 2 {
		t.Fatalf("logs %d", n)
	}
	// ส่งสถานะเดิม = 200 ไม่เขียน log (lead MQ5 / V1)
	expect(t, call(t, app, "POST", agentStatusPath, map[string]any{"id": c.share.ID, "status": "ACTIVE"}, c.comTok), 200, 200)
	if n := logs(); n != 2 {
		t.Fatalf("สถานะเดิมต้องไม่เขียน log ได้ %d", n)
	}
}

func TestDetailStatusGameEffective(t *testing.T) { // MGMT-20 · MGMT-29 · lead E3-1
	app := setup2(t)
	c := buildChain(t, app)
	// comp01 ปิด scratch_card ของ share01 → agent01 (ลูก share01) เปิดเองอยู่ แต่เล่นไม่ได้
	expect(t, call(t, app, "POST", agentDetailUpdatePath, map[string]any{"id": c.share.ID, "status_game": map[string]bool{"scratch_card": false}}, c.comTok), 200, 200)
	r := call(t, app, "POST", agentDetailPath, map[string]any{"id": c.agent.ID}, c.comTok)
	expect(t, r, 200, 200)
	var d struct {
		StatusGame          map[string]bool `json:"status_game"`
		StatusGameEffective map[string]bool `json:"status_game_effective"`
	}
	_ = json.Unmarshal(r.Data, &d)
	if !d.StatusGame["scratch_card"] || d.StatusGameEffective["scratch_card"] || !d.StatusGameEffective["coin_toss"] || len(d.StatusGameEffective) != 3 {
		t.Fatalf("status_game %v effective %v", d.StatusGame, d.StatusGameEffective)
	}
}

func TestShareResellerMasterCreateMemberOnly(t *testing.T) { // MGMT-02 · lead H1
	app := setup2(t)
	c := buildChain(t, app)
	_, resTok := mustCreate(t, app, c.saTok, agentBody("COMPANY_SEAMLESS_RESELLER", "reseller01", nil, childPT(80, 0, 0, 0)))
	_, shareResTok := mustCreate(t, app, resTok, agentBody("SHARE_B2C", "shareres01", []string{"THB"}, childPT(70, 0, 0, 0)))
	r := call(t, app, "POST", createAgentPath, agentBody("AGENT", "agentres01", nil, childPT(60, 0, 0, 0)), shareResTok)
	expect(t, r, 200, 402301)
	expectMsgHas(t, r.Msg, "SHARE_RESELLER", "MEMBER", "members/create")
	expect(t, call(t, app, "POST", createMemberPath, memberBody("memres01", 0), shareResTok), 200, 200)

	// Share Master (ใต้ CSM) ก็สร้างได้แค่ Member เหมือนกัน (lead V5)
	_, csmTok := mustCreate(t, app, c.saTok, agentBody("COMPANY_SEAMLESS_MASTER", "master01", nil, childPT(80, 0, 0, 0)))
	_, smTok := mustCreate(t, app, csmTok, agentBody("SHARE_B2C", "sharemas01", []string{"THB"}, commissionOnly(0)))
	r = call(t, app, "POST", createAgentPath, agentBody("AGENT", "agentmas01", nil, childPT(60, 0, 0, 0)), smTok)
	expect(t, r, 200, 402301)
	expectMsgHas(t, r.Msg, "SHARE_MASTER", "MEMBER", "members/create")
	expect(t, call(t, app, "POST", createMemberPath, memberBody("memmas01", 0), smTok), 200, 200)
	if n := countRows(t, &models.UserAgent{}, "username IN ?", []string{"agentres01", "agentmas01"}); n != 0 {
		t.Fatalf("ต้องไม่สร้าง Agent %d", n)
	}
}

func TestRemovedPaths(t *testing.T) { // spec หัวข้อ 5: เส้นที่ลบได้ 404 (ไม่คง alias)
	app := setup2(t)
	c := buildChain(t, app)
	for _, p := range []string{"downlines/search", "agents/detail", "agents/update-info", "agents/update-pt", "agents/update-games",
		"agents/update-status", "agents/copy-sources", "subaccounts/detail", "subaccounts/update-info", "subaccounts/update-status"} {
		for _, m := range []string{"GET", "POST"} {
			if r := call(t, app, m, "/api/v1/bo/pr/manage/"+p, map[string]any{"id": c.share.ID}, c.comTok); r.Status != 404 {
				t.Fatalf("%s %s ต้องไม่มีแล้ว ได้ HTTP %d code %d", m, p, r.Status, r.Code)
			}
		}
	}
}
