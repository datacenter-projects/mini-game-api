//go:build integration

// API test ของเส้น admin reset (AUTH-44 – AUTH-50 · module admin_management) — helper อยู่ใน agent_auth_phase2_integration_test.go
package backoffice_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	agentAuthCore "app/app/core/agent_auth"
	"app/app/models"
	"app/platform/database"
)

func TestAdminResetAccess(t *testing.T) {
	app := setup2(t)
	admin := createAccount(t, "support01", models.AgentRoleAdmin, nil)
	createAccount(t, "support02", models.AgentRoleAdmin, nil)
	super := createAccount(t, "root", models.AgentRoleSuperAdmin, nil)
	company := createAccount(t, "company01", models.AgentRoleCompany, nil)
	agent := createAgent(t, "agent01", models.AgentStatusActive, &company.ID)
	createSub(t, agent, "staff", models.AgentStatusActive)
	createAgent(t, "nopasscode", models.AgentStatusActive, nil)
	createAgent(t, "locked", models.AgentStatusLocked, nil)
	setPasscode(t, models.AccountTypeAgent, agent.ID, passcode)

	adminTok := loginReady(t, app, models.AccountTypeAgent, admin.ID, "support01")
	superTok := loginReady(t, app, models.AccountTypeAgent, super.ID, "root")
	agentTok := loginToken(t, app, "agent01")
	reset := func(username string) map[string]string {
		return map[string]string{"username": username, "passcode": passcode}
	}

	expect(t, call(t, app, "POST", resetPasswordPath, reset("agent01"), agentTok), 403, 401308)
	expect(t, call(t, app, "POST", resetPasswordPath, reset("agent01"), superTok), 403, 401308)
	expect(t, call(t, app, "POST", resetPasswordPath, map[string]string{"username": "agent01", "passcode": "000000"}, adminTok), 200, 401204)

	tests := []struct {
		name, path, username string
		code                 int
	}{
		{"ไม่พบ", resetPasswordPath, "ghost", 401404},
		{"ไม่พบ sub", resetPasswordPath, "agent01@nobody", 401404},
		{"SUPERADMIN", resetPasswordPath, "root", 401406},
		{"ADMIN คนอื่น", resetPasswordPath, "support02", 401406},
		{"ตัวเอง", resetPasscodePath, "support01", 401406},
		{"เป้าหมาย LOCKED", resetPasswordPath, "locked", 401407},
		{"ยังไม่ได้ตั้ง passcode", resetPasscodePath, "nopasscode", 401405},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expect(t, call(t, app, "POST", tt.path, reset(tt.username), adminTok), 200, tt.code)
		})
	}

	t.Run("sub ที่ผู้สร้างถูก LOCK", func(t *testing.T) {
		setStatus(t, agent.ID, models.AgentStatusLocked)
		defer setStatus(t, agent.ID, models.AgentStatusActive)
		expect(t, call(t, app, "POST", resetPasswordPath, reset("agent01@staff"), adminTok), 200, 401407)
	})

	var n int64
	database.DBConn.Table("auth_audit_logs").Count(&n)
	if n != 0 {
		t.Fatalf("รีเซ็ตที่ไม่สำเร็จต้องไม่มี audit log ได้ %d", n)
	}
}

func TestAdminResetPasscode(t *testing.T) { // AUTH-45–AUTH-47, AUTH-49
	app := setup2(t)
	admin := createAccount(t, "support01", models.AgentRoleAdmin, nil)
	owner := createAgent(t, "agent01", models.AgentStatusActive, nil)
	sub := createSub(t, owner, "staff", models.AgentStatusActive)
	adminTok := loginReady(t, app, models.AccountTypeAgent, admin.ID, "support01")
	subTok := loginReady(t, app, models.AccountTypeSub, sub.ID, "agent01@staff")

	for i := 0; i < 4; i++ { // ให้มีตัวนับค้างไว้
		expect(t, call(t, app, "POST", txPath, map[string]string{"passcode": "000000"}, subTok), 200, 401204)
	}

	r := call(t, app, "POST", resetPasscodePath, map[string]string{"username": "Agent01@Staff", "passcode": passcode}, adminTok)
	expect(t, r, 200, 200)
	if got := r.Header.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
	var d struct {
		Username      string    `json:"username"`
		TempPasscode  string    `json:"temp_passcode"`
		TempExpiresAt time.Time `json:"temp_expires_at"`
	}
	_ = json.Unmarshal(r.Data, &d)
	if d.Username != "agent01@staff" || !agentAuthCore.IsValidPasscode(d.TempPasscode) || d.TempPasscode == passcode {
		t.Fatalf("unexpected %+v", d)
	}
	if until := time.Until(d.TempExpiresAt); until < 23*time.Hour || until > 24*time.Hour {
		t.Fatalf("ค่าชั่วคราวต้องหมดอายุใน 24 ชม. ได้ %v", until)
	}
	expect(t, call(t, app, "GET", gatedPath, nil, subTok), 401, 401203) // session ของเป้าหมายถูกลบ

	ld := loginAs(t, app, "agent01@staff", password)
	if !ld.MustChangePasscode {
		t.Fatal("ต้องถูกบังคับเปลี่ยน passcode")
	}
	expect(t, call(t, app, "GET", gatedPath, nil, ld.Token), 403, 401307)
	// ตัวนับเดิม 4 ครั้งต้องถูกล้างตอนรีเซ็ต — ผิดอีก 1 ครั้งยังไม่ถูกตัด
	expect(t, call(t, app, "POST", passcodeChangePath, map[string]string{"old_passcode": "000000", "new_passcode": "654321", "confirm_passcode": "654321"}, ld.Token), 200, 401204)
	expect(t, call(t, app, "POST", passcodeChangePath, map[string]string{"old_passcode": d.TempPasscode, "new_passcode": d.TempPasscode, "confirm_passcode": d.TempPasscode}, ld.Token), 200, 401403)
	expect(t, call(t, app, "POST", passcodeChangePath, map[string]string{"old_passcode": d.TempPasscode, "new_passcode": "654321", "confirm_passcode": "654321"}, ld.Token), 200, 200)
	expect(t, call(t, app, "GET", gatedPath, nil, ld.Token), 200, 200)

	logs := auditRows(t, models.AuthAuditResetPasscode)
	if len(logs) != 1 || logs[0].ActorType != models.AuthAuditActorAgent || *logs[0].ActorID != admin.ID ||
		*logs[0].ActorUsername != "support01" || *logs[0].TargetType != models.AccountTypeSub || *logs[0].TargetID != sub.ID ||
		logs[0].TargetUsername != "agent01@staff" || logs[0].IP == nil || logs[0].RequestID == nil {
		t.Fatalf("audit log ไม่ถูกต้อง %+v", logs)
	}
	if n := len(auditRows(t, models.AuthAuditPasscodeChange)); n != 1 {
		t.Fatalf("เปลี่ยน passcode หลังถูกรีเซ็ตต้องมี audit 1 แถว ได้ %d", n)
	}
}

func TestAdminResetPasscodeClearsBlock(t *testing.T) { // AUTH-47
	app := setup2(t)
	admin := createAccount(t, "support01", models.AgentRoleAdmin, nil)
	a := createAgent(t, "agent01", models.AgentStatusActive, nil)
	adminTok := loginReady(t, app, models.AccountTypeAgent, admin.ID, "support01")
	tok := loginReady(t, app, models.AccountTypeAgent, a.ID, "agent01")

	for i := 0; i < 5; i++ {
		call(t, app, "POST", txPath, map[string]string{"passcode": "000000"}, tok)
	}
	expect(t, login(t, app, "agent01", password), 403, 401309)
	expect(t, call(t, app, "POST", resetPasscodePath, map[string]string{"username": "agent01", "passcode": passcode}, adminTok), 200, 200)
	if !loginAs(t, app, "agent01", password).MustChangePasscode {
		t.Fatal("ต้องถูกบังคับเปลี่ยน passcode")
	}
}

func TestAdminResetPassword(t *testing.T) { // AUTH-46, AUTH-48, AUTH-50
	app := setup2(t)
	admin := createAccount(t, "support01", models.AgentRoleAdmin, nil)
	a := createAgent(t, "agent01", models.AgentStatusActive, nil)
	adminTok := loginReady(t, app, models.AccountTypeAgent, admin.ID, "support01")
	setPasscode(t, models.AccountTypeAgent, a.ID, passcode)

	for i := 0; i < 5; i++ { // ถูกบล็อก login (AUTH-10)
		login(t, app, "agent01", "wrong")
	}
	expect(t, login(t, app, "agent01", password), 403, 401303)

	r := call(t, app, "POST", resetPasswordPath, map[string]string{"username": "agent01", "passcode": passcode}, adminTok)
	expect(t, r, 200, 200)
	var d struct {
		TempPassword string `json:"temp_password"`
	}
	_ = json.Unmarshal(r.Data, &d)
	if len(d.TempPassword) != 12 || strings.ContainsAny(d.TempPassword, "0Oo1lI") ||
		agentAuthCore.CheckPasswordPolicy(d.TempPassword) != agentAuthCore.PasswordOK {
		t.Fatalf("temp_password ไม่ถูกรูปแบบ %q", d.TempPassword)
	}

	expect(t, login(t, app, "agent01", password), 401, 401201)
	ld := loginAs(t, app, "agent01", d.TempPassword) // บล็อกถูกล้าง
	if !ld.MustChangePassword {
		t.Fatal("ต้องถูกบังคับเปลี่ยนรหัสผ่าน")
	}
	expect(t, call(t, app, "GET", gatedPath, nil, ld.Token), 403, 401306)
	nw := map[string]string{"old_password": d.TempPassword, "new_password": password, "confirm_password": password}
	expect(t, call(t, app, "POST", passwordChangePath, nw, ld.Token), 200, 422) // รหัสเดิมของ test ไม่ผ่าน AUTH-36 (ไม่มีตัวเลข)
	nw["new_password"], nw["confirm_password"] = newPassword, newPassword
	expect(t, call(t, app, "POST", passwordChangePath, nw, ld.Token), 200, 200) // ไม่ต้องส่ง passcode
	expect(t, call(t, app, "GET", gatedPath, nil, ld.Token), 200, 200)
}
