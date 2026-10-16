package playerauth

import (
	"context"
	"errors"

	playerAuthCore "app/app/core/player_auth"
	playerAuthDto "app/app/internals/frontend/dto/player_auth"
	playerAuthPostgres "app/app/repository/postgres/player_auth"
	playerAuthRedis "app/app/repository/redis/player_auth"
	"app/pkg/apperr"
	"app/pkg/configs"
	"app/pkg/utils"
	"app/platform/database"
)

// LoginService — PA1 POST /api/v1/player/pb/login (PA-03, PA-04, PA-22, PA-31, PA-40, PA-41)
// ใช้ได้กับผู้เล่นที่มีรหัสผ่านเท่านั้น · ไม่พบ / รหัสผิด / ไม่มีรหัส → 407201 ตัวเดียว เวลาตอบใกล้เคียงกัน
func LoginService(ctx context.Context, req playerAuthDto.LoginRequest, meta RequestMeta) (playerAuthDto.LoginResponse, error) {
	var res playerAuthDto.LoginResponse
	cfg := configs.Cfg.Auth

	// PA-40: จำกัดจำนวนครั้งต่อ IP
	count, err := playerAuthRedis.IncrLoginIPRepository(ctx, meta.IP)
	if err != nil {
		return res, err
	}
	if count > cfg.LoginIPLimit {
		return res, apperr.ErrTooMany
	}

	username := playerAuthCore.NormalizeUsername(req.Username) // PA-04

	// PA-41: username ที่ถูกระงับชั่วคราว ห้าม login แม้รหัสถูก
	blocked, err := playerAuthRedis.IsLoginBlockedRepository(ctx, username)
	if err != nil {
		return res, err
	}
	if blocked {
		return res, apperr.ErrPlayerLoginBlocked
	}

	db := database.DBConn.WithContext(ctx)
	m, err := playerAuthPostgres.GetMemberForLoginRepository(db, username)
	if errors.Is(err, apperr.ErrNotFound) {
		utils.CheckPasswordDummy(req.Password) // PA-04: เวลาตอบเท่ากับกรณีรหัสผิด
		return res, loginFailure(ctx, username)
	}
	if err != nil {
		return res, err
	}
	if m.PasswordHash == "" { // PA-03: ผู้เล่นไม่มีรหัส = รหัสผิด (ไม่ใช้ 407307 ในเส้นนี้)
		utils.CheckPasswordDummy(req.Password)
		return res, loginFailure(ctx, username)
	}
	if !utils.CheckPassword(m.PasswordHash, req.Password) {
		return res, loginFailure(ctx, username)
	}
	if err := playerAuthRedis.ClearLoginFailRepository(ctx, username); err != nil {
		logWarn(ctx, "clear player login fail counter failed", username, err)
	}

	// PA-31: เช็คสถานะหลังรหัสผ่านถูกแล้วเท่านั้น
	st, err := loadMemberState(db, m)
	if err != nil {
		return res, err
	}
	if err := st.gate(); err != nil {
		return res, err
	}

	s, err := issueSession(ctx, db, m.ID, meta.IP)
	if err != nil {
		return res, err
	}
	p := newPlayer(m, st, "")
	return playerAuthDto.LoginResponse{Token: s.Token, ExpiresAt: s.ExpiresAt, Member: memberView(p)}, nil
}

// loginFailure นับรหัสผิด (PA-41) แล้วคืน 407201 เสมอ (ครั้งที่ทำให้ถูกระงับก็ยังบอกแค่ว่ารหัสผิด)
func loginFailure(ctx context.Context, username string) error {
	recordPasswordFailure(ctx, username)
	return apperr.ErrPlayerInvalidCredentials
}

func memberView(p Player) playerAuthDto.MemberView {
	return playerAuthDto.MemberView{Username: p.Username, Name: p.Name, Currency: p.Currency, Status: string(p.Status), CanBet: p.CanBet}
}
