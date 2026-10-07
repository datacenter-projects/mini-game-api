package postgres

import (
	"errors"
	"time"

	"app/app/models"
	"app/pkg/apperr"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// repository ของ module ② — docs/modules/agent_management.md หัวข้อ 6

// AdvisoryXactLockRepository — pg_advisory_xact_lock ของข้อความ key (ปลดเองตอน tx จบ) · ใช้ใน tx เท่านั้น
func AdvisoryXactLockRepository(db *gorm.DB, key string) error {
	return db.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", key).Error
}

// GetAgentProfileRepository — ข้อมูลประเภทของบัญชีฝั่ง agent · ไม่พบคืน apperr.ErrNotFound
func GetAgentProfileRepository(db *gorm.DB, id uint) (models.UserAgent, error) {
	var a models.UserAgent
	err := db.Select("id", "parent_id", "username", "role", "agent_type", "status").Where("id = ?", id).Take(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return a, apperr.ErrNotFound
	}
	return a, err
}

// GetChainCompanyTypeRepository — agent_type ของ Company หัวสาย (รวมตัวเอง) · ไม่อยู่ใต้ Company (เช่น Superadmin) = nil
func GetChainCompanyTypeRepository(db *gorm.DB, agentID uint) (*models.AgentType, error) {
	var rows []struct{ AgentType *models.AgentType }
	err := db.Raw(`
		WITH RECURSIVE chain AS (
			SELECT id, parent_id, role, agent_type FROM user_agents WHERE id = ?
			UNION -- ไม่ใช้ UNION ALL: ถ้าข้อมูลสายวนเป็นลูป query จะหยุดเองไม่ค้าง
			SELECT u.id, u.parent_id, u.role, u.agent_type FROM user_agents u JOIN chain c ON u.id = c.parent_id
		)
		SELECT agent_type FROM chain WHERE role = ? LIMIT 1`, agentID, models.AgentRoleCompany).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows[0].AgentType, nil
}

// ListAgentCurrenciesRepository — สกุลของบัญชี เรียง A→Z
func ListAgentCurrenciesRepository(db *gorm.DB, agentID uint) ([]string, error) {
	var out []string
	err := db.Model(&models.AgentCurrency{}).Where("agent_id = ?", agentID).Order("currency").Pluck("currency", &out).Error
	return out, err
}

// ListAgentGameSettingsRepository — ค่าหุ้นส่วนทุกเกมของบัญชี
func ListAgentGameSettingsRepository(db *gorm.DB, agentID uint) ([]models.AgentGameSetting, error) {
	var out []models.AgentGameSetting
	err := db.Where("agent_id = ?", agentID).Order("game_code").Find(&out).Error
	return out, err
}

// UsernameExistsRepository — username นี้มีในฝั่ง agent หรือ Member แล้วไหม (MGMT-05)
func UsernameExistsRepository(db *gorm.DB, username string) (bool, error) {
	var exists bool
	err := db.Raw(`SELECT EXISTS (SELECT 1 FROM user_agents WHERE username = ?) OR EXISTS (SELECT 1 FROM members WHERE username = ?)`,
		username, username).Scan(&exists).Error
	return exists, err
}

// AgentPhoneExistsRepository / MemberPhoneExistsRepository — เบอร์ซ้ำภายในตาราง (MGMT-08)
func AgentPhoneExistsRepository(db *gorm.DB, phone string) (bool, error) {
	var n int64
	err := db.Model(&models.UserAgent{}).Where("phone = ?", phone).Limit(1).Count(&n).Error
	return n > 0, err
}

func MemberPhoneExistsRepository(db *gorm.DB, phone string) (bool, error) {
	var n int64
	err := db.Model(&models.Member{}).Where("phone = ?", phone).Limit(1).Count(&n).Error
	return n > 0, err
}

// GetCreateRequestRepository — ไม่พบคืน apperr.ErrNotFound
func GetCreateRequestRepository(db *gorm.DB, requestID string) (models.CreateRequest, error) {
	var r models.CreateRequest
	err := db.Where("request_id = ?", requestID).Take(&r).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r, apperr.ErrNotFound
	}
	return r, err
}

func CreateCreateRequestRepository(db *gorm.DB, r *models.CreateRequest) error {
	return db.Create(r).Error
}

func CreateAgentCurrenciesRepository(db *gorm.DB, rows []models.AgentCurrency) error {
	if len(rows) == 0 {
		return nil
	}
	return db.Create(&rows).Error
}

func CreateAgentGameSettingsRepository(db *gorm.DB, rows []models.AgentGameSetting) error {
	if len(rows) == 0 {
		return nil
	}
	return db.Create(&rows).Error
}

func CreateMemberRepository(db *gorm.DB, m *models.Member) error {
	err := db.Create(m).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return apperr.ErrConflict.Wrap(err)
	}
	return err
}

// GetMemberProfileRepository — ไม่พบคืน apperr.ErrNotFound
func GetMemberProfileRepository(db *gorm.DB, id uint) (models.Member, error) {
	var m models.Member
	err := db.Select("id", "agent_id", "username", "status", "currency").Where("id = ?", id).Take(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return m, apperr.ErrNotFound
	}
	return m, err
}

func CreateMemberGameSettingsRepository(db *gorm.DB, rows []models.MemberGameSetting) error {
	if len(rows) == 0 {
		return nil
	}
	return db.Create(&rows).Error
}

// LockAgentBalancesRepository — SELECT ... FOR UPDATE ยอดของบัญชีตามสกุล เรียงตามสกุลกัน deadlock (กฎข้อ 10)
// สกุลที่ไม่มีแถว = ไม่อยู่ในผลลัพธ์
func LockAgentBalancesRepository(db *gorm.DB, agentID uint, currencies []string) ([]models.AgentBalance, error) {
	var out []models.AgentBalance
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("agent_id = ? AND currency IN ?", agentID, currencies).Order("currency").Find(&out).Error
	return out, err
}

func UpdateAgentBalanceRepository(db *gorm.DB, agentID uint, currency string, amount int64, at time.Time) error {
	return db.Model(&models.AgentBalance{}).Where("agent_id = ? AND currency = ?", agentID, currency).
		Updates(map[string]any{"amount": amount, "updated_at": at}).Error
}

func CreateAgentBalancesRepository(db *gorm.DB, rows []models.AgentBalance) error {
	if len(rows) == 0 {
		return nil
	}
	return db.Create(&rows).Error
}

func CreateMemberBalancesRepository(db *gorm.DB, rows []models.MemberBalance) error {
	if len(rows) == 0 {
		return nil
	}
	return db.Create(&rows).Error
}

func CreateBalanceLedgerRepository(db *gorm.DB, rows []models.BalanceLedger) error {
	if len(rows) == 0 {
		return nil
	}
	return db.Create(&rows).Error
}

func CreateAccountChangeLogsRepository(db *gorm.DB, rows []models.AccountChangeLog) error {
	if len(rows) == 0 {
		return nil
	}
	return db.Create(&rows).Error
}
