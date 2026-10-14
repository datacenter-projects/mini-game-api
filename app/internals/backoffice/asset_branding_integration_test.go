//go:build integration

package backoffice_test

import (
	"encoding/json"
	"strings"
	"testing"

	"app/app/models"
	"app/pkg/testutil"
)

func TestAssetBranding(t *testing.T) { // AST-41 · AST-42 · AST-43 · AST-47 · AST-53 · AST-56
	app := setup2(t)
	fake := testutil.NewFakeS3(t)
	w := buildAssetWorld(t, app)
	sid, aid := w.share.ID, w.agent.ID
	get := func(id uint, tok string) brandingData {
		t.Helper()
		r := assetCall(t, app, "branding/detail/get", byAgent(id), tok)
		expect(t, r, 200, 200)
		return decode[brandingData](t, r)
	}
	setName := func(id uint, name any, tok string) testutil.Response {
		return assetCall(t, app, "branding/detail/update", byAgent(id, "site_name", name), tok)
	}
	logo := func(id uint, file []byte, tok string) testutil.Response {
		return postForm(t, app, "branding/logo/upload", map[string]string{"agent_id": itoa(id)}, file, tok)
	}

	// AST-41: ไม่มีใครตั้ง = source "" ทุก field "" (ไม่มี GLOBAL)
	if d := get(sid, w.shareTok); d != (brandingData{}) {
		t.Fatalf("ยังไม่มีใครตั้ง: %+v", d)
	}
	// branding ไม่มี scope · scope GLOBAL = 422 (AST-01)
	expect(t, assetCall(t, app, "branding/detail/update", map[string]any{"scope": "GLOBAL", "agent_id": sid, "site_name": "X"}, w.saTok), 200, 422)

	// AST-42: site_name
	for name, v := range map[string]any{"ว่าง": "", "มีแต่ช่องว่าง": "   ", "51 ตัวอักษร": strings.Repeat("a", 51), "newline กลางชื่อ": "Share\nOne",
		"tab กลางชื่อ": "Share\tOne", "NUL": "Share\x00One", "null": nil, "ตัวเลข": 123} {
		if r := setName(sid, v, w.shareTok); r.Code != 422 && r.Code != 400 {
			t.Errorf("site_name %s: want 422 got %d (%s)", name, r.Code, r.Msg)
		}
	}
	expect(t, assetCall(t, app, "branding/detail/update", byAgent(sid, "site_name", "X", "site_theme", "dark"), w.shareTok), 200, 422)
	expect(t, assetCall(t, app, "branding/detail/update", byAgent(sid), w.shareTok), 200, 422) // ไม่ส่ง site_name
	if n := countRows(t, &models.SiteBranding{}, "1 = 1"); n != 0 {
		t.Fatalf("422 ต้องไม่เขียนอะไร: %d", n)
	}
	expect(t, setName(sid, "  Share One  ", w.shareTok), 200, 200)
	if d := get(sid, w.shareTok); d.Source != "self" || d.SiteName != "Share One" || d.Logo != "" || d.InheritedFrom != "" {
		t.Fatalf("หลังตั้งชื่อ (ตัดช่องว่าง): %+v", d)
	}
	expect(t, setName(sid, strings.Repeat("ก", 50), w.shareTok), 200, 200) // 50 ตัวอักษรไทย (เกิน 50 byte) ผ่าน
	expect(t, setName(sid, "Share One", w.shareTok), 200, 200)

	// AST-43: โลโก้ — ชนิด / ขนาดไฟล์ · เขียนทับ · ไฟล์เดิมไม่ถูกลบ · ไม่แตะชื่อ
	expect(t, logo(sid, []byte("<svg></svg>"), w.shareTok), 200, codeFileType)
	expect(t, logo(sid, []byte("%PDF-1.4"), w.shareTok), 200, codeFileType)
	expect(t, logo(sid, padded(pngOf(t, 16, 16), 5*1024*1024+1), w.shareTok), 200, codeFileTooLarge)
	expect(t, logo(sid, nil, w.shareTok), 200, 422)
	if fake.Count() != 0 {
		t.Fatalf("ไฟล์ผิดต้องไม่ขึ้น bucket: %d", fake.Count())
	}
	r := logo(sid, pngOf(t, 100, 40), w.shareTok) // ไม่ตรวจขนาดภาพ
	expect(t, r, 200, 200)
	var l1 struct {
		URL string `json:"url"`
	}
	_ = json.Unmarshal(r.Data, &l1)
	if !strings.HasPrefix(l1.URL, cdnBase+"sites/") || !strings.Contains(l1.URL, "/logo/") {
		t.Fatalf("รูปแบบ key ของโลโก้: %s", l1.URL)
	}
	obj, ok := fake.Object(keyOf(l1.URL))
	if !ok || obj.ContentType != "image/png" || obj.CacheControl != "public, max-age=31536000, immutable" {
		t.Errorf("object ของโลโก้: %+v ok=%v", obj, ok)
	}
	r = logo(sid, pngOf(t, 50, 50), w.shareEditTok) // sub อัปได้
	expect(t, r, 200, 200)
	var l2 struct {
		URL string `json:"url"`
	}
	_ = json.Unmarshal(r.Data, &l2)
	if l2.URL == l1.URL {
		t.Error("ต้องใช้ key ใหม่ทุกครั้ง")
	}
	if _, ok := fake.Object(keyOf(l1.URL)); !ok {
		t.Error("ไฟล์โลโก้เดิมต้องไม่ถูกลบ (lead GQ5)")
	}
	if d := get(sid, w.shareTok); d.Logo != l2.URL || d.SiteName != "Share One" {
		t.Fatalf("เขียนทับโลโก้ ชื่อต้องไม่หาย: %+v", d)
	}
	if n := countRows(t, &models.SiteBranding{}, "agent_id = ?", sid); n != 1 {
		t.Errorf("1 แถวต่อบัญชี: %d", n)
	}

	// สิทธิ์ (AST-04 / AST-02): สายล่าง = 405301 · นอกสาย = 402402 · Share Master = 405301
	expect(t, setName(aid, "X", w.shareTok), 200, codeNoPermission)
	expect(t, logo(aid, pngOf(t, 10, 10), w.shareTok), 200, codeNoPermission)
	expect(t, assetCall(t, app, "branding/reset", byAgent(aid), w.shareTok), 200, codeNoPermission)
	expect(t, setName(w.csm.ID, "X", w.shareTok), 200, codeOutsideLine)
	expect(t, setName(w.sm1.ID, "X", w.sm1Tok), 200, codeNoPermission)
	expect(t, logo(w.sm1.ID, pngOf(t, 10, 10), w.sm1Tok), 200, codeNoPermission)
	// ชั้นบนดูของสายล่างได้ · view sub ดูได้
	if d := get(aid, w.shareTok); d.SiteName != "Share One" {
		t.Errorf("share01 ดูของ agent01: %+v", d)
	}
	if d := get(sid, w.shareViewTok); d.SiteName != "Share One" {
		t.Errorf("view sub: %+v", d)
	}

	// AST-47 · AST-53: agent01 ไม่ตั้ง = ได้ของ share01 ทั้งแถว (upline + inherited_from)
	if d := get(aid, w.agentTok); d.Source != "upline" || d.InheritedFrom != "share01" || d.SiteName != "Share One" || d.Logo != l2.URL {
		t.Fatalf("agent01 ต้องได้ของ share01: %+v", d)
	}
	// agent01 ตั้งแค่โลโก้ → ได้ของตัวเอง (ชื่อว่าง) ไม่เอาชื่อของ share01 มาผสม
	r = logo(aid, pngOf(t, 30, 30), w.agentTok)
	expect(t, r, 200, 200)
	d := get(aid, w.agentTok)
	if d.Source != "self" || d.InheritedFrom != "" || d.SiteName != "" || d.Logo == "" || d.Logo == l2.URL {
		t.Fatalf("agent01 ตั้งแค่โลโก้ ต้องไม่ผสมกับ share01: %+v", d)
	}
	// agent01 ตั้งชื่อ → ได้ของตัวเองครบ · reset → กลับไปใช้ของ share01
	expect(t, setName(aid, "Agent One", w.agentTok), 200, 200)
	if d = get(aid, w.agentTok); d.SiteName != "Agent One" || d.Source != "self" {
		t.Fatalf("agent01 ตั้งชื่อ: %+v", d)
	}
	expect(t, assetCall(t, app, "branding/reset", byAgent(aid), w.agentTok), 200, 200)
	if d = get(aid, w.agentTok); d.Source != "upline" || d.SiteName != "Share One" || d.InheritedFrom != "share01" {
		t.Fatalf("หลัง reset: %+v", d)
	}
	if n := countRows(t, &models.SiteBranding{}, "agent_id = ?", aid); n != 0 {
		t.Error("reset ต้องลบแถวของบัญชี")
	}
	if _, ok := fake.Object(keyOf(l2.URL)); !ok {
		t.Error("reset ไม่ลบไฟล์โลโก้ใน storage (AST-29)")
	}
	// สายลึก: agent02 ใต้ agent01 ได้ของ share01 (ข้ามชั้นที่ไม่ได้ตั้ง)
	a2, _ := mustCreate(t, app, w.agentTok, agentBody("AGENT", "agent02", nil, childPT(50, 0, 0, 0)))
	if d = get(a2.ID, w.agentTok); d.Source != "upline" || d.InheritedFrom != "share01" {
		t.Fatalf("agent02: %+v", d)
	}

	// share01 reset → ทั้งสายไม่มี branding
	expect(t, assetCall(t, app, "branding/reset", byAgent(sid), w.shareTok), 200, 200)
	if d = get(aid, w.agentTok); d != (brandingData{}) {
		t.Fatalf("หลัง share01 reset: %+v", d)
	}

	// Share Master ได้ของ CSM (lead AV1) — ทั้งสองคนเหมือนกัน
	expect(t, setName(w.csm.ID, "CSM Brand", w.csmTok), 200, 200)
	for _, c := range []struct {
		tok string
		id  uint
	}{{w.sm1Tok, w.sm1.ID}, {w.sm2Tok, w.sm2.ID}} {
		if d = get(c.id, c.tok); d.Source != "upline" || d.InheritedFrom != "csm01" || d.SiteName != "CSM Brand" {
			t.Errorf("Share Master ต้องได้ของ csm01: %+v", d)
		}
	}
}

func TestAssetBundle(t *testing.T) { // AST-45 · AST-46 · AST-47 · AST-48 (preview) · AST-53
	app := setup2(t)
	testutil.NewFakeS3(t)
	w := buildAssetWorld(t, app)
	sid, aid := w.share.ID, w.agent.ID
	bundle := func(id uint, game string, tok string) (bundleData, testutil.Response) {
		t.Helper()
		r := assetCall(t, app, "bundle/detail/get", byAgent(id, "game_code", game), tok)
		expect(t, r, 200, 200)
		return decode[bundleData](t, r), r
	}

	// ไม่มีอะไรเลย: assets {} · source "" · theme ค่ากลาง · site ว่าง (ไม่ใช่ error · ไม่มี null)
	b, raw := bundle(aid, "scratch_card", w.agentTok)
	if b.Source != "" || b.Assets == nil || len(b.Assets) != 0 || b.Site.SiteName != "" || b.Site.Logo != "" {
		t.Fatalf("bundle ว่าง: %+v", b)
	}
	if strings.Contains(string(raw.Data), "null") {
		t.Errorf("ไม่มี null ใน response: %s", raw.Data)
	}
	var th map[string]any
	_ = json.Unmarshal(b.Theme, &th)
	if th["source"] != "global" || th["font_family"] != "noto" {
		t.Errorf("theme ค่ากลาง: %v", th)
	}
	// AST-48: theme ใน bundle ไม่มี inherited_from (ไม่ส่ง username ชั้นบนให้ผู้เล่น) · shades ครบ
	if _, has := th["inherited_from"]; has {
		t.Error("theme ใน bundle ต้องไม่มี inherited_from")
	}
	if s1, _ := th["main_color_1_shades"].(map[string]any); len(s1) != 19 {
		t.Errorf("shades: %v", th["main_color_1_shades"])
	}

	// ใส่ asset / theme / branding หลายชั้น
	gBack := uploadScratch(t, app, w.saTok, formGlobal(), "background")
	sIcon := uploadScratch(t, app, w.shareTok, formAgent(sid), "icon_button")
	aWin := uploadScratch(t, app, w.agentTok, formAgent(aid), "winner")
	expect(t, assetCall(t, app, "theme/detail/update", putTheme(themeBody("scratch_card", "font_family", "mitr"), scopeAgent(sid)), w.shareTok), 200, 200)
	expect(t, assetCall(t, app, "branding/detail/update", byAgent(sid, "site_name", "Share One"), w.shareTok), 200, 200)

	b, _ = bundle(aid, "scratch_card", w.agentTok)
	want := map[string]string{"background": gBack.URL, "icon_button": sIcon.URL, "winner": aWin.URL}
	if len(b.Assets) != 3 {
		t.Fatalf("assets ต้องมีเฉพาะช่องที่มีไฟล์: %+v", b.Assets)
	}
	for slot, url := range want {
		if b.Assets[slot] != url {
			t.Errorf("assets[%s] = %q want %q", slot, b.Assets[slot], url)
		}
	}
	if b.Source != "mixed" || b.Site.SiteName != "Share One" {
		t.Errorf("bundle: source=%q site=%+v", b.Source, b.Site)
	}
	_ = json.Unmarshal(b.Theme, &th)
	if th["font_family"] != "mitr" || th["source"] != "upline" {
		t.Errorf("theme ต้องไล่สายจาก share01: %v", th)
	}

	// BO preview = used/list + theme/detail/get (กฎเดียวกัน — AST-48)
	d := usedItems(t, app, w.agentTok, scopeAgent(aid, "game_code", "scratch_card"))
	n := 0
	for _, it := range d.Items {
		if it.URL != "" {
			n++
			if b.Assets[it.AssetType] != it.URL {
				t.Errorf("bundle กับ used/list ไม่ตรงกัน: %s", it.AssetType)
			}
		}
	}
	if n != len(b.Assets) {
		t.Errorf("จำนวนช่องที่มีไฟล์ไม่ตรงกัน: used/list %d bundle %d", n, len(b.Assets))
	}
	rt := assetCall(t, app, "theme/detail/get", scopeAgent(aid, "game_code", "scratch_card"), w.agentTok)
	td := decode[themeData](t, rt)
	if td.FontFamily != th["font_family"] || td.MainColor1 != th["main_color_1"] || td.Source != th["source"] {
		t.Errorf("bundle.theme ต้องตรงกับ theme/detail/get: %+v vs %v", td, th)
	}

	// ทุกช่องมาจากที่เดียว = ค่านั้น (ไม่ใช่ mixed)
	b, _ = bundle(w.com.ID, "scratch_card", w.comTok)
	if b.Source != "global" || len(b.Assets) != 1 {
		t.Errorf("comp01: %+v", b)
	}
	// เกมอื่นไม่ปน
	if b, _ = bundle(aid, "coin_toss", w.agentTok); len(b.Assets) != 0 || b.Source != "" {
		t.Errorf("coin_toss: %+v", b)
	}

	// สิทธิ์: ชั้นบนดูของสายล่างได้ · นอกสาย 402402 · ไม่ส่ง agent_id 422 · game ผิด 405401
	if bb, _ := bundle(aid, "scratch_card", w.shareTok); len(bb.Assets) != 3 {
		t.Errorf("share01 ดู bundle ของ agent01: %+v", bb)
	}
	expect(t, assetCall(t, app, "bundle/detail/get", byAgent(w.csm.ID, "game_code", "scratch_card"), w.shareTok), 200, codeOutsideLine)
	expect(t, assetCall(t, app, "bundle/detail/get", map[string]any{"game_code": "scratch_card"}, w.shareTok), 200, 422)
	expect(t, assetCall(t, app, "bundle/detail/get", byAgent(aid, "game_code", "plinko"), w.agentTok), 200, codeSlotNotFound)
	expect(t, assetCall(t, app, "bundle/detail/get", byAgent(aid), w.agentTok), 200, 422)
	// Share Master ได้ bundle จาก CSM
	expect(t, assetCall(t, app, "branding/detail/update", byAgent(w.csm.ID, "site_name", "CSM"), w.csmTok), 200, 200)
	sb, _ := bundle(w.sm1.ID, "scratch_card", w.sm1Tok)
	if sb.Site.SiteName != "CSM" {
		t.Errorf("sm01 bundle: %+v", sb)
	}
}
