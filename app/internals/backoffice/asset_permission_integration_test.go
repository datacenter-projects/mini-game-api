//go:build integration

package backoffice_test

import (
	"testing"

	"app/app/models"
	"app/pkg/testutil"
	"app/platform/database"
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

func TestAssetWritePermission(t *testing.T) { // AST-02 · AST-03 · AST-04
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
		if r := up(tok, formAgent(id)); r.Code != 200 {
			t.Errorf("%s ตั้งของตัวเอง: code %d (%s)", name, r.Code, r.Msg)
		}
	}
	// AST-02 · Share Master ตั้งเองไม่ได้ (ใช้ของ CSM)
	expect(t, up(w.sm1Tok, formAgent(w.sm1.ID)), 200, codeNoPermission)
	expect(t, assetCall(t, app, "theme/detail/update", scopeAgent(w.sm1.ID, "game_code", "scratch_card", "font_family", "kanit",
		"main_color_1", "#112233", "main_color_2", "#ffffff", "stats_bar_color", "main_color_1", "stats_bar_font_family", "prompt"), w.sm1Tok), 200, codeNoPermission)
	expect(t, assetCall(t, app, "branding/detail/update", byAgent(w.sm1.ID, "site_name", "X"), w.sm1Tok), 200, codeNoPermission)
	// AST-02 · Superadmin ตั้ง AGENT ของตัวเองไม่ได้ (ค่าของ Superadmin คือ GLOBAL)
	var sa models.UserAgent
	if err := database.DBConn.Where("username = ?", "superadmin").Take(&sa).Error; err != nil {
		t.Fatal(err)
	}
	expect(t, up(w.saTok, formAgent(sa.ID)), 200, codeNoPermission)

	// AST-04 · ชั้นบนแก้สายล่างไม่ได้ · นอกสาย = 402402
	expect(t, up(w.shareTok, formAgent(w.agent.ID)), 200, codeNoPermission)
	expect(t, up(w.comTok, formAgent(w.agent.ID)), 200, codeNoPermission)
	expect(t, up(w.csmTok, formAgent(w.sm1.ID)), 200, codeNoPermission) // CSM ก็แก้ Share Master ตรงๆ ไม่ได้ (สืบทอดเอง)
	expect(t, up(w.shareTok, formAgent(w.csm.ID)), 200, codeOutsideLine)
	expect(t, up(w.agentTok, formAgent(w.share.ID)), 200, codeOutsideLine) // ชั้นบนของตัวเองไม่ใช่ "สายล่าง" → นอกสาย
}
