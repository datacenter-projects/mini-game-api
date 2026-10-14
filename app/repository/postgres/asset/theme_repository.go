package asset

import (
	"app/app/models"

	"gorm.io/gorm"
)

// ธีมเกม (game_themes) — docs/modules/asset.md AST-33 – AST-38, AST-55

// ListThemeCandidatesRepository — แถว theme ของเกม ที่เจ้าของอยู่ในสาย หรือเป็น GLOBAL (query เดียว — AST-57)
// chainIDs ว่าง = เป้าหมาย GLOBAL
func ListThemeCandidatesRepository(db *gorm.DB, gameCode string, chainIDs []uint) ([]models.GameTheme, error) {
	q := db.Select("agent_id", "font_family", "main_color_1", "main_color_2", "stats_bar_color", "stats_bar_font_family").
		Where("game_code = ?", gameCode)
	if len(chainIDs) == 0 {
		q = q.Where("agent_id IS NULL")
	} else {
		q = q.Where("(agent_id IS NULL OR agent_id IN ?)", chainIDs)
	}
	var rows []models.GameTheme
	err := q.Find(&rows).Error
	return rows, err
}

// UpsertGameThemeRepository — เขียนทั้ง 5 ค่าแทนแถวเดิม (ไม่มี partial update — AST-33)
// agentID nil = แถวค่ากลาง (GLOBAL) · created_by ตั้งตอนสร้างเท่านั้น
func UpsertGameThemeRepository(db *gorm.DB, agentID *uint, gameCode string, t models.GameTheme, by string) error {
	if agentID == nil {
		return db.Exec(`
			INSERT INTO game_themes (scope, agent_id, game_code, font_family, main_color_1, main_color_2, stats_bar_color, stats_bar_font_family, created_by, updated_by)
			VALUES ('GLOBAL', NULL, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (game_code) WHERE agent_id IS NULL
			DO UPDATE SET font_family = EXCLUDED.font_family, main_color_1 = EXCLUDED.main_color_1, main_color_2 = EXCLUDED.main_color_2,
			              stats_bar_color = EXCLUDED.stats_bar_color, stats_bar_font_family = EXCLUDED.stats_bar_font_family,
			              updated_by = EXCLUDED.updated_by, updated_at = now()`,
			gameCode, t.FontFamily, t.MainColor1, t.MainColor2, t.StatsBarColor, t.StatsBarFontFamily, by, by).Error
	}
	return db.Exec(`
		INSERT INTO game_themes (scope, agent_id, game_code, font_family, main_color_1, main_color_2, stats_bar_color, stats_bar_font_family, created_by, updated_by)
		VALUES ('AGENT', ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (agent_id, game_code) WHERE agent_id IS NOT NULL
		DO UPDATE SET font_family = EXCLUDED.font_family, main_color_1 = EXCLUDED.main_color_1, main_color_2 = EXCLUDED.main_color_2,
		              stats_bar_color = EXCLUDED.stats_bar_color, stats_bar_font_family = EXCLUDED.stats_bar_font_family,
		              updated_by = EXCLUDED.updated_by, updated_at = now()`,
		*agentID, gameCode, t.FontFamily, t.MainColor1, t.MainColor2, t.StatsBarColor, t.StatsBarFontFamily, by, by).Error
}

// DeleteGameThemeRepository — ลบแถว override ของ (บัญชี, เกม) เพื่อกลับไปใช้ของชั้นบน (AST-55) · ไม่มีแถว = ไม่ทำอะไร
func DeleteGameThemeRepository(db *gorm.DB, agentID uint, gameCode string) error {
	return db.Where("agent_id = ? AND game_code = ?", agentID, gameCode).Delete(&models.GameTheme{}).Error
}
