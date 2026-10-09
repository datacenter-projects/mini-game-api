package agentauth

import (
	"context"
	"time"

	agentAuthDto "app/app/internals/backoffice/dto/agent_auth"
	"app/app/models"
	agentAuthRedis "app/app/repository/redis/agent_auth"
	"app/pkg/apperr"
	"app/pkg/configs"
	"app/pkg/utils"
	"app/platform/database"
	"app/platform/logger"

	"gorm.io/gorm"
)

// ResetPasswordService — POST /api/v1/bo/pr/admin/password/reset (AUTH-44–AUTH-46, AUTH-48, AUTH-49)
func ResetPasswordService(ctx context.Context, actor Actor, req agentAuthDto.ResetCredentialRequest, meta RequestMeta) (agentAuthDto.ResetPasswordResponse, error) {
	var res agentAuthDto.ResetPasswordResponse
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
		if temp, err = newTempPassword(acc); err != nil {
			return err
		}
		hash, err := utils.HashPassword(temp)
		if err != nil {
			return err
		}
		if err := updatePassword(tx, acc, hash, true, &expiresAt, now); err != nil {
			return err
		}
		return writeAudit(ctx, tx, actorEntry(models.AuthAuditResetPassword, actor, acc), meta)
	})
	if err != nil {
		return res, err
	}

	// หลัง commit: ล้างบล็อก login (AUTH-10) และเตะ session ของเป้าหมาย
	if err := agentAuthRedis.ClearBOLoginBlockRepository(ctx, target.Username); err != nil {
		logger.Ctx(ctx).Warnw("clear login block failed", "username", target.Username, "error", err)
	}
	endAllSessions(ctx, target)

	return agentAuthDto.ResetPasswordResponse{Username: target.Username, TempPassword: temp, TempExpiresAt: expiresAt}, nil
}
