package playerauth

import (
	"context"
	"errors"
	"time"

	agentManagementCore "app/app/core/agent_management"
	gameCore "app/app/core/game"
	playerAuthDto "app/app/internals/frontend/dto/player_auth"
	agentManagementPostgres "app/app/repository/postgres/agent_management"
	playerAuthPostgres "app/app/repository/postgres/player_auth"
	playerAuthRedis "app/app/repository/redis/player_auth"
	assetService "app/app/service/asset"
	"app/pkg/apperr"
	"app/pkg/utils"
	"app/platform/database"
	"app/platform/logger"

	"gorm.io/gorm"
)

// PasswordUpdateService — PA4 POST /api/v1/player/pr/auth/password/update (PA-50, PA-51, PA-41)
// รหัสเดิมผิดนับรวมตัวนับเดียวกับ login · ครบ limit → 407303 และตัด session · สำเร็จแล้ว session ปัจจุบันใช้ต่อ (1 session ต่อคน — Q10)
func PasswordUpdateService(ctx context.Context, p Player, req playerAuthDto.PasswordUpdateRequest) error {
	blocked, err := playerAuthRedis.IsLoginBlockedRepository(ctx, p.Username) // PA-41: ตัวนับ / บล็อกตัวเดียวกับ login
	if err != nil {
		return err
	}
	if blocked {
		return apperr.ErrPlayerLoginBlocked
	}
	newHash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	err = database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		m, err := playerAuthPostgres.LockMemberCredentialsRepository(tx, p.MemberID)
		if err != nil {
			return err
		}
		if m.PasswordHash == "" { // PA-51: ผู้เล่นที่ไม่มีรหัสเปลี่ยนรหัสไม่ได้
			return apperr.ErrPlayerNoPassword
		}
		if !utils.CheckPassword(m.PasswordHash, req.OldPassword) {
			if recordPasswordFailure(ctx, m.Username) {
				return errPasswordBlocked
			}
			return apperr.ErrPlayerOldPasswordIncorrect
		}
		if utils.CheckPassword(m.PasswordHash, req.NewPassword) { // ต้องไม่ซ้ำรหัสเดิม
			return apperr.ErrValidation.WithMessage("new_password ต้องไม่ซ้ำกับรหัสผ่านเดิม", "new_password must differ from the current password")
		}
		return playerAuthPostgres.UpdateMemberPasswordRepository(tx, p.MemberID, newHash, time.Now())
	})
	if errors.Is(err, errPasswordBlocked) { // ครบ limit: ตัด session + บอกว่าถูกระงับ (ทำนอก tx เพื่อไม่ถูก rollback)
		endSession(ctx, p.SessionID, p.MemberID)
		return apperr.ErrPlayerLoginBlocked
	}
	if err != nil {
		return err
	}
	if err := playerAuthRedis.ClearLoginFailRepository(ctx, p.Username); err != nil {
		logWarn(ctx, "clear player login fail counter failed", p.Username, err)
	}
	return nil
}

// errPasswordBlocked — สัญญาณภายใน: รหัสเดิมผิดจนครบ limit (ให้ tx rollback ก่อนแล้วค่อยตอบ 407303)
var errPasswordBlocked = errors.New("player password attempts exceeded")

// MeService — PA5 POST /api/v1/player/pr/me/get (PA-54) · ไม่มี credit (ใช้ BalanceService) · ข้อมูลมาจาก middleware แล้ว ไม่ query เพิ่ม
func MeService(p Player) playerAuthDto.MemberView { return memberView(p) }

// BalanceService — PA6 POST /api/v1/player/pr/balance/get (PA-56) · อ่าน user_members.credit ตรง ไม่ cache (กฎข้อ 14)
// หน้าเกมเรียกทุก 5–10 นาทีเป็น ping กันหลุด idle (PA-21 — ต่ออายุที่ middleware)
func BalanceService(ctx context.Context, p Player) (playerAuthDto.BalanceResponse, error) {
	m, err := playerAuthPostgres.GetMemberBalanceRepository(database.DBConn.WithContext(ctx), p.MemberID)
	if errors.Is(err, apperr.ErrNotFound) {
		return playerAuthDto.BalanceResponse{}, apperr.ErrPlayerSessionEnded
	}
	if err != nil {
		return playerAuthDto.BalanceResponse{}, err
	}
	return playerAuthDto.BalanceResponse{Currency: m.Currency, Credit: m.Credit}, nil
}

// GamesListService — PA7 POST /api/v1/player/pr/games/list (PA-55 · PA-60)
// เกมทั้งหมดเรียงตาม catalog คงที่ในโค้ด · open = เปิดจริงตาม agent + สายบน (Member ไม่มีสวิตช์รายคน · ไม่มีแถว = เปิด) ·
// ปก = ช่อง game_cover ไล่สายจากผู้สร้างตรง ผ่านกฎเดียวกับ BO (asset AST-51) · ไม่มีปก / อ่านปกไม่สำเร็จ = ""
func GamesListService(ctx context.Context, p Player) (playerAuthDto.GamesListResponse, error) {
	db := database.DBConn.WithContext(ctx)
	settings, err := agentManagementPostgres.ListAgentGameSettingsByIDsRepository(db, p.UplineIDs)
	if err != nil {
		return playerAuthDto.GamesListResponse{}, err
	}
	statuses := map[string][]bool{}
	for _, s := range settings {
		statuses[s.GameCode] = append(statuses[s.GameCode], s.StatusGame)
	}
	covers, err := assetService.GameCoversOfAgentService(ctx, p.AgentID)
	if err != nil {
		logger.Ctx(ctx).Warnw("player games/list: read covers failed", "member_id", p.MemberID, "error", err)
		covers = map[string]string{}
	}
	games := gameCore.All()
	items := make([]playerAuthDto.GameItem, 0, len(games))
	for _, g := range games {
		th, en, _ := gameCore.NameOf(g.GameCode)
		items = append(items, playerAuthDto.GameItem{GameCode: g.GameCode, NameTH: th, NameEN: en,
			Open: agentManagementCore.IsGameOpen(statuses[g.GameCode]), GameCover: covers[g.GameCode]})
	}
	return playerAuthDto.GamesListResponse{Items: items}, nil
}
