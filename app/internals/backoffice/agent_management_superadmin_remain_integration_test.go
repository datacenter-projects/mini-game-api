//go:build integration

// MGMT-18A (2026-10-14): Superadmin ไม่ตั้ง remain_quota ให้ลูก (ส่วนที่เหลือคืน Superadmin อยู่แล้ว) · force ตั้งได้ตามเดิม
package backoffice_test

import (
	"testing"

	"app/app/models"
	"app/platform/database"
)

const migSuperadminRemain = 20261014120000 // MGMT-18A

func TestSuperadminRemain(t *testing.T) {
	app := setup2(t)
	c := buildChain(t, app) // superadmin → comp01 (90) → share01 (70) → agent01 (60)

	for username, ut := range map[string]string{"remcomp01": "COMPANY_TRANSFER", "remcsm01": "COMPANY_SEAMLESS_MASTER"} {
		r := call(t, app, "POST", createAgentPath, agentBody(ut, username, nil, childPT(80, 0, 5, 0.5)), c.saTok)
		expect(t, r, 200, 422)
		expectMsgHas(t, r.Msg, "Superadmin ตั้งไม่ได้")
	}
	// remain 0 + force ตั้งได้ตามเดิม
	co, _ := mustCreate(t, app, c.saTok, agentBody("COMPANY_TRANSFER", "remok01", nil, childPT(80, 10, 0, 0.5)))
	for _, s := range settingsOf(t, co.ID) {
		if s.Force != 10 || s.Remain != 0 {
			t.Fatalf("settings %+v", s)
		}
	}
	// แก้ Company: remain ไม่ใช่ 0 = 422 · ไม่บันทึก
	r := call(t, app, "POST", updatePTPath, ptBody(c.com.ID, 90, 0, 10, 0.5), c.saTok)
	expect(t, r, 200, 422)
	expectMsgHas(t, r.Msg, "Superadmin ตั้งไม่ได้")
	expect(t, call(t, app, "POST", updatePTPath, ptBody(c.com.ID, 90, 5, 0, 0.5), c.saTok), 200, 200)
	// ชั้นอื่นตั้งได้ตามเดิม
	expect(t, call(t, app, "POST", updatePTPath, ptBody(c.share.ID, 70, 0, 10, 0.5), c.comTok), 200, 200)
	for _, s := range settingsOf(t, c.share.ID) {
		if s.Remain != 10 {
			t.Fatalf("share remain %+v", s)
		}
	}
}

func TestMigrationSuperadminRemain(t *testing.T) { // MGMT-18A: แถวที่ผู้สร้างเป็น Superadmin → remain 0 · ค่าอื่นไม่แตะ
	app := setup2(t)
	c := buildChain(t, app)
	expect(t, call(t, app, "POST", updatePTPath, ptBody(c.share.ID, 70, 5, 10, 0.5), c.comTok), 200, 200)

	migrateDownTo(t, migSuperadminRemain-1)
	execSQL(t, "UPDATE agent_game_settings SET remain = 20, force = 5 WHERE agent_id = ?", c.com.ID)
	gooseUpTo(t, migSuperadminRemain)

	for _, s := range settingsOf(t, c.com.ID) {
		if s.Remain != 0 || s.Force != 5 || s.PTFromParent != 90 {
			t.Fatalf("company %+v", s)
		}
	}
	for _, s := range settingsOf(t, c.share.ID) { // ผู้สร้างไม่ใช่ Superadmin — ไม่แตะ
		if s.Remain != 10 || s.Force != 5 {
			t.Fatalf("share %+v", s)
		}
	}
	var sa models.AgentGameSetting
	database.DBConn.Where("parent_id IS NULL").Take(&sa)
	if sa.PTFromParent != 100 {
		t.Fatalf("superadmin %+v", sa)
	}
}
