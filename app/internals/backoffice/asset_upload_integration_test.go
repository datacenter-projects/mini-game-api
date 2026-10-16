//go:build integration

package backoffice_test

import (
	"strings"
	"testing"

	"app/app/models"
	"app/pkg/testutil"
	"app/platform/database"
)

func TestAssetTargetValidation(t *testing.T) { // AST-08 · AST-09
	app := setup2(t)
	w := buildAssetWorld(t, app)
	tok := w.saTok

	bad := func(name, op string, body any, tk string) {
		t.Helper()
		r := assetCall(t, app, op, body, tk)
		if r.Code != 422 {
			t.Errorf("%s: want 422 got code %d (%s)", name, r.Code, r.Msg)
		}
	}
	game := map[string]any{"game_code": "scratch_card"}
	with := func(kv ...any) map[string]any {
		m := map[string]any{"game_code": "scratch_card"}
		for i := 0; i+1 < len(kv); i += 2 {
			m[kv[i].(string)] = kv[i+1]
		}
		return m
	}
	_ = game

	bad("ไม่ส่ง scope", "used/list", with(), w.shareTok)
	bad("scope ตัวพิมพ์เล็ก", "used/list", with("scope", "global"), tok)
	bad("scope เป็น null", "used/list", with("scope", nil), tok)
	bad("scope เป็นตัวเลข", "used/list", with("scope", 1), tok)
	// ไม่รับ agent_id ทุกเส้น (lead 2026-10-16 — AST-08): ส่งมาค่าใดก็ตาม = 422 (ไม่เมิน)
	bad("AGENT ส่ง agent_id ของตัวเอง", "used/list", with("scope", "AGENT", "agent_id", w.share.ID), w.shareTok)
	bad("AGENT ส่ง agent_id ของลูก", "used/list", with("scope", "AGENT", "agent_id", w.agent.ID), w.shareTok)
	bad("AGENT agent_id = 0", "used/list", with("scope", "AGENT", "agent_id", 0), w.shareTok)
	bad("AGENT agent_id เป็นข้อความ", "used/list", with("scope", "AGENT", "agent_id", "abc"), w.shareTok)
	bad("GLOBAL แต่ส่ง agent_id", "used/list", with("scope", "GLOBAL", "agent_id", w.share.ID), tok)
	bad("GLOBAL agent_id = 0", "used/list", with("scope", "GLOBAL", "agent_id", 0), tok)
	bad("ไม่มี body", "used/list", nil, tok)
	bad("branding ส่ง scope GLOBAL", "branding/detail/get", map[string]any{"scope": "GLOBAL"}, w.shareTok)
	bad("branding ส่ง agent_id", "branding/detail/get", map[string]any{"agent_id": w.share.ID}, w.shareTok)
	bad("bundle ส่ง agent_id", "bundle/detail/get", with("agent_id", w.agent.ID), w.shareTok)
	// เส้น branding / bundle ไม่ต้องส่งอะไรนอกจาก field ของเส้น (ของตัวเองเสมอ)
	expect(t, assetCall(t, app, "branding/detail/get", map[string]any{}, w.shareTok), 200, 200)
	expect(t, assetCall(t, app, "bundle/detail/get", game, w.shareTok), 200, 200)

	// scope = GLOBAL ไม่ถูกตีความเงียบๆ จากค่าว่าง: ไม่ส่ง scope ห้ามกลายเป็นการเขียนค่ากลาง
	r := postForm(t, app, "files/upload", map[string]string{"game_code": "scratch_card", "asset_type": "icon_button"}, pngOf(t, 32, 32), w.saTok)
	if r.Code != 422 {
		t.Errorf("upload ไม่ส่ง scope: want 422 got %d", r.Code)
	}
	// multipart ส่ง scope ทาง query แทน form field = ไม่ส่ง field (lead GQ4)
	r = postForm(t, app, "files/upload?scope=AGENT", map[string]string{"game_code": "scratch_card", "asset_type": "icon_button"}, pngOf(t, 32, 32), w.shareTok)
	if r.Code != 422 {
		t.Errorf("upload scope ใน query: want 422 got %d", r.Code)
	}
	// multipart ส่ง agent_id เป็น form field = 422
	r = postForm(t, app, "files/upload", formOwn("agent_id", itoa(w.agent.ID), "game_code", "scratch_card", "asset_type", "icon_button"), pngOf(t, 32, 32), w.shareTok)
	if r.Code != 422 {
		t.Errorf("upload ส่ง agent_id: want 422 got %d", r.Code)
	}
	if n := countRows(t, &models.AssetConfig{}, "1 = 1"); n != 0 {
		t.Errorf("คำขอที่ 422 ต้องไม่สร้าง config: %d", n)
	}
}

func TestAssetCatalogEndpoints(t *testing.T) { // AST-10 · AST-12 · AST-13 · AST-51 · lead GE1 · GE2-1
	app := setup2(t)
	testutil.NewFakeS3(t)
	w := buildAssetWorld(t, app)

	// slots/list: ขนาด · ไม่มี description · เทียบ catalog
	r := assetCall(t, app, "slots/list", map[string]any{"game_code": "scratch_card"}, w.shareTok)
	expect(t, r, 200, 200)
	if strings.Contains(string(r.Data), "description") {
		t.Errorf("slots/list ต้องไม่มี description: %s", r.Data)
	}
	slots := decode[[]struct {
		AssetType    string `json:"asset_type"`
		AllowedSizes []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"allowed_sizes"`
	}](t, r)
	if len(slots) != 19 {
		t.Fatalf("scratch_card ต้องมี 19 ช่อง ได้ %d", len(slots))
	}
	for _, s := range slots {
		if s.AssetType == "icon_button" && len(s.AllowedSizes) != 2 {
			t.Errorf("icon_button ต้องรับ 2 ขนาด: %+v", s)
		}
		if s.AssetType == "win_highlight" && (len(s.AllowedSizes) != 1 || s.AllowedSizes[0].Width != 425) {
			t.Errorf("win_highlight: %+v", s)
		}
	}
	r = assetCall(t, app, "slots/list", map[string]any{"game_code": "coin_toss"}, w.shareTok)
	expect(t, r, 200, 200)
	for _, s := range decode[[]struct {
		AllowedSizes []any `json:"allowed_sizes"`
	}](t, r) {
		if s.AllowedSizes == nil || len(s.AllowedSizes) != 0 {
			t.Fatalf("coin_toss ไม่มีสเปกขนาด ต้องได้ [] (ไม่ใช่ null): %s", r.Data)
		}
	}
	// AST-10: รหัสเกมที่ไม่มี · ตัวพิมพ์ต่าง
	expect(t, assetCall(t, app, "slots/list", map[string]any{"game_code": "plinko"}, w.shareTok), 200, codeSlotNotFound)
	expect(t, assetCall(t, app, "slots/list", map[string]any{"game_code": "Coin_Toss"}, w.shareTok), 200, codeSlotNotFound)
	expect(t, assetCall(t, app, "slots/list", map[string]any{}, w.shareTok), 200, 422)

	// AST-13: ช่องที่เลิกใช้ไม่อยู่ใน slots/list
	retireSlot(t, "scratch_card", "you_lose")
	r = assetCall(t, app, "slots/list", map[string]any{"game_code": "scratch_card"}, w.shareTok)
	if strings.Contains(string(r.Data), "you_lose") {
		t.Error("ช่องที่เลิกใช้ต้องไม่อยู่ใน slots/list")
	}

	// AST-51 · GE1: games/list ครบ 3 เกมเสมอ ปกที่ไม่มี = ""
	r = assetCall(t, app, "games/list", nil, w.shareTok)
	expect(t, r, 200, 200)
	games := decode[[]struct {
		GameCode  string `json:"game_code"`
		GameCover string `json:"game_cover"`
	}](t, r)
	if len(games) != 3 || games[0].GameCode != "scratch_card" || games[1].GameCode != "coin_toss" || games[2].GameCode != "rock_paper_scissors" {
		t.Fatalf("games/list: %+v", games)
	}
	for _, g := range games {
		if g.GameCover != "" {
			t.Errorf("ยังไม่มีปก ต้องได้ \"\": %+v", g)
		}
	}

	// ปก GLOBAL ของ scratch_card
	gcover := uploadScratch(t, app, w.saTok, formGlobal(), "game_cover")
	r = assetCall(t, app, "games/list", nil, w.agentTok)
	if coverOf(t, r, "scratch_card") != gcover.URL || coverOf(t, r, "coin_toss") != "" {
		t.Fatalf("agent01 ต้องเห็นปก GLOBAL: %s", r.Data)
	}
	// share01 ตั้งปกของตัวเอง → agent01 (ลูก) ได้ปกของ share01 (ไล่สายจากผู้เรียก — GE1-1) · เกมอื่นได้ GLOBAL
	own := uploadScratch(t, app, w.shareTok, formOwn(), "game_cover")
	r = assetCall(t, app, "games/list", nil, w.agentTok)
	if coverOf(t, r, "scratch_card") != own.URL {
		t.Fatalf("agent01 ต้องได้ปกของ share01: %s", r.Data)
	}
	// comp01 (ชั้นบนของ share01) ยังเห็นปก GLOBAL · sa เห็น GLOBAL
	r = assetCall(t, app, "games/list", nil, w.comTok)
	if coverOf(t, r, "scratch_card") != gcover.URL {
		t.Fatalf("comp01: %s", r.Data)
	}
	// ปกที่เลือกใช้ชนะไฟล์ที่อัปล่าสุด (ใช้ used เท่านั้น — AST-24)
	uploadScratch(t, app, w.saTok, formGlobal("use_now", "false"), "game_cover")
	r = assetCall(t, app, "games/list", nil, w.comTok)
	if coverOf(t, r, "scratch_card") != gcover.URL {
		t.Fatalf("ไฟล์ที่อัปด้วย use_now=false ต้องไม่ถูกใช้: %s", r.Data)
	}
	// ปิดเกมของบัญชี (status_game) ไม่ทำให้เกมหายจากรายการ (GE1-2)
	if err := database.DBConn.Model(&models.AgentGameSetting{}).Where("agent_id = ? AND game_code = ?", w.agent.ID, "coin_toss").
		Update("status_game", false).Error; err != nil {
		t.Fatal(err)
	}
	r = assetCall(t, app, "games/list", nil, w.agentTok)
	if g := decode[[]struct{}](t, r); len(g) != 3 {
		t.Fatalf("เกมที่ status_game ปิดต้องยังอยู่ในรายการ: %s", r.Data)
	}
}

// coverOf — game_cover ของเกมจาก response ของ games/list (หาด้วยชื่อเกม ไม่พึ่งลำดับ)
func coverOf(t *testing.T, r testutil.Response, game string) string {
	t.Helper()
	for _, g := range decode[[]struct {
		GameCode  string `json:"game_code"`
		GameCover string `json:"game_cover"`
	}](t, r) {
		if g.GameCode == game {
			return g.GameCover
		}
	}
	t.Fatalf("games/list ไม่มีเกม %s: %s", game, r.Data)
	return ""
}

func TestAssetUploadChecks(t *testing.T) { // AST-11 · AST-14 – AST-17 · AST-19 · AST-20
	app := setup2(t)
	fake := testutil.NewFakeS3(t)
	w := buildAssetWorld(t, app)
	up := func(fields map[string]string, file []byte) testutil.Response {
		f := formOwn("game_code", "scratch_card", "asset_type", "icon_button")
		for k, v := range fields {
			f[k] = v
		}
		return postForm(t, app, "files/upload", f, file, w.shareTok)
	}
	noObjects := func(what string) {
		t.Helper()
		if fake.Count() != 0 || fake.Puts() != 0 {
			t.Errorf("%s: ต้องไม่มี object ขึ้น bucket (มี %d)", what, fake.Count())
		}
		if n := countRows(t, &models.AssetConfig{}, "1 = 1"); n != 0 {
			t.Errorf("%s: ต้องไม่มี config (มี %d)", what, n)
		}
	}
	typesBefore := countRows(t, &models.AssetType{}, "1 = 1")

	// AST-11 · AST-14: ต้องตรง catalog ทุกตัวอักษร · ไม่สร้างช่องใหม่
	expect(t, up(map[string]string{"asset_type": "win_highlite"}, pngOf(t, 32, 32)), 200, codeSlotNotFound)
	expect(t, up(map[string]string{"asset_type": "Icon_Button"}, pngOf(t, 32, 32)), 200, codeSlotNotFound)
	expect(t, up(map[string]string{"asset_type": "icon button"}, pngOf(t, 32, 32)), 200, codeSlotNotFound)
	expect(t, up(map[string]string{"game_code": "plinko"}, pngOf(t, 32, 32)), 200, codeSlotNotFound)
	if got := countRows(t, &models.AssetType{}, "1 = 1"); got != typesBefore {
		t.Errorf("asset_types ต้องไม่เพิ่ม: %d → %d", typesBefore, got)
	}
	noObjects("slot ผิด")

	// ไม่แนบไฟล์ · ไม่ส่ง field
	expect(t, up(nil, nil), 200, 422)
	expect(t, up(map[string]string{"asset_type": ""}, pngOf(t, 32, 32)), 200, 422)

	// AST-16: ชนิดไฟล์ตรวจจากเนื้อไฟล์ ไม่เชื่อ Content-Type / นามสกุล
	expect(t, up(nil, []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32"></svg>`)), 200, codeFileType)
	expect(t, up(nil, []byte("%PDF-1.4\n%fake")), 200, codeFileType)
	expect(t, up(nil, []byte{}), 200, codeFileType)
	noObjects("ชนิดไฟล์ผิด")

	// AST-16: ≤ 5 MB (5 MB พอดีผ่าน · เกิน 1 ไบต์ = 405103 ไม่ใช่ 413 ของ BodyLimit)
	big := padded(pngOf(t, 32, 32), 5*1024*1024+1)
	expect(t, up(nil, big), 200, codeFileTooLarge)
	noObjects("ไฟล์ใหญ่เกิน")
	// ลำดับ AST-15: catalog ก่อนขนาดไฟล์
	expect(t, up(map[string]string{"asset_type": "nope"}, big), 200, codeSlotNotFound)
	exact := padded(pngOf(t, 32, 32), 5*1024*1024)
	r := up(nil, exact)
	expect(t, r, 200, 200)
	first := decode[uploadData](t, r)

	// AST-16: object มี Content-Type ตามชนิดจริง และ Cache-Control ยาว
	obj, ok := fake.Object(keyOf(first.URL))
	if !ok {
		t.Fatalf("ไม่พบ object %s", keyOf(first.URL))
	}
	if obj.ContentType != "image/png" || obj.CacheControl != "public, max-age=31536000, immutable" {
		t.Errorf("headers ของ object: %+v", obj)
	}
	if !strings.HasPrefix(first.URL, cdnBase+"games/scratch_card/agents/") || !strings.HasSuffix(first.URL, ".png") {
		t.Errorf("รูปแบบ URL/key: %s", first.URL)
	}

	// AST-17: ขนาดภาพ
	before := fake.Puts()
	r = up(map[string]string{"asset_type": "win_highlight"}, pngOf(t, 400, 400))
	expect(t, r, 200, codeImageSize)
	if !strings.Contains(r.Msg, "425×425") {
		t.Errorf("msg ต้องบอกขนาดที่รับ: %s", r.Msg)
	}
	if fake.Puts() != before {
		t.Error("ภาพผิดขนาดต้องไม่ถูกอัปขึ้น bucket (ตรวจก่อนอัป — AST-15)")
	}
	expect(t, up(nil, pngOf(t, 40, 40)), 200, 200)                                                                      // ขนาดที่ 2 ของ icon_button
	expect(t, up(nil, pngOf(t, 36, 36)), 200, codeImageSize)                                                            // ไม่ตรงทั้งสองขนาด
	expect(t, up(map[string]string{"game_code": "coin_toss", "asset_type": "background"}, pngOf(t, 123, 45)), 200, 200) // ช่องไม่มีสเปก
	expect(t, up(nil, webpUnreadable()), 200, codeFileType)                                                             // อ่านขนาดไม่ได้ = ปฏิเสธ (fail-closed — lead A2)

	// AST-20: อัปช่องเดิมกี่ครั้งก็เก็บทุกเวอร์ชัน
	for i := 0; i < 3; i++ {
		expect(t, up(nil, pngOf(t, 32, 32)), 200, 200)
	}
	if n := countRows(t, &models.AssetConfig{}, "asset_type_id = ? AND agent_id = ?", slotID(t, "scratch_card", "icon_button"), w.share.ID); n < 5 { // webp ที่อ่านขนาดไม่ได้ไม่ถูกนับแล้ว (lead A2)
		t.Errorf("config ของช่องเดิมต้องเก็บทุกเวอร์ชัน (มี %d)", n)
	}
	if n := countRows(t, &models.AssetConfig{}, "is_deleted = true"); n != 0 {
		t.Errorf("ไม่มีการลบ/ตัดเวอร์ชันอัตโนมัติ: %d", n)
	}
}

func TestAssetRetiredSlot(t *testing.T) { // AST-13 · AST-15 · AST-23
	app := setup2(t)
	fake := testutil.NewFakeS3(t)
	w := buildAssetWorld(t, app)
	f := formOwn("game_code", "scratch_card", "asset_type", "winner")

	ok := uploadScratch(t, app, w.shareTok, formOwn(), "winner")
	retireSlot(t, "scratch_card", "winner")

	// ช่องที่เลิกใช้: อัปไม่ได้ (ตรวจก่อนขนาดภาพ — AST-15) · เลือกใช้ไม่ได้ · ไม่มีไฟล์ขึ้น bucket
	puts := fake.Puts()
	expect(t, postForm(t, app, "files/upload", f, pngOf(t, 10, 10), w.shareTok), 200, codeSlotRetired)
	expect(t, postForm(t, app, "files/upload", f, pngOf(t, 502, 514), w.shareTok), 200, codeSlotRetired)
	if fake.Puts() != puts {
		t.Error("ช่องที่เลิกใช้ต้องไม่มีไฟล์ขึ้น bucket")
	}
	expect(t, assetCall(t, app, "used/update", scopeOwn("game_code", "scratch_card", "config_id", ok.ConfigID), w.shareTok), 200, codeSlotRetired)

	// ไฟล์ที่ใช้งานอยู่แล้วยังอยู่ (เกมที่ยังอ้างช่องนี้ต้องไม่พัง)
	d := usedItems(t, app, w.shareTok, scopeOwn("game_code", "scratch_card"))
	if it := findItem(t, d, "winner"); it.URL != ok.URL || !it.IsRetired {
		t.Errorf("ช่องที่เลิกใช้แต่ยังมีไฟล์ ต้องยังแสดง: %+v", it)
	}
	r := assetCall(t, app, "bundle/detail/get", own("game_code", "scratch_card"), w.shareTok)
	expect(t, r, 200, 200)
	if b := decode[bundleData](t, r); b.Assets["winner"] != ok.URL {
		t.Errorf("bundle ต้องยังมีช่องที่เลิกใช้: %+v", b.Assets)
	}

	// ช่องที่เลิกใช้และไม่มีไฟล์: ไม่แสดงใน used/list
	retireSlot(t, "scratch_card", "you_lose")
	d = usedItems(t, app, w.agentTok, scopeOwn("game_code", "scratch_card"))
	for _, it := range d.Items {
		if it.AssetType == "you_lose" {
			t.Error("ช่องที่เลิกใช้และไม่มีไฟล์ต้องไม่อยู่ใน used/list")
		}
	}
}

func TestAssetStorageFailures(t *testing.T) { // AST-19 · AST-30
	app := setup2(t)
	w := buildAssetWorld(t, app)
	up := func(tok string) testutil.Response {
		return postForm(t, app, "files/upload", formOwn("game_code", "scratch_card", "asset_type", "icon_button"), pngOf(t, 32, 32), tok)
	}

	// ไม่ได้ตั้ง storage (S3_BUCKET ว่าง) = ระบบยังทำงาน · อัปได้ 505801 · ไม่มี config
	expect(t, up(w.shareTok), 503, codeStorageDown)
	if n := countRows(t, &models.AssetConfig{}, "1 = 1"); n != 0 {
		t.Errorf("storage ไม่พร้อม ต้องไม่มี config: %d", n)
	}
	expect(t, postForm(t, app, "branding/logo/upload", map[string]string{}, pngOf(t, 10, 10), w.shareTok), 503, codeStorageDown)
	// เส้นอ่าน / เส้นที่ไม่แตะ storage ทำงานปกติ
	expect(t, assetCall(t, app, "used/list", scopeOwn("game_code", "scratch_card"), w.shareTok), 200, 200)
	expect(t, assetCall(t, app, "theme/detail/get", scopeOwn("game_code", "scratch_card"), w.shareTok), 200, 200)
	expect(t, assetCall(t, app, "games/list", nil, w.shareTok), 200, 200)

	// storage พร้อมแต่อัปล้มเหลว = 505802 · ไม่มี config
	fake := testutil.NewFakeS3(t)
	fake.FailPuts(true)
	expect(t, up(w.shareTok), 502, codeStorageFailed)
	if n := countRows(t, &models.AssetConfig{}, "1 = 1"); n != 0 {
		t.Errorf("อัปล้มเหลว ต้องไม่มี config: %d", n)
	}
	fake.FailPuts(false)
	expect(t, up(w.shareTok), 200, 200)
}

func TestAssetUseNow(t *testing.T) { // AST-21 · AST-22
	app := setup2(t)
	testutil.NewFakeS3(t)
	w := buildAssetWorld(t, app)
	sid := w.share.ID

	// ไม่ส่ง use_now = true: config + used ใน tx เดียว
	first := uploadScratch(t, app, w.shareTok, formOwn(), "icon_button")
	if !first.IsUsed || usedCount(t, &sid, "scratch_card", "icon_button") != 1 {
		t.Fatalf("use_now ค่าเริ่มต้น ต้องตั้ง used: %+v", first)
	}
	d := usedItems(t, app, w.shareTok, scopeOwn("game_code", "scratch_card"))
	if it := findItem(t, d, "icon_button"); it.ConfigID != first.ConfigID || it.Source != "self" || it.URL != first.URL {
		t.Fatalf("used/list: %+v", it)
	}

	// use_now=false: เก็บในคลังเฉยๆ ไม่แตะ used
	second := uploadScratch(t, app, w.shareTok, formOwn("use_now", "false"), "icon_button")
	if second.IsUsed {
		t.Error("use_now=false ต้องได้ is_used = false")
	}
	d = usedItems(t, app, w.shareTok, scopeOwn("game_code", "scratch_card"))
	if findItem(t, d, "icon_button").ConfigID != first.ConfigID {
		t.Error("use_now=false ต้องไม่เปลี่ยนไฟล์ที่ใช้งาน")
	}
	if usedCount(t, &sid, "scratch_card", "icon_button") != 1 {
		t.Error("ช่องหนึ่งมี used ได้ 1 แถว")
	}
	// use_now=true ครั้งถัดไป = แก้แถวเดิม (AST-22)
	third := uploadScratch(t, app, w.shareTok, formOwn("use_now", "true"), "icon_button")
	if usedCount(t, &sid, "scratch_card", "icon_button") != 1 {
		t.Error("use_now=true ต้องแก้แถวเดิม ไม่ใช่เพิ่มแถว")
	}
	d = usedItems(t, app, w.shareTok, scopeOwn("game_code", "scratch_card"))
	if findItem(t, d, "icon_button").ConfigID != third.ConfigID {
		t.Error("ต้องชี้ไฟล์ที่เพิ่งอัป")
	}
	// ค่าที่ไม่ใช่ true/false
	expect(t, postForm(t, app, "files/upload", formOwn("game_code", "scratch_card", "asset_type", "icon_button", "use_now", "maybe"),
		pngOf(t, 32, 32), w.shareTok), 200, 422)
}
