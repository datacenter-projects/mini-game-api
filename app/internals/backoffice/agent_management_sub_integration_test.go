//go:build integration

// API test ของ agent_management phase 5 (sub) ตาม docs/modules/agent_management.md หัวข้อ 7 (MGMT-40 – MGMT-46)
package backoffice_test

import (
	"encoding/json"
	"testing"

	"app/app/models"
	"app/platform/database"

	"github.com/gofiber/fiber/v2"
)

const (
	subListPath   = "/api/v1/bo/pr/manage/subaccounts/list"
	subDetailPath = "/api/v1/bo/pr/manage/subaccounts/detail/get"
	subCreatePath = "/api/v1/bo/pr/manage/subaccounts/create"
	subUpdatePath = "/api/v1/bo/pr/manage/subaccounts/detail/update"
	subStatusPath = "/api/v1/bo/pr/manage/subaccounts/status/update"
)

type subData struct {
	ID          uint              `json:"id"`
	Username    string            `json:"username"`
	Name        string            `json:"name"`
	Phone       string            `json:"phone"`
	Status      string            `json:"status"`
	Permissions map[string]string `json:"permissions"`
	CreatedAt   string            `json:"created_at"`
	LastLoginAt string            `json:"last_login_at"`
}

func subBody(suffix string, perms map[string]string) map[string]any {
	b := map[string]any{"name_suffix": suffix, "password": mgPassword, "name": "staff01", "phone": ""}
	if perms != nil {
		b["permissions"] = perms
	}
	return b
}

func createSubAPI(t *testing.T, app *fiber.App, tok, suffix string, perms map[string]string) created {
	t.Helper()
	r := call(t, app, "POST", subCreatePath, subBody(suffix, perms), tok)
	expect(t, r, 200, 200)
	var d created
	_ = json.Unmarshal(r.Data, &d)
	return d
}

func TestSubCreate(t *testing.T) { // MGMT-40, MGMT-41, MGMT-50, MGMT-52
	app := setup2(t)
	c := buildChain(t, app)

	r := call(t, app, "POST", subCreatePath, subBody("Staff", map[string]string{"member": "edit", "report": "view"}), c.comTok)
	expect(t, r, 200, 200)
	var d created
	_ = json.Unmarshal(r.Data, &d)
	if d.Username != "comp01@staff" {
		t.Fatalf("username %q", d.Username)
	}
	var s models.Subaccount
	database.DBConn.Where("id = ?", d.ID).Take(&s)
	if s.AgentID != c.com.ID || s.Status != models.AgentStatusActive || s.Name == nil || *s.Name != "staff01" {
		t.Fatalf("sub %+v", s)
	}
	var perms map[string]string
	_ = json.Unmarshal([]byte(s.Permissions), &perms)
	if _, hasPT := perms["pt"]; len(perms) != 8 || hasPT || perms["api_credential"] != "off" || perms["member"] != "edit" || perms["report"] != "view" {
		t.Fatalf("permissions %v", perms)
	}

	// login ด้วยตัวพิมพ์ใหญ่ได้ (AUTH-01)
	setPasscode(t, models.AccountTypeSub, d.ID, passcode)
	if tok := loginToken2(t, app, "Comp01@STAFF", mgPassword); tok == "" {
		t.Fatal("sub login ไม่สำเร็จ")
	}

	tests := []struct {
		name string
		tok  string
		body map[string]any
		code int
	}{
		{"ชื่อซ้ำ", c.comTok, subBody("staff", nil), 402401},
		{"ส่วนหลัง @ สั้น", c.comTok, subBody("ab", nil), 422},
		{"ส่วนหลัง @ มี _", c.comTok, subBody("st_aff", nil), 422},
		{"ระดับ VIEW ตัวใหญ่", c.comTok, subBody("staff2", map[string]string{"member": "VIEW"}), 422},
		{"dashboard edit", c.comTok, subBody("staff2", map[string]string{"dashboard": "edit"}), 422},
		{"Company ให้ rate", c.comTok, subBody("staff2", map[string]string{"rate": "view"}), 422},
		{"Superadmin ให้ announcement", c.saTok, subBody("staff2", map[string]string{"announcement": "view"}), 422},
		{"รหัสผ่านผิดกฎ", c.comTok, map[string]any{"name_suffix": "staff2", "password": "aaaa1111", "name": "staff01", "phone": ""}, 422},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expect(t, call(t, app, "POST", subCreatePath, tt.body, tt.tok), 200, tt.code)
		})
	}

	// Superadmin ให้ rate ได้ · ชื่อเล่นภาษาไทย · เบอร์ซ้ำได้ (MGMT-41)
	b := subBody("ratestaff", map[string]string{"rate": "edit"})
	b["name"], b["phone"] = "พนักงาน1", "0899999999"
	expect(t, call(t, app, "POST", subCreatePath, b, c.saTok), 200, 200)
	b = subBody("staff2", nil)
	b["phone"] = "0899999999"
	expect(t, call(t, app, "POST", subCreatePath, b, c.comTok), 200, 200)

	// sub สร้าง / ดูรายชื่อ sub ไม่ได้ (MGMT-40)
	subTok := readyToken(t, app, models.AccountTypeSub, d.ID, "comp01@staff")
	expect(t, call(t, app, "POST", subCreatePath, subBody("other", nil), subTok), 200, 402311)
	expect(t, call(t, app, "POST", subListPath, map[string]any{}, subTok), 200, 402311)
}

func TestSubListAndDetail(t *testing.T) { // MGMT-45, MGMT-46
	app := setup2(t)
	c := buildChain(t, app)
	createSubAPI(t, app, c.shareTok, "bstaff", map[string]string{"report": "view"})
	a := createSubAPI(t, app, c.shareTok, "astaff", nil)
	createSubAPI(t, app, c.comTok, "comstaff", nil)

	list := func(tok string, body map[string]any) ([]subData, int64) {
		t.Helper()
		r := call(t, app, "POST", subListPath, body, tok)
		expect(t, r, 200, 200)
		var p struct {
			Data  []subData `json:"data"`
			Total int64     `json:"total_count"`
		}
		_ = json.Unmarshal(r.Data, &p)
		return p.Data, p.Total
	}

	rows, total := list(c.shareTok, map[string]any{})
	if total != 2 || rows[0].Username != "share01@astaff" || rows[1].Username != "share01@bstaff" || rows[1].Permissions["report"] != "view" ||
		len(rows[0].Permissions) != 8 || rows[0].Status != "ACTIVE" || rows[0].CreatedAt == "" {
		t.Fatalf("share subs %+v", rows)
	}
	// ชั้นบนดู sub ของสายล่างได้ · q บางส่วน · page / limit ใน body
	rows, total = list(c.comTok, map[string]any{"owner_id": c.share.ID, "q": "BST"})
	if total != 1 || rows[0].Username != "share01@bstaff" {
		t.Fatalf("q %+v", rows)
	}
	rows, total = list(c.comTok, map[string]any{"owner_id": c.share.ID, "limit": 1, "page": 2})
	if total != 2 || len(rows) != 1 || rows[0].Username != "share01@bstaff" {
		t.Fatalf("page 2 %+v", rows)
	}
	expect(t, call(t, app, "POST", subListPath, map[string]any{"owner_id": c.com.ID}, c.shareTok), 200, 402402) // สายบน

	// รายละเอียด: เจ้าของ และชั้นบน ดูได้ · สายอื่น = 402404
	for _, tok := range []string{c.shareTok, c.comTok, c.saTok} {
		expect(t, call(t, app, "POST", subDetailPath, map[string]any{"id": a.ID}, tok), 200, 200)
	}
	expect(t, call(t, app, "POST", subDetailPath, map[string]any{"id": a.ID}, c.agentTok), 200, 402404)
	expect(t, call(t, app, "POST", subDetailPath, map[string]any{"id": 99999}, c.shareTok), 200, 402404)

	// เจ้าของถูกระงับ → sub แสดงสถานะของเจ้าของ (MGMT-43)
	setStatus(t, c.share.ID, models.AgentStatusSuspended)
	rows, _ = list(c.comTok, map[string]any{"owner_id": c.share.ID})
	if rows[0].Status != "SUSPENDED" {
		t.Fatalf("status %s", rows[0].Status)
	}
}

func TestSubUpdate(t *testing.T) { // MGMT-42, MGMT-43, MGMT-44, MGMT-45, MGMT-60
	app := setup2(t)
	c := buildChain(t, app)
	s := createSubAPI(t, app, c.shareTok, "staff", map[string]string{"member": "edit"})

	upd := map[string]any{"id": s.ID, "name": "ใจดี", "phone": "0811111111", "permissions": map[string]string{"report": "view"}}
	expect(t, call(t, app, "POST", subUpdatePath, upd, c.shareTok), 200, 200)
	r := call(t, app, "POST", subDetailPath, map[string]any{"id": s.ID}, c.shareTok)
	var d subData
	_ = json.Unmarshal(r.Data, &d)
	if d.Name != "ใจดี" || d.Phone != "0811111111" || d.Permissions["report"] != "view" || d.Permissions["member"] != "off" {
		t.Fatalf("after update %+v", d)
	}
	expect(t, call(t, app, "POST", subUpdatePath, upd, c.comTok), 200, 402404) // ชั้นบนแก้ไม่ได้
	expect(t, call(t, app, "POST", subUpdatePath, map[string]any{"id": s.ID, "name": "ใจดี", "phone": ""}, c.shareTok), 200, 422)
	expect(t, call(t, app, "POST", subUpdatePath, map[string]any{"id": s.ID, "name": "ใจดี", "permissions": map[string]string{}}, c.shareTok), 200, 422)

	// INACTIVE เก็บเป็น SUSPENDED · แสดง INACTIVE · ตั้งกลับ ACTIVE ได้
	expect(t, call(t, app, "POST", subStatusPath, map[string]any{"id": s.ID, "status": "INACTIVE"}, c.shareTok), 200, 200)
	var row models.Subaccount
	database.DBConn.Where("id = ?", s.ID).Take(&row)
	if row.Status != models.AgentStatusSuspended {
		t.Fatalf("stored %s", row.Status)
	}
	r = call(t, app, "POST", subDetailPath, map[string]any{"id": s.ID}, c.shareTok)
	_ = json.Unmarshal(r.Data, &d)
	if d.Status != "INACTIVE" {
		t.Fatalf("status %s", d.Status)
	}
	setPasscode(t, models.AccountTypeSub, s.ID, passcode)
	subTok := loginToken2(t, app, "share01@staff", mgPassword)
	if subTok == "" {
		t.Fatal("INACTIVE ต้อง login ได้")
	}
	// AUTH-54: เปิดได้แค่ Profile / Report · route อื่น 401311 (เช็คก่อนสิทธิ์ของ sub)
	expect(t, call(t, app, "GET", profilePath, nil, subTok), 200, 200)
	expect(t, call(t, app, "POST", downlinesPath, map[string]any{}, subTok), 403, 401311)
	expect(t, call(t, app, "POST", subStatusPath, map[string]any{"id": s.ID, "status": "ACTIVE"}, c.shareTok), 200, 200)
	expect(t, call(t, app, "POST", downlinesPath, map[string]any{}, subTok), 200, 402303) // กลับมาใช้สิทธิ์ตามปกติ (ไม่มีสิทธิ์ member)
	expect(t, call(t, app, "POST", subStatusPath, map[string]any{"id": s.ID, "status": "INACTIVE"}, c.comTok), 200, 402404)
	expect(t, call(t, app, "POST", subStatusPath, map[string]any{"id": s.ID, "status": "SUSPENDED"}, c.shareTok), 200, 422)

	var logs []models.AccountChangeLog
	database.DBConn.Where("target_type = ? AND target_id = ?", "SUB", s.ID).Order("id").Find(&logs)
	if len(logs) != 4 || logs[0].Action != models.ChangeCreate || logs[1].Action != models.ChangeUpdateInfo || logs[2].Action != models.ChangeSubStatus {
		t.Fatalf("logs %+v", logs)
	}
}
