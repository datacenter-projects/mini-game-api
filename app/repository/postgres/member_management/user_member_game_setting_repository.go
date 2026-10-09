package membermanagement

import (
	"time"

	"app/app/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// repository ของตาราง user_member_game_settings — docs/modules/agent_management.md หัวข้อ 6

func CreateUserMemberGameSettingsRepository(db *gorm.DB, rows []models.UserMemberGameSetting) error {
	if len(rows) == 0 {
		return nil
	}
	return db.Create(&rows).Error
}

func ListUserMemberGameSettingsByIDsRepository(db *gorm.DB, ids []uint) ([]models.UserMemberGameSetting, error) {
	var out []models.UserMemberGameSetting
	if len(ids) == 0 {
		return out, nil
	}
	err := db.Where("user_member_id IN ?", ids).Order("user_member_id, game_code").Find(&out).Error
	return out, err
}

// LockUserMemberGameSettingsRepository — SELECT ... FOR UPDATE แถว PT / Commission ของ Member
func LockUserMemberGameSettingsRepository(db *gorm.DB, memberID uint) ([]models.UserMemberGameSetting, error) {
	var out []models.UserMemberGameSetting
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_member_id = ?", memberID).Order("game_code").Find(&out).Error
	return out, err
}

// UpdateUserMemberPTRepository — PT · remain · Commission ของ Member ทุกเกมในกลุ่ม (MGMT-21 แก้ 2026-10-09)
func UpdateUserMemberPTRepository(db *gorm.DB, memberID uint, gameCodes []string, pt, remain, commission float64, by string, at time.Time) error {
	return db.Model(&models.UserMemberGameSetting{}).Where("user_member_id = ? AND game_code IN ?", memberID, gameCodes).
		Updates(map[string]any{"pt": pt, "remain": remain, "commission": commission, "updated_by": by, "updated_at": at}).Error
}
