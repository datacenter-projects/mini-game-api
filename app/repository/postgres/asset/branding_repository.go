package asset

import (
	"app/app/models"

	"gorm.io/gorm"
)

// branding ของบัญชี (site_brandings) — docs/modules/asset.md AST-41 – AST-43, AST-47, AST-56

// ListBrandingCandidatesRepository — แถว branding ของบัญชีในสาย (query เดียว — AST-57) · ไม่มี GLOBAL
func ListBrandingCandidatesRepository(db *gorm.DB, chainIDs []uint) ([]models.SiteBranding, error) {
	if len(chainIDs) == 0 {
		return nil, nil
	}
	var rows []models.SiteBranding
	err := db.Select("agent_id", "site_name", "logo_key").Where("agent_id IN ?", chainIDs).Find(&rows).Error
	return rows, err
}

// UpsertSiteBrandingNameRepository — ตั้งชื่อเว็บ · สร้างแถวถ้ายังไม่มี (โลโก้ว่าง) · ไม่แตะโลโก้ (AST-42)
func UpsertSiteBrandingNameRepository(db *gorm.DB, agentID uint, siteName, by string) error {
	return db.Exec(`
		INSERT INTO site_brandings (agent_id, site_name, created_by, updated_by) VALUES (?, ?, ?, ?)
		ON CONFLICT (agent_id) DO UPDATE SET site_name = EXCLUDED.site_name, updated_by = EXCLUDED.updated_by, updated_at = now()`,
		agentID, siteName, by, by).Error
}

// UpsertSiteBrandingLogoRepository — ตั้ง key ของโลโก้ (เขียนทับ — AST-43) · สร้างแถวถ้ายังไม่มี (ชื่อว่าง) · ไม่แตะชื่อ
func UpsertSiteBrandingLogoRepository(db *gorm.DB, agentID uint, logoKey, by string) error {
	return db.Exec(`
		INSERT INTO site_brandings (agent_id, logo_key, created_by, updated_by) VALUES (?, ?, ?, ?)
		ON CONFLICT (agent_id) DO UPDATE SET logo_key = EXCLUDED.logo_key, updated_by = EXCLUDED.updated_by, updated_at = now()`,
		agentID, logoKey, by, by).Error
}

// DeleteSiteBrandingRepository — ลบแถว branding ของบัญชีเพื่อกลับไปใช้ของชั้นบน (AST-56) · ไม่มีแถว = ไม่ทำอะไร
func DeleteSiteBrandingRepository(db *gorm.DB, agentID uint) error {
	return db.Where("agent_id = ?", agentID).Delete(&models.SiteBranding{}).Error
}
