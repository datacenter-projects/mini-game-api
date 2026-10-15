//go:build integration

// MGMT-18A (lead pt_calculation Q3 · 2026-10-15): ห้ามตั้ง force ให้ Company Seamless Master · Company อื่นตั้งได้ตามเดิม
package backoffice_test

import (
	"testing"
)

const migCSMForce = 20261015150000 // MGMT-18A ห้ามตั้ง force ให้ CSM

func TestCSMForce(t *testing.T) {
	app := setup2(t)
	k := buildCSMChain(t, app) // superadmin → comp01 (Transfer) … · master01 (CSM 80) → sharem1 / sharem2

	r := call(t, app, "POST", createAgentPath, agentBody("COMPANY_SEAMLESS_MASTER", "csmforce01", nil, childPT(80, 10, 0, 0.5)), k.saTok)
	expect(t, r, 200, 422)
	expectMsgHas(t, r.Msg, "Company Seamless Master ถือสู้ไม่ได้")
	// step ผิดมาก่อน
	r = call(t, app, "POST", createAgentPath, agentBody("COMPANY_SEAMLESS_MASTER", "csmforce02", nil, childPT(80, 10.3, 0, 0.5)), k.saTok)
	expect(t, r, 200, 422)
	expectMsgHas(t, r.Msg, "ทีละ 0.5%")
	// Company ประเภทอื่นตั้ง force ได้ตามเดิม
	co, _ := mustCreate(t, app, k.saTok, agentBody("COMPANY_TRANSFER", "csmforceok", nil, childPT(80, 10, 0, 0.5)))
	for _, s := range settingsOf(t, co.ID) {
		if s.Force != 10 {
			t.Fatalf("transfer force %+v", s)
		}
	}

	// แก้ CSM: force ≠ 0 = 422 · ไม่บันทึก · Share Master ไม่ถูก sync
	r = call(t, app, "POST", updatePTPath, ptBody(k.csm.ID, 85, 5, 0, 0.5), k.saTok)
	expect(t, r, 200, 422)
	expectMsgHas(t, r.Msg, "Company Seamless Master ถือสู้ไม่ได้")
	for _, id := range []uint{k.csm.ID, k.sm1.ID} {
		for _, s := range settingsOf(t, id) {
			if s.Force != 0 || s.PTFromParent != 80 {
				t.Fatalf("agent %d %+v", id, s)
			}
		}
	}
	expect(t, call(t, app, "POST", updatePTPath, ptBody(k.csm.ID, 85, 0, 0, 0.5), k.saTok), 200, 200)
}

func TestMigrationCSMForce(t *testing.T) { // CSM เดิม force ≠ 0 → 0 · Company อื่นไม่แตะ · ค่าอื่นไม่แตะ
	app := setup2(t)
	k := buildCSMChain(t, app)

	migrateDownTo(t, migCSMForce-1)
	execSQL(t, "UPDATE agent_game_settings SET force = 20 WHERE agent_id IN (?, ?)", k.csm.ID, k.com.ID)
	gooseUpTo(t, migCSMForce)

	for _, s := range settingsOf(t, k.csm.ID) {
		if s.Force != 0 || s.PTFromParent != 80 || s.Commission != 0.5 {
			t.Fatalf("csm %+v", s)
		}
	}
	for _, s := range settingsOf(t, k.com.ID) { // Company Transfer ไม่แตะ
		if s.Force != 20 {
			t.Fatalf("company %+v", s)
		}
	}
}
