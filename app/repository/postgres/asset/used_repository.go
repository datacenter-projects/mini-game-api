package asset

import (
	"app/app/models"

	"gorm.io/gorm"
)

// ไฟล์ที่ใช้งานจริงต่อช่อง (asset_used) — docs/modules/asset.md AST-22, AST-24, AST-54, AST-57

// UpsertAssetUsedRepository — ตั้งไฟล์ใช้งานของ (เป้าหมาย, ช่อง) = แก้แถวเดิม ไม่ใช่เพิ่มแถว (AST-22)
// agentID nil = GLOBAL (partial unique index แยกจาก AGENT เพราะ Postgres มอง NULL ไม่เท่ากัน)
// ต้องเรียกใน tx ที่ล็อกแถว config แล้ว (AST-28)
func UpsertAssetUsedRepository(db *gorm.DB, agentID *uint, assetTypeID, configID uint, by string) error {
	if agentID == nil {
		return db.Exec(`
			INSERT INTO asset_used (scope, agent_id, asset_type_id, asset_config_id, created_by, updated_by)
			VALUES ('GLOBAL', NULL, ?, ?, ?, ?)
			ON CONFLICT (asset_type_id) WHERE agent_id IS NULL
			DO UPDATE SET asset_config_id = EXCLUDED.asset_config_id, updated_by = EXCLUDED.updated_by, updated_at = now()`,
			assetTypeID, configID, by, by).Error
	}
	return db.Exec(`
		INSERT INTO asset_used (scope, agent_id, asset_type_id, asset_config_id, created_by, updated_by)
		VALUES ('AGENT', ?, ?, ?, ?, ?)
		ON CONFLICT (agent_id, asset_type_id) WHERE agent_id IS NOT NULL
		DO UPDATE SET asset_config_id = EXCLUDED.asset_config_id, updated_by = EXCLUDED.updated_by, updated_at = now()`,
		*agentID, assetTypeID, configID, by, by).Error
}

// DeleteAssetUsedRepository — ลบแถว used ของ (บัญชี, ช่อง) เพื่อกลับไปใช้ของชั้นบน (AST-54) · ไม่มีแถว = ไม่ทำอะไร
func DeleteAssetUsedRepository(db *gorm.DB, agentID, assetTypeID uint) error {
	return db.Where("agent_id = ? AND asset_type_id = ?", agentID, assetTypeID).Delete(&models.AssetUsed{}).Error
}

// UsedRow — แถว used ที่เป็นไปได้ของช่องหนึ่ง พร้อมไฟล์
type UsedRow struct {
	AssetTypeID   uint
	AgentID       *uint // nil = GLOBAL
	AssetConfigID uint
	StorageKey    string
}

// ListUsedCandidatesRepository — แถว used ทุกช่องของเกม ที่เจ้าของอยู่ในสาย หรือเป็น GLOBAL (query เดียว — AST-57)
// chainIDs ว่าง = เป้าหมาย GLOBAL (เอาเฉพาะแถว GLOBAL)
func ListUsedCandidatesRepository(db *gorm.DB, gameCode string, chainIDs []uint) ([]UsedRow, error) {
	q := db.Table("asset_used u").
		Select("u.asset_type_id, u.agent_id, u.asset_config_id, c.storage_key").
		Joins("JOIN asset_types t ON t.id = u.asset_type_id").
		Joins("JOIN asset_configs c ON c.id = u.asset_config_id").
		Where("t.game_code = ?", gameCode)
	if len(chainIDs) == 0 {
		q = q.Where("u.agent_id IS NULL")
	} else {
		q = q.Where("(u.agent_id IS NULL OR u.agent_id IN ?)", chainIDs)
	}
	var rows []UsedRow
	err := q.Scan(&rows).Error
	return rows, err
}

// HasGlobalUsedRepository — ช่องนี้มี used ของ GLOBAL แล้วไหม (สคริปต์อัปรูปค่ากลาง ข้ามช่องที่มีแล้ว — AST-61)
func HasGlobalUsedRepository(db *gorm.DB, assetTypeID uint) (bool, error) {
	var n int64
	err := db.Model(&models.AssetUsed{}).Where("asset_type_id = ? AND agent_id IS NULL", assetTypeID).Count(&n).Error
	return n > 0, err
}

// SlotRow — ช่องใน catalog (ชื่อเกม + ชื่อช่อง)
type SlotRow struct {
	GameCode  string
	AssetType string
}

// ListSlotsWithoutGlobalUsedRepository — ช่องที่ยังใช้งานอยู่ (ไม่ใช่ที่เลิกใช้) ซึ่ง GLOBAL ยังไม่มี used · เรียงตาม catalog (AST-61)
func ListSlotsWithoutGlobalUsedRepository(db *gorm.DB) ([]SlotRow, error) {
	var rows []SlotRow
	err := db.Raw(`
		SELECT t.game_code, t.asset_type FROM asset_types t
		WHERE t.is_retired = false
		  AND NOT EXISTS (SELECT 1 FROM asset_used u WHERE u.asset_type_id = t.id AND u.agent_id IS NULL)
		ORDER BY t.id`).Scan(&rows).Error
	return rows, err
}
