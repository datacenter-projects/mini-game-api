package postgres

import (
	"time"

	"app/app/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// repository ของการแก้บัญชี (module ② phase 4) — docs/modules/agent_management.md MGMT-09 – MGMT-31

// LockAgentGameSettingsRepository — lock แถวค่าหุ้นส่วนของหลายบัญชี เรียง agent_id, game_code กัน deadlock (กฎข้อ 10)
// strength: "UPDATE" (จะแก้) หรือ "SHARE" (อ่านค่าที่ใช้ตัดสิน — กันชั้นบนแก้ระหว่างนั้น)
func LockAgentGameSettingsRepository(db *gorm.DB, agentIDs []uint, strength string) ([]models.AgentGameSetting, error) {
	var out []models.AgentGameSetting
	if len(agentIDs) == 0 {
		return out, nil
	}
	err := db.Clauses(clause.Locking{Strength: strength}).Where("agent_id IN ?", agentIDs).
		Order("agent_id, game_code").Find(&out).Error
	return out, err
}

// ListAgentChildIDsRepository — id ของลูกตรงฝั่ง agent เรียงจากน้อยไปมาก
func ListAgentChildIDsRepository(db *gorm.DB, parentID uint) ([]uint, error) {
	var ids []uint
	err := db.Model(&models.UserAgent{}).Where("parent_id = ?", parentID).Order("id").Pluck("id", &ids).Error
	return ids, err
}

// UpdateChildPTRepository — ค่าที่ผู้สร้างตั้งให้ลูก ของทุกเกมในกลุ่ม (MGMT-16, MGMT-23)
func UpdateChildPTRepository(db *gorm.DB, agentID uint, gameCodes []string, v models.AgentGameSetting, by string, at time.Time) error {
	return db.Model(&models.AgentGameSetting{}).Where("agent_id = ? AND game_code IN ?", agentID, gameCodes).Updates(map[string]any{
		"pt_from_parent_bp": v.PTFromParentBP, "force_bp": v.ForceBP, "remain_bp": v.RemainBP,
		"commission_bp": v.CommissionBP, "status": v.Status, "updated_by": by, "updated_at": at,
	}).Error
}

// UpdateOwnPTRepository — ค่าถือของบัญชีเอง ของทุกเกมในกลุ่ม (MGMT-22)
func UpdateOwnPTRepository(db *gorm.DB, agentID uint, gameCodes []string, ptBP int, by string, at time.Time) error {
	return db.Model(&models.AgentGameSetting{}).Where("agent_id = ? AND game_code IN ?", agentID, gameCodes).
		Updates(map[string]any{"pt_bp": ptBP, "updated_by": by, "updated_at": at}).Error
}

// LockMemberGameSettingsRepository — SELECT ... FOR UPDATE แถว Commission ของ Member
func LockMemberGameSettingsRepository(db *gorm.DB, memberID uint) ([]models.MemberGameSetting, error) {
	var out []models.MemberGameSetting
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("member_id = ?", memberID).Order("game_code").Find(&out).Error
	return out, err
}

// UpdateMemberCommissionRepository — Commission ของ Member ทุกเกมในกลุ่ม (MGMT-21)
func UpdateMemberCommissionRepository(db *gorm.DB, memberID uint, gameCodes []string, bp int, by string, at time.Time) error {
	return db.Model(&models.MemberGameSetting{}).Where("member_id = ? AND game_code IN ?", memberID, gameCodes).
		Updates(map[string]any{"commission_bp": bp, "updated_by": by, "updated_at": at}).Error
}

// LockUserAgentRowRepository — SELECT ... FOR UPDATE ข้อมูลที่แก้ได้ของบัญชีฝั่ง agent · ไม่พบ = ผลว่าง (id 0)
func LockUserAgentRowRepository(db *gorm.DB, id uint) (models.UserAgent, error) {
	var a models.UserAgent
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id", "parent_id", "username", "name", "phone", "role", "agent_type", "status").
		Where("id = ?", id).Limit(1).Find(&a).Error
	return a, err
}

// LockMemberRowRepository — SELECT ... FOR UPDATE · ไม่พบ = ผลว่าง (id 0)
func LockMemberRowRepository(db *gorm.DB, id uint) (models.Member, error) {
	var m models.Member
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id", "agent_id", "username", "name", "phone", "status").
		Where("id = ?", id).Limit(1).Find(&m).Error
	return m, err
}

// UpdateUserAgentInfoRepository — ชื่อ · เบอร์ (phone nil = ไม่ตั้ง) — MGMT-09
func UpdateUserAgentInfoRepository(db *gorm.DB, id uint, name string, phone *string, at time.Time) error {
	return db.Model(&models.UserAgent{}).Where("id = ?", id).
		Updates(map[string]any{"name": name, "phone": phone, "updated_at": at}).Error
}

func UpdateMemberInfoRepository(db *gorm.DB, id uint, name string, phone *string, at time.Time) error {
	return db.Model(&models.Member{}).Where("id = ?", id).
		Updates(map[string]any{"name": name, "phone": phone, "updated_at": at}).Error
}

// UpdateUserAgentStatusRepository / UpdateMemberStatusRepository — สถานะที่ตั้งกับบัญชีเอง (MGMT-30)
func UpdateUserAgentStatusRepository(db *gorm.DB, id uint, status models.AgentStatus, at time.Time) error {
	return db.Model(&models.UserAgent{}).Where("id = ?", id).Updates(map[string]any{"status": status, "updated_at": at}).Error
}

func UpdateMemberStatusRepository(db *gorm.DB, id uint, status models.AgentStatus, at time.Time) error {
	return db.Model(&models.Member{}).Where("id = ?", id).Updates(map[string]any{"status": status, "updated_at": at}).Error
}

// AgentPhoneTakenByOtherRepository / MemberPhoneTakenByOtherRepository — เบอร์นี้มีบัญชีอื่นในตารางใช้แล้วไหม (MGMT-08)
func AgentPhoneTakenByOtherRepository(db *gorm.DB, phone string, selfID uint) (bool, error) {
	var n int64
	err := db.Model(&models.UserAgent{}).Where("phone = ? AND id <> ?", phone, selfID).Limit(1).Count(&n).Error
	return n > 0, err
}

func MemberPhoneTakenByOtherRepository(db *gorm.DB, phone string, selfID uint) (bool, error) {
	var n int64
	err := db.Model(&models.Member{}).Where("phone = ? AND id <> ?", phone, selfID).Limit(1).Count(&n).Error
	return n > 0, err
}

// UpdateStatusGameRepository — เปิด / ปิดเกมรายบัญชี (MGMT-20) · ไม่แตะ updated_* ของค่า PT
func UpdateStatusGameRepository(db *gorm.DB, agentID uint, gameCodes []string, on bool) error {
	if len(gameCodes) == 0 {
		return nil
	}
	return db.Model(&models.AgentGameSetting{}).Where("agent_id = ? AND game_code IN ?", agentID, gameCodes).
		Update("status_game", on).Error
}
