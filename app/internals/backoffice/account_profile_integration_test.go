//go:build integration

// API test ของ Profile ตามตาราง test case ใน docs/modules/account.md หัวข้อ 7 (ACC-11–ACC-14)
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
	Username        string     `json:"username"`
	Role            string     `json:"role"`
	Status          string     `json:"status"`
	EffectiveStatus string     `json:"effective_status"`
	IsSubaccount    bool       `json:"is_subaccount"`
	OwnerUsername   *string    `json:"owner_username"`
	PasscodeSet     bool       `json:"passcode_set"`
	LastLoginAt     *time.Time `json:"last_login_at"`
	LastLoginIP     *string    `json:"last_login_ip"`
	CreatedAt       time.Time  `json:"created_at"`
}

func getProfile(t *testing.T, app *fiber.App, tok string) profileData {
	t.Helper()
	r := call(t, app, "GET", profilePath, nil, tok)
	expect(t, r, 200, 200)
	var d profileData
	if err := json.Unmarshal(r.Data, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestProfileOwnAccount(t *testing.T) { // ACC-11, ACC-12, ACC-14
	app := setup2(t)
	super := createAccount(t, "root", models.AgentRoleSuperAdmin, nil)
	admin := createAccount(t, "support01", models.AgentRoleAdmin, nil)
	a := createAgent(t, "agent01", models.AgentStatusActive, &super.ID)

	for _, acc := range []struct {
		id         uint
		user, role string
	}{
		{a.ID, "agent01", "AGENT"},
		{super.ID, "root", "SUPERADMIN"},
		{admin.ID, "support01", "ADMIN"},
	} {
		tok := loginReady(t, app, models.AccountTypeAgent, acc.id, acc.user)
		d := getProfile(t, app, tok)
		if d.Username != acc.user || d.Role != acc.role || d.IsSubaccount || d.OwnerUsername != nil ||
			!d.PasscodeSet || d.Status != "ACTIVE" || d.EffectiveStatus != "ACTIVE" {
			t.Fatalf("%s: unexpected %+v", acc.user, d)
		}
		if d.LastLoginAt == nil || time.Since(*d.LastLoginAt) > time.Minute || d.LastLoginIP == nil || d.CreatedAt.IsZero() {
			t.Fatalf("%s: login ล่าสุด / วันที่สร้างต้องมีค่า %+v", acc.user, d)
		}
	}
}

func TestProfileSubaccount(t *testing.T) { // ACC-11, ACC-12
	app := setup2(t)
	owner := createAgent(t, "agent01", models.AgentStatusActive, nil)
	sub := createSub(t, owner, "staff", models.AgentStatusActive)
	tok := loginReady(t, app, models.AccountTypeSub, sub.ID, "agent01@staff")

	d := getProfile(t, app, tok)
	if d.Username != "agent01@staff" || !d.IsSubaccount || d.OwnerUsername == nil || *d.OwnerUsername != "agent01" || d.Role != "AGENT" {
		t.Fatalf("unexpected %+v", d)
	}

	setStatus(t, owner.ID, models.AgentStatusSuspended)
	d = getProfile(t, app, tok)
	if d.Status != "ACTIVE" || d.EffectiveStatus != "SUSPENDED" {
		t.Fatalf("ผู้สร้าง SUSPENDED: status ของ sub ต้อง ACTIVE effective ต้อง SUSPENDED ได้ %+v", d)
	}
}

func TestProfileRequiresGates(t *testing.T) { // ACC-11
	app := setup2(t)
	createAgent(t, "agent01", models.AgentStatusActive, nil)
	tok := loginToken(t, app, "agent01") // ยังไม่ตั้ง passcode
	expect(t, call(t, app, "GET", profilePath, nil, tok), 403, 401304)
	expect(t, call(t, app, "GET", profilePath, nil, ""), 401, 401202)
}
