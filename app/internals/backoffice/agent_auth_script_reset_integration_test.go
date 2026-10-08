//go:build integration

// test ของ scripts/reset_credentials (AUTH-51) — เรียก service ตรงเพราะ script ไม่ใช่ HTTP route
package backoffice_test

import (
	"context"
	"errors"
	"testing"

	"app/app/models"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
)

func scriptReset(username string, password, passcode bool) (agentAuthService.ScriptResetResult, error) {
	return agentAuthService.ScriptResetCredentialsService(context.Background(), agentAuthService.ScriptResetRequest{
		Username: username, Password: password, Passcode: passcode, Operator: "ops01",
	})
}

func TestScriptResetCredentials(t *testing.T) { // AUTH-51
	app := setup2(t)
	sa := createAccount(t, "superadmin1", models.AgentRoleSuperAdmin, nil)
	setPasscode(t, models.AccountTypeAgent, sa.ID, passcode)
	saTok := loginReady(t, app, models.AccountTypeAgent, sa.ID, "superadmin1")

	// รีเซ็ตทั้งสองอย่าง → session เดิมหลุด · login ด้วยค่าชั่วคราว ถูกบังคับเปลี่ยนทั้งคู่
	res, err := scriptReset("SuperAdmin1", true, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.TempPassword) != 12 || len(res.TempPasscode) != 6 || res.Username != "superadmin1" {
		t.Fatalf("result %+v", res)
	}
	expect(t, call(t, app, "GET", gatedPath, nil, saTok), 401, 401203)
	expect(t, login(t, app, "superadmin1", password), 401, 401201)
	ld := loginAs(t, app, "superadmin1", res.TempPassword)
	if !ld.MustChangePassword || !ld.MustChangePasscode {
		t.Fatalf("ต้องถูกบังคับเปลี่ยนทั้งคู่ %+v", ld)
	}

	// audit: actor = SCRIPT · actor_username = ผู้รัน · ไม่มี actor_id
	for _, action := range []models.AuthAuditAction{models.AuthAuditResetPassword, models.AuthAuditResetPasscode} {
		rows := auditRows(t, action)
		if len(rows) != 1 || rows[0].ActorType != models.AuthAuditActorScript || rows[0].ActorID != nil ||
			rows[0].ActorUsername == nil || *rows[0].ActorUsername != "ops01" || rows[0].TargetUsername != "superadmin1" {
			t.Fatalf("%s audit %+v", action, rows)
		}
	}

	// ADMIN ที่ยังไม่ตั้ง passcode: รีเซ็ตรหัสผ่านได้ · passcode ไม่ได้ (AUTH-47)
	createAccount(t, "support01", models.AgentRoleAdmin, nil)
	if _, err := scriptReset("support01", true, false); err != nil {
		t.Fatal(err)
	}
	if _, err := scriptReset("support01", false, true); !errors.Is(err, apperr.ErrResetTargetNoPasscode) {
		t.Fatalf("passcode ของบัญชีที่ยังไม่ตั้ง: %v", err)
	}

	// บัญชีอื่น / ไม่มีบัญชี / ไม่เลือกอะไร
	createAgent(t, "agent01", models.AgentStatusActive, nil)
	if _, err := scriptReset("agent01", true, false); !errors.Is(err, apperr.ErrResetNotAllowed) {
		t.Fatalf("agent: %v", err)
	}
	if _, err := scriptReset("nobody", true, false); !errors.Is(err, apperr.ErrResetTargetNotFound) {
		t.Fatalf("not found: %v", err)
	}
	if _, err := scriptReset("superadmin1", false, false); !errors.Is(err, apperr.ErrValidation) {
		t.Fatalf("no flag: %v", err)
	}
}
