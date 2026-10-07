//go:build integration

// API test ของ Profile ตามตาราง test case ใน docs/modules/account.md หัวข้อ 7 (ACC-11–ACC-16, ACC-30, ACC-32)
package backoffice_test

import (
	"encoding/json"
	"testing"
	"time"

	"app/app/models"

	"github.com/gofiber/fiber/v2"
)

const profilePath = "/api/v1/bo/pr/account/profile"

type profileData struct {
	Username      string            `json:"username"`
	Role          string            `json:"role"`
	UserType      string            `json:"user_type"`
	Status        string            `json:"status"`
	IsSubaccount  bool              `json:"is_subaccount"`
	OwnerUsername string            `json:"owner_username"`
	PasscodeSet   bool              `json:"passcode_set"`
	LastLoginAt   string            `json:"last_login_at"`
	LastLoginIP   string            `json:"last_login_ip"`
	CreatedAt     string            `json:"created_at"`
	Currencies    []string          `json:"currencies"`
	Balances      []json.RawMessage `json:"balances"`
	PT            map[string]any    `json:"pt"`
	StatusGame    map[string]bool   `json:"status_game"`
	Permissions   map[string]string `json:"permissions"`
}

func getProfile(t *testing.T, app *fiber.App, tok string) (profileData, map[string]json.RawMessage) {
	t.Helper()
	r := call(t, app, "GET", profilePath, nil, tok)
	expect(t, r, 200, 200)
	var d profileData
	if err := json.Unmarshal(r.Data, &d); err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(r.Data, &raw); err != nil {
		t.Fatal(err)
	}
	return d, raw
}

// ACC-32: ไม่มี null ใน response · ส่วนของ module ② ใช้ค่าชั่วคราว (สกุลครบ 27 ยอด 0.00 · สิทธิ์ off)
func assertNoNull(t *testing.T, raw map[string]json.RawMessage, d profileData) {
	t.Helper()
	for k, v := range raw {
		if string(v) == "null" {
			t.Fatalf("field %s เป็น null", k)
		}
	}
	for k, want := range map[string]string{"pt": "{}", "status_game": "{}"} {
		if string(raw[k]) != want {
			t.Fatalf("field %s = %s, want %s", k, raw[k], want)
		}
	}
	if d.Role == "ADMIN" {
		if len(d.Currencies) != 0 || len(d.Balances) != 0 || len(d.Permissions) != 0 {
			t.Fatalf("ADMIN ต้องไม่มีสกุล ยอด และสิทธิ์ %+v", d)
		}
	} else {
		if len(d.Currencies) != 27 || len(d.Balances) != 27 || string(d.Balances[0]) != `{"currency":"ARS","amount":0.00}` {
			t.Fatalf("ต้องมี 27 สกุล ยอด 0.00 ได้ %v %s", d.Currencies, d.Balances[0])
		}
		if len(d.Permissions) != 9 {
			t.Fatalf("ต้องมีสิทธิ์ 9 เมนู ได้ %v", d.Permissions)
		}
		for m, lv := range d.Permissions {
			if lv != "off" {
				t.Fatalf("เมนู %s ต้องเป็น off ได้ %s", m, lv)
			}
		}
	}
	if _, ok := raw["effective_status"]; ok {
		t.Fatal("ต้องไม่มี field effective_status (ACC-30)")
	}
}

func TestProfileOwnAccount(t *testing.T) { // ACC-11, ACC-12, ACC-14, ACC-32
	app := setup2(t)
	super := createAccount(t, "root", models.AgentRoleSuperAdmin, nil)
	admin := createAccount(t, "support01", models.AgentRoleAdmin, nil)
	company := createAccount(t, "comp01", models.AgentRoleCompany, &super.ID)
	a := createAgent(t, "agent01", models.AgentStatusActive, &company.ID)

	for _, acc := range []struct {
		id                   uint
		user, role, userType string
	}{
		{a.ID, "agent01", "AGENT", "AGENT"},
		{super.ID, "root", "SUPERADMIN", "SUPERADMIN"},
		{admin.ID, "support01", "ADMIN", "ADMIN"},
		{company.ID, "comp01", "COMPANY", ""}, // ประเภทย่อยรอ module ②
	} {
		tok := loginReady(t, app, models.AccountTypeAgent, acc.id, acc.user)
		d, raw := getProfile(t, app, tok)
		if d.Username != acc.user || d.Role != acc.role || d.UserType != acc.userType || d.IsSubaccount ||
			d.OwnerUsername != "" || !d.PasscodeSet || d.Status != "ACTIVE" {
			t.Fatalf("%s: unexpected %+v", acc.user, d)
		}
		at, err := time.Parse(time.RFC3339Nano, d.LastLoginAt)
		if err != nil || time.Since(at) > time.Minute || d.LastLoginIP == "" || d.CreatedAt == "" {
			t.Fatalf("%s: login ล่าสุด / วันที่สร้างต้องมีค่า %+v", acc.user, d)
		}
		assertNoNull(t, raw, d)
	}
}

func TestProfileSubaccount(t *testing.T) { // ACC-11, ACC-12, ACC-15, ACC-30
	app := setup2(t)
	owner := createAgent(t, "agent01", models.AgentStatusActive, nil)
	sub := createSub(t, owner, "staff", models.AgentStatusActive)
	tok := loginReady(t, app, models.AccountTypeSub, sub.ID, "agent01@staff")

	d, raw := getProfile(t, app, tok)
	if d.Username != "agent01@staff" || !d.IsSubaccount || d.OwnerUsername != "agent01" || d.Role != "AGENT" || d.UserType != "AGENT" {
		t.Fatalf("unexpected %+v", d)
	}
	assertNoNull(t, raw, d)

	// ผู้สร้างถูกระงับ → status ของ sub = สถานะที่ใช้งานจริง และยังเปิด Profile ได้ (ACC-31 / AUTH-54)
	setStatus(t, owner.ID, models.AgentStatusSuspended)
	d, _ = getProfile(t, app, tok)
	if d.Status != "SUSPENDED" {
		t.Fatalf("ผู้สร้าง SUSPENDED: status ของ sub ต้องเป็น SUSPENDED ได้ %+v", d)
	}
}

func TestProfileRequiresGates(t *testing.T) { // ACC-11
	app := setup2(t)
	createAgent(t, "agent01", models.AgentStatusActive, nil)
	tok := loginToken(t, app, "agent01") // ยังไม่ตั้ง passcode
	expect(t, call(t, app, "GET", profilePath, nil, tok), 403, 401304)
	expect(t, call(t, app, "GET", profilePath, nil, ""), 401, 401202)
}
