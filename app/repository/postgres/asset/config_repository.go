package asset

import (
	"errors"
	"time"

	"app/app/models"
	"app/pkg/apperr"
	"app/pkg/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// คลังรูป (asset_configs) — docs/modules/asset.md AST-20, AST-27, AST-28

// CreateAssetConfigRepository — เพิ่มรูปเข้าคลัง 1 เวอร์ชัน (ไม่มีโควตา ไม่มีการตัดเวอร์ชันเก่า — AST-20)
func CreateAssetConfigRepository(db *gorm.DB, c *models.AssetConfig) error {
	return db.Omit("id", "is_deleted", "deleted_at", "deleted_by", "created_at").Create(c).Error
}

var configColumns = []string{"id", "scope", "agent_id", "asset_type_id", "storage_key", "created_by", "is_deleted", "created_at"}

// GetAssetConfigRepository — config ตาม id (ทั้งที่ลบแล้ว) · ไม่พบคืน apperr.ErrNotFound
func GetAssetConfigRepository(db *gorm.DB, id uint) (models.AssetConfig, error) {
	var c models.AssetConfig
	err := db.Select(configColumns).Where("id = ?", id).Take(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c, apperr.ErrNotFound
	}
	return c, err
}

// LockAssetConfigRepository — SELECT ... FOR UPDATE ใช้ใน tx ที่ service เปิด · ทั้ง `files/delete` และ `used/update`
// ล็อกแถวนี้ก่อนตัดสิน จึงชนะได้ฝั่งเดียวเมื่อแข่งกัน (AST-28)
func LockAssetConfigRepository(db *gorm.DB, id uint) (models.AssetConfig, error) {
	return GetAssetConfigRepository(db.Clauses(clause.Locking{Strength: "UPDATE"}), id)
}

// SoftDeleteAssetConfigRepository — ตั้ง is_deleted (ทั้ง GLOBAL และ AGENT — AST-28) · ไม่แตะ storage (AST-29)
func SoftDeleteAssetConfigRepository(db *gorm.DB, id uint, by string, at time.Time) error {
	return db.Model(&models.AssetConfig{}).Where("id = ? AND is_deleted = false", id).
		Updates(map[string]any{"is_deleted": true, "deleted_at": at, "deleted_by": by}).Error
}

// CountAssetUsedByConfigRepository — จำนวนแถว used ที่ชี้มาที่ config (AST-32)
func CountAssetUsedByConfigRepository(db *gorm.DB, configID uint) (int64, error) {
	var n int64
	err := db.Model(&models.AssetUsed{}).Where("asset_config_id = ?", configID).Count(&n).Error
	return n, err
}

// ConfigRow — รูปในคลังพร้อมสถานะว่าเป็นไฟล์ที่เป้าหมายเลือกใช้อยู่ไหม
type ConfigRow struct {
	ID         uint
	Scope      models.AssetScope
	StorageKey string
	CreatedBy  string
	CreatedAt  time.Time
	IsUsed     bool
}

// ListAssetConfigsRepository — คลังรูปของช่อง (AST-27): ของเป้าหมาย + ของ GLOBAL (เมื่อเป้าหมายเป็น AGENT) ที่ยังไม่ถูกลบ
// เรียงใหม่ → เก่า · agentID nil = เป้าหมาย GLOBAL (เฉพาะของ GLOBAL) · IsUsed = แถว used ของเป้าหมายชี้ config นี้
func ListAssetConfigsRepository(db *gorm.DB, assetTypeID uint, agentID *uint, page utils.Page) ([]ConfigRow, int64, error) {
	const where = `c.asset_type_id = @slot AND c.is_deleted = false AND (c.agent_id IS NULL OR c.agent_id = @agent)`
	args := map[string]any{"slot": assetTypeID, "agent": agentOrZero(agentID), "limit": page.Limit, "offset": page.Offset()}

	var total int64
	if err := db.Raw(`SELECT COUNT(*) FROM asset_configs c WHERE `+where, args).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []ConfigRow
	err := db.Raw(`
		SELECT c.id, c.scope, c.storage_key, c.created_by, c.created_at,
		       EXISTS (SELECT 1 FROM asset_used u WHERE u.asset_config_id = c.id
		               AND u.asset_type_id = c.asset_type_id AND u.agent_id IS NOT DISTINCT FROM CAST(NULLIF(@agent, 0) AS BIGINT)) AS is_used
		FROM asset_configs c WHERE `+where+`
		ORDER BY c.created_at DESC, c.id DESC LIMIT @limit OFFSET @offset`, args).Scan(&rows).Error
	return rows, total, err
}

// agentOrZero — แปลง nil (GLOBAL) เป็น 0 เพื่อส่งเป็น parameter · ไม่มี agent id = 0 (ids เริ่มที่ 1)
func agentOrZero(id *uint) uint {
	if id == nil {
		return 0
	}
	return *id
}
