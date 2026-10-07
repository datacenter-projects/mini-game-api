package postgres

import (
	"errors"
	"time"

	"app/app/models"
	"app/pkg/apperr"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// คอลัมน์ที่ auth ใช้ทุก request — docs/modules/agent_auth_phase2.md
var subaccountAuthColumns = []string{"id", "agent_id", "username", "status",
	"passcode_hash", "must_change_password", "must_change_passcode"}

var subaccountCredentialColumns = append(append([]string{}, subaccountAuthColumns...),
	"password_hash", "previous_password_hash", "temp_password_expires_at", "temp_passcode_expires_at")

// GetSubaccountForLoginRepository — ไม่พบคืน apperr.ErrNotFound
func GetSubaccountForLoginRepository(db *gorm.DB, username string) (models.Subaccount, error) {
	var s models.Subaccount
	err := db.Select(subaccountCredentialColumns).Where("username = ?", username).Take(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s, apperr.ErrNotFound
	}
	return s, err
}

// GetSubaccountAuthByIDRepository — ใช้ใน middleware ทุก request · ไม่พบคืน apperr.ErrNotFound
func GetSubaccountAuthByIDRepository(db *gorm.DB, id uint) (models.Subaccount, error) {
	var s models.Subaccount
	err := db.Select(subaccountAuthColumns).Where("id = ?", id).Take(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s, apperr.ErrNotFound
	}
	return s, err
}

// GetSubaccountCredentialsByIDRepository — อ่านรหัสผ่าน/passcode โดยไม่ lock · ไม่พบคืน apperr.ErrNotFound
func GetSubaccountCredentialsByIDRepository(db *gorm.DB, id uint) (models.Subaccount, error) {
	var s models.Subaccount
	err := db.Select(subaccountCredentialColumns).Where("id = ?", id).Take(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s, apperr.ErrNotFound
	}
	return s, err
}

// LockSubaccountCredentialsRepository — SELECT ... FOR UPDATE ก่อนเปลี่ยน/รีเซ็ต (AUTH-42)
func LockSubaccountCredentialsRepository(db *gorm.DB, id uint) (models.Subaccount, error) {
	return GetSubaccountCredentialsByIDRepository(db.Clauses(clause.Locking{Strength: "UPDATE"}), id)
}

func UpdateSubaccountLastLoginRepository(db *gorm.DB, id uint, ip string, at time.Time) error {
	return db.Model(&models.Subaccount{}).Where("id = ?", id).
		Updates(map[string]any{"last_login_at": at, "last_login_ip": ip, "updated_at": at}).Error
}

// SetSubaccountPasscodeIfEmptyRepository ตั้ง passcode เฉพาะตอนยังไม่มี — คืน false ถ้าไม่มีแถวถูกอัปเดต (AUTH-32)
func SetSubaccountPasscodeIfEmptyRepository(db *gorm.DB, id uint, hash string, at time.Time) (bool, error) {
	res := db.Model(&models.Subaccount{}).Where("id = ? AND passcode_hash IS NULL", id).
		Updates(map[string]any{"passcode_hash": hash, "updated_at": at})
	return res.RowsAffected > 0, res.Error
}

func UpdateSubaccountPasscodeRepository(db *gorm.DB, id uint, hash string, mustChange bool, tempExpiresAt *time.Time, at time.Time) error {
	return db.Model(&models.Subaccount{}).Where("id = ?", id).Updates(map[string]any{
		"passcode_hash":            hash,
		"must_change_passcode":     mustChange,
		"temp_passcode_expires_at": tempExpiresAt,
		"updated_at":               at,
	}).Error
}

func UpdateSubaccountPasswordRepository(db *gorm.DB, id uint, hash, previousHash string, mustChange bool, tempExpiresAt *time.Time, at time.Time) error {
	return db.Model(&models.Subaccount{}).Where("id = ?", id).Updates(map[string]any{
		"password_hash":            hash,
		"previous_password_hash":   previousHash,
		"must_change_password":     mustChange,
		"temp_password_expires_at": tempExpiresAt,
		"updated_at":               at,
	}).Error
}

// CreateSubaccountRepository — รอบนี้ใช้ใน integration test เท่านั้น (การสร้าง sub อยู่ใน module subaccount)
func CreateSubaccountRepository(db *gorm.DB, s *models.Subaccount) error {
	err := db.Create(s).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return apperr.ErrConflict.Wrap(err)
	}
	return err
}

// GetSubaccountProfileRepository — คอลัมน์ของหน้า Profile (account ACC-12) · ไม่พบคืน apperr.ErrNotFound
func GetSubaccountProfileRepository(db *gorm.DB, id uint) (models.Subaccount, error) {
	var s models.Subaccount
	err := db.Select("id", "agent_id", "username", "last_login_at", "last_login_ip", "created_at").Where("id = ?", id).Take(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s, apperr.ErrNotFound
	}
	return s, err
}
