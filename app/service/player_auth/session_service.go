package playerauth

import (
	"context"
	"errors"
	"time"

	agentAuthCore "app/app/core/agent_auth"
	playerAuthPostgres "app/app/repository/postgres/player_auth"
	playerAuthRedis "app/app/repository/redis/player_auth"
	"app/pkg/apperr"
	"app/pkg/configs"
	"app/platform/database"
	"app/platform/logger"
)

func logWarn(ctx context.Context, msg, username string, err error) {
	logger.Ctx(ctx).Warnw(msg, "username", username, "error", err)
}

// AuthenticateService ตรวจ token + session + สถานะของ request ผู้เล่น — ใช้โดย middleware PlayerAuthenticated
// rule: PA-20 – PA-24, PA-30, PA-32 · SUSPENDED ผ่านได้ (PA-33) · ไม่ cache สถานะ (PA-30)
func AuthenticateService(ctx context.Context, rawToken string) (Player, error) {
	cfg := configs.Cfg.Auth

	claims, err := parseToken(rawToken)
	if err != nil {
		return Player{}, err
	}
	sid, memberID := claims.SessionID, claims.AccountID

	session, err := playerAuthRedis.GetSessionRepository(ctx, sid, memberID)
	if errors.Is(err, playerAuthRedis.ErrKicked) { // PA-22
		return Player{}, apperr.ErrPlayerKicked
	}
	if errors.Is(err, apperr.ErrNotFound) {
		return Player{}, apperr.ErrPlayerSessionEnded
	}
	if err != nil {
		return Player{}, err
	}

	ttl := agentAuthCore.SessionTTL(time.Now(), session.ExpiresAt, cfg.PlayerSessionIdleTimeout)
	if ttl <= 0 { // absolute หมด
		endSession(ctx, sid, memberID)
		return Player{}, apperr.ErrPlayerSessionEnded
	}

	db := database.DBConn.WithContext(ctx)
	m, err := playerAuthPostgres.GetMemberAuthByIDRepository(db, memberID)
	if errors.Is(err, apperr.ErrNotFound) {
		endSession(ctx, sid, memberID)
		return Player{}, apperr.ErrPlayerSessionEnded
	}
	if err != nil {
		return Player{}, err
	}
	st, err := loadMemberState(db, m)
	if err != nil {
		return Player{}, err
	}
	if err := st.gate(); err != nil { // PA-32: LOCKED ระหว่างใช้งาน → ปฏิเสธและลบ session
		endSession(ctx, sid, memberID)
		return Player{}, err
	}

	if err := playerAuthRedis.TouchSessionRepository(ctx, sid, ttl); err != nil { // PA-21
		logger.Ctx(ctx).Warnw("touch player session failed", "member_id", memberID, "error", err)
	}
	return newPlayer(m, st, sid), nil
}

// LogoutService — PA3 POST /api/v1/player/pr/auth/logout (PA-26)
// ไม่ผ่าน middleware PlayerAuthenticated: session ที่หลุดไปแล้ว (logout แล้ว · ถูกเตะ · idle หมด) ต้อง logout สำเร็จได้
// token ปลอม / เลย absolute → 407202 · logout ด้วย token เก่าไม่กระทบ session ปัจจุบัน (ลบเฉพาะถ้า pointer ยังชี้ sid นี้)
func LogoutService(ctx context.Context, rawToken string) error {
	claims, err := parseToken(rawToken)
	if err != nil {
		return err
	}
	return playerAuthRedis.DeleteSessionRepository(ctx, claims.SessionID, claims.AccountID)
}
