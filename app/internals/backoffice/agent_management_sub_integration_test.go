//go:build integration

// API test ของ agent_management phase 5 (sub) ตาม docs/modules/agent_management.md หัวข้อ 7 (MGMT-40 – MGMT-46)
package backoffice_test

import (
	"encoding/json"
	"strings"
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
	PhoneCode   string            `json:"phone_country_code"`
	Phone       string            `json:"phone"`
	Status      string            `json:"status"`
	Permissions map[string]string `json:"permissions"`
	CreatedAt   string            `json:"created_at"`
	LastLoginAt string            `json:"last_login_at"`
}

func subBody(suffix string, perms map[string]string) map[string]any {
	b := map[string]any{"name_suffix": suffix, "password": mgPassword, "name": "staff01", "phone_country_code": "", "phone": ""}
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
		{"รหัสผ่านผิดกฎ", c.comTok, map[string]any{"name_suffix": "staff2", "password": "aaaa1111", "name": "staff01", "phone_country_code": "", "phone": ""}, 422},
		{"เบอร์ไม่มีรหัสประเทศ", c.comTok, map[string]any{"name_suffix": "staff2", "password": mgPassword, "name": "staff01", "phone_country_code": "", "phone": "899999999"}, 422},
		{"รหัสประเทศไม่มีจริง", c.comTok, map[string]any{"name_suffix": "staff2", "password": mgPassword, "name": "staff01", "phone_country_code": "999", "phone": "899999999"}, 422},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expect(t, call(t, app, "POST", subCreatePath, tt.body, tt.tok), 200, tt.code)
		})
	}

	// Superadmin ให้ rate ได้ · ชื่อเล่นภาษาไทย · เบอร์ซ้ำได้ (MGMT-41)
	b := subBody("ratestaff", map[string]string{"rate": "edit"})
	b["name"], b["phone_country_code"], b["phone"] = "พนักงาน1", "66", "899999999"
	expect(t, call(t, app, "POST", subCreatePath, b, c.saTok), 200, 200)
	b = subBody("staff2", nil)
	b["phone_country_code"], b["phone"] = "66", "899999999"
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
	// ชั้นบนดู sub ของสายล่างได้ · keyword บางส่วน ไม่สนตัวพิมพ์ 4–32 ตัว · page / limit ใน body
	rows, total = list(c.comTok, map[string]any{"owner_id": c.share.ID, "keyword": " BSTA "})
	if total != 1 || rows[0].Username != "share01@bstaff" {
		t.Fatalf("keyword %+v", rows)
	}
	for _, k := range []string{"bst", strings.Repeat("a", 33)} {
		expect(t, call(t, app, "POST", subListPath, map[string]any{"keyword": k}, c.shareTok), 200, 422)
	}
	if _, total = list(c.shareTok, map[string]any{"q": "bstaff"}); total != 2 {
		t.Fatalf("q เลิกใช้แล้ว ต้องไม่กรอง ได้ %d", total)
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

func TestSubUpdate(t *testing.T) { // MGMT-42, MGMT-44, MGMT-45, MGMT-60 · lead SN4 / SQ5 / SQ6
	app := setup2(t)
	c := buildChain(t, app)
	s := createSubAPI(t, app, c.shareTok, "staff", map[string]string{"member": "edit"})
	detail := func() subData {
		t.Helper()
		r := call(t, app, "POST", subDetailPath, map[string]any{"id": s.ID}, c.shareTok)
		expect(t, r, 200, 200)
		var d subData
		_ = json.Unmarshal(r.Data, &d)
		return d
	}
	logsOf := func(action models.AccountChangeAction) int64 {
		return countRows(t, &models.AccountChangeLog{}, "target_type = ? AND target_id = ? AND action = ?", "SUB", s.ID, action)
	}

	// รูปแบบ: ไม่ส่ง section = 422 · info {} = 422 · permissions null = 422 · เมนูที่ไม่มี = 422
	r := call(t, app, "POST", subUpdatePath, map[string]any{"id": s.ID}, c.shareTok)
	expect(t, r, 200, 422)
	expectMsgHas(t, r.Msg, "ต้องส่งอย่างน้อย 1 section")
	expect(t, call(t, app, "POST", subUpdatePath, map[string]any{"id": s.ID, "info": map[string]any{}}, c.shareTok), 200, 422)
	expect(t, call(t, app, "POST", subUpdatePath, map[string]any{"id": s.ID, "permissions": nil}, c.shareTok), 200, 422)
	expect(t, call(t, app, "POST", subUpdatePath, map[string]any{"id": s.ID, "permissions": map[string]string{"pt": "edit"}}, c.shareTok), 200, 422)

	// section info อย่างเดียว · ส่งบาง field ได้ · สิทธิ์คงเดิม
	expect(t, call(t, app, "POST", subUpdatePath, map[string]any{"id": s.ID, "info": map[string]any{"name": "ใจดี"}}, c.shareTok), 200, 200)
	if d := detail(); d.Name != "ใจดี" || d.Permissions["member"] != "edit" {
		t.Fatalf("info only %+v", d)
	}
	// section permissions อย่างเดียว = แทนทั้งชุด (เมนูที่ไม่ส่ง = off) · ชื่อคงเดิม
	expect(t, call(t, app, "POST", subUpdatePath, map[string]any{"id": s.ID, "permissions": map[string]string{"report": "view"}}, c.shareTok), 200, 200)
	if d := detail(); d.Name != "ใจดี" || d.Permissions["report"] != "view" || d.Permissions["member"] != "off" {
		t.Fatalf("permissions only %+v", d)
	}
	if logsOf(models.ChangeUpdateInfo) != 1 || logsOf(models.ChangeSubPermissions) != 1 {
		t.Fatalf("log info %d permissions %d", logsOf(models.ChangeUpdateInfo), logsOf(models.ChangeSubPermissions))
	}
	// 2 section พร้อมกัน = log 2 แถว
	expect(t, call(t, app, "POST", subUpdatePath, map[string]any{"id": s.ID,
		"info":        map[string]any{"phone_country_code": "66", "phone": "811111111"},
		"permissions": map[string]string{"member": "view", "report": "view"}}, c.shareTok), 200, 200)
	if d := detail(); d.PhoneCode != "66" || d.Phone != "811111111" || d.Permissions["member"] != "view" {
		t.Fatalf("both %+v", d)
	}
	if logsOf(models.ChangeUpdateInfo) != 2 || logsOf(models.ChangeSubPermissions) != 2 {
		t.Fatal("2 section ต้องได้ log แยก 2 แถว")
	}
	// ค่าเดิมทั้งหมด = 200 ไม่เขียน log (สิทธิ์เทียบแบบครบทุกเมนู — ส่ง off ตรงๆ ก็นับว่าเดิม)
	expect(t, call(t, app, "POST", subUpdatePath, map[string]any{"id": s.ID,
		"info":        map[string]any{"name": "ใจดี", "phone_country_code": "66", "phone": "811111111"},
		"permissions": map[string]string{"member": "view", "report": "view", "payment": "off"}}, c.shareTok), 200, 200)
	if logsOf(models.ChangeUpdateInfo) != 2 || logsOf(models.ChangeSubPermissions) != 2 {
		t.Fatal("ค่าเดิมต้องไม่เขียน log")
	}
	// info ถูก + permissions ผิด = 422 และชื่อไม่เปลี่ยน
	expect(t, call(t, app, "POST", subUpdatePath, map[string]any{"id": s.ID, "info": map[string]any{"name": "ชื่อใหม่"},
		"permissions": map[string]string{"dashboard": "edit"}}, c.shareTok), 200, 422)
	if d := detail(); d.Name != "ใจดี" {
		t.Fatalf("permissions ผิดต้องไม่บันทึก info %+v", d)
	}
	expect(t, call(t, app, "POST", subUpdatePath, map[string]any{"id": s.ID, "info": map[string]any{"name": "ใจดี"}}, c.comTok), 200, 402404) // ชั้นบนแก้ไม่ได้

	// เบอร์ (MGMT-08): ต้องส่งคู่กัน · ล้างด้วย "" ทั้งคู่
	for name, v := range map[string]map[string]any{
		"ส่งแค่ phone": {"phone": "811111111"},
		"code null":    {"phone_country_code": nil, "phone": "811111111"},
		"ล้างแค่ code": {"phone_country_code": "", "phone": "811111111"},
		"0 นำหน้า":     {"phone_country_code": "66", "phone": "0811111111"},
	} {
		r := call(t, app, "POST", subUpdatePath, map[string]any{"id": s.ID, "info": v}, c.shareTok)
		expect(t, r, 200, 422)
		if !strings.Contains(r.Msg, "info.") {
			t.Fatalf("%s: msg %q", name, r.Msg)
		}
	}
	expect(t, call(t, app, "POST", subUpdatePath, map[string]any{"id": s.ID, "info": map[string]any{"phone_country_code": "", "phone": ""}}, c.shareTok), 200, 200)
	var cleared models.Subaccount
	database.DBConn.Where("id = ?", s.ID).Take(&cleared)
	if cleared.Phone != nil || cleared.PhoneCountryCode != nil {
		t.Fatalf("ส่งค่าว่างทั้งคู่ ต้องเก็บ NULL %+v", cleared)
	}

	// แก้สิทธิ์แล้ว request ถัดไปของ sub ใช้สิทธิ์ใหม่ทันที
	expect(t, call(t, app, "POST", subUpdatePath, map[string]any{"id": s.ID, "permissions": map[string]string{"member": "view"}}, c.shareTok), 200, 200)
	subTok := readyToken(t, app, models.AccountTypeSub, s.ID, "share01@staff")
	expect(t, call(t, app, "POST", downlinesPath, map[string]any{}, subTok), 200, 200)
	expect(t, call(t, app, "POST", subUpdatePath, map[string]any{"id": s.ID, "permissions": map[string]string{}}, c.shareTok), 200, 200) // {} = ปิดทุกเมนู
	expect(t, call(t, app, "POST", downlinesPath, map[string]any{}, subTok), 200, 402303)
}

func TestSubInactive(t *testing.T) { // MGMT-43 · lead SA1 / SQ1: INACTIVE = login ไม่ได้ + session เดิมถูกตัด (401312)
	app := setup2(t)
	c := buildChain(t, app)
	s := createSubAPI(t, app, c.shareTok, "staff", map[string]string{"member": "view", "api_credential": "view"})
	subTok := readyToken(t, app, models.AccountTypeSub, s.ID, "share01@staff")
	expect(t, call(t, app, "GET", apiCredentialPath, nil, subTok), 200, 200)

	expect(t, call(t, app, "POST", subStatusPath, map[string]any{"id": s.ID, "status": "INACTIVE"}, c.shareTok), 200, 200)
	var row models.Subaccount
	database.DBConn.Where("id = ?", s.ID).Take(&row)
	if row.Status != models.AgentStatusSuspended {
		t.Fatalf("stored %s", row.Status)
	}
	r := call(t, app, "POST", subDetailPath, map[string]any{"id": s.ID}, c.shareTok)
	var d subData
	_ = json.Unmarshal(r.Data, &d)
	if d.Status != "INACTIVE" {
		t.Fatalf("status %s", d.Status)
	}
	// session เดิมถูกตัดทันที · ไม่เข้าข่าย "SUSPENDED ดูได้" (AUTH-54) — Profile / api-credential ก็ไม่ได้
	expect(t, call(t, app, "GET", profilePath, nil, subTok), 401, 401203)
	expect(t, call(t, app, "GET", apiCredentialPath, nil, subTok), 401, 401203)
	// login ใหม่ไม่ได้ (หลังรหัสผ่านถูก) · รหัสผิดยังได้ 401201 เหมือนเดิม
	expect(t, login(t, app, "share01@staff", mgPassword), 403, 401312)
	expect(t, login(t, app, "share01@staff", "wrong"+mgPassword), 401, 401201)

	// ตั้งกลับ ACTIVE = login ได้ · สถานะเดิม = 200 ไม่เขียน log (SA1)
	expect(t, call(t, app, "POST", subStatusPath, map[string]any{"id": s.ID, "status": "ACTIVE"}, c.shareTok), 200, 200)
	expect(t, call(t, app, "POST", subStatusPath, map[string]any{"id": s.ID, "status": "ACTIVE"}, c.shareTok), 200, 200)
	subTok = readyToken(t, app, models.AccountTypeSub, s.ID, "share01@staff")
	expect(t, call(t, app, "POST", downlinesPath, map[string]any{}, subTok), 200, 200)
	if n := countRows(t, &models.AccountChangeLog{}, "target_type = ? AND target_id = ? AND action = ?", "SUB", s.ID, models.ChangeSubStatus); n != 2 {
		t.Fatalf("SUB_STATUS logs %d, want 2", n)
	}
	expect(t, call(t, app, "POST", subStatusPath, map[string]any{"id": s.ID, "status": "INACTIVE"}, c.comTok), 200, 402404)
	expect(t, call(t, app, "POST", subStatusPath, map[string]any{"id": s.ID, "status": "SUSPENDED"}, c.shareTok), 200, 422)

	// ตั้ง INACTIVE ตรงใน DB (เหมือนลบ session ไม่สำเร็จ) — middleware ปฏิเสธเอง
	setCols(t, models.AccountTypeSub, s.ID, map[string]any{"status": models.AgentStatusSuspended})
	expect(t, call(t, app, "POST", downlinesPath, map[string]any{}, subTok), 403, 401312)
}

func TestSubMustChangePassword(t *testing.T) { // MGMT-41 · lead SQ3 · AUTH-43
	app := setup2(t)
	c := buildChain(t, app)
	s := createSubAPI(t, app, c.shareTok, "newstaff", map[string]string{"member": "view"})
	var row models.Subaccount
	database.DBConn.Where("id = ?", s.ID).Take(&row)
	if !row.MustChangePassword || row.TempPasswordExpiresAt != nil {
		t.Fatalf("sub ใหม่ต้อง must_change_password = true ไม่มีวันหมดอายุ %+v", row)
	}
	r := login(t, app, "share01@newstaff", mgPassword)
	expect(t, r, 200, 200)
	var ld struct {
		Token              string `json:"token"`
		MustChangePassword bool   `json:"must_change_password"`
	}
	_ = json.Unmarshal(r.Data, &ld)
	if !ld.MustChangePassword {
		t.Fatalf("login ต้องบอก must_change_password %s", r.Data)
	}
	expect(t, call(t, app, "POST", downlinesPath, map[string]any{}, ld.Token), 403, 401306)

	// ADMIN สร้าง sub ไม่ได้ (AUTH-43)
	admin := createAccount(t, "admin01", models.AgentRoleAdmin, nil)
	adminTok := loginReady(t, app, models.AccountTypeAgent, admin.ID, "admin01")
	expect(t, call(t, app, "POST", subCreatePath, subBody("staff", nil), adminTok), 200, 402301)
}
