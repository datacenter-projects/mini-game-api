package postgres

import (
	"errors"
	"time"

	"app/app/models"
	"app/pkg/apperr"

	"gorm.io/gorm"
)

// คอลัมน์ที่ auth ใช้ — ไม่ดึง password_hash ในจุดที่ไม่ต้องใช้
var userAgentAuthColumns = []string{"id", "parent_id", "username", "role", "status"}

// GetUserAgentForLoginRepository — ไม่พบคืน apperr.ErrNotFound
func GetUserAgentForLoginRepository(db *gorm.DB, username string) (models.UserAgent, error) {
	var a models.UserAgent
	err := db.Select(append(userAgentAuthColumns, "password_hash")).
		Where("username = ?", username).
		Take(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return a, apperr.ErrNotFound
	}
	return a, err
}

// GetUserAgentAuthByIDRepository — ใช้ใน middleware ทุก request (primary key lookup) · ไม่พบคืน apperr.ErrNotFound
func GetUserAgentAuthByIDRepository(db *gorm.DB, id uint) (models.UserAgent, error) {
	var a models.UserAgent
	err := db.Select(userAgentAuthColumns).Where("id = ?", id).Take(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return a, apperr.ErrNotFound
	}
	return a, err
}

// HasLockedAncestorRepository — มี upline คนใด (ไม่นับตัวเอง) เป็น LOCKED หรือไม่
// ไล่สายด้วย recursive CTE ใน query เดียว (ไม่ N+1)
func HasLockedAncestorRepository(db *gorm.DB, agentID uint) (bool, error) {
	var found bool
	err := db.Raw(`
		WITH RECURSIVE ancestors AS (
			SELECT parent_id FROM user_agents WHERE id = ?
			UNION -- ไม่ใช้ UNION ALL: ถ้าข้อมูลสายวนเป็นลูป query จะหยุดเองไม่ค้าง
			SELECT ua.parent_id FROM user_agents ua JOIN ancestors a ON ua.id = a.parent_id
		)
		SELECT EXISTS (
			SELECT 1 FROM user_agents ua JOIN ancestors a ON ua.id = a.parent_id
			WHERE ua.status = ?
		)`, agentID, models.AgentStatusLocked).Scan(&found).Error
	return found, err
}

func UpdateUserAgentLastLoginRepository(db *gorm.DB, id uint, ip string, at time.Time) error {
	return db.Model(&models.UserAgent{}).Where("id = ?", id).
		Updates(map[string]any{"last_login_at": at, "last_login_ip": ip, "updated_at": at}).Error
}

func CreateUserAgentRepository(db *gorm.DB, a *models.UserAgent) error {
	err := db.Create(a).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return apperr.ErrConflict.Wrap(err)
	}
	return err
}
