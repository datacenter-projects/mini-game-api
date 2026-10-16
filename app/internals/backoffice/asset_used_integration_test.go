//go:build integration

package backoffice_test

import (
	"sync"
	"testing"

	"app/app/models"
	"app/pkg/testutil"
	"app/platform/database"
)

type fileItem struct {
	ConfigID  uint   `json:"config_id"`
	URL       string `json:"url"`
	Source    string `json:"source"`
	IsUsed    bool   `json:"is_used"`
	CreatedBy string `json:"created_by"`
}

type pageData struct {
	CurrentPage int        `json:"current_page"`
	TotalPage   int        `json:"total_page"`
	TotalCount  int64      `json:"total_count"`
	Limit       int        `json:"limit"`
	HasNext     bool       `json:"has_next"`
	Data        []fileItem `json:"data"`
}

func TestAssetUsedUpdate(t *testing.T) { // AST-22 · AST-23 · AST-07
	app := setup2(t)
	testutil.NewFakeS3(t)
	w := buildAssetWorld(t, app)
	sid := w.share.ID

	g1 := uploadScratch(t, app, w.saTok, formGlobal(), "icon_button")
	g2 := uploadScratch(t, app, w.saTok, formGlobal("use_now", "false"), "icon_button")
	s1 := uploadScratch(t, app, w.shareTok, formOwn("use_now", "false"), "icon_button")

	updGlobal := func(cfg uint, tok string) testutil.Response {
		return assetCall(t, app, "used/update", scopeGlobal("game_code", "scratch_card", "config_id", cfg), tok)
	}
	updAgent := func(game string, cfg uint, tok string) testutil.Response { // ของบัญชีของ token
		return assetCall(t, app, "used/update", scopeOwn("game_code", game, "config_id", cfg), tok)
	}

	// AST-22: เลือกไฟล์ใหม่ = แก้แถวเดิม
	expect(t, updGlobal(g2.ConfigID, w.saTok), 200, 200)
	expect(t, updGlobal(g1.ConfigID, w.saTok), 200, 200)
	expect(t, updGlobal(g2.ConfigID, w.saTok), 200, 200)
	if usedCount(t, nil, "scratch_card", "icon_button") != 1 {
		t.Fatal("GLOBAL ของช่องหนึ่งต้องมี used 1 แถว")
	}
	d := usedItems(t, app, w.saTok, scopeGlobal("game_code", "scratch_card"))
	if it := findItem(t, d, "icon_button"); it.ConfigID != g2.ConfigID || it.Source != "global" {
		t.Fatalf("GLOBAL used/list: %+v", it)
	}

	// AST-22: GLOBAL เลือกพร้อมกัน 2 คำขอ → ยังมี 1 แถว
	var wg sync.WaitGroup
	for _, cfg := range []uint{g1.ConfigID, g2.ConfigID, g1.ConfigID, g2.ConfigID} {
		wg.Add(1)
		go func(c uint) {
			defer wg.Done()
			if r := updGlobal(c, w.saTok); r.Code != 200 {
				t.Errorf("used/update พร้อมกัน: code %d (%s)", r.Code, r.Msg)
			}
		}(cfg)
	}
	wg.Wait()
	if usedCount(t, nil, "scratch_card", "icon_button") != 1 {
		t.Fatal("แข่งกันแล้ว GLOBAL ต้องยังมี used 1 แถว (partial unique index)")
	}

	// AST-23 ข้อ ข: config ต้องเป็นเกมเดียวกับที่ส่ง
	expect(t, updAgent("coin_toss", s1.ConfigID, w.shareTok), 200, codeConfigNotFound)
	// AST-23 ข้อ ค: GLOBAL เลือกของบัญชีอื่นไม่ได้ · บัญชีเลือกของชั้นบนไม่ได้
	expect(t, updGlobal(s1.ConfigID, w.saTok), 200, codeConfigNotFound)
	expect(t, updAgent("scratch_card", s1.ConfigID, w.agentTok), 200, codeConfigNotFound)
	// บัญชีเลือก config GLOBAL ได้ → used ของบัญชี (source = self)
	expect(t, updAgent("scratch_card", g1.ConfigID, w.shareTok), 200, 200)
	d = usedItems(t, app, w.shareTok, scopeOwn("game_code", "scratch_card"))
	if it := findItem(t, d, "icon_button"); it.ConfigID != g1.ConfigID || it.Source != "self" {
		t.Fatalf("share01 เลือก config GLOBAL: %+v", it)
	}
	// ของตัวเอง
	expect(t, updAgent("scratch_card", s1.ConfigID, w.shareTok), 200, 200)
	if usedCount(t, &sid, "scratch_card", "icon_button") != 1 {
		t.Fatal("share01 ต้องมี used 1 แถวต่อช่อง")
	}
	// ไม่พบ · 0 · ถูกลบแล้ว → 405402 / 422
	expect(t, updAgent("scratch_card", 999999, w.shareTok), 200, codeConfigNotFound)
	expect(t, updAgent("scratch_card", 0, w.shareTok), 200, 422)
	tmp := uploadScratch(t, app, w.shareTok, formOwn("use_now", "false"), "icon_button")
	expect(t, assetCall(t, app, "files/delete", scopeOwn("config_id", tmp.ConfigID), w.shareTok), 200, 200)
	expect(t, updAgent("scratch_card", tmp.ConfigID, w.shareTok), 200, codeConfigNotFound)

	// AST-07: ข้อความของ "ไม่พบ" / "ผิด scope" / "ของบัญชีอื่น" ต้องเหมือนกันทุกตัวอักษร (กันไล่เดา id)
	missing := updAgent("scratch_card", 999999, w.shareTok)
	foreign := updAgent("scratch_card", s1.ConfigID, w.agentTok)
	wrongScope := updGlobal(s1.ConfigID, w.saTok)
	if missing.Msg != foreign.Msg || missing.Msg != wrongScope.Msg || missing.Code != foreign.Code {
		t.Errorf("ข้อความต้องเหมือนกัน: %q / %q / %q", missing.Msg, foreign.Msg, wrongScope.Msg)
	}
	// ลบ config ของบัญชีอื่นด้วย scope ของตัวเอง → 405402
	expect(t, assetCall(t, app, "files/delete", scopeOwn("config_id", s1.ConfigID), w.agentTok), 200, codeConfigNotFound)
	expect(t, assetCall(t, app, "files/delete", scopeOwn("config_id", 999999), w.shareTok), 200, codeConfigNotFound)
}

func TestAssetInheritance(t *testing.T) { // AST-24 · AST-25 · AST-53 · AST-57
	app := setup2(t)
	testutil.NewFakeS3(t)
	w := buildAssetWorld(t, app)

	// GLOBAL: background · winner มีไฟล์ · bet_limit อัปแต่ไม่ใช้ (use_now=false)
	gBack := uploadScratch(t, app, w.saTok, formGlobal(), "background")
	gWin := uploadScratch(t, app, w.saTok, formGlobal(), "winner")
	uploadScratch(t, app, w.saTok, formGlobal("use_now", "false"), "bet_limit")
	// share01 ตั้ง icon_button · agent01 ตั้ง winner ของตัวเอง
	sIcon := uploadScratch(t, app, w.shareTok, formOwn(), "icon_button")
	aWin := uploadScratch(t, app, w.agentTok, formOwn(), "winner")

	d := usedItems(t, app, w.agentTok, scopeOwn("game_code", "scratch_card"))
	if len(d.Items) != 19 {
		t.Fatalf("used/list ต้องส่งทุกช่อง (19) ได้ %d", len(d.Items))
	}
	if d.Source != "mixed" {
		t.Errorf("ที่มาหลายแห่งต้องเป็น mixed: %q", d.Source)
	}
	if it := findItem(t, d, "icon_button"); it.URL != sIcon.URL || it.Source != "upline" || it.InheritedFrom != "share01" {
		t.Errorf("icon_button ต้องได้จาก share01: %+v", it)
	}
	if it := findItem(t, d, "winner"); it.URL != aWin.URL || it.Source != "self" || it.InheritedFrom != "" {
		t.Errorf("winner ของตัวเองต้องชนะ GLOBAL: %+v", it)
	}
	if it := findItem(t, d, "background"); it.URL != gBack.URL || it.Source != "global" || it.InheritedFrom != "" {
		t.Errorf("background ต้องได้จาก GLOBAL: %+v", it)
	}
	// ช่องที่ไม่มีไฟล์: ไม่ใช้ไฟล์ล่าสุดที่อัปด้วย use_now=false (AST-24) · ได้ url "" config_id 0
	for _, slot := range []string{"bet_limit", "you_lose", "main_play"} {
		if it := findItem(t, d, slot); it.URL != "" || it.ConfigID != 0 || it.Source != "" || it.InheritedFrom != "" {
			t.Errorf("%s ต้องว่าง: %+v", slot, it)
		}
	}

	// เป้าหมายจาก token (lead 2026-10-16): comp01 (ชั้นบน) เรียกแล้วได้ของ comp01 เอง ไม่ใช่ของ agent01 / share01
	// ชั้นบนไม่ได้ค่าจากสายล่าง — icon_button ของ share01 ไม่ไหลขึ้นไป · GLOBAL ไม่มี icon_button ในชุดนี้
	dv := usedItems(t, app, w.comTok, scopeOwn("game_code", "scratch_card"))
	if it := findItem(t, dv, "icon_button"); it.URL != "" {
		t.Errorf("comp01 ต้องไม่เห็นของสายล่าง: %+v", it)
	}

	// สายลึก: agent02 ใต้ agent01 ใต้ share01 — ไม่ตั้งอะไร ได้ของชั้นบนที่ใกล้ที่สุด (AST-57)
	_, a2Tok := mustCreate(t, app, w.agentTok, agentBody("AGENT", "agent02", nil, childPT(50, 0, 0, 0)))
	_, a3Tok := mustCreate(t, app, a2Tok, agentBody("AGENT", "agent03", nil, childPT(40, 0, 0, 0)))
	var a3 models.UserAgent
	if err := database.DBConn.Where("username = ?", "agent03").Take(&a3).Error; err != nil {
		t.Fatal(err)
	}
	d3 := usedItems(t, app, a3Tok, scopeOwn("game_code", "scratch_card"))
	if it := findItem(t, d3, "icon_button"); it.URL != sIcon.URL || it.Source != "upline" || it.InheritedFrom != "share01" {
		t.Errorf("agent03: icon_button ต้องได้จาก share01 (ข้ามชั้นที่ไม่ได้ตั้ง): %+v", it)
	}
	if it := findItem(t, d3, "winner"); it.URL != aWin.URL || it.Source != "upline" || it.InheritedFrom != "agent01" {
		t.Errorf("agent03: winner ต้องได้จาก agent01: %+v", it)
	}
	if it := findItem(t, d3, "background"); it.URL != gBack.URL || it.Source != "global" {
		t.Errorf("agent03: background ต้องได้ GLOBAL: %+v", it)
	}
	// agent02 ตั้งช่องเดียวกัน → agent03 ได้ของ agent02 (ใกล้กว่า)
	a2Icon := uploadScratch(t, app, a2Tok, formOwn(), "icon_button")
	d3 = usedItems(t, app, a3Tok, scopeOwn("game_code", "scratch_card"))
	if it := findItem(t, d3, "icon_button"); it.URL != a2Icon.URL || it.InheritedFrom != "agent02" {
		t.Errorf("agent03: icon_button ต้องได้จาก agent02: %+v", it)
	}

	// reset: agent01 ยกเลิก winner ของตัวเอง → กลับไปใช้ GLOBAL · config ยังอยู่ในคลัง
	expect(t, assetCall(t, app, "used/reset", scopeOwn("game_code", "scratch_card", "asset_type", "winner"), w.agentTok), 200, 200)
	d = usedItems(t, app, w.agentTok, scopeOwn("game_code", "scratch_card"))
	if it := findItem(t, d, "winner"); it.URL != gWin.URL || it.Source != "global" {
		t.Errorf("หลัง reset winner ต้องกลับไปใช้ GLOBAL: %+v", it)
	}
	if n := countRows(t, &models.AssetConfig{}, "id = ? AND is_deleted = false", aWin.ConfigID); n != 1 {
		t.Error("reset ต้องไม่ลบ config (ยังอยู่ในคลัง)")
	}

	// Share Master ใต้ CSM เดียวกันได้ค่าชุดเดียวกัน (lead AV1)
	cIcon := uploadScratch(t, app, w.csmTok, formOwn(), "icon_button")
	for _, c := range []struct {
		tok string
		id  uint
	}{{w.sm1Tok, w.sm1.ID}, {w.sm2Tok, w.sm2.ID}} {
		sd := usedItems(t, app, c.tok, scopeOwn("game_code", "scratch_card"))
		if it := findItem(t, sd, "icon_button"); it.URL != cIcon.URL || it.Source != "upline" || it.InheritedFrom != "csm01" {
			t.Errorf("Share Master ต้องได้ของ csm01: %+v", it)
		}
	}
	// share01 ไม่อยู่ใต้ csm01 — ไม่ได้ค่าของ csm01
	if it := findItem(t, usedItems(t, app, w.shareTok, scopeOwn("game_code", "scratch_card")), "icon_button"); it.URL != sIcon.URL {
		t.Errorf("share01 ต้องได้ของตัวเอง: %+v", it)
	}
}

func TestAssetFilesList(t *testing.T) { // AST-27
	app := setup2(t)
	testutil.NewFakeS3(t)
	w := buildAssetWorld(t, app)

	g := uploadScratch(t, app, w.saTok, formGlobal(), "icon_button")
	var mine []uploadData
	for i := 0; i < 3; i++ {
		mine = append(mine, uploadScratch(t, app, w.shareTok, formOwn("use_now", map[bool]string{true: "true", false: "false"}[i == 0]), "icon_button"))
	}
	other := uploadScratch(t, app, w.agentTok, formOwn(), "icon_button") // ของ agent01 ต้องไม่ปน

	list := func(tok string, body map[string]any) pageData {
		t.Helper()
		r := assetCall(t, app, "files/list", body, tok)
		expect(t, r, 200, 200)
		return decode[pageData](t, r)
	}
	body := func(kv ...any) map[string]any {
		m := scopeOwn("game_code", "scratch_card", "asset_type", "icon_button")
		for i := 0; i+1 < len(kv); i += 2 {
			m[kv[i].(string)] = kv[i+1]
		}
		return m
	}

	p := list(w.shareTok, body())
	if p.TotalCount != 4 || len(p.Data) != 4 || p.Limit != 20 || p.CurrentPage != 1 || p.HasNext {
		t.Fatalf("share01: %+v", p)
	}
	// ใหม่ → เก่า · ของตัวเอง + GLOBAL · ไม่มีของ agent01
	want := []uint{mine[2].ConfigID, mine[1].ConfigID, mine[0].ConfigID, g.ConfigID}
	for i, it := range p.Data {
		if it.ConfigID != want[i] {
			t.Errorf("ลำดับที่ %d: got %d want %d", i, it.ConfigID, want[i])
		}
		if it.ConfigID == other.ConfigID {
			t.Error("ต้องไม่มีไฟล์ของบัญชีอื่น")
		}
	}
	for _, it := range p.Data {
		switch it.ConfigID {
		case g.ConfigID:
			if it.Source != "global" || it.IsUsed || it.CreatedBy != "superadmin" {
				t.Errorf("GLOBAL: %+v", it)
			}
		case mine[0].ConfigID:
			if it.Source != "self" || !it.IsUsed {
				t.Errorf("ไฟล์ที่ใช้อยู่: %+v", it)
			}
		default:
			if it.Source != "self" || it.IsUsed {
				t.Errorf("ไฟล์ในคลัง: %+v", it)
			}
		}
	}
	// แบ่งหน้า · เพดาน limit 100
	p = list(w.shareTok, body("limit", 2))
	if p.TotalCount != 4 || len(p.Data) != 2 || !p.HasNext || p.TotalPage != 2 {
		t.Fatalf("limit 2: %+v", p)
	}
	p = list(w.shareTok, body("limit", 2, "page", 2))
	if len(p.Data) != 2 || p.HasNext || p.Data[1].ConfigID != g.ConfigID {
		t.Fatalf("หน้า 2: %+v", p)
	}
	if p = list(w.shareTok, body("limit", 500)); p.Limit != 100 {
		t.Errorf("limit ต้องถูกปรับเป็น 100: %d", p.Limit)
	}
	// เป้าหมาย GLOBAL เห็นเฉพาะของ GLOBAL
	p = list(w.saTok, scopeGlobal("game_code", "scratch_card", "asset_type", "icon_button"))
	if p.TotalCount != 1 || p.Data[0].ConfigID != g.ConfigID || !p.Data[0].IsUsed {
		t.Fatalf("GLOBAL: %+v", p)
	}
	// ไฟล์ที่ลบแล้วหายจากคลัง
	expect(t, assetCall(t, app, "files/delete", scopeOwn("config_id", mine[2].ConfigID), w.shareTok), 200, 200)
	if p = list(w.shareTok, body()); p.TotalCount != 3 {
		t.Fatalf("หลังลบ: %+v", p)
	}
	// ช่อง / เกมผิด
	expect(t, assetCall(t, app, "files/list", scopeOwn("game_code", "scratch_card", "asset_type", "nope"), w.shareTok), 200, codeSlotNotFound)
}

func TestAssetDelete(t *testing.T) { // AST-28 – AST-32
	app := setup2(t)
	fake := testutil.NewFakeS3(t)
	w := buildAssetWorld(t, app)
	del := func(scope map[string]any, cfg uint, tok string) testutil.Response {
		scope["config_id"] = cfg
		return assetCall(t, app, "files/delete", scope, tok)
	}

	// ยังถูกเลือกใช้อยู่ (used ของตัวเองชี้) = 405403 · reset ก่อนแล้วจึงลบได้
	c1 := uploadScratch(t, app, w.shareTok, formOwn(), "icon_button")
	inUse := del(scopeOwn(), c1.ConfigID, w.shareTok)
	expect(t, inUse, 200, codeInUse)
	expect(t, assetCall(t, app, "used/reset", scopeOwn("game_code", "scratch_card", "asset_type", "icon_button"), w.shareTok), 200, 200)
	expect(t, del(scopeOwn(), c1.ConfigID, w.shareTok), 200, 200)
	row := configRow(t, c1.ConfigID)
	if !row.IsDeleted || row.DeletedAt == nil || row.DeletedBy == nil || *row.DeletedBy != "share01" {
		t.Fatalf("soft delete: %+v", row)
	}
	// AST-29: ไม่แตะ storage · object ยังอยู่ใน bucket
	if fake.Deletes() != 0 {
		t.Errorf("ลบไม่ต้องเรียก storage: %d", fake.Deletes())
	}
	if _, ok := fake.Object(keyOf(c1.URL)); !ok {
		t.Error("object ต้องยังอยู่ (ให้ job กวาดภายหลัง)")
	}
	// ลบซ้ำ = ไม่พบ
	expect(t, del(scopeOwn(), c1.ConfigID, w.shareTok), 200, codeConfigNotFound)

	// AST-28: ไม่ผูกกับคนอัป — sub ลบของเจ้าของได้ · เจ้าของลบของ sub ได้
	c2 := uploadScratch(t, app, w.shareTok, formOwn("use_now", "false"), "icon_button")
	expect(t, del(scopeOwn(), c2.ConfigID, w.shareEditTok), 200, 200)
	c3 := uploadScratch(t, app, w.shareEditTok, formOwn("use_now", "false"), "icon_button")
	expect(t, del(scopeOwn(), c3.ConfigID, w.shareTok), 200, 200)

	// GLOBAL: soft delete เช่นกัน · ไม่ใช่คนอัปก็ลบได้ (ตัด 405303)
	g1 := uploadScratch(t, app, w.saTok, formGlobal("use_now", "false"), "winner")
	expect(t, del(scopeGlobal(), g1.ConfigID, w.saEditTok), 200, 200)
	if row = configRow(t, g1.ConfigID); !row.IsDeleted {
		t.Error("GLOBAL ต้อง soft delete (ไม่ลบแถวจริง)")
	}
	// GLOBAL ที่ GLOBAL ใช้อยู่ = 405403
	g2 := uploadScratch(t, app, w.saTok, formGlobal(), "winner")
	expect(t, del(scopeGlobal(), g2.ConfigID, w.saTok), 200, codeInUse)
	// GLOBAL ที่บัญชีเลือกใช้ = 405403 ข้อความเดียวกัน
	g3 := uploadScratch(t, app, w.saTok, formGlobal("use_now", "false"), "winner")
	expect(t, assetCall(t, app, "used/update", scopeOwn("game_code", "scratch_card", "config_id", g3.ConfigID), w.shareTok), 200, 200)
	byOther := del(scopeGlobal(), g3.ConfigID, w.saTok)
	expect(t, byOther, 200, codeInUse)
	if byOther.Msg != inUse.Msg {
		t.Errorf("405403 ต้องข้อความเดียว: %q / %q", byOther.Msg, inUse.Msg)
	}
	// reset แล้วลบได้
	expect(t, assetCall(t, app, "used/reset", scopeOwn("game_code", "scratch_card", "asset_type", "winner"), w.shareTok), 200, 200)
	expect(t, del(scopeGlobal(), g3.ConfigID, w.saTok), 200, 200)

	// AST-31: บัญชีลบ config GLOBAL ผ่าน scope ตัวเองไม่ได้ · ลบของบัญชีอื่นไม่ได้
	g4 := uploadScratch(t, app, w.saTok, formGlobal("use_now", "false"), "winner")
	expect(t, del(scopeOwn(), g4.ConfigID, w.shareTok), 200, codeConfigNotFound)
	s5 := uploadScratch(t, app, w.shareTok, formOwn("use_now", "false"), "winner")
	expect(t, del(scopeOwn(), s5.ConfigID, w.agentTok), 200, codeConfigNotFound)
	expect(t, del(scopeGlobal(), s5.ConfigID, w.saTok), 200, codeConfigNotFound)

	// AST-32: ลูกสืบทอดอยู่ → share01 (เจ้าของ used) ลบไม่ได้ · agent01 เห็นรูปต่อ · reset แล้ว agent01 ได้ค่าชั้นถัดไป
	s6 := uploadScratch(t, app, w.shareTok, formOwn(), "you_lose")
	expect(t, del(scopeOwn(), s6.ConfigID, w.shareTok), 200, codeInUse)
	d := usedItems(t, app, w.agentTok, scopeOwn("game_code", "scratch_card"))
	if it := findItem(t, d, "you_lose"); it.URL != s6.URL {
		t.Fatalf("agent01 ต้องยังเห็นรูปเดิม: %+v", it)
	}
	expect(t, assetCall(t, app, "used/reset", scopeOwn("game_code", "scratch_card", "asset_type", "you_lose"), w.shareTok), 200, 200)
	expect(t, del(scopeOwn(), s6.ConfigID, w.shareTok), 200, 200)
	d = usedItems(t, app, w.agentTok, scopeOwn("game_code", "scratch_card"))
	if it := findItem(t, d, "you_lose"); it.URL != "" {
		t.Errorf("หลัง reset + ลบ agent01 ต้องไม่มีรูปในช่องนี้ (GLOBAL ไม่มี): %+v", it)
	}
}

func TestAssetDeleteRacesUsedUpdate(t *testing.T) { // AST-28: ลบกับเลือกใช้พร้อมกัน ชนะได้ฝั่งเดียว ไม่มี 500
	app := setup2(t)
	testutil.NewFakeS3(t)
	w := buildAssetWorld(t, app)

	for i := 0; i < 6; i++ {
		c := uploadScratch(t, app, w.shareTok, formOwn("use_now", "false"), "icon_button")
		var del, upd testutil.Response
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			del = assetCall(t, app, "files/delete", scopeOwn("config_id", c.ConfigID), w.shareTok)
		}()
		go func() {
			defer wg.Done()
			upd = assetCall(t, app, "used/update", scopeOwn("game_code", "scratch_card", "config_id", c.ConfigID), w.shareTok)
		}()
		wg.Wait()

		for _, r := range []testutil.Response{del, upd} {
			if r.Status != 200 {
				t.Fatalf("รอบ %d: ต้องไม่มี HTTP %d (%s)", i, r.Status, r.Msg)
			}
		}
		row := configRow(t, c.ConfigID)
		used := countRows(t, &models.AssetUsed{}, "asset_config_id = ?", c.ConfigID)
		switch {
		case del.Code == 200 && upd.Code == codeConfigNotFound: // ลบชนะ
			if !row.IsDeleted || used != 0 {
				t.Fatalf("รอบ %d (ลบชนะ): deleted=%v used=%d", i, row.IsDeleted, used)
			}
		case upd.Code == 200 && del.Code == codeInUse: // เลือกใช้ชนะ
			if row.IsDeleted || used != 1 {
				t.Fatalf("รอบ %d (เลือกชนะ): deleted=%v used=%d", i, row.IsDeleted, used)
			}
		default:
			t.Fatalf("รอบ %d: ผลที่เป็นไปไม่ได้ del=%d upd=%d", i, del.Code, upd.Code)
		}
		// ล้างผลก่อนรอบถัดไป
		_ = assetCall(t, app, "used/reset", scopeOwn("game_code", "scratch_card", "asset_type", "icon_button"), w.shareTok)
	}
}

func TestAssetReset(t *testing.T) { // AST-54 · AST-55 · AST-56
	app := setup2(t)
	testutil.NewFakeS3(t)
	w := buildAssetWorld(t, app)
	g := "scratch_card"

	// ไม่มีแถวอยู่แล้ว = สำเร็จ (idempotent)
	expect(t, assetCall(t, app, "used/reset", scopeOwn("game_code", g, "asset_type", "icon_button"), w.shareTok), 200, 200)
	expect(t, assetCall(t, app, "theme/reset", scopeOwn("game_code", g), w.shareTok), 200, 200)
	expect(t, assetCall(t, app, "branding/reset", own(), w.shareTok), 200, 200)

	// scope GLOBAL = 422 (ค่ากลางห้ามหาย)
	uploadScratch(t, app, w.saTok, formGlobal(), "icon_button")
	expect(t, assetCall(t, app, "used/reset", scopeGlobal("game_code", g, "asset_type", "icon_button"), w.saTok), 200, 422)
	expect(t, assetCall(t, app, "theme/reset", scopeGlobal("game_code", g), w.saTok), 200, 422)
	if usedCount(t, nil, g, "icon_button") != 1 {
		t.Error("GLOBAL used ต้องไม่หาย")
	}

	// ต้องมีสิทธิ์เขียนของตัวเอง: Share Master = 405301 · ระบุบัญชีอื่นไม่ได้ (ส่ง agent_id = 422) · view = 402303
	expect(t, assetCall(t, app, "used/reset", scopeOwn("game_code", g, "asset_type", "icon_button"), w.sm1Tok), 200, codeNoPermission)
	expect(t, assetCall(t, app, "used/reset", scopeOwn("game_code", g, "asset_type", "icon_button", "agent_id", w.agent.ID), w.shareTok), 200, 422)
	expect(t, assetCall(t, app, "theme/reset", scopeOwn("game_code", g, "agent_id", w.csm.ID), w.shareTok), 200, 422)
	expect(t, assetCall(t, app, "branding/reset", own(), w.shareViewTok), 200, codeNoMenu)
	expect(t, assetCall(t, app, "used/reset", scopeOwn("game_code", g, "asset_type", "nope"), w.shareTok), 200, codeSlotNotFound)
	expect(t, assetCall(t, app, "theme/reset", scopeOwn("game_code", "plinko"), w.shareTok), 200, codeSlotNotFound)
}
