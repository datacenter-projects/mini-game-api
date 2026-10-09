package agentmanagement

import (
	"errors"
	"strings"
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

// GetAgentChainRepository — user_agents.cnf (JSON สายชั้นบน — MGMT-61) · ไม่พบคืน apperr.ErrNotFound
func GetAgentChainRepository(db *gorm.DB, id uint) (string, error) {
	var a models.UserAgent
	err := db.Select("id", "cnf").Where("id = ?", id).Take(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", apperr.ErrNotFound
	}
	return a.Cnf, err
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
	err := db.Raw(`SELECT EXISTS (SELECT 1 FROM user_agents WHERE username = ?) OR EXISTS (SELECT 1 FROM user_members WHERE username = ?)`,
		username, username).Scan(&exists).Error
	return exists, err
}

// AgentPhoneExistsRepository — เบอร์ซ้ำภายในตาราง (MGMT-08)
func AgentPhoneExistsRepository(db *gorm.DB, phone string) (bool, error) {
	var n int64
	err := db.Model(&models.UserAgent{}).Where("phone = ?", phone).Limit(1).Count(&n).Error
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

// LockAgentBalancesRepository — SELECT ... FOR UPDATE ยอดของบัญชีตามสกุล เรียงตามสกุลกัน deadlock (กฎข้อ 10)
// สกุลที่ไม่มีแถว = ไม่อยู่ในผลลัพธ์
func LockAgentBalancesRepository(db *gorm.DB, agentID uint, currencies []string) ([]models.AgentBalance, error) {
	var out []models.AgentBalance
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("agent_id = ? AND currency IN ?", agentID, currencies).Order("currency").Find(&out).Error
	return out, err
}

func UpdateAgentBalanceRepository(db *gorm.DB, agentID uint, currency string, amount float64, at time.Time) error {
	return db.Model(&models.AgentBalance{}).Where("agent_id = ? AND currency = ?", agentID, currency).
		Updates(map[string]any{"amount": amount, "updated_at": at}).Error
}

func CreateAgentBalancesRepository(db *gorm.DB, rows []models.AgentBalance) error {
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

// IsInDownlineRepository — id อยู่ในสายล่างของ ancestorID ไหม (ไม่นับตัวเอง · กฎข้อ 22 · MGMT-26)
func IsInDownlineRepository(db *gorm.DB, ancestorID, id uint) (bool, error) {
	var ok bool
	err := db.Raw(`
		WITH RECURSIVE up AS (
			SELECT parent_id FROM user_agents WHERE id = ?
			UNION -- ไม่ใช้ UNION ALL: ถ้าข้อมูลสายวนเป็นลูป query จะหยุดเองไม่ค้าง
			SELECT u.parent_id FROM user_agents u JOIN up ON u.id = up.parent_id
		)
		SELECT EXISTS (SELECT 1 FROM up WHERE parent_id = ?)`, id, ancestorID).Scan(&ok).Error
	return ok, err
}

// DownlineRow — ลูกตรง 1 แถว (ฝั่ง agent หรือ Member) — MGMT-28
type DownlineRow struct {
	ID          uint
	IsMember    bool
	Role        models.AgentRole
	AgentType   *models.AgentType
	Username    string
	Name        *string
	Phone       *string
	Status      models.AgentStatus
	Currency    *string // Member เท่านั้น
	LastLoginAt *time.Time
	LastLoginIP *string
	CreatedAt   time.Time
}

// downlineSQL — ลูกตรงของ parent ทั้งฝั่ง agent (ไม่รวม ADMIN — AUTH-43) และ Member · กรอง username บางส่วน
const downlineSQL = `
	SELECT id, false AS is_member, role, agent_type, username, name, phone, status, NULL AS currency, last_login_at, last_login_ip, created_at
	FROM user_agents WHERE parent_id = @parent AND role <> 'ADMIN' AND (@q = '' OR username LIKE @like ESCAPE '\')
	UNION ALL
	SELECT id, true, 'MEMBER', NULL, username, name, phone, status, currency, last_login_at, last_login_ip, created_at
	FROM user_members WHERE agent_id = @parent AND (@q = '' OR username LIKE @like ESCAPE '\')`

// ListDownlinesRepository — ลูกตรงเรียง username A→Z แบ่งหน้า + จำนวนทั้งหมด (MGMT-26, MGMT-27)
// q = ข้อความค้น (ตัวเล็กแล้ว) · ว่าง = ไม่กรอง
func ListDownlinesRepository(db *gorm.DB, parentID uint, q string, offset, limit int) ([]DownlineRow, int64, error) {
	args := map[string]any{"parent": parentID, "q": q, "like": "%" + likeEscaper.Replace(q) + "%"}
	var total int64
	if err := db.Raw("SELECT count(*) FROM ("+downlineSQL+") d", args).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []DownlineRow
	args["offset"], args["limit"] = offset, limit
	err := db.Raw("SELECT * FROM ("+downlineSQL+") d ORDER BY username LIMIT @limit OFFSET @offset", args).Scan(&rows).Error
	return rows, total, err
}

var likeEscaper = strings.NewReplacer(`\`, `\`, `%`, `\%`, `_`, `\_`)

// ListAgentDirectChildrenRepository — ลูกตรงฝั่ง agent (ไม่รวม ADMIN) เรียง A→Z · q = ค้น username บางส่วน (ตัวเล็กแล้ว · ว่าง = ไม่กรอง) · ไม่เกิน limit แถว (MGMT-35)
func ListAgentDirectChildrenRepository(db *gorm.DB, parentID uint, q string, limit int) ([]models.UserAgent, error) {
	var out []models.UserAgent
	query := db.Select("id", "username", "name", "role", "agent_type", "status").
		Where("parent_id = ? AND role <> ?", parentID, models.AgentRoleAdmin)
	if q != "" {
		query = query.Where(`username LIKE ? ESCAPE '\'`, "%"+likeEscaper.Replace(q)+"%")
	}
	err := query.Order("username").Limit(limit).Find(&out).Error
	return out, err
}

// GetAgentDetailRepository — ข้อมูลบัญชีฝั่ง agent สำหรับหน้ารายละเอียด (MGMT-29) · passcode_hash ใช้แค่บอกว่าตั้งแล้ว
func GetAgentDetailRepository(db *gorm.DB, id uint) (models.UserAgent, error) {
	var a models.UserAgent
	err := db.Select("id", "parent_id", "username", "name", "phone", "role", "agent_type", "status", "passcode_hash",
		"last_login_at", "last_login_ip", "created_at", "cnf").Where("id = ?", id).Take(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return a, apperr.ErrNotFound
	}
	return a, err
}

// ---- อ่านทีละหลายบัญชี (กัน N+1 ในหน้ารายชื่อ) ----

func ListAgentCurrenciesByIDsRepository(db *gorm.DB, ids []uint) ([]models.AgentCurrency, error) {
	var out []models.AgentCurrency
	if len(ids) == 0 {
		return out, nil
	}
	err := db.Where("agent_id IN ?", ids).Order("agent_id, currency").Find(&out).Error
	return out, err
}

func ListAgentBalancesByIDsRepository(db *gorm.DB, ids []uint) ([]models.AgentBalance, error) {
	var out []models.AgentBalance
	if len(ids) == 0 {
		return out, nil
	}
	err := db.Select("agent_id", "currency", "amount").Where("agent_id IN ?", ids).Find(&out).Error
	return out, err
}

func ListAgentGameSettingsByIDsRepository(db *gorm.DB, ids []uint) ([]models.AgentGameSetting, error) {
	var out []models.AgentGameSetting
	if len(ids) == 0 {
		return out, nil
	}
	err := db.Where("agent_id IN ?", ids).Order("agent_id, game_code").Find(&out).Error
	return out, err
}

// DownlineSearchRow — ผลค้นทั้งสาย (MGMT-27A) = แถวรายชื่อ + ผู้สร้างตรง + สถานะที่ใช้งานจริง
type DownlineSearchRow struct {
	DownlineRow
	ParentUsername  string
	EffectiveStatus models.AgentStatus
}

// downlineSearchSQL — ทุกบัญชีใต้ @root ทุกชั้น (ไม่รวม root · ไม่รวม ADMIN) ที่ username ตรง @like
// tree เก็บสถานะที่ใช้งานจริงสะสมจากบนลงล่าง (rank: ACTIVE 0 · SUSPENDED 1 · LOCKED 2) จึงไม่ต้อง query หัวสายทีละแถว
const downlineSearchSQL = `
	WITH RECURSIVE tree AS (
		SELECT id, username, CASE @root_status WHEN 'LOCKED' THEN 2 WHEN 'SUSPENDED' THEN 1 ELSE 0 END AS rnk
		FROM user_agents WHERE id = @root
		UNION -- ไม่ใช้ UNION ALL: ถ้าข้อมูลสายวนเป็นลูป query จะหยุดเองไม่ค้าง
		SELECT u.id, u.username,
			GREATEST(t.rnk, CASE u.status WHEN 'LOCKED' THEN 2 WHEN 'SUSPENDED' THEN 1 ELSE 0 END)
		FROM user_agents u JOIN tree t ON u.parent_id = t.id
		WHERE u.role <> 'ADMIN'
	), hits AS (
		SELECT u.id, false AS is_member, u.role, u.agent_type, u.username, u.name, u.phone, u.status, NULL AS currency,
			u.last_login_at, u.last_login_ip, u.created_at,
			p.username AS parent_username, GREATEST(p.rnk, CASE u.status WHEN 'LOCKED' THEN 2 WHEN 'SUSPENDED' THEN 1 ELSE 0 END) AS rnk
		FROM user_agents u JOIN tree p ON u.parent_id = p.id
		WHERE u.role <> 'ADMIN' AND u.username LIKE @like ESCAPE '\'
		UNION ALL
		SELECT m.id, true, 'MEMBER', NULL, m.username, m.name, m.phone, m.status, m.currency,
			m.last_login_at, m.last_login_ip, m.created_at,
			p.username, GREATEST(p.rnk, CASE m.status WHEN 'LOCKED' THEN 2 WHEN 'SUSPENDED' THEN 1 ELSE 0 END)
		FROM user_members m JOIN tree p ON m.agent_id = p.id
		WHERE m.username LIKE @like ESCAPE '\'
	)`

// SearchDownlinesRepository — ค้นทั้งสายใต้ rootID เรียง username A→Z แบ่งหน้า + จำนวนทั้งหมด (MGMT-27A)
// rootStatus = สถานะที่ใช้งานจริงของ root (รวมหัวสายของ root แล้ว) · q ตัวเล็กแล้ว
func SearchDownlinesRepository(db *gorm.DB, rootID uint, rootStatus models.AgentStatus, q string, offset, limit int) ([]DownlineSearchRow, int64, error) {
	args := map[string]any{"root": rootID, "root_status": string(rootStatus), "like": "%" + likeEscaper.Replace(q) + "%"}
	var total int64
	if err := db.Raw(downlineSearchSQL+" SELECT count(*) FROM hits", args).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []struct {
		DownlineRow
		ParentUsername string
		Rnk            int
	}
	args["offset"], args["limit"] = offset, limit
	err := db.Raw(downlineSearchSQL+" SELECT * FROM hits ORDER BY username LIMIT @limit OFFSET @offset", args).Scan(&rows).Error
	out := make([]DownlineSearchRow, len(rows))
	for i, r := range rows {
		status := models.AgentStatusActive
		switch r.Rnk {
		case 1:
			status = models.AgentStatusSuspended
		case 2:
			status = models.AgentStatusLocked
		}
		out[i] = DownlineSearchRow{DownlineRow: r.DownlineRow, ParentUsername: r.ParentUsername, EffectiveStatus: status}
	}
	return out, total, err
}
