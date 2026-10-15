// Package playerauth คือ query ของ Player Auth — ทุกฟังก์ชันรับ db เป็น param แรก ไม่เปิด transaction (กฎข้อ 5)
package playerauth

import (
	"errors"
	"time"

	"app/app/models"
	"app/pkg/apperr"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func one(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperr.ErrNotFound
	}
	return err
}

// GetMemberForLoginRepository — Member ตาม username (ตัวเล็กแล้ว) สำหรับ login · ไม่พบ = apperr.ErrNotFound
func GetMemberForLoginRepository(db *gorm.DB, username string) (models.UserMember, error) {
	var m models.UserMember
	err := db.Select("id", "agent_id", "username", "password_hash", "name", "currency", "status", "cnf", "win_limit_per_day").
		Where("username = ?", username).Take(&m).Error
	return m, one(err)
}

// GetMemberAuthByIDRepository — Member ตาม id สำหรับ middleware (PA-30) · ไม่พบ = apperr.ErrNotFound
func GetMemberAuthByIDRepository(db *gorm.DB, id uint) (models.UserMember, error) {
	var m models.UserMember
	err := db.Select("id", "agent_id", "username", "name", "currency", "status", "cnf", "win_limit_per_day").
		Where("id = ?", id).Take(&m).Error
	return m, one(err)
}

// LockMemberCredentialsRepository — SELECT ... FOR UPDATE ของแถว Member (เปลี่ยนรหัส PA-50) · ไม่พบ = apperr.ErrNotFound
func LockMemberCredentialsRepository(db *gorm.DB, id uint) (models.UserMember, error) {
	var m models.UserMember
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id", "username", "password_hash").
		Where("id = ?", id).Take(&m).Error
	return m, one(err)
}

// GetMemberBalanceRepository — สกุล + ยอดของ Member (PA6 · ไม่ cache — กฎข้อ 14) · ไม่พบ = apperr.ErrNotFound
func GetMemberBalanceRepository(db *gorm.DB, id uint) (models.UserMember, error) {
	var m models.UserMember
	err := db.Select("id", "currency", "credit").Where("id = ?", id).Take(&m).Error
	return m, one(err)
}

// UpdateMemberLastLoginRepository — บันทึก last_login_at / last_login_ip (PA-25)
func UpdateMemberLastLoginRepository(db *gorm.DB, id uint, ip string, at time.Time) error {
	return db.Model(&models.UserMember{}).Where("id = ?", id).
		Updates(map[string]any{"last_login_at": at, "last_login_ip": ip}).Error
}

// UpdateMemberPasswordRepository — ตั้ง hash รหัสผ่านใหม่ (PA-50)
func UpdateMemberPasswordRepository(db *gorm.DB, id uint, hash string, at time.Time) error {
	return db.Model(&models.UserMember{}).Where("id = ?", id).
		Updates(map[string]any{"password_hash": hash, "updated_at": at}).Error
}

// ListAgentStatusesByIDsRepository — สถานะของ agent ตาม id (สายบนของ Member จาก cnf.parent — PA-30) query เดียว
func ListAgentStatusesByIDsRepository(db *gorm.DB, ids []uint) ([]models.AgentStatus, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var out []models.AgentStatus
	err := db.Model(&models.UserAgent{}).Where("id IN ?", ids).Pluck("status", &out).Error
	return out, err
}

// GetOperatorAgentRepository — agent เจ้าของ Key ตาม username (PA-10) · ไม่พบ = apperr.ErrNotFound
func GetOperatorAgentRepository(db *gorm.DB, username string) (models.UserAgent, error) {
	var a models.UserAgent
	err := db.Select("id", "username", "role", "agent_type", "status").Where("username = ?", username).Take(&a).Error
	return a, one(err)
}

// GetMemberByUsernameRepository — Member ตาม username แบบเบา (operator login / register) · ไม่พบ = apperr.ErrNotFound
func GetMemberByUsernameRepository(db *gorm.DB, username string) (models.UserMember, error) {
	var m models.UserMember
	err := db.Select("id", "agent_id", "username", "name", "currency", "status", "cnf", "win_limit_per_day").
		Where("username = ?", username).Take(&m).Error
	return m, one(err)
}
