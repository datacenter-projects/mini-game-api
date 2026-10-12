package asset

import (
	"context"

	assetCore "app/app/core/asset"
	assetDto "app/app/internals/backoffice/dto/asset"
	"app/app/models"
	assetPostgres "app/app/repository/postgres/asset"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
	"app/platform/database"

	"gorm.io/gorm"
)

// ListUsedService — POST asset/used/list (GN6 · AST-24, AST-25, lead GE6-1)
// ส่ง**ทุกช่อง**ของเกม ไล่สายแล้ว · ช่องที่ไม่มีไฟล์ในสายและ GLOBAL = url "" · ช่องเลิกใช้แสดงเฉพาะที่ยังมีไฟล์ (AST-13)
func ListUsedService(ctx context.Context, actor agentAuthService.Actor, target assetCore.Target, req assetDto.GameRequest) (assetDto.UsedListResponse, error) {
	var res assetDto.UsedListResponse
	if err := requireGame(req.GameCode); err != nil {
		return res, err
	}
	db := database.DBConn.WithContext(ctx)
	chain, err := targetChain(db, target)
	if err != nil {
		return res, err
	}
	types, err := assetPostgres.ListAssetTypesRepository(db, req.GameCode, true)
	if err != nil {
		return res, err
	}
	resolved, err := resolveUsed(db, req.GameCode, chain)
	if err != nil {
		return res, err
	}

	var uplineIDs []uint
	for _, r := range resolved {
		if r.Source == assetCore.SourceUpline {
			uplineIDs = append(uplineIDs, r.InheritedFrom)
		}
	}
	names, err := usernames(db, uplineIDs)
	if err != nil {
		return res, err
	}

	items := make([]assetDto.UsedItem, 0, len(types))
	var sources []assetCore.Source
	for _, t := range types {
		r, ok := resolved[t.ID]
		if !ok {
			if !t.IsRetired {
				items = append(items, assetDto.UsedItem{AssetType: t.AssetType}) // ช่องว่าง: url "" · config_id 0 (ACC-32)
			}
			continue
		}
		sources = append(sources, r.Source)
		items = append(items, assetDto.UsedItem{AssetType: t.AssetType, ConfigID: r.ConfigID, URL: urlOf(r.StorageKey),
			Source: string(r.Source), InheritedFrom: names[r.InheritedFrom], IsRetired: t.IsRetired})
	}
	return assetDto.UsedListResponse{Source: string(assetCore.OverallSource(sources)), Items: items}, nil
}

// configUsableBy — config ที่เป้าหมายเลือกใช้ได้ (AST-23 ข้อ ค): GLOBAL เลือกได้เฉพาะ config GLOBAL ·
// AGENT เลือกได้ config GLOBAL หรือของบัญชีนั้นเอง (ไม่เลือกของชั้นบน — ใช้การสืบทอดแทน)
func configUsableBy(c models.AssetConfig, t assetCore.Target) bool {
	if c.Scope == models.AssetScopeGlobal {
		return true
	}
	return t.Scope == assetCore.ScopeAgent && c.AgentID != nil && *c.AgentID == t.AgentID
}

// UpdateUsedService — POST asset/used/update (GN7 · AST-23)
// ผิดข้อ (ก) ไม่พบ / ลบแล้ว (ข) ไม่ใช่เกมที่ส่ง (ค) ไม่ใช่ของที่เป้าหมายเลือกได้ = 405402 ข้อความเดียวกัน (กันไล่เดา id)
// ตรวจความเป็นเจ้าของก่อนบอกว่าช่องเลิกใช้ (405304) — ไม่เปิดเผยสถานะช่องของ config ที่ไม่ใช่ของตัวเอง
func UpdateUsedService(ctx context.Context, actor agentAuthService.Actor, target assetCore.Target, req assetDto.UsedUpdateRequest) error {
	if !target.Resolved {
		return errTargetNotResolved
	}
	if err := requireGame(req.GameCode); err != nil {
		return err
	}
	return database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		cfg, err := assetPostgres.LockAssetConfigRepository(tx, req.ConfigID) // ล็อกแถวเดียวกับ files/delete (AST-28)
		if isNotFound(err) {
			return apperr.ErrAssetConfigNotFound
		}
		if err != nil {
			return err
		}
		if cfg.IsDeleted || !configUsableBy(cfg, target) {
			return apperr.ErrAssetConfigNotFound
		}
		slot, err := assetPostgres.GetAssetTypeRepository(tx, cfg.AssetTypeID)
		if err != nil {
			return err
		}
		if slot.GameCode != req.GameCode {
			return apperr.ErrAssetConfigNotFound
		}
		if slot.IsRetired {
			return apperr.ErrAssetSlotRetired
		}
		return assetPostgres.UpsertAssetUsedRepository(tx, targetAgentID(target), slot.ID, cfg.ID, actor.Username)
	})
}

// ResetUsedService — POST asset/used/reset (GN8 · AST-54): ลบแถว used ของ (บัญชีตัวเอง, ช่อง) กลับไปใช้ของชั้นบน / GLOBAL
// ใช้ได้เฉพาะ scope AGENT · ไม่มีแถวอยู่แล้ว = สำเร็จ · ไม่ลบ config (ยังอยู่ในคลัง)
func ResetUsedService(ctx context.Context, actor agentAuthService.Actor, target assetCore.Target, req assetDto.UsedResetRequest) error {
	if !target.Resolved {
		return errTargetNotResolved
	}
	if target.Scope != assetCore.ScopeAgent {
		return resetGlobalNotAllowed()
	}
	db := database.DBConn.WithContext(ctx)
	slot, err := findSlot(db, req.GameCode, req.AssetType)
	if err != nil {
		return err
	}
	return assetPostgres.DeleteAssetUsedRepository(db, target.AgentID, slot.ID)
}

func resetGlobalNotAllowed() error {
	return apperr.ErrValidation.WithMessage("reset ใช้ได้เฉพาะ scope AGENT (ค่ากลางไม่มีชั้นบนให้กลับไปใช้)",
		"reset only works on scope AGENT (the global value has nothing above it)")
}
