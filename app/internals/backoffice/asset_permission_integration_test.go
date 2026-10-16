//go:build integration

package backoffice_test

import (
	"strings"
	"testing"

	"app/app/models"
	"app/pkg/testutil"
)

const (
	codeNoPermission   = 405301
	codeNoMenu         = 402303
	codeOutsideLine    = 402402
	codeSlotNotFound   = 405401
	codeConfigNotFound = 405402
	codeInUse          = 405403
	codeSlotRetired    = 405304
	codeImageSize      = 405101
	codeFileType       = 405102
	codeFileTooLarge   = 405103
	codeStorageDown    = 505801
	codeStorageFailed  = 505802
	codeSuspended      = 401311
)

func TestAssetWritePermission(t *testing.T) { // AST-02 · AST-03 · AST-04 · AST-08 (เป้าหมายจาก token — lead 2026-10-16)
	app := setup2(t)
	testutil.NewFakeS3(t)
	w := buildAssetWorld(t, app)

	up := func(tok string, fields map[string]string) testutil.Response {
		f := map[string]string{"game_code": "scratch_card", "asset_type": "icon_button"}
		for k, v := range fields {
			f[k] = v
		}
		return postForm(t, app, "files/upload", f, pngOf(t, 32, 32), tok)
	}

	// AST-03 · GLOBAL
	r := up(w.saTok, formGlobal())
	expect(t, r, 200, 200)
	if d := decode[uploadData](t, r); d.CreatedBy != "superadmin" || d.Scope != "GLOBAL" || d.AgentID != 0 {
		t.Fatalf("GLOBAL upload: %+v", d)
	}
	r = up(w.saEditTok, formGlobal())
	expect(t, r, 200, 200)
	if d := decode[uploadData](t, r); d.CreatedBy != "superadmin@ops" {
		t.Fatalf("sub ของ superadmin ที่มี asset=edit: created_by = %q", d.CreatedBy)
	}
	expect(t, up(w.saViewTok, formGlobal()), 200, codeNoMenu) // asset = view อัปไม่ได้
	expect(t, up(w.shareTok, formGlobal()), 200, codeNoPermission)
	expect(t, up(w.comTok, formGlobal()), 200, codeNoPermission)
	expect(t, up(w.shareEditTok, formGlobal()), 200, codeNoPermission) // sub ของ share ก็ไม่ใช่ Superadmin

	// AST-02 · ตั้งของตัวเอง (บัญชีหลักทุกประเภทยกเว้น Share Master)
	for name, tok := range map[string]string{"comp01": w.comTok, "share01": w.shareTok, "agent01": w.agentTok, "csm01": w.csmTok} {
		id := map[string]uint{"comp01": w.com.ID, "share01": w.share.ID, "agent01": w.agent.ID, "csm01": w.csm.ID}[name]
		r := up(tok, formOwn())
		if r.Code != 200 {
			t.Errorf("%s ตั้งของตัวเอง: code %d (%s)", name, r.Code, r.Msg)
			continue
		}
		if d := decode[uploadData](t, r); d.Scope != "AGENT" || d.AgentID != id { // เป้าหมาย = บัญชีของ token
			t.Errorf("%s: รูปต้องเป็นของบัญชีที่ login: %+v", name, d)
		}
	}
	// AST-02 · Share Master ตั้งเองไม่ได้ (ใช้ของ CSM)
	expect(t, up(w.sm1Tok, formOwn()), 200, codeNoPermission)
	expect(t, assetCall(t, app, "theme/detail/update", scopeOwn("game_code", "scratch_card", "font_family", "kanit",
		"main_color_1", "#112233", "main_color_2", "#ffffff", "stats_bar_color", "main_color_1", "stats_bar_font_family", "prompt"), w.sm1Tok), 200, codeNoPermission)
	expect(t, assetCall(t, app, "branding/detail/update", own("site_name", "X"), w.sm1Tok), 200, codeNoPermission)
	// AST-02 · Superadmin ตั้ง AGENT ของตัวเองไม่ได้ (ค่าของ Superadmin คือ GLOBAL)
	expect(t, up(w.saTok, formOwn()), 200, codeNoPermission)

	// AST-04 · AST-08 · ระบุบัญชีอื่นไม่ได้: ส่ง agent_id (ลูก / นอกสาย / ตัวเอง / 0) = 422 · ไม่มีรูปเข้าของใคร
	before := countRows(t, &models.AssetConfig{}, "1 = 1")
	for name, c := range map[string]struct {
		tok string
		id  uint
	}{"share01 → ลูก agent01": {w.shareTok, w.agent.ID}, "csm01 → Share Master": {w.csmTok, w.sm1.ID},
		"share01 → นอกสาย csm01": {w.shareTok, w.csm.ID}, "agent01 → ชั้นบน share01": {w.agentTok, w.share.ID},
		"share01 → ตัวเอง": {w.shareTok, w.share.ID}, "agent_id = 0": {w.shareTok, 0}} {
		f := formOwn("agent_id", itoa(c.id))
		if r := up(c.tok, f); r.Code != 422 || !strings.Contains(r.Msg, "agent_id") {
			t.Errorf("%s: ต้อง 422 บอก agent_id ได้ %d (%s)", name, r.Code, r.Msg)
		}
	}
	if r := up(w.saTok, formGlobal("agent_id", "0")); r.Code != 422 {
		t.Errorf("GLOBAL + agent_id: ต้อง 422 ได้ %d", r.Code)
	}
	if n := countRows(t, &models.AssetConfig{}, "1 = 1"); n != before {
		t.Errorf("422 ต้องไม่สร้าง config: %d → %d", before, n)
	}
}
