package membermanagement

import (
	"errors"
	"time"

	"app/app/models"
	"app/pkg/apperr"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// repository ของตาราง user_members — docs/modules/agent_management.md หัวข้อ 6

// UserMemberPhoneExistsRepository — เบอร์ซ้ำภายในตาราง (MGMT-08)
func UserMemberPhoneExistsRepository(db *gorm.DB, phone string) (bool, error) {
	var n int64
	err := db.Model(&models.UserMember{}).Where("phone = ?", phone).Limit(1).Count(&n).Error
	return n > 0, err
}

func CreateUserMemberRepository(db *gorm.DB, m *models.UserMember) error {
	err := db.Create(m).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return apperr.ErrConflict.Wrap(err)
	}
	return err
}

// GetUserMemberProfileRepository — ไม่พบคืน apperr.ErrNotFound
func GetUserMemberProfileRepository(db *gorm.DB, id uint) (models.UserMember, error) {
	var m models.UserMember
	err := db.Select("id", "agent_id", "username", "status", "currency").Where("id = ?", id).Take(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return m, apperr.ErrNotFound
	}
	return m, err
}

// GetUserMemberDetailRepository — ไม่พบคืน apperr.ErrNotFound
func GetUserMemberDetailRepository(db *gorm.DB, id uint) (models.UserMember, error) {
	var m models.UserMember
	err := db.Select("id", "agent_id", "username", "name", "phone", "currency", "status", "last_login_at", "last_login_ip", "created_at").
		Where("id = ?", id).Take(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return m, apperr.ErrNotFound
	}
	return m, err
}

// LockUserMemberRowRepository — SELECT ... FOR UPDATE · ไม่พบ = ผลว่าง (id 0)
func LockUserMemberRowRepository(db *gorm.DB, id uint) (models.UserMember, error) {
	var m models.UserMember
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id", "agent_id", "username", "name", "phone", "status").
		Where("id = ?", id).Limit(1).Find(&m).Error
	return m, err
}

// UpdateUserMemberInfoRepository — ชื่อ · เบอร์ (phone nil = ไม่ตั้ง) — MGMT-09A
func UpdateUserMemberInfoRepository(db *gorm.DB, id uint, name string, phone *string, at time.Time) error {
	return db.Model(&models.UserMember{}).Where("id = ?", id).
		Updates(map[string]any{"name": name, "phone": phone, "updated_at": at}).Error
}

// UpdateUserMemberStatusRepository — สถานะที่ตั้งกับบัญชีเอง (MGMT-30)
func UpdateUserMemberStatusRepository(db *gorm.DB, id uint, status models.AgentStatus, at time.Time) error {
	return db.Model(&models.UserMember{}).Where("id = ?", id).Updates(map[string]any{"status": status, "updated_at": at}).Error
}

// UserMemberPhoneTakenByOtherRepository — เบอร์นี้มี Member อื่นใช้แล้วไหม (MGMT-08)
func UserMemberPhoneTakenByOtherRepository(db *gorm.DB, phone string, selfID uint) (bool, error) {
	var n int64
	err := db.Model(&models.UserMember{}).Where("phone = ? AND id <> ?", phone, selfID).Limit(1).Count(&n).Error
	return n > 0, err
}

// FindUserMemberByUsernameRepository — Member (ตารางแยก username ซ้ำกับฝั่ง agent ได้)
func FindUserMemberByUsernameRepository(db *gorm.DB, username string) (models.UserMember, error) {
	var m models.UserMember
	err := db.Select("id", "agent_id", "username", "status", "last_login_at", "last_login_ip", "created_at").
		Where("username = ?", username).Limit(1).Find(&m).Error
	return m, err
}
