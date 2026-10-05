package postgres

import (
	"app/app/models"

	"gorm.io/gorm"
)

// CreateAuthAuditLogRepository — เรียกใน transaction เดียวกับการรีเซ็ต (AUTH-49)
func CreateAuthAuditLogRepository(db *gorm.DB, l *models.AuthAuditLog) error {
	return db.Create(l).Error
}
