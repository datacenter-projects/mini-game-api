package asset

import (
	"context"

	assetCore "app/app/core/asset"
	assetDto "app/app/internals/backoffice/dto/asset"
	assetPostgres "app/app/repository/postgres/asset"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
	"app/platform/database"
	"app/platform/logger"

	"gorm.io/gorm"
)

// GamesListService — POST asset/games/list (GN1 · AST-51)
// ปกเกม = ช่อง game_cover ที่ไล่สายจากผู้เรียก ด้วยกฎเดียวกับ used/list (core.ResolveUsed — lead GE1-1) · ทุกเกมใน query เดียว
// ไม่กรองเกมที่ status_game ปิด (lead GE1-2) · เกมที่ไม่มีปก / อ่านปกไม่สำเร็จ = "" โดยไม่ทำให้ทั้งเส้นล้ม
func GamesListService(ctx context.Context, actor agentAuthService.Actor) ([]assetDto.GameItem, error) {
	db := database.DBConn.WithContext(ctx)
	chain, err := chainOf(db, actor.AgentID) // บัญชีหลักที่ request ทำงานในนาม (sub = เจ้าของ — AUTH-26) · ไม่พบ = session สิ้นสุด
	if err != nil {
		return nil, err
	}
	covers, err := gameCovers(db, chain)
	if err != nil {
		logger.Ctx(ctx).Warnw("asset games/list: read covers failed", "error", err)
	}
	games := assetCore.GameCodes()
	out := make([]assetDto.GameItem, len(games))
	for i, game := range games {
		out[i] = assetDto.GameItem{GameCode: game, GameCover: covers[game]}
	}
	return out, nil
}

// gameCovers — URL ปกที่ใช้งานจริงของทุกเกม (game_code → url) · เกมที่ไม่มีปกไม่อยู่ใน map
func gameCovers(db *gorm.DB, chain []uint) (map[string]string, error) {
	rows, err := assetPostgres.ListUsedCandidatesByTypeRepository(db, assetCore.ChannelGameCover, chain)
	if err != nil {
		return nil, err
	}
	gameOf := make(map[uint]string, len(rows))
	cands := make([]assetCore.UsedCandidate, len(rows))
	for i, r := range rows {
		gameOf[r.AssetTypeID] = r.GameCode
		cands[i] = assetCore.UsedCandidate{SlotID: r.AssetTypeID, OwnerID: r.AgentID, ConfigID: r.AssetConfigID, StorageKey: r.StorageKey}
	}
	out := make(map[string]string, len(gameOf))
	for slotID, r := range assetCore.ResolveUsed(chain, cands) {
		out[gameOf[slotID]] = urlOf(r.StorageKey)
	}
	return out, nil
}

// SlotsListService — POST asset/slots/list (GN2 · AST-12) · ไม่รวมช่องที่เลิกใช้ · ไม่มี description (lead GE2-1)
func SlotsListService(ctx context.Context, gameCode string) ([]assetDto.SlotItem, error) {
	if err := requireGame(gameCode); err != nil {
		return nil, err
	}
	types, err := assetPostgres.ListAssetTypesRepository(database.DBConn.WithContext(ctx), gameCode, false)
	if err != nil {
		return nil, err
	}
	out := make([]assetDto.SlotItem, 0, len(types))
	for _, t := range types {
		sizes, err := assetCore.ParseAllowedSizes(t.AllowedSizes)
		if err != nil {
			return nil, apperr.ErrInternal.Wrap(err)
		}
		out = append(out, assetDto.SlotItem{AssetType: t.AssetType, AllowedSizes: sizeItems(sizes)})
	}
	return out, nil
}

func sizeItems(sizes []assetCore.Size) []assetDto.SizeItem {
	out := make([]assetDto.SizeItem, len(sizes))
	for i, s := range sizes {
		out[i] = assetDto.SizeItem{Width: s.Width, Height: s.Height}
	}
	return out
}

// GameCoversOfAgentService — URL ปกที่ใช้งานจริงของทุกเกม ไล่สายจาก agent นี้ (game_code → url · เกมไม่มีปกไม่อยู่ใน map)
// ใช้โดย player_auth games/list (PA-55): ผู้เล่นเห็นปกแบบเดียวกับที่ BO preview ของผู้สร้างตรงเห็น (AST-51 / AST-57) ผ่านกฎเดียวกับ used/list
// อ่านปกไม่สำเร็จ = map ว่าง + log warn (ปกเป็นส่วนเสริม ไม่ทำให้ทั้งเส้นล้ม — AST-46)
func GameCoversOfAgentService(ctx context.Context, agentID uint) (map[string]string, error) {
	db := database.DBConn.WithContext(ctx)
	chain, err := chainOf(db, agentID)
	if err != nil {
		return nil, err
	}
	covers, err := gameCovers(db, chain)
	if err != nil {
		logger.Ctx(ctx).Warnw("asset covers of agent failed", "agent_id", agentID, "error", err)
		return map[string]string{}, nil
	}
	return covers, nil
}
