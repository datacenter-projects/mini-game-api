//go:build integration

// ฝาก / ถอนเงินให้ลูกตรง — docs/modules/deposit_withdraw.md หัวข้อ 7
package backoffice_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	depositWithdrawDto "app/app/internals/backoffice/dto/deposit_withdraw"
	"app/app/models"
	depositWithdrawPostgres "app/app/repository/postgres/deposit_withdraw"
	depositWithdrawRedis "app/app/repository/redis/deposit_withdraw"
	agentAuthService "app/app/service/agent_auth"
	depositWithdrawService "app/app/service/deposit_withdraw"
	"app/pkg/apperr"
	"app/platform/database"

	"github.com/gofiber/fiber/v2"
	goredis "github.com/redis/go-redis/v9"
)

const (
	dwCreatePath    = "/api/v1/bo/pr/deposit-withdraw/create"
	dwDownlinesPath = "/api/v1/bo/pr/deposit-withdraw/downlines/list"
	dwLogsPath      = "/api/v1/bo/pr/deposit-withdraw/logs/list"
)

func dwBody(role string, userID uint, typ string, amount float64) map[string]any {
	return map[string]any{"request_id": newRequestID(), "role": role, "user_id": userID, "currency": "THB",
		"amount": amount, "transaction_type": typ, "note": "", "passcode": passcode}
}

type dwResult struct {
	ID                 uint    `json:"id"`
	RequestID          string  `json:"request_id"`
	Username           string  `json:"username"`
	ParentUnlimited    bool    `json:"parent_unlimited"`
	ParentBalanceAfter float64 `json:"parent_balance_after"`
	TargetBalanceAfter float64 `json:"target_balance_after"`
	CreatedAt          string  `json:"created_at"`
}

func decodeDW(t *testing.T, raw json.RawMessage) dwResult {
	t.Helper()
	var d dwResult
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

func memberCredit(t *testing.T, id uint) float64 {
	t.Helper()
	var m models.UserMember
	database.DBConn.Select("credit").Where("id = ?", id).Take(&m)
	return m.Credit
}

func ledgerReasons(t *testing.T, requestID string) []string {
	t.Helper()
	var out []string
	database.DBConn.Model(&models.BalanceLedger{}).Where("request_id = ?", requestID).Order("id").Pluck("reason", &out)
	return out
}

func createMemberOf(t *testing.T, app *fiber.App, tok, username string) created {
	t.Helper()
	return createMemberWithPT(t, app, tok, username, 0)
}

func TestDWDepositWithdraw(t *testing.T) { // DW-12 · DW-13 · DW-23 · DW-25 · DW-26 · DW-33
	app := setup2(t)
	c := buildChain(t, app)
	setBalance(t, c.com.ID, "THB", 50000)

	b := dwBody("SHAREHOLDER", c.share.ID, "DEPOSIT", 10000)
	b["note"] = "  เติมรอบแรก  "
	r := call(t, app, "POST", dwCreatePath, b, c.comTok)
	expect(t, r, 200, 200)
	d := decodeDW(t, r.Data)
	if agentBalance(t, c.com.ID, "THB") != 40000 || agentBalance(t, c.share.ID, "THB") != 10000 ||
		d.ParentBalanceAfter != 40000 || d.TargetBalanceAfter != 10000 || d.ParentUnlimited {
		t.Fatalf("ฝาก: %+v", d)
	}
	var l models.DepositWithdrawLog
	database.DBConn.Where("id = ?", d.ID).Take(&l)
	if l.Note != "เติมรอบแรก" || l.ParentID != c.com.ID || l.TargetRole != "SHAREHOLDER" || l.ActorUsername != "comp01" {
		t.Fatalf("log %+v", l)
	}
	if got := ledgerReasons(t, d.RequestID); fmt.Sprint(got) != "[DEPOSIT_OUT DEPOSIT_IN]" {
		t.Fatalf("ledger %v", got)
	}
	if countRows(t, &models.CreateRequest{}, "request_id = ? AND target_type = 'DW' AND target_id = ?", d.RequestID, d.ID) != 1 {
		t.Fatal("ไม่มีแถว create_requests (DW-26)")
	}

	// กดซ้ำหลังจบ = ผลเดิม ไม่ย้ายเงินซ้ำ (DW-25)
	b["note"] = "เติมรอบแรก"
	r = call(t, app, "POST", dwCreatePath, b, c.comTok)
	expect(t, r, 200, 200)
	if again := decodeDW(t, r.Data); again != d || agentBalance(t, c.com.ID, "THB") != 40000 {
		t.Fatalf("ส่งซ้ำ %+v vs %+v", again, d)
	}
	// request_id เดิม ค่าไม่ตรง · บัญชีอื่นใช้ = 422
	b["amount"] = 1
	expect(t, call(t, app, "POST", dwCreatePath, b, c.comTok), 200, 422)
	b["amount"] = 10000
	expect(t, call(t, app, "POST", dwCreatePath, b, c.saTok), 200, 422)

	// ถอน
	w := dwBody("SHAREHOLDER", c.share.ID, "WITHDRAW", 4000)
	r = call(t, app, "POST", dwCreatePath, w, c.comTok)
	expect(t, r, 200, 200)
	if agentBalance(t, c.com.ID, "THB") != 44000 || agentBalance(t, c.share.ID, "THB") != 6000 {
		t.Fatalf("ถอน: com %v share %v", agentBalance(t, c.com.ID, "THB"), agentBalance(t, c.share.ID, "THB"))
	}
	if got := ledgerReasons(t, w["request_id"].(string)); fmt.Sprint(got) != "[WITHDRAW_OUT WITHDRAW_IN]" && fmt.Sprint(got) != "[WITHDRAW_IN WITHDRAW_OUT]" {
		t.Fatalf("ledger ถอน %v", got)
	}

	// ยอดไม่พอ (DW-12 / DW-13) · ไม่มี log / ledger
	over := dwBody("SHAREHOLDER", c.share.ID, "DEPOSIT", 44000.0001)
	expect(t, call(t, app, "POST", dwCreatePath, over, c.comTok), 200, 404304)
	expect(t, call(t, app, "POST", dwCreatePath, dwBody("SHAREHOLDER", c.share.ID, "WITHDRAW", 6000.0001), c.comTok), 200, 404306)
	if countRows(t, &models.BalanceLedger{}, "request_id = ?", over["request_id"]) != 0 ||
		countRows(t, &models.CreateRequest{}, "request_id = ?", over["request_id"]) != 0 {
		t.Fatal("รายการที่ไม่ผ่านต้องไม่มี ledger / create_requests")
	}

	// 0.1 + 0.2 = 0.3 (กฎข้อ 9)
	m := createMemberOf(t, app, c.shareTok, "memfloat")
	expect(t, call(t, app, "POST", dwCreatePath, dwBody("MEMBER", m.ID, "DEPOSIT", 0.1), c.shareTok), 200, 200)
	expect(t, call(t, app, "POST", dwCreatePath, dwBody("MEMBER", m.ID, "DEPOSIT", 0.2), c.shareTok), 200, 200)
	if memberCredit(t, m.ID) != 0.3 {
		t.Fatalf("credit = %v", memberCredit(t, m.ID))
	}
	// ถอนหมด
	expect(t, call(t, app, "POST", dwCreatePath, dwBody("MEMBER", m.ID, "WITHDRAW", 0.3), c.shareTok), 200, 200)
	if memberCredit(t, m.ID) != 0 {
		t.Fatalf("ถอนหมด credit = %v", memberCredit(t, m.ID))
	}
}

func TestDWSuperadmin(t *testing.T) { // DW-12 · DW-14 · DW-33 (DQ5) · DW-04 sub ของ Superadmin
	app := setup2(t)
	c := buildChain(t, app)

	r := call(t, app, "POST", dwCreatePath, dwBody("COMPANY", c.com.ID, "DEPOSIT", 1000000), c.saTok)
	expect(t, r, 200, 200)
	d := decodeDW(t, r.Data)
	if !d.ParentUnlimited || d.ParentBalanceAfter != 0 || agentBalance(t, c.com.ID, "THB") != 1000000 {
		t.Fatalf("Superadmin ฝาก %+v", d)
	}
	if got := ledgerReasons(t, d.RequestID); fmt.Sprint(got) != "[DEPOSIT_FROM_SUPERADMIN]" {
		t.Fatalf("ledger %v", got)
	}
	w := dwBody("COMPANY", c.com.ID, "WITHDRAW", 500)
	expect(t, call(t, app, "POST", dwCreatePath, w, c.saTok), 200, 200)
	if got := ledgerReasons(t, w["request_id"].(string)); fmt.Sprint(got) != "[WITHDRAW_TO_SUPERADMIN]" || agentBalance(t, c.com.ID, "THB") != 999500 {
		t.Fatalf("Superadmin ถอน %v", got)
	}

	sub := createSubAPI(t, app, c.saTok, "Pay", map[string]string{"payment": "edit"})
	subTok := readyToken(t, app, models.AccountTypeSub, sub.ID, sub.Username)
	r = call(t, app, "POST", dwCreatePath, dwBody("COMPANY", c.com.ID, "DEPOSIT", 1000), subTok)
	expect(t, r, 200, 200)
	var l models.DepositWithdrawLog
	database.DBConn.Where("id = ?", decodeDW(t, r.Data).ID).Take(&l)
	if l.ActorType != models.AccountTypeSub || l.ActorUsername != sub.Username || !l.ParentUnlimited {
		t.Fatalf("sub ของ Superadmin %+v", l)
	}
}

func TestDWRules(t *testing.T) { // DW-01 – DW-07 · DW-10 · DW-11 · DW-15
	app := setup2(t)
	c := buildChain(t, app)
	setBalance(t, c.share.ID, "THB", 1000)
	mem := createMemberOf(t, app, c.shareTok, "memrule")
	grand := createMemberOf(t, app, c.agentTok, "memgrand")

	expect(t, call(t, app, "POST", dwCreatePath, dwBody("MEMBER", grand.ID, "DEPOSIT", 1), c.shareTok), 200, 404401)        // หลาน
	expect(t, call(t, app, "POST", dwCreatePath, dwBody("MEMBER", c.agent.ID, "DEPOSIT", 1), c.shareTok), 200, 404401)      // id ของ agent แต่ role MEMBER
	expect(t, call(t, app, "POST", dwCreatePath, dwBody("COMPANY", c.agent.ID, "DEPOSIT", 1), c.shareTok), 200, 404401)     // role ไม่ตรง
	expect(t, call(t, app, "POST", dwCreatePath, dwBody("MEMBER", 999999, "DEPOSIT", 1), c.shareTok), 200, 404401)          // ไม่มี id
	expect(t, call(t, app, "POST", dwCreatePath, dwBody("SHAREHOLDER", c.share.ID, "DEPOSIT", 1), c.agentTok), 200, 404401) // ชั้นบน
	expect(t, call(t, app, "POST", dwCreatePath, dwBody("member", mem.ID, "DEPOSIT", 1), c.shareTok), 200, 422)

	// สถานะ: ฝากไม่ได้ · ถอนได้ (DQ1)
	expect(t, call(t, app, "POST", dwCreatePath, dwBody("MEMBER", mem.ID, "DEPOSIT", 300), c.shareTok), 200, 200)
	database.DBConn.Model(&models.UserMember{}).Where("id = ?", mem.ID).Update("status", models.AgentStatusLocked)
	expect(t, call(t, app, "POST", dwCreatePath, dwBody("MEMBER", mem.ID, "DEPOSIT", 1), c.shareTok), 200, 404301)
	expect(t, call(t, app, "POST", dwCreatePath, dwBody("MEMBER", mem.ID, "WITHDRAW", 100), c.shareTok), 200, 200)
	if memberCredit(t, mem.ID) != 200 || agentBalance(t, c.share.ID, "THB") != 800 {
		t.Fatalf("ถอนจากลูก LOCKED: credit %v share %v", memberCredit(t, mem.ID), agentBalance(t, c.share.ID, "THB"))
	}
	database.DBConn.Model(&models.UserAgent{}).Where("id = ?", c.agent.ID).Update("status", models.AgentStatusSuspended)
	expect(t, call(t, app, "POST", dwCreatePath, dwBody("AGENT", c.agent.ID, "DEPOSIT", 1), c.shareTok), 200, 404301)

	// Seamless (DW-06)
	reseller, _ := mustCreate(t, app, c.saTok, agentBody("COMPANY_SEAMLESS_RESELLER", "reseller01", nil, childPT(80, 0, 0, 0)))
	expect(t, call(t, app, "POST", dwCreatePath, dwBody("COMPANY", reseller.ID, "DEPOSIT", 1), c.saTok), 200, 404302)

	// สกุล (DW-11)
	j := dwBody("SHAREHOLDER", c.share.ID, "DEPOSIT", 1)
	j["currency"] = "USD"
	expect(t, call(t, app, "POST", dwCreatePath, j, c.comTok), 200, 404303)

	// รูปแบบ (DW-10 · DW-15)
	for _, v := range []any{0, -5, 1.00001, 1000000001} {
		b := dwBody("MEMBER", mem.ID, "WITHDRAW", 1)
		b["amount"] = v
		expect(t, call(t, app, "POST", dwCreatePath, b, c.shareTok), 200, 422)
	}
	b := dwBody("MEMBER", mem.ID, "WITHDRAW", 1)
	b["amount"] = "100"
	expect(t, call(t, app, "POST", dwCreatePath, b, c.shareTok), 200, 400)
	b = dwBody("MEMBER", mem.ID, "WITHDRAW", 1)
	b["note"] = nil
	expect(t, call(t, app, "POST", dwCreatePath, b, c.shareTok), 200, 422)
	b = dwBody("MEMBER", mem.ID, "WITHDRAW", 1)
	b["note"] = strings.Repeat("ก", 201)
	expect(t, call(t, app, "POST", dwCreatePath, b, c.shareTok), 200, 422)
	b = dwBody("MEMBER", mem.ID, "WITHDRAW", 1)
	b["transaction_type"] = "deposit"
	expect(t, call(t, app, "POST", dwCreatePath, b, c.shareTok), 200, 422)
	b = dwBody("MEMBER", mem.ID, "WITHDRAW", 1)
	delete(b, "passcode")
	expect(t, call(t, app, "POST", dwCreatePath, b, c.shareTok), 200, 422) // DW-07

	// ADMIN = 404305 ทุกเส้น (DW-03) · sub ไม่มี payment = 402303 (DW-04)
	admin := createAccount(t, "admin01", models.AgentRoleAdmin, nil)
	adminTok := loginReady(t, app, models.AccountTypeAgent, admin.ID, "admin01")
	expect(t, call(t, app, "POST", dwCreatePath, dwBody("COMPANY", c.com.ID, "DEPOSIT", 1), adminTok), 200, 404305)
	expect(t, call(t, app, "POST", dwDownlinesPath, map[string]any{}, adminTok), 200, 404305)
	expect(t, call(t, app, "POST", dwLogsPath, map[string]any{"date_from": "2026-10-01", "date_to": "2026-10-01"}, adminTok), 200, 404305)
	viewer := createSubAPI(t, app, c.shareTok, "Viewer", map[string]string{"member": "edit", "payment": "view"})
	viewerTok := readyToken(t, app, models.AccountTypeSub, viewer.ID, viewer.Username)
	expect(t, call(t, app, "POST", dwDownlinesPath, map[string]any{}, viewerTok), 200, 200)
	expect(t, call(t, app, "POST", dwCreatePath, dwBody("MEMBER", mem.ID, "WITHDRAW", 1), viewerTok), 200, 402303)
}

func TestDWFirstDepositCreatesRow(t *testing.T) { // DW-31
	app := setup2(t)
	c := buildChain(t, app)
	setBalance(t, c.share.ID, "THB", 100)
	if countRows(t, &models.AgentBalance{}, "agent_id = ?", c.agent.ID) != 0 {
		t.Skip("agent01 มีแถวยอดอยู่แล้ว")
	}
	expect(t, call(t, app, "POST", dwCreatePath, dwBody("AGENT", c.agent.ID, "DEPOSIT", 40), c.shareTok), 200, 200)
	if agentBalance(t, c.agent.ID, "THB") != 40 || agentBalance(t, c.share.ID, "THB") != 60 {
		t.Fatalf("ฝากครั้งแรก: agent %v share %v", agentBalance(t, c.agent.ID, "THB"), agentBalance(t, c.share.ID, "THB"))
	}
	// แม่ไม่มีแถวยอด ถอนแล้วสร้างแถวให้แม่
	database.DBConn.Where("agent_id = ?", c.share.ID).Delete(&models.AgentBalance{})
	expect(t, call(t, app, "POST", dwCreatePath, dwBody("AGENT", c.agent.ID, "WITHDRAW", 15), c.shareTok), 200, 200)
	if agentBalance(t, c.share.ID, "THB") != 15 {
		t.Fatalf("ถอนเข้าแม่ที่ไม่มีแถว = %v", agentBalance(t, c.share.ID, "THB"))
	}
}

func TestDWRequestIDCrossRoute(t *testing.T) { // DW-26 · lead DQ10
	app := setup2(t)
	c := buildChain(t, app)
	setBalance(t, c.share.ID, "THB", 100)

	mb := memberBody("memcross", 0)
	expect(t, call(t, app, "POST", createMemberPath, mb, c.shareTok), 200, 200)
	var m models.UserMember
	database.DBConn.Where("username = ?", "memcross").Take(&m)
	b := dwBody("MEMBER", m.ID, "DEPOSIT", 1)
	b["request_id"] = mb["request_id"]
	expect(t, call(t, app, "POST", dwCreatePath, b, c.shareTok), 200, 422)

	d := dwBody("MEMBER", m.ID, "DEPOSIT", 1)
	expect(t, call(t, app, "POST", dwCreatePath, d, c.shareTok), 200, 200)
	mb2 := memberBody("memcross2", 0)
	mb2["request_id"] = d["request_id"]
	expect(t, call(t, app, "POST", createMemberPath, mb2, c.shareTok), 200, 422)
	ab := agentBody("AGENT", "agentcross", nil, childPT(10, 0, 0, 0))
	ab["request_id"] = d["request_id"]
	expect(t, call(t, app, "POST", createAgentPath, ab, c.shareTok), 200, 422)
}

func TestDWLock(t *testing.T) { // DW-20 – DW-22 · DW-32
	app := setup2(t)
	c := buildChain(t, app)
	setBalance(t, c.com.ID, "THB", 1000)
	setBalance(t, c.share.ID, "THB", 1000)
	mem := createMemberOf(t, app, c.shareTok, "memlock")
	ctx := context.Background()

	// มีคนถือ lock อยู่ = 404307 ทันที
	database.DBRedis.Set(ctx, fmt.Sprintf("dw:lock:MEMBER:%d", mem.ID), "other", 10*time.Second)
	expect(t, call(t, app, "POST", dwCreatePath, dwBody("MEMBER", mem.ID, "DEPOSIT", 1), c.shareTok), 200, 404307)
	// compare-and-delete: token ไม่ตรงไม่ลบ
	if err := depositWithdrawRedis.ReleaseDWLockRepository(ctx, models.BalanceOwnerMember, mem.ID, "mine"); err != nil {
		t.Fatal(err)
	}
	if !redisExists(t, fmt.Sprintf("dw:lock:MEMBER:%d", mem.ID)) {
		t.Fatal("ปลด lock ของคนอื่น")
	}
	database.DBRedis.Del(ctx, fmt.Sprintf("dw:lock:MEMBER:%d", mem.ID))

	// ยิงพร้อมกันที่ลูกเดียวกัน: สำเร็จ + 404307 · ยอดตรงกับจำนวนที่สำเร็จ
	var jobs []memberJob
	for i := 0; i < 6; i++ {
		jobs = append(jobs, memberJob{dwCreatePath, dwBody("MEMBER", mem.ID, "DEPOSIT", 10), c.shareTok})
	}
	got := countCodes(t, runJobs(app, jobs), 200, 404307)
	if memberCredit(t, mem.ID) != float64(got[200]*10) || agentBalance(t, c.share.ID, "THB") != 1000-float64(got[200]*10) {
		t.Fatalf("พร้อมกัน %v: credit %v share %v", got, memberCredit(t, mem.ID), agentBalance(t, c.share.ID, "THB"))
	}

	// lock แถว share01 ร่วมกัน (comp→share · share→agent) ไม่ deadlock
	jobs = nil
	for i := 0; i < 5; i++ {
		jobs = append(jobs, memberJob{dwCreatePath, dwBody("SHAREHOLDER", c.share.ID, "DEPOSIT", 1), c.comTok},
			memberJob{dwCreatePath, dwBody("AGENT", c.agent.ID, "DEPOSIT", 1), c.shareTok})
	}
	countCodes(t, runJobs(app, jobs), 200, 404307)

	// request_id เดียวกัน ลูกต่างกัน (lock คนละ key): ย้ายเงินครั้งเดียว อีกอัน 422 (DW-27)
	rid := newRequestID()
	a := dwBody("MEMBER", mem.ID, "DEPOSIT", 1)
	bb := dwBody("AGENT", c.agent.ID, "DEPOSIT", 1)
	a["request_id"], bb["request_id"] = rid, rid
	got = countCodes(t, runJobs(app, []memberJob{{dwCreatePath, a, c.shareTok}, {dwCreatePath, bb, c.shareTok}}), 200, 422)
	if got[200] != 1 || countRows(t, &models.DepositWithdrawLog{}, "request_id = ?", rid) != 1 {
		t.Fatalf("request_id เดียวกันพร้อมกัน %v", got)
	}
}

func TestDWRedisDown(t *testing.T) { // DW-21 fail closed — เรียก service ตรง (Redis ล่มทำให้ Authenticated ล้มก่อนถึง lock)
	app := setup2(t)
	c := buildChain(t, app)
	_ = app
	old := database.DBRedis
	database.DBRedis = goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:1", DialTimeout: 200 * time.Millisecond, MaxRetries: -1})
	defer func() { _ = database.DBRedis.Close(); database.DBRedis = old }()

	req := depositWithdrawDto.CreateRequest{RequestID: newRequestID(), Role: "AGENT", UserID: c.agent.ID, Currency: "THB", Amount: 1, TransactionType: "DEPOSIT"}
	if err := req.Validate(); err != nil {
		t.Fatal(err)
	}
	actor := agentAuthService.Actor{AccountType: models.AccountTypeAgent, AgentID: c.share.ID, Role: models.AgentRoleShareholder, Username: "share01"}
	_, err := depositWithdrawService.CreateDepositWithdrawService(context.Background(), actor, req, agentAuthService.RequestMeta{})
	if !errors.Is(err, apperr.ErrDWLockUnavailable) || apperr.From(err).HTTPStatus != 503 {
		t.Fatalf("Redis ล่ม = %v", err)
	}
}

func TestDWBalanceConstraint(t *testing.T) { // DW-34
	setup2(t)
	var names []string
	database.DBConn.Raw(`SELECT conname FROM pg_constraint WHERE conname IN ('agent_balances_amount_check', 'user_members_credit_check') ORDER BY conname`).Scan(&names)
	if fmt.Sprint(names) != "[agent_balances_amount_check user_members_credit_check]" {
		t.Fatalf("ชื่อ CHECK = %v", names)
	}
	a := createAgent(t, "cknegative", models.AgentStatusActive, nil)
	setBalance(t, a.ID, "THB", 1)
	err := depositWithdrawPostgres.UpdateDWAgentBalanceRepository(database.DBConn, a.ID, "THB", -1, time.Now())
	if !errors.Is(err, depositWithdrawPostgres.ErrBalanceNegative) {
		t.Fatalf("ยอดติดลบ = %v", err)
	}
}

func TestDWDownlinesList(t *testing.T) { // DW-40 – DW-44
	app := setup2(t)
	c := buildChain(t, app)
	b2b, _ := mustCreate(t, app, c.comTok, agentBody("SHARE_B2B", "shareb2b", []string{"THB", "USD"}, childPT(50, 0, 0, 0)))
	setBalance(t, b2b.ID, "THB", 7)
	createMemberOf(t, app, c.shareTok, "memlist")
	createMemberOf(t, app, c.agentTok, "memlistgrand")

	type row struct {
		ID         uint                             `json:"id"`
		Role       string                           `json:"role"`
		Username   string                           `json:"username"`
		HasBalance bool                             `json:"has_balance"`
		Balances   []depositWithdrawDto.BalanceView `json:"balances"`
	}
	type page struct {
		TotalCount int64 `json:"total_count"`
		Data       []row `json:"data"`
	}
	list := func(tok string, body map[string]any) page {
		r := call(t, app, "POST", dwDownlinesPath, body, tok)
		expect(t, r, 200, 200)
		var p page
		_ = json.Unmarshal(r.Data, &p)
		return p
	}

	p := list(c.shareTok, map[string]any{})
	if p.TotalCount != 2 || p.Data[0].Username != "agent01" || p.Data[1].Username != "memlist" || p.Data[1].Role != "MEMBER" {
		t.Fatalf("share01 = %+v", p)
	}
	if p = list(c.shareTok, map[string]any{"keyword": "MEML"}); p.TotalCount != 1 || p.Data[0].Username != "memlist" {
		t.Fatalf("keyword = %+v", p)
	}
	if p = list(c.shareTok, map[string]any{"keyword": "grand"}); p.TotalCount != 0 {
		t.Fatalf("keyword ไม่ลงหลาน = %+v", p)
	}
	expect(t, call(t, app, "POST", dwDownlinesPath, map[string]any{"keyword": "ab"}, c.shareTok), 200, 422)

	p = list(c.comTok, map[string]any{})
	for _, r := range p.Data {
		if r.Username == "shareb2b" && fmt.Sprint(r.Balances) != "[{THB 7} {USD 0}]" {
			t.Fatalf("balances หลายสกุล %+v", r.Balances)
		}
	}
	mustCreate(t, app, c.saTok, agentBody("COMPANY_SEAMLESS_RESELLER", "reseller01", nil, childPT(80, 0, 0, 0)))
	for _, r := range list(c.saTok, map[string]any{}).Data {
		if (r.Username == "reseller01") == r.HasBalance {
			t.Fatalf("has_balance %+v", r)
		}
	}
}

func TestDWLogsList(t *testing.T) { // DW-46 – DW-48
	app := setup2(t)
	c := buildChain(t, app)
	setBalance(t, c.com.ID, "THB", 100)
	setBalance(t, c.share.ID, "THB", 100)
	expect(t, call(t, app, "POST", dwCreatePath, dwBody("SHAREHOLDER", c.share.ID, "DEPOSIT", 10), c.comTok), 200, 200)
	expect(t, call(t, app, "POST", dwCreatePath, dwBody("AGENT", c.agent.ID, "DEPOSIT", 5), c.shareTok), 200, 200)
	expect(t, call(t, app, "POST", dwCreatePath, dwBody("AGENT", c.agent.ID, "WITHDRAW", 2), c.shareTok), 200, 200)

	today := time.Now().In(time.FixedZone("Asia/Bangkok", 7*3600)).Format("2006-01-02")
	logs := func(tok string, body map[string]any) int64 {
		body["date_from"], body["date_to"] = today, today
		r := call(t, app, "POST", dwLogsPath, body, tok)
		expect(t, r, 200, 200)
		var p struct {
			TotalCount int64 `json:"total_count"`
		}
		_ = json.Unmarshal(r.Data, &p)
		return p.TotalCount
	}
	if n := logs(c.shareTok, map[string]any{}); n != 2 {
		t.Fatalf("share01 เห็น %d", n)
	}
	if n := logs(c.comTok, map[string]any{}); n != 1 {
		t.Fatalf("comp01 เห็น %d (ต้องไม่เห็นของ share01)", n)
	}
	if n := logs(c.shareTok, map[string]any{"transaction_type": "WITHDRAW"}); n != 1 {
		t.Fatalf("กรองถอน %d", n)
	}
	if n := logs(c.shareTok, map[string]any{"role": "AGENT", "user_id": c.agent.ID}); n != 2 {
		t.Fatalf("กรองลูก %d", n)
	}
	for _, body := range []map[string]any{
		{"date_from": "2026-10-14", "date_to": "2026-10-01"},
		{"date_from": "2026-10-01", "date_to": "2026-11-01"},
		{"date_from": "2026/10/01", "date_to": "2026-10-01"},
		{"date_from": "2026-10-01", "date_to": "2026-10-01", "role": "AGENT"},
	} {
		expect(t, call(t, app, "POST", dwLogsPath, body, c.shareTok), 200, 422)
	}
}
