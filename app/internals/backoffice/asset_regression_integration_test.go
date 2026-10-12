//go:build integration

package backoffice_test

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"app/app/models"
	assetService "app/app/service/asset"
	"app/pkg/testutil"
	"app/platform/database"
)

// test ของจุดที่แก้จากการรีวิวโค้ด asset (2026-10-12)

func TestAssetThemeRestoresMissingGlobal(t *testing.T) { // AST-37 · lead GE10-1: แถว GLOBAL หาย ต้องบันทึกค่าตั้งต้นคืนได้
	app := setup2(t)
	testutil.NewFakeS3(t)
	w := buildAssetWorld(t, app)
	if err := database.DBConn.Exec(`DELETE FROM game_themes WHERE agent_id IS NULL AND game_code = 'coin_toss'`).Error; err != nil {
		t.Fatal(err)
	}

	// ค่าตั้งต้น = ค่าที่อ่านได้ตอนแถวหาย · ต้องไม่ถูกมองเป็น "ค่าเดิม" (no-op) — ต้องเขียนแถวกลับ
	body := putTheme(themeBody("coin_toss", "font_family", "noto", "main_color_1", "#f7447e", "main_color_2", "#ffffff",
		"stats_bar_color", "main_color_1", "stats_bar_font_family", "prompt"), scopeGlobal())
	expect(t, assetCall(t, app, "theme/detail/update", body, w.saTok), 200, 200)

	globalRow := func() models.GameTheme {
		t.Helper()
		var rows []models.GameTheme
		if err := database.DBConn.Where("agent_id IS NULL AND game_code = ?", "coin_toss").Find(&rows).Error; err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 {
			t.Fatalf("แถว GLOBAL ของ coin_toss ต้องมี 1 แถว ได้ %d", len(rows))
		}
		return rows[0]
	}
	before := globalRow()

	// มีแถวแล้ว: ส่งค่าเดิมทั้งชุดด้วยบัญชีอื่น = ไม่เขียน (updated_by / updated_at ไม่เปลี่ยน)
	expect(t, assetCall(t, app, "theme/detail/update", body, w.saEditTok), 200, 200)
	if after := globalRow(); after.UpdatedBy != before.UpdatedBy || !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Errorf("ค่าเดิมทั้งชุดต้องไม่เขียน: ก่อน %q %v · หลัง %q %v", before.UpdatedBy, before.UpdatedAt, after.UpdatedBy, after.UpdatedAt)
	}
}

func TestAssetTargetCheckedBeforeBody(t *testing.T) { // 7.1: สิทธิ์ต่อเป้าหมาย (ขั้น 4) มาก่อนตรวจ body (ขั้น 5)
	app := setup2(t)
	testutil.NewFakeS3(t)
	w := buildAssetWorld(t, app)
	aid := w.agent.ID // agent01 อยู่สายล่างของ share01 — share01 ดูได้ แก้ไม่ได้

	// ไม่มีสิทธิ์ + body ผิด = error สิทธิ์ ไม่ใช่ 422
	expect(t, assetCall(t, app, "theme/detail/update", scopeAgent(aid, "font_family", "noto"), w.shareTok), 200, codeNoPermission)
	expect(t, assetCall(t, app, "used/update", scopeAgent(aid, "game_code", "scratch_card"), w.shareTok), 200, codeNoPermission)
	expect(t, assetCall(t, app, "branding/detail/update", byAgent(aid, "site_theme", "x"), w.shareTok), 200, codeNoPermission)
	expect(t, postForm(t, app, "files/upload", formAgent(aid, "game_code", "scratch_card"), nil, w.shareTok), 200, codeNoPermission)
	expect(t, assetCall(t, app, "theme/detail/update", putTheme(themeBody("scratch_card"), scopeGlobal()), w.shareTok), 200, codeNoPermission)
	// นอกสาย + body ผิด (เส้นอ่าน) = 402402
	expect(t, assetCall(t, app, "used/list", scopeAgent(w.csm.ID), w.shareTok), 200, codeOutsideLine)

	// มีสิทธิ์ + body ผิด = 422 ตามปกติ
	expect(t, assetCall(t, app, "used/update", scopeAgent(w.share.ID, "game_code", "scratch_card"), w.shareTok), 200, 422)
	expect(t, assetCall(t, app, "used/list", scopeAgent(aid), w.shareTok), 200, 422)
	// ผิดที่ scope / agent_id / null (ขั้น 3) = 422 ก่อนตรวจสิทธิ์ แม้เป้าหมายจะอยู่นอกสาย
	expect(t, assetCall(t, app, "used/list", map[string]any{"agent_id": w.csm.ID, "game_code": "scratch_card"}, w.shareTok), 200, 422)
	expect(t, assetCall(t, app, "used/list", scopeAgent(w.csm.ID, "game_code", nil), w.shareTok), 200, 422)
}

func TestAssetUploadCreatedAtMatchesList(t *testing.T) { // created_at ใน response ของ upload = ค่าที่ DB บันทึก (เดียวกับ files/list)
	app := setup2(t)
	testutil.NewFakeS3(t)
	w := buildAssetWorld(t, app)
	sid := w.share.ID

	up := uploadScratch(t, app, w.shareTok, formAgent(sid), "icon_button")
	r := assetCall(t, app, "files/list", scopeAgent(sid, "game_code", "scratch_card", "asset_type", "icon_button"), w.shareTok)
	expect(t, r, 200, 200)
	page := decode[struct {
		Data []struct {
			ConfigID  uint   `json:"config_id"`
			CreatedAt string `json:"created_at"`
		} `json:"data"`
	}](t, r)
	for _, it := range page.Data {
		if it.ConfigID == up.ConfigID {
			if it.CreatedAt != up.CreatedAt || up.CreatedAt == "" {
				t.Fatalf("created_at ไม่ตรง: upload %q · files/list %q", up.CreatedAt, it.CreatedAt)
			}
			return
		}
	}
	t.Fatalf("ไม่พบ config %d ใน files/list", up.ConfigID)
}

func TestSeedGlobalDuplicateSlotFiles(t *testing.T) { // AST-61: หลายไฟล์ของช่องเดียวกัน = ล้มเหลวทุกไฟล์ของช่องนั้น ไม่เดาว่าไฟล์ไหนถูก
	setup2(t)
	fake := testutil.NewFakeS3(t)
	dir := t.TempDir()
	pngFile(t, filepath.Join(dir, "scratch_card", "icon_button.png"), 32, 32)
	pngFile(t, filepath.Join(dir, "scratch_card", "icon_button.webp"), 32, 32) // ชื่อช่องซ้ำ
	pngFile(t, filepath.Join(dir, "coin_toss", "background.png"), 100, 100)

	var out bytes.Buffer
	res, err := assetService.SeedGlobalService(context.Background(), dir, true, assetService.SeedCreatedBy("tester"), &out)
	if err != nil {
		t.Fatal(err)
	}
	if res.Uploaded != 1 || res.Failed != 2 {
		t.Fatalf("ผล: %+v\n%s", res, out.String())
	}
	if !strings.Contains(out.String(), "มีหลายไฟล์ของช่องเดียวกัน") || !strings.Contains(out.String(), "อัปแล้ว  coin_toss/background") {
		t.Errorf("รายงาน:\n%s", out.String())
	}
	if fake.Puts() != 1 {
		t.Errorf("ช่องที่ไฟล์ซ้ำต้องไม่อัปขึ้น storage เลย: puts = %d", fake.Puts())
	}
	if n := usedCount(t, nil, "scratch_card", "icon_button"); n != 0 {
		t.Errorf("icon_button ต้องไม่มี used ของ GLOBAL ได้ %d", n)
	}
}
