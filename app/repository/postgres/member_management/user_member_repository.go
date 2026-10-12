package membermanagement

import (
	"errors"
	"strings"
	"time"

	"app/app/models"
	"app/pkg/apperr"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// repository ของตาราง user_members — docs/modules/member_management.md หัวข้อ 4

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
	err := db.Select("id", "agent_id", "username", "name", "phone_country_code", "phone", "currency", "credit", "status",
		"payment_type", "payment_account", "win_limit_per_day", "last_login_at", "last_login_ip", "created_at").
		Where("id = ?", id).Take(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return m, apperr.ErrNotFound
	}
	return m, err
}

// LockUserMemberRowRepository — SELECT ... FOR NO KEY UPDATE ข้อมูลที่แก้ได้ของ Member · ไม่พบ = ผลว่าง (id 0)
// NO KEY UPDATE (ไม่แก้ id / username): ไม่ขวาง FOR KEY SHARE ที่ Postgres ใช้เช็ค FK ตอน insert ตารางลูกของ Member
// FOR UPDATE ทำ deadlock แบบเดียวกับ LockUserAgentRowRepository (docs/lead/bo-agent-management-round1.md)
func LockUserMemberRowRepository(db *gorm.DB, id uint) (models.UserMember, error) {
	var m models.UserMember
	err := db.Clauses(clause.Locking{Strength: "NO KEY UPDATE"}).Select("id", "agent_id", "username", "name", "phone_country_code", "phone", "status",
		"payment_type", "payment_account", "win_limit_per_day").
		Where("id = ?", id).Limit(1).Find(&m).Error
	return m, err
}

// UpdateUserMemberPaymentRepository — วิธีจ่ายเงิน · บัญชีธนาคาร (JSONB · nil = ไม่มีบัญชี) — MGMT-63 / 64
// ชน unique ของบัญชี (uq_user_members_payment_account) คืน apperr.ErrConflict
func UpdateUserMemberPaymentRepository(db *gorm.DB, id uint, paymentType models.PaymentType, account *string, at time.Time) error {
	err := db.Model(&models.UserMember{}).Where("id = ?", id).
		Updates(map[string]any{"payment_type": paymentType, "payment_account": account, "updated_at": at}).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return apperr.ErrConflict.Wrap(err)
	}
	return err
}

// UpdateUserMemberWinLimitRepository — ลิมิตชนะต่อวัน (MGMT-65)
func UpdateUserMemberWinLimitRepository(db *gorm.DB, id uint, winLimit float64, at time.Time) error {
	return db.Model(&models.UserMember{}).Where("id = ?", id).Updates(map[string]any{"win_limit_per_day": winLimit, "updated_at": at}).Error
}

// UserMemberPaymentAccountTakenByOtherRepository — บัญชี (ธนาคาร + เลข) นี้มี Member อื่นของผู้สร้างคนเดียวกันใช้แล้วไหม (MGMT-64 · lead P4)
// selfID 0 = Member ใหม่ · expression ตรงกับ index uq_user_members_payment_account
func UserMemberPaymentAccountTakenByOtherRepository(db *gorm.DB, agentID uint, bankCode, accountNumber string, selfID uint) (bool, error) {
	var n int64
	err := db.Model(&models.UserMember{}).
		Where("agent_id = ? AND payment_account IS NOT NULL AND (payment_account ->> 'bank_code') = ? AND (payment_account ->> 'account_number') = ? AND id <> ?",
			agentID, bankCode, accountNumber, selfID).
		Limit(1).Count(&n).Error
	return n > 0, err
}

// UpdateUserMemberInfoRepository — ชื่อ · เบอร์ 2 field (nil = ไม่ตั้ง · nil คู่กัน) — MGMT-08, MGMT-09A
// ชน unique ของเบอร์ (uq_user_members_phone) คืน apperr.ErrConflict
func UpdateUserMemberInfoRepository(db *gorm.DB, id uint, name string, phoneCountryCode, phone *string, at time.Time) error {
	err := db.Model(&models.UserMember{}).Where("id = ?", id).
		Updates(map[string]any{"name": name, "phone_country_code": phoneCountryCode, "phone": phone, "updated_at": at}).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return apperr.ErrConflict.Wrap(err)
	}
	return err
}

// ListUserMemberCreditsByIDsRepository — สกุล + ยอดเงิน (credit) ของ Member หลายคน
func ListUserMemberCreditsByIDsRepository(db *gorm.DB, ids []uint) ([]models.UserMember, error) {
	var out []models.UserMember
	if len(ids) == 0 {
		return out, nil
	}
	err := db.Select("id", "currency", "credit").Where("id IN ?", ids).Find(&out).Error
	return out, err
}

// UpdateUserMemberCreditRepository — ตั้งยอดเงิน (credit) ของ Member · ผู้เรียกต้อง lock แถว FOR UPDATE หรือเป็นแถวที่เพิ่งสร้างใน tx เดียวกัน (lead MA5)
func UpdateUserMemberCreditRepository(db *gorm.DB, id uint, credit float64, at time.Time) error {
	return db.Model(&models.UserMember{}).Where("id = ?", id).Updates(map[string]any{"credit": credit, "updated_at": at}).Error
}

// UpdateUserMemberStatusRepository — สถานะที่ตั้งกับบัญชีเอง (MGMT-30)
func UpdateUserMemberStatusRepository(db *gorm.DB, id uint, status models.AgentStatus, at time.Time) error {
	return db.Model(&models.UserMember{}).Where("id = ?", id).Updates(map[string]any{"status": status, "updated_at": at}).Error
}

// UserMemberPhoneTakenByOtherRepository — เบอร์ (คู่) นี้มี Member อื่นของผู้สร้างคนเดียวกันใช้แล้วไหม (MGMT-08 · lead MQ1)
// selfID 0 = Member ใหม่ · ใช้ index uq_user_members_phone (agent_id, phone_country_code, phone)
func UserMemberPhoneTakenByOtherRepository(db *gorm.DB, agentID uint, code, phone string, selfID uint) (bool, error) {
	var n int64
	err := db.Model(&models.UserMember{}).Where("agent_id = ? AND phone_country_code = ? AND phone = ? AND id <> ?", agentID, code, phone, selfID).
		Limit(1).Count(&n).Error
	return n > 0, err
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// ListUserMemberDirectChildrenRepository — Member ที่ agent นี้สร้าง เรียง username A→Z · q = ค้น username บางส่วน
// (ตัวเล็กแล้ว · ว่าง = ไม่กรอง) · ไม่เกิน limit แถว (MGMT-35M) · ใช้ index idx_user_members_agent_username
func ListUserMemberDirectChildrenRepository(db *gorm.DB, agentID uint, q string, limit int) ([]models.UserMember, error) {
	var out []models.UserMember
	query := db.Select("id", "username", "name", "status").Where("agent_id = ?", agentID)
	if q != "" {
		query = query.Where(`username LIKE ? ESCAPE '\'`, "%"+likeEscaper.Replace(q)+"%")
	}
	err := query.Order("username").Limit(limit).Find(&out).Error
	return out, err
}

// FindUserMemberByUsernameRepository — Member ตาม username · ไม่พบ = ผลว่าง (id 0) · username ห้ามซ้ำข้ามตาราง user_agents / user_members (MGMT-05)
func FindUserMemberByUsernameRepository(db *gorm.DB, username string) (models.UserMember, error) {
	var m models.UserMember
	err := db.Select("id", "agent_id", "username", "status", "last_login_at", "last_login_ip", "created_at").
		Where("username = ?", username).Limit(1).Find(&m).Error
	return m, err
}
