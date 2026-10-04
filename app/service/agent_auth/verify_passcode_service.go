package agentauth

import (
	"context"
	"errors"
	"fmt"

	agentAuthCore "app/app/core/agent_auth"
	redisRepo "app/app/repository/redis"
	"app/pkg/apperr"
	"app/pkg/configs"
	"app/pkg/utils"
	"app/platform/database"
	"app/platform/logger"
)

// VerifyPasscodeService — ใช้โดย middleware RequirePasscode (AUTH-33, AUTH-35)
func VerifyPasscodeService(ctx context.Context, actor Actor, passcode string) error {
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
	return checkPasscode(ctx, acc, passcode)
}

// checkPasscode เทียบ passcode กับ hash ของบัญชีพร้อมนับครั้งที่ผิด (AUTH-35)
//   - ถูก → ล้างตัวนับ
//   - ผิดครั้งที่ 1–4 → ErrPasscodeIncorrect พร้อมจำนวนครั้งที่เหลือ
//   - ผิดครบ → บล็อก login 1 ชม., ลบ session ทั้งหมดของบัญชี → ErrPasscodeTooManyFails
func checkPasscode(ctx context.Context, acc account, passcode string) error {
	cfg := configs.Cfg.Auth
	if utils.CheckPassword(*acc.PasscodeHash, passcode) {
		if err := redisRepo.ClearBOPasscodeFailRepository(ctx, acc.Type, acc.ID); err != nil {
			logger.Ctx(ctx).Warnw("clear passcode fail counter failed", "account_type", acc.Type, "account_id", acc.ID, "error", err)
		}
		return nil
	}

	fails, err := redisRepo.IncrBOPasscodeFailRepository(ctx, acc.Type, acc.ID, cfg.PasscodeFailWindow)
	if err != nil {
		return err
	}
	if !agentAuthCore.IsPasscodeBlocked(fails, cfg.PasscodeFailLimit) {
		left := agentAuthCore.PasscodeAttemptsLeft(fails, cfg.PasscodeFailLimit)
		return apperr.ErrPasscodeIncorrect.WithMessage(
			fmt.Sprintf("passcode ไม่ถูกต้อง (เหลือ %d ครั้ง)", left),
			fmt.Sprintf("Incorrect passcode (%d attempts left)", left))
	}

	if err := redisRepo.BlockBOPasscodeRepository(ctx, acc.Type, acc.ID, cfg.PasscodeBlockDuration); err != nil {
		return err
	}
	endAllSessions(ctx, acc)
	logger.Ctx(ctx).Warnw("passcode blocked", "account_type", acc.Type, "account_id", acc.ID, "fails", fails)
	return apperr.ErrPasscodeTooManyFails
}
