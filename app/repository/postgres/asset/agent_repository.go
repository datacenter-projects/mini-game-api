package asset

import (
	"gorm.io/gorm"
)

// UsernameRow — id → username ของบัญชีฝั่ง agent
type UsernameRow struct {
	ID       uint
	Username string
}

// ListAgentUsernamesRepository — username ของชั้นบนที่ให้ค่ามา (`inherited_from` — AST-53) · query เดียวต่อหลาย id
func ListAgentUsernamesRepository(db *gorm.DB, ids []uint) (map[uint]string, error) {
	out := make(map[uint]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var rows []UsernameRow
	if err := db.Table("user_agents").Select("id", "username").Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ID] = r.Username
	}
	return out, nil
}
