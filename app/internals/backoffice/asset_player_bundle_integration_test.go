//go:build integration

package backoffice_test

import (
	"encoding/json"
	"testing"

	"app/app/models"
	"app/pkg/testutil"
	"app/platform/database"
)

const plAssetBundlePath = "/api/v1/player/pr/asset/bundle"

// TestPlayerAssetBundle — GN23 · AST-48 (ตัวสร้างเดียวกับ BO preview) · AST-52 (เว็บจาก session · ไม่รับ agent_id) · lead 2026-10-16
func TestPlayerAssetBundle(t *testing.T) {
	e := plSetup(t) // Superadmin → comp01 → share01 → agent01 → mem01
	testutil.NewFakeS3(t)
	c := e.c

	// ค่าหลายชั้น: GLOBAL background · share01 icon_button + ธีม + ชื่อเว็บ · agent01 winner
	uploadScratch(t, e.app, c.saTok, formGlobal(), "background")
	uploadScratch(t, e.app, c.shareTok, formOwn(), "icon_button")
	uploadScratch(t, e.app, c.agentTok, formOwn(), "winner")
	expect(t, assetCall(t, e.app, "theme/detail/update", putTheme(themeBody("scratch_card", "font_family", "mitr"), scopeOwn()), c.shareTok), 200, 200)
	expect(t, assetCall(t, e.app, "branding/detail/update", own("site_name", "Share One"), c.shareTok), 200, 200)

	tok := plToken(t, e.app, "mem01", mgPassword)
	bundle := func(body any, tok string) testutil.Response {
		return call(t, e.app, "POST", plAssetBundlePath, body, tok)
	}

	// AST-48: ผู้เล่นได้เท่ากับ preview ของผู้สร้างตรง (agent01) ทุกส่วน
	r := bundle(map[string]any{"game_code": "scratch_card"}, tok)
	expect(t, r, 200, 200)
	bo := assetCall(t, e.app, "bundle/detail/get", own("game_code", "scratch_card"), c.agentTok)
	expect(t, bo, 200, 200)
	if string(r.Data) != string(bo.Data) {
		t.Fatalf("bundle ผู้เล่นต้องเท่ากับ preview ของ agent01:\nplayer %s\nbo     %s", r.Data, bo.Data)
	}
	b := decode[bundleData](t, r)
	if len(b.Assets) != 3 || b.Source != "mixed" || b.Site.SiteName != "Share One" {
		t.Fatalf("bundle ผู้เล่น: %+v", b)
	}
	var th map[string]any
	_ = json.Unmarshal(b.Theme, &th)
	if th["font_family"] != "mitr" || th["source"] != "upline" {
		t.Errorf("theme ต้องไล่สายจาก share01: %v", th)
	}
	if _, has := th["inherited_from"]; has { // lead Q6: ไม่เปิด username ชั้นบนให้ผู้เล่น
		t.Error("theme ฝั่งผู้เล่นต้องไม่มี inherited_from")
	}

	// AST-52: ส่ง agent_id ของเว็บอื่นมา = ถูกเมิน ได้ของเว็บตัวเองเสมอ
	for _, id := range []uint{c.share.ID, c.com.ID, 999999} {
		r2 := bundle(map[string]any{"game_code": "scratch_card", "agent_id": id}, tok)
		expect(t, r2, 200, 200)
		if string(r2.Data) != string(r.Data) {
			t.Errorf("ส่ง agent_id=%d ต้องได้ผลเดิม: %s", id, r2.Data)
		}
	}

	// ผู้เล่นของเว็บอื่น (mem02 ใต้ share01) ได้ของ share01 — ไม่เห็น winner ของ agent01
	createMember(t, e.app, c.shareTok, "mem02", nil)
	tok2 := plToken(t, e.app, "mem02", mgPassword)
	b2 := decode[bundleData](t, bundle(map[string]any{"game_code": "scratch_card"}, tok2))
	if len(b2.Assets) != 2 || b2.Assets["winner"] != "" || b2.Site.SiteName != "Share One" {
		t.Errorf("mem02 ต้องได้ของ share01 เท่านั้น: %+v", b2)
	}

	// เกมที่ไม่มีรูปเลย = assets {} ไม่ใช่ error · ธีมค่ากลาง
	if b3 := decode[bundleData](t, bundle(map[string]any{"game_code": "rock_paper_scissors"}, tok)); len(b3.Assets) != 0 || b3.Source != "" {
		t.Errorf("เกมที่ไม่มีรูป: %+v", b3)
	}

	// lead Q5: เกมที่ปิดอยู่ตอบปกติ (Bet เช็คเกมปิดเอง)
	if err := database.DBConn.Model(&models.AgentGameSetting{}).Where("agent_id = ? AND game_code = ?", c.share.ID, "scratch_card").
		Update("status_game", false).Error; err != nil {
		t.Fatal(err)
	}
	r4 := bundle(map[string]any{"game_code": "scratch_card"}, tok)
	expect(t, r4, 200, 200)
	if string(r4.Data) != string(r.Data) {
		t.Errorf("เกมปิดต้องได้ bundle ตามปกติ: %s", r4.Data)
	}

	// SUSPENDED ผ่าน (ดูข้อมูลได้)
	plSetMemberStatus(t, e.member.ID, models.AgentStatusSuspended)
	expect(t, bundle(map[string]any{"game_code": "scratch_card"}, tok), 200, 200)
	plSetMemberStatus(t, e.member.ID, models.AgentStatusActive)

	// validation · เกมผิด
	expect(t, bundle(map[string]any{}, tok), 200, 422)
	expect(t, bundle(map[string]any{"game_code": nil}, tok), 200, 422)
	expect(t, bundle(map[string]any{"game_code": "plinko"}, tok), 200, codeSlotNotFound)

	// token: ไม่มี / token ของ BO = 407202 (ผู้เล่นใช้ token ของตัวเองเท่านั้น)
	expect(t, bundle(map[string]any{"game_code": "scratch_card"}, ""), 401, 407202)
	expect(t, bundle(map[string]any{"game_code": "scratch_card"}, c.agentTok), 401, 407202)
}
