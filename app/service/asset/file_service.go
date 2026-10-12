package asset

import (
	"context"
	"errors"
	"time"

	assetCore "app/app/core/asset"
	assetDto "app/app/internals/backoffice/dto/asset"
	"app/app/models"
	assetPostgres "app/app/repository/postgres/asset"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
	"app/pkg/utils"
	"app/platform/database"
	"app/platform/logger"
	"app/platform/storage"

	"gorm.io/gorm"
)

// cacheControl ของ object — key ไม่ซ้ำทุกครั้งที่อัป จึง cache ยาวได้ (AST-16)
const cacheControl = "public, max-age=31536000, immutable"

// checkImage — ตรวจขนาดไฟล์ + ชนิดจากเนื้อไฟล์ (AST-16) · ไม่ผ่าน = 405103 / 405102
func checkImage(file []byte) (assetCore.ImageInfo, error) {
	if len(file) > assetCore.MaxFileSize {
		return assetCore.ImageInfo{}, apperr.ErrAssetFileTooLarge
	}
	info, err := assetCore.DetectImage(file)
	if errors.Is(err, assetCore.ErrUnsupportedImage) {
		return info, apperr.ErrAssetFileType
	}
	return info, err
}

// putObject — อัปขึ้น storage · ไม่พร้อม = 505801 · อัปไม่สำเร็จ / timeout = 505802 (AST-19)
func putObject(ctx context.Context, key string, file []byte, info assetCore.ImageInfo) error {
	if !storage.Available() {
		return apperr.ErrStorageUnavailable
	}
	if err := storage.Put(ctx, key, file, info.ContentType, cacheControl); err != nil {
		if errors.Is(err, storage.ErrUnavailable) {
			return apperr.ErrStorageUnavailable
		}
		return apperr.ErrStorageUploadFailed.Wrap(err)
	}
	return nil
}

// removeObject — ลบ object ที่อัปแล้วเมื่อบันทึก DB ไม่สำเร็จ (AST-19) · ลบไม่ได้แค่ log พร้อม key (ให้ job กวาดภายหลัง)
func removeObject(ctx context.Context, key string) {
	if err := storage.Delete(ctx, key); err != nil {
		logger.Ctx(ctx).Errorw("asset: cleanup of uploaded object failed", "key", key, "error", err)
	}
}

// newObjectSuffix — ส่วน {rand} ของ storage key: 8 hex จาก crypto/rand (AST-49)
func newObjectSuffix() (string, error) { return utils.RandomString("0123456789abcdef", 8) }

// UploadFileService — POST asset/files/upload (GN3 · AST-14 – AST-21)
// ลำดับ: catalog → เลิกใช้ → ขนาด / ชนิดไฟล์ → ขนาดภาพ → storage → DB (AST-15) · ไม่ผ่านข้อใด = ไม่มีไฟล์ขึ้น bucket
// use_now = true → ตั้ง used ของช่องใน tx เดียวกับ config (AST-21)
func UploadFileService(ctx context.Context, actor agentAuthService.Actor, target assetCore.Target,
	req assetDto.UploadRequest, file []byte) (assetDto.UploadResponse, error) {
	var res assetDto.UploadResponse
	if !target.Resolved {
		return res, errTargetNotResolved
	}
	db := database.DBConn.WithContext(ctx)

	slot, err := findSlot(db, req.GameCode, req.AssetType)
	if err != nil {
		return res, err
	}
	if slot.IsRetired {
		return res, apperr.ErrAssetSlotRetired
	}
	info, err := checkImage(file)
	if err != nil {
		return res, err
	}
	allowed, err := assetCore.ParseAllowedSizes(slot.AllowedSizes)
	if err != nil {
		return res, apperr.ErrInternal.Wrap(err)
	}
	if info.SizeKnown && !assetCore.SizeAllowed(allowed, info.Width, info.Height) { // อ่านขนาดไม่ได้ = ผ่าน (AST-17)
		return res, apperr.ErrAssetImageSize.WithMessage(
			"ขนาดภาพไม่ตรงกับที่ช่องนี้รับ (รับ "+assetCore.FormatSizes(allowed)+")",
			"Image size is not accepted by this slot (accepts "+assetCore.FormatSizes(allowed)+")")
	}

	suffix, err := newObjectSuffix()
	if err != nil {
		return res, err
	}
	key := assetCore.AssetObjectKey(target.Scope, target.AgentID, slot.GameCode, slot.AssetType, time.Now().UnixNano(), suffix, info.Ext())
	if err := putObject(ctx, key, file, info); err != nil {
		return res, err
	}

	cfg := models.AssetConfig{Scope: target.Scope, AgentID: targetAgentID(target), AssetTypeID: slot.ID, StorageKey: key, CreatedBy: actor.Username}
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := assetPostgres.CreateAssetConfigRepository(tx, &cfg); err != nil {
			return err
		}
		if req.UseNowValue {
			return assetPostgres.UpsertAssetUsedRepository(tx, cfg.AgentID, slot.ID, cfg.ID, actor.Username)
		}
		return nil
	})
	if err != nil {
		removeObject(ctx, key)
		return res, err
	}

	var agentID uint
	if cfg.AgentID != nil {
		agentID = *cfg.AgentID
	}
	return assetDto.UploadResponse{ConfigID: cfg.ID, Scope: string(target.Scope), AgentID: agentID, GameCode: slot.GameCode,
		AssetType: slot.AssetType, URL: urlOf(key), IsUsed: req.UseNowValue, CreatedBy: actor.Username, CreatedAt: timeText(time.Now())}, nil
}

// ListFilesService — POST asset/files/list (GN4 · AST-27) · คลังรูปของช่อง: ของเป้าหมาย + ของ GLOBAL ใหม่ → เก่า
func ListFilesService(ctx context.Context, actor agentAuthService.Actor, target assetCore.Target,
	req assetDto.FilesListRequest, page utils.Page) ([]assetDto.FileItem, int64, error) {
	if !target.Resolved {
		return nil, 0, errTargetNotResolved
	}
	db := database.DBConn.WithContext(ctx)
	slot, err := findSlot(db, req.GameCode, req.AssetType)
	if err != nil {
		return nil, 0, err
	}
	rows, total, err := assetPostgres.ListAssetConfigsRepository(db, slot.ID, targetAgentID(target), page)
	if err != nil {
		return nil, 0, err
	}
	items := make([]assetDto.FileItem, len(rows))
	for i, r := range rows {
		source := string(assetCore.SourceGlobal)
		if r.Scope == models.AssetScopeAgent {
			source = string(assetCore.SourceSelf)
		}
		items[i] = assetDto.FileItem{ConfigID: r.ID, URL: urlOf(r.StorageKey), Source: source, IsUsed: r.IsUsed,
			CreatedBy: r.CreatedBy, CreatedAt: timeText(r.CreatedAt)}
	}
	return items, total, nil
}

// configOwnedBy — config เป็นของเป้าหมาย: GLOBAL = config GLOBAL · AGENT = config ของบัญชีนั้นเอง (AST-07, AST-28)
func configOwnedBy(c models.AssetConfig, t assetCore.Target) bool {
	if t.Scope == assetCore.ScopeGlobal {
		return c.Scope == models.AssetScopeGlobal
	}
	return c.Scope == models.AssetScopeAgent && c.AgentID != nil && *c.AgentID == t.AgentID
}

// DeleteFileService — POST asset/files/delete (GN5 · AST-28 – AST-32)
// soft delete ทั้ง GLOBAL และ AGENT · ไม่ลบไฟล์ใน storage (AST-29) · ไม่ผูกกับคนอัป
// ไม่พบ / ไม่ใช่ของเป้าหมาย / ลบแล้ว = 405402 ข้อความเดียวกัน · ยังมี used ชี้อยู่ = 405403
func DeleteFileService(ctx context.Context, actor agentAuthService.Actor, target assetCore.Target, req assetDto.FilesDeleteRequest) error {
	if !target.Resolved {
		return errTargetNotResolved
	}
	return database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		cfg, err := assetPostgres.LockAssetConfigRepository(tx, req.ConfigID) // ล็อกแถวก่อนตัดสิน กันแข่งกับ used/update (AST-28)
		if isNotFound(err) {
			return apperr.ErrAssetConfigNotFound
		}
		if err != nil {
			return err
		}
		if cfg.IsDeleted || !configOwnedBy(cfg, target) {
			return apperr.ErrAssetConfigNotFound
		}
		n, err := assetPostgres.CountAssetUsedByConfigRepository(tx, cfg.ID)
		if err != nil {
			return err
		}
		if n > 0 {
			return apperr.ErrAssetInUse
		}
		return assetPostgres.SoftDeleteAssetConfigRepository(tx, cfg.ID, actor.Username, time.Now())
	})
}
