package agentauth

import (
	"context"
	"time"

	agentAuthCore "app/app/core/agent_auth"
	agentAuthDto "app/app/internals/backoffice/dto/agent_auth"
	"app/pkg/apperr"
	"app/pkg/utils"
	"app/platform/database"

	"gorm.io/gorm"
)

// ChangePasscodeService — POST /api/v1/bo/pr/auth/passcode/change (AUTH-34, AUTH-35)
// สำเร็จแล้ว session เดิมใช้ต่อได้
func ChangePasscodeService(ctx context.Context, actor Actor, req agentAuthDto.ChangePasscodeRequest) error {
	newHash, err := utils.HashPassword(req.NewPasscode.Value)
	if err != nil {
		return err
	}
	return database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		acc, err := lockAccountCredentials(tx, actor.AccountType, actor.AccountID())
		if err != nil {
			return err
		}
		if !acc.PasscodeSet() {
			return apperr.ErrPasscodeNotSet
		}
		now := time.Now()
		if agentAuthCore.IsTempExpired(acc.MustChangePasscode, acc.TempPasscodeExpiresAt, now) {
			return apperr.ErrTempCredentialExpired
		}
		if err := checkPasscode(ctx, acc, req.OldPasscode.Value); err != nil {
			return err
		}
		if utils.CheckPassword(*acc.PasscodeHash, req.NewPasscode.Value) { // รวมค่าชั่วคราวจาก admin
			return apperr.ErrPasscodeReused
		}
		return updatePasscode(tx, acc, newHash, false, nil, now)
	})
}
