package membermanagement

import (
	"app/app/models"

	"gorm.io/gorm"
)

// repository ของตาราง user_member_balances — docs/modules/agent_management.md หัวข้อ 6

func CreateUserMemberBalancesRepository(db *gorm.DB, rows []models.UserMemberBalance) error {
	if len(rows) == 0 {
		return nil
	}
	return db.Create(&rows).Error
}

func ListUserMemberBalancesByIDsRepository(db *gorm.DB, ids []uint) ([]models.UserMemberBalance, error) {
	var out []models.UserMemberBalance
	if len(ids) == 0 {
		return out, nil
	}
	err := db.Select("user_member_id", "currency", "amount").Where("user_member_id IN ?", ids).Find(&out).Error
	return out, err
}
