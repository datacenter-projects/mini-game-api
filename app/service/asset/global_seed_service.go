package asset

import (
	"context"

	assetPostgres "app/app/repository/postgres/asset"
	"app/platform/database"
)

// สำหรับสคริปต์อัปรูปค่ากลาง (scripts/seed_asset_global — AST-61): อัปรูปจริงใช้ UploadFileService ตัวเดียวกับ API

// HasGlobalUsedService — ช่องนี้มี used ของ GLOBAL แล้วไหม · ไม่พบช่อง = 405401
func HasGlobalUsedService(ctx context.Context, gameCode, assetType string) (bool, error) {
	db := database.DBConn.WithContext(ctx)
	slot, err := findSlot(db, gameCode, assetType)
	if err != nil {
		return false, err
	}
	return assetPostgres.HasGlobalUsedRepository(db, slot.ID)
}

// GlobalGap — ช่องที่ GLOBAL ยังไม่มีรูป
type GlobalGap struct {
	GameCode  string
	AssetType string
}

// ListGlobalGapsService — ช่องของทุกเกมที่ GLOBAL ยังไม่มีรูป (ไม่นับช่องที่เลิกใช้) — ก่อน UAT ต้องว่าง (AST-24)
func ListGlobalGapsService(ctx context.Context) ([]GlobalGap, error) {
	rows, err := assetPostgres.ListSlotsWithoutGlobalUsedRepository(database.DBConn.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	out := make([]GlobalGap, len(rows))
	for i, r := range rows {
		out[i] = GlobalGap{GameCode: r.GameCode, AssetType: r.AssetType}
	}
	return out, nil
}
