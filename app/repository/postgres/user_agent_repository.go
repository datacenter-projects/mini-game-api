package postgres

import (
	"errors"
	"time"

	"app/app/models"
	"app/pkg/apperr"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// คอลัมน์ที่ auth ใช้ทุก request — ไม่ดึง password_hash ในจุดที่ไม่ต้องใช้
// passcode_hash ดึงมาเพื่อรู้ว่าตั้ง passcode แล้วหรือยัง (AUTH-29)
var userAgentAuthColumns = []string{"id", "parent_id", "username", "role", "status",
	"passcode_hash", "must_change_password", "must_change_passcode"}

// คอลัมน์รหัสผ่าน/passcode ทั้งหมด — ใช้ตอน login และตอนเปลี่ยน/รีเซ็ต
var userAgentCredentialColumns = append(append([]string{}, userAgentAuthColumns...),
	"password_hash", "previous_password_hash", "temp_password_expires_at", "temp_passcode_expires_at")

// GetUserAgentForLoginRepository — ไม่พบคืน apperr.ErrNotFound
func GetUserAgentForLoginRepository(db *gorm.DB, username string) (models.UserAgent, error) {
	var a models.UserAgent
	err := db.Select(userAgentCredentialColumns).Where("username = ?", username).Take(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return a, apperr.ErrNotFound
	}
	return a, err
}

// GetUserAgentAuthByIDRepository — ใช้ใน middleware ทุก request (primary key lookup) · ไม่พบคืน apperr.ErrNotFound
func GetUserAgentAuthByIDRepository(db *gorm.DB, id uint) (models.UserAgent, error) {
	var a models.UserAgent
	err := db.Select(userAgentAuthColumns).Where("id = ?", id).Take(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return a, apperr.ErrNotFound
	}
	return a, err
}

// GetUserAgentCredentialsByIDRepository — อ่านรหัสผ่าน/passcode โดยไม่ lock · ไม่พบคืน apperr.ErrNotFound
func GetUserAgentCredentialsByIDRepository(db *gorm.DB, id uint) (models.UserAgent, error) {
	var a models.UserAgent
	err := db.Select(userAgentCredentialColumns).Where("id = ?", id).Take(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return a, apperr.ErrNotFound
	}
	return a, err
}

// LockUserAgentCredentialsRepository — SELECT ... FOR UPDATE ก่อนเปลี่ยน/รีเซ็ต (AUTH-42) · ไม่พบคืน apperr.ErrNotFound
func LockUserAgentCredentialsRepository(db *gorm.DB, id uint) (models.UserAgent, error) {
	return GetUserAgentCredentialsByIDRepository(db.Clauses(clause.Locking{Strength: "UPDATE"}), id)
}

// ListAncestorStatusesRepository — สถานะ (ไม่ซ้ำ) ของ upline ทุกคน ไม่นับตัวเอง
// ไล่สายด้วย recursive CTE ใน query เดียว (ไม่ N+1) · ใช้ทุก request ใน middleware (AUTH-27, AUTH-53)
func ListAncestorStatusesRepository(db *gorm.DB, agentID uint) ([]models.AgentStatus, error) {
	var statuses []models.AgentStatus
	err := db.Raw(`
		WITH RECURSIVE ancestors AS (
			SELECT parent_id FROM user_agents WHERE id = ?
			UNION -- ไม่ใช้ UNION ALL: ถ้าข้อมูลสายวนเป็นลูป query จะหยุดเองไม่ค้าง
			SELECT ua.parent_id FROM user_agents ua JOIN ancestors a ON ua.id = a.parent_id
		)
		SELECT DISTINCT ua.status FROM user_agents ua JOIN ancestors a ON ua.id = a.parent_id`,
		agentID).Scan(&statuses).Error
	return statuses, err
}

func UpdateUserAgentLastLoginRepository(db *gorm.DB, id uint, ip string, at time.Time) error {
	return db.Model(&models.UserAgent{}).Where("id = ?", id).
		Updates(map[string]any{"last_login_at": at, "last_login_ip": ip, "updated_at": at}).Error
}

// SetUserAgentPasscodeIfEmptyRepository ตั้ง passcode เฉพาะตอนยังไม่มี — คืน false ถ้าไม่มีแถวถูกอัปเดต (AUTH-32)
func SetUserAgentPasscodeIfEmptyRepository(db *gorm.DB, id uint, hash string, at time.Time) (bool, error) {
	res := db.Model(&models.UserAgent{}).Where("id = ? AND passcode_hash IS NULL", id).
		Updates(map[string]any{"passcode_hash": hash, "updated_at": at})
	return res.RowsAffected > 0, res.Error
}

// UpdateUserAgentPasscodeRepository เขียน passcode ใหม่ · tempExpiresAt = nil คือไม่ใช่ค่าชั่วคราว
func UpdateUserAgentPasscodeRepository(db *gorm.DB, id uint, hash string, mustChange bool, tempExpiresAt *time.Time, at time.Time) error {
	return db.Model(&models.UserAgent{}).Where("id = ?", id).Updates(map[string]any{
		"passcode_hash":            hash,
		"must_change_passcode":     mustChange,
		"temp_passcode_expires_at": tempExpiresAt,
		"updated_at":               at,
	}).Error
}

// UpdateUserAgentPasswordRepository เขียนรหัสผ่านใหม่และเก็บรหัสเดิมไว้ 1 รหัส (AUTH-38)
func UpdateUserAgentPasswordRepository(db *gorm.DB, id uint, hash, previousHash string, mustChange bool, tempExpiresAt *time.Time, at time.Time) error {
	return db.Model(&models.UserAgent{}).Where("id = ?", id).Updates(map[string]any{
		"password_hash":            hash,
		"previous_password_hash":   previousHash,
		"must_change_password":     mustChange,
		"temp_password_expires_at": tempExpiresAt,
		"updated_at":               at,
	}).Error
}

func CreateUserAgentRepository(db *gorm.DB, a *models.UserAgent) error {
	err := db.Create(a).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return apperr.ErrConflict.Wrap(err)
	}
	return err
}

// GetUserAgentProfileRepository — คอลัมน์ของหน้า Profile (account ACC-12) · ไม่พบคืน apperr.ErrNotFound
func GetUserAgentProfileRepository(db *gorm.DB, id uint) (models.UserAgent, error) {
	var a models.UserAgent
	err := db.Select("id", "username", "role", "agent_type", "last_login_at", "last_login_ip", "created_at").Where("id = ?", id).Take(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return a, apperr.ErrNotFound
	}
	return a, err
}
