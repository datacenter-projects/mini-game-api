package agentmanagement

import (
	"errors"
	"fmt"
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

// chainContains — ค่าเทียบ cnf @> ของ "สายชั้นบนมี id นี้" (MGMT-61 · ใช้ GIN index jsonb_path_ops)
func chainContains(id uint) string {
	return fmt.Sprintf(`{"parent":[{"id":%d}]}`, id)
}

// IsInDownlineRepository — id (บัญชีฝั่ง agent) อยู่ในสายล่างของ ancestorID ไหม (ไม่นับตัวเอง — cnf ไม่รวมตัวเอง · กฎข้อ 22 · MGMT-26)
func IsInDownlineRepository(db *gorm.DB, ancestorID, id uint) (bool, error) {
	var ok bool
	err := db.Raw(`SELECT EXISTS (SELECT 1 FROM user_agents WHERE id = ? AND cnf @> ?::jsonb)`, id, chainContains(ancestorID)).Scan(&ok).Error
	return ok, err
}

// DownlineRow — ลูกตรง 1 แถว (ฝั่ง agent หรือ Member) — MGMT-28
type DownlineRow struct {
	ID               uint
	IsMember         bool
	Role             models.AgentRole
	AgentType        *models.AgentType
	Username         string
	Name             *string
	PhoneCountryCode *string
	Phone            *string
	Status           models.AgentStatus
	Currency         *string // Member เท่านั้น
	LastLoginAt      *time.Time
	LastLoginIP      *string
	CreatedAt        time.Time
}

// downlineSQL — ลูกตรงของ parent ทั้งฝั่ง agent (ไม่รวม ADMIN — AUTH-43) และ Member · กรอง username บางส่วน
const downlineSQL = `
	SELECT id, false AS is_member, role, agent_type, username, name, phone_country_code, phone, status, NULL AS currency, last_login_at, last_login_ip, created_at
	FROM user_agents WHERE parent_id = @parent AND role <> 'ADMIN' AND (@q = '' OR username LIKE @like ESCAPE '\')
	UNION ALL
	SELECT id, true, 'MEMBER', NULL, username, name, phone_country_code, phone, status, currency, last_login_at, last_login_ip, created_at
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

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

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
	err := db.Select("id", "parent_id", "username", "name", "phone_country_code", "phone", "role", "agent_type", "status", "passcode_hash",
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

// downlineSearchSQL — ทุกบัญชีใต้ @root ทุกชั้น (cnf @> + GIN index — MGMT-61 · ไม่รวม root · ไม่รวม ADMIN) ที่ username ตรง @like
// creator_id = ผู้สร้างตรง · cnf ใช้คิดสถานะที่ใช้งานจริงเฉพาะแถวในหน้า (downlineSearchPageSQL)
const downlineSearchSQL = `
	WITH hits AS (
		SELECT id, false AS is_member, role, agent_type, username, name, phone_country_code, phone, status, NULL AS currency,
			last_login_at, last_login_ip, created_at, parent_id AS creator_id, cnf
		FROM user_agents WHERE cnf @> CAST(@chain AS jsonb) AND role <> 'ADMIN' AND username LIKE @like ESCAPE '\'
		UNION ALL
		SELECT id, true, 'MEMBER', NULL, username, name, phone_country_code, phone, status, currency,
			last_login_at, last_login_ip, created_at, agent_id, cnf
		FROM user_members WHERE cnf @> CAST(@chain AS jsonb) AND username LIKE @like ESCAPE '\'
	)`

// downlineSearchPageSQL — 1 หน้าของผลค้น + ผู้สร้างตรง + rank สถานะที่ใช้งานจริง (ACTIVE 0 · SUSPENDED 1 · LOCKED 2)
// = แย่สุดของ root (@root_status รวมหัวสายของ root แล้ว) · ตัวเอง · ทุกบัญชีใน cnf ที่อยู่ใต้ root
// cnf ของทุกแถวใต้ root ขึ้นต้นด้วย cnf ของ root + root เสมอ → บัญชีใต้ root = ลำดับที่ (ORDINALITY) เกินความยาว cnf ของ root + 1
const downlineSearchPageSQL = `
	, root AS (
		SELECT jsonb_array_length(cnf -> 'parent') + 1 AS pos FROM user_agents WHERE id = @root
	)
	SELECT h.id, h.is_member, h.role, h.agent_type, h.username, h.name, h.phone_country_code, h.phone, h.status, h.currency,
		h.last_login_at, h.last_login_ip, h.created_at, p.username AS parent_username,
		GREATEST(CASE @root_status WHEN 'LOCKED' THEN 2 WHEN 'SUSPENDED' THEN 1 ELSE 0 END,
			CASE h.status WHEN 'LOCKED' THEN 2 WHEN 'SUSPENDED' THEN 1 ELSE 0 END, COALESCE(up.rnk, 0)) AS rnk
	FROM (SELECT * FROM hits ORDER BY username LIMIT @limit OFFSET @offset) h
	CROSS JOIN root
	JOIN user_agents p ON p.id = h.creator_id
	CROSS JOIN LATERAL (
		SELECT max(CASE a.status WHEN 'LOCKED' THEN 2 WHEN 'SUSPENDED' THEN 1 ELSE 0 END) AS rnk
		FROM jsonb_array_elements(h.cnf -> 'parent') WITH ORDINALITY AS e(node, ord)
		JOIN user_agents a ON a.id = (e.node ->> 'id')::bigint
		WHERE e.ord > root.pos
	) up
	ORDER BY h.username`

// SearchDownlinesRepository — ค้นทั้งสายใต้ rootID เรียง username A→Z แบ่งหน้า + จำนวนทั้งหมด (MGMT-27A)
// rootStatus = สถานะที่ใช้งานจริงของ root (รวมหัวสายของ root แล้ว) · q ตัวเล็กแล้ว
func SearchDownlinesRepository(db *gorm.DB, rootID uint, rootStatus models.AgentStatus, q string, offset, limit int) ([]DownlineSearchRow, int64, error) {
	args := map[string]any{"root": rootID, "root_status": string(rootStatus), "chain": chainContains(rootID),
		"like": "%" + likeEscaper.Replace(q) + "%"}
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
	err := db.Raw(downlineSearchSQL+downlineSearchPageSQL, args).Scan(&rows).Error
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

// ListUsernamesByIDsRepository — id → username ของบัญชีฝั่ง agent (ใช้เขียน log หลายแถวในคำสั่งเดียว)
func ListUsernamesByIDsRepository(db *gorm.DB, ids []uint) (map[uint]string, error) {
	out := make(map[uint]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var rows []models.UserAgent
	if err := db.Select("id", "username").Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ID] = r.Username
	}
	return out, nil
}
