package gamerate

import (
	"app/app/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// สวิตช์ปิดเกมทั้งระบบ (RE8 · RATE-01B) — อ่าน / lock status_game ของทุกบัญชี Superadmin ในตาราง agent_game_settings
// (ไม่มีตารางใหม่ — แก้ค่าด้วย UpdateStatusGameManyRepository · เพิ่มแถวที่ขาดด้วย CreateAgentGameSettingsRepository ของ agent_management)

// SuperadminGameStatus — status_game ของเกมหนึ่งของบัญชี Superadmin 1 บัญชี · HasRow false = ยังไม่มีแถวของเกมนั้น (lead G-S1)
type SuperadminGameStatus struct {
	AgentID    uint
	Username   string
	HasRow     bool
	StatusGame bool
}

// LockSuperadminGameStatusRepository — ทุกบัญชี Superadmin + แถวของเกมนั้น · ใช้ใน tx เท่านั้น
// lock แถว user_agents ของ Superadmin (FOR NO KEY UPDATE · เรียง id — กฎข้อ 10) ให้ RE8 ที่ยิงพร้อมกันรอกัน แม้บัญชีที่ยังไม่มีแถวของเกม
// แล้ว lock แถว agent_game_settings ที่มี (FOR UPDATE)
func LockSuperadminGameStatusRepository(db *gorm.DB, gameCode string) ([]SuperadminGameStatus, error) {
	var agents []models.UserAgent
	if err := db.Select("id", "username").Where("role = ?", models.AgentRoleSuperAdmin).Order("id").
		Clauses(clause.Locking{Strength: "NO KEY UPDATE"}).Find(&agents).Error; err != nil {
		return nil, err
	}
	if len(agents) == 0 {
		return nil, nil
	}
	ids := make([]uint, len(agents))
	for i, a := range agents {
		ids[i] = a.ID
	}
	var rows []models.AgentGameSetting
	if err := db.Select("agent_id", "status_game").Where("agent_id IN ? AND game_code = ?", ids, gameCode).Order("agent_id").
		Clauses(clause.Locking{Strength: "UPDATE"}).Find(&rows).Error; err != nil {
		return nil, err
	}
	status := make(map[uint]bool, len(rows))
	for _, r := range rows {
		status[r.AgentID] = r.StatusGame
	}
	out := make([]SuperadminGameStatus, len(agents))
	for i, a := range agents {
		on, ok := status[a.ID]
		out[i] = SuperadminGameStatus{AgentID: a.ID, Username: a.Username, HasRow: ok, StatusGame: on}
	}
	return out, nil
}

// ListClosedGamesRepository — รหัสเกมที่ Superadmin อย่างน้อย 1 บัญชีปิดไว้ (RE1 status_game = false)
func ListClosedGamesRepository(db *gorm.DB) ([]string, error) {
	var out []string
	err := db.Raw(`SELECT DISTINCT s.game_code
		FROM agent_game_settings s JOIN user_agents a ON a.id = s.agent_id
		WHERE a.role = ? AND s.status_game = false`, models.AgentRoleSuperAdmin).Scan(&out).Error
	return out, err
}
