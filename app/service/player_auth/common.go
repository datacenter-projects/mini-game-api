// Package playerauth คือ business logic ของ Player Auth — spec: docs/modules/player_auth.md
// ผู้เล่น: login / logout / session / สถานะ · operator (สาย Transfer): ยืนยัน Key + IP, login / register ผู้เล่น
package playerauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"time"

	agentAuthCore "app/app/core/agent_auth"
	agentManagementCore "app/app/core/agent_management"
	playerAuthCore "app/app/core/player_auth"
	"app/app/models"
	playerAuthPostgres "app/app/repository/postgres/player_auth"
	playerAuthRedis "app/app/repository/redis/player_auth"
	"app/pkg/apperr"
	"app/pkg/configs"
	"app/pkg/utils"
	"app/platform/logger"

	"gorm.io/gorm"
)

// RequestMeta — ข้อมูลของ request ที่ service ใช้ (IP สำหรับ rate limit / last_login_ip)
type RequestMeta struct {
	IP string
}

// Player คือผู้เล่นของ request — middleware PlayerAuthenticated set ไว้ใน Locals (struct เดียว)
type Player struct {
	MemberID  uint
	AgentID   uint   // ผู้สร้างตรง
	UplineIDs []uint // สายบนทั้งหมดรวมผู้สร้าง (cnf.parent) — ไว้คิดสถานะ / เกมเปิดจริง
	Username  string
	Name      string
	Currency  string
	Status    models.AgentStatus // สถานะที่ใช้งานจริง: ACTIVE / SUSPENDED (LOCKED ไม่ผ่าน middleware)
	CanBet    bool
	SessionID string
}

// memberState — สถานะของ Member รวมสายบน (หัวข้อ 3.4)
type memberState struct {
	Effective    models.AgentStatus
	OwnLocked    bool
	UplineLocked bool
	UplineIDs    []uint
}

// gate คืน error ของสถานะ LOCKED (407301 / 407302) · ผ่าน = nil — ลำดับ: ตัวเอง → สายบน
func (s memberState) gate() error {
	switch {
	case s.OwnLocked:
		return apperr.ErrPlayerLocked
	case s.UplineLocked:
		return apperr.ErrPlayerUplineLocked
	}
	return nil
}

// loadMemberState คิดสถานะที่ใช้งานจริงของ Member จากสถานะตัวเอง + สายบนใน cnf.parent (query เดียว — PA-30)
func loadMemberState(db *gorm.DB, m models.UserMember) (memberState, error) {
	var chain agentManagementCore.Chain
	if err := json.Unmarshal([]byte(m.Cnf), &chain); err != nil {
		return memberState{}, apperr.ErrInternal.Wrap(err)
	}
	ids := make([]uint, len(chain.Parent))
	for i, p := range chain.Parent {
		ids[i] = p.ID
	}
	statuses, err := playerAuthPostgres.ListAgentStatusesByIDsRepository(db, ids)
	if err != nil {
		return memberState{}, err
	}
	upline := agentAuthCore.WorstStatus(statuses...)
	return memberState{
		Effective:    playerAuthCore.EffectiveStatus(m.Status, statuses...),
		OwnLocked:    m.Status == models.AgentStatusLocked,
		UplineLocked: upline == models.AgentStatusLocked,
		UplineIDs:    ids,
	}, nil
}

func newPlayer(m models.UserMember, st memberState, sid string) Player {
	return Player{MemberID: m.ID, AgentID: m.AgentID, UplineIDs: st.UplineIDs, Username: m.Username, Name: m.Name, Currency: m.Currency,
		Status: st.Effective, CanBet: playerAuthCore.CanBet(st.Effective, m.WinLimitPerDay), SessionID: sid}
}

// issuedSession — ผลของ issueSession
type issuedSession struct {
	Token     string
	ExpiresAt time.Time
}

// issueSession สร้าง session + token ของผู้เล่น (PA-20 – PA-23) และบันทึก last login (PA-25)
// ออก token ได้ก็ต่อเมื่อสร้าง session สำเร็จ (PA-23) · login ใหม่เตะ session เดิม (PA-22)
func issueSession(ctx context.Context, db *gorm.DB, memberID uint, ip string) (issuedSession, error) {
	cfg := configs.Cfg.Auth
	now := time.Now()
	expiresAt := now.Add(cfg.PlayerSessionAbsoluteTimeout)
	sid, err := newSessionID()
	if err != nil {
		return issuedSession{}, err
	}
	token, err := utils.SignSessionToken(cfg.JWTSecret, utils.TokenTypePlayer, sid,
		utils.SessionSubject{AccountType: utils.AccountTypeMember, AccountID: memberID}, now, expiresAt)
	if err != nil {
		return issuedSession{}, err
	}
	idleTTL := agentAuthCore.SessionTTL(now, expiresAt, cfg.PlayerSessionIdleTimeout)
	session := playerAuthRedis.Session{MemberID: memberID, IP: ip, CreatedAt: now, ExpiresAt: expiresAt}
	if err := playerAuthRedis.CreateSessionRepository(ctx, sid, session, idleTTL); err != nil {
		return issuedSession{}, err
	}
	if err := playerAuthPostgres.UpdateMemberLastLoginRepository(db, memberID, ip, now); err != nil { // PA-25: ไม่ทำให้ login ล้ม
		logger.Ctx(ctx).Warnw("update member last login failed", "member_id", memberID, "error", err)
	}
	return issuedSession{Token: token, ExpiresAt: expiresAt}, nil
}

// newSessionID — 256-bit จาก crypto/rand
func newSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// parseToken ตรวจ token ผู้เล่น: ลายเซ็น / หมดอายุ / typ "pl" / account_type MEMBER (PA-20 · PA-24) — ไม่ผ่านคืน ErrPlayerInvalidToken
func parseToken(rawToken string) (*utils.SessionClaims, error) {
	claims, err := utils.ParseSessionToken(configs.Cfg.Auth.JWTSecret, utils.TokenTypePlayer, rawToken)
	if err != nil {
		return nil, apperr.ErrPlayerInvalidToken.Wrap(err)
	}
	if claims.AccountType != utils.AccountTypeMember {
		return nil, apperr.ErrPlayerInvalidToken
	}
	return claims, nil
}

func endSession(ctx context.Context, sid string, memberID uint) {
	if err := playerAuthRedis.DeleteSessionRepository(ctx, sid, memberID); err != nil {
		logger.Ctx(ctx).Warnw("delete player session failed", "member_id", memberID, "error", err)
	}
}

// EndSessionsOfMemberService ลบ session ของผู้เล่น — members/status/update ตั้ง LOCKED เรียกหลัง commit (PA-34)
// Redis ล่ม = log warn ไม่ทำให้ผู้เรียกล้ม (middleware เช็คสถานะทุก request อยู่แล้ว จึงยังกันได้)
func EndSessionsOfMemberService(ctx context.Context, memberID uint) {
	if err := playerAuthRedis.DeleteSessionsOfMemberRepository(ctx, memberID); err != nil {
		logger.Ctx(ctx).Warnw("delete player sessions after lock failed", "member_id", memberID, "error", err)
	}
}

// recordPasswordFailure นับรหัสผิด (login PA1 และ old_password PA4 — PA-41) · ครบ limit → ระงับ username ชั่วคราว
// คืน true เมื่อครบแล้วถูกระงับ
func recordPasswordFailure(ctx context.Context, username string) bool {
	cfg := configs.Cfg.Auth
	fails, err := playerAuthRedis.IncrLoginFailRepository(ctx, username, cfg.LoginFailWindow)
	if err != nil {
		logger.Ctx(ctx).Warnw("count player login failure failed", "username", username, "error", err)
		return false
	}
	if !agentAuthCore.IsLoginBlocked(fails, cfg.LoginFailLimit) {
		return false
	}
	if err := playerAuthRedis.BlockLoginRepository(ctx, username, cfg.LoginBlockDuration); err != nil {
		logger.Ctx(ctx).Warnw("block player login failed", "username", username, "error", err)
		return false
	}
	logger.Ctx(ctx).Warnw("player login temporarily blocked", "username", username, "fails", fails)
	return true
}
