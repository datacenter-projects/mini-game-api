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
// ปกเกม = ช่อง game_cover ที่ไล่สายจากผู้เรียก ผ่าน resolveUsed ตัวเดียวกับ used/list (lead GE1-1)
// ไม่กรองเกมที่ status_game ปิด (lead GE1-2) · เกมที่ไม่มีปก / อ่านปกไม่สำเร็จ = "" โดยไม่ทำให้ทั้งเส้นล้ม
func GamesListService(ctx context.Context, actor agentAuthService.Actor) ([]assetDto.GameItem, error) {
	db := database.DBConn.WithContext(ctx)
	owner, err := loadOwner(db, actor)
	if err != nil {
		return nil, err
	}
	chain, err := chainOf(db, owner.ID)
	if err != nil {
		return nil, err
	}
	games := assetCore.GameCodes()
	out := make([]assetDto.GameItem, 0, len(games))
	for _, game := range games {
		item := assetDto.GameItem{GameCode: game}
		cover, err := gameCover(db, game, chain)
		if err != nil {
			logger.Ctx(ctx).Warnw("asset games/list: read cover failed", "game", game, "error", err)
		} else {
			item.GameCover = cover
		}
		out = append(out, item)
	}
	return out, nil
}

func gameCover(db *gorm.DB, game string, chain []uint) (string, error) {
	slot, err := assetPostgres.FindAssetTypeRepository(db, game, assetCore.ChannelGameCover)
	if err != nil {
		if isNotFound(err) {
			return "", nil
		}
		return "", err
	}
	resolved, err := resolveUsed(db, game, chain)
	if err != nil {
		return "", err
	}
	if r, ok := resolved[slot.ID]; ok {
		return urlOf(r.StorageKey), nil
	}
	return "", nil
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
