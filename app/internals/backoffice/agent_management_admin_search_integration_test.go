//go:build integration

// API test ของเส้นค้นหาบัญชีของ ADMIN (MGMT-27B) ตาม docs/modules/agent_management.md หัวข้อ 7
package backoffice_test

import (
	"encoding/json"
	"testing"

	"app/app/models"
	"app/platform/database"
)

const adminSearchPath = "/api/v1/bo/pr/admin/accounts/search"

type adminRow struct {
	Username       string `json:"username"`
	Role           string `json:"role"`
	UserType       string `json:"user_type"`
	IsSubaccount   bool   `json:"is_subaccount"`
	Status         string `json:"status"`
	ParentUsername string `json:"parent_username"`
	CreatedAt      string `json:"created_at"`
	LastLoginAt    string `json:"last_login_at"`
	LastLoginIP    string `json:"last_login_ip"`
}

func TestAdminSearchAccounts(t *testing.T) { // MGMT-27B
	app := setup2(t)
	c := buildChain(t, app) // superadmin → comp01 → share01 → agent01
	admin := createAccount(t, "support01", models.AgentRoleAdmin, nil)
	adminTok := loginReady(t, app, models.AccountTypeAgent, admin.ID, "support01")
	sub := createSubAPI(t, app, c.comTok, "staff", map[string]string{"member": "view"})
	expect(t, call(t, app, "POST", createMemberPath, memberBody("mem01", 0.3), c.agentTok), 200, 200)

	search := func(username string) []adminRow {
		t.Helper()
		r := call(t, app, "POST", adminSearchPath, map[string]any{"username": username}, adminTok)
		expect(t, r, 200, 200)
		var rows []adminRow
		if err := json.Unmarshal(r.Data, &rows); err != nil {
			t.Fatalf("data %s", r.Data)
		}
		return rows
	}

	// ตรงทั้งคำ ไม่สนตัวพิมพ์ · ตัดช่องว่าง
	rows := search("  COMP01 ")
	if len(rows) != 1 || rows[0].Role != "COMPANY" || rows[0].UserType != "COMPANY_TRANSFER" || rows[0].IsSubaccount ||
		rows[0].ParentUsername != "superadmin" || rows[0].Status != "ACTIVE" || rows[0].CreatedAt == "" || rows[0].LastLoginAt == "" {
		t.Fatalf("comp01 %+v", rows)
	}
	if rows := search("comp"); len(rows) != 0 {
		t.Fatalf("บางส่วนต้องไม่พบ %+v", rows)
	}
	if rows := search("nobody"); rows == nil || len(rows) != 0 {
		t.Fatal("ไม่พบต้องได้ []")
	}

	// sub: role / user_type ของเจ้าของ · parent = เจ้าของ · ยังไม่เคย login = ""
	rows = search("comp01@staff")
	if len(rows) != 1 || !rows[0].IsSubaccount || rows[0].Role != "COMPANY" || rows[0].ParentUsername != "comp01" || rows[0].LastLoginAt != "" {
		t.Fatalf("sub %+v (id %d)", rows, sub.ID)
	}

	// Member · SUPERADMIN / ADMIN (ไม่มีผู้สร้าง)
	if rows := search("mem01"); len(rows) != 1 || rows[0].Role != "MEMBER" || rows[0].ParentUsername != "agent01" {
		t.Fatalf("member %+v", rows)
	}
	if rows := search("superadmin"); len(rows) != 1 || rows[0].Role != "SUPERADMIN" || rows[0].ParentUsername != "" {
		t.Fatalf("superadmin %+v", rows)
	}
	if rows := search("support01"); len(rows) != 1 || rows[0].Role != "ADMIN" {
		t.Fatalf("admin %+v", rows)
	}

	// status = สถานะที่ใช้งานจริง: หัวสายถูกระงับ ลูกทุกชั้นเป็น SUSPENDED
	if err := database.DBConn.Model(&models.UserAgent{}).Where("id = ?", c.share.ID).Update("status", models.AgentStatusSuspended).Error; err != nil {
		t.Fatal(err)
	}
	if rows := search("agent01"); rows[0].Status != "SUSPENDED" {
		t.Fatalf("agent01 status %s", rows[0].Status)
	}
	if rows := search("mem01"); rows[0].Status != "SUSPENDED" {
		t.Fatalf("mem01 status %s", rows[0].Status)
	}

	// ADMIN เท่านั้น · validation
	expect(t, call(t, app, "POST", adminSearchPath, map[string]any{"username": "comp01"}, c.saTok), 403, 401308)
	expect(t, call(t, app, "POST", adminSearchPath, map[string]any{"username": "comp01"}, c.comTok), 403, 401308)
	expect(t, call(t, app, "POST", adminSearchPath, map[string]any{"username": ""}, adminTok), 200, 422)
	expect(t, call(t, app, "POST", adminSearchPath, map[string]any{"username": nil}, adminTok), 200, 422)
}
