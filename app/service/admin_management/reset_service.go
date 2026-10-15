package adminmanagement

import (
	"context"

	agentAuthCore "app/app/core/agent_auth"
	adminManagementDto "app/app/internals/backoffice/dto/admin_management"
	"app/app/models"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
)

// ADMIN รีเซ็ตรหัสให้บัญชีอื่น — spec: docs/modules/admin_management.md (AUTH-44 – AUTH-49, AUTH-52)
// กฎว่า "รีเซ็ตใครได้" อยู่ที่นี่ · การตั้งค่าชั่วคราว / audit / ตัด session ใช้ agentAuthService.ResetCredentialsService

// ResetPasscodeService — POST /api/v1/bo/pr/admin/passcode/reset (AUTH-45 – AUTH-47, AUTH-49)
func ResetPasscodeService(ctx context.Context, actor agentAuthService.Actor, req adminManagementDto.ResetCredentialRequest,
	meta agentAuthService.RequestMeta) (adminManagementDto.ResetPasscodeResponse, error) {
	t, err := loadResetTarget(ctx, actor, req.Username)
	if err != nil {
		return adminManagementDto.ResetPasscodeResponse{}, err
	}
	res, err := agentAuthService.ResetCredentialsService(ctx, t, agentAuthService.ResetOptions{Passcode: true, RefuseLocked: true},
		agentAuthService.ResetBy{Actor: &actor, Meta: meta})
	if err != nil {
		return adminManagementDto.ResetPasscodeResponse{}, err
	}
	return adminManagementDto.ResetPasscodeResponse{Username: res.Username, TempPasscode: res.TempPasscode, TempExpiresAt: res.TempExpiresAt}, nil
}

// ResetPasswordService — POST /api/v1/bo/pr/admin/password/reset (AUTH-45, AUTH-46, AUTH-48, AUTH-49)
func ResetPasswordService(ctx context.Context, actor agentAuthService.Actor, req adminManagementDto.ResetCredentialRequest,
	meta agentAuthService.RequestMeta) (adminManagementDto.ResetPasswordResponse, error) {
	t, err := loadResetTarget(ctx, actor, req.Username)
	if err != nil {
		return adminManagementDto.ResetPasswordResponse{}, err
	}
	res, err := agentAuthService.ResetCredentialsService(ctx, t, agentAuthService.ResetOptions{Password: true, RefuseLocked: true},
		agentAuthService.ResetBy{Actor: &actor, Meta: meta})
	if err != nil {
		return adminManagementDto.ResetPasswordResponse{}, err
	}
	return adminManagementDto.ResetPasswordResponse{Username: res.Username, TempPassword: res.TempPassword, TempExpiresAt: res.TempExpiresAt}, nil
}

// loadResetTarget — เป้าหมายที่ ADMIN รีเซ็ตได้ (AUTH-45): Company / Share / Agent และ sub · ไม่ใช่ตัวเอง · ไม่ถูกล็อก (ตัวเองหรือหัวสาย)
func loadResetTarget(ctx context.Context, actor agentAuthService.Actor, username string) (agentAuthService.ResetTarget, error) {
	t, err := agentAuthService.FindResetTargetService(ctx, username)
	if err != nil {
		return t, err
	}
	isSelf := t.AccountType() == actor.AccountType && t.ID() == actor.AccountID()
	if isSelf || (!t.IsSub() && !agentAuthCore.IsResettableRole(t.Role())) {
		return t, apperr.ErrResetNotAllowed
	}
	if t.Status() == models.AgentStatusLocked || t.UplineLocked() {
		return t, apperr.ErrResetTargetLocked
	}
	return t, nil
}
