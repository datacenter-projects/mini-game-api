package agentauth

import (
	"context"
	"time"

	agentAuthCore "app/app/core/agent_auth"
	agentAuthDto "app/app/internals/backoffice/dto/agent_auth"
	"app/app/models"
	redisRepo "app/app/repository/redis"
	"app/pkg/apperr"
	"app/pkg/configs"
	"app/pkg/utils"
	"app/platform/database"
	"app/platform/logger"

	"gorm.io/gorm"
)

// ResetPasscodeService — POST /api/v1/bo/pr/admin/passcode/reset (AUTH-44–AUTH-47, AUTH-49)
func ResetPasscodeService(ctx context.Context, actor Actor, req agentAuthDto.ResetCredentialRequest, meta RequestMeta) (agentAuthDto.ResetPasscodeResponse, error) {
	var res agentAuthDto.ResetPasscodeResponse
	target, err := loadResetTarget(ctx, actor, req.Username)
	if err != nil {
		return res, err
	}

	now := time.Now()
	expiresAt := now.Add(configs.Cfg.Auth.TempCredentialTTL)
	var temp string
	err = database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		acc, err := lockAccountCredentials(tx, target.Type, target.ID)
		if err != nil {
			return err
		}
		if acc.Status == models.AgentStatusLocked {
			return apperr.ErrResetTargetLocked
		}
		if !acc.PasscodeSet() { // AUTH-47
			return apperr.ErrResetTargetNoPasscode
		}
		for temp == "" || utils.CheckPassword(*acc.PasscodeHash, temp) { // ห้ามซ้ำกับตัวปัจจุบัน
			if temp, err = utils.RandomString("0123456789", agentAuthCore.PasscodeLength); err != nil {
				return err
			}
		}
		hash, err := utils.HashPassword(temp)
		if err != nil {
			return err
		}
		if err := updatePasscode(tx, acc, hash, true, &expiresAt, now); err != nil {
			return err
		}
		return writeAudit(ctx, tx, actorEntry(models.AuthAuditResetPasscode, actor, acc), meta)
	})
	if err != nil {
		return res, err
	}

	// หลัง commit: ล้างตัวนับ/บล็อก passcode และเตะ session ของเป้าหมาย
	if err := redisRepo.ClearBOPasscodeBlockRepository(ctx, target.Type, target.ID); err != nil {
		logger.Ctx(ctx).Warnw("clear passcode block failed", "account_type", target.Type, "account_id", target.ID, "error", err)
	}
	endAllSessions(ctx, target)

	return agentAuthDto.ResetPasscodeResponse{Username: target.Username, TempPasscode: temp, TempExpiresAt: expiresAt}, nil
}
