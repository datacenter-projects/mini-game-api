//go:build integration

// API test ของวิธีจ่ายเงิน · บัญชีธนาคาร · ลิมิตชนะต่อวัน ของ Member — docs/modules/member_management.md MGMT-63 – MGMT-65
// (lead review 2026-10-12 · docs/reviews/member-payment.md P1–P15 · รายการ C4)
package backoffice_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"app/app/models"
	agentAuthPostgres "app/app/repository/postgres/agent_auth"
	"app/pkg/utils"
	"app/platform/database"

	"github.com/gofiber/fiber/v2"
)

// kbank — บัญชีตัวอย่าง (แก้ field ได้ผ่าน kv)
func kbank(number string, kv ...string) map[string]any {
	a := map[string]any{"bank_code": "KBANK", "account_number": number}
	for i := 0; i+1 < len(kv); i += 2 {
		a[kv[i]] = kv[i+1]
	}
	return a
}

func withPayment(t string, account map[string]any) func(b map[string]any) {
	return func(b map[string]any) {
		b["payment_type"] = t
		if account != nil {
			b["payment_account"] = account
		}
	}
}

func storedAccount(t *testing.T, id uint) (models.MemberPaymentAccount, bool) {
	t.Helper()
	m := getMember(t, id)
	if m.PaymentAccount == nil {
		return models.MemberPaymentAccount{}, false
	}
	var a models.MemberPaymentAccount
	if err := json.Unmarshal([]byte(*m.PaymentAccount), &a); err != nil {
		t.Fatal(err)
	}
	return a, true
}

type memberPaymentDetail struct {
	PaymentType    string            `json:"payment_type"`
	PaymentAccount map[string]string `json:"payment_account"`
	WinLimitPerDay float64           `json:"win_limit_per_day"`
}

func memberPayment(t *testing.T, app *fiber.App, id uint, tok string) memberPaymentDetail {
	t.Helper()
	r := call(t, app, "POST", memberDetailPath, map[string]any{"id": id}, tok)
	expect(t, r, 200, 200)
	var d memberPaymentDetail
	if err := json.Unmarshal(r.Data, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestMemberPaymentCreate(t *testing.T) { // MGMT-63 – MGMT-65 · P1–P9 · P12
	app := setup2(t)
	c := buildChain(t, app) // share01 → agent01

	// สร้างครบ: ตัวเล็ก / ช่องว่างถูกปรับ · เก็บ JSONB · detail/get แสดงเลขเต็ม
	m := createMember(t, app, c.agentTok, "mempay1", func(b map[string]any) {
		withPayment(" credit ", kbank(" 7876766654 ", "bank_code", "kbank", "name_th", " ใจดี ", "surname_en", "Mee-Suk"))(b)
		b["win_limit_per_day"] = 5000.5
	})
	if a, ok := storedAccount(t, m.ID); !ok || a.BankCode != "KBANK" || a.AccountNumber != "7876766654" || a.NameTH != "ใจดี" || a.SurnameEN != "Mee-Suk" {
		t.Fatalf("payment_account ที่เก็บ %+v", a)
	}
	d := memberPayment(t, app, m.ID, c.agentTok)
	if d.PaymentType != "CREDIT" || d.WinLimitPerDay != 5000.5 || d.PaymentAccount["account_number"] != "7876766654" ||
		d.PaymentAccount["bank_code"] != "KBANK" || d.PaymentAccount["midname_en"] != "" || len(d.PaymentAccount) != 8 {
		t.Fatalf("detail %+v", d)
	}

	// ลิมิตเท่าเพดาน 1,000,000 พอดี = ได้ (MGMT-65)
	top := createMember(t, app, c.agentTok, "mempaytop", func(b map[string]any) { b["win_limit_per_day"] = 1000000 })
	if got := getMember(t, top.ID).WinLimitPerDay; got != 1000000 {
		t.Fatalf("win_limit_per_day %v, want 1000000", got)
	}

	// ไม่ส่งบัญชี / ลิมิต: NULL · detail ทุก field "" · ลิมิต 0
	plain := createMember(t, app, c.agentTok, "mempay2", nil)
	if _, ok := storedAccount(t, plain.ID); ok || getMember(t, plain.ID).WinLimitPerDay != 0 {
		t.Fatal("ไม่ส่งบัญชีต้องเก็บ NULL และลิมิต 0")
	}
	d = memberPayment(t, app, plain.ID, c.agentTok)
	if d.PaymentType != "CASH" || len(d.PaymentAccount) != 8 || d.PaymentAccount["bank_code"] != "" || d.PaymentAccount["account_number"] != "" {
		t.Fatalf("ไม่มีบัญชีต้องได้ทุก field \"\" %+v", d)
	}
	for i, empty := range []map[string]any{{}, {"bank_code": "", "account_number": ""}, {"bank_code": " ", "account_number": " "}} { // = ไม่มีบัญชี
		e := createMember(t, app, c.agentTok, fmt.Sprintf("mempayempty%d", i), withPayment("CASH", empty))
		if _, ok := storedAccount(t, e.ID); ok {
			t.Fatalf("%v ต้องเก็บ NULL", empty)
		}
	}

	// รูปแบบ = 422 และ msg บอก field
	for name, tc := range map[string]struct {
		edit  func(b map[string]any)
		field string
	}{
		"ไม่ส่ง payment_type":        {func(b map[string]any) { delete(b, "payment_type") }, "payment_type"},
		"payment_type DEBIT":         {withPayment("DEBIT", nil), "payment_type"},
		"ส่งแค่ bank_code":           {withPayment("CASH", map[string]any{"bank_code": "KBANK"}), "เลขบัญชี"},
		"ส่งแค่ account_number":      {withPayment("CASH", map[string]any{"account_number": "7876766654"}), "ธนาคาร"},
		"ธนาคารไม่อยู่ในรายการ":      {withPayment("CASH", kbank("7876766654", "bank_code", "XYZ")), "ธนาคาร"},
		"เลขบัญชี 9 หลัก":            {withPayment("CASH", kbank("787676665")), "เลขบัญชี"},
		"เลขบัญชีมีขีด":              {withPayment("CASH", kbank("787-676-6654")), "เลขบัญชี"},
		"ชื่อโดยไม่มีเลขบัญชี":       {withPayment("CASH", map[string]any{"name_th": "ใจดี"}), "ชื่อ (ไทย)"},
		"name_th เป็นอังกฤษ":         {withPayment("CASH", kbank("7876766654", "name_th", "Jaidee")), "ชื่อ (ไทย)"},
		"name_en เป็นไทย":            {withPayment("CASH", kbank("7876766654", "name_en", "ใจดี")), "ชื่อ (อังกฤษ)"},
		"ชื่อ 51 ตัว":                {withPayment("CASH", kbank("7876766654", "surname_en", strings.Repeat("a", 51))), "นามสกุล (อังกฤษ)"},
		"payment_account null":       {func(b map[string]any) { b["payment_account"] = nil }, "payment_account"},
		"win_limit ติดลบ":            {func(b map[string]any) { b["win_limit_per_day"] = -1 }, "win_limit_per_day"},
		"win_limit ทศนิยม 5 ตำแหน่ง": {func(b map[string]any) { b["win_limit_per_day"] = 10.12345 }, "win_limit_per_day"},
		"win_limit เกิน 1,000,000":   {func(b map[string]any) { b["win_limit_per_day"] = 1000000.5 }, "ต้องไม่เกิน 1000000"},
	} {
		b := memberBody("mempaybad", 0)
		tc.edit(b)
		r := call(t, app, "POST", createMemberPath, b, c.agentTok)
		expect(t, r, 200, 422)
		if !strings.Contains(r.Msg, tc.field) {
			t.Fatalf("%s: msg %q ต้องบอก field %s", name, r.Msg, tc.field)
		}
	}

	// win_limit_per_day รับเป็น number (float64 — ทีมตกลง 2026-10-12) · ส่งเป็นข้อความ = 400 (body parse ไม่ผ่าน)
	sb := memberBody("mempaystr", 0)
	sb["win_limit_per_day"] = "100"
	expect(t, call(t, app, "POST", createMemberPath, sb, c.agentTok), 200, 400)

	// ซ้ำที่คู่ใต้ผู้สร้างเดียวกัน = 402405 · คนละผู้สร้าง / คนละธนาคาร = ได้
	expect(t, call(t, app, "POST", createMemberPath, func() map[string]any {
		b := memberBody("mempaydup", 0)
		withPayment("CASH", kbank("7876766654"))(b)
		return b
	}(), c.agentTok), 200, 402405)
	createMember(t, app, c.shareTok, "mempay6", withPayment("CASH", kbank("7876766654")))
	createMember(t, app, c.agentTok, "mempay7", withPayment("CASH", kbank("7876766654", "bank_code", "SCB")))

	// log CREATE: เลขบัญชีปิดบางส่วน · มี payment_type / win_limit_per_day
	cl := memberLogs(t, m.ID, models.ChangeCreate)
	if len(cl) != 1 || !strings.Contains(*cl[0].NewValue, "787•••6654") || strings.Contains(*cl[0].NewValue, "7876766654") ||
		!strings.Contains(*cl[0].NewValue, `"payment_type": "CREDIT"`) || !strings.Contains(*cl[0].NewValue, `"win_limit_per_day": 5000.5`) {
		t.Fatalf("CREATE log %s", *cl[0].NewValue)
	}

	// ผู้สร้างฝั่ง Seamless ส่ง CREDIT = เก็บ CASH (P1)
	_, oneTok := mustCreate(t, app, c.saTok, agentBody("COMPANY_SEAMLESS_1TO1", "one2one", []string{"JPY"}, childPT(80, 0, 0, 0)))
	sm := createMember(t, app, oneTok, "mempayseam", withPayment("CREDIT", nil))
	if got := getMember(t, sm.ID).PaymentType; got != models.PaymentTypeCash {
		t.Fatalf("Seamless payment_type %q, want CASH", got)
	}
}

func TestMemberPaymentUpdate(t *testing.T) { // MGMT-32M · MGMT-63 – MGMT-65 · P13
	app := setup2(t)
	c := buildChain(t, app)
	m := createMember(t, app, c.agentTok, "mempayupd", withPayment("CASH", nil))
	other := createMember(t, app, c.agentTok, "mempayoth", withPayment("CASH", kbank("1112223334")))
	upd := func(body map[string]any, code int) {
		t.Helper()
		expect(t, call(t, app, "POST", memberDetailUpdatePath, body, c.agentTok), 200, code)
	}
	logCount := func() int64 {
		return countRows(t, &models.AccountChangeLog{}, "target_type = ? AND target_id = ? AND action IN ?", "MEMBER", m.ID,
			[]models.AccountChangeAction{models.ChangeUpdatePayment, models.ChangeUpdateLimit})
	}

	// รูปแบบ
	upd(map[string]any{"id": m.ID, "payment": map[string]any{}}, 422)
	upd(map[string]any{"id": m.ID, "limit": map[string]any{}}, 422)
	upd(map[string]any{"id": m.ID, "payment": map[string]any{"payment_type": 1}}, 422)
	upd(map[string]any{"id": m.ID, "payment": map[string]any{"payment_account": kbank("7876766654", "bank_code", "XYZ")}}, 422)
	upd(map[string]any{"id": m.ID, "limit": map[string]any{"win_limit_per_day": -5}}, 422)
	upd(map[string]any{"id": m.ID, "limit": map[string]any{"win_limit_per_day": 1000001}}, 422) // เกินเพดาน (MGMT-65)

	// payment + limit = log 2 แถว · เลขบัญชีใน log ปิดบางส่วน
	upd(map[string]any{"id": m.ID, "payment": map[string]any{"payment_type": "CREDIT", "payment_account": kbank("7876766654", "name_en", "Jaidee")},
		"limit": map[string]any{"win_limit_per_day": 3000}}, 200)
	got := getMember(t, m.ID)
	if a, ok := storedAccount(t, m.ID); !ok || got.PaymentType != models.PaymentTypeCredit || got.WinLimitPerDay != 3000 || a.AccountNumber != "7876766654" || a.NameEN != "Jaidee" {
		t.Fatalf("หลังแก้ %+v %+v", got, a)
	}
	pl := memberLogs(t, m.ID, models.ChangeUpdatePayment)
	if len(pl) != 1 || len(memberLogs(t, m.ID, models.ChangeUpdateLimit)) != 1 ||
		!strings.Contains(*pl[0].NewValue, "787•••6654") || strings.Contains(*pl[0].NewValue, "7876766654") || !strings.Contains(*pl[0].OldValue, `"payment_type": "CASH"`) {
		t.Fatalf("log payment %+v", pl)
	}

	// ส่งค่าเดิม = 200 ไม่มี log เพิ่ม
	before := logCount()
	upd(map[string]any{"id": m.ID, "payment": map[string]any{"payment_type": "credit", "payment_account": kbank(" 7876766654 ", "name_en", "Jaidee")},
		"limit": map[string]any{"win_limit_per_day": 3000}}, 200)
	if logCount() != before {
		t.Fatal("ส่งค่าเดิมต้องไม่เขียน log")
	}

	// ส่งแค่ payment_type = บัญชีคงเดิม · payment_account แทนทั้งก้อน (ชื่อที่ไม่ส่ง = ล้าง)
	upd(map[string]any{"id": m.ID, "payment": map[string]any{"payment_type": "CASH"}}, 200)
	if a, _ := storedAccount(t, m.ID); a.NameEN != "Jaidee" || getMember(t, m.ID).PaymentType != models.PaymentTypeCash {
		t.Fatal("ส่งแค่ payment_type ต้องไม่แตะบัญชี")
	}
	upd(map[string]any{"id": m.ID, "payment": map[string]any{"payment_account": kbank("7876766654")}}, 200)
	if a, _ := storedAccount(t, m.ID); a.NameEN != "" {
		t.Fatal("payment_account ต้องแทนทั้งก้อน")
	}

	// ซ้ำกับ Member อื่นของผู้สร้างเดียวกัน = 402405 · tx เดียว: info ที่ถูกก่อนหน้าไม่ถูกบันทึก
	r := call(t, app, "POST", memberDetailUpdatePath, map[string]any{"id": m.ID, "info": map[string]any{"name": "ชื่อใหม่"},
		"payment": map[string]any{"payment_account": kbank("1112223334")}}, c.agentTok)
	expect(t, r, 200, 402405)
	if getMember(t, m.ID).Name == "ชื่อใหม่" {
		t.Fatal("payment ซ้ำต้องไม่บันทึก info")
	}
	upd(map[string]any{"id": other.ID, "payment": map[string]any{"payment_account": kbank("1112223334")}}, 200) // บัญชีเดิมของตัวเอง

	// ล้างบัญชี {} = NULL · แล้วบัญชีนั้นว่างให้คนอื่นใช้
	upd(map[string]any{"id": other.ID, "payment": map[string]any{"payment_account": map[string]any{}}}, 200)
	if _, ok := storedAccount(t, other.ID); ok {
		t.Fatal("payment_account {} ต้องเก็บ NULL")
	}
	upd(map[string]any{"id": m.ID, "payment": map[string]any{"payment_account": kbank("1112223334")}}, 200)

	// ผู้สร้างโดยตรงเท่านั้น
	expect(t, call(t, app, "POST", memberDetailUpdatePath, map[string]any{"id": m.ID, "limit": map[string]any{"win_limit_per_day": 1}}, c.shareTok), 200, 402304)

	// ผู้สร้างฝั่ง Seamless ส่ง CREDIT = คง CASH ไม่มี log
	_, oneTok := mustCreate(t, app, c.saTok, agentBody("COMPANY_SEAMLESS_1TO1", "one2one", []string{"JPY"}, childPT(80, 0, 0, 0)))
	sm := createMember(t, app, oneTok, "mempayseam", nil)
	expect(t, call(t, app, "POST", memberDetailUpdatePath, map[string]any{"id": sm.ID, "payment": map[string]any{"payment_type": "CREDIT"}}, oneTok), 200, 200)
	if getMember(t, sm.ID).PaymentType != models.PaymentTypeCash || len(memberLogs(t, sm.ID, models.ChangeUpdatePayment)) != 0 {
		t.Fatal("Seamless ต้องเป็น CASH เสมอ")
	}
}

func TestMemberPaymentSubPermission(t *testing.T) { // MGMT-51 · lead P14
	app := setup2(t)
	c := buildChain(t, app)
	m := createMember(t, app, c.agentTok, "mempaysub", nil)
	hash, _ := utils.HashPassword(mgPassword)
	sub := models.Subaccount{AgentID: c.agent.ID, Username: "agent01@staff", PasswordHash: hash, Status: models.AgentStatusActive,
		Permissions: `{"member":"edit"}`}
	if err := agentAuthPostgres.CreateSubaccountRepository(database.DBConn, &sub); err != nil {
		t.Fatal(err)
	}
	tok := readyToken(t, app, models.AccountTypeSub, sub.ID, sub.Username)
	payment := map[string]any{"id": m.ID, "payment": map[string]any{"payment_type": "CREDIT"}}

	expect(t, call(t, app, "POST", memberDetailUpdatePath, payment, tok), 200, 402303)
	expect(t, call(t, app, "POST", memberDetailUpdatePath, map[string]any{"id": m.ID, "limit": map[string]any{"win_limit_per_day": 100}}, tok), 200, 200)
	expect(t, call(t, app, "POST", memberDetailUpdatePath, map[string]any{"id": m.ID, "payment": map[string]any{}}, tok), 200, 422) // รูปแบบก่อนสิทธิ์
	setCols(t, models.AccountTypeSub, sub.ID, map[string]any{"permissions": `{"member":"edit","payment":"edit"}`})
	expect(t, call(t, app, "POST", memberDetailUpdatePath, payment, tok), 200, 200)
	if getMember(t, m.ID).PaymentType != models.PaymentTypeCredit {
		t.Fatal("sub ที่มี payment edit ต้องแก้ได้")
	}
}

func TestMemberPaymentConcurrent(t *testing.T) { // MGMT-64: ยิงพร้อมกัน · แถวที่ยังไม่ commit ชน unique = 402405 ไม่ใช่ 409
	app := setup2(t)
	c := buildChain(t, app)

	var jobs []memberJob
	for i := 0; i < 6; i++ {
		b := memberBody(fmt.Sprintf("mempayrace%d", i), 0)
		withPayment("CASH", kbank("5556667778"))(b)
		jobs = append(jobs, memberJob{createMemberPath, b, c.agentTok})
	}
	if got := countCodes(t, runJobs(app, jobs), 200, 402405); got[200] != 1 {
		t.Fatalf("สร้างบัญชีเดียวกันพร้อมกัน %v", got)
	}

	// อีก tx ใส่แถวบัญชีเดียวกันไว้ยังไม่ commit → request ชน unique ใน DB หลัง commit
	target := createMember(t, app, c.agentTok, "mempaytarget", nil)
	hash, _ := utils.HashPassword(mgPassword)
	for _, tc := range []struct {
		name, user, number string
		job                memberJob
	}{
		{"สร้าง", "mempayext1", "9998887776", memberJob{createMemberPath, func() map[string]any {
			b := memberBody("mempayapi1", 0)
			withPayment("CASH", kbank("9998887776", "bank_code", "SCB"))(b)
			return b
		}(), c.agentTok}},
		{"แก้", "mempayext2", "9998887775", memberJob{memberDetailUpdatePath, map[string]any{"id": target.ID,
			"payment": map[string]any{"payment_account": kbank("9998887775", "bank_code", "SCB")}}, c.agentTok}},
	} {
		a := `{"bank_code": "SCB", "account_number": "` + tc.number + `"}` // แถวที่อีก tx ใส่ไว้แต่ยังไม่ commit
		o := models.UserMember{AgentID: c.agent.ID, Username: tc.user, PasswordHash: hash, Name: "ext", Currency: "THB",
			Status: models.AgentStatusActive, PaymentAccount: &a}
		tx := database.DBConn.Begin()
		if err := tx.Create(&o).Error; err != nil {
			tx.Rollback()
			t.Fatalf("%s: %v", tc.name, err)
		}
		done := make(chan int, 1)
		go func(j memberJob) { done <- callCode(app, j.path, j.body, j.tok) }(tc.job)
		time.Sleep(500 * time.Millisecond) // ให้ request ไปรอ lock ของ unique index
		if err := tx.Commit().Error; err != nil {
			t.Fatalf("%s: commit %v", tc.name, err)
		}
		if got := <-done; got != 402405 {
			t.Fatalf("%s: ได้ code %d, want 402405", tc.name, got)
		}
	}
}
