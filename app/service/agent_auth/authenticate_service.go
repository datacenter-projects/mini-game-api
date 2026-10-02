package agentauth

import (
	"context"
	"errors"
	"time"

	agentAuthCore "app/app/core/agent_auth"
	"app/app/models"
	"app/app/repository/postgres"
	redisRepo "app/app/repository/redis"
	"app/pkg/apperr"
	"app/pkg/configs"
	"app/pkg/utils"
	"app/platform/database"
	"app/platform/logger"
)

// AuthenticateService ตรวจ token + session ของ request หลังบ้าน — ใช้โดย middleware Authenticated
// rule: AUTH-07, AUTH-08, AUTH-14, AUTH-15, AUTH-16
func AuthenticateService(ctx context.Context, rawToken, ip string) (Actor, error) {
	cfg := configs.Cfg.Auth

	claims, err := utils.ParseSessionToken(cfg.JWTSecret, utils.TokenTypeBackoffice, rawToken)
	if err != nil {
		return Actor{}, apperr.ErrInvalidToken.Wrap(err)
	}

	session, err := redisRepo.GetBOSessionRepository(ctx, claims.SessionID, claims.AgentID)
	if errors.Is(err, apperr.ErrNotFound) {
		return Actor{}, apperr.ErrSessionEnded
	}
	if err != nil {
		return Actor{}, err
	}

	ttl := agentAuthCore.SessionTTL(time.Now(), session.ExpiresAt, cfg.SessionIdleTimeout)
	if ttl <= 0 {
		endSession(ctx, claims.SessionID, claims.AgentID)
		return Actor{}, apperr.ErrSessionEnded
	}

	agent, err := postgres.GetUserAgentAuthByIDRepository(database.DBConn.WithContext(ctx), claims.AgentID)
	if errors.Is(err, apperr.ErrNotFound) {
		endSession(ctx, claims.SessionID, claims.AgentID)
		return Actor{}, apperr.ErrSessionEnded
	}
	if err != nil {
		return Actor{}, err
	}

	if agent.Status == models.AgentStatusLocked { // AUTH-15
		if err := redisRepo.DeleteBOSessionsOfAgentRepository(ctx, agent.ID); err != nil {
			logger.Ctx(ctx).Warnw("delete sessions of locked agent failed", "agent_id", agent.ID, "error", err)
		}
		return Actor{}, apperr.ErrAccountLocked
	}

	if ip != session.IP { // AUTH-16: log อย่างเดียว
		logger.Ctx(ctx).Warnw("session ip mismatch", "agent_id", agent.ID, "login_ip", session.IP, "request_ip", ip)
	}

	if err := redisRepo.TouchBOSessionRepository(ctx, claims.SessionID, ttl); err != nil {
		logger.Ctx(ctx).Warnw("touch session failed", "agent_id", agent.ID, "error", err)
	}

	return Actor{
		AgentID:   agent.ID,
		ParentID:  agent.ParentID,
		Username:  agent.Username,
		Role:      agent.Role,
		Status:    agent.Status,
		SessionID: claims.SessionID,
	}, nil
}

func endSession(ctx context.Context, sid string, agentID uint) {
	if err := redisRepo.DeleteBOSessionRepository(ctx, sid, agentID); err != nil {
		logger.Ctx(ctx).Warnw("delete session failed", "agent_id", agentID, "error", err)
	}
}
