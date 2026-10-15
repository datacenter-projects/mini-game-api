// Package ptcalculation คือ repository ของ module pt_calculation — spec: docs/modules/pt_calculation.md
// อ่านอย่างเดียว ไม่ lock (PTC-30) · ไม่มีตารางของตัวเอง
package ptcalculation

import (
	"errors"

	"app/app/models"
	"app/pkg/apperr"

	"gorm.io/gorm"
)

// GetPTCMemberRepository — Member + สายชั้นบน (cnf) · ไม่พบคืน apperr.ErrNotFound
func GetPTCMemberRepository(db *gorm.DB, id uint) (models.UserMember, error) {
	var m models.UserMember
	err := db.Select("id", "agent_id", "cnf").Where("id = ?", id).Take(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return m, apperr.ErrNotFound
	}
	return m, err
}

// GetPTCMemberSettingRepository — pt + commission ของ Member ในเกมนี้ · ไม่มีแถว = found false
func GetPTCMemberSettingRepository(db *gorm.DB, memberID uint, gameCode string) (models.UserMemberGameSetting, bool, error) {
	var s models.UserMemberGameSetting
	err := db.Select("user_member_id", "game_code", "parent_id", "pt", "commission").
		Where("user_member_id = ? AND game_code = ?", memberID, gameCode).Take(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s, false, nil
	}
	return s, err == nil, err
}

// ChainSettingRow — ค่าหุ้นส่วนของชั้นในสาย 1 เกม + role / agent_type (หา CSM — PTC-04A)
type ChainSettingRow struct {
	AgentID      uint
	ParentID     *uint
	Role         models.AgentRole
	AgentType    *models.AgentType
	PTFromParent float64
	Force        float64
	Remain       float64
	Commission   float64
	Status       bool
}

// ListPTCChainSettingsRepository — settings ของทุกชั้นในสายในเกมนี้ query เดียว (ห้ามวนทีละชั้น — กฎข้อ 23 · PTC-30)
// ใช้ PK (agent_id, game_code) · ชั้นที่ไม่มีแถวจะไม่อยู่ในผล (service ตรวจ — PTC-09)
func ListPTCChainSettingsRepository(db *gorm.DB, agentIDs []uint, gameCode string) ([]ChainSettingRow, error) {
	var rows []ChainSettingRow
	err := db.Table("agent_game_settings s").
		Select("s.agent_id, s.parent_id, a.role, a.agent_type, s.pt_from_parent, s.force, s.remain, s.commission, s.status").
		Joins("JOIN user_agents a ON a.id = s.agent_id").
		Where("s.agent_id IN ? AND s.game_code = ?", agentIDs, gameCode).
		Scan(&rows).Error
	return rows, err
}
