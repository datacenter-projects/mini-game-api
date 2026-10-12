package gamerate

import (
	"app/app/models"

	"gorm.io/gorm"
)

// สวิตช์ปิดเกมทั้งระบบ (RE8 · RATE-01B) — อ่าน / lock status_game ของทุกบัญชี Superadmin ในตาราง agent_game_settings
// (ไม่มีตารางใหม่ — แก้ค่าด้วย UpdateStatusGameManyRepository ของ agent_management)

// SuperadminGameStatus — status_game ของเกมหนึ่งของบัญชี Superadmin 1 บัญชี
type SuperadminGameStatus struct {
	AgentID    uint   `gorm:"column:agent_id"`
	Username   string `gorm:"column:username"`
	StatusGame bool   `gorm:"column:status_game"`
}

// LockSuperadminGameStatusRepository — แถวของเกมนั้นของทุกบัญชี Superadmin · FOR UPDATE เรียง agent_id (กฎข้อ 10) · ใช้ใน tx เท่านั้น
func LockSuperadminGameStatusRepository(db *gorm.DB, gameCode string) ([]SuperadminGameStatus, error) {
	var out []SuperadminGameStatus
	err := db.Raw(`SELECT s.agent_id, a.username, s.status_game
		FROM agent_game_settings s JOIN user_agents a ON a.id = s.agent_id
		WHERE a.role = ? AND s.game_code = ? ORDER BY s.agent_id FOR UPDATE OF s`, models.AgentRoleSuperAdmin, gameCode).Scan(&out).Error
	return out, err
}

// ListClosedGamesRepository — รหัสเกมที่ Superadmin อย่างน้อย 1 บัญชีปิดไว้ (RE1 status_game = false)
func ListClosedGamesRepository(db *gorm.DB) ([]string, error) {
	var out []string
	err := db.Raw(`SELECT DISTINCT s.game_code
		FROM agent_game_settings s JOIN user_agents a ON a.id = s.agent_id
		WHERE a.role = ? AND s.status_game = false`, models.AgentRoleSuperAdmin).Scan(&out).Error
	return out, err
}
