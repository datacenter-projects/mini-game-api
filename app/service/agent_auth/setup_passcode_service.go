package agentauth

import (
	"context"
	"time"

	agentAuthDto "app/app/internals/backoffice/dto/agent_auth"
	"app/app/models"
	"app/pkg/apperr"
	"app/pkg/utils"
	"app/platform/database"

	"gorm.io/gorm"
)

// SetupPasscodeService — POST /api/v1/bo/pr/auth/passcode/setup (AUTH-30, AUTH-32)
// ใช้ได้ครั้งเดียว: มี passcode แล้วตอบ ErrPasscodeAlreadySet เสมอ
func SetupPasscodeService(ctx context.Context, actor Actor, req agentAuthDto.SetupPasscodeRequest, meta RequestMeta) error {
	hash, err := utils.HashPassword(req.Passcode.Value)
	if err != nil {
		return err
	}
	return database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		acc, err := lockAccountCredentials(tx, actor.AccountType, actor.AccountID())
		if err != nil {
			return err
		}
		if acc.PasscodeSet() {
			return apperr.ErrPasscodeAlreadySet
		}
		// UPDATE ... WHERE passcode_hash IS NULL กันยิงพร้อมกันอีกชั้น
		ok, err := setPasscodeIfEmpty(tx, acc, hash, time.Now())
		if err != nil {
			return err
		}
		if !ok {
			return apperr.ErrPasscodeAlreadySet
		}
		return writeAudit(ctx, tx, actorEntry(models.AuthAuditPasscodeSetup, actor, acc), meta)
	})
}
