// Package gamerate คือ repository ของตาราง game_rate_* — docs/modules/game_rate.md หัวข้อ 6
// ไม่มี tx (service เปิด) · อ่านหลายเวอร์ชันแบบ batch (กฎข้อ 23) · ไม่พบ = ผลว่าง (id 0) / slice ว่าง
package gamerate

import (
	"errors"

	"app/app/models"
	"app/pkg/apperr"

	"gorm.io/gorm"
)

var versionColumns = []string{"id", "game_code", "version", "win_percent", "actor_type", "actor_id", "created_by", "ip", "request_id", "rollback_from", "created_at"}

// LockGameRateRepository — advisory lock ต่อเกม (ปลดเองตอน tx จบ) · กันบันทึก / ย้อนเกมเดียวกันพร้อมกัน (RATE-05B) · ใช้ใน tx เท่านั้น
func LockGameRateRepository(db *gorm.DB, gameCode string) error {
	return db.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "game_rate:"+gameCode).Error
}

// GetLatestVersionRepository — เวอร์ชันปัจจุบันของเกม (เลขมากสุด · ใช้ uq_game_rate_versions) · ยังไม่เคยตั้ง = id 0
func GetLatestVersionRepository(db *gorm.DB, gameCode string) (models.GameRateVersion, error) {
	var v models.GameRateVersion
	err := db.Select(versionColumns).Where("game_code = ?", gameCode).Order("version DESC").Limit(1).Find(&v).Error
	return v, err
}

// ListLatestVersionsRepository — เวอร์ชันปัจจุบันของทุกเกมที่ระบุใน query เดียว (RE1) · เกมที่ยังไม่เคยตั้งไม่มีในผล
func ListLatestVersionsRepository(db *gorm.DB, gameCodes []string) ([]models.GameRateVersion, error) {
	var out []models.GameRateVersion
	if len(gameCodes) == 0 {
		return out, nil
	}
	err := db.Raw(`SELECT DISTINCT ON (game_code) id, game_code, version, win_percent, actor_type, actor_id, created_by, ip, request_id, rollback_from, created_at
		FROM game_rate_versions WHERE game_code IN ? ORDER BY game_code, version DESC`, gameCodes).Scan(&out).Error
	return out, err
}

// GetVersionRepository — เวอร์ชันที่ระบุของเกม · ไม่พบ = id 0
func GetVersionRepository(db *gorm.DB, gameCode string, version int) (models.GameRateVersion, error) {
	var v models.GameRateVersion
	err := db.Select(versionColumns).Where("game_code = ? AND version = ?", gameCode, version).Limit(1).Find(&v).Error
	return v, err
}

// GetVersionByIDRepository — เวอร์ชันตาม id (เดิมพันอ้าง rate_version_id — RH1) · ไม่พบ = id 0
func GetVersionByIDRepository(db *gorm.DB, id uint) (models.GameRateVersion, error) {
	var v models.GameRateVersion
	err := db.Select(versionColumns).Where("id = ?", id).Limit(1).Find(&v).Error
	return v, err
}

// ListVersionsRepository — เวอร์ชันของเกม ใหม่ → เก่า แบ่งหน้า + จำนวนทั้งหมด (RE5)
func ListVersionsRepository(db *gorm.DB, gameCode string, offset, limit int) ([]models.GameRateVersion, int64, error) {
	var total int64
	if err := db.Model(&models.GameRateVersion{}).Where("game_code = ?", gameCode).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []models.GameRateVersion
	err := db.Select(versionColumns).Where("game_code = ?", gameCode).Order("version DESC").Offset(offset).Limit(limit).Find(&out).Error
	return out, total, err
}

// ListStepsByVersionIDsRepository — ขั้นของหลายเวอร์ชันใน query เดียว เรียง version_id, step
func ListStepsByVersionIDsRepository(db *gorm.DB, versionIDs []uint) ([]models.GameRateStep, error) {
	var out []models.GameRateStep
	if len(versionIDs) == 0 {
		return out, nil
	}
	err := db.Select("version_id", "step", "multiplier", "win_percent", "status").Where("version_id IN ?", versionIDs).
		Order("version_id, step").Find(&out).Error
	return out, err
}

// ListPrizesByVersionIDsRepository — รางวัลของหลายเวอร์ชันใน query เดียว เรียง version_id, row_no
func ListPrizesByVersionIDsRepository(db *gorm.DB, versionIDs []uint) ([]models.GameRatePrize, error) {
	var out []models.GameRatePrize
	if len(versionIDs) == 0 {
		return out, nil
	}
	err := db.Select("version_id", "row_no", "multiplier", "share_percent", "status").Where("version_id IN ?", versionIDs).
		Order("version_id, row_no").Find(&out).Error
	return out, err
}

// CreateVersionRepository — แถวเวอร์ชันใหม่ · เลขซ้ำ (uq_game_rate_versions — หลุด lock) คืน apperr.ErrConflict
func CreateVersionRepository(db *gorm.DB, v *models.GameRateVersion) error {
	err := db.Create(v).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return apperr.ErrConflict.Wrap(err)
	}
	return err
}

// CreateStepsRepository — ขั้นของเวอร์ชันแบบ batch
func CreateStepsRepository(db *gorm.DB, rows []models.GameRateStep) error {
	if len(rows) == 0 {
		return nil
	}
	return db.Create(&rows).Error
}

// CreatePrizesRepository — รางวัลของเวอร์ชันแบบ batch
func CreatePrizesRepository(db *gorm.DB, rows []models.GameRatePrize) error {
	if len(rows) == 0 {
		return nil
	}
	return db.Create(&rows).Error
}
