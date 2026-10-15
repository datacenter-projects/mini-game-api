//go:build integration

// API test ของ player_auth (ฝั่งผู้เล่น PA1 / PA3–PA7) ตามตาราง test case ใน docs/modules/player_auth.md หัวข้อ 7
// อยู่ในแพ็กเกจนี้เพื่อใช้ helper สร้างสาย / Member ร่วมกับ test หลังบ้าน (buildChain · createMember · setup2)
package backoffice_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"app/app/models"
	"app/pkg/configs"
	"app/pkg/testutil"
	"app/platform/database"

	"github.com/gofiber/fiber/v2"
)

const (
	plLoginPath    = "/api/v1/player/pb/login"
	plLogoutPath   = "/api/v1/player/pr/auth/logout"
	plPasswordPath = "/api/v1/player/pr/auth/password/update"
	plMePath       = "/api/v1/player/pr/me/get"
	plBalancePath  = "/api/v1/player/pr/balance/get"
	plGamesPath    = "/api/v1/player/pr/games/list"
)

// plEnv — สาย Superadmin → Company Transfer → Share B2C → Agent + Member mem01 (สร้างผ่าน BO · มีรหัส mgPassword)
type plEnv struct {
	app    *fiber.App
	c      chain
	member created
}

func plSetup(t *testing.T) plEnv {
	t.Helper()
	app := setup2(t)
	configs.Cfg.PlayerLobbyURL = "https://lobby.test/"
	c := buildChain(t, app)
	m := createMember(t, app, c.agentTok, "mem01", nil)
	return plEnv{app: app, c: c, member: m}
}

type plLoginData struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
	Member    struct {
		Username string `json:"username"`
		Name     string `json:"name"`
		Currency string `json:"currency"`
		Status   string `json:"status"`
		CanBet   bool   `json:"can_bet"`
	} `json:"member"`
}

func plLogin(t *testing.T, app *fiber.App, username, pw string) testutil.Response {
	t.Helper()
	return call(t, app, "POST", plLoginPath, map[string]any{"username": username, "password": pw}, "")
}

// plToken — login สำเร็จแล้วคืน token
func plToken(t *testing.T, app *fiber.App, username, pw string) string {
	t.Helper()
	r := plLogin(t, app, username, pw)
	expect(t, r, 200, 200)
	var d plLoginData
	if err := json.Unmarshal(r.Data, &d); err != nil {
		t.Fatal(err)
	}
	return d.Token
}

func plData(t *testing.T, r testutil.Response) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(r.Data, &m); err != nil {
		t.Fatalf("data ไม่ใช่ object: %s", r.Data)
	}
	return m
}

func plSetMemberStatus(t *testing.T, id uint, status models.AgentStatus) {
	t.Helper()
	if err := database.DBConn.Model(&models.UserMember{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		t.Fatal(err)
	}
}

func plSetAgentStatus(t *testing.T, id uint, status models.AgentStatus) {
	t.Helper()
	if err := database.DBConn.Model(&models.UserAgent{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		t.Fatal(err)
	}
}

func TestPlayerLoginSuccess(t *testing.T) { // PA-04, PA-25, PA-54
	e := plSetup(t)
	r := plLogin(t, e.app, "  Mem01 ", mgPassword) // PA-04: trim + ตัวเล็ก
	expect(t, r, 200, 200)
	var d plLoginData
	_ = json.Unmarshal(r.Data, &d)
	if d.Token == "" || d.ExpiresAt == "" {
		t.Fatalf("ต้องได้ token + expires_at: %s", r.Data)
	}
	if d.Member.Username != "mem01" || d.Member.Currency != "THB" || d.Member.Status != "ACTIVE" || !d.Member.CanBet {
		t.Errorf("member = %+v", d.Member)
	}
	raw := plData(t, r)["member"].(map[string]any)
	for _, banned := range []string{"credit", "id"} { // lead รอบ 3: ไม่ใส่ credit / id
		if _, ok := raw[banned]; ok {
			t.Errorf("member ต้องไม่มี %q", banned)
		}
	}
	if m := getMember(t, e.member.ID); m.LastLoginAt == nil || m.LastLoginIP == nil { // PA-25
		t.Error("ต้องบันทึก last_login_at / last_login_ip")
	}
}

func TestPlayerLoginRejects(t *testing.T) { // PA-03, PA-04, PA-31
	e := plSetup(t)
	expect(t, plLogin(t, e.app, "nobody", mgPassword), 401, 407201)
	expect(t, plLogin(t, e.app, "mem01", "wrong-pw-1A"), 401, 407201)

	// PA-03: ผู้เล่นที่ไม่มีรหัส login ตรงไม่ได้ — ตอบเหมือนรหัสผิด (ไม่ใช่ 407307)
	if err := database.DBConn.Model(&models.UserMember{}).Where("id = ?", e.member.ID).Update("password_hash", "").Error; err != nil {
		t.Fatal(err)
	}
	expect(t, plLogin(t, e.app, "mem01", mgPassword), 401, 407201)
	expect(t, plLogin(t, e.app, "mem01", ""), 200, 422) // ไม่ส่งรหัส = validation
}

func TestPlayerLoginStatusGate(t *testing.T) { // PA-31, PA-30
	e := plSetup(t)
	plSetMemberStatus(t, e.member.ID, models.AgentStatusLocked)
	expect(t, plLogin(t, e.app, "mem01", "wrong-pw-1A"), 401, 407201) // รหัสผิด ไม่บอกว่าล็อก
	expect(t, plLogin(t, e.app, "mem01", mgPassword), 403, 407301)

	plSetMemberStatus(t, e.member.ID, models.AgentStatusActive)
	plSetAgentStatus(t, e.c.share.ID, models.AgentStatusLocked) // สายบนล็อก
	expect(t, plLogin(t, e.app, "mem01", mgPassword), 403, 407302)

	plSetAgentStatus(t, e.c.share.ID, models.AgentStatusSuspended) // SUSPENDED login ได้ · can_bet = false
	r := plLogin(t, e.app, "mem01", mgPassword)
	expect(t, r, 200, 200)
	var d plLoginData
	_ = json.Unmarshal(r.Data, &d)
	if d.Member.Status != "SUSPENDED" || d.Member.CanBet {
		t.Errorf("SUSPENDED: status %q can_bet %v", d.Member.Status, d.Member.CanBet)
	}
}

func TestPlayerSingleSession(t *testing.T) { // PA-22
	e := plSetup(t)
	a := plToken(t, e.app, "mem01", mgPassword)
	expect(t, call(t, e.app, "POST", plMePath, nil, a), 200, 200)
	b := plToken(t, e.app, "mem01", mgPassword)
	expect(t, call(t, e.app, "POST", plMePath, nil, a), 401, 407204) // ถูก login จากที่อื่น
	expect(t, call(t, e.app, "POST", plMePath, nil, b), 200, 200)
}

func TestPlayerLogout(t *testing.T) { // PA-26
	e := plSetup(t)
	tok := plToken(t, e.app, "mem01", mgPassword)
	expect(t, call(t, e.app, "POST", plLogoutPath, nil, tok), 200, 200)
	expect(t, call(t, e.app, "POST", plLogoutPath, nil, tok), 200, 200) // ซ้ำได้
	expect(t, call(t, e.app, "POST", plMePath, nil, tok), 401, 407203)  // logout แล้วใช้ token เดิมไม่ได้
	expect(t, call(t, e.app, "POST", plLogoutPath, nil, "garbage"), 401, 407202)
	expect(t, call(t, e.app, "POST", plLogoutPath, nil, ""), 401, 407202)

	// logout ด้วย token เก่า (ถูกเตะแล้ว) ไม่กระทบ session ปัจจุบัน
	old := plToken(t, e.app, "mem01", mgPassword)
	cur := plToken(t, e.app, "mem01", mgPassword)
	expect(t, call(t, e.app, "POST", plLogoutPath, nil, old), 200, 200)
	expect(t, call(t, e.app, "POST", plMePath, nil, cur), 200, 200)
}

func TestPlayerTokenTypes(t *testing.T) { // PA-20
	e := plSetup(t)
	pl := plToken(t, e.app, "mem01", mgPassword)
	// token ผู้เล่นยิงเส้นหลังบ้าน → 401202 · token หลังบ้านยิงเส้นผู้เล่น → 407202
	expect(t, call(t, e.app, "POST", "/api/v1/bo/pr/auth/logout", nil, pl), 401, 401202)
	expect(t, call(t, e.app, "POST", plMePath, nil, e.c.agentTok), 401, 407202)
	expect(t, call(t, e.app, "POST", plMePath, nil, ""), 401, 407202)
	expect(t, call(t, e.app, "POST", plMePath, nil, "not.a.jwt"), 401, 407202)
}

func TestPlayerSessionTimeouts(t *testing.T) { // PA-21
	e := plSetup(t)
	t.Run("idle", func(t *testing.T) {
		configs.Cfg.Auth.PlayerSessionIdleTimeout = 2 * time.Second
		configs.Cfg.Auth.PlayerSessionAbsoluteTimeout = time.Hour
		tok := plToken(t, e.app, "mem01", mgPassword)
		time.Sleep(1200 * time.Millisecond)
		expect(t, call(t, e.app, "POST", plBalancePath, nil, tok), 200, 200) // balance/get = ping ต่ออายุ
		time.Sleep(1200 * time.Millisecond)
		expect(t, call(t, e.app, "POST", plBalancePath, nil, tok), 200, 200) // รวม 2.4s แต่ใช้งานต่อเนื่อง
		time.Sleep(2500 * time.Millisecond)
		expect(t, call(t, e.app, "POST", plBalancePath, nil, tok), 401, 407203)
	})
	t.Run("absolute", func(t *testing.T) {
		configs.Cfg.Auth.PlayerSessionIdleTimeout = 2 * time.Second
		configs.Cfg.Auth.PlayerSessionAbsoluteTimeout = 3 * time.Second
		tok := plToken(t, e.app, "mem01", mgPassword)
		for i := 0; i < 2; i++ {
			time.Sleep(1200 * time.Millisecond)
			expect(t, call(t, e.app, "POST", plMePath, nil, tok), 200, 200)
		}
		time.Sleep(1200 * time.Millisecond) // รวม 3.6s เกิน absolute แม้ใช้งานต่อเนื่อง
		r := call(t, e.app, "POST", plMePath, nil, tok)
		if r.Status != 401 || (r.Code != 407203 && r.Code != 407202) { // token หมดอายุพร้อม absolute
			t.Fatalf("ต้องหลุดเมื่อครบ absolute ได้ HTTP %d code %d", r.Status, r.Code)
		}
	})
}

func TestPlayerStatusMidSession(t *testing.T) { // PA-30, PA-32, PA-33, PA-34
	e := plSetup(t)

	t.Run("SUSPENDED ใช้ดูข้อมูลได้ can_bet false", func(t *testing.T) { // PA-33
		tok := plToken(t, e.app, "mem01", mgPassword)
		plSetMemberStatus(t, e.member.ID, models.AgentStatusSuspended)
		for _, p := range []string{plMePath, plBalancePath, plGamesPath} {
			expect(t, call(t, e.app, "POST", p, nil, tok), 200, 200)
		}
		r := call(t, e.app, "POST", plMePath, nil, tok)
		if d := plData(t, r); d["status"] != "SUSPENDED" || d["can_bet"] != false {
			t.Errorf("me = %v", d)
		}
		plSetMemberStatus(t, e.member.ID, models.AgentStatusActive)
	})

	t.Run("สายบน LOCKED ระหว่างใช้งาน", func(t *testing.T) { // PA-32
		tok := plToken(t, e.app, "mem01", mgPassword)
		plSetAgentStatus(t, e.c.com.ID, models.AgentStatusLocked)
		expect(t, call(t, e.app, "POST", plMePath, nil, tok), 403, 407302)
		plSetAgentStatus(t, e.c.com.ID, models.AgentStatusActive)
		expect(t, call(t, e.app, "POST", plMePath, nil, tok), 401, 407203) // session ถูกลบตอนปฏิเสธ
	})

	t.Run("ตัวเอง LOCKED ระหว่างใช้งาน", func(t *testing.T) {
		tok := plToken(t, e.app, "mem01", mgPassword)
		plSetMemberStatus(t, e.member.ID, models.AgentStatusLocked)
		expect(t, call(t, e.app, "POST", plMePath, nil, tok), 403, 407301)
		plSetMemberStatus(t, e.member.ID, models.AgentStatusActive)
	})

	t.Run("members/status/update LOCKED ลบ session หลัง commit", func(t *testing.T) { // PA-34
		tok := plToken(t, e.app, "mem01", mgPassword)
		r := call(t, e.app, "POST", memberStatusPath, map[string]any{"id": e.member.ID, "status": "LOCKED"}, e.c.agentTok)
		expect(t, r, 200, 200)
		expect(t, call(t, e.app, "POST", plMePath, nil, tok), 401, 407203) // session ถูกลบ ไม่ใช่รอเช็คสถานะ (ได้ 407203 ไม่ใช่ 407301)
		expect(t, plLogin(t, e.app, "mem01", mgPassword), 403, 407301)

		// ส่ง LOCKED ซ้ำ = 200 (ลองลบ session อีกรอบได้) · ตั้งกลับ ACTIVE แล้ว login ได้
		expect(t, call(t, e.app, "POST", memberStatusPath, map[string]any{"id": e.member.ID, "status": "LOCKED"}, e.c.agentTok), 200, 200)
		expect(t, call(t, e.app, "POST", memberStatusPath, map[string]any{"id": e.member.ID, "status": "ACTIVE"}, e.c.agentTok), 200, 200)
		expect(t, plLogin(t, e.app, "mem01", mgPassword), 200, 200)
	})
}

func TestPlayerCanBet(t *testing.T) { // PA-35
	e := plSetup(t)
	tok := plToken(t, e.app, "mem01", mgPassword)
	canBet := func() any { return plData(t, call(t, e.app, "POST", plMePath, nil, tok))["can_bet"] }
	if canBet() != true {
		t.Fatal("ACTIVE + win limit > 0 ต้อง can_bet")
	}
	if err := database.DBConn.Model(&models.UserMember{}).Where("id = ?", e.member.ID).Update("win_limit_per_day", 0).Error; err != nil {
		t.Fatal(err)
	}
	if canBet() != false {
		t.Error("win_limit_per_day = 0 (เล่นไม่ได้) ต้อง can_bet = false")
	}
}

func TestPlayerMeAndBalance(t *testing.T) { // PA-54, PA-56
	e := plSetup(t)
	tok := plToken(t, e.app, "mem01", mgPassword)

	me := plData(t, call(t, e.app, "POST", plMePath, nil, tok))
	for _, k := range []string{"username", "name", "currency", "status", "can_bet"} {
		if _, ok := me[k]; !ok {
			t.Errorf("me/get ขาด %q", k)
		}
	}
	if _, ok := me["credit"]; ok { // lead รอบ 3: เอา credit ออกจาก me/get
		t.Error("me/get ต้องไม่มี credit")
	}

	// balance/get ไม่ cache: เปลี่ยนยอดใน DB แล้วอ่านซ้ำได้ค่าใหม่ทันที (กฎข้อ 14)
	for _, credit := range []float64{0, 1250.5, 99.1234} {
		if err := database.DBConn.Model(&models.UserMember{}).Where("id = ?", e.member.ID).Update("credit", credit).Error; err != nil {
			t.Fatal(err)
		}
		r := call(t, e.app, "POST", plBalancePath, nil, tok)
		expect(t, r, 200, 200)
		d := plData(t, r)
		if d["currency"] != "THB" || d["credit"] != credit {
			t.Errorf("balance = %v want THB %v", d, credit)
		}
		if len(d) != 2 {
			t.Errorf("balance ต้องมีแค่ currency + credit: %v", d)
		}
	}
}

func TestPlayerGamesList(t *testing.T) { // PA-55, PA-60
	e := plSetup(t)
	tok := plToken(t, e.app, "mem01", mgPassword)

	type item struct {
		GameCode  string `json:"game_code"`
		NameTH    string `json:"name_th"`
		NameEN    string `json:"name_en"`
		Open      bool   `json:"open"`
		GameCover string `json:"game_cover"`
	}
	list := func() []item {
		r := call(t, e.app, "POST", plGamesPath, nil, tok)
		expect(t, r, 200, 200)
		var d struct {
			Items []item `json:"items"`
		}
		if err := json.Unmarshal(r.Data, &d); err != nil {
			t.Fatal(err)
		}
		return d.Items
	}

	items := list()
	if len(items) != 3 || items[0].GameCode != "scratch_card" || items[1].GameCode != "coin_toss" || items[2].GameCode != "rock_paper_scissors" {
		t.Fatalf("ต้องเรียงตาม catalog: %+v", items)
	}
	for _, it := range items {
		if it.NameTH == "" || it.NameEN == "" || !it.Open || it.GameCover != "" {
			t.Errorf("ค่าเริ่มต้นผิด: %+v", it) // ทุกเกมมีชื่อ 2 ภาษา · เปิด · ยังไม่มีปก = ""
		}
	}
	raw := plData(t, call(t, e.app, "POST", plGamesPath, nil, tok))["items"].([]any)[0].(map[string]any)
	if _, ok := raw["rate"]; ok {
		t.Error("games/list ต้องไม่ส่งค่า rate")
	}

	// ปิดเกมที่ Share (สายบน) → ผู้เล่นเห็น open = false ไม่ซ่อน (PA-60)
	if err := database.DBConn.Model(&models.AgentGameSetting{}).Where("agent_id = ? AND game_code = ?", e.c.share.ID, "coin_toss").
		Update("status_game", false).Error; err != nil {
		t.Fatal(err)
	}
	items = list()
	if len(items) != 3 {
		t.Fatalf("เกมปิดต้องไม่ถูกซ่อน: %+v", items)
	}
	for _, it := range items {
		if want := it.GameCode != "coin_toss"; it.Open != want {
			t.Errorf("%s open = %v want %v", it.GameCode, it.Open, want)
		}
	}
}

func TestPlayerLoginBruteForce(t *testing.T) { // PA-41
	e := plSetup(t)
	for i := 0; i < 5; i++ {
		expect(t, plLogin(t, e.app, "mem01", "wrong-pw-1A"), 401, 407201)
	}
	expect(t, plLogin(t, e.app, "mem01", mgPassword), 403, 407303) // ถูกระงับ แม้รหัสถูก
	// username ที่ไม่มีในระบบก็ถูกนับ / ระงับเหมือนกัน (ไม่เปิดเผยว่ามีบัญชีไหม)
	for i := 0; i < 5; i++ {
		expect(t, plLogin(t, e.app, "ghost", "wrong-pw-1A"), 401, 407201)
	}
	expect(t, plLogin(t, e.app, "ghost", "wrong-pw-1A"), 403, 407303)
}

func TestPlayerLoginFailCounterReset(t *testing.T) { // PA-41: login สำเร็จล้างตัวนับ
	e := plSetup(t)
	for i := 0; i < 4; i++ {
		expect(t, plLogin(t, e.app, "mem01", "wrong-pw-1A"), 401, 407201)
	}
	expect(t, plLogin(t, e.app, "mem01", mgPassword), 200, 200)
	for i := 0; i < 4; i++ {
		expect(t, plLogin(t, e.app, "mem01", "wrong-pw-1A"), 401, 407201)
	}
	expect(t, plLogin(t, e.app, "mem01", mgPassword), 200, 200) // ผิดรวม 8 แต่ถูกล้างกลางทาง
}

func TestPlayerLoginIPRateLimit(t *testing.T) { // PA-40
	e := plSetup(t)
	configs.Cfg.Auth.LoginIPLimit = 20
	for i := 1; i <= 20; i++ { // username ต่างกันทุกครั้ง ไม่ให้ไปโดน PA-41 ก่อน
		expect(t, plLogin(t, e.app, fmt.Sprintf("ghost%d", i), "x"), 401, 407201)
	}
	expect(t, plLogin(t, e.app, "ghost21", "x"), 429, 429)
}

func TestPlayerPasswordUpdate(t *testing.T) { // PA-50, PA-51, PA-41
	e := plSetup(t)
	tok := plToken(t, e.app, "mem01", mgPassword)
	const newPw = "Zx9!kq2m"
	update := func(old, nw string) testutil.Response {
		return call(t, e.app, "POST", plPasswordPath, map[string]any{"old_password": old, "new_password": nw}, tok)
	}

	expect(t, update("", newPw), 200, 422)
	expect(t, update(mgPassword, ""), 200, 422)
	expect(t, update(mgPassword, "short1"), 200, 422)               // ไม่ผ่าน AUTH-36
	expect(t, update(mgPassword, mgPassword), 200, 422)             // ซ้ำรหัสเดิม
	expect(t, update("wrong-pw-1A", newPw), 200, 407205)            // รหัสเดิมผิด
	expect(t, update(mgPassword, newPw), 200, 200)                  // สำเร็จ
	expect(t, call(t, e.app, "POST", plMePath, nil, tok), 200, 200) // session ปัจจุบันใช้ต่อ (Q10)
	expect(t, plLogin(t, e.app, "mem01", mgPassword), 401, 407201)  // รหัสเก่าใช้ไม่ได้
	expect(t, plLogin(t, e.app, "mem01", newPw), 200, 200)          // รหัสใหม่ใช้ได้
}

func TestPlayerPasswordUpdateBruteForce(t *testing.T) { // PA-41: รหัสเดิมผิดนับรวมกับ login
	e := plSetup(t)
	tok := plToken(t, e.app, "mem01", mgPassword)
	update := func(old string) testutil.Response {
		return call(t, e.app, "POST", plPasswordPath, map[string]any{"old_password": old, "new_password": "Zx9!kq2m"}, tok)
	}
	// ผสม login ผิด 2 + old_password ผิด 3 = 5 → ครั้งที่ 5 ได้ 407303 และตัด session
	expect(t, plLogin(t, e.app, "mem01", "wrong-pw-1A"), 401, 407201)
	expect(t, plLogin(t, e.app, "mem01", "wrong-pw-2B"), 401, 407201)
	// login สำเร็จล้างตัวนับ จึงต้องไม่ login สำเร็จคั่น — token เดิมยังใช้ได้เพราะไม่มี login ใหม่
	expect(t, update("wrong-pw-3C"), 200, 407205)
	expect(t, update("wrong-pw-4D"), 200, 407205)
	expect(t, update("wrong-pw-5E"), 403, 407303)
	expect(t, call(t, e.app, "POST", plMePath, nil, tok), 401, 407203) // session ถูกตัด
	expect(t, plLogin(t, e.app, "mem01", mgPassword), 403, 407303)     // ตัวนับ / บล็อกตัวเดียวกับ login
}

func TestPlayerPasswordUpdateNoPassword(t *testing.T) { // PA-51
	e := plSetup(t)
	tok := plToken(t, e.app, "mem01", mgPassword)
	if err := database.DBConn.Model(&models.UserMember{}).Where("id = ?", e.member.ID).Update("password_hash", "").Error; err != nil {
		t.Fatal(err)
	}
	r := call(t, e.app, "POST", plPasswordPath, map[string]any{"old_password": mgPassword, "new_password": "Zx9!kq2m"}, tok)
	expect(t, r, 200, 407307)
}

func TestPlayerNoNullBody(t *testing.T) { // ParseBodyNoNull
	e := plSetup(t)
	expect(t, call(t, e.app, "POST", plLoginPath, map[string]any{"username": nil, "password": "x"}, ""), 200, 422)
	expect(t, call(t, e.app, "POST", plLoginPath, map[string]any{"username": strings.Repeat("a", 101), "password": "x"}, ""), 200, 422)
}
