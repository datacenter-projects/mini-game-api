// Package depositwithdraw คือ repository ของ module deposit_withdraw — spec: docs/modules/deposit_withdraw.md
// ตาราง deposit_withdraw_logs เป็นของ module นี้ · แตะ agent_balances / user_members เฉพาะยอดเงิน (ลำดับ lock — ARCHITECTURE.md)
package depositwithdraw

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"app/app/models"
	"app/pkg/apperr"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrBalanceNegative — UPDATE ยอดแล้วชน CHECK ยอดติดลบ (DW-34) · service แปลงเป็น 404304 / 404306 ตามทิศของรายการ
var ErrBalanceNegative = errors.New("balance would go negative")

// ชื่อ CHECK ที่ Postgres ตั้งให้อัตโนมัติจาก CREATE TABLE / ADD COLUMN (ยืนยันด้วย integration test DW-34)
const (
	agentBalanceCheck = "agent_balances_amount_check"
	memberCreditCheck = "user_members_credit_check" //nolint:gosec // ชื่อ CHECK constraint ไม่ใช่ credential
)

// balanceErr — แยก CHECK ยอดติดลบด้วยรหัส SQLSTATE + ชื่อ constraint (กฎข้อ 16 ห้ามเทียบข้อความ error)
func balanceErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23514" &&
		(pgErr.ConstraintName == agentBalanceCheck || pgErr.ConstraintName == memberCreditCheck) {
		return ErrBalanceNegative
	}
	return err
}

// GetDWMemberRepository — Member ปลายทาง (ไม่ lock) สำหรับตรวจลูกตรง / สถานะ / สกุล · ไม่พบคืน apperr.ErrNotFound
func GetDWMemberRepository(db *gorm.DB, id uint) (models.UserMember, error) {
	var m models.UserMember
	err := db.Select("id", "agent_id", "username", "currency", "credit", "status").Where("id = ?", id).Take(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return m, apperr.ErrNotFound
	}
	return m, err
}

// LockDWMemberRepository — SELECT ... FOR NO KEY UPDATE ยอดของ Member (lock หลัง agent_balances เสมอ — DW-32) · ไม่พบคืน apperr.ErrNotFound
func LockDWMemberRepository(db *gorm.DB, id uint) (models.UserMember, error) {
	var m models.UserMember
	err := db.Clauses(clause.Locking{Strength: "NO KEY UPDATE"}).Select("id", "agent_id", "credit", "status").Where("id = ?", id).Take(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return m, apperr.ErrNotFound
	}
	return m, err
}

// EnsureAgentBalanceRowsRepository — สร้างแถวยอด 0 ที่ยังไม่มี ก่อน lock (DW-31 · lead DQ3) · agentIDs ต้องเรียงน้อย→มาก
func EnsureAgentBalanceRowsRepository(db *gorm.DB, agentIDs []uint, currency string, at time.Time) error {
	if len(agentIDs) == 0 {
		return nil
	}
	rows := make([]models.AgentBalance, len(agentIDs))
	for i, id := range agentIDs {
		rows[i] = models.AgentBalance{AgentID: id, Currency: currency, UpdatedAt: at}
	}
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
}

// LockDWAgentBalancesRepository — SELECT ... FOR UPDATE ยอดสกุลเดียวของหลายบัญชี เรียง agent_id น้อย→มาก (DW-32 · กฎข้อ 10)
func LockDWAgentBalancesRepository(db *gorm.DB, agentIDs []uint, currency string) ([]models.AgentBalance, error) {
	var out []models.AgentBalance
	if len(agentIDs) == 0 {
		return out, nil
	}
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Select("agent_id", "currency", "amount").
		Where("agent_id IN ? AND currency = ?", agentIDs, currency).Order("agent_id").Find(&out).Error
	return out, err
}

// UpdateDWAgentBalanceRepository — ตั้งยอดของ agent สกุลเดียว · ผู้เรียกต้อง lock แถวแล้ว · ติดลบ = ErrBalanceNegative
func UpdateDWAgentBalanceRepository(db *gorm.DB, agentID uint, currency string, amount float64, at time.Time) error {
	err := db.Model(&models.AgentBalance{}).Where("agent_id = ? AND currency = ?", agentID, currency).
		Updates(map[string]any{"amount": amount, "updated_at": at}).Error
	return balanceErr(err)
}

// UpdateDWMemberCreditRepository — ตั้งยอดของ Member · ผู้เรียกต้อง lock แถวแล้ว · ติดลบ = ErrBalanceNegative
func UpdateDWMemberCreditRepository(db *gorm.DB, id uint, credit float64, at time.Time) error {
	err := db.Model(&models.UserMember{}).Where("id = ?", id).Updates(map[string]any{"credit": credit, "updated_at": at}).Error
	return balanceErr(err)
}

func CreateDWLogRepository(db *gorm.DB, l *models.DepositWithdrawLog) error {
	return db.Create(l).Error
}

// GetDWLogRepository — log 1 แถว (ใช้คืนผลเดิมตอนส่ง request_id ซ้ำ — DW-25) · ไม่พบคืน apperr.ErrNotFound
func GetDWLogRepository(db *gorm.DB, id uint) (models.DepositWithdrawLog, error) {
	var l models.DepositWithdrawLog
	err := db.Where("id = ?", id).Take(&l).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return l, apperr.ErrNotFound
	}
	return l, err
}

// DWDownlineRow — ลูกตรง 1 แถวของ DW1 (ฝั่ง agent หรือ Member)
type DWDownlineRow struct {
	ID          uint
	Role        string // role ของ agent · Member = "MEMBER"
	Username    string
	Name        *string
	Status      models.AgentStatus
	Currency    *string // Member เท่านั้น
	Credit      float64 // Member เท่านั้น
	LastLoginAt *time.Time
	LastLoginIP *string
}

// dwDownlineSQL — ลูกตรงของ parent ทั้งฝั่ง agent (ไม่รวม ADMIN — AUTH-43) และ Member · กรอง username บางส่วน (DW-40 / DW-41)
const dwDownlineSQL = `
	SELECT id, role, username, name, status, NULL AS currency, 0::double precision AS credit, last_login_at, last_login_ip
	FROM user_agents WHERE parent_id = @parent AND role <> 'ADMIN' AND (@q = '' OR username LIKE @like ESCAPE '\')
	UNION ALL
	SELECT id, 'MEMBER', username, name, status, currency, credit, last_login_at, last_login_ip
	FROM user_members WHERE agent_id = @parent AND (@q = '' OR username LIKE @like ESCAPE '\')`

// likeEscaper — ให้ \ % _ ใน keyword เป็นตัวอักษรธรรมดา (ESCAPE '\')
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func dwDownlineArgs(parentID uint, q string) map[string]any {
	return map[string]any{"parent": parentID, "q": q, "like": "%" + likeEscaper.Replace(q) + "%"}
}

// ListDWDownlinesRepository — ลูกตรงเรียง username A→Z แบ่งหน้า + จำนวนทั้งหมด · q = keyword ตัวเล็กแล้ว (ว่าง = ไม่กรอง)
// index: idx_user_agents_parent_username · idx_user_members_agent_username
func ListDWDownlinesRepository(db *gorm.DB, parentID uint, q string, offset, limit int) ([]DWDownlineRow, int64, error) {
	args := dwDownlineArgs(parentID, q)
	var total int64
	if err := db.Raw("SELECT count(*) FROM ("+dwDownlineSQL+") d", args).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []DWDownlineRow
	if total == 0 {
		return rows, 0, nil
	}
	args["offset"], args["limit"] = offset, limit
	err := db.Raw("SELECT * FROM ("+dwDownlineSQL+") d ORDER BY username LIMIT @limit OFFSET @offset", args).Scan(&rows).Error
	return rows, total, err
}

// DWSumRow — ยอดรวมของลูกตรงต่อ role ต่อสกุล (total_sum — DW-44A) · Currency nil = role นี้มีลูกแต่ยังไม่มีแถวยอด
type DWSumRow struct {
	Role     string
	Currency *string
	Total    float64
}

// SumDWDownlinesRepository — ยอดรวมของลูกตรงทั้งหมด (ทุกหน้า · ตาม keyword เดียวกับรายชื่อ) แยก role / สกุล
// ฝั่ง agent LEFT JOIN agent_balances (PK agent_id, currency) · Member รวม credit ตามสกุลของตัวเอง
func SumDWDownlinesRepository(db *gorm.DB, parentID uint, q string) ([]DWSumRow, error) {
	var out []DWSumRow
	err := db.Raw(`
		SELECT a.role AS role, b.currency AS currency, COALESCE(SUM(b.amount), 0) AS total
		FROM user_agents a LEFT JOIN agent_balances b ON b.agent_id = a.id
		WHERE a.parent_id = @parent AND a.role <> 'ADMIN' AND (@q = '' OR a.username LIKE @like ESCAPE '\')
		GROUP BY a.role, b.currency
		UNION ALL
		SELECT 'MEMBER', currency, SUM(credit)
		FROM user_members WHERE agent_id = @parent AND (@q = '' OR username LIKE @like ESCAPE '\')
		GROUP BY currency`, dwDownlineArgs(parentID, q)).Scan(&out).Error
	return out, err
}

// ListAgentBalancesOfRepository — ยอดทุกสกุลของหลายบัญชีในครั้งเดียว (DW-43) · ไม่ lock (อ่านแสดงผล)
func ListAgentBalancesOfRepository(db *gorm.DB, agentIDs []uint) ([]models.AgentBalance, error) {
	var out []models.AgentBalance
	if len(agentIDs) == 0 {
		return out, nil
	}
	err := db.Select("agent_id", "currency", "amount").Where("agent_id IN ?", agentIDs).Find(&out).Error
	return out, err
}

// FindDWDownlineAccountRepository — บัญชี (agent / Member) ที่ username ตรงตัว และอยู่ใต้ ancestorID ทุกชั้น (DW-56 · กฎข้อ 22)
// cnf ไม่รวมตัวเอง → ค้นตัวเองไม่เจอ · ADMIN ไม่มีสาย → ไม่เจอ · ไม่พบคืน apperr.ErrNotFound
// index: uq_user_agents_username · uq_members_username (username ไม่ซ้ำข้าม 2 ตาราง — MGMT-05)
func FindDWDownlineAccountRepository(db *gorm.DB, ancestorID uint, username string) (models.BalanceOwnerType, uint, error) {
	var rows []struct {
		Owner models.BalanceOwnerType
		ID    uint
	}
	chain := fmt.Sprintf(`{"parent":[{"id":%d}]}`, ancestorID)
	// ใช้ CAST(@chain AS jsonb) ห้ามเขียน @chain::jsonb — GORM อ่านชื่อ named param จนเจอช่องว่าง / , / ) จึงได้ชื่อ "chain::jsonb" แล้วไม่แทนค่า
	err := db.Raw(`
		SELECT 'AGENT' AS owner, id FROM user_agents WHERE username = @u AND cnf @> CAST(@chain AS jsonb)
		UNION ALL
		SELECT 'MEMBER', id FROM user_members WHERE username = @u AND cnf @> CAST(@chain AS jsonb)
		LIMIT 1`, map[string]any{"u": username, "chain": chain}).Scan(&rows).Error
	if err != nil {
		return "", 0, err
	}
	if len(rows) == 0 {
		return "", 0, apperr.ErrNotFound
	}
	return rows[0].Owner, rows[0].ID, nil
}

// ListDWHistoryRepository — log ที่ทำให้บัญชีหนึ่ง ช่วง [start, end] รวมขอบ เรียงใหม่→เก่า แบ่งหน้า (DW-58 – DW-60)
// index: idx_deposit_withdraw_logs_target (target_type, target_id, created_at DESC)
func ListDWHistoryRepository(db *gorm.DB, owner models.BalanceOwnerType, id uint, start, end time.Time,
	offset, limit int) ([]models.DepositWithdrawLog, int64, error) {
	base := func() *gorm.DB { // สร้างใหม่ทุกครั้ง — Count แก้ statement ของ query ที่ใช้ต่อไม่ได้
		return db.Model(&models.DepositWithdrawLog{}).
			Where("target_type = ? AND target_id = ? AND created_at >= ? AND created_at <= ?", owner, id, start, end)
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []models.DepositWithdrawLog
	if total == 0 {
		return rows, 0, nil
	}
	err := base().Select("id", "transaction_type", "actor_type", "actor_id", "actor_username", "target_id", "target_role",
		"target_username", "currency", "amount", "note", "status", "created_at").
		Order("created_at DESC, id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

// DWLogFilter — ตัวกรองของ DW3 · scope ด้วย ParentID เสมอ (DW-46 · กฎข้อ 22)
type DWLogFilter struct {
	ParentID   uint
	From, To   time.Time // [From, To) — To = เที่ยงคืนของวันถัดจาก date_to
	Type       models.DWTransactionType
	TargetType models.BalanceOwnerType // "" = ทุกลูก
	TargetID   uint
}

// ListDWLogsRepository — ประวัติเรียงใหม่→เก่า แบ่งหน้า + จำนวนทั้งหมด (DW-46 – DW-49) · index: idx_deposit_withdraw_logs_parent
func ListDWLogsRepository(db *gorm.DB, f DWLogFilter, offset, limit int) ([]models.DepositWithdrawLog, int64, error) {
	// สร้าง query ใหม่ทุกครั้ง — Count แก้ statement ของ query ที่ใช้ต่อไม่ได้
	base := func() *gorm.DB {
		q := db.Model(&models.DepositWithdrawLog{}).Where("parent_id = ? AND created_at >= ? AND created_at < ?", f.ParentID, f.From, f.To)
		if f.Type != "" {
			q = q.Where("transaction_type = ?", f.Type)
		}
		if f.TargetType != "" {
			q = q.Where("target_type = ? AND target_id = ?", f.TargetType, f.TargetID)
		}
		return q
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []models.DepositWithdrawLog
	if total == 0 {
		return rows, 0, nil
	}
	err := base().Select("id", "request_id", "transaction_type", "actor_username", "target_type", "target_id", "target_role", "target_username",
		"currency", "amount", "note", "status", "ip", "parent_unlimited", "parent_balance_before", "parent_balance_after",
		"target_balance_before", "target_balance_after", "created_at").
		Order("created_at DESC, id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}
