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
	redisRepo "app/app/repository/redis"
	"app/pkg/apperr"
	"app/pkg/configs"
	"app/pkg/utils"
	"app/platform/database"
	"app/platform/logger"
)

// LoginService — POST /api/v1/bo/pb/auth/login (agent และ sub ใช้เส้นเดียวกัน — AUTH-18)
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

	// AUTH-10, AUTH-24: username ที่ถูกระงับชั่วคราว ห้าม login แม้รหัสถูก
	blocked, err := redisRepo.IsBOLoginBlockedRepository(ctx, username)
	if err != nil {
		return res, err
	}
	if blocked {
		return res, apperr.ErrLoginTemporarilyBlocked
	}

	// AUTH-18: sub ที่รูปแบบผิด (เช่น name สั้นเกิน) ไม่มีทางมีอยู่จริง — ตอบเหมือนไม่พบ
	if agentAuthCore.IsSubaccountUsername(username) {
		if _, _, ok := agentAuthCore.SplitSubaccountUsername(username); !ok {
			utils.CheckPasswordDummy(req.Password)
			return res, recordLoginFailure(ctx, username)
		}
	}

	db := database.DBConn.WithContext(ctx)

	acc, err := loadAccountForLogin(db, username)
	if errors.Is(err, apperr.ErrNotFound) {
		utils.CheckPasswordDummy(req.Password) // AUTH-02, AUTH-18: เวลาตอบเท่ากับกรณีรหัสผิด
		return res, recordLoginFailure(ctx, username)
	}
	if err != nil {
		return res, err
	}
	if !utils.CheckPassword(acc.PasswordHash, req.Password) {
		return res, recordLoginFailure(ctx, username)
	}

	if err := redisRepo.ClearBOLoginFailRepository(ctx, username); err != nil {
		logger.Ctx(ctx).Warnw("clear login fail counter failed", "username", username, "error", err)
	}

	// เช็คสถานะหลังรหัสผ่านถูกแล้วเท่านั้น
	// ลำดับ 401301 → 401302 → 401309 → 401310 (เสนอใน review รอบ 2 ยังรอ lead ยืนยัน)
	if acc.Status == models.AgentStatusLocked { // AUTH-04, AUTH-20
		return res, apperr.ErrAccountLocked
	}
	if acc, err = withUplineStatus(db, acc); err != nil { // AUTH-05, AUTH-21
		return res, err
	}
	if acc.UplineLocked() {
		return res, apperr.ErrUplineLocked
	}
	passcodeBlocked, err := redisRepo.IsBOPasscodeBlockedRepository(ctx, acc.Type, acc.ID) // AUTH-28
	if err != nil {
		return res, err
	}
	if passcodeBlocked {
		return res, apperr.ErrPasscodeBlocked
	}
	now := time.Now()
	if agentAuthCore.IsTempExpired(acc.MustChangePassword, acc.TempPasswordExpiresAt, now) ||
		agentAuthCore.IsTempExpired(acc.MustChangePasscode, acc.TempPasscodeExpiresAt, now) { // AUTH-28
		return res, apperr.ErrTempCredentialExpired
	}

	// AUTH-06, AUTH-07, AUTH-13: ออก token ได้ก็ต่อเมื่อสร้าง session สำเร็จ
	expiresAt := now.Add(cfg.SessionAbsoluteTimeout)
	sid, err := newSessionID()
	if err != nil {
		return res, err
	}
	subject := utils.SessionSubject{AccountType: string(acc.Type), AccountID: acc.ID}
	if acc.IsSub() {
		subject.AgentID = acc.AgentID
	}
	token, err := utils.SignSessionToken(cfg.JWTSecret, utils.TokenTypeBackoffice, sid, subject, now, expiresAt)
	if err != nil {
		return res, err
	}
	session := redisRepo.BOSession{AccountType: acc.Type, AccountID: acc.ID, AgentID: acc.AgentID, IP: ip, CreatedAt: now, ExpiresAt: expiresAt}
	idleTTL := agentAuthCore.SessionTTL(now, expiresAt, cfg.SessionIdleTimeout)
	if err := redisRepo.CreateBOSessionRepository(ctx, sid, session, idleTTL); err != nil {
		return res, err
	}

	// AUTH-12: บันทึกไม่สำเร็จไม่ทำให้ login ล้ม
	if err := updateLastLogin(db, acc, ip, now); err != nil {
		logger.Ctx(ctx).Warnw("update last login failed", "account_type", acc.Type, "account_id", acc.ID, "error", err)
	}

	return agentAuthDto.LoginResponse{
		Token:              token,
		Username:           acc.Username,
		Role:               string(acc.Role),
		IsSubaccount:       acc.IsSub(),
		PasscodeSet:        acc.PasscodeSet(),
		MustChangePassword: acc.MustChangePassword,
		MustChangePasscode: acc.MustChangePasscode,
		ExpiresAt:          expiresAt,
	}, nil
}

// recordLoginFailure นับครั้งที่ผิด ครบ limit → ระงับ username ชั่วคราว (AUTH-10)
// คืน ErrInvalidCredentials เสมอ (ครั้งที่ทำให้ถูกระงับก็ยังบอกแค่ว่ารหัสผิด)
func recordLoginFailure(ctx context.Context, username string) error {
	recordPasswordFailure(ctx, username)
	return apperr.ErrInvalidCredentials
}

// recordPasswordFailure นับรหัสผ่านผิด (login และ old_password — AUTH-10, AUTH-39) คืน true เมื่อครบแล้วถูกระงับ
func recordPasswordFailure(ctx context.Context, username string) bool {
	cfg := configs.Cfg.Auth
	fails, err := redisRepo.IncrBOLoginFailRepository(ctx, username, cfg.LoginFailWindow)
	if err != nil {
		logger.Ctx(ctx).Warnw("count login failure failed", "username", username, "error", err)
		return false
	}
	if !agentAuthCore.IsLoginBlocked(fails, cfg.LoginFailLimit) {
		return false
	}
	if err := redisRepo.BlockBOLoginRepository(ctx, username, cfg.LoginBlockDuration); err != nil {
		logger.Ctx(ctx).Warnw("block login failed", "username", username, "error", err)
		return false
	}
	logger.Ctx(ctx).Warnw("login temporarily blocked", "username", username, "fails", fails)
	return true
}

// newSessionID — 256-bit จาก crypto/rand
func newSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
