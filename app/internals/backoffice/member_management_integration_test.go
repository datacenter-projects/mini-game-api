//go:build integration

// API test ของ member_management ตามตาราง test case ใน docs/modules/member_management.md หัวข้อ 5 (lead review 2026-10-09 M1–M5 · MQ1–MQ6)
// test ที่ใช้ร่วมกับ agent_management (สร้าง · PT · R1–R3 · CSM) อยู่ในไฟล์ agent_management_*_integration_test.go
package backoffice_test

import (
	"encoding/json"
	"strings"
	"testing"

	"app/app/models"
	agentAuthPostgres "app/app/repository/postgres/agent_auth"
	"app/pkg/utils"
	"app/platform/database"

	"github.com/gofiber/fiber/v2"
)

const (
	memberDetailUpdatePath = "/api/v1/bo/pr/manage/members/detail/update"
	memberListPath         = "/api/v1/bo/pr/manage/members/list"
)

// createMember — สร้าง Member ผ่าน API แล้วคืน id · body จาก memberBody แก้ได้ผ่าน edit
func createMember(t *testing.T, app *fiber.App, tok, username string, edit func(b map[string]any)) created {
	t.Helper()
	b := memberBody(username, 0.3)
	if edit != nil {
		edit(b)
	}
	r := call(t, app, "POST", createMemberPath, b, tok)
	expect(t, r, 200, 200)
	var d created
	_ = json.Unmarshal(r.Data, &d)
	return d
}

func memberLogs(t *testing.T, memberID uint, action models.AccountChangeAction) []models.AccountChangeLog {
	t.Helper()
	var out []models.AccountChangeLog
	database.DBConn.Where("target_type = ? AND target_id = ? AND action = ?", "MEMBER", memberID, action).Order("id").Find(&out)
	return out
}

func getMember(t *testing.T, id uint) models.UserMember {
	t.Helper()
	var m models.UserMember
	database.DBConn.Where("id = ?", id).Take(&m)
	return m
}

func TestMemberPhone(t *testing.T) { // MGMT-08 · lead E1-5 / MQ1
	app := setup2(t)
	c := buildChain(t, app) // share01 → agent01
	phone := func(code, p string) func(b map[string]any) {
		return func(b map[string]any) { b["phone_country_code"], b["phone"] = code, p }
	}
	m := createMember(t, app, c.agentTok, "memph1", phone(" 66 ", " 812345678 "))
	if got := getMember(t, m.ID); got.PhoneCountryCode == nil || *got.PhoneCountryCode != "66" || got.Phone == nil || *got.Phone != "812345678" {
		t.Fatalf("เบอร์ต้องเก็บแยก 2 field ตัดช่องว่าง %+v", got)
	}
	for name, edit := range map[string]func(b map[string]any){
		"ส่งแค่ phone":       phone("", "812345678"),
		"ส่งแค่ code":        phone("66", ""),
		"phone 0 นำหน้า":     phone("66", "0812345678"),
		"code ไม่มีจริง":     phone("999", "812345678"),
		"code มี +":          phone("+66", "812345678"),
		"รวมเกิน 15 หลัก":    phone("66", "81234567890123"),
		"ไม่ส่ง code (null)": func(b map[string]any) { b["phone_country_code"] = nil },
	} {
		b := memberBody("memphbad", 0)
		edit(b)
		r := call(t, app, "POST", createMemberPath, b, c.agentTok)
		expect(t, r, 200, 422)
		if !strings.Contains(r.Msg, "เบอร์โทร") && !strings.Contains(r.Msg, "รหัสประเทศ") {
			t.Fatalf("%s: msg ต้องบอก field %q", name, r.Msg)
		}
	}

	// ซ้ำกับ Member อื่นของผู้สร้างเดียวกัน = 402403 · คนละผู้สร้าง / ซ้ำกับ agent / คนละรหัสประเทศ = ได้
	expect(t, call(t, app, "POST", createMemberPath, func() map[string]any { b := memberBody("memph2", 0); phone("66", "812345678")(b); return b }(), c.agentTok), 200, 402403)
	createMember(t, app, c.shareTok, "memph3", phone("66", "812345678"))
	createMember(t, app, c.agentTok, "memph4", phone("856", "812345678"))
	expect(t, call(t, app, "POST", agentDetailUpdatePath, map[string]any{"id": c.agent.ID,
		"info": map[string]any{"phone_country_code": "66", "phone": "812345678"}}, c.shareTok), 200, 200)

	// detail/get แสดง 2 field · ไม่กรอก = ""
	r := call(t, app, "POST", memberDetailPath, map[string]any{"id": m.ID}, c.agentTok)
	expect(t, r, 200, 200)
	var d struct {
		Code  string `json:"phone_country_code"`
		Phone string `json:"phone"`
	}
	_ = json.Unmarshal(r.Data, &d)
	if d.Code != "66" || d.Phone != "812345678" {
		t.Fatalf("detail phone %s", r.Data)
	}
	noPhone := createMember(t, app, c.agentTok, "memph5", nil)
	r = call(t, app, "POST", memberDetailPath, map[string]any{"id": noPhone.ID}, c.agentTok)
	raw := map[string]json.RawMessage{}
	_ = json.Unmarshal(r.Data, &raw)
	if string(raw["phone_country_code"]) != `""` || string(raw["phone"]) != `""` {
		t.Fatalf("ไม่กรอกเบอร์ต้องได้ \"\" %s", r.Data)
	}

	// downlines/list แถว Member มีรหัสประเทศจริง
	r = call(t, app, "POST", downlinesPath, map[string]any{"keyword": "memph1"}, c.agentTok)
	expect(t, r, 200, 200)
	var p struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(r.Data, &p)
	if len(p.Data) != 1 || p.Data[0]["phone_country_code"] != "66" || p.Data[0]["phone"] != "812345678" {
		t.Fatalf("downlines/list แถว Member %v", p.Data)
	}

	// แก้เบอร์ใน detail/update: ซ้ำกับ Member อื่นของผู้สร้างเดียวกัน = 402403 · ล้าง "" ทั้งคู่ = NULL
	expect(t, call(t, app, "POST", memberDetailUpdatePath, map[string]any{"id": noPhone.ID,
		"info": map[string]any{"phone_country_code": "66", "phone": "812345678"}}, c.agentTok), 200, 402403)
	expect(t, call(t, app, "POST", memberDetailUpdatePath, map[string]any{"id": m.ID,
		"info": map[string]any{"phone_country_code": "66", "phone": "812345678"}}, c.agentTok), 200, 200) // เบอร์เดิมของตัวเอง
	expect(t, call(t, app, "POST", memberDetailUpdatePath, map[string]any{"id": m.ID,
		"info": map[string]any{"phone_country_code": "", "phone": ""}}, c.agentTok), 200, 200)
	if got := getMember(t, m.ID); got.Phone != nil || got.PhoneCountryCode != nil {
		t.Fatal(`ส่ง "" ทั้งคู่ ต้องเก็บเป็น NULL ทั้งคู่`)
	}
	expect(t, call(t, app, "POST", memberDetailUpdatePath, map[string]any{"id": noPhone.ID,
		"info": map[string]any{"phone_country_code": "66", "phone": "812345678"}}, c.agentTok), 200, 200) // เบอร์ว่างแล้ว
}

func TestMemberDetailUpdate(t *testing.T) { // MGMT-32M · MGMT-32N · lead M3-1 / MN2 / MQ4
	app := setup2(t)
	c := buildChain(t, app) // agent01 ได้รับ 60
	m := createMemberWithPT(t, app, c.agentTok, "memupd", 50)
	upd := func(body map[string]any, tok string, code int) {
		t.Helper()
		expect(t, call(t, app, "POST", memberDetailUpdatePath, body, tok), 200, code)
	}
	logCount := func() int64 {
		return countRows(t, &models.AccountChangeLog{}, "target_type = ? AND target_id = ? AND action IN ?", "MEMBER", m.ID,
			[]models.AccountChangeAction{models.ChangeUpdateInfo, models.ChangeUpdatePT})
	}

	// รูปแบบ
	r := call(t, app, "POST", memberDetailUpdatePath, map[string]any{"id": m.ID}, c.agentTok)
	expect(t, r, 200, 422)
	expectMsgHas(t, r.Msg, "ต้องส่งอย่างน้อย 1 section")
	upd(map[string]any{"id": m.ID, "info": map[string]any{}}, c.agentTok, 422)
	upd(map[string]any{"id": m.ID, "info": nil}, c.agentTok, 422)
	upd(map[string]any{"id": m.ID, "pt": map[string]any{}}, c.agentTok, 422)
	upd(map[string]any{"id": m.ID, "pt": map[string]any{"minigame": map[string]any{"pt": 40}}}, c.agentTok, 422) // ไม่ส่ง commission_percent
	upd(map[string]any{"id": m.ID, "pt": map[string]any{"minigame": map[string]any{"pt": 40.25, "commission_percent": 0.3}}}, c.agentTok, 422)
	upd(map[string]any{"id": m.ID, "pt": map[string]any{"minigame": map[string]any{"pt": 40, "commission_percent": 0.3, "remain_quota": 20}}}, c.agentTok, 422)
	upd(map[string]any{"id": m.ID, "pt": map[string]any{"minigame": map[string]any{"pt": 40, "commission_percent": 0.3, "status": true}}}, c.agentTok, 422)

	// info + pt ใน request เดียว = log 2 แถว
	upd(map[string]any{"id": m.ID, "info": map[string]any{"name": "ใจดี"}, "pt": map[string]any{"minigame": map[string]any{"pt": 40, "commission_percent": 0.3}}}, c.agentTok, 200)
	if got := getMember(t, m.ID); got.Name != "ใจดี" {
		t.Fatalf("name %q", got.Name)
	}
	if s := memberSetting(t, m.ID); s.PT != 40 || s.Remain != 20 || s.Commission != 0.3 || s.UpdatedBy != "agent01" {
		t.Fatalf("pt %+v", s)
	}
	if len(memberLogs(t, m.ID, models.ChangeUpdateInfo)) != 1 || len(memberLogs(t, m.ID, models.ChangeUpdatePT)) != 1 {
		t.Fatal("ต้องมี log UPDATE_INFO 1 แถว และ UPDATE_PT 1 แถว")
	}
	pl := memberLogs(t, m.ID, models.ChangeUpdatePT)[0]
	if pl.OldValue == nil || !strings.Contains(*pl.OldValue, `"pt": 50`) || !strings.Contains(*pl.NewValue, `"pt": 40`) { // jsonb เก็บแบบมีช่องว่าง
		t.Fatalf("log pt %s → %s", *pl.OldValue, *pl.NewValue)
	}

	// ส่งค่าเดิมทั้งหมด = 200 ไม่มี log เพิ่ม
	before := logCount()
	upd(map[string]any{"id": m.ID, "info": map[string]any{"name": "ใจดี", "phone_country_code": "", "phone": ""},
		"pt": map[string]any{"minigame": map[string]any{"pt": 40, "commission_percent": 0.3}}}, c.agentTok, 200)
	if logCount() != before {
		t.Fatal("ส่งค่าเดิมต้องไม่เขียน log")
	}

	// tx เดียว: pt ผิด (เกินที่ได้รับ) หลัง info ถูก = 402305 และชื่อไม่เปลี่ยน
	r = call(t, app, "POST", memberDetailUpdatePath, map[string]any{"id": m.ID, "info": map[string]any{"name": "ชื่อใหม่"},
		"pt": map[string]any{"minigame": map[string]any{"pt": 60.5, "commission_percent": 0.3}}}, c.agentTok)
	expect(t, r, 200, 402305)
	expectMsgHas(t, r.Msg, "PT ที่ถือสู้กับสมาชิก", "60%")
	if got := getMember(t, m.ID); got.Name != "ใจดี" {
		t.Fatalf("pt ผิดต้องไม่บันทึก info ได้ชื่อ %q", got.Name)
	}
	upd(memberPTBody(m.ID, 30, 1.1), c.agentTok, 402309)

	// pt เท่าที่ได้รับ = remain_quota 0
	upd(memberPTBody(m.ID, 60, 0.3), c.agentTok, 200)
	if s := memberSetting(t, m.ID); s.PT != 60 || s.Remain != 0 {
		t.Fatalf("pt 60 %+v", s)
	}

	// ผู้สร้างโดยตรงเท่านั้น: ชั้นบน = 402304 · นอกสาย / ไม่มี = 402402
	upd(memberPTBody(m.ID, 30, 0.3), c.shareTok, 402304)
	upd(map[string]any{"id": m.ID, "info": map[string]any{"name": "ชั้นบน"}}, c.comTok, 402304)
	_, otherTok := mustCreate(t, app, c.saTok, agentBody("COMPANY_TRANSFER", "othercom", nil, childPT(50, 0, 0, 0)))
	upd(memberPTBody(m.ID, 30, 0.3), otherTok, 402402)
	upd(memberPTBody(99999, 30, 0.3), c.agentTok, 402402)

	// MGMT-32N: ไม่มีแถวค่าตั้ง (ข้อมูลก่อน module ②) = ข้ามเงียบ 200 ไม่เขียน log
	execSQL(t, "DELETE FROM user_member_game_settings WHERE user_member_id = ?", m.ID)
	before = logCount()
	upd(memberPTBody(m.ID, 10, 0.1), c.agentTok, 200)
	if logCount() != before || countRows(t, &models.UserMemberGameSetting{}, "user_member_id = ?", m.ID) != 0 {
		t.Fatal("ไม่มีแถวค่าตั้งต้องไม่บันทึกและไม่เขียน log")
	}
}

func TestMemberStatus(t *testing.T) { // MGMT-30 · lead MQ5
	app := setup2(t)
	c := buildChain(t, app)
	m := createMember(t, app, c.agentTok, "memst", nil)
	expect(t, call(t, app, "POST", memberStatusPath, map[string]any{"id": m.ID, "status": "LOCKED"}, c.agentTok), 200, 200)
	expect(t, call(t, app, "POST", memberStatusPath, map[string]any{"id": m.ID, "status": "ACTIVE"}, c.agentTok), 200, 200)
	if n := len(memberLogs(t, m.ID, models.ChangeUpdateStatus)); n != 2 {
		t.Fatalf("status logs %d, want 2", n)
	}
	expect(t, call(t, app, "POST", memberStatusPath, map[string]any{"id": m.ID, "status": "ACTIVE"}, c.agentTok), 200, 200) // สถานะเดิม
	if n := len(memberLogs(t, m.ID, models.ChangeUpdateStatus)); n != 2 {
		t.Fatalf("ส่งสถานะเดิมต้องไม่เขียน log ได้ %d แถว", n)
	}
	expect(t, call(t, app, "POST", memberStatusPath, map[string]any{"id": m.ID, "status": "LOCKED"}, c.shareTok), 200, 402304)
	expect(t, call(t, app, "POST", memberStatusPath, map[string]any{"id": m.ID, "status": "PAUSED"}, c.agentTok), 200, 422)

	// หัวสายถูกระงับ = Member แสดง SUSPENDED
	setStatus(t, c.share.ID, models.AgentStatusSuspended)
	r := call(t, app, "POST", memberDetailPath, map[string]any{"id": m.ID}, c.comTok)
	expect(t, r, 200, 200)
	var d struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal(r.Data, &d)
	if d.Status != "SUSPENDED" {
		t.Fatalf("status %q, want SUSPENDED", d.Status)
	}
}

func TestMemberList(t *testing.T) { // MGMT-35M · lead M5
	app := setup2(t)
	c := buildChain(t, app)
	createMember(t, app, c.agentTok, "memzed", nil)
	createMember(t, app, c.agentTok, "memalpha", func(b map[string]any) { b["name"] = "ใจดี" })
	createMember(t, app, c.shareTok, "memofshare", nil) // ไม่ใช่ลูกตรงของ agent01

	list := func(tok string, body map[string]any) []map[string]any {
		t.Helper()
		r := call(t, app, "POST", memberListPath, body, tok)
		expect(t, r, 200, 200)
		var out []map[string]any
		if err := json.Unmarshal(r.Data, &out); err != nil {
			t.Fatalf("%s: %v", r.Data, err)
		}
		return out
	}
	rows := list(c.agentTok, map[string]any{})
	if len(rows) != 2 || rows[0]["username"] != "memalpha" || rows[1]["username"] != "memzed" || rows[0]["name"] != "ใจดี" ||
		rows[0]["status"] != "ACTIVE" || len(rows[0]) != 4 {
		t.Fatalf("members/list %v", rows)
	}
	if rows := list(c.agentTok, map[string]any{"keyword": " ALPH "}); len(rows) != 1 || rows[0]["username"] != "memalpha" {
		t.Fatalf("keyword %v", rows)
	}
	if rows := list(c.agentTok, map[string]any{"keyword": "a_ph"}); len(rows) != 0 { // _ เป็นตัวอักษรธรรมดา ไม่ใช่ wildcard
		t.Fatalf("keyword _ %v", rows)
	}
	expect(t, call(t, app, "POST", memberListPath, map[string]any{"keyword": "mem"}, c.agentTok), 200, 422) // สั้นกว่า 4 ตัว
	r := call(t, app, "POST", memberListPath, map[string]any{}, c.comTok)                                   // ไม่มี Member ลูกตรง
	expect(t, r, 200, 200)
	if string(r.Data) != "[]" {
		t.Fatalf("ไม่มี Member ลูกตรงต้องได้ [] ได้ %s", r.Data)
	}

	// status = สถานะของ Member (ผู้เรียกที่ถูกระงับเรียกเส้นนี้ไม่ได้ — 401311)
	zed := rows[1]["id"].(float64)
	expect(t, call(t, app, "POST", memberStatusPath, map[string]any{"id": zed, "status": "LOCKED"}, c.agentTok), 200, 200)
	if rows := list(c.agentTok, map[string]any{}); rows[1]["status"] != "LOCKED" {
		t.Fatalf("status %v", rows)
	}
}

func TestMemberSubPermission(t *testing.T) { // MGMT-51 · lead MN4 / U3
	app := setup2(t)
	c := buildChain(t, app)
	m := createMember(t, app, c.agentTok, "memsub", nil)
	hash, _ := utils.HashPassword(mgPassword)
	sub := models.Subaccount{AgentID: c.agent.ID, Username: "agent01@staff", PasswordHash: hash, Status: models.AgentStatusActive,
		Permissions: `{"member":"view"}`}
	if err := agentAuthPostgres.CreateSubaccountRepository(database.DBConn, &sub); err != nil {
		t.Fatal(err)
	}
	tok := readyToken(t, app, models.AccountTypeSub, sub.ID, sub.Username)

	// view: detail/get · list ได้ (pt แสดงเสมอ) · แก้ / สร้างไม่ได้
	r := call(t, app, "POST", memberDetailPath, map[string]any{"id": m.ID}, tok)
	expect(t, r, 200, 200)
	raw := map[string]json.RawMessage{}
	_ = json.Unmarshal(r.Data, &raw)
	if len(raw["pt"]) < 3 {
		t.Fatalf("sub member view ต้องเห็น pt %s", r.Data)
	}
	expect(t, call(t, app, "POST", memberListPath, map[string]any{}, tok), 200, 200)
	expect(t, call(t, app, "POST", memberDetailUpdatePath, memberPTBody(m.ID, 10, 0.1), tok), 200, 402303)
	expect(t, call(t, app, "POST", memberStatusPath, map[string]any{"id": m.ID, "status": "LOCKED"}, tok), 200, 402303)
	expect(t, call(t, app, "POST", createMemberPath, memberBody("memsub2", 0), tok), 200, 402303)

	// edit: แก้ได้ · สร้างพร้อม balance ต้องมี payment edit (เช็คหลังรูปแบบ — U3)
	setCols(t, models.AccountTypeSub, sub.ID, map[string]any{"permissions": `{"member":"edit"}`})
	expect(t, call(t, app, "POST", memberDetailUpdatePath, memberPTBody(m.ID, 10, 0.1), tok), 200, 200)
	if s := memberSetting(t, m.ID); s.UpdatedBy != "agent01@staff" { // MGMT-16: คนกดจริง
		t.Fatalf("updated_by %q", s.UpdatedBy)
	}
	setBalance(t, c.agent.ID, "THB", 100)
	b := memberBody("memsub3", 0)
	b["balance"] = map[string]any{"THB": 10}
	expect(t, call(t, app, "POST", createMemberPath, b, tok), 200, 402303)
	b["username"] = "x" // รูปแบบผิดมาก่อนสิทธิ์ payment
	expect(t, call(t, app, "POST", createMemberPath, b, tok), 200, 422)
	setCols(t, models.AccountTypeSub, sub.ID, map[string]any{"permissions": `{"member":"edit","payment":"edit"}`})
	b["username"] = "memsub3"
	expect(t, call(t, app, "POST", createMemberPath, b, tok), 200, 200)
}

func TestMemberCreateRules(t *testing.T) { // MGMT-15A · MGMT-60 · หัวข้อ 2.1 (lead U3 / MQ2 / Q-R4 / E1-4)
	app := setup2(t)
	c := buildChain(t, app) // agent01 ได้รับ 60 · ยอด 0
	createMember(t, app, c.agentTok, "memtaken", func(b map[string]any) { b["phone_country_code"], b["phone"] = "66", "811111111" })
	bad := func(edit func(b map[string]any)) map[string]any {
		b := memberBody("memnew", 0.3)
		edit(b)
		return b
	}
	// ลำดับเช็คที่ต้องดู DB: username → เบอร์ → balance → pt → commission_percent
	all := func(b map[string]any) {
		b["phone_country_code"], b["phone"] = "66", "811111111"
		b["balance"] = map[string]any{"THB": 10}
		b["pt"] = map[string]any{"minigame": map[string]any{"pt": 65, "commission_percent": 1}}
	}
	expect(t, call(t, app, "POST", createMemberPath, bad(func(b map[string]any) { all(b); b["username"] = "agent01" }), c.agentTok), 200, 402401)
	expect(t, call(t, app, "POST", createMemberPath, bad(all), c.agentTok), 200, 402403)
	expect(t, call(t, app, "POST", createMemberPath, bad(func(b map[string]any) { all(b); b["phone"] = "822222222" }), c.agentTok), 200, 402312)
	expect(t, call(t, app, "POST", createMemberPath, bad(func(b map[string]any) { all(b); b["phone"] = "822222222"; delete(b, "balance") }), c.agentTok), 200, 402305)

	// ยอดตั้งต้น: credit · ledger 2 แถว · log INITIAL_BALANCE ใช้ key amounts · target_type MEMBER
	setBalance(t, c.agent.ID, "THB", 100)
	rich := memberBody("memrich", 0)
	rich["balance"] = map[string]any{"THB": 25.5}
	r := call(t, app, "POST", createMemberPath, rich, c.agentTok)
	expect(t, r, 200, 200)
	var m created
	_ = json.Unmarshal(r.Data, &m)
	if got := getMember(t, m.ID); got.Credit != 25.5 || agentBalance(t, c.agent.ID, "THB") != 74.5 {
		t.Fatalf("credit %v agent %v", got.Credit, agentBalance(t, c.agent.ID, "THB"))
	}
	if n := countRows(t, &models.BalanceLedger{}, "request_id = ?", rich["request_id"]); n != 2 {
		t.Fatalf("ledger %d, want 2", n)
	}
	il := memberLogs(t, m.ID, models.ChangeInitialBalance)
	if len(il) != 1 || il[0].NewValue == nil || !strings.Contains(*il[0].NewValue, `"amounts": {"THB": 25.5}`) || strings.Contains(*il[0].NewValue, "amounts_minor") {
		t.Fatalf("INITIAL_BALANCE %+v", il)
	}
	if cl := memberLogs(t, m.ID, models.ChangeCreate); len(cl) != 1 || !strings.Contains(*cl[0].NewValue, `"phone_country_code": ""`) {
		t.Fatalf("CREATE log %+v", cl)
	}

	// request_id: ส่งซ้ำของตัวเอง = id เดิม ไม่โอนซ้ำ · ของคนอื่น / ที่ใช้สร้าง agent = 422
	r = call(t, app, "POST", createMemberPath, rich, c.agentTok)
	expect(t, r, 200, 200)
	var again created
	_ = json.Unmarshal(r.Data, &again)
	if again.ID != m.ID || agentBalance(t, c.agent.ID, "THB") != 74.5 {
		t.Fatalf("replay id %d (want %d) agent %v", again.ID, m.ID, agentBalance(t, c.agent.ID, "THB"))
	}
	other := memberBody("memother", 0)
	other["request_id"] = rich["request_id"]
	expect(t, call(t, app, "POST", createMemberPath, other, c.shareTok), 200, 422)
	ag := agentBody("AGENT", "agentrid", nil, childPT(10, 0, 0, 0))
	expect(t, call(t, app, "POST", createAgentPath, ag, c.agentTok), 200, 200)
	other["request_id"] = ag["request_id"]
	expect(t, call(t, app, "POST", createMemberPath, other, c.agentTok), 200, 422)

	// สาย Seamless ส่ง balance ได้ (MGMT-15 แก้ 2026-10-15) · หักจากผู้สร้าง · payment_type ยังเป็น CASH (MGMT-63)
	one, oneTok := mustCreate(t, app, c.saTok, agentBody("COMPANY_SEAMLESS_1TO1", "one2one", []string{"JPY"}, childPT(80, 0, 0, 0)))
	setBalance(t, one.ID, "JPY", 100)
	sb := memberBody("memseam", 0)
	sb["balance"] = map[string]any{"JPY": 10}
	expect(t, call(t, app, "POST", createMemberPath, sb, oneTok), 200, 200)
	var seam models.UserMember
	database.DBConn.Select("credit", "payment_type").Where("username = ?", "memseam").Take(&seam)
	if seam.Credit != 10 || agentBalance(t, one.ID, "JPY") != 90 || seam.PaymentType != models.PaymentTypeCash {
		t.Fatalf("Seamless member balance: credit %v · company %v · payment %s", seam.Credit, agentBalance(t, one.ID, "JPY"), seam.PaymentType)
	}
}

func TestMemberRemovedPaths(t *testing.T) { // lead MN1: path เดิมได้ 404 (ไม่คง alias)
	app := setup2(t)
	c := buildChain(t, app)
	m := createMember(t, app, c.agentTok, "memold", nil)
	for _, p := range []string{"members/detail", "members/update-info", "members/update-status", "members/update-pt", "members/update-commission"} {
		if r := call(t, app, "POST", "/api/v1/bo/pr/manage/"+p, map[string]any{"id": m.ID}, c.agentTok); r.Status != 404 {
			t.Fatalf("%s ต้องไม่มีแล้ว ได้ HTTP %d code %d", p, r.Status, r.Code)
		}
	}
}
