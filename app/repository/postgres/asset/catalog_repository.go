package asset

import (
	"errors"

	"app/app/models"
	"app/pkg/apperr"

	"gorm.io/gorm"
)

// catalog ของช่อง — docs/modules/asset.md AST-11 · repository อ่านอย่างเดียว (เพิ่ม / แก้ด้วย migration)

var assetTypeColumns = []string{"id", "game_code", "asset_type", "allowed_sizes", "is_retired"}

// FindAssetTypeRepository — ช่องตาม (game_code, asset_type) · ไม่พบคืน apperr.ErrNotFound · ไม่กรอง is_retired
func FindAssetTypeRepository(db *gorm.DB, gameCode, assetType string) (models.AssetType, error) {
	var t models.AssetType
	err := db.Select(assetTypeColumns).Where("game_code = ? AND asset_type = ?", gameCode, assetType).Take(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return t, apperr.ErrNotFound
	}
	return t, err
}

// GetAssetTypeRepository — ช่องตาม id · ไม่พบคืน apperr.ErrNotFound
func GetAssetTypeRepository(db *gorm.DB, id uint) (models.AssetType, error) {
	var t models.AssetType
	err := db.Select(assetTypeColumns).Where("id = ?", id).Take(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return t, apperr.ErrNotFound
	}
	return t, err
}

// ListAssetTypesRepository — ช่องของเกม เรียงตามลำดับใน catalog (id) · includeRetired = false ตัดช่องที่เลิกใช้
func ListAssetTypesRepository(db *gorm.DB, gameCode string, includeRetired bool) ([]models.AssetType, error) {
	q := db.Select(assetTypeColumns).Where("game_code = ?", gameCode)
	if !includeRetired {
		q = q.Where("is_retired = false")
	}
	var out []models.AssetType
	err := q.Order("id").Find(&out).Error
	return out, err
}
