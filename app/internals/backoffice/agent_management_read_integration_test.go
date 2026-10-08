//go:build integration

// API test ของ agent_management phase 3 (รายชื่อดาวน์ไลน์ · รายละเอียด · copy-sources) ตาม docs/modules/agent_management.md หัวข้อ 7
package backoffice_test

import (
	"encoding/json"
	"testing"

	"app/app/models"
	agentAuthPostgres "app/app/repository/postgres/agent_auth"
	"app/pkg/utils"
	"app/platform/database"
)

const (
	downlinesPath    = "/api/v1/bo/pr/manage/downlines/list"
	agentDetailPath  = "/api/v1/bo/pr/manage/agents/detail"
	memberDetailPath = "/api/v1/bo/pr/manage/members/detail"
	copySourcesPath  = "/api/v1/bo/pr/manage/agents/copy-sources"
)

type downlineRow struct {
	ID       uint              `json:"id"`
	Role     string            `json:"role"`
	UserType string            `json:"user_type"`
	Username string            `json:"username"`
	Name     string            `json:"name"`
	Phone    string            `json:"phone"`
	Status   string            `json:"status"`
	PT       json.RawMessage   `json:"pt"`
	Balances []json.RawMessage `json:"balances"`
}

type downlinePage struct {
	Items []downlineRow `json:"data"`
	Total int64         `json:"total_count"`
}

func TestDownlines(t *testing.T) { // MGMT-26, MGMT-27, MGMT-28
	app := setup2(t)
	c := buildChain(t, app)
	// agent01 มีทั้ง Agent และ Member เป็นลูก
	mustCreate(t, app, c.agentTok, agentBody("AGENT", "zagent", nil, childPT(10, 0, 0, 0)))
	expect(t, call(t, app, "POST", createMemberPath, memberBody("bmember", 0.3), c.agentTok), 200, 200)
	expect(t, call(t, app, "POST", createMemberPath, memberBody("amember", 0.2), c.agentTok), 200, 200)
	setBalance(t, c.agent.ID, "THB", 5000)

	get := func(tok string, body map[string]any, query string) downlinePage {
		t.Helper()
		r := call(t, app, "POST", downlinesPath+query, body, tok)
		expect(t, r, 200, 200)
		var p downlinePage
		if err := json.Unmarshal(r.Data, &p); err != nil {
			t.Fatalf("%s: %v", r.Data, err)
		}
		return p
	}

	// ไม่ระบุ = ลูกตรงของตัวเอง
	p := get(c.shareTok, map[string]any{}, "")
	if p.Total != 1 || p.Items[0].Username != "agent01" || p.Items[0].UserType != "AGENT" || p.Items[0].Status != "ACTIVE" {
		t.Fatalf("share downlines %+v", p)
	}
	if string(p.Items[0].Balances[0]) != `{"currency":"THB","amount":50.00}` {
		t.Fatalf("balances %s", p.Items[0].Balances[0])
	}

	// ไล่ลงทีละชั้น: Agent + Member ปนกัน เรียง A→Z · role ถูกต้อง
	p = get(c.comTok, map[string]any{"parent_id": c.agent.ID}, "")
	got := ""
	for _, it := range p.Items {
		got += it.Username + ":" + it.Role + " "
	}
	if p.Total != 3 || got != "amember:MEMBER bmember:MEMBER zagent:AGENT " {
		t.Fatalf("agent01 downlines = %q (total %d)", got, p.Total)
	}
	var mpt map[string]map[string]any
	_ = json.Unmarshal(p.Items[0].PT, &mpt)
	if len(mpt["minigame"]) != 5 || mpt["minigame"]["commission_percent"] != 0.2 || mpt["minigame"]["created_by"] != "agent01" {
		t.Fatalf("member pt %s", p.Items[0].PT)
	}
	var apt map[string]map[string]any
	_ = json.Unmarshal(p.Items[2].PT, &apt)
	if apt["minigame"]["pt_from_parent"] != float64(10) {
		t.Fatalf("agent pt %s", p.Items[2].PT)
	}

	// q บางส่วน ไม่สนตัวพิมพ์ · limit / page
	p = get(c.comTok, map[string]any{"parent_id": c.agent.ID, "q": "MEM"}, "")
	if p.Total != 2 {
		t.Fatalf("q=MEM total %d", p.Total)
	}
	p = get(c.comTok, map[string]any{"parent_id": c.agent.ID, "limit": 1, "page": 2}, "")
	if p.Total != 3 || len(p.Items) != 1 || p.Items[0].Username != "bmember" {
		t.Fatalf("page 2 %+v", p)
	}
	p = get(c.comTok, map[string]any{"parent_id": c.agent.ID, "q": "%"}, "") // % ไม่ใช่ wildcard
	if p.Total != 0 {
		t.Fatalf("q=%% total %d", p.Total)
	}

	// บัญชีสายอื่น / สายบน / ไม่มีอยู่ = 402402
	_, otherTok := mustCreate(t, app, c.saTok, agentBody("COMPANY_TRANSFER", "othercom", nil, childPT(50, 0, 0, 0)))
	expect(t, call(t, app, "POST", downlinesPath, map[string]any{"parent_id": c.agent.ID}, otherTok), 200, 402402)
	expect(t, call(t, app, "POST", downlinesPath, map[string]any{"parent_id": c.com.ID}, c.shareTok), 200, 402402)
	expect(t, call(t, app, "POST", downlinesPath, map[string]any{"parent_id": 99999}, c.comTok), 200, 402402)

	// ADMIN ไม่อยู่ในรายการ (AUTH-43): DB กันไว้แล้ว — ADMIN มี parent ไม่ได้ (ck_user_agents_admin_no_parent) · query ก็กรองซ้ำ

	// status ที่ใช้งานจริง: หัวสายถูกระงับ → ลูกแสดง SUSPENDED (ACC-30)
	setStatus(t, c.share.ID, models.AgentStatusSuspended)
	p = get(c.comTok, map[string]any{"parent_id": c.share.ID}, "")
	if p.Items[0].Status != "SUSPENDED" {
		t.Fatalf("status %s, want SUSPENDED", p.Items[0].Status)
	}
}

func TestDownlinesPTPermission(t *testing.T) { // MGMT-51
	app := setup2(t)
	c := buildChain(t, app)
	hash, _ := utils.HashPassword(mgPassword)
	sub := models.Subaccount{AgentID: c.com.ID, Username: "comp01@staff", PasswordHash: hash, Status: models.AgentStatusActive,
		Permissions: `{"member":"view"}`}
	if err := agentAuthPostgres.CreateSubaccountRepository(database.DBConn, &sub); err != nil {
		t.Fatal(err)
	}
	tok := readyToken(t, app, models.AccountTypeSub, sub.ID, sub.Username)

	r := call(t, app, "POST", downlinesPath, map[string]any{}, tok)
	expect(t, r, 200, 200)
	var raw struct {
		Items []map[string]json.RawMessage `json:"data"`
	}
	_ = json.Unmarshal(r.Data, &raw)
	if len(raw.Items) != 1 {
		t.Fatalf("items %s", r.Data)
	}
	if _, ok := raw.Items[0]["pt"]; ok {
		t.Fatal("ไม่มีสิทธิ์ pt ต้องไม่มี field pt")
	}
	r = call(t, app, "POST", agentDetailPath, map[string]any{"id": c.share.ID}, tok)
	expect(t, r, 200, 200)
	var d map[string]json.RawMessage
	_ = json.Unmarshal(r.Data, &d)
	if _, ok := d["pt"]; ok {
		t.Fatal("รายละเอียด: ไม่มีสิทธิ์ pt ต้องไม่มี field pt")
	}
	expect(t, call(t, app, "GET", copySourcesPath, nil, tok), 200, 402303)

	setCols(t, models.AccountTypeSub, sub.ID, map[string]any{"permissions": `{"pt":"view"}`})
	expect(t, call(t, app, "POST", downlinesPath, map[string]any{}, tok), 200, 402303) // member off
	expect(t, call(t, app, "GET", copySourcesPath, nil, tok), 200, 200)
}

func TestAccountDetail(t *testing.T) { // MGMT-29
	app := setup2(t)
	c := buildChain(t, app)
	setPhone := "0812345678"
	setCols(t, models.AccountTypeAgent, c.agent.ID, map[string]any{"phone": setPhone})
	r := call(t, app, "POST", agentDetailPath, map[string]any{"id": c.agent.ID}, c.comTok)
	expect(t, r, 200, 200)
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(r.Data, &raw)
	for k, v := range raw {
		if string(v) == "null" {
			t.Fatalf("%s เป็น null", k)
		}
	}
	for _, banned := range []string{"password", "password_hash", "passcode", "passcode_hash", "token"} {
		if _, ok := raw[banned]; ok {
			t.Fatalf("ห้ามมี field %s", banned)
		}
	}
	var d struct {
		UserType       string                    `json:"user_type"`
		Phone          string                    `json:"phone"`
		ParentUsername string                    `json:"parent_username"`
		Currencies     []string                  `json:"currencies"`
		PT             map[string]map[string]any `json:"pt"`
		StatusGame     map[string]bool           `json:"status_game"`
		PasscodeSet    bool                      `json:"passcode_set"`
		LastLoginAt    string                    `json:"last_login_at"`
	}
	_ = json.Unmarshal(r.Data, &d)
	if d.UserType != "AGENT" || d.Phone != setPhone || d.ParentUsername != "share01" || len(d.Currencies) != 1 ||
		d.PT["minigame"]["pt_from_parent"] != float64(60) || len(d.StatusGame) != 3 || !d.PasscodeSet || d.LastLoginAt == "" {
		t.Fatalf("detail %s", r.Data)
	}

	// ตัวเอง / สายบน / สายอื่น = 402402 · id ผิดรูปแบบ = 422
	expect(t, call(t, app, "POST", agentDetailPath, map[string]any{"id": c.agent.ID}, c.agentTok), 200, 402402)
	expect(t, call(t, app, "POST", agentDetailPath, map[string]any{"id": c.com.ID}, c.agentTok), 200, 402402)
	expect(t, call(t, app, "POST", agentDetailPath, map[string]any{"id": 0}, c.comTok), 200, 422)

	// Member: ผู้สร้างเอง และชั้นบนดูได้ · ไม่มี status_game / passcode_set
	r = call(t, app, "POST", createMemberPath, memberBody("mem01", 0.3), c.agentTok)
	expect(t, r, 200, 200)
	var m created
	_ = json.Unmarshal(r.Data, &m)
	for _, tok := range []string{c.agentTok, c.comTok, c.saTok} {
		r = call(t, app, "POST", memberDetailPath, map[string]any{"id": m.ID}, tok)
		expect(t, r, 200, 200)
	}
	raw = map[string]json.RawMessage{}
	_ = json.Unmarshal(r.Data, &raw)
	if _, ok := raw["status_game"]; ok {
		t.Fatal("Member ต้องไม่มี status_game")
	}
	var memPT map[string]map[string]any
	_ = json.Unmarshal(raw["pt"], &memPT)
	mg := memPT["minigame"]
	if string(raw["parent_username"]) != `"agent01"` || string(raw["role"]) != `"MEMBER"` || len(memPT) != 1 || len(mg) != 5 ||
		mg["commission_percent"] != 0.3 || mg["created_by"] != "agent01" || mg["updated_by"] != "agent01" ||
		mg["created_at"] == "" || mg["created_at"] != mg["updated_at"] {
		t.Fatalf("member detail %s", r.Data)
	}
	_, otherTok := mustCreate(t, app, c.saTok, agentBody("COMPANY_TRANSFER", "othercom", nil, childPT(50, 0, 0, 0)))
	expect(t, call(t, app, "POST", memberDetailPath, map[string]any{"id": m.ID}, otherTok), 200, 402402)
	expect(t, call(t, app, "POST", memberDetailPath, map[string]any{"id": 99999}, c.comTok), 200, 402402)
}

func TestCopySources(t *testing.T) { // MGMT-35
	app := setup2(t)
	c := buildChain(t, app)
	mustCreate(t, app, c.comTok, agentBody("SHARE_B2B", "ashare", []string{"THB"}, childPT(40, 5, 5, 0.1)))
	expect(t, call(t, app, "POST", createMemberPath, memberBody("nomem", 0), c.agentTok), 200, 200)

	r := call(t, app, "GET", copySourcesPath, nil, c.comTok)
	expect(t, r, 200, 200)
	var list []struct {
		Username   string                    `json:"username"`
		UserType   string                    `json:"user_type"`
		PT         map[string]map[string]any `json:"pt"`
		StatusGame map[string]bool           `json:"status_game"`
	}
	if err := json.Unmarshal(r.Data, &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Username != "ashare" || list[1].Username != "share01" || list[0].UserType != "SHARE_B2B" ||
		list[0].PT["minigame"]["force"] != float64(5) || len(list[0].StatusGame) != 3 {
		t.Fatalf("copy-sources %s", r.Data)
	}
	// ไม่มีลูก = []
	r = call(t, app, "GET", copySourcesPath, nil, c.agentTok) // ลูกของ agent01 เป็น Member อย่างเดียว
	expect(t, r, 200, 200)
	if string(r.Data) != "[]" {
		t.Fatalf("ไม่มีลูกฝั่ง agent ต้องได้ [] ได้ %s", r.Data)
	}
}

func TestDownlinesSearch(t *testing.T) { // MGMT-27A
	app := setup2(t)
	c := buildChain(t, app) // comp01 → share01 → agent01
	const searchPath = "/api/v1/bo/pr/manage/downlines/search"
	mustCreate(t, app, c.comTok, agentBody("SHARE_B2B", "share02", []string{"THB"}, childPT(40, 0, 0, 0)))
	mustCreate(t, app, c.agentTok, agentBody("AGENT", "sharedeep", nil, childPT(10, 0, 0, 0))) // ชื่อมี "share" อยู่ลึก 3 ชั้น
	expect(t, call(t, app, "POST", createMemberPath, memberBody("memshare", 0.2), c.agentTok), 200, 200)
	_, otherTok := mustCreate(t, app, c.saTok, agentBody("COMPANY_TRANSFER", "othercom", nil, childPT(50, 0, 0, 0)))
	mustCreate(t, app, otherTok, agentBody("SHARE_B2C", "shareother", []string{"THB"}, childPT(40, 0, 0, 0)))

	search := func(tok, q string) ([]map[string]any, int64) {
		t.Helper()
		r := call(t, app, "POST", searchPath, map[string]any{"q": q}, tok)
		expect(t, r, 200, 200)
		var p struct {
			Data  []map[string]any `json:"data"`
			Total int64            `json:"total_count"`
		}
		_ = json.Unmarshal(r.Data, &p)
		return p.Data, p.Total
	}
	names := func(rows []map[string]any) string {
		s := ""
		for _, r := range rows {
			s += r["username"].(string) + "<" + r["parent_username"].(string) + " "
		}
		return s
	}

	// comp01 ค้น "SHARE" → ทุกชั้นใต้ตัวเอง ไม่เห็นสายอื่น
	rows, total := search(c.comTok, "SHARE")
	if total != 4 || names(rows) != "memshare<agent01 share01<comp01 share02<comp01 sharedeep<agent01 " {
		t.Fatalf("comp01 search = %q (%d)", names(rows), total)
	}
	if rows[0]["role"] != "MEMBER" || rows[1]["pt"] == nil {
		t.Fatalf("row shape %v", rows[0])
	}
	// share01 ค้น → เห็นแค่ใต้ตัวเอง ไม่เห็นตัวเอง / share02 สายข้าง / หัวสาย
	rows, _ = search(c.shareTok, "share")
	if names(rows) != "memshare<agent01 sharedeep<agent01 " {
		t.Fatalf("share01 search = %q", names(rows))
	}
	rows, total = search(c.shareTok, "comp")
	if total != 0 || len(rows) != 0 {
		t.Fatalf("ต้องไม่เห็นหัวสาย %v", rows)
	}
	// status ที่ใช้งานจริง: share01 ถูกระงับ → แถวใต้ share01 เป็น SUSPENDED
	setStatus(t, c.share.ID, models.AgentStatusSuspended)
	rows, _ = search(c.comTok, "deep")
	if len(rows) != 1 || rows[0]["status"] != "SUSPENDED" {
		t.Fatalf("status %v", rows)
	}
	setStatus(t, c.share.ID, models.AgentStatusActive)
	// page / limit
	r := call(t, app, "POST", searchPath, map[string]any{"q": "share", "limit": 1, "page": 2}, c.comTok)
	var p struct {
		Data  []map[string]any `json:"data"`
		Total int64            `json:"total_count"`
	}
	_ = json.Unmarshal(r.Data, &p)
	if p.Total != 4 || len(p.Data) != 1 || p.Data[0]["username"] != "share01" {
		t.Fatalf("page 2 %v", p)
	}
	expect(t, call(t, app, "POST", searchPath, map[string]any{"q": "s"}, c.comTok), 200, 422)
	expect(t, call(t, app, "POST", searchPath, map[string]any{}, c.comTok), 200, 422)
}
