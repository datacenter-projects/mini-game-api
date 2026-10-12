package agentmanagement

import (
	"time"

	"app/app/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// sub ของ module ② (phase 5) — docs/modules/agent_management.md MGMT-40 – MGMT-46

// คอลัมน์ที่แสดงในรายชื่อ / รายละเอียด sub (MGMT-46) — ไม่มีรหัสผ่าน / passcode
var subaccountViewColumns = []string{"id", "agent_id", "username", "name", "phone_country_code", "phone", "status", "permissions",
	"created_at", "last_login_at", "last_login_ip"}

// SubaccountUsernameExistsRepository — username นี้มี sub ใช้แล้วไหม
func SubaccountUsernameExistsRepository(db *gorm.DB, username string) (bool, error) {
	var n int64
	err := db.Model(&models.Subaccount{}).Where("username = ?", username).Limit(1).Count(&n).Error
	return n > 0, err
}

// ListSubaccountsRepository — sub ของเจ้าของ เรียง username A→Z แบ่งหน้า + จำนวนทั้งหมด (MGMT-46)
// keyword = ข้อความค้น (ตัวเล็กแล้ว 4–32 ตัว) · ว่าง = ไม่กรอง
func ListSubaccountsRepository(db *gorm.DB, ownerID uint, keyword string, offset, limit int) ([]models.Subaccount, int64, error) {
	base := db.Model(&models.Subaccount{}).Where("agent_id = ?", ownerID)
	if keyword != "" {
		base = base.Where(`username LIKE ? ESCAPE '\'`, "%"+likeEscaper.Replace(keyword)+"%")
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []models.Subaccount
	err := base.Select(subaccountViewColumns).Order("username").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

// GetSubaccountViewRepository — รายละเอียด sub · ไม่พบ = ผลว่าง (id 0)
func GetSubaccountViewRepository(db *gorm.DB, id uint) (models.Subaccount, error) {
	var s models.Subaccount
	err := db.Select(subaccountViewColumns).Where("id = ?", id).Limit(1).Find(&s).Error
	return s, err
}

// LockSubaccountViewRepository — SELECT ... FOR UPDATE ก่อนแก้ · ไม่พบ = ผลว่าง (id 0)
func LockSubaccountViewRepository(db *gorm.DB, id uint) (models.Subaccount, error) {
	return GetSubaccountViewRepository(db.Clauses(clause.Locking{Strength: "UPDATE"}), id)
}

// UpdateSubaccountInfoRepository — ชื่อเล่น · เบอร์ (nil = ไม่ตั้ง) · สิทธิ์ (JSON) แทนทั้งชุด (MGMT-42)
func UpdateSubaccountInfoRepository(db *gorm.DB, id uint, name string, code, phone *string, permissions string, at time.Time) error {
	return db.Model(&models.Subaccount{}).Where("id = ?", id).
		Updates(map[string]any{"name": name, "phone_country_code": code, "phone": phone, "permissions": permissions, "updated_at": at}).Error
}

// UpdateSubaccountStatusRepository — สถานะที่ตั้งกับ sub เอง (MGMT-43 · INACTIVE เก็บเป็น SUSPENDED)
func UpdateSubaccountStatusRepository(db *gorm.DB, id uint, status models.AgentStatus, at time.Time) error {
	return db.Model(&models.Subaccount{}).Where("id = ?", id).Updates(map[string]any{"status": status, "updated_at": at}).Error
}
