package agentauth

import (
	"context"
	"errors"
	"fmt"

	agentAuthCore "app/app/core/agent_auth"
	"app/app/models"
	agentAuthRedis "app/app/repository/redis/agent_auth"
	"app/pkg/apperr"
	"app/pkg/configs"
	"app/pkg/utils"
	"app/platform/database"
	"app/platform/logger"
)

// VerifyPasscodeService — ใช้โดย middleware RequirePasscode (AUTH-33, AUTH-35)
func VerifyPasscodeService(ctx context.Context, actor Actor, passcode string, meta RequestMeta) error {
	acc, err := loadAccountCredentials(database.DBConn.WithContext(ctx), actor.AccountType, actor.AccountID())
	if errors.Is(err, apperr.ErrNotFound) {
		return apperr.ErrSessionEnded
	}
	if err != nil {
		return err
	}
	if !acc.PasscodeSet() {
		return apperr.ErrPasscodeNotSet
	}
	return checkPasscode(ctx, acc, passcode, meta)
}

// checkPasscode เทียบ passcode กับ hash ของบัญชีพร้อมนับครั้งที่ผิด (AUTH-35)
//   - ถูก → ล้างตัวนับ
//   - ผิดครั้งที่ 1–4 → ErrPasscodeIncorrect พร้อมจำนวนครั้งที่เหลือ
//   - ผิดครบ → บล็อก login 1 ชม., ลบ session ทั้งหมดของบัญชี → ErrPasscodeTooManyFails
func checkPasscode(ctx context.Context, acc account, passcode string, meta RequestMeta) error {
	cfg := configs.Cfg.Auth
	if utils.CheckPassword(*acc.PasscodeHash, passcode) {
		if err := agentAuthRedis.ClearBOPasscodeFailRepository(ctx, acc.Type, acc.ID); err != nil {
			logger.Ctx(ctx).Warnw("clear passcode fail counter failed", "account_type", acc.Type, "account_id", acc.ID, "error", err)
		}
		return nil
	}

	fails, err := agentAuthRedis.IncrBOPasscodeFailRepository(ctx, acc.Type, acc.ID, cfg.PasscodeFailWindow)
	if err != nil {
		return err
	}
	if !agentAuthCore.IsPasscodeBlocked(fails, cfg.PasscodeFailLimit) {
		left := agentAuthCore.PasscodeAttemptsLeft(fails, cfg.PasscodeFailLimit)
		return apperr.ErrPasscodeIncorrect.WithMessage(
			fmt.Sprintf("passcode ไม่ถูกต้อง (เหลือ %d ครั้ง)", left),
			fmt.Sprintf("Incorrect passcode (%d attempts left)", left))
	}

	if err := agentAuthRedis.BlockBOPasscodeRepository(ctx, acc.Type, acc.ID, cfg.PasscodeBlockDuration); err != nil {
		return err
	}
	endAllSessions(ctx, acc)
	writeAuditOutsideTx(ctx, auditEntry{Action: models.AuthAuditPasscodeBlocked, ActorType: models.AuthAuditActorSystem, Target: &acc}, meta)
	logger.Ctx(ctx).Warnw("passcode blocked", "account_type", acc.Type, "account_id", acc.ID, "fails", fails)
	return apperr.ErrPasscodeTooManyFails
}
