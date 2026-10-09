//go:build integration

// เงินและ % เป็นทศนิยม float ปัด 4 ตำแหน่ง (CLAUDE.md กฎข้อ 9 — แก้ 2026-10-09)
package backoffice_test

import (
	"testing"

	"app/app/models"
	"app/platform/database"
)

func TestBalanceFourDecimals(t *testing.T) {
	app := setup2(t)
	c := buildChain(t, app)
	setBalance(t, c.com.ID, "THB", 50000)

	b := agentBody("SHARE_B2C", "sharefloat", []string{"THB"}, childPT(50.5, 0, 0, 0.3))
	b["balance"] = map[string]any{"THB": 10000.1234}
	d, _ := mustCreate(t, app, c.comTok, b)

	if got := agentBalance(t, d.ID, "THB"); got != 10000.1234 {
		t.Fatalf("share THB = %v, want 10000.1234", got)
	}
	if got := agentBalance(t, c.com.ID, "THB"); got != 39999.8766 { // ปัด 4 ตำแหน่งหลังลบ
		t.Fatalf("company THB = %v, want 39999.8766", got)
	}
	var ledger []models.BalanceLedger
	database.DBConn.Where("request_id = ?", b["request_id"]).Order("id").Find(&ledger)
	if len(ledger) != 2 || ledger[0].Amount != -10000.1234 || ledger[0].BalanceAfter != 39999.8766 || ledger[1].Amount != 10000.1234 {
		t.Fatalf("ledger = %+v", ledger)
	}
	s := gameSetting(t, d.ID)
	if s.PTFromParent != 50.5 || s.PT != 50.5 || s.Commission != 0.3 {
		t.Fatalf("settings = %+v", s)
	}

	// ทศนิยมเกิน 4 ตำแหน่ง = 422
	b = agentBody("SHARE_B2C", "sharefloat2", []string{"THB"}, childPT(50, 0, 0, 0))
	b["balance"] = map[string]any{"THB": 1.12345}
	expect(t, call(t, app, "POST", createAgentPath, b, c.comTok), 200, 422)
}
