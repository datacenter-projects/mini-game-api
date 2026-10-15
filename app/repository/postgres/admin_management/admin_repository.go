package adminmanagement

import (
	"app/app/models"

	"gorm.io/gorm"
)

// repository ของ module admin_management — เส้นค้นหาบัญชีของ ADMIN (MGMT-27B) · username ตรงทั้งคำ · ไม่พบ = ผลว่าง (id 0)

// FindUserAgentByUsernameRepository — บัญชีฝั่ง agent ทุก role รวม SUPERADMIN / ADMIN
func FindUserAgentByUsernameRepository(db *gorm.DB, username string) (models.UserAgent, error) {
	var a models.UserAgent
	err := db.Select("id", "parent_id", "username", "role", "agent_type", "status", "last_login_at", "last_login_ip", "created_at").
		Where("username = ?", username).Limit(1).Find(&a).Error
	return a, err
}

// FindSubaccountByUsernameRepository — บัญชีย่อย (username = owner@name)
func FindSubaccountByUsernameRepository(db *gorm.DB, username string) (models.Subaccount, error) {
	var s models.Subaccount
	err := db.Select("id", "agent_id", "username", "status", "last_login_at", "last_login_ip", "created_at").
		Where("username = ?", username).Limit(1).Find(&s).Error
	return s, err
}
