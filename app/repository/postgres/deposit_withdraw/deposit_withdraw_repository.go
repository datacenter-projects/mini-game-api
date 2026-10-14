// Package depositwithdraw คือ repository ของ module deposit_withdraw — spec: docs/modules/deposit_withdraw.md
// ตาราง deposit_withdraw_logs เป็นของ module นี้ · แตะ agent_balances / user_members เฉพาะยอดเงิน (ลำดับ lock — ARCHITECTURE.md)
package depositwithdraw

import (
	"errors"
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
	ID        uint
	Role      string // role ของ agent · Member = "MEMBER"
	AgentType *models.AgentType
	Username  string
	Name      *string
	Status    models.AgentStatus
	Currency  *string // Member เท่านั้น
	Credit    float64 // Member เท่านั้น
}

// dwDownlineSQL — ลูกตรงของ parent ทั้งฝั่ง agent (ไม่รวม ADMIN — AUTH-43) และ Member · กรอง username บางส่วน (DW-40 / DW-41)
const dwDownlineSQL = `
	SELECT id, role, agent_type, username, name, status, NULL AS currency, 0::double precision AS credit
	FROM user_agents WHERE parent_id = @parent AND role <> 'ADMIN' AND (@q = '' OR username LIKE @like ESCAPE '\')
	UNION ALL
	SELECT id, 'MEMBER', NULL, username, name, status, currency, credit
	FROM user_members WHERE agent_id = @parent AND (@q = '' OR username LIKE @like ESCAPE '\')`

var likeEscaper = strings.NewReplacer(`\`, `\`, `%`, `\%`, `_`, `\_`)

// ListDWDownlinesRepository — ลูกตรงเรียง username A→Z แบ่งหน้า + จำนวนทั้งหมด · q = keyword ตัวเล็กแล้ว (ว่าง = ไม่กรอง)
// index: idx_user_agents_parent_username · idx_user_members_agent_username
func ListDWDownlinesRepository(db *gorm.DB, parentID uint, q string, offset, limit int) ([]DWDownlineRow, int64, error) {
	args := map[string]any{"parent": parentID, "q": q, "like": "%" + likeEscaper.Replace(q) + "%"}
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

// ListAgentCurrenciesOfRepository — สกุลของหลายบัญชีในครั้งเดียว (DW-43 ห้าม N+1) เรียง agent_id, currency
func ListAgentCurrenciesOfRepository(db *gorm.DB, agentIDs []uint) ([]models.AgentCurrency, error) {
	var out []models.AgentCurrency
	if len(agentIDs) == 0 {
		return out, nil
	}
	err := db.Select("agent_id", "currency").Where("agent_id IN ?", agentIDs).Order("agent_id, currency").Find(&out).Error
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
		"currency", "amount", "note", "parent_unlimited", "parent_balance_before", "parent_balance_after",
		"target_balance_before", "target_balance_after", "created_at").
		Order("created_at DESC, id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}
