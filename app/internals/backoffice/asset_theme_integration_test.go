//go:build integration

package backoffice_test

import (
	"strings"
	"testing"
	"time"

	"app/app/models"
	"app/pkg/testutil"
	"app/platform/database"
)

func themeBody(game string, kv ...any) map[string]any {
	m := map[string]any{"game_code": game, "font_family": "kanit", "main_color_1": "#1E90FF", "main_color_2": "#FFFFFF",
		"stats_bar_color": "main_color_2", "stats_bar_font_family": "prompt"}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i].(string)] = kv[i+1]
	}
	return m
}

func putTheme(kv map[string]any, scope map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range kv {
		out[k] = v
	}
	for k, v := range scope {
		out[k] = v
	}
	return out
}

func TestAssetReadPermission(t *testing.T) { // AST-05 · AST-06
	app := setup2(t)
	testutil.NewFakeS3(t)
	w := buildAssetWorld(t, app)
	used := func(body map[string]any, tok string) testutil.Response {
		return assetCall(t, app, "used/list", body, tok)
	}
	g := scopeGlobal("game_code", "scratch_card")

	// AST-05: อ่านค่ากลางได้ทุกบัญชีที่มีสิทธิ์ asset ≥ view
	for name, tok := range map[string]string{"agent01": w.agentTok, "share01": w.shareTok, "sm01": w.sm1Tok, "comp01": w.comTok,
		"sa": w.saTok, "view sub": w.shareViewTok, "edit sub": w.shareEditTok} {
		if r := used(g, tok); r.Code != 200 {
			t.Errorf("%s อ่าน GLOBAL: code %d (%s)", name, r.Code, r.Msg)
		}
	}
	// ไม่มีสิทธิ์ asset (off) = 402303 ทุกเส้น (AST-06 · AUTH-26)
	expect(t, used(g, w.shareNone), 200, codeNoMenu)
	expect(t, assetCall(t, app, "games/list", nil, w.shareNone), 200, codeNoMenu)
	expect(t, assetCall(t, app, "slots/list", map[string]any{"game_code": "scratch_card"}, w.shareNone), 200, codeNoMenu)
	expect(t, assetCall(t, app, "bundle/detail/get", byAgent(w.share.ID, "game_code", "scratch_card"), w.shareNone), 200, codeNoMenu)
	// view sub: อ่านได้ เขียนไม่ได้
	expect(t, used(scopeAgent(w.share.ID, "game_code", "scratch_card"), w.shareViewTok), 200, 200)
	expect(t, assetCall(t, app, "theme/detail/update", putTheme(themeBody("scratch_card"), scopeAgent(w.share.ID)), w.shareViewTok), 200, codeNoMenu)
	expect(t, assetCall(t, app, "branding/detail/update", byAgent(w.share.ID, "site_name", "X"), w.shareViewTok), 200, codeNoMenu)

	// ตัวเอง · สายล่าง · นอกสาย
	expect(t, used(scopeAgent(w.share.ID, "game_code", "scratch_card"), w.shareTok), 200, 200)
	expect(t, used(scopeAgent(w.agent.ID, "game_code", "scratch_card"), w.shareTok), 200, 200)
	expect(t, used(scopeAgent(w.agent.ID, "game_code", "scratch_card"), w.saTok), 200, 200)
	expect(t, used(scopeAgent(w.csm.ID, "game_code", "scratch_card"), w.shareTok), 200, codeOutsideLine)
	expect(t, used(scopeAgent(w.share.ID, "game_code", "scratch_card"), w.agentTok), 200, codeOutsideLine) // ชั้นบนของตัวเอง
	expect(t, used(scopeAgent(w.agent.ID, "game_code", "scratch_card"), w.sm1Tok), 200, codeOutsideLine)
	expect(t, used(scopeAgent(999999, "game_code", "scratch_card"), w.shareTok), 200, codeOutsideLine) // ไม่มีบัญชีนี้ = นอกสาย
	// sub อ่านในนามเจ้าของ: sub ของ share01 ดูของ agent01 (สายล่างของเจ้าของ) ได้
	expect(t, used(scopeAgent(w.agent.ID, "game_code", "scratch_card"), w.shareViewTok), 200, 200)
	// Share Master อ่านค่าของตัวเอง (ที่ได้จาก CSM) ได้
	expect(t, assetCall(t, app, "theme/detail/get", scopeAgent(w.sm1.ID, "game_code", "scratch_card"), w.sm1Tok), 200, 200)

	// AST-06: ADMIN ไม่มีเมนู asset · ไม่ login = 401 · บัญชีถูกระงับ = 401311
	admin := createAccount(t, "admin01", models.AgentRoleAdmin, nil)
	adminTok := loginReady(t, app, models.AccountTypeAgent, admin.ID, "admin01")
	expect(t, assetCall(t, app, "games/list", nil, adminTok), 200, codeNoMenu)
	expect(t, assetCall(t, app, "used/list", g, ""), 401, 401202)
	setCols(t, models.AccountTypeAgent, w.share.ID, map[string]any{"status": models.AgentStatusSuspended})
	expect(t, used(g, w.shareTok), 403, codeSuspended)
	expect(t, used(g, w.agentTok), 403, codeSuspended) // ลูกของบัญชีที่ถูกระงับก็ถูกระงับ (AUTH-53)
	expect(t, used(g, w.comTok), 200, 200)
}

func TestAssetTheme(t *testing.T) { // AST-33 – AST-40 · lead GE10-1
	app := setup2(t)
	testutil.NewFakeS3(t)
	w := buildAssetWorld(t, app)
	sid, aid := w.share.ID, w.agent.ID
	get := func(id uint, tok string) themeData {
		t.Helper()
		r := assetCall(t, app, "theme/detail/get", scopeAgent(id, "game_code", "scratch_card"), tok)
		expect(t, r, 200, 200)
		return decode[themeData](t, r)
	}
	upd := func(id uint, body map[string]any, tok string) testutil.Response {
		return assetCall(t, app, "theme/detail/update", putTheme(body, scopeAgent(id)), tok)
	}

	// AST-37: ไม่มีใครตั้ง = ค่ากลางทั้ง 5 ค่า (seed)
	d := get(sid, w.shareTok)
	if d.Source != "global" || d.InheritedFrom != "" || d.FontFamily != "noto" || d.MainColor1 != "#f7447e" || d.MainColor2 != "#ffffff" ||
		d.StatsBarColor != "main_color_1" || d.StatsBarColorHex != "#f7447e" || d.StatsBarFontFamily != "prompt" {
		t.Fatalf("ค่ากลาง: %+v", d)
	}
	// AST-39: shades ครบ 19 ระดับ · 900 = สีหลัก · คำนวณตามสูตร
	if len(d.Shades1) != 19 || d.Shades1["900"] != "#f7447e" || d.Shades1["50"] != "#feecf2" || d.Shades1["1800"] != "#19070d" {
		t.Errorf("shades: %+v", d.Shades1)
	}

	// AST-33 · AST-34 · AST-35 · AST-36: 422 พร้อมบอก field
	bad := []struct {
		name  string
		body  map[string]any
		field string
	}{
		{"ขาด font_family", themeBody("scratch_card", "font_family", ""), "font_family"},
		{"ไม่ส่ง main_color_1", func() map[string]any { b := themeBody("scratch_card"); delete(b, "main_color_1"); return b }(), "main_color_1"},
		{"ขาด main_color_2", themeBody("scratch_card", "main_color_2", ""), "main_color_2"},
		{"ขาด stats_bar_color", themeBody("scratch_card", "stats_bar_color", ""), "stats_bar_color"},
		{"ขาด stats_bar_font_family", themeBody("scratch_card", "stats_bar_font_family", ""), "stats_bar_font_family"},
		{"ฟอนต์ตัวพิมพ์ใหญ่", themeBody("scratch_card", "font_family", "Kanit"), "font_family"},
		{"ฟอนต์นอก whitelist", themeBody("scratch_card", "font_family", "Comic Sans"), "font_family"},
		{"CSS injection", themeBody("scratch_card", "font_family", "noto; color:red"), "font_family"},
		{"stats bar font นอก whitelist", themeBody("scratch_card", "stats_bar_font_family", "Arial"), "stats_bar_font_family"},
		{"สี #RGB", themeBody("scratch_card", "main_color_1", "#FFF"), "main_color_1"},
		{"สีไม่มี #", themeBody("scratch_card", "main_color_1", "FF0000"), "main_color_1"},
		{"สีไม่ใช่ hex", themeBody("scratch_card", "main_color_2", "#GG0000"), "main_color_2"},
		{"สียาวเกิน", themeBody("scratch_card", "main_color_2", "#ff00001"), "main_color_2"},
		{"stats_bar_color เป็น hex", themeBody("scratch_card", "stats_bar_color", "#ffffff"), "stats_bar_color"},
		{"null", themeBody("scratch_card", "font_family", nil), "font_family"},
	}
	for _, b := range bad {
		r := upd(sid, b.body, w.shareTok)
		if r.Code != 422 || !strings.Contains(r.Msg, b.field) {
			t.Errorf("%s: want 422 พร้อม field %q, got code %d msg %q", b.name, b.field, r.Code, r.Msg)
		}
	}
	if n := countRows(t, &models.GameTheme{}, "agent_id IS NOT NULL"); n != 0 {
		t.Fatalf("422 ต้องไม่เขียนอะไร: %d แถว", n)
	}
	expect(t, upd(sid, themeBody("plinko"), w.shareTok), 200, codeSlotNotFound)

	// เขียนสำเร็จ: สีเป็นตัวพิมพ์เล็ก · source = self · stats bar ตามสีหลัก (AST-35 · AST-36)
	expect(t, upd(sid, themeBody("scratch_card"), w.shareTok), 200, 200)
	d = get(sid, w.shareTok)
	if d.Source != "self" || d.FontFamily != "kanit" || d.MainColor1 != "#1e90ff" || d.MainColor2 != "#ffffff" ||
		d.StatsBarColor != "main_color_2" || d.StatsBarColorHex != "#ffffff" {
		t.Fatalf("หลังตั้ง: %+v", d)
	}
	if d.Shades1["900"] != "#1e90ff" || d.Shades1["50"] != "#e9f4ff" || d.Shades1["1800"] != "#030e1a" || d.Shades2["1000"] != "#e6e6e6" {
		t.Errorf("shades ของ #1e90ff / #ffffff: %+v / %+v", d.Shades1, d.Shades2)
	}
	// AST-36: เปลี่ยน main_color_2 แล้ว stats_bar_color_hex ตาม
	expect(t, upd(sid, themeBody("scratch_card", "main_color_2", "#00FF00"), w.shareTok), 200, 200)
	if d = get(sid, w.shareTok); d.StatsBarColorHex != "#00ff00" {
		t.Errorf("stats_bar_color_hex ต้องตามสีหลัก: %+v", d)
	}
	expect(t, upd(sid, themeBody("scratch_card"), w.shareTok), 200, 200)

	// GE10-1: ส่งค่าเดิมทั้งชุด = 200 ไม่แตะ updated_* ไม่สร้างแถวใหม่
	var before models.GameTheme
	if err := database.DBConn.Where("agent_id = ? AND game_code = ?", sid, "scratch_card").Take(&before).Error; err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	expect(t, upd(sid, themeBody("scratch_card", "main_color_1", "#1e90ff"), w.shareEditTok), 200, 200) // sub ส่งค่าเดิม (ตัวพิมพ์ต่างกัน)
	var after models.GameTheme
	if err := database.DBConn.Where("agent_id = ? AND game_code = ?", sid, "scratch_card").Take(&after).Error; err != nil {
		t.Fatal(err)
	}
	if !after.UpdatedAt.Equal(before.UpdatedAt) || after.UpdatedBy != before.UpdatedBy || after.ID != before.ID {
		t.Errorf("ส่งค่าเดิมต้องไม่แตะแถว: %+v → %+v", before, after)
	}
	if n := countRows(t, &models.GameTheme{}, "agent_id = ?", sid); n != 1 {
		t.Errorf("ต้องมี 1 แถว: %d", n)
	}
	// ส่งค่าต่างแล้ว sub เป็นคนแก้ → updated_by = username ของ sub · created_by ไม่เปลี่ยน
	expect(t, upd(sid, themeBody("scratch_card", "font_family", "mitr"), w.shareEditTok), 200, 200)
	database.DBConn.Where("agent_id = ? AND game_code = ?", sid, "scratch_card").Take(&after)
	if after.UpdatedBy != "share01@design" || after.CreatedBy != "share01" {
		t.Errorf("created_by / updated_by: %+v", after)
	}
	expect(t, upd(sid, themeBody("scratch_card"), w.shareTok), 200, 200)

	// ไล่สาย: agent01 ไม่ตั้ง = ได้ของ share01 ทั้งชุด (upline + inherited_from)
	d = get(aid, w.agentTok)
	if d.Source != "upline" || d.InheritedFrom != "share01" || d.FontFamily != "kanit" || d.MainColor1 != "#1e90ff" {
		t.Fatalf("agent01 ต้องได้ของ share01: %+v", d)
	}
	// GE10-1 (ไล่สายแล้ว): agent01 ส่งค่าเท่ากับที่ได้จาก share01 = 200 ไม่สร้างแถว (ยังได้ค่าของชั้นบนต่อไป)
	expect(t, upd(aid, themeBody("scratch_card"), w.agentTok), 200, 200)
	if n := countRows(t, &models.GameTheme{}, "agent_id = ?", aid); n != 0 {
		t.Fatalf("ส่งค่าเท่ากับที่สืบทอดมา ต้องไม่สร้างแถว: %d", n)
	}
	// ส่งค่าต่างจากที่ใช้อยู่ 1 ค่า = สร้างแถวของ agent01 ทั้ง 5 ค่า (ไม่ผสมกับ share01)
	expect(t, upd(aid, themeBody("scratch_card", "font_family", "sarabun"), w.agentTok), 200, 200)
	d = get(aid, w.agentTok)
	if d.Source != "self" || d.InheritedFrom != "" || d.FontFamily != "sarabun" || d.MainColor1 != "#1e90ff" {
		t.Fatalf("agent01 หลังตั้งเอง: %+v", d)
	}
	// reset: agent01 กลับไปใช้ของ share01 · share01 reset แล้วทั้งสายกลับไปค่ากลาง
	expect(t, assetCall(t, app, "theme/reset", scopeAgent(aid, "game_code", "scratch_card"), w.agentTok), 200, 200)
	if d = get(aid, w.agentTok); d.Source != "upline" || d.InheritedFrom != "share01" {
		t.Fatalf("หลัง reset: %+v", d)
	}
	expect(t, assetCall(t, app, "theme/reset", scopeAgent(sid, "game_code", "scratch_card"), w.shareTok), 200, 200)
	if d = get(aid, w.agentTok); d.Source != "global" || d.FontFamily != "noto" {
		t.Fatalf("share01 reset แล้ว agent01 ต้องได้ค่ากลาง: %+v", d)
	}

	// AST-37: แก้ค่ากลางมีผลกับบัญชีที่ไม่เคยตั้งทันที · บัญชีที่ตั้งเองไม่เปลี่ยน
	expect(t, upd(sid, themeBody("scratch_card"), w.shareTok), 200, 200)
	r := assetCall(t, app, "theme/detail/update", scopeGlobal(
		"game_code", "scratch_card", "font_family", "mitr", "main_color_1", "#112233", "main_color_2", "#445566",
		"stats_bar_color", "main_color_1", "stats_bar_font_family", "noto"), w.saTok)
	expect(t, r, 200, 200)
	if d = get(w.com.ID, w.comTok); d.Source != "global" || d.FontFamily != "mitr" || d.MainColor1 != "#112233" {
		t.Errorf("comp01 ต้องได้ค่ากลางใหม่: %+v", d)
	}
	if d = get(sid, w.shareTok); d.Source != "self" || d.FontFamily != "kanit" {
		t.Errorf("share01 ที่ตั้งเองต้องไม่เปลี่ยน: %+v", d)
	}
	// AST-38: กฎเดียวกันทั้ง GLOBAL — GLOBAL ส่งสีผิดก็ 422 · ไม่มี used/reset ของค่ากลาง
	expect(t, assetCall(t, app, "theme/detail/update", scopeGlobal("game_code", "scratch_card", "font_family", "kanit",
		"main_color_1", "#FFF", "main_color_2", "#ffffff", "stats_bar_color", "main_color_1", "stats_bar_font_family", "prompt"), w.saTok), 200, 422)
	if n := countRows(t, &models.GameTheme{}, "agent_id IS NULL"); n != 3 {
		t.Errorf("แถวค่ากลางต้องมี 3 (หนึ่งต่อเกม): %d", n)
	}
	// เกมอื่นไม่เปลี่ยน
	r = assetCall(t, app, "theme/detail/get", scopeAgent(sid, "game_code", "coin_toss"), w.shareTok)
	if d = decode[themeData](t, r); d.Source != "global" || d.FontFamily != "noto" {
		t.Errorf("coin_toss: %+v", d)
	}
}

func TestAssetThemeShareMasterFollowsCSM(t *testing.T) { // AST-37 · lead AV1
	app := setup2(t)
	w := buildAssetWorld(t, app)
	expect(t, assetCall(t, app, "theme/detail/update", putTheme(themeBody("scratch_card", "font_family", "sarabun"), scopeAgent(w.csm.ID)), w.csmTok), 200, 200)
	for _, c := range []struct {
		tok string
		id  uint
	}{{w.sm1Tok, w.sm1.ID}, {w.sm2Tok, w.sm2.ID}} {
		r := assetCall(t, app, "theme/detail/get", scopeAgent(c.id, "game_code", "scratch_card"), c.tok)
		expect(t, r, 200, 200)
		if d := decode[themeData](t, r); d.Source != "upline" || d.InheritedFrom != "csm01" || d.FontFamily != "sarabun" {
			t.Errorf("Share Master ต้องได้ของ csm01: %+v", d)
		}
	}
	// Share Master ตั้งเองไม่ได้ · csm01 แก้ให้ Share Master ตรงๆ ก็ไม่ได้ (สืบทอดเอง)
	expect(t, assetCall(t, app, "theme/detail/update", putTheme(themeBody("scratch_card"), scopeAgent(w.sm1.ID)), w.sm1Tok), 200, codeNoPermission)
	expect(t, assetCall(t, app, "theme/detail/update", putTheme(themeBody("scratch_card"), scopeAgent(w.sm1.ID)), w.csmTok), 200, codeNoPermission)
	expect(t, assetCall(t, app, "theme/reset", scopeAgent(w.sm1.ID, "game_code", "scratch_card"), w.sm1Tok), 200, codeNoPermission)
	// csm01 แก้แล้ว Share Master ทุกคนเปลี่ยนตามทันที
	expect(t, assetCall(t, app, "theme/detail/update", putTheme(themeBody("scratch_card", "font_family", "mitr"), scopeAgent(w.csm.ID)), w.csmTok), 200, 200)
	r := assetCall(t, app, "theme/detail/get", scopeAgent(w.sm2.ID, "game_code", "scratch_card"), w.sm2Tok)
	if d := decode[themeData](t, r); d.FontFamily != "mitr" {
		t.Errorf("sm02 ต้องตามการแก้ของ csm01: %+v", d)
	}
}
