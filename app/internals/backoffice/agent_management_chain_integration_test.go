//go:build integration

// สายชั้นบน user_agents.cnf (MGMT-61) และ agent_game_settings.parent_id (MGMT-62) — เพิ่ม 2026-10-09
package backoffice_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	agentManagementCore "app/app/core/agent_management"
	"app/app/models"
	"app/platform/database"
)

func chainOf(t *testing.T, id uint) (models.UserAgent, []agentManagementCore.ChainNode) {
	t.Helper()
	var a models.UserAgent
	if err := database.DBConn.Select("id", "parent_id", "cnf").Where("id = ?", id).Take(&a).Error; err != nil {
		t.Fatal(err)
	}
	var c agentManagementCore.Chain
	if err := json.Unmarshal([]byte(a.Cnf), &c); err != nil {
		t.Fatalf("cnf %q: %v", a.Cnf, err)
	}
	return a, c.Parent
}

func TestCreateStoresChain(t *testing.T) { // MGMT-61, MGMT-62
	app := setup2(t)
	c := buildChain(t, app) // superadmin → comp01 → share01 → agent01

	com, _ := chainOf(t, c.com.ID)
	saID := *com.ParentID
	_, saChain := chainOf(t, saID)
	if len(saChain) != 0 {
		t.Fatalf("superadmin cnf.parent = %v, want []", saChain)
	}
	_, got := chainOf(t, c.agent.ID)
	want := []agentManagementCore.ChainNode{{ID: saID, Position: "superadmin"}, {ID: c.com.ID, Position: "company"}, {ID: c.share.ID, Position: "shareholder"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("agent01 cnf.parent = %v, want %v", got, want)
	}

	// agent ซ้อน agent
	agent2, _ := mustCreate(t, app, c.agentTok, agentBody("AGENT", "agent02", nil, childPT(50, 0, 0, 0)))
	_, got = chainOf(t, agent2.ID)
	if len(got) != 4 || got[3] != (agentManagementCore.ChainNode{ID: c.agent.ID, Position: "agent"}) {
		t.Fatalf("agent02 cnf.parent = %v", got)
	}

	// ทุกแถว agent_game_settings ของบัญชีใหม่มี parent_id = ผู้สร้าง
	var settings []models.AgentGameSetting
	database.DBConn.Select("agent_id", "game_code", "parent_id").Where("agent_id = ?", agent2.ID).Find(&settings)
	if len(settings) == 0 {
		t.Fatal("ไม่มีแถว agent_game_settings")
	}
	for _, s := range settings {
		if s.ParentID == nil || *s.ParentID != c.agent.ID {
			t.Fatalf("%s parent_id = %v, want %d", s.GameCode, s.ParentID, c.agent.ID)
		}
	}

	// ค้นทั้งสายลงจาก company ด้วย cnf @> (MGMT-61)
	var ids []uint
	database.DBConn.Model(&models.UserAgent{}).Where("cnf @> ?::jsonb", fmt.Sprintf(`{"parent":[{"id":%d}]}`, c.com.ID)).Order("id").Pluck("id", &ids)
	if !reflect.DeepEqual(ids, []uint{c.share.ID, c.agent.ID, agent2.ID}) {
		t.Fatalf("downline ของ comp01 = %v", ids)
	}
}
