package agentauth

import (
	"context"
	"errors"
	"time"

	agentAuthCore "app/app/core/agent_auth"
	"app/app/models"
	redisRepo "app/app/repository/redis"
	"app/pkg/apperr"
	"app/pkg/configs"
	"app/pkg/utils"
	"app/platform/database"
	"app/platform/logger"
)

// AuthenticateService ตรวจ token + session ของ request หลังบ้าน — ใช้โดย middleware Authenticated
// rule: AUTH-07, AUTH-08, AUTH-14, AUTH-15, AUTH-16 · phase 2: AUTH-23, AUTH-27
func AuthenticateService(ctx context.Context, rawToken, ip string) (Actor, error) {
	cfg := configs.Cfg.Auth

	claims, accountType, err := parseToken(rawToken)
	if err != nil {
		return Actor{}, err
	}
	sid, accountID := claims.SessionID, claims.AccountID

	session, err := redisRepo.GetBOSessionRepository(ctx, sid, accountType, accountID)
	if errors.Is(err, apperr.ErrNotFound) {
		return Actor{}, apperr.ErrSessionEnded
	}
	if err != nil {
		return Actor{}, err
	}

	ttl := agentAuthCore.SessionTTL(time.Now(), session.ExpiresAt, cfg.SessionIdleTimeout)
	if ttl <= 0 {
		endSession(ctx, sid, accountType, accountID)
		return Actor{}, apperr.ErrSessionEnded
	}

	db := database.DBConn.WithContext(ctx)
	acc, err := loadAccountAuth(db, accountType, accountID)
	if errors.Is(err, apperr.ErrNotFound) {
		endSession(ctx, sid, accountType, accountID)
		return Actor{}, apperr.ErrSessionEnded
	}
	if err != nil {
		return Actor{}, err
	}

	if acc.Status == models.AgentStatusLocked { // AUTH-15, AUTH-27
		endAllSessions(ctx, acc)
		return Actor{}, apperr.ErrAccountLocked
	}
	// AUTH-27: ผู้สร้าง (กรณี sub) หรือ upline คนใดถูก LOCK → เตะออก (ทั้ง agent และ sub — แทน Phase 1 หัวข้อ 9 ข้อ 2)
	if acc, err = withUplineStatus(db, acc); err != nil {
		return Actor{}, err
	}
	if acc.UplineLocked() {
		endAllSessions(ctx, acc)
		return Actor{}, apperr.ErrUplineLocked
	}

	if ip != session.IP { // AUTH-16: log อย่างเดียว
		logger.Ctx(ctx).Warnw("session ip mismatch", "account_type", acc.Type, "account_id", acc.ID, "login_ip", session.IP, "request_ip", ip)
	}

	if err := redisRepo.TouchBOSessionRepository(ctx, sid, ttl); err != nil {
		logger.Ctx(ctx).Warnw("touch session failed", "account_type", acc.Type, "account_id", acc.ID, "error", err)
	}

	return newActor(acc, sid), nil
}

// parseToken ตรวจ token และชนิดบัญชีใน claim — ไม่ผ่านคืน ErrInvalidToken
func parseToken(rawToken string) (*utils.SessionClaims, models.AccountType, error) {
	claims, err := utils.ParseSessionToken(configs.Cfg.Auth.JWTSecret, utils.TokenTypeBackoffice, rawToken)
	if err != nil {
		return nil, "", apperr.ErrInvalidToken.Wrap(err)
	}
	t := models.AccountType(claims.AccountType)
	switch {
	case t == models.AccountTypeAgent:
	case t == models.AccountTypeSub && claims.AgentID != 0:
	default:
		return nil, "", apperr.ErrInvalidToken
	}
	return claims, t, nil
}

func endSession(ctx context.Context, sid string, t models.AccountType, id uint) {
	if err := redisRepo.DeleteBOSessionRepository(ctx, sid, t, id); err != nil {
		logger.Ctx(ctx).Warnw("delete session failed", "account_type", t, "account_id", id, "error", err)
	}
}

func endAllSessions(ctx context.Context, a account) {
	if err := redisRepo.DeleteBOSessionsOfAccountRepository(ctx, a.Type, a.ID); err != nil {
		logger.Ctx(ctx).Warnw("delete sessions of account failed", "account_type", a.Type, "account_id", a.ID, "error", err)
	}
}
