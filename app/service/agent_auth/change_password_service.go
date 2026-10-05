package agentauth

import (
	"context"
	"time"

	agentAuthCore "app/app/core/agent_auth"
	agentAuthDto "app/app/internals/backoffice/dto/agent_auth"
	"app/app/models"
	"app/pkg/apperr"
	"app/pkg/utils"
	"app/platform/database"

	"gorm.io/gorm"
)

// ChangePasswordService — POST /api/v1/bo/pr/auth/password/change (AUTH-37–AUTH-41)
// passcode (ตอนไม่ได้ถูกบังคับ) ตรวจแล้วที่ middleware · สำเร็จแล้ว session เดิมใช้ต่อได้
func ChangePasswordService(ctx context.Context, actor Actor, req agentAuthDto.ChangePasswordRequest, meta RequestMeta) error {
	newHash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	return database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		acc, err := lockAccountCredentials(tx, actor.AccountType, actor.AccountID())
		if err != nil {
			return err
		}
		now := time.Now()
		if agentAuthCore.IsTempExpired(acc.MustChangePassword, acc.TempPasswordExpiresAt, now) { // AUTH-41
			return apperr.ErrTempCredentialExpired
		}
		if !utils.CheckPassword(acc.PasswordHash, req.OldPassword) { // AUTH-39
			if recordPasswordFailure(ctx, acc.Username, &acc, meta) {
				endAllSessions(ctx, acc)
				return apperr.ErrLoginTemporarilyBlocked
			}
			return apperr.ErrOldPasswordIncorrect
		}
		if isRecentPassword(acc, req.NewPassword) { // AUTH-38
			return apperr.ErrPasswordReused
		}
		if err := updatePassword(tx, acc, newHash, false, nil, now); err != nil {
			return err
		}
		return writeAudit(ctx, tx, actorEntry(models.AuthAuditPasswordChange, actor, acc), meta)
	})
}

// isRecentPassword — ตรงกับรหัสปัจจุบันหรือรหัสก่อนหน้า 1 รหัส (AUTH-38)
func isRecentPassword(acc account, plain string) bool {
	if utils.CheckPassword(acc.PasswordHash, plain) {
		return true
	}
	return acc.PreviousPasswordHash != nil && utils.CheckPassword(*acc.PreviousPasswordHash, plain)
}
