// Package cronjob คือ query ของตาราง cronjobs — ทุกฟังก์ชันรับ db เป็น param แรก ไม่เปิด transaction (กฎข้อ 5)
// spec: docs/modules/currency_rate.md หัวข้อ 6.2 (CJ-01 – CJ-06)
package cronjob

import (
	"errors"
	"time"

	"app/app/models"
	"app/pkg/apperr"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// EnsureCronjobRepository — boot: INSERT (code, enabled = ค่าเริ่มต้นในโค้ด) ON CONFLICT DO NOTHING (CJ-02)
// **ไม่ทับค่าที่ตั้งไว้แล้ว** (enabled / updated_by ที่ตั้งด้วย SQL คงเดิม)
func EnsureCronjobRepository(db *gorm.DB, code string, enabled bool) error {
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.Cronjob{Code: code, Enabled: enabled, UpdatedBy: "system"}).Error
}

// GetCronjobRepository — แถวของ job · ไม่พบ = apperr.ErrNotFound
func GetCronjobRepository(db *gorm.DB, code string) (models.Cronjob, error) {
	var c models.Cronjob
	err := db.Where("code = ?", code).Take(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c, apperr.ErrNotFound
	}
	return c, err
}

// RecordCronjobRunRepository — บันทึกผลรอบล่าสุด last_* (CJ-04) · ไม่แตะ enabled / updated_by
func RecordCronjobRunRepository(db *gorm.DB, code string, startedAt, finishedAt time.Time, status models.CronjobStatus, lastError string) error {
	return db.Model(&models.Cronjob{}).Where("code = ?", code).Updates(map[string]any{
		"last_started_at":  startedAt,
		"last_finished_at": finishedAt,
		"last_status":      string(status),
		"last_error":       lastError,
	}).Error
}
