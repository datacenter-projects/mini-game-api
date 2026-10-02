package agentauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	agentAuthCore "app/app/core/agent_auth"
	agentAuthDto "app/app/internals/backoffice/dto/agent_auth"
	"app/app/models"
	"app/app/repository/postgres"
	redisRepo "app/app/repository/redis"
	"app/pkg/apperr"
	"app/pkg/configs"
	"app/pkg/utils"
	"app/platform/database"
	"app/platform/logger"
)

// LoginService — POST /api/v1/bo/pb/auth/login
func LoginService(ctx context.Context, req agentAuthDto.LoginRequest, ip string) (agentAuthDto.LoginResponse, error) {
	var res agentAuthDto.LoginResponse
	cfg := configs.Cfg.Auth

	// AUTH-11: จำกัดจำนวนครั้งต่อ IP
	count, err := redisRepo.IncrBOLoginIPRepository(ctx, ip)
	if err != nil {
		return res, err
	}
	if count > cfg.LoginIPLimit {
		return res, apperr.ErrTooMany
	}

	username := agentAuthCore.NormalizeUsername(req.Username) // AUTH-01

	// subaccount ยังไม่เปิด (spec หัวข้อ 4 ข้อ 10) — ตอบเหมือนรหัสผิด
	if agentAuthCore.IsSubaccountUsername(username) {
		utils.CheckPasswordDummy(req.Password)
		return res, apperr.ErrInvalidCredentials
	}

	// AUTH-10: username ที่ถูกระงับชั่วคราว ห้าม login แม้รหัสถูก
	blocked, err := redisRepo.IsBOLoginBlockedRepository(ctx, username)
	if err != nil {
		return res, err
	}
	if blocked {
		return res, apperr.ErrLoginTemporarilyBlocked
	}

	db := database.DBConn.WithContext(ctx)

	agent, err := postgres.GetUserAgentForLoginRepository(db, username)
	if errors.Is(err, apperr.ErrNotFound) {
		utils.CheckPasswordDummy(req.Password) // AUTH-02: เวลาตอบเท่ากับกรณีรหัสผิด
		return res, recordLoginFailure(ctx, username)
	}
	if err != nil {
		return res, err
	}
	if !utils.CheckPassword(agent.PasswordHash, req.Password) {
		return res, recordLoginFailure(ctx, username)
	}

	if err := redisRepo.ClearBOLoginFailRepository(ctx, username); err != nil {
		logger.Ctx(ctx).Warnw("clear login fail counter failed", "username", username, "error", err)
	}

	// AUTH-04, AUTH-05: เช็คสถานะหลังรหัสผ่านถูกแล้วเท่านั้น
	if agent.Status == models.AgentStatusLocked {
		return res, apperr.ErrAccountLocked
	}
	uplineLocked, err := postgres.HasLockedAncestorRepository(db, agent.ID)
	if err != nil {
		return res, err
	}
	if uplineLocked {
		return res, apperr.ErrUplineLocked
	}

	// AUTH-06, AUTH-07, AUTH-13: ออก token ได้ก็ต่อเมื่อสร้าง session สำเร็จ
	now := time.Now()
	expiresAt := now.Add(cfg.SessionAbsoluteTimeout)
	sid, err := newSessionID()
	if err != nil {
		return res, err
	}
	token, err := utils.SignSessionToken(cfg.JWTSecret, utils.TokenTypeBackoffice, sid, agent.ID, now, expiresAt)
	if err != nil {
		return res, err
	}
	session := redisRepo.BOSession{AgentID: agent.ID, IP: ip, CreatedAt: now, ExpiresAt: expiresAt}
	idleTTL := agentAuthCore.SessionTTL(now, expiresAt, cfg.SessionIdleTimeout)
	if err := redisRepo.CreateBOSessionRepository(ctx, sid, session, idleTTL); err != nil {
		return res, err
	}

	// AUTH-12: บันทึกไม่สำเร็จไม่ทำให้ login ล้ม
	if err := postgres.UpdateUserAgentLastLoginRepository(db, agent.ID, ip, now); err != nil {
		logger.Ctx(ctx).Warnw("update last login failed", "agent_id", agent.ID, "error", err)
	}

	return agentAuthDto.LoginResponse{
		Token:        token,
		Username:     agent.Username,
		Role:         string(agent.Role),
		IsSubaccount: false,
		PasscodeSet:  false,
		ExpiresAt:    expiresAt,
	}, nil
}

// recordLoginFailure นับครั้งที่ผิด ครบ limit → ระงับ username ชั่วคราว (AUTH-10)
// คืน ErrInvalidCredentials เสมอ (ครั้งที่ทำให้ถูกระงับก็ยังบอกแค่ว่ารหัสผิด)
func recordLoginFailure(ctx context.Context, username string) error {
	cfg := configs.Cfg.Auth
	fails, err := redisRepo.IncrBOLoginFailRepository(ctx, username, cfg.LoginFailWindow)
	if err != nil {
		logger.Ctx(ctx).Warnw("count login failure failed", "username", username, "error", err)
		return apperr.ErrInvalidCredentials
	}
	if agentAuthCore.IsLoginBlocked(fails, cfg.LoginFailLimit) {
		if err := redisRepo.BlockBOLoginRepository(ctx, username, cfg.LoginBlockDuration); err != nil {
			logger.Ctx(ctx).Warnw("block login failed", "username", username, "error", err)
		} else {
			logger.Ctx(ctx).Warnw("login temporarily blocked", "username", username, "fails", fails)
		}
	}
	return apperr.ErrInvalidCredentials
}

// newSessionID — 256-bit จาก crypto/rand
func newSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
