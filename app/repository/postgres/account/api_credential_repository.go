package account

import (
	"errors"

	"app/app/models"
	"app/pkg/apperr"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ข้อมูลรับรอง API — docs/modules/account.md หัวข้อ 6

// GetAPICredentialRepository — ไม่พบคืน apperr.ErrNotFound
func GetAPICredentialRepository(db *gorm.DB, agentID uint) (models.APICredential, error) {
	var c models.APICredential
	err := db.Select("agent_id", "api_key", "callback_url").Where("agent_id = ?", agentID).Take(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c, apperr.ErrNotFound
	}
	return c, err
}

// LockAPICredentialRepository — SELECT ... FOR UPDATE ใช้ใน tx ที่ service เปิด (ACC-08) · ไม่พบคืน apperr.ErrNotFound
func LockAPICredentialRepository(db *gorm.DB, agentID uint) (models.APICredential, error) {
	return GetAPICredentialRepository(db.Clauses(clause.Locking{Strength: "UPDATE"}), agentID)
}

// CreateAPICredentialIfAbsentRepository — สร้างแถว Key ถ้ายังไม่มี (ACC-05) · มีอยู่แล้วไม่ทำอะไร
// เรียกพร้อมกันได้ Key เดียวเสมอ เพราะ agent_id เป็น primary key
func CreateAPICredentialIfAbsentRepository(db *gorm.DB, c *models.APICredential) error {
	return db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "agent_id"}}, DoNothing: true}).
		Select("agent_id", "api_key").Create(c).Error
}

// UpdateAPICredentialCallbackRepository — callbackURL nil = ยังไม่ตั้ง
func UpdateAPICredentialCallbackRepository(db *gorm.DB, agentID uint, callbackURL *string) error {
	return db.Model(&models.APICredential{}).Where("agent_id = ?", agentID).
		Updates(map[string]any{"callback_url": callbackURL, "updated_at": gorm.Expr("now()")}).Error
}

// ListAPIAllowedIPsRepository — เรียงตามลำดับที่บันทึก
func ListAPIAllowedIPsRepository(db *gorm.DB, agentID uint) ([]string, error) {
	var cidrs []string
	err := db.Model(&models.APIAllowedIP{}).Where("agent_id = ?", agentID).Order("id").Pluck("cidr", &cidrs).Error
	return cidrs, err
}

// ReplaceAPIAllowedIPsRepository — แทนรายการ IP ทั้งชุด (ACC-08) · ต้องเรียกใน tx ที่ service เปิด
func ReplaceAPIAllowedIPsRepository(db *gorm.DB, agentID uint, cidrs []string) error {
	if err := db.Where("agent_id = ?", agentID).Delete(&models.APIAllowedIP{}).Error; err != nil {
		return err
	}
	if len(cidrs) == 0 {
		return nil
	}
	rows := make([]models.APIAllowedIP, len(cidrs))
	for i, c := range cidrs {
		rows[i] = models.APIAllowedIP{AgentID: agentID, CIDR: c}
	}
	return db.Select("agent_id", "cidr").Create(&rows).Error
}

// CreateAPICredentialLogRepository — ประวัติการบันทึก (ACC-09)
func CreateAPICredentialLogRepository(db *gorm.DB, l *models.APICredentialLog) error {
	return db.Omit("id", "created_at").Create(l).Error
}
